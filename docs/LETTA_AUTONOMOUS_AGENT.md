# Letta as Wirebot Subsystem — Structured State Engine

> **Status:** Design doc — not yet implemented  
> **Principle:** Wirebot is the agent. Letta is the state engine. Autonomy belongs to Wirebot.

---

## Mental Model

```
┌─────────────────────────────────────────────────────────┐
│                    WIREBOT (the agent)                   │
│                                                         │
│  Surfaces: Discord, WINS Portal, Cron, Webhooks         │
│  Brain:    OpenClaw Gateway (kimi-coding/k2p5)          │
│  Memory:   Mem0 (facts) + Letta (structured state)      │
│  Actions:  Respond, Score, Alert, Ship, Remember         │
│                                                         │
│  ┌─────────────────────────────────────────────────┐    │
│  │              SUBSYSTEMS                          │    │
│  │                                                  │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │    │
│  │  │  Mem0    │  │  Letta   │  │  Scoreboard  │  │    │
│  │  │  (facts) │  │  (state) │  │  (scoring)   │  │    │
│  │  └──────────┘  └──────────┘  └──────────────┘  │    │
│  │                                                  │    │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────┐  │    │
│  │  │ Checklist│  │ Memory   │  │ Integrations │  │    │
│  │  │ (tasks)  │  │ Queue    │  │ (Stripe,GH)  │  │    │
│  │  └──────────┘  └──────────┘  └──────────────┘  │    │
│  └─────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────┘
```

**Wirebot** is the one entity the operator talks to. It decides, acts, remembers.  
**Letta** is one of Wirebot's subsystems — the structured state backend that gives Wirebot a persistent, self-organizing understanding of the operator's business.

Letta doesn't alert. Letta doesn't decide. Letta **structures and maintains state** so that when Wirebot needs to think, it has rich, current context.

---

## Current State (Passive Store)

Letta holds 4 blocks — `human`, `goals`, `kpis`, `business_stage` — totaling ~7.5KB of structured business context. Today:

- **Writes:** Memory bridge sends business-relevant snippets to Letta via `lettaSendMessage()`. Letta's internal LLM decides how to update its own blocks. This only fires when Discord conversations contain business keywords.
- **Reads:** `wirebot_recall` and `wirebot_business_state` tools read blocks on demand. Memory sync daemon writes `BUSINESS_STATE.md` to workspace every 30s.
- **Gap:** Blocks go stale between conversations. Approved memories never reach Letta. Scoreboard events don't update KPIs. Letta has no awareness of what happened since the last Discord chat.

---

## Target State (Active Subsystem)

Letta becomes Wirebot's **continuously-updated business model** — a structured representation of the operator's world that stays fresh without human prompting.

### What changes

| Aspect | Before | After |
|--------|--------|-------|
| Block freshness | Stale between conversations | Updated by event feed every 30s |
| Approved memories | Go to Mem0 only | Also ingested by Letta → block updates |
| Scoreboard events | Not seen by Letta | Fed to Letta → KPI/goal block updates |
| Integration data | Not seen by Letta | Stripe MRR, GitHub ships → KPI updates |
| Wirebot context | Reads blocks on-demand | Blocks are always current when read |

### What doesn't change

- **Wirebot is the agent.** Letta doesn't talk to users, generate alerts, or make decisions.
- **Memory bridge tools stay.** `wirebot_recall`, `wirebot_remember`, `wirebot_business_state` work as-is.
- **Scoreboard owns scoring, alerts, and the dashboard.** Letta doesn't push to UI.
- **Mem0 owns facts.** Letta owns structured state. No overlap.

---

## Architecture

```
Events flow IN to Letta (write path):

  Approved memory ──┐
  Scoreboard event ─┤
  Integration data ─┤──→ State Feeder ──→ Letta agent ──→ self-edits blocks
  Daily cron ───────┘    (goroutine)      (kimi LLM)      (memory_replace)
                              │
                              │ "Process this event and update
                              │  your blocks if relevant."
                              │
                         async message
                         (fire & forget)


Wirebot reads OUT from Letta (read path):

  Discord user asks ──→ OpenClaw ──→ wirebot_recall ──→ Letta blocks
  about business                     (memory bridge)     (always fresh)
                                          │
                                          ├──→ Mem0 facts
                                          └──→ Letta archival

  Scoreboard needs  ──→ memory-syncd ──→ Letta blocks ──→ BUSINESS_STATE.md
  workspace state        (30s poll)                        (workspace file)


Wirebot acts BASED ON Letta state:

  Wirebot cron (6am) ──→ reads blocks ──→ generates standup
  Wirebot scores     ──→ reads kpis   ──→ contextual scoring  
  Wirebot responds   ──→ reads goals  ──→ goal-aware conversation
  Scorebd alerts     ──→ reads blocks ──→ "goal deadline in 3 days"
```

