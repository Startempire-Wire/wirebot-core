# Focusa × Wirebot — Complete System Map

> **How the full Focusa cognitive governance system operates
> on top of the Wirebot Memory Bridge infrastructure.**
>
> Every field, every enum, every threshold, every invariant is verbatim
> from the Focusa spec documents. Source doc cited for every section.

---

## 1. Focusa Architecture Overview

Source: `01-architecture-overview.md`

Focusa is a local cognitive proxy that:
- intercepts prompts and responses
- manages focus state and context fidelity
- injects minimal structured context and references
- maintains lightweight memory
- exposes local observability (API/GUI/CLI)

Focusa MUST remain backend-agnostic. It cannot depend on internal APIs of
Letta or other harnesses. It implements adapter "drivers" that speak generic
I/O protocols (stdin/stdout, HTTP).

### 5 Planes

1. **Cognitive Control Plane** — Focus Stack (HEC), Focus Gate
2. **Context Fidelity Plane** — ASCC, ECS, CLT
3. **Memory Plane** — Semantic memory, Procedural memory
4. **Background Cognition Plane** — Worker pipeline (async)
5. **Interfaces** — CLI, Local API, Menubar UI

### Determinism & Safety Rules

1. Focus Gate is **advisory** only.
2. Prompt Assembly is **deterministic** given state + inputs.
3. Any large data MUST be externalized when above threshold.
4. No component may introduce blocking latency to request/response path beyond a strict budget.
5. All state mutations must be logged as events.

### Performance Budgets (MVP)

- Hot path (proxy request processing): **< 20ms** additional overhead on prompt assembly on typical machines.
- Background tasks: async; never block hot path.
- Storage: local file store operations should be batched where possible.

**Bridge note:** Focusa spec requires <20ms hot path. The bridge architecture
uses HTTP calls to Mem0 (~200ms) and Letta (~100ms). This is a fundamental
architecture difference vs the spec's local Rust daemon design. The bridge
must either: (a) pre-cache these reads before the hot path, (b) accept
higher latency with the understanding that the spec's 20ms target applies
to local state only, or (c) run reads in parallel to stay under ~300ms total.

### Data Persistence (MVP)

Must survive daemon restart:
- focus stack state
- ASCC checkpoints
- ECS artifacts + index
- semantic/procedural memory
- event log (bounded)

---

## 2. Core Reducer

Source: `core-reducer.md`

### Contract

```
reduce(state: FocusaState, event: FocusaEvent) -> ReductionResult

ReductionResult {
  new_state: FocusaState
  emitted_events: Vec<FocusaEvent>
}
```

### Canonical State Shape

```
FocusaState {
  session: Option<SessionState>
  focus_stack: FocusStack
  focus_gate: FocusGateState
  reference_index: ReferenceIndex
  memory: ExplicitMemory
  version: u64
}
```

⚠️ Conversation history is NEVER part of FocusaState.

### Canonical Event Types (15 total)

```
enum FocusaEvent {
  // Session Lifecycle
  SessionStarted { session_id }
  SessionRestored { session_id }
  SessionClosed { reason }

  // Focus Stack
  FocusFramePushed { frame_id, beads_issue_id, title, goal }
  FocusFrameCompleted { frame_id, completion_reason }
  FocusFrameSuspended { frame_id, reason }

  // Focus State
  FocusStateUpdated { frame_id, delta: FocusStateDelta }

  // Intuition → Gate
  IntuitionSignalObserved { signal_id, signal_type, severity, related_frame_id? }
  CandidateSurfaced { candidate_id, description, pressure, related_frame_id? }
  CandidatePinned { candidate_id }
  CandidateSuppressed { candidate_id, scope }

  // Reference Store
  ArtifactRegistered { artifact_id, artifact_type, summary, storage_uri }
  ArtifactPinned { artifact_id }
  ArtifactGarbageCollected { artifact_id }

  // Errors
  InvariantViolation { invariant, details }
}
```

### Global Invariants (Checked Pre/Post)

```
INVARIANT: At most one active Focus Frame exists
INVARIANT: Every Focus Frame maps to a Beads issue
INVARIANT: Focus State sections always exist
INVARIANT: Intuition Engine cannot mutate focus
INVARIANT: Focus Gate is advisory only
INVARIANT: Artifacts are immutable once registered
INVARIANT: Conversation never mutates cognition
```

### Reducer Guarantees

- Deterministic
- Replayable from event log
- Crash-safe
- Testable in isolation
- Free of side effects

### Reducer Algorithm

The full algorithm handles each of the 15 event types. Key behaviors:

- `FocusFramePushed`: assert beads_issue_exists, suspend current active frame, push new frame, emit FocusFrameActivated
- `FocusFrameCompleted`: assert active_frame_id matches, assert completion_reason exists, complete frame, restore parent, emit FocusFrameArchived
- `FocusFrameSuspended`: assert active_frame_id matches, suspend with reason, emit confirmed
- `FocusStateUpdated`: assert active_frame_id matches, apply_incremental_focus_state_delta, emit FocusStateCommitted
- `IntuitionSignalObserved`: aggregate_signal into focus_gate, emit IntuitionSignalAggregated
- `CandidateSurfaced`: upsert_candidate with all fields, emit CandidateVisible
- `ArtifactRegistered`: register in reference_index, emit ArtifactIndexed

State version incremented on every successful reduction: `state.version += 1`

**Canonical rule:** If a cognition change cannot be expressed as a reducer event, it does not belong in Focusa.

---

## 3. Daemon Runtime

Source: `G1-detail-03-runtime-daemon.md`

### AppState (Full)

```
AppState {
  focus_stack: FocusStackState
  focus_gate: FocusGateState
  ascc: AsccState
  ecs: EcsState
  memory: MemoryState
  workers: WorkerState
  adapters: AdapterState        // minimal bookkeeping
  metrics: MetricsState         // counters, last activity timestamps
  current_session_id: SessionId // UPDATE: session identity
  sessions: HashMap<SessionId, SessionMeta>  // UPDATE
}
```

### SessionMeta (from UPDATE)

```
SessionMeta {
  session_id
  created_at
  adapter_id
  workspace_id?               // Optional, string or hash (cwd hash)
  status: active | closed
}
```

### Session Identity Invariants

1. All state mutations must include session_id
2. Reducer rejects cross-session writes
3. Events without session_id are invalid

### Process Model

- Single daemon process
- One Tokio runtime
- State mutated via internal reducer (event-driven)
- Concurrency: single owner task with `mpsc` command channel (actor model preferred)

### Command Handling (Action Enum)

```
Action enum:
  // Focus
  PushFrame, PopFrame, SetActiveFrame
  // Gate
  IngestSignal, SurfaceCandidate, SuppressCandidate
  // ASCC
  UpdateCheckpointDelta
  // ECS
  StoreArtifact, ResolveHandle
  // Memory
  UpsertSemantic, ReinforceRule, DecayTick
  // Worker
  WorkerEnqueue, WorkerComplete
```

### Event Log (MVP)

Append-only JSONL. Every state mutation emits an event with:
- id (monotonic or UUIDv7)
- timestamp
- type
- payload
- correlation_id (request/turn id)
- origin (cli/gui/adapter/worker)

Bounded: keep last N MB or last N days (config). Older logs can be compacted (non-MVP).

### Background Scheduling

Single periodic tick every `T` seconds:
- run decay tick
- flush pending persistence batch
- check worker queue

Workers must not block hot path.

### Startup

1. load config
2. ensure directories exist
3. load state snapshots
4. open event log
5. start API server
6. start worker scheduler

### Shutdown

- flush persistence
- stop API
- close event log cleanly

### Persistence

Local directory (default: `~/.focusa/`). JSON files for state + append-only JSONL event log.
ECS artifacts stored as files under `~/.focusa/ecs/`.

**Bridge mapping:** Persistence directory → `/data/wirebot/focusa-state/`

---

## 4. Focus Stack (HEC)

Source: `03-focus-stack.md`, `G1-detail-05-focus-stack-hec.md`

### Core Invariants

1. Exactly one active Focus Frame exists at any time
2. Every Focus Frame has a concrete intent
3. Every Focus Frame maps to a Beads issue
4. Frames are entered and exited explicitly
5. Completed frames are archived, not forgotten

### Data Model

#### FrameId
String UUIDv7 (preferred) or ULID.

#### FrameStatus

