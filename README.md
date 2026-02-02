# Wirebot Core

> **Wirebot is a private AI operating partner for serious founders—built on Clawdbot, governed by Focusa cognitive architecture, and integrated with the Startempire Wire ecosystem.**

## What Wirebot Is

Wirebot is a **business operating dashboard** where AI is embedded in the workflow—not a chat window you talk to. It's a persistent, stage-aware, context-rich, accountability-driven partner that helps founders move from Idea → Launch → Growth.

Wirebot is **not**: a generic chatbot, Discord toy, prompt playground, or SaaS wrapper.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Wirebot Architecture                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐    ┌───────────────┐    ┌──────────────────┐ │
│  │  Dashboard    │    │  Clawdbot     │    │  Memory Systems  │ │
│  │  (Frontend)   │───▶│  Gateway      │───▶│                  │ │
│  │  Mobile-first │    │  :18789       │    │  memory-core     │ │
│  └──────────────┘    │              │    │  (embedded,      │ │
│                       │  Skills      │    │   SQLite+vec)    │ │
│  ┌──────────────┐    │  Cron Jobs   │    │                  │ │
│  │  Cloudflare   │    │  Sessions    │    │  Mem0 (:8200)    │ │
│  │  Tunnel       │───▶│  Plugins     │    │  (fact extract)  │ │
│  │  helm.wire-   │    └──────┬───────┘    │                  │ │
│  │  bot.chat     │           │            │  Letta (:8283)   │ │
│  └──────────────┘           │            │  (business state)│ │
│                              │            └──────────────────┘ │
│                    ┌─────────▼──────────┐                      │
│                    │  Focusa Cognitive   │                      │
│                    │  Governance Layer   │                      │
│                    │  (Memory Bridge     │                      │
│                    │   Plugin)           │                      │
│                    └────────────────────┘                       │
│                                                                 │
│  ┌──────────────┐    ┌───────────────┐    ┌──────────────────┐ │
│  │  Clawd       │    │  Focusa State │    │  Startempire     │ │
│  │  Workspace   │    │  /data/wire-  │    │  Wire Ecosystem  │ │
│  │  /home/wire- │    │  bot/focusa-  │    │  .com / .network │ │
│  │  bot/clawd/  │    │  state/       │    │  / Chrome Ext    │ │
│  └──────────────┘    └───────────────┘    └──────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Runtime Foundation

**Clawdbot** is the runtime gateway. It provides: channels, WebSocket/HTTP API, skills system, sessions, cron jobs, model failover, plugin SDK, Control UI. Wirebot provides: business skills, network intelligence, product shell, cognitive governance (Focusa).

### Memory Systems (3 — coordinated via bridge)

| System | Role | Port/Location | Storage |
|--------|------|---------------|---------|
| **memory-core** | Workspace file recall (instant) | Embedded in gateway | SQLite+vec at `/data/wirebot/users/verious/memory/verious.sqlite` |
| **Mem0** | LLM-extracted conversation facts | `:8200` (systemd) | Qdrant at `/data/wirebot/mem0/qdrant` |
| **Letta** | Structured self-editing business state | `:8283` (podman) | PostgreSQL-backed |

Architecture: Write-Through, Read-Cascade. Writes go to the appropriate system. Reads cascade: memory-core (0ms) → Mem0 (200ms) → Letta blocks (100ms).

### Cognitive Governance (Focusa)

Focusa is a cognitive governance layer that preserves focus, intent, and meaning across long-running sessions. It provides:

- **Focus Stack** — hierarchical task attention (one active frame at a time)
- **Focus Gate** — advisory salience filter (surfaces priorities without auto-acting)
- **ASCC** — structured context checkpointing (survives compaction)
- **ECS** — externalized artifact storage (handles, not inline content)
- **UXP/UFI** — transparent experience calibration (no hidden inference)
- **Autonomy Calibration** — earned trust via evidence-based scoring
- **Agent Constitution** — versioned, immutable reasoning charter
- **Constitution Synthesizer** — evidence-driven evolution of agent identity

Full specification: [FOCUSA_WIREBOT_INTEGRATION.md](docs/FOCUSA_WIREBOT_INTEGRATION.md) (152KB, 58 sections, 67/67 spec docs mapped)

## Current Status

**Phase 0 (Foundation)** — ✅ Complete | **Phase 1 (Memory Bridge)** — 🟡 In Progress

