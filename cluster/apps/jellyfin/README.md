# Jellyfin Media Server

Media server with web-based file browser for managing and streaming media content.

## Overview

Two-component deployment:
- **Jellyfin**: Media server for organizing and streaming movies, TV shows, music
- **FileBrowser**: Web-based file manager for uploading and managing media files

## Access

- **Jellyfin**: `http://jellyfin-homelab.feist-gondola.ts.net` (Tailscale)
- **FileBrowser**: `http://filebrowser-homelab.feist-gondola.ts.net` (Tailscale)

## Components

### Jellyfin Server

**Image**: `jellyfin/jellyfin:latest`
**Port**: 8096
**Storage**:
- Configuration: 2Gi PVC (`jellyfin-pvc-data`)
- Media library: 100Gi PVC (`jellyfin-pvc-media`)

**Resources**:
- Memory: 1Gi request, 2Gi limit
- CPU: 500m request, 1000m limit

### FileBrowser

**Image**: `filebrowser/filebrowser:latest`
**Port**: 80
**Storage**: Shared media PVC (100Gi)

**Resources**:
- Memory: 128Mi request, 256Mi limit
- CPU: 100m request, 200m limit

## Setup

1. **Apply manifests**:
   ```bash
   kubectl apply -f cluster/apps/jellyfin/
   ```

2. **Initial Jellyfin setup**:
   - Access Jellyfin via Tailscale URL
   - Complete first-run wizard
   - Configure media libraries pointing to `/media`

3. **FileBrowser setup**:
   - Default credentials: `admin` / `admin`
   - Change password immediately
   - Upload media to `/media` directory

## Usage

### Adding Media

1. Use FileBrowser to upload files to appropriate directories:
   - `/media/movies/`
   - `/media/tv/`
   - `/media/music/`

2. In Jellyfin, trigger library scan:
   - Dashboard → Libraries → Scan All Libraries

### Managing Storage

```bash
# Check PVC usage
kubectl exec -n jellyfin deployment/jellyfin -- df -h /config
kubectl exec -n jellyfin deployment/jellyfin -- df -h /media

# Resize PVCs if needed
kubectl edit pvc jellyfin-pvc-media -n jellyfin
```

## Maintenance

### Restart Services

```bash
kubectl rollout restart deployment/jellyfin -n jellyfin
kubectl rollout restart deployment/filebrowser -n jellyfin
```

### View Logs

```bash
kubectl logs -n jellyfin -l app=jellyfin -f
kubectl logs -n jellyfin -l app=filebrowser -f
```

### Backup Configuration

```bash
# Backup Jellyfin config
kubectl exec -n jellyfin deployment/jellyfin -- tar czf - /config > jellyfin-config-backup.tar.gz
```

## Troubleshooting

### Jellyfin Not Detecting Media

- Check file permissions in FileBrowser
- Verify media is in correct directory structure
- Trigger manual library scan in Jellyfin

### Can't Access via Tailscale

```bash
kubectl get svc -n jellyfin
kubectl describe svc jellyfin-external -n jellyfin
kubectl get pods -n tailscale
```

## References

- [Jellyfin Documentation](https://jellyfin.org/docs/)
- [FileBrowser Documentation](https://filebrowser.org/)
