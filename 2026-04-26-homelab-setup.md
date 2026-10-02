# Homelab Setup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract NixOS infrastructure from current Ardent Forge into a standalone Homelab repository, establishing the shared foundation for both Ardent Forge and nb containers.

**Architecture:** Homelab is a NixOS flake providing rootless Podman via Quadlet, network isolation with nftables, secrets management via agenix, and shared service containers (Ollama). Both application containers (ardent-forge.container, nb.container) and project containers are managed by systemd-managed Quadlet units.

**Tech Stack:** NixOS, Quadlet (systemd containers), rootless Podman, nftables, agenix, Tailscale, Grafana/Prometheus/Loki/Tempo

---

### Task 1: Initialize Homelab NixOS Flake

**Files:**
- Create: `flake.nix`
- Create: `flake.lock`
- Create: `.gitignore`

- [ ] **Step 1: Create flake.nix with inputs**

```nix
{
  description = "Homelab NixOS flake for Ardent Forge and nb";

  inputs = {
    nixpkgs.url = "github:nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
    agenix.url = "github:ryantm/agenix";
    agenix.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, flake-utils, agenix }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            nix
            git
            agenix.packages.${system}.default
          ];
        };
      }
    ) // {
      nixosConfigurations.homelab = nixpkgs.lib.nixosSystem {
        system = "aarch64-linux";
        modules = [
          agenix.nixosModules.age
          ./hosts/bee-link/configuration.nix
          ./modules/podman.nix
          ./modules/nftables.nix
          ./modules/ollama.nix
          ./modules/monitoring.nix
        ];
        specialArgs = {
          inherit agenix;
        };
      };
    };
}
```

- [ ] **Step 2: Create .gitignore**

```
result/
result-*
.dirlocals
secrets/*.age
!secrets/public-keys.txt
```

- [ ] **Step 3: Initialize git and commit**

```bash
cd ~/Repos/github.com/t-eckert/homelab
git init
git add flake.nix .gitignore
git commit -m "init: create flake with basic structure"
```

---

### Task 2: Create NixOS Host Configuration

**Files:**
- Create: `hosts/bee-link/configuration.nix`
- Create: `hosts/bee-link/hardware-configuration.nix`

- [ ] **Step 1: Create minimal hardware configuration**

```nix
# hosts/bee-link/hardware-configuration.nix
{ config, lib, pkgs, ... }:
{
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;

  networking.hostName = "bee-link";
  networking.useDHCP = true;

  fileSystems."/" = {
    device = "/dev/disk/by-label/root";
    fsType = "ext4";
  };

  fileSystems."/boot" = {
    device = "/dev/disk/by-label/boot";
    fsType = "vfat";
  };
}
```

- [ ] **Step 2: Create host configuration**

```nix
# hosts/bee-link/configuration.nix
{ config, lib, pkgs, agenix, ... }:
{
  imports = [
    ./hardware-configuration.nix
    ../../modules/podman.nix
    ../../modules/nftables.nix
    ../../modules/ollama.nix
    ../../modules/monitoring.nix
  ];

  system.stateVersion = "24.05";

  nix.settings.experimental-features = [ "nix-command" "flakes" ];
  nix.settings.trusted-users = [ "root" "@wheel" ];

  networking.firewall.enable = false;
  networking.nameservers = [ "1.1.1.1" "1.0.0.1" ];

  time.timeZone = "America/Toronto";

  users.users.root.openssh.authorizedKeys.keys = [
    "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user@machine"  # Replace with actual key
  ];

  services.openssh.enable = true;
  services.openssh.settings.PasswordAuthentication = false;

  environment.systemPackages = with pkgs; [
    git
    curl
    wget
    htop
    tailscale
  ];
}
```

- [ ] **Step 3: Commit**

```bash
git add hosts/
git commit -m "feat: add bee-link host configuration"
```

---

### Task 3: Create Podman and Quadlet Module

**Files:**
- Create: `modules/podman.nix`

- [ ] **Step 1: Write podman module**

```nix
# modules/podman.nix
{ config, lib, pkgs, ... }:
{
  virtualisation.podman.enable = true;
  virtualisation.podman.dockerCompat = false;
  virtualisation.podman.autoPrune.enable = true;

  virtualisation.containers.enable = true;

  systemd.user.services.podman.after = lib.mkForce [ "network-online.target" ];
  systemd.user.services.podman.wants = lib.mkForce [ "network-online.target" ];

  environment.systemPackages = with pkgs; [
    podman
    podman-compose
  ];

  users.users.podman = {
    isSystemUser = true;
    group = "podman";
  };

  users.groups.podman = {};

  systemd.tmpfiles.rules = [
    "d /var/lib/podman-containers 0755 root root -"
  ];
}
```

