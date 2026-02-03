#!/bin/bash
# Sync Obsidian vault from MacBook via Tailscale
# Runs daily at 4 AM PT
set -euo pipefail

LOG="/data/wirebot/logs/obsidian-sync.log"
REMOTE="vsmith@100.113.124.59"
REMOTE_PATH="/Volumes/Macintosh HD/Users/vsmith/Documents/Obsidian/"
LOCAL_PATH="/data/wirebot/obsidian/"
SSH_OPTS="-o IdentitiesOnly=yes -i /root/.ssh/id_rsa -o ConnectTimeout=10 -o StrictHostKeyChecking=no"

echo "[$(date -Iseconds)] Starting Obsidian sync" >> "$LOG"

# Check if Mac is reachable
if ! ssh $SSH_OPTS "$REMOTE" "echo OK" >/dev/null 2>&1; then
  echo "[$(date -Iseconds)] Mac offline, skipping" >> "$LOG"
  exit 0
fi

rsync -avz --delete \
  --exclude='.obsidian/' --exclude='.trash/' --exclude='_resources/' --exclude='.git/' \
  -e "ssh $SSH_OPTS" \
  "$REMOTE:$REMOTE_PATH" "$LOCAL_PATH" >> "$LOG" 2>&1

CHANGED=$(tail -20 "$LOG" | grep -c "\.md$" || true)
echo "[$(date -Iseconds)] Sync complete. $CHANGED files changed." >> "$LOG"

# If files changed, trigger re-ingestion
if [ "$CHANGED" -gt 0 ]; then
  TOKEN="${WIREBOT_OPERATOR_TOKEN:-65b918ba-baf5-4996-8b53-6fb0f662a0c3}"
  curl -s -X POST "http://localhost:8100/v1/pairing/scan-vault" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" >> "$LOG" 2>&1
  echo "[$(date -Iseconds)] Triggered vault re-scan" >> "$LOG"
fi
