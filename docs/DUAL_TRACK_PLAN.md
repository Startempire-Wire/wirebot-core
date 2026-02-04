# Wirebot Dual-Track Business Plan (OpenClaw-Based)

> **Standalone + Network. Full OpenClaw available day 0 for top tier.**

---

## Overview

Wirebot uses OpenClaw as the runtime and ships two tracks:

1. **Track A: Standalone** — wirebot.chat
2. **Track B: Network-Integrated** — Startempire Wire members

Both tracks can provision **full OpenClaw** for top tier from day 0.

---

## Infrastructure Split

| Tier | Infrastructure | Notes |
|------|----------------|-------|
| **Top tier** | Dedicated OpenClaw container | Full channels + autonomy |
| **Lower tiers** | Shared OpenClaw gateway | Multi-tenant agents |

---

## Track A: Standalone

- Auth: WordPress
- Billing: Stripe
- Runtime: OpenClaw

**Example tiering** (prices TBD):

| Tier | Infra | Channels |
|------|-------|----------|
| Free | Shared | Demo only |
| Basic | Shared | Web only |
| Standard | Shared | Web + Discord/Telegram |
| Premium | Shared | + SMS* |
| Power | Dedicated | Full OpenClaw |

*SMS depends on Android/iMessage or email‑to‑SMS gateway (unreliable).

---

## Track B: Network-Integrated

- Auth: Ring Leader SSO
- Billing: MemberPress
- Runtime: OpenClaw + network intelligence

| Tier | Infra | Channels | Network Intel |
|------|-------|----------|---------------|
| Free | Shared | Demo only | ❌ |
| FreeWire | Shared | Web only | ❌ |
| Wire | Shared | Discord/Telegram | ✅ |
| ExtraWire | Shared | Full shared channels | ✅ Deep |
| Sovereign | Dedicated | Full OpenClaw | ✅ Deep |

---

## OpenClaw Config (Shared Gateway)

```json5
{
  gateway: { port: 18789, auth: { mode: "token", token: "<shared>" } },
  skills: { load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] } },
  plugins: { load: { paths: ["/home/wirebot/wirebot-core/plugins"] } },
  agents: { list: [{ id: "user_1" }, { id: "user_2" }] },
  bindings: [
    { agentId: "user_1", match: { channel: "discord", peer: { kind: "dm", id: "123" } } }
  ]
}
```

---

## SMS Reality

OpenClaw does **not** include Twilio SMS.

Options (supported plan):
- Android node (`sms.send`)
- iMessage SMS (macOS channel)
- Email‑to‑SMS gateways (carrier addresses, unreliable)

Optional: toll‑free SMS provider later (no A2P 10DLC).

---

## Network Intelligence (Track B Only)

- Similar founders
- Connection suggestions
- Event recommendations
- Content curation

Injected into Letta + Mem0 per user.

---

## Upgrade Path

Lower tier → top tier:
- Provision OpenClaw container
- Migrate Letta agent + Mem0 namespace
- Enable full channels

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Network integration
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Phase roadmap (with status)
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's deployed now
- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
