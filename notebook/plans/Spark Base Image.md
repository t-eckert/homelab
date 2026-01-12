# Spark Base Image Plan

## Overview

Create a Debian-based Docker image with pre-installed tools and dotfiles to reduce Spark startup time from ~2 minutes to <30 seconds.

## Goals

1. **Fast Startup**: All dependencies pre-installed, no network downloads during pod creation
2. **Developer Ready**: Complete development environment based on t-eckert/dotfiles
3. **Consistent**: Same environment across all Spark instances
4. **Maintainable**: Easy to update and rebuild when dotfiles change

## Architecture

```
debian:bookworm (base)
  └─> spark-base:latest
       ├─ System packages (openssh-server, sudo, etc.)
       ├─ Development tools (git, neovim, go, node, etc.)
       ├─ Dotfiles cloned and installed
       ├─ Shell configuration (zsh, oh-my-zsh, starship)
       └─ User account pre-configured
```

## Dockerfile Structure

### Stage 1: System Setup
- Base: `debian:bookworm`
- Install system packages via apt
- Configure SSH server
- Create user account

### Stage 2: Development Tools
- Install language runtimes (Go, Node.js, Python)
- Install CLI tools (kubectl, helm, gh, etc.)
- Install shell tools (zsh, oh-my-zsh, starship)

### Stage 3: Dotfiles Integration
- Clone dotfiles repository
- Install configurations
- Set up shell environment

## Package Breakdown

### Critical Packages (Must Have)

**System Essentials:**
- `openssh-server` - SSH access
- `sudo` - Privilege escalation
- `ca-certificates` - SSL/TLS
- `curl`, `wget` - Download utilities
- `git` - Version control

**Shell Environment:**
- `zsh` - Shell (instead of bash)
- `starship` - Prompt (from binary release)
- `stow` - Dotfile symlink management

**Editors:**
- `neovim` - Primary editor (build from source or use PPA)
- `vim` - Fallback editor

**Core Development Tools:**
- `build-essential` - gcc, g++, make
- `postgresql-client` - Database access
- `jq` - JSON processor
- `ripgrep` - Fast grep alternative
- `fzf` - Fuzzy finder
- `bat` - Better cat with syntax highlighting
- `tmux` - Terminal multiplexer (fallback if zellij fails)

### Language Runtimes

**Go:**
- Download from official releases: https://go.dev/dl/
- Install to `/usr/local/go`
- Version: 1.23.x (latest stable)

**Node.js:**
- Use NodeSource repository
- Install npm and yarn
- Version: 20.x LTS

**Python:**
- Debian package: `python3`, `python3-pip`, `python3-venv`
- Version: 3.11+ (Bookworm default)

### Cloud/Kubernetes Tools

**Kubernetes:**
- `kubectl` - Kubernetes CLI
- `helm` - Kubernetes package manager
- `k9s` - Terminal UI for Kubernetes (optional, large binary)

**Cloud CLIs:**
- `gh` - GitHub CLI (from official repo)
- `doctl` - DigitalOcean CLI (optional)
- `flyctl` - Fly.io CLI (optional)

**Note**: Some cloud tools from Brewfile may be too platform-specific (azure-cli, gcloud). Consider making these opt-in or documenting manual installation.

### Tools to Skip (Platform-Specific or Unnecessary)

**Skip for Debian Image:**
- `iproute2mac` - macOS only
- `arp-scan` - May require privileges, install per-spark if needed
- `docker` - Can be installed per-spark if needed, or use host Docker
- `tailscale` - Already handled at cluster level
- GUI tools: `amethyst`, etc.

**Large/Optional Tools** (document for manual install):
- `ffmpeg` - Only if media processing needed
- `llvm` - Large, only if needed
- `azure-cli`, `gcloud` - Very large, cloud-specific

## Dotfiles Integration Strategy

### Installation Process

1. **Clone dotfiles** during image build:
   ```bash
   git clone https://github.com/t-eckert/dotfiles.git /etc/skel/.dotfiles
   ```

