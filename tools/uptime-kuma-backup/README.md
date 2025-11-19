# Uptime Kuma Backup Tool

A Go CLI tool for backing up Uptime Kuma data from a Kubernetes deployment to a persistent volume.

## Features

- Creates a new PersistentVolumeClaim for backup storage
- Copies data from running Uptime Kuma pod to the backup PVC
- Automatic cleanup of backup jobs after completion
- Timestamped backup names for easy organization

## Prerequisites

- Go 1.22 or later
- kubectl configured with access to your Kubernetes cluster
- Uptime Kuma running in Kubernetes
- RBAC permissions applied (see Installation section)

## Installation

### 1. Apply RBAC Permissions

First, apply the necessary RBAC permissions for the backup job:

```bash
kubectl apply -f ../../cluster/uptime-kuma/backup-rbac.yaml
```

### 2. Build the Tool

```bash
# Download dependencies
make deps

# Build the binary
make build

# Optional: Install to /usr/local/bin
make install
```

## Usage

### Basic Usage

Run a backup with default settings:

```bash
./uptime-kuma-backup
```

This will:
- Create a PVC named `uptime-kuma-backup-YYYYMMDD-HHMMSS` in the `uptime-kuma` namespace
- Create a Kubernetes Job to copy data from the pod
- Wait for the backup to complete

### Custom Options

```bash
./uptime-kuma-backup \
  --namespace uptime-kuma \
  --pod uptime-kuma-0 \
  --container uptime-kuma \
  --source /app/data \
  --size 5Gi \
  --storage-class local-path \
  --name my-backup-name
```

### Available Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--namespace` | `uptime-kuma` | Namespace where Uptime Kuma is running |
| `--pod` | `uptime-kuma-0` | Name of the Uptime Kuma pod |
| `--container` | `uptime-kuma` | Name of the container in the pod |
| `--source` | `/app/data` | Path to backup from inside the container |
| `--size` | `5Gi` | Size of the backup PVC |
| `--storage-class` | `local-path` | Storage class for the backup PVC |
| `--name` | `uptime-kuma-backup-{timestamp}` | Name for the backup PVC |
| `--kubeconfig` | `~/.kube/config` | Path to kubeconfig file |

## How It Works

1. **PVC Creation**: Creates a new PersistentVolumeClaim with the specified size and storage class
2. **Job Creation**: Deploys a Kubernetes Job that:
   - Mounts the backup PVC
   - Uses `kubectl exec` to stream data from the source pod
   - Extracts the data to the backup volume
3. **Monitoring**: Waits for the job to complete and reports status
4. **Cleanup**: Job automatically deletes after 1 hour (configurable in code)

## Restoring from Backup

To restore from a backup, you can:

### Option 1: Update StatefulSet to use backup PVC

Edit your StatefulSet to mount the backup PVC instead of emptyDir:

```yaml
volumes:
  - name: uptime-storage
    persistentVolumeClaim:
      claimName: uptime-kuma-backup-YYYYMMDD-HHMMSS  # Your backup name
```

### Option 2: Copy data to a new deployment

Create a temporary pod to copy data from backup to a new volume:

```bash
kubectl run -n uptime-kuma restore-helper \
  --image=busybox \
  --command -- sleep infinity

# Mount your backup PVC and new PVC, then copy data
```

## Troubleshooting

### Backup Job Fails

Check the job logs:

```bash
kubectl logs -n uptime-kuma job/uptime-kuma-backup-YYYYMMDD-HHMMSS-job
```

### Permission Errors

Ensure RBAC permissions are applied:

```bash
kubectl get role uptime-kuma-backup -n uptime-kuma
kubectl get rolebinding uptime-kuma-backup -n uptime-kuma
```

### PVC Not Created

Check storage class availability:

```bash
kubectl get storageclass
```

## Development

### Project Structure

```
.
├── main.go           # Main application code
├── go.mod            # Go module definition
├── Makefile          # Build automation
└── README.md         # This file
```

### Building

```bash
make build
```

### Cleaning

```bash
make clean
```

## License

Part of the t-eckert/homelab repository.
