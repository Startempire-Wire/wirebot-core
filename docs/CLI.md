# Wirebot CLI Reference

Wirebot has two CLI entry points:

| Command | Path | Purpose |
|---------|------|---------|
| `wb` | `/usr/local/bin/wb` | High-level shortcut for all Wirebot tools |
| `wirebot` | `/usr/local/bin/wirebot` → `/data/wirebot/bin/wirebot` | OpenClaw wrapper with secret injection |

Both are in the system `PATH` and available to any shell session.

---

## `wb` — Wirebot Tool CLI

The primary interface. All commands hit the OpenClaw gateway HTTP API (`127.0.0.1:18789`).

### Quick Reference

```
PAIRING         wb pair | pair status | pair skip | pair reset
BUSINESSES      wb overview | businesses | focus | add-business
CHECKLIST       wb status | next | daily | complete | skip | add | list | detail | stage
MEMORY          wb recall | remember | state | cache | memory | sync
SYSTEM          wb health | services | logs | pillars
SCOREBOARD      wb score | streak | season | feed | ship | submit | pending | approve | reject
                wb intent | audit | lock | card | scoreboard/wins
DISCOVERY       wb discover scan|watch|status|backfill | projects | approve-project | reject-project
ADVANCED        wb raw
```

---

### Checklist Commands

The Business Setup Checklist Engine tracks 64+ tasks across three stages: **Idea → Launch → Growth**. Each task has a category, priority, dependencies, and completion status.

#### `wb status`

Show overall business setup progress.

```bash
$ wb status
📊 Business Setup — IDEA
Overall: 1/64 (2%)

IDEA:   █░░░░░░░░░░░░░░░░░░░ 5% (1/22)
LAUNCH: ░░░░░░░░░░░░░░░░░░░░ 0% (0/22)
GROWTH: ░░░░░░░░░░░░░░░░░░░░ 0% (0/20)
```

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb next`

Get the next recommended task based on current stage, priority, and dependencies.

```bash
$ wb next
→ Register Business Entity (Legal, high priority)
  Why: Required before opening business bank account, signing contracts, or filing taxes.
  Depends on: Create Mission Statement ✓
```

The engine considers:
- Current stage (idea/launch/growth)
- Task priority (critical > high > medium > low)
- Dependency chains (blocked tasks are skipped)
- Already completed/skipped tasks

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb daily`

Daily standup view: what's due, what's blocked, what's next.

```bash
$ wb daily
📋 Daily Standup — Sun Feb 1, 2026

Completed recently:
  ✓ Create Mission Statement (Identity)

Up next:
  → Register Business Entity (Legal, high)
  → Define Target Market (Strategy, high)
  → Set Up Business Bank Account (Finance, high)

Blocked:
  ⊘ Set Up Business Bank Account — waiting on: Register Business Entity
```

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb complete <task-id>`

Mark a task as done.

```bash
$ wb complete biz-entity
✓ Register Business Entity marked complete
  Next recommended: Define Target Market
```

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (significant action). The fact "completed task X at time Y" becomes searchable across all surfaces.

---

#### `wb skip <task-id>`

Skip a task. It won't appear in `next` or `daily` recommendations.

```bash
$ wb skip social-media
⊘ Set Up Social Media Accounts skipped
  Reason: Will not appear in next/daily. Use 'wb detail social-media' to unskip.
```

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (significant action).

---

#### `wb add <title> [flags]`

Add a custom task to the checklist.

```bash
$ wb add "Set up Stripe payments" --stage launch --category Finance --priority high
✓ Added: Set up Stripe payments (launch/Finance/high)
  ID: stripe-payments
