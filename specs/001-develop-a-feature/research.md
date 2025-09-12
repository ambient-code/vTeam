# Research Findings: Multi-Tenant Project-Based Session Management

## Overview
This document consolidates research findings for implementing multi-tenancy in the Ambient Agentic Runner platform, focusing on Kubernetes-native patterns, OpenShift integration, webhook security, and CRD versioning strategies.

## 1. Kubernetes Multi-Tenant Patterns

### Decision: Namespace-per-Project with Enhanced Security
**Rationale**: Modern multi-tenancy in Kubernetes requires defense-in-depth with three pillars: isolation, fair resource usage, and tenant autonomy. Namespace-per-project provides natural resource boundaries while leveraging Kubernetes' built-in security primitives.

**Implementation Approach**:
- Each project maps to a dedicated Kubernetes namespace
- Multi-layer security: RBAC + Network Policies + Resource Quotas
- Controller-runtime patterns with namespace-scoped operators for better isolation
- Filtered watches with predicates to reduce reconciliation overhead

**Alternatives Considered**:
- **Virtual Clusters**: Too complex for current scale requirements
- **Shared Namespaces with Labels**: Insufficient isolation for sensitive AI workloads
- **Cluster-per-Tenant**: Over-engineering for projected 50+ projects

**Key Patterns Applied**:
- Owner references for automatic garbage collection
- Finalizers for proper cleanup during namespace deletion
- Hierarchical namespace structure for large organizations
- Event-driven architecture using OpenShift's event system

## 2. OpenShift Identity Provider Integration

### Decision: Basic OpenShift OAuth Integration
**Rationale**: Simple OAuth integration provides user authentication without complex group management or impersonation patterns. Focus on core functionality first.

**Implementation Approach**:
- Use OpenShift OAuth tokens for API authentication
- Extract user identity and groups from token validation
- Simple RBAC supporting both users and existing OpenShift groups
- Service account tokens for bot accounts

**Security Architecture**:
- Bearer token authentication for all API calls
- User and group-to-project permission mapping
- No complex group management - use existing OpenShift groups
- Basic audit logging with user context

## 3. Webhook Security for Jira Integration

### Decision: Simple JWT Authentication
**Rationale**: JWT tokens provide sufficient security for webhook authentication with minimal complexity. Easy to implement and validate.

**Security Framework**:
- **Authentication**: JWT tokens with shared secret
- **Authorization**: Basic RBAC check for bot account permissions
- **Rate Limiting**: Simple application-level rate limiting
- **Payload Validation**: Basic JSON schema validation

**Implementation Architecture**:
```
Jira → Webhook Service → K8s Operator → AgenticSession
       JWT Validation   RBAC Check     Resource Creation
```

**Bot Account Management**:
- Service accounts with pre-generated JWT tokens
- Basic namespace-scoped permissions for session creation
- Simple audit logging

## 4. CRD Design Strategy

### Decision: Clean Slate Multi-Tenant CRD Design
**Rationale**: Since backward compatibility is not required, design optimal multi-tenant CRDs from scratch without legacy constraints.

**CRD Structure**:
- **Project CRD**: Container for sessions with basic permission management
- **AgenticSession CRD**: Multi-tenant from the start with project references
- **BotAccount CRD**: Simple service account wrapper for automation

**Schema Design**:
```yaml
# AgenticSession v1alpha1 (clean slate)
spec:
  prompt: string
  websiteURL: string
  projectRef:
    name: string
    namespace: string
  userContext:
    userId: string
    displayName: string
  # Optional bot context
  botAccount:
    serviceAccountName: string
```

**Implementation**:
- Start with v1alpha1 version to indicate experimental/evolving status
- No conversion webhooks needed initially
- Clean, minimal schema design
- Standard Kubernetes patterns
- Can evolve to v1beta1 → v1 as the API stabilizes

## 5. Performance and Scale Considerations

### Expected Scale: Start small, 10-20 Projects, 50+ Sessions
- **Namespace Creation**: Standard Kubernetes timing
- **RBAC Operations**: Simple role binding creation
- **Webhook Processing**: Basic processing, no specific latency targets

