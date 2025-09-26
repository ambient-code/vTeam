# RBAC Manifests

This directory contains RBAC definitions for the vTeam platform.

## Roles

### Project-Level Roles

These ClusterRoles are bound to users/groups at the project (namespace) level:

- **ambient-project-view**: Read-only access to project resources
  - View RFE workflows, sessions, and project settings
  - Cannot create or modify resources

- **ambient-project-edit**: Edit access to project resources
  - All view permissions
  - Create and modify RFE workflows and sessions
  - Manage runner secrets
  - Cannot delete resources or manage RBAC

- **ambient-project-admin**: Administrative access to project resources
  - All edit permissions
  - Delete workflows and sessions
  - Manage project RBAC (RoleBindings)
  - Full secret and ConfigMap management

### Service Account Roles

- **ambient-backend-cluster-role**: Backend service permissions
  - Cross-namespace CRD management
  - Project/namespace lifecycle
  - RBAC operations
  - Runner Job/Pod management

## Usage

Bind users to project roles using RoleBindings:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alice-project-admin
  namespace: my-project
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: ambient-project-admin
subjects:
  - kind: User
    name: alice@company.com
    apiGroup: rbac.authorization.k8s.io
```

## Validation

The backend service validates these permissions using SubjectAccessReview:

- FR-014: View access requires `ambient-project-view`
- FR-014a: Edit access requires `ambient-project-edit`
- FR-014b: Admin access requires `ambient-project-admin`