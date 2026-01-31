#!/usr/bin/env bash
set -euo pipefail

# Usage: register-user.sh <user_id> <channel> <peer_id>
user_id=${1:?user_id required}
channel=${2:?channel required}
peer_id=${3:?peer_id required}

CONFIG=${CLAWDBOT_CONFIG_PATH:-$HOME/.clawdbot/clawdbot.json}

# Add agent
clawdbot config set agents.list --json "$(clawdbot config get agents.list --json | jq -c '. + [{"id":"'$user_id'"}]')"

# Add binding
clawdbot config set bindings --json "$(clawdbot config get bindings --json | jq -c '. + [{"agentId":"'$user_id'","match":{"channel":"'$channel'","peer":{"kind":"dm","id":"'$peer_id'"}}}]')"

echo "Registered user $user_id for $channel:$peer_id"
