# Architectural Decision Records (ADRs)

This directory contains Architectural Decision Records (ADRs) documenting significant architectural decisions made for the Ambient Code Platform.

## What is an ADR?

An ADR captures:

- **Context:** What problem were we solving?
- **Options:** What alternatives did we consider?
- **Decision:** What did we choose and why?
- **Consequences:** What are the trade-offs?

ADRs are immutable once accepted. If a decision changes, we create a new ADR that supersedes the old one.

## When to Create an ADR

Create an ADR for decisions that:

- Affect the overall architecture
- Are difficult or expensive to reverse
- Impact multiple components or teams
- Involve significant trade-offs
- Will be questioned in the future ("Why did we do it this way?")

**Examples:**

- Choosing a programming language or framework
- Selecting a database or messaging system
- Defining authentication/authorization approach
- Establishing API design patterns
- Multi-tenancy architecture decisions

**Not ADR-worthy:**

- Trivial implementation choices
- Decisions easily reversed
- Component-internal decisions with no external impact

## ADR Workflow

1. **Propose:** Copy `template.md` to `NNNN-title.md` with status "Proposed"
2. **Discuss:** Share with team, gather feedback
3. **Decide:** Update status to "Accepted" or "Rejected"
4. **Implement:** Reference ADR in PRs
5. **Learn:** Update "Implementation Notes" with gotchas discovered

## ADR Status Meanings

- **Proposed:** Decision being considered, open for discussion
- **Accepted:** Decision made and being implemented
- **Deprecated:** Decision no longer relevant but kept for historical context
- **Superseded by ADR-XXXX:** Decision replaced by a newer ADR

## Current ADRs

| ADR | Title | Status | Date |
|-----|-------|--------|------|
| [0001](0001-kubernetes-native-architecture.md) | Kubernetes-Native Architecture | Accepted | 2024-11-21 |
| [0002](0002-user-token-authentication.md) | User Token Authentication for API Operations | Accepted | 2024-11-21 |
| [0003](0003-multi-repo-support.md) | Multi-Repository Support in AgenticSessions | Accepted | 2024-11-21 |
| [0004](0004-go-backend-python-runner.md) | Go Backend with Python Claude Runner | Accepted | 2024-11-21 |
| [0005](0005-nextjs-shadcn-react-query.md) | Next.js with Shadcn UI and React Query | Accepted | 2024-11-21 |

## References

- [ADR GitHub Organization](https://adr.github.io/) - ADR best practices
- [Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions) - Original proposal by Michael Nygard
