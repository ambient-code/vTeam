# Multi-Tenant Kubernetes Operators: Namespace-per-Tenant Patterns

## Executive Summary

This document outlines architectural patterns for implementing multi-tenant AI session management platforms using Kubernetes operators with namespace-per-tenant isolation. The research reveals three critical architectural pillars: **isolation**, **fair resource usage**, and **tenant autonomy**. Modern approaches have evolved beyond simple namespace isolation to incorporate hierarchical namespaces, virtual clusters, and Internal Kubernetes Platforms (IKPs).

## 1. Best Practices for Namespace-as-Tenant Boundaries

### Core Multi-Tenancy Model

The **namespaces-as-a-service** model assigns each tenant a dedicated set of namespaces within a shared cluster. This approach requires implementing multiple isolation layers:

```yaml
# Tenant CRD Example
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: tenants.platform.ai
spec:
  group: platform.ai
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              namespaces:
                type: array
                items:
                  type: string
              resourceQuota:
                type: object
                properties:
                  cpu: { type: string }
                  memory: { type: string }
                  storage: { type: string }
              rbacConfig:
                type: object
                properties:
                  users: { type: array }
                  serviceAccounts: { type: array }
```

### Three Pillars of Multi-Tenancy

1. **Isolation**: Network policies, RBAC, and resource boundaries
2. **Fair Resource Usage**: Resource quotas and limits per tenant
3. **Tenant Autonomy**: Self-service namespace provisioning and management

### Evolution Beyond Simple Namespace Isolation

Modern architectures combine multiple approaches:
- **Hierarchical Namespaces**: Parent-child relationships with policy inheritance
- **Virtual Clusters**: Isolated control planes within shared infrastructure
- **Internal Kubernetes Platforms (IKPs)**: Pre-configured tenant environments

## 2. Namespace Lifecycle Management from Custom Operators

### Controller-Runtime Reconciliation Pattern

```go
// TenantReconciler manages tenant namespace lifecycle
type TenantReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    Log    logr.Logger
}

func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    tenant := &platformv1.Tenant{}
    if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Ensure tenant namespaces exist
    for _, nsName := range tenant.Spec.Namespaces {
        if err := r.ensureNamespace(ctx, nsName, tenant); err != nil {
            return ctrl.Result{}, err
        }
    }

    // Apply RBAC configurations
    if err := r.applyRBAC(ctx, tenant); err != nil {
        return ctrl.Result{}, err
    }

    // Set resource quotas
    if err := r.applyResourceQuotas(ctx, tenant); err != nil {
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}

func (r *TenantReconciler) ensureNamespace(ctx context.Context, nsName string, tenant *platformv1.Tenant) error {
    ns := &corev1.Namespace{
        ObjectMeta: metav1.ObjectMeta{
            Name: nsName,
            Labels: map[string]string{
                "tenant.platform.ai/name": tenant.Name,
                "tenant.platform.ai/managed": "true",
            },
        },
    }

    // Set owner reference for cleanup
    if err := ctrl.SetControllerReference(tenant, ns, r.Scheme); err != nil {
        return err
    }

    return r.Client.Create(ctx, ns)
}
```

### Automated Tenant Provisioning

The reconciliation loop handles:
- **Namespace Creation**: Dynamic provisioning based on tenant specifications
- **Policy Application**: Automatic application of RBAC, network policies, and quotas
- **Cleanup Management**: Owner references ensure proper garbage collection

### Hierarchical Namespace Controller Integration

```yaml
# HNC Configuration for tenant hierarchy
apiVersion: hnc.x-k8s.io/v1alpha2
kind: HierarchicalNamespace
metadata:
  name: tenant-a-dev
  namespace: tenant-a
spec:
  parent: tenant-a
---
apiVersion: hnc.x-k8s.io/v1alpha2
kind: HNCConfiguration
metadata:
  name: config
spec:
  types:
  - apiVersion: v1
    kind: ResourceQuota
    mode: Propagate
  - apiVersion: networking.k8s.io/v1
    kind: NetworkPolicy
    mode: Propagate
```

## 3. Cross-Namespace Resource Management and Communication

### Controlled Cross-Namespace Access

