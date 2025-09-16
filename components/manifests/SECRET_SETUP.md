# API Key Secret Setup

The vTeam platform requires an Anthropic API key to function.

## Secret Creation

```bash
# Create the secret directly
oc create secret generic ambient-code-secrets \
  -n your-namespace \
  --from-literal=anthropic-api-key="your-actual-key-here"

# Or update an existing secret
oc patch secret ambient-code-secrets -n your-namespace \
  -p '{"stringData":{"anthropic-api-key":"your-actual-key-here"}}'
```

## Verification

Check that the secret was created correctly:

```bash
# Verify secret exists
oc get secret ambient-code-secrets -n your-namespace

# Check the secret content (base64 encoded)
oc get secret ambient-code-secrets -n your-namespace -o yaml

# Decode and verify the key (optional)
oc get secret ambient-code-secrets -n your-namespace -o jsonpath='{.data.anthropic-api-key}' | base64 -d
```
