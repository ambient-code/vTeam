# Vertex AI Setup Guide for RFE Builder

This guide covers the complete setup process for using the RFE Builder with Google Cloud Vertex AI.

## Prerequisites

- Google Cloud Project with billing enabled
- `gcloud` CLI installed and configured
- Python 3.10+ environment

## Step 1: Google Cloud Project Setup

### 1.1 Enable Required APIs
```bash
# Enable Vertex AI API
gcloud services enable aiplatform.googleapis.com

# Enable Compute Engine API (required for some Vertex AI operations)
gcloud services enable compute.googleapis.com

# Verify APIs are enabled
gcloud services list --enabled --filter="name:(aiplatform.googleapis.com OR compute.googleapis.com)"
```

### 1.2 Set Default Project
```bash
# Set your project as default
gcloud config set project YOUR_PROJECT_ID

# Verify configuration
gcloud config get-value project
```

## Step 2: Authentication Setup

### Option A: Application Default Credentials (Recommended - Most Secure)
```bash
# Login and set application default credentials
gcloud auth application-default login

# This creates credentials at:
# ~/.config/gcloud/application_default_credentials.json
```

**🔒 Security Note**: Application Default Credentials are the recommended approach as they:
- Don't require downloading service account keys to local files
- Automatically rotate and are managed by Google Cloud
- Reduce risk of accidental credential exposure

### Option B: Service Account (For Production Environments Only)
```bash
# Create a service account
gcloud iam service-accounts create vertex-ai-demo \
    --display-name="Vertex AI Demo Service Account"

# Grant required permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:vertex-ai-demo@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/aiplatform.user"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:vertex-ai-demo@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/ml.developer"

# Create and download key
gcloud iam service-accounts keys create vertex-ai-key.json \
    --iam-account=vertex-ai-demo@YOUR_PROJECT_ID.iam.gserviceaccount.com

# Set environment variable
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/vertex-ai-key.json"
```

**⚠️ SECURITY WARNING**: 
- **NEVER commit service account keys to version control**
- Store keys securely and rotate them regularly
- Use Application Default Credentials (Option A) whenever possible
- Add `*.json` to your `.gitignore` to prevent accidental commits

## Step 3: Required IAM Permissions

Ensure your user/service account has these roles:

```bash
# Check current permissions
gcloud projects get-iam-policy YOUR_PROJECT_ID \
    --filter="bindings.members:YOUR_EMAIL_OR_SERVICE_ACCOUNT"

# Add required roles if missing
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="user:YOUR_EMAIL" \
    --role="roles/aiplatform.user"

gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="user:YOUR_EMAIL" \
    --role="roles/ml.developer"
```

**Required Roles:**
- `roles/aiplatform.user` - Access to Vertex AI services
- `roles/ml.developer` - Access to ML models and endpoints
- `roles/serviceusage.serviceUsageConsumer` - API usage permissions

## Step 4: Verify Claude Model Access

### 4.1 Check Model Availability
```bash
# List available models in your region
gcloud ai models list --region=us-east5 --filter="displayName:claude" --limit=10

# If no models are listed, try other regions:
gcloud ai models list --region=us-central1 --filter="displayName:claude" --limit=10
gcloud ai models list --region=europe-west1 --filter="displayName:claude" --limit=10
```

### 4.2 Test Model Access
```bash
# Note: Direct model testing via gcloud is complex for Claude models
# The RFE Builder demo will automatically test model access when you run it
# If you need to test manually, use the check_vertex_setup.py script instead
```

## Step 5: Environment Variables Setup

Create or update your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
# Required environment variables
export CLAUDE_CODE_USE_VERTEX=1
export CLOUD_ML_REGION=us-east5  # or your preferred region
export ANTHROPIC_VERTEX_PROJECT_ID=your-project-id

# Supported models (choose appropriate ones)
export ANTHROPIC_MODEL='claude-sonnet-4@20250514'
export ANTHROPIC_SMALL_FAST_MODEL='claude-3-5-haiku@20241022'

# Optional: Configure timeouts and retry behavior  
export ANTHROPIC_TIMEOUT=30.0
export ANTHROPIC_MAX_RETRIES=3

