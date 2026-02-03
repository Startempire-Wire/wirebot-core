# Memory Approval System — Design Doc

## Problem Statement

When ingesting documents (Obsidian vault, 800+ files) or extracting facts from conversations,
the current system has no quality gate:

1. **No approval queue** — facts go straight to Mem0 without review
2. **No provenance** — can't trace where a fact came from
3. **Naive extraction** — "Providence" hospital misread as location
4. **No confidence scoring** — all facts treated equally
5. **No correction propagation** — fixing in one place doesn't fix all layers

## Proposed Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        DOCUMENT INGESTION                                │
│  Obsidian Vault │ Conversation │ Bootstrap │ External APIs              │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    INTELLIGENT EXTRACTION                                │
│                                                                          │
│  1. Context window (surrounding text, not just keywords)                │
│  2. Entity disambiguation (Providence = hospital vs city?)              │
│  3. Confidence scoring (0.0 - 1.0)                                      │
│  4. Source tracking (file:line, conversation turn, API response)        │
│  5. Category inference (identity, preference, fact, relationship)       │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    PENDING QUEUE (SQLite)                               │
│                                                                          │
│  memory_queue table:                                                     │
│    id, memory_text, source_type, source_file, source_context,          │
│    confidence, status (pending|approved|rejected|corrected),            │
│    correction, created_at, reviewed_at                                   │
│                                                                          │
│  Auto-approve rules:                                                     │
│    - confidence >= 0.95 AND source_type = 'conversation'                │
│    - is correction to previously-approved fact                          │
│                                                                          │
│  Always-queue rules:                                                     │
│    - confidence < 0.7                                                    │
│    - source_type = 'document' (bulk ingestion)                          │
│    - entity mentions (names, locations, organizations)                  │
└────────────────────────────┬────────────────────────────────────────────┘
                             │
                    ┌────────┴────────┐
                    │                 │
                    ▼                 ▼
        ┌───────────────┐   ┌───────────────────────┐
        │ AUTO-APPROVE  │   │ APPROVAL UI           │
        │ (high conf)   │   │                       │
        │               │   │ - List pending items  │
        │ Straight to   │   │ - Show source context │
        │ Mem0/MEMORY   │   │ - Approve/Reject/Edit │
        │               │   │ - Bulk operations     │
        └───────────────┘   └───────────────────────┘
                                      │
                                      ▼
                        ┌─────────────────────────┐
                        │ MEMORY BRIDGE           │
                        │                         │
                        │ - Mem0 (cross-surface)  │
                        │ - MEMORY.md (workspace) │
                        │ - Letta (if relevant)   │
                        └─────────────────────────┘

```

## Intelligent Extraction (LLM-Powered)

Instead of naive keyword extraction, use structured prompts:

```
Given this document excerpt from {source_file}:
---
{context_window}
---

Extract factual claims about the USER (not general information).
For each claim, provide:
1. The claim text (concise, first-person where appropriate)
2. Confidence (0.0-1.0) based on:
   - Explicit statement (0.9+) vs inference (0.5-0.8)
   - Recency (recent > old)
   - Consistency with other docs
3. Category: identity | preference | fact | relationship | business | temporal
4. Entities mentioned (names, places, orgs) — flag for disambiguation

Example output:
{
  "claims": [
    {
      "text": "User is located in Corona, CA",
      "confidence": 0.4,
      "category": "identity",
      "entities": ["Corona, CA"],
      "reasoning": "Mentioned SoCal hospitals, but unclear if user lives there or was researching"
    }
  ]
}
```

## Auto-Approve Thresholds

| Condition | Auto-Approve? |
|-----------|---------------|
| confidence >= 0.95 AND no entities | ✅ Yes |
| confidence >= 0.9 AND source = 'direct_statement' | ✅ Yes |
| confidence < 0.7 | ❌ No, queue for review |
| contains location/person/org entity | ❌ No, queue for disambiguation |
| source = 'bulk_document_scan' | ❌ No, queue for review |
| is correction to rejected fact | ❌ No, queue for review |

## Approval UI (Scoreboard PWA)

### Settings → Memory Review Tab

```
┌─────────────────────────────────────────────────────────────────┐
│ 🧠 Memory Review                              [Pending: 47]     │
├─────────────────────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ "User is located in Providence, RI"                         │ │
│ │                                                             │ │
│ │ 📁 Source: USER.md (bootstrap)                             │ │
│ │ 📊 Confidence: 0.4 (inferred from timezone mention)        │ │
│ │ ⚠️  Entity: Providence, RI (location — needs verification)  │ │
│ │                                                             │ │
│ │ Context:                                                    │ │
│ │ "...operates on California time...Providence, RI area..."  │ │
│ │                                                             │ │
│ │ [✅ Approve] [❌ Reject] [✏️ Correct]                        │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ ┌─────────────────────────────────────────────────────────────┐ │
│ │ "User prefers diplomatic communication"                     │ │
│ │                                                             │ │
│ │ 📁 Source: conversation (2026-02-01)                       │ │
│ │ 📊 Confidence: 0.92                                         │ │
│ │                                                             │ │
│ │ [✅ Approve] [❌ Reject] [✏️ Correct]                        │ │
│ └─────────────────────────────────────────────────────────────┘ │
│                                                                 │
│ [Approve All High-Confidence] [Reject All Low-Confidence]      │
└─────────────────────────────────────────────────────────────────┘
```

## Correction Propagation

When user corrects a fact:

1. Store correction with link to original
2. Update Mem0 (POST /v1/store with "CORRECTION:" prefix)
3. Update MEMORY.md (sed replacement)
4. If business-related, send message to Letta agent
5. Create audit trail in memory_queue table

## Implementation Phases

### Phase 1: Queue Infrastructure (done)
- [x] memory_queue SQLite table
- [x] GET/POST /v1/memory/queue API
- [x] approve/reject/correct actions

### Phase 2: Approval UI
- [ ] Memory Review tab in Settings
- [ ] List pending items with source context
- [ ] Approve/Reject/Correct buttons
- [ ] Bulk operations

### Phase 3: Intelligent Extraction
- [ ] LLM-powered extraction prompt
- [ ] Confidence scoring
- [ ] Entity disambiguation
- [ ] Auto-approve rules

### Phase 4: Ingestion Integration
- [ ] Wire vault scan to queue instead of direct Mem0
- [ ] Wire conversation extraction to queue
- [ ] Wire bootstrap facts to queue

### Phase 5: Correction Propagation
- [ ] Multi-layer update on correction
- [ ] Audit trail
- [ ] Undo capability
