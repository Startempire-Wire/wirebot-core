# Business Performance Scoreboard — Wirebot Integration

> Source: `Business Performance Scoreboard` ChatGPT session (57 messages, 8092 lines, Dec 24-28 2025)
> Also: `Asset Portfolio Activation` (20 messages, 2025 lines, Dec 26 2025)
> Saved: `/data/wirebot/reference/business-performance-scoreboard.md`, `/data/wirebot/reference/asset-portfolio-activation.md`

---

## TL;DR

The Business Performance Scoreboard is a **reality-backed execution accountability system** that treats building a business like a competitive sport. It was designed as a standalone SaaS idea, but its architecture maps almost perfectly onto Wirebot's Red-to-Black operating mode. Rather than building it as a separate product, **Wirebot IS the scoreboard** — the sovereign partner that keeps score.

---

## Core Concepts to Absorb

### 1. "Am I Winning Today?" (The One Question)

The scoreboard answers one question at a glance. Not "am I busy?" Not "did I work?" But: **did reality objectively change because I worked?**

This is the exact framing Wirebot needs for the morning standup and EOD review.

### 2. The Three-Plane Architecture

The session arrived at a critically important separation:

| Plane | What It Contains | Scored? |
|-------|-----------------|---------|
| **Reality** | Shipped artifacts, revenue events, deploys, published URLs | ✅ Yes |
| **Behavior** | Focus contracts, distraction breaches, time patterns | ⚠️ Deductions only |
| **Reflection** | Mood, energy, clarity, personal challenges | ❌ Never scored |

**Most products fail by mixing these planes. Wirebot must not.**

- Revenue is Reality. It scores.
- Skipping tasks is Behavior. It's visible but not punished unless a commitment was made.
- Feeling overwhelmed is Reflection. It's tracked for sustainability, never for scoring.

### 3. The 5 Scoreboard Zones

| Zone | What It Tracks | Wirebot Implementation |
|------|---------------|----------------------|
| **Game Clock** | Day/week/quarter/season progress, urgency | Timezone-aware, injected into every standup |
| **Score** | Shipped outcomes (only completed outputs count) | Checklist completions + verified external artifacts |
| **Possession** | ONE active initiative (focus control) | Active business in `wb focus` |
| **Momentum** | Streak days, trend arrows, energy | Ship streak, consecutive focus days |
| **Score Differential** | Planned vs. actual, target vs. reality | Break-even target vs. actual revenue |

### 4. External Verification (Anti-Gaming)

The session's most powerful insight: **only externally verifiable artifacts score.**

Strong signals:
- Git commits to main / releases / deploys
- Stripe payment events
- Published URLs (blog, docs, landing pages)
- App store submissions
- DNS changes, systemd services, infrastructure provisioned
- Emails sent (campaigns, not drafts)
- YouTube videos published
- Social posts with real content (AI-scored for business relevance)

Banned signals:
- Time spent
- Tasks "checked off" (self-reported)
- Focus sessions
- Self-reported progress
- Drafts

**Wirebot implication:** The checklist engine scores task completion, but true scoring needs integration with real systems. Phase 1 uses task completion. Phase 2 connects to Stripe, GitHub, email, social APIs.

### 5. Shipping Bias

> "Only completed outputs score. Progress is only counted when something leaves your local machine."

A "ship" is defined as:
- Code pushed / deployed
- Article published
- Feature demo recorded
- Proposal sent
- Revenue-generating action completed
- Infrastructure system activated

This maps directly to Wirebot's accountability cadence. The morning standup should ask: **"What will ship today?"** The EOD should verify: **"What shipped?"**

### 6. Seasons, Not Infinite Timelines

- 30/60/90-day "seasons" with score resets
- History preserved but counters restart
- Season themes (e.g., "Revenue Proof", "Audience First", "Debt Reduction")
- Prevents hoarding progress and creates urgency

**Wirebot:** The current operating mode (Red-to-Black) is essentially a season. Define it: "Season 1: Red-to-Black — 90 days — break even."

### 7. Focus Integrity (Possession)

> "You can only score while you have possession."

- One active initiative at a time
- Context switches are visible penalties
- Multi-business awareness doesn't mean multi-business simultaneous execution

**Wirebot:** `wb focus` already tracks active business. Add: context switch detection from CLI usage patterns.

### 8. The Asset Hierarchy (From Portfolio Activation)

The flywheel order that actually works:

```
1. Cash-Generating Asset (Foundation) — what pays bills NOW
2. Audience / Distribution Asset — what amplifies reach
3. Productized Knowledge Asset — what captures expertise
4. Scalable Software Asset — what scales without you
5. Network / Ecosystem Asset — what compounds with users
```

**Critical insight from the session:**
> "You are currently trying to operate at levels 4–5 without fully stabilizing levels 1–2."

**Wirebot:** Map each business to its asset tier. Advise the operator when they're trying to build level 5 while level 1 is shaky.

### 9. The Daily Questions (Needle-Moving)

**Morning Lock-In:**
1. If today only had one win, what must ship for it to count?
2. What is the smallest version of that I can finish today?
3. What am I allowed to ignore until this ships?

