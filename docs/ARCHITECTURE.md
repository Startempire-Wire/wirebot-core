# Wirebot Architecture

> **Two-Layer System: Intelligence Runtime + Product Shell**

---

## Overview

Wirebot is a **dual-layer system**:

```
┌─────────────────────────────────────────────────────────────┐
│              LAYER B: WordPress (Product Shell)             │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Ring Leader  │  │   Connect    │  │  Wirebot     │      │
│  │ (identity)   │  │ (ext bridge) │  │  Plugin      │      │
│  └──────────────┘  └──────────────┘  └──────┬───────┘      │
│                                              │              │
│         Trust ceiling, JWT issuer, UX,       │              │
│         billing, modules, consent            │              │
└──────────────────────────────────────────────┼──────────────┘
                                               │
                                          JWT + scopes
                                               │
                                               ▼
┌─────────────────────────────────────────────────────────────┐
│              LAYER A: Wirebot Gateway (Intelligence)        │
│                                                             │
│  Sessions │ Memory │ Scheduling │ Reasoning │ Trust Enforce │
│                                                             │
│  Node 20 (TS) │ Redis │ MariaDB │ Podman                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Layer A: Wirebot Gateway (Intelligence Runtime)

The Gateway is the **single brain** that:
- Owns sessions and memory
- Manages scheduling (cron / wakeups)
- Maintains tool registry
- Routes to AI providers
- Enforces trust modes
- Applies surface-aware policies

### Technology Stack
- **Runtime:** Node 20 (TypeScript)
- **Cache/Sessions:** Redis 7.2
- **Persistence:** MariaDB 10.6
- **Containers:** Podman
- **Reverse Proxy:** LiteSpeed

### Endpoints
- `api.wirebot.chat` — HTTP + WebSocket API
- Twilio webhooks for SMS/Voice

---

## Layer B: WordPress Plugin Ecosystem (Product Shell)

WordPress handles **governance, UX, and monetization**.

### Core Plugin: `startempire-wirebot`

Responsibilities:
- Identity & trust ceiling management
- JWT issuer for Gateway authentication
- Settings & configuration UI
- Module management
- Data ownership boundary
- Billing / entitlements

### Integration with Existing Plugins

| Plugin | Role | Wirebot Integration |
|--------|------|---------------------|
| Ring Leader | Identity, network graph, auth broker | Consumes identity |
| Connect | Extension auth bridge | Uses for extension auth |
| WebSockets | Realtime transport | Optional streaming |
| Screenshots | Visual enrichment | Not core |

---

## Data Flow

```
Founder → WordPress Login
       → Wirebot Plugin generates JWT
       → JWT includes: user_id, workspace_id, trust_mode_max, scopes
       → Browser/Extension sends JWT to api.wirebot.chat
       → Gateway validates, enforces trust ceiling
       → Response flows back through same channel
```

---

## Deployment Architecture

### Single VPS (Current)

```
AlmaLinux VPS (10 cores, 14GB RAM)
├── LiteSpeed (cPanel)
├── MariaDB 10.6
├── Redis 7.2
└── Podman Containers
    ├── wirebot-gateway (Modes 0-2)
    ├── wirebot-sovereign (Mode 3, localhost only)
    └── [other services]
```

### DNS Structure

| Subdomain | Purpose |
|-----------|---------|
| `wirebot.chat` | Marketing / public demo (Mode 0) |
| `app.wirebot.chat` | Founder dashboard (Mode 1-2) |
| `api.wirebot.chat` | Gateway API (HTTP + WS) |
| `go.wirebot.chat` | Short links / redirects |

---

## Key Design Principles

1. **Separation of concerns** — Gateway thinks, WordPress governs
2. **Thin clients** — All surfaces are dumb, Gateway owns intelligence
3. **Trust enforcement at both layers** — JWT ceiling + runtime checks
4. **Progressive isolation** — Mode 3 runs separately from Modes 0-2

---

## Correct Mental Model

**Incorrect:**
> "Wirebot is a WordPress plugin that calls OpenAI."

**Correct:**
> "Wirebot is a secure intelligence service. WordPress is the product shell that governs access, trust, UX, and monetization."

---

## Summary

- **WordPress = legitimacy, safety, commercial viability**
- **Gateway = thinking, memory, execution**
- Both required. Neither optional.
