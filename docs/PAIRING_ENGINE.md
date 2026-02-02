# Pairing Engine — Implementation Documentation

> Technical documentation for `cmd/scoreboard/pairing.go` — the living profile engine
> that builds, maintains, and adapts a psychometric-grade Founder Profile in real-time.
>
> **Spec docs:** [PAIRING_SCIENCE.md](./PAIRING_SCIENCE.md) (formulas), [PAIRING_V2.md](./PAIRING_V2.md) (UI/UX)
> **Source:** `cmd/scoreboard/pairing.go` (1624 lines)
> **Status:** v2.0 — engine core operational, wiring in progress

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Data Flow — Signal Pipeline](#2-data-flow--signal-pipeline)
3. [Type Reference](#3-type-reference)
4. [The 7 Constructs (Φ1–Φ7)](#4-the-7-constructs-φ1φ7)
5. [Dual-Track EMA System](#5-dual-track-ema-system)
6. [Signal Processors](#6-signal-processors)
7. [Assessment Instruments & Scoring Tables](#7-assessment-instruments--scoring-tables)
8. [NLP Feature Extraction](#8-nlp-feature-extraction)
9. [Context Window System](#9-context-window-system)
10. [Drift Detection](#10-drift-detection)
11. [Complement Rebalancer](#11-complement-rebalancer)
12. [Calibration Engine](#12-calibration-engine)
13. [Pairing Score Computation](#13-pairing-score-computation)
14. [Accuracy Convergence](#14-accuracy-convergence)
15. [Override System](#15-override-system)
16. [Evidence Trail](#16-evidence-trail)
17. [Chat Context Injection](#17-chat-context-injection)
18. [Persistence & Lifecycle](#18-persistence--lifecycle)
19. [API Endpoints](#19-api-endpoints)
20. [CLI Commands](#20-cli-commands)
21. [Wiring Into Scoreboard](#21-wiring-into-scoreboard)
22. [File Map](#22-file-map)

---

## 1. Architecture Overview

The Pairing Engine is a **continuously-running background system** embedded in the
`wirebot-scoreboard` Go binary. It processes every observable signal (chat messages,
scoreboard events, documents, connected account data, assessment answers, approval
actions) through a unified pipeline that builds and refines the Founder Profile.

```
                        ┌──────────────────┐
                        │   Signal Sources  │
                        └────────┬─────────┘
                                 │
          ┌──────────────────────┼──────────────────────┐
          │                      │                      │
    ┌─────▼──────┐   ┌──────────▼──────────┐  ┌────────▼───────┐
    │ Chat msgs  │   │ Scoreboard events   │  │  Assessment    │
    │ (webhook)  │   │ (DB insert hook)    │  │  answers (API) │
    └─────┬──────┘   └──────────┬──────────┘  └────────┬───────┘
          │                      │                      │
          │  ┌───────────────────┼───────────────┐      │
          │  │                   │               │      │
    ┌─────▼──▼───┐   ┌──────────▼──┐   ┌────────▼──────▼──┐
    │ Approvals  │   │  Documents  │   │ Account data    │
    │ (hook)     │   │  (API)      │   │ (pollers)       │
    └─────┬──────┘   └──────┬──────┘   └────────┬────────┘
          │                  │                   │
          └──────────────────┼───────────────────┘
                             │
                    ┌────────▼────────┐
                    │  Signal Channel │  (buffered, capacity=1000)
                    │  (non-blocking) │
                    └────────┬────────┘
                             │ single goroutine (serial processing)
                    ┌────────▼────────┐
                    │  Process Loop   │
                    │                 │
                    │  For each signal:
                    │  1. Extract features (NLP/metadata)
                    │  2. Route to processor (message/event/etc.)
                    │  3. Update dual-track EMAs
                    │  4. Detect drift
                    │  5. Evaluate context windows
                    │  6. Recompute complement vector
                    │  7. Update calibration parameters
                    │  8. Log evidence entry
                    │  9. Mark dirty → persist
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │  profile.json   │  (written every 10 signals or 60s)
                    └─────────────────┘
```

### Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Single goroutine processing | EMAs require serial ordering. Parallelism would corrupt running averages. |
| Buffered channel (1000) | Prevents back-pressure on the HTTP handlers and event inserters. Drops signals gracefully if overwhelmed. |
| In-memory + periodic flush | ~1-5ms per signal processing. Flushing every 10 signals or 60s balances durability with performance. |
| All state in one JSON file | Simple, inspectable, debuggable. Profile is ~50KB even fully populated. No DB overhead for profile data. |
| Evidence kept in memory (10K cap) | Evidence is transient display data. Oldest entries evicted. Persistent evidence goes to SQLite (future). |

---

## 2. Data Flow — Signal Pipeline

Every signal follows the exact same pipeline. No signal type gets special treatment
in terms of ordering — they all go through the same 9 steps:

```
Signal arrives via pe.Ingest(sig)    ← non-blocking send to channel
         │
         ▼
Step 1:  extractFeatures(sig)        ← NLP for messages, metadata pass-through for events
         │
Step 2:  Build EvidenceEntry         ← timestamp, source, empty impact map
         │
Step 3:  Route to processor          ← processMessage / processEvent / processAssessment /
         │                              processApproval / processDocument / processAccountData
         │                              Each processor calls dt.UpdateEMA() on relevant constructs
         │
Step 4:  detectDrift(ev)             ← Scans all constructs for drift ≥ 2.0σ
         │                              Records DriftEvent if significant
         │
Step 5:  evaluateContextWindows()    ← Decays all windows, records active ones in evidence
         │
Step 6:  recomputeComplement()       ← Rebuilds wirebot_effort = inverse(founder_state)
         │
Step 7:  updateCalibration()         ← Maps profile into Wirebot behavior parameters
         │
Step 8:  summarizeEvidence()         ← Human-readable summary string for the evidence log
         │
Step 9:  Increment meta counters     ← SignalsProcessed++, mark dirty
```

### Timing

| Operation | Typical Latency |
|-----------|----------------|
| NLP feature extraction | ~0.5ms (pure string analysis, no ML) |
| EMA updates (all constructs) | ~0.1ms |
| Drift detection | ~0.1ms |
| Context window evaluation | ~0.05ms |
| Complement rebalance | ~0.05ms |
| Calibration update | ~0.05ms |
| **Total per signal** | **~1-2ms** |
| JSON persistence (when triggered) | ~5-10ms |

Anything over 50ms logs a warning: `[pairing] Slow signal processing`.

---

## 3. Type Reference

### Core Types

| Type | Description | Location |
|------|-------------|----------|
| `Signal` | Input to the engine. Has Type, Source, Timestamp, Content, Metadata, Features. | L34-42 |
| `SignalType` | Enum: `message`, `event`, `document`, `account`, `assessment`, `approval` | L28-33 |
| `EvidenceEntry` | Record of what a signal did to the profile. ID, timestamp, features extracted, profile impact, constructs affected. | L46-54 |
| `PredictionEntry` | A prediction the engine made + whether it was correct. For accuracy tracking. | L58-67 |
| `DualTrackDimension` | The core data structure. Every construct uses this. Contains Trait, State, Drift, Alpha, Effective, SigmaTrait, Observations per dimension. | L71-80 |
| `ContextWindow` | Activation level for a detected context (FINANCIAL_PRESSURE, SHIPPING_SPRINT, etc.). Has decay. | L177-185 |
| `ProfileOverride` | Founder's manual correction to a score. Decays over 30 days unless confirmed by behavioral data. | L214-225 |
| `ComplementVector` | Wirebot's effort allocation across 10 dimensions. Sums to 1.0. Inverse of founder's effective scores. | L239-252 |
| `CalibrationParams` | The actual behavioral parameters Wirebot uses: max message words, lead-with style, nudge frequency, etc. | L292-318 |
| `AccuracyLedger` | Overall accuracy, per-construct accuracy, calibration lift, drift patterns. | L338-350 |
| `DriftEvent` | Historical record of a significant drift. Construct, dimension, magnitude, associated context window. | L365-374 |
| `AssessmentAnswer` | One answer to one assessment question. InstrumentID (ASI-12, CSI-8, etc.), QuestionID, Value. | L378-383 |
| `FounderProfileV2` | The master profile struct. Contains all 7 constructs, observed comm, context windows, complement, calibration, overrides, answers, accuracy, meta. | L387-460 |
| `EffectiveProfile` | The blended (α × trait + (1-α) × state) profile. This is what the chat system and UI read. | L1367-1379 |

### Engine Type

| Type | Description |
|------|-------------|
| `PairingEngine` | The engine itself. Holds the profile, signal channel, evidence log, NLP extractor, behavioral accumulators (recent ships, projects, approval latencies, daily event counts). |

---

## 4. The 7 Constructs (Φ1–Φ7)

Each construct is a `DualTrackDimension` with named dimensions:

| # | Construct | Dimensions | Assessment Instrument | Behavioral Sources |
|---|-----------|-----------|----------------------|-------------------|
| Φ1 | **Action Style** | FF (Fact Finder), FT (Follow Through), QS (Quick Start), IM (Implementor) | ASI-12 (12 forced-choice pairs) | Shipping cadence, context switches, completion ratio |
| Φ2 | **Communication DNA** | D (Driver), I (Influencer), S (Steady), C (Analytical) | CSI-8 (8 scenario cards) | NLP inference from messages (25+ features → DISC mapping) |
| Φ3 | **Energy Topology** | W (Wonder), N (Invention), D_disc (Discernment), G (Galvanizing), E (Enablement), T (Tenacity) | ETM-6 (1 drag-to-sort ranking) | Event patterns, approval selectivity |
| Φ4 | **Risk Disposition** | tolerance, speed, ambiguity, sunk_cost, loss_aversion, bias_to_action | RDS-6 (6 sliders, 0-100) | Decision speed from approvals, revenue response patterns |
| Φ5 | **Business Reality** | stage, revenue, debt, focus | Quick taps + conversation | Stripe revenue, MemberPress transactions, connected accounts |
| Φ6 | **Temporal Patterns** | chronotype, consistency, peak_hour | None (fully inferred) | Event timestamps, activity hour distribution |
| Φ7 | **Cognitive Style** | holistic, sequential, abstract, concrete | COG-8 (8 forced-choice pairs) | Document vocabulary, list usage, message structure |

### Dimension Naming Convention

- `D_disc` in Energy Topology = Discernment (avoids collision with DISC `D` = Driver)
- All scores are on a **0-10 scale** internally
- Risk dimensions are normalized from 0-100 slider input to 0-10 (÷10)

---

## 5. Dual-Track EMA System

This is the mathematical core. Every dimension in every construct has **two** exponential
moving averages running simultaneously:

```
TRAIT EMA (slow):
  λ_slow = 0.02  →  half-life ≈ 35 observations
  trait(t) = trait(t-1) × (1 - 0.02) + observation × 0.02

  This captures: who the founder IS — stable, long-term tendencies.
  Moves very slowly. Even a burst of unusual behavior barely budges it.

STATE EMA (fast):
  λ_fast = 0.15  →  half-life ≈ 4 observations
  state(t) = state(t-1) × (1 - 0.15) + observation × 0.15

  This captures: who the founder is RIGHT NOW — current operating mode.
  Responsive. Catches sprints, stalls, pressure shifts within days.
```

### Initialization

On the **first observation** for a dimension, both trait and state are initialized
to the observed value. No warm-up period — the profile starts working immediately
from the first assessment answer.

### Sigma (Variance Estimate)

A running estimate of trait variance, used for drift detection:

```
σ_trait(t) = σ_trait(t-1) × 0.95 + |observation - trait| × 0.05
Floor: σ_trait ≥ 0.1 (prevents division by zero in drift formula)
Initial: σ_trait = 2.0 (wide uncertainty at start)
```

### Drift Formula

```
drift(dim) = |state(dim) - trait(dim)| / σ_trait(dim)

Interpretation:
  drift < 1.0  →  Normal variance. No action.
  drift ∈ [1.0, 2.0)  →  Mild shift. α adjusts. Logged.
  drift ≥ 2.0  →  Significant shift. DriftEvent recorded. Context window may activate.
```

### Alpha (Blend Coefficient)

```
stability = 1 / (1 + drift)
α = 0.30 + 0.40 × stability

Range:
  No drift (drift=0):    α = 0.70 (70% trait, 30% state) — stable, trust the long-term
  High drift (drift=3):  α = 0.40 (40% trait, 60% state) — turbulent, trust the current
  Max drift:             α = 0.30 (30% trait, 70% state) — floor, always some trait anchor
```

### Effective Score

```
effective(dim) = α × trait(dim) + (1 - α) × state(dim)
```

This is the score used for all downstream decisions: complement, calibration, chat injection, UI display.

### Why Two EMAs Instead of One?

A single EMA either tracks too slowly (misses sprints/stalls) or too quickly (overreacts
to noise). The dual-track lets the system have both: a stable identity (trait) and a
responsive reading (state). The blend coefficient α automatically shifts trust between
them based on how much the founder's current behavior diverges from their baseline.

**Concrete example:**
- Founder's trait Quick Start = 8 (they're naturally fast starters)
- During a financial crunch, they slow down — state QS drops to 5
- drift = |5 - 8| / 1.5 = 2.0 → significant
- α drops from 0.70 to 0.43
- effective QS = 0.43 × 8 + 0.57 × 5 = 6.3
- Wirebot sees "moderately fast starter right now" — not the usual 8, not the temporary 5

---

## 6. Signal Processors

Each signal type has a dedicated processor. All processors:
1. Receive the Signal and a mutable EvidenceEntry pointer
2. Extract domain-specific information
3. Call `dt.UpdateEMA()` on relevant constructs
4. Append affected construct names to the evidence entry

### processMessage (chat messages)

**Input:** User message text from chat conversations.

**Actions:**
1. Increments `Meta.TotalMessagesAnalyzed`
2. Updates `CommunicationDNA` from NLP-inferred DISC scores (disc_D, disc_I, disc_S, disc_C)
3. Updates `ObservedComm` (directness, formality, detail_preference, emotion_expression, pace_preference, decision_style) using EMA with λ=0.10
4. Updates `ObservedComm.Confidence` = min(1.0, messages/200)
5. Updates `CognitiveStyle` from holistic_vs_sequential and abstract_vs_concrete features
6. Checks for financial pressure keywords → activates FINANCIAL_PRESSURE context window
7. Checks for life event keywords → activates LIFE_EVENT context window

**Constructs affected:** communication_dna, cognitive_style

### processEvent (scoreboard events)

**Input:** Scoreboard event metadata (event_type, lane, project, amount).

**Actions:**
1. Tracks ships (TASK_COMPLETED, PRODUCT_RELEASE, FEATURE_SHIPPED, CODE_PUBLISHED, EXTENSION_PUBLISHED, DOCS_PUBLISHED) in rolling window (last 100)
2. Tracks unique projects per day
3. Tracks daily event counts
4. Updates `TemporalPatterns.peak_hour` from event timestamp hour
5. Shipping cadence → Quick Start signal (5+ ships in 72h = sprint)
6. Context switch detection (5+ unique projects today = CONTEXT_EXPLOSION)
7. Revenue events → Business Reality (amount/100, capped at 10)
8. Stall detection (24h+ since last ship)
9. Recovery detection (ship after stall)
10. Celebration detection (PAYOUT_RECEIVED, PRODUCT_RELEASE)

**Context windows activated:** SHIPPING_SPRINT, CONTEXT_EXPLOSION, STALL, RECOVERY_PERIOD, CELEBRATION

**Constructs affected:** action_style, business_reality, temporal_patterns

### processAssessment (assessment answers)

**Input:** Batch of answers with instrument_id, question_id, value.

**Actions:**
1. Stores each answer in `profile.Answers`
2. Routes to instrument-specific scorer:
   - ASI-12 → `scoreASI()` → ActionStyle
   - CSI-8 → `scoreCSI()` → CommunicationDNA
   - ETM-6 → `scoreETM()` → EnergyTopology
   - RDS-6 → `scoreRDS()` → RiskDisposition
   - COG-8 → `scoreCOG()` → CognitiveStyle

### processApproval (event approve/reject actions)

**Input:** Approval action with latency_seconds.

**Actions:**
1. Tracks approval latencies in rolling window (last 100)
2. Fast approval (<5 min) → Quick Start signal (quick decisions)
3. Rejection → Discernment signal (selective evaluation)

**Constructs affected:** action_style, energy_topology

### processDocument (ingested documents)

**Input:** Document text content.

**Actions:**
1. Extracts NLP features from document text
2. Vocabulary richness → abstract cognitive style
3. List usage → sequential cognitive style

**Constructs affected:** cognitive_style

### processAccountData (connected account poller data)

**Input:** Provider name + provider-specific metrics.

**Actions:**
1. Tracks connected accounts list (deduped)
2. Stripe: monthly_revenue → Business Reality revenue
3. GitHub: weekly_commits → Action Style implementor

**Constructs affected:** business_reality

---

## 7. Assessment Instruments & Scoring Tables

### ASI-12 — Action Style Inventory

12 forced-choice pairs. Each question presents two options (A, B). Each maps to a
dimension (FF, FT, QS, IM) with a score value. The chosen option's dimension gets
updated via EMA.

| Question | Choice A | Choice B |
|----------|----------|----------|
| ASI-01 | QS = 9 (jump in) | FF = 8 (research first) |
| ASI-02 | QS = 8 (keep options open) | FT = 8 (make a plan) |
| ASI-03 | IM = 9 (hands-on build) | FF = 7 (analyze the design) |
| ASI-04 | FT = 9 (follow the process) | QS = 7 (skip unnecessary steps) |
| ASI-05 | FF = 9 (need more data) | IM = 8 (prototype it) |
| ASI-06 | QS = 8 (start now, adjust later) | IM = 9 (build it properly) |
| ASI-07 | FT = 8 (finish what I started) | FF = 9 (evaluate if it's still worth it) |
| ASI-08 | IM = 8 (build the MVP) | FT = 7 (document the spec) |
| ASI-09 | QS = 9 (launch immediately) | FT = 9 (thorough QA first) |
| ASI-10 | FF = 8 (read everything first) | QS = 8 (learn by doing) |
| ASI-11 | IM = 7 (physical prototype) | FT = 8 (systematic rollout) |
| ASI-12 | FF = 7 (detailed analysis) | IM = 8 (working demo) |

**Why forced-choice (ipsative)?** Prevents "all high" gaming. If you pick QS, you
can't also pick FF on the same question. The scores naturally balance.

### CSI-8 — Communication Style Inventory

8 scenario cards. Each has 4 response options mapping to D/I/S/C. The chosen
response's DISC dimension gets updated.

**All 8 scenarios offer all 4 DISC options.** Score values per scenario range from
7-9 to differentiate scenario-specific signal strength.

### ETM-6 — Energy Topology Map

1 drag-to-sort interaction. Founder ranks 6 work types from "gives me energy" to
"drains me." Position mapping:

| Position | Score | Classification |
|----------|-------|---------------|
| 1st | 10 | Working Genius |
| 2nd | 8 | Working Genius |
| 3rd | 6 | Working Competency |
| 4th | 4 | Working Competency |
| 5th | 2 | Working Frustration |
| 6th | 0 | Working Frustration |

**Dimensions:** W (Wonder), N (Invention), D_disc (Discernment), G (Galvanizing), E (Enablement), T (Tenacity)

### RDS-6 — Risk Disposition Scale

6 slider questions, each 0-100. Directly mapped to dimensions:

| Question | Dimension | What It Measures |
|----------|-----------|-----------------|
| RDS-01 | tolerance | "Move fast and fix mistakes vs. move slow and avoid them" |
| RDS-02 | ambiguity | "Comfortable deciding with incomplete information" |
| RDS-03 | sunk_cost | "Hard to quit even when I should" |
| RDS-04 | loss_aversion | "Think about worst-case scenarios" |
| RDS-05 | speed | "Quick decisions vs. deliberate decisions" |
| RDS-06 | bias_to_action | "Rather act than wait" |

**Normalization:** Slider 0-100 → dimension 0-10 (÷10)

### COG-8 — Cognitive Style Inventory

8 forced-choice pairs mapping to holistic/sequential and abstract/concrete.

---

## 8. NLP Feature Extraction

> **Implementation:** `pairing_nlp.go` (separate file, referenced by `pe.nlp`)

The NLP extractor produces 25+ features from raw text. No ML models — pure
lexical/statistical analysis for zero-latency extraction.

### Feature Categories

**Linguistic Features (from text content):**

| Feature | Formula | What It Signals |
|---------|---------|----------------|
| `avg_sentence_length` | words / sentences | Analytical (long) vs. Direct (short) |
| `vocabulary_richness` | unique_words / total_words | Openness, education level |
| `hedging_ratio` | hedge_phrases / sentences | Confidence, S-style |
| `action_verb_density` | action_verbs / total_verbs | D-style, Quick Start |
| `question_ratio` | questions / sentences | Curiosity, S/I-style |
| `exclamation_ratio` | exclamations / sentences | I-style, emotional expression |
| `first_person_ratio` | I/me/my / total_words | Self-focus vs. other-focus |
| `emoji_frequency` | emojis / total_words | Informality, I-style |
| `temporal_urgency` | urgent_words / temporal_words | Pace preference, QS |
| `imperative_ratio` | imperative sentences / sentences | D-style, directness |
| `list_usage` | list markers / sentences | Sequential cognitive style |

**Derived Features (computed from linguistic):**

| Feature | Formula | Maps To |
|---------|---------|---------|
| `directness` | f(1/sent_len, 1-hedging, action_verbs, imperatives) | Communication style |
| `formality` | f(vocab_richness, 1-emoji, 1-exclamation) | Communication style |
| `detail_preference` | f(sent_len, vocab_richness, question_ratio) | Communication style |
| `emotion_expression` | f(exclamation, emoji, 1-hedging) | Communication style |
| `pace_preference` | f(temporal_urgency, action_verbs) | Communication style |
| `decision_style` | f(imperatives, 1-questions, 1-hedging) | D vs. C style |
| `holistic_vs_sequential` | f(abstract_words, 1-list_usage, 1-numbers) | Cognitive style |
| `abstract_vs_concrete` | f(abstract_words, 1-concrete_words) | Cognitive style |

**Contextual Features (keyword detection):**

| Feature | Detection | What It Signals |
|---------|-----------|----------------|
| `financial_pressure` | Presence of: debt, money, afford, broke, budget, expenses, rent, bills, payroll, overdraft | FINANCIAL_PRESSURE context window |
| `life_event` | Presence of: health, hospital, family, divorce, baby, moving, funeral, surgery, accident | LIFE_EVENT context window |

### DISC Inference

Messages are mapped to DISC scores using the NLP features:

```
D = 0.30×imperative + 0.25×(1-hedging) + 0.20×action_verbs + 0.15×(1/sent_len) + 0.10×urgency
I = 0.30×exclamation + 0.25×emoji + 0.20×emotion + 0.15×(1-formality) + 0.10×question_ratio
S = 0.30×hedging + 0.25×question_ratio + 0.20×(1-urgency) + 0.15×long_sentences + 0.10×first_person
C = 0.30×vocab_richness + 0.25×(1-emoji) + 0.20×list_usage + 0.15×formality + 0.10×detail_pref
```

Scores are **relative** (they indicate signal strength for each style) and are
fed into the CommunicationDNA dual-track as `disc_D`, `disc_I`, `disc_S`, `disc_C`
features, each multiplied by 10 to match the 0-10 dimension scale.

---

## 9. Context Window System

Context windows are **auto-detected operating modes** that modulate how Wirebot
interprets behavior and calibrates its responses.

### Window Definitions

| Window | Detection Signals | Decay τ | Calibration Override |
|--------|-------------------|---------|---------------------|
| `FINANCIAL_PRESSURE` | Revenue drop, debt keywords in messages, Stripe payment failures | 72h (3 days) | Risk framing → cautious, data density ↑ |
| `SHIPPING_SPRINT` | 5+ ships in 72h | 48h (2 days) | Nudge frequency ↓, intensity ↓ (don't interrupt flow) |
| `RECOVERY_PERIOD` | Ship after stall | 72h (3 days) | Nudge intensity ↓↓, stall intervention → 24h |
| `CONTEXT_EXPLOSION` | 5+ unique projects in one day | 48h (2 days) | Focus prompts ↑, sequencing suggestions |
| `STALL` | 24h+ since last ship | 24h (1 day) | Nudge intensity ↑, stall intervention → 4h |
| `CELEBRATION` | PAYOUT_RECEIVED or PRODUCT_RELEASE | 24h (1 day) | Celebration intensity ↑ |
| `LIFE_EVENT` | Health/family/moving keywords in messages | 168h (7 days) | Reduce all pressure, extend deadlines |

### Activation Mechanics

```go
func (cw *ContextWindow) AddSignal(strength float64) {
    // Each signal pushes activation through a sigmoid
    cw.Activation = sigmoid(cw.Activation + strength * 0.3)
    // Sigmoid prevents activation from exceeding 1.0
    // Multiple signals compound: 3 signals with strength 0.5 → activation ≈ 0.72
}
```

### Decay Mechanics

```go
func (cw *ContextWindow) Decay(now time.Time) {
    hours := now.Sub(*cw.LastSignal).Hours()
    cw.Activation *= exp(-hours / τ_decay)
    // If activation drops below 0.05: window deactivates completely
}
```

### Multiple Active Windows

Context windows **compose**. Multiple can be active simultaneously. Their calibration
overrides apply additively (clamped to valid ranges).

- FINANCIAL_PRESSURE + SHIPPING_SPRINT = "desperate grind" mode
- RECOVERY_PERIOD + LIFE_EVENT = "full protection" mode
- STALL + FINANCIAL_PRESSURE = "urgent intervention" mode

### Window Lifecycle

```
Inactive (activation=0)
    │ signal received
    ▼
Activating (activation < 0.3) — not yet triggering calibration changes
    │ more signals
    ▼
Active (activation ≥ 0.3) — calibration overrides applied
    │ signals stop
    ▼
Decaying (activation decreasing) — overrides weakening proportionally
    │ activation < 0.05
    ▼
Deactivated (activation=0, ActivatedAt=nil, SignalCount=0)
```

---

## 10. Drift Detection

Drift detection runs after every signal. It scans all 6 tracked constructs
(BusinessReality and TemporalPatterns are excluded from drift tracking):

```
For each construct (action_style, communication_dna, energy_topology,
     risk_disposition, temporal_patterns, cognitive_style):
  For each dimension:
    if drift(dim) ≥ 2.0:
      → Record DriftEvent with timestamp, construct, dimension, magnitude
      → Associate with active context window (if any)
      → Increment Meta.TotalStateShifts
      → Log in evidence as profile_impact["drift.construct.dim"] = magnitude
```

### Drift Events (Historical)

Drift events are stored in `pe.driftHistory` for pattern learning (future: drift memory).
Each event records:

- **ID** — sequential
- **Timestamp** — when detected
- **Construct** — which of the 7 constructs
- **Dimension** — which specific dimension within the construct
- **Magnitude** — drift value (always ≥ 2.0 for recorded events)
- **Context** — which context window was active (if any) when drift occurred
- **ResolvedAt** — when drift returned below 1.0 (future)
- **RecoveryDays** — how long the drift lasted (future)

---

## 11. Complement Rebalancer

The complement vector answers: "Where should Wirebot allocate effort?"

### The Inverse Formula

```
For each dimension with an effective score:
  gap(dim) = max(0, 10 - effective(dim))

  // The LOWER the founder's score → the HIGHER Wirebot's effort
  // A founder with QS=9 → gap=1 → Wirebot barely supplements Quick Start
  // A founder with T=1  → gap=9 → Wirebot heavily supplements Tenacity

complement(dim) = gap(dim) / Σ(all gaps)

  // Normalized to sum to 1.0
  // Represents proportional effort allocation
```

### Input Sources

The rebalancer reads from:
- `ActionStyle.Effective` — FF, FT, QS, IM
- `EnergyTopology.Effective` — W, N, D_disc, G, E, T

### When It Runs

After **every signal** that changes any construct. The complement vector is always
fresh. This means if the founder's Quick Start drops from 8 to 5 during financial
pressure, the complement immediately shifts MORE effort to Quick Start supplementation.

### Example

```
Founder effective scores:
  FF=4, FT=3, QS=8, IM=6, W=6, N=8, D=4, G=2, E=1, T=1

Gaps:
  FF=6, FT=7, QS=2, IM=4, W=4, N=2, D=6, G=8, E=9, T=9
  Total = 57

Complement vector:
  Tenacity     = 9/57 = 15.8%  ← biggest gap
  Enablement   = 9/57 = 15.8%
  Galvanizing  = 8/57 = 14.0%
  Follow Thru  = 7/57 = 12.3%
  Fact Finder  = 6/57 = 10.5%
  Discernment  = 6/57 = 10.5%
  Implementor  = 4/57 =  7.0%
  Wonder       = 4/57 =  7.0%
  Quick Start  = 2/57 =  3.5%  ← smallest gap (founder's strength)
  Invention    = 2/57 =  3.5%
```

---

## 12. Calibration Engine

The calibration engine translates the abstract profile into concrete Wirebot behavior parameters.

### Communication Calibration

| Observed Metric | Condition | Calibration Change |
|-----------------|-----------|-------------------|
| Directness > 0.65 | High | MaxMessageWords → 200, LeadWith → "recommendation" |
| Directness < 0.35 | Low | MaxMessageWords → 500, LeadWith → "context" |
| Formality (any) | — | ToneFormality mirrors observed formality |
| EmotionExpr (any) | — | EmojiMirrorRatio + CelebrationIntensity mirror emotion |
| DISC D > C and D > 6 | D-primary | LeadWith → "recommendation", QuestionFrequency → "low" |
| DISC C > D and C > 6 | C-primary | LeadWith → "data", QuestionFrequency → "moderate" |

### Temporal Calibration

| Peak Hour | Chronotype | StandupHour |
|-----------|------------|-------------|
| 20-4 (8 PM - 4 AM) | Night owl | 11 AM |
| 5-8 (5 AM - 8 AM) | Early bird | 7 AM |
| 9-19 | Normal | 9 AM |

### Context Window Overrides

| Active Window | Override |
|---------------|---------|
| FINANCIAL_PRESSURE (>0.5) | RiskFraming → "cautious", DataDensity +0.2 |
| SHIPPING_SPRINT (>0.5) | NudgeFrequency ≥ 12h, NudgeIntensity -0.2 |
| RECOVERY_PERIOD (>0.5) | NudgeIntensity -0.3, StallIntervention → 24h |
| STALL (>0.5) | NudgeIntensity +0.3, StallIntervention → 4h |

All overrides apply **additively** and are **clamped** to valid ranges (0.0-1.0 for
intensities, reasonable minimums/maximums for hours).

---

## 13. Pairing Score Computation

The pairing score (0-100) indicates how well Wirebot knows the founder.
Computed from 9 weighted components:

| Component | Weight | What Fills It | Formula |
|-----------|--------|---------------|---------|
| S1: Action Style | 15% | ASI-12 assessment | 1.0 if ≥3 dims have data, 0.5 if ≥1, else 0 |
| S2: Communication | 10% | CSI-8 assessment | 1.0 if ≥3 dims, 0.5 if ≥1 |
| S3: Energy | 10% | ETM-6 assessment | 1.0 if ≥4 dims, 0.5 if ≥1 |
| S4: Risk | 10% | RDS-6 assessment | 1.0 if ≥4 dims, 0.5 if ≥1 |
| S5: Business (declared) | 15% | Quick taps + chat | 1.0 if ≥2 dims, 0.5 if ≥1 |
| S6: Business (verified) | 10% | Connected accounts | min(1.0, accounts/3) |
| S7: Comm inferred | 15% | Message scanner | min(1.0, messages/50) |
| S8: Behavioral | 10% | Usage over time | min(1.0, active_days/7) |
| S9: Continuous | 5% | All signals | min(1.0, signals/500) |

**Composite:** `Σ(componentᵢ × weightᵢ) × 100`

### Hard Caps

| Condition | Cap | Reason |
|-----------|-----|--------|
| Messages < 50 | Score ≤ 60 | Can't trust profile without comm scanning |
| Active days < 30 | Score ≤ 80 | Can't reach Bonded without 30 days behavioral |

### Levels

| Score | Level | Wirebot Behavior |
|-------|-------|-----------------|
| 0-15 | **Stranger** | Generic responses. Heavy nudges to complete assessment. |
| 16-35 | **Acquaintance** | Basic personalization. Knows name, stage, tone preference. |
| 36-60 | **Partner** | Solid model. Personalized recommendations. Complementary gap-filling starts. |
| 61-80 | **Trusted** | Deep model. Proactive suggestions. Pattern recognition. Auto-sequencing. |
| 81-100 | **Bonded** | Full sovereign mode. Anticipatory. Acts autonomously within trust bounds. |

---

## 14. Accuracy Convergence

The accuracy function implements the convergence equation from PAIRING_SCIENCE.md §14:

```
A(t) = 1 - (1 - A₀) × e^(-t/τ) × Π(1 - Δᵢ)

where:
  A₀ = 0.35 (initial accuracy from assessment alone)
  τ = 30 days (primary convergence time constant)
  Δ_chat = 0.15 × (1 - e^(-messages/100))
  Δ_events = 0.12 × (1 - e^(-events/500))
  Δ_docs = 0.08 × min(1.0, documents/5)
  Δ_accounts = 0.10 × min(1.0, accounts/3)
  Δ_retest = 0.05 × min(1.0, retests/2)      [not yet implemented]
  Δ_drift = 0.05 × min(1.0, drift_events/5)
```

### Trajectory

| Day | Messages | Events | Accuracy |
|-----|----------|--------|----------|
| 1 | 3 | 5 | ~35% (assessment only) |
| 7 | 40 | 60 | ~50% |
| 30 | 200 | 400 | ~72% |
| 90 | 700 | 1500 | ~88% |
| 365 | 4000 | 8000 | ~97% |

**Ceiling:** 0.97 (hard cap). Humans contain irreducible noise.

---

## 15. Override System

Founders can manually challenge any score through the Override system.

### Override Struct

```go
type ProfileOverride struct {
    ID           int64     // sequential
    Trait        string    // construct name (e.g., "action_style")
    Dimension    string    // dimension name (e.g., "QS")
    Value        float64   // founder's correction value
    Reason       string    // optional explanation text
    CreatedAt    time.Time
    Weight       float64   // current weight (decaying)
    Confirmed    bool      // behavioral data confirmed the override
    Contradicted bool      // behavioral data contradicted the override
}
```

### Weight Decay

```
Fresh override: weight = 0.30
After t days:   weight = 0.30 × e^(-t/30)

Day 0:   0.30
Day 7:   0.24
Day 14:  0.19
Day 21:  0.15
Day 30:  0.10
Day 60:  0.04 → effectively gone

Exception: If behavioral data CONFIRMS the override (observed score within ±1.0
of override value), the override becomes permanent with weight = 0.15.
```

---

## 16. Evidence Trail

Every signal produces an `EvidenceEntry` — a full record of what happened:

```go
type EvidenceEntry struct {
    ID                 int64
    Timestamp          time.Time
    SignalType         SignalType           // "message", "event", etc.
    Source             string               // "chat", "scoreboard", "stripe", etc.
    Summary            string               // "Chat message (14 words)"
    FeaturesExtracted  map[string]float64   // all NLP/metadata features
    ProfileImpact      map[string]float64   // what changed and by how much
    ConstructsAffected []string             // which constructs were touched
}
```

Evidence is stored in memory (capped at 10,000 entries). The UI reads it via
the `/v1/pairing/evidence` API endpoint.

### Example Evidence Entries

**Chat message:**
```json
{
  "summary": "Chat message (14 words)",
  "features_extracted": {
    "disc_D": 0.45, "disc_I": 0.22, "disc_S": 0.12, "disc_C": 0.28,
    "directness": 0.78, "formality": 0.35, "action_verb_density": 0.21
  },
  "profile_impact": {},
  "constructs_affected": ["communication_dna", "cognitive_style"]
}
```

**Scoreboard event:**
```json
{
  "summary": "Scoreboard event: PRODUCT_RELEASE",
  "features_extracted": {},
  "profile_impact": { "context.CELEBRATION": 0.6 },
  "constructs_affected": ["action_style", "temporal_patterns"]
}
```

**Significant drift:**
```json
{
  "summary": "Chat message (8 words)",
  "profile_impact": { "drift.action_style.QS": 2.3 },
  "constructs_affected": ["communication_dna", "cognitive_style"]
}
```

---

## 17. Chat Context Injection

`GetChatContextSummary()` produces a concise 3-5 line text block injected into the
Wirebot system message instead of the raw PAIRING.md document.

### Format

```
Founder: Driver-primary (72%), high Quick Start
Communication: recommendation first, 200-word max, formality=35%
Complement focus: Tenacity (16%), Enablement (16%), Galvanizing (14%)
Active contexts: SHIPPING_SPRINT
Pairing: 67/100 (Trusted) | Accuracy: 82%
```

### Why This Replaces Raw PAIRING.md Injection

Previously, the entire PAIRING.md file (~350 lines) was injected into every chat
message when pairing was incomplete. This wasted context window tokens and didn't
provide actionable calibration data.

The new injection is:
- **5 lines** instead of 350
- **Computed from live data** instead of static questions
- **Includes calibration parameters** that Wirebot directly uses
- **Updates with every signal** — each chat message makes the next one better

---

## 18. Persistence & Lifecycle

### File Location

```
/data/wirebot/pairing/profile.json
```

### Save Strategy

| Trigger | When |
|---------|------|
| Every 10 signals | Batch save for efficiency |
| Every 60 seconds | Time-based fallback |
| On explicit Save() call | API or CLI triggered |
| On shutdown (future) | Graceful shutdown hook |

### Load Strategy

On engine startup (`NewPairingEngine(path)`):
1. Try to read and unmarshal `profile.json`
2. If valid v2 profile → load it, log stats
3. If missing or invalid → create fresh `NewFounderProfile()`

### Profile Size

Fully populated profile with 100 answers, 10K evidence, 100 drift events: ~50KB JSON.
Typical early profile: ~5KB.

### Lifecycle

```
Server Start
    │
    ▼
NewPairingEngine(path)
    │ loads or creates profile
    │
    ▼
pe.Start()
    │ spawns processLoop goroutine (reads from signalChan)
    │ spawns periodicTasks goroutine (5-min ticker for decay + score recompute)
    │
    ▼
Running...
    │ pe.Ingest(sig) called from HTTP handlers, event hooks, pollers
    │ processLoop drains signalChan serially
    │ periodicTasks decays context windows, recomputes pairing score
    │
    ▼
Shutdown (future)
    │ close(pe.signalChan)
    │ processLoop exits
    │ final pe.Save()
```

---

## 19. API Endpoints

> **Implementation:** `pairing_api.go` (planned)

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| GET | `/v1/pairing/status` | Score + level + basic stats | Member |
| GET | `/v1/pairing/profile` | Full FounderProfileV2 JSON | Member |
| GET | `/v1/pairing/profile/effective` | Effective scores only (blended) | Member |
| GET | `/v1/pairing/evidence` | Evidence log (paginated, filterable by type/source) | Member |
| GET | `/v1/pairing/evidence/:id` | Single evidence entry with full features | Member |
| GET | `/v1/pairing/formulas` | Current formula state: all inputs + outputs for every formula | Member |
| GET | `/v1/pairing/accuracy` | Accuracy metrics + convergence curve data | Member |
| GET | `/v1/pairing/drift` | Current drift readings + context windows + drift history | Member |
| GET | `/v1/pairing/complement` | Current complement vector + allocation percentages | Member |
| GET | `/v1/pairing/predictions` | Prediction log + accuracy track record | Member |
| GET | `/v1/pairing/insights` | Latest inferences + deltas + active contexts | Member |
| POST | `/v1/pairing/answers` | Submit assessment answers (batch) | Member |
| POST | `/v1/pairing/override` | Submit manual override (trait, dimension, value, reason) | Member |
| GET | `/v1/pairing/overrides` | List active overrides + decay status | Member |
| DELETE | `/v1/pairing/overrides/:id` | Remove an override | Member |
| POST | `/v1/pairing/scan` | Trigger communication scan on chat history | Admin |
| DELETE | `/v1/pairing/reset` | Full profile reset (requires confirmation token) | Admin |

---

## 20. CLI Commands

> **Implementation:** `/usr/local/bin/wb` additions (planned)

| Command | Description |
|---------|-------------|
| `wb profile` | Display full profile summary (equalizer-style ASCII) |
| `wb profile effective` | Show effective scores only |
| `wb profile json` | Dump raw FounderProfileV2 JSON |
| `wb pair` | Show pairing score, level, progress to next level |
| `wb pair start` | Open assessment URL or print inline questions |
| `wb evidence` | Show recent evidence entries |
| `wb evidence --type message` | Filter evidence by signal type |
| `wb drift` | Show current drift readings across all constructs |
| `wb drift history` | Show historical drift events |
| `wb complement` | Show complement vector (ASCII bar chart) |
| `wb accuracy` | Show accuracy metrics + convergence position |
| `wb override <trait> <dim> <value>` | Submit manual override |
| `wb override list` | List active overrides |
| `wb calibration` | Show current calibration parameters |
| `wb contexts` | Show active context windows |

---

## 21. Wiring Into Scoreboard

### Event Insert Hook

In the scoreboard's event insert handler, after inserting an event to SQLite:

```go
// After: db.Exec("INSERT INTO events ...")
pe.Ingest(Signal{
    Type:      SignalEvent,
    Source:    source,
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "event_type": eventType,
        "lane":       lane,
        "project":    project,
        "amount":     amount,
    },
})
```

### Chat Message Hook

In the chat proxy handler, after receiving the user's message:

```go
// After: receiving user message, before proxying to OpenClaw
pe.Ingest(Signal{
    Type:      SignalMessage,
    Source:    "chat",
    Timestamp: time.Now(),
    Content:   userMessage,
})
```

### Event Approval Hook

In the approve/reject handler:

```go
// After: UPDATE events SET status='approved'
pe.Ingest(Signal{
    Type:      SignalApproval,
    Source:    "scoreboard",
    Timestamp: time.Now(),
    Metadata: map[string]interface{}{
        "action":          "approve", // or "reject"
        "latency_seconds": latency,
        "event_id":        eventID,
    },
})
```

### Chat Context Injection Upgrade

Replace the current PAIRING.md injection with:

```go
// Instead of: reading PAIRING.md and injecting the whole file
// Now:
pairingSummary := pe.GetChatContextSummary()
systemMsg += "\n\n" + pairingSummary
```

---

## 22. File Map

```
cmd/scoreboard/
├── main.go              ← Existing scoreboard (5524 lines)
├── pairing.go           ← Engine core: types, Signal Bus, EMA, drift, complement,
│                           calibration, scoring, accuracy, evidence (1624 lines)
├── pairing_nlp.go       ← NLP feature extraction, word lists, DISC inference (planned)
├── pairing_api.go       ← 17 HTTP endpoint handlers (planned)
├── go.mod
├── go.sum
├── static/              ← Built PWA files
│   └── sw.js
└── ui/
    └── src/
        └── lib/
            ├── Profile.svelte       ← Equalizer view (planned)
            ├── PairingFlow.svelte   ← Assessment cards (planned)
            ├── ProfileRadar.svelte  ← 7-construct spider chart (planned)
            └── ... (existing views)

docs/
├── PAIRING.md           ← v1 protocol (22 questions, conversation flow)
├── PAIRING_V2.md        ← v2 UI/UX spec (assessment cards, equalizer, evidence, overrides)
├── PAIRING_SCIENCE.md   ← Scientific specification (every formula, Bayesian proof, accuracy)
└── PAIRING_ENGINE.md    ← THIS FILE (implementation documentation)
```

---

## Appendix A: Formula Quick Reference

| Formula | Code Location | Spec Reference |
|---------|---------------|---------------|
| Trait EMA | `DualTrackDimension.UpdateEMA()` | PAIRING_SCIENCE.md §0 |
| State EMA | `DualTrackDimension.UpdateEMA()` | PAIRING_SCIENCE.md §0 |
| Drift detection | `DualTrackDimension.UpdateEMA()` | PAIRING_SCIENCE.md §0 |
| Alpha blend | `DualTrackDimension.UpdateEMA()` | PAIRING_SCIENCE.md §0 |
| Effective score | `DualTrackDimension.UpdateEMA()` | PAIRING_SCIENCE.md §0 |
| Context window activation | `ContextWindow.AddSignal()` | PAIRING_SCIENCE.md §0 |
| Context window decay | `ContextWindow.Decay()` | PAIRING_SCIENCE.md §0 |
| Override weight decay | `ProfileOverride.CurrentWeight()` | PAIRING_V2.md (Override system) |
| Complement vector | `ComplementVector.Rebalance()` | PAIRING_SCIENCE.md §0 |
| Pairing score composite | `PairingEngine.recomputePairingScore()` | PAIRING_SCIENCE.md §8 |
| Accuracy convergence | `PairingEngine.computeAccuracy()` | PAIRING_SCIENCE.md §14 |
| DISC inference | `NLPExtractor.InferDISC()` | PAIRING_SCIENCE.md §4.2 |

## Appendix B: Dimension Code → Human Name

| Code | Construct | Human Name |
|------|-----------|-----------|
| FF | Φ1 Action Style | Fact Finder |
| FT | Φ1 Action Style | Follow Through |
| QS | Φ1 Action Style | Quick Start |
| IM | Φ1 Action Style | Implementor |
| D | Φ2 Communication | Driver |
| I | Φ2 Communication | Influencer |
| S | Φ2 Communication | Steady |
| C | Φ2 Communication | Analytical |
| W | Φ3 Energy | Wonder |
| N | Φ3 Energy | Invention |
| D_disc | Φ3 Energy | Discernment |
| G | Φ3 Energy | Galvanizing |
| E | Φ3 Energy | Enablement |
| T | Φ3 Energy | Tenacity |
| tolerance | Φ4 Risk | Risk Tolerance |
| speed | Φ4 Risk | Decision Speed |
| ambiguity | Φ4 Risk | Ambiguity Comfort |
| sunk_cost | Φ4 Risk | Sunk-Cost Sensitivity |
| loss_aversion | Φ4 Risk | Loss Aversion |
| bias_to_action | Φ4 Risk | Bias to Action |
| stage | Φ5 Business | Business Stage |
| revenue | Φ5 Business | Revenue Level |
| debt | Φ5 Business | Debt Level |
| focus | Φ5 Business | Focus Capacity |
| chronotype | Φ6 Temporal | Chronotype |
| consistency | Φ6 Temporal | Activity Consistency |
| peak_hour | Φ6 Temporal | Peak Activity Hour |
| holistic | Φ7 Cognitive | Holistic Thinking |
| sequential | Φ7 Cognitive | Sequential Thinking |
| abstract | Φ7 Cognitive | Abstract Thinking |
| concrete | Φ7 Cognitive | Concrete Thinking |

---

*This document is the implementation companion to PAIRING_SCIENCE.md (the math)
and PAIRING_V2.md (the UI). Together they form the complete Pairing System specification.
Every struct in the code maps to a concept in the spec. Every formula in the spec
maps to a function in the code.*
