# Workbench Development Environment

Multi-container development environment with Debian container and Jupyter Lab for data analysis and experimentation.

## Overview

Development platform consisting of:
- **Debian Container**: General-purpose development environment with SSH access
- **Jupyter Lab**: Interactive notebook environment for data science

Both containers share a common data volume for collaboration.

## Access

- **Debian SSH**: `ssh user@workbench.feist-gondola.ts.net` (Tailscale)
- **Jupyter Lab**: `http://jupyter-homelab.feist-gondola.ts.net` (Tailscale)

## Components

### Debian Development Container

**Image**: `debian:bookworm`
**Purpose**: General-purpose development environment

**Features**:
- SSH server (port 22)
- Development tools pre-installed
- Shared `/home/user/code` and `/home/user/data` volumes
- Persistent environment

**Resources**:
- Memory: 512Mi request, 1Gi limit
- CPU: 500m request, 1000m limit

### Jupyter Lab

**Image**: `jupyter/datascience-notebook:latest`
**Port**: 8888
**Purpose**: Interactive data analysis and visualization

**Features**:
- Python, R, Julia kernels
- Scientific computing libraries (NumPy, Pandas, SciPy, etc.)
- Visualization tools (Matplotlib, Seaborn, Plotly)
- Shared data access with Debian container

**Resources**:
- Memory: 1Gi request, 2Gi limit
- CPU: 500m request, 1000m limit

## Shared Storage

Both containers mount the same PVCs:
- `/home/user/code`: 10Gi - Source code and projects
- `/home/user/data`: 20Gi - Datasets and analysis results

## Setup

1. **Deploy workbench**:
   ```bash
   kubectl apply -f cluster/apps/workbench/
   ```

2. **Get Jupyter token**:
   ```bash
   kubectl logs -n workbench -l app=jupyter | grep "?token="
   ```

3. **SSH into Debian**:
   ```bash
   task workbench:ssh
   # or
   ssh user@workbench.feist-gondola.ts.net
   ```

## Usage

### SSH Access

```bash
# Via Taskfile
task workbench:ssh

# Direct SSH
ssh user@workbench.feist-gondola.ts.net

# Set terminal type for better compatibility
TERM=xterm-256color ssh user@workbench.feist-gondola.ts.net
```

### Mount Workbench Filesystems Locally

```bash
# Mount both code and data directories via SSHFS
task workbench:mount

# Access at:
# ~/mnt/workbench-code
# ~/mnt/workbench-data

# Unmount when done
task workbench:unmount
```

### Jupyter Lab

1. Access via Tailscale URL
2. Enter token from pod logs
3. Create notebooks in `/home/jovyan/data` to share with Debian container

### Installing Software

**In Debian container**:
```bash
ssh user@workbench.feist-gondola.ts.net
sudo apt update
sudo apt install <package>
```

**In Jupyter**:
```bash
kubectl exec -it -n workbench deployment/jupyter -- bash
conda install <package>
# or
pip install <package>
```

## Workflows

### Data Analysis Workflow

1. Upload data via SSHFS or FileBrowser to `/data`
2. Process in Jupyter notebooks
3. Save results to `/data`
4. Access from Debian container for further work

### Development Workflow

1. Clone repos to `/code` via SSH
2. Edit code in Jupyter or via SSH
3. Run scripts in Debian environment
4. Commit changes via SSH (Git configured)

## Maintenance

### Restart Containers

```bash
kubectl rollout restart deployment/debian -n workbench
kubectl rollout restart deployment/jupyter -n workbench
```

### View Logs

```bash
kubectl logs -n workbench -l app=debian -f
kubectl logs -n workbench -l app=jupyter -f
```

### Check Storage Usage

```bash
# From Debian container
ssh user@workbench.feist-gondola.ts.net "df -h /home/user/code /home/user/data"

# Or via kubectl
kubectl exec -n workbench deployment/debian -- df -h /home/user/code /home/user/data
```

### Backup Data

```bash
# Backup code directory
kubectl exec -n workbench deployment/debian -- tar czf - /home/user/code > workbench-code-backup.tar.gz

# Backup data directory
kubectl exec -n workbench deployment/debian -- tar czf - /home/user/data > workbench-data-backup.tar.gz
```

## Troubleshooting

### Can't SSH to Debian

Check if pod is running:
```bash
kubectl get pods -n workbench
kubectl logs -n workbench -l app=debian
```

Check Tailscale service:
```bash
kubectl get svc -n workbench
kubectl describe svc debian-external -n workbench
```

### Jupyter Token Not Working

Get fresh token:
```bash
kubectl logs -n workbench -l app=jupyter | grep "?token="
```

Or reset token:
```bash
kubectl exec -n workbench deployment/jupyter -- jupyter server list
```

### SSHFS Mount Fails

Check SSH connectivity first:
```bash
ssh user@workbench.feist-gondola.ts.net
```

Ensure SSHFS is installed locally:
```bash
# macOS
brew install macfuse sshfs

# Linux
sudo apt install sshfs
```

### Storage Full

```bash
# Check usage
kubectl exec -n workbench deployment/debian -- du -sh /home/user/code/* /home/user/data/*

# Clean up old files
ssh user@workbench.feist-gondola.ts.net
rm -rf /home/user/data/old-project
```

## Installed Tools

### Debian Container

Default packages (customize via Dockerfile):
- Git
- Python 3
- Build essentials
- SSH server
- Vim/Nano

### Jupyter Lab

Pre-installed via datascience-notebook:
- Python (via conda)
- R
- Julia
- Pandas, NumPy, SciPy
- Matplotlib, Seaborn
- Scikit-learn
- TensorFlow, PyTorch (may vary)

## Security Notes

- SSH key authentication recommended
- Jupyter token authentication required
- Services only accessible via Tailscale (not public)
- Consider adding network policies for additional isolation

## References

- [Jupyter Docker Stacks](https://jupyter-docker-stacks.readthedocs.io/)
- [Debian Documentation](https://www.debian.org/doc/)
- [SSHFS](https://github.com/libfuse/sshfs)