```

**Flags:**

| Flag | Values | Default |
|------|--------|---------|
| `--stage` | `idea`, `launch`, `growth` | Current stage |
| `--category` | Any string (e.g., `Finance`, `Legal`, `Marketing`) | Uncategorized |
| `--priority` | `critical`, `high`, `medium`, `low` | `medium` |

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (significant action).

---

#### `wb list [stage]`

List all tasks, optionally filtered by stage.

```bash
$ wb list idea
IDEA STAGE (1/22 complete):

  ✓ Create Mission Statement (Identity, high)
  → Register Business Entity (Legal, high)
  → Define Target Market (Strategy, high)
  → Create Business Plan (Strategy, medium)
  ...
```

```bash
$ wb list          # All stages
$ wb list launch   # Launch tasks only
$ wb list growth   # Growth tasks only
```

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb detail <task-id>`

Show full details for a specific task.

```bash
$ wb detail biz-entity
📋 Register Business Entity

  Stage:      Idea
  Category:   Legal
  Priority:   High
  Status:     Pending
  Depends on: Create Mission Statement ✓
  Blocks:     Set Up Business Bank Account, File for EIN

  Description:
    Choose and register your business structure (LLC, S-Corp, sole proprietorship).
    Consider liability protection, tax implications, and future funding needs.
```

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb stage <stage>`

Set the current business stage. This changes which tasks appear in `next` and `daily`.

```bash
$ wb stage launch
✓ Stage set to LAUNCH
  Idea: 18/22 complete
  Launch: 0/22 — starting now
```

Valid stages: `idea`, `launch`, `growth`

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (significant action — stage transitions are milestones).

---

### Memory Commands

Wirebot's memory system has three layers that `wb recall` searches in parallel:

| Layer | System | What It Stores | Search Type |
|-------|--------|---------------|-------------|
| Facts | Mem0 (port 8200) | Conversation-extracted facts (80+) | Semantic vector (fastembed, 768 dims) |
| State | Letta blocks (port 8283) | Structured business state (4 blocks) | Direct read (always included) |
| Archival | Letta archival (port 8283) | Business docs (6 passages) | Semantic vector (OpenRouter, 1536 dims) |
| Cache | Go daemon (port 8201) | Hot mirror of Mem0+Letta | Substring match (<1ms) |

#### `wb recall <query>`

Search all memory layers in parallel. Returns Mem0 facts + Letta blocks + Letta archival docs.

```bash
$ wb recall "revenue goals"
Found 14 result(s) for "revenue goals":

[fact] Focuses on revenue generation and deal closing (75%)
[fact] Revenue score: 5/20 (73%)
[fact] Uses revenue-first sequencing approach (71%)

[state:human] Name: Verious Smith III ...
[state:goals] Active Goals: 1. Build Dashboard Frontend ...
[state:business_stage] Operating Mode: Red-to-Black ...
[state:kpis] Beta testers: 0 ...

[archival] [SCOREBOARD_PRODUCT] # Business Performance Scoreboard — Product Spec ...
[archival] [OPERATOR_REALITY] # Operator Reality — Verious Smith III ...
```

**Performance:** ~200ms (parallel Mem0 semantic + Letta blocks + Letta archival search)

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only — would create circular reference).

---

#### `wb remember <fact>`

Store a durable fact in long-term memory. The LLM extracts and indexes the fact asynchronously.

```bash
$ wb remember "Beta launch target is March 15, 2026"
Remembered: "Beta launch target is March 15, 2026"
```

The fact is:
1. Sent to Mem0 for LLM extraction and vector indexing
2. Available via `wb recall` within seconds
3. Synced to workspace files by the Go daemon on next poll cycle
4. Available across all Wirebot surfaces (dashboard, API, future channels)

**Performance:** ~1.9s (LLM extraction is async — CLI returns immediately, extraction happens in background)

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (that's the whole point).

---

#### `wb state [block]`

Read business state from Letta's structured memory blocks.

```bash
$ wb state                    # All blocks
$ wb state goals              # Just the goals block
$ wb state business_stage     # Current stage + decisions
$ wb state kpis               # Key performance indicators
$ wb state human              # Operator profile
```

**Available blocks:**

| Block | Contains |
|-------|----------|
| `business_stage` | Current stage, mode, milestones, key decisions, architecture |
| `goals` | Active goals with status, 12 Pillars reference |
| `kpis` | Key performance indicators, metrics |
| `human` | Operator profile: name, business, role, timezone, ecosystem |

**Memory:** Logged to `cli.jsonl`. Not stored to Mem0 (read-only).

---

#### `wb cache <query>`

Search the Go daemon hot cache directly. Sub-millisecond substring match across all cached facts and blocks.

```bash
$ wb cache "wirebot"
Cache results: 6 (age: 12ms)
  [mem0] Wirebot membership tiers: Free (Mode 0), FreeWire (Mode 1)...
  [mem0] Wirebot sovereign mode: Jarvis for solopreneurs...
  [letta:human] Name: Verious Smith III Business: Startempire Wire...
  [letta:business_stage] Stage: Idea Mode: Sovereign...