- [ ] **Step 2: Create quadlet configs directory structure**

```bash
mkdir -p quadlet/ardent-forge quadlet/nb quadlet/monitoring
```

- [ ] **Step 3: Commit**

```bash
git add modules/podman.nix quadlet/
git commit -m "feat: add podman module and quadlet directories"
```

---

### Task 4: Configure nftables for Egress Filtering

**Files:**
- Create: `modules/nftables.nix`

- [ ] **Step 1: Write nftables module**

```nix
# modules/nftables.nix
{ config, lib, pkgs, ... }:
{
  networking.nftables.enable = true;
  networking.nftables.ruleset = ''
    table inet filter {
      chain input {
        type filter hook input priority filter; policy drop;
        iif lo accept
        ct state invalid drop
        ct state established,related accept
        ip protocol icmp accept
        ip6 nextheader icmpv6 accept
        tcp dport 22 accept
        tcp dport 80 accept
        tcp dport 443 accept
      }

      chain forward {
        type filter hook forward priority filter; policy drop;
      }

      chain output {
        type filter hook output priority filter; policy accept;
      }
    }

    table inet containers {
      chain output {
        type filter hook output priority filter; policy accept;

        oif "podman*" {
          ct state established,related accept
          ct state invalid drop
        }

        oif "veth*" {
          ct state established,related accept
          ct state invalid drop
        }
      }
    }
  '';
}
```

- [ ] **Step 2: Commit**

```bash
git add modules/nftables.nix
git commit -m "feat: add nftables egress filtering rules"
```

---

### Task 5: Setup Ollama Container and Monitoring Stack

**Files:**
- Create: `modules/ollama.nix`
- Create: `modules/monitoring.nix`
- Create: `quadlet/ollama.container`
- Create: `grafana/dashboards/homelab.json`

- [ ] **Step 1: Write ollama module**

```nix
# modules/ollama.nix
{ config, lib, pkgs, ... }:
{
  systemd.tmpfiles.rules = [
    "d /var/lib/ollama 0755 root root -"
  ];

  systemd.services.ollama = {
    description = "Ollama embedding service";
    after = [ "podman.service" ];
    wants = [ "podman.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "forking";
      Restart = "always";
      RestartSec = "5s";
      Environment = [
        "OLLAMA_HOST=0.0.0.0:11434"
        "OLLAMA_MODELS=/var/lib/ollama/models"
      ];
    };

    script = ''
      exec ${pkgs.ollama}/bin/ollama serve
    '';
  };
}
```

- [ ] **Step 2: Write monitoring module**

```nix
# modules/monitoring.nix
{ config, lib, pkgs, ... }:
{
  services.prometheus = {
    enable = true;
    port = 9090;
    globalConfig.scrape_interval = "15s";
    scrapeConfigs = [
      {
        job_name = "prometheus";
        static_configs = [{ targets = [ "localhost:9090" ]; }];
      }
    ];
  };

  services.grafana = {
    enable = true;
    settings = {
      server = {
        http_port = 3000;
        http_addr = "127.0.0.1";
      };
    };
    provisioning.datasources.settings.datasources = [
      {
        name = "Prometheus";
        type = "prometheus";
        access = "proxy";
        url = "http://localhost:9090";
        isDefault = true;
      }
    ];
  };

  services.loki = {
    enable = true;
    configuration = {
      auth_enabled = false;
      ingester.chunk_idle_period = "3m";
      ingester.max_chunk_age = "1h";
      limits_config.enforce_metric_name = false;
      schema_config.configs = [
        {
          from = "2024-01-01";
          store = "boltdb-shipper";
          object_store = "filesystem";
          schema = "v11";
          index.prefix = "index_";
          index.period = "24h";
        }
      ];
      server.http_listen_port = 3100;
    };
  };
}
```

- [ ] **Step 3: Create ollama Quadlet container**

```ini
# quadlet/ollama.container
[Unit]
Description=Ollama embedding service
After=network.target
Wants=network.target

[Container]
Image=ollama/ollama:latest
ContainerName=ollama
Publish=11434:11434
Volume=%h/.ollama:/root/.ollama:Z
Environment=OLLAMA_HOST=0.0.0.0:11434

[Install]
WantedBy=multi-user.target default.target
```

