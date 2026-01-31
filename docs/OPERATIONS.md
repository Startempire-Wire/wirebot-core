# Wirebot Operations Guide

> **Production operations: systemd service, launcher, logs, restart procedures.**

---

## Service Architecture

```
┌────────────────────────────────────┐
│  Cloudflare Tunnel                 │
│  cloudflared-wirebot.service       │
│  helm.wirebot.chat → 127.0.0.1:18789  │
│  api.wirebot.chat  → localhost:8100    │
└──────────────┬─────────────────────┘
               │
┌──────────────▼─────────────────────┐
│  Clawdbot Gateway                  │
│  clawdbot-gateway.service          │
│  Port: 18789 (loopback)           │
│  User: wirebot                     │
│  PID type: simple (Node.js)       │
└────────────────────────────────────┘
```

---

## Systemd Service

### Unit File

**Path:** `/etc/systemd/system/clawdbot-gateway.service`

Key properties:
- **Type:** `simple` (foreground Node.js process)
- **User:** `wirebot`
- **ExecStartPre:** `+/data/wirebot/bin/inject-gateway-secrets.sh` (runs as root, injects secrets from rbw)
- **ExecStart:** `/data/wirebot/bin/clawdbot-gateway.sh`
- **Restart:** `on-failure` (10s delay)
- **Hardening:** `ProtectSystem=strict`, `ProtectHome=read-only`, `NoNewPrivileges=true`, `PrivateTmp=true`
- **ReadWritePaths:** `/data/wirebot`, `/home/wirebot/logs`, `/home/wirebot/.nvm`, `/run/wirebot`

### Common Commands

```bash
# Service lifecycle
systemctl start clawdbot-gateway
systemctl stop clawdbot-gateway
systemctl restart clawdbot-gateway
systemctl status clawdbot-gateway

# Enable/disable on boot
systemctl enable clawdbot-gateway
systemctl disable clawdbot-gateway

# View recent logs
journalctl -u clawdbot-gateway -n 50 --no-pager

# Follow live logs
journalctl -u clawdbot-gateway -f
```

---

## Launcher Script

**Path:** `/data/wirebot/bin/clawdbot-gateway.sh`

The launcher:
1. Sources NVM (Node 22 via `/home/wirebot/.nvm/nvm.sh`)
2. Sets `CLAWDBOT_STATE_DIR`, `CLAWDBOT_CONFIG_PATH`, `CLAWDBOT_GATEWAY_PORT`
3. Sources runtime secrets from `/run/wirebot/gateway.env` (written by ExecStartPre)
4. Sets `NODE_OPTIONS=--max-old-space-size=1024` (prevent OOM)
5. Execs `clawdbot gateway run --port 18789 --bind loopback`

**Note:** The launcher runs as `wirebot` user. Secrets are injected by root via `inject-gateway-secrets.sh` before the main process starts.

---

## Secret Injection (ExecStartPre)

**Path:** `/data/wirebot/bin/inject-gateway-secrets.sh`

Runs as root (`+` prefix in systemd unit). Pulls secrets from Bitwarden vault via `rbw`:

1. Creates `/run/wirebot/` (tmpfs, mode 700, wirebot-owned)
2. Retrieves `OPENROUTER_API_KEY` from rbw vault
3. Writes `/run/wirebot/gateway.env` (mode 600, wirebot-owned)

**Per server policy:** All new secrets must come from `rbw` (Bitwarden CLI). No plaintext `.env` files.

See [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) for full details.

---

## Log Files

| Log | Path | Contents |
|-----|------|----------|
| Gateway log | `/home/wirebot/logs/clawdbot-gateway.log` | Clawdbot gateway stdout/stderr |
| Clawdbot daily log | `/tmp/clawdbot/clawdbot-YYYY-MM-DD.log` | Clawdbot internal rotating log |
| Systemd journal | `journalctl -u clawdbot-gateway` | Service start/stop/crash events |

### Log Rotation

