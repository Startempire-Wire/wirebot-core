# Provisioning (Clawdbot)

> **Top tier = dedicated container. Lower tiers = shared gateway.**

---

## State Directory Structure (Production)

All user state lives under `/data/wirebot/users/`:

```
/data/wirebot/
├── bin/
│   ├── clawdbot-gateway.sh          # Launcher (wirebot, 750)
│   └── inject-gateway-secrets.sh    # Secret injector (root, 700)
└── users/
    └── <user_id>/                   # Per-user state (wirebot, 700)
        ├── clawdbot.json            # Gateway config (600)
        ├── credentials/             # Channel pairing + allowlists
        ├── cron/                    # Cron job definitions
        ├── devices/                 # Paired devices
        ├── identity/                # Gateway identity
        ├── sessions/                # Legacy session store
        └── agents/
            ├── main/
            │   └── agent/
            │       └── auth-profiles.json  # Auth secrets (600)
            └── <agentId>/
                ├── agent/
                │   └── auth-profiles.json  # Auth secrets (600)
                └── sessions/
                    └── sessions.json
```

---

## Top Tier: Dedicated Container

### Environment Variables

```bash
CLAWDBOT_STATE_DIR=/data/wirebot/users/<user_id>
CLAWDBOT_CONFIG_PATH=/data/wirebot/users/<user_id>/clawdbot.json
CLAWDBOT_GATEWAY_PORT=<unique-port>
```

### Provisioning Steps

1. Create state dir with correct permissions
2. Write JSON5 config with unique port and gateway token
3. Create agent directories + auth-profiles.json
4. Create launcher + secret injector scripts
5. Install systemd service (unique unit name per user)
6. Enable and start service

### Example Script

```bash
#!/bin/bash
# /data/wirebot/provisioning/provision-dedicated.sh <user_id>
set -euo pipefail

user_id="$1"
state_dir="/data/wirebot/users/${user_id}"
port=$((18000 + $(echo -n "$user_id" | cksum | awk '{print $1}') % 1000))
token=$(python3 -c "import uuid; print(uuid.uuid4())")

# 1. Create state dir
mkdir -p "$state_dir"/{credentials,agents/main/agent}
chown -R wirebot:wirebot "$state_dir"
chmod -R 700 "$state_dir"

# 2. Write config
cat > "$state_dir/clawdbot.json" << EOF
{
  "gateway": {
    "port": ${port},
    "mode": "local",
    "bind": "loopback",
    "auth": { "mode": "token", "token": "${token}" },
    "trustedProxies": ["127.0.0.1"]
  },
  "agents": {
    "list": [{ "id": "${user_id}", "name": "Wirebot: ${user_id}" }]
  },
  "skills": {
    "load": { "extraDirs": ["/home/wirebot/wirebot-core/skills"] }
  },
  "plugins": { "allow": ["memory-core"] }
}
EOF
chmod 600 "$state_dir/clawdbot.json"
chown wirebot:wirebot "$state_dir/clawdbot.json"

# 3. Create auth-profiles.json (using rbw for API keys)
OR_KEY=$(~/.cargo/bin/rbw get "openrouter.ai" --field "Cursor Code Editor API Key" 2>/dev/null || true)
cat > "$state_dir/agents/main/agent/auth-profiles.json" << EOF
{
  "profiles": {
    "openrouter:default": {
      "type": "api_key",
      "provider": "openrouter",
      "key": "${OR_KEY}"
    }
  },
  "usageStats": {}
}
EOF
chmod 600 "$state_dir/agents/main/agent/auth-profiles.json"
chown wirebot:wirebot "$state_dir/agents/main/agent/auth-profiles.json"

echo "Provisioned user ${user_id} on port ${port}"
echo "Token: ${token}"
```

### Per-User Systemd Service

For multiple dedicated containers, create per-user service units:

```bash
# Template: /etc/systemd/system/clawdbot-gateway@.service
# Usage: systemctl start clawdbot-gateway@user_id
```

---

## Lower Tiers: Shared Gateway

### Config Requirements

- `agents.list` — one agent entry per user
- `bindings` — route channel → agent

```json5
{
  agents: { list: [{ id: "user_1" }, { id: "user_2" }] },
  bindings: [
    { agentId: "user_1", match: { channel: "discord", peer: { kind: "dm", id: "123" } } }
  ]
}
```

### Registration Script

```bash
#!/bin/bash
# /home/wirebot/wirebot-core/provisioning/register-user.sh <user_id> <channel> <peer_id>
user_id="$1"
channel="$2"
peer_id="$3"

# Add agent to config via CLI
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot config set agents.list --json "[...existing, {\"id\": \"'$user_id'\"}]"'

# Add binding
# (config.patch via RPC is recommended for atomicity)
```

---

## Pairing / Allowlist

Default DM policy is **pairing** (users must be approved before chatting).

Pairing files stored in: `$CLAWDBOT_STATE_DIR/credentials/`

```bash
# List pending pairing requests
clawdbot pairing list --channel telegram

# Approve a user
clawdbot pairing approve telegram ABCD1234

# Or write allowlist directly
cat > "$CLAWDBOT_STATE_DIR/credentials/telegram-allowFrom.json" << 'EOF'
{ "version": 1, "allowFrom": ["123456789"] }
EOF
```

See [PAIRING_ALLOWLIST.md](./PAIRING_ALLOWLIST.md) and [WP_PAIRING_FLOW.md](./WP_PAIRING_FLOW.md).

---

## Secret Management in Provisioning

Per server policy: **all new secrets via rbw** (Bitwarden vault).

- API keys retrieved from vault during provisioning
- Gateway tokens generated fresh per user (stored in config)
- OAuth tokens require user login (or sync from existing Claude Code credentials)

See [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md).

---

## See Also

- [INSTALLATION.md](./INSTALLATION.md) — Initial server setup
- [OPERATIONS.md](./OPERATIONS.md) — Service management
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Secret handling during provisioning
- [GATEWAY.md](./GATEWAY.md) — Config reference
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Infrastructure models
- [SHARED_GATEWAY_CONFIG.md](./SHARED_GATEWAY_CONFIG.md) — Multi-tenant example
- [DEDICATED_GATEWAY_CONFIG.md](./DEDICATED_GATEWAY_CONFIG.md) — Per-user example
- [PAIRING_ALLOWLIST.md](./PAIRING_ALLOWLIST.md) — DM approval flow
- [WP_PAIRING_FLOW.md](./WP_PAIRING_FLOW.md) — WordPress auto-approve
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's deployed
