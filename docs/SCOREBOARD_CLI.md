# Scoreboard CLI Reference

> All scoreboard commands are available through `wb` (Wirebot CLI).
> API: `http://127.0.0.1:8100` (internal) / `https://wins.wirebot.chat` (public stadium)
> Auth: Bearer token `$TOKEN` (same as gateway token)

---

## Quick Start

```bash
wb score                    # Am I winning today?
wb ship "Deployed v2.0"     # Log a ship (+points)
wb intent "Ship the MVP"    # Declare today's goal
wb feed                     # See what happened
wb scoreboard               # Quick glance
```

---

## Score Commands

### `wb score [date]`
Show today's (or a specific date's) execution score with full lane breakdown.

```bash
$ wb score
⚡ EXECUTION SCORE: 37
  SHIPPING:     31/40
  DISTRIBUTION: 3/25
  REVENUE:      0/20
  SYSTEMS:      3/15
  Ships today:  5
  Intent:       🎯 Ship full scoreboard PWA
  Result:       ❌ LOSS (need ≥50 to win)
  Streak:       🔥 1 days (best: 1)
  Season:       Red-to-Black — Day 0 — 0W-1L

$ wb score 2026-02-01       # Check a specific date
```

**Exit codes:** 0 = success, 1 = API unreachable

### `wb streak`
Show current and best ship streaks.

```bash
$ wb streak
🔥 Ship Streak: 1 days
🏆 Best Streak: 1 days
🚀 Last Ship:   Scoreboard PWA: 3-view nav
📅 Last Date:   2026-02-01
```

A "ship day" = any day with at least one shipping-lane event.

### `wb scoreboard` (alias: `wb wins`)
Compact one-screen scoreboard view. Shows everything at a glance.

```bash
$ wb scoreboard
🌐 wins.wirebot.chat

🟡 37 — PRESSURE
🔥 Streak: 1  🏆 Best: 1  📊 0W-1L  🚀 Ships: 5
⚡ Startempire Wire
SHIP 31/40  DIST  3/25  REV  0/20  SYS  3/15
Red-to-Black — Day 0 of 88
```

**Signals:** 🟢 GREEN (≥50, WINNING) | 🟡 YELLOW (30-49, PRESSURE) | 🔴 RED (<30, STALLING)

---

## Ship & Intent

### `wb ship "<title>" [--lane <lane>] [--url <artifact-url>]`
Log a ship event. This is the primary way to score points from the terminal.

```bash
$ wb ship "Deployed scoreboard v2"
🚀 SHIPPED: Deployed scoreboard v2
   +5 points → Score: 42
   🔥 Streak: 1 days

$ wb ship "Published blog post" --lane distribution
$ wb ship "Set up monitoring" --lane systems --url https://status.example.com
```

**Lanes:** `shipping` (default), `distribution`, `revenue`, `systems`

**Points by event type:**
| Event | Lane | Points |
|-------|------|--------|
| `FEATURE_SHIPPED` | shipping | 6 |
| `PRODUCT_RELEASE` | shipping | 10 |
| `DEPLOY_SUCCESS` | shipping | 8 |
| `PUBLIC_ARTIFACT` | shipping/distribution | 5 |
| `BLOG_PUBLISHED` | distribution | 6 |
| `VIDEO_PUBLISHED` | distribution | 7 |
| `PAYMENT_RECEIVED` | revenue | 10 |
| `SUBSCRIPTION_CREATED` | revenue | 12 |
| `AUTOMATION_DEPLOYED` | systems | 6 |
| `SOP_DOCUMENTED` | systems | 4 |

### `wb intent ["<text>"]`
Declare what you intend to ship today. Called without arguments, shows the current intent.

```bash
$ wb intent "Ship the checkout flow"
🎯 Intent locked: Ship the checkout flow

$ wb intent                 # Show current
🎯 Today: Ship the checkout flow
```

Intents are tracked per-day. Unfulfilled intents are visible in the audit trail.
Future: unfulfilled intents will trigger `COMMITMENT_BREACH` (-10 points).

---

## Feed & Audit

### `wb feed [limit]`
Show the activity feed — recent events with icons, lane tags, and score deltas.

```bash
$ wb feed
📋 Activity Feed (7 events)
──────────────────────────────────────────────────
🚀 + 5  Scoreboard PWA: 3-view nav             [shipping] 2026-02-01 22:17
🚀 + 7  wins.wirebot.chat tunnel configured     [shipping] 2026-02-01 22:05
⚙️ + 3  wirebot-scoreboard service live         [systems] 2026-02-01 22:05
📣 + 3  Scoreboard product spec                 [distribution] 2026-02-01 22:05

$ wb feed 5                 # Show last 5 events only
```

**Icons:** 🚀 shipping | 📣 distribution | 💰 revenue | ⚙️ systems

### `wb audit [lane]`
Full audit trail — every event with ID, type, lane, points, source, and title.
Every point in your score is traceable to a specific event.

