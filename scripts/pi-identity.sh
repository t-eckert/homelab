#!/usr/bin/env bash
#
# Capture a Raspberry Pi's immutable identity and print it as a Markdown block
# ready to paste into notebook/Hardware.md.
#
# Hardware.md used to key each machine on its IP address. IPs change whenever the
# router does, which is how we ended up unable to tell the 1 GB Pi from the 4 GB
# one. Serial number and MAC addresses never change, so record those instead and
# treat the IP as a "last seen" note.
#
# Usage:
#   PI_HOST=sdr.local ./scripts/pi-identity.sh     # over SSH
#   ./scripts/pi-identity.sh                       # locally, on the Pi itself
#
# Environment:
#   PI_HOST  Host to SSH into. If unset, runs against the local machine.
#   PI_USER  SSH user (default: current user).

set -euo pipefail

PI_HOST="${PI_HOST:-}"
PI_USER="${PI_USER:-$(id -un)}"

# Runs on the Pi. Emits `key=value` lines rather than delimited columns: any
# field can legitimately be empty (a Pi with no wlan0, say), and empty columns
# are unreliable to parse — tab is IFS whitespace, so `read` silently collapses
# consecutive tabs and shifts every later value into the wrong variable.
probe='
  set -u

  emit() { printf "%s=%s\n" "$1" "$2"; }

  read_or() {
    if [ -r "$1" ]; then tr -d "\0" < "$1"; else printf "%s" "$2"; fi
  }

  emit model    "$(read_or /proc/device-tree/model unknown)"
  emit revision "$(awk "/^Revision/ {print \$3}" /proc/cpuinfo)"
  emit serial   "$(awk "/^Serial/ {print \$3}" /proc/cpuinfo)"
  emit mem_kb   "$(awk "/^MemTotal/ {print \$2}" /proc/meminfo)"
  emit eth      "$(read_or /sys/class/net/eth0/address "")"
  emit wlan     "$(read_or /sys/class/net/wlan0/address "")"
  emit ip       "$(hostname -I 2>/dev/null | awk "{print \$1}")"
  emit host     "$(hostname)"
  emit kernel   "$(uname -srm)"
  emit os       "$( (. /etc/os-release 2>/dev/null && printf "%s" "${PRETTY_NAME:-unknown}") || printf unknown )"
'

if [[ -n "$PI_HOST" ]]; then
  raw=$(ssh -o BatchMode=yes -o ConnectTimeout=10 "${PI_USER}@${PI_HOST}" "$probe")
else
  raw=$(bash -c "$probe")
fi

model="" revision="" serial="" mem_kb="" eth="" wlan="" ip="" host="" os="" kernel=""

while IFS= read -r line; do
  key=${line%%=*}
  value=${line#*=}
  case "$key" in
    model) model=$value ;;
    revision) revision=$value ;;
    serial) serial=$value ;;
    mem_kb) mem_kb=$value ;;
    eth) eth=$value ;;
    wlan) wlan=$value ;;
    ip) ip=$value ;;
    host) host=$value ;;
    os) os=$value ;;
    kernel) kernel=$value ;;
  esac
done <<<"$raw"

if [[ ! $mem_kb =~ ^[0-9]+$ ]]; then
  echo "ERROR: could not read MemTotal from ${PI_HOST:-localhost}." >&2
  echo "       Is this a Linux machine? Raw probe output:" >&2
  printf '%s\n' "$raw" >&2
  exit 1
fi

# Round MemTotal up to the marketed size. MemTotal always reads low because the
# GPU carve-out and kernel reservations come off the top — a 1 GB Pi reports
# about 926 MB, a 4 GB one about 3794 MB.
mem_mb=$((mem_kb / 1024))
if   [[ $mem_mb -lt 1200 ]]; then ram="1 GB"
elif [[ $mem_mb -lt 2400 ]]; then ram="2 GB"
elif [[ $mem_mb -lt 4600 ]]; then ram="4 GB"
else                              ram="8 GB"
fi

# On a Pi 4 B the first nibble of the revision code encodes the memory size,
# which independently corroborates MemTotal.
case "${revision:0:1}" in
  a) rev_ram="1 GB" ;;
  b) rev_ram="2 GB" ;;
  c) rev_ram="4 GB" ;;
  d) rev_ram="8 GB" ;;
  *) rev_ram="" ;;
esac

cat <<EOF
### ${model}

Serial: \`${serial:-unknown}\`
MAC (eth0): \`${eth:-n/a}\`
MAC (wlan0): \`${wlan:-n/a}\`
Board revision: \`${revision:-unknown}\`
RAM: ${ram}
ARCH: ARMv8 64 bit
Hostname: \`${host}\`
Last seen at: \`${ip:-unknown}\`
OS: ${os}
Kernel: ${kernel}
EOF

if [[ -n $rev_ram && $rev_ram != "$ram" ]]; then
  {
    echo
    echo "WARNING: MemTotal reports ${ram} but revision code ${revision} says ${rev_ram}."
    echo "         Trust the revision code and check this board by hand."
  } >&2
fi
