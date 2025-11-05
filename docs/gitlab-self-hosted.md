# Self-Hosted GitLab Configuration for vTeam

This guide covers everything you need to configure vTeam with self-hosted GitLab instances (GitLab Community Edition or GitLab Enterprise Edition).

## Overview

vTeam fully supports self-hosted GitLab installations, including:
- ✅ GitLab Community Edition (CE)
- ✅ GitLab Enterprise Edition (EE)
- ✅ Custom domains and ports
- ✅ Self-signed SSL certificates (with configuration)
- ✅ Internal/private networks
- ✅ Air-gapped environments (with limitations)

---

## Prerequisites

### Network Requirements

**From vTeam Backend Pods**:
- HTTPS access to GitLab instance (typically port 443)
- DNS resolution of GitLab hostname
- No firewall blocking outbound connections

**From GitLab Instance**:
- No inbound access required from vTeam
- All communication is outbound from vTeam to GitLab

### GitLab Requirements

**Minimum GitLab Version**: 13.0+
- Recommended: 14.0+ for best API compatibility
- API v4 must be enabled (default)
- Personal Access Tokens enabled (default)

**Required GitLab Features**:
- API access enabled
- Git over HTTPS enabled
- Personal Access Tokens not disabled by administrator

### User Permissions

**GitLab User Requirements**:
- Active user account on GitLab instance
- Ability to create Personal Access Tokens (may be restricted by admin)
- Member of repositories with at least Developer role

**GitLab Administrator** (for installation setup):
- Access to GitLab admin area (optional, for troubleshooting)
- Can verify token settings and rate limits

---

## Configuration

### Step 1: Verify GitLab Accessibility

Before configuring vTeam, verify GitLab is accessible from Kubernetes cluster:

```bash
# From your local machine (should work if GitLab is public)
curl -I https://gitlab.company.com

# From vTeam backend pod (critical test)
kubectl exec -it <backend-pod-name> -n vteam-backend -- \
  curl -I https://gitlab.company.com
```

**Expected Response**:
```
HTTP/2 200
server: nginx
...
```

**Common Issues**:

**Connection refused**:
```
curl: (7) Failed to connect to gitlab.company.com port 443: Connection refused
```
- Firewall blocking traffic from Kubernetes
- GitLab not accessible from cluster network
- Wrong hostname or port

**DNS resolution failed**:
```
curl: (6) Could not resolve host: gitlab.company.com
```
- DNS not configured in Kubernetes cluster
- Hostname typo
- Internal DNS not reachable from pods

**SSL certificate error**:
```
curl: (60) SSL certificate problem: self signed certificate
```
- Self-signed certificate (see SSL Configuration section below)

---

### Step 2: Create Personal Access Token

Follow the [GitLab PAT Setup Guide](./gitlab-token-setup.md) with these self-hosted specific notes:

**Access Token Page URL**:
```
https://gitlab.company.com/-/profile/personal_access_tokens
```
(Replace `gitlab.company.com` with your instance hostname)

**Required Scopes** (same as GitLab.com):
- ✅ `api`
- ✅ `read_api`
- ✅ `read_user`
- ✅ `write_repository`

**Expiration Policy**:
- Check with your GitLab administrator
- Some organizations enforce maximum expiration (e.g., 90 days)
- Some require periodic rotation

---

### Step 3: Test GitLab API Access

Verify token works with your self-hosted instance:

```bash
# Test user API
curl -H "Authorization: Bearer glpat-your-token" \
  https://gitlab.company.com/api/v4/user
```

**Expected Response** (200 OK):
```json
{
  "id": 123,
  "username": "yourname",
  "name": "Your Name",
  "state": "active",
  "web_url": "https://gitlab.company.com/yourname"
}
```

**Test from Backend Pod** (critical):
```bash
kubectl exec -it <backend-pod> -n vteam-backend -- \
  curl -H "Authorization: Bearer glpat-your-token" \
  https://gitlab.company.com/api/v4/user
```

If this fails but works from your machine, there's a network/firewall issue.

---

### Step 4: Connect to vTeam

**Via API** (recommended for initial testing):

```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <vteam-auth-token>" \
  -d '{
    "personalAccessToken": "glpat-your-gitlab-token",
    "instanceUrl": "https://gitlab.company.com"
  }'
```

