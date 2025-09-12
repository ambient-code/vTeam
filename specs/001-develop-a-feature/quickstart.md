# Quickstart Guide: Multi-Tenant Project-Based Session Management

## Overview
This quickstart guide demonstrates the key workflows for the new multi-tenant features in Ambient Agentic Runner v2.0. Follow these scenarios to understand project management, user permissions, session operations, and bot account integration.

## Prerequisites
- OpenShift or Kubernetes cluster with Ambient Agentic Runner v2.0 deployed in namespace `ambient-code`
- Valid OpenShift user account with appropriate permissions
- CLI tools: `kubectl`, `oc`, `curl`
- Platform admin access (for bot account scenarios)

## Scenario 1: Create and Manage Ambient Projects

### 1.1 Create Ambient Project via API
```bash
# Create new Ambient project through API (recommended)
curl -X POST "$BACKEND_URL/projects" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ml-research-team",
    "displayName": "ML Research Team",
    "description": "Project for machine learning research experiments",
    "projectType": "research",
    "resourceQuota": {
      "cpu": "8",
      "memory": "16Gi",
      "persistentvolumeclaims": "10"
    }
  }'
```

**This API call automatically**:
- Creates OpenShift project with `oc new-project`
- Applies `ambient-code.io/managed=true` label
- Adds appropriate annotations (`project-type`, `created-by=ui`, `created-at`)
- Triggers ProjectSettings CRD creation
- Sets up resource quotas

### 1.2 Manual OpenShift Project + Labeling
```bash
# Create OpenShift project manually
oc new-project ml-research-team \
  --display-name="ML Research Team" \
  --description="Project for machine learning research experiments"

# Label as Ambient project (triggers ProjectSettings creation)
oc label namespace ml-research-team ambient-code.io/managed=true --overwrite
oc annotate namespace ml-research-team \
  ambient-code.io/project-type=research \
  ambient-code.io/created-by=cli \
  ambient-code.io/created-at="$(date -Iseconds)"

# Add users and groups with appropriate permissions
oc adm policy add-role-to-user admin john.doe@company.com -n ml-research-team
oc adm policy add-role-to-group edit ml-researchers -n ml-research-team
oc adm policy add-role-to-group view ml-stakeholders -n ml-research-team

# Set resource quotas
oc create quota team-quota \
  --hard=cpu=8,memory=16Gi,persistentvolumeclaims=10 \
  -n ml-research-team

# Create project credentials secret
kubectl create secret generic project-credentials \
  --from-literal=anthropic-api-key="your-api-key-here" \
  --from-literal=jira-api-token="your-jira-token" \
  -n ml-research-team
```

**Expected OpenShift Resources**:
- Project: `ml-research-team` with display name and description
- RoleBindings: admin, edit, view roles assigned to users/groups
- ResourceQuota: CPU, memory, and storage limits
- Secret: `project-credentials` for API keys

### 1.3 ProjectSettings (operator-internal)
The ProjectSettings CR is auto-created by the operator when a namespace is labeled `ambient-code.io/managed=true`. It is not managed via public REST endpoints.

### 1.4 List and Verify Ambient Projects
```bash
# List Ambient-managed projects via API
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects"

# Get specific project details via API
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team"

# Manual verification commands
oc get projects -l ambient-code.io/managed=true  # Only Ambient projects
oc describe project ml-research-team
oc get rolebindings -n ml-research-team
oc get resourcequota -n ml-research-team
kubectl get secrets -n ml-research-team

# Check ProjectSettings CRD (auto-created after labeling)
kubectl get projectsettings -n ml-research-team
kubectl describe projectsettings project-settings -n ml-research-team

# Verify Ambient project labels and annotations
kubectl get namespace ml-research-team -o yaml | grep -A 10 -E "(labels|annotations):"
```

**Expected OpenShift Resources**:
- Project: `ml-research-team` namespace with Ambient labels:
  - Label: `ambient-code.io/managed=true`
  - Annotations: `ambient-code.io/project-type=research`, `ambient-code.io/created-by=ui`
- RoleBindings: john.doe (admin), ml-researchers (edit), ml-stakeholders (view)
- ResourceQuota: team-quota with CPU/memory limits
- Secret: `project-credentials` with API keys
- ProjectSettings: `project-settings` with bots, groups, and resource configuration (auto-created)
- ServiceAccounts: `jira-integration-bot`, `github-ci-bot` (created by ProjectSettings operator)

## Scenario 2: Session Management with Project Context

### 2.1 Create a Session in a Project
```bash
# Create an AI session for website analysis
curl -X POST "$BACKEND_URL/projects/ml-research-team/agentic-sessions" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Analyze the user experience of this ML platform, focusing on navigation and accessibility",
    "websiteURL": "https://example-ml-platform.com",
    "displayName": "ML Platform UX Analysis",
    "llmSettings": { "model": "claude-3-5-sonnet-20241022", "temperature": 0.7, "maxTokens": 4000 },
    "timeout": 300,
    "userContext": { "userId": "john.doe@company.com", "displayName": "John Doe", "groups": ["ml-researchers", "company-employees"] },
    "resourceOverrides": { "priorityClass": "high" }
  }'
```