**EOD Score Lock:**
4. What objectively changed in the world today because I worked?
5. What can be verified without my explanation?

These replace generic standup prompts. They force clarity and shipping.

### 10. Social Sharing / Build-in-Public

Auto-generated, non-editable cards:
- Daily: "Shipped Today: X" + link to artifact
- Weekly: Ship count, revenue events, longest streak
- Monthly: Score trend, focus lane breakdown

**Wirebot:** Future dashboard feature. The scoreboard view IS the social card.

---

## What Wirebot Absorbs vs. What Remains a Separate Product

### Wirebot Absorbs (Now)

| Concept | How |
|---------|-----|
| "Am I winning today?" framing | Morning standup + EOD review |
| Three-plane separation | Reality (scored) vs. Behavior (tracked) vs. Reflection (private) |
| Shipping bias | Only completed+verified artifacts advance score |
| Focus possession | `wb focus` = one business at a time |
| Season model | Red-to-Black = Season 1, 90 days |
| Daily needle-moving questions | Replace generic standup prompts |
| Asset hierarchy | Map businesses to tier, warn when order is wrong |
| Execution Score (0-100) | Complement health score with execution score |
| Ship streak tracking | Consecutive days with a verified ship |
| Score differential | Break-even target vs. actual revenue |

### Remains Separate Product (Later / Maybe)

| Concept | Why Separate |
|---------|-------------|
| TV/LED stadium display | Physical product, different audience |
| League/competitive layer | Multi-user, needs scale |
| Local machine agent scanning | Desktop app, different deployment |
| Public social sharing cards | Marketing feature, not core ops |
| Generic SaaS for other founders | Product-market fit needs validation first |

### Key Principle

> **Wirebot is the first customer of the Business Performance Scoreboard.**

Build it for Verious first. If it changes his shipping behavior in 30 days, it becomes a product feature of the Startempire Wire Network.

---

## Implementation: Execution Score

Add to the existing Business Health Score:

```
Health Score (existing):    How healthy is this business overall?
Execution Score (new):      Am I winning TODAY?
```

### Execution Score Components (0-100)

| Lane | Weight | What Counts |
|------|--------|-------------|
| **Shipping** | 40% | Checklist tasks completed, verified artifacts |
| **Distribution** | 25% | Content published, outreach sent, social posts |
| **Revenue** | 20% | Stripe events, payments received, deals closed |
| **Systems** | 15% | Infrastructure deployed, automations created, SOPs written |

### Daily Score Calculation

```
morning: declare intent ("I will ship X today")
day: evidence accumulates (commits, completions, events)
evening: score locks

Ship today? +40 base
Distribution action? +25
Revenue event? +20
System improvement? +15
No ship? score capped at 30 (no matter what else happened)
Context switch penalty? -5 per switch after 2nd
```

### Streak Tracking

```json
{
  "current_ship_streak": 3,
  "best_ship_streak": 7,
  "last_ship_date": "2026-02-01",
  "last_ship_artifact": "Wirebot multi-business architecture deployed",
  "no_ship_days": 0,
  "season": {
    "name": "Red-to-Black",
    "start": "2026-02-01",
    "end": "2026-05-01",
    "days_elapsed": 1,
    "days_remaining": 89,
    "total_score": 65,
    "avg_daily_score": 65
  }
}
```

---

## See Also

- [MULTI_BUSINESS.md](./MULTI_BUSINESS.md) — Business health scoring
- [PAIRING.md](./PAIRING.md) — Financial reality mapping
- [SINGLEEYE_CONCEPTS.md](./SINGLEEYE_CONCEPTS.md) — Background processing for overnight analysis
- [VISION.md](./VISION.md) — Sovereign mode philosophy
- Source: `/data/wirebot/reference/business-performance-scoreboard.md`
- Source: `/data/wirebot/reference/asset-portfolio-activation.md`

## Lane Infrastructure

### Automated Sources (trusted, auto-approve)
| Source | Lane | Interval | Status |
|--------|------|----------|--------|
| Git Discovery | shipping | 5 min | ✅ Active |
| GitHub Webhooks | shipping | real-time | ✅ Active (7 repos) |
| Blog RSS (2 feeds) | distribution | 15 min | ✅ Active |
| Sendy Campaigns | distribution | 1 hour | ✅ Active (4 brands) |
| Stripe Webhook | revenue | real-time | ✅ Active (9 event types) |
| MemberPress Webhook | revenue | real-time | ✅ Active (mu-plugin) |
| WooCommerce Poller | revenue | 30 min | ✅ Active (no orders) |
| Cloudflare Poller | systems | daily | ✅ Active (50 zones) |
| Systems Health Check | systems | 6 hours | ✅ Active (4 services) |
| SSL/Disk/Integration Audit | systems | daily cron | ✅ Active |

### Pending Integration (need browser-side API key entry)
- UptimeRobot, PostHog, RescueTime, Discord, HubSpot

### Pending OAuth Apps
- GitHub, YouTube (need app creation on provider platforms)
