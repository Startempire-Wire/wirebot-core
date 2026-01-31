# Clawdbot Memory Deep Dive

> **Built‑in memory system: Markdown source → hybrid index → recall tools.**

---

## Overview

Clawdbot memory is a **two‑layer Markdown store** plus a **hybrid search index**.
Memory is plain files in the agent workspace. Indexes are derived data.

---

## Memory Layout (Source of Truth)

```
~/clawd/
├── MEMORY.md              # Long‑term curated knowledge
└── memory/
    ├── 2026-01-26.md      # Daily append‑only log
    ├── 2026-01-25.md
    └── ...
```

- **Daily logs**: append‑only, written throughout the day
- **MEMORY.md**: curated long‑term decisions, preferences, lessons

---

## Index Layout (Derived Data)

```
~/.clawdbot/memory/
├── main.sqlite
└── work.sqlite
```

- One SQLite index per agent
- Indexed by `agentId + workspaceDir`

---

## Auto‑Indexing Pipeline

```
File saved (MEMORY.md or memory/YYYY-MM-DD.md)
  → File watcher (Chokidar)
  → Debounce (~1.5s)
  → Chunking (~400 tokens, 80 token overlap)
  → Embeddings + keyword index
  → SQLite index (sqlite‑vec + FTS5)
```

---

## Hybrid Search (Default)

Two strategies run in parallel:
- **Vector search** (semantic)
- **BM25** (keyword)

Weighted blend:

```
finalScore = (0.7 * vectorScore) + (0.3 * textScore)
```

Why hybrid?
- Vectors catch meaning
- BM25 catches exact tokens (IDs, dates, names)

---

## Memory Read Tools

Clawdbot exposes memory tools (example shown):

```json
{
  "name": "memory_search",
  "parameters": {
    "query": "What did we decide about the API?",
    "maxResults": 6,
    "minScore": 0.35
  }
}
```

---

## Compaction

When the context window is near limit:
- Old turns are summarized
- **Pre‑compaction flush** writes recent info to daily log first

---

## Multi‑Agent Isolation

- Each agent has its own workspace + SQLite index
- No cross‑agent memory by default
- Soft isolation (workspace is default cwd, not hard sandbox)

---

## What Wirebot Adds

Wirebot uses Clawdbot memory as‑is and adds:
- **Letta** for structured business state
- **Mem0** for cross‑surface sync (browser → agents)

See [MEMORY.md](./MEMORY.md) for the full Wirebot stack.

---

## See Also

- [MEMORY.md](./MEMORY.md) — Full Wirebot memory stack
- [MEM0_PLUGIN.md](./MEM0_PLUGIN.md) — Cross-surface sync layer
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [OPERATIONS.md](./OPERATIONS.md) — Service management (where memory lives)
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Memory-related issues