```

**Performance:** <1ms. This is the fastest way to search. Results come from the in-memory cache refreshed every 60 seconds by the Go daemon.

**Memory:** Not logged (diagnostic command).

---

#### `wb memory`

Full memory system summary — fact counts, block counts, archival passages, cache status, sync schedule.

```bash
$ wb memory
── Memory System Summary ──

  Mem0:          80 facts (fastembed, 768 dims)
  Letta blocks:  4 (2160 chars) [goals, human, kpis, business_stage]
  Letta archival: 6 passages (OpenRouter embeddings, 1536 dims)
  Hot cache:     80 facts + 4 blocks (60s refresh)
  Nightly sync:  cron 0 0 * * 0

  Layers: memory-core (files) → Mem0 (facts) → Letta (state+archival)
  Recall: wb recall <query> searches all 3 in parallel
```

**Memory:** Not logged (diagnostic command).

---

#### `wb sync`

Manually trigger memory sync (normally runs weekly via cron on Sunday midnight PT):
- Mem0 facts → append new facts to `MEMORY.md` (dedup)
- Letta blocks → snapshot to `BUSINESS_STATE.md`
- memory-core auto-re-indexes on file changes

```bash
$ wb sync
Running memory sync...
[memory-sync] Mem0: 80 total facts, 3 new appended to MEMORY.md
[memory-sync] Letta: 4 blocks snapshot → BUSINESS_STATE.md
[memory-sync] Sync complete. memory-core will auto-re-index on file changes.
Done.
```

**Memory:** Not logged (maintenance command).

---

### System Commands

#### `wb health`

Full health check across all Wirebot components.

```bash
$ wb health
── Gateway ──
  Status: running (port 18789)

── Memory Sync Daemon ──
  Status:       ok
  Cache facts:  80
  Cache blocks: 4
  Total syncs:  1582
  Uptime:       19h32m

── Mem0 ──
  Status:   running (port 8200)
  Memories: 80
  Embedder: fastembed/bge-base-en-v1.5

── Letta ──
  Status: running (port 8283)
  Blocks:   4 [goals, human, kpis, business_stage]
  Embedder: text-embedding-3-small (1536 dims)
  Archival: 6 passages

── Scoreboard ──
  Status:  running (port 8100)
  Score:   21/100
  Ships:   3 today

── Cloudflare Tunnel ──
  Status: active (helm.wirebot.chat)
```

**Memory:** Not logged (diagnostic command).

---

#### `wb services`

Show systemd service status for all Wirebot components.

```bash
$ wb services
SERVICE                        STATUS
-------                        ------
openclaw-gateway               active
mem0-wirebot                   active
wirebot-memory-syncd           active
wirebot-scoreboard             active
cloudflared-wirebot            active
letta-wirebot (podman)         letta-wirebot Up 43 hours
```

**Memory:** Not logged (diagnostic command).

---

#### `wb logs [n]`

Show the last `n` lines of the gateway log. Default: 50.

```bash
$ wb logs        # Last 50 lines
$ wb logs 200    # Last 200 lines
```

Log location: `/home/wirebot/logs/openclaw-gateway.log`

**Memory:** Not logged (diagnostic command).

---

#### `wb pillars`

Display the 12 Operating Pillars quick reference card.

```bash
$ wb pillars
⚡ The 12 Pillars — Wirebot Operating Philosophy

  TIER 1 — FOUNDATION (non-negotiable)
    1. Calm                      Composed under any conditions
    2. Rigor                     Every detail verified
    3. Radical Truth (Diplomatic) Say what needs saying, respectfully
  ...
