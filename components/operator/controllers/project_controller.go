// Project Controller Component
// C4 Architecture: Reconciles Project CRs, manages namespaces
// Creates and manages project namespaces with RBAC and quotas

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	vteamv1alpha1 "github.com/jeremyeder/vteam/operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	ResourceManager *ResourceManager
}

//+kubebuilder:rbac:groups=vteam.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=vteam.io,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=vteam.io,resources=projects/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=resourcequotas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles Project CR changes
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Project instance
	project := &vteamv1alpha1.Project{}
	err := r.Get(ctx, req.NamespacedName, project)
	if err != nil {
		if errors.IsNotFound(err) {
			// Project deleted, cleanup handled by Kubernetes garbage collection
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Check if project is being deleted
	if !project.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, project)
	}

	// Add finalizer if not present
	if !containsString(project.Finalizers, "vteam.io/project-finalizer") {
		project.Finalizers = append(project.Finalizers, "vteam.io/project-finalizer")
		if err := r.Update(ctx, project); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create or update project namespace
	namespace := fmt.Sprintf("vteam-%s", project.Name)
	if err := r.ensureNamespace(ctx, project, namespace); err != nil {
		log.Error(err, "Failed to ensure namespace")
		return r.updateStatus(ctx, project, "Failed", err.Error())
	}

	// Create resource quotas
	if err := r.ResourceManager.CreateResourceQuota(ctx, namespace, project.Spec.Quotas); err != nil {
		log.Error(err, "Failed to create resource quota")
		return r.updateStatus(ctx, project, "Failed", err.Error())
	}

	// Create RBAC for project members
	if err := r.ResourceManager.CreateProjectRBAC(ctx, namespace, project.Spec.Owner, project.Spec.Members); err != nil {
		log.Error(err, "Failed to create RBAC")
		return r.updateStatus(ctx, project, "Failed", err.Error())
	}

	// Create network policies for isolation
	if err := r.ResourceManager.CreateNetworkPolicies(ctx, namespace); err != nil {
		log.Error(err, "Failed to create network policies")
		return r.updateStatus(ctx, project, "Failed", err.Error())
	}

	// Update status
	return r.updateStatus(ctx, project, "Active", "Project resources created successfully")
}

// ensureNamespace creates the project namespace if it doesn't exist
func (r *ProjectReconciler) ensureNamespace(ctx context.Context, project *vteamv1alpha1.Project, namespaceName string) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
			Labels: map[string]string{
				"app":     "vteam",
				"project": project.Name,
				"owner":   project.Spec.Owner,
			},
			Annotations: map[string]string{
				"description": project.Spec.Description,
			},
		},
	}

	// Set owner reference for garbage collection
	ctrl.SetControllerReference(project, namespace, r.Scheme)

	// Create or update namespace
	err := r.Get(ctx, client.ObjectKey{Name: namespaceName}, &corev1.Namespace{})
	if err != nil {
		if errors.IsNotFound(err) {
			return r.Create(ctx, namespace)
		}
		return err
	}

	// Namespace exists, update it
	return r.Update(ctx, namespace)
}

// handleDeletion handles project deletion
func (r *ProjectReconciler) handleDeletion(ctx context.Context, project *vteamv1alpha1.Project) (ctrl.Result, error) {
	if containsString(project.Finalizers, "vteam.io/project-finalizer") {
		// Cleanup project resources
		namespace := fmt.Sprintf("vteam-%s", project.Name)

		// Delete namespace (this will cascade delete all resources)
		ns := &corev1.Namespace{}
		err := r.Get(ctx, client.ObjectKey{Name: namespace}, ns)
		if err == nil {
			if err := r.Delete(ctx, ns); err != nil && !errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}

		// Remove finalizer
		project.Finalizers = removeString(project.Finalizers, "vteam.io/project-finalizer")
		if err := r.Update(ctx, project); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// updateStatus updates the project status
func (r *ProjectReconciler) updateStatus(ctx context.Context, project *vteamv1alpha1.Project, phase string, message string) (ctrl.Result, error) {
	project.Status.Phase = phase
	project.Status.Namespace = fmt.Sprintf("vteam-%s", project.Name)
	project.Status.LastUpdated = metav1.Now()

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

	project.Status.Conditions = []metav1.Condition{condition}

	if err := r.Status().Update(ctx, project); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vteamv1alpha1.Project{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ResourceQuota{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

// Helper functions
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	result := []string{}
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}