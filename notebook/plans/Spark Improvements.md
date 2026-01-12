# Spark Improvements Plan

## Overview

This document outlines a multi-phase improvement plan for Spark, inspired by Fly.io's Sprites project and informed by current implementation gaps.

## Sprites vs Spark: Feature Comparison

### What Sprites Has That Spark Lacks

| Feature | Sprites | Spark | Priority |
|---------|---------|-------|----------|
| **Fast Instantiation** | Sub-second via checkpoint/restore | ~2 minutes (container build) | High |
| **Lifecycle Management** | Auto-sleep after inactivity | Always running | High |
| **State Persistence** | Checkpoint snapshots | None (ephemeral on delete) | Medium |
| **Resource Efficiency** | Sleep mode reduces costs | Full resource allocation 24/7 | Medium |
| **Pre-configured Tools** | Language runtimes, databases | Basic Debian + manual setup | Medium |
| **Storage** | 100GB persistent volume | 10GB PVC | Low |
| **Monitoring** | Usage metrics, activity tracking | None | Low |

### What Spark Has That Sprites Lacks

- **Dedicated PostgreSQL Database**: Each spark gets its own database (Sprites uses SQLite)
- **Tailscale Integration**: Built-in secure external access
- **Kubernetes Native**: Runs on existing homelab infrastructure
- **Full Root Access**: No restrictions on what can be installed

## Phase 1: Critical Fixes (Immediate)

These issues prevent basic functionality and must be fixed first.

### 1.1 Fix Claude Code Installation
**Problem**: Installation script returns 404 error
**Impact**: Missing primary development tool
**Implementation**:
- Investigate the install script URL in buildInitScript()
- Check if anthropics/claude-code repo has moved installation method
- Consider alternative installation methods (npm, binary download)
- Add error handling and retry logic

**Files**: `internal/k8s/resources.go:313`

### 1.2 Install PostgreSQL Client
**Problem**: No psql client in containers to test database connectivity
**Impact**: Cannot verify database creation or run queries
**Implementation**:
- Add `postgresql-client` to apt-get install list in buildInitScript()
- Test connection with: `psql $DATABASE_URL -c '\l'`

**Files**: `internal/k8s/resources.go:268`

### 1.3 Fix Dotfiles Installation
**Problem**: Homebrew not in PATH during dotfiles install
**Impact**: Incomplete development environment setup
**Implementation**:
- Source `.profile` or `.bashrc` before running dotfiles install
- Consider installing Homebrew in init script before dotfiles
- Add verification step to check dotfiles installation

**Files**: `internal/k8s/resources.go:318-322`

### 1.4 Create Dedicated Spark PostgreSQL User
**Problem**: Using postgres admin user (security concern)
**Impact**: All sparks have admin access to entire database
**Implementation**:
- Create dedicated `spark` user in PostgreSQL with limited permissions
- Grant only CREATEDB privilege, not superuser
- Update config defaults back to POSTGRES_USER="spark"
- Document setup in cluster/spark/SETUP.md

**Files**:
- `internal/config/config.go:26`
- `cluster/spark/SETUP.md`

## Phase 2: Sprites-Inspired Features (High Value)

These features significantly improve developer experience based on Sprites' design.

### 2.1 Implement Lifecycle Management
**Goal**: Auto-sleep sparks after inactivity, wake on SSH connection

**Implementation Steps**:
1. Add activity tracking to spark pods (last SSH connection timestamp)
2. Create controller/cronjob to check for inactive sparks (>2 hours)
3. Implement sleep command: `spark sleep <name>` (scales deployment to 0)
4. Implement wake command: `spark wake <name>` (scales deployment to 1)
5. Add automatic wake-on-connect using Tailscale proxy (if possible)

**Benefits**:
- Reduces resource usage for idle sparks
- Allows more concurrent sparks on limited hardware
- More sustainable for long-running "parked" projects

**Files to Create**:
- `cmd/sleep.go` - Sleep command
- `cmd/wake.go` - Wake command
- `cmd/status.go` - Show spark state (running/sleeping)
- `internal/k8s/lifecycle.go` - Lifecycle operations

### 2.2 State Snapshots
**Goal**: Checkpoint and restore spark state for faster instantiation

**Implementation Steps**:
1. Add snapshot command: `spark snapshot <name>`
2. Create snapshot by copying PVC contents to backup location
3. Store snapshot metadata (creation time, size, description)
4. Add restore option to create: `spark create --from-snapshot <snapshot-name>`
5. List snapshots: `spark snapshots [name]`

**Benefits**:
- Faster spin-up for similar environments
- Ability to save working states before experiments
- Template creation for common project types

**Files to Create**:
- `cmd/snapshot.go` - Snapshot commands
- `internal/storage/snapshots.go` - Snapshot management

### 2.3 Container Image Optimization
**Goal**: Reduce startup time from ~2 minutes to <30 seconds

**Implementation Steps**:
1. Create custom base image with common tools pre-installed:
   - openssh-server, sudo, curl, git, vim, tmux
   - Claude Code CLI
   - PostgreSQL client
   - Common language runtimes (Node.js, Python, Go)
2. Build and push image to local registry or Docker Hub
3. Update deployment to use custom image
4. Move user-specific setup (dotfiles, git clone) to post-start hook

**Benefits**:
- Much faster pod startup
- More reliable installs (no network dependencies)
- Consistent environment across all sparks

**Files**:
- `build/Dockerfile` - Custom image definition
- `internal/k8s/resources.go:148` - Update image reference

