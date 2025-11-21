# Amber Issue-to-PR Workflow Diagrams

## High-Level Flow

```mermaid
graph TB
    A[Developer Creates Issue] --> B{Uses Template?}
    B -->|Yes| C[Template Auto-Labels]
    B -->|No| D[Manually Add Label]
    C --> E[GitHub Actions Triggered]
    D --> E
    E --> F[Amber Agent Executes]
    F --> G{Changes Made?}
    G -->|Yes| H[Create Feature Branch]
    G -->|No| I[Comment: No Changes Needed]
    H --> J[Run Linters & Tests]
    J --> K{All Pass?}
    K -->|Yes| L[Commit Changes]
    K -->|No| M[Comment: Errors Found]
    L --> N[Push Branch]
    N --> O[Create Pull Request]
    O --> P[Link PR to Issue]
    P --> Q[Team Review]
    Q --> R{Approved?}
    R -->|Yes| S[Merge PR]
    R -->|No| T[Request Changes]
    S --> U[Auto-Close Issue]
```

## Detailed Workflow Steps

```mermaid
sequenceDiagram
    participant Dev as Developer
    participant GH as GitHub
    participant GHA as GitHub Actions
    participant Amber as Amber Agent
    participant CI as CI/CD

    Dev->>GH: Create issue with label
    GH->>GHA: Trigger workflow (on: issues.labeled)
    GHA->>GHA: Extract issue details
    Note over GHA: Parse title, body, files
    GHA->>Amber: Execute with prompt
    Note over Amber: Analyze codebase
    Amber->>Amber: Apply fixes/refactoring
    Amber->>GHA: Return changes
    GHA->>GHA: Run linters (gofmt, black, etc.)
    alt Linters Pass
        GHA->>GHA: Run tests
        alt Tests Pass
            GHA->>GH: Create feature branch
            GHA->>GH: Commit changes
            GHA->>GH: Push branch
            GHA->>GH: Create pull request
            GH->>Dev: Notify: PR created
            Dev->>GH: Review PR
            GH->>CI: Run CI checks
            CI->>GH: Report results
            Dev->>GH: Approve & merge
            GH->>GH: Auto-close issue
        else Tests Fail
            GHA->>GH: Comment: Test failures
        end
    else Linters Fail
        GHA->>GH: Comment: Linting errors
    end
```

## Risk-Based Decision Tree

```mermaid
graph TD
    A[Amber Analyzes Issue] --> B{Risk Level?}
    B -->|Low Risk| C[Auto-Fix Category]
    B -->|Medium Risk| D[Proposal Category]
    B -->|High Risk| E[Report-Only Category]

    C --> C1{Changes Type}
    C1 -->|Formatting| C2[Run formatters: gofmt, black]
    C1 -->|Linting| C3[Run linters: golangci-lint]
    C1 -->|Imports| C4[Remove unused imports]
    C2 --> C5[Create PR]
    C3 --> C5
    C4 --> C5
    C5 --> C6[Require Review]
    C6 --> C7{Review Pass?}
    C7 -->|Yes| C8[Merge PR]
    C7 -->|No| C9[Close PR]

    D --> D1{Proposal Type}
    D1 -->|Refactoring| D2[Break files, extract patterns]
    D1 -->|Tests| D3[Add unit/contract tests]
    D1 -->|Error Handling| D4[Improve error handling]
    D2 --> D5[Create detailed PR]
    D3 --> D5
    D4 --> D5
    D5 --> D6[Require Approval]
    D6 --> D7{Approved?}
    D7 -->|Yes| D8[Merge PR]
    D7 -->|No| D9[Close PR]

    E --> E1{Report Type}
    E1 -->|Security| E2[Create security issue]
    E1 -->|Breaking Change| E3[Create RFC]
    E1 -->|Architecture| E4[Schedule review]
    E2 --> E5[Notify security team]
    E3 --> E5
    E4 --> E5
    E5 --> E6[Manual Decision Required]
```

## Label-Triggered Workflows

```mermaid
graph LR
    A[Issue Created] --> B{Label Applied}
    B -->|amber:auto-fix| C[Auto-Fix Workflow]
    B -->|amber:refactor| D[Refactor Workflow]
    B -->|amber:test-coverage| E[Test Coverage Workflow]
    B -->|Other labels| F[No Action]

    C --> C1[Parse issue for files]
    C1 --> C2[Run formatters/linters]
    C2 --> C3[Create PR with fixes]

    D --> D1[Parse current/desired state]
    D1 --> D2[Analyze code structure]
    D2 --> D3[Implement refactoring]
    D3 --> D4[Update tests]
    D4 --> D5[Create PR with changes]

    E --> E1[Parse untested code]
    E1 --> E2[Identify test scenarios]
    E2 --> E3[Generate tests]
    E3 --> E4[Verify tests pass]
    E4 --> E5[Create PR with tests]
```

## Comment-Triggered Workflow

```mermaid
graph TD
    A[Developer Comments on Issue] --> B{Contains /amber execute?}
    B -->|Yes| C[Extract Proposal from Issue Body]
    B -->|No| D[No Action]
    C --> E[Verify Approval in Issue]
    E --> F{Approved?}
    F -->|Yes| G[Execute Proposal]
    F -->|No| H[Comment: Requires Approval]
    G --> I[Apply Changes]
    I --> J[Create PR]
    J --> K[Link to Issue]
```