```go
// ServiceDiscovery manages cross-tenant service communication
type ServiceDiscovery struct {
    client.Client
    allowedConnections map[string][]string
}

func (sd *ServiceDiscovery) EnsureNetworkPolicies(ctx context.Context, tenant *platformv1.Tenant) error {
    for _, ns := range tenant.Spec.Namespaces {
        policy := &networkingv1.NetworkPolicy{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "tenant-isolation",
                Namespace: ns,
            },
            Spec: networkingv1.NetworkPolicySpec{
                PodSelector: metav1.LabelSelector{}, // Apply to all pods
                PolicyTypes: []networkingv1.PolicyType{
                    networkingv1.PolicyTypeIngress,
                    networkingv1.PolicyTypeEgress,
                },
                Ingress: []networkingv1.NetworkPolicyIngressRule{
                    {
                        From: []networkingv1.NetworkPolicyPeer{
                            {
                                NamespaceSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "tenant.platform.ai/name": tenant.Name,
                                    },
                                },
                            },
                        },
                    },
                },
            },
        }

        if err := sd.Client.Create(ctx, policy); err != nil {
            return err
        }
    }
    return nil
}
```

### Shared Platform Services Pattern

```yaml
# Cross-tenant service access via dedicated namespace
apiVersion: v1
kind: Namespace
metadata:
  name: platform-shared
  labels:
    platform.ai/shared: "true"
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-platform-access
  namespace: platform-shared
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          tenant.platform.ai/managed: "true"
```

## 4. Security Considerations and RBAC Patterns

### Multi-Layer Security Architecture

#### Role-Based Access Control (RBAC)

```yaml
# Tenant-specific RBAC template
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: "{{ .TenantNamespace }}"
  name: tenant-admin
rules:
- apiGroups: ["*"]
  resources: ["*"]
  verbs: ["*"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
  resourceNames: ["{{ .TenantNamespace }}"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: tenant-admin-binding
  namespace: "{{ .TenantNamespace }}"
subjects:
- kind: User
  name: "{{ .TenantUser }}"
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: Role
  name: tenant-admin
  apiGroup: rbac.authorization.k8s.io
```

#### Network Isolation Strategies

```go
// NetworkPolicyManager ensures tenant network isolation
func (npm *NetworkPolicyManager) CreateTenantIsolation(ctx context.Context, tenant *platformv1.Tenant) error {
    // Default deny all policy
    denyAll := &networkingv1.NetworkPolicy{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "default-deny-all",
            Namespace: tenant.Spec.PrimaryNamespace,
        },
        Spec: networkingv1.NetworkPolicySpec{
            PodSelector: metav1.LabelSelector{},
            PolicyTypes: []networkingv1.PolicyType{
                networkingv1.PolicyTypeIngress,
                networkingv1.PolicyTypeEgress,
            },
        },
    }

    // Allow intra-tenant communication
    allowIntraTenant := &networkingv1.NetworkPolicy{
        ObjectMeta: metav1.ObjectMeta{
            Name:      "allow-intra-tenant",
            Namespace: tenant.Spec.PrimaryNamespace,
        },
        Spec: networkingv1.NetworkPolicySpec{
            PodSelector: metav1.LabelSelector{},
            PolicyTypes: []networkingv1.PolicyType{
                networkingv1.PolicyTypeIngress,
                networkingv1.PolicyTypeEgress,
            },
            Ingress: []networkingv1.NetworkPolicyIngressRule{
                {
                    From: []networkingv1.NetworkPolicyPeer{
                        {
                            NamespaceSelector: &metav1.LabelSelector{
                                MatchLabels: map[string]string{
                                    "tenant.platform.ai/name": tenant.Name,
                                },
                            },
                        },
                    },
                },
            },
            Egress: []networkingv1.NetworkPolicyEgressRule{
                {
                    To: []networkingv1.NetworkPolicyPeer{
                        {
                            NamespaceSelector: &metav1.LabelSelector{
                                MatchLabels: map[string]string{
                                    "tenant.platform.ai/name": tenant.Name,
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    return npm.applyPolicies(ctx, denyAll, allowIntraTenant)
}
```

### DNS Isolation