| Component | Status | Details |
|-----------|--------|---------|
| Clawdbot Gateway | ✅ Running | systemd, port 18789, v2026.1.24-3 |
| Cloudflare Tunnel | ✅ Active | `helm.wirebot.chat` → `:18789` |
| Auth (Anthropic + OpenRouter) | ✅ Working | OAuth + API key via rbw vault |
| memory-core | ✅ Operational | OpenRouter embeddings, 1536 dims, hybrid BM25+vector, 2 files / 4 chunks |
| Mem0 Server | ✅ Running | systemd, `:8200`, 4 memories, Qdrant store |
| Letta Server | ✅ Running | podman, `:8283`, PostgreSQL, 1 agent with 3 tools |
| Clawd Workspace | ✅ Bootstrapped | IDENTITY.md, SOUL.md, USER.md, MEMORY.md + daily logs |
| Cron Jobs | ✅ Active | Daily Standup (8AM PT), EOD Review (6PM PT), Weekly Planning (Mon 7AM PT) |
| Skills (4 loaded) | ✅ Active | core, accountability, memory, network |
| Memory Bridge Plugin | 🟡 Building | TypeScript extension coordinating all 3 memory systems |
| Focusa State Engine | 🟡 Designing | Reducer, Focus Stack, Focus Gate, ASCC mapped to bridge |
| Dashboard Frontend | ⬚ Not started | Mobile-first, Figma mockup ready |
| WordPress Plugin | ⬚ Not started | Ring Leader API integration |

See [CURRENT_STATE.md](docs/CURRENT_STATE.md) for full details.

## Quick Start (Operations)

```bash
# Service status
systemctl status clawdbot-gateway mem0-wirebot
podman ps | grep letta

# Gateway health
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot gateway probe'

# Model probe
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot models status --probe'

# Restart all
systemctl restart clawdbot-gateway mem0-wirebot
podman restart letta-wirebot

# Logs
tail -f /home/wirebot/logs/clawdbot-gateway.log
journalctl -u mem0-wirebot -f
podman logs -f letta-wirebot
```

## Key Paths

| Path | Purpose |
|------|---------|
| `/data/wirebot/users/verious/clawdbot.json` | Gateway config |
| `/data/wirebot/users/verious/` | State directory (sessions, memory SQLite) |
| `/data/wirebot/users/verious/cron/jobs.json` | Cron job definitions |
| `/data/wirebot/bin/` | Launcher + secret injector scripts |
| `/data/wirebot/mem0/` | Mem0 data (Qdrant vectors) |
| `/data/wirebot/focusa/` | Focusa source specs (67 docs, 416KB) |
| `/data/wirebot/focusa-state/` | Focusa cognitive state (future) |
| `/home/wirebot/clawd/` | Agent workspace (identity, soul, memory, tools) |
| `/home/wirebot/wirebot-core/` | This repository |
| `/home/wirebot/logs/clawdbot-gateway.log` | Gateway log |
| `/etc/systemd/system/clawdbot-gateway.service` | Gateway systemd unit |
| `/etc/systemd/system/mem0-wirebot.service` | Mem0 systemd unit |
| `/etc/cloudflared/wirebot.yml` | Tunnel config |
| `/run/wirebot/gateway.env` | Runtime secrets (tmpfs, injected by rbw) |

## Documentation

### Cognitive Architecture (NEW)

| Document | Size | Purpose |
|----------|------|---------|
| [FOCUSA_WIREBOT_INTEGRATION.md](docs/FOCUSA_WIREBOT_INTEGRATION.md) | 152KB | **Complete Focusa system map** — all 67 specs, 58 sections, every field/enum/threshold |
| [MEMORY_BRIDGE_STRATEGY.md](docs/MEMORY_BRIDGE_STRATEGY.md) | 18KB | Write-Through Read-Cascade bridge architecture |
| [DISCOVERY_NOTES.md](docs/DISCOVERY_NOTES.md) | 17KB | Figma mockup analysis + ecosystem integration |

### Operations & Infrastructure

| Document | Purpose |
|----------|---------|
| [OPERATIONS.md](docs/OPERATIONS.md) | Systemd services, launcher scripts, logs, restart procedures |
| [AUTH_AND_SECRETS.md](docs/AUTH_AND_SECRETS.md) | rbw vault integration, auth profiles, OAuth sync |
| [CURRENT_STATE.md](docs/CURRENT_STATE.md) | What's deployed vs planned, with status |
| [MONITORING.md](docs/MONITORING.md) | Health checks, probes, alerting patterns |
| [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) | Common failures: OOM, auth errors, tunnel issues |
| [INSTALLATION.md](docs/INSTALLATION.md) | Full production install procedure |

### Architecture & Design

| Document | Purpose |
|----------|---------|
| [VISION.md](docs/VISION.md) | Unified high-level vision |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System architecture + service topology |
| [GATEWAY.md](docs/GATEWAY.md) | Clawdbot gateway config reference |
| [CAPABILITIES.md](docs/CAPABILITIES.md) | Feature / capability matrix |
| [TRUST_MODES.md](docs/TRUST_MODES.md) | Trust mode enforcement (Free → ExtraWire) |
| [MEMORY.md](docs/MEMORY.md) | Memory stack overview (memory-core + Letta + Mem0) |
| [CLAWDBOT_MEMORY_DEEP_DIVE.md](docs/CLAWDBOT_MEMORY_DEEP_DIVE.md) | Clawdbot memory-core internals |

