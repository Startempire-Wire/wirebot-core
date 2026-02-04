# Wirebot White-Label Frontend

> **Branded, tier-scoped chat interface for clients. Connects to your OpenClaw gateway API.**

---

## Concept

The Wirebot White-Label Frontend is a **lightweight, brandable chat application** that connects to the central OpenClaw gateway. Each client deployment is scoped to their trust level and shows only the capabilities their tier permits.

```
Client's domain (ai.clientbiz.com)
        │
        ▼
┌─────────────────────────────────────┐
│  White-Label Frontend               │
│  - Client branding (logo, name,     │
│    colors, favicon)                 │
│  - Tier-scoped features             │
│  - No admin/ops complexity          │
│  - Static deploy (CDN, VPS, tunnel) │
└──────────────┬──────────────────────┘
               │ HTTPS (API + WebSocket)
               ▼
┌──────────────────────────────────────┐
│  Wirebot Hub (your VPS)              │
│  OpenClaw Gateway                    │
│  Per-client agent + session + memory │
│  Skills, cron, channels              │
└──────────────────────────────────────┘
```

**Key principle:** The frontend knows the client's tier. The gateway enforces it. Both agree on what's visible.

---

## Feature Scoping by Tier

| Feature | Mode 0 (Demo) | Mode 1 (Standard) | Mode 2 (Advanced) | Mode 3 (Sovereign) |
|---------|---------------|--------------------|--------------------|---------------------|
| Chat | ✅ (limited) | ✅ | ✅ | ✅ |
| Session history | ❌ | ✅ | ✅ | ✅ |
| Memory recall | ❌ | ❌ | ✅ | ✅ |
| Skills panel | ❌ | ❌ | ✅ (view) | ✅ (manage) |
| Cron / schedules | ❌ | ❌ | ✅ (view/run) | ✅ (create/edit) |
| Channel status | ❌ | ❌ | ❌ | ✅ |
| File workspace | ❌ | ❌ | ❌ | ✅ |
| Thinking toggle | ❌ | ❌ | ✅ | ✅ |
| Model selection | ❌ | ❌ | ❌ | ✅ |
| Abort / stop | ✅ | ✅ | ✅ | ✅ |

The frontend reads the tier from its deploy config and renders only permitted panels.
The gateway enforces permissions server-side (agent config, skill allowlists, tool policies).

---

## Branding Config

Each deployment includes a `brand.json`:

```json
{
  "name": "BizBot",
  "tagline": "Your AI business partner",
  "logo": "/assets/logo.svg",
  "favicon": "/assets/favicon.ico",
  "colors": {
    "primary": "#2563EB",
    "secondary": "#1E40AF",
    "accent": "#F59E0B",
    "background": "#0F172A",
    "surface": "#1E293B",
    "text": "#F8FAFC"
  },
  "tier": 2,
  "agentId": "client_acme",
  "gatewayUrl": "wss://helm.wirebot.chat",
  "apiUrl": "https://helm.wirebot.chat",
  "features": {
    "chat": true,
    "sessions": true,
    "skills": true,
    "cron": true,
    "channels": false,
    "workspace": false,
    "thinking": true,
    "modelSelect": false
  }
}
```

**Token is NOT in brand.json.** Client enters token on first connect (stored in browser localStorage), or token is injected server-side via reverse proxy header.

---

## Gateway-Side Setup (Per Client)

### 1. Add agent to config

```json5
{
  agents: {
    list: [
      // ... existing agents
      {
        id: "client_acme",
        name: "BizBot",
        identity: {
          name: "BizBot",
          emoji: "🚀",
          avatar: "https://acme.com/bot-avatar.png"
        },
        workspace: "~/clients/acme"
      }
    ]
  }
}
```

### 2. Create auth profile for client's agent

```bash
mkdir -p /data/wirebot/users/verious/agents/client_acme/agent
# Copy or create auth-profiles.json with appropriate provider keys
```

### 3. Enable HTTP API (one-time, gateway-wide)

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

### 4. Generate client token

Separate from the gateway admin token. Use gateway token for this client's access:

```bash
# Per-client tokens can be managed via the gateway auth system
# For now: shared gateway token (all agents share it)
# Future: per-agent token support when OpenClaw adds it
```

---

## Frontend Architecture

### Tech Stack

- **Vanilla HTML/CSS/JS** or **Lit** (same as OpenClaw UI, familiar)
- No framework dependency — keeps bundle small, deploys anywhere
- WebSocket client for real-time chat streaming
- HTTP client for REST API calls (fallback)
- Responsive (mobile + desktop)
- Dark/light theme support

### Component Structure

