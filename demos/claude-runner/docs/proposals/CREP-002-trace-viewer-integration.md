# CREP-002: Playwright Trace Viewer Integration

**Authors:** @gkrumbac  
**Status:** Draft  
**Creation Date:** 2024-12-19  
**Last Updated:** 2024-12-19

## Summary
Integrate Playwright's Trace Viewer directly into Claude Runner to provide comprehensive browser automation analysis, debugging capabilities, and complete research transparency through interactive trace visualization.

## Motivation
### Goals
- Provide comprehensive audit trail of all browser-based research activities
- Enable interactive debugging and analysis of research sessions
- Enhance client transparency by visualizing the complete research methodology  
- Improve system reliability through detailed error diagnosis capabilities
- Support research quality assurance through replay and verification

### Non-Goals  
- Replace existing logging and monitoring infrastructure
- Provide video recording capabilities (traces are sufficient)
- Support trace editing or modification
- Real-time trace streaming (post-session analysis focus)

### User Stories
- As a **research client**, I want to see exactly how my research was conducted with full browser interaction details
- As a **platform operator**, I want to debug failed research sessions by replaying the exact browser interactions
- As a **quality assurance team member**, I want to verify research methodology and identify potential improvements
- As a **compliance officer**, I want complete audit trails of data collection activities
- As a **developer**, I want to understand performance bottlenecks in research workflows

## Proposal
### Overview
This proposal extends Claude Runner with Playwright's trace recording capabilities and embeds the official Playwright Trace Viewer directly into the web interface. This provides unprecedented visibility into browser automation activities while maintaining security and performance.

### Current State vs Target State

| Aspect | Current State | Target State |
|--------|---------------|--------------|
| **Browser Automation** | Basic Playwright MCP with screenshots | Enhanced MCP with full trace recording |
| **Debugging** | Console logs only | Interactive trace replay with network/DOM analysis |
| **Audit Trail** | Text logs and static screenshots | Complete browser interaction timeline |
| **Error Analysis** | Manual log inspection | Visual debugging with exact failure reproduction |
| **Client Transparency** | Final results only | Complete methodology visualization |
| **Performance Analysis** | None | Detailed timing and resource usage metrics |
| **Storage** | Local container storage only | Persistent PVC-based artifact storage |

### Implementation Details

#### Component Architecture
```
┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐
│   Frontend      │  │  Trace Viewer   │  │    Backend      │
│                 │  │    Service      │  │                 │
│ - Session UI    │◄─┤ - Static App    │  │ - Artifact API  │
│ - Trace Modal   │  │ - CORS Config   │  │ - File Serving  │
│ - Integration   │  │ - URL Params    │  │ - Status Update │
└─────────────────┘  └─────────────────┘  └─────────────────┘
         ▲                      ▲                      ▲
         │                      │                      │
         └──────────────────────┼──────────────────────┘
                                │
         ┌─────────────────┐    │    ┌─────────────────┐
         │ Claude Runner   │    │    │   Kubernetes    │
         │                 │    │    │                 │
         │ - Enhanced MCP  │────┘    │ - PVC Storage   │
         │ - Trace Save    │         │ - Job Lifecycle │
         │ - Artifacts     │         │ - CRD Updates   │
         └─────────────────┘         └─────────────────┘
```

#### Trace Recording Flow
1. **Session Creation**: Operator configures job with trace-enabled MCP
2. **Browser Initialization**: Claude runner starts MCP with `--save-trace` flag
3. **Research Execution**: All browser interactions recorded automatically
4. **Trace Storage**: Completed traces saved to persistent storage
5. **Artifact Registration**: CRD updated with trace file locations and viewer URLs
6. **Client Access**: Frontend provides embedded trace viewer with direct file access

### API Changes

#### CRD Enhancement (ResearchSession)
```yaml
# New fields in spec
spec:
  traceSettings:
    enabled: boolean (default: true)
    retention: duration (default: "168h") # 7 days
    
# New fields in status  
status:
  traceViewerUrl: string
  artifacts:
    - type: "trace" | "screenshot" | "pdf"
      filename: string
      path: string
      size: integer
      viewerUrl: string # Direct trace viewer URL
      createdAt: string
```

#### Backend API Extensions
```go
// New endpoints
GET  /api/research-sessions/{name}/artifacts
GET  /api/artifacts/{path}
GET  /api/trace-viewer/{session}/{trace}
```

### Migration Strategy

#### Phase 1: Infrastructure Setup
- Deploy Playwright Trace Viewer service
- Create artifacts PVC and storage configuration
- Update operator with trace recording capabilities

#### Phase 2: Core Integration  
- Enhance Claude runner with trace recording
- Update CRD with artifact tracking fields
- Implement backend artifact serving APIs

#### Phase 3: Frontend Integration
- Add trace viewer iframe integration
- Create artifact display components
- Implement session trace navigation