```
enum FrameStatus {
  active      // only one frame can be active
  paused      // parent frames on stack path when child is active
  completed
  archived
}
```

#### FrameRecord

```
FrameRecord {
  id: FrameId
  parent_id: Option<FrameId>
  created_at: ts
  updated_at: ts
  status: FrameStatus
  title: String                         // short
  goal: String                          // one sentence
  tags: Vec<String>                     // optional
  priority_hint: Option<String>         // optional; not numeric
  ascc_checkpoint_id: Option<String>    // anchor pointer; see ASCC
  stats: FrameStats                     // optional
  handles: Vec<HandleRef>              // references used in this frame
  constraints: Vec<String>             // optional, short
}
```

#### FrameStats (MVP minimal)

```
FrameStats {
  turn_count: u64
  last_turn_id: Option<String>
  last_token_estimate: Option<u32>
}
```

#### FocusStackState

```
FocusStackState {
  root_id: FrameId
  active_id: FrameId
  frames: HashMap<FrameId, FrameRecord>
  stack_path_cache: Vec<FrameId>       // derived, cached for fast reads
  version: u64                          // monotonic; increments on mutation
}
```

### Operations

#### PushFrame

Creates a new child frame under the current active frame.

Inputs: `title`, `goal`, `constraints?`, `tags?`

Rules:
- New frame becomes `active`
- Previous active becomes `paused`
- Emit events: `focus.frame_pushed`, `focus.active_changed`

#### PopFrame (Complete)

Returns focus to parent frame.

Rules:
- Current active frame status becomes `completed`
- Requires completion_reason
- Parent frame restores to `active`
- Emit events: `focus.frame_completed`, `focus.active_changed`

#### Completion Reasons (required)

```
enum CompletionReason {
  goal_achieved
  blocked
  abandoned
  superseded
  error
}
```

### Parent Context Rules

When assembling Focus State:
- Active frame is always included
- Parent frames contribute selectively: intent, decisions, constraints
- Artifacts from parent frames included only if referenced

### Invalid Operations (Forbidden)

- Multiple active frames
- Implicit frame switching
- Editing archived frames
- Frames without Beads linkage
- Skipping completion reasons

### Interaction with Other Components

- **Intuition Engine**: May observe frame duration; may emit time-based signals
- **Focus Gate**: May surface candidates related to inactive frames; never auto-resumes frames
- **Expression Engine**: Receives serialized Focus State derived from stack

**Bridge mapping:** Focus Stack state → `/data/wirebot/focusa-state/stack.json`

---

## 5. Focus State

Source: `06-focus-state.md`

### Core Invariants

1. Focus State is explicit and structured
2. Focus State is deterministic
3. Focus State is incrementally updated
4. Focus State is injected every turn
5. Focus State never inferred implicitly

### Required Sections

```
FocusState {
  intent
  decisions
  constraints
  artifacts          // references only
  failures
  next_steps
  current_state
}
```

Each section may be empty but must exist.

### Update Rules

- Only changed sections are updated (incremental)
- No full regeneration
- Anchored to frame lifecycle
- Contradictions must be logged
- Prior decisions preserved
- Resolution recorded explicitly

### Injection Policy

Every model invocation includes:
- serialized Focus State
- deterministic ordering
- bounded token budget

If budget exceeded:
- lower-priority sections truncated first
- truncation is explicit and logged

### Forbidden Behaviors

- Implicit summarization
- Silent overwrites
- Hidden inference
- Mixing conversation with state

**Bridge mapping:** Focus State per frame → `/data/wirebot/focusa-state/frames/<frame_id>.json`

---

## 6. Focus Gate

Source: `04-focus-gate.md`, `G1-detail-06-focus-gate.md`

### Core Invariants

1. The Focus Gate never mutates Focus State or Focus Stack
2. The Focus Gate never triggers actions
3. The Focus Gate only surfaces candidates
4. All surfaced items are explainable
5. Decay and pressure are deterministic

### Signal Model

```
Signal {
  id: SignalId
  ts: timestamp
  origin: "adapter" | "worker" | "daemon" | "cli" | "gui"
  kind: SignalKind
  frame_context: Option<FrameId>        // active at time of signal
  summary: String                        // short, <= 200 chars
  payload_ref: Option<HandleRef>        // if large; store in ECS
  tags: Vec<String>                     // optional
}
```

#### SignalKind (MVP — 9 values)

```
enum SignalKind {
  user_input
  assistant_output
  tool_output
  error
  warning
  artifact_changed
  repeated_pattern
  deadline_tick         // optional
  manual_pin            // user explicitly flags something
}
```

### Candidate Model

```
Candidate {
  id: CandidateId
  created_at
  updated_at
  kind: CandidateKind
  label: String                         // user-facing
  origin_signal_ids: Vec<SignalId>
  related_frame_id: Option<FrameId>
  state: CandidateState
  pressure: f32                         // internal
  last_seen_at
  times_seen: u32
  suppressed_until: Option<timestamp>
  resolution: Option<String>            // when completed/dismissed
}
```

#### CandidateKind (MVP — 5 values)

```
enum CandidateKind {
  suggest_push_frame
  suggest_resume_frame
  suggest_check_artifact
  suggest_fix_error
  suggest_pin_memory
}
```

#### CandidateState (4 values)

```
enum CandidateState {
  latent
  surfaced
  suppressed
  resolved
}
```

### Surface Pressure

A candidate's `pressure` increases with:
- persistence (repeated occurrence)
- goal alignment to active frame or near ancestors
- risk signals (errors, contradictions)
- novelty spikes

Pressure decreases with:
- suppression
- completion/resolution
- decay over time

### Focus Gate Algorithm (5 Steps)

**Step 1: Normalize signals**

On ingest:
- if `payload_ref` missing and payload is large → store to ECS → set payload_ref
- derive tags: error class, file path hints, tool name hints
- create fingerprint for dedupe: `hash(kind + normalized summary + frame_context + key tags)`

**Step 2: Candidate matching or creation**

If fingerprint matches an existing candidate:
- increment `times_seen`
- update `last_seen_at`
- increase `pressure` by `Δp`

Else:
- create new candidate with base pressure

**Step 3: Pressure update rules**

Pressure update uses additive factors:
- persistence, goal alignment, risk signals, novelty spikes (increase)
- suppression, completion, time decay (decrease)

**Step 4: Surfacing**

A candidate is surfaced when:
- pressure exceeds threshold
- not suppressed
- not already visible

Surfacing produces:
- CLI/UI notification
- event log entry

**Step 5: User actions**

User may:
- accept candidate → triggers frame operation
- suppress candidate → pressure to zero, audit trail retained
- pin candidate → bypass decay, persist across sessions
- ignore → natural decay

### Pinning

Pinned candidates:
- bypass decay
- persist across sessions
- must be explicitly unpinned
- do NOT force focus changes

### Suppression

- temporary, permanent, or per-session
- reduces pressure to zero
- retains audit trail

**Bridge mapping:** Focus Gate state → `/data/wirebot/focusa-state/gate.json`

---

## 7. Intuition Engine

Source: `05-intuition-engine.md`

### Core Invariants

1. Runs asynchronously only
2. Cannot block the hot path
3. Cannot mutate Focus State or Focus Stack
4. Emits signals, not commands
5. All signals are explainable

### Signal Sources (MVP)

- **Temporal**: Frame duration exceeds expected bounds; prolonged inactivity
- **Repetition**: Repeated errors, edits, tool invocations
- **Consistency**: Contradictory decisions; drift between stated intent and actions
- **Structural**: Deep stack nesting; frequent frame switching

### Signal Model

Each signal includes:
- signal_id
- signal_type
- severity
- related_frame_id
- metadata
- timestamp

Signals are ephemeral until promoted by Focus Gate.

### Aggregation

Signals aggregated by: type, related frame, time window.
Produces: cumulative pressure, summarized description.

### Emission

Aggregated signals emitted to Focus Gate.
- idempotent
- updates existing candidates where possible
- creates new candidates only when necessary

### Events Emitted

- `intuition.signal.created`
- `intuition.signal.updated`
- `intuition.signal.expired`

### Performance Constraints

- Zero blocking
- Bounded memory
- O(1) per signal processing target

### Forbidden Behaviors

- Writing memory
- Altering focus
- Triggering actions
- Injecting prompt content

**Bridge mapping:** Intuition Engine → in-memory in bridge plugin process

---

