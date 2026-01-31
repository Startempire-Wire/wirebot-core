# Wirebot Current State

> **What's actually deployed, running, and operational — vs what's planned.**
>
> Last updated: 2026-01-31

---

## Phase Status

| Phase | Status | Notes |
|-------|--------|-------|
| **Phase 0: Foundation** | 🟢 Core Complete | Gateway running, auth working, skills loaded, **memory operational**, cron active. Letta/Mem0 deferred. |
| Phase 1: Dogfooding | 🟡 Starting | Memory + identity + accountability cadence live. Dashboard frontend next. |
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
| **Workspace** | `/home/wirebot/clawd` |
| **Launcher** | `/data/wirebot/bin/clawdbot-gateway.sh` |
| **Log** | `/home/wirebot/logs/clawdbot-gateway.log` |
| **Default model** | `anthropic/claude-opus-4-5` |
| **Auth** | Anthropic OAuth (Claude Max 5x) + OpenRouter API key |
| **Secrets** | rbw vault injection via systemd ExecStartPre |

### ✅ Memory System (memory-core)

| Property | Value |
|----------|-------|
| **Plugin** | `memory-core` (built-in) |
| **Provider** | local (embeddinggemma-300M, Q8_0 GGUF) |
| **Search** | Hybrid: BM25 + vector (sqlite-vec) |
| **Store** | `/data/wirebot/users/verious/memory/verious.sqlite` |
| **Files indexed** | 2/2 (MEMORY.md + memory/2026-01-31.md) |
| **Chunks** | 4 |
| **FTS** | Ready |
| **Vector** | Ready (768 dims) |
| **Cache** | Enabled (50K cap) |
| **File watcher** | Active (auto-reindex on changes) |

### ✅ Workspace Identity

| File | Status | Content |
|------|--------|---------|
| `IDENTITY.md` | ✅ Populated | Wirebot = AI business operating partner, ⚡ |
| `SOUL.md` | ✅ Populated | Accountability-first mentor, 4-pillar business coaching |
| `USER.md` | ✅ Populated | Verious Smith III context |
| `MEMORY.md` | ✅ Populated | Architecture, tiers, coaching model, decisions |
| `memory/2026-01-31.md` | ✅ Created | Day 1 log |
| `AGENTS.md` | ✅ Pre-existing | Agent operating instructions |
| `TOOLS.md` | ✅ Pre-existing | Tool notes |
| `HEARTBEAT.md` | ✅ Pre-existing | Heartbeat checklist |

### ✅ Accountability Cron

| Job | Schedule | Next Run |
|-----|----------|----------|
| Daily Standup | 8:00 AM PT daily | ~19h |
| EOD Review | 6:00 PM PT daily | ~5h |
| Weekly Planning | 7:00 AM PT Mondays | ~2d |

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

| Skill | Status |
|-------|--------|
| `wirebot-core` | ✅ Loaded |
| `wirebot-accountability` | ✅ Loaded |
| `wirebot-memory` | ✅ Loaded |
| `wirebot-network` | ✅ Loaded |

### ✅ Auth Profiles

| Profile | Provider | Type | Status |
|---------|----------|------|--------|
| `anthropic:claude-cli` | Anthropic | OAuth | ✅ Working (auto-refresh) |
| `openrouter:default` | OpenRouter | API Key | ⚠️ Cursor-provisioned (no direct API) |

---

## Infrastructure (Not Yet Deployed)

### ⏸️ Mem0 Server

- Python package installed (mem0ai 1.0.2), plugin skeleton exists
- Needs embedding API key (OpenAI, Gemini, or real OpenRouter key)
- Primary use case: browser sync (OpenMemory → Wirebot)
- **Deferred**: memory-core covers search/recall needs
- See [MEM0_PLUGIN.md](./MEM0_PLUGIN.md)

### ⏸️ Letta Server

- `letta` CLI is Letta Code (coding agent), NOT the memory server
- Would need separate Python Letta server for structured state
- **Deferred**: business state can be modeled in workspace files for now
- See [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md)

### ⏸️ memory-lancedb

- Plugin exists but hardcoded for OpenAI embeddings
- Config supports OpenAI-compatible via `memorySearch.remote.baseUrl`
- **Blocked**: OpenRouter Cursor key returns 401 on direct API calls
- **Unblocked when**: real OpenRouter API key generated at openrouter.ai/keys

### ❌ WordPress Plugin (`startempire-wirebot`)

- Not started
- Required for tier routing, provisioning UI, channel setup
- See [PLUGIN.md](./PLUGIN.md)

### ❌ Dashboard Frontend

- Figma mockup analyzed ([DISCOVERY_NOTES.md](./DISCOVERY_NOTES.md))
- Mobile-first business operating dashboard
- Not a chat app — "Ask Wirebot" is one input element
- Needs: checklist engine, progress tracking, standup UI

### ❌ api.wirebot.chat

- Route exists in Cloudflare tunnel config (`localhost:8100`)
- No service listening on port 8100

---

## Config (Current)

```json5
{
  agents: {
    defaults: {
      workspace: "/home/wirebot/clawd",
      skipBootstrap: true,
      userTimezone: "America/Los_Angeles",
      memorySearch: {
        provider: "local",
        fallback: "none",
        query: { hybrid: { enabled: true, vectorWeight: 0.7, textWeight: 0.3 } },
        cache: { enabled: true, maxEntries: 50000 },
        sync: { watch: true }
      }
    },
    list: [{
      id: "verious",
      name: "Wirebot",
      identity: { name: "Wirebot", theme: "AI business operating partner", emoji: "⚡" }
    }]
  },
  gateway: {
    port: 18789, mode: "local", bind: "loopback",
    auth: { mode: "token", token: "<redacted>", allowTailscale: true }
  },
  skills: { load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] } },
  plugins: { allow: ["memory-core"] }
}
```

---

## What's Working (Can Dogfood Now)

- ✅ Gateway with WebSocket RPC (v3 protocol)
- ✅ Cloudflare tunnel (helm.wirebot.chat)
- ✅ Anthropic Claude Opus 4.5 via OAuth
- ✅ Memory: hybrid search (vector + BM25), local embeddings, file watcher
- ✅ Identity: IDENTITY.md, SOUL.md, USER.md, MEMORY.md all populated
- ✅ Skills: 4 wirebot skills + ~12 bundled skills eligible
- ✅ Accountability cron: daily standup, EOD review, weekly planning
- ✅ Systemd service with auto-restart + rbw secret injection

## What's Needed Next

- 🔜 Dashboard frontend (Figma → code, mobile-first)
- 🔜 Business setup checklist data model (Idea→Launch→Growth tasks)
- 🔜 HTTP API endpoints (chatCompletions + responses) for frontend
- 🔜 Real OpenRouter API key for enhanced embeddings + model fallbacks
- 🔜 WordPress plugin for onboarding + tier gating
- 🔜 First beta tester onboarded

---

## See Also

- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Full roadmap
- [DISCOVERY_NOTES.md](./DISCOVERY_NOTES.md) — Figma + ecosystem analysis
- [OPERATIONS.md](./OPERATIONS.md) — How to operate what's running
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Current auth setup
- [MONITORING.md](./MONITORING.md) — How to verify health
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Target architecture
