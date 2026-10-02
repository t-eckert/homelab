#!/usr/bin/env bash
#
# Flash a Raspberry Pi OS image to an SD card and configure it for headless
# first boot: hostname, user, SSH key, and Wi-Fi.
#
# THIS ERASES THE TARGET CARD. It refuses to run against anything that isn't a
# removable card reader, and asks for confirmation before writing.
#
# Passwords are read interactively and never appear in shell history, the
# environment, or this repository.
#
# Usage:
#   PI_IMAGE=~/Downloads/2026-06-18-raspios-trixie-arm64-lite.img.xz ./scripts/flash-pi.sh
#
# Environment:
#   PI_IMAGE     Path to a .img.xz Raspberry Pi OS image (required).
#   PI_DISK      Target device, e.g. /dev/disk4. Auto-detected if unset.
#   PI_HOSTNAME  Hostname, reachable as <hostname>.local (default: sdr).
#   PI_SSID      Wi-Fi network to join. Prompted for if unset.
#   PI_COUNTRY   Wireless regulatory domain (default: CA). See note below.
#   PI_SSH_KEY   Public key to authorise (default: ~/.ssh/id_ed25519.pub).
#   PI_TIMEZONE  (default: America/Toronto)
#
# The country code is not optional in practice: without a regulatory domain the
# Pi keeps wlan0 rfkill-blocked on 5 GHz, so a 5 GHz network is simply invisible
# and the board boots with no network and no explanation.

set -euo pipefail

PI_IMAGE="${PI_IMAGE:-}"
PI_HOSTNAME="${PI_HOSTNAME:-sdr}"
PI_SSID="${PI_SSID:-}"
PI_COUNTRY="${PI_COUNTRY:-CA}"
PI_SSH_KEY="${PI_SSH_KEY:-$HOME/.ssh/id_ed25519.pub}"
PI_TIMEZONE="${PI_TIMEZONE:-America/Toronto}"
PI_DISK="${PI_DISK:-}"

die() { echo "ERROR: $*" >&2; exit 1; }

[[ "$(uname -s)" == "Darwin" ]] || die "this script targets macOS (diskutil)"
[[ -n "$PI_IMAGE" ]] || die "set PI_IMAGE to a .img.xz Raspberry Pi OS image"
[[ -r "$PI_IMAGE" ]] || die "cannot read image: $PI_IMAGE"
[[ -r "$PI_SSH_KEY" ]] || die "cannot read SSH public key: $PI_SSH_KEY"

# No default SSID: the network name is kept out of this public repository.
if [[ -z "$PI_SSID" ]]; then
  read -r -p "Wi-Fi SSID: " PI_SSID
  [[ -n "$PI_SSID" ]] || die "SSID cannot be empty"
fi

# ---------------------------------------------------------------------------
# Find the card, and refuse to touch anything that isn't one.
# ---------------------------------------------------------------------------

if [[ -z "$PI_DISK" ]]; then
  mapfile -t candidates < <(
    for d in $(diskutil list | awk '/^\/dev\/disk[0-9]+ \(/ {gsub(/:/,""); print $1}'); do
      if diskutil info "$d" 2>/dev/null | grep -q "Removable Media:.*Removable"; then
        echo "$d"
      fi
    done
  )
  [[ ${#candidates[@]} -gt 0 ]] || die "no removable disk found. Is the card inserted?"
  [[ ${#candidates[@]} -eq 1 ]] || die "several removable disks found (${candidates[*]}). Set PI_DISK explicitly."
  PI_DISK="${candidates[0]}"
fi

info=$(diskutil info "$PI_DISK" 2>/dev/null) || die "no such device: $PI_DISK"

grep -q "Removable Media:.*Removable" <<<"$info" \
  || die "$PI_DISK is not removable. Refusing to write."
grep -q "Media Read-Only:.*No" <<<"$info" \
  || die "$PI_DISK is read-only (is the card's lock switch on?)"

media=$(awk -F: '/Device \/ Media Name/ {sub(/^ +/,"",$2); print $2}' <<<"$info")
size=$(awk -F: '/Disk Size/ {sub(/^ +/,"",$2); print $2}' <<<"$info")
bytes=$(awk '/Disk Size/ {gsub(/[()]/,""); for(i=1;i<=NF;i++) if($i=="Bytes)"||$i=="Bytes") print $(i-1)}' <<<"$info" | head -1)

# A card bigger than 2 TB is almost certainly a misidentified external drive.
if [[ -n "$bytes" && "$bytes" -gt 2199023255552 ]]; then
  die "$PI_DISK is ${size} — too large to be an SD card. Refusing to write."
fi

cat <<EOF

  Target      : $PI_DISK
  Media       : $media
  Size        : $size

  Image       : $(basename "$PI_IMAGE")
  Hostname    : $PI_HOSTNAME  (reachable as ${PI_HOSTNAME}.local)
  Wi-Fi SSID  : $PI_SSID
  Country     : $PI_COUNTRY
  SSH key     : $PI_SSH_KEY
  Timezone    : $PI_TIMEZONE

  Everything currently on $PI_DISK will be destroyed.

EOF

read -r -p "Type ERASE to continue: " confirm
[[ "$confirm" == "ERASE" ]] || die "aborted"

# ---------------------------------------------------------------------------
# Credentials. Read before the long write so it can run unattended after this.
# ---------------------------------------------------------------------------

read -r -p "Username for the Pi: " PI_USERNAME
[[ -n "$PI_USERNAME" ]] || die "username cannot be empty"

read -r -s -p "Password for ${PI_USERNAME}: " user_pw; echo
read -r -s -p "Confirm: " user_pw2; echo
[[ "$user_pw" == "$user_pw2" ]] || die "passwords do not match"
[[ -n "$user_pw" ]] || die "password cannot be empty"

read -r -s -p "Wi-Fi password for '${PI_SSID}': " wifi_pw; echo
read -r -s -p "Confirm: " wifi_pw2; echo
[[ "$wifi_pw" == "$wifi_pw2" ]] || die "Wi-Fi passwords do not match"
[[ -n "$wifi_pw" ]] || die "Wi-Fi password cannot be empty"

# SHA-512 crypt, the format /etc/shadow expects. Falls back to storing the
# password in the clear only if this Python has no crypt module (3.13 removed
# it), and says so rather than doing it silently.
if user_pw_hash=$(PW="$user_pw" python3 -c '
import crypt, os
print(crypt.crypt(os.environ["PW"], crypt.mksalt(crypt.METHOD_SHA512)))
' 2>/dev/null); then
  user_pw_encrypted=true
else
  echo "NOTE: no crypt module available; storing the account password unhashed" >&2
  echo "      on the boot partition. Change it after first boot with passwd." >&2
  user_pw_hash="$user_pw"
  user_pw_encrypted=false
fi

# ---------------------------------------------------------------------------
# Write.
# ---------------------------------------------------------------------------

echo
echo "Caching sudo credentials for the write..."
sudo -v

raw="${PI_DISK/\/dev\/disk//dev/rdisk}"   # character device: much faster

echo "Unmounting $PI_DISK..."
diskutil unmountDisk "$PI_DISK"

echo "Writing $(basename "$PI_IMAGE") to $raw. This takes a few minutes."
IMG="$PI_IMAGE" python3 -c '
import lzma, os, sys, time
path = os.environ["IMG"]
written = 0
start = time.monotonic()
last = 0.0
with lzma.open(path, "rb") as f:
    while True:
        chunk = f.read(4 * 1024 * 1024)
        if not chunk:
            break
        sys.stdout.buffer.write(chunk)
        written += len(chunk)
        now = time.monotonic()
        if now - last > 2:
            last = now
            mb = written / 1048576
            sys.stderr.write(f"\r  {mb:,.0f} MiB written ({mb/(now-start):,.1f} MiB/s)")
            sys.stderr.flush()
sys.stdout.buffer.flush()
sys.stderr.write(f"\r  {written/1048576:,.0f} MiB written. Flushing to card...\n")
' | sudo dd of="$raw" bs=4m 2>/dev/null

sync
echo "Write complete."

# ---------------------------------------------------------------------------
# Headless first-boot configuration.
# ---------------------------------------------------------------------------

echo "Remounting to write first-boot configuration..."
diskutil mountDisk "$PI_DISK" >/dev/null 2>&1 || true

boot=""
for _ in $(seq 1 30); do
  for candidate in /Volumes/bootfs /Volumes/boot; do
    [[ -d "$candidate" ]] && boot="$candidate" && break 2
  done
  sleep 1
done
[[ -n "$boot" ]] || die "boot partition did not mount; cannot write configuration"

ssh_key_line=$(< "$PI_SSH_KEY")

# A Wi-Fi password containing a quote or backslash would otherwise produce
# invalid TOML, which first boot skips silently — leaving a Pi with no network
# and nothing to explain why. SHA-512 crypt hashes only use [a-zA-Z0-9./], so
# they need no escaping, but user input does.
toml_escape() { printf '%s' "$1" | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'; }

esc_hostname=$(toml_escape "$PI_HOSTNAME")
esc_username=$(toml_escape "$PI_USERNAME")
esc_ssid=$(toml_escape "$PI_SSID")
esc_wifi_pw=$(toml_escape "$wifi_pw")
# No-op for a crypt hash; matters on the unhashed fallback path.
esc_user_pw=$(toml_escape "$user_pw_hash")

# custom.toml is Raspberry Pi OS's own first-boot provisioning format (Bookworm
# and later) — the same one Raspberry Pi Imager writes.
umask 077
cat > "$boot/custom.toml" <<EOF
# Written by scripts/flash-pi.sh. Consumed and removed on first boot.
config_version = 1

[system]
hostname = "$esc_hostname"

[user]
name = "$esc_username"
password = "$esc_user_pw"
password_encrypted = $user_pw_encrypted

[ssh]
enabled = true
password_authentication = false
authorized_keys = [ "$ssh_key_line" ]

[wlan]
ssid = "$esc_ssid"
password = "$esc_wifi_pw"
password_encrypted = false
hidden = false
country = "$PI_COUNTRY"

[locale]
keymap = "us"
timezone = "$PI_TIMEZONE"
EOF
umask 022

echo
echo "Configuration written. Checking it landed:"
grep -E "^(hostname|name|ssid|country|enabled|password_authentication) *=" "$boot/custom.toml" \
  | sed 's/^/  /'
echo "  authorized_keys: $(grep -c ssh- "$boot/custom.toml") key(s)"
echo "  wlan password:   $(grep -q '^password = ""' "$boot/custom.toml" && echo MISSING || echo set)"

sync
diskutil eject "$PI_DISK" >/dev/null && echo && echo "Card ejected. Put it in the Pi and power up."
echo
echo "First boot resizes the filesystem and reboots itself — give it 2-3 minutes, then:"
echo "  ping -c2 ${PI_HOSTNAME}.local"
echo "  task pi:identity HOST=${PI_HOSTNAME}.local"
