# Quickstart: LangGraph Workflow Integration

**Feature**: LangGraph Workflow Integration
**Estimated Time**: 30 minutes
**Prerequisites**: Cluster administrator access for workflow registration, project member access for session execution

This guide demonstrates the complete workflow lifecycle: registering a workflow definition, creating a session, monitoring progress, and interacting with human-in-the-loop prompts.

---

## Table of Contents

1. [Setup](#setup)
2. [Register a Workflow](#step-1-register-a-workflow)
3. [Create a Workflow Session](#step-2-create-a-workflow-session)
4. [Monitor Real-Time Progress](#step-3-monitor-real-time-progress)
5. [Respond to Interactive Prompts](#step-4-respond-to-interactive-prompts)
6. [View Results](#step-5-view-results)
7. [Resume an Interrupted Session](#step-6-resume-an-interrupted-session)
8. [Verify Unified Session List](#step-7-verify-unified-session-list)
9. [Cleanup](#cleanup)
10. [Troubleshooting](#troubleshooting)

---

## Setup

### Prerequisites Check

```bash
# Verify cluster access
kubectl auth can-i create workflowdefinitions.vteam.ambient-code --all-namespaces
# Expected output: yes (cluster admin only)

# Verify project access
kubectl auth can-i create workflowsessions --namespace=data-team
# Expected output: yes (project edit permission)

# Set environment variables
export BACKEND_URL="https://api.ambient-code.example.com"
export TOKEN=$(oc whoami -t)  # Or your bearer token
export PROJECT_NAME="data-team"
```

### Sample Workflow

For this quickstart, we'll use a pre-built CSV forecast analyzer workflow.

**Container Image**: `quay.io/ambient_code/workflows/csv-forecast-analyzer:v1.0.0`

**Input Schema**:
- `csv_url` (required): URL to CSV file
- `forecast_column` (required): Column name to forecast
- `forecast_periods` (optional): Number of periods (default: 30)
- `confidence_interval` (optional): Confidence level (default: 0.95)

**Workflow Behavior**:
1. Load CSV file from URL
2. Detect statistical outliers
3. **Pause for human approval** of outlier removal
4. Generate forecasts with approved parameters
5. Return forecast data and model metrics

---

## Step 1: Register a Workflow

### Via UI (Recommended)

1. Navigate to **Workflows** page (cluster-wide registry)
2. Click **Register New Workflow**
3. Fill in the form:
   - **Name**: `csv-forecast-analyzer`
   - **Display Name**: "CSV Forecast Analyzer"
   - **Description**: "Analyze CSV data and generate forecasts with human-in-the-loop validation"
   - **Container Image**: `quay.io/ambient_code/workflows/csv-forecast-analyzer:v1.0.0`
   - **Version**: `v1.0.0`
   - **Tags**: `data-analysis`, `forecasting`, `interactive`
4. Paste the **Input Schema** (see below)
5. Click **Register**

**Input Schema (JSON Schema):**

```json
{
  "type": "object",
  "required": ["csv_url", "forecast_column"],
  "properties": {
    "csv_url": {
      "type": "string",
      "format": "uri",
      "title": "CSV File URL",
      "description": "URL to the CSV file to analyze"
    },
    "forecast_column": {
      "type": "string",
      "title": "Column to Forecast",
      "description": "Name of the column containing values to forecast"
    },
    "forecast_periods": {
      "type": "integer",
      "minimum": 1,
      "maximum": 365,
      "default": 30,
      "title": "Forecast Periods",
      "description": "Number of periods to forecast"
    },
    "confidence_interval": {
      "type": "number",
      "minimum": 0.01,
      "maximum": 0.99,
      "default": 0.95,
      "title": "Confidence Interval",
      "description": "Confidence level for forecast intervals"
    }
  }
}
```

### Via API (Alternative)

```bash
curl -X POST "$BACKEND_URL/api/workflows" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "csv-forecast-analyzer",
    "displayName": "CSV Forecast Analyzer",
    "description": "Analyze CSV data and generate forecasts with human-in-the-loop validation",
    "containerImage": "quay.io/ambient_code/workflows/csv-forecast-analyzer:v1.0.0",
    "version": "v1.0.0",
    "tags": ["data-analysis", "forecasting", "interactive"],
    "inputSchema": {
      "type": "object",
      "required": ["csv_url", "forecast_column"],
      "properties": {
        "csv_url": {
          "type": "string",
          "format": "uri",
          "title": "CSV File URL"
        },
        "forecast_column": {
          "type": "string",
          "title": "Column to Forecast"
        },
        "forecast_periods": {
          "type": "integer",
          "minimum": 1,
          "maximum": 365,
          "default": 30,
          "title": "Forecast Periods"
        },
        "confidence_interval": {
          "type": "number",
          "minimum": 0.01,
          "maximum": 0.99,
          "default": 0.95,
          "title": "Confidence Interval"
        }
      }
    }
  }'
```

**Expected Response:**

```json
{
  "message": "Workflow definition registered successfully",
  "name": "csv-forecast-analyzer"
}
```

### Verification

```bash
# List workflows
curl -X GET "$BACKEND_URL/api/workflows" \
  -H "Authorization: Bearer $TOKEN"

# Get specific workflow
curl -X GET "$BACKEND_URL/api/workflows/csv-forecast-analyzer" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected**: Workflow appears with `status.phase: Active`

---

## Step 2: Create a Workflow Session

### Via UI (Recommended)

1. Navigate to **Projects** → **data-team** → **Sessions**
2. Click **New Workflow Session**
3. Select **csv-forecast-analyzer** from dropdown
4. **Dynamically generated form** appears based on input schema
5. Fill in the form:
   - **Session Name**: `sales-forecast-2025-11`
   - **CSV File URL**: `https://storage.example.com/sales_2024.csv`
   - **Column to Forecast**: `revenue`
   - **Forecast Periods**: `30` (default)
   - **Confidence Interval**: `0.95` (default)
6. Click **Create Session**

**Expected**: Redirect to session detail page, status shows `pending`

### Via API (Alternative)

```bash
curl -X POST "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "sessionName": "sales-forecast-2025-11",
    "workflowName": "csv-forecast-analyzer",
    "inputData": {
      "csv_url": "https://storage.example.com/sales_2024.csv",
      "forecast_column": "revenue",
      "forecast_periods": 30,
      "confidence_interval": 0.95
    }
  }'
```

**Expected Response:**

```json
{
  "message": "Workflow session created successfully",
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "sessionName": "sales-forecast-2025-11"
}
```

### What Happens Behind the Scenes

1. Backend validates `inputData` against workflow's `inputSchema`
2. Database record created in `workflow_sessions` table with status `pending`
3. Operator detects new session and spawns Kubernetes Job
4. LangGraph runner pod starts executing workflow
5. Status updates to `running` within ~15 seconds

---

## Step 3: Monitor Real-Time Progress

### Via UI (Recommended)

1. On the **Session Detail** page, observe:
   - **Status Badge**: Changes from `pending` → `running`
   - **Progress Bar**: Updates as workflow progresses
   - **Message Feed**: Real-time messages from workflow

**Example Messages:**

```
[10:00:15] System: Session started
[10:00:20] Workflow: Loading CSV file from storage (25% complete)
[10:00:45] Workflow: Processing 4,000 rows...
[10:01:30] Workflow: Detected 3 statistical outliers
[10:02:00] Workflow: Waiting for approval...
```

2. **Status Badge** changes to `waiting_for_input` (yellow)

### Via WebSocket (For Developers)

```javascript
const ws = new WebSocket(
  `wss://api.ambient-code.example.com/ws?sessionName=sales-forecast-2025-11&token=${token}`
);

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(`[${message.type}]`, message.data);
};

// Example output:
// [workflow_progress] {step: "load_data", progress: 0.25, message: "Loading CSV..."}
// [workflow_progress] {step: "detect_outliers", progress: 0.50, message: "Analyzing..."}
// [workflow_waiting_for_input] {prompt: "Found 3 outliers. Remove?", options: [...]}
```

### Via API (Polling)

```bash
# Get session status
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11" \
  -H "Authorization: Bearer $TOKEN"

# Get messages
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11/messages" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Step 4: Respond to Interactive Prompts

When the workflow detects outliers, it pauses and waits for human approval.

### Via UI (Recommended)

1. **Prompt appears** in the session detail page:
   ```
   ⏸️ Workflow is waiting for your input

   Found 3 statistical outliers. Remove them?

   Outliers:
   - 2024-01-15: $250,000 (2.5σ above mean)
   - 2024-06-22: $180,000 (2.1σ above mean)
   - 2024-12-03: $320,000 (3.0σ above mean)
   ```

2. **Response Options**: [Approve] [Reject]

3. Click **Approve**

4. **Expected**:
   - Prompt disappears
   - Status badge changes back to `running`
   - Workflow resumes execution

### Via API (Alternative)

```bash
curl -X POST "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "response": {
      "approval": "approved",
      "comment": "Outliers look correct to remove"
    }
  }'
```

**Expected Response:**

```json
{
  "message": "User input received, session resuming"
}
```

### What Happens Behind the Scenes

1. User response stored in `session_messages` table
2. LangGraph checkpointer has already saved workflow state
3. Runner receives user input (via polling or callback)
4. Workflow resumes from checkpoint with user's decision
5. Execution continues to completion

---

## Step 5: View Results

After workflow completes (~2-3 minutes total):

### Via UI (Recommended)

1. **Status Badge** changes to `completed` (green)
2. **Completion Time** displayed: "Completed in 3m 15s"
3. **Output Section** appears with results:

```json
{
  "forecast": [
    {
      "date": "2025-12-01",
      "predicted": 125000,
      "ci_lower": 118000,
      "ci_upper": 132000
    },
    {
      "date": "2025-12-02",
      "predicted": 127000,
      "ci_lower": 120000,
      "ci_upper": 134000
    },
    ...
  ],
  "model_metrics": {
    "mae": 2500,
    "rmse": 3200,
    "mape": 0.02
  }
}
```

4. **Download Options**: Export as JSON, CSV, or Chart

### Via API

```bash
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.outputData'
```

---

## Step 6: Resume an Interrupted Session

Scenario: You start a workflow, it waits for input, but you leave for the day and return tomorrow.

### Simulate Interruption

1. Create a new session: `sales-forecast-2025-12`
2. Wait for it to reach `waiting_for_input` status
3. **Close browser tab** (or disconnect)
4. **Wait 24 hours** (or just 5 minutes for testing)

### Resume Session

1. Navigate to **Projects** → **data-team** → **Sessions**
2. Find session `sales-forecast-2025-12`
3. **Status shows**: `waiting_for_input` (still)
4. Click session name to open detail page
5. **Prompt is still visible** (loaded from database)
6. Respond to prompt as in [Step 4](#step-4-respond-to-interactive-prompts)
7. Workflow resumes execution

### Verification

```bash
# Check session status (should be waiting_for_input)
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-12" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.status'

# Expected: "waiting_for_input"

# Get checkpoint count (verify checkpoint was saved)
psql -h $POSTGRES_HOST -U $POSTGRES_USER -d ambient_code -c \
  "SELECT COUNT(*) FROM checkpoints WHERE thread_id = 'project-data-team:session-sales-forecast-2025-12';"

# Expected: >= 1
```

**Key Point**: Checkpoint persistence enables seamless resumption even after days of inactivity.

---

## Step 7: Verify Unified Session List

Ensure both Claude Code sessions and workflow sessions appear in the same UI.

### Via UI

1. Navigate to **Projects** → **data-team** → **Sessions**
2. **Expected View**:

```
┌────────────────────────────────────────────────────────────────┐
│ Sessions                                            [New Session ▼] │
├────────────────────────────────────────────────────────────────┤
│ 🔷 Claude Code Session                                         │
│   fix-authentication-bug                                       │
│   Status: completed  •  2 hours ago                            │
│   [View]                                                       │
├────────────────────────────────────────────────────────────────┤
│ 🟦 Workflow Session                                            │
│   sales-forecast-2025-11  (csv-forecast-analyzer)              │
│   Status: completed  •  30 minutes ago                         │
│   [View]                                                       │
├────────────────────────────────────────────────────────────────┤
│ 🟦 Workflow Session                                            │
│   sales-forecast-2025-12  (csv-forecast-analyzer)              │
│   Status: waiting_for_input  •  5 minutes ago                  │
│   [Continue]                                                   │
└────────────────────────────────────────────────────────────────┘
```

3. **Verify**:
   - Different icons/badges for session types
   - Workflow sessions show workflow name in parentheses
   - Appropriate actions: "View" (completed), "Continue" (waiting)

### Via API

```bash
# List all sessions (both types)
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/sessions" \
  -H "Authorization: Bearer $TOKEN"

# Expected: Mixed array of AgenticSession and WorkflowSession objects
# with `sessionType` field distinguishing them
```

---

## Cleanup

### Delete Workflow Sessions

```bash
# Via API
curl -X DELETE "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11" \
  -H "Authorization: Bearer $TOKEN"

curl -X DELETE "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-12" \
  -H "Authorization: Bearer $TOKEN"
```

**Expected**:
- Session records deleted from database
- Associated messages and checkpoints deleted (cascade)
- If session was running, Kubernetes Job is cancelled

### Delete Workflow Definition (Optional)

**Note**: Cannot delete if active sessions exist.

```bash
# Verify no active sessions
curl -X GET "$BACKEND_URL/api/workflows/csv-forecast-analyzer" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.status.activeSessions'

# Expected: 0

# Delete workflow
curl -X DELETE "$BACKEND_URL/api/workflows/csv-forecast-analyzer" \
  -H "Authorization: Bearer $TOKEN"
```

---

## Troubleshooting

### Session Stuck in `pending` Status

**Symptoms**: Session created but never transitions to `running`

**Possible Causes**:
1. Operator not running or not watching workflow sessions
2. Image pull failure (invalid registry or credentials)
3. Resource quota exceeded in project namespace

**Debug Steps**:

```bash
# Check operator logs
kubectl logs -n vteam-system deployment/vteam-operator --tail=50

# Check for job creation
kubectl get jobs -n $PROJECT_NAME | grep sales-forecast-2025-11

# Describe job to see image pull errors
kubectl describe job <job-name> -n $PROJECT_NAME

# Check resource quota
kubectl describe resourcequota -n $PROJECT_NAME
```

**Solution**:
- Fix image reference in workflow definition
- Request quota increase from cluster admin
- Restart operator if needed

### Workflow Fails with "Input Validation Error"

**Symptoms**: Session immediately transitions to `failed` with error message

**Cause**: Input data doesn't match workflow's input schema

**Debug Steps**:

```bash
# Get error message
curl -X GET "$BACKEND_URL/api/projects/$PROJECT_NAME/workflow-sessions/sales-forecast-2025-11" \
  -H "Authorization: Bearer $TOKEN" \
  | jq '.errorMessage'

# Example: "Property 'forecast_column' is required"
```

**Solution**:
- Delete failed session
- Create new session with corrected input data
- Use UI form (auto-validates) instead of API

### WebSocket Connection Fails

**Symptoms**: No real-time updates, must refresh page

**Possible Causes**:
1. Invalid or expired bearer token
2. Firewall blocking WebSocket protocol
3. Backend WebSocket service not running

**Debug Steps**:

```bash
# Test WebSocket connection
wscat -c "wss://api.ambient-code.example.com/ws?sessionName=test&token=$TOKEN"

# Expected: Connection established
# Error: 401 Unauthorized or connection refused
```

**Solution**:
- Refresh token: `export TOKEN=$(oc whoami -t)`
- Check network/firewall rules
- Contact platform administrator

### Output Data Exceeds 100MB Limit

**Symptoms**: Session fails with "output data exceeds 100MB limit"

**Cause**: Workflow produced results larger than database constraint

**Solution**:
- Workflow should store large outputs externally (S3, object storage)
- Only store URLs/metadata in `outputData`
- Example:

```python
# In workflow runner
if len(results) > 90 * 1024 * 1024:  # 90MB safety margin
    url = await upload_to_s3(results)
    output_data = {
        "result_url": url,
        "size_bytes": len(results),
        "summary": extract_summary(results)
    }
else:
    output_data = results
```

---

## Success Criteria

✅ **Workflow Registered**: Appears in cluster-wide registry
✅ **Session Created**: Within 30 seconds of form submission
✅ **Real-Time Updates**: Progress messages appear within 5 seconds
✅ **Interactive Approval**: Prompt appears, response resumes workflow
✅ **Session Resumed**: Works after hours/days of inactivity
✅ **Results Displayed**: Output data visible on completion
✅ **Unified Session List**: Both session types visible with clear labels
✅ **Zero Disruption**: Existing Claude Code sessions unaffected

**Time to First Session**: Target < 30 minutes ✅

---

## Next Steps

- **Build Custom Workflow**: Follow [Workflow Development Guide](#) (future docs)
- **Integrate with CI/CD**: Auto-register workflows from GitOps pipeline
- **Monitor Performance**: Set up metrics dashboards for session execution
- **Scale Testing**: Create 10+ concurrent sessions to test resource limits

---

**Feedback**: Report issues or suggestions at https://github.com/ambient-code/platform/issues
