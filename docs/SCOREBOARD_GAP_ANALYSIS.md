# Scoreboard Gap Analysis — Vision vs Implementation

> Deep dive comparing the original 8,092-line ChatGPT brainstorm + product spec 
> against what's actually built. Organized by severity.

---

## CRITICAL GAPS (Missing from Core Vision)

### 1. ❌ Real Account Verification — NO INTEGRATIONS CONNECTED

**Vision:** "If it can't be externally verified, it doesn't count." The entire system was designed around *evidence from real accounts* — GitHub repos, Stripe payments, YouTube uploads, blog RSS, email providers, CRM, social media, DNS changes, deploy platforms.

**Reality:** Zero integrations are actually connected. The GitHub and Stripe webhook endpoints exist but no external service is pointed at them. Every single event in the system is currently self-reported via `wb ship` or `wb complete`. This fundamentally contradicts the anti-gaming principle.

**What's needed:**
- [ ] GitHub webhook pointed at `wins.wirebot.chat/v1/webhooks/github`
- [ ] Stripe webhook pointed at `wins.wirebot.chat/v1/webhooks/stripe`
- [ ] RSS/sitemap poller for blog content (startempirewire.com, wirebot.chat)
- [ ] YouTube API poller for channel videos
- [ ] Connected account registry (OAuth or API key per service)
- [ ] Verification level tagging on every event (STRONG/MEDIUM/WEAK per the spec)
- [ ] Events from unverified sources auto-gated to pending
- [ ] Integration settings UI in the PWA

**Priority:** P0 — without this, the scoreboard is a self-reported task tracker, not a "reality engine"

---

### 2. ❌ Wirebot Deep Integration — SCOREBOARD IS DISCONNECTED FROM THE AI

**Vision:** "Wirebot IS the scoreboard — the sovereign partner that keeps score." The scoreboard should be Wirebot's primary lens for understanding the operator's state.

**Reality:** The scoreboard is a standalone Go binary + Svelte PWA. Wirebot (OpenClaw gateway) has no native awareness of it. The `wirebot_checklist` tool in the bridge plugin doesn't read the score. The morning standup and EOD review cron jobs include scoreboard curl commands in their prompts, but Wirebot can't natively query or reason about scoreboard state.

**What's needed:**
- [ ] `wirebot_score` tool in the memory bridge plugin (direct API call, returns structured score data)
- [ ] Scoreboard state injected into every Wirebot conversation context (like IDENTITY.md and MEMORY.md)
- [ ] Wirebot auto-pushes events when it verifies artifacts (deploys it made, files it published, systems it activated)
- [ ] Morning standup uses score context to shape the 3 daily questions
- [ ] EOD review auto-calls `/v1/lock` and reports the result
- [ ] Wirebot can approve/reject pending events on behalf of operator (with confirmation)

**Priority:** P0 — the original vision says Wirebot and the scoreboard are the same thing

---

### 3. ❌ Network/League/Competition Layer — NOT DESIGNED AT ALL

**Vision:** "Make this generic enough to allow for other indie devs or solo founders to connect into this daily and maybe eventually compete or compare progress."

**Reality:** Single-user only. No concept of multiple users, accounts, leagues, divisions, peer comparison, or anonymous benchmarking exists anywhere in the codebase.

**What's needed:**
- [ ] User/account model (user_id on events, daily_scores, seasons)
- [ ] User registration + auth (separate from admin token)
- [ ] Division system (Pre-Revenue, First $1, First $1k, First $10k, Sustaining)
- [ ] League model (cohorts of users with same rules, opt-in)
- [ ] Percentile scoring within divisions (not raw comparisons)
- [ ] Anonymous benchmark aggregation (global averages per division)
- [ ] Opt-in/opt-out per user for each sharing level

**Priority:** P1 — needed before any external user can use this

---

### 4. ❌ Granular Share/Privacy Settings — NOT IMPLEMENTED

**Vision:** "Public-optional, private-first." "Anonymous by default. Opt-in identity." The user decides what is shared, with whom, and how.

**Reality:** The entire API is either public (no auth, like `/v1/card/daily`) or admin-only (bearer token). No concept of share levels, privacy toggles, or selective visibility.

**What's needed:**
- [ ] Three-tier visibility model:
  - **Private** (default): Only operator + Wirebot see everything
  - **Shared**: Selected stats visible to linked peers/league
  - **Public**: Social cards, season record visible to anyone
- [ ] Per-metric share toggles (share streak but not revenue, share score but not feed details)
- [ ] Anonymization layer for global benchmark contribution
- [ ] Social card generation respects privacy settings (only shows opted-in metrics)
- [ ] Settings UI in PWA for controlling all share preferences
- [ ] API respects visibility level per request context

**Priority:** P1 — required before network/league features

---

### 5. ❌ Score Differential / Projection View — NOT BUILT

