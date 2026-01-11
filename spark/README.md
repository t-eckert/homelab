# Spark ⚡

Quick-deploy dev environments for one-off projects in your homelab.

Inspired by Fly.io's [Sprites](https://fly.io/blog/code-and-let-live/), Spark creates ephemeral development environments in your Kubernetes cluster. Perfect for vibe coding, experiments, and quick prototypes.

## Features

Each spark automatically gets:

- **Random name**: Auto-generated adjective-noun combinations (e.g., `brave-dolphin`, `wise-falcon`)
- **Dev environment**: Debian container with SSH, Claude Code CLI, and your dotfiles
- **Network access**: Tailscale connectivity for external access
- **Database**: Dedicated PostgreSQL database with connection string pre-configured
- **Secrets**: Environment variables for `ANTHROPIC_API_KEY`, `GITHUB_TOKEN`, `DATABASE_URL`
- **Git integration**: Optional automatic cloning of a repository
- **Persistent storage**: 10GB volume mounted at `/home/user`

## Quick Start

### Prerequisites

- Kubernetes cluster with kubectl configured
- PostgreSQL instance accessible from the cluster
- Tailscale operator installed in the cluster
- Environment variables:
  - `ANTHROPIC_API_KEY` - Your Anthropic API key
  - `POSTGRES_PASSWORD` - PostgreSQL admin password
  - SSH public key at `~/.ssh/id_ed25519.pub` (or set `SSH_PUBLIC_KEY_PATH`)
  - `GITHUB_TOKEN` (optional) - For private repository access

### Installation

```bash
go install github.com/t-eckert/homelab/spark@latest
```

Or build from source:

```bash
git clone https://github.com/t-eckert/homelab.git
cd homelab/spark
go build -o spark
sudo mv spark /usr/local/bin/
```

### Usage

**Create a new spark:**

```bash
spark create
```

This will:
1. Generate a random name (e.g., `brave-dolphin`)
2. Create a PostgreSQL database
3. Deploy a Kubernetes pod with your dev environment
4. Wait for the pod to be ready
5. Automatically SSH into the container

**Create with a git repository:**

```bash
spark create --repo https://github.com/username/project.git
```

The repository will be cloned to `/home/user/project`.

**List active sparks:**

```bash
spark list
```

**Connect to an existing spark:**

```bash
spark shell brave-dolphin
```

**Delete a spark:**

```bash
spark delete brave-dolphin
```

This removes the Kubernetes resources and PostgreSQL database.

## Configuration

Spark uses environment variables for configuration:

| Variable | Default | Description |
|----------|---------|-------------|
| `ANTHROPIC_API_KEY` | *required* | Anthropic API key for Claude Code |
| `POSTGRES_PASSWORD` | *required* | PostgreSQL admin password |
| `POSTGRES_HOST` | `postgres.postgres.svc.cluster.local` | PostgreSQL hostname |
| `POSTGRES_PORT` | `5432` | PostgreSQL port |
| `POSTGRES_USER` | `spark` | PostgreSQL username |
| `POSTGRES_DB` | `postgres` | PostgreSQL admin database |
| `SSH_PUBLIC_KEY_PATH` | `~/.ssh/id_ed25519.pub` | Path to SSH public key |
| `GITHUB_TOKEN` | - | GitHub token for private repos (optional) |

## Architecture

### Kubernetes Resources

Each spark creates the following resources in the `spark` namespace:

- **Deployment**: Single replica running Debian with init script
- **Service**: LoadBalancer with Tailscale integration
- **PersistentVolumeClaim**: 10GB storage for `/home/user`
- **ConfigMap**: SSH authorized keys and configuration
- **Secret**: Database credentials, API keys, GitHub token

### Container Setup

The Debian container runs an init script that:

1. Installs system dependencies (SSH, git, curl, etc.)
2. Creates a non-root user (`user`) with sudo access
3. Configures SSH with your public key
4. Installs Claude Code CLI
5. Clones your dotfiles from `github.com/t-eckert/dotfiles`
6. Optionally clones a specified git repository
7. Starts SSH daemon

### Database

A PostgreSQL database is created with the same name as the spark. The connection string is available in the container as `$DATABASE_URL`:

```
host=postgres.postgres.svc.cluster.local port=5432 user=spark password=*** dbname=brave-dolphin sslmode=disable
```

### Networking

Sparks are accessible via Tailscale:

```bash
ssh user@spark-brave-dolphin
```

The Tailscale operator creates a proxy pod that handles the LoadBalancer service.

## Development

### Project Structure

```
spark/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and help
│   ├── create.go          # Create command
│   ├── list.go            # List command
│   ├── shell.go           # Shell command
│   └── delete.go          # Delete command
├── internal/
│   ├── k8s/               # Kubernetes client and resources
│   │   ├── client.go      # K8s API operations
│   │   └── resources.go   # Resource templates
│   ├── db/                # PostgreSQL operations
│   │   └── postgres.go    # Database creation/deletion
│   ├── config/            # Configuration loading
│   │   └── config.go      # Environment variable parsing
│   └── names/             # Name generation
│       └── generator.go   # Random adjective-noun names
├── main.go                # Entry point
└── go.mod                 # Dependencies
```

### Building

```bash
go build -o spark
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `k8s.io/client-go` - Kubernetes API client
- `k8s.io/api` - Kubernetes API types
- `github.com/lib/pq` - PostgreSQL driver

## Inspiration

This project is inspired by Fly.io's [Sprites](https://fly.io/blog/code-and-let-live/), which provides ephemeral dev environments with excellent UX. Spark brings a similar experience to self-hosted Kubernetes environments.

## License

MIT

