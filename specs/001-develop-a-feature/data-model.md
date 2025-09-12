# Data Model: Multi-Tenant Project-Based Session Management

## Overview
This document defines the data models and relationships for extending the Ambient Agentic Runner platform with multi-tenancy support. The design maintains backward compatibility with existing AgenticSession resources while adding project isolation and user context.

## Core Entities

**Design Decision**: Two-CRD model using AgenticSession + ProjectSettings. Projects are standard OpenShift projects/namespaces with ProjectSettings CRD for UI management. Bot accounts are standard ServiceAccounts with RBAC.

### 1. OpenShift Project (Standard Kubernetes Namespace)
**Purpose**: Native OpenShift project serves as container for organizing sessions
**Implementation**: Standard `oc new-project` command creates namespace with RBAC
**Ambient Integration**: Projects must be labeled as "ambient projects" to enable ProjectSettings

**Example OpenShift Project Creation**:
```bash
# Create new OpenShift project
oc new-project ml-research-team --description="ML Research Team" --display-name="ML Research Team"

# Label as Ambient project (enables ProjectSettings management)
oc label namespace ml-research-team ambient-code.io/managed=true
oc annotate namespace ml-research-team ambient-code.io/project-type=research

# Add users to project with different roles
oc adm policy add-role-to-user admin john.doe@company.com -n ml-research-team
oc adm policy add-role-to-user edit jane.smith@company.com -n ml-research-team
oc adm policy add-role-to-group view ml-stakeholders -n ml-research-team

# Set resource quotas (optional)
oc create quota team-quota --hard=cpu=8,memory=16Gi,persistentvolumeclaims=10 -n ml-research-team
```

**Ambient Project Identification**:
- **Label**: `ambient-code.io/managed=true` - Indicates project is managed by Ambient platform
- **Annotation**: `ambient-code.io/project-type=<type>` - Optional project classification (research, production, demo)
- **Annotation**: `ambient-code.io/created-by=<ui|cli|api>` - Source of project creation
- **Annotation**: `ambient-code.io/created-at=<timestamp>` - Creation timestamp for audit

**Built-in OpenShift Features**:
- Automatic RBAC with admin/edit/view roles
- Network isolation between projects
- Resource quotas and limits
- Audit logging and monitoring

### 2. ProjectSettings (v1alpha1)
**Purpose**: UI-managed configuration for project resources, bots, and access controls
**Implementation**: One ProjectSettings CRD per Ambient-labeled OpenShift project namespace
**Trigger**: Automatically created when namespace is labeled with `ambient-code.io/managed=true`

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: ProjectSettings
metadata:
  name: project-settings
  namespace: ml-team
spec:
  displayName: "ML Research Team"
  description: "Machine learning research and experiments"

  # Simple bot registry - creates ServiceAccounts
  bots:
  - name: jira-integration-bot
    description: "Jira webhook integration"
  - name: github-ci-bot
    description: "GitHub CI integration"

  # Group access management - creates RoleBindings
  groupAccess:
  - groupName: ml-researchers
    role: edit  # Can create sessions in project
  - groupName: ml-stakeholders
    role: view  # Can view + clone sessions

  # Available resources for sessions in this project
  availableResources:
    # LLM models available to this project
    models:
    - name: "claude-3-5-sonnet-20241022"
      displayName: "Claude 3.5 Sonnet"
      costPerToken: 0.00001
      maxTokens: 200000
      default: true
    - name: "claude-3-haiku-20240307"
      displayName: "Claude 3 Haiku"
      costPerToken: 0.000001
      maxTokens: 200000
    - name: "gpt-4o"
      displayName: "GPT-4o"
      costPerToken: 0.00002
      maxTokens: 128000

    # Resource limits per session
    resourceLimits:
      cpu: "2000m"
      memory: "4Gi"
      storage: "10Gi"
      maxDurationMinutes: 120

    # Priority classes available
    priorityClasses:
    - "low"
    - "normal"
    - "high"

    # Integrations available
    integrations:
    - type: "browser"
      enabled: true
    - type: "code-execution"
      enabled: true
    - type: "file-upload"
      enabled: false  # Not allowed for this project

  # Default settings when creating sessions
  defaults:
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.7
    maxTokens: 4000
    timeout: 300
    priorityClass: "normal"

  # Project-level constraints
  constraints:
    maxConcurrentSessions: 10
    maxSessionsPerUser: 3
    maxCostPerSession: 50.0  # USD
    maxCostPerUserPerDay: 200.0  # USD
    allowSessionCloning: true
    allowBotAccounts: true