```

**Memory:** Not logged (reference command).

---

### Pairing Commands

Wirebot's pairing protocol is a 22-question conversational onboarding. Until pairing is complete, Wirebot nudges before every command.

#### `wb pair`

Start or resume the pairing conversation. Shows instructions to open the scoreboard chat.

#### `wb pair status`

Show pairing progress: score, phase, questions answered/remaining.

```bash
$ wb pair status
⚡ Pairing Status
  Paired:    false
  Score:     0%
  Phase:     0 of 4
  Answered:  0 / 22 questions
```

#### `wb pair skip`

Dismiss the pairing nudge for 4 hours.

#### `wb pair reset`

Revoke pairing and clear all personalization (destructive — requires confirmation).

---

### Business Commands

Wirebot is operator-centric, not business-centric. These commands manage the multi-business portfolio.

#### `wb overview`

Operator-level view — all businesses with health scores and focus recommendation.

#### `wb businesses` (alias: `wb biz`)

List all tracked businesses with health scores.

#### `wb focus <name>`

Switch active business context. Changes which tasks appear in `wb status/next/daily`.

#### `wb add-business <name>`

Add a new business to the portfolio. Flags: `--stage`, `--priority`, `--domain`.

---

### Scoreboard Commands

The Business Performance Scoreboard lives at `wins.wirebot.chat`. These commands interact with it from the CLI.

#### `wb score [date]`

Today's execution score with lane breakdown.

```bash
$ wb score
⚡ EXECUTION SCORE: 21
  SHIPPING:     15/40
  DISTRIBUTION: 0/25
  REVENUE:      5/20
  SYSTEMS:      1/15
  Ships today:  3
  Intent:       🎯 Wire all fallback models + fix scoreboard issues
  Result:       ❌ LOSS
```

#### `wb streak`

Current and best ship streaks.

```bash
$ wb streak
🔥 STREAK: 2 days
  Best ever: 2 days
  Last ship: 2026-02-02
```

#### `wb season`

Season progress, record, and average score.

```bash
$ wb season
📅 SEASON 1: Red-to-Black
  Feb 01 → May 01 (Day 1 of 88)
  Record: 0W-2L
  Avg:    30/day
  Total:  61 pts
