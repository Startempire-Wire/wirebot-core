# Letta Autonomous Business State Agent — Architecture

> **Status:** Design doc — not yet implemented
> **Goal:** Transform Letta from a passive block store into an autonomous business state agent ("Jarvis")

---

## Current State (Passive)

```
                    ┌──────────────┐
  Discord msg       │              │
  ────────────────→ │   OpenClaw   │ ──→ response
                    │   Gateway    │
                    └──────┬───────┘
                           │ agent_end hook
                           ▼
               ┌───────────────────────┐
               │  Memory Bridge Plugin │
               │  (wirebot-memory-     │
               │   bridge/index.ts)    │
               └──┬────────┬────────┬──┘
                  │        │        │
           keyword│   store│   route│ if biz keywords
           match  │   fact │   msg  │
                  ▼        ▼        ▼
              ┌──────┐ ┌──────┐ ┌──────┐
              │Queue │ │ Mem0 │ │Letta │
              │(SQL) │ │:8200 │ │:8283 │
              └──┬───┘ └──────┘ └──┬───┘
                 │                  │
           human │            sync  │ every 30s
           review│            daemon│
                 ▼                  ▼
            MEMORY.md        BUSINESS_STATE.md
            (workspace)       (workspace)
```

**Problems with passive mode:**
1. Letta only updates when OpenClaw explicitly sends it a message
2. Keyword detection is brittle — misses subtle business changes
3. 1166+ pending memories sit unprocessed — Letta never sees approved ones
4. No feedback loop — Letta can't alert, can't ask questions, can't act
5. Blocks go stale between conversations (goals, KPIs outdated)

---

## Target State (Autonomous)

```
                    ┌──────────────┐
  Discord msg       │              │
  ────────────────→ │   OpenClaw   │ ──→ response
                    │   Gateway    │
                    └──────┬───────┘
                           │ agent_end hook
                           ▼
               ┌───────────────────────┐
               │  Memory Bridge Plugin │ (unchanged — still does recall/remember/state)
               └──┬────────┬────────┬──┘
                  │        │        │
                  ▼        ▼        ▼
              ┌──────┐ ┌──────┐ ┌──────────────────────────┐
              │Queue │ │ Mem0 │ │     Letta Agent          │
              │(SQL) │ │:8200 │ │     :8283                │
              └──┬───┘ └──────┘ │                          │
                 │              │  ┌─────────────────────┐ │
           human │              │  │ Event Intake Loop    │ │
           review│              │  │ (30s poll or webhook)│ │
                 │              │  └──────────┬──────────┘ │
                 │              │             │             │
                 │   ┌──────── │ ──┐    think│about it     │
                 │   │ Letta   │   │         │             │
                 │   │ Tools:  │   │         ▼             │
                 │   │         │   │  ┌────────────────┐   │
                 │   │ • read_ │   │  │ Self-edit      │   │
                 │   │   score │   │  │ blocks via     │   │
                 │   │ board   │   │  │ memory_replace │   │
                 │   │         │   │  └────────────────┘   │
                 │   │ • read_ │   │         │             │
                 │   │   queue │   │         ▼             │
                 │   │         │   │  ┌────────────────┐   │
                 │   │ • push_ │   │  │ Generate       │   │
                 │   │   alert │   │  │ alerts /       │   │
                 │   │         │   │  │ insights       │   │
                 │   │ • store │   │  └────────────────┘   │
                 │   │   archv │   │         │             │
                 │   │         │   │         ▼             │
                 │   │ • write │   │  BUSINESS_STATE.md    │
                 │   │   back  │   │  (auto-updated)       │
                 │   └─────────┘   │                       │
                 │                 └───────────────────────┘
                 ▼                          │
              Approved                      │ alerts
              memories ────────────────────→│ (pushed to
              auto-ingested                 │  dashboard)
                                            ▼
                                    ┌──────────────┐
                                    │ WINS Portal  │
                                    │ Alert Strip  │
                                    └──────────────┘
```

---

## Architecture Components

### 1. Event Feeder Service (`letta-feeder`)

A lightweight Go service (or goroutine in scoreboard) that polls for new events and feeds them to Letta as async messages.

**Event sources:**