### 2.2 Start Session Execution
```bash
# Start the session
curl -X POST "$BACKEND_URL/projects/ml-research-team/agentic-sessions/platform-ux-analysis/start" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN"
```

### 2.3 Monitor Session Progress
```bash
# Check session status
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team/agentic-sessions/platform-ux-analysis"

# Watch Kubernetes job execution
kubectl get jobs -n ml-research-team -w
kubectl logs -f job/agenticsession-platform-ux-analysis-abc123 -n ml-research-team
```

### 2.4 Clone Session to Another Project
```bash
# Create target OpenShift project first (if needed)
oc new-project product-team --display-name="Product Team"

# Clone session to target project
curl -X POST "$BACKEND_URL/projects/ml-research-team/agentic-sessions/platform-ux-analysis/clone" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targetProject": "product-team",
    "newSessionName": "cloned-ux-analysis"
  }'
```

## Scenario 3: Project Access and Group Management

### 3.1 Check Project Access
```bash
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team/access"
```

### 3.2 Manage Group Access (Admin)
```bash
# List group access
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team/groups"

# Grant group access (admin only)
curl -X POST "$BACKEND_URL/projects/ml-research-team/groups" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "groupName": "data-scientists",
    "role": "edit"
  }'

# Revoke group access (admin only)
curl -X DELETE "$BACKEND_URL/projects/ml-research-team/groups/data-scientists" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN"
```

### 3.3 Manage Project Access Keys
```bash
# List project access keys
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  $BACKEND_URL/projects/ml-research-team/keys

# Create project access key (admin/edit)
curl -X POST $BACKEND_URL/projects/ml-research-team/keys \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "ci-pipeline",
    "description": "CI integration key",
    "role": "edit"
  }'

# Delete project access key (admin)
curl -X DELETE $BACKEND_URL/projects/ml-research-team/keys/<key-id> \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN"
```

### 3.4 Manage Group Access
```bash
# List group access
curl -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team/groups"

# Grant group access (admin only)
curl -X POST "$BACKEND_URL/projects/ml-research-team/groups" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "groupName": "data-scientists",
    "role": "edit"
  }'

# Revoke group access (admin only)
curl -X DELETE "$BACKEND_URL/projects/ml-research-team/groups/data-scientists" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN"
```

## Scenario 4: Access Key (ServiceAccount) Automation

### 4.1 (Optional) Manual ServiceAccount and RBAC
```bash
# Create ServiceAccount for Jira integration bot
kubectl create serviceaccount jira-integration-bot -n ml-research-team

# Create Role with session creation permissions
kubectl apply -f - <<EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: ml-research-team
  name: session-creator
rules:
- apiGroups: ["vteam.ambient-code"]
  resources: ["agenticsessions"]
  verbs: ["create", "get", "list", "patch", "update"]
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
EOF

# Bind ServiceAccount to Role
kubectl create rolebinding jira-bot-session-creator \
  --role=session-creator \
  --serviceaccount=ml-research-team:jira-integration-bot \
  -n ml-research-team

# Create webhook secret placeholder (if using external integrations)
kubectl create secret generic jira-webhook-config \
  --from-literal=webhook-secret="your-webhook-secret-here" \
  --from-literal=jira-base-url="https://company.atlassian.net" \
  -n ml-research-team
```

### 4.2 Get Access Key Token
```bash
# Extract bot service account token
BOT_TOKEN=$(kubectl get secret -n ml-research-team \
  $(kubectl get serviceaccount jira-integration-bot -n ml-research-team -o jsonpath='{.secrets[0].name}') \
  -o jsonpath='{.data.token}' | base64 -d)
```

### 4.3 Test Access Key Authentication
```bash
# Create session using bot token
curl -X POST "$BACKEND_URL/projects/ml-research-team/agentic-sessions" \
  -H "Authorization: Bearer $BOT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Analyze this Jira issue for technical complexity",
    "websiteURL": "https://company.atlassian.net/browse/PROJ-123",
    "displayName": "Jira Issue Analysis",
    "userContext": { "userId": "system-bot", "displayName": "CI Bot", "groups": ["bots", "automation"] },
    "botAccount": { "name": "ci-pipeline" }
  }'
```

## Scenario 5: (Deprecated) Jira Webhook
Direct Jira webhook endpoints are not part of the public API. Use project access keys (ServiceAccounts) with standard session endpoints instead.

## Scenario 6: Permission and Access Control

### 6.1 Test Permission Boundaries
```bash
# Try to access project without permissions (should fail)
curl -H "Authorization: Bearer $UNAUTHORIZED_TOKEN" \
  "$BACKEND_URL/projects/ml-research-team"

# Expected: 403 Forbidden
```