```

#### `wb feed [n]`

Activity feed — last n events (default 15).

#### `wb ship "<title>" [--lane <lane>] [--url <url>]`

Log a ship event. Adds points to the scoreboard.

```bash
$ wb ship "Memory architecture fixes complete" --lane systems
✅ Shipped: Memory architecture fixes complete (+5 systems)
```

#### `wb submit "<title>" [--lane <lane>] [--url <url>]`

Submit a gated event (pending operator approval before it scores).

#### `wb pending`

List events awaiting approval.

#### `wb approve <event-id>`

Approve a pending event (it now scores).

#### `wb reject <event-id>`

Reject a pending event (removed from scoring).

#### `wb intent "<text>"`

Declare today's shipping intent. Without an argument, shows current intent.

```bash
$ wb intent "Complete memory fixes and update docs"
🎯 Intent set: Complete memory fixes and update docs
```

#### `wb audit [lane]`

Full audit trail with score derivation. Optionally filter by lane.

#### `wb lock [date]`

Lock end-of-day score and check intent fulfillment.

#### `wb card daily|weekly|season`

Open a social share card (SVG) for the specified period.

#### `wb scoreboard` (alias: `wb wins`)

Quick scoreboard view — combined score + streak + season.

---

### Discovery Commands

The Git Discovery Engine tracks VPS repositories and auto-discovers commits for scoreboard scoring.

#### `wb discover scan`

Scan VPS for git repositories. Discovers new repos not yet tracked.

#### `wb discover watch`

Check tracked repos for new commits. Runs every 5 minutes via cron.

#### `wb discover status`

Show tracked repos and pending discovery events.

#### `wb discover backfill [n]`

Discover commits from the last n days (default 7).

#### `wb projects`

List all discovered projects with approval status.

```bash
$ wb projects
PROJECT                        STATUS     COMMITS
wirebot-core                   approved   58
Startempire-Wire-Network       pending    12
focusa                         pending    3
```

#### `wb approve-project <name>`

Approve a project — bulk-approves all pending commits and auto-approves future commits.

#### `wb reject-project <name>`

Reject a project — bulk-rejects pending commits and silently skips future commits.

---

### Advanced Commands

#### `wb raw <tool> '<json-args>'`

Invoke any registered gateway tool with raw JSON arguments. For power users and debugging.

```bash
$ wb raw wirebot_checklist '{"action":"list","stage":"idea"}'
$ wb raw wirebot_recall '{"query":"startup funding","layers":["mem0"]}'
$ wb raw wirebot_business_state '{"action":"update","block":"kpis","value":"MRR: $0"}'
$ wb raw wirebot_remember '{"fact":"Launched beta on March 15"}'
```

**Registered tools:**

| Tool | Description |
|------|-------------|
| `wirebot_recall` | Search all memory layers (Mem0 facts + Letta blocks + Letta archival) |
| `wirebot_remember` | Store fact to long-term memory (Mem0 LLM extraction) |
| `wirebot_business_state` | Read/update Letta business state blocks |
| `wirebot_checklist` | Business Setup Checklist Engine (64 tasks, 3 stages) |
| `wirebot_score` | Submit scoreboard events (gated, pending approval) |

**Memory:** Logged to `cli.jsonl` AND stored to Mem0 (significant action).

---

## `wirebot` — OpenClaw Wrapper

A thin wrapper around the `openclaw` CLI that sources secrets from `/run/wirebot/gateway.env` before delegating. Use this when you need direct access to OpenClaw features not exposed by `wb`.

```bash
$ wirebot gateway status
$ wirebot models list
$ wirebot plugins list
$ wirebot cron list
$ wirebot health
$ wirebot sessions list
$ wirebot --help          # Full OpenClaw help
```

### Key OpenClaw Commands via `wirebot`

```bash
# Gateway management
wirebot gateway status          # Gateway health
wirebot gateway run             # Start gateway (normally via systemd)

# Model configuration
wirebot models list             # Available models
wirebot models set <model>      # Change primary model

# Plugin management
wirebot plugins list            # Loaded plugins
wirebot plugins reload          # Reload plugins

# Cron / Accountability
wirebot cron list               # Scheduled jobs
wirebot cron run <job-id>       # Manually trigger a cron job

# Sessions
wirebot sessions list           # Conversation history

# Skills
wirebot skills list             # Available skills

# Browser (headless automation)
wirebot browser launch          # Start dedicated browser
wirebot browser status          # Browser status

