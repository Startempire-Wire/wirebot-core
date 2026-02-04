# Wirebot Launch Order (OpenClaw + Dual-Path)

> **OpenClaw base. Full power available day 0.**
>
> See [CURRENT_STATE.md](./CURRENT_STATE.md) for what's actually deployed.

---

## Phase 0: Foundation (OpenClaw) — 🟡 In Progress

**Goal:** Your OpenClaw + shared gateway + provisioning ready.

### Deliverables

| # | Component | Status | Notes |
|---|-----------|--------|-------|
| 1 | OpenClaw install (yours) | ✅ Done | v2026.1.24-3, Node 22.22.0 |
| 2 | Gateway config + auth | ✅ Done | Token auth, loopback, systemd service |
| 3 | Auth profiles (Anthropic + OpenRouter) | ✅ Done | OAuth + API key from rbw |
| 4 | Systemd service + rbw secrets | ✅ Done | `openclaw-gateway.service`, auto-restart |
| 5 | Cloudflare tunnel | ✅ Done | helm.wirebot.chat → 127.0.0.1:18789 |
| 6 | Wirebot skills | ✅ Done | 4 skills loaded via extraDirs |
| 7 | Operations docs | ✅ Done | Full ops, auth, monitoring, troubleshooting |
| 8 | Letta server | ❌ Not started | Multi-tenant ready needed |
| 9 | Mem0 server | ❌ Not started | Multi-tenant ready needed |
| 10 | Shared gateway config | ⬜ Template only | `agents.list + bindings` pattern documented |
| 11 | Provisioning script | ⬜ Skeleton only | Per-user container provisioning |
| 12 | WordPress plugin stub | ❌ Not started | Tier routing + channel setup |

### Key Config Facts (Actual)

- Config file: `/data/wirebot/users/verious/openclaw.json` (JSON5, mode 600)
- State dir: `/data/wirebot/users/verious/` (mode 700)
- Skills dir: `/home/wirebot/wirebot-core/skills/` (4 skills)
- Plugins: `memory-core` (built-in)
- Auth: Anthropic OAuth (Claude Max 5x) + OpenRouter API key (rbw vault)
- Service: `openclaw-gateway.service` (systemd, enabled)
- Node: v22.22.0 via nvm
- Secrets: rbw (Bitwarden vault) → tmpfs injection

### Remaining for Phase 0

1. Deploy Letta server (structured business state)
2. Deploy Mem0 server (cross-surface sync)
3. Create working provisioning automation
4. Start WordPress plugin stub

---

## Phase 1: Dogfooding — ⬜ Not Started

**Prerequisite:** Phase 0 complete (Letta + Mem0 deployed).

Use daily. Refine skills + channel behavior.

- Morning standup via WebChat / Control UI
- EOD reflection
- Weekly planning
- Proactive nudges (cron)
- Test memory recall quality
- Tune model + skill behavior

**Can partially start now** with OpenClaw memory (built-in) + skills, even without Letta/Mem0.

---

## Phase 2: Rollout Prep — ⬜ Not Started

- Automate provisioning (API, not manual script)
- Channel setup UI (WordPress admin)
- Shared gateway monitoring dashboard
- Document SMS path
- Load testing (shared gateway with multiple agents)

---

## Phase 3: Network Integration — ⬜ Not Started

- Ring Leader SSO integration
- Network intelligence skill (calls Ring Leader APIs)
- Track A vs Track B routing in WordPress plugin
- Member provisioning webhook flow

---

## Phase 4: Scale — ⬜ Not Started

**Trigger:** ~50 users

- Move OpenClaw + Letta + Mem0 to separate VPS
- Podman pods or lightweight orchestration
- Dedicated gateway provisioning automation
- Monitoring + alerting at scale

---

## See Also

- [CURRENT_STATE.md](./CURRENT_STATE.md) — Detailed deployment status
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [INSTALLATION.md](./INSTALLATION.md) — Setup procedure
- [OPERATIONS.md](./OPERATIONS.md) — Service management
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Phase 3 details
- [DUAL_TRACK_PLAN.md](./DUAL_TRACK_PLAN.md) — Business plan