**Vision:** "Planned vs. actual, target vs. reality" and "Projection view allows hypothetical modeling and adjustments of each parameter to attempt to project what the outcomes would be."

**Reality:** No target-setting, no planned-vs-actual comparison, no projection engine, no "What-If Lab."

**What's needed:**
- [ ] Revenue target per season (break-even amount, goal MRR)
- [ ] Planned ships per week (declared at season start)
- [ ] Actual vs planned comparison on Season view
- [ ] Score differential display ("behind by 3 ships this week")
- [ ] Projection engine (rules-based first): "if you ship 2x/week, projected score = X"
- [ ] What-If sliders in the UI (hypothetical lane adjustments)

**Priority:** P2 — valuable but not blocking core usage

---

## SIGNIFICANT GAPS

### 6. ⚠️ AI Content Scoring — NOT IMPLEMENTED

**Vision:** "Monitor all social media specifically for business type posts... This should be scored per post by AI." Social posts get AI-scored for business relevance, expertise demonstration, originality, depth.

**Reality:** Social posts are just another event type pushed manually. No AI scoring of content quality.

**What's needed:**
- [ ] Social platform polling (LinkedIn, X at minimum)
- [ ] AI classification pipeline: each detected post scored on relevance, expertise, originality, artifact linkage
- [ ] Quality score multiplier on event points (not just "1 post = 4 points")
- [ ] Banned patterns: motivation fluff, engagement bait, retweets without commentary

**Priority:** P2

---

### 7. ⚠️ Focus Contracts & Distraction Tracking — NOT IMPLEMENTED

**Vision:** Elaborate system of FocusContracts with allowed/blocked categories, grace periods, distraction detection via RescueTime, focus integrity scoring, buyback multipliers.

**Reality:** Only context-switch counting from `wb focus` changes. No focus contracts, no distraction detection, no RescueTime integration, no focus integrity metric.

**What's needed:**
- [ ] Focus contract model (start/end, allowed categories, project link)
- [ ] `wb focus-start` / `wb focus-end` commands
- [ ] Integration with time tracking (RescueTime, Toggl, or ActivityWatch)
- [ ] Focus integrity metric (clean blocks / total blocks)
- [ ] Distraction breach detection (only during active focus contract)
- [ ] Focus-based multipliers on same-day shipping events

**Priority:** P2 — powerful but complex; original spec says "V1 basic, V2 full"

---

### 8. ⚠️ Reflection/Subjective Telemetry — NOT IMPLEMENTED

**Vision:** Mood, energy, stress, confidence sliders (1-5). Correlation analytics. Personal challenges. Gamified cosmetic rewards. All explicitly sandboxed from scoring.

**Reality:** No reflection system exists. The three-plane architecture is enforced only by documentation, not by code.

**What's needed:**
- [ ] `ReflectionEntry` model (mood, energy, stress, confidence — 1-5 scale)
- [ ] `wb reflect` CLI command (quick 30-second entry)
- [ ] Reflection view in PWA (emoji/slider input)
- [ ] Correlation engine (score vs mood over time, shipping velocity vs energy)
- [ ] Insight unlocks (cosmetic: "Your Optimal Focus Length", "Best Shipping Time")
- [ ] Personal challenges (opt-in, non-scoring)

**Priority:** P3 — important for long-term retention and self-knowledge, not blocking

---

### 9. ⚠️ Audit Ledger / Spreadsheet View — NOT IMPLEMENTED

**Vision:** "Robust audit log + geekified spreadsheet with read-only view." Immutable event log with forensic replay. Read-only spreadsheet projection. Evidence chain.

**Reality:** GET `/v1/audit` returns events as JSON (or CSV), but no spreadsheet-style UI exists. No evidence chain, no integrity hashing, no replay engine.

**What's needed:**
- [ ] Spreadsheet view in PWA (table with sortable columns, filters)
- [ ] Evidence object model (URL, screenshot, webhook receipt, API response hash)
- [ ] Integrity chain (event_id → prev_hash → this_hash for tamper evidence)
- [ ] Audit replay: recompute any day's score from events alone
- [ ] CSV/JSON export from UI (already have API endpoint)

**Priority:** P2 for spreadsheet view, P3 for integrity chain

---

### 10. ⚠️ Local Agent / Machine Scanning — NOT DESIGNED

**Vision:** "A local-first + cloud-verified evidence engine" with a local daemon that scans git repos, project folders, build artifacts, local servers/ports.

**Reality:** All detection is push-based (webhooks, CLI commands). No local agent exists.

**What's needed:**
- [ ] Local agent spec (daemon or cron that watches for shipping signals)
- [ ] Git repo scanner (detect commits to main, tags, releases in configured repos)
- [ ] Service detector (systemd services started, Docker containers running)
- [ ] Port/URL reachability checker (is the thing you deployed actually up?)
- [ ] Agent pushes verified events to scoreboard API

**Priority:** P3 — powerful vision but needs desktop/laptop deployment

