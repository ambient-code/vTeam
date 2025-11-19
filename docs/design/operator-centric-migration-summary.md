# Operator-Centric Migration Summary

This document captures the user-facing implications of the operator-centric migration now that implementation is complete.

## Status & Conditions

- `AgenticSessionStatus` now exposes structured `conditions[]` instead of a plain `phase/message/is_error` trio.
- Key conditions include `PVCReady`, `SecretsReady`, `JobCreated`, `RunnerStarted`, `ReposReconciled`, `WorkflowReconciled`, `Ready`, `Completed`, and `Failed`.
- `observedGeneration`, `startTime`, `completionTime`, `runnerPodName`, `reconciledRepos`, and `reconciledWorkflow` provide declarative insight into reconciliation state.
- The frontend shows these conditions inside the session details modal so users can see why the operator is waiting.

## CRD & Spec Changes

- `spec.prompt` has been renamed to `spec.initialPrompt`. The operator injects it into the runner as `INITIAL_PROMPT`.
- Backend endpoints that mutate spec now return the updated session object and no longer fire WebSocket control messages.
- `StopSession` simply marks the session as `Stopped`; the operator handles cleanup and restarts.

## Runner & Exit Codes

- The runner (`wrapper.py`) no longer patches the CR status. Instead it exits with:
  - `0` – session completed successfully
  - `1` – runtime error/SDK failure
  - `2` – prerequisite validation failure (e.g. missing `spec.md`)
- Operator exit-code handling maps those to the appropriate `Completed` / `Failed` conditions with detailed reasons.
- Runner-per-session RBAC no longer grants `agenticsessions/status` access; the service account can only read/update the CR spec (annotations).

## Removed/Deprecated API Endpoints

The following backend endpoints have been removed:

- `PUT /agentic-sessions/:id/status`
- `POST /agentic-sessions/:id/spawn-content-pod`
- `GET /agentic-sessions/:id/content-pod-status`
- `DELETE /agentic-sessions/:id/content-pod`

Existing clients must rely on the operator and the new condition-driven status model rather than direct pod spawning or runner-driven status updates.

## Frontend UX Updates

- Session details now show condition history and disable spec editing while a session is running.
- Workspace tab messaging no longer references the deleted temp content pod flow and instead reflects operator-driven availability.

## Testing

- Backend and operator packages: `go test ./components/backend/...` and `go test ./components/operator/...`
- Frontend lint: `npm run lint`
- Runner syntax check: `python3 -m compileall components/runners/claude-code-runner/wrapper.py` (pytest not available in the runner image)

This summary should be referenced when upgrading existing clusters so that operators and application teams are aware of the new declarative workflow and the removal of runner-managed status updates.

