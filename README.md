# Wirebot Core

> **Wirebot is a private AI operating partner for serious founders—built on Clawdbot and integrated with the Startempire Wire ecosystem.**

## Overview

Wirebot is **not** a generic chatbot, Discord toy, or prompt playground.

Wirebot **is**: persistent, stage-aware, context-rich, accountability-driven, intentionally restrained in public, deeply powerful in private.

## Runtime Foundation

Wirebot uses **Clawdbot** as the runtime/gateway:
- Clawdbot provides: channels, gateway, skills system, sessions, cron, Control UI
- Wirebot provides: skills + network intelligence + product shell

## Current Status

**Phase 0 (Foundation)** — 🟡 In Progress

| Component | Status |
|-----------|--------|
| Clawdbot Gateway | ✅ Running (systemd, port 18789) |
| Cloudflare Tunnel | ✅ helm.wirebot.chat active |
| Auth (Anthropic + OpenRouter) | ✅ OAuth + API key from rbw |
| Skills (4 loaded) | ✅ core, accountability, memory, network |
| Letta Server | ❌ Not deployed |
| Mem0 Server | ❌ Not deployed |
| WordPress Plugin | ❌ Not started |

See [CURRENT_STATE.md](docs/CURRENT_STATE.md) for full details.

## Quick Start (Operations)

```bash
# Check gateway status
systemctl status clawdbot-gateway

# Restart gateway
systemctl restart clawdbot-gateway

# Health check
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot gateway probe'

# Auth check
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot models status --probe'
```

## Key Paths

| Path | Purpose |
|------|---------|
| `/data/wirebot/users/verious/clawdbot.json` | Gateway config |
| `/data/wirebot/users/verious/` | State directory |
| `/data/wirebot/bin/` | Launcher + secret injector scripts |
| `/home/wirebot/wirebot-core/skills/` | Wirebot skills |
| `/home/wirebot/logs/clawdbot-gateway.log` | Gateway log |
| `/etc/systemd/system/clawdbot-gateway.service` | Systemd unit |
| `/etc/cloudflared/wirebot.yml` | Tunnel config |

## Documentation

### Operations & Infrastructure (New)

| Document | Purpose |
|----------|---------|
| [OPERATIONS.md](docs/OPERATIONS.md) | Systemd service, launcher, logs, restart procedures |
| [AUTH_AND_SECRETS.md](docs/AUTH_AND_SECRETS.md) | rbw integration, auth profiles, OAuth sync |
| [CURRENT_STATE.md](docs/CURRENT_STATE.md) | What's deployed vs planned |
| [MONITORING.md](docs/MONITORING.md) | Health checks, probes, alerting patterns |
| [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common failures, OOM, auth errors |
| [INSTALLATION.md](docs/INSTALLATION.md) | Full production install procedure |

### Architecture & Design

| Document | Purpose |
|----------|---------|
| [VISION.md](docs/VISION.md) | Unified high-level vision |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture + service topology |
| [GATEWAY.md](docs/GATEWAY.md) | Clawdbot gateway config reference |
| [CAPABILITIES.md](docs/CAPABILITIES.md) | Feature / capability matrix |
| [TRUST_MODES.md](docs/TRUST_MODES.md) | Trust mode enforcement |
| [MEMORY.md](docs/MEMORY.md) | Memory stack (Clawdbot + Letta + Mem0) |
| [CLAWDBOT_MEMORY_DEEP_DIVE.md](docs/CLAWDBOT_MEMORY_DEEP_DIVE.md) | Clawdbot memory internals |

### Business & Rollout

| Document | Purpose |
|----------|---------|
| [DUAL_TRACK_PLAN.md](docs/DUAL_TRACK_PLAN.md) | Standalone + Network tracks |
| [LAUNCH_ORDER.md](docs/LAUNCH_ORDER.md) | Phase roadmap (with status) |
| [NETWORK_INTEGRATION.md](docs/NETWORK_INTEGRATION.md) | Startempire Wire integration |
| [PLUGIN.md](docs/PLUGIN.md) | WordPress plugin specification |

### Configuration & Provisioning

| Document | Purpose |
|----------|---------|
| [PROVISIONING.md](docs/PROVISIONING.md) | User provisioning (shared + dedicated) |
| [SHARED_GATEWAY_CONFIG.md](docs/SHARED_GATEWAY_CONFIG.md) | Sample multi-tenant config |
| [DEDICATED_GATEWAY_CONFIG.md](docs/DEDICATED_GATEWAY_CONFIG.md) | Sample per-user config |
| [PAIRING_ALLOWLIST.md](docs/PAIRING_ALLOWLIST.md) | DM pairing + allowlists |
| [WP_PAIRING_FLOW.md](docs/WP_PAIRING_FLOW.md) | WordPress auto-approve flow |
| [SMS_OPTIONS.md](docs/SMS_OPTIONS.md) | SMS alternatives (no A2P 10DLC) |

### Integrations

| Document | Purpose |
|----------|---------|
| [LETTA_INTEGRATION.md](docs/LETTA_INTEGRATION.md) | Letta agent integration |
| [MEM0_PLUGIN.md](docs/MEM0_PLUGIN.md) | Mem0 plugin details |

## Repository Structure

```
wirebot-core/
├── docs/                    # 25 documentation files
│   ├── OPERATIONS.md        # ← Start here for ops
│   ├── CURRENT_STATE.md     # ← Start here for status
│   ├── ARCHITECTURE.md      # ← Start here for architecture
│   └── ...
├── skills/                  # Wirebot SKILL.md files
│   ├── wirebot-core/
│   ├── wirebot-accountability/
│   ├── wirebot-memory/
│   └── wirebot-network/
├── plugins/                 # Clawdbot plugins
│   ├── memory-mem0/         # Mem0 memory slot (skeleton)
│   └── sms-twilio/          # SMS plugin (skeleton)
├── provisioning/            # Provisioning scripts
│   └── register-user.sh     # Shared gateway registration
└── README.md
```

## Related Repositories

- [Startempire-Wire-Network](https://github.com/Startempire-Wire/Startempire-Wire-Network) — Chrome Extension
- [Startempire-Wire-Network-Ring-Leader](https://github.com/Startempire-Wire/Startempire-Wire-Network-Ring-Leader) — Central messenger plugin
- [Startempire-Wire-Network-Connect](https://github.com/Startempire-Wire/Startempire-Wire-Network-Connect) — Member site connection plugin

## Core Principle

> **Wirebot scales trust as deliberately as it scales capability — because the closer it gets to a founder's life and business, the more carefully it must be designed.**

## License

Proprietary — Startempire Wire

---

*Part of the Startempire Wire ecosystem*
