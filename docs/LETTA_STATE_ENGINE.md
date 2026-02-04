# Letta State Engine — Wirebot Subsystem Design

> **Status:** Design doc — not yet implemented  
> **Principle:** Wirebot is the agent. Letta is the state engine. Autonomy belongs to Wirebot.

---

## Position in the Memory Stack

Wirebot has three non-overlapping memory subsystems (per [MEMORY.md](./MEMORY.md) and [MEMORY_ARCHITECTURE_AUDIT.md](./MEMORY_ARCHITECTURE_AUDIT.md)):

| Subsystem | Owns | Role |
|-----------|------|------|
| **memory-core** (OpenClaw built-in) | Identity, soul, daily logs, MEMORY.md | Workspace knowledge recall — instant hybrid search |
| **Mem0** (:8200) | Conversation facts, preferences, decisions | Cross-surface fact extraction + dedup + contradiction resolution |
| **Letta** (:8283) | `human`, `goals`, `kpis`, `business_stage` blocks + archival | **Structured, self-editing business state** |

**Ownership rules** (from [MEMORY_ARCHITECTURE_AUDIT.md](./MEMORY_ARCHITECTURE_AUDIT.md)):
- Raw conversation facts → Mem0 ONLY
- Structured state → Letta ONLY
- Identity/personality → workspace files ONLY
- Mem0 never writes to Letta. Letta never writes to Mem0.
- The bridge is the ONLY cross-system writer.

This doc addresses **Letta's role only** — how to evolve it from passive store to active state engine while keeping it a subsystem, not a competing agent.

---

## What Letta Is (and Isn't)

**Letta IS:**
- Wirebot's structured business memory
- A self-editing state engine — its internal LLM maintains 4 blocks
- The subsystem that gives Wirebot persistent business awareness
- Fed by events, updated autonomously, read on demand

**Letta IS NOT:**
- An autonomous agent (Wirebot is the agent)
- A decision-maker (the operator decides)
- An alerting system (the scoreboard alerts based on Letta state)
- A conversation partner (OpenClaw handles all user interaction)
- A replacement for Mem0 (facts stay in Mem0, structure stays in Letta)

---

## Current State

**What works** (per [MEMORY_ARCHITECTURE_AUDIT.md](./MEMORY_ARCHITECTURE_AUDIT.md)):
- ✅ 4 blocks maintained: `human` (2.9KB), `goals` (2KB), `kpis` (849B), `business_stage` (1.7KB)
- ✅ `agent_end` hook routes business/pairing keywords → Letta agent messages
- ✅ `wirebot_recall` reads blocks + searches archival (all 3 layers)
- ✅ `wirebot_business_state` supports read/update/message actions
- ✅ `memory-syncd` writes `BUSINESS_STATE.md` to workspace every 30s
- ✅ Letta agent has 3 tools: `conversation_search`, `memory_insert`, `memory_replace`
- ✅ LLM via local gateway (kimi-coding/k2p5, $0/month)
- ✅ Embeddings via letta/letta-free (local, $0)

**What's missing:**
- ❌ Approved memories never reach Letta — only go to Mem0
- ❌ Scoreboard events (ships, revenue) don't update Letta blocks
- ❌ Integration events (Stripe, GitHub) don't flow to Letta
- ❌ Blocks go stale between Discord conversations
- ❌ No mechanism for Wirebot to act on Letta state changes (alerts, nudges)

---

## Design: State Feeder

A goroutine in the scoreboard that keeps Letta's blocks current by feeding events as messages. This is a **data pipeline**, not an agent loop.

### Event Sources

| Source | Trigger | Message to Letta |
|--------|---------|-----------------|
| Approved memories | `memory_queue.status` → `approved` | "Approved fact: {text}. Update blocks if relevant." |
| Scoreboard events | New row in `events` table | "Event [{source}/{kind}]: {summary}. Update KPIs/goals." |
| Integration webhooks | Stripe, GitHub, etc. | "Integration [{provider}]: {event}. Update KPIs." |
| Daily cron | 6am Pacific | "Morning check-in. Summarize current state of all blocks." |

### Implementation

```go
// Goroutine in scoreboard main.go
func (s *Server) lettaStateFeeder() {
    ticker := time.NewTicker(30 * time.Second)
    var lastApprovedID, lastEventID int64
    var msgCount int
    var windowStart time.Time

    for range ticker.C {
        // Rate limit: max 10 messages per 5-minute window
        if time.Since(windowStart) > 5*time.Minute {
            msgCount = 0
            windowStart = time.Now()
        }
        if msgCount >= 10 { continue }

        // Approved memories → Letta
        rows, _ := s.db.Query(`
            SELECT id, memory_text FROM memory_queue
            WHERE status='approved' AND id > ?
            ORDER BY id LIMIT 3`, lastApprovedID)
        // ... send each as async message, track lastApprovedID

        // Scoreboard events → Letta
        rows, _ = s.db.Query(`
            SELECT id, source, kind, summary FROM events
            WHERE id > ? ORDER BY id LIMIT 3`, lastEventID)
        // ... send each as async message, track lastEventID
    }
}
```

### Why async messages (not direct block writes)

From the audit: "Letta's core value is that its agent REASONS about what to store. When we overwrite, we lose that intelligence."

The feeder sends context. Letta's internal LLM decides:
- Which block(s) to update
- What to keep vs replace
- How to summarize vs append

This respects Letta's design as a self-editing system.

---

## How Wirebot Benefits

With always-fresh blocks, Wirebot's existing capabilities improve without new code:

| Wirebot capability | How Letta state enables it |
|-------------------|----------------------------|
| `wirebot_recall` | Blocks are current — no stale "KPIs from 2 weeks ago" |
| `wirebot_business_state` | Reading returns live state, not last-conversation snapshot |
| Morning standup cron | Reads fresh goals + kpis → generates accurate brief |
| Conversation context | `agent_start` injects `BUSINESS_STATE.md` → Wirebot knows what happened overnight |
| Scoring context | Scoreboard reads kpis block → contextual scoring adjustments |

### Future: Scoreboard-driven alerts (not Letta-driven)

The scoreboard (deterministic Go logic) reads Letta blocks and generates alerts:

```go
// Scoreboard cron, not Letta
func (s *Server) checkGoalDeadlines() {
    blocks := lettaGetBlocks(s.lettaURL, s.lettaAgentID)
    goals := parseGoals(findBlock(blocks, "goals").Value)
    for _, g := range goals {
        if g.DaysLeft <= 3 && !g.Complete {
            s.insertAlert("goal_deadline", g.Title, g.DaysLeft)
        }
    }
}
```

Alerts come from the scoreboard reading structured state — not from Letta deciding to alert. This keeps alerting deterministic and testable.

---

## Cross-Reference: Existing Docs

### [MEMORY.md](./MEMORY.md) — Root Architecture
- ✅ Correctly positions Letta as "structured, queryable state"
- ✅ "This is not conversational memory. It is structured, queryable state."
- ⚠️ Still says "OpenClaw memory" in places — was Clawdbot, now OpenClaw. Partially updated.
- **No conflicts** with this design. State feeder is an enhancement, not a rewrite.

### [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md) — Integration Guide
- ✅ Correctly says "Letta = structured business state + agent runtime"
- ⚠️ Still says "not yet deployed" — Letta is now **deployed and active**
- ⚠️ Shows bare API calls — doesn't mention memory bridge or state feeder
- **Action:** Update deployment status. Add cross-ref to this doc for state feeder design.

### [MEMORY_BRIDGE_STRATEGY.md](./MEMORY_BRIDGE_STRATEGY.md) — Bridge Design
- ✅ Write-Through, Read-Cascade design is correct and implemented
- ✅ Correctly separates write ownership (facts → Mem0, state → Letta, files → workspace)
- ✅ "Flow 3: Business State → Letta (Event-driven)" aligns with state feeder
- ⚠️ Flow 3 planned "1-5/day" — state feeder increases to ~50-100/day
- **No conflicts.** State feeder implements the planned "event-driven" flow at higher volume.

### [MEMORY_ARCHITECTURE_AUDIT.md](./MEMORY_ARCHITECTURE_AUDIT.md) — Audit Results
- ✅ Identified exactly the gaps this design fills:
  - "Scoreboard events never update Letta blocks" → state feeder fixes
  - "Pairing answers stored in Mem0 only" → approved memories feed both
  - "Direct API PUT bypasses agent" → feeder uses async messages
- ✅ All 7 fixes marked done
- ⚠️ Fix 4 ("Wire scoreboard events → Letta") marked done but only for agent_end hook — scoreboard events/integrations still don't flow. State feeder completes this.
- **Action:** After implementation, update audit fix #4 status.

### [MEM0_PLUGIN.md](./MEM0_PLUGIN.md) — Mem0 Details
- ⚠️ Outdated — still says "Clawdbot", references old config format, says "not yet deployed"
- ⚠️ Shows skeleton plugin tools — actual implementation is in memory bridge, not this plugin
- **No conflicts** with Letta design. Mem0 and Letta have clean ownership boundaries.
- **Action:** Update Clawdbot→OpenClaw, mark as deployed, reference memory bridge as actual implementation.

### [CLAWDBOT_MEMORY_DEEP_DIVE.md](./CLAWDBOT_MEMORY_DEEP_DIVE.md) — memory-core Internals
- ⚠️ Title and content still say "Clawdbot" throughout
- ✅ Correctly scoped to memory-core (workspace file indexing)
- ✅ "Wirebot uses Clawdbot memory as-is and adds Letta + Mem0" — correct framing
- **No conflicts.** memory-core is unaffected by state feeder.
- **Action:** Rename Clawdbot→OpenClaw throughout.

### [MEMORY_APPROVAL_SYSTEM.md](./MEMORY_APPROVAL_SYSTEM.md) — Queue Design
- ✅ Covers extraction → queue → human review → approve/reject
- ❌ Does not mention Letta as a consumer of approved memories
- **Action:** Add "approved memories also feed Letta state feeder" to the approval flow diagram.

---

## What Needs Updating (Doc Hygiene)

| Doc | Issue | Fix |
|-----|-------|-----|
| `LETTA_INTEGRATION.md` | Says "not yet deployed" | Update status: deployed, active, kimi model |
| `LETTA_INTEGRATION.md` | No mention of state feeder | Add cross-ref to this doc |
| `MEM0_PLUGIN.md` | Says "Clawdbot", "not yet deployed", shows skeleton | Update: OpenClaw, deployed, reference memory bridge |
| `CLAWDBOT_MEMORY_DEEP_DIVE.md` | Filename and content say "Clawdbot" | Rename file + content to OpenClaw |
| `MEMORY_BRIDGE_STRATEGY.md` | Says "Clawdbot" in places | Update to OpenClaw |
| `MEMORY_ARCHITECTURE_AUDIT.md` | Fix #4 incomplete | Update after state feeder ships |
| `MEMORY_APPROVAL_SYSTEM.md` | Missing Letta as approved-memory consumer | Add to approval flow |
| 26 docs total | Still reference "Clawdbot" | Batch find/replace Clawdbot→OpenClaw |

---

## Implementation Plan

### Phase 1: State Feeder (est. 2 hours)
1. Add `lettaStateFeeder()` goroutine to scoreboard
2. Feed approved memories → Letta async messages
3. Feed scoreboard events → Letta async messages
4. Rate limiter: max 10 messages per 5-minute window
5. Track watermarks (last processed IDs) in SQLite

### Phase 2: Agent Prompt Update (est. 30 min)
1. Update Letta system prompt — subsystem role, not agent role
2. Clarify block maintenance rules in prompt
3. Test: send approved memory → verify block update

### Phase 3: Scoreboard Reads State (est. 2 hours)
1. Alerts table + API endpoints
2. Scoreboard cron reads Letta blocks → generates alerts (goal deadlines, KPI drops)
3. WINS dashboard alert display

### Phase 4: Doc Cleanup (est. 1 hour)
1. Batch Clawdbot→OpenClaw across all 26 docs
2. Update deployment statuses
3. Add cross-references between memory docs

---

## Cost

| Component | Cost |
|-----------|------|
| Letta block updates via kimi gateway | $0/month |
| ~50-100 async messages/day | Within kimi free tier |
| **Total** | **$0/month** |

---

## See Also

- [MEMORY.md](./MEMORY.md) — Root memory architecture (3-layer stack)
- [MEMORY_BRIDGE_STRATEGY.md](./MEMORY_BRIDGE_STRATEGY.md) — Write-Through, Read-Cascade design
- [MEMORY_ARCHITECTURE_AUDIT.md](./MEMORY_ARCHITECTURE_AUDIT.md) — Planned vs actual vs corrected
- [MEM0_PLUGIN.md](./MEM0_PLUGIN.md) — Mem0 fact store details
- [CLAWDBOT_MEMORY_DEEP_DIVE.md](./CLAWDBOT_MEMORY_DEEP_DIVE.md) — memory-core internals
- [MEMORY_APPROVAL_SYSTEM.md](./MEMORY_APPROVAL_SYSTEM.md) — Extraction queue + human review
- [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md) — Letta API usage + per-user agents
- [LLM_MODEL_INVENTORY.md](./LLM_MODEL_INVENTORY.md) — All model references including Letta
