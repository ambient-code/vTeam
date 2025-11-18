# Lab 1: Your First Agentic Session

## Objective 🎯

Learn to create and monitor an AgenticSession using the Ambient Code Platform web interface, understanding how AI-powered automation executes tasks in a Kubernetes-native environment.

**By the end of this lab, you will:**

- Successfully create an AgenticSession using the web UI
- Understand session configuration options (interactive vs headless, single vs multi-repo)
- Monitor real-time session execution and status
- Review session results and understand output artifacts

## Prerequisites 📋

- [ ] Ambient Code Platform installed and running ([Getting Started Guide](../../user-guide/getting-started.md))
- [ ] Anthropic API key configured in ProjectSettings
- [ ] At least one project created
- [ ] Web browser for accessing the platform interface
- [ ] Basic understanding of GitHub repositories (optional, for multi-repo exercises)

## Estimated Time ⏱️

**30-45 minutes** (including session execution time)

## Lab Scenario

You're a developer who wants to automate code analysis and documentation tasks. You'll create your first AgenticSession to analyze a simple Python repository and generate a README file describing its functionality.

## Step 1: Access the Platform Interface

1. **Ensure the platform is running**. For local development with OpenShift Local (CRC):

   ```bash
   cd platform
   make dev-start
   ```

2. **Get the frontend URL**:

   ```bash
   echo "https://$(oc get route vteam-frontend -n vteam-dev -o jsonpath='{.spec.host}')"
   ```

3. **Open your browser** to the frontend URL

4. **Verify the interface**:
   - You should see the dashboard
   - Navigate to your project (or create one if needed)
   - Look for the "Agentic Sessions" section

**✅ Checkpoint**: Confirm you can access the interface and see the sessions list.

## Step 2: Create Your First Session (Single Repository)

Let's start with a simple single-repository session.

1. **Click "Create Session"** or similar button in the UI

2. **Configure the session**:

   **Basic Settings:**
   - **Prompt**: `Analyze this Python repository and create a comprehensive README.md file`
   - **Repository URL**: `https://github.com/anthropics/anthropic-sdk-python` (or any small Python repo)
   - **Branch**: `main`
   - **Interactive Mode**: `false` (headless/batch mode)

   **Advanced Settings** (optional):
   - **Timeout**: `3600` (1 hour, default is fine)
   - **Model**: `claude-sonnet-4` (default)

3. **Click "Create Session"** to submit

4. **Observe the Kubernetes resources**:

   ```bash
   # Watch the AgenticSession Custom Resource
   oc get agenticsessions -n your-project-name -w

   # Watch the Job that gets created
   oc get jobs -n your-project-name -w

   # Watch the Pod executing Claude Code
   oc get pods -n your-project-name -w
   ```

**✅ Checkpoint**: You should see an AgenticSession CR created, followed by a Job and Pod spawning.

## Step 3: Monitor Session Execution

Real-time monitoring is crucial for understanding session progress.

### Via Web Interface

1. **Navigate to the session detail page** by clicking on your session
2. **Watch the status updates**:
   - `Pending`: Session created, waiting for Job
   - `Running`: Job pod is executing Claude Code
   - `Completed`: Session finished successfully
   - `Failed`: Session encountered an error

3. **View real-time logs** (if UI provides streaming)

### Via CLI

```bash
# Get session status
oc get agenticsession <session-name> -n <project-name> -o yaml

# Watch Job status
oc describe job <session-name> -n <project-name>

# Stream pod logs
oc logs -f job/<session-name> -n <project-name>
```

**Sample log output:**
```
Cloning repository https://github.com/anthropics/anthropic-sdk-python...
Running Claude Code CLI with prompt: Analyze this Python repository...
Claude: I'll analyze this repository structure...
Creating README.md with comprehensive documentation...
Session completed successfully.
```

**✅ Checkpoint**: Session should transition through Pending → Running → Completed within 5-10 minutes.

## Step 4: Review Session Results

Once the session completes, examine the results.

1. **Check the session status** in the UI:
   - Look for completion timestamp
   - Check for any error messages
   - Review execution summary

2. **View the output** (if repository forking is enabled):
   - A pull request may be created with the generated README.md
   - Or changes may be pushed to the output repository

3. **Inspect the AgenticSession CR**:

   ```bash
   oc get agenticsession <session-name> -n <project-name> -o jsonpath='{.status}' | jq
   ```

   **Expected status fields:**
   ```json
   {
     "phase": "Completed",
     "startTime": "2025-10-30T10:00:00Z",
     "completionTime": "2025-10-30T10:08:32Z",
     "results": "Successfully created README.md with 250 lines...",
     "repos": [
       {
         "url": "https://github.com/anthropics/anthropic-sdk-python",
         "status": "pushed"
       }
     ]
   }
   ```

**✅ Checkpoint**: Session status should show "Completed" with results summary.

## Step 5: Create an Interactive Session

Interactive sessions allow back-and-forth conversation with Claude Code.

1. **Create a new session** with these settings:
   - **Prompt**: `Help me refactor this Python codebase for better maintainability`
   - **Repository**: Same as before
   - **Interactive Mode**: `true`

2. **Understand interactive mode**:
   - Session runs indefinitely until you signal completion
   - Uses inbox/outbox files for asynchronous communication
   - Allows multi-turn conversations

