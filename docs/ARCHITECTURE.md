# Wirebot Architecture (Clawdbot-Based)

> **Clawdbot is the runtime. Wirebot is skills + product shell + network intelligence.**

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Public Internet                          │
│  helm.wirebot.chat ──→ Cloudflare Tunnel                    │
│  api.wirebot.chat  ──→ Cloudflare Tunnel                    │
└──────────────┬──────────────────────────────────────────────┘
               │
┌──────────────▼──────────────────────────────────────────────┐
│              Production VPS (AlmaLinux 8)                     │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Cloudflare Tunnel (cloudflared-wirebot.service)     │     │
│  │  helm.wirebot.chat → 127.0.0.1:18789                │     │
│  │  api.wirebot.chat  → localhost:8100                  │     │
│  └──────────────┬──────────────────────────────────────┘     │
│                 │                                             │
│  ┌──────────────▼──────────────────────────────────────┐     │
│  │  Clawdbot Gateway (clawdbot-gateway.service)         │     │
│  │  Port: 18789 (loopback only)                         │     │
│  │  User: wirebot | Node: v22.22.0                      │     │
│  │                                                       │     │
│  │  ┌─────────────────────────────────────────────┐     │     │
│  │  │ Control UI + WebChat (built-in)              │     │     │
│  │  │ WebSocket control plane                       │     │     │
│  │  │ Channel routing engine                        │     │     │
│  │  │ Session manager                               │     │     │
│  │  │ Cron engine                                   │     │     │
│  │  │ Model router + failover                       │     │     │
│  │  │ Memory (markdown + hybrid search)             │     │     │
│  │  └─────────────────────────────────────────────┘     │     │
│  │                                                       │     │
│  │  Skills: wirebot-core, wirebot-accountability,        │     │
│  │          wirebot-memory, wirebot-network               │     │
│  │                                                       │     │
│  │  Plugins: memory-core (built-in)                      │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐     │
│  │  Future: Letta Server (structured business state)     │     │
│  │  Future: Mem0 Server (cross-surface sync)             │     │
│  │  Future: WordPress Plugin (product shell)             │     │
│  └──────────────────────────────────────────────────────┘     │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐     │
│  │  Secret Management                                    │     │
│  │  rbw (Bitwarden vault) → /run/wirebot/gateway.env     │     │
│  │  Claude Code OAuth → auth-profiles.json               │     │
│  └──────────────────────────────────────────────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Responsibilities

### Clawdbot Gateway (Runtime)