## Error Handling Flow

```mermaid
graph TD
    A[Amber Executes] --> B{Changes Applied?}
    B -->|No| C[No files matched pattern]
    C --> D[Comment: No changes needed]

    B -->|Yes| E{Linters Pass?}
    E -->|No| F[Linting errors found]
    F --> G[Comment: Linting failed]
    G --> H[Attach error logs]

    E -->|Yes| I{Tests Pass?}
    I -->|No| J[Test failures found]
    J --> K[Comment: Tests failed]
    K --> L[Attach test output]

    I -->|Yes| M{Create Branch?}
    M -->|Error| N[Git operation failed]
    N --> O[Comment: Branch creation error]

    M -->|Success| P{Push Branch?}
    P -->|Error| Q[Push failed]
    Q --> R[Comment: Push error]

    P -->|Success| S{Create PR?}
    S -->|Error| T[PR creation failed]
    T --> U[Comment: PR creation error]

    S -->|Success| V[Success!]
    V --> W[Comment: PR created]
```

## Multi-File Refactoring Example

```mermaid
graph LR
    A[sessions.go<br/>3495 lines] --> B[Amber Analyzes]
    B --> C{Extract Modules}

    C --> D[lifecycle.go<br/>CreateSession<br/>DeleteSession]
    C --> E[status.go<br/>UpdateStatus<br/>GetStatus]
    C --> F[jobs.go<br/>CreateJob<br/>MonitorJob]
    C --> G[validation.go<br/>ValidateSpec<br/>SanitizeInput]

    D --> H[Update imports]
    E --> H
    F --> H
    G --> H

    H --> I[Run tests]
    I --> J{All Pass?}
    J -->|Yes| K[Create PR]
    J -->|No| L[Rollback changes]
```

## Constitution Compliance Check

```mermaid
graph TD
    A[Amber Reviews Changes] --> B{Constitution Check}

    B --> C[Principle I: Kubernetes-Native]
    C --> D{Uses CRDs/Operators?}
    D -->|Yes| E1[✓ Pass]
    D -->|No| E2[✗ Fail: Report issue]

    B --> F[Principle II: Security-First]
    F --> G{User-scoped auth?}
    G -->|Yes| H1[✓ Pass]
    G -->|No| H2[✗ Fail: Add RBAC checks]

    B --> I[Principle III: Type Safety]
    I --> J{No context.TODO?}
    J -->|Yes| K1[✓ Pass]
    J -->|No| K2[✗ Fail: Fix context usage]

    B --> L[Principle IV: Testing]
    L --> M{Tests exist?}
    M -->|Yes| N1[✓ Pass]
    M -->|No| N2[✗ Fail: Add tests]

    B --> O[Principle V: Modularity]
    O --> P{Files < 400 lines?}
    P -->|Yes| Q1[✓ Pass]
    P -->|No| Q2[✗ Fail: Refactor required]

    E1 --> R[All Checks Complete]
    H1 --> R
    K1 --> R
    N1 --> R
    Q1 --> R
    R --> S{All Pass?}
    S -->|Yes| T[Create PR]
    S -->|No| U[Create proposal issue]
```

## Monitoring Dashboard (Conceptual)

```mermaid
graph TB
    subgraph Metrics
        A[PR Merge Rate<br/>90%+]
        B[Avg Time to Merge<br/>< 2 hours]
        C[Developer Time Saved<br/>10 hours/week]
        D[Issue Resolution<br/>95%]
    end

    subgraph Activity
        E[Auto-Fix PRs<br/>30/month]
        F[Refactoring PRs<br/>15/month]
        G[Test PRs<br/>5/month]
    end

    subgraph Health
        H[Success Rate<br/>85%]
        I[Error Rate<br/>10%]
        J[Skipped<br/>5%]
    end

    Metrics --> K[Overall Health: Excellent]
    Activity --> K
    Health --> K
```

## Integration Points

```mermaid
graph LR
    A[Amber Core] --> B[GitHub API]
    A --> C[Anthropic API]
    A --> D[Git Operations]
    A --> E[Testing Frameworks]
    A --> F[Linting Tools]

    B --> B1[Issues]
    B --> B2[Pull Requests]
    B --> B3[Comments]
    B --> B4[Labels]

    C --> C1[Claude Sonnet 4.5]
    C --> C2[Extended Thinking]

    D --> D1[Clone]
    D --> D2[Branch]
    D --> D3[Commit]
    D --> D4[Push]

    E --> E1[Go: go test]
    E --> E2[Python: pytest]
    E --> E3[TypeScript: jest]

    F --> F1[Go: gofmt, golangci-lint]
    F --> F2[Python: black, isort, flake8]
    F --> F3[TypeScript: eslint, prettier]
```

---

## Legend

- **Rectangle**: Process/Action
- **Diamond**: Decision Point
- **Oval**: Start/End State
- **Parallelogram**: Input/Output
- **Solid Arrow**: Sequential Flow
- **Dashed Arrow**: Conditional Flow
