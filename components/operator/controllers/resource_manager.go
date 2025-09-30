// Resource Manager Component
// C4 Architecture: Manages quotas, network policies, RBAC
// Handles resource provisioning for multi-tenant projects

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	vteamv1alpha1 "github.com/jeremyeder/vteam/operator/api/v1alpha1"
)

// ResourceManager manages Kubernetes resources for projects
type ResourceManager struct {
	client client.Client
}

// NewResourceManager creates a new ResourceManager
func NewResourceManager(client client.Client) *ResourceManager {
	return &ResourceManager{
		client: client,
	}
}

// CreateResourceQuota creates resource quotas for a project namespace
func (r *ResourceManager) CreateResourceQuota(ctx context.Context, namespace string, quotas vteamv1alpha1.ResourceQuotas) error {
	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-quota",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "vteam",
				"managed": "true",
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{},
		},
	}

	// Parse and set CPU quota
	if quotas.MaxCPU != "" {
		cpuQuantity, err := resource.ParseQuantity(quotas.MaxCPU)
		if err != nil {
			return fmt.Errorf("invalid CPU quota: %v", err)
		}
		quota.Spec.Hard[corev1.ResourceLimitsCPU] = cpuQuantity
		quota.Spec.Hard[corev1.ResourceRequestsCPU] = cpuQuantity
	}

	// Parse and set memory quota
	if quotas.MaxMemory != "" {
		memQuantity, err := resource.ParseQuantity(quotas.MaxMemory)
		if err != nil {
			return fmt.Errorf("invalid memory quota: %v", err)
		}
		quota.Spec.Hard[corev1.ResourceLimitsMemory] = memQuantity
		quota.Spec.Hard[corev1.ResourceRequestsMemory] = memQuantity
	}

	// Parse and set storage quota
	if quotas.MaxStorage != "" {
		storageQuantity, err := resource.ParseQuantity(quotas.MaxStorage)
		if err != nil {
			return fmt.Errorf("invalid storage quota: %v", err)
		}
		quota.Spec.Hard[corev1.ResourceRequestsStorage] = storageQuantity
		quota.Spec.Hard[corev1.ResourcePersistentVolumeClaims] = resource.MustParse("10")
	}

	// Set pod count based on max sessions
	if quotas.MaxSessions > 0 {
		quota.Spec.Hard[corev1.ResourcePods] = resource.MustParse(fmt.Sprintf("%d", quotas.MaxSessions))
	}

	// Additional quotas
	quota.Spec.Hard[corev1.ResourceServices] = resource.MustParse("10")
	quota.Spec.Hard[corev1.ResourceConfigMaps] = resource.MustParse("20")
	quota.Spec.Hard[corev1.ResourceSecrets] = resource.MustParse("20")

	// Create or update the ResourceQuota
	existingQuota := &corev1.ResourceQuota{}
	err := r.client.Get(ctx, client.ObjectKey{Name: quota.Name, Namespace: namespace}, existingQuota)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.client.Create(ctx, quota)
		}
		return err
	}

	// Update existing quota
	existingQuota.Spec = quota.Spec
	return r.client.Update(ctx, existingQuota)
}

// CreateProjectRBAC creates RBAC rules for project access
func (r *ResourceManager) CreateProjectRBAC(ctx context.Context, namespace, owner string, members []string) error {
	// Create Role for project members
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-member",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "vteam",
				"managed": "true",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods", "pods/log", "services", "configmaps", "secrets"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
			{
				APIGroups: []string{"batch"},
				Resources: []string{"jobs"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"vteam.io"},
				Resources: []string{"agenticsessions"},
				Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
			},
		},
	}

	// Create or update Role
	existingRole := &rbacv1.Role{}
	err := r.client.Get(ctx, client.ObjectKey{Name: role.Name, Namespace: namespace}, existingRole)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, role); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		existingRole.Rules = role.Rules
		if err := r.client.Update(ctx, existingRole); err != nil {
			return err
		}
	}

	// Create RoleBinding for owner
	ownerBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-owner",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "vteam",
				"managed": "true",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: owner,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "admin",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	// Create or update owner RoleBinding
	existingOwnerBinding := &rbacv1.RoleBinding{}
	err = r.client.Get(ctx, client.ObjectKey{Name: ownerBinding.Name, Namespace: namespace}, existingOwnerBinding)
	if err != nil {
		if errors.IsNotFound(err) {
			if err := r.client.Create(ctx, ownerBinding); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		existingOwnerBinding.Subjects = ownerBinding.Subjects
		if err := r.client.Update(ctx, existingOwnerBinding); err != nil {
			return err
		}
	}

	// Create RoleBinding for members
	if len(members) > 0 {
		memberSubjects := make([]rbacv1.Subject, len(members))
		for i, member := range members {
			memberSubjects[i] = rbacv1.Subject{
				Kind: "User",
				Name: member,
			}
		}

		memberBinding := &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "project-members",
				Namespace: namespace,
				Labels: map[string]string{
					"app":     "vteam",
					"managed": "true",
				},
			},
			Subjects: memberSubjects,
			RoleRef: rbacv1.RoleRef{
				Kind:     "Role",
				Name:     "project-member",
				APIGroup: "rbac.authorization.k8s.io",
			},
		}

		// Create or update member RoleBinding
		existingMemberBinding := &rbacv1.RoleBinding{}
		err = r.client.Get(ctx, client.ObjectKey{Name: memberBinding.Name, Namespace: namespace}, existingMemberBinding)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(ctx, memberBinding); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			existingMemberBinding.Subjects = memberBinding.Subjects
			if err := r.client.Update(ctx, existingMemberBinding); err != nil {
				return err
			}
		}
	}

	return nil
}

// CreateNetworkPolicies creates network isolation for the project
func (r *ResourceManager) CreateNetworkPolicies(ctx context.Context, namespace string) error {
	// Default deny all ingress except from same namespace
	denyIngress := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-ingress",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "vteam",
				"managed": "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
			},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							PodSelector: &metav1.LabelSelector{},
						},
					},
				},
			},
		},
	}

	// Allow egress to internet and Kubernetes API
	allowEgress := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-egress",
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "vteam",
				"managed": "true",
			},
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeEgress,
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					// Allow all egress (can be restricted further)
					To: []networkingv1.NetworkPolicyPeer{},
				},
			},
		},
	}

	// Create network policies
	for _, policy := range []*networkingv1.NetworkPolicy{denyIngress, allowEgress} {
		existing := &networkingv1.NetworkPolicy{}
		err := r.client.Get(ctx, client.ObjectKey{Name: policy.Name, Namespace: namespace}, existing)
		if err != nil {
			if errors.IsNotFound(err) {
				if err := r.client.Create(ctx, policy); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}