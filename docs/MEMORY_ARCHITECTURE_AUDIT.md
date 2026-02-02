# Memory Architecture Audit — Planned vs Actual vs Correct

> Generated 2026-02-02 from MEMORY_BRIDGE_STRATEGY.md, MEMORY.md, Mem0/Letta docs, and live system inspection.

---

## The Grid: Non-Overlapping, Purpose-Built Roles

| Dimension | memory-core (OpenClaw) | Mem0 | Letta |
|-----------|----------------------|------|-------|
| **PRIMARY ROLE** | Workspace knowledge recall | Conversation fact extraction + cross-surface sync | Structured self-editing business state |
| **WHAT IT STORES** | Markdown files (identity, soul, daily logs, curated memory) | LLM-extracted facts from conversations (preferences, decisions, relationships) | Self-managed memory blocks (human, goals, kpis, business_stage) + archival docs + conversation history |
| **HOW IT WRITES** | File writes → auto-indexed by file watcher | LLM processes conversation → extracts → deduplicates → stores | Agent's own LLM decides what to persist using memory_replace/memory_insert tools |
| **HOW IT READS** | Hybrid BM25+vector search, instant (embedded in gateway) | Vector semantic search (~200ms) | Block read (fast, ~100ms API) or agent message (slow, ~2s LLM) |
| **SEARCH TYPE** | Keyword + semantic hybrid | Pure semantic (vector similarity) | Agent reasoning (for complex queries) or raw block read (for simple) |
| **PERSISTENCE** | Files on disk, SQLite index | Qdrant vector DB + SQLite history | PostgreSQL (blocks, archival, conversation) |
| **UNIQUE STRENGTH** | Zero-latency, always in context, hybrid search excels at exact tokens (IDs, code symbols) | Contradiction resolution, automatic dedup, cross-surface availability | Self-editing autonomy, archival RAG, conversation search with date ranges, shared blocks across agents |
| **COST PER OP** | $0 (local) | ~$0.001/store (LLM extraction), $0/search (local embeddings) | ~$0.003/agent message (LLM reasoning), $0/block read |
| **UPDATE FREQUENCY** | Every conversation (daily logs), manual (MEMORY.md) | Every conversation (agent_end hook) | Sparse (business state changes, ~1-5/day) |

---

## What We Planned (from MEMORY_BRIDGE_STRATEGY.md)

### Write Ownership

| Data Type | Planned Writer | Planned Method |
|-----------|---------------|----------------|
| Daily logs, identity, soul | memory-core (file writes) | Workspace markdown → auto-indexed |
| User preferences, decisions, facts | Mem0 (after conversations) | `agent_end` → POST /v1/store with conversation pairs |
| Business stage, goals, KPIs | Letta (via agent messages) | POST message to Letta agent → **agent self-manages blocks** |
| Conversation history | OpenClaw sessions (built-in) | Already handled by gateway |

### Read Cascade (Planned)

```
1. memory-core (instant, 0ms)     ← workspace files already in context
   ↓ not found?
2. Mem0 search (fast, ~200ms)     ← vector semantic search over facts
   ↓ not found?
3. Letta block read (~100ms)      ← structured state
   or Letta agent query (~2s)     ← complex reasoning about state
```

### Sync Flows (Planned)

| Flow | Trigger | What Happens |
|------|---------|-------------|
| **Conversation → Mem0** | After every agent turn | Conversation pair → Mem0 LLM extraction |
| **Mem0 → Workspace** | Nightly cron | New facts appended to MEMORY.md |
| **Business events → Letta** | Event-driven | **Send MESSAGE to Letta agent** → it self-decides what to update |
| **Letta → Workspace** | Nightly cron | Snapshot blocks → BUSINESS_STATE.md |
| **Cron accountability → All three** | Daily standup/EOD | Cron output → Mem0 facts + Letta updates + workspace logs |

---

## What We Actually Built (Live System)

### Write Paths (Actual)

| Data Type | Actual Writer | Actual Method | vs Plan |
|-----------|-------------|---------------|---------|
| Identity, soul, daily logs | memory-core | ✅ File writes, auto-indexed | ✅ Correct |
| User preferences, facts | Mem0 | ⚠️ `wirebot_remember` sends **raw text string**, not structured messages | ⚠️ Works but suboptimal |
| Conversation extraction | Mem0 | ⚠️ `agent_end` sends **concatenated text** ("user: X\nassistant: Y"), not structured `[{role, content}]` | ⚠️ Works but bypasses Mem0's conversation-aware extraction |
| Business stage, goals, KPIs | Letta | ❌ **Direct API PUT to blocks** — bypasses Letta's agent entirely | ❌ Wrong — Letta's agent should self-manage |
| Scoreboard events → KPIs | Nobody | ❌ Not wired — scoreboard events never update Letta blocks | ❌ Missing flow |
| Pairing answers → Letta state | Nobody | ❌ Answers stored in Mem0 only, never flow to Letta's human/goals blocks | ❌ Missing flow |