#### Phase 4: Production Rollout
- Deploy to staging environment
- Validate trace generation and viewing
- Gradual production deployment with feature flags

## Design Details

### Security Considerations

#### Access Control
- Trace files contain sensitive browsing data - restrict access to session owners
- CORS configuration limited to known domains
- Artifact URLs include session-specific tokens for authorization
- PVC access restricted through Kubernetes RBAC

#### Data Protection
- Traces may contain form data, cookies, and personal information
- Implement automatic trace sanitization for sensitive fields
- Configurable retention policies with automatic cleanup
- Optional trace encryption at rest

#### Network Security
- Trace viewer service runs in isolated network namespace
- All artifact access logged for audit purposes
- Rate limiting on artifact download endpoints

### Performance Impact

#### Storage Requirements
- Trace files: ~50-200MB per complex research session
- Screenshots: ~1-5MB each (5-10 per session typical)
- Estimated storage: ~100-500MB per session
- Recommended PVC size: 100GB for 200-1000 sessions

#### Compute Impact
- Trace recording adds ~10-15% CPU overhead to browser automation
- Memory usage increase: ~200-500MB per session
- Network transfer: Additional ~50-200MB per session for trace download
- Minimal impact on research performance

#### Scalability Considerations
- Async trace processing to avoid blocking session completion
- Configurable trace retention policies
- Background cleanup jobs for expired traces
- CDN integration option for large-scale deployments

### Monitoring & Observability

#### Metrics
- Trace generation success/failure rates
- Average trace file sizes and generation times  
- Artifact storage utilization
- Trace viewer access patterns
- Session debugging frequency

#### Alerting
- Failed trace generation (>5% failure rate)
- Storage approaching capacity (>80% utilization)
- Trace viewer service availability
- Unusual artifact access patterns (security monitoring)

#### Logging
- Trace file lifecycle events (creation, access, deletion)
- Artifact serving requests with user context
- Trace viewer integration errors
- Storage cleanup operations


## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| **Large trace files impact performance** | High | Implement async processing, configurable quality settings, compression |
| **Storage costs escalate quickly** | Medium | Automatic cleanup policies, storage quotas, optional cloud storage tiers |
| **Trace viewer security vulnerabilities** | High | Regular updates, security scanning, restricted network access |
| **Browser automation overhead affects research** | Medium | Performance monitoring, optional disable flag, resource limits |
| **Sensitive data exposure in traces** | High | Data sanitization, access controls, encryption at rest |
| **Trace viewer service downtime** | Low | Multi-replica deployment, health checks, fallback to download |

## Implementation Timeline

### Phase 1: Foundation (2 weeks)
- Create trace viewer Docker image and service
- Set up PVC and storage infrastructure  
- Update CRD with artifact tracking fields
- Basic trace recording in claude-runner

### Phase 2: Integration (2 weeks)
- Backend artifact serving APIs
- Frontend trace viewer integration
- Session trace navigation UI
- Artifact metadata management

### Phase 3: Production Ready (1 week)
- Security hardening and access controls
- Performance optimization
- Monitoring and alerting setup
- Documentation and user guides

### Phase 4: Rollout (1 week)  
- Staging environment validation
- Production deployment with feature flags
- User training and feedback collection
- Performance monitoring and tuning

## Alternatives Considered

### Alternative 1: External Trace Viewer
**Approach:** Use Playwright's hosted trace.playwright.dev service
**Pros:** No infrastructure to maintain, always up-to-date
**Cons:** Security concerns uploading sensitive traces externally, dependency on external service
**Decision:** Rejected due to security and compliance requirements

### Alternative 2: Custom Trace Analysis
**Approach:** Build custom trace parsing and visualization
**Pros:** Full control, integrated UI experience  
**Cons:** Massive development effort, inferior to Playwright's mature viewer
**Decision:** Rejected - not worth reinventing sophisticated tooling

### Alternative 3: Video Recording Instead
**Approach:** Record video of browser sessions instead of traces
**Pros:** Easier to understand, smaller files potentially
**Cons:** No interactivity, no network analysis, larger files, no debugging capability
**Decision:** Rejected - traces provide much richer debugging information

### Alternative 4: Trace Export Only
**Approach:** Generate traces but only provide download, no embedded viewer
**Pros:** Simple implementation, lower resource usage
**Cons:** Poor user experience, requires local Playwright installation
**Decision:** Considered for Phase 1, but embedded viewer provides much better UX

## References

- [Playwright Trace Viewer Documentation](https://playwright.dev/docs/trace-viewer)
- [Microsoft Playwright MCP Server](https://github.com/microsoft/playwright-mcp) 
- [Playwright Trace Format Specification](https://github.com/microsoft/playwright/blob/main/packages/trace-viewer/README.md)
- [OpenShift Security Context Constraints](https://docs.openshift.com/container-platform/4.12/authentication/managing-security-context-constraints.html)
- [Kubernetes Persistent Volume Claims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)
