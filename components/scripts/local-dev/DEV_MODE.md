# CRC Development Mode with Hot-Reloading

This guide explains how to use the development mode for hot-reloading with OpenShift Local (CRC).

**🎉 STATUS: FULLY WORKING - Project creation, authentication, and hot-reloading all functional!**

## Prerequisites

1. **CRC is running**: Ensure CRC is started with `crc start`
2. **fswatch installed** (for file watching):
   ```bash
   # macOS
   brew install fswatch
   
   # Linux
   sudo apt-get install fswatch  # Ubuntu/Debian
   sudo yum install fswatch       # RHEL/CentOS
   ```

## Starting Development Mode

### 1. Start with Development Images

```bash
# Set development mode and start
DEV_MODE=true make dev-start

# Or directly
DEV_MODE=true ./components/scripts/local-dev/crc-start.sh
```

This will:
- Build development Docker images with Air (Go) and Next.js dev server
- Push them to the CRC internal registry
- Deploy with volume mounts for source code
- Enable hot-reloading

### 2. Start Source Code Syncing

In a **separate terminal**, run the sync script:

```bash
# Sync both backend and frontend (recommended)
make dev-sync

# Or sync individually  
make dev-sync-backend
make dev-sync-frontend

# Or directly:
./components/scripts/local-dev/crc-dev-sync.sh both
```

The sync script will:
- Do an initial sync of your source code
- Watch for file changes
- Automatically sync changes to the pods

## How It Works

### Backend (Go with Air)
- Uses [Air](https://github.com/air-verse/air) for live reloading
- Watches `.go`, `.yaml`, and `.html` files
- Automatically rebuilds and restarts on changes
- Build artifacts go to `tmp/` directory

### Frontend (Next.js)
- Uses Next.js development server
- Hot Module Replacement (HMR) enabled
- Instant updates for React components
- Fast Refresh preserves component state

## Development Workflow

1. **Edit code locally** in your IDE/editor
2. **Save files** - changes are detected by fswatch
3. **Automatic sync** to CRC pods via `oc rsync`
4. **Hot reload** triggers in the containers
5. **See changes** immediately in your browser

## Accessing Services

After starting in dev mode:
```bash
# Get URLs
oc get routes -n vteam-dev

# Backend health check
curl -k https://vteam-backend-vteam-dev.apps-crc.testing/health

# Frontend
open https://vteam-frontend-vteam-dev.apps-crc.testing
```

## Viewing Logs

```bash
# Backend logs (with Air output)
oc logs -f deployment/vteam-backend -n vteam-dev

# Frontend logs (with Next.js output)
oc logs -f deployment/vteam-frontend -n vteam-dev
```

## Troubleshooting

### Sync Not Working
```bash
# Check if pods are running
oc get pods -n vteam-dev

# Manually trigger sync
oc rsync ./components/backend/ $(oc get pod -l app=vteam-backend -o name | head -1):/app/
```

### Hot Reload Not Triggering
```bash
# Restart the deployment
oc rollout restart deployment/vteam-backend -n vteam-dev
oc rollout restart deployment/vteam-frontend -n vteam-dev
```

### Permission Issues
```bash
# Fix permissions in pod
oc exec -it deployment/vteam-backend -- chmod -R 777 /app
```

## Switching Back to Production Mode

```bash
# Stop sync script (Ctrl+C)
# Then redeploy without DEV_MODE
make dev-start
```

## Advanced Configuration

### Custom Air Configuration
Edit `.air.toml` in the backend directory to customize:
- File watch patterns
- Build commands
- Restart delays

### Environment Variables
```bash
# Custom project name
PROJECT_NAME=my-dev DEV_MODE=true make dev-start

# Different CRC instance
CRC_PROFILE=myprofile DEV_MODE=true make dev-start
```

## Performance Tips

1. **Exclude large directories** from sync (node_modules, .git, etc.)
2. **Use specific sync targets** instead of syncing everything
3. **Increase PVC size** if needed for large projects
4. **Adjust resource limits** in deployments for better performance

## Known Limitations

- Initial sync can be slow for large codebases
- Some file changes might require manual pod restart
- Binary files and compiled assets need special handling