status:
  phase: "Active"  # Active|Pending|Error
  botsCreated: 2
  groupBindingsCreated: 2
  lastReconciled: "2025-09-15T15:30:00Z"
  currentUsage:
    activeSessions: 3
    totalCostToday: 45.67
  conditions:
  - type: BotsReady
    status: "True"
    reason: ServiceAccountsCreated
    message: "All bot ServiceAccounts created successfully"
  - type: RBACReady
    status: "True"
    reason: RoleBindingsCreated
    message: "All group RoleBindings created successfully"
```

**ProjectSettings Operator Behavior**:
1. **Namespace Watch**: Watches for namespaces with label `ambient-code.io/managed=true`
2. **Auto-Creation**: Creates default ProjectSettings CR when labeled namespace is detected
3. **ServiceAccount Creation**: Creates ServiceAccounts for each bot in `spec.bots[]`
4. **RBAC Management**: Creates RoleBindings for each group in `spec.groupAccess[]`
5. **Status Updates**: Maintains status with bot creation and binding success/failure

**Validation Rules**:
- Must have at least one available model with default=true
- Resource limits must be within cluster capacity
- Priority classes must exist in cluster
- Cost constraints must be positive numbers
- Namespace must have `ambient-code.io/managed=true` label

### 3. AgenticSession (v1alpha1)
**Purpose**: AI-powered automation session with project context and user attribution
**Backward Compatibility**: v1 sessions automatically migrated with default project assignment

```yaml
apiVersion: vteam.ambient-code/v1alpha1
kind: AgenticSession
metadata:
  name: "website-analysis-session-001"
  namespace: "ml-research-team"  # Lives in OpenShift project namespace
  labels:
    user.ambient-code/id: "john.doe@company.com"
    session.ambient-code/type: "analysis"
spec:
  # Core session configuration (preserved from v1)
  prompt: "Analyze the user experience of this machine learning platform"
  websiteURL: "https://example-ml-platform.com"
  displayName: "ML Platform UX Analysis"

  llmSettings:
    model: "claude-3-5-sonnet-20241022"
    temperature: 0.7
    maxTokens: 4000

  timeout: 300

  # Multi-tenancy fields (v1alpha1)
  userContext:
    userId: "john.doe@company.com"
    displayName: "John Doe"
    groups: ["ml-researchers", "company-employees"]

  # Bot account for automated sessions (optional)
  botAccount:
    serviceAccountName: "jira-integration-bot"
    automated: true  # Indicates session created by bot

  # Session-specific overrides
  resourceOverrides:
    maxDurationMinutes: 45
    priorityClass: "high"

  # Immutability settings
  locked: false  # If true, session configuration cannot be modified

status:
  phase: "Running"  # Pending|Creating|Running|Completed|Failed|Stopped|Error
  message: "Session executing successfully"
  startTime: "2025-09-14T15:00:00Z"
  completionTime: null

  # Execution metadata
  jobName: "agenticsession-website-analysis-001-abc123"
  podName: "agenticsession-website-analysis-001-abc123-pod"

  # Results
  finalOutput: ""
  cost: 0.0
  tokenUsage:
    inputTokens: 0
    outputTokens: 0

  # Audit trail
  createdBy: "john.doe@company.com"
  lastModifiedBy: "john.doe@company.com"
  lastModifiedAt: "2025-09-14T15:00:00Z"

  # Resource allocation from namespace limits
  resourceAllocation:
    cpu: "500m"
    memory: "1Gi"
```

**State Transitions**:
- Pending → Creating → Running → (Completed|Failed|Stopped)
- Error state possible from any phase
- Locked sessions can transition between execution states but not modify spec

### 4. RBAC Implementation (Custom Roles + Standard ServiceAccounts)
**Purpose**: Project-specific permissions using custom RBAC roles
**Security**: Leverages Kubernetes RBAC with custom roles for fine-grained access control

#### **Custom RBAC Roles:**

**agenticsession-admin** (Project administrators):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agenticsession-admin
  namespace: ml-team
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions", "projectsettings"]
  verbs: ["create", "get", "list", "update", "patch", "delete"]
- apiGroups: [""]
  resources: ["secrets", "serviceaccounts"]
  verbs: ["create", "get", "list", "update", "patch", "delete"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch"]
```

