## OpenShift OAuth Setup (with oauth-proxy sidecar)

This project secures the frontend using the OpenShift oauth-proxy sidecar. The proxy handles login against the cluster and forwards authenticated requests to the Next.js app.

You only need to do two one-time items per cluster: create an OAuthClient and provide its secret to the app. Also ensure the Route host uses your cluster apps domain.

### Quick checklist (copy/paste)
Admin (one-time per cluster):
1. Set the Route host to your cluster domain
```bash
ROUTE_DOMAIN=$(oc get ingresses.config cluster -o jsonpath='{.spec.domain}')
oc -n ambient-code patch route frontend-route --type=merge -p '{"spec":{"host":"ambient-code.'"$ROUTE_DOMAIN"'"}}'
```
2. Create OAuthClient and keep the secret
```bash
ROUTE_HOST=$(oc -n ambient-code get route frontend-route -o jsonpath='{.spec.host}')
SECRET="$(openssl rand -base64 32 | tr -d '\n=+/0OIl')"; echo "$SECRET"
cat <<EOF | oc apply -f -
apiVersion: oauth.openshift.io/v1
kind: OAuthClient
metadata:
  name: ambient-frontend
secret: $SECRET
redirectURIs:
- https://$ROUTE_HOST/oauth/callback
grantMethod: auto
EOF
```

Deployer (per install):
3. Put the client secret in the app Secret and restart
```bash
oc -n ambient-code create secret generic frontend-oauth-config \
  --from-literal=client-secret="$SECRET" \
  --from-literal=cookie_secret="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)" \
  --dry-run=client -o yaml | oc apply -f -
oc -n ambient-code rollout restart deployment/frontend
```
4. Open the app: `oc -n ambient-code get route frontend-route -o jsonpath='{.spec.host}' | sed 's#^#https://#'`

### Prerequisites
- oc CLI configured to your cluster
- cluster-admin (to create `OAuthClient`), or an admin to run those steps for you
- Namespace: `ambient-code`

### What the manifests already do
- Deploy the frontend with an `oauth-proxy` sidecar (HTTPS on port 8443)
- Expose `frontend-service` with ports `http:3000` and `dashboard-ui:8443`
- Create a Route to `frontend-service:dashboard-ui` with edge TLS termination

### What you must still do
0) Set the Route host to your real cluster apps domain (if not already)
1) Create a cluster-scoped `OAuthClient` named `ambient-frontend` with a strong secret and a redirect URI that matches your Route
2) Put that same secret into the namespaced Secret `frontend-oauth-config` (keys: `client-secret`, `cookie_secret`)

---

### Step 1 — Create the OAuthClient (cluster-admin)

1. Get your Route host for the app:
```bash
ROUTE_HOST=$(oc -n ambient-code get route frontend -o jsonpath='{.spec.host}')
echo "$ROUTE_HOST"
```

2. Generate a strong client secret:
```bash
SECRET="$(openssl rand -base64 32 | tr -d '\n=+/0OIl')"
echo "$SECRET"
```

3. Create or update the OAuthClient:
```bash
cat <<EOF | oc apply -f -
apiVersion: oauth.openshift.io/v1
kind: OAuthClient
metadata:
  name: ambient-frontend
secret: $SECRET
redirectURIs:
- https://$ROUTE_HOST/oauth/callback
grantMethod: auto
EOF
```

4. Verify:
```bash
oc get oauthclient ambient-frontend -o jsonpath='{.secret}{"\n"}{.redirectURIs[0]}{"\n"}'
```

Notes:
- The OAuthClient name (ambient-frontend) must match the proxy arg `--client-id=ambient-frontend` set in `frontend-deployment.yaml`.
- The redirect URI must exactly match the app Route + `/oauth/callback`.

### Step 2 — Provide the secret to the app (namespaced Secret)

Option A) Using the deploy script with `.env`:
```bash
cd components/manifests
cat >> ../.env <<EOF
OCP_OAUTH_CLIENT_SECRET=$SECRET
# Optional: provide your own cookie secret; otherwise the script will generate one
# OCP_OAUTH_COOKIE_SECRET=$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)
EOF
./deploy.sh secrets
oc -n ambient-code rollout restart deployment/frontend
```

Option B) Manually create/update the Secret:
```bash
oc -n ambient-code create secret generic frontend-oauth-config \
  --from-literal=client-secret="$SECRET" \
  --from-literal=cookie_secret="$(LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 32)" \
  --dry-run=client -o yaml | oc apply -f -
oc -n ambient-code rollout restart deployment/frontend
```

The Deployment mounts this Secret at `/etc/oauth/config` and reads:
- `--client-secret-file=/etc/oauth/config/client-secret`
- `--cookie-secret-file=/etc/oauth/config/cookie_secret`

### Step 3 — Open the app
```bash
oc -n ambient-code get route frontend -o jsonpath='{.spec.host}' | sed 's#^#https://#'
```
Visit the printed URL. You should be redirected to OpenShift login and returned to the app after authentication.

---

### Troubleshooting
- Pod fails: "secret \"frontend-oauth-config\" not found"
  - Create the Secret (Step 2) and restart the Deployment.

- Login redirects back to an error or a wrong host
  - Ensure the OAuthClient redirect URI matches exactly `https://<route-host>/oauth/callback`.
  - If you changed the Route host, update the OAuthClient accordingly.

- 403 after login
  - The proxy arg `--openshift-delegate-urls` should include the backend API paths you need. Adjust based on your cluster policy.

- Cookie secret errors
  - Use an alphanumeric 32-char value for `cookie_secret` (or let the script generate it).

### Notes
- You do NOT need ODH secret generators or a ServiceAccount OAuth redirect for this minimal setup.
- You do NOT need app-level env like `OAUTH_SERVER_URL`; the sidecar handles the flow.

### Reference
- ODH Dashboard uses a similar oauth-proxy sidecar pattern (with more bells and whistles):
  [opendatahub-io/odh-dashboard](https://github.com/opendatahub-io/odh-dashboard)