---

### 11. ⚠️ Buyback Time System — NOT IMPLEMENTED

**Vision:** Tracks "bought-back time" from automations, delegation, meeting removal, SOPs. Buyback multipliers amplify scores. This is a major differentiator.

**Reality:** Systems lane tracks `AUTOMATION_DEPLOYED`, `DELEGATION_COMPLETED` etc. as flat events. No buyback tracking, no multiplier system, no "removed meetings" detection.

**What's needed:**
- [ ] Buyback artifact model (automation installed, meeting removed, SOP created)
- [ ] Cumulative buyback score (total hours saved)
- [ ] Buyback multiplier on future shipping events
- [ ] `wb buyback` CLI command
- [ ] Visualization: "Hours bought back this season"

**Priority:** P2

---

### 12. ⚠️ Multi-LLM Event Protocol — PARTIALLY DONE

**Vision:** "Allowable for other local LLMs doing work for a user."

**Reality:** `/root/.agent-kb/SCOREBOARD.md` tells agents about the API, and agents can push gated events. But there's no standardized protocol, no agent registration, no per-agent rate limits, no agent-specific dashboards.

**What's needed:**
- [ ] Agent registration model (agent name, capabilities, trust level)
- [ ] Per-agent event quotas / rate limits
- [ ] Agent dashboard: "Claude pushed 3 events today, 2 approved"
- [ ] Standardized event envelope with agent metadata
- [ ] Agent trust tiers (same as Wirebot trust modes)
- [ ] Agent-to-agent event visibility (can Claude see what Pi shipped?)

**Priority:** P1 for network readiness

---

## MINOR GAPS

### 13. Midday Reality Check prompt (from spec) — not in crons
### 14. Weekly Coach Review questions (5 questions) — partially in weekly cron
### 15. TV/Stadium Mode — API supports it, no dedicated TV-optimized UI built
### 16. Audible "ship confirmed" signal — not implemented
### 17. Decay functions on events — not implemented (all events score flat)
### 18. Multipliers from confidence scores — partially (confidence scales delta but no multi-factor multipliers)
### 19. Absence simulation ("what if the founder stops for 3 days") — not built
### 20. Event supersession chain ("this event replaces that one") — not built

---

## SUMMARY TABLE

| # | Gap | Severity | Status |
|---|-----|----------|--------|
| 1 | Real account verification / integrations | **CRITICAL** | Zero connected |
| 2 | Wirebot deep integration | **CRITICAL** | Standalone, disconnected |
| 3 | Network/league/competition | **CRITICAL** | Not designed |
| 4 | Granular share/privacy settings | **CRITICAL** | Not implemented |
| 5 | Score differential / projections | Significant | Not built |
| 6 | AI content scoring | Significant | Not built |
| 7 | Focus contracts / distraction | Significant | Minimal |
| 8 | Reflection / subjective telemetry | Significant | Not built |
| 9 | Audit ledger / spreadsheet UI | Significant | API only |
| 10 | Local agent / machine scanning | Significant | Not designed |
| 11 | Buyback time system | Significant | Not built |
| 12 | Multi-LLM event protocol | Significant | Partial |
| 13-20 | Minor items | Low | Various |

---

## RECOMMENDED BUILD ORDER

**Phase 3 (Now → 2 weeks): Make It Real**
1. Wire GitHub + Stripe webhooks (existing endpoints, just point external services)
2. Build `wirebot_score` tool into memory bridge plugin
3. Inject score context into every Wirebot conversation
4. Add verification level enforcement (STRONG/MEDIUM/WEAK on events)
5. Blog/YouTube RSS poller (cron that checks + auto-pushes verified events)

**Phase 4 (2-4 weeks): Network Foundation**
6. User/account model (multi-tenant events table)
7. User registration + API key generation
8. Division assignment (based on revenue state)
9. Privacy/share settings model + UI
10. Anonymous benchmark aggregation

**Phase 5 (4-8 weeks): Intelligence Layer**
11. AI content scoring pipeline
12. Reflection system + correlation engine
13. Projection engine (rules-based)
14. Spreadsheet/ledger view in PWA
15. Focus contracts (basic)

**Phase 6 (8-12 weeks): Competition & Scale**
16. League model + cohort matching
17. Percentile scoring within divisions
18. Public leaderboards (opt-in)
19. Buyback time tracking
20. Local agent (desktop daemon)

---

## See Also

- Original vision: `/data/wirebot/reference/business-performance-scoreboard.md` (8,092 lines, 57 messages)
- Product spec: `/home/wirebot/wirebot-core/docs/SCOREBOARD_PRODUCT.md`
- Integration concepts: `/home/wirebot/wirebot-core/docs/SCOREBOARD.md`
- CLI reference: `/home/wirebot/wirebot-core/docs/SCOREBOARD_CLI.md`
- Agent guide: `/root/.agent-kb/SCOREBOARD.md`
