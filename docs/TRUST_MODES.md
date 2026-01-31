# Wirebot Trust Modes (Clawdbot-Based)

> **Trust modes are product policy. Enforced via tiers + skills + channels.**

---

## Overview

Trust modes still apply, but enforcement is **not** a custom gateway.

Enforcement points:
- WordPress plugin (tier + entitlements)
- Clawdbot config (skills allowlist, channel enablement)
- Dedicated vs shared infrastructure

---

## Mode Overview

| Mode | Name | Who | Infrastructure |
|------|------|-----|----------------|
| 0 | Public / Demo | Anyone | Shared gateway |
| 1 | Standard Founder | Paid users | Shared gateway |
| 2 | Advanced Trusted | Vetted users | Shared gateway |
| 3 | Sovereign | Owner + top tier | **Dedicated Clawdbot** |

---

## Enforcement Mechanisms

### 1) Skills Allowlist

```json5
skills: {
  allowBundled: ["wirebot-accountability", "wirebot-business", "wirebot-memory"],
  load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] }
}
```

### 2) Channel Enablement

Channels can be enabled/disabled per tier:

```json5
channels: {
  discord: { enabled: true },
  telegram: { enabled: true },
  whatsapp: { enabled: false }
}
```

### 3) Per-Channel Skill Filters

Discord/Telegram can restrict skills per channel:

```json5
channels: {
  discord: {
    guilds: {
      "my-guild": {
        channels: {
          "general": { skills: ["wirebot-accountability"] }
        }
      }
    }
  }
}
```

### 4) Dedicated vs Shared

Mode 3 = dedicated Clawdbot container.
Lower modes = shared gateway with bindings.

---

## Mode Details

### Mode 0 — Public / Demo
- No personalization
- Limited skills
- Shared gateway

### Mode 1 — Standard Founder
- Core skills (accountability, business)
- Shared gateway
- Limited channels

### Mode 2 — Advanced Trusted
- Extended skills (memory/patterns)
- Shared gateway
- More channels

### Mode 3 — Sovereign
- Dedicated Clawdbot container
- Full channels (including WhatsApp)
- Full skills + autonomy

---

## Notes

- Clawdbot default DM policy is **pairing**.
- For SMB onboarding, automate pairing or set `dmPolicy: "open"` + allowlist.

---

## See Also

- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix per tier
- [GATEWAY.md](./GATEWAY.md) — Gateway config for trust enforcement
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Infrastructure models (shared vs dedicated)
- [PROVISIONING.md](./PROVISIONING.md) — Tier-based provisioning
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Auth enforcement