```bash
$ wb audit
📊 Audit Trail (7 records)
ID                   Type                   Lane          Pts Source       Title
────────────────────────────────────────────────────────────────────────────────────
evt-17699852752301   FEATURE_SHIPPED        shipping       +5 wb-cli       Scoreboard PWA
evt-17699835588299   DEPLOY_SUCCESS         shipping       +7 cloudflare   wins.wirebot.chat

$ wb audit shipping         # Filter by lane
$ wb audit revenue          # Show only revenue events
```

**CSV export:** Use the API directly:
```bash
curl -s "http://127.0.0.1:8100/v1/audit?format=csv&token=$TOKEN" > audit.csv
```

---

## Season

### `wb season`
Show season progress, record, and theme.

```bash
$ wb season
🏆 Red-to-Black — Season 1
   2026-02-01 → 2026-05-01
   Progress: [░░░░░░░░░░░░░░░░░░░░] 0%
   Day 0 of 88 (88 remaining)
   Record: 0W-1L
   Avg Score: 32
   "Break even. Ship what makes money. Get out of the red."
```

**Win condition:** Daily score ≥50 = WIN. Below 50 = LOSS.
**Season end:** Scores reset, history preserved, new season starts.

---

## Score Engine Rules

### Lane Caps
Each lane has a maximum daily contribution:
- **Shipping:** 40 points max (40% weight)
- **Distribution:** 25 points max (25% weight)
- **Revenue:** 20 points max (20% weight)
- **Systems:** 15 points max (15% weight)
- **Total possible:** 100 points/day

### No-Ship Penalty
If zero shipping-lane events for the day, score is **capped at 30** regardless of other lane activity. You can't win a day without shipping something.

### Streak Rules
- Ship streak = consecutive days with ≥1 shipping-lane event
- Streak resets on a day with zero shipping events
- Best streak is preserved permanently

### Win/Loss
- Score ≥ 50 → **WIN** ✅
- Score < 50 → **LOSS** ❌
- Record tracked per season (e.g., 12W-3L)

### Gated Events (Approval Required)
Some events can be submitted with `"status": "pending"` and require operator approval before they count toward the score. This prevents agents or automated systems from inflating the score without human verification.

```bash
# Submit a gated event (won't score until approved)
wb submit "Redesigned landing page" --lane shipping

# Review pending events
wb pending

# Approve a pending event (now it scores)
wb approve <event-id>

# Reject a pending event (removed, no score)
wb reject <event-id>
```

**Use cases:**
- AI agents logging work that needs human verification
- Automated CI/CD events that need quality check
- Self-reported items that need artifact proof
- Any event where confidence < 0.80

---

## Auto-Events

The following `wb` commands automatically push events to the scoreboard:

| Command | Event Type | Lane | Points |
|---------|-----------|------|--------|
| `wb complete <task>` | `TASK_COMPLETED` | shipping | 4 |
| `wb ship "<title>"` | `FEATURE_SHIPPED` | (specified) | 5+ |
| `wb add-business` | (planned) | systems | 7 |
| `wb stage <stage>` | (planned) | systems | 4 |

### Cron Auto-Events
The gateway cron jobs (Daily Standup, EOD Review) can push events:
- **Morning:** Reads yesterday's score, checks intent fulfillment
- **EOD:** Locks the day's score, calculates streak, detects no-ship days

---

## API Reference (for agents & scripts)

All endpoints at `http://127.0.0.1:8100`. Auth: `Authorization: Bearer <token>` or `?token=<token>`.

### Event Ingestion
```bash
# Push a single event
curl -X POST http://127.0.0.1:8100/v1/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"FEATURE_SHIPPED","lane":"shipping","source":"my-agent","artifact_title":"Built X","confidence":0.9}'

# Push a gated event (needs approval)
curl -X POST http://127.0.0.1:8100/v1/events \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"FEATURE_SHIPPED","lane":"shipping","source":"claude","artifact_title":"Refactored auth","status":"pending","confidence":0.85}'

# Batch push
curl -X POST http://127.0.0.1:8100/v1/events/batch \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"events":[{"event_type":"DEPLOY_SUCCESS","lane":"shipping","source":"github","artifact_title":"v2.0"},{"event_type":"BLOG_PUBLISHED","lane":"distribution","source":"wordpress","artifact_title":"Launch post"}]}'
```