3. **Interact with the session**:

   ```bash
   # Write to inbox file (send message to Claude)
   oc exec -it <session-pod-name> -n <project-name> -- \
     bash -c 'echo "Focus on the authentication module first" > /workspace/inbox.txt'

   # Read from outbox file (get Claude's response)
   oc exec -it <session-pod-name> -n <project-name> -- \
     cat /workspace/outbox.txt
   ```

4. **Stop the interactive session** when done:
   - Write `EXIT` to inbox.txt
   - Or delete the AgenticSession CR

**✅ Checkpoint**: Interactive session should remain in "Running" state until you signal completion.

## Step 6: Multi-Repository Session (Advanced)

The Ambient Code Platform supports operating on multiple repositories simultaneously.

1. **Create a multi-repo session**:

   **Prompt**: `Compare the API design patterns in these two SDK repositories and create a best practices document`

   **Repositories**:
   - Repo 1 (main workspace):
     - URL: `https://github.com/anthropics/anthropic-sdk-python`
     - Branch: `main`
   - Repo 2 (reference):
     - URL: `https://github.com/anthropics/anthropic-sdk-typescript`
     - Branch: `main`

   **Main Repo Index**: `0` (Python SDK is the working directory)

2. **Understand multi-repo behavior**:
   - All repos are cloned to the workspace
   - `mainRepoIndex` specifies which repo Claude works in
   - Claude can reference and analyze all repos
   - Changes are typically made to the main repo

3. **Review per-repo status**:

   ```bash
   oc get agenticsession <session-name> -n <project-name> -o jsonpath='{.status.repos}' | jq
   ```

   **Expected output:**
   ```json
   [
     {
       "url": "https://github.com/anthropics/anthropic-sdk-python",
       "status": "pushed"
     },
     {
       "url": "https://github.com/anthropics/anthropic-sdk-typescript",
       "status": "abandoned"
     }
   ]
   ```

**✅ Checkpoint**: Multi-repo session should successfully clone and analyze multiple repositories.

## Validation & Testing

### Test Your Understanding

**Question 1**: What are the two session modes, and when would you use each?
- **Headless (interactive: false)**: Single-prompt execution with timeout, ideal for batch tasks
- **Interactive (interactive: true)**: Long-running chat sessions, ideal for iterative development

**Question 2**: What Kubernetes resources are created when you submit an AgenticSession?
- AgenticSession Custom Resource
- Job (managed by the Operator)
- Pod (executes Claude Code runner)
- Secret (for API keys, via ProjectSettings)
- PersistentVolumeClaim (workspace storage)

**Question 3**: How can you tell if a session completed successfully?
- Status phase is "Completed"
- No error messages in status
- Completion timestamp is set
- Results field contains summary

### Verify Session Quality

A successful AgenticSession should have:

- [ ] **Valid Custom Resource** with spec and status fields
- [ ] **Job completion** without errors
- [ ] **Results summary** in status.results
- [ ] **Proper phase transition** (Pending → Running → Completed)
- [ ] **Per-repo status** showing push/abandon decisions

## Troubleshooting 🛠️

### Session Stuck in Pending

- **Cause**: Operator not running or Job creation failed
- **Solution**: Check operator logs and RBAC permissions
  ```bash
  oc logs deployment/vteam-operator -n vteam-dev
  oc describe job <session-name> -n <project-name>
  ```

### Session Fails Immediately

- **Cause**: Invalid API key, repository access issues, or misconfigured ProjectSettings
- **Solution**: Verify API key in Secret and check pod logs
  ```bash
  oc get secret runner-secrets -n <project-name> -o yaml
  oc logs job/<session-name> -n <project-name>
  ```

### Pod ImagePullBackOff

- **Cause**: Container image not accessible or wrong registry
- **Solution**: Verify image tag and registry permissions
  ```bash
  oc describe pod <pod-name> -n <project-name>
  oc get pods -n <project-name> -o jsonpath='{.items[*].spec.containers[*].image}'
  ```

### Session Timeout

- **Cause**: Task took longer than configured timeout
- **Solution**: Increase timeout value or simplify prompt
  ```yaml
  spec:
    timeout: 7200  # 2 hours
  ```

## Key Learnings 📚

After completing this lab, you should understand:

1. **AgenticSession Lifecycle**: How sessions are created, executed, and completed
2. **Kubernetes Integration**: How the platform uses CRs, Operators, and Jobs
3. **Session Modes**: When to use interactive vs headless execution
4. **Multi-Repo Support**: How to work with multiple repositories simultaneously
5. **Monitoring**: How to track session progress via UI and CLI

## Further Exploration 🔍

Ready to dig deeper?

- **Try complex prompts**: Multi-step refactoring or feature implementation
- **Experiment with timeouts**: Find optimal values for different task types
- **Explore multi-repo workflows**: Cross-repository analysis and migration
- **Customize ProjectSettings**: Configure default models, timeouts, and API keys
- **Review CLAUDE.md**: Understand the complete AgenticSession specification

## Success Criteria ✅

You've successfully completed Lab 1 when:

- [ ] Created at least one successful AgenticSession
- [ ] Monitored session execution via UI and CLI
- [ ] Understood the difference between interactive and headless modes
- [ ] Reviewed session results and status
- [ ] Can explain how the platform uses Kubernetes resources

**Congratulations!** You've mastered the fundamentals of the Ambient Code Platform's AgenticSession workflow. You're now ready to automate development tasks using AI-powered agents in a Kubernetes-native environment.

---

**Next Steps**: Explore advanced configuration options in the [User Guide](../../user-guide/getting-started.md) or dive into the [Reference Documentation](../../reference/index.md) to understand all AgenticSession capabilities.