# Optional: If using Vertex AI, you do NOT need an Anthropic API key.
# Leave ANTHROPIC_API_KEY unset. If present in secrets, it will be ignored when
# CLAUDE_CODE_USE_VERTEX=1 (you may also set ANTHROPIC_API_KEY="using-vertex-ai").

# Reload your shell configuration
source ~/.bashrc  # or ~/.zshrc
```

## Step 6: Install Dependencies

```bash
# Install all required dependencies
uv pip install -r requirements.txt

# Or manually install Vertex AI dependencies
uv pip install "anthropic[vertex]" google-cloud-aiplatform google-auth
```

## Step 7: Verification

### 7.1 Test Authentication
```bash
# Verify authentication works
gcloud auth list
gcloud auth application-default print-access-token
```

### 7.2 Test Python Imports
```python
# Test in Python
python3 -c "
import google.auth
from anthropic import AnthropicVertex
print('✅ All imports successful')

# Test client creation
try:
    client = AnthropicVertex(project_id='your-project', region='us-east5')
    print('✅ AnthropicVertex client created')
except Exception as e:
    print(f'⚠️  Client creation issue: {e}')
"
```

## Common Issues and Solutions

### Issue 1: "No module named 'google.auth'"
**Solution:** Install missing dependencies
```bash
uv pip install google-auth google-cloud-aiplatform
```

### Issue 2: "Default credentials not found"
**Solution:** Set up authentication
```bash
gcloud auth application-default login
```

### Issue 3: "Permission denied" or "Forbidden"
**Solution:** Check IAM permissions
```bash
# Verify you have required roles
gcloud projects get-iam-policy YOUR_PROJECT_ID --filter="bindings.members:YOUR_EMAIL"
```

### Issue 4: "Model not found" or "Region not supported"
**Solution:** Check model availability in your region
```bash
# Try different regions
gcloud ai models list --region=us-central1 --filter="displayName:claude"
```

### Issue 5: "Quota exceeded"
**Solution:** Check and request quota increases
```bash
# Check current quotas
gcloud compute project-info describe --project=YOUR_PROJECT_ID
```

## Environment Variables Reference

| Variable | Required | Description | Example |
|----------|----------|-------------|---------|
| `CLAUDE_CODE_USE_VERTEX` | Yes | Enable Vertex AI mode | `1` |
| `ANTHROPIC_VERTEX_PROJECT_ID` | Yes | Your GCP project ID | `my-project-12345` |
| `CLOUD_ML_REGION` | Yes | Vertex AI region | `us-east5` |
| `ANTHROPIC_MODEL` | No | Default model | `claude-sonnet-4@20250514` |
| `ANTHROPIC_SMALL_FAST_MODEL` | No | Fast model for quick responses | `claude-3-5-haiku@20241022` |
| `ANTHROPIC_TIMEOUT` | No | Connection timeout (seconds) | `30.0` |
| `ANTHROPIC_MAX_RETRIES` | No | Maximum retry attempts | `3` |
| `GOOGLE_APPLICATION_CREDENTIALS` | Optional | Service account key path | `/path/to/key.json` |
| `ANTHROPIC_API_KEY` | Not required in Vertex mode | Direct API key (only for non-Vertex usage) | `sk-ant-...` |

## Supported Regions and Models

### Regions with Claude Model Support
- `us-east5` (recommended)
- `us-central1`
- `us-west1`
- `us-west4`
- `europe-west1`
- `europe-west4`
- `asia-southeast1`

### Supported Claude Models
- `claude-3-5-sonnet@20241022` (recommended for quality)
- `claude-3-5-haiku@20241022` (recommended for speed/cost)
- `claude-sonnet-4@20250514` (latest, most capable)
- `claude-3-sonnet@20240229`
- `claude-3-haiku@20240307`

## Need Help?

If you're still having issues after following this guide:

1. Check the [Google Cloud Vertex AI documentation](https://cloud.google.com/vertex-ai/docs)
2. Verify your project has billing enabled
3. Ensure you're using a supported region
4. Check the demo's troubleshooting section in the main README
5. Contact your GCP administrator for permission issues