```yaml
# CoreDNS configuration for tenant DNS isolation
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-custom
  namespace: kube-system
data:
  tenant-isolation.server: |
    platform.ai:53 {
        kubernetes cluster.local in-addr.arpa ip6.arpa {
            pods insecure
            fallthrough in-addr.arpa ip6.arpa
            ttl 30
        }
        k8s_external hostname
        prometheus :9153
        forward . /etc/resolv.conf
        cache 30
        loop
        reload
        loadbalance
        import /etc/coredns/custom/*.server
    }
```

## 5. Resource Quota and Limit Management

### Dynamic Resource Allocation

```go
// ResourceQuotaManager handles per-tenant resource allocation
type ResourceQuotaManager struct {
    client.Client
    defaultQuotas map[string]resource.Quantity
}

func (rqm *ResourceQuotaManager) ApplyTenantQuotas(ctx context.Context, tenant *platformv1.Tenant) error {
    for _, ns := range tenant.Spec.Namespaces {
        quota := &corev1.ResourceQuota{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "tenant-quota",
                Namespace: ns,
            },
            Spec: corev1.ResourceQuotaSpec{
                Hard: corev1.ResourceList{
                    corev1.ResourceCPU:              tenant.Spec.ResourceQuota.CPU,
                    corev1.ResourceMemory:           tenant.Spec.ResourceQuota.Memory,
                    corev1.ResourceRequestsStorage:  tenant.Spec.ResourceQuota.Storage,
                    corev1.ResourcePods:             resource.MustParse("50"),
                    corev1.ResourceServices:         resource.MustParse("10"),
                    corev1.ResourcePersistentVolumeClaims: resource.MustParse("5"),
                },
            },
        }

        if err := ctrl.SetControllerReference(tenant, quota, rqm.Scheme); err != nil {
            return err
        }

        if err := rqm.Client.Create(ctx, quota); err != nil {
            return err
        }
    }
    return nil
}
```

### Resource Monitoring and Alerting

```yaml
# Prometheus rules for tenant resource monitoring
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: tenant-resource-alerts
  namespace: monitoring
spec:
  groups:
  - name: tenant.rules
    rules:
    - alert: TenantResourceQuotaExceeded
      expr: |
        (
          kube_resourcequota{type="used"} /
          kube_resourcequota{type="hard"}
        ) > 0.9
      for: 5m
      labels:
        severity: warning
        tenant: "{{ $labels.namespace }}"
      annotations:
        summary: "Tenant {{ $labels.namespace }} approaching resource limit"
        description: "Resource {{ $labels.resource }} is at {{ $value }}% of quota"
```

## 6. Monitoring and Observability Across Tenant Namespaces

### Multi-Tenant Metrics Collection

```go
// MetricsCollector aggregates tenant-specific metrics
type MetricsCollector struct {
    client.Client
    metricsClient metrics.Interface
}

func (mc *MetricsCollector) CollectTenantMetrics(ctx context.Context) (*TenantMetrics, error) {
    tenants := &platformv1.TenantList{}
    if err := mc.List(ctx, tenants); err != nil {
        return nil, err
    }

    metrics := &TenantMetrics{
        Tenants: make(map[string]TenantResourceUsage),
    }

    for _, tenant := range tenants.Items {
        usage, err := mc.getTenantUsage(ctx, &tenant)
        if err != nil {
            continue
        }
        metrics.Tenants[tenant.Name] = *usage
    }

    return metrics, nil
}

func (mc *MetricsCollector) getTenantUsage(ctx context.Context, tenant *platformv1.Tenant) (*TenantResourceUsage, error) {
    var totalCPU, totalMemory resource.Quantity

    for _, ns := range tenant.Spec.Namespaces {
        nsMetrics, err := mc.metricsClient.MetricsV1beta1().
            NodeMetricses().
            List(ctx, metav1.ListOptions{
                LabelSelector: fmt.Sprintf("namespace=%s", ns),
            })
        if err != nil {
            return nil, err
        }

        // Aggregate metrics across namespace
        for _, metric := range nsMetrics.Items {
            totalCPU.Add(metric.Usage[corev1.ResourceCPU])
            totalMemory.Add(metric.Usage[corev1.ResourceMemory])
        }
    }

    return &TenantResourceUsage{
        CPU:    totalCPU,
        Memory: totalMemory,
    }, nil
}
```