### Memory & AI Integrations

| Document | Purpose |
|----------|---------|
| [LETTA_INTEGRATION.md](docs/LETTA_INTEGRATION.md) | Letta server setup, agent config, memory blocks |
| [MEM0_PLUGIN.md](docs/MEM0_PLUGIN.md) | Mem0 REST API, fact extraction, Qdrant store |

### Business & Rollout

| Document | Purpose |
|----------|---------|
| [DUAL_TRACK_PLAN.md](docs/DUAL_TRACK_PLAN.md) | Standalone + Network tracks |
| [LAUNCH_ORDER.md](docs/LAUNCH_ORDER.md) | Phase roadmap with status |
| [NETWORK_INTEGRATION.md](docs/NETWORK_INTEGRATION.md) | Startempire Wire ecosystem integration |
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

### Design Assets

| Path | Purpose |
|------|---------|
| `design/figma-home-overview.png` | Figma mockup: Home Overview screen |
| `design/figma-overview.png` | Figma mockup: All 4 frames |
| `design/figma-layers-labels.png` | Figma mockup: Layer structure |
| `design/figma-all-files.png` | Figma project files |

## Repository Structure

```
wirebot-core/
├── README.md                    # This file
├── AGENTS.md                    # Multi-agent coordination rules
├── docs/                        # 29 documentation files
│   ├── FOCUSA_WIREBOT_INTEGRATION.md  # ← Complete cognitive architecture (152KB)
│   ├── MEMORY_BRIDGE_STRATEGY.md      # ← Bridge design
│   ├── OPERATIONS.md                  # ← Start here for ops
│   ├── CURRENT_STATE.md               # ← Start here for status
│   ├── ARCHITECTURE.md                # ← Start here for architecture
│   └── ...
├── design/                      # Figma screenshots
├── skills/                      # Wirebot SKILL.md files
│   ├── wirebot-core/            # Core personality + routing
│   ├── wirebot-accountability/  # Daily standup, EOD review, weekly planning
│   ├── wirebot-memory/          # Memory search + workspace recall
│   └── wirebot-network/         # Startempire Wire ecosystem awareness
├── plugins/                     # Clawdbot extension plugins
│   ├── memory-mem0/             # Mem0 integration (skeleton → bridge)
│   └── sms-twilio/              # SMS plugin (skeleton)
├── provisioning/                # User provisioning scripts
│   └── register-user.sh
└── .beads/                      # Task tracking (beads workspace)
```

## Related Infrastructure

### Agent Workspace (`/home/wirebot/clawd/`)

| File | Purpose |
|------|---------|
| `IDENTITY.md` | Who Wirebot is — AI business operating partner |
| `SOUL.md` | Accountability-first coaching framework, 4 pillars |
| `USER.md` | Verious Smith III founder profile, Pacific timezone |
| `MEMORY.md` | Ecosystem map, membership tiers, key decisions |
| `AGENTS.md` | Multi-agent coordination protocol |
| `TOOLS.md` | Available tool reference |
| `BOOTSTRAP.md` | Workspace initialization guide |
| `HEARTBEAT.md` | Health monitoring config |
| `memory/*.md` | Daily memory logs |
| `canvas/` | Working documents |

### Focusa Source Specs (`/data/wirebot/focusa/`)

- `focusa-chatgpt-conversation.md` — Original design conversation (1.4MB, 395 messages)
- `docs/` — 95 extracted spec documents with `INDEX.md`
- `docs-final/` — 67 finalized specs (416KB total)

### Ecosystem

| Service | URL | Purpose |
|---------|-----|---------|
| startempirewire.com | Production | Main membership site (MemberPress + BuddyBoss) |
| startempirewire.network | Production | Ring Leader Plugin + Screenshots Plugin |
| wirebot.chat | Production | Standalone AI partner (Cloudflare tunnel) |
| helm.wirebot.chat | Production | Gateway Control UI (CF Access protected) |
| Chrome Extension | In repo | Svelte-based browser extension |

## Membership Tiers → Trust Modes

| Tier | Trust Mode | Capabilities |
|------|-----------|-------------|
| Free | Mode 0 | Read-only, view accountability feed |
| FreeWire | Mode 1 | Basic Q&A, template checklists |
| Wire | Mode 2 | Full AI coaching, custom goals, proactive nudges |
| ExtraWire | Mode 3 | Deep integration, API access, custom skills |

## Core Principles

> **Wirebot is a business operating dashboard where AI is embedded in the workflow — not a chat window you talk to.**

> **Wirebot scales trust as deliberately as it scales capability — because the closer it gets to a founder's life and business, the more carefully it must be designed.**

> **Focusa preserves continuity of mind across long sessions by separating focus, memory, and expression from fragile conversation history.**

> **Agents grow by learning how to act within their values — not by rewriting them.**

## License

Proprietary — Startempire Wire

---

*Part of the Startempire Wire ecosystem*
# Season 1: Red-to-Black