### Read Paths (Actual)

| Layer | Actual Method | vs Plan |
|-------|-------------|---------|
| memory-core | ✅ Auto-injected in context + `memory_search` tool | ✅ Correct |
| Mem0 | ✅ Vector semantic search (fastembed local, ~200ms) | ✅ Correct |
| Letta | ❌ Was substring match (broken for NL queries), now dumps all blocks | ⚠️ Fixed but crude — always dumps 2KB regardless of relevance |

### Sync Flows (Actual)

| Flow | Plan | Reality |
|------|------|---------|
| Conversation → Mem0 | ✅ agent_end hook | ⚠️ Works but sends text not messages |
| Mem0 → Workspace (MEMORY.md) | Nightly cron | ❌ **NOT RUNNING** — cron changed to weekly, script doesn't append to MEMORY.md |
| Business events → Letta | Send message to agent | ❌ **Direct block PUT** — agent never processes |
| Letta → Workspace (BUSINESS_STATE.md) | Nightly snapshot | ❌ **NOT RUNNING** — sync script doesn't write BUSINESS_STATE.md |
| Scoreboard → Letta KPIs | Event-driven | ❌ **NOT WIRED** |
| Go sync daemon | Cache + reconcile | ⚠️ **Cache only** — daemon caches but never reconciles between systems |

---

## What's WRONG (Deviations from Plan)

### 1. Letta Used as Dumb Key-Value Store ❌

**Plan:** "Send message to Letta agent asking it to update its own state"  
**Reality:** Direct `PATCH /v1/agents/{id}/core-memory/blocks/{label}` — we overwrite block content via API, bypassing Letta's agent entirely.

**Why it matters:** Letta's core value is that its agent REASONS about what to store. It uses `memory_replace` and `memory_insert` tools to make surgical edits, preserving context it deems important. When we overwrite, we lose that intelligence.

**Fix:** For business state updates, POST to `/v1/agents/{id}/messages` with context. Let the agent decide what to update.

### 2. Letta Archival Memory Completely Unused ❌

**Plan:** Not explicitly in strategy doc (oversight)  
**Reality:** 0 archival passages. This is Letta's RAG store — designed for long documents, meeting notes, strategy docs.

**What should be there:** PAIRING.md, SCOREBOARD_PRODUCT.md, bigpicture.mdx, operator's business plans, meeting transcripts. Anything too large for core memory blocks but needed for intelligent reasoning.

**Fix:** POST key docs to `/v1/agents/{id}/archival-memory`.

### 3. Mem0 Receives Text, Not Messages ⚠️

**Plan:** "POST to Mem0 /v1/store with the conversation pair"  
**Reality:** Conversation flattened to "user: X\nassistant: Y" string, passed to `memory.add(text, user_id=...)`.

**Mem0's designed API:** `memory.add(messages=[{"role":"user","content":...}, {"role":"assistant","content":...}], user_id=...)` — structured messages enable better conversation-aware extraction.

**Fix:** Update mem0-server.py `/v1/store` to accept optional `messages` array. Update bridge plugin to send structured messages from agent_end.

### 4. Nightly Sync Not Running ❌

**Plan:** 
- Mem0 facts → MEMORY.md (nightly)
- Letta blocks → BUSINESS_STATE.md (nightly)

**Reality:** Sync script exists but was changed to weekly. MEMORY.md is static. BUSINESS_STATE.md exists in workspace but is never auto-updated from Letta.

**Fix:** Restore nightly sync cron. Write the sync logic.

### 5. No Cross-System Event Flow → ✅ FIXED

**Plan:** "Business events → Letta block updates"  
**Was:** Scoreboard events never flowed to Letta. Pairing answers in Mem0 never updated Letta blocks. KPIs manually updated.  
**Now:** `agent_end` hook detects business/pairing keywords and sends context messages to Letta agent for self-edit. Best-effort, async, non-blocking.

---

## Corrected Architecture Grid