**Instance URL Format**:
| ✅ Correct | ❌ Incorrect |
|------------|--------------|
| `https://gitlab.company.com` | `gitlab.company.com` (missing protocol) |
| `https://git.internal.corp` | `https://gitlab.company.com/` (trailing slash) |
| `https://gitlab.company.com:8443` | `https://gitlab.company.com/api/v4` (includes path) |

**Success Response**:
```json
{
  "userId": "user-123",
  "gitlabUserId": "456789",
  "username": "yourname",
  "instanceUrl": "https://gitlab.company.com",
  "connected": true,
  "message": "GitLab account connected successfully"
}
```

**Check Connection**:
```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer <vteam-token>"
```

Expected:
```json
{
  "connected": true,
  "username": "yourname",
  "instanceUrl": "https://gitlab.company.com",
  "gitlabUserId": "456789"
}
```

---

## SSL Certificate Configuration

### Trusted SSL Certificates

If your GitLab instance uses SSL certificates from a trusted CA (e.g., Let's Encrypt, DigiCert), no additional configuration needed.

**Verify**:
```bash
curl https://gitlab.company.com
# Should not show certificate errors
```

---

### Self-Signed SSL Certificates

If your GitLab uses self-signed certificates, you must configure vTeam backend pods to trust them.

#### Option 1: Add CA Certificate to Backend Pods

**Step 1: Get GitLab CA Certificate**

```bash
# Download GitLab's CA certificate
echo | openssl s_client -showcerts -connect gitlab.company.com:443 2>/dev/null | \
  openssl x509 -outform PEM > gitlab-ca.crt
```

**Step 2: Create Kubernetes ConfigMap**

```bash
kubectl create configmap gitlab-ca-cert \
  --from-file=gitlab-ca.crt=gitlab-ca.crt \
  -n vteam-backend
```

**Step 3: Mount CA Certificate in Backend Deployment**

Edit backend deployment:
```bash
kubectl edit deployment vteam-backend -n vteam-backend
```

Add volume and volumeMount:
```yaml
spec:
  template:
    spec:
      containers:
      - name: backend
        # ... existing config ...
        volumeMounts:
        - name: gitlab-ca-cert
          mountPath: /etc/ssl/certs/gitlab-ca.crt
          subPath: gitlab-ca.crt
          readOnly: true
      volumes:
      - name: gitlab-ca-cert
        configMap:
          name: gitlab-ca-cert
```

**Step 4: Update CA Certificates in Pod**

Add init container to update CA trust store:
```yaml
spec:
  template:
    spec:
      initContainers:
      - name: update-ca-certificates
        image: alpine:latest
        command:
        - sh
        - -c
        - |
          cp /etc/ssl/certs/gitlab-ca.crt /usr/local/share/ca-certificates/
          update-ca-certificates
        volumeMounts:
        - name: gitlab-ca-cert
          mountPath: /etc/ssl/certs/gitlab-ca.crt
          subPath: gitlab-ca.crt
```

**Step 5: Verify**

```bash
# After pod restarts
kubectl exec -it <backend-pod> -n vteam-backend -- \
  curl https://gitlab.company.com
# Should not show certificate errors
```

---

#### Option 2: Disable SSL Verification (NOT RECOMMENDED)

**Security Warning**: Only use for testing/development. Never in production.

Set environment variable in backend deployment:
```yaml
env:
- name: GIT_SSL_NO_VERIFY
  value: "true"
```

---

### Certificate Troubleshooting

**Problem**: "x509: certificate signed by unknown authority"

**Cause**: Self-signed certificate not trusted

**Solutions**:
1. Add CA certificate (Option 1 above)
2. Check certificate chain is complete
3. Verify certificate matches hostname

**Problem**: "x509: certificate has expired"

**Cause**: GitLab certificate expired

**Solutions**:
1. Contact GitLab administrator to renew certificate
2. Cannot be worked around from vTeam side

**Problem**: "x509: certificate is valid for gitlab.local, not gitlab.company.com"

**Cause**: Certificate hostname mismatch

**Solutions**:
1. Use correct hostname in `instanceUrl`
2. GitLab admin must issue certificate for correct hostname
3. Add hostname to certificate SAN (Subject Alternative Names)

---

## Network Configuration

### Firewall Rules

**Required Outbound Access** (from vTeam backend pods):
```
Source: vTeam backend pods (namespace: vteam-backend)
Destination: GitLab instance
Protocol: HTTPS (TCP)
Port: 443 (or custom if GitLab uses different port)
```

**Example Firewall Rule**:
```
ALLOW tcp from 10.0.0.0/8 (Kubernetes pod network) to 192.168.1.100 port 443
```

**No Inbound Access Required**: GitLab doesn't need to reach vTeam.

---

### DNS Configuration

vTeam backend pods must be able to resolve GitLab hostname.

**Test DNS Resolution**:
```bash
kubectl exec -it <backend-pod> -n vteam-backend -- \
  nslookup gitlab.company.com
```

**Expected**:
```
Server:   10.96.0.10
Address:  10.96.0.10#53

Name: gitlab.company.com
Address: 192.168.1.100
```

**DNS Issues**:

**Problem**: "server can't find gitlab.company.com: NXDOMAIN"

**Cause**: Internal DNS not configured

**Solutions**:
1. Configure CoreDNS to forward internal domains
2. Add custom DNS to backend pods:
   ```yaml
   spec:
     dnsPolicy: "None"
     dnsConfig:
       nameservers:
       - 192.168.1.10  # Internal DNS server
       searches:
       - company.com
   ```

---

### Proxy Configuration

If vTeam backend pods require HTTP proxy to reach GitLab:

**Add Proxy Environment Variables**:
```yaml
env:
- name: HTTP_PROXY
  value: "http://proxy.company.com:8080"
- name: HTTPS_PROXY
  value: "http://proxy.company.com:8080"
- name: NO_PROXY
  value: "localhost,127.0.0.1,.cluster.local"
```

**For Authenticated Proxy**:
```yaml
- name: HTTP_PROXY
  value: "http://username:password@proxy.company.com:8080"
```

**Git Proxy Configuration** (for git operations):
```yaml
- name: GIT_PROXY_COMMAND
  value: "http://proxy.company.com:8080"
```

---

## Custom Ports

If your GitLab instance runs on a non-standard port:

**Standard Ports**:
- HTTPS: 443 (default)
- HTTP: 80 (not recommended, insecure)

**Custom Port Example**:
```
GitLab URL: https://gitlab.company.com:8443
```

**Configure in vTeam**:
```json
{
  "personalAccessToken": "glpat-xxx",
  "instanceUrl": "https://gitlab.company.com:8443"
}
```

**API URL Construction**:
```
Instance URL: https://gitlab.company.com:8443
API URL: https://gitlab.company.com:8443/api/v4
```

vTeam automatically preserves the custom port.

---

## GitLab Administrator Configuration

### Rate Limits

Self-hosted GitLab administrators can configure custom rate limits.

**Check Current Limits** (as admin):
1. Admin Area → Settings → Network → Rate Limits
2. Default: Same as GitLab.com (300/min, 10000/hour)

**Recommended Settings for vTeam**:
- Authenticated API rate limit: 300 requests/minute (default)
- Unauthenticated rate limit: Can be lower
- Protected paths rate limit: Not applicable to vTeam

**If Users Hit Rate Limits Frequently**:
- Consider increasing limits for authenticated API calls
- Monitor GitLab performance
- Check for misconfigured integrations

---

### Personal Access Token Settings

Administrators can restrict PAT creation and usage.

**Settings Location**:
1. Admin Area → Settings → General
2. Expand "Account and limit"
3. Personal Access Token section

**Recommended Settings**:
- ✅ Personal Access Tokens: **Enabled**
- ✅ Project Access Tokens: Can be disabled (not used by vTeam)
- ⚠️ Token expiration: Enforce 90-day maximum (recommended)
- ⚠️ Limit token lifetime: Yes (security best practice)

**If PAT Creation is Disabled**:
- Users cannot connect to vTeam
- Administrator must enable PATs
- Or use alternative authentication (not currently supported)

---

### API Access

**Verify API is Enabled**:
1. Admin Area → Settings → General
2. Expand "Visibility and access controls"
3. Check "Enable access to the GitLab API" is **ON**

If disabled, vTeam cannot function.

---

## Air-Gapped Environments

For completely air-gapped GitLab installations:

### Requirements

**GitLab Instance**:
- Accessible from Kubernetes cluster (internal network)
- Does NOT need internet access

**vTeam Backend**:
- Must reach GitLab instance (internal network)
- Does NOT need internet access for GitLab operations

### Configuration

Same as standard self-hosted setup - air-gap doesn't affect vTeam → GitLab communication.

**Network Diagram**:
```
┌─────────────────────┐         ┌──────────────────┐
│  vTeam Backend Pods │────────▶│  GitLab Instance │
│   (Kubernetes)      │ HTTPS   │  (Self-Hosted)   │
└─────────────────────┘         └──────────────────┘
       Internal Network Only
      (No Internet Required)
```

---

## Multi-Instance Support

vTeam users can connect to multiple self-hosted GitLab instances simultaneously.

**Limitation**: Each vTeam user can connect to **one** GitLab instance at a time.

**Use Cases**:

**Scenario 1: Different Users, Different Instances**
- User A connects to `https://gitlab-dev.company.com`
- User B connects to `https://gitlab-prod.company.com`
- ✅ Supported - each user has their own connection

**Scenario 2: One User, Multiple Instances**
- User needs access to both `gitlab-dev` and `gitlab-prod`
- ❌ Not supported - must choose one instance per user
- Workaround: Use different vTeam user accounts

**Scenario 3: Mixed GitLab.com and Self-Hosted**
- User connects to `https://gitlab.company.com` (self-hosted)
- Same user wants `https://gitlab.com` (SaaS)
- ❌ Not supported - must choose one

---

## Troubleshooting

### Connection Issues

**Problem**: "Failed to connect to GitLab instance"

**Debug Steps**:

1. **Test from local machine**:
   ```bash
   curl https://gitlab.company.com
   ```
   - If fails: GitLab is down or DNS issue
   - If succeeds: Continue to step 2

2. **Test from backend pod**:
   ```bash
   kubectl exec -it <backend-pod> -n vteam-backend -- \
     curl https://gitlab.company.com
   ```
   - If fails: Firewall or network issue
   - If succeeds: Continue to step 3

3. **Test GitLab API**:
   ```bash
   kubectl exec -it <backend-pod> -n vteam-backend -- \
     curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.company.com/api/v4/user
   ```
   - If fails: API disabled or token invalid
   - If succeeds: vTeam configuration issue

4. **Check vTeam logs**:
   ```bash
   kubectl logs -l app=vteam-backend -n vteam-backend | grep -i gitlab
   ```

---

### API Version Issues

**Problem**: "API endpoint not found" (404)

**Cause**: GitLab version too old or API disabled

**Check GitLab Version**:
```bash
curl https://gitlab.company.com/api/v4/version
```

**Expected** (v13.0+):
```json
{
  "version": "14.10.0",
  "revision": "abc123"
}
```

**Solutions**:
- Upgrade GitLab to 13.0+
- Contact administrator to enable API

---

### Performance Issues

**Problem**: Slow GitLab API responses

**Debug**:

1. **Check API response times**:
   ```bash
   time curl -H "Authorization: Bearer glpat-xxx" \
     https://gitlab.company.com/api/v4/user
   ```
   - Should complete in < 1 second
   - If > 5 seconds: GitLab performance issue

2. **Check network latency**:
   ```bash
   kubectl exec -it <backend-pod> -n vteam-backend -- \
     ping -c 5 gitlab.company.com
   ```
   - Should be < 50ms for same datacenter
   - > 200ms indicates network issues

3. **Contact GitLab Administrator**:
   - Check GitLab resource utilization (CPU, memory, disk I/O)
   - Review Sidekiq queue length
   - Check PostgreSQL query performance

---

## Security Considerations

### Token Storage

**Where Tokens Are Stored**:
- Kubernetes Secret: `gitlab-user-tokens` in `vteam-backend` namespace
- Encrypted at rest (Kubernetes default encryption)
- Never logged in plaintext

**Access Control**:
- Only vTeam backend pods can read secret
- Kubernetes RBAC enforced
- Administrators can view secret but not decode tokens automatically

**Audit Trail**:
- GitLab logs all API calls with user information
- Check GitLab audit log for token usage
- Admin Area → Monitoring → Audit Events

---

### Network Security

**Recommendations**:

1. **Use HTTPS Only**
   - Never use HTTP for GitLab
   - All tokens sent over HTTPS

2. **Restrict Network Access**
   - Firewall: Only allow vTeam backend pods → GitLab
   - No direct user access from pods to GitLab UI needed

3. **SSL/TLS Configuration**
   - Use trusted certificates (Let's Encrypt, etc.)
   - If self-signed: Properly configure CA trust
   - Never disable SSL verification in production

4. **Audit Logging**
   - Enable GitLab audit logging
   - Monitor for unusual API activity
   - Review PAT usage regularly

---

### Compliance

For regulated environments (HIPAA, SOC 2, etc.):

**Token Security**:
- ✅ Tokens encrypted at rest in Kubernetes Secrets
- ✅ Tokens encrypted in transit (HTTPS only)
- ✅ Tokens automatically redacted in logs
- ✅ Token rotation supported (manually)

**Audit Trail**:
- ✅ GitLab logs all API calls with user identity
- ✅ vTeam logs all operations with redacted tokens
- ✅ Kubernetes audit logs track secret access

**Access Control**:
- ✅ RBAC controls who can access vTeam
- ✅ GitLab permissions control repository access
- ✅ No elevated privileges required

---

## Best Practices

### For GitLab Administrators

1. **Enable and Monitor Audit Logs**
   - Admin Area → Monitoring → Audit Events
   - Track PAT creation and usage
   - Alert on unusual activity

2. **Enforce Token Expiration**
   - Set maximum token lifetime (90 days recommended)
   - Users must rotate tokens regularly

3. **Configure Rate Limits Appropriately**
   - Default limits work for most use cases
   - Increase only if legitimate usage hits limits
   - Monitor API performance impact

4. **Maintain GitLab Version**
   - Keep GitLab up to date (security patches)
   - Test vTeam compatibility before major upgrades
   - Minimum: GitLab 13.0+

5. **SSL Certificate Management**
   - Use trusted certificates (Let's Encrypt, etc.)
   - Automate certificate renewal
   - Plan for certificate expiration

---

### For vTeam Users

1. **Use Strong Tokens**
   - Create separate token for vTeam
   - Use descriptive name: "vTeam Integration"
   - Minimum required scopes only

2. **Rotate Tokens Regularly**
   - Every 90 days recommended
   - Before expiration date
   - Immediately if compromised

3. **Monitor Token Usage**
   - Check "Last Used" date in GitLab
   - Revoke unused tokens
   - Contact admin if suspicious activity

4. **Repository Access**
   - Request minimum necessary access
   - Developer role sufficient for most use cases
   - Avoid Owner/Maintainer unless needed

---

## Reference

### API Endpoints Used by vTeam

vTeam uses these GitLab API v4 endpoints:

**Authentication & User**:
```
GET  /api/v4/user
GET  /api/v4/personal_access_tokens/self
```

**Repository Operations**:
```
GET  /api/v4/projects/:id
GET  /api/v4/projects/:id/repository/branches
GET  /api/v4/projects/:id/repository/tree
GET  /api/v4/projects/:id/repository/files/:file_path
```

**Git Operations** (via git protocol, not API):
```
git clone https://oauth2:TOKEN@gitlab.company.com/owner/repo.git
git push https://oauth2:TOKEN@gitlab.company.com/owner/repo.git
```

### Required Minimum Scopes

| Scope | Purpose | Required |
|-------|---------|----------|
| `api` | Full API access | ✅ Yes |
| `read_api` | Read API access | ✅ Yes |
| `read_user` | User info | ✅ Yes |
| `write_repository` | Git push | ✅ Yes |

---

## Support

**For Self-Hosted GitLab Issues**:
- Contact your GitLab administrator
- Check GitLab logs: `/var/log/gitlab/`
- GitLab Community Forum: https://forum.gitlab.com

**For vTeam Integration Issues**:
- vTeam GitHub Issues: https://github.com/natifridman/vTeam/issues
- Check vTeam logs: `kubectl logs -l app=vteam-backend -n vteam-backend`

**For Network/Firewall Issues**:
- Contact your network/infrastructure team
- Provide: Source IPs (pod network), destination (GitLab), port (443)

---

## Quick Reference

**Test Connectivity**:
```bash
# From backend pod
kubectl exec -it <backend-pod> -n vteam-backend -- \
  curl https://gitlab.company.com
```

**Test API**:
```bash
kubectl exec -it <backend-pod> -n vteam-backend -- \
  curl -H "Authorization: Bearer glpat-xxx" \
  https://gitlab.company.com/api/v4/user
```

**Connect to vTeam**:
```bash
curl -X POST http://vteam-backend:8080/api/auth/gitlab/connect \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <vteam-token>" \
  -d '{"personalAccessToken":"glpat-xxx","instanceUrl":"https://gitlab.company.com"}'
```

**Check Status**:
```bash
curl -X GET http://vteam-backend:8080/api/auth/gitlab/status \
  -H "Authorization: Bearer <vteam-token>"
```

**View Logs**:
```bash
kubectl logs -l app=vteam-backend -n vteam-backend | grep -i gitlab
```