2. **Pre-install dependencies** (normally from Brewfile):
   - Extract core tools from Brewfile
   - Install via apt where possible
   - Download binaries for tools not in apt

3. **Run modified install.sh**:
   - Skip Homebrew installation (not suitable for Debian)
   - Skip `brew bundle` (manually handled above)
   - Run stow commands for config symlinks
   - Install Go tools: `go install ./tools/*`

4. **Configure for new users**:
   - Place dotfiles in `/etc/skel/.dotfiles`
   - Place configs in `/etc/skel/.config`
   - Place `.zshrc` in `/etc/skel/.zshrc`
   - When user is created, files copy automatically to `/home/user`

### Brewfile Translation

Map Homebrew packages to Debian equivalents:

| Brew Package | Debian Equivalent | Notes |
|--------------|-------------------|-------|
| `atuin` | Download binary | Not in Debian repos |
| `bat` | `bat` or build from source | May be named `batcat` |
| `buf` | Download binary | Protocol buffer tool |
| `deno` | Download binary | JavaScript runtime |
| `fzf` | `fzf` | In Debian repos |
| `gh` | GitHub CLI repo | Official deb package |
| `go` | Official tarball | golang.org/dl |
| `helm` | Official binary | helm.sh |
| `jq` | `jq` | In Debian repos |
| `k9s` | Download binary | GitHub releases |
| `kubectl` | Kubernetes repo | Official deb package |
| `lazygit` | Download binary | GitHub releases |
| `neovim` | PPA or build | Debian version may be old |
| `node` | NodeSource repo | Official node.js repo |
| `postgresql@15` | `postgresql-client` | Debian repos |
| `protobuf` | `protobuf-compiler` | In Debian repos |
| `ripgrep` | `ripgrep` | In Debian repos |
| `starship` | Download binary | starship.rs |
| `terraform` | HashiCorp repo | Official deb package |
| `tree` | `tree` | In Debian repos |
| `wget` | `wget` | In Debian repos |
| `yq` | Download binary | GitHub releases |
| `zellij` | Download binary | GitHub releases |

## Dockerfile Example Structure