## 8. ASCC (Anchored Structured Context Checkpointing)

Source: `G1-07-ascc.md`

### Purpose

Maintains a persistent structured summary per focus frame that:
- replaces linear chat history in prompts
- updates incrementally using anchors
- preserves high-fidelity task continuity

### Checkpoint Schema

```
CheckpointRecord {
  frame_id: FrameId
  revision: u64
  updated_at
  anchor_turn_id: String                // last processed turn
  sections: AsccSections
  breadcrumbs: Vec<HandleRef>          // optional handles to external artifacts
  confidence: AsccConfidence            // optional; MVP can omit
  history: Vec<AsccDeltaMeta>          // bounded; optional
}
```

### AsccSections (10 fixed slots)

```
AsccSections {
  intent: String                        // 1–3 sentences
  current_focus: String                 // 1–3 sentences
  decisions: Vec<String>               // bullets; each <= 160 chars
  artifacts: Vec<ArtifactLine>         // typed lines
  constraints: Vec<String>             // short
  open_questions: Vec<String>
  next_steps: Vec<String>
  recent_results: Vec<String>          // short outputs or references
  failures: Vec<String>               // what failed and why
  notes: Vec<String>                   // misc, bounded
}
```

#### ArtifactLine

```
ArtifactLine {
  kind: "file" | "diff" | "log" | "url" | "handle" | "other"
  label: String
  ref: Option<HandleRef>
  path_or_id: Option<String>
}
```

Large artifact details stored in ECS and referenced via handle.

### Anchor Model

Anchors are turn IDs emitted by adapter. Each user prompt/assistant response
pair is a `turn_id`. ASCC only summarizes content up to the anchor.
`anchor_turn_id` in checkpoint = last applied turn.

### Update Pipeline (MVP)

Inputs for ASCC Update:
- `frame_id`
- `turn_id`
- `raw_user_input` (small)
- `assistant_output` (small or handle)
- `tool_outputs` (handles)
- `events` relevant to this frame
- optionally: extracted facts/preferences from worker

### Delta Summarization Rule

When a new turn arrives:
1. Determine "delta content" = only new items since last anchor
2. Summarize delta into structured slots
3. Merge into existing checkpoint using deterministic merge rules

Pluggable summarizer interface:
`Summarizer::summarize_delta(existing_checkpoint, delta_input) -> delta_sections`

### Merge Rules (Deterministic — All 10 Slots)

**`intent`:**
- if empty → set from delta
- else update only if delta contains explicit intent change marker

**`current_focus`:**
- update with the latest concise statement (replace)

**`decisions`:**
- append new unique bullets; dedupe by normalized text
- cap length: **default 30 items**

**`artifacts`:**
- append new artifact lines; dedupe by `(kind + path_or_id + label)`
- cap length: **default 50 lines**

**`constraints`:**
- append unique constraints
- cap: **30**

**`open_questions`:**
- append unique; if question is answered in delta, remove it (simple match heuristic)
- cap: **20**

**`next_steps`:**
- replace with latest suggested steps derived from active frame state
- cap: **15**

**`recent_results`:**
- keep last **10** results, newest first

**`failures`:**
- append failure bullets
- cap: **20**

**`notes`:**
- append
- cap: **20**; decay oldest first

**Always update after merge:**
- `revision += 1`
- `anchor_turn_id = turn_id`
- `updated_at = now`

**Always emit:** `ascc.delta_applied`

### Section Pinning (from UPDATE)

Any ASCC section may be marked pinned.

Pinned sections:
- cannot be dropped during prompt degradation
- are immune to slot-priority eviction

Section metadata: `pinned: bool`, `last_updated_at`

### Prompt Degradation Hooks (from UPDATE)

ASCC exposes `to_digest()` → ultra-compact fallback summary.
Used only when prompt budget cannot be satisfied.

Invariants:
- ASCC degradation is explicit
- ASCC never silently truncates pinned sections

### Prompt Serialization

Two serializers:
- `to_string_compact()`
- `to_messages_slots()`

Example (messages format):
```
FOCUS_FRAME: <title>
INTENT: ...
CURRENT_FOCUS: ...
DECISIONS: ...
ARTIFACTS: ...
CONSTRAINTS: ...
OPEN_QUESTIONS: ...
NEXT_STEPS: ...
```

**Bridge mapping:** ASCC live state → Letta blocks. ASCC durable snapshots → workspace `BUSINESS_STATE.md`. Nightly sync keeps them aligned.

---

## 9. ECS / Reference Store

Source: `07-reference-store.md`, `G1-detail-08-ecs.md`

### Core Invariants

1. Artifacts are never implicitly injected
2. Artifacts are referenced by handles only
3. Artifacts are immutable once written
4. Rehydration is explicit and auditable
5. Storage is session-scoped by default

### Handle Model

#### HandleId
UUIDv7 or hash-based id (sha256 prefix). Preferred: UUIDv7 for uniqueness + store sha256 in metadata.

#### HandleKind (MVP — 7 values)

```
enum HandleKind {
  log
  diff
  text
  json
  url
  file_snapshot
  other
}
```

#### HandleRef (prompt-safe)

```
HandleRef {
  id: HandleId
  kind: HandleKind
  label: String                 // short
  size: u64                     // bytes
  sha256: String                // hex
  created_at
}
```

Prompt representation: `[HANDLE:<kind>:<id> "<label>"]`

### Artifact Fields

```
Artifact {
  artifact_id
  type: diff | log | output | file | note
  summary: String               // ≤ 2 lines
  storage_uri
  created_at
  session_id
  pinned: bool
}
```

### Storage Layout

Root: `~/.focusa/ecs/`
- `objects/` — immutable content-addressed blobs
- `handles/` — metadata json by id
- `index.json` — small index (id → metadata)

### StoreArtifact Operation

Input: `kind`, `label`, `content_bytes` or `content_string`, optional `content_type`, `origin` + `correlation_id` + `frame_id`

Process:
1. compute sha256
2. generate id
3. write blob file
4. write metadata file
5. update index
6. emit `ecs.artifact_stored`

Return: HandleRef

### ResolveHandle Operation

Input: handle id
Output: metadata + content (streaming ok)

API: `GET /v1/ecs/resolve/:handle_id`
CLI: `focusa ecs cat <handle_id>`, `focusa ecs meta <handle_id>`

### Threshold Policy (MVP)

```
ecs.externalize_bytes_threshold = 8KB (default)
ecs.externalize_token_estimate_threshold = 800 tokens (default)
```

If either exceeded → externalize.

### Prompt Inclusion Policy

Include handles only in prompts. Explicit rehydration for content:
- `focusa ecs rehydrate <id> --max-tokens N`
- returns: first N tokens + trailing summary line with "truncated; fetch more if needed"

### Session Scoping (from UPDATE)

- Every handle includes `session_id`
- Cross-session resolution forbidden by default
- Explicit override required

### Human Pinning (from UPDATE)

Pinned handles:
- never garbage collected
- always shown in ECS listings
- surfaced preferentially in Focus Gate

### Garbage Collection (MVP Minimal)

- keep everything by default
- optional config: delete blobs older than N days
- ensure index consistency on startup (repair pass)

### Security Invariants

ECS must never: auto-inline content, fetch remote data, mutate stored artifacts.

**Bridge mapping:** ECS/Reference Store → workspace files in `/home/wirebot/clawd/**`. memory-core indexes via inotify. Rehydration → `wirebot_recall` search. Handle metadata → `/data/wirebot/focusa-state/ecs/index.json`.

---

## 10. Context Lineage Tree (CLT)

Source: `17-context-lineage-tree.md`

### Purpose

Answers: "What interaction paths existed, which were followed, which were abandoned, and how were they compacted over time?"

Does NOT answer: what the system believes, what the current goal is, what should be done next.

### Core Design Rules (7 — Non-Negotiable)

1. CLT is **append-only**
2. Nodes are **immutable once written**
3. CLT never mutates Focus State
4. Focus State references **exactly one CLT node** as its lineage head
5. Compaction inserts nodes — it never deletes history
6. Branches may be abandoned, summarized, but never erased
7. CLT is inspectable, navigable, and replayable

### CLT Node Model

```json
{
  "node_id": "clt_000124",
  "node_type": "interaction | summary | branch_marker",
  "parent_id": "clt_000118",
  "created_at": "2025-02-18T13:44:10Z",
  "session_id": "session_42",
  "payload": { },
  "metadata": { }
}
```