- [ ] **Step 4: Create Grafana dashboard**

```json
{
  "dashboard": {
    "title": "Homelab Overview",
    "panels": [
      {
        "title": "Container Status",
        "targets": [
          {
            "expr": "container_last_seen"
          }
        ]
      }
    ]
  }
}
```

- [ ] **Step 5: Commit**

```bash
git add modules/ollama.nix modules/monitoring.nix quadlet/ollama.container grafana/
git commit -m "feat: add ollama and monitoring stack"
```

---

### Task 6: Setup agenix Secrets and Verification

**Files:**
- Create: `secrets/public-keys.txt`
- Create: `secrets/secrets.nix`
- Create: `secrets/ollama-apikey.age` (encrypted)

- [ ] **Step 1: Create secrets configuration**

```nix
# secrets/secrets.nix
let
  bee-link = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... root@bee-link";
  users = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5... user@machine";
in
{
  "ollama-apikey.age".publicKeys = [ bee-link users ];
  "anthropic-api-key.age".publicKeys = [ bee-link users ];
  "linear-api-key.age".publicKeys = [ bee-link users ];
}
```

- [ ] **Step 2: Create public-keys.txt reference**

```
# Public SSH keys for secret encryption
# Update with actual keys from /root/.ssh/id_ed25519.pub and local machine

bee-link: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...
user-machine: ssh-ed25519 AAAAC3NzaC1lZDI1NTE5...
```

- [ ] **Step 3: Update flake.nix to include agenix module**

```nix
# In flake.nix, update nixosConfigurations.homelab.modules to include:
agenix.nixosModules.age
```

- [ ] **Step 4: Test build without secrets**

```bash
cd ~/Repos/github.com/t-eckert/homelab
nix flake check
```

Expected: No errors (secrets will be missing at runtime, which is expected for local dev)

- [ ] **Step 5: Commit**

```bash
git add secrets/
git commit -m "feat: add agenix secrets structure"
```

---

### Task 7: Verify Homelab Flake Structure and Test Build

**Files:**
- (no new files, verification only)

- [ ] **Step 1: List flake outputs**

```bash
nix flake show ~/Repos/github.com/t-eckert/homelab
```

Expected output shows `nixosConfigurations.homelab` and `devShells.default`

- [ ] **Step 2: Test flake can be built (dry run)**

```bash
nix build ~/Repos/github.com/t-eckert/homelab#nixosConfigurations.homelab.config.system.build.toplevel --dry-run
```

Expected: Build would succeed (no actual build on dry-run)

- [ ] **Step 3: Verify module imports are correct**

```bash
nix eval ~/Repos/github.com/t-eckert/homelab#nixosConfigurations.homelab.config.system.stateVersion
```

Expected: "24.05"

- [ ] **Step 4: Commit final structure**

```bash
git status
# Should show clean working tree
git log --oneline
# Should show 6 commits from initialization through agenix
```

- [ ] **Step 5: Create README documenting Homelab setup**

```markdown
# Homelab Infrastructure

Shared NixOS infrastructure for Ardent Forge and nb applications.

## Quick Start

```bash
# Enter dev shell
nix flake update
nix develop

# For actual deployment on bee-link (aarch64-linux):
nixos-rebuild switch --flake .#homelab --target-host root@bee-link
```

## Structure

- `flake.nix` — Flake inputs and outputs
- `hosts/bee-link/` — Host-specific configuration
- `modules/` — Reusable NixOS modules (podman, nftables, monitoring, ollama)
- `quadlet/` — Systemd Quadlet container definitions
- `secrets/` — agenix encrypted secrets (not in git)
- `grafana/` — Dashboard definitions

## Services

- **Podman**: Rootless container runtime (systemd-managed)
- **Ollama**: Embedding service (port 11434)
- **Grafana**: Observability dashboard (port 3000)
- **Prometheus**: Metrics collection (port 9090)
- **Loki**: Log aggregation (port 3100)
- **nftables**: Egress filtering per container

## Deployment

1. Update `secrets/secrets.nix` with actual SSH public keys
2. Run `agenix -e secrets/ollama-apikey.age` to create encrypted secrets
3. Deploy: `nixos-rebuild switch --flake .#homelab --target-host root@bee-link`
```

git add README.md
git commit -m "docs: add homelab README and deployment guide"
```

---

**Next Steps:**

Homelab is now ready as the foundation. Both Ardent Forge and nb will reference this flake and deploy their containers via Quadlet units managed by this NixOS configuration.
