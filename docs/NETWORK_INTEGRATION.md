# Startempire Wire Network Integration

> **How Wirebot integrates into the Startempire Wire ecosystem.**

---

## Big Picture Map

```
StartempireWire.com (MemberPress + BuddyBoss)
        │
        ▼
Ring Leader Plugin (identity + network graph)
        │
        ├─ Connect Plugin (member sites)
        ├─ Chrome Extension (network UI)
        └─ Wirebot Plugin (auth + provisioning)
                 │
                 ▼
           OpenClaw Runtime
                 │
                 ▼
        Letta + Mem0 (memory)
```

---

## Integration Layers

### 1) Identity + Tier

- **MemberPress** determines tier
- **Ring Leader** exposes network identity
- **Wirebot Plugin** routes:
  - Top tier → dedicated OpenClaw
  - Lower tier → shared gateway

### 2) Network Context

Wirebot network skill pulls context from Ring Leader:

**Expected endpoints (example):**
- `/ring-leader/v1/member/{id}`
- `/ring-leader/v1/member/{id}/connections`
- `/ring-leader/v1/events/recommended?member={id}`
- `/ring-leader/v1/content/recommended?member={id}`
- `/ring-leader/v1/directory/search?industry=...`

Context injected into Letta + Mem0.

### 3) Surfaces

| Surface | Source | Role |
|---------|--------|------|
| Web UI | Wirebot WP plugin | Private founder sessions |
| Chrome Extension | Connect + Ring Leader | Network browsing + AI sidebar |
| Discord | Network community | Presence + summaries |
| Member Sites | Connect Plugin | Optional Wirebot widget |

---

## Track B Flow (Network Users)

```
Member logs in (startempirewire.com)
    │
    ▼
Wirebot plugin detects tier + member_id
    │
    ├─ Top tier → provision dedicated OpenClaw
    └─ Lower tier → shared gateway (agents.list + bindings)
    │
    ▼
Wirebot skills call Ring Leader APIs
    │
    ▼
Letta + Mem0 updated with network context
    │
    ▼
User interacts via web / Discord / extension
```

---

## Network Intelligence Skill

Core functions:
- Similar founders
- Connection suggestions
- Event recommendations
- Content curation
- Intro drafting

**Gated by tier:** Wire+ only.

---

## Provisioning Rules

| Tier | Infrastructure | Network Intel |
|------|----------------|---------------|
| Wire/ExtraWire | Shared gateway | ✅ |
| Sovereign | Dedicated OpenClaw | ✅ |

---

## Security + Consent

- Network context only for authenticated members
- No cross-member leakage
- OpenClaw default DM policy = pairing
- Channel allowlists must be managed by plugin

---

## Notes

- Shared gateway must use `agents.list + bindings`
- Ring Leader API must expose member context endpoints
- SMS requires Android node, iMessage SMS, or email‑to‑SMS gateway

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [DUAL_TRACK_PLAN.md](./DUAL_TRACK_PLAN.md) — Business plan
- [PLUGIN.md](./PLUGIN.md) — WordPress plugin spec
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Phase 3 (Network Integration)
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Deployment status
- [TRUST_MODES.md](./TRUST_MODES.md) — Tier-gated features
