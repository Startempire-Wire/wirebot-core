# Installation & Setup

> **Complete production install procedure: Node, Clawdbot, config, auth, systemd, tunnel.**

---

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| **Node.js** | 22+ | Via nvm (recommended) |
| **OS** | AlmaLinux 8+ / any systemd Linux | Virtuozzo VPS supported |
| **User** | `wirebot` (cPanel account) | All runtime files owned by this user |
| **rbw** | 1.15+ | Bitwarden CLI for secret management (server-wide) |
| **cloudflared** | latest | Cloudflare tunnel (if public access needed) |

---

## Step 1: Node.js (via nvm)

```bash
# As wirebot user
su - wirebot

# Install nvm
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
source ~/.nvm/nvm.sh

# Install Node 22
nvm install 22
nvm alias default 22
node --version  # v22.x.x
```

---

## Step 2: Install Clawdbot

```bash
# As wirebot user
npm install -g clawdbot@latest
clawdbot --version  # 2026.x.x
```

Binary installed to: `/home/wirebot/.nvm/versions/node/v22.x.x/bin/clawdbot`

---

## Step 3: Create State Directory

```bash
# As root (directory under /data is root-managed)
mkdir -p /data/wirebot/users/verious
mkdir -p /data/wirebot/bin
chown -R wirebot:wirebot /data/wirebot/users/verious
chmod 700 /data/wirebot/users/verious
```

---

## Step 4: Write Config

```bash
# As wirebot user (or use as-user wirebot)
cat > /data/wirebot/users/verious/clawdbot.json << 'EOF'
{
  "gateway": {
    "port": 18789,
    "mode": "local",
    "bind": "loopback",
    "controlUi": { "allowInsecureAuth": false },
    "auth": {
      "mode": "token",
      "token": "<generate-uuid>",
      "allowTailscale": true
    },
    "trustedProxies": ["127.0.0.1"]
  },
  "agents": {
    "defaults": { "maxConcurrent": 4 },
    "list": [{ "id": "verious", "name": "Wirebot: verious" }]
  },
  "skills": {
    "load": { "extraDirs": ["/home/wirebot/wirebot-core/skills"] }
  },
  "plugins": { "allow": ["memory-core"] }
}
EOF

chmod 600 /data/wirebot/users/verious/clawdbot.json
```

Generate a token:
```bash
python3 -c "import uuid; print(uuid.uuid4())"
# Or: openssl rand -hex 32
```

---

## Step 5: Set Up Auth Profiles

Create the agent directory structure and auth profiles:

```bash
# As root
mkdir -p /data/wirebot/users/verious/agents/{main,verious}/agent
chown -R wirebot:wirebot /data/wirebot/users/verious/agents
chmod -R 700 /data/wirebot/users/verious/agents
```

### Anthropic OAuth (from Claude Code)

```bash
# Extract Claude Code credentials
ACCESS=$(jq -r '.claudeAiOauth.accessToken' /root/.claude/.credentials.json)
REFRESH=$(jq -r '.claudeAiOauth.refreshToken' /root/.claude/.credentials.json)
EXPIRES=$(jq -r '.claudeAiOauth.expiresAt' /root/.claude/.credentials.json)
```

### OpenRouter API Key (from rbw vault)

```bash
OR_KEY=$(~/.cargo/bin/rbw get "openrouter.ai" --field "Cursor Code Editor API Key")
```

### Write auth-profiles.json

```bash
cat > /tmp/auth-profiles.json << EOF
{
  "profiles": {
    "anthropic:claude-cli": {
      "type": "oauth",
      "provider": "anthropic",
      "access": "${ACCESS}",
      "refresh": "${REFRESH}",
      "expires": ${EXPIRES}
    },
    "openrouter:default": {
      "type": "api_key",
      "provider": "openrouter",
      "key": "${OR_KEY}"
    }
  },
  "usageStats": {}
}
EOF

# Install for both agent dirs
for agent in main verious; do
  cp /tmp/auth-profiles.json /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
  chown wirebot:wirebot /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
  chmod 600 /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
done
rm /tmp/auth-profiles.json
```

See [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) for full details.

---

## Step 6: Create Launcher Scripts

### Gateway Launcher

See [OPERATIONS.md](./OPERATIONS.md#launcher-script) for the full script.

**Path:** `/data/wirebot/bin/clawdbot-gateway.sh` (wirebot:wirebot, mode 750)

### Secret Injector

See [OPERATIONS.md](./OPERATIONS.md#secret-injection-execstartpre) for the full script.

**Path:** `/data/wirebot/bin/inject-gateway-secrets.sh` (root:root, mode 700)

---

## Step 7: Install Systemd Service

```bash
# Copy unit file (see OPERATIONS.md for full content)
cat > /etc/systemd/system/clawdbot-gateway.service << 'EOF'
[Unit]
Description=Clawdbot Gateway (Wirebot)
After=network-online.target cloudflared-wirebot.service
Wants=network-online.target
StartLimitIntervalSec=300
StartLimitBurst=5

[Service]
Type=simple
ExecStartPre=+/data/wirebot/bin/inject-gateway-secrets.sh
User=wirebot
Group=wirebot
ExecStart=/data/wirebot/bin/clawdbot-gateway.sh
WorkingDirectory=/home/wirebot
Restart=on-failure
RestartSec=10
StandardOutput=append:/home/wirebot/logs/clawdbot-gateway.log
StandardError=append:/home/wirebot/logs/clawdbot-gateway.log
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=read-only
ReadWritePaths=/data/wirebot /home/wirebot/logs /home/wirebot/.nvm /run/wirebot
PrivateTmp=true
Environment=HOME=/home/wirebot
Environment=CLAWDBOT_STATE_DIR=/data/wirebot/users/verious
Environment=CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json
Environment=CLAWDBOT_GATEWAY_PORT=18789

[Install]
WantedBy=multi-user.target
EOF

# Create log directory
as-user wirebot 'mkdir -p ~/logs'

# Enable and start
systemctl daemon-reload
systemctl enable clawdbot-gateway
systemctl start clawdbot-gateway
```

---

## Step 8: Set Up Cloudflare Tunnel (Optional)

```bash
# Install cloudflared
# (already installed system-wide)

# Create tunnel config
cat > /etc/cloudflared/wirebot.yml << 'EOF'
tunnel: <tunnel-id>
credentials-file: /etc/cloudflared/<tunnel-id>.json

ingress:
  - hostname: helm.wirebot.chat
    service: http://127.0.0.1:18789
  - hostname: api.wirebot.chat
    service: http://localhost:8100
  - service: http_status:404
EOF

# Create systemd service for the tunnel
# (see /etc/systemd/system/cloudflared-wirebot.service)

systemctl enable cloudflared-wirebot
systemctl start cloudflared-wirebot
```

---

## Step 9: Verify

```bash
# Service running
systemctl status clawdbot-gateway

# Port listening (wait 15–25s for startup)
ss -tlnp | grep 18789

# Gateway probe
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot gateway probe'

# Auth probe
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot models status --probe'

# Public URL (if tunnel configured)
curl -sI https://helm.wirebot.chat
```

---

## See Also

- [OPERATIONS.md](./OPERATIONS.md) — Day-to-day service management
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Auth profile setup
- [GATEWAY.md](./GATEWAY.md) — Gateway configuration
- [MONITORING.md](./MONITORING.md) — Post-install health checks
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — If something goes wrong
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What should be running after install
