# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Spark is a Go CLI tool for creating ephemeral development environments in a Kubernetes homelab. Each "spark" provisions a Debian container with SSH, Claude Code CLI, dotfiles, a dedicated PostgreSQL database, and Tailscale connectivity.

## Commands

```bash
# Build
go build -o spark

# Run directly
go run . create
go run . list
go run . shell <name>
go run . delete <name>

# Install locally
go build -o spark && sudo mv spark /usr/local/bin/
```

## Architecture

The CLI uses Cobra for command structure. Each command orchestrates two backends:

- **Kubernetes** (`internal/k8s/`): Creates Deployment, Service (Tailscale LoadBalancer), PVC, ConfigMap, Secret in the `spark` namespace
- **PostgreSQL** (`internal/db/`): Creates/drops databases using a `spark` user with CREATEDB privilege

### Flow: `spark create`

1. `config.Load()` reads env vars and SSH public key
2. `names.Generate()` creates random adjective-noun name
3. `db.Client` creates PostgreSQL database with spark name
4. `k8s.Client.CreateSpark()` creates all K8s resources (ConfigMap → Secret → PVC → Service → Deployment)
5. Waits for pod readiness, then SSHs into container

### Key Files

- `internal/k8s/resources.go`: Contains `SparkResources` struct and all K8s resource builders, including the container init script that installs dependencies, creates user, configures SSH, and installs Claude Code
- `internal/k8s/client.go`: K8s API operations using client-go
- `internal/db/postgres.go`: Database creation/deletion and connection string builders
- `internal/config/config.go`: Environment variable loading with defaults

## Required Environment Variables

- `ANTHROPIC_API_KEY` - Required
- `POSTGRES_PASSWORD` - Required, password for `spark` PostgreSQL user
- `GITHUB_TOKEN` - Optional, for private repo access

## Resource Naming Convention

For a spark named `brave-dolphin`:
- Deployment: `brave-dolphin`
- Service: `brave-dolphin-ssh` (Tailscale hostname: `spark-brave-dolphin`)
- PVC: `brave-dolphin-storage`
- Secret: `brave-dolphin-secret`
- ConfigMap: `brave-dolphin-config`
- Database: `brave-dolphin`
