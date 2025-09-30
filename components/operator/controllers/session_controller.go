// Session Controller Component
// C4 Architecture: Reconciles AgenticSession CRs, creates Jobs
// Manages the lifecycle of AI execution jobs

package controllers

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vteamv1alpha1 "github.com/jeremyeder/vteam/operator/api/v1alpha1"
)

// SessionReconciler reconciles an AgenticSession object
type SessionReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	JobOrchestrator *JobOrchestrator
}

//+kubebuilder:rbac:groups=vteam.io,resources=agenticsessions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vteam.io,resources=agenticsessions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vteam.io,resources=agenticsessions/finalizers,verbs=update
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=pods/log,verbs=get

// Reconcile handles AgenticSession CR changes
func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the AgenticSession instance
	session := &vteamv1alpha1.AgenticSession{}
	err := r.Get(ctx, req.NamespacedName, session)
	if err != nil {
		if errors.IsNotFound(err) {
			// Session deleted, cleanup handled by Kubernetes garbage collection
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if session is being deleted
	if !session.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, session)
	}

	// Add finalizer if not present
	if !containsString(session.Finalizers, "vteam.io/session-finalizer") {
		session.Finalizers = append(session.Finalizers, "vteam.io/session-finalizer")
		if err := r.Update(ctx, session); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Handle session based on current phase
	switch session.Status.Phase {
	case "", "Pending":
		return r.handlePendingSession(ctx, session)
	case "Running":
		return r.handleRunningSession(ctx, session)
	case "Succeeded", "Failed":
		// Terminal state, nothing to do
		return ctrl.Result{}, nil
	default:
		log.Info("Unknown session phase", "phase", session.Status.Phase)
		return ctrl.Result{}, nil
	}
}

// handlePendingSession creates a Job for the session
func (r *SessionReconciler) handlePendingSession(ctx context.Context, session *vteamv1alpha1.AgenticSession) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Create Job using JobOrchestrator
	job, err := r.JobOrchestrator.CreateJobForSession(ctx, session)
	if err != nil {
		log.Error(err, "Failed to create Job for session")
		return r.updateStatus(ctx, session, "Failed", fmt.Sprintf("Failed to create job: %v", err), "", "")
	}

	// Update session status
	now := metav1.Now()
	session.Status.Phase = "Running"
	session.Status.Message = "Job created and running"
	session.Status.JobName = job.Name
	session.Status.StartTime = &now

	if err := r.Status().Update(ctx, session); err != nil {
		return ctrl.Result{}, err
	}

	log.Info("Created Job for session", "job", job.Name)
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// handleRunningSession monitors the Job status
func (r *SessionReconciler) handleRunningSession(ctx context.Context, session *vteamv1alpha1.AgenticSession) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if session.Status.JobName == "" {
		return r.updateStatus(ctx, session, "Failed", "No job associated with session", "", "")
	}

	// Get the Job
	job := &batchv1.Job{}
	err := r.Get(ctx, client.ObjectKey{
		Name:      session.Status.JobName,
		Namespace: session.Namespace,
	}, job)

	if err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatus(ctx, session, "Failed", "Job not found", "", "")
		}
		return ctrl.Result{}, err
	}

	// Check Job status
	if job.Status.Succeeded > 0 {
		// Job succeeded, get output from pod logs
		output, err := r.JobOrchestrator.GetJobOutput(ctx, job)
		if err != nil {
			log.Error(err, "Failed to get job output")
			output = "Output retrieval failed"
		}

		now := metav1.Now()
		session.Status.Phase = "Succeeded"
		session.Status.Message = "Session completed successfully"
		session.Status.EndTime = &now
		session.Status.Output = output

		if err := r.Status().Update(ctx, session); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Session completed successfully", "session", session.Name)
		return ctrl.Result{}, nil
	}

	if job.Status.Failed > 0 {
		// Job failed, get error from pod logs
		errorMsg, _ := r.JobOrchestrator.GetJobOutput(ctx, job)

		now := metav1.Now()
		session.Status.Phase = "Failed"
		session.Status.Message = "Session failed"
		session.Status.EndTime = &now
		session.Status.Error = errorMsg

		if err := r.Status().Update(ctx, session); err != nil {
			return ctrl.Result{}, err
		}

		log.Info("Session failed", "session", session.Name, "error", errorMsg)
		return ctrl.Result{}, nil
	}

	// Check for timeout
	if session.Status.StartTime != nil && session.Spec.Timeout > 0 {
		elapsed := time.Since(session.Status.StartTime.Time)
		if elapsed > time.Duration(session.Spec.Timeout)*time.Second {
			// Timeout exceeded, delete the job
			if err := r.Delete(ctx, job); err != nil {
				log.Error(err, "Failed to delete timed out job")
			}

			now := metav1.Now()
			session.Status.Phase = "Failed"
			session.Status.Message = "Session timed out"
			session.Status.EndTime = &now
			session.Status.Error = fmt.Sprintf("Timeout exceeded: %d seconds", session.Spec.Timeout)

			if err := r.Status().Update(ctx, session); err != nil {
				return ctrl.Result{}, err
			}

			log.Info("Session timed out", "session", session.Name)
			return ctrl.Result{}, nil
		}
	}

	// Job still running, requeue
	return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

// handleDeletion handles session deletion
func (r *SessionReconciler) handleDeletion(ctx context.Context, session *vteamv1alpha1.AgenticSession) (ctrl.Result, error) {
	if containsString(session.Finalizers, "vteam.io/session-finalizer") {
		// Delete associated Job if it exists
		if session.Status.JobName != "" {
			job := &batchv1.Job{}
			err := r.Get(ctx, client.ObjectKey{
				Name:      session.Status.JobName,
				Namespace: session.Namespace,
			}, job)

			if err == nil {
				// Delete the job
				if err := r.Delete(ctx, job); err != nil && !errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
			}
		}

		// Remove finalizer
		session.Finalizers = removeString(session.Finalizers, "vteam.io/session-finalizer")
		if err := r.Update(ctx, session); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// updateStatus updates the session status
func (r *SessionReconciler) updateStatus(ctx context.Context, session *vteamv1alpha1.AgenticSession, phase, message, output, errorMsg string) (ctrl.Result, error) {
	session.Status.Phase = phase
	session.Status.Message = message

	if output != "" {
		session.Status.Output = output
	}
	if errorMsg != "" {
		session.Status.Error = errorMsg
	}

	// Update conditions
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             phase,
		Message:            message,
	}

	if phase == "Failed" {
		condition.Status = metav1.ConditionFalse
	}

	session.Status.Conditions = []metav1.Condition{condition}

	if err := r.Status().Update(ctx, session); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vteamv1alpha1.AgenticSession{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}