### Observability Dashboard Configuration

```yaml
# Grafana dashboard for tenant metrics
apiVersion: v1
kind: ConfigMap
metadata:
  name: tenant-dashboard
  namespace: monitoring
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "Multi-Tenant Resource Usage",
        "panels": [
          {
            "title": "CPU Usage by Tenant",
            "type": "graph",
            "targets": [
              {
                "expr": "sum by (tenant) (rate(container_cpu_usage_seconds_total{namespace=~\"tenant-.*\"}[5m]))",
                "legendFormat": "{{ tenant }}"
              }
            ]
          },
          {
            "title": "Memory Usage by Tenant",
            "type": "graph",
            "targets": [
              {
                "expr": "sum by (tenant) (container_memory_usage_bytes{namespace=~\"tenant-.*\"})",
                "legendFormat": "{{ tenant }}"
              }
            ]
          }
        ]
      }
    }
```

## 7. Common Pitfalls and Anti-Patterns to Avoid

### Pitfall 1: Inadequate RBAC Scope

**Anti-Pattern**: Using cluster-wide permissions for namespace-scoped operations

```go
// BAD: Cluster-wide RBAC for tenant operations
//+kubebuilder:rbac:groups=*,resources=*,verbs=*

// GOOD: Namespace-scoped RBAC
//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=*
//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=*
```

### Pitfall 2: Shared CRD Limitations

**Problem**: CRDs are cluster-scoped, creating challenges for tenant-specific schemas

**Solution**: Use tenant-aware CRD designs with validation

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: aisessions.platform.ai
spec:
  group: platform.ai
  scope: Namespaced  # Critical for multi-tenancy
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        properties:
          spec:
            properties:
              tenantId:
                type: string
                pattern: "^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
            required: ["tenantId"]
```

### Pitfall 3: Resource Leak in Reconciliation

**Anti-Pattern**: Not cleaning up orphaned resources

```go
// BAD: No cleanup logic
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Create resources but no cleanup
    return ctrl.Result{}, nil
}

// GOOD: Proper cleanup with finalizers
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    tenant := &platformv1.Tenant{}
    if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion
    if tenant.DeletionTimestamp != nil {
        return r.handleDeletion(ctx, tenant)
    }

    // Add finalizer if not present
    if !controllerutil.ContainsFinalizer(tenant, TenantFinalizer) {
        controllerutil.AddFinalizer(tenant, TenantFinalizer)
        return ctrl.Result{}, r.Update(ctx, tenant)
    }

    // Normal reconciliation logic
    return r.reconcileNormal(ctx, tenant)
}
```

### Pitfall 4: Excessive Reconciliation

**Anti-Pattern**: Triggering unnecessary reconciliations

```go
// BAD: Watching too many resources without filtering
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&platformv1.Tenant{}).
        Owns(&corev1.Namespace{}).
        Owns(&corev1.ResourceQuota{}).
        Complete(r) // This watches ALL namespaces and quotas
}

// GOOD: Filtered watches with predicates
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&platformv1.Tenant{}).
        Owns(&corev1.Namespace{}).
        Owns(&corev1.ResourceQuota{}).
        WithOptions(controller.Options{
            MaxConcurrentReconciles: 1,
        }).
        WithEventFilter(predicate.Funcs{
            UpdateFunc: func(e event.UpdateEvent) bool {
                // Only reconcile if spec changed
                return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
            },
        }).
        Complete(r)
}
```

### Pitfall 5: Missing Network Isolation

**Anti-Pattern**: Assuming namespace boundaries provide network isolation

```yaml
# BAD: No network policies = flat networking
# Pods can communicate across all namespaces

