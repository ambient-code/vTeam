# GitLab Integration API Endpoints

This document describes the GitLab integration API endpoints available in vTeam.

## Base URL

```
http://vteam-backend:8080/api
```

For production deployments, replace with your actual backend URL.

## Authentication

All endpoints require authentication via Bearer token in the Authorization header:

```http
Authorization: Bearer <your-vteam-auth-token>
```

---

## Endpoints

### 1. Connect GitLab Account

Connect a GitLab account to vTeam by providing a Personal Access Token.

**Endpoint**: `POST /auth/gitlab/connect`

**Request Headers**:
```http
Content-Type: application/json
Authorization: Bearer <vteam-auth-token>
```

**Request Body**:
```json
{
  "personalAccessToken": "glpat-xxxxxxxxxxxxxxxxxxxx",
  "instanceUrl": "https://gitlab.com"
}
```

**Request Parameters**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `personalAccessToken` | string | Yes | GitLab Personal Access Token (PAT) |
| `instanceUrl` | string | No | GitLab instance URL. Defaults to "https://gitlab.com" if not provided |

**Example Request**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <vteam-token>" \
  -d '{
    "personalAccessToken": "glpat-xyz123abc456",
    "instanceUrl": "https://gitlab.com"
  }'
```

**Success Response** (`200 OK`):
```json
{
  "userId": "user-123",
  "gitlabUserId": "456789",
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

**Error Responses**:

**400 Bad Request** - Invalid request body:
```json
{
  "error": "Invalid request body",
  "statusCode": 400
}
```

**401 Unauthorized** - Not authenticated:
```json
{
  "error": "User not authenticated",
  "statusCode": 401
}
```

**500 Internal Server Error** - Token validation failed:
```json
{
  "error": "GitLab token validation failed: 401 Unauthorized",
  "statusCode": 500
}
```

**Notes**:
- Token is validated by calling GitLab API before storing
- Token is stored securely in Kubernetes Secrets
- Connection metadata stored in ConfigMap
- For self-hosted GitLab, `instanceUrl` must include protocol (https://)

---

### 2. Get GitLab Connection Status

Check if user has a GitLab account connected and retrieve connection details.

**Endpoint**: `GET /auth/gitlab/status`

**Request Headers**:
```http
Authorization: Bearer <vteam-auth-token>
```

**Example Request**:
```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer <vteam-token>"
```

**Success Response (Connected)** (`200 OK`):
```json
{
  "connected": true,
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "gitlabUserId": "456789"
}
```

**Success Response (Not Connected)** (`200 OK`):
```json
{
  "connected": false
}
```

**Error Responses**:

**401 Unauthorized** - Not authenticated:
```json
{
  "error": "User not authenticated",
  "statusCode": 401
}
```

**500 Internal Server Error** - Failed to retrieve status:
```json
{
  "error": "Failed to retrieve GitLab connection status",
  "statusCode": 500
}
```

---

### 3. Disconnect GitLab Account

Disconnect GitLab account from vTeam and remove stored credentials.

**Endpoint**: `POST /auth/gitlab/disconnect`

**Request Headers**:
```http
Authorization: Bearer <vteam-auth-token>
```

**Example Request**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
  -H "Authorization: Bearer <vteam-token>"
```

**Success Response** (`200 OK`):
```json
{
  "message": "GitLab account disconnected successfully",
  "connected": false
}
```

**Error Responses**:

**401 Unauthorized** - Not authenticated:
```json
{
  "error": "User not authenticated",
  "statusCode": 401
}
```

**500 Internal Server Error** - Disconnect failed:
```json
{
  "error": "Failed to disconnect GitLab account",
  "statusCode": 500
}
```

**Notes**:
- Removes GitLab PAT from Kubernetes Secrets
- Removes connection metadata from ConfigMap
- Does not affect your GitLab account itself
- AgenticSessions using GitLab will fail after disconnection

---

## Data Models

### ConnectGitLabRequest

Request body for connecting GitLab account.

```typescript
interface ConnectGitLabRequest {
  personalAccessToken: string;  // Required
  instanceUrl?: string;          // Optional, defaults to "https://gitlab.com"
}
```

**Validation Rules**:
- `personalAccessToken`: Must be non-empty string
- `instanceUrl`: Must be valid HTTPS URL if provided

**Example**:
```json
{
  "personalAccessToken": "glpat-xyz123abc456",
  "instanceUrl": "https://gitlab.company.com"
}
```

---

### ConnectGitLabResponse

Response from connecting GitLab account.

```typescript
interface ConnectGitLabResponse {
  userId: string;
  gitlabUserId: string;
  username: string;
  instanceUrl: string;
  connected: boolean;
  message: string;
}
```

**Fields**:
- `userId`: vTeam user ID
- `gitlabUserId`: GitLab user ID (from GitLab API)
- `username`: GitLab username
- `instanceUrl`: GitLab instance URL (GitLab.com or self-hosted)
- `connected`: Always `true` on success
- `message`: Success message

**Example**:
```json
{
  "userId": "user-abc123",
  "gitlabUserId": "789456",
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

---

### GitLabStatusResponse

Response from checking GitLab connection status.

```typescript
interface GitLabStatusResponse {
  connected: boolean;
  username?: string;      // Only present if connected
  instanceUrl?: string;   // Only present if connected
  gitlabUserId?: string;  // Only present if connected
}
```

**Fields**:
- `connected`: Whether GitLab account is connected
- `username`: GitLab username (omitted if not connected)
- `instanceUrl`: GitLab instance URL (omitted if not connected)
- `gitlabUserId`: GitLab user ID (omitted if not connected)

**Example (Connected)**:
```json
{
  "connected": true,
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "gitlabUserId": "789456"
}
```

**Example (Not Connected)**:
```json
{
  "connected": false
}
```

---

## Error Handling

### Error Response Format

All error responses follow this format:

```json
{
  "error": "Error message describing what went wrong",
  "statusCode": 400
}
```

### Common Error Codes

| Status Code | Meaning | Common Causes |
|-------------|---------|---------------|
| 400 | Bad Request | Invalid request body, missing required fields |
| 401 | Unauthorized | vTeam authentication token missing or invalid |
| 500 | Internal Server Error | GitLab token validation failed, database error, network error |

### GitLab-Specific Errors

When GitLab token validation fails, error messages include specific remediation:

**Invalid Token**:
```json
{
  "error": "GitLab token validation failed: 401 Unauthorized. Please ensure your token is valid and not expired",
  "statusCode": 500
}
```

**Insufficient Permissions**:
```json
{
  "error": "GitLab token validation failed: 403 Forbidden. Ensure your token has 'api', 'read_api', 'read_user', and 'write_repository' scopes",
  "statusCode": 500
}
```

**Network Error**:
```json
{
  "error": "Failed to connect to GitLab instance. Please check network connectivity and instance URL",
  "statusCode": 500
}
```

---

## Usage Examples

### Complete Workflow

#### 1. Check if Connected

```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer $VTEAM_TOKEN"
```

Response:
```json
{"connected": false}
```

#### 2. Connect Account

```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VTEAM_TOKEN" \
  -d '{
    "personalAccessToken": "'"$GITLAB_TOKEN"'",
    "instanceUrl": "https://gitlab.com"
  }'
```

Response:
```json
{
  "userId": "user-123",
  "gitlabUserId": "456789",
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

#### 3. Verify Connection

```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer $VTEAM_TOKEN"
```

Response:
```json
{
  "connected": true,
  "username": "johndoe",
  "instanceUrl": "https://gitlab.com",
  "gitlabUserId": "456789"
}
```

#### 4. Disconnect (if needed)

```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/disconnect \
  -H "Authorization: Bearer $VTEAM_TOKEN"
```

Response:
```json
{
  "message": "GitLab account disconnected successfully",
  "connected": false
}
```

---

### Self-Hosted GitLab Example

```bash
# Connect to self-hosted instance
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $VTEAM_TOKEN" \
  -d '{
    "personalAccessToken": "glpat-selfhosted-token",
    "instanceUrl": "https://gitlab.company.com"
  }'
```

Response indicates self-hosted instance:
```json
{
  "userId": "user-123",
  "gitlabUserId": "12345",
  "username": "jdoe",
  "instanceUrl": "https://gitlab.company.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

---

## Security Considerations

### Token Storage

- GitLab PATs stored in Kubernetes Secret: `gitlab-user-tokens`
- Stored in backend namespace (not user's project namespace)
- Encrypted at rest by Kubernetes
- Never exposed in API responses
- Automatically redacted in logs

### Token Scopes

Required GitLab token scopes:
- `api` - Full API access
- `read_api` - Read API access
- `read_user` - Read user information
- `write_repository` - Push to repositories

### Best Practices

1. **Use HTTPS**: Always use HTTPS for API calls in production
2. **Rotate Tokens**: Encourage users to rotate GitLab tokens every 90 days
3. **Minimum Scopes**: Only request required scopes
4. **Token Expiration**: Set expiration dates on GitLab tokens
5. **Secure vTeam Tokens**: Protect vTeam authentication tokens

---

## Rate Limiting

### GitLab.com Limits

- 300 requests per minute per user
- 10,000 requests per hour per user

### Self-Hosted Limits

Limits configured by GitLab administrator (may differ from GitLab.com).

### vTeam Behavior

- No rate limit enforcement on vTeam side
- GitLab API errors (429 Too Many Requests) passed through to user
- Error messages include wait time recommendation

---

## Testing

### Unit Tests

```bash
cd components/backend
go test ./handlers/... -run TestGitLab -v
```

### Integration Tests

```bash
export INTEGRATION_TESTS=true
export GITLAB_TEST_TOKEN="glpat-xxx"
export GITLAB_TEST_REPO_URL="https://gitlab.com/user/repo.git"

go test ./tests/integration/gitlab/... -v
```

### Manual Testing with cURL

See examples throughout this document.

---

## Troubleshooting

### "Invalid request body"

**Cause**: JSON malformed or missing required fields

**Solution**:
- Verify JSON is valid
- Ensure `personalAccessToken` field is present
- Check Content-Type header is `application/json`

### "User not authenticated"

**Cause**: vTeam authentication token missing or invalid

**Solution**:
- Include `Authorization: Bearer <token>` header
- Verify vTeam token is valid
- Check token hasn't expired

### "GitLab token validation failed: 401"

**Cause**: GitLab token is invalid or expired

**Solution**:
- Create new GitLab Personal Access Token
- Verify token is copied correctly (no extra spaces)
- Check token hasn't been revoked in GitLab

### "GitLab token validation failed: 403"

**Cause**: Token lacks required scopes

**Solution**:
- Recreate token with all required scopes:
  - `api`
  - `read_api`
  - `read_user`
  - `write_repository`

---

## Related Documentation

- [GitLab Integration Guide](../gitlab-integration.md)
- [GitLab Token Setup](../gitlab-token-setup.md)
- [Self-Hosted GitLab Configuration](../gitlab-self-hosted.md)
- [GitLab Testing Procedures](../gitlab-testing-procedures.md)

---

## Changelog

### v1.1.0 (2025-11-05)

- ✨ Initial GitLab integration API release
- Added `/auth/gitlab/connect` endpoint
- Added `/auth/gitlab/status` endpoint
- Added `/auth/gitlab/disconnect` endpoint
- Support for GitLab.com and self-hosted instances
- Personal Access Token authentication