### Reading Data
```bash
# Today's score
curl http://127.0.0.1:8100/v1/score?token=$TOKEN

# Score for a date range
curl "http://127.0.0.1:8100/v1/score?range=week&token=$TOKEN"
curl "http://127.0.0.1:8100/v1/score?range=season&token=$TOKEN"

# Public scoreboard (no auth needed)
curl http://127.0.0.1:8100/v1/scoreboard
curl "http://127.0.0.1:8100/v1/scoreboard?mode=dashboard"

# Activity feed
curl "http://127.0.0.1:8100/v1/feed?limit=20&token=$TOKEN"
curl "http://127.0.0.1:8100/v1/feed?lane=revenue&token=$TOKEN"

# Audit trail (JSON or CSV)
curl "http://127.0.0.1:8100/v1/audit?token=$TOKEN"
curl "http://127.0.0.1:8100/v1/audit?format=csv&token=$TOKEN" > audit.csv

# Season info
curl "http://127.0.0.1:8100/v1/season?token=$TOKEN"

# Season retrospective (Wrapped)
curl "http://127.0.0.1:8100/v1/season/wrapped?token=$TOKEN"

# Daily history (for heatmaps/charts)
curl "http://127.0.0.1:8100/v1/history?range=season&token=$TOKEN"

# Pending events (gated, awaiting approval)
curl "http://127.0.0.1:8100/v1/pending?token=$TOKEN"

# Approve/reject a pending event
curl -X POST "http://127.0.0.1:8100/v1/events/<id>/approve?token=$TOKEN"
curl -X POST "http://127.0.0.1:8100/v1/events/<id>/reject?token=$TOKEN"
```

### Intent
```bash
# Set today's intent
curl -X POST http://127.0.0.1:8100/v1/intent \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"intent":"Ship the checkout flow"}'

# Read today's intent
curl http://127.0.0.1:8100/v1/intent?token=$TOKEN
```

### Webhooks
```bash
# GitHub webhook URL (add to repo settings):
# POST http://127.0.0.1:8100/v1/webhooks/github?token=$TOKEN
# Events: release, pull_request (merged), workflow_run (success)

# Stripe webhook URL (add to Stripe dashboard):
# POST http://127.0.0.1:8100/v1/webhooks/stripe?token=$TOKEN
# Events: payment_intent.succeeded, customer.subscription.created, invoice.paid
```

---

## For Other Agents (Claude, Pi, Letta, OpenCode)

The scoreboard API is available to **any agent or LLM running on this server**. To integrate:

### 1. Push Events When You Ship
When you complete meaningful work (deploy, publish, configure), push an event:
```bash
curl -X POST http://127.0.0.1:8100/v1/events \
  -H "Authorization: Bearer 65b918ba-baf5-4996-8b53-6fb0f662a0c3" \
  -H "Content-Type: application/json" \
  -d '{"event_type":"FEATURE_SHIPPED","lane":"shipping","source":"<your-agent-name>","artifact_title":"<what you did>","artifact_url":"<proof>","confidence":0.9,"status":"pending"}'
```

**Important:** Use `"status":"pending"` for agent-submitted events. The operator approves them before they score. This prevents score inflation from automated systems.

### 2. Check Score Before Acting
Before deciding what to work on, check the current score to understand priorities:
```bash
curl http://127.0.0.1:8100/v1/scoreboard
```
- Score < 30 (RED)? Focus on shipping.
- Revenue lane empty? Suggest revenue-generating tasks.
- No intent declared? Ask the operator what ships today.

### 3. Read the Feed
Understand what's happened today before making recommendations:
```bash
curl "http://127.0.0.1:8100/v1/feed?limit=10&token=65b918ba-baf5-4996-8b53-6fb0f662a0c3"
```

### 4. Respect the Three Planes
- **Reality plane** (scored): Only push events for shipped, verifiable artifacts
- **Behavior plane** (tracked): Don't push events for "worked on" or "researched"
- **Reflection plane** (private): Never push events for feelings, energy, or mood

### 5. Event Confidence Guide
| Source | Confidence | Example |
|--------|-----------|---------|
| Automated (webhook/CI) | 0.95-0.99 | Stripe payment, GitHub release |
| Agent-verified (checked URL/API) | 0.85-0.95 | Confirmed deploy, published page |
| Agent-reported (unverified) | 0.70-0.85 | "I completed this task" → **use pending** |
| Self-reported (human typed) | 0.80-0.90 | `wb ship "..."` (operator says it shipped) |

---

## Paths & Services

| Component | Location |
|-----------|----------|
| Binary | `/data/wirebot/bin/wirebot-scoreboard` |
| Service | `wirebot-scoreboard.service` (systemd) |
| Database | `/data/wirebot/scoreboard/events.db` (SQLite WAL) |
| UI source | `/home/wirebot/wirebot-core/cmd/scoreboard/ui/` (Svelte) |
| Go source | `/home/wirebot/wirebot-core/cmd/scoreboard/main.go` |
| Public URL | `https://wins.wirebot.chat` (CF tunnel) |
| Internal API | `http://127.0.0.1:8100` |
| Checklist | `/home/wirebot/clawd/checklist.json` (read for possession) |
| Scoreboard JSON | `/home/wirebot/clawd/scoreboard.json` (season config) |
| Token | Same as gateway: `65b918ba-baf5-4996-8b53-6fb0f662a0c3` |

---

## See Also

- [SCOREBOARD.md](./SCOREBOARD.md) — Integration concepts & three-plane philosophy
- [SCOREBOARD_PRODUCT.md](./SCOREBOARD_PRODUCT.md) — Full product spec
- [CLI.md](./CLI.md) — Full wb CLI reference
- [MULTI_BUSINESS.md](./MULTI_BUSINESS.md) — Business health scoring (complementary to execution scoring)