# GOOD: Explicit network isolation
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all
  namespace: tenant-namespace
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
```

## 8. CRD Design for Tenant-Scoped Resources

### Tenant Resource Hierarchy

```yaml
# Primary Tenant CRD
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: tenants.platform.ai
spec:
  group: platform.ai
  scope: Cluster  # Tenant management is cluster-scoped
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              displayName:
                type: string
              adminUsers:
                type: array
                items:
                  type: string
              namespaces:
                type: array
                items:
                  type: object
                  properties:
                    name:
                      type: string
                    purpose:
                      type: string
                      enum: ["development", "staging", "production"]
              resourceQuotas:
                type: object
                properties:
                  cpu:
                    type: string
                    pattern: "^[0-9]+(m|[0-9]*\\.?[0-9]*)?$"
                  memory:
                    type: string
                    pattern: "^[0-9]+([EPTGMK]i?)?$"
                  storage:
                    type: string
                    pattern: "^[0-9]+([EPTGMK]i?)?$"
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Active", "Terminating", "Failed"]
              conditions:
                type: array
                items:
                  type: object
                  properties:
                    type:
                      type: string
                    status:
                      type: string
                    reason:
                      type: string
                    message:
                      type: string
                    lastTransitionTime:
                      type: string
                      format: date-time
              namespaceStatus:
                type: object
                additionalProperties:
                  type: object
                  properties:
                    ready:
                      type: boolean
                    resourceUsage:
                      type: object
                      properties:
                        cpu:
                          type: string
                        memory:
                          type: string
                        storage:
                          type: string

---
# AI Session CRD (namespace-scoped)
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: aisessions.platform.ai
spec:
  group: platform.ai
  scope: Namespaced  # Sessions are tenant-scoped
  versions:
  - name: v1
    schema:
      openAPIV3Schema:
        type: object
        properties:
          spec:
            type: object
            properties:
              tenantRef:
                type: object
                properties:
                  name:
                    type: string
                required: ["name"]
              sessionType:
                type: string
                enum: ["analysis", "automation", "research"]
              aiModel:
                type: string
                enum: ["claude-3-sonnet", "claude-3-haiku", "gpt-4"]
              resources:
                type: object
                properties:
                  cpu:
                    type: string
                    default: "500m"
                  memory:
                    type: string
                    default: "1Gi"
              timeout:
                type: string
                default: "30m"
            required: ["tenantRef", "sessionType"]
          status:
            type: object
            properties:
              phase:
                type: string
                enum: ["Pending", "Running", "Completed", "Failed", "Terminated"]
              startTime:
                type: string
                format: date-time
              completionTime:
                type: string
                format: date-time
              results:
                type: object
                properties:
                  outputData:
                    type: string
                  metrics:
                    type: object
                    properties:
                      tokensUsed:
                        type: integer
                      executionTime:
                        type: string
```

## 9. Architectural Recommendations for AI Session Management Platform

### Multi-Tenant Operator Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Platform Control Plane                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │ Tenant Operator │  │Session Operator │  │Resource Manager │ │
│  │                 │  │                 │  │                 │ │
│  │ - Namespace     │  │ - AI Sessions   │  │ - Quotas        │ │
│  │   Lifecycle     │  │ - Job Creation  │  │ - Monitoring    │ │
│  │ - RBAC Setup    │  │ - Status Mgmt   │  │ - Alerting      │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
              │                    │                    │
              ▼                    ▼                    ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Tenant Namespaces                         │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│ │  tenant-a   │ │  tenant-b   │ │  tenant-c   │ │ shared-svc  │ │
│ │             │ │             │ │             │ │             │ │
│ │ AI Sessions │ │ AI Sessions │ │ AI Sessions │ │ Monitoring  │ │
│ │ Workloads   │ │ Workloads   │ │ Workloads   │ │ Logging     │ │
│ │ Storage     │ │ Storage     │ │ Storage     │ │ Metrics     │ │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Key Architectural Decisions

1. **Namespace-per-Tenant**: Each tenant receives dedicated namespaces for workload isolation
2. **Hierarchical Resource Management**: Parent tenant CRDs manage child AI session resources
3. **Cross-Namespace Service Discovery**: Controlled communication via shared service namespaces
4. **Resource Quota Inheritance**: Tenant-level quotas automatically applied to all namespaces
5. **Automated Lifecycle Management**: Full automation of provisioning, scaling, and cleanup

This architectural framework provides a robust foundation for building scalable, secure, and maintainable multi-tenant AI platforms on Kubernetes, leveraging proven patterns while avoiding common pitfalls in operator development.