| Source | Trigger | What Letta sees |
|--------|---------|-----------------|
| Approved memories | `memory_queue.status` changes to `approved` | "New approved fact: {text}. Update relevant blocks." |
| Scoreboard events | `/v1/events` new entries | "Event: shipped X at 2pm. Score: 72." |
| Discord conversations | `agent_end` hook (already exists) | Business-relevant snippets (already working) |
| Daily cron | 6am Pacific | "Morning brief: generate daily state summary." |
| KPI changes | Stripe webhook, integration events | "MRR changed: $X → $Y. Reason: {event}." |

**Implementation:**
```go
// In scoreboard main.go or standalone letta-feeder.go
func (s *Server) lettaFeederLoop() {
    ticker := time.NewTicker(30 * time.Second)
    for range ticker.C {
        // 1. Check for newly approved memories since last check
        rows := s.db.Query(`
            SELECT id, memory_text FROM memory_queue 
            WHERE status='approved' AND letta_synced=false
            ORDER BY created_at LIMIT 5
        `)
        for rows.Next() {
            // Send to Letta as async message
            lettaSendMessageAsync(agentID, formatApprovedMemory(text))
            // Mark synced
            s.db.Exec(`UPDATE memory_queue SET letta_synced=true WHERE id=?`, id)
        }
        
        // 2. Check for recent scoreboard events
        // 3. Generate daily brief at 6am
    }
}
```

**Key design choice:** Use `messages.createAsync()` (SDK) so the feeder doesn't block waiting for Letta to think. Letta processes in background, self-edits blocks, and results appear on next sync cycle.

### 2. Letta Agent Tools (registered via SDK)

The agent currently has 3 tools: `conversation_search`, `memory_insert`, `memory_replace`. These are self-referential (Letta editing its own memory). To become Jarvis, it needs tools to **read external state** and **push outputs**.

**New tools to register:**

| Tool | Direction | Purpose |
|------|-----------|---------|
| `read_scoreboard` | Letta → Scoreboard | Read current score, streak, lanes, recent events |
| `read_memory_queue` | Letta → Queue | See pending/approved counts, recent approved facts |
| `read_integrations` | Letta → Scoreboard | Check connected accounts, last activity times |
| `push_alert` | Letta → Alerts table | Create an alert for the WINS dashboard |
| `push_archival` | Letta → Letta archival | Store processed insights in long-term archival |
| `read_checklist` | Letta → Checklist JSON | See business tasks, completion rates, blocked items |

**Registration via SDK:**
```typescript
import Letta from '@letta-ai/letta-client';

const client = new Letta({ baseUrl: 'http://127.0.0.1:8283' });

// Register tool that Letta can call
await client.tools.create({
    name: 'read_scoreboard',
    description: 'Read current business score, streak, and recent events from the WINS scoreboard.',
    source_code: `
def read_scoreboard() -> str:
    """Read current scoreboard state."""
    import requests
    resp = requests.get('http://127.0.0.1:8100/v1/scoreboard', 
                       headers={'Authorization': 'Bearer ${SCOREBOARD_TOKEN}'})
    data = resp.json()
    return f"Score: {data['score']}/100, Streak: {data['streak']}, Last ship: {data['last_ship']}"
    `,
    source_type: 'python',
});

// Attach tool to agent
await client.agents.tools.attach(agentId, toolId);
```

**Note:** Letta tools execute Python inside the container's sandbox. The tool functions make HTTP calls back to the scoreboard from inside the Letta container. Since we use `--network host`, `127.0.0.1:8100` is reachable.

### 3. Alerts System

New `alerts` table in scoreboard SQLite:

```sql
CREATE TABLE alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL DEFAULT 'letta',      -- letta, system, cron
    severity TEXT NOT NULL DEFAULT 'info',       -- info, warning, critical
    title TEXT NOT NULL,
    body TEXT,
    acknowledged BOOLEAN DEFAULT false,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at DATETIME
);
```

**Letta generates alerts via `push_alert` tool:**
- "⚠️ MRR dropped 12% this week — 2 cancellations detected"
- "🎯 Goal 'Onboard 25 members' is 3 days from deadline with 0 signups"
- "✅ All 3 memory systems operational, 47 new facts processed today"
- "💡 Shipping streak at 5 days — longest this season"