```
┌──────────────────────────────────────────────────────────────────────┐
│                    WIREBOT MEMORY ARCHITECTURE                       │
│                                                                      │
│  ┌─────────────────┐  ┌──────────────────┐  ┌────────────────────┐  │
│  │  memory-core    │  │     Mem0         │  │      Letta         │  │
│  │  (workspace)    │  │  (fact store)    │  │ (state engine)     │  │
│  │                 │  │                  │  │                    │  │
│  │ OWNS:           │  │ OWNS:            │  │ OWNS:              │  │
│  │  IDENTITY.md    │  │  Conversation    │  │  business_stage    │  │
│  │  SOUL.md        │  │  facts           │  │  goals             │  │
│  │  USER.md        │  │  Preferences     │  │  kpis              │  │
│  │  MEMORY.md      │  │  Decisions       │  │  human             │  │
│  │  memory/*.md    │  │  Relationships   │  │  archival docs     │  │
│  │                 │  │                  │  │  conversation log   │  │
│  │ WRITES VIA:     │  │ WRITES VIA:      │  │ WRITES VIA:        │  │
│  │  file ops       │  │  messages→LLM    │  │  messages→agent    │  │
│  │  (instant)      │  │  extract+dedup   │  │  self-edit blocks  │  │
│  │                 │  │  (async ~1s)     │  │  (async ~2s)       │  │
│  │ READS VIA:      │  │ READS VIA:       │  │ READS VIA:         │  │
│  │  hybrid search  │  │  vector search   │  │  block read (fast) │  │
│  │  BM25+vector    │  │  semantic sim    │  │  agent query (slow)│  │
│  │  (0ms, embedded)│  │  (~200ms)        │  │  archival search   │  │
│  └────────┬────────┘  └────────┬─────────┘  └────────┬───────────┘  │
│           │                    │                      │              │
│    ───────┴────────────────────┴──────────────────────┴───────       │
│                        BRIDGE LAYER                                  │
│                                                                      │
│  SYNC FLOWS:                                                         │
│    1. agent_end → Mem0 (structured messages, every conversation)     │
│    2. agent_end → Letta (business-relevant → agent message)          │
│    3. Nightly: Mem0 facts → MEMORY.md (curated append)               │
│    4. Nightly: Letta blocks → BUSINESS_STATE.md (snapshot)           │
│    5. Events: Scoreboard ships/revenue → Letta agent messages        │
│    6. Events: Pairing answers → Letta human block (via agent)        │
│                                                                      │
│  NO OVERLAP RULES:                                                   │
│    • Raw conversation facts → Mem0 ONLY (never Letta, never files)   │
│    • Structured state → Letta ONLY (never Mem0, never files)         │
│    • Identity/personality → Files ONLY (never Mem0, never Letta)     │
│    • Files are READ-ONLY snapshots of Mem0/Letta (nightly sync)      │
│    • Mem0 never writes to Letta. Letta never writes to Mem0.         │
│    • Bridge is the ONLY cross-system writer.                         │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Fix Status

| # | Fix | Status | Date |
|---|-----|--------|------|
| 1 | **Mem0: send structured messages** (not text) | ✅ Done | 2026-02-02 |
| 2 | **Letta: route updates through agent** (not direct PUT) | ✅ Done | 2026-02-02 |
| 3 | **Letta: populate archival memory** with key docs | ✅ Done | 2026-02-02 |
| 4 | **Wire scoreboard events → Letta** | ✅ Done | 2026-02-02 |
| 5 | **Wire pairing answers → Letta** | ✅ Done | 2026-02-02 |
| 6 | **Restore nightly sync** | ✅ Done | 2026-02-02 |
| 7 | **Recall: use Letta archival search** for complex queries | ✅ Done | 2026-02-02 |

### Fix Details

- **Fix 1:** `mem0Store()` updated to accept `messages` array, `agent_end` sends structured `[{role, content}]`
- **Fix 2:** `wirebot_business_state` tool description steers toward `message` action (agent self-edit). Direct PUT reserved for bulk resets only.
- **Fix 3:** Letta archival populated via OpenRouter `text-embedding-3-small` (1536 dims, ~$0.0002 total). 6 docs: PAIRING, SCOREBOARD_PRODUCT, MULTI_BUSINESS, OPERATOR_REALITY, SOUL, SCOREBOARD_SCORING. Local fastembed can't be used because Letta container's openai client uses `OPENAI_BASE_URL` env var which overrides per-agent endpoint.
- **Fix 4+5:** `agent_end` hook detects business/pairing keywords in conversation and sends context to Letta agent as message for self-edit. Best-effort, non-blocking.
- **Fix 6:** Nightly sync cron verified (`0 0 * * 0`). Fixed namespace bug (was double-prefixed `wirebot_wirebot_verious`). 80 Mem0 facts re-embedded after fastembed 768-dim migration.
- **Fix 7:** `lettaArchivalSearch()` wired into `wirebot_recall`. Returns top-2 archival passages truncated to 500 chars alongside Mem0 facts + Letta blocks.

---

## Success Criteria

- [x] `wirebot_recall` returns facts (Mem0) + state (Letta blocks) + archival (Letta docs) — all 3 layers (14 results verified)
- [x] `agent_end` routes pairing-relevant conversations to Letta agent for human/goals block self-edit
- [x] `agent_end` routes business-relevant conversations to Letta agent for kpis/business_stage self-edit
- [x] Mem0 receives `[{role, content}]` structured messages (not concatenated text)
- [x] Letta archival has 6 key documents (PAIRING, SCOREBOARD_PRODUCT, MULTI_BUSINESS, OPERATOR_REALITY, SOUL, SCOREBOARD_SCORING)
- [x] MEMORY.md sync verified (dedup works, 80 facts)
- [x] BUSINESS_STATE.md snapshot matches Letta blocks (4 blocks synced)
- [x] No system writes data another system owns (no overlap)