**Owns:**
- WebSocket control plane (ws://127.0.0.1:18789)
- Channel connections (WhatsApp, Telegram, Discord, Slack, Signal, iMessage, etc.)
- Session lifecycle + history
- Cron scheduling + hooks
- Tool execution (sandboxed)
- Model routing + auth profile failover
- Memory system (markdown + sqlite-vec + FTS5 hybrid search)
- Control UI + WebChat (served on gateway port)
- Skills loading + execution

**Does not own:**
- User identity / billing (→ WordPress)
- Structured business state (→ Letta, planned)
- Cross-surface memory sync (→ Mem0, planned)
- Network intelligence (→ Ring Leader APIs, planned)

### Wirebot Skills (Business Logic)

Skills loaded from `/home/wirebot/wirebot-core/skills/`:

| Skill | Purpose |
|-------|---------|
| `wirebot-core` | Core Wirebot identity + behavior |
| `wirebot-accountability` | Standups, reflections, goal tracking |
| `wirebot-memory` | Memory management + recall patterns |
| `wirebot-network` | Startempire Wire network intelligence |

Skills are SKILL.md files (pi-coding-agent format) loaded via `skills.load.extraDirs`.

### Cloudflare Tunnel (Public Access)

Exposes the loopback-bound gateway to the public internet via encrypted tunnel.

- **Service:** `cloudflared-wirebot.service`
- **Config:** `/etc/cloudflared/wirebot.yml`
- **Tunnel ID:** `57df17a8-b9d1-4790-bab9-8157ac51641b`

### Secret Management (rbw + OAuth)

- **Provider API keys:** Retrieved from Bitwarden vault (`rbw`) at service start
- **Anthropic OAuth:** Synced from Claude Code credentials, auto-refreshed by Clawdbot
- **Gateway token:** Stored in `clawdbot.json` (mode 600)
- **Runtime secrets:** Written to tmpfs `/run/wirebot/gateway.env` (cleared on reboot)

See [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) for details.

---

## Data Architecture

### Per-User Isolation

Each user gets isolated state:

| Layer | Isolation Key | Storage |
|-------|--------------|---------|
| Clawdbot agent | `agentId` | `agents/<id>/agent/` + workspace |
| Clawdbot sessions | `agentId + channel + peer` | `agents/<id>/sessions/` |
| Clawdbot memory | `agentId + workspaceDir` | SQLite per-agent |
| Letta (planned) | `agent_<user_id>` | Letta server |
| Mem0 (planned) | `wirebot_<user_id>` | Mem0 server |

### State Directory Layout

```
/data/wirebot/users/<user_id>/           # State dir (700, wirebot)
├── clawdbot.json                        # Config (600)
├── credentials/                          # Channel pairing + allowlists
├── cron/                                 # Cron job definitions
├── devices/                              # Paired devices
├── identity/                             # Gateway identity
├── sessions/                             # Legacy session store
└── agents/
    ├── main/
    │   └── agent/
    │       └── auth-profiles.json        # Auth secrets (600)
    └── <agentId>/
        ├── agent/
        │   └── auth-profiles.json        # Auth secrets (600)
        └── sessions/
            └── sessions.json             # Session store
```

### Workspace Layout

```
/home/wirebot/clawd/                     # Default agent workspace
├── MEMORY.md                            # Long-term curated memory
├── memory/                              # Daily append-only logs
│   ├── 2026-01-30.md
│   └── ...
└── canvas/                              # Canvas UI static files
```

---

## Infrastructure Models

### Shared Gateway (Lower Tiers)

Single Clawdbot gateway with multi-tenant agents:

```
One gateway process (port 18789)
├── Agent: user_1 → Discord DM binding
├── Agent: user_2 → Telegram DM binding
└── Agent: user_3 → Web UI binding
```

Config: `agents.list` + `bindings` for channel → agent routing.

### Dedicated Gateway (Top Tier)

Per-user Clawdbot container with unique port:

```
Gateway process (port 18xxx)
├── Agent: <user_id> (sole agent)
├── Full channel access (WhatsApp, etc.)
└── Own workspace + credentials
```

Provisioned via script with per-user `CLAWDBOT_STATE_DIR` and port.

---

## Memory Stack

```
┌─────────────────────────────┐
│  Clawdbot Memory (Built-in) │  ← Daily logs + MEMORY.md + hybrid search
│  SQLite + sqlite-vec + FTS5 │     Operational. Used now.
├─────────────────────────────┤
│  Letta (Planned)            │  ← Structured business state (goals, KPIs, stage)
│  REST API + PostgreSQL      │     Not deployed yet.
├─────────────────────────────┤
│  Mem0 (Planned)             │  ← Cross-surface sync (browser → agents)
│  Vector + graph memory      │     Not deployed yet.
└─────────────────────────────┘
```

See [MEMORY.md](./MEMORY.md) for the full memory architecture.

---

## Network Integration (Planned)

```
StartempireWire.com (MemberPress + BuddyBoss)
        │
Ring Leader Plugin (identity + network graph)
        │
Wirebot Plugin (auth + provisioning + tier routing)
        │
Clawdbot Runtime (gateway + skills + channels)
        │
Letta + Mem0 (memory + context)
```

See [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) for details.

---

## Service Topology (Production)

| Service | Unit | Port | User | Status |
|---------|------|------|------|--------|
| Clawdbot Gateway | `clawdbot-gateway.service` | 18789 | wirebot | ✅ Running |
| Cloudflare Tunnel | `cloudflared-wirebot.service` | N/A | root | ✅ Running |
| Browser Control | (embedded) | 18791 | wirebot | ✅ Running |
| Letta Server | — | TBD | — | ❌ Not deployed |
| Mem0 Server | — | TBD | — | ❌ Not deployed |
| api.wirebot.chat | — | 8100 | — | ❌ No listener |

See [CURRENT_STATE.md](./CURRENT_STATE.md) for detailed deployment status.

---

## See Also

- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's actually deployed
- [OPERATIONS.md](./OPERATIONS.md) — Service management
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Secret management
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [MEMORY.md](./MEMORY.md) — Memory stack details
- [MONITORING.md](./MONITORING.md) — Health checks
- [INSTALLATION.md](./INSTALLATION.md) — Setup procedure
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [VISION.md](./VISION.md) — High-level vision
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Implementation roadmap
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Startempire Wire integration
- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix
- [TRUST_MODES.md](./TRUST_MODES.md) — Trust enforcement
- [WHITE_LABEL.md](./WHITE_LABEL.md) — White-label frontend for clients