`parent_id = null` indicates root. Only one node per session is the current head.

### Node Types (3)

**Interaction Node:**
```json
{
  "node_type": "interaction",
  "payload": {
    "role": "user | assistant | system",
    "content_ref": "ref://artifact/abc123"
  },
  "metadata": {
    "task_id": "beads-124",
    "agent_id": "focusa-default",
    "model_id": "claude-3.5"
  }
}
```

CLT does not store raw text. Content lives in Reference Store. CLT stores only handles.

**Summary Node (Compaction):**
```json
{
  "node_type": "summary",
  "payload": {
    "summary_type": "abandoned_path | compaction",
    "summary_ref": "ref://artifact/summary_91af"
  },
  "metadata": {
    "covers_range": ["clt_000112", "clt_000118"],
    "reason": "context_compaction"
  }
}
```

**Branch Marker Node:**
```json
{
  "node_type": "branch_marker",
  "payload": {
    "branch_reason": "user_rephrase | alternative_strategy",
    "label": "retry_with_constraints"
  },
  "metadata": {
    "initiator": "user | agent"
  }
}
```

### Focus State Integration

Focus State references exactly one CLT node:
```json
{
  "focus_state": {
    "active_frame_id": "frame_7",
    "lineage_head": "clt_000124"
  }
}
```

Rules:
- Focus State always advances the CLT head
- Switching focus does not mutate CLT
- CLT does not select focus

### Compaction Rules

1. Identify contiguous path segment
2. Generate structured summary
3. Insert summary node
4. Reattach active head to summary node
5. Preserve original nodes as ancestors

Nothing is deleted.

### Complexity Guarantees

- Append: O(1)
- Branch: O(1)
- Context reconstruction: O(depth)
- No linear scans required

### Relationship to Other Systems

| System | Interaction |
|---|---|
| Focus State | References CLT head |
| Reducer | Emits CLT nodes (never reads entire tree) |
| Reference Store | Stores content referenced by CLT |
| Intuition Engine | Observes patterns (read-only) |
| CS | Consumes summaries & branch history |
| UFI | Links friction signals to CLT nodes |
| UI | Visualizes tree & navigation |

**Canonical rule:** The Context Lineage Tree preserves where we have been, not what we currently believe.

**Bridge mapping:** CLT → append-only `/data/wirebot/focusa-state/clt.jsonl`. Daily log files (`memory/*.md`) serve as human-readable CLT shadows.

---

## 11. Memory

Source: `G1-09-memory.md`

### Memory Types (MVP)

1. Semantic Memory (facts/preferences)
2. Procedural Memory (rules/habits)
3. Decay mechanism

Not in MVP: episodic store, schema emergence, meta-memory.

### Semantic Memory

#### SemanticRecord

```
SemanticRecord {
  key: String                           // e.g., "user.response_style"
  value: String                         // short
  created_at
  updated_at
  source: "user" | "worker" | "manual"
  confidence: f32                       // optional; default 1.0 for user-set
  ttl: Option<duration>                // optional
  tags: Vec<String>
}
```

MVP keys to support:
- `user.response_style` (e.g., concise steps)
- `project.name` (optional)
- `env.preferences` (optional)

#### Prompt Injection (Semantic)

Only include whitelisted keys in prompt:
- response style
- explicit project constraints

Serialize as compact: `PREFS: user.response_style=concise_steps`

### Procedural Memory

#### RuleRecord

```
RuleRecord {
  id: String                            // stable rule id
  rule: String                          // compact imperative
  weight: f32                           // internal
  reinforced_count: u32
  last_reinforced_at
  scope: RuleScope
  enabled: bool
}
```

#### RuleScope (3 values)

```
enum RuleScope {
  global
  frame:<frame_id>
  project:<name>                        // optional; later
}
```

#### Prompt Injection (Procedural)

Injected as "operating constraints":
`RULES: Prefer concise bullet steps; avoid verbosity.`

Cap: at most **5 rules** injected per turn. Ordered by weight descending and scope relevance to active frame.

### Memory Operations

- **UpsertSemantic**: set or update a semantic record. Emit `memory.semantic_upserted`.
- **ReinforceRule**: increase rule weight. Emit `memory.rule_reinforced`.
- **DecayTick**: periodic. `rule.weight *= 0.99`. If weight below threshold and not reinforced in long time → disable or remove (configurable). Emit `memory.decay_tick`.

### Memory Trust Rules (from UPDATE)

1. Memory is **opt-in**
2. Memory writes require: explicit user command OR user-confirmed candidate promotion
3. Workers may only *suggest* memory

### Pinned Memory (from UPDATE)

Pinned memory:
- immune to decay
- always eligible for prompt inclusion (within caps)

### Non-Goals (Explicit, from UPDATE)

- No automatic personality drift
- No silent preference learning
- No speculative inference

### Persistence

`~/.focusa/state/memory.json`

**Bridge mapping:** Semantic memory → Mem0 (:8200), archived nightly to `MEMORY.md`. Procedural memory → `/data/wirebot/focusa-state/rules.json` + human-readable `SOUL.md`.

---

## 12. Background Workers

Source: `G1-10-workers.md`

### Design Constraints

- Runs inside daemon process
- Uses async task queue
- Limited concurrency (default: 1–2 workers)
- Strict time budget per job
- Can be paused/disabled via config

### Worker Responsibilities (MVP)

- classify signals
- extract ASCC deltas
- propose Focus Gate candidates
- propose memory updates (advisory only)

Workers do NOT: mutate focus stack directly, assemble prompts, execute tools, call the harness/model.

### WorkerJob

```
WorkerJob {
  id
  kind: WorkerJobKind
  created_at
  priority: Low | Normal | High
  payload_ref: Option<HandleRef>
  frame_context: Option<FrameId>
  correlation_id
  timeout_ms
}
```

### WorkerJobKind (MVP — 5 values)

```
enum WorkerJobKind {
  classify_turn
  extract_ascc_delta
  detect_repetition
  scan_for_errors
  suggest_memory
}
```

### Job Definitions

**classify_turn:**
Input: turn transcript (via handle or small text).
Output: tags (file paths, errors, tools, intent shifts). Emit `gate.signal_ingested`.

**extract_ascc_delta:**
Input: delta turn content + current ASCC checkpoint.
Output: structured delta proposal. Reducer applies merge rules (worker does not mutate state).

**detect_repetition:**
Input: recent signals/candidates.
Output: repetition hint → Focus Gate.

**scan_for_errors:**
Input: tool outputs / assistant output.
Output: error signals with severity.

**suggest_memory:**
Input: repeated stable patterns.
Output: candidate memory suggestion (not applied automatically).

### WorkerQueue

- async channel (`mpsc`)
- bounded size: **default 100 jobs**
- backpressure: drop low-priority jobs if queue full

### Job Execution Rules

- jobs enqueued by daemon reducer
- high-priority jobs first
- max execution time per job: **default 200ms**
- if timeout exceeded → cancel and emit failure event
- workers must be panic-isolated
- failure does not affect daemon state

### Worker → Reducer Interaction

Workers return **results**, not state changes. Reducer decides whether to accept results, emit Focus Gate signals, or enqueue follow-up jobs.

### Worker Events (4)

- `worker.job_enqueued`
- `worker.job_started`
- `worker.job_completed`
- `worker.job_failed`

Each event includes: job id, kind, duration_ms, correlation_id.

### Persistence

Workers have **no persistent state**. All persistence handled by reducer.

**Bridge mapping:** Workers → async functions in bridge plugin, triggered by Clawdbot `afterAgentTurn` hook.

---

## 13. Expression Engine (Prompt Assembly)

Source: `08-expression-engine.md`, `G1-detail-11-prompt-assembly.md`

### Core Invariants

1. Deterministic output
2. Explicit structure
3. Bounded token usage
4. No silent truncation
5. No reasoning or planning

### Input

- Active Focus Frame
- Selected parent frame context
- Optional surfaced candidates (annotated)
- Invocation metadata
- ASCC checkpoints
- Selected semantic memory
- Selected procedural rules
- Handles (ECS)
- Raw user input
- Harness formatting requirements

### Slot-Based Structure (Canonical — 7 slots)