```
wirebot-client/
├── index.html              # Entry point
├── brand.json              # Per-deployment branding config
├── assets/
│   ├── logo.svg            # Client logo
│   └── favicon.ico         # Client favicon
├── src/
│   ├── app.js              # Main app (reads brand.json, renders panels)
│   ├── ws-client.js        # WebSocket connection + RPC
│   ├── http-client.js      # HTTP API fallback
│   ├── chat.js             # Chat panel (send, stream, history, abort)
│   ├── sessions.js         # Session list + switch
│   ├── skills.js           # Skills panel (view, enable/disable)
│   ├── cron.js             # Cron panel (list, run, create)
│   ├── auth.js             # Token entry + localStorage
│   ├── theme.js            # Branding + dark/light
│   └── tier-gate.js        # Feature gating by tier
└── styles/
    ├── base.css            # Reset + variables (from brand.json)
    ├── chat.css            # Chat UI styles
    └── panels.css          # Side panels
```

### Communication Protocol

The frontend uses the **same WebSocket RPC** as the OpenClaw Control UI:

| Feature | RPC Methods |
|---------|-------------|
| Chat | `chat.send`, `chat.history`, `chat.abort`, `chat.inject` |
| Sessions | `sessions.list`, `sessions.patch` |
| Skills | `skills.status` |
| Cron | `cron.list`, `cron.run`, `cron.add`, `cron.enable`, `cron.disable` |
| Status | `status`, `health` |

**OR** the HTTP API for simpler integrations:

| Feature | HTTP Endpoint |
|---------|--------------|
| Chat | `POST /v1/chat/completions` (OpenAI-compatible) |
| Tools | `POST /tools/invoke` |
| Hooks | `POST /hooks/agent` |

---

## Deployment Options

### Option A: Static files on client's VPS

```bash
# Copy wirebot-client/ to their server
scp -r wirebot-client/ user@client-vps:/var/www/ai.clientbiz.com/

# Nginx config on their server
server {
    server_name ai.clientbiz.com;
    root /var/www/ai.clientbiz.com;
    index index.html;
    
    # WebSocket proxy to your gateway
    location /ws {
        proxy_pass https://helm.wirebot.chat;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $websocket_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # API proxy to your gateway
    location /api/ {
        proxy_pass https://helm.wirebot.chat/;
        proxy_set_header Authorization "Bearer $client_token";
    }
}
```

### Option B: Cloudflare tunnel from client's domain

```yaml
# On their VPS or yours
ingress:
  - hostname: ai.clientbiz.com
    service: http://127.0.0.1:3000  # Local wirebot-client server
```

### Option C: CDN deploy (Cloudflare Pages, Netlify, Vercel)

Static site deploy. brand.json baked in at build time. API calls go directly to `helm.wirebot.chat` (CORS must be configured on gateway).

---

## Security

- **Client token** is per-deployment, not the admin gateway token
- **Agent scoping**: `x-openclaw-agent-id` header ensures client only accesses their agent
- **Server-side enforcement**: gateway agent config + tool policies enforce tier limits regardless of frontend
- **No admin RPC exposed**: frontend never calls `config.get`, `config.apply`, `logs.tail`, `update.run`, etc.
- **CORS**: gateway must allow the client's domain for direct API calls

---

## Provisioning a New Client

1. Add agent to gateway config (`agents.list`)
2. Create agent workspace + auth profiles
3. Copy `wirebot-client/` template
4. Customize `brand.json` (name, logo, colors, tier, agentId)
5. Deploy to client's subdomain (any of the options above)
6. Share client token
7. Verify: client opens `ai.clientbiz.com`, enters token, chats

---

## Relationship to Existing Docs

This replaces the vague "WordPress plugin" concept for client-facing UI (at least for Phase 0/1). The WordPress plugin (PLUGIN.md) becomes a higher-level integration that may embed or link to this frontend.

The trust modes (TRUST_MODES.md) map directly to the `tier` field in `brand.json`:
- Mode 0 → `tier: 0` (demo, limited chat)
- Mode 1 → `tier: 1` (standard, chat + sessions)
- Mode 2 → `tier: 2` (advanced, + skills + cron)
- Mode 3 → `tier: 3` (sovereign, full access)

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Hub-and-spoke model
- [TRUST_MODES.md](./TRUST_MODES.md) — Tier definitions
- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix
- [GATEWAY.md](./GATEWAY.md) — API endpoints + config
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Token management
- [PROVISIONING.md](./PROVISIONING.md) — Client provisioning
- [OPERATIONS.md](./OPERATIONS.md) — Gateway management
- [PLUGIN.md](./PLUGIN.md) — WordPress integration (higher-level)
- OpenClaw docs: `gateway/openai-http-api` — HTTP API reference
- OpenClaw docs: `web/control-ui` — WebSocket RPC reference