# Channels (future: SMS, email, etc.)
wirebot channels list           # Configured channels
wirebot channels login          # Link a new channel
```

---

## Memory Architecture

Every `wb` CLI interaction is captured in two layers:

### Layer 1: Local JSONL Log (all commands)

**File:** `/home/wirebot/clawd/memory/cli.jsonl`

Every command is appended as a JSON line:

```json
{"ts": "2026-02-01T20:48:12Z", "cmd": "status", "detail": "checklist status", "result": "📊 Business Setup — IDEA\nOverall: 1/64 (2%)..."}
{"ts": "2026-02-01T20:48:12Z", "cmd": "recall", "detail": "membership", "result": "Found 4 result(s)..."}
{"ts": "2026-02-01T20:48:15Z", "cmd": "complete", "detail": "task:biz-entity", "result": "✓ Register Business Entity marked complete"}
```

- **Speed:** Instant (local file append)
- **Scope:** Every command, including read-only
- **Result truncation:** 500 characters max
- **Picked up by:** Go daemon file watcher → synced to hot cache
- **Searchable via:** `grep`, `jq`, Go daemon cache

### Layer 2: Mem0 Async (significant actions only)

State-changing commands are also stored to Mem0 for cross-surface recall:

| Command | Stored to Mem0? | Why |
|---------|-----------------|-----|
| `status`, `next`, `daily`, `list`, `detail` | ❌ | Read-only, no state change |
| `complete`, `skip`, `add` | ✅ | Task state changes are milestones |
| `recall`, `cache` | ❌ | Read-only |
| `remember` | ✅ | Explicit memory store |
| `stage` | ✅ | Stage transitions are milestones |
| `health`, `services`, `logs` | ❌ | Diagnostic |
| `raw` | ✅ | Could be anything, log it |

Mem0 storage is async (fires in background, ~1.9s) — CLI returns immediately.

### Querying CLI History

```bash
# Recent commands
tail -10 /home/wirebot/clawd/memory/cli.jsonl | jq -r '"\(.ts) [\(.cmd)] \(.detail)"'

# All completions
grep '"cmd":"complete"' /home/wirebot/clawd/memory/cli.jsonl | jq -r '"\(.ts) \(.detail)"'

# Commands today
grep "$(date -u +%Y-%m-%d)" /home/wirebot/clawd/memory/cli.jsonl | jq -r '"\(.ts) [\(.cmd)] \(.detail)"'

# Via Wirebot recall (searches Mem0)
wb recall "completed task"
```

---

## Infrastructure Map

```
┌─────────────────────────────────────────────────────────────┐
│                        wb CLI                               │
│                   /usr/local/bin/wb                          │
└──────────────────────┬──────────────┬───────────────────────┘
                       │              │
          /tools/invoke│              │ HTTP direct
                       ▼              ▼
┌──────────────────────────┐  ┌──────────────────────────────┐
│  OpenClaw Gateway (:18789)│  │  Scoreboard API (:8100)      │
│  systemd: openclaw-gateway│  │  systemd: wirebot-scoreboard │
│                          │  │  SQLite event store           │
│  5 Tools:                │  │  14+ endpoints                │
│   wirebot_recall ─┐      │  │  Chat proxy → Gateway         │
│   wirebot_remember│      │  │  Integration pollers (8)      │
│   wirebot_state  ─┤→ mem │  │  Tenant manager               │
│   wirebot_checklist│     │  └──────────────────────────────┘
│   wirebot_score ──┘      │
└──────────────────────────┘
           │
  ┌────────┼────────┬────────────┐
  ▼        ▼        ▼            ▼
┌────────┐┌──────┐┌──────────┐┌──────────────┐
│Go Syncd││ Mem0 ││  Letta   ││ memory-core  │
│ :8201  ││:8200 ││  :8283   ││ (embedded)   │
│hot     ││fast- ││4 blocks  ││ BM25+vector  │
│cache   ││embed ││6 archival││ workspace    │
│60s     ││Qdrant││PostgreSQL││ files        │
│refresh ││80+   ││agent LLM ││              │
└────────┘└──────┘└──────────┘└──────────────┘
     │
     ▼
