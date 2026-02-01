# Business Performance Scoreboard — Product Spec

> A standalone, API-driven, TV-ready visual scoreboard for any bot or business.
> First customer: Wirebot. First user: Verious. First market: Startempire Wire Network.

---

## Product Definition

**Business Performance Scoreboard** is a real-time, evidence-backed visual display that answers one question:

> *"Am I winning today?"*

It is **not** a task manager, habit tracker, or vanity dashboard.  
It is an **execution accountability surface** — like a sports scoreboard for building a business.

### Properties
- **Always visible** — TV, wall display, browser tab, mobile, LED
- **Evidence-driven** — scores from APIs and verified artifacts, not self-reporting
- **Time-boxed** — daily games, weekly matchups, 90-day seasons
- **Anti-gaming** — only externally verifiable actions score
- **API-first** — any bot, tool, or integration can push events
- **Artifact preservation** — every score is backed by a permanent audit trail

---

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                    INTEGRATIONS (Event Sources)                │
│                                                                │
│  GitHub  Stripe  Email  LinkedIn  YouTube  Blog RSS  Beads    │
│  Vercel  Docker  DNS    CRM      Calendar  Custom Webhooks    │
│                                                                │
└──────────────────────┬───────────────────────────────────────┘
                       │ Events (webhook / poll / push)
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                    EVENT INGESTION API                         │
│                                                                │
│  POST /v1/events       — Push an event                        │
│  POST /v1/events/batch — Push multiple events                 │
│  GET  /v1/events       — Query event log                      │
│                                                                │
│  Every event: source, timestamp, artifact, confidence,        │
│               verifiers[], score_delta, metadata              │
│                                                                │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                    SCORE ENGINE                                │
│                                                                │
│  Outcome classifier (rules + heuristics)                      │
│  Lane scoring (Shipping 40%, Distribution 25%,                │
│                Revenue 20%, Systems 15%)                       │
│  Streak calculator                                            │
│  Season aggregator                                            │
│  Multipliers + penalties                                      │
│                                                                │
└──────────────────────┬───────────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────────┐
│                    DISPLAY SURFACES                            │
│                                                                │
│  Stadium Mode (TV/wall)    — Giant score, streaks, clock      │
│  Dashboard Mode (browser)  — Full detail, lanes, feed         │
│  Mobile Mode (phone)       — Quick glance, notifications      │
│  API Mode (bots)           — JSON for Wirebot/any AI          │
│  Social Cards (share)      — Auto-generated, non-editable     │
│  Wrapped (quarterly)       — Spotify-style retrospective      │
│                                                                │
└──────────────────────────────────────────────────────────────┘
```

---

## API Specification

### Base URL
`https://api.wirebot.chat/scoreboard/v1`

(Initially served from the same VPS. Separate domain later if it becomes its own product.)

### Authentication
- API key per user/bot (header: `Authorization: Bearer <key>`)
- Wirebot uses its gateway token internally
- External bots get their own keys via dashboard

### Core Endpoints

#### Events

```
POST /v1/events
{
  "event_type": "PRODUCT_RELEASE",
  "source": "github",
  "timestamp": "2026-02-01T20:00:00Z",
  "artifact": {
    "type": "release",
    "url": "https://github.com/org/repo/releases/tag/v1.0",
    "title": "v1.0.0 — Multi-business architecture",
    "metadata": {}
  },
  "confidence": 0.96,
  "verifiers": ["github_api", "deploy_webhook"],
  "business_id": "optional-uuid",
  "lane": "shipping"
}

Response:
{
  "event_id": "uuid",
  "score_delta": 12,
  "new_daily_score": 52,
  "streak": { "current": 4, "best": 7 }
}
```

#### Score

```
GET /v1/score
GET /v1/score?date=2026-02-01
GET /v1/score?range=week
GET /v1/score?range=season

Response:
{
  "date": "2026-02-01",
  "execution_score": 65,
  "lanes": {
    "shipping": 32,
    "distribution": 15,
    "revenue": 0,
    "systems": 18
  },
  "streak": { "current": 1, "best": 1 },
  "season": {
    "name": "Red-to-Black",
    "day": 1,
    "remaining": 89,
    "avg_score": 65,
    "record": "1W-0L"
  }
}
```

