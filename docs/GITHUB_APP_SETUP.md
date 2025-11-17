# GitHub App Setup

This guide explains how to configure a GitHub App for the Ambient Code Platform so users can browse repositories, clone, and push changes during agentic sessions and RFE seeding.

## Prerequisites

- A GitHub account (or organization) where the App will be installed
- Permissions to create a GitHub App
- Deployed ACP backend and frontend
- Ability to set environment variables on the backend Deployment

## 1) Create a GitHub App

1. Go to GitHub → Settings → Developer settings → GitHub Apps → New GitHub App
2. Use these base settings:
   - GitHub App name: Ambient Code Platform (or your own)
   - Homepage URL: your frontend route (e.g., https://ambient-code.<apps-domain>)
   - Callback URL (optional if using user OAuth verification): https://<frontend>/api/auth/github/user/callback
   - Webhook: Not required
3. Repository permissions (minimum):
   - Contents: Read and write (required for clone/push)
   - Pull requests: Read and write (recommended for PR creation)
   - Metadata: Read-only (default)
4. Account permissions: None required
5. Subscribe to events: None required
6. Where can this GitHub App be installed? Any account
7. Create the App and generate a Private Key (PEM). Keep the App ID handy.

## 2) Configure backend environment

Set these environment variables on the backend Deployment (the manifests read them from the `github-app-secret` Secret):

- GITHUB_APP_ID: The numeric App ID (e.g., 123456)
- GITHUB_PRIVATE_KEY: The PEM contents; raw PEM or base64-encoded PEM are both supported

Optional (for user OAuth verification flow):
- GITHUB_CLIENT_ID: OAuth app client ID for the GitHub App
- GITHUB_CLIENT_SECRET: OAuth app client secret
- GITHUB_STATE_SECRET: A random secret used to sign state (e.g., a long random string)

Example (base64-encoding PEM for easy env injection):
```bash
openssl genrsa -out app.pem 4096
# or download the PEM from the GitHub App page
export GITHUB_APP_ID=123456
export GITHUB_PRIVATE_KEY=$(base64 -w0 app.pem)
```

Note: The backend accepts either raw PEM or base64-encoded PEM.

## 3) Deploy/Restart backend

Apply your changes and restart the backend so it loads the new env vars.

## 4) Install the App

- Navigate to your frontend → Integrations → Connect GitHub
- You’ll be redirected to the GitHub App installation page
- Choose the account/org and repositories to grant access (at least the repos you will browse/clone/push)

When redirected back, the frontend links the installation by calling the backend endpoint:
- POST /api/auth/github/install

The backend stores a mapping of the current user to their installation in a ConfigMap (`github-app-installations`) in the backend namespace.

## 5) Verify the integration

- GET /api/auth/github/status should return installed: true for the logged-in user
- In the UI → Integrations, you should see the connected installation

## 6) Using the integration

- Repo browsing (tree/blob) proxies use the installation token minted server-side
- Agentic sessions and RFE seeding can clone/push using the token provided to the runner

## Troubleshooting

- 401/403 from GitHub API
  - Ensure the App is installed for the same user shown in the UI
  - Ensure the selected repositories are included in the installation
  - Verify backend env: GITHUB_APP_ID and GITHUB_PRIVATE_KEY
- Private key errors
  - If you base64-encoded the PEM, ensure no line breaks; use `base64 -w0` (Linux) or `base64 | tr -d '\n'`
- Linking fails
  - Check backend logs for `GitHub App not configured` or token mint failures
- User OAuth callback (optional)
  - If using verification via `/api/auth/github/user/callback`, set GITHUB_CLIENT_ID, GITHUB_CLIENT_SECRET, and GITHUB_STATE_SECRET

## GitHub Enterprise (GHE)

The Ambient Code Platform primarily targets github.com. If you need GHE, additional host configuration may be required throughout the codepaths (API base URL); contact maintainers before enabling.