Gateway log is appended by systemd (`StandardOutput=append:`). Manual rotation:

```bash
# Rotate (truncate, gateway keeps writing)
as-user wirebot 'cp ~/logs/clawdbot-gateway.log ~/logs/clawdbot-gateway.log.1 && > ~/logs/clawdbot-gateway.log'

# Or restart to get a clean log
systemctl restart clawdbot-gateway
```

---

## Startup Timeline

Typical gateway startup takes **15–25 seconds**:

1. **0s** — ExecStartPre injects secrets from rbw
2. **2s** — systemd starts main process (clawdbot launcher)
3. **3s** — NVM loads, Node starts clawdbot binary
4. **10–20s** — Clawdbot initializes (config validation, channel setup, skill loading)
5. **15–25s** — Gateway listening on ws://127.0.0.1:18789

**Important:** Health checks and port probes must account for this delay.

---

## Restart Procedures

### Graceful Restart (Preferred)

```bash
systemctl restart clawdbot-gateway
```

Sends SIGTERM → gateway shuts down cleanly → systemd starts fresh instance.

### Force Kill (Last Resort)

```bash
systemctl kill -s SIGKILL clawdbot-gateway
systemctl start clawdbot-gateway
```

### Config Reload (Hot)

Clawdbot supports hot-reload for safe config changes:

```bash
# Via CLI
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot gateway call config.patch --params '"'"'{"raw": "{...}", "baseHash": "<hash>"}'"'"''
```

Or edit config and let the file watcher pick it up (default `gateway.reload.mode: "hybrid"`).

---

## Cloudflare Tunnel

### Service

**Unit:** `cloudflared-wirebot.service`
**Config:** `/etc/cloudflared/wirebot.yml`
**Tunnel ID:** `57df17a8-b9d1-4790-bab9-8157ac51641b`

### Routes

| Hostname | Origin |
|----------|--------|
| `helm.wirebot.chat` | `http://127.0.0.1:18789` (Clawdbot Gateway) |
| `api.wirebot.chat` | `http://localhost:8100` (not yet active) |

### Commands

```bash
systemctl restart cloudflared-wirebot
systemctl status cloudflared-wirebot
```

---

## File Permissions Summary

| Path | Mode | Owner | Purpose |
|------|------|-------|---------|
| `/data/wirebot/users/verious/` | `700` | wirebot | State dir root |
| `/data/wirebot/users/verious/clawdbot.json` | `600` | wirebot | Gateway config |
| `/data/wirebot/users/verious/agents/*/agent/auth-profiles.json` | `600` | wirebot | Auth secrets |
| `/data/wirebot/bin/clawdbot-gateway.sh` | `750` | wirebot | Launcher script |
| `/data/wirebot/bin/inject-gateway-secrets.sh` | `700` | root | Secret injector |
| `/run/wirebot/gateway.env` | `600` | wirebot | Runtime secrets (tmpfs) |
| `/etc/systemd/system/clawdbot-gateway.service` | `644` | root | Systemd unit |

---

## Environment Variables

| Variable | Value | Source |
|----------|-------|--------|
| `CLAWDBOT_STATE_DIR` | `/data/wirebot/users/verious` | Launcher script |
| `CLAWDBOT_CONFIG_PATH` | `/data/wirebot/users/verious/clawdbot.json` | Launcher script |
| `CLAWDBOT_GATEWAY_PORT` | `18789` | Launcher script |
| `OPENROUTER_API_KEY` | `sk-or-v1-...` | rbw vault → `/run/wirebot/gateway.env` |
| `NODE_OPTIONS` | `--max-old-space-size=1024` | Launcher script |
| `HOME` | `/home/wirebot` | Systemd Environment= |

---

## See Also

- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Secret management + auth profiles
- [MONITORING.md](./MONITORING.md) — Health checks + alerting
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Common failures + fixes
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's deployed vs planned
- [INSTALLATION.md](./INSTALLATION.md) — Initial setup procedure