#### Scoreboard (Display-Ready)

```
GET /v1/scoreboard
GET /v1/scoreboard?mode=stadium
GET /v1/scoreboard?mode=dashboard
GET /v1/scoreboard?mode=mobile

Response (stadium mode):
{
  "mode": "stadium",
  "score": 65,
  "possession": "Wirebot",
  "ship_today": 1,
  "streak": 1,
  "record": "1W-0L",
  "season_day": "Day 1 of 90",
  "last_ship": "Multi-business architecture deployed",
  "clock": {
    "day_progress": 0.83,
    "week_progress": 0.14,
    "season_progress": 0.01
  },
  "signal": "green"
}
```

#### Feed (Activity Log)

```
GET /v1/feed
GET /v1/feed?date=2026-02-01
GET /v1/feed?type=shipping

Response:
{
  "items": [
    {
      "id": "uuid",
      "timestamp": "2026-02-01T20:00:00Z",
      "icon": "🟢",
      "type": "PRODUCT_RELEASE",
      "title": "Released multi-business architecture",
      "source": "github",
      "score_delta": 12,
      "artifact_url": "https://github.com/...",
      "confidence": 0.96
    }
  ]
}
```

#### Audit (Spreadsheet View)

```
GET /v1/audit
GET /v1/audit?format=csv
GET /v1/audit?range=week&lane=shipping

Response: Full event log with score derivation — every point traceable to an event.
```

#### Season

```
GET /v1/season
POST /v1/season  — Start new season
GET /v1/season/wrapped  — Quarterly/yearly retrospective data

Response (wrapped):
{
  "season": "Red-to-Black",
  "duration_days": 90,
  "total_ships": 47,
  "best_streak": 12,
  "total_revenue_events": 8,
  "days_won": 52,
  "record": "52W-38L",
  "top_artifacts": [...],
  "patterns": {
    "best_day_of_week": "Tuesday",
    "best_lane": "shipping",
    "avg_score_trend": "↑"
  }
}
```

#### Social Cards

```
GET /v1/card/daily
GET /v1/card/weekly
GET /v1/card/season

Response: PNG or SVG image, auto-generated, non-editable.
Stats shown: score, streak, record, top artifact.
```

---

## Event Taxonomy

### Shipping Lane (40% weight)
| Event Type | Source | Confidence | Points |
|-----------|--------|-----------|--------|
| `PRODUCT_RELEASE` | GitHub/registry | 0.95+ | 10 |
| `DEPLOY_SUCCESS` | Vercel/Docker/systemd | 0.95+ | 8 |
| `FEATURE_SHIPPED` | GitHub PR merged to main | 0.90+ | 6 |
| `APP_STORE_SUBMIT` | App Store/Play Console | 0.95+ | 10 |
| `PUBLIC_ARTIFACT` | Published URL | 0.85+ | 5 |
| `INFRASTRUCTURE_ACTIVATED` | DNS/systemd/container | 0.90+ | 7 |

### Distribution Lane (25% weight)
| Event Type | Source | Confidence | Points |
|-----------|--------|-----------|--------|
| `BLOG_PUBLISHED` | RSS/CMS/git | 0.90+ | 6 |
| `VIDEO_PUBLISHED` | YouTube API | 0.95+ | 7 |
| `EMAIL_CAMPAIGN_SENT` | ESP API | 0.90+ | 5 |
| `SOCIAL_POST_BUSINESS` | Social API + AI score | 0.70+ | 3-5 |
| `COLD_OUTREACH` | CRM/email | 0.80+ | 4 |
| `PODCAST_PUBLISHED` | RSS feed | 0.90+ | 6 |

### Revenue Lane (20% weight)
| Event Type | Source | Confidence | Points |
|-----------|--------|-----------|--------|
| `PAYMENT_RECEIVED` | Stripe webhook | 0.99 | 10 |
| `SUBSCRIPTION_CREATED` | Stripe webhook | 0.99 | 12 |
| `DEAL_CLOSED` | CRM | 0.85+ | 8 |
| `PROPOSAL_SENT` | Email/CRM | 0.80+ | 4 |
| `INVOICE_PAID` | Stripe/accounting | 0.99 | 8 |