```dockerfile
FROM debian:bookworm

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive

# Install system packages
RUN apt-get update && apt-get install -y \
    openssh-server \
    sudo \
    curl \
    wget \
    git \
    vim \
    neovim \
    zsh \
    build-essential \
    ca-certificates \
    postgresql-client \
    jq \
    ripgrep \
    fzf \
    tree \
    stow \
    python3 \
    python3-pip \
    python3-venv \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Install Node.js from NodeSource
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && npm install -g yarn

# Install Go
RUN wget https://go.dev/dl/go1.23.5.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.23.5.linux-amd64.tar.gz \
    && rm go1.23.5.linux-amd64.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/root/go"

# Install kubectl
RUN curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.31/deb/Release.key | \
    gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg \
    && echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.31/deb/ /' | \
    tee /etc/apt/sources.list.d/kubernetes.list \
    && apt-get update \
    && apt-get install -y kubectl

# Install GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | \
    dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | \
    tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update \
    && apt-get install -y gh

# Install starship
RUN curl -sS https://starship.rs/install.sh | sh -s -- --yes

# Install helm
RUN curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install binary tools from GitHub releases
RUN wget https://github.com/jesseduffield/lazygit/releases/download/v0.43.1/lazygit_0.43.1_Linux_x86_64.tar.gz \
    && tar xzf lazygit_0.43.1_Linux_x86_64.tar.gz -C /usr/local/bin \
    && rm lazygit_0.43.1_Linux_x86_64.tar.gz

RUN wget https://github.com/zellij-org/zellij/releases/download/v0.40.1/zellij-x86_64-unknown-linux-musl.tar.gz \
    && tar xzf zellij-x86_64-unknown-linux-musl.tar.gz -C /usr/local/bin \
    && rm zellij-x86_64-unknown-linux-musl.tar.gz

RUN wget https://github.com/mikefarah/yq/releases/download/v4.44.1/yq_linux_amd64 \
    && mv yq_linux_amd64 /usr/local/bin/yq \
    && chmod +x /usr/local/bin/yq

# Install Oh My Zsh (system-wide template)
RUN sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)" "" --unattended

# Install zsh-autosuggestions plugin
RUN git clone https://github.com/zsh-users/zsh-autosuggestions \
    ${ZSH_CUSTOM:-/root/.oh-my-zsh/custom}/plugins/zsh-autosuggestions

# Clone dotfiles to /etc/skel
RUN git clone https://github.com/t-eckert/dotfiles.git /etc/skel/.dotfiles

# Install dotfiles configurations to /etc/skel
WORKDIR /etc/skel/.dotfiles
RUN cp .zshrc /etc/skel/.zshrc && \
    cp config/.gitconfig /etc/skel/.gitconfig && \
    cp config/.editorconfig /etc/skel/.editorconfig && \
    mkdir -p /etc/skel/.config && \
    cp -r config/nvim /etc/skel/.config/ && \
    cp -r config/atuin /etc/skel/.config/ && \
    cp -r config/gh /etc/skel/.config/ && \
    cp -r config/k9s /etc/skel/.config/ && \
    cp -r config/zellij /etc/skel/.config/ && \
    cp config/starship.toml /etc/skel/.config/starship.toml

# Install Go tools from dotfiles
RUN if [ -d "./tools" ]; then \
        go install ./tools/*; \
    fi

# Configure SSH
RUN mkdir -p /run/sshd && \
    ssh-keygen -A && \
    echo "PermitRootLogin no" >> /etc/ssh/sshd_config && \
    echo "PasswordAuthentication no" >> /etc/ssh/sshd_config && \
    echo "PubkeyAuthentication yes" >> /etc/ssh/sshd_config && \
    echo "PermitUserEnvironment yes" >> /etc/ssh/sshd_config

WORKDIR /root

# Default command (will be overridden by Spark deployment)
CMD ["/usr/sbin/sshd", "-D"]
```

## Build and Deployment Process

### 1. Create Dockerfile
- Location: `homelab/spark/docker/Dockerfile`
- Based on structure above
- Test locally before pushing

### 2. Build Image
```bash
cd homelab/spark/docker
docker build -t spark-base:latest .
```

### 3. Test Image
```bash
# Run container locally
docker run -it --rm spark-base:latest /bin/zsh

# Verify installations
go version
node --version
kubectl version --client
gh --version
nvim --version
```

### 4. Push to Registry

**Option A: Docker Hub**
```bash
docker tag spark-base:latest teckert/spark-base:latest
docker push teckert/spark-base:latest
```

**Option B: GitHub Container Registry**
```bash
docker tag spark-base:latest ghcr.io/t-eckert/spark-base:latest
docker push ghcr.io/t-eckert/spark-base:latest
```

**Option C: Local Registry** (if running in homelab)
```bash
# Set up local registry in cluster
kubectl apply -f cluster/registry/

# Push to local registry
docker tag spark-base:latest registry.homelab.local/spark-base:latest
docker push registry.homelab.local/spark-base:latest
```

### 5. Update Spark Resources

In `internal/k8s/resources.go`, change:
```go
Image: "debian:bookworm",
```

To:
```go
Image: "teckert/spark-base:latest",  // or your registry
```

### 6. Simplify Init Script

With pre-installed base image, the init script becomes much simpler:

```bash
#!/bin/bash
set -e

echo "==> Creating user..."
if ! id -u user >/dev/null 2>&1; then
    useradd -u 1000 -m -d /home/user -s /bin/zsh user
    echo "user ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/user
    chmod 440 /etc/sudoers.d/user
fi

echo "==> Copying dotfiles from /etc/skel..."
cp -r /etc/skel/.dotfiles /home/user/
cp /etc/skel/.zshrc /home/user/
cp /etc/skel/.gitconfig /home/user/
cp /etc/skel/.editorconfig /home/user/
cp -r /etc/skel/.config /home/user/

echo "==> Setting up SSH..."
mkdir -p /home/user/.ssh
cp /tmp/spark-config/authorized_keys /home/user/.ssh/authorized_keys
chmod 600 /home/user/.ssh/authorized_keys
chmod 700 /home/user/.ssh

# Configure environment for SSH
cat > /home/user/.ssh/environment <<EOF
DATABASE_URL=$DATABASE_URL
ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY
SPARK_NAME=$SPARK_NAME
EOF
chmod 600 /home/user/.ssh/environment

# Configure GitHub CLI if token provided
if [ -f /tmp/spark-secret/GITHUB_TOKEN ]; then
    mkdir -p /home/user/.config/gh
    echo "github.com:" > /home/user/.config/gh/hosts.yml
    echo "    user: t-eckert" >> /home/user/.config/gh/hosts.yml
    echo "    oauth_token: $(cat /tmp/spark-secret/GITHUB_TOKEN)" >> /home/user/.config/gh/hosts.yml
    echo "    git_protocol: https" >> /home/user/.config/gh/hosts.yml
    chmod 700 /home/user/.config/gh
    chmod 600 /home/user/.config/gh/hosts.yml
fi

# Clone git repository if specified
if [ -n "$GIT_REPO" ] && [ ! -d /home/user/project ]; then
    su - user -c "git clone $GIT_REPO /home/user/project"
fi

echo "==> Setting ownership..."
chown -R 1000:1000 /home/user
chmod 755 /home/user

echo "==> Starting SSH daemon..."
echo "Spark is ready! Connect with: ssh user@spark-$SPARK_NAME"
exec /usr/sbin/sshd -D -e
```

## Version Management

### Image Tagging Strategy
- `spark-base:latest` - Latest build (for development)
- `spark-base:v1.0.0` - Stable releases
- `spark-base:YYYYMMDD` - Date-based tags for tracking

### Updating the Image
When dotfiles or tools change:
1. Rebuild image with new tag
2. Test in single spark
3. Update Spark code to use new tag
4. Roll out to all new sparks

## Estimated Size

- Base Debian: ~120 MB
- System packages: ~200 MB
- Language runtimes (Go, Node, Python): ~500 MB
- Tools (kubectl, gh, etc.): ~200 MB
- Dotfiles and configs: ~50 MB
- **Total: ~1 GB**

This is reasonable for a development container. Can be optimized with multi-stage builds if needed.

## Testing Checklist

After building image, verify:
- [ ] SSH server starts
- [ ] User can be created with useradd
- [ ] zsh is default shell
- [ ] oh-my-zsh is installed
- [ ] starship prompt works
- [ ] nvim opens and works
- [ ] Git operations work
- [ ] Go programs can compile
- [ ] Node.js/npm work
- [ ] kubectl connects to cluster
- [ ] gh CLI authenticates
- [ ] psql connects to database
- [ ] fzf, ripgrep, bat work
- [ ] Dotfiles configs are present
- [ ] Custom Go tools are installed

## Next Steps

1. Create `homelab/spark/docker/` directory
2. Write Dockerfile based on plan above
3. Build and test image locally
4. Push to container registry
5. Update `internal/k8s/resources.go` to use new image
6. Test spark creation with new image
7. Measure startup time improvement
8. Document image build process in README

## Open Questions

1. **Registry choice**: Docker Hub (public), GHCR (public), or local registry (private)?
2. **Update frequency**: Rebuild weekly? Monthly? On-demand?
3. **Tool versions**: Pin versions or use latest?
4. **Image size**: Is 1GB acceptable, or should we optimize?
5. **Oh My Zsh**: Install system-wide or per-user?
6. **Claude Code**: Still install in base image or keep in init script?
