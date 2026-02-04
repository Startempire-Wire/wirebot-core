# Wirebot Gateway (OpenClaw)

> **OpenClaw is the gateway. No custom gateway build.**

---

## Overview

Wirebot uses the **OpenClaw Gateway** as its control plane.

Capabilities built-in:
- WebSocket control plane
- Channel routing (WhatsApp, Telegram, Discord, Slack, Signal, iMessage, etc.)
- Sessions + history
- Cron + hooks
- Control UI + WebChat
- Model routing + failover
- Skills system
- Memory (markdown + hybrid search)

---

## Production Deployment

### Service

The gateway runs as a **systemd service**:

```bash
systemctl status openclaw-gateway    # Check status
systemctl restart openclaw-gateway   # Restart
systemctl stop openclaw-gateway      # Stop
```

See [OPERATIONS.md](./OPERATIONS.md) for full service management.

### Config File (JSON5)

**Path:** `/data/wirebot/users/verious/openclaw.json`

**Not** the default `~/.openclaw/openclaw.json` — overridden via environment variables:

| Variable | Value |
|----------|-------|
| `OPENCLAW_STATE_DIR` | `/data/wirebot/users/verious` |
| `OPENCLAW_CONFIG_PATH` | `/data/wirebot/users/verious/openclaw.json` |
| `OPENCLAW_GATEWAY_PORT` | `18789` |

### Current Config (Production)

```json5
{
  meta: { lastTouchedVersion: "2026.1.24-3" },
  update: { channel: "dev", checkOnStart: true },
  agents: {
    defaults: { maxConcurrent: 4, subagents: { maxConcurrent: 8 } },
    list: [{ id: "verious", name: "Wirebot: verious" }]
  },
  messages: { ackReactionScope: "group-mentions" },
  commands: { native: "auto", nativeSkills: "auto" },
  gateway: {
    port: 18789,
    mode: "local",
    bind: "loopback",
    controlUi: { allowInsecureAuth: false },
    auth: { mode: "token", token: "<redacted>", allowTailscale: true },
    trustedProxies: ["127.0.0.1"]
  },
  skills: { load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] } },
  plugins: { allow: ["memory-core"] }
}
```

---

## Gateway Auth

OpenClaw supports token/password auth on the WebSocket connection:

```json5
{
  gateway: {
    auth: {
      mode: "token",
      token: "<uuid-token>"
    }
  }
}
```

**Security rules:**
- Gateway token is stored in `openclaw.json` (mode 600, wirebot-owned)
- **WordPress plugin must never expose this token to the client** — use server-side proxy calls
- Token is required even on loopback (OpenClaw default since recent versions)
- Control UI authenticates via `connect.params.auth.token` (stored in browser settings)

### Trusted Proxies

When behind a reverse proxy (e.g., Cloudflare tunnel), set:

```json5
{
  gateway: {
    trustedProxies: ["127.0.0.1"]
  }
}
```

The Cloudflare tunnel (`cloudflared-wirebot`) connects from localhost, so `127.0.0.1` is correct.

---

## Public Access (Cloudflare Tunnel)

The gateway is exposed via Cloudflare tunnel:

| Public URL | Origin | Purpose |
|------------|--------|---------|
| `helm.wirebot.chat` | `http://127.0.0.1:18789` | Control UI + WebChat + WebSocket |
| `api.wirebot.chat` | `http://localhost:8100` | REST API (not yet active) |

Tunnel config: `/etc/cloudflared/wirebot.yml`
Tunnel service: `cloudflared-wirebot.service`

---

## Shared Gateway (Lower Tiers)

Use a **single** OpenClaw gateway with multiple agents and bindings:

```json5
{
  agents: {
    list: [
      { id: "user_1" },
      { id: "user_2" }
    ]
  },
  bindings: [
    { agentId: "user_1", match: { channel: "discord", peer: { kind: "dm", id: "123" } } },
    { agentId: "user_2", match: { channel: "telegram", peer: { kind: "dm", id: "456" } } }
  ],
  channels: {
    discord: { dm: { policy: "pairing" } },
    telegram: { dmPolicy: "pairing" }
  }
}
```

See [SHARED_GATEWAY_CONFIG.md](./SHARED_GATEWAY_CONFIG.md) for a full example.

---

## Dedicated Gateway (Top Tier)

Each top-tier user gets their **own** gateway instance with unique port and state dir:

```
OPENCLAW_STATE_DIR=/data/wirebot/users/<user_id>
OPENCLAW_CONFIG_PATH=/data/wirebot/users/<user_id>/openclaw.json
OPENCLAW_GATEWAY_PORT=18xxx
```

See [DEDICATED_GATEWAY_CONFIG.md](./DEDICATED_GATEWAY_CONFIG.md) and [PROVISIONING.md](./PROVISIONING.md).

---

## HTTP Endpoints (Optional)

OpenClaw can expose OpenAI-compatible REST endpoints:

```json5
{
  gateway: {
    http: {
      endpoints: {
        chatCompletions: { enabled: true },
        responses: { enabled: true }
      }
    }
  }
}
```

---

## Control UI + WebChat

OpenClaw serves Control UI and WebChat by default on the gateway port:

- **Local:** `http://127.0.0.1:18789/`
- **Public:** `https://helm.wirebot.chat/` (via tunnel)

Features: chat, channels, sessions, cron, skills, config editor, logs, debug tools.

See [OpenClaw Control UI docs](https://docs.clawd.bot/web/control-ui) for full reference.

---

## Hot Reload

OpenClaw watches the config file and supports hot-reload:

- `gateway.reload.mode: "hybrid"` (default): hot-apply safe changes, restart for critical ones
- Other modes: `hot`, `restart`, `off`

---

## CLI Commands (Quick Reference)

```bash
# All commands need env vars set
export OPENCLAW_STATE_DIR=/data/wirebot/users/verious
export OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json

# Gateway
openclaw gateway probe          # Deep health check
openclaw gateway health         # Quick health

# Models
openclaw models status          # Auth overview
openclaw models status --probe  # Live auth probe

# Config
openclaw config get             # View config
openclaw doctor                 # Diagnose issues
openclaw doctor --fix           # Auto-fix issues

# Channels
openclaw channels list          # List connected channels
openclaw channels login         # Connect a channel (WhatsApp QR, etc.)

# Sessions
openclaw sessions list          # Active sessions
```

---

## SMS Reality

OpenClaw does **not** include Twilio SMS. See [SMS_OPTIONS.md](./SMS_OPTIONS.md) for alternatives.

---

## See Also

- [OPERATIONS.md](./OPERATIONS.md) — Systemd service, launcher, logs
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Provider auth + secret management
- [MONITORING.md](./MONITORING.md) — Health probes + alerting
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Common gateway issues
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's actually running
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [SHARED_GATEWAY_CONFIG.md](./SHARED_GATEWAY_CONFIG.md) — Multi-tenant config
- [DEDICATED_GATEWAY_CONFIG.md](./DEDICATED_GATEWAY_CONFIG.md) — Per-user config
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [INSTALLATION.md](./INSTALLATION.md) — Initial setup
- [WHITE_LABEL.md](./WHITE_LABEL.md) — Client-facing frontend (uses HTTP + WS APIs)
- [OpenClaw Gateway Docs](https://docs.clawd.bot/gateway) — Foundation reference