### Systems Lane (15% weight)
| Event Type | Source | Confidence | Points |
|-----------|--------|-----------|--------|
| `AUTOMATION_DEPLOYED` | Webhook/n8n/cron | 0.85+ | 6 |
| `SOP_DOCUMENTED` | Published URL/wiki | 0.80+ | 4 |
| `TOOL_INTEGRATED` | API connection | 0.85+ | 5 |
| `DELEGATION_COMPLETED` | CRM/task system | 0.80+ | 6 |
| `MONITORING_ENABLED` | Alert webhook | 0.85+ | 4 |

### Penalties (Deductions)
| Event Type | Trigger | Points |
|-----------|---------|--------|
| `CONTEXT_SWITCH` | 3rd+ switch in a day | -5 each |
| `NO_SHIP_DAY` | EOD with zero shipping events | -0 (but can't win) |
| `COMMITMENT_BREACH` | Declared intent not fulfilled | -10 |

---

## Display Modes

### Stadium Mode (TV / Wall / LED)

```
┌─────────────────────────────────────────────────────────┐
│  RED-TO-BLACK          Season 1          Day 15 of 90   │
│                                                          │
│                        ┌─────┐                           │
│                        │ 72  │  ← EXECUTION SCORE        │
│                        └─────┘                           │
│                                                          │
│  🔥 STREAK: 5 days          RECORD: 12W-3L              │
│                                                          │
│  ⚡ POSSESSION: Wirebot                                  │
│                                                          │
│  SHIPPING ████████░░░ 32/40   REVENUE ██░░░░░░░░ 8/20   │
│  DISTRIB  █████░░░░░ 18/25   SYSTEMS ████████░░ 14/15   │
│                                                          │
│  LAST SHIP: Dashboard v0.1 deployed (2h ago)            │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│  ⏰ Day: ████████████████░░░░ 83%   Week: ██░░░░ 28%    │
└─────────────────────────────────────────────────────────┘
```

Design rules:
- **High contrast** — readable from 10 feet on a TV
- **Huge typography** — score is the biggest element
- **Color-only state** — green (winning), yellow (pressure), red (stalling)
- **No scroll, no drill-down** — everything visible at once
- **Flashing alerts** — "No ship in 36h" blinks red
- **Audible "ship confirmed" signal** (optional)

### Dashboard Mode (Browser)

Full detail: all 4 lanes with breakdowns, activity feed, audit trail, streak history, season progress, business health integration.

### Mobile Mode

Quick glance: score, streak, last ship, next intent. Push notification on ship confirmation or stall warning.

### Wrapped Mode (Quarterly)

Auto-generated story cards showing:
- Your record this season
- Shipping velocity over time
- Best streak + when it happened
- Top 5 artifacts shipped
- Revenue trajectory
- Patterns: best day, best lane, best time of day
- Comparison to previous season
- Shareable social cards

---

## Wirebot Integration

### Wirebot as Event Source
Wirebot pushes events to the scoreboard API automatically:
- `wb complete <task>` → `FEATURE_SHIPPED` or `PUBLIC_ARTIFACT` event
- `wb add-business` → `INFRASTRUCTURE_ACTIVATED` event
- Mem0 fact storage → no score (behavior plane)
- Chat interaction → no score (reflection plane)

### Wirebot as Display Consumer
- `wb score` → calls `/v1/score`, shows today's execution score
- `wb scoreboard` → calls `/v1/scoreboard?mode=mobile`, shows compact view
- `wb streak` → shows current + best ship streak
- `wb season` → shows season progress + record
- `wb feed` → shows recent activity feed
- Morning standup includes score + streak + season clock

### Any Bot as Event Source
The API is generic. Any bot or tool can push events:
- n8n automation → pushes `AUTOMATION_DEPLOYED`
- GitHub Action → pushes `DEPLOY_SUCCESS`
- Stripe webhook → pushes `PAYMENT_RECEIVED`
- Custom script → pushes any event type

---

## Tech Stack (V1)

| Component | Technology | Why |
|-----------|-----------|-----|
| API server | Go (extend `wirebot-memory-syncd`) or standalone | Fast, low memory, already have Go infra |
| Event store | SQLite (local-first) | Audit trail, queryable, exportable |
| Score engine | Go (deterministic, pure functions) | Reproducible scores |
| Stadium UI | Static HTML/CSS/JS | TV-optimized, no framework overhead |
| Dashboard UI | Svelte (matches Chrome Extension stack) | Reactive, lightweight |
| Social cards | SVG templates → PNG (sharp/canvas) | Non-editable, auto-generated |
| Hosting | `api.wirebot.chat` (port 8100, CF tunnel) | Already configured, just needs listener |

### V1 Build Order
1. Event schema + SQLite store
2. Score engine (deterministic, replayable)
3. REST API (events, score, scoreboard, feed)
4. Stadium mode HTML (TV-ready)
5. `wb score` / `wb scoreboard` CLI commands
6. Stripe webhook integration
7. GitHub webhook integration
8. Dashboard mode (Svelte)
9. Social cards
10. Wrapped retrospective

---

## Data Model

### `events` table
```sql
CREATE TABLE events (
  id TEXT PRIMARY KEY,
  event_type TEXT NOT NULL,
  lane TEXT NOT NULL,           -- shipping, distribution, revenue, systems
  source TEXT NOT NULL,         -- github, stripe, manual, wirebot, etc.
  timestamp TEXT NOT NULL,      -- ISO 8601
  artifact_type TEXT,           -- release, deploy, post, payment, etc.
  artifact_url TEXT,
  artifact_title TEXT,
  confidence REAL DEFAULT 1.0,
  verifiers TEXT,               -- JSON array of verification sources
  score_delta INTEGER DEFAULT 0,
  business_id TEXT,             -- optional, for multi-business
  metadata TEXT,                -- JSON blob
  created_at TEXT NOT NULL
);

CREATE INDEX idx_events_date ON events(timestamp);
CREATE INDEX idx_events_lane ON events(lane);
CREATE INDEX idx_events_type ON events(event_type);
```

### `daily_scores` table
```sql
CREATE TABLE daily_scores (
  date TEXT PRIMARY KEY,
  execution_score INTEGER,
  shipping_score INTEGER,
  distribution_score INTEGER,
  revenue_score INTEGER,
  systems_score INTEGER,
  penalties INTEGER DEFAULT 0,
  ships_count INTEGER DEFAULT 0,
  intent TEXT,
  intent_fulfilled BOOLEAN,
  won BOOLEAN,                  -- score >= 50 = win
  created_at TEXT NOT NULL
);
```

### `streaks` table
```sql
CREATE TABLE streaks (
  id TEXT PRIMARY KEY,
  streak_type TEXT,             -- ship, no_zero, focus
  start_date TEXT,
  end_date TEXT,
  length INTEGER,
  is_current BOOLEAN
);
```

### `seasons` table
```sql
CREATE TABLE seasons (
  id TEXT PRIMARY KEY,
  name TEXT,
  number INTEGER,
  start_date TEXT,
  end_date TEXT,
  theme TEXT,
  total_score INTEGER DEFAULT 0,
  days_won INTEGER DEFAULT 0,
  days_played INTEGER DEFAULT 0,
  is_active BOOLEAN
);
```

---

## Open Questions

1. **Standalone product or Wirebot feature?** → Both. API-first means it works either way.
2. **Pricing?** → Free tier (1 business, basic lanes). Paid tier (multi-business, integrations, Wrapped, social cards). Part of Startempire Wire membership.
3. **Name?** → "Scoreboard" is generic. Working names: ShipScore, GameDay, BoardRoom, ScoreWire.
4. **Domain?** → `score.wirebot.chat`? `scoreboard.startempirewire.com`? Separate domain?

---

## See Also

- [SCOREBOARD.md](./SCOREBOARD.md) — Wirebot-internal integration concepts
- [MULTI_BUSINESS.md](./MULTI_BUSINESS.md) — Business health scoring (complementary)
- [PAIRING.md](./PAIRING.md) — Financial reality mapping feeds revenue lane
- Source: `/data/wirebot/reference/business-performance-scoreboard.md`
- Source: `/data/wirebot/reference/asset-portfolio-activation.md`
