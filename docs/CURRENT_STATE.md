# Wirebot Current State

> **What's actually deployed, running, and operational — vs what's planned.**
>
> Last updated: 2026-01-30

---

## Phase Status

| Phase | Status | Notes |
|-------|--------|-------|
| **Phase 0: Foundation** | 🟡 In Progress | Gateway running, auth working, skills loaded. Letta/Mem0 not deployed. |
| Phase 1: Dogfooding | ⬜ Not Started | Blocked on Phase 0 completion |
| Phase 2: Rollout Prep | ⬜ Not Started | |
| Phase 3: Network Integration | ⬜ Not Started | |
| Phase 4: Scale | ⬜ Not Started | |

See [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) for the full roadmap.

---

## Infrastructure (Running)

### ✅ Clawdbot Gateway

| Property | Value |
|----------|-------|
| **Service** | `clawdbot-gateway.service` (systemd, enabled) |
| **Version** | Clawdbot 2026.1.24-3 |
| **Node** | v22.22.0 (nvm) |
| **Port** | 18789 (loopback) |
| **Config** | `/data/wirebot/users/verious/clawdbot.json` |
| **State dir** | `/data/wirebot/users/verious/` |
| **Launcher** | `/data/wirebot/bin/clawdbot-gateway.sh` |
| **Log** | `/home/wirebot/logs/clawdbot-gateway.log` |
| **Default model** | `anthropic/claude-opus-4-5` |
| **Auth** | Anthropic OAuth (Claude Max 5x) + OpenRouter API key |
| **Secrets** | rbw vault injection via systemd ExecStartPre |

### ✅ Cloudflare Tunnel

| Property | Value |
|----------|-------|
| **Service** | `cloudflared-wirebot.service` (systemd, enabled) |
| **Tunnel ID** | `57df17a8-b9d1-4790-bab9-8157ac51641b` |
| **Config** | `/etc/cloudflared/wirebot.yml` |
| **Routes** | `helm.wirebot.chat` → `127.0.0.1:18789` |
| | `api.wirebot.chat` → `localhost:8100` (no listener yet) |

### ✅ Wirebot Skills (Loaded)

Skills loaded from `/home/wirebot/wirebot-core/skills/`:

| Skill | Path | Status |
|-------|------|--------|
| `wirebot-core` | `skills/wirebot-core/SKILL.md` | ✅ Loaded |
| `wirebot-accountability` | `skills/wirebot-accountability/SKILL.md` | ✅ Loaded |
| `wirebot-memory` | `skills/wirebot-memory/SKILL.md` | ✅ Loaded |
| `wirebot-network` | `skills/wirebot-network/SKILL.md` | ✅ Loaded |

### ✅ Auth Profiles

| Profile | Provider | Type | Status |
|---------|----------|------|--------|
| `anthropic:claude-cli` | Anthropic | OAuth | ✅ Working (auto-refresh) |
| `openrouter:default` | OpenRouter | API Key | ✅ Working |

---

## Infrastructure (Not Yet Deployed)

### ❌ Letta Server

- Not installed yet
- Required for structured business state (goals, KPIs, stage tracking)
- See [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md)

### ❌ Mem0 Server

- Not installed yet
- Required for cross-surface memory sync (browser → agents)
- See [MEM0_PLUGIN.md](./MEM0_PLUGIN.md)

### ❌ WordPress Plugin (`startempire-wirebot`)

- Not started
- Required for tier routing, provisioning UI, channel setup
- See [PLUGIN.md](./PLUGIN.md)

### ❌ api.wirebot.chat

- Route exists in Cloudflare tunnel config (`localhost:8100`)
- No service listening on port 8100
- Purpose TBD (REST API? separate service?)

### ❌ Ring Leader Integration

- Planned for Phase 3
- See [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md)

---

## Agents (Configured)

| Agent ID | Name | Sessions | Auth |
|----------|------|----------|------|
| `verious` | Wirebot: verious | 1 (stale, 4+ days old) | auth-profiles.json |
| `main` | (default) | — | auth-profiles.json (copy of verious) |

---

## File System Layout (Actual)

```
/data/wirebot/
├── bin/
│   ├── clawdbot-gateway.sh          # Launcher (wirebot:wirebot, 750)
│   └── inject-gateway-secrets.sh    # Secret injector (root:root, 700)
└── users/
    └── verious/                     # State dir (wirebot:wirebot, 700)
        ├── clawdbot.json            # Gateway config (600)
        ├── credentials/             # Channel pairing + allowlists
        ├── cron/                    # Cron job definitions
        │   └── jobs.json
        ├── devices/                 # Paired devices
        ├── identity/                # Gateway identity
        ├── sessions/                # Legacy session store
        └── agents/
            ├── main/
            │   └── agent/
            │       └── auth-profiles.json  (600)
            └── verious/
                ├── agent/
                │   └── auth-profiles.json  (600)
                └── sessions/
                    └── sessions.json

/home/wirebot/
├── .nvm/                            # Node version manager
│   └── versions/node/v22.22.0/
│       └── bin/clawdbot             # Clawdbot binary
├── logs/
│   └── clawdbot-gateway.log        # Gateway log (appended by systemd)
├── clawd/                           # Default agent workspace
│   └── canvas/                      # Canvas UI static files
└── wirebot-core/                    # This repository
    ├── docs/                        # Documentation
    ├── skills/                      # Wirebot skills
    ├── plugins/                     # Clawdbot plugins (skeleton)
    └── provisioning/                # Provisioning scripts (skeleton)

/etc/
├── systemd/system/
│   └── clawdbot-gateway.service     # Systemd unit (root, 644)
└── cloudflared/
    └── wirebot.yml                  # Tunnel config

/run/wirebot/                        # Tmpfs (cleared on reboot)
└── gateway.env                      # Runtime secrets (600)
```

---

## Config (Current)

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

## What's Working (Can Dogfood Now)

- ✅ Gateway responds to WebSocket connections via `helm.wirebot.chat`
- ✅ Control UI accessible via tunnel
- ✅ Anthropic Claude Opus 4.5 via OAuth (Claude Max 5x)
- ✅ OpenRouter as fallback provider
- ✅ Skills loaded (core, accountability, memory, network)
- ✅ Cron engine available
- ✅ Memory (Clawdbot built-in markdown + hybrid search)
- ✅ Systemd service with auto-restart + rbw secret injection

## What's Not Working Yet

- ❌ Agent "verious" has no recent sessions (last activity 4+ days ago)
- ❌ No Letta server (structured business state not available)
- ❌ No Mem0 server (cross-surface sync not available)
- ❌ No WordPress plugin (no user-facing product shell)
- ❌ No channels connected (no WhatsApp, Telegram, Discord)
- ❌ No model fallbacks configured (single model, no `fallbacks` array)
- ❌ `api.wirebot.chat` has no listener (port 8100 unused)

---

## See Also

- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Full roadmap
- [OPERATIONS.md](./OPERATIONS.md) — How to operate what's running
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Current auth setup
- [MONITORING.md](./MONITORING.md) — How to verify health
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Target architecture
