# Homelab Scripts

Automation scripts for managing the homelab Kubernetes cluster.

## Raspberry Pi Flashing

### Overview

`flash-pi.sh` writes a Raspberry Pi OS `.img.xz` to an SD card and configures headless first
boot — hostname, user, SSH key, and Wi-Fi — by generating a `custom.toml` on the boot
partition. That's Raspberry Pi OS's own first-boot provisioning format, the same one
Raspberry Pi Imager writes, so this is the CLI equivalent of Imager's customisation dialog.

**It erases the target card.** It refuses to write to anything that isn't removable, rejects
devices larger than 2 TB, and requires typing `ERASE` to proceed.

### Usage

```bash
# Download an image first (checksums are published alongside)
curl -L -o rpios.img.xz https://downloads.raspberrypi.com/raspios_lite_arm64_latest
curl -L    https://downloads.raspberrypi.com/raspios_lite_arm64_latest.sha256 | shasum -a 256 -c

# Via Task
task pi:flash IMAGE=rpios.img.xz

# Direct
PI_IMAGE=rpios.img.xz ./scripts/flash-pi.sh
```

It prompts for the username, account password, and Wi-Fi password (and the SSID, if `PI_SSID` is unset). Those are read
interactively so they never reach shell history, the environment, or this repository. The
account password is hashed with SHA-512 crypt before being written.

### Configuration

- `PI_IMAGE` — path to a `.img.xz` image (required)
- `PI_DISK` — target device. Auto-detected when exactly one removable disk is present.
- `PI_HOSTNAME` — default `sdr`; the Pi is reachable at `<hostname>.local`
- `PI_SSID` — Wi-Fi network to join; prompted for if unset
- `PI_COUNTRY` — default `CA`
- `PI_SSH_KEY` — default `~/.ssh/id_ed25519.pub`
- `PI_TIMEZONE` — default `America/Toronto`

**`PI_COUNTRY` is not optional in practice.** Without a wireless regulatory domain the Pi
keeps `wlan0` rfkill-blocked on 5 GHz, so a 5 GHz network is invisible to it and the board
boots with no network and no indication why.

## Raspberry Pi Identity

### Overview

`pi-identity.sh` captures a Pi's **immutable** identifiers — serial number, `eth0`/`wlan0` MAC
addresses, and board revision — and prints them as a Markdown block ready to paste into
[notebook/Hardware.md](../notebook/Hardware.md).

Hardware.md used to record each machine by IP address. A router change invalidated all of them
at once and left two visually identical Pi 4 Bs (1 GB and 4 GB) indistinguishable without
booting them. Serials and MACs don't change, so record those and treat the IP as a
*last seen* note.

### Usage

```bash
# Via Task (recommended)
task pi:identity HOST=sdr.local

# Direct, over SSH
PI_HOST=sdr.local ./scripts/pi-identity.sh

# Direct, running on the Pi itself
./scripts/pi-identity.sh
```

### How It Works

Reads `/proc/device-tree/model`, `/proc/cpuinfo`, `/proc/meminfo`, and `/sys/class/net/*/address`,
then cross-checks the reported RAM two independent ways: `MemTotal`, and the memory nibble of
the Pi 4 B revision code (`a` = 1 GB, `b` = 2 GB, `c` = 4 GB, `d` = 8 GB). It warns on stderr if
they disagree — `MemTotal` reads low because of the GPU carve-out, so the revision code wins.

### Configuration

- `PI_HOST` — host to SSH into. If unset, runs against the local machine.
- `PI_USER` — SSH user (default: current user).

See [notebook/Raspberry Pi Provisioning.md](../notebook/Raspberry%20Pi%20Provisioning.md) for the
full imaging and setup procedure.

## Image Update Automation

### Overview

The `update-images.sh` script provides automated container image updates based on Flux ImagePolicy resources. This is a workaround for Flux ImageUpdateAutomation not automatically committing changes to the repository.

### Usage

#### Via Task (Recommended)

```bash
# Check for available updates (dry-run)
task images:check

# Update images and commit locally
task images:update

# Update, commit, and push to remote
task images:update-and-push

# Show status of all ImagePolicy resources
task images:status
```

#### Direct Script Execution

```bash
# Dry-run mode
DRY_RUN=true ./scripts/update-images.sh

# Update and commit
./scripts/update-images.sh

# Update, commit, and push
AUTO_PUSH=true ./scripts/update-images.sh
```

### How It Works

1. **Queries Flux ImagePolicy resources** across all namespaces
2. **Checks readiness** - skips policies that aren't ready
3. **Finds deployment manifests** with `{"$imagepolicy": "namespace:name"}` markers
4. **Compares versions** - current image tag vs latest from policy
5. **Updates manifests** when newer images are available
6. **Creates git commits** with detailed change summaries

### Requirements

- `kubectl` - Kubernetes CLI
- `jq` - JSON processor
- `git` - Version control
- Access to the Kubernetes cluster with Flux installed
- Flux ImagePolicy resources configured

### Example Output

```bash
$ task images:check
INFO: Fetching ImagePolicy resources...
INFO: Processing field-theories/field-theories -> ghcr.io/t-eckert/field-theories/site:20260117-2124-sha-27bf745
INFO: Updating ./cluster/field-theories/deployment.yaml
INFO:   FROM: ghcr.io/t-eckert/field-theories/site:20260115-0509-sha-f048662
INFO:   TO:   ghcr.io/t-eckert/field-theories/site:20260117-2124-sha-27bf745
WARNING: [DRY RUN] Would update ./cluster/field-theories/deployment.yaml

SUCCESS: Updated 1 file(s)
WARNING: DRY RUN mode - no changes committed
```

### Configuration

Environment variables:

- `DRY_RUN` - Set to `true` to preview changes without modifying files (default: `false`)
- `AUTO_PUSH` - Set to `true` to automatically push commits to remote (default: `false`)
- `CLUSTER_DIR` - Path to cluster manifests directory (default: `./cluster`)

### Integration with Flux

This script complements Flux's existing image automation:

- **ImageRepository** - Scans container registries for new images
- **ImagePolicy** - Determines which image tag to use based on policy
- **This Script** - Updates manifests and commits changes (replaces ImageUpdateAutomation)

The script respects Flux's `{"$imagepolicy": "namespace:name"}` marker format, ensuring compatibility with Flux's update detection system.

### Troubleshooting

**No files found with marker**
- Ensure deployment manifests have the correct `{"$imagepolicy": "namespace:name"}` marker
- Check that the ImagePolicy name and namespace match exactly

**ImagePolicy not ready**
- Run `task images:status` to see policy status
- Check ImageRepository and ImagePolicy resources with `flux get images all`

**Script can't find images**
- Verify ImagePolicy has `status.latestRef.name` and `status.latestRef.tag` set
- Check that the ImageRepository is scanning successfully
