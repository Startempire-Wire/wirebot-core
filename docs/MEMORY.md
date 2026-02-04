# Wirebot Memory Architecture (OpenClaw + Letta + Mem0)

> **OpenClaw memory is built‑in. Wirebot adds structured state + cross‑surface sync.**

---

## Principle

OpenClaw already provides a full memory system (markdown + hybrid search + compaction).
Wirebot **layers on top**:

- **Letta** for structured business state
- **Mem0** for cross‑surface recall (browser → agents, multi‑surface sync)

---

## OpenClaw Memory (Built‑In)

### Two‑Layer Markdown Store

```
~/clawd/
├── MEMORY.md              # Long‑term curated knowledge
└── memory/
    ├── 2026-01-26.md      # Daily append‑only log
    ├── 2026-01-25.md
    └── ...
```

- **Layer 1 (daily logs)**: append‑only notes throughout the day
- **Layer 2 (MEMORY.md)**: curated, long‑term decisions + preferences

### Auto‑Indexing Pipeline

- File watcher (Chokidar) on `MEMORY.md` + `memory/**/*.md`
- Debounced ~1.5s to batch writes
- Chunking: ~400 tokens with ~80 token overlap
- Index storage: SQLite + **sqlite‑vec** (vector) + **FTS5** (keyword)

### Hybrid Search (Default)

- Semantic (vector) + keyword (BM25)
- Weighted blend:

```
finalScore = (0.7 * vectorScore) + (0.3 * textScore)
```

### Compaction

- When context is near limit, old turns are summarized
- **Pre‑compaction flush** writes recent info to daily log before summary

### Multi‑Agent Isolation

- Each agent has its own workspace + SQLite index
- Keyed by `agentId + workspaceDir`
- No cross‑agent memory by default

---

## Wirebot Additions (On Top of OpenClaw)

### Letta — Structured Business State

Letta stores **authoritative business context**:
- Stage (Idea → Launch → Growth)
- Goals + KPIs
- Checklists
- Preferences

This is not conversational memory. It is **structured, queryable state**.

### Mem0 — Cross‑Surface Sync

Mem0 is **not a replacement** for OpenClaw memory. Use it to:
- Sync browser chats (OpenMemory extension → Mem0 → Wirebot)
- Share memories across devices/surfaces
- Store long‑term vector/graph memory outside OpenClaw

---

## Recommended Stack

| Layer | System | Purpose |
|------|--------|---------|
| **Core memory** | **OpenClaw built‑in** | Daily logs + long‑term memory + hybrid search |
| **Structured state** | **Letta** | Business context + accountability |
| **Cross‑surface** | **Mem0** | Browser sync + shared vector/graph memory |

---

## Where to Write (Rules of Thumb)

- **Daily log** (`memory/YYYY-MM-DD.md`): conversation notes, quick facts
- **MEMORY.md**: durable preferences, key decisions, long‑term constraints
- **Letta**: stage, goals, KPIs, checklists, structured records
- **Mem0**: browser‑captured context, external system data, shared insights

---

## Memory Tool Slot

Default: **OpenClaw memory‑core** (built‑in system).

If you want OpenClaw’s memory tools to point at Mem0 instead:

```json5
plugins: { slots: { memory: "memory-mem0" } }
```

Otherwise keep **memory‑core** and call Mem0 via Wirebot skills.

---

## Data Scope

Per‑user isolation across all layers:

- OpenClaw agent + workspace
- Mem0 namespace: `memory_<user_id>` (or `wirebot_<user_id>`)
- Letta agent: `agent_<user_id>`

---

## Retention

- OpenClaw: local, configurable
- Letta: long‑term until deletion
- Mem0: long‑term, GDPR deletable

---

## See Also

- [CLAWDBOT_MEMORY_DEEP_DIVE.md](./CLAWDBOT_MEMORY_DEEP_DIVE.md) — OpenClaw memory internals
- [MEM0_PLUGIN.md](./MEM0_PLUGIN.md) — Mem0 plugin details
- [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md) — Structured state layer
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Full memory stack diagram
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Memory system status (what's deployed)
