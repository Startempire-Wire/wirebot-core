---
name: wirebot-memory
description: Memory operations for Wirebot. Use when recalling decisions, storing durable context, syncing browser memory, or updating Letta/Mem0/Clawdbot memory.
---

# Wirebot Memory

## Purpose

Coordinate memory across:
- Clawdbot core memory (daily logs + MEMORY.md)
- Letta structured state
- Mem0 cross-surface memory

## Use Cases

- "What did we decide before?"
- Store durable preferences or decisions
- Sync browser chat context into memory
- Prune or consolidate memory items

## Workflow

1) Recall: check Clawdbot memory first
2) If missing: query Mem0 and Letta
3) Decide destination:
   - Daily log for recent notes
   - MEMORY.md for durable facts
   - Letta for structured business state
   - Mem0 for cross-surface context
4) Store with clear tags (date, source, type)

## Write Rules

- Daily log: `memory/YYYY-MM-DD.md`
- Long-term: `MEMORY.md`
- Letta: stage/goals/KPIs/checklists
- Mem0: browser-derived context + shared insights

## Output Template

```
Memory Update
- Source:
- Type: decision | preference | context | blocker
- Destination: daily | long-term | letta | mem0
- Summary:
```

## Notes

- Do not overwrite durable facts without confirmation.
- Prefer Clawdbot memory for conversational recall.
