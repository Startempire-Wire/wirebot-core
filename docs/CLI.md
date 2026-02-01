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
CHECKLIST       wb status | next | daily | complete | skip | add | list | detail | stage
MEMORY          wb recall | remember | state | cache
SYSTEM          wb health | services | logs | pillars
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

Wirebot's memory system has three layers: **Mem0** (conversation facts, vector search), **Letta** (structured business state blocks), and **Go daemon hot cache** (sub-ms substring search across both).

#### `wb recall <query>`

Search all memory layers. Uses the Go daemon cache first (<1ms), falls back to direct Mem0 + Letta queries if cache misses.

```bash
$ wb recall "membership tiers"
Found 4 result(s) for "membership tiers":

[fact] Wirebot membership tiers: Free (Mode 0), FreeWire (Mode 1), Wire (Mode 2), ExtraWire (Mode 3)

[state:human] Name: Verious Smith III
Business: Startempire Wire — membership-based entrepreneurial network
...
```

**Performance:** ~3ms (cache hit), ~80ms (cache miss, parallel Mem0+Letta)

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

### System Commands

#### `wb health`

Full health check across all Wirebot components.

```bash
$ wb health
── Gateway ──
  Status: running (port 18789)

── Memory Sync Daemon ──
  Status:       ok
  Cache facts:  12
  Cache blocks: 4
  Total syncs:  847
  Uptime:       19h32m

── Mem0 ──
  Status:   running (port 8200)
  Memories: 12
  Embedder: BAAI/bge-base-en-v1.5

── Letta ──
  Status: running (port 8283)

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
cloudflared-wirebot            active
letta-wirebot (podman)         letta-wirebot Up 19 hours
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
| `wirebot_checklist` | Business Setup Checklist Engine |
| `wirebot_recall` | Cascading memory search |
| `wirebot_remember` | Store fact to long-term memory |
| `wirebot_business_state` | Read/update Letta business state blocks |

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
└──────────────────────┬──────────────────────────────────────┘
                       │ HTTP POST /tools/invoke
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              OpenClaw Gateway (:18789)                       │
│         systemd: openclaw-gateway.service                   │
│                                                             │
│  Tools:                                                     │
│    wirebot_recall ──→ Go cache (:8201) → Mem0 → Letta     │
│    wirebot_remember ──→ Mem0 (:8200)                       │
│    wirebot_business_state ──→ Letta (:8283)                │
│    wirebot_checklist ──→ local checklist.json               │
└─────────────────────────────────────────────────────────────┘
                       │
        ┌──────────────┼──────────────┐
        ▼              ▼              ▼
  ┌──────────┐  ┌──────────┐  ┌──────────────┐
  │ Go Syncd │  │  Mem0    │  │    Letta     │
  │  :8201   │  │  :8200   │  │    :8283     │
  │ hot cache│  │ fastembed│  │  PostgreSQL  │
  │ file watch│ │ Qdrant   │  │  4 blocks    │
  └──────────┘  └──────────┘  └──────────────┘
        │
        ▼
  ┌──────────────────────┐
  │  /home/wirebot/clawd │
  │  ├── memory/         │
  │  │   ├── cli.jsonl   │  ← wb CLI log
  │  │   └── 2026-*.md   │  ← daily logs
  │  ├── IDENTITY.md     │
  │  ├── SOUL.md         │
  │  ├── checklist.json  │
  │  └── ...             │
  └──────────────────────┘
```

---

## Configuration

| Setting | Location | Value |
|---------|----------|-------|
| Gateway URL | hardcoded in `wb` | `http://127.0.0.1:18789` |
| Gateway token | hardcoded in `wb` | `65b918ba-...` |
| Sync daemon URL | hardcoded in `wb` | `http://127.0.0.1:8201` |
| Gateway config | `/data/wirebot/users/verious/openclaw.json` | Full OpenClaw config |
| Auth profiles | `.../agents/verious/agent/auth-profiles.json` | OpenRouter API key |
| CLI memory log | `/home/wirebot/clawd/memory/cli.jsonl` | Append-only JSONL |
| Gateway log | `/home/wirebot/logs/openclaw-gateway.log` | Rotating log |
| Checklist data | `/home/wirebot/clawd/checklist.json` | 64 seed tasks |
| Workspace | `/home/wirebot/clawd/` | Agent workspace root |

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
