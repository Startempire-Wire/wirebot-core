# Wirebot WordPress Plugin Specification (OpenClaw-Based)

> **Product shell for Wirebot. OpenClaw is the runtime.**

---

## Plugin Overview

**Name:** `startempire-wirebot`

**Role:**
- Identity + tier management
- Provisioning (shared gateway vs dedicated OpenClaw)
- Channel setup UI
- Network integration (Ring Leader)

**Not responsible for AI runtime.**

---

## Core Responsibilities

### 1) Tier Routing

- **Top tier:** provision dedicated OpenClaw container
- **Lower tiers:** register user in shared gateway

### 2) Provisioning

Top tier provisioning script:

```
./provision-openclaw.sh <user_id>
```

Script responsibilities:
- Create state dir (`OPENCLAW_STATE_DIR`)
- Write JSON5 config
- Install Wirebot skills
- Start gateway on unique port

### 3) Shared Gateway Registration

Lower tier registration:

```
./register-user.sh <user_id>
```

Responsibilities:
- Create Letta agent
- Create Mem0 namespace
- Add agent to `agents.list`
- Add channel bindings

### 4) Channel Setup UI

User dashboard:
- Connect Discord (OAuth)
- Connect Telegram (bot token)
- Connect WhatsApp (QR, top tier only)
- SMS notes (Android/iMessage/custom plugin)

### 5) Network Integration

For Track B:
- Read Ring Leader identity
- Provide network context to Wirebot skills
- Expose network intelligence surfaces

---

## Config Management (OpenClaw)

OpenClaw config is JSON5: `~/.openclaw/openclaw.json`

Use CLI for writes:

```bash
openclaw config set gateway.auth.token "<token>"
openclaw config set skills.load.extraDirs --json '["/home/wirebot/wirebot-core/skills"]'
openclaw config set plugins.load.paths --json '["/home/wirebot/wirebot-core/plugins"]'
```

**Do not expose gateway token to clients.**

---

## Tier → Infrastructure Mapping

| Tier | Infrastructure | Notes |
|------|----------------|-------|
| Top tier | Dedicated OpenClaw container | Full channel access |
| Lower tiers | Shared gateway | Limited channels |

---

## Track Detection (Standalone vs Network)

```php
function get_wirebot_track() {
    if (function_exists('ring_leader_get_user_identity')) {
        $network_identity = ring_leader_get_user_identity(get_current_user_id());
        if ($network_identity && !empty($network_identity['is_network_member'])) {
            return 'network';
        }
    }
    return 'standalone';
}
```

---

## Gateway Auth

OpenClaw gateway uses token/password auth:

```json5
gateway: {
  auth: { mode: "token", token: "<token>" }
}
```

Plugin must proxy requests server‑side.

---

## Skills + Plugins

Wirebot skills live in:
```
/home/wirebot/wirebot-core/skills
```

OpenClaw loads via:
```json5
skills: { load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] } }
```

Custom plugins live in:
```
/home/wirebot/wirebot-core/plugins
```

OpenClaw loads via:
```json5
plugins: { load: { paths: ["/home/wirebot/wirebot-core/plugins"] } }
```

---

## Network Integration (Track B)

Wirebot network skill calls Ring Leader APIs:
- Member profile
- Connections
- Events
- Content recommendations
- Directory

All gated by membership tier.

---

## Data Ownership

**WordPress owns:**
- Billing + tier state
- Consent
- Network identity

**OpenClaw owns:**
- Sessions
- Channel routing
- Tool execution

**Letta/Mem0 own:**
- Memory + long‑term context

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [GATEWAY.md](./GATEWAY.md) — Gateway config + auth
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [PAIRING_ALLOWLIST.md](./PAIRING_ALLOWLIST.md) — DM approval flow
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Network features
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Gateway auth (never expose to clients)
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Plugin status (not yet started)
- [TRUST_MODES.md](./TRUST_MODES.md) — Tier enforcement
