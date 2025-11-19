# Implementation: Fix Multi-Attach PVC Node Affinity

## Problem
AgenticSessions get stuck in "Creating" phase when the temp-content initContainer and the Job's main pods are scheduled on different nodes while using ReadWriteOnce (RWO) PVCs. This causes a Multi-Attach error because RWO PVCs can only be mounted on one node at a time.

## Root Cause
In `components/operator/internal/handlers/sessions.go`, the Job spec creates:
1. InitContainer `init-workspace` - prepares directories on the PVC
2. InitContainer `temp-content` - starts content service temporarily
3. Main containers `ambient-content` and `ambient-code-runner` - the actual workload

When these containers are scheduled on different nodes, the PVC cannot attach to both nodes simultaneously.

## Solution
Add node affinity to ensure all containers in the Job run on the same node as the PVC attachment.

## Code Changes Required

**File**: `components/operator/internal/handlers/sessions.go`

**Location**: In the `corev1.PodSpec` section (around line 385), add a node affinity configuration.

**Change**: Add this field to the PodSpec struct after `AutomountServiceAccountToken`:

```go
Spec: corev1.PodSpec{
    RestartPolicy: corev1.RestartPolicyNever,
    AutomountServiceAccountToken: boolPtr(false),

    // Add node affinity to ensure pod schedules on same node as PVC
    Affinity: &corev1.Affinity{
        PodAffinity: &corev1.PodAffinity{
            PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
                {
                    Weight: 100,
                    PodAffinityTerm: corev1.PodAffinityTerm{
                        LabelSelector: &v1.LabelSelector{
                            MatchExpressions: []v1.LabelSelectorRequirement{
                                {
                                    Key:      "agentic-session",
                                    Operator: v1.LabelSelectorOpIn,
                                    Values:   []string{name},
                                },
                            },
                        },
                        TopologyKey: "kubernetes.io/hostname",
                    },
                },
            },
        },
    },

    Volumes: []corev1.Volume{
        // ... existing volumes ...
```

## Testing

1. Create a new AgenticSession
2. Verify the session pod starts without Multi-Attach errors
3. Check pod events: `oc describe pod <pod-name> -n <namespace>`
4. Verify all containers are on the same node: `oc get pods -o wide`

## Alternative Approaches Considered

- **ReadWriteMany (RWX) PVC**: Would allow multi-node attachment but requires NFS or similar storage backend (not always available)
- **Single InitContainer**: Would work but requires refactoring the workspace prep and content service startup logic
- **Node Selector**: Too rigid, doesn't work well with cluster autoscaling

The node affinity approach is preferred because it's least invasive and works with existing RWO storage.

## Rollback

If this causes issues, revert the commit and manually delete stuck pods:
```bash
git revert <commit-sha>
make deploy NAMESPACE=<namespace>
oc delete pod -l agentic-session=<session-name> -n <namespace>
```
