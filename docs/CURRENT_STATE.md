# Wirebot Current State

> **What's actually deployed, running, and operational.**
>
> Last updated: 2026-02-02

---

## Phase Status

| Phase | Status | Notes |
|-------|--------|-------|
| **Phase 0: Foundation** | 🟢 Complete | Gateway, auth, memory, cron, pairing, Drift |
| **Phase 1: Dogfooding** | 🟢 Active | Dashboard, Scoreboard, Extension, Connect overlay all live |
| **Phase 2: Rollout Prep** | 🟡 Starting | Auto-provisioning built, onboarding flow ready |
| Phase 3: Network Integration | ⬜ Planned | Ring Leader content distribution |
| Phase 4: Scale | ⬜ Planned | Multi-tenant beta test |

---

## Surfaces (All Live)

### 🏠 Scoreboard PWA — `wins.wirebot.chat`
- **Stack**: Go + Svelte 5 + SQLite + PWA
- **Views**: Dashboard (Home), Score, Feed, Season, Wrapped, Settings
- **Auth**: Operator token + Ring Leader JWT
- **Score**: 85/100 (Ship 40/40, Dist 25/25, Rev 5/20, Sys 15/15)
- **API**: 60+ endpoints (events, score, feed, chat, pairing, drift, integrations, reconciliation)
- **Features**: 4-lane scoring, approve-once trust, Drift system, R.A.B.I.T., pairing v2, chat, share cards

### 🧩 Chrome Extension — v0.2.2
- **Components**: WirebotTab (chat), ProfileSummary, Drift card, R.A.B.I.T. alert, WebRing, NetworkOverlay
- **Auth**: WP login → Ring Leader JWT
- **Download**: `https://wins.wirebot.chat/sewn-extension-v0.2.2.zip`

### 🔌 Connect Plugin — v0.3.1
- **Deployed**: startempirewire.com
- **Features**: Wirebot overlay widget, Drift bar, R.A.B.I.T. alert, Ring Leader client, REST API

### 🔑 Ring Leader — v0.2.1
- **Deployed**: startempirewire.network
- **Endpoints**: auth (validate/token/issue), content (posts/events/podcasts/activity/directory), member (me/scoreboard/profile/integrations), network (stats/members)
- **Features**: JWT issuance, profile sync, preferences, auto-provisioning, tier-gated content
- **Security**: Tier derived from JWT only (no client-provided tier spoofing)

### 🤖 Gateway — OpenClaw v2026.1.30
- **URL**: helm.wirebot.chat → port 18789
- **Tools**: wirebot_recall, wirebot_remember, wirebot_business_state, wirebot_checklist, wirebot_score
- **Models**: Kimi → GLM → OpenRouter (3-tier fallback)

### 🌐 startempirewire.com
- **Stack**: WordPress + BuddyBoss + MemberPress + Connect Plugin
- **Auth**: MemberPress → Ring Leader JWT relay
- **SSO**: mu-plugin for auto-redirect to scoreboard

---

## Memory Systems (All Operational)

| System | Port | Purpose | Status |
|--------|------|---------|--------|
| **memory-core** | embedded | Workspace file recall (BM25+vector) | ✅ Active |
| **Mem0** | 8200 | Conversation facts (fastembed, 80+ memories) | ✅ Active |
| **Letta** | 8283 | Structured business state (4 blocks, PostgreSQL) | ✅ Active |
| **memory-syncd** | 8201 | Hot cache + sync daemon (Go, sub-ms) | ✅ Active |

---

## Pairing Engine v2 (Fully Operational)

- **Score**: 60/100 (Partner), 47.5% accuracy
- **Instruments**: 7 (ASI-12, CSI-8, ETM-6, RDS-6, COG-8, BIZ-6, TIME-6) — 47 questions
- **Signals**: 1,326 total (879 messages, 421 events, 853 vault docs)
- **Drift**: 78/100 (IN DRIFT), modesty reflex 0.375 (open)
- **R.A.B.I.T.**: Clear
- **Evidence**: 455+ records in SQLite
- **Vault**: 853 Obsidian docs ingested (2.96M words, 2013–2025)

---

## Integrations (6 Active)

| Integration | Type | Interval | Status |
|-------------|------|----------|--------|
| Blog RSS × 2 | Poller | 15 min | ✅ |
| Cloudflare | Poller | Daily | ✅ 50 zones |
| WooCommerce | Poller | 30 min | ✅ |
| Sendy | Poller | Hourly | ✅ 4 brands, 1,070 subs |
| Stripe | Webhook | Real-time | ✅ 9 event types |
| MemberPress | Webhook | Real-time | ✅ mu-plugin |

**Ready but need credentials**: FreshBooks, PostHog, UptimeRobot, RescueTime, Discord, HubSpot, GitHub OAuth, Google OAuth

---

## Cron Jobs

| Job | Schedule | Script |
|-----|----------|--------|
| Git Discovery Watch | */5 min | `/data/wirebot/bin/wb-discover watch` |
| Systems Health | Daily 6 AM PT | `/data/wirebot/bin/wb-systems-check` |
| Obsidian Vault Sync | Daily 4 AM PT | `/data/wirebot/bin/obsidian-sync.sh` |
| Weekly Memory Sync | Sunday midnight | `/data/wirebot/bin/memory-sync.sh` |

---

## CLI (`wb`)

1,290 lines, 30+ commands across 7 sections:
- **Pairing**: pair, pair status, pair skip, pair reset
- **Businesses**: overview, businesses, focus, add-business
- **Checklist**: status, next, daily, complete, skip, add, list, detail, stage
- **Memory**: recall, remember, state, cache, memory, sync
- **System**: health, services, logs, pillars
- **Scoreboard**: score, streak, season, feed, ship, submit, pending, approve, reject, intent, discover, projects, lock, audit, wins, card
- **Drift**: drift, handshake, rabbit

---

## GitHub Repos

| Repo | Latest Commit | Lines |
|------|---------------|-------|
| wirebot-core | `1a4086d` Dashboard + onboarding | ~15K Go + ~3K Svelte |
| Startempire-Wire-Network | `abddcce` Extension v0.2.2 | ~2K Svelte |
| Startempire-Wire-Network-Connect | `e023388` Overlay v0.3.1 | ~1.5K PHP+JS |
| Startempire-Wire-Network-Ring-Leader | `0eb69f1` Auto-provisioning | ~1.5K PHP |
| focusa | Cognitive memory OS specs | 55 docs |

---

## Key Metrics

| Metric | Value |
|--------|-------|
| Execution Score | 85/100 |
| Ship Streak | 🔥 2 days |
| Season | Red-to-Black (Day 1) |
| Record | 1W-1L |
| Ships Today | 39 |
| Drift Score | 78/100 (IN DRIFT) |
| Pairing Accuracy | 47.5% |
| Total Events | ~300+ |
| Approved Sources | 9 |
| Active Services | 7 |

---

## Next Steps

1. **Connect FreshBooks** — Paste Bearer Token + Account ID in Settings
2. **Create OAuth apps** — GitHub, Stripe, Google (manual, 2 min each)
3. **Beta test** — Invite first external user through full onboarding flow
4. **Revenue lane** — Get real financial data flowing (currently 5/20)
5. **Multi-account per provider** — UI "Add another" button