```
1. SYSTEM HEADER          — Static, short.
2. OPERATING RULES        — Procedural memory. At most 5 rules. Ordered by scope relevance + weight.
3. ACTIVE FOCUS FRAME     — ASCC checkpoint serialized for active frame (all 10 slots).
4. PARENT CONTEXT         — Optional, bounded. Intent/decisions/constraints from parent frames.
5. ARTIFACT HANDLES       — ECS refs (handles only, not content).
6. USER INPUT             — Raw user input.
7. EXECUTION DIRECTIVE    — Task-specific instructions.
```

### Token Budget Contract

Configurable per adapter:
- `max_prompt_tokens`: **default 6000**
- `reserve_for_response`: **default 2000**

Assembly must **never exceed** budget.

### Degradation Cascade (4 steps, ordered)

If budget is exceeded:
1. drop lowest-priority parent frames
2. drop non-essential ASCC slots
3. truncate rehydrated handles
4. fail only as last resort

Emit `prompt.assembled` with warnings.

### Priority Order (for truncation)

1. Intent (highest priority — last to truncate)
2. Constraints
3. Decisions
4. Current state
5. Next steps
6. Failures
7. Artifacts (lowest priority — first to truncate)

All truncation is: explicit, logged, reversible.

### Delta Injection Rule (from source)

ASCC deltas injected as structured slot content, not raw conversation replay.

### Degradation Strategy

If budget exceeded:
- emit degradation event
- annotate missing sections
- never silently drop meaning

### Forbidden Behaviors

- Implicit summarization
- Dynamic prompt shaping
- Content inference
- Memory mutation

**Bridge mapping:** Expression Engine → `beforeAgentTurn` hook in bridge plugin. Assembles system prompt prefix from Focusa state.

---

## 14. Proxy Adapter

Source: `09-proxy-adapter.md`, `G1-detail-04-proxy-adapter.md`

### Adapter Responsibilities

- Intercept model requests
- Invoke Expression Engine
- Inject Focus State
- Forward requests to model
- Capture responses
- Emit events

### Supported Harnesses (MVP)

- Letta
- Claude Code
- Codex CLI
- Gemini CLI
- Generic OpenAI-compatible APIs

### Integration Modes (from Gen1 detail)

**Mode A — Wrap Harness CLI (MVP Primary):**
Focusa wraps the harness's stdin/stdout. Intercepts all I/O.

**Mode B — HTTP Proxy (Optional):**
Focusa runs as HTTP proxy between harness and model endpoint.

### Adapter Contract with Daemon (from Gen1 detail)

Required daemon endpoints:
- current Focus State read
- ASCC checkpoint read
- ECS handle resolve
- event emit

### Turn Data Shapes (from Gen1 detail)

Each adapter normalizes I/O into:
- user input (text + optional tool calls)
- assistant output (text + optional tool results)
- turn_id (monotonic per session)

### Failure Handling

If Focusa fails:
- adapter passes through raw request (fail-safe passthrough)
- emits failure event
- does not block harness

### Performance Constraints

- <20ms overhead typical
- Zero blocking
- Async I/O only

### Thresholds (MVP Defaults, from Gen1 detail)

- `proxy.max_inject_tokens = 2000`
- `proxy.passthrough_on_error = true`

**Bridge mapping:** Proxy Adapter = Clawdbot gateway itself. Clawdbot already intercepts all model requests. Bridge hooks (`beforeAgentTurn`, `afterAgentTurn`) serve as the adapter interface points.

---

## 15. Autonomy Calibration

Source: `12-autonomy-scoring.md`, `37-autonomy-calibration-spec.md`

### Key Principle

> **Autonomy is a contract between permission and evidence.**

### Autonomy Level (AL)

Range: `AL0` → `AL5`. Discrete, explicitly granted, scoped, revocable.

#### AL Level Definitions

Source `12-autonomy-scoring.md` (authoritative for capabilities):

| Level | Capabilities |
|-------|-------------|
| AL0 | Advisory only |
| AL1 | Auto-resume frames; safe reads |
| AL2 | Select next task within scope |
| AL3 | Create subtasks; guarded edits |
| AL4 | Unattended operation (hours) |
| AL5 | Multi-day autonomy with check-ins |

Source `37-autonomy-calibration-spec.md` (example labels):

| Level | Label |
|-------|-------|
| 0 | Advisory only |
| 1 | Assisted execution |
| 2 | Conditional autonomy |
| 3 | Limited unattended runs |
| 4 | Extended autonomous operation |
| 5 | Long-horizon autonomy (future) |

Levels are **per agent + model + harness**.

### Autonomy Reliability Index (ARI)

Quantitative score 0–100 representing how reliably the system has operated within its granted autonomy.

ARI is: computed from facts, derived from events, explainable, reproducible.
ARI does NOT: imply permission, cause automatic promotion, hide uncertainty.

### Data Sources (All Verifiable)

**Primary:** Reducer event log, Focus Stack transitions, Focus State updates, Reference Store usage, Beads task lifecycle events.

**Metadata:** model_id, harness_id, repo_signature, task_class, risk_profile, context_pressure indicators.

No inferred or hidden data permitted.

### Scoring Dimensions (6)

Source: `37-autonomy-calibration-spec.md`

| Dimension | Description |
|-----------|-------------|
| Correctness | Constraint compliance, validation pass rate |
| Stability | Low rework, low abandonment |
| Efficiency | Tokens, time, tool economy |
| Trust | UXP/UFI-adjusted satisfaction |
| Grounding | Reference correctness |
| Recovery | Error correction behavior |

Each dimension tracked independently.

### Scoring Categories & Weights (from `12-autonomy-scoring.md`)

| Category | Weight | Signals |
|----------|--------|---------|
| Outcome | 50% | completion_rate, regression_penalty, block_correctness |
| Efficiency | 20% | time_ratio, rework_penalty |
| Discipline | 15% | focus_discipline_score, artifact_compliance_score |
| Safety | 15% | safety_penalty, escalation_correctness |

### ARI Calculation

```
ARI = clamp(
  weighted_average(outcome_score, efficiency_score, discipline_score, safety_score)
  / expected_difficulty_factor
, 0, 100)
```

Expected difficulty factor derived from: model capability class, harness behavior, task class, repo complexity, context pressure.

ARI always accompanied by: sample size, confidence band (low / medium / high).
Low sample size reduces promotion eligibility, not ARI itself.

### Promotion Rules (Never Automatic)

1. Explicit permission grant
2. Minimum ARI threshold
3. Minimum sample size
4. Defined scope + TTL

Focusa may **recommend** promotion, never execute it.

### Calibration Modes

- On-Demand: explicitly triggered, short bounded task suite
- Continuous Background: passive observation, rolling metrics, no disruption

### Calibration Suite

Each task defines: allowed tools, risk level, expected invariants, success checks, max budget (tokens/time).
Suites are: model-specific, harness-specific, domain-specific.

### Storage (Local DB, recommended SQLite)

Tables (MVP): `runs`, `tasks`, `events_index`, `scores`, `capability_grants`, `environment_signatures`. All entries append-only or versioned.

**Bridge mapping:** Autonomy profiles → Letta block `autonomy`. Scoring engine → bridge plugin. Storage → SQLite or JSON at `/data/wirebot/focusa-state/autonomy/`.

---

## 16. Reliability Focus Mode (RFM)

Source: `36-reliability-focus-mode.md`

### Reliability Levels (per Focus Frame)

| Level | Name | Behavior |
|-------|------|---------|
| R0 | Normal | No reliability escalation |
| R1 | Validation | Spawn validator microcells |
| R2 | Regeneration | Validate → regenerate once on failure |
| R3 | Ensemble | Multiple generators + validators (rare) |

RFM level is **decided by Focus Gate**, not the agent.

### Microcells

Isolated, narrow-scope sub-agents invoked for verification, not creativity.
- have their own context
- do not see full session history
- do not modify Focus State
- return structured evidence

### Microcell Types (MVP — 4 validators)

1. **Schema Validator** — checks formatting, JSON schema, required fields
2. **Constraint Validator** — checks explicit constraints (files, scope, tools)
3. **Consistency Validator** — checks internal contradictions
4. **Reference-Grounding Validator** — checks claims against Reference Store / CLT

Each returns:
```json
{
  "result": "pass | fail",
  "reason": "string",
  "citations": [{ "type": "ref | clt", "id": "uuid" }]
}
```

### Triggers