┌──────────────────────────┐
│  /home/wirebot/clawd     │
│  ├── memory/cli.jsonl    │  ← wb CLI log
│  ├── memory/2026-*.md    │  ← daily logs
│  ├── IDENTITY.md         │  ← agent identity
│  ├── SOUL.md             │  ← 12 pillars
│  ├── MEMORY.md           │  ← synced Mem0 facts
│  ├── BUSINESS_STATE.md   │  ← synced Letta blocks
│  ├── pairing.json        │  ← pairing progress
│  └── checklist.json      │  ← task state
└──────────────────────────┘
```

---

## Configuration

| Setting | Location | Value |
|---------|----------|-------|
| Gateway URL | hardcoded in `wb` | `http://127.0.0.1:18789` |
| Gateway token | hardcoded in `wb` | `65b918ba-...` |
| Scoreboard URL | hardcoded in `wb` | `http://127.0.0.1:8100` |
| Sync daemon URL | hardcoded in `wb` | `http://127.0.0.1:8201` |
| Gateway config | `/data/wirebot/users/verious/openclaw.json` | Full OpenClaw config |
| Auth profiles | `.../agents/verious/agent/auth-profiles.json` | Kimi + ZAI + OpenRouter keys |
| Gateway secrets | `/run/wirebot/gateway.env` (tmpfs) | Injected by ExecStartPre |
| Scoreboard secrets | `/run/wirebot/scoreboard.env` (tmpfs) | Token, RL JWT secret |
| CLI memory log | `/home/wirebot/clawd/memory/cli.jsonl` | Append-only JSONL |
| Gateway log | `/home/wirebot/logs/openclaw-gateway.log` | Rotating log |
| Checklist data | `/home/wirebot/clawd/checklist.json` | 64 seed tasks |
| Pairing state | `/home/wirebot/clawd/pairing.json` | 22-question progress |
| Workspace | `/home/wirebot/clawd/` | Agent workspace root |
| Memory sync script | `/data/wirebot/bin/memory-sync.sh` | Weekly cron (Sun midnight PT) |
| Discovery script | `/data/wirebot/bin/wb-discover` | Git commit scanner |
| Scoreboard DB | `/data/wirebot/scoreboard/events.db` | SQLite event store |
| Tenant DBs | `/data/wirebot/scoreboard/tenants/{randID}/events.db` | Per-member DBs |

---

## Troubleshooting

### `wb` returns "Parse error" or empty output

Gateway is down. Check:
```bash
wb health                       # Quick check
systemctl status openclaw-gateway
wb logs 20                      # Recent gateway log
```

### `wb recall` returns no results

1. Check if Mem0 is running: `curl http://127.0.0.1:8200/health`
2. Check if Go cache is populated: `wb cache "test"`
3. Check sync daemon: `systemctl status wirebot-memory-syncd`

### `wb complete` says task not found

Task IDs are auto-generated slugs. Use `wb list` to see available IDs, or `wb detail <partial>` to search.

### CLI memory log not being written

Check permissions:
```bash
ls -la /home/wirebot/clawd/memory/cli.jsonl
# Should be writable. If root-owned, fix:
chown wirebot:wirebot /home/wirebot/clawd/memory/cli.jsonl
```

### Gateway takes 40+ seconds to start

Normal. The memory-core plugin indexes workspace files on startup. The `wb` CLI will return errors until the gateway is fully ready. Wait for `[gateway] listening on ws://127.0.0.1:18789` in logs.

### Mem0 facts not appearing in recall

Mem0 LLM extraction is async (~1.9s). Wait a few seconds after `wb remember`, then `wb recall`. If still missing, check Mem0 directly:
```bash
curl -sS http://127.0.0.1:8200/v1/search \
  -H "Content-Type: application/json" \
  -d '{"query": "your fact", "namespace": "wirebot_verious", "top_k": 3}'
```

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture overview
- [GATEWAY.md](./GATEWAY.md) — OpenClaw gateway configuration
- [MEMORY.md](./MEMORY.md) — Memory system deep dive
- [MEMORY_BRIDGE_STRATEGY.md](./MEMORY_BRIDGE_STRATEGY.md) — Three-system memory bridge
- [OPERATIONS.md](./OPERATIONS.md) — Service management, systemd units
- [MONITORING.md](./MONITORING.md) — Health checks, alerting
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Common issues and fixes