**WINS dashboard** shows alerts in a new strip above the score card (or in the existing system health panel for admin).

### 4. Memory Bridge Changes

The existing plugin stays mostly unchanged. Additions:

**a) Approved memory auto-ingest (new hook in feeder, not plugin)**
Currently approved memories go to Mem0 only. The feeder routes them to Letta too:
```
Approved memory → Mem0 store (existing)
                → Letta async message (new) → Letta self-edits blocks
```

**b) Letta archival as recall layer (upgrade wirebot_recall)**
Currently `wirebot_recall` searches Letta blocks + archival. With the autonomous agent generating insights, archival becomes richer:
```
wirebot_recall("revenue trend")
  → Mem0 facts: "MRR is $X" 
  → Letta blocks: kpis, goals
  → Letta archival: "Weekly analysis: MRR declining 3% WoW due to..."  ← NEW depth
```

**c) No changes to wirebot_remember, wirebot_business_state, wirebot_checklist**
These tools work as-is. The autonomous agent enhances them by keeping blocks fresher.

### 5. Agent System Prompt (Updated)

```markdown
You are the Business State Engine for Wirebot — the AI operating partner 
for Verious Smith III.

You are AUTONOMOUS. You receive events, approved memories, and daily briefs 
without being asked. Your job:

1. **Maintain state** — Keep your 4 memory blocks accurate and current:
   - human: Owner context, preferences, working style
   - goals: Active goals with dates, progress, blockers
   - kpis: Metrics with current values and trends
   - business_stage: Score, stage, pairing profile, active contexts

2. **Detect patterns** — When you notice:
   - KPI trends (up or down)
   - Goal deadlines approaching
   - Inconsistencies between stated goals and actual activity
   - Opportunities from new facts
   → Use push_alert to surface insights

3. **Process approved facts** — When new facts arrive:
   - Update relevant blocks (memory_replace)
   - Archive processed insights (push_archival)  
   - Cross-reference with goals and KPIs

4. **Generate daily brief** — On morning trigger:
   - Summarize overnight changes
   - Flag items needing attention
   - Update KPIs from integrations

Rules:
- NEVER fabricate data. Use read_scoreboard and read_integrations for real numbers.
- Keep blocks concise (<3000 chars each). Archive verbose analysis.
- Alerts should be actionable, not noise. Max 3 per day unless critical.
- You serve ONE operator (Verious). All context is their business context.
```

---

## Data Flow: End-to-End Example

**Scenario:** User ships a feature via GitHub, discusses it on Discord.

```
1. GitHub webhook → Scoreboard event (source: github, type: commit)
2. Discord conversation → OpenClaw gateway → response
3. agent_end hook fires:
   a. Memory bridge → Mem0 store("User shipped auth module for STA")
   b. Memory bridge → Scoreboard extraction queue
   c. Memory bridge → Letta message (keyword: "shipped") 
      → Letta thinks → updates goals block, increments KPI
4. Feeder loop (30s later):
   a. Picks up approved memory from queue
   b. Sends to Letta: "Approved fact: Auth module shipped for STA"
   c. Letta cross-references → "Goal #2 partially complete"
   d. Letta calls push_alert: "🎯 Goal 'Finalize MVP' — auth module done, 3 features remaining"
5. Memory sync daemon (30s):
   a. Detects block changes → writes BUSINESS_STATE.md
6. User opens WINS:
   a. Dashboard shows alert: "🎯 Goal progress update"
   b. Score reflects the ship event
   c. System health shows all 5 services green
```

---

## Implementation Plan

### Phase 1: Foundation (est. 2-3 hours)
1. Add `letta_synced` column to `memory_queue` table
2. Add `alerts` table to scoreboard SQLite
3. Build feeder loop in scoreboard (goroutine, 30s poll)
4. Add `GET /v1/alerts` + `POST /v1/alerts` endpoints
5. Wire alerts into WINS dashboard (new component or extend SystemStatus)

### Phase 2: Agent Tools (est. 2-3 hours)
1. Write Python tool functions: `read_scoreboard`, `read_memory_queue`, `push_alert`
2. Register tools on Letta agent via SDK or direct API
3. Update agent system prompt for autonomous behavior
4. Test: send a fact → verify Letta processes it → check alert appears