**Structural:** Frame marked `risk: high`, write/destructive ops, security-sensitive, external system interaction.
**Behavioral:** Low gate acceptance rate, high rework ratio, recent cache bust, CLT branch abandonment spike.
**Human:** Rising UFI, explicit user override.
**Autonomy:** Level ≥ threshold, calibration policy recommends.

### Execution Flow

1. Focus Gate selects RFM level
2. Primary agent produces candidate output
3. Validator microcells invoked in parallel
4. Validation results aggregated
5. Gate decision: accept / reject+regenerate / escalate
6. Outcome recorded in CLT + telemetry

### Failure Handling

- Validation failure does NOT mutate Focus State
- Failures create CLT child nodes + telemetry events
- Regeneration limited: max 1 in R2
- R3 never automatic without policy approval

### Artifact Integrity Score (AIS) — from UPDATE

```
AIS ∈ [0.0, 1.0]
AIS = known_artifacts_referenced / known_artifacts_expected
```

Artifact categories tracked per frame: files_read, files_modified, files_created, symbols_touched, external_refs_used.

**AIS Thresholds:**
- AIS ≥ 0.90 → Safe
- 0.70 ≤ AIS < 0.90 → Degraded
- AIS < 0.70 → Reliability Focus Mode auto-activates

When AIS drops below threshold:
1. Pause autonomy escalation
2. Spin up validator sub-agents
3. Force artifact reconciliation step
4. Re-anchor Focus State with explicit artifact listing
5. Emit explanation to UI/TUI

An agent cannot earn autonomy while losing artifact integrity.

### Telemetry Events

- `rfm.invoked`
- `rfm.level`
- `validator.pass` / `validator.fail`
- `rfm.regeneration`
- `rfm.escalation`
- `artifact.integrity.violation`

**Bridge mapping:** RFM logic → bridge plugin. Microcells could use Letta's tool execution framework.

---

## 17. Thread Thesis

Source: `38-thread-thesis-spec.md`

### Core Definition

A structured, continuously refined representation of: user intent, goals, constraints, open questions, confidence level — that governs how Focusa interprets and prioritizes all subsequent input. The top-level cognitive anchor for a session.

**Not** a summary. **Not** a transcript. **Not** a prompt.

### Design Principles

1. Meaning over words — captures semantic intent, not phrasing
2. Stable but revisable — changes deliberately, not continuously
3. Structured, not free-form — machine-evaluable, not prose
4. Explainable — every update has provenance
5. Non-authoritative — informs decisions but does not enact them

### Canonical Schema

```json
{
  "thesis_id": "uuid",
  "version": "int",
  "created_at": "timestamp",
  "updated_at": "timestamp",

  "primary_intent": "string",
  "secondary_goals": ["string"],

  "explicit_constraints": ["string"],
  "implicit_constraints": ["string"],

  "open_questions": ["string"],

  "assumptions": ["string"],

  "confidence": {
    "score": 0.0,
    "rationale": "string"
  },

  "scope": {
    "domain": "string",
    "time_horizon": "short | medium | long",
    "risk_level": "low | medium | high"
  },

  "sources": [
    { "type": "clt | user | system", "id": "uuid" }
  ]
}
```

### Lifecycle

**Creation:** At session start, after onboarding, after explicit goal-setting. Initial confidence is low.

**Update triggers:**
- User explicitly redefines goals
- Focus Stack root changes
- Repeated clarifications occur
- UFI spikes (indicating misalignment)
- Calibration recommends re-centering
- Long sessions exceed thresholds
- Autonomy level changes

Updates are event-driven, not per-turn.

**Update process:**
1. Reducer proposes thesis update
2. Focus Gate evaluates: alignment, evidence, stability impact
3. If accepted: version increments, old version archived
4. Change recorded in CLT

### Safeguards

**Drift prevention:** Minimum confidence delta required for change. Cooldown between updates.
**Overfitting prevention:** Do not absorb single anomalous turns. Require corroboration over time.

### Prompt Assembly Rules

Thread Thesis is **never injected raw**. Distilled signals (intent, constraints) may influence system instructions, tool selection, validator constraints. Prevents prompt bloat and cache pollution.

### Interaction with Other Systems

- **Focus State**: must be consistent with Thesis. Conflicts trigger clarification.
- **Focus Gate**: uses Thesis to score relevance, detect drift, justify rejections.
- **CLT**: Thesis updates become lineage nodes.
- **RFM**: High risk thesis → higher reliability defaults.
- **Autonomy**: Confidence impacts autonomy ceilings.

### Telemetry

- `thesis.updated`
- `thesis.version`
- `thesis.confidence_delta`
- citations to supporting CLT nodes

**Bridge mapping:** Thread Thesis → workspace `BUSINESS_BRIEF.md` + Letta `human` and `goals` blocks.

---

## 18. Threads (Cognitive Workspaces)

Source: `39-thread-lifecycle-spec.md`

### Definition

A Thread is a persistent cognitive workspace that binds:
- a Thread Thesis
- a Context Lineage Tree (CLT)
- a Focus Stack
- a Reference Store namespace
- telemetry and autonomy history

Threads are the unit of continuity in Focusa.

### Thread Identity

```json
{
  "thread_id": "uuid",
  "name": "string",
  "status": "active | paused | archived",
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

### Thread Operations (6)

**Create:** Triggered by explicit user action, API call, or CLI. Creates new thread_id, CLT root, empty Focus Stack, new Thread Thesis (low confidence), isolated Reference Store namespace, resets telemetry counters. No state inherited unless explicitly requested.

**Create with Inheritance:** Optional flags: constitution, preferences, reference subset, calibration profile. Inheritance is explicit, never implicit.

**Resume (Continue):** Rehydrates: latest Thesis version, CLT active head, Focus Stack, autonomy profile, cache permission state. Does NOT: replay conversation, re-inject full history, auto-escalate autonomy.

**Save (Checkpoint):** Commits Focus Stack head, persists Thesis version, snapshots autonomy + telemetry state, records checkpoint marker in CLT. Idempotent and lightweight.

**Rename:** Updates human-readable metadata only. Does not alter cognition or lineage. Always reversible.

**Fork:** Creates new thread from existing CLT node. New thread_id, selected CLT node becomes new root, Thesis cloned (with reduced confidence), Focus Stack resets, Reference Store optionally pruned. Preserves exploration without cognitive contamination.

**Archive:** Freezes thread state. Disallows new Focus Frames. Allows inspection and export. Preserves telemetry for training. Archived threads are immutable.

### Thread Guarantees (5)

1. Threads never share mutable state
2. One active Thread per agent session
3. CLT nodes belong to exactly one Thread
4. Telemetry is thread-scoped
5. Autonomy is thread-specific

**Bridge mapping:** One Thread per founder relationship. Thread state → `/data/wirebot/focusa-state/threads/<id>.json`.

---

## 19. Instances, Sessions, Attachments

Source: `40-instance-session-attachment-spec.md`

### Definitions

- **Instance** = where (a concrete runtime integration point connected to the daemon)
- **Session** = when (a temporal execution window within an Instance)
- **Attachment** = what (a live binding between an Instance/Session and a Thread)

### Instance Schema

```json
{
  "instance_id": "uuid",
  "created_at": "timestamp",
  "updated_at": "timestamp",

  "kind": "acp | cli | tui | gui | background",
  "integration": {
    "product": "zed | claude_code | codex | gemini | tmux | other",
    "protocol": "acp | stdio | http | grpc | other",
    "version": "string"
  },

  "host": {
    "machine_id": "string",
    "user_id": "string",
    "cwd": "string",
    "repo_root": "string|null"
  },

  "status": "online | offline | degraded",
  "labels": ["string"],

  "capability_scope": {
    "allowed": ["capability_id"],
    "denied": ["capability_id"]
  }
}
```

### Session Schema

```json
{
  "session_id": "uuid",
  "instance_id": "uuid",

  "started_at": "timestamp",
  "ended_at": "timestamp|null",
  "status": "active | ended | timed_out",

  "harness": {
    "name": "claude_code | codex_cli | gemini_cli | zed_acp | other",
    "mode": "proxy | observe",
    "details": { "key": "value" }
  },

  "model_context": {
    "provider": "openai | anthropic | google | local | other",
    "model": "string",
    "temperature": 0.0,
    "max_tokens": 0
  },

  "cache_context": {
    "cache_key": "string|null",
    "policy": "normal | conservative | aggressive"
  }
}
```

Sessions may exist without model_context if in pure observe mode.
`cache_context` is for Focusa-internal caching, not provider-specific.

### Attachment Schema

```json
{
  "attachment_id": "uuid",
  "thread_id": "uuid",
  "instance_id": "uuid",
  "session_id": "uuid",

  "attached_at": "timestamp",
  "detached_at": "timestamp|null",

  "status": "attached | detached",

  "role": "active | assistant | observer | background",
  "priority": 0,

  "focus_read": true,
  "proposal_write": true,

  "notes": "string|null"
}
```

Role semantics:
- **active**: primary interactive context for that user surface
- **assistant**: secondary surface (may propose, not canonical)
- **observer**: read + telemetry only (no proposals)
- **background**: Intuition Engine work (validators, retrieval, calibration)

### Invariants (6)

1. Instances can have many Sessions over time
2. Sessions belong to exactly one Instance
3. Attachments bind a Session/Instance to exactly one Thread
4. A Session can attach to multiple Threads (rare) but MUST declare one **primary** attachment (highest priority)
5. A Thread can be attached by many Instances simultaneously
6. Attachments do not grant mutation authority — only proposal authority

### Lifecycles

**Instance:** created at first connect → updated on reconnect/metadata change → offline on disconnect → never deleted automatically (archivable).
**Session:** created on connect → active until disconnect → ended explicitly or timed out.
**Attachment:** created on bind → detached on explicit action or session end → detaching does not delete history.

### Telemetry Requirements

Every event must include: thread_id (if applicable), instance_id, session_id, attachment_id (if applicable).

Events: `instance.connected`, `instance.disconnected`, `session.started`, `session.ended`, `session.timed_out`, `thread.attached`, `thread.detached`, `proposal.submitted`, `proposal.resolved`.

**Bridge mapping:** Each Clawdbot channel (WebSocket, Discord, Telegram, SMS) = one Instance. Each Clawdbot session = one Session. One Thread per founder.

---

## 20. Proposal Resolution Engine (PRE)

Source: `41-proposal-resolution-engine.md`

### Purpose

Enables timestamped, async concurrency across multiple Instances and Sessions without locks. Resolves competing decisional proposals into a single canonical outcome.

### Observations vs Decisions

**Observations** (always concurrent, append-only): CLT nodes, reference additions, validator results, telemetry events. Never conflict.

**Decisions** (subject to resolution): focus change, focus stack mutation, thesis update, autonomy adjustment, constitution update. Expressed as proposals.

### Proposal Schema

```json
{
  "proposal_id": "uuid",
  "thread_id": "uuid",
  "instance_id": "uuid",
  "session_id": "uuid",
  "attachment_id": "uuid|null",

  "timestamp": "timestamp",
  "type": "focus.change | thesis.update | autonomy.change | constitution.propose",
  "payload": { "key": "value" },

  "confidence": 0.0,
  "evidence": [
    { "type": "clt|ref|telemetry", "id": "uuid" }
  ],

  "status": "pending | accepted | rejected | superseded",
  "resolution": {
    "resolved_at": "timestamp|null",
    "winner": "proposal_id|null",
    "reason": "string|null",
    "citations": [{ "type": "clt|telemetry", "id": "uuid" }]
  }
}
```

### Resolution Windows

Per thread, per target class (focus, thesis, autonomy, constitution). Time bounded: **default 500ms–2000ms** (configurable).

Key tuple: `(thread_id, target, window_start)`

### Resolution Algorithm

At window close:
1. Gather all pending proposals in window
2. Compute score for each (deterministic)
3. Select single winner or no-winner (request clarification)
4. Emit resolution events
5. Apply winner to canonical state via reducer
6. Record outcome in CLT + telemetry

### Scoring Inputs (Deterministic)

- **Evidence Strength**: validator pass rate, grounding evidence, references cited
- **Alignment**: Thread Thesis alignment, active Focus Frame consistency
- **Risk & Reliability**: if high risk → require validator support; if RFM active → weight validators strongly
- **Source Trust**: instance role (active > assistant > background > observer), autonomy level ceiling
- **Recency**: slight bias to later proposals within window (configurable)

### Outcomes

- **Accept One**: winner applied, others rejected as conflicting
- **Reject All**: proposals too divergent, evidence insufficient, or policy requires human confirmation
- **Supersede**: new proposal supersedes earlier pending ones

### Canonical State Invariants (even with concurrency)

- Focus State is singular per thread
- Thesis version is linear per thread
- Autonomy level is singular per thread
- History is never erased

### Telemetry Events

- `proposal.submitted`
- `proposal.window.opened`
- `proposal.window.closed`
- `proposal.resolved`
- `proposal.rejected`
- `proposal.clarification_required`

**Bridge mapping:** PRE → bridge plugin. Active when multiple channels interact with same founder simultaneously.

---

## 21. Capability Permissions

Source: `25-capability-permissions.md`

### Canonical Principle

> **Observation is cheap. Authority is expensive.**

### Permission Model

Scopes expressed as: `<domain>:<action>`

Examples: `state:read`, `lineage:read`, `constitution:propose`, `commands:submit`, `contribute:approve`

### Permission Classes (3)

**Read:** Non-destructive, safe. `state:read`, `lineage:read`, `references:read`, `metrics:read`, `cache:read`, `events:read`.

**Command:** Intentional mutation via commands. `commands:submit`, `constitution:activate`, `contribute:pause`, `export:start`. Always require policy validation, audit logging, autonomy checks.

**Administrative:** Reserved for local owner. `admin:tokens`, `admin:shutdown`, `admin:config`. Not exposed to agents.

### Default Permission Sets

**Local Owner (CLI/UI):**
```json
{ "read:*": true, "commands:submit": true, "constitution:*": true,
  "contribute:*": true, "export:*": true, "admin:*": true }
```

**Agent (Default):**
```json
{ "state:read": true, "lineage:read": true, "references:read": true,
  "metrics:read": true, "intuition:read": true, "autonomy:read": true,
  "commands:submit": false }
```

Agents can observe cognition, not control it.

**External Tool/Integration:**
```json
{ "state:read": true, "lineage:read": true, "events:read": true,
  "commands:submit": false }
