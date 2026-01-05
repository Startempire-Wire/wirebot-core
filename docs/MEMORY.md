# Wirebot Memory Architecture

> **Structured state + summaries, not infinite chat memory.**

---

## Principle

Wirebot should not be "infinite chat memory."

It should be **structured state + summaries**, with optional retrieval.

---

## Memory Layers

### Layer 1 — Structured State (Authoritative)

Per founder + per workspace:

| Data | Description | Storage |
|------|-------------|---------|
| Business Profile | Industry, offer, audience, constraints | MariaDB |
| Current Stage | Idea / Launch / Growth | MariaDB |
| Checklist Progress | Items, priority, due dates | MariaDB |
| Goals + KPIs | MRR, leads, conversion targets | MariaDB |
| Preferences | Tone, cadence, constraints | MariaDB |

This is the **source of truth** for business context.

```sql
-- Example: Workspace profile
{
  "industry": "SaaS",
  "offer": "Project management for agencies",
  "audience": "Creative agencies 10-50 employees",
  "constraints": ["Bootstrap", "Solo founder", "Part-time"],
  "monthly_revenue_target": 10000,
  "current_mrr": 2400
}
```

### Layer 2 — Rolling Summaries (Operational Memory)

Updated periodically by scheduled jobs and after major conversations.

| Summary Type | Description | Update Frequency |
|--------------|-------------|------------------|
| Weekly Operating Summary | What happened this week | Weekly (Sunday) |
| Current Blockers | Active obstacles | On change |
| Current Plan | Next priorities | On change |
| Decisions Log | Key decisions made | On decision |
| Patterns | Observed behavioral patterns | Monthly |

Storage: MariaDB `summaries` table (text blobs with timestamps)

```sql
-- Example: Weekly summary
{
  "week": "2026-W01",
  "highlights": [
    "Launched beta to 10 users",
    "Fixed critical auth bug",
    "Started content calendar"
  ],
  "blockers": [
    "Waiting on Stripe approval"
  ],
  "next_week_focus": [
    "User interviews",
    "Landing page copy"
  ],
  "mood": "optimistic"
}
```

### Layer 3 — Transcript Store (Audit / Trace)

Full conversation logs stored, but **not always injected** into context.

Used for:
- Support and debugging
- "Show me what I said last month"
- Compliance / data export
- Training data (with consent)

Storage: MariaDB `transcripts` table

**Not the primary prompt input.** Only referenced when explicitly needed.

### Layer 4 — Optional Vector Retrieval (Enhancement)

Use vector store for:
- Semantic recall of past discussions
- Finding "similar past decisions"
- Pulling "user's prior answers" without token bloat
- Pattern matching across long timeframes

**Do NOT rely on vector store for core operations.** It's enhancement, not foundation.

Technology: Qdrant (containerized, later phase)

---

## Scoping Rules

Memory is scoped at three levels:

### 1. Founder Level
- Cross-workspace patterns
- Global preferences
- Billing/identity data

### 2. Workspace Level
- Business context
- Checklists and goals
- Rolling summaries
- Stage progression

### 3. Session Level
- Transient context window
- Current conversation thread
- Surface-specific state

---

## Context Window Construction

When processing a request, the Gateway builds context:

```
[System prompt]
[Founder profile summary]
[Workspace context: stage, profile, current blockers]
[Rolling summary: this week's operating context]
[Recent session messages (last N)]
[Optional: retrieved relevant chunks from vector store]
[Current user input]
```

**Token budget:** ~8K-16K tokens for context, leaving room for response.

---

## Time-Windowing Strategy

| Data Type | Window | Injection Rule |
|-----------|--------|----------------|
| Business profile | Always | Always injected |
| Current stage | Always | Always injected |
| Rolling summaries | Latest | Weekly summary always, others on-demand |
| Session messages | Last 10-20 | Always injected |
| Transcripts | On-demand | Only when explicitly referenced |
| Vector chunks | On-demand | Retrieved by semantic similarity |

---

## Memory Update Triggers

### Automatic (Scheduled Jobs)
- **Daily:** Update "current blockers" from recent conversations
- **Weekly:** Generate weekly operating summary
- **Monthly:** Pattern analysis and recalibration

### Explicit (User Actions)
- Complete checklist item → update progress
- Set goal → add to goals table
- "Remember this" → store as decision/note

### Implicit (Conversation Analysis)
- Detect stage transition → prompt for confirmation
- Detect repeated blocker → surface in summary
- Detect decision made → log to decisions

---

## Mode 3 (Sovereign) Memory

Completely separate:
- Separate database/schema (`wirebot_sovereign`)
- Separate Redis keyspace (`wbs:`)
- Separate encryption keys
- No cross-mode retrieval by default (explicit opt-in only)

---

## Data Lifecycle

### Retention Defaults
- Structured state: Indefinite (until deleted)
- Rolling summaries: 1 year
- Transcripts: 90 days (configurable)
- Vector embeddings: Synced with source data

### Export
- Full data export available (GDPR compliance)
- JSON format with all structured data
- Transcript archive

### Deletion
- "Delete my data" removes:
  - All structured state
  - All summaries
  - All transcripts
  - All vector embeddings
- Audit log entry retained (anonymized)

---

## Implementation Notes

### Database Indexes

```sql
-- Fast workspace lookup
CREATE INDEX idx_workspaces_founder ON workspaces(founder_id);

-- Fast summary retrieval
CREATE INDEX idx_summaries_workspace_type ON summaries(workspace_id, type);
CREATE INDEX idx_summaries_generated ON summaries(generated_at);

-- Transcript search
CREATE INDEX idx_transcripts_workspace ON transcripts(workspace_id);
CREATE INDEX idx_transcripts_created ON transcripts(created_at);
```

### Redis Caching

```
wb:profile:{workspace_id}     # Cached workspace profile (5 min TTL)
wb:summary:{workspace_id}     # Cached latest summaries (1 hour TTL)
wb:session:{session_id}       # Active session state (24 hour TTL)
```

### Memory Cleanup Job

```javascript
// Run daily
async function cleanupOldData() {
  // Archive old transcripts
  await archiveTranscriptsOlderThan(90);
  
  // Compact old summaries
  await compactSummariesOlderThan(365);
  
  // Clear expired sessions
  await clearExpiredSessions();
}
```

---

## Principle Reminder

> **Memory exists to serve the founder's clarity, not to impress with recall.**

Keep it:
- Structured
- Relevant
- Scoped
- Actionable
