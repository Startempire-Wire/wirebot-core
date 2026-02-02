# Wirebot Pairing — Scientific Specification

> Every score is a formula. Every weight is justified. Every inference has a confidence interval.
> The system gets smarter with every message, event, document, and connected account.

> **Core axiom: The profile is NEVER static.** People behave differently under stress vs. calm,
> in debt vs. flush, single-project vs. scattered, energized vs. burned out, winter vs. summer.
> The profile is a **living signal** that shifts as the founder shifts — and Wirebot's complement
> shifts with it. There is no "done." There is only "current reading."

---

## 0. The Living Profile Principle

### Why Static Profiles Fail

Traditional assessments (StrengthsFinder, DISC, Myers-Briggs) produce a **snapshot** — a fixed
label from a single sitting. But founders don't operate in a single state:

- **Under debt pressure:** A normally creative (Wonder-genius) founder becomes survival-focused,
  acting more like a Tenacity operator. Wirebot must detect this and shift from feeding their
  creativity to providing structure for their new emergency mode.

- **After a big win:** A normally cautious (high Fact Finder) founder may spike in Quick Start
  behavior — riding momentum, starting 3 new things. Wirebot must detect the spike, ride the
  wave with them, but also protect against the crash.

- **During burnout:** All scores compress toward zero. Activity drops, messages get shorter,
  approval latency spikes. Wirebot must detect this before the founder admits it, shift from
  pushing to protecting (Pillar 10: Sustainability).

- **Seasonal life shifts:** Tax season, holidays, school schedules, health events, relationship
  changes — all alter the operating context. The profile must absorb these without the founder
  explicitly reporting them.

### The Always-Running Background Algorithm

The pairing engine is **not** something the founder interacts with. It is a **daemon** — a
continuous background process that:

1. **Observes** every signal (message, event, document, account data, timing)
2. **Extracts** features in real-time
3. **Updates** the Bayesian profile (prior → posterior, every time)
4. **Detects** state shifts (drift from baseline)
5. **Adjusts** Wirebot's calibration parameters automatically
6. **Never stops**

The founder sees the EFFECTS (Wirebot communicates differently, nudges differently, recommends
differently) but never has to think about the algorithm. It just works. It just adapts.

### Trait vs. State Separation

Every dimension has TWO readings:

```
TRAIT = long-term stable tendency (updated slowly, high inertia)
STATE = current operating mode (updated fast, low inertia)

Wirebot calibrates to: α × TRAIT + (1-α) × STATE
  where α = state_stability_coefficient (see §0.2)

When STATE ≈ TRAIT: α ≈ 0.70 (trait-dominant, stable period)
When STATE diverges from TRAIT: α drops toward 0.30 (state-dominant, transition period)
```

This means:
- In stable times, Wirebot operates on the deep profile (trait)
- In turbulent times, Wirebot operates on the current reading (state)
- The blend is automatic and continuous

### State Shift Detection

```
For each construct Φ:

  trait(Φ) = EMA(Φ, λ_slow)     where λ_slow = 0.02  (half-life ≈ 35 data points)
  state(Φ) = EMA(Φ, λ_fast)     where λ_fast = 0.15  (half-life ≈ 4 data points)

  drift(Φ) = |state(Φ) - trait(Φ)| / σ_trait(Φ)

  if drift(Φ) < 1.0:  → Normal variance. No action.
  if drift(Φ) ∈ [1.0, 2.0):  → Mild shift. Adjust α. Log.
  if drift(Φ) ≥ 2.0:  → Significant shift. Flag. Adjust calibration. Optionally surface:
     "I've noticed a shift in [observable behavior]. Want to talk about it?"

  // State stability coefficient:
  state_stability(Φ) = 1 / (1 + drift(Φ))
  α(Φ) = 0.30 + 0.40 × state_stability(Φ)
  // Range: 0.30 (pure state, high drift) to 0.70 (pure trait, no drift)
```

### Seasonal & Contextual Modulation

The algorithm tracks **context windows** that modulate interpretation:

```
Context Windows (detected automatically from events + calendar + behavior):

  FINANCIAL_PRESSURE:
    Signal: Revenue drop, debt events, subscription cancellations, "money" in messages
    Effect: Bias interpretation toward survival mode
    Calibration: More revenue-first recommendations, tighter accountability, reduce vanity

  SHIPPING_SPRINT:
    Signal: Ship count > 2σ above mean for 3+ consecutive days
    Effect: Ride the momentum, don't interrupt flow
    Calibration: Reduce check-in frequency, increase supply of next tasks, celebrate

  RECOVERY_PERIOD:
    Signal: Activity < 0.5σ below mean for 2+ days after sprint
    Effect: Protect, don't push
    Calibration: Lighter nudges, suggest rest, defer non-urgent

  CONTEXT_EXPLOSION:
    Signal: Unique projects per day > 2σ above mean
    Effect: Founder is scattered — intervene
    Calibration: Increase focus prompts, surface sequencing, ask "which ONE?"

  STALL:
    Signal: Zero ships for > stall_threshold hours
    Effect: Graduated intervention
    Calibration: Gentle at 4h, direct at 8h, urgent at 24h

  CELEBRATION:
    Signal: Major revenue event, product launch, milestone hit
    Effect: Amplify the win, then redirect energy
    Calibration: Big celebration → "What's next?" within 4h

  LIFE_EVENT:
    Signal: Schedule disruption, mentions of health/family/travel, activity pattern break
    Effect: Adapt expectations, don't penalize
    Calibration: Reduce accountability pressure, extend deadlines, hold context
```

Each context window has:
- **Detection formula** (from signals above)
- **Activation threshold** (how many signals before activating)
- **Decay rate** (how quickly the window closes after signals stop)
- **Calibration overrides** (what Wirebot parameters change)

```
window_activation(W) = sigmoid(Σ signal_strength(s) - threshold(W))

// Active windows modulate the calibration engine:
For each active window W:
  Apply calibration_overrides(W) with strength = window_activation(W)

// Multiple windows can be active simultaneously:
// FINANCIAL_PRESSURE + SHIPPING_SPRINT = "desperate grind" mode
// RECOVERY_PERIOD + LIFE_EVENT = "full protection" mode
// The overrides compose additively (clamped to valid ranges)
```

### The Complement Shifts With the Founder

This is the key insight: **Wirebot is not a fixed complement to a fixed profile.**

```
At time t:

  founder_state(t) = {
    action: [FF=3, FT=2, QS=9, IM=5],    // currently very Quick Start heavy
    energy: [W=6, N=8, D=4, G=2, E=1, T=1], // inventing hard, everything else neglected
    risk: { tolerance: 0.85, speed: 0.90 },   // moving fast right now
    context: [SHIPPING_SPRINT, FINANCIAL_PRESSURE]
  }

  wirebot_complement(t) = inverse(founder_state(t))
  // Founder is QS=9 right now? Wirebot provides FT (structure, checklists, tracking)
  // Founder energy is all Invention? Wirebot handles Tenacity + Enablement
  // Founder is high risk tolerance? Wirebot provides the safety-net thinking
  // Sprint + Financial Pressure? Wirebot channels the energy toward revenue items

  // But 2 weeks later:
  founder_state(t+14d) = {
    action: [FF=5, FT=4, QS=4, IM=6],     // sprint ended, back to building
    energy: [W=3, N=5, D=6, G=3, E=4, T=4], // more balanced, evaluating what shipped
    risk: { tolerance: 0.50, speed: 0.40 },   // cautious after the sprint
    context: [RECOVERY_PERIOD]
  }

  wirebot_complement(t+14d) = inverse(founder_state(t+14d))
  // Founder is now balanced? Wirebot shifts to light Galvanizing (distribution push)
  // Founder is cautious? Wirebot provides gentle encouragement, not deadline pressure
  // Recovery mode? Wirebot protects rest, prepares next phase
```

**The complement formula:**

```
For each dimension D with range [1, 10]:

  complement_need(D, t) = 10 - founder_state(D, t)

  // Normalize so complement effort sums to 1.0:
  wirebot_effort(D, t) = complement_need(D, t) / Σ complement_need(all D, t)

  // Apply context window modulation:
  For each active window W:
    wirebot_effort(D, t) *= context_modifier(W, D)

  // Re-normalize after modulation:
  wirebot_effort = normalize(wirebot_effort)
```

### Background Processing Cadence

```
| Process | Frequency | What It Does |
|---------|-----------|-------------|
| Message feature extraction | On every message | Extract NLP features, update EMA |
| Event signal processing | On every event | Behavioral signals, temporal patterns |
| State EMA update | On every signal | Fast-moving state estimate |
| Trait EMA update | On every signal | Slow-moving trait estimate |
| Drift detection | Every 10 signals | Check state vs. trait divergence |
| Context window evaluation | Every 10 signals | Activate/deactivate context windows |
| Calibration parameter update | On drift change | Adjust Wirebot behavior params |
| Complement rebalance | On calibration change | Shift effort allocation |
| Weekly batch analysis | Sunday midnight | Full behavioral pattern recompute |
| Monthly stability check | 1st of month | Test-retest on assessment traits |
| Document ingestion | On upload/discovery | Full pipeline (§6) |
| Connected account refresh | Per poller schedule | Ground truth data update |
```

**All of this runs without the founder doing anything.** The assessment (Phase 1-3) is the
seed. Everything after that is passive observation, continuous inference, and automatic adjustment.

The founder's only experience is: "Wirebot just... gets it. And it keeps getting better."

---

## Table of Contents

1. [Measurement Framework](#1-measurement-framework)
2. [Instrument Design](#2-instrument-design)
3. [Scoring Formulas](#3-scoring-formulas)
4. [Communication Inference Engine](#4-communication-inference-engine)
5. [Behavioral Signal Processing](#5-behavioral-signal-processing)
6. [Document & Context Ingestion](#6-document--context-ingestion)
7. [Bayesian Profile Updating](#7-bayesian-profile-updating)
8. [Composite Pairing Score](#8-composite-pairing-score)
9. [Calibration Engine](#9-calibration-engine)
10. [Confidence & Validity](#10-confidence--validity)
11. [Continuous Learning Loop](#11-continuous-learning-loop)

---

## 1. Measurement Framework

### Construct Model

Wirebot measures **7 latent constructs** that together predict how to best serve a founder:

```
Founder Profile = f(
  Φ₁: Action Style,           — HOW they instinctively act
  Φ₂: Communication Style,    — HOW they process and express
  Φ₃: Energy Topology,        — WHAT gives/drains energy
  Φ₄: Risk Disposition,       — HOW they handle uncertainty
  Φ₅: Business Reality,       — WHERE they actually are
  Φ₆: Temporal Patterns,      — WHEN they operate
  Φ₇: Cognitive Style         — HOW they think and decide
)
```

Each construct is measured through **multiple methods** (triangulation):

| Method | Type | Reliability | When |
|--------|------|-------------|------|
| **Self-report assessment** | Active, structured | Moderate (α ≈ 0.70-0.85) | Phase 1-3 (one-time + periodic retest) |
| **Linguistic analysis** | Passive, unstructured | Improves with n (converges at n ≈ 200 messages) | Continuous |
| **Behavioral telemetry** | Passive, structured | High for stable traits (r ≈ 0.85 at 30d) | Continuous |
| **Document content analysis** | Passive, unstructured | Varies by document type | On ingestion |
| **Connected account data** | Passive, structured | Very high (ground truth) | Continuous |
| **Interaction pattern analysis** | Passive, structured | Moderate-high (r ≈ 0.75 at 14d) | Continuous |

### Psychometric Standards

Every instrument in this system targets:

| Criterion | Target | How Verified |
|-----------|--------|-------------|
| **Internal consistency** (Cronbach's α) | ≥ 0.70 | Item-total correlations across users |
| **Test-retest reliability** | r ≥ 0.75 at 30 days | Periodic re-assessment prompts |
| **Convergent validity** | r ≥ 0.50 with behavioral measures | Self-report ↔ observed correlation |
| **Discriminant validity** | r ≤ 0.30 between unrelated constructs | Cross-construct correlation matrix |
| **Predictive validity** | Wirebot calibration improves outcomes | A/B: calibrated vs. uncalibrated response quality |

---

## 2. Instrument Design

### 2.1 Action Style Inventory (ASI-12)

**Construct:** Conative instinct — how the founder naturally takes action.

**Model:** 4 modes × 3 items each = 12 forced-choice pairs.

Each item presents two statements. The founder picks which resonates MORE.
Forced-choice (ipsative) prevents acquiescence bias and social desirability inflation.

**Item matrix:**

| # | Statement A (Mode X) | Statement B (Mode Y) | Measures |
|---|---------------------|----------------------|----------|
| 1 | "I'd rather start building than keep planning" | "I'd rather have a solid plan before I touch anything" | QS vs. FF |
| 2 | "I like to keep my options open and improvise" | "I like to have systems and checklists for everything" | QS vs. FT |
| 3 | "I need to touch and prototype things to understand them" | "I can work with abstract concepts and spreadsheets" | IM vs. FF |
| 4 | "I research extensively before committing" | "I trust my gut and course-correct later" | FF vs. QS |
| 5 | "I document my processes so they're repeatable" | "Every situation is different — rigid processes slow me down" | FT vs. QS |
| 6 | "I prefer building physical/tangible things" | "I prefer building systems, strategies, and frameworks" | IM vs. FT |
| 7 | "I want ALL the data before deciding" | "I want the key facts and then I decide fast" | FF vs. QS |
| 8 | "I follow the process I've set even when it's tedious" | "I skip steps that feel unnecessary in the moment" | FT vs. QS |
| 9 | "I make things with my hands / in code / tangibly" | "I orchestrate — I design it, someone/something else builds it" | IM vs. FF |
| 10 | "I read the manual and case studies first" | "I learn by doing, not reading" | FF vs. IM |
| 11 | "Consistency is how I win" | "Bursts of intensity is how I win" | FT vs. QS |
| 12 | "I need a physical workspace / whiteboard / tools" | "I need a quiet room and my thoughts" | IM vs. FF |

**Legend:** FF = Fact Finder, FT = Follow Through, QS = Quick Start, IM = Implementor

**Scoring:**

```
For each mode M ∈ {FF, FT, QS, IM}:

  items_involving(M) = set of items where M appears as option A or B
  selections(M) = number of times M was selected

  raw_score(M) = selections(M) / |items_involving(M)|

  // Normalize to 1-10 scale with mean=5
  z(M) = (raw_score(M) - μ_raw) / σ_raw
  scaled_score(M) = round(5 + 2.5 × z(M))
  clamped_score(M) = clamp(scaled_score(M), 1, 10)
```

**Ipsative constraint:** Because forced-choice, Σ raw_score(M) = constant. This is a feature — prevents all-high inflation. A founder who is high Quick Start MUST be lower on something else.

**Internal consistency check:** If an item pair is answered inconsistently with related pairs (e.g., picks "research first" on Q1 but "trust gut" on Q4, which both measure FF vs. QS), the inconsistency is flagged and optionally a tiebreaker item is inserted.

```
consistency(M) = 1 - (contradictions_involving(M) / items_involving(M))
if consistency(M) < 0.50:
  insert_tiebreaker_item(M)
  flag_low_reliability(M)
```

---

### 2.2 Communication Style Inventory (CSI-8)

**Construct:** Interpersonal communication preference under business pressure.

**Model:** DISC-adapted. 4 styles × 8 scenario items. Each scenario has 4 response options, one per style. Founder ranks top 2 (forced partial rank).

**Scoring:**

```
For each style S ∈ {D, I, S, C}:

  // First-choice selections weighted 2, second-choice weighted 1
  weighted_score(S) = Σᵢ (2 × first_choice(S, i) + 1 × second_choice(S, i))
  max_possible = 2 × 8 = 16

  raw_pct(S) = weighted_score(S) / max_possible

  // Ipsative normalization: percentages should sum to ~1.0
  normalized(S) = raw_pct(S) / Σ raw_pct(all styles)
```

**Output:**
```json
{
  "primary": "D",         // highest normalized
  "secondary": "C",       // second highest
  "tertiary": "I",        // third
  "stress_avoid": "S",    // lowest (what they avoid under pressure)
  "scores": { "D": 0.38, "I": 0.22, "S": 0.12, "C": 0.28 },
  "confidence": 0.72      // based on consistency + response time variance
}
```

**Confidence calculation:**

```
// Items where first and second choice are consistent with overall profile
profile_consistent_items = count(items where primary or secondary matches top-2 selections)

consistency_ratio = profile_consistent_items / 8

// Fast responses on forced-choice indicate genuine preference (< 5s)
// Slow responses indicate deliberation/uncertainty (> 15s)
avg_response_ms = mean(response_times)
speed_confidence = sigmoid((5000 - avg_response_ms) / 2000)  // peaks at fast responses

confidence(CSI) = 0.6 × consistency_ratio + 0.4 × speed_confidence
```

---

### 2.3 Energy Topology Map (ETM-6)

**Construct:** What work activities energize vs. drain the founder. Based on Lencioni's Working Genius framework but adapted for solo/small founders.

**Model:** 6 work activity types, drag-to-rank from "most energizing" to "most draining."

**Activities:**
```
W = Wonder     (pondering possibilities, questioning the status quo)
N = Invention  (creating novel solutions, designing new things)
D = Discernment (evaluating quality, curating, choosing wisely)
G = Galvanizing (rallying people, creating urgency, selling)
E = Enablement  (supporting execution, coordinating, enabling others)
T = Tenacity    (pushing through to completion, grinding, shipping)
```

**Scoring:**

```
// Rank position → standardized score (Thurstone scaling)
position_to_score = {1: 10, 2: 8, 3: 6, 4: 4, 5: 2, 6: 0}

For each activity A:
  score(A) = position_to_score[rank_position(A)]

// Classification
genius(A)      if score(A) ≥ 8   (top 2)
competency(A)  if 4 ≤ score(A) ≤ 6  (middle 2)
frustration(A) if score(A) ≤ 2   (bottom 2)
```

**Wirebot complement vector:**

```
// Wirebot's effort allocation inversely proportional to founder energy
wirebot_weight(A) = (10 - score(A)) / Σ(10 - score(all))

// This means: Wirebot spends the MOST effort on what DRAINS the founder
// Example: Founder frustration = Tenacity (score=0)
//   wirebot_weight(T) = 10/30 = 0.33  ← 33% of Wirebot effort on follow-through
```

**Stability metric:** Re-tested at 30-day intervals. Working Genius is trait-level (stable), so test-retest r should be ≥ 0.80. If r < 0.60, flag as unstable and increase weight of behavioral validation.

---

### 2.4 Risk Disposition Scale (RDS-6)

**Construct:** Multi-dimensional risk attitude relevant to business decisions.

**Model:** 6 Likert-type slider items (0-100 continuous scale). Each item loads on 1-2 sub-dimensions.

**Sub-dimensions and items:**

| # | Item Text | Loads On | Polarity |
|---|-----------|----------|----------|
| 1 | "I'd rather move fast and fix mistakes than move slow and avoid them" | Speed, Risk Tolerance | + |
| 2 | "I'm comfortable making decisions with incomplete information" | Ambiguity Tolerance | + |
| 3 | "When I commit to something, I find it hard to quit even when I should" | Sunk-Cost Sensitivity | + (reverse = bad) |
| 4 | "I think about worst-case scenarios before acting" | Loss Aversion | + (protective) |
| 5 | "I prefer small, safe bets over big, uncertain ones" | Risk Aversion | + (reverse-scored for risk tolerance) |
| 6 | "I often take action before I feel ready" | Bias to Action | + |

**Scoring:**

```
// Raw values are 0-100 slider positions
// Some items are reverse-scored
reverse = {3, 4, 5}  // higher = more cautious

For item i:
  adjusted(i) = 100 - raw(i)  if i ∈ reverse
  adjusted(i) = raw(i)        otherwise

// Sub-dimension extraction via simple factor model:
risk_tolerance = 0.40 × adj(1) + 0.30 × adj(5) + 0.30 × adj(6)
decision_speed = 0.50 × adj(1) + 0.50 × adj(6)
ambiguity_tolerance = 0.60 × adj(2) + 0.40 × adj(6)
sunk_cost_sensitivity = adj(3)  // single item, note reverse interpretation
loss_aversion = adj(4)          // single item
bias_to_action = 0.40 × adj(1) + 0.30 × adj(2) + 0.30 × adj(6)

// All normalized to 0.00-1.00
normalize(x) = x / 100
```

**Interpretation rules:**

```
if risk_tolerance > 0.70 AND sunk_cost_sensitivity > 0.70:
  → "Charges forward but can't quit — dangerous combination"
  → Wirebot flag: provide explicit kill criteria on every project

if risk_tolerance < 0.30 AND decision_speed < 0.30:
  → "Analysis paralysis risk"
  → Wirebot flag: add deadline pressure, reduce options presented

if loss_aversion > 0.70 AND bias_to_action > 0.70:
  → "Anxious executor — acts fast but worries about downsides"
  → Wirebot flag: proactive risk assessment before recommending, provide safety nets
```

---

### 2.5 Cognitive Style Inventory (COG-8)

**Construct:** How the founder processes information, solves problems, and forms conclusions.

**Model:** 8 items measuring 4 cognitive sub-dimensions. Forced-choice pairs.

**Sub-dimensions:**

| Dimension | Pole A | Pole B |
|-----------|--------|--------|
| **Processing** | Sequential (step-by-step) | Holistic (big-picture-first) |
| **Input** | Concrete (facts, examples, data) | Abstract (concepts, theories, models) |
| **Decision** | Analytical (logic, pros/cons) | Intuitive (gut feel, pattern recognition) |
| **Output** | Convergent (narrow to one answer) | Divergent (generate many options) |

**Items:**

| # | Statement A | Statement B | Measures |
|---|------------|-------------|----------|
| 1 | "I work through problems step by step" | "I see the whole picture first, then zoom in" | Seq vs. Hol |
| 2 | "I trust data and facts" | "I trust patterns and instinct" | Con vs. Abs |
| 3 | "I make pros/cons lists" | "I know the right answer before I can explain why" | Ana vs. Int |
| 4 | "I want to find THE answer" | "I want to generate MANY possible answers" | Conv vs. Div |
| 5 | "I prefer clear instructions and specs" | "I prefer a vision and freedom to execute" | Seq vs. Hol |
| 6 | "Show me the numbers" | "Tell me the story" | Con vs. Abs |
| 7 | "I decide by eliminating bad options" | "I decide by feeling which option excites me" | Ana vs. Int |
| 8 | "Too many options paralyze me" | "Too few options bore me" | Conv vs. Div |

**Scoring:** Same ipsative method as ASI-12. Each pole gets a 1-10 score.

```
For each dimension:
  score_A = selections_for_A / items_for_dimension × 10
  score_B = 10 - score_A  // forced ipsative
```

**Output:**
```json
{
  "processing": { "sequential": 3, "holistic": 7 },
  "input": { "concrete": 6, "abstract": 4 },
  "decision": { "analytical": 4, "intuitive": 6 },
  "output": { "convergent": 5, "divergent": 5 },
  "signature": "Holistic-Concrete-Intuitive-Balanced"
}
```

**Wirebot adaptation:**

```
if holistic > 7:
  → Start every recommendation with the big picture
  → "Here's the strategic view... and here's step 1"

if sequential > 7:
  → Start every recommendation with the immediate next step
  → "Step 1: do X. Here's why this matters strategically..."

if intuitive > 7:
  → Present recommendations as narratives, not spreadsheets
  → "This feels like the move because..."

if analytical > 7:
  → Present recommendations as structured comparisons
  → "Option A: [data]. Option B: [data]. Tradeoffs: [table]"
```

---

## 3. Scoring Formulas

### 3.1 Per-Instrument Reliability

```
// Cronbach's alpha for each instrument (computed across user population)
α = (k / (k-1)) × (1 - Σ σ²ᵢ / σ²_total)

where:
  k = number of items in the instrument
  σ²ᵢ = variance of item i across all users
  σ²_total = variance of total scores across all users

// For single-user reliability: use odd-even split-half
r_split = correlation(score_odd_items, score_even_items)
α_estimated = 2 × r_split / (1 + r_split)  // Spearman-Brown correction
```

### 3.2 Standard Error of Measurement

```
SEM(M) = σ_observed × √(1 - α)

// Confidence interval for any trait score:
CI_95(M) = score(M) ± 1.96 × SEM(M)

// A score of 7 ± 1.2 means: "we're 95% confident the true score is 5.8-8.2"
// Only report differences between traits if they exceed 2 × SEM
```

### 3.3 Composite Trait Score (Multi-Method Fusion)

When a trait is measured by both self-report AND behavioral observation:

```
// Weighted composite that favors higher-reliability source
composite(M) = w_sr × score_self_report(M) + w_beh × score_behavioral(M) + w_doc × score_document(M)

where:
  w_sr = α_self_report / (α_self_report + α_behavioral + α_document)
  w_beh = α_behavioral / (α_self_report + α_behavioral + α_document)
  w_doc = α_document / (α_self_report + α_behavioral + α_document)

// As behavioral data grows (more days, more messages), α_behavioral increases,
// and weight naturally shifts from self-report to observation.
```

**Key property:** Early on, self-report dominates (it's all we have). Over time, behavioral data takes over. The founder's profile becomes *what they do*, not what they say.

---

## 4. Communication Inference Engine

### 4.1 Feature Extraction

Every text message (chat, email, commit message, blog post, campaign) is processed:

```python
def extract_features(text: str) -> dict:
    sentences = split_sentences(text)
    words = tokenize(text)
    
    return {
        # Structural
        "msg_length_chars": len(text),
        "msg_length_words": len(words),
        "avg_sentence_length": mean([len(tokenize(s)) for s in sentences]),
        "sentence_count": len(sentences),
        "paragraph_count": text.count("\n\n") + 1,
        
        # Lexical
        "vocabulary_richness": len(set(words)) / max(len(words), 1),  # type-token ratio
        "avg_word_length": mean([len(w) for w in words]),
        "rare_word_ratio": count(w for w in words if w not in top_5000) / max(len(words), 1),
        
        # Pragmatic
        "question_ratio": count(s for s in sentences if s.strip().endswith("?")) / max(len(sentences), 1),
        "exclamation_ratio": count(s for s in sentences if s.strip().endswith("!")) / max(len(sentences), 1),
        "imperative_ratio": count(s for s in sentences if starts_with_verb(s)) / max(len(sentences), 1),
        
        # Hedging & Certainty (LIWC-inspired categories)
        "hedging_ratio": count_matches(HEDGE_WORDS, words) / max(len(words), 1),
        # HEDGE_WORDS = {"maybe", "perhaps", "possibly", "might", "could", "seems", "sort of",
        #   "kind of", "I think", "I guess", "probably", "not sure", "I wonder"}
        "certainty_ratio": count_matches(CERTAIN_WORDS, words) / max(len(words), 1),
        # CERTAIN_WORDS = {"definitely", "absolutely", "certainly", "always", "never",
        #   "must", "will", "clearly", "obviously", "no question"}
        
        # Agency & Action
        "action_verb_ratio": count_matches(ACTION_VERBS, words) / max(len(words), 1),
        # ACTION_VERBS = {"build", "ship", "launch", "create", "fix", "deploy", "push",
        #   "implement", "execute", "deliver", "close", "sell", "start", "finish"}
        "passive_verb_ratio": count_passive_constructions(sentences) / max(len(sentences), 1),
        
        # Self-reference
        "first_person_sing": count_matches({"i", "me", "my", "mine", "myself"}, words) / max(len(words), 1),
        "first_person_plur": count_matches({"we", "us", "our", "ours"}, words) / max(len(words), 1),
        "second_person": count_matches({"you", "your", "yours"}, words) / max(len(words), 1),
        
        # Emotional Valence
        "positive_emotion": count_matches(POS_EMOTION_WORDS, words) / max(len(words), 1),
        "negative_emotion": count_matches(NEG_EMOTION_WORDS, words) / max(len(words), 1),
        "emoji_count": count_emojis(text),
        "emoji_ratio": count_emojis(text) / max(len(words), 1),
        
        # Temporal
        "future_orientation": count_matches(FUTURE_WORDS, words) / max(len(words), 1),
        # FUTURE_WORDS = {"will", "going to", "plan", "next", "tomorrow", "soon", "eventually"}
        "past_orientation": count_matches(PAST_WORDS, words) / max(len(words), 1),
        "present_orientation": count_matches(PRESENT_WORDS, words) / max(len(words), 1),
        "urgency": count_matches(URGENCY_WORDS, words) / max(len(words), 1),
        # URGENCY_WORDS = {"now", "today", "asap", "immediately", "right away", "urgent", "critical"}
        
        # Complexity
        "subordinate_clause_ratio": count_subordinate_clauses(sentences) / max(len(sentences), 1),
        "list_usage": (text.count("- ") + text.count("* ") + text.count("1.")) / max(len(sentences), 1),
        "code_block_presence": 1 if "```" in text else 0,
    }
```

### 4.2 Feature → Trait Mapping

Features are mapped to communication traits using **weighted regression coefficients** derived from psycholinguistic research (Pennebaker et al., Schwartz et al.):

```
DISC Inference:
  D_signal = (
      0.30 × imperative_ratio
    + 0.25 × (1 - hedging_ratio)
    + 0.20 × action_verb_ratio
    + 0.15 × (1 / avg_sentence_length)   // shorter = more direct
    + 0.10 × urgency
  )

  I_signal = (
      0.30 × exclamation_ratio
    + 0.25 × positive_emotion
    + 0.20 × emoji_ratio
    + 0.15 × first_person_plur            // "we" oriented
    + 0.10 × (1 - rare_word_ratio)        // accessible language
  )

  S_signal = (
      0.30 × hedging_ratio
    + 0.25 × question_ratio
    + 0.20 × (1 - urgency)
    + 0.15 × second_person                // "you" oriented
    + 0.10 × passive_verb_ratio
  )

  C_signal = (
      0.30 × rare_word_ratio
    + 0.25 × avg_sentence_length_normalized
    + 0.20 × list_usage
    + 0.15 × certainty_ratio
    + 0.10 × subordinate_clause_ratio
  )

  // Normalize to sum to 1.0
  total = D_signal + I_signal + S_signal + C_signal
  D_inferred = D_signal / total
  I_inferred = I_signal / total
  S_inferred = S_signal / total
  C_inferred = C_signal / total
```

**Big Five Inference (supplementary — deeper personality layer):**

```
  Openness = (
      0.35 × vocabulary_richness
    + 0.25 × rare_word_ratio
    + 0.20 × question_ratio
    + 0.20 × subordinate_clause_ratio
  )

  Conscientiousness = (
      0.30 × list_usage
    + 0.25 × (1 - emoji_ratio)
    + 0.25 × certainty_ratio
    + 0.20 × avg_word_length_normalized
  )

  Extraversion = (
      0.30 × exclamation_ratio
    + 0.25 × emoji_ratio
    + 0.25 × positive_emotion
    + 0.20 × (1 / mean_response_latency)
  )

  Agreeableness = (
      0.30 × second_person
    + 0.25 × positive_emotion
    + 0.25 × hedging_ratio
    + 0.20 × first_person_plur
  )

  Neuroticism = (
      0.35 × negative_emotion
    + 0.25 × hedging_ratio
    + 0.20 × (sentence_length_variance / mean_sentence_length)  // erratic writing
    + 0.20 × question_ratio
  )
```

### 4.3 Running Averages with Exponential Decay

Features are not averaged equally across all time — recent messages matter more:

```
For each feature f, running exponential moving average:

  EMA(f, t) = λ × f(t) + (1 - λ) × EMA(f, t-1)

  where λ = decay_factor, controls recency weighting
  Default: λ = 0.05 (slow adaptation, stable traits)
  For mood/state signals: λ = 0.20 (fast adaptation, volatile)

// Number of effective samples:
  n_eff = 1 / (1 - (1-λ)²)  ≈ 1/λ for small λ

// Confidence grows with message count:
  inference_confidence = 1 - e^(-n_messages / τ)
  where τ = 50 (half-max confidence at ~35 messages, 90% at ~115 messages)
```

### 4.4 Source-Weighted Analysis

Not all text is equal. Different sources reveal different things:

```
Source weights for communication inference:

| Source              | Weight | Rationale                                      |
|---------------------|--------|------------------------------------------------|
| Chat with Wirebot   | 1.00   | Natural, unfiltered, conversational             |
| Discord/Slack msgs  | 0.90   | Casual, high-signal for real communication style|
| Email (sent)        | 0.75   | More formal, still authentic                    |
| Blog posts          | 0.50   | Edited, public-facing, not conversational       |
| Sendy campaigns     | 0.40   | Marketing voice ≠ personal voice                |
| Git commit messages  | 0.60   | Terse but reveals thought patterns              |
| Documents uploaded   | 0.45   | May be collaborative, not purely founder voice  |
| Code comments       | 0.55   | Technical but personality leaks through         |

weighted_feature(f) = Σ (source_weight(s) × f(s)) / Σ source_weight(s)
```

---

## 5. Behavioral Signal Processing

### 5.1 Temporal Pattern Extraction

From scoreboard events, CLI usage, and chat timestamps:

```
// Circadian activity profile — 24 bins (one per hour)
activity_histogram[h] = count(events where hour(timestamp) == h) / total_events

// Peak detection:
peak_hours = hours where activity_histogram[h] > mean + 1.5 × stddev

// Chronotype classification:
morning_activity = Σ activity_histogram[5..11]
afternoon_activity = Σ activity_histogram[12..17]
evening_activity = Σ activity_histogram[18..23]
night_activity = Σ activity_histogram[0..4]

chronotype = argmax(morning, afternoon, evening, night)

// Regularity index (how consistent is the daily pattern):
// Compute autocorrelation of hourly activity at lag=24
regularity = autocorrelation(event_timestamps_binned_hourly, lag=24)
// > 0.70 = very regular, < 0.30 = erratic
```

### 5.2 Shipping Cadence Analysis

```
// Daily ship counts over a rolling window
ships_per_day[d] = count(events where lane='shipping' AND date=d AND status='approved')

// Burst detection using coefficient of variation:
CV = stddev(ships_per_day) / mean(ships_per_day)
// CV > 1.5 = extreme burst pattern (validates high Quick Start)
// CV < 0.5 = steady pattern (validates high Follow Through)

// Streak sensitivity:
// How does shipping volume change after a streak break?
post_break_recovery = mean(ships_per_day[break+1..break+3]) / mean(ships_per_day[break-3..break-1])
// < 0.5 = streak-dependent (needs Wirebot to prevent breaks)
// > 0.8 = resilient (bounces back independently)

// Context switch rate:
unique_projects_per_day[d] = count(distinct project in events where date=d)
switch_rate = mean(unique_projects_per_day) / mean(ships_per_day)
// > 0.7 = high switching (scattered focus)
// < 0.3 = deep focus (single-project days)
```

### 5.3 Completion Ratio & Follow-Through Signal

```
// Tasks created vs. completed over rolling 14-day window
created_14d = count(events where event_type contains 'CREATED' in last 14d)
completed_14d = count(events where event_type contains 'COMPLETED' in last 14d)
completion_ratio = completed_14d / max(created_14d, 1)

// Approval latency (how fast does founder approve pending events):
approval_latencies = [approved_at - created_at for events where status changed to approved]
median_approval_latency = median(approval_latencies)
// < 1 hour = engaged, checking regularly
// > 24 hours = distant, needs nudges
// > 72 hours = disengaged, intervention needed

// Revenue follow-through:
// When a revenue event occurs, does shipping increase or decrease?
// Measures if founder "celebrates and coasts" or "doubles down"
post_revenue_shipping = mean(ships_per_day[revenue_event+1..+3])
pre_revenue_shipping = mean(ships_per_day[revenue_event-3..-1])
revenue_response = post_revenue_shipping / max(pre_revenue_shipping, 0.1)
// > 1.2 = doubles down (great)
// < 0.8 = coasts (flag for accountability)
```

### 5.4 Behavioral → Trait Validation

```
// Cross-validate self-reported traits against behavioral signals:

behavioral_quick_start = normalize(CV_shipping, 0, 3)        // burst pattern
behavioral_follow_through = normalize(completion_ratio, 0, 1) // finishes what starts
behavioral_fact_finder = normalize(median_approval_latency_inverse, ...)  // researches before approving?
behavioral_tenacity = normalize(post_break_recovery, 0, 1)    // bounces back after breaks

For each trait M:
  self_report_score(M) = from ASI-12
  behavioral_score(M) = from behavioral signals
  delta(M) = behavioral_score(M) - self_report_score(M)

  // Delta is a KEY insight:
  if |delta(M)| > 2.0:
    → "Self-perception gap detected"
    → Wirebot calibrates to BEHAVIORAL score, not self-report
    → Surface insight to founder when appropriate:
      "You see yourself as a [trait], but your patterns suggest [other trait].
       This isn't wrong — it means [interpretation]."
```

---

## 6. Document & Context Ingestion

### 6.1 Document Types & Extraction

Every document ingested feeds the profile model:

| Document Type | What's Extracted | Feeds Construct |
|---------------|-----------------|-----------------|
| **Business plan / pitch deck** | Revenue model, market, team, milestones | Φ₅ (Business Reality), Φ₇ (Cognitive: analytical vs. intuitive) |
| **Financial statements** | Actual revenue, expenses, runway | Φ₅ (ground truth override for self-report) |
| **Contracts / agreements** | Obligations, timelines, partners | Φ₅ (dependencies), Φ₄ (risk — what commitments exist) |
| **Blog posts / articles** | Writing style, topics, publishing cadence | Φ₂ (Communication), Φ₃ (Energy — what they write about = what energizes) |
| **Git history** | Commit style, frequency, languages, project breadth | Φ₁ (Action), Φ₆ (Temporal), Φ₃ (Energy — what they build) |
| **Email threads** | Communication style, response patterns, network | Φ₂ (Communication), Φ₆ (Temporal) |
| **Calendar** | Time allocation, meeting types, free time ratio | Φ₅ (Reality), Φ₆ (Temporal), Φ₃ (Energy vs. drain) |
| **Social media posts** | Public voice, engagement patterns | Φ₂ (Communication), Φ₃ (Galvanizing energy) |
| **Chat history** | Unfiltered communication, topic patterns | ALL constructs |
| **Scoreboard events** | What they actually ship, when, how much | Φ₁, Φ₅, Φ₆ (behavioral ground truth) |

### 6.2 Document Scoring Pipeline

```
On document ingestion:

1. CLASSIFY document type (LLM classifier or extension match)
2. EXTRACT structured fields:
   - For financial docs: revenue, expenses, debt, runway
   - For business plans: stage, market, team size, funding
   - For contracts: obligations, deadlines, counterparties
   - For all text: run Communication Inference Engine (§4)
3. CALCULATE trait signal updates:
   - Each extraction produces a set of (trait, value, confidence) tuples
   - Confidence based on document quality and relevance
4. UPDATE profile via Bayesian fusion (§7)
5. LOG in ingestion ledger:
   {doc_id, type, extracted_signals, confidence, timestamp}
```

### 6.3 Context Event Scoring

Every scoreboard event carries implicit behavioral signal:

```
For each event e:
  signals = {
    "action_style": infer_from_event_type(e),
    "energy_signal": infer_from_lane(e),
    "temporal_signal": e.timestamp,
    "focus_signal": e.project,
    "momentum_signal": e.score_delta,
  }

// Event type → Action Style mapping:
  PRODUCT_RELEASE → Quick Start + Implementor (high)
  DOCS_PUBLISHED → Fact Finder + Follow Through (high)
  DEPLOY → Implementor + Tenacity (high)
  BLOG_PUBLISHED → Wonder + Galvanizing (high)
  CODE_PUSHED → Implementor (moderate)
  CAMPAIGN_SENT → Galvanizing (high)
  PAYMENT_RECEIVED → (no direct trait signal, but reinforces revenue reality)

// Lane → Energy mapping:
  shipping → Invention + Tenacity energy
  distribution → Galvanizing + Wonder energy
  revenue → Discernment + Tenacity energy
  systems → Follow Through + Implementor energy

// Accumulate over time:
  For each activity type A:
    event_energy_score(A) = count(events mapping to A) / total_events × 10
  
  // Compare with self-reported ETM-6 ranking:
  // If founder ranked Galvanizing as "draining" but 30% of their events
  // are distribution (Galvanizing), there's a mismatch worth surfacing.
```

### 6.4 Progressive Document Value

Documents don't all contribute equally. Value depends on **recency, specificity, and verification level**:

```
document_value(d) = base_value(d.type) × recency_weight(d) × verification_weight(d)

recency_weight(d) = e^(-age_days(d) / half_life(d.type))
  // Financial docs: half_life = 90 days (stale fast)
  // Personality text: half_life = 365 days (stable)
  // Business plans: half_life = 180 days

verification_weight(d) = {
  "self_reported": 0.60,
  "uploaded_document": 0.75,
  "connected_account_api": 0.95,
  "third_party_verified": 1.00,
}

// A Stripe API showing $5K MRR (verification=0.95, recency=today)
// overrides a self-reported "about $5K/mo" (verification=0.60) completely.
```

---

## 7. Bayesian Profile Updating

### 7.1 Core Update Rule

Each trait score is treated as a probability distribution, not a point estimate.

```
// Prior: what we believed before new evidence
P(trait = θ | prior_data) ~ N(μ_prior, σ²_prior)

// Likelihood: what the new evidence suggests
P(new_evidence | trait = θ) ~ N(μ_evidence, σ²_evidence)

// Posterior: updated belief (conjugate normal-normal)
σ²_posterior = 1 / (1/σ²_prior + 1/σ²_evidence)
μ_posterior = σ²_posterior × (μ_prior/σ²_prior + μ_evidence/σ²_evidence)

// In practice:
//   σ²_evidence is large when evidence is weak (single message, low confidence)
//   σ²_evidence is small when evidence is strong (100 messages, API-verified data)
//   So weak evidence barely moves the posterior. Strong evidence dominates.
```

### 7.2 Update Triggers

```
| Trigger | What Updates | Expected Shift |
|---------|-------------|----------------|
| Assessment answer submitted | Relevant trait | Large (first time), small (retest) |
| 10 new chat messages accumulated | Communication traits | Small per batch, cumulative |
| New document ingested | Business reality + communication | Medium |
| Connected account data refresh | Business reality | Large (ground truth) |
| 24 hours of new scoreboard events | Behavioral traits | Small per day, cumulative |
| Weekly behavioral analysis batch | All behavioral traits | Medium (validated patterns) |
| 30-day retest prompt completed | Assessment traits | Confirms or corrects |
```

### 7.3 Conflict Resolution

When sources disagree (self-report says X, behavior says Y):

```
conflict_magnitude = |μ_self_report - μ_behavioral| / max(σ_self_report, σ_behavioral)

if conflict_magnitude < 1.0:
  → Normal variance, use composite (§3.3)

if 1.0 ≤ conflict_magnitude < 2.0:
  → Moderate discrepancy
  → Weight behavioral higher: w_beh = 0.65, w_sr = 0.35
  → Log as "self-perception gap" (informational, not alarming)

if conflict_magnitude ≥ 2.0:
  → Strong discrepancy
  → Weight behavioral dominant: w_beh = 0.80, w_sr = 0.20
  → Surface to founder diplomatically:
    "Your [trait] assessment and your actual patterns differ.
     This is common and useful to know. Here's what I observe..."
  → Store delta as a meta-trait (self-awareness calibration)
```

### 7.4 Confidence Accumulation Formula

```
// Overall trait confidence (how sure we are about this score):
trait_confidence(M) = 1 - Π(1 - source_confidence(s, M))
                       for all sources s that measure M

// Each source's confidence:
assessment_confidence(M) = α_instrument × consistency(M)  // instrument reliability × item consistency
inference_confidence(M) = 1 - e^(-n_messages / τ)           // grows with message count
behavioral_confidence(M) = 1 - e^(-n_days / τ_days)         // grows with observation days
document_confidence(M) = verification_weight × recency_weight

// Example: Assessment α=0.80, consistency=0.90 → 0.72
//          Inference after 100 msgs → 0.86
//          Behavioral after 14 days → 0.68
//          Combined: 1 - (0.28 × 0.14 × 0.32) = 0.987 → very confident
```

---

## 8. Composite Pairing Score

### 8.1 Component Scores

```
S₁ = assessment_completion × assessment_quality
     assessment_completion = questions_answered / total_questions
     assessment_quality = mean(consistency scores across instruments)

S₂ = communication_inference_confidence
     = 1 - e^(-n_analyzed_messages / 50)

S₃ = behavioral_pattern_confidence
     = 1 - e^(-n_observed_days / 14)

S₄ = business_reality_verification
     = Σ (verified_dimensions / total_dimensions)
     dimensions: revenue, debt, stage, team, products, timeline
     each dimension: 0 (unknown), 0.5 (self-reported), 1.0 (verified)

S₅ = document_context_richness
     = min(1.0, Σ document_value(d) / target_document_value)
     target_document_value = 10.0 (calibrated threshold)

S₆ = trait_stability
     = mean(test_retest_correlation across retested traits)
     // Only contributes after first retest (≥ 30 days)

S₇ = profile_coherence
     = 1 - mean(|delta(M)| for all traits M where delta exists)
     // High coherence = self-report matches behavior
     // Low coherence = significant gaps (still informative, but less certain)
```

### 8.2 Weighted Composite

```
Pairing Score = Σ wᵢ × Sᵢ × 100

Weights:
  w₁ = 0.20  (assessment — structured self-report)
  w₂ = 0.20  (communication inference — passive, deep)
  w₃ = 0.15  (behavioral patterns — what they do)
  w₄ = 0.15  (business reality verification — ground truth)
  w₅ = 0.10  (document context — richness of knowledge)
  w₆ = 0.10  (trait stability — confirmed over time)
  w₇ = 0.10  (profile coherence — self-awareness)
  ──────
  Σ = 1.00

// Score range: 0-100
// Note: S₆ starts at 0 until first retest, which means
// maximum achievable score in first 30 days ≈ 90
// This is intentional — true bonding requires time validation.
```

### 8.3 Level Thresholds (revised)

```
Score  Level         Gate Condition
─────  ─────         ──────────────
0-15   Stranger      None
16-35  Acquaintance  S₁ ≥ 0.30 (some assessment done)
36-55  Partner       S₁ ≥ 0.80 AND S₂ ≥ 0.30 (assessment complete + some inference)
56-75  Trusted       S₂ ≥ 0.60 AND S₃ ≥ 0.40 (meaningful inference + behavioral data)
76-90  Bonded        S₂ ≥ 0.80 AND S₃ ≥ 0.70 AND S₆ > 0 (deep inference + stability check)
91-100 Sovereign     ALL Sᵢ ≥ 0.70 (no weak dimension)

// Gate conditions prevent score gaming:
// You can't reach Bonded by only doing assessments.
// You can't reach Sovereign without time (stability requires 30+ days).
```

---

## 9. Calibration Engine

### 9.1 Wirebot Behavior Parameters

The profile directly sets Wirebot's operating parameters:

```json
{
  "communication": {
    "max_message_length": f(DISC_primary, detail_preference),
    // D-primary: 150 words max. C-primary: 500 words OK.
    
    "lead_with": f(cognitive_processing),
    // Sequential: "Step 1..." / Holistic: "Big picture..."
    
    "tone_formality": f(observed_formality),
    // Range 0-1, directly from inference engine
    
    "emoji_usage": f(observed_emoji_ratio),
    // Mirror the founder's emoji density ± 20%
    
    "question_frequency": f(DISC_primary, advice_style_preference),
    // D-primary: minimal questions, just recommend
    // S-primary: more questions, collaborative feel
    
    "celebration_intensity": f(I_score, positive_emotion_observed),
    // High I + high positive emotion: "🎉 AMAZING! You crushed it!"
    // Low I + low emotion: "Good. Shipped. Moving on."
  },
  
  "accountability": {
    "nudge_frequency_hours": f(completion_ratio, approval_latency),
    // Low completion + slow approval → nudge every 4h
    // High completion + fast approval → nudge every 12h
    
    "nudge_intensity": f(accountability_preference, tenacity_score),
    // Preference "drill sergeant" + low tenacity → strong pushes
    // Preference "gentle" + high tenacity → light touches
    
    "deadline_pressure": f(quick_start_score, bias_to_action),
    // High QS: less artificial deadlines (they self-start)
    // Low QS: more deadlines (they need external pressure)
    
    "streak_emphasis": f(post_break_recovery),
    // Low recovery → heavy streak emphasis (don't let it break)
    // High recovery → moderate emphasis (they bounce back anyway)
  },
  
  "recommendations": {
    "options_presented": f(convergent_divergent),
    // Convergent: 1-2 options with clear recommendation
    // Divergent: 3-5 options for exploration
    
    "data_density": f(analytical_intuitive, fact_finder_score),
    // Analytical + high FF: lots of data, comparisons, evidence
    // Intuitive + low FF: narrative, pattern-based, light data
    
    "planning_depth": f(follow_through_score, sequential_holistic),
    // High FT + sequential: detailed multi-step plans
    // Low FT + holistic: just the next action + the why
    
    "risk_framing": f(loss_aversion, risk_tolerance),
    // High loss aversion: "Here's the safety net if this fails..."
    // Low loss aversion: "Here's the upside if this works..."
  },
  
  "proactive": {
    "morning_standup_hour": f(chronotype, peak_hours),
    // Night owl: standup at 11 AM not 8 AM
    
    "peak_hour_tasks": f(energy_topology, chronotype),
    // Assign genius-work to peak hours, frustration-work to off-peak
    // "It's 10 PM and you're in the zone — here's the invention work"
    // "It's 2 PM (your low) — here's the admin I prepped for you"
    
    "complement_ratio": wirebot_weight (from ETM-6),
    // % of proactive suggestions in each work category
    // Weighted toward founder's frustration areas
    
    "intervention_threshold": f(stall_hours, tenacity_score, streak_sensitivity),
    // How long before Wirebot intervenes on a stall
    // Low tenacity + streak-sensitive: intervene at 4h stall
    // High tenacity + resilient: intervene at 12h stall
  }
}
```

### 9.2 Parameter Update Cadence

```
| Parameter Category | Update Frequency | Trigger |
|-------------------|------------------|---------|
| Communication tone | Every 50 messages | EMA update |
| Accountability timing | Weekly | Behavioral batch analysis |
| Recommendation style | Every 100 messages OR retest | Profile change > 1 stddev |
| Proactive scheduling | On chronotype change detection | 7-day rolling pattern shift |
| Complement allocation | Monthly | ETM retest or behavioral divergence |
```

---

## 10. Confidence & Validity

### 10.1 Per-Trait Confidence Reporting

Every trait in the profile carries an explicit confidence:

```json
{
  "action_style": {
    "quick_start": {
      "score": 8,
      "confidence": 0.88,
      "CI_95": [6.4, 9.6],
      "sources": {
        "assessment": { "value": 9, "weight": 0.35 },
        "behavioral": { "value": 7.2, "weight": 0.45 },
        "document": { "value": null, "weight": 0.00 },
        "inference": { "value": 7.8, "weight": 0.20 }
      },
      "self_perception_delta": -1.8,
      "last_updated": "2026-02-15T08:00:00Z",
      "n_observations": 847
    }
  }
}
```

### 10.2 System-Level Validity Checks

```
// Run weekly:

1. INTERNAL CONSISTENCY CHECK
   For each instrument: recalculate Cronbach's α across all users
   Flag if α drops below 0.65 → item may need revision

2. CONVERGENT VALIDITY CHECK
   For each trait measured by 2+ methods:
   Calculate correlation between methods
   Flag if r < 0.40 → methods may be measuring different things

3. PREDICTIVE VALIDITY CHECK
   Does calibration improve outcomes?
   Metric: User engagement (message count, approval speed) before vs. after calibration
   Metric: Stated satisfaction with Wirebot recommendations

4. TEST-RETEST STABILITY
   For users who have retested (30+ day gap):
   Calculate r for each trait
   Flag traits with r < 0.60 → may be state (not trait), adjust model accordingly

5. PROFILE COHERENCE AUDIT
   For each user: calculate mean |delta| across all multi-method traits
   If mean |delta| > 2.5 → profile may be unreliable, increase all σ² (widen uncertainty)
```

### 10.3 Minimum Viable Confidence

```
Wirebot uses trait scores ONLY when confidence exceeds threshold:

| Calibration Parameter | Min Confidence Required | Fallback If Below |
|----------------------|------------------------|-------------------|
| Message length | 0.30 | Use default (300 words) |
| Tone formality | 0.40 | Use default (0.50, neutral) |
| Nudge frequency | 0.50 | Use default (8h) |
| Deadline pressure | 0.50 | Use preference from assessment |
| Complement allocation | 0.60 | Equal weight all areas |
| Autonomous actions | 0.75 | Always ask permission |
| Proactive scheduling | 0.60 | Use stated preference |

// If confidence is below threshold, Wirebot explicitly says:
// "I'm still learning your patterns. Using default settings for [X].
//  I'll calibrate this after [what's needed — more messages, more days, etc.]"
```

---

## 11. Continuous Learning Loop

### 11.1 The Flywheel

```
                    ┌─────────────────┐
                    │  New Evidence    │
                    │  (message, event,│
                    │   document, API) │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Feature Extract  │
                    │ (NLP, temporal,  │
                    │  behavioral)     │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Bayesian Update  │
                    │ (prior → post)   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
     ┌────────▼───┐  ┌──────▼──────┐  ┌───▼────────┐
     │ Profile     │  │ Confidence   │  │ Delta      │
     │ Updated     │  │ Updated      │  │ Detected   │
     └────────┬───┘  └──────┬──────┘  └───┬────────┘
              │              │              │
              └──────────────┼──────────────┘
                             │
                    ┌────────▼────────┐
                    │ Calibration      │
                    │ Engine Adjusts   │
                    │ Wirebot Behavior │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Wirebot Interacts│
                    │ (better aligned) │
                    └────────┬────────┘
                             │
                    ┌────────▼────────┐
                    │ Founder Responds │
                    │ (new evidence)   │
                    └─────────────────┘
                             ↻
```

### 11.2 What Gets Better With More Data

```
| Data Milestone | What Improves | Approximate Timeline |
|----------------|---------------|---------------------|
| 10 messages | Basic directness/formality inference | Day 1 |
| Assessment complete | All 7 constructs have priors | Day 1 (10 min) |
| 50 messages | DISC inference reaches 50% confidence | Day 3-5 |
| 1 document ingested | Business reality fills gaps | Day 1-7 |
| 7 days events | Chronotype + shipping cadence detected | Week 1 |
| 1 connected account | Ground truth override for 1 dimension | Week 1-2 |
| 100 messages | Communication style inference at 70%+ | Week 2-3 |
| 200 messages | Big Five inference meaningful | Week 3-4 |
| 14 days events | Completion ratio + switch rate stable | Week 2 |
| 30 days + retest | Trait stability confirmed or revised | Month 1 |
| 3+ connected accounts | Business reality mostly verified | Month 1-2 |
| 500 messages | Communication inference > 90% confidence | Month 2-3 |
| 5+ documents | Rich contextual model | Ongoing |
| 90 days (season) | Full behavioral pattern library | Season 1 |
| Multiple seasons | Longitudinal growth tracking | Ongoing |
```

### 11.3 Document Ingestion Intelligence

Every new document doesn't just add knowledge — it **refines trait estimates**:

```
On new document D:

1. EXTRACT text features → Communication Inference update
2. EXTRACT business facts → Business Reality update
3. ANALYZE document TYPE for energy signal:
   - Founder wrote a pitch deck → Galvanizing energy signal (they did the work)
   - Founder uploaded a spreadsheet → Fact Finder + Analytical signal
   - Founder shared a design mockup → Implementor + Invention signal
   - Founder shared meeting notes → Enablement + Discernment signal

4. COMPARE document content with stated goals:
   - Pitch deck mentions "$10M ARR" but self-reported revenue is $1K/mo
   → Flag aspiration vs. reality gap (inform Wirebot's calibration)
   
   - Business plan mentions "3 team members" but all evidence shows solo
   → Flag social desirability on team dimension
   
   - Contract shows monthly obligation of $5K but self-reported "manageable debt"
   → Silently increase debt severity estimate (verified > self-reported)

5. CROSS-REFERENCE with existing profile:
   - New doc's writing style consistent with existing inference? → Increase confidence
   - New doc's writing style divergent? → Investigate (different author? Different context?)
   
6. CALCULATE information gain:
   info_gain(D) = Σ (σ²_prior(M) - σ²_posterior(M)) for all traits M affected
   // High info gain = document significantly refined the profile
   // Low info gain = document confirmed what we already knew (still valuable for confidence)
```

### 11.4 Event Intelligence

Every scoreboard event refines the profile, even mundane ones:

```
On new event E:

1. TEMPORAL SIGNAL:
   hour = E.timestamp.hour()
   activity_histogram[hour] += 1
   → Chronotype update (§5.1)

2. ACTION SIGNAL:
   lane = E.lane
   event_type = E.event_type
   → Action Style behavioral score update (§5.4)
   → Energy Topology behavioral validation (§6.3)

3. FOCUS SIGNAL:
   project = E.metadata.project
   unique_projects_today += (1 if new project today)
   → Context switch rate update (§5.2)

4. MOMENTUM SIGNAL:
   days_since_last_ship = E.timestamp - last_ship_timestamp
   if days_since_last_ship > 24h:
     → Stall detection
     → Post-break recovery tracking (§5.2)
   
5. APPROVAL SIGNAL (when operator approves a pending event):
   approval_latency = approved_at - created_at
   → Engagement level update
   → If fast approval after stall: recovery signal
   → If selective approval (approve some, reject others): Discernment signal

6. REVENUE SIGNAL:
   if E.lane == "revenue":
     actual_revenue += E.metadata.amount
     → Business Reality ground truth update
     → Overrides self-reported revenue bracket

7. CROSS-EVENT PATTERNS (batch, weekly):
   - Shipping bursts followed by long stalls → "sprint-crash" pattern
   - Revenue events followed by reduced shipping → "coast after win" pattern  
   - Distribution events cluster on certain days → "batch content" pattern
   - Systems events spike after shipping failures → "reactive infrastructure" pattern
   Each pattern has a name, detection formula, and calibration implication.
```

---

## Appendix A: Word Lists

```
HEDGE_WORDS = [
  "maybe", "perhaps", "possibly", "might", "could", "may",
  "seems", "sort of", "kind of", "I think", "I guess",
  "probably", "not sure", "I wonder", "I believe",
  "appears to", "tends to", "it seems like", "I suppose",
  "arguably", "presumably"
]

CERTAIN_WORDS = [
  "definitely", "absolutely", "certainly", "always", "never",
  "must", "will", "clearly", "obviously", "no question",
  "without doubt", "guaranteed", "for sure", "100%",
  "undeniably", "unquestionably"
]

ACTION_VERBS = [
  "build", "ship", "launch", "create", "fix", "deploy", "push",
  "implement", "execute", "deliver", "close", "sell", "start",
  "finish", "complete", "release", "publish", "send", "submit",
  "acquire", "convert", "generate", "produce", "develop",
  "install", "configure", "migrate", "refactor", "optimize"
]

POS_EMOTION_WORDS = [
  "love", "great", "amazing", "awesome", "excellent", "fantastic",
  "beautiful", "brilliant", "happy", "excited", "proud", "grateful",
  "wonderful", "incredible", "perfect", "thrilled", "stoked",
  "pumped", "blessed", "fortunate", "enjoy", "celebrate"
]

NEG_EMOTION_WORDS = [
  "hate", "terrible", "awful", "horrible", "frustrated", "angry",
  "disappointed", "worried", "anxious", "stressed", "overwhelmed",
  "exhausted", "confused", "stuck", "failed", "struggling",
  "burned out", "discouraged", "hopeless", "dread"
]

FUTURE_WORDS = [
  "will", "going to", "plan", "next", "tomorrow", "soon",
  "eventually", "goal", "target", "aim", "hope to",
  "intend", "expect", "project", "forecast", "roadmap"
]

URGENCY_WORDS = [
  "now", "today", "asap", "immediately", "right away",
  "urgent", "critical", "emergency", "time-sensitive",
  "deadline", "overdue", "behind", "hurry", "rush"
]
```

## Appendix B: Minimum Data Requirements Per Level

```
Level        | Assessment | Messages | Days | Docs | Accounts | Retest
─────────────┼────────────┼──────────┼──────┼──────┼──────────┼───────
Stranger     | 0          | 0        | 0    | 0    | 0        | No
Acquaintance | Partial    | 0        | 0    | 0    | 0        | No
Partner      | Complete   | 30+      | 3+   | 0    | 0        | No
Trusted      | Complete   | 100+     | 7+   | 1+   | 1+       | No
Bonded       | Complete   | 200+     | 30+  | 3+   | 2+       | Yes
Sovereign    | Complete   | 500+     | 90+  | 5+   | 3+       | Yes ×2
```

## Appendix C: Profile Schema (v2)

```json
{
  "version": 2,
  "schema_version": "2026-02-02",
  "founder_id": "verious",
  
  "pairing_score": {
    "composite": 0,
    "level": "Stranger",
    "components": {
      "S1_assessment": 0.00,
      "S2_communication_inference": 0.00,
      "S3_behavioral_patterns": 0.00,
      "S4_business_verification": 0.00,
      "S5_document_richness": 0.00,
      "S6_trait_stability": 0.00,
      "S7_profile_coherence": 0.00
    },
    "gates_met": [],
    "next_gate": "S1 ≥ 0.30 → Acquaintance"
  },
  
  "constructs": {
    "Φ1_action_style": {
      "fact_finder":    { "score": null, "confidence": 0, "CI_95": null, "sources": {} },
      "follow_through": { "score": null, "confidence": 0, "CI_95": null, "sources": {} },
      "quick_start":    { "score": null, "confidence": 0, "CI_95": null, "sources": {} },
      "implementor":    { "score": null, "confidence": 0, "CI_95": null, "sources": {} }
    },
    "Φ2_communication_style": {
      "disc": { "D": null, "I": null, "S": null, "C": null, "primary": null },
      "observed": {
        "directness": null, "formality": null, "detail_preference": null,
        "emotion_expression": null, "pace_preference": null
      },
      "big_five": {
        "openness": null, "conscientiousness": null, "extraversion": null,
        "agreeableness": null, "neuroticism": null
      },
      "messages_analyzed": 0,
      "inference_confidence": 0
    },
    "Φ3_energy_topology": {
      "wonder": null, "invention": null, "discernment": null,
      "galvanizing": null, "enablement": null, "tenacity": null,
      "genius": [], "frustration": [],
      "wirebot_complement_weights": {}
    },
    "Φ4_risk_disposition": {
      "risk_tolerance": null, "decision_speed": null,
      "ambiguity_tolerance": null, "sunk_cost_sensitivity": null,
      "loss_aversion": null, "bias_to_action": null
    },
    "Φ5_business_reality": {
      "declared": {},
      "verified": {},
      "verification_level": 0.00
    },
    "Φ6_temporal_patterns": {
      "chronotype": null, "peak_hours": [],
      "regularity_index": null,
      "shipping_cadence_cv": null,
      "days_observed": 0
    },
    "Φ7_cognitive_style": {
      "processing": null, "input": null,
      "decision": null, "output": null,
      "signature": null
    }
  },
  
  "behavioral_signals": {
    "completion_ratio": null,
    "context_switch_rate": null,
    "post_break_recovery": null,
    "revenue_response": null,
    "approval_latency_median_hours": null,
    "streak_sensitivity": null
  },
  
  "self_perception_deltas": {},

  "dual_track": {
    "// Every construct has both a trait (slow) and state (fast) reading": "",
    "Φ1_action_style": {
      "trait": { "QS": null, "FF": null, "FT": null, "IM": null },
      "state": { "QS": null, "FF": null, "FT": null, "IM": null },
      "drift": { "QS": 0, "FF": 0, "FT": 0, "IM": 0 },
      "alpha": { "QS": 0.70, "FF": 0.70, "FT": 0.70, "IM": 0.70 },
      "effective": { "QS": null, "FF": null, "FT": null, "IM": null }
    },
    "Φ2_communication_style": {
      "trait": { "D": null, "I": null, "S": null, "C": null },
      "state": { "D": null, "I": null, "S": null, "C": null },
      "drift": { "D": 0, "I": 0, "S": 0, "C": 0 },
      "alpha": 0.70,
      "effective": { "D": null, "I": null, "S": null, "C": null }
    },
    "Φ3_energy_topology": {
      "trait": { "W": null, "N": null, "D": null, "G": null, "E": null, "T": null },
      "state": { "W": null, "N": null, "D": null, "G": null, "E": null, "T": null },
      "drift": {},
      "alpha": 0.70,
      "effective": {}
    },
    "Φ4_risk_disposition": {
      "trait": {},
      "state": {},
      "drift": {},
      "alpha": 0.70,
      "effective": {}
    }
  },

  "context_windows": {
    "active": [],
    "history": [],
    "// Each window": {
      "name": "FINANCIAL_PRESSURE | SHIPPING_SPRINT | RECOVERY_PERIOD | CONTEXT_EXPLOSION | STALL | CELEBRATION | LIFE_EVENT",
      "activation": 0.0,
      "activated_at": null,
      "signals": [],
      "calibration_overrides": {}
    }
  },

  "complement_vector": {
    "// Current Wirebot effort allocation (sums to 1.0)": "",
    "fact_finder": 0,
    "follow_through": 0,
    "quick_start": 0,
    "implementor": 0,
    "wonder": 0,
    "invention": 0,
    "discernment": 0,
    "galvanizing": 0,
    "enablement": 0,
    "tenacity": 0,
    "last_rebalanced": null
  },
  
  "calibration": {
    "communication": {
      "max_message_words": 300,
      "lead_with": "recommendation",
      "tone_formality": 0.50,
      "emoji_mirror_ratio": 0.50,
      "question_frequency": "moderate",
      "celebration_intensity": 0.50
    },
    "accountability": {
      "nudge_frequency_hours": 8,
      "nudge_intensity": 0.50,
      "deadline_pressure": 0.50,
      "streak_emphasis": 0.50,
      "stall_intervention_hours": 8
    },
    "recommendations": {
      "options_presented": 2,
      "data_density": 0.50,
      "planning_depth": "moderate",
      "risk_framing": "balanced"
    },
    "proactive": {
      "standup_hour": 8,
      "peak_task_type": "genius_work",
      "offpeak_task_type": "frustration_work",
      "intervention_threshold_hours": 8,
      "complement_ratio": {}
    }
  },
  
  "meta": {
    "created_at": null,
    "last_assessment": null,
    "last_retest": null,
    "last_inference_update": null,
    "last_behavioral_batch": null,
    "last_drift_check": null,
    "last_complement_rebalance": null,
    "last_context_window_eval": null,
    "total_messages_analyzed": 0,
    "total_events_analyzed": 0,
    "total_documents_ingested": 0,
    "total_state_shifts_detected": 0,
    "connected_accounts": [],
    "engine_version": "2.0",
    "signals_processed": 0
  }
}
```

---

## 12. Runtime Architecture — The Background Daemon

### 12.1 Process Model

The pairing engine runs as part of the **wirebot-scoreboard** Go binary (or a dedicated
`wirebot-pairing-engine` process if load requires it). It is NOT a cron job or batch process.
It is an event-driven reactor that processes signals as they arrive.

```
┌──────────────────────────────────────────────────────────────┐
│                    PAIRING ENGINE DAEMON                      │
│                                                              │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────────┐  │
│  │ Signal       │   │ Profile      │   │ Calibration      │  │
│  │ Ingestion    │──▶│ Updater      │──▶│ Engine           │  │
│  │ Bus          │   │ (Bayesian)   │   │ (Complement +    │  │
│  │              │   │              │   │  Context Windows) │  │
│  └──────┬───────┘   └──────┬───────┘   └────────┬─────────┘  │
│         │                  │                     │            │
│  ┌──────▼───────┐   ┌──────▼───────┐   ┌────────▼─────────┐  │
│  │ Feature      │   │ Drift        │   │ Behavior         │  │
│  │ Extractors   │   │ Detector     │   │ Parameters       │  │
│  │ (NLP, time,  │   │ (trait vs    │   │ (what Wirebot    │  │
│  │  behavioral) │   │  state)      │   │  actually does)  │  │
│  └──────────────┘   └──────────────┘   └──────────────────┘  │
│                                                              │
│  INPUT STREAMS:                    OUTPUT:                    │
│  • Chat messages (webhook)         • profile.json (on disk)  │
│  • Scoreboard events (DB watch)    • calibration params      │
│  • Document uploads                • complement vector       │
│  • Connected account polls         • context windows         │
│  • Assessment answers              • drift alerts            │
│  • Approval actions                                          │
└──────────────────────────────────────────────────────────────┘
```

### 12.2 Signal Bus Architecture

Every signal that enters the system follows the same pipeline:

```go
type Signal struct {
    Type       string    // "message", "event", "document", "account", "assessment", "approval"
    Source     string    // "chat", "scoreboard", "github", "stripe", "sendy", etc.
    Timestamp  time.Time
    Content    string    // raw text (for messages/docs)
    Metadata   map[string]interface{}  // structured data
    Features   map[string]float64      // extracted features (populated by extractors)
}

// Pipeline:
func (engine *PairingEngine) ProcessSignal(sig Signal) {
    // 1. Extract features based on signal type
    sig.Features = engine.extractFeatures(sig)
    
    // 2. Update trait EMA (slow) and state EMA (fast)
    engine.updateTraitEMA(sig.Features)
    engine.updateStateEMA(sig.Features)
    
    // 3. Check for drift
    drifts := engine.detectDrift()
    
    // 4. Evaluate context windows
    engine.evaluateContextWindows(sig)
    
    // 5. Recompute complement vector if anything shifted
    if engine.calibrationDirty {
        engine.recomputeComplement()
        engine.updateCalibrationParams()
        engine.calibrationDirty = false
    }
    
    // 6. Persist profile
    engine.saveProfile()
    
    // 7. Emit drift alerts if significant
    for _, d := range drifts {
        if d.Magnitude >= 2.0 {
            engine.emitAlert(d)
        }
    }
    
    engine.profile.Meta.SignalsProcessed++
}
```

### 12.3 Real-Time Hooks

The engine attaches to existing infrastructure via hooks:

```
1. CHAT MESSAGE HOOK (in scoreboard Go proxy):
   After proxying a message to OpenClaw, before returning response:
   → engine.ProcessSignal(Signal{Type: "message", Content: userMessage, ...})
   → Also process the RESPONSE (Wirebot's own messages are calibration feedback)

2. SCOREBOARD EVENT HOOK (in event insert handler):
   After inserting any event (approved or pending):
   → engine.ProcessSignal(Signal{Type: "event", Metadata: eventData, ...})

3. EVENT APPROVAL HOOK (in approve handler):
   When operator approves/rejects:
   → engine.ProcessSignal(Signal{Type: "approval", Metadata: {
       event_id, latency_seconds, action: "approve"|"reject"
     }})

4. DOCUMENT UPLOAD HOOK (new endpoint):
   POST /v1/pairing/documents
   → engine.ProcessSignal(Signal{Type: "document", Content: docText, ...})

5. CONNECTED ACCOUNT DATA HOOK (in poller cycle):
   When a poller fetches new data:
   → engine.ProcessSignal(Signal{Type: "account", Source: provider, ...})

6. ASSESSMENT ANSWER HOOK (in pairing answer endpoint):
   POST /v1/pairing/answers
   → engine.ProcessSignal(Signal{Type: "assessment", Metadata: answers, ...})
```

### 12.4 The Engine Never Blocks

All signal processing is **asynchronous and non-blocking**:

```go
// Signals are buffered into a channel
engine.signalChan <- sig  // non-blocking send

// Background goroutine processes signals serially (order matters for EMA)
go func() {
    for sig := range engine.signalChan {
        engine.ProcessSignal(sig)  // ~1-5ms per signal
    }
}()

// If the channel is full (burst of events), signals are batched:
// This ensures Wirebot's response time is never affected by profile computation
```

### 12.5 Profile Persistence & Access

```
Profile location: /data/wirebot/pairing/profile.json
Profile cache: In-memory (Go struct), persisted every 10 signals or 60 seconds

// Chat context injection reads the EFFECTIVE scores (trait×α + state×(1-α)):
func (engine *PairingEngine) GetEffectiveProfile() EffectiveProfile {
    p := engine.profile
    eff := EffectiveProfile{}
    
    for _, construct := range p.DualTrack {
        for dim, trait := range construct.Trait {
            state := construct.State[dim]
            alpha := construct.Alpha[dim]
            if trait != nil && state != nil {
                eff[dim] = alpha * *trait + (1-alpha) * *state
            } else if trait != nil {
                eff[dim] = *trait
            } else if state != nil {
                eff[dim] = *state
            }
        }
    }
    
    eff.ComplementVector = p.ComplementVector
    eff.Calibration = p.Calibration
    eff.ContextWindows = p.ContextWindows.Active
    eff.PairingScore = p.PairingScore.Composite
    eff.Level = p.PairingScore.Level
    
    return eff
}

// This effective profile is what gets injected into the chat system message:
// "Founder profile: D-primary (0.72), high Quick Start (state: 9, usually 8),
//  currently in SHIPPING_SPRINT context. Communication: direct, low formality.
//  Complement focus: Tenacity (0.33), Follow Through (0.22)."
```

### 12.6 Adaptation Lifecycle Example

```
Week 1 (Assessment + Early Signals):
  ├── Founder completes assessment: QS=8, FT=3, FF=4, IM=6
  ├── TRAIT initialized from assessment
  ├── STATE initialized = TRAIT (no behavioral data yet)
  ├── α = 0.70 everywhere (trait-dominant)
  ├── Complement: FT=0.28, FF=0.24, QS=0.08, IM=0.16 (heavy on Follow Through)
  ├── Calibration: structure-heavy, checklist-oriented
  └── Context: none active

Week 2 (Behavioral Data Accumulates):
  ├── 47 chat messages analyzed → DISC inference: D=0.65, C=0.25
  ├── Shipping cadence CV = 2.1 → confirms burst pattern (QS validation)
  ├── Completion ratio = 0.45 → lower than average (FT validation)
  ├── Trait EMA barely moves (λ_slow)
  ├── State = Trait (no drift yet)
  ├── Communication calibration tightens: shorter messages, bottom-line-first
  └── Confidence: S2=0.52, S3=0.41 → Pairing score ~45 (Partner level)

Week 3 (Financial Pressure Detected):
  ├── 3 messages mention "debt", "money tight", "can't afford"
  ├── Stripe shows revenue dip: $400 → $200
  ├── FINANCIAL_PRESSURE window activates (strength=0.78)
  ├── STATE shifts: QS drops to 5 (less starting, more grinding)
  │   FT rises to 6 (suddenly finishing things)
  │   drift(QS) = 1.8, drift(FT) = 1.5 → significant
  ├── α(QS) drops to 0.38, α(FT) drops to 0.42 → state-dominant
  ├── EFFECTIVE QS = 0.38×8 + 0.62×5 = 6.1 (down from 8)
  ├── Complement REBALANCES: less FT supplement needed (they're doing it)
  │   MORE Galvanizing (help them sell), MORE Discernment (help them choose what to cut)
  ├── Calibration shifts: revenue-first recs, subscription audit, "what can you ship for $?"
  └── Wirebot surfaces: "I notice you're more focused than usual. Revenue pressure?
      Here's what I see as the fastest path to $500 this week."

Week 5 (Pressure Resolves):
  ├── Stripe shows revenue recovery: $800
  ├── Messages shift back to creative topics
  ├── FINANCIAL_PRESSURE window decays (signals stop)
  ├── STATE drifts back toward TRAIT
  ├── α recovers toward 0.70
  ├── Complement rebalances back to baseline
  └── Wirebot: "Revenue stabilized. Nice. Ready to get back to building?"

Month 3 (Deep Profile):
  ├── 300+ messages, 90 days behavioral data, 3 connected accounts
  ├── First retest: FT self-report rises from 3→5 (founder is learning)
  │   But behavioral FT is still 4.2 → delta of 0.8 (mild gap)
  ├── TRAIT updates incorporate retest: FT_trait = 4.5 (average of history)
  ├── Profile coherence = 0.78 (good — self-perception improving)
  ├── Pairing score: 82 → BONDED level
  └── Wirebot now acts with high autonomy within calibrated parameters

Season 2 (New Context):
  ├── Revenue goal shifts from "break even" to "grow"
  ├── New business started → business count 4→5
  ├── CONTEXT_EXPLOSION detected (5 projects, daily switches)
  ├── QS STATE spikes to 10 (they're starting everything)
  ├── Wirebot AUTO-ADJUSTS: "You've touched 5 projects this week.
  │   I know you're excited. Pick TWO for this month. I'll hold the rest."
  ├── Complement: heavy Discernment (help choose), heavy Tenacity (help finish chosen ones)
  └── Calibration: sequencing-heavy, focus-protecting, celebration for completion not starting
```

---

*This specification defines a measurement system, not a feature.
Every formula is implementable. Every weight is adjustable.
Every confidence interval is real. The system doesn't guess — it converges.
And it never stops converging. As the founder changes, the profile changes.
As the profile changes, the complement changes. The algorithm breathes with the human.*