```

### Token Types (3)

- **Owner Token**: Full permissions, stored locally, rotatable
- **Agent Token**: Scoped permissions, bound to agent_id, revocable
- **Integration Token**: Read-only by default, expirable

### Enforcement Rules

1. Every API request authenticated
2. Permissions checked per endpoint
3. Commands require explicit permission
4. Lack of permission → 403 forbidden
5. Permissions never inferred

### Policy Interaction

Permissions are necessary but not sufficient. Even with permission: Focus Gate may block, autonomy level may prevent action, contribution policy may deny export, constitution rules may override.

**Policy always wins over permission.**

### Canonical Rule

> **Permissions grant access. Policy grants authority. Cognition grants action.**

**Bridge mapping:** Permissions → `/data/wirebot/focusa-state/permissions.json`. Enforced by bridge plugin on every tool call.

---

## 22. Agent Skill Bundle

Source: `34-agent-skills-spec.md`

### Canonical Principle

> **Agents may understand the system deeply before they are allowed to change it at all.**

Skills expose state and reasoning, not authority. The skill bundle is the **only sanctioned way** for agents to reason with Focusa's internal state.

### Skill Categories (4)

**1. Cognition Inspection (read-only):**

| Skill | Returns | API |
|-------|---------|-----|
| `focusa.get_focus_state` | `{intent, constraints, active_frame, confidence, focus_depth}` | `GET /v1/state/current` |
| `focusa.get_focus_stack` | `{stack: [{frame_id, label}]}` | `GET /v1/state/stack` |
| `focusa.get_lineage_tree` | `{root, nodes: [...]}` | `GET /v1/lineage/tree` |
| `focusa.get_gate_explanation` | `{candidates: [{id, score, accepted}], reason}` | `GET /v1/gate/explain` |
| `focusa.get_salient_references` | `{references: [{ref_id, type, summary}]}` | `GET /v1/references/salient` |
| `focusa.get_constitution` | `{version, text}` | `GET /v1/constitution/current` |
| `focusa.get_autonomy_status` | `{level, confidence, constraints}` | `GET /v1/autonomy/status` |

**2. Telemetry & Metrics (read-only):**

| Skill | Returns | API |
|-------|---------|-----|
| `focusa.get_token_stats` | `{prompt_tokens, completion_tokens, efficiency}` | `GET /v1/telemetry/tokens` |
| `focusa.get_cognitive_metrics` | `{avg_focus_depth, abandonment_rate, gate_acceptance}` | `GET /v1/telemetry/process` |
| `focusa.get_ux_signals` | `{uxp, ufi, evidence: [...]}` | `GET /v1/telemetry/ux` |

**3. Explanation & Traceability (read-only):**

| Skill | Returns | API |
|-------|---------|-----|
| `focusa.explain_last_decision` | focus change rationale, gate reasoning, supporting evidence | `GET /v1/state/explain` |
| `focusa.trace_reference_usage` | `{ref_id, used_in: ["clt_12", "clt_18"]}` | `GET /v1/references/trace` |

**4. Proposal & Request (guarded — do not enact change):**

| Skill | Payload | API |
|-------|---------|-----|
| `focusa.propose_focus_change` | `{new_focus, reason}` | `POST /v1/commands/request (type: focus.change)` |
| `focusa.request_cache_bust` | `{reason}` | `POST /v1/commands/request (type: cache.bust)` |
| `focusa.propose_constitution_update` | `{draft, justification}` | `POST /v1/constitution/propose` |
| `focusa.request_command` | `{command_type, params}` | `POST /v1/commands/request` |

### Explicitly Prohibited Skills

Must **never** exist:
- `focusa.set_focus_state`
- `focusa.modify_lineage`
- `focusa.write_reference`
- `focusa.activate_constitution`
- `focusa.escalate_autonomy`
- `focusa.approve_export`

Reason: breaks auditability, collapses autonomy ladder, destroys trust.

### Skill Permissions vs Autonomy

| Autonomy Level | Skill Access |
|----------------|-------------|
| Level 0 | Inspection only |
| Level 1 | Telemetry + explanations |
| Level 2 | Proposals allowed |
| Level 3+ | Faster approvals, fewer confirmations |
| Level N | (Future) Limited autonomous command execution |

The skill surface does not change — only gate behavior does.

### Canonical Rule

> **Skills reveal truth. Gates decide action. Autonomy is earned.**

**Bridge mapping:** Each Focusa skill → Clawdbot registered tool via `api.registerTool()` in bridge plugin.

---

## 23. Events & Observability

Source: `G1-detail-15-events-observability.md`

### Event Types (from Gen1 — complete taxonomy)

**Focus Stack events:** `focus.frame_pushed`, `focus.frame_completed`, `focus.active_changed`

**Focus Gate events:** `gate.signal_ingested`, `gate.candidate_surfaced`, `gate.candidate_suppressed`

**ASCC events:** `ascc.delta_applied`, `ascc.checkpoint_saved`

**ECS events:** `ecs.artifact_stored`, `ecs.handle_resolved`

**Memory events:** `memory.semantic_upserted`, `memory.rule_reinforced`, `memory.decay_tick`

**Prompt events:** `prompt.assembled`, `prompt.degradation`

**Worker events:** `worker.job_enqueued`, `worker.job_started`, `worker.job_completed`, `worker.job_failed`

**Adapter/Turn events:** `adapter.turn_started`, `adapter.turn_completed`

**Replay Invariant (from UPDATE):** Events must support deterministic replay. Given the same event sequence, the reducer must produce the same state. Events are the authoritative log — state snapshots are accelerators.

**Bridge mapping:** Telemetry → append-only `/data/wirebot/focusa-state/telemetry.jsonl` + systemd journal.

---

## 24. Storage Mapping: Every Focusa Object → Bridge Backend

```
FOCUSA OBJECT                         PRIMARY BACKEND           LOCATION
────────────────────────────          ─────────────             ─────────────────────────────
FocusaState (reducer snapshot)        Local file                /data/wirebot/focusa-state/state.json
Reducer event log                     Local append-only         /data/wirebot/focusa-state/events.jsonl
Focus Stack (FocusStackState)         Local file                /data/wirebot/focusa-state/stack.json
Focus State (per frame)               Local file                /data/wirebot/focusa-state/frames/<id>.json
ASCC checkpoints (live)               Letta blocks              business_stage, goals, kpis, + custom
ASCC checkpoints (snapshot)           Workspace file            clawd/BUSINESS_STATE.md
ECS/Reference Store artifacts         Workspace files           /home/wirebot/clawd/**
ECS metadata index                    Local file                /data/wirebot/focusa-state/ecs/index.json
CLT nodes                             Local append-only         /data/wirebot/focusa-state/clt.jsonl
CLT human-readable shadow             Workspace files           clawd/memory/*.md
Focus Gate candidates                 Local file                /data/wirebot/focusa-state/gate.json
Intuition Engine signals              In-memory                 (ephemeral until Gate promotes)
Semantic memory (live)                Mem0 (:8200)              namespace: wirebot_<user_id>
Semantic memory (archive)             Workspace file            clawd/MEMORY.md
Procedural memory rules               Local file                /data/wirebot/focusa-state/rules.json
Procedural memory (readable)          Workspace file            clawd/SOUL.md
Thread Thesis                         Workspace file            clawd/BUSINESS_BRIEF.md
Thread Thesis (structured)            Letta blocks              human + goals
Thread state                          Local file                /data/wirebot/focusa-state/threads/<id>.json
Autonomy profile                      Letta block               autonomy
Autonomy scoring DB                   Local SQLite/JSON         /data/wirebot/focusa-state/autonomy/
Telemetry events                      Local append-only         /data/wirebot/focusa-state/telemetry.jsonl
Telemetry (system)                    systemd journal           journalctl -u clawdbot-gateway
Agent Constitution                    Workspace file            clawd/SOUL.md
Capability permissions                Local config              /data/wirebot/focusa-state/permissions.json
Worker queue                          In-memory                 (ephemeral, bounded 100)
```

Design principle: Every piece of state has a **primary backend** (real-time) and a **workspace shadow** (.md file memory-core indexes). Nightly sync aligns them. Workspace shadows + reducer event log can reconstruct everything.

---

## 25. Spec Document → Implementation File Mapping

```
FOCUSA SPEC                                       BRIDGE FILE
──────────────────────────────────                ──────────────────────────
core-reducer.md                                →  bridge/reducer.ts
G1-detail-03-runtime-daemon.md                 →  bridge/daemon.ts (state management)
G1-detail-05-focus-stack-hec.md                →  bridge/focus-stack.ts
03-focus-stack.md                              →  bridge/focus-stack.ts (invariants)
06-focus-state.md                              →  bridge/focus-state.ts
04-focus-gate.md + G1-detail-06                →  bridge/focus-gate.ts
05-intuition-engine.md                         →  bridge/intuition.ts
G1-07-ascc.md                                  →  bridge/ascc.ts
G1-detail-08-ecs.md + 07-reference-store.md    →  bridge/reference-store.ts
G1-09-memory.md                                →  bridge/memory.ts
G1-10-workers.md                               →  bridge/workers.ts
08-expression-engine.md + G1-detail-11         →  bridge/expression-engine.ts
09-proxy-adapter.md + G1-detail-04             →  bridge/adapter.ts
17-context-lineage-tree.md                     →  bridge/clt.ts
38-thread-thesis-spec.md                       →  bridge/thesis.ts
39-thread-lifecycle-spec.md                    →  bridge/threads.ts
40-instance-session-attachment-spec.md         →  bridge/instances.ts
41-proposal-resolution-engine.md               →  bridge/pre.ts
12-autonomy-scoring.md + 37-calibration        →  bridge/autonomy.ts
36-reliability-focus-mode.md                   →  bridge/rfm.ts
16-agent-constitution.md + 16-synth            →  bridge/constitution.ts
29-telemetry-spec.md + 30-schema               →  bridge/telemetry.ts
G1-detail-15-events-observability.md           →  bridge/events.ts
23-capabilities-api.md                         →  bridge/api.ts
34-agent-skills-spec.md                        →  bridge/tools.ts
25-capability-permissions.md                   →  bridge/permissions.ts
00-glossary.md                                 →  bridge/types.ts
01-architecture-overview.md                    →  (this document)
```

---

## See Also

- [MEMORY_BRIDGE_STRATEGY.md](./MEMORY_BRIDGE_STRATEGY.md) — Bridge storage design
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Wirebot system architecture
- [VISION.md](./VISION.md) — Product vision
- `/data/wirebot/focusa/docs-final/INDEX.md` — All 67 Focusa spec documents