### Resource Management:
- Basic resource quota per project namespace
- Standard Kubernetes cleanup patterns
- Simple error handling

## 6. Security and Compliance

### Basic Security:
- Token validation for all API requests
- Simple RBAC for resource access
- Basic audit logging for session operations
- Standard Kubernetes security patterns

### Audit Requirements:
- Basic audit trail of session creation and execution
- User context logging for accountability
- Simple webhook activity logging
- Bot account activity tracking

## 7. Technical Debt Considerations

### Identified Risks:
- **RBAC Complexity**: Keep roles simple and minimal
- **Token Management**: Basic token lifecycle with manual rotation
- **Feature Creep**: Start simple, add complexity only when needed

### Mitigation Strategies:
- Simple, well-tested patterns
- Clear documentation
- Incremental feature addition
- Focus on core functionality first

## 8. Implementation Priority

### Phase 0 (Foundation):
1. Basic namespace-per-project setup
2. Simple RBAC with OpenShift integration
3. Basic webhook handling
4. Clean CRD design

### Phase 1 (Core Features):
1. Project management API
2. Basic user permission assignment
3. Simple bot account support
4. Session cloning capabilities

### Phase 2 (Integration):
1. Simple Jira webhook integration
2. Basic multi-tenancy UI
3. Essential monitoring

## 9. Authentication and Observability Strategy

### Decision: Unified OAuth Proxy Architecture with Webhook Bypass
**Rationale**: OAuth2 proxy provides consistent authentication for ALL user-facing requests while allowing webhook endpoints to bypass authentication through separate paths. This ensures security without breaking automation.

**OAuth Proxy Architecture**:
- **OAuth2 Proxy**: Sits in front of ALL user-facing services (frontend, backend API)
- **Authenticated Path**: `/api/*` - all user API calls go through OAuth proxy
- **Webhook Bypass Path**: `/webhooks/*` - direct backend access for external systems
- **Token Validation**: OAuth proxy validates OpenShift tokens, backend validates JWT for webhooks
- **Observability**: Backend pushes data to external tools (no authentication needed)

**Implementation Pattern**:
```
Users → OAuth2 Proxy → Backend API (/api/*)
        ↓ validates     ↓ authenticated requests
     OpenShift token    ↓ includes user context

External Systems → Backend API (/webhooks/*)
(Jira, etc.)       ↓ direct access, bypasses OAuth proxy
                   ↓ validates JWT token internally

Backend → Observability Tools
        ↓ pushes metrics/traces
      No authentication needed
```

**Observability Integration** (TBD):
- **Solution Selection**: Evaluate open source options (Langfuse, Phoenix, OpenLIT, etc.)
- **Project Isolation**: Whatever solution chosen must support project-based data segregation
- **Integration**: Platform sends session traces via API to selected observability backend

## 10. Credential Management Strategy

### Decision: Project-Level Credential Storage in Kubernetes Secrets
**Rationale**: Each project manages its own API keys (LLM, Jira, etc.) for security isolation and cost control. Prevents credential sharing between projects.

**Implementation Approach**:
- **Per-Project Secrets**: Each project namespace contains its own credential secrets
- **Secret Templates**: Standard secret formats for different integrations
- **Access Control**:
  - Viewers: Can see sessions but NOT credentials
  - Editors: Can create/run sessions but NOT view/edit credentials
  - Admins: Can manage credentials and grant credential access
- **Bot Account Access**: Bots can use credentials without viewing them

**Secret Structure per Project**:
```yaml
# In project namespace: project-ml-research-team
apiVersion: v1
kind: Secret
metadata:
  name: project-credentials
  namespace: project-ml-research-team
type: Opaque
data:
  anthropic-api-key: <base64-encoded>
  jira-api-token: <base64-encoded>
  jira-webhook-secret: <base64-encoded>
```

**Benefits**:
- **Cost Attribution**: Each project pays for its own API usage
- **Security Isolation**: Credential leaks limited to single project
- **Team Autonomy**: Projects can use different providers/accounts
- **Audit Trail**: Clear tracking of which project used which credentials

This research provides the foundation for implementing simple, effective multi-tenancy with proper observability and credential isolation while maintaining the platform's existing strengths in AI automation and Kubernetes-native operations.