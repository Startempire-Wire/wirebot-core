# Memory Bridge Strategy

## The Problem

Three memory systems running in isolation with zero coordination:

| Layer | Port | What It Does | What It Knows Today |
|-------|------|-------------|-------------------|
| **memory-core** | embedded in :18789 | Hybrid BM25+vector search over workspace .md files | 2 files, 4 chunks |
| **Mem0** | :8200 | LLM-extracted facts from conversations, vector+graph | 4 test facts |
| **Letta** | :8283 | Stateful agent with self-editing memory blocks, PostgreSQL-backed | 1 test message |

No caller for Mem0. No caller for Letta. The `wirebot-memory` skill describes the dream but has no executable code.

---

## Each System's Actual Strengths (From Research)

### memory-core (OpenClaw built-in)
**What it is:** File-watching vector index over the workspace directory.
**Strengths:**
- Zero-latency: embedded in gateway process, no HTTP call
- Auto-indexes `.md` files on change (inotify watcher)
- Hybrid search: 70% vector + 30% BM25 text matching
- OpenRouter embeddings (1536 dims) with local fallback (768 dims)
- Integrated with agent context — search results injected into system prompt

**Best for:** Conversational recall of workspace knowledge — what's in MEMORY.md, daily logs, identity docs. The "what did we write down" layer.

**Limitation:** Only sees files in the workspace. Doesn't extract facts from conversations. Can't store structured state.

### Mem0 (REST server wrapping mem0ai)
**What it is:** LLM-powered fact extraction + vector search + optional graph memory.
**Strengths (from research paper, arXiv 2504.19413):**
- **Automatic fact extraction**: LLM processes conversation pairs, extracts salient facts, deduplicates against existing memories
- **Contradiction resolution**: When new facts conflict with old ones, Mem0 updates (not duplicates)
- **91% lower p95 latency** vs full-context approach, 90%+ token savings
- **Cross-session persistence**: Facts survive session boundaries
- **Graph memory** (Mem0^g): Stores entities + relationships as directed graph — "Verious → founded → Startempire Wire" — for multi-hop reasoning
- **Update/delete/history**: Full CRUD on individual memories with change history

**Best for:** Extracting durable facts from conversations — user preferences, decisions, relationships, business context. The "what did you tell me" layer. Cross-surface: same facts available to dashboard, Chrome extension, cron jobs.

**Limitation:** Requires LLM call for every store operation (fact extraction). Not real-time for high-volume writes.

### Letta (MemGPT-based stateful agent)
**What it is:** A full stateful agent that manages its own memory blocks + archival storage.
**Strengths (from Letta docs):**
- **Self-editing memory blocks**: Agent autonomously decides what to persist in labeled blocks (persona, human, business_stage, goals, kpis, etc.)
- **Memory hierarchy**: Core memory (always in context) → Recall memory (searchable conversation history) → Archival memory (long-term vector DB)
- **Persistent state**: Single perpetual thread per agent — no sessions, everything is one continuous interaction
- **Sleep-time compute**: Can process and reorganize memories asynchronously during idle periods
- **Shared blocks**: Multiple agents can read/write the same block — coordination primitive
- **Custom tools**: Can register Python functions as agent tools (including HTTP calls to external services)
- **Built-in tools**: `memory_replace`, `memory_insert`, `memory_rethink`, `conversation_search`, `archival_memory_insert/search`

**Best for:** Structured business state that evolves over time — stage progression, goal tracking, KPI snapshots, checklist state. The "what stage am I in and what's next" layer. The agent actively maintains its own understanding.

**Limitation:** Every interaction costs an LLM call. Cold start is slow (agent reasoning loop). Not suitable for high-frequency reads.

---

## Bridge Architecture

### Design Principle: **Write-Through, Read-Cascade**

Each system has a **primary role** with clear write ownership. Reads cascade from fast→slow when the fast layer doesn't have the answer.

```
┌─────────────────────────────────────────────────────────────────┐
│                    WIREBOT MEMORY BRIDGE                        │
│                                                                 │
│  ┌─────────────┐   ┌──────────────┐   ┌───────────────────┐   │
│  │ memory-core │   │    Mem0      │   │      Letta        │   │
│  │ (workspace) │   │  (facts)     │   │ (business state)  │   │
│  │             │   │              │   │                    │   │
│  │ READS:      │   │ READS:       │   │ READS:            │   │
│  │  instant    │   │  ~200ms      │   │  ~1-3s (LLM)      │   │
│  │  auto-index │   │  vector srch │   │  block read (fast) │   │
│  │             │   │              │   │  agent msg (slow)  │   │
│  │ WRITES:     │   │ WRITES:      │   │ WRITES:           │   │
│  │  file write │   │  POST /store │   │  block update API  │   │
│  │  → auto-idx │   │  → LLM extract│  │  or agent msg      │   │
│  └──────┬──────┘   └──────┬───────┘   └───────┬───────────┘   │
│         │                 │                    │                │
│    ─────┴─────────────────┴────────────────────┴──────          │
│                    BRIDGE LAYER                                  │
│              (OpenClaw plugin + cron)                           │
└─────────────────────────────────────────────────────────────────┘
```

### Write Ownership (Who Writes What)

| Data Type | Primary Writer | Why |
|-----------|---------------|-----|
| Daily logs, identity, soul | **memory-core** (via workspace files) | Files are the source of truth; auto-indexed |
| User preferences, decisions, facts | **Mem0** (via bridge after conversations) | LLM-extracted, deduplicated, contradiction-resolved |
| Business stage, goals, KPIs, checklists | **Letta** (via memory blocks) | Agent self-manages structured state |
| Conversation history | **OpenClaw sessions** (built-in) | Already handled by gateway |

### Read Cascade (How Recall Works)

When Wirebot needs to recall something:

```
1. memory-core (instant, 0ms overhead)
   ↓ not found?
2. Mem0 search (fast, ~200ms HTTP)
   ↓ not found?
3. Letta block read (fast, ~100ms HTTP) or agent query (slow, ~2s)
```

This is implemented as a single OpenClaw tool: `wirebot_recall`.

### Sync Flows (How Data Moves Between Systems)

#### Flow 1: Conversation → Mem0 (Post-turn extraction)
**Trigger:** After every agent turn (OpenClaw hook: `afterAgentTurn`)
**Logic:**
```
1. Get the last user message + assistant response
2. POST to Mem0 /v1/store with the conversation pair
3. Mem0's LLM extracts facts, deduplicates, stores
```
**Cost:** ~$0.001 per turn (Haiku-class LLM for extraction)
**Latency:** Async, doesn't block the response

#### Flow 2: Mem0 → Workspace (Nightly consolidation)
**Trigger:** Cron, daily at midnight PT
**Logic:**
```
1. GET all Mem0 facts for user "verious"
2. Diff against current MEMORY.md
3. Append new durable facts to MEMORY.md
4. memory-core auto-re-indexes on file change
```
**Purpose:** Ensures workspace files stay the authoritative long-term record. Mem0 is the fast-access cache; MEMORY.md is the archive.

#### Flow 3: Business State → Letta (Event-driven)
**Trigger:** When business-critical events happen (stage change, goal set, KPI update)
**Logic:**
```
1. Bridge detects business state change (keyword match or explicit command)
2. Update Letta memory blocks via REST API:
   - PUT /v1/blocks/{block_id} with new value
   - Or: POST message to Letta agent asking it to update its own state
3. Letta agent self-manages block organization
```
**Cost:** Only on business state changes (rare, ~1-5/day)

#### Flow 4: Letta → Workspace (Nightly snapshot)
**Trigger:** Cron, daily at midnight PT (same job as Flow 2)
**Logic:**
```
1. GET Letta agent state + all blocks
2. Write to workspace: clawd/BUSINESS_STATE.md
3. memory-core auto-indexes
```
**Purpose:** Ensures Letta state is recoverable from workspace alone.

#### Flow 5: Cron Accountability → All Three
**Trigger:** Daily Standup (8AM), EOD Review (6PM), Weekly Planning (Mon 7AM)
**Logic:**
```
1. Cron runs OpenClaw agent turn (existing behavior)
2. Bridge extracts facts from cron output → Mem0
3. Bridge checks for business state changes → Letta block updates
4. Agent writes daily log → workspace → memory-core auto-indexes
```

---

## Implementation Plan

### Component 1: OpenClaw Plugin — `wirebot-memory-bridge`

A OpenClaw extension (TypeScript, loaded at gateway startup) that:
- Registers 3 tools: `wirebot_recall`, `wirebot_remember`, `wirebot_business_state`
- Hooks into `afterAgentTurn` for automatic Mem0 extraction
- Runs as `kind: "extension"` (not `kind: "memory"`, so no slot conflict)

```typescript
// wirebot-memory-bridge/index.ts (simplified)

const bridge = {
  id: "wirebot-memory-bridge",
  name: "Wirebot Memory Bridge",
  description: "Coordinates memory across memory-core, Mem0, and Letta",
  kind: "extension" as const,

  register(api: OpenClawPluginApi) {

    // === Tool: wirebot_recall ===
    // Cascading search across all three memory layers
    api.registerTool({
      name: "wirebot_recall",
      description: "Search Wirebot's complete memory. Checks workspace, facts, and business state.",
      parameters: Type.Object({
        query: Type.String({ description: "What to recall" }),
      }),
      async execute(_id, params) {
        const { query } = params;
        const results = [];

        // Layer 1: memory-core (via built-in — already in context)
        // Skip — memory-core results are already in the agent's context

        // Layer 2: Mem0 fact search
        const mem0 = await fetch("http://127.0.0.1:8200/v1/search", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ query, namespace: "wirebot_verious", limit: 5 }),
        }).then(r => r.json());
        if (mem0.results?.length) {
          results.push(...mem0.results.map(r => `[fact] ${r.memory} (${(r.score*100).toFixed(0)}%)`));
        }

        // Layer 3: Letta block read (fast, no LLM call)
        const blocks = await fetch("http://127.0.0.1:8283/v1/blocks/", {
          headers: { "Content-Type": "application/json" },
        }).then(r => r.json());
        for (const block of blocks) {
          if (block.value?.toLowerCase().includes(query.toLowerCase())) {
            results.push(`[state:${block.label}] ${block.value}`);
          }
        }

        return { content: [{ type: "text", text: results.join("\n") || "Nothing found." }] };
      },
    });

    // === Tool: wirebot_remember ===
    // Store a fact in Mem0
    api.registerTool({
      name: "wirebot_remember",
      description: "Store a fact or preference in long-term memory. Use for decisions, preferences, context.",
      parameters: Type.Object({
        fact: Type.String({ description: "The fact to remember" }),
      }),
      async execute(_id, params) {
        const result = await fetch("http://127.0.0.1:8200/v1/store", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            text: params.fact,
            namespace: "wirebot_verious",
          }),
        }).then(r => r.json());
        return { content: [{ type: "text", text: `Stored: ${params.fact}` }] };
      },
    });

    // === Tool: wirebot_business_state ===
    // Read/update business state via Letta blocks
    api.registerTool({
      name: "wirebot_business_state",
      description: "Read or update business state (stage, goals, KPIs). Use for business coaching.",
      parameters: Type.Object({
        action: Type.Union([Type.Literal("read"), Type.Literal("update")]),
        block_label: Type.Optional(Type.String()),
        value: Type.Optional(Type.String()),
      }),
      async execute(_id, params) {
        const LETTA_AGENT = "agent-82610d14-ec65-4d10-9ec2-8c479848cea9";

        if (params.action === "read") {
          const agent = await fetch(`http://127.0.0.1:8283/v1/agents/${LETTA_AGENT}`).then(r => r.json());
          const blocks = agent.blocks || [];
          const text = blocks.map(b => `[${b.label}] ${b.value}`).join("\n") || "No business state blocks yet.";
          return { content: [{ type: "text", text }] };
        }

        // Update: send message to Letta agent, let it self-manage
        const resp = await fetch(`http://127.0.0.1:8283/v1/agents/${LETTA_AGENT}/messages/`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            messages: [{ role: "user", content: `Update business state: ${params.value}` }],
          }),
        }).then(r => r.json());

        return { content: [{ type: "text", text: "Business state update sent to Letta agent." }] };
      },
    });
  },
};
```

### Component 2: Letta Agent Memory Blocks

Create structured blocks on the Letta agent so it has labeled, self-editing state:

```bash
# Create blocks via Letta API
curl -X POST http://127.0.0.1:8283/v1/blocks/ -H "Content-Type: application/json" \
  -d '{"label":"business_stage","description":"Current business stage: Idea, Launch, or Growth. Update when milestones are hit.","value":"Stage: Idea\nMilestone: Pre-MVP\nNext: Build dashboard prototype","limit":2000}'

curl -X POST http://127.0.0.1:8283/v1/blocks/ -H "Content-Type: application/json" \
  -d '{"label":"goals","description":"Active business goals with deadlines. Update when goals change or complete.","value":"1. Onboard first beta tester (2 weeks)\n2. Build Business Setup Checklist Engine\n3. Build Dashboard Frontend matching Figma","limit":3000}'

curl -X POST http://127.0.0.1:8283/v1/blocks/ -H "Content-Type: application/json" \
  -d '{"label":"kpis","description":"Key performance indicators. Update with latest numbers.","value":"Beta testers: 0\nDaily active users: 0\nChecklist completion rate: N/A\nMembership signups: 0","limit":2000}'

curl -X POST http://127.0.0.1:8283/v1/blocks/ -H "Content-Type: application/json" \
  -d '{"label":"human","description":"Information about the user (founder/business owner).","value":"Name: Verious Smith III\nBusiness: Startempire Wire\nTimezone: Pacific\nStage: Idea\nFocus: AI-powered business coaching dashboard","limit":3000}'
```

### Component 3: Nightly Sync Cron

```bash
#!/usr/bin/env bash
# /data/wirebot/bin/memory-sync.sh
# Runs nightly: consolidates Mem0 facts → MEMORY.md, snapshots Letta → BUSINESS_STATE.md

# 1. Pull all Mem0 facts
FACTS=$(curl -s http://127.0.0.1:8200/v1/list -X POST \
  -H "Content-Type: application/json" \
  -d '{"namespace":"wirebot_verious"}')

# 2. Append new facts to MEMORY.md (dedup by content hash)
echo "$FACTS" | jq -r '.results[].memory' | while read -r fact; do
  if ! grep -qF "$fact" /home/wirebot/clawd/MEMORY.md; then
    echo "- $fact" >> /home/wirebot/clawd/MEMORY.md
  fi
done

# 3. Snapshot Letta blocks → BUSINESS_STATE.md
LETTA_AGENT="agent-82610d14-ec65-4d10-9ec2-8c479848cea9"
curl -sL "http://127.0.0.1:8283/v1/agents/${LETTA_AGENT}" | \
  jq -r '.blocks[] | "## \(.label)\n\(.value)\n"' > /home/wirebot/clawd/BUSINESS_STATE.md

# 4. memory-core auto-re-indexes on file change (inotify watcher)
```

### Component 4: Letta Agent System Prompt Enhancement

Update the Letta agent's system prompt to know its role in the Wirebot ecosystem:

```
You are Wirebot's business state engine. You maintain structured memory blocks about Verious Smith's business:
- business_stage: Current stage (Idea/Launch/Growth), milestones, next steps
- goals: Active goals with deadlines and progress
- kpis: Key metrics (beta testers, DAU, completion rates, signups)
- human: User context (name, business, timezone, focus)

When you receive updates, use memory_replace or memory_insert to update the relevant block.
Keep blocks concise and current. Remove completed goals. Update KPIs with latest numbers.
You are NOT a conversational assistant — you are a state management engine.
```

---

## Race Condition Prevention

### Problem: Concurrent writes from cron + live conversation

**Solution:** 
1. **Mem0 is idempotent** — `add()` deduplicates via vector similarity (>0.95 = same fact). No race condition possible.
2. **Letta blocks use last-write-wins** — acceptable because updates are sparse (~1-5/day) and semantic (not numeric counters).
3. **Workspace files** — Only one writer at a time (OpenClaw agent or sync script). Sync script runs at midnight when no live conversation is expected. If collision occurs, `memory-core` re-indexes on next file change.
4. **Bridge extraction is async** — `afterAgentTurn` hook fires after response is sent, so extraction doesn't block live conversation.

### Problem: Stale reads across layers

**Solution:** Read cascade always starts with memory-core (most fresh, auto-indexed). Mem0 facts are durable (not time-sensitive). Letta blocks are structured state (not conversational). Staleness window is acceptable for each layer's data type.

---

## Cost Analysis

| Operation | LLM Cost | Frequency | Monthly Cost |
|-----------|----------|-----------|-------------|
| Mem0 fact extraction (Haiku) | ~$0.001/turn | ~50 turns/day | ~$1.50 |
| Letta block updates | ~$0.003/update | ~5/day | ~$0.45 |
| Nightly sync (no LLM) | $0.00 | 1/day | $0.00 |
| memory-core search | $0.00 | embedded | $0.00 |
| Embeddings (text-embedding-3-small) | ~$0.0001/query | ~100/day | ~$0.30 |
| **Total** | | | **~$2.25/month** |

---

## Implementation Order

1. **Create Letta memory blocks** — 10 min, REST API calls
2. **Build wirebot-memory-bridge plugin** — 2 hours, TypeScript
3. **Wire plugin into gateway config** — 5 min, config change
4. **Build nightly sync script** — 30 min, bash + cron
5. **Update Letta agent system prompt** — 10 min, REST API
6. **Test end-to-end recall cascade** — 30 min
7. **Update wirebot-memory skill** — 10 min, document actual wiring

---

## Success Criteria

- [ ] `wirebot_recall "membership tiers"` returns results from all three layers
- [ ] After a conversation, new facts appear in Mem0 within 5 seconds
- [ ] Business state changes (e.g., "I got my first beta tester") update Letta blocks
- [ ] Nightly sync produces BUSINESS_STATE.md and updates MEMORY.md
- [ ] No duplicate facts across layers after 1 week of operation
- [ ] Gateway restart doesn't lose any state (all three persist independently)