## Phase 3: Developer Experience (Medium Priority)

### 3.1 Pre-installed Developer Tools
**Goal**: Have common tools ready without manual installation

**Tools to Include**:
- **Editors**: vim, nano, helix
- **Languages**: node/npm, python/pip, go
- **Utilities**: jq, yq, fd, ripgrep, bat, fzf
- **Containers**: docker CLI (pointing to host daemon)
- **Cloud**: kubectl, gh (GitHub CLI)

**Implementation**: Add to custom base image (Phase 2.3)

### 3.2 Spark Templates
**Goal**: Quick-start templates for common project types

**Implementation Steps**:
1. Define template structure (language, tools, environment)
2. Create templates:
   - `web-sveltekit` - SvelteKit with Node.js, pnpm, database
   - `api-go` - Go API with PostgreSQL
   - `python-data` - Python with pandas, jupyter, postgres
   - `fullstack-ts` - Node.js + React + PostgreSQL
3. Add flag: `spark create --template web-sveltekit`
4. Template system applies pre-configuration and installs dependencies

**Files to Create**:
- `templates/` - Template definitions
- `internal/templates/` - Template application logic
- `cmd/create.go` - Add --template flag

### 3.3 Better Logging and Feedback
**Goal**: Improve visibility into what's happening during creation

**Implementation Steps**:
1. Add structured logging with different levels
2. Show progress for long-running operations:
   - "Installing packages... (1/5)"
   - "Cloning repository..."
   - "Setting up environment..."
3. Add `spark logs <name>` command to view container logs
4. Add `spark events <name>` to show k8s events

**Files**:
- `cmd/logs.go` - New logs command
- `cmd/create.go` - Enhanced progress reporting
- `internal/k8s/client.go` - Log streaming support

### 3.4 Port Forwarding Support
**Goal**: Easy access to web services running in sparks

**Implementation Steps**:
1. Add `spark ports <name>` command to list exposed ports
2. Add `spark forward <name> <local-port>:<remote-port>` for kubectl port-forward
3. Optionally expose common ports (3000, 5000, 8000, 8080) via Tailscale

**Files to Create**:
- `cmd/forward.go` - Port forwarding command

## Phase 4: Operational Improvements (Lower Priority)

### 4.1 Usage Metrics and Monitoring
**Goal**: Track spark usage and resource consumption

**Metrics to Collect**:
- Number of active/sleeping/total sparks
- CPU/memory usage per spark
- SSH connection count and duration
- Database size per spark
- Last activity timestamp

**Implementation**:
- Add Prometheus metrics endpoint
- Create Grafana dashboard
- Add `spark stats` command for CLI view

### 4.2 Automatic Cleanup
**Goal**: Prevent abandoned sparks from consuming resources

**Implementation Steps**:
1. Add creation timestamp to spark metadata
2. Add TTL (time-to-live) flag: `spark create --ttl 7d`
3. Create cleanup controller that:
   - Warns user before deletion (if contact method available)
   - Deletes sparks past TTL
   - Can be extended with `spark extend <name> --ttl 7d`
4. Add `spark list --expiring` to show sparks near deletion

### 4.3 Resource Quotas
**Goal**: Prevent resource exhaustion from too many sparks

**Implementation**:
- Define max concurrent running sparks (e.g., 10)
- Define max total sparks (e.g., 50)
- Enforce during creation
- Add queue system if needed

### 4.4 Backup and Disaster Recovery
**Goal**: Protect against data loss

**Implementation**:
- Automatic PVC snapshots (daily for running sparks)
- PostgreSQL database backups
- Metadata backup (spark configuration)
- Restore command: `spark restore <name> <backup-id>`

## Implementation Order

### Week 1: Critical Fixes
- [ ] Fix Claude Code installation
- [ ] Install PostgreSQL client
- [ ] Fix dotfiles installation
- [ ] Create dedicated spark PostgreSQL user
- [ ] Update documentation

### Week 2: Container Image Optimization
- [ ] Create custom base Docker image
- [ ] Add common tools and runtimes
- [ ] Build and test image
- [ ] Update deployment to use new image
- [ ] Measure startup time improvement

### Week 3: Lifecycle Management
- [ ] Implement sleep/wake commands
- [ ] Add activity tracking
- [ ] Create status command
- [ ] Test resource savings

### Week 4: State Snapshots
- [ ] Implement snapshot creation
- [ ] Add snapshot restore to create command
- [ ] Add snapshot listing
- [ ] Test snapshot workflow

### Week 5: Developer Experience
- [ ] Add logging and progress feedback
- [ ] Create spark templates
- [ ] Add logs and events commands
- [ ] Test improved UX

### Week 6+: Operational Features
- [ ] Port forwarding support
- [ ] Usage metrics and monitoring
- [ ] Automatic cleanup with TTL
- [ ] Resource quotas
- [ ] Backup and restore

## Success Metrics

- **Startup Time**: < 30 seconds (from ~2 minutes)
- **Resource Usage**: 50% reduction via sleep mode
- **Reliability**: 99% successful creations
- **Developer Satisfaction**: Quick enough for "vibe coding" one-offs
- **Operational**: Zero manual intervention needed for cleanup

## Open Questions

1. Should we implement checkpoint/restore like Sprites, or is sleep/wake sufficient?
2. What's the right default TTL for sparks? (7 days? 30 days? Never?)
3. Should sleeping sparks keep their PostgreSQL database running?
4. Do we want a web UI for managing sparks, or is CLI sufficient?
5. Should we support multiple storage classes (fast SSD vs slow HDD)?