### Phase 3: Event Sources (est. 1-2 hours)
1. Feeder ingests approved memories → Letta
2. Feeder ingests scoreboard events → Letta
3. Daily cron trigger (6am) → Letta morning brief
4. Integration events (Stripe, GitHub) → Letta

### Phase 4: Polish (est. 1-2 hours)
1. Rate limiting (max N Letta messages per hour to avoid runaway)
2. Alert dedup (don't repeat same insight)
3. Dashboard alert strip with acknowledge/dismiss
4. Archival search improvements in `wirebot_recall`

---

## SDK Usage

**Required package:** `@letta-ai/letta-client` (already installed as dep of `@letta-ai/letta-code`)

**Key SDK calls:**

```typescript
import Letta from '@letta-ai/letta-client';

const letta = new Letta({ baseUrl: 'http://127.0.0.1:8283' });
const AGENT = 'agent-82610d14-ec65-4d10-9ec2-8c479848cea9';

// Async message (non-blocking — Letta processes in background)
const run = await letta.agents.messages.createAsync(AGENT, {
  messages: [{ role: 'user', content: 'New approved fact: ...' }]
});
// run.id can be polled via letta.runs.retrieve(run.id)

// Read blocks (typed)
const blocks = await letta.agents.blocks.list(AGENT);

// Register tool
const tool = await letta.tools.create({
  name: 'push_alert',
  source_code: '...',
  source_type: 'python'
});
await letta.agents.tools.attach(tool.id, { agent_id: AGENT });

// Search archival
const passages = await letta.agents.passages.search(AGENT, {
  query: 'revenue trend'
});

// Compact conversation (prevent context overflow)
await letta.agents.messages.compact(AGENT, {});
```

**Where SDK lives vs raw HTTP:**

| Component | Current | With SDK |
|-----------|---------|----------|
| Memory bridge plugin (TypeScript) | Raw fetch ✅ | Replace with SDK ✅ |
| Memory sync daemon (Go) | Raw HTTP | Keep Go — no SDK needed |
| Scoreboard feeder (Go) | New | Raw HTTP from Go (simpler) |
| Tool registration | N/A | One-time SDK script |
| WINS dashboard | Via scoreboard API | No change |

The SDK is most valuable in the TypeScript plugin (type safety, streaming, async) and for one-time tool registration. The Go services stay with raw HTTP — simpler, no Node dependency.

---

## Cost Impact

| Component | Model | Cost |
|-----------|-------|------|
| Letta LLM (autonomous thinking) | kimi-coding/k2p5 via gateway | $0 |
| Letta embeddings | letta/letta-free (local) | $0 |
| Feeder → Letta messages | ~50-100/day | $0 (kimi free tier) |
| Tool execution (Python in container) | N/A | $0 |
| **Total** | | **$0/month** |

**Rate budget:** kimi free tier allows ~1000 requests/day. Autonomous agent at 100 messages/day = 10% of budget. Plenty of headroom.

---

## Risk Mitigations

| Risk | Mitigation |
|------|------------|
| Letta generates noise alerts | Max 3 alerts/day unless severity=critical. Acknowledge dismisses. |
| Runaway message loop | Rate limiter: max 10 Letta messages per 5-minute window |
| kimi model hallucinates KPIs | Tools read real data from scoreboard. Prompt: "NEVER fabricate." |
| Context window overflow | Monthly `messages.compact()` cron. Archival for long-term storage. |
| Letta container crash | Feeder retries with backoff. Events queue in SQLite, nothing lost. |
| Block corruption | memory-syncd snapshots BUSINESS_STATE.md every 30s. Git history is backup. |

---

## What Changes, What Doesn't

**Changes:**
- Scoreboard gets: feeder goroutine, alerts table, alerts API
- Letta agent gets: 3-5 new tools, updated system prompt
- WINS gets: alert display component
- Memory bridge: minor — use SDK for cleaner Letta calls (optional)

**Doesn't change:**
- Memory pipeline (extraction → queue → approval → Mem0)
- OpenClaw gateway (still routes Discord, still free models)
- Scoreboard scoring engine
- Checklist system
- Integration connectors (Stripe, GitHub, etc.)
- memory-syncd (still syncs blocks → workspace files)