---

## Component 1: State Feeder

A goroutine in the scoreboard that feeds events to Letta for block maintenance.

**Not an agent loop.** This is a data pipeline — it takes structured events and asks Letta's LLM to update its blocks accordingly. Letta doesn't decide what to do with events; it just structures them into its state model.

```go
// In scoreboard main.go
func (s *Server) lettaStateFeeder() {
    ticker := time.NewTicker(30 * time.Second)
    var lastApprovedID int64
    var lastEventID int64
    
    for range ticker.C {
        // 1. Ingest newly approved memories
        rows, _ := s.db.Query(`
            SELECT id, memory_text FROM memory_queue 
            WHERE status='approved' AND id > ? 
            ORDER BY id LIMIT 5`, lastApprovedID)
        for rows.Next() {
            var id int64; var text string
            rows.Scan(&id, &text)
            s.lettaIngest(fmt.Sprintf(
                "Approved fact from memory queue: %s\n"+
                "Update your blocks if this is relevant to goals, KPIs, "+
                "business stage, or operator context.", text))
            lastApprovedID = id
        }
        
        // 2. Ingest recent scoreboard events
        rows, _ = s.db.Query(`
            SELECT id, source, kind, summary FROM events 
            WHERE id > ? ORDER BY id LIMIT 5`, lastEventID)
        for rows.Next() {
            var id int64; var source, kind, summary string
            rows.Scan(&id, &source, &kind, &summary)
            s.lettaIngest(fmt.Sprintf(
                "Scoreboard event [%s/%s]: %s\n"+
                "Update KPIs or goals if relevant.", source, kind, summary))
            lastEventID = id
        }
    }
}

func (s *Server) lettaIngest(message string) {
    // Fire-and-forget async message to Letta
    // Letta's LLM processes it and self-edits blocks
    go func() {
        body, _ := json.Marshal(map[string]interface{}{
            "messages": []map[string]string{
                {"role": "user", "content": message},
            },
        })
        req, _ := http.NewRequest("POST", 
            s.lettaURL+"/v1/agents/"+s.lettaAgentID+"/messages/", 
            bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
        client := &http.Client{Timeout: 60 * time.Second}
        resp, err := client.Do(req)
        if err == nil { resp.Body.Close() }
    }()
}
```

**Rate limiting:** Max 10 messages per 5-minute window. Events queue in SQLite — nothing lost if Letta is slow.

### Component 2: Letta Agent Prompt (Updated)

Letta's system prompt shifts from "you are the Business State Engine" to a clearer subsystem role:

```markdown
You are a structured state engine — a subsystem of Wirebot.

Your job: maintain 4 memory blocks that represent the operator's 
current business reality. You receive events and facts and update 
your blocks to keep them accurate.

Blocks:
- human: Operator identity, preferences, working style, context
- goals: Active goals with dates, progress, blockers
- kpis: Key metrics with current values and directional trends  
- business_stage: Score, stage, pairing profile, active contexts

When you receive an event or fact:
1. Decide which block(s) it affects
2. Use memory_replace to update the relevant section
3. Keep blocks concise (<3000 chars). Summarize, don't append.
4. If a fact contradicts existing state, update to the newer truth.
5. If a fact is not relevant to any block, do nothing.

You do NOT:
- Generate alerts (Wirebot's scoreboard handles that)
- Talk to users (Wirebot handles conversation)
- Make business decisions (the operator decides)
- Fabricate data (only use what you're given)

You ARE:
- The operator's structured business memory
- Always current, always concise, always accurate
- A subsystem that makes Wirebot smarter
```

### Component 3: Wirebot Uses Fresh State

The payoff: Wirebot reads from Letta and gets **current** context instead of stale blocks.

**Already working (no changes needed):**
- `wirebot_recall` reads Letta blocks → now they're fresh
- `wirebot_business_state` reads/writes blocks → now auto-maintained
- `memory-syncd` writes `BUSINESS_STATE.md` → now reflects latest events
- `agent_start` hook injects `SCOREBOARD_STATE.md` → Letta blocks add depth

**Wirebot gains autonomy through fresh state, not through Letta acting independently:**

| Wirebot capability | Enabled by fresh Letta state |
|-------------------|------------------------------|
| Goal-aware responses | "You mentioned goal X is due Friday — are you on track?" |
| KPI-contextual scoring | Score weight adjusts based on current business stage |
| Morning standup | Reads goals + kpis + recent events → generates brief |
| Proactive nudges | Scoreboard checks goals block → flags approaching deadlines |
| Conversation continuity | Blocks carry forward between sessions automatically |

### Component 4: Scoreboard Alert Logic (Optional Enhancement)

The **scoreboard** (not Letta) can generate alerts by reading Letta's blocks:

```go
// Scoreboard cron check (daily or hourly)
func (s *Server) checkGoalDeadlines() {
    blocks := lettaGetBlocks(s.lettaURL, s.lettaAgentID)
    goalsBlock := findBlock(blocks, "goals")
    
    // Parse goal deadlines from structured text
    for _, goal := range parseGoals(goalsBlock.Value) {
        daysLeft := goal.Deadline.Sub(time.Now()).Hours() / 24
        if daysLeft <= 3 && daysLeft > 0 && !goal.Complete {
            s.createAlert("warning", 
                fmt.Sprintf("Goal deadline in %d days: %s", int(daysLeft), goal.Title))
        }
    }
}
```

This keeps alerting in the scoreboard (Go, deterministic logic) rather than in Letta (LLM, non-deterministic). Letta provides the structured data; the scoreboard acts on it.

---

## Data Flow: End-to-End Example

**Scenario:** Operator ships a feature, discusses it on Discord, memory gets approved.

```
1. GitHub webhook → Scoreboard event (source: github, kind: push)
   
2. Discord: "I just shipped the auth module for Startempire"
   → OpenClaw responds with encouragement
   → agent_end hook:
     a. Memory bridge → Mem0 store("Shipped auth module for STA")
     b. Memory bridge → Scoreboard extraction queue  
     c. Memory bridge → Letta message (keyword: "shipped")
        → Letta updates goals block: "Auth module — DONE ✅"

3. Operator approves memory in WINS Memory Review UI

4. State Feeder (next 30s tick):
   a. Picks up approved memory
   b. Sends to Letta: "Approved fact: Shipped auth module for STA"
   c. Letta updates kpis block: "Features shipped this week: 3"

5. Memory sync daemon (next 30s):
   a. Detects block changes → writes BUSINESS_STATE.md
   
6. Next Discord conversation:
   → wirebot_recall("what did I ship recently?")
   → Letta blocks return: goals shows auth complete, kpis shows 3 shipped
   → Wirebot responds with full context — no "I don't remember"

7. Scoreboard daily check (6am):
   → Reads goals block → "Goal 'Finalize MVP' — 4/7 features done"
   → Generates morning standup with progress update
```

---

## Implementation Plan

### Phase 1: State Feeder (est. 2 hours)
1. Add `lettaStateFeeder()` goroutine to scoreboard
2. Feed approved memories → Letta (track last processed ID)
3. Feed scoreboard events → Letta (track last event ID)
4. Rate limiter: max 10 messages per 5-minute window
5. Update Letta agent system prompt for subsystem role

### Phase 2: Richer Reads (est. 1 hour)
1. Enhance `wirebot_recall` to weight fresh Letta blocks higher
2. Add Letta block timestamps to recall output
3. Verify memory-syncd picks up feeder-driven block changes

### Phase 3: Scoreboard Alerts from State (est. 2 hours)
1. Add `alerts` table to scoreboard SQLite
2. Add `GET/POST /v1/alerts` endpoints
3. Scoreboard cron reads Letta blocks → generates alerts (goal deadlines, KPI drops)
4. WINS dashboard alert strip (admin section or dashboard top)

### Phase 4: Integration Events (est. 1 hour)
1. Stripe webhook events → feeder → Letta → kpis block
2. GitHub push events → feeder → Letta → goals/kpis block
3. Daily cron morning trigger → feeder → Letta → "generate summary of current state"

---

## Cost Impact

| Component | Model | Cost |
|-----------|-------|------|
| Letta block updates (feeder) | kimi-coding/k2p5 via gateway | $0 |
| Letta embeddings (archival) | letta/letta-free (local) | $0 |
| Expected volume | ~50-100 messages/day | $0 (kimi free tier) |
| **Total** | | **$0/month** |

---

## What This Enables for Wirebot

With always-fresh Letta blocks, Wirebot gains:

1. **Persistent business awareness** — knows the operator's current state without being told each session
2. **Goal tracking** — "You're 3 days from your onboarding deadline with 0 signups" comes naturally in conversation
3. **KPI context** — scoring and responses adapt to actual business metrics
4. **Cross-session continuity** — approved memories flow into structured state automatically
5. **Morning briefs** — daily cron reads fresh blocks and generates actionable standup

Letta isn't the agent. Letta is the structured memory that makes Wirebot **act like one.**
