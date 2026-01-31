# Wirebot Capability Matrix (Clawdbot-Based)

> **Track × Tier × Infrastructure × Channels**

---

## Infrastructure Types

| Type | Description |
|------|-------------|
| **Dedicated** | Full Clawdbot container per user (top tier) |
| **Shared** | Single Clawdbot gateway with multi-tenant agents (lower tiers) |

---

## Track × Tier Mapping

| Track | Tiers | Notes |
|-------|-------|-------|
| **Standalone** | Free, Basic, Standard, Premium, Power | wirebot.chat |
| **Network** | Free, FreeWire, Wire, ExtraWire, Sovereign | startempirewire.com |

---

## Core Capabilities (All Tiers)

| Capability | Shared | Dedicated |
|-----------|--------|-----------|
| Business context | ✅ | ✅ |
| Accountability engine | ✅ | ✅ |
| Goals + checklists | ✅ | ✅ |
| Letta agent | ✅ (user-scoped) | ✅ (user-scoped) |
| Mem0 memory | ✅ (user-scoped) | ✅ (user-scoped) |
| Wirebot skills | ✅ | ✅ |

---

## Channel Availability

### Shared Gateway (Lower Tiers)

| Channel | Availability | Notes |
|---------|-------------|-------|
| Web UI | ✅ | WordPress plugin / Clawdbot WebChat |
| Discord | ✅ | Multi-tenant bot |
| Telegram | ✅ | Multi-tenant bot |
| SMS | ⚠️ | Android/iMessage or email‑to‑SMS (unreliable) |
| WhatsApp | ❌ | Requires dedicated container |

### Dedicated Clawdbot (Top Tier)

| Channel | Availability | Notes |
|---------|-------------|-------|
| Web UI | ✅ | Control UI + WebChat |
| Discord | ✅ | Full access |
| Telegram | ✅ | Full access |
| WhatsApp | ✅ | Own session (QR) |
| SMS | ⚠️ | Android/iMessage or email‑to‑SMS (unreliable) |
| Other Clawdbot channels | ✅ | Slack, Signal, etc. |

---

## Track A (Standalone) — Example Tiering

| Tier | Infrastructure | Channels |
|------|----------------|----------|
| Free | Shared | Demo only |
| Basic | Shared | Web only |
| Standard | Shared | Web + Discord/Telegram |
| Premium | Shared | Web + Discord/Telegram + SMS* |
| Power | Dedicated | Full Clawdbot channels |

*SMS depends on Android/iMessage or email‑to‑SMS gateway (unreliable).

---

## Track B (Network) — Example Tiering

| Tier | Infrastructure | Channels | Network Intel |
|------|----------------|----------|---------------|
| Free | Shared | Demo only | ❌ |
| FreeWire | Shared | Web only | ❌ |
| Wire | Shared | Web + Discord/Telegram | ✅ |
| ExtraWire | Shared | Full shared channels | ✅ Deep |
| Sovereign | Dedicated | Full Clawdbot channels | ✅ Deep |

---

## Network Intelligence (Track B Only)

| Feature | Wire | ExtraWire | Sovereign |
|---------|------|-----------|-----------|
| Similar founders | ✅ | ✅ | ✅ |
| Connection suggestions | ✅ | ✅ | ✅ |
| Event recommendations | ✅ | ✅ | ✅ |
| Content curation | ✅ | ✅ | ✅ |
| Intro drafting | ❌ | ✅ | ✅ |
| Deep pattern analysis | ❌ | ✅ | ✅ |

---

## Notes / Caveats

- SMS is not built into Clawdbot (no Twilio by default).
- Supported SMS paths: Android node, iMessage SMS, or email‑to‑SMS gateway.
- Toll‑free SMS provider possible later (no A2P 10DLC).
- Clawdbot default DM policy is **pairing**.
- Shared gateway must use `agents.list + bindings` for routing.

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [SMS_OPTIONS.md](./SMS_OPTIONS.md) — SMS alternatives
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Network features
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What's deployed now
- [TRUST_MODES.md](./TRUST_MODES.md) — Trust enforcement