### 6.2 Test Resource Limits
```bash
# Try to create session exceeding project limits
curl -X POST "$BACKEND_URL/projects/ml-research-team/agentic-sessions" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "Test limits",
    "websiteURL": "https://example.com",
    "resourceOverrides": { "maxDurationMinutes": 2000 }
  }'

# Expected: 400 Bad Request - exceeds project limits
```

### 6.3 Session Update
```bash
# Update session configuration
curl -X PUT "$BACKEND_URL/projects/ml-research-team/agentic-sessions/bot-created-session" \
  -H "Authorization: Bearer $OPENSHIFT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"prompt": "Modified prompt", "llmSettings": {"model": "claude-3-haiku-20240307"}}'
```

## Scenario 7: Monitoring and Observability

### 7.1 Check Platform Health
```bash
# Health check
curl "$BACKEND_URL/health"

# Prometheus metrics
curl "$BACKEND_URL/metrics"
```

### 7.2 View Audit Logs
```bash
# Check operator logs for audit trail
kubectl logs -n ambient-code deployment/agentic-operator -f

# ProjectSettings logic is included in the same operator
kubectl logs -n ambient-code deployment/agentic-operator -f

# Check session execution logs
kubectl logs -n ml-research-team job/agenticsession-platform-ux-analysis-abc123
```

### 7.3 Monitor Resource Usage
```bash
# Check project resource consumption
kubectl top pods -n ml-research-team

# View resource quotas and usage
kubectl describe resourcequota -n ml-research-team

# Check ProjectSettings status
kubectl get projectsettings -n ml-research-team -o yaml
```

## Validation Checklist

### Project Management ✓
- [ ] Project creation with proper RBAC setup
- [ ] ProjectSettings CRD creation and configuration
- [ ] Namespace and resource quota creation
- [ ] User permission assignment and validation
- [ ] Bot ServiceAccount creation via ProjectSettings
- [ ] Group access management via API
- [ ] Project deletion with cascade cleanup

### Session Operations ✓
- [ ] Session creation with project context
- [ ] Session execution with proper resource allocation
- [ ] Session cloning between projects (with permissions)
- [ ] Session locking and immutability enforcement

### Bot Integration ✓
- [ ] ServiceAccount and RBAC creation for bots
- [ ] Service account token authentication
- [ ] Project-scoped RBAC permissions for bots
- [ ] Automated session creation via webhooks

### Security and Compliance ✓
- [ ] OpenShift OAuth integration
- [ ] RBAC enforcement at API level
- [ ] Resource limit enforcement
- [ ] Audit trail generation

### Error Handling ✓
- [ ] Permission denied scenarios
- [ ] Resource limit violations
- [ ] Invalid webhook payloads
- [ ] Network and system failures

## Troubleshooting

### Common Issues

**Project Creation Fails**
```bash
# Check operator logs
kubectl logs -n ambient-code deployment/agenticsession-operator
kubectl logs -n ambient-code deployment/projectsettings-operator

# Verify user permissions
oc whoami
oc auth can-i create projects

# Check ProjectSettings CRD
kubectl get crd projectsettings.vteam.ambient-code
kubectl get projectsettings -n ml-research-team

# Verify project is properly labeled for Ambient management
kubectl get namespace ml-research-team -o yaml | grep -E "ambient-code.io"

# If missing labels, add them manually
oc label namespace ml-research-team ambient-code.io/managed=true
oc annotate namespace ml-research-team ambient-code.io/project-type=research
```

**Sessions Not Visible in Project**
```bash
# Check if project is labeled as Ambient project
oc get namespace ml-research-team -o jsonpath='{.metadata.labels.ambient-code\.io/managed}'

# Expected output: "true"
# If empty, the project is not managed by Ambient platform

# Check ProjectSettings exists
kubectl get projectsettings -n ml-research-team
# If not found, label the namespace to trigger creation
```

**Session Won't Start**
```bash
# Check project namespace and quotas
kubectl describe namespace ml-research-team
kubectl get resourcequota -n ml-research-team

# Verify session permissions
kubectl auth can-i create jobs --as=system:serviceaccount:ml-research-team:default -n ml-research-team
```

**Bot Authentication Fails**
```bash
# Check service account and tokens in OpenShift project namespace
kubectl get serviceaccounts -n ml-research-team
kubectl describe serviceaccount jira-integration-bot -n ml-research-team

# Verify bot permissions
kubectl auth can-i create agenticsessions --as=system:serviceaccount:ml-research-team:jira-integration-bot -n ml-research-team

# Check role bindings
kubectl get rolebindings -n ml-research-team
kubectl describe rolebinding jira-bot-session-creator -n ml-research-team
```

This quickstart guide demonstrates the complete multi-tenant workflow from project creation to automated session execution. Each scenario can be executed independently and builds upon the previous ones to showcase the full platform capabilities.