# ADR-0001: Kubernetes-Native Architecture

**Status:** Accepted
**Date:** 2024-11-21
**Deciders:** Platform Architecture Team
**Technical Story:** Initial platform architecture design

## Context and Problem Statement

We needed to build an AI automation platform that could:

- Execute long-running AI agent sessions
- Isolate execution environments for security
- Scale based on demand
- Integrate with existing OpenShift/Kubernetes infrastructure
- Support multi-tenancy

How should we architect the platform to meet these requirements?

## Decision Drivers

- **Multi-tenancy requirement:** Need strong isolation between projects
- **Enterprise context:** Red Hat runs on OpenShift/Kubernetes
- **Resource management:** AI sessions have varying resource needs
- **Security:** Must prevent cross-project access and resource interference
- **Scalability:** Need to handle variable workload
- **Operational excellence:** Leverage existing K8s operational expertise

## Considered Options

1. **Kubernetes-native with CRDs and Operators**
2. **Traditional microservices on VMs**
3. **Serverless functions (e.g., AWS Lambda, OpenShift Serverless)**
4. **Container orchestration with Docker Swarm**

## Decision Outcome

Chosen option: "Kubernetes-native with CRDs and Operators", because:

1. **Natural multi-tenancy:** K8s namespaces provide isolation
2. **Declarative resources:** CRDs allow users to declare desired state
3. **Built-in scaling:** K8s handles pod scheduling and resource allocation
4. **Enterprise alignment:** Matches Red Hat's OpenShift expertise
5. **Operational maturity:** Established patterns for monitoring, logging, RBAC

### Consequences

**Positive:**

- Strong multi-tenant isolation via namespaces
- Declarative API via Custom Resources (AgenticSession, ProjectSettings, RFEWorkflow)
- Automatic cleanup via OwnerReferences
- RBAC integration for authorization
- Native integration with OpenShift OAuth
- Horizontal scaling of operator and backend components
- Established operational patterns (logs, metrics, events)

**Negative:**

- Higher learning curve for developers unfamiliar with K8s
- Requires K8s cluster for all deployments (including local dev)
- Operator complexity vs. simpler stateless services
- CRD versioning and migration challenges
- Resource overhead of K8s control plane

**Risks:**

- CRD API changes require careful migration planning
- Operator bugs can affect many sessions simultaneously
- K8s version skew between dev/prod environments

## Implementation Notes

**Architecture Components:**

1. **Custom Resources (CRDs):**
   - AgenticSession: Represents AI execution session
   - ProjectSettings: Project-scoped configuration
   - RFEWorkflow: Multi-agent refinement workflows

2. **Operator Pattern:**
   - Watches CRs and reconciles desired state
   - Creates Kubernetes Jobs for session execution
   - Updates CR status with results

3. **Job-Based Execution:**
   - Each AgenticSession spawns a Kubernetes Job
   - Job runs Claude Code runner pod
   - Results stored in CR status, PVCs for workspace

4. **Multi-Tenancy:**
   - Each project = one K8s namespace
   - RBAC enforces access control
   - Backend validates user tokens before CR operations

**Key Files:**
- `components/manifests/base/*-crd.yaml` - CRD definitions
- `components/operator/internal/handlers/sessions.go` - Operator reconciliation
- `components/backend/handlers/sessions.go` - API to CR translation

## Validation

**Success Metrics:**

- ✅ Multi-tenant isolation validated via RBAC tests
- ✅ Sessions scale from 1 to 50+ concurrent executions
- ✅ Zero cross-project access violations in testing
- ✅ Operator handles CRD updates without downtime

**Lessons Learned:**

- OwnerReferences critical for automatic cleanup
- Status subresource prevents race conditions in updates
- Job monitoring requires separate goroutine per session
- Local dev requires kind/CRC for K8s environment

## Links

- [Kubernetes Operator Pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
- [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- Related: ADR-0002 (User Token Authentication)