**agenticsession-edit** (Can create sessions in project):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agenticsession-edit
  namespace: ml-team
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["create", "get", "list", "update", "patch", "delete"]
- apiGroups: ["vteam.ambient-code"]
  resources: ["projectsettings"]
  verbs: ["get", "list"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch"]
```

**agenticsession-view** (Can view + clone sessions):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agenticsession-view
  namespace: ml-team
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions", "projectsettings"]
  verbs: ["get", "list"]
- apiGroups: ["batch"]
  resources: ["jobs"]
  verbs: ["get", "list", "watch"]
```

**agenticsession-bot** (ServiceAccounts for automation):
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: agenticsession-bot
  namespace: ml-team
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["create"]
- apiGroups: ["vteam.ambient-code"]
  resources: ["projectsettings"]
  verbs: ["get"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
```

#### **Generated RoleBindings:**
The ProjectSettings operator creates RoleBindings based on the `groupAccess` configuration:

```yaml
# Example: ml-researchers group gets edit access
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: ml-researchers-edit
  namespace: ml-team
subjects:
- kind: Group
  name: ml-researchers
roleRef:
  kind: Role
  name: agenticsession-edit
  apiGroup: rbac.authorization.k8s.io
---
# Example: Bot ServiceAccount gets limited access
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: jira-integration-bot
  namespace: ml-team
subjects:
- kind: ServiceAccount
  name: jira-integration-bot
  namespace: ml-team
roleRef:
  kind: Role
  name: agenticsession-bot
  apiGroup: rbac.authorization.k8s.io
```

## Data Relationships

### Enhanced Entity Relationships
```
OpenShift Project (1) ──── (1) ProjectSettings
        │                           │
        │                           ├── bots[] → ServiceAccounts
        │                           ├── groupAccess[] → RoleBindings
        │                           └── availableResources → Session validation
        │
        ├── (n) AgenticSession ──── validates against → ProjectSettings
        │         │
        │         └── (created by) → User/ServiceAccount
        │
        ├── RBAC ────────────────── User/Group (via RoleBindings)
        └── RBAC ────────────────── ServiceAccount (via RoleBindings)
```

### Relationship Rules
1. **OpenShift Project → ProjectSettings**: One-to-one, each project has exactly one settings CR
2. **OpenShift Project → AgenticSession**: One-to-many, sessions must belong to exactly one project
3. **ProjectSettings → ServiceAccounts**: ProjectSettings.spec.bots creates ServiceAccounts in namespace
4. **ProjectSettings → RoleBindings**: ProjectSettings.spec.groupAccess creates RoleBindings in namespace
5. **AgenticSession → ProjectSettings**: Sessions validated against available resources and constraints
6. **AgenticSession → User/ServiceAccount**: Each session has creator context in userContext field

## Data Migration Strategy

### v1 → v1alpha1 AgenticSession Migration
```yaml
# Default values for v1 sessions during migration
userContext:
  userId: "system-migration"       # System user for migrated sessions
  displayName: "System Migration"
  groups: ["migrated-users"]
resourceOverrides: {}              # No overrides for migrated sessions
```

### Migration Process
1. **Pre-migration**: Create default OpenShift project for migrated sessions: `oc new-project legacy-sessions`
2. **Namespace Migration**: Move existing sessions to `legacy-sessions` project
3. **Schema Conversion**: Apply default userContext to all v1 sessions
4. **Validation**: Verify all sessions accessible via v1alpha1 API

## Validation and Constraints

### Cross-Entity Validation
- Sessions can only be created in OpenShift projects where user has RBAC permissions
- ServiceAccounts (bots) can only operate in namespaces with appropriate role bindings
- Resource requests cannot exceed OpenShift project quotas
- Session cloning requires access to both source and destination OpenShift projects

### Business Rules
- OpenShift project deletion cascades to all contained sessions and RBAC resources
- User removal from OpenShift project preserves existing sessions but prevents new ones
- ServiceAccount deletion immediately blocks new session creation
- Session execution continues but configuration remains immutable

### Security Constraints
- All OpenShift project operations use standard OpenShift RBAC
- Bot operations must include valid service account token
- Cross-project session access requires explicit RBAC grants
- Audit logs cannot be modified, only appended

## Storage Considerations

### Enhanced Kubernetes Resource Organization
```
Cluster Level:
├── CRDs (CustomResourceDefinitions)
│   ├── agenticsessions.vteam.ambient-code
│   └── projectsettings.vteam.ambient-code
└── OpenShift Projects (standard namespaces)

Platform Namespace (ambient-code):
├── Operator components
│   ├── AgenticSession controller
│   └── ProjectSettings controller
└── Platform-wide RBAC roles
    ├── agenticsession-admin
    ├── agenticsession-edit
    ├── agenticsession-view
    └── agenticsession-bot

OpenShift Project Namespaces:
├── ProjectSettings (one per namespace)
├── AgenticSessions (tenant-scoped)
├── ServiceAccounts (created by ProjectSettings operator)
├── Kubernetes Jobs (execution)
├── RoleBindings (created by ProjectSettings operator)
└── Secrets (project credentials)
```

### Resource Quotas and Limits
- Each OpenShift project namespace has standard ResourceQuota (set by cluster admin)
- Session pods have resource requests/limits based on user RBAC permissions
- ServiceAccounts use standard Kubernetes resource accounting
- Monitoring data retained for 90 days, audit logs for 1 year

This extremely simplified data model provides secure, scalable multi-tenancy using only standard OpenShift/Kubernetes primitives with a single custom CRD for AgenticSessions.