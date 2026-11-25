#!/bin/sh
# Builder Sprite setup (Miniapp Studio, Phase B).
#
# Runs ONCE per sprite via sprite exec, as root. Idempotent: safe to
# re-run after a partial failure; every step checks current state first.
# Fails closed with a clear message whenever a prerequisite is missing —
# provisioning must refuse rather than limp along half-configured.
#
# Pinned inputs (mirrored in the Phase B evidence; re-verify per the
# amendment's rules before changing):
set -eu

OPENCODE_VERSION="v1.19.6"
OPENCODE_ASSET="opencode-linux-x64.tar.gz"
OPENCODE_SHA256="47734e08968b30cc22a0bffb6d9feeed76ad4aa743213a12122040392d09a057"
OPENCODE_URL="https://github.com/anomalyco/opencode/releases/download/v1.19.6/opencode-linux-x64.tar.gz"
PLUGIN_REPO="https://github.com/superfly/sprites-opencode-plugin.git"
PLUGIN_COMMIT="f63a85a7da9fa2f42b21877ab58b35d18ccaa682"

GW_USER="studio-gw"
AGENT_USER="studio-agent"
IO_GROUP="studio-io"
STAGING="/srv/studio-staging"
GW_HOME="/srv/studio-gateway"
AGENT_HOME="/home/studio-agent"

fail() { echo "setup: $*" >&2; exit 1; }

# --- 1. Privilege and prerequisites (fail closed, with reasons). ---
[ "$(id -u)" = "0" ] || fail "must run as root (sprite exec privilege mechanism is a live-verification item; see runbook)"
for cmd in curl tar git sha256sum install getent useradd grep awk; do
  command -v "$cmd" >/dev/null 2>&1 || fail "missing prerequisite: $cmd (system packages are a runbook prerequisite)"
done
command -v node >/dev/null 2>&1 || fail "missing prerequisite: node >= 20 (runbook prerequisite)"
command -v bun >/dev/null 2>&1 || fail "missing prerequisite: bun (plugin auto-install; runbook prerequisite)"
[ -f "$STAGING/studio-gateway" ] || fail "missing staged gateway binary"
[ -f "$STAGING/opencode.json" ] || fail "missing staged opencode.json"
[ -f "$STAGING/builder.md" ] || fail "missing staged builder.md"

# --- 2. Identities (separate Unix users per the merged amendment). ---
getent group "$IO_GROUP" >/dev/null 2>&1 || groupadd "$IO_GROUP"
id "$GW_USER" >/dev/null 2>&1 || useradd -m -U -G "$IO_GROUP" -s /bin/false "$GW_USER"
id "$AGENT_USER" >/dev/null 2>&1 || useradd -m -U -G "$IO_GROUP" -s /bin/bash "$AGENT_USER"
usermod -a -G "$IO_GROUP" "$GW_USER" 2>/dev/null || true
usermod -a -G "$IO_GROUP" "$AGENT_USER" 2>/dev/null || true

# --- 3. Directory layout. ---
# Staging may hold world-readable key copies from provisioning transport:
# lock it down before touching key material, and remove the staging
# copies once installed (B1: no lingering key duplicates).
chmod 0700 "$STAGING" || fail "cannot lock down staging dir"
for keys in "$STAGING"/keys-*.json; do
  [ -e "$keys" ] || continue
  chmod 0600 "$keys" || fail "cannot lock down staged $keys"
done
mkdir -p "$GW_HOME/keys" "$GW_HOME/state" /srv/studio-agent/builds "$AGENT_HOME/.config/opencode"
chown "$GW_USER:$GW_USER" "$GW_HOME/keys" "$GW_HOME/state"
chmod 0700 "$GW_HOME/keys" "$GW_HOME/state"
chown -R "$AGENT_USER:$AGENT_USER" /srv/studio-agent "$AGENT_HOME/.config"
install -m 0755 -o root -g root "$STAGING/studio-gateway" "$GW_HOME/studio-gateway"
install -m 0644 -o root -g root "$STAGING/builder.md" "$GW_HOME/builder.md"
install -m 0600 -o "$AGENT_USER" -g "$AGENT_USER" "$STAGING/opencode.json" "$AGENT_HOME/.config/opencode/opencode.json"
# Per-build HMAC key files accumulate (one file per build, never merged).
# Installed atomically (temp + rename): the live gateway re-reads the
# keys dir per lookup, so a torn file would flake commands.
rm -f "$GW_HOME/keys"/.keys-*.tmp
for keys in "$STAGING"/keys-*.json; do
  [ -e "$keys" ] || continue
  base=${keys##*/}
  install -m 0600 -o "$GW_USER" -g "$GW_USER" "$keys" "$GW_HOME/keys/.$base.tmp" || fail "key stage failed: $base"
  mv "$GW_HOME/keys/.$base.tmp" "$GW_HOME/keys/$base" || fail "key install failed: $base"
done
[ -n "$(ls -A "$GW_HOME/keys" 2>/dev/null)" ] || fail "no gateway key files staged"
rm -f "$STAGING"/keys-*.json
for keys in "$STAGING"/keys-*.json; do
  [ -e "$keys" ] && fail "staging key cleanup failed: $keys"
done
[ "$(ls -ld "$STAGING" | awk '{print $1}')" = "drwx------" ] || fail "staging dir not 0700"

# --- 4. OpenCode (pinned asset + SHA, skipped when already correct). ---
if command -v opencode >/dev/null 2>&1 && opencode --version 2>/dev/null | grep -q "1.19.6"; then
  echo "setup: opencode v1.19.6 already installed"
else
  cd /tmp
  curl -fsSL -o "$OPENCODE_ASSET" "$OPENCODE_URL" || fail "opencode download failed"
  echo "$OPENCODE_SHA256  $OPENCODE_ASSET" | sha256sum -c - || fail "opencode checksum mismatch"
  tar -xzf "$OPENCODE_ASSET" -C /usr/local/bin || fail "opencode extract failed"
  rm -f "$OPENCODE_ASSET"
  opencode --version 2>/dev/null | grep -q "1.19.6" || fail "opencode install did not yield v1.19.6"
fi

# --- 5. Official plugin checkout (pinned commit, skipped when current). ---
if [ -d /srv/studio-agent/plugin/.git ]; then
  if [ "$(git -C /srv/studio-agent/plugin rev-parse HEAD 2>/dev/null)" = "$PLUGIN_COMMIT" ]; then
    echo "setup: plugin already at $PLUGIN_COMMIT"
  else
    rm -rf /srv/studio-agent/plugin
    git clone "$PLUGIN_REPO" /srv/studio-agent/plugin || fail "plugin clone failed"
    git -C /srv/studio-agent/plugin checkout "$PLUGIN_COMMIT" || fail "plugin checkout failed"
  fi
else
  git clone "$PLUGIN_REPO" /srv/studio-agent/plugin || fail "plugin clone failed"
  git -C /srv/studio-agent/plugin checkout "$PLUGIN_COMMIT" || fail "plugin checkout failed"
fi
chown -R "$AGENT_USER:$AGENT_USER" /srv/studio-agent/plugin

# --- 6. OAuth state report (bootstrap itself is the owner ceremony). ---
if [ -f "$AGENT_HOME/.local/share/opencode/mcp-auth.json" ]; then
  echo "setup: MCP OAuth tokens present"
else
  echo "setup: WARNING: no MCP OAuth tokens yet; complete the owner bootstrap ceremony (runbook P-E ladder) before agent runs need sprites_* tools"
fi

echo "setup: complete"
