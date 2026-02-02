// Package main — Wirebot Scoreboard + Pairing Engine
//
// This file implements the Pairing Engine: a continuously-running background
// system that builds a psychometric-grade Founder Profile from every observable
// signal. The engine processes chat messages, scoreboard events, assessment
// answers, approval actions, document uploads, and connected account data
// through a unified 9-step pipeline.
//
// ## Architecture
//
// All signals enter via pe.Ingest() (non-blocking channel send) and are
// processed serially in a single goroutine. Serial processing is required
// because the dual-track EMA system depends on observation ordering.
//
// Pipeline per signal:
//   1. Feature extraction (NLP for messages, pass-through for events)
//   2. Evidence entry creation
//   3. Signal-type-specific processing (updates relevant constructs)
//   4. Drift detection (trait vs state divergence across all constructs)
//   5. Context window evaluation (decay + check active windows)
//   6. Complement rebalance (inverse of founder's effective scores)
//   7. Calibration update (translate profile → Wirebot behavior params)
//   8. Evidence summary
//   9. Meta counter update + dirty flag
//
// ## Data Model
//
// The Founder Profile contains 7 constructs (Φ1-Φ7), each implemented as a
// DualTrackDimension with independent trait (slow EMA, λ=0.02) and state
// (fast EMA, λ=0.15) readings per named dimension. The blend coefficient α
// automatically shifts from trait-dominant (stable periods) to state-dominant
// (turbulent periods) based on detected drift.
//
// ## Persistence
//
// Profile is stored as JSON at /data/wirebot/pairing/profile.json, written
// every 10 signals or 60 seconds (whichever comes first). Evidence entries
// are kept in memory (capped at 10,000, oldest evicted).
//
// ## Related Documentation
//
//   - docs/PAIRING_ENGINE.md  — Implementation walkthrough (this code)
//   - docs/PAIRING_SCIENCE.md — Mathematical specification (every formula)
//   - docs/PAIRING_V2.md      — UI/UX specification (assessment + equalizer)
//   - docs/PAIRING.md          — v1 protocol (22 conversational questions)
package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// SIGNAL TYPES — The 6 input categories the engine processes
//
// Every observable event in the system is classified into one of these types.
// Each type has a dedicated processor that knows how to extract profile-relevant
// information from the signal's content and metadata.
// ═══════════════════════════════════════════════════════════════════════════════

// SignalType classifies the input source. Determines which processor handles it.
type SignalType string

const (
	// SignalMessage — a chat message from the founder (or Wirebot's response).
	// Processed by: processMessage(). Affects: CommunicationDNA, CognitiveStyle,
	// ObservedComm, and context windows (financial_pressure, life_event).
	SignalMessage SignalType = "message"

	// SignalEvent — a scoreboard event (TASK_COMPLETED, PAYOUT_RECEIVED, etc.).
	// Processed by: processEvent(). Affects: ActionStyle, TemporalPatterns,
	// BusinessReality, and context windows (sprint, stall, celebration, etc.).
	SignalEvent SignalType = "event"

	// SignalDocument — an ingested document (business plan, pitch deck, etc.).
	// Processed by: processDocument(). Affects: CognitiveStyle via vocabulary
	// and structure analysis.
	SignalDocument SignalType = "document"

	// SignalAccount — data from a connected account poller (Stripe, GitHub, etc.).
	// Processed by: processAccountData(). Affects: BusinessReality (ground truth
	// revenue, shipping velocity).
	SignalAccount SignalType = "account"

	// SignalAssessment — batch of assessment answers (ASI-12, CSI-8, ETM-6, RDS-6, COG-8).
	// Processed by: processAssessment(). Affects: whichever construct the instrument
	// measures. This is the primary self-report input.
	SignalAssessment SignalType = "assessment"

	// SignalApproval — operator approved or rejected a pending scoreboard event.
	// Processed by: processApproval(). Affects: ActionStyle (decision speed) and
	// EnergyTopology (discernment signal on reject).
	SignalApproval SignalType = "approval"
)

// Signal is the universal input to the pairing engine. Every observable event
// in the system is wrapped in a Signal before being ingested.
//
// Fields:
//   - Type:      which processor handles this signal
//   - Source:    origin identifier ("chat", "scoreboard", "stripe", "github", etc.)
//   - Timestamp: when the underlying event occurred (not when ingested)
//   - Content:   raw text for messages and documents (empty for events/approvals)
//   - Metadata:  structured data (event_type, lane, amount, provider, latency, etc.)
//   - Features:  populated by extractFeatures() during processing (NLP scores, etc.)
type Signal struct {
	Type      SignalType             `json:"type"`
	Source    string                 `json:"source"`
	Timestamp time.Time             `json:"timestamp"`
	Content   string                `json:"content,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Features  map[string]float64     `json:"features,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// EVIDENCE TRAIL — Transparency layer
//
// Every signal that enters the engine produces an EvidenceEntry: a full record
// of what features were extracted, what profile parameters changed, and which
// constructs were affected. This is what the Profile Equalizer UI reads to show
// the founder exactly why their scores are what they are.
//
// Evidence entries are kept in memory (capped at 10,000, FIFO eviction) and
// served via GET /v1/pairing/evidence.
// ═══════════════════════════════════════════════════════════════════════════════

// EvidenceEntry records what a single signal did to the founder profile.
// The UI displays these in a chronological feed (Evidence Log tab).
//
// Fields:
//   - FeaturesExtracted: all NLP/metadata features computed from the signal
//   - ProfileImpact: which profile parameters changed and by how much
//     (e.g., "drift.action_style.QS": 2.3 or "context.SHIPPING_SPRINT": 0.82)
//   - ConstructsAffected: list of construct names touched (for filtering)
type EvidenceEntry struct {
	ID             int64              `json:"id"`
	Timestamp      time.Time          `json:"timestamp"`
	SignalType     SignalType         `json:"signal_type"`
	Source         string             `json:"source"`
	Summary        string             `json:"summary"`
	FeaturesExtracted map[string]float64 `json:"features_extracted"`
	ProfileImpact  map[string]float64 `json:"profile_impact"`
	ConstructsAffected []string       `json:"constructs_affected"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// PREDICTION TRACKING — Accuracy self-measurement
//
// The engine records predictions it makes about founder behavior and later
// checks whether they were correct. This feeds the accuracy ledger and enables
// self-correcting weight adjustment (PAIRING_SCIENCE.md §13.4).
// ═══════════════════════════════════════════════════════════════════════════════

// PredictionEntry records a prediction the engine made and whether it resolved
// correctly. Used for accuracy tracking and weight self-correction.
//
// Example: Predicted "founder will respond within 2h at 10 PM nudge",
// Actual: responded in 45 min → correct=true, error=0.625.
type PredictionEntry struct {
	ID          int64     `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Parameter   string    `json:"parameter"`
	Predicted   string    `json:"predicted"`
	Actual      string    `json:"actual,omitempty"`
	Error       float64   `json:"error"`
	Correct     bool      `json:"correct"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// DUAL-TRACK EMA SYSTEM — The mathematical core
//
// Every dimension in every construct has two exponential moving averages:
//
//   TRAIT (slow, λ=0.02, half-life ≈ 35 obs):  Who the founder IS
//   STATE (fast, λ=0.15, half-life ≈ 4 obs):   Who the founder is RIGHT NOW
//
// The effective score blends them: effective = α × trait + (1-α) × state
//
// α adjusts automatically based on drift:
//   No drift:   α = 0.70 (trust the long-term identity)
//   High drift: α = 0.30 (trust the current reading)
//
// This means Wirebot calibrates to "who you are right now" during turbulent
// periods and "who you really are" during stable periods — without anyone
// needing to flip a switch.
//
// See: PAIRING_SCIENCE.md §0 (The Living Profile Principle)
// ═══════════════════════════════════════════════════════════════════════════════

// DualTrackDimension holds both the slow-moving trait estimate and the fast-moving
// state estimate for a set of named dimensions within a construct.
//
// For example, ActionStyle has dimensions ["FF", "FT", "QS", "IM"] — each gets
// independent trait, state, drift, alpha, effective, sigma, and observation count.
//
// Fields:
//   - Trait:        slow EMA (λ=0.02), barely moves, captures stable identity
//   - State:        fast EMA (λ=0.15), responsive, captures current mode
//   - Drift:        |state - trait| / σ_trait per dimension — how far current ≠ baseline
//   - Alpha:        blend coefficient per dimension — 0.30 (state-heavy) to 0.70 (trait-heavy)
//   - Effective:    α × trait + (1-α) × state — the score everything downstream reads
//   - SigmaTrait:   running variance estimate of trait (for drift normalization)
//   - Observations: count per dimension (for confidence calculation)
type DualTrackDimension struct {
	Trait     map[string]*float64 `json:"trait"`
	State     map[string]*float64 `json:"state"`
	Drift     map[string]float64  `json:"drift"`
	Alpha     map[string]float64  `json:"alpha"`
	Effective map[string]*float64 `json:"effective"`
	SigmaTrait map[string]float64 `json:"sigma_trait"`
	Observations map[string]int   `json:"observations"`
}

// NewDualTrackDimension creates a fresh dual-track for the given dimension names.
// Initial state: α=0.70 (trait-dominant), σ_trait=2.0 (wide uncertainty), no observations.
func NewDualTrackDimension(dims []string) *DualTrackDimension {
	dt := &DualTrackDimension{
		Trait:        make(map[string]*float64),
		State:        make(map[string]*float64),
		Drift:        make(map[string]float64),
		Alpha:        make(map[string]float64),
		Effective:    make(map[string]*float64),
		SigmaTrait:   make(map[string]float64),
		Observations: make(map[string]int),
	}
	for _, d := range dims {
		dt.Alpha[d] = 0.70          // start trait-dominant
		dt.SigmaTrait[d] = 2.0     // wide initial uncertainty
	}
	return dt
}

// UpdateEMA processes a new observation for a named dimension, updating:
//   1. Trait EMA (slow, λ=0.02) — captures long-term stable tendency
//   2. State EMA (fast, λ=0.15) — captures current operating mode
//   3. SigmaTrait — running variance estimate (EMA of |observation - trait|)
//   4. Drift — |state - trait| / σ_trait (normalized divergence)
//   5. Alpha — blend coefficient: 0.30 + 0.40 × (1/(1+drift))
//   6. Effective — α × trait + (1-α) × state (the score downstream reads)
//
// On first observation for a dimension, both trait and state initialize to the
// observed value (no warm-up period). Subsequent observations update via EMA.
//
// The value parameter should be on a 0-10 scale for consistency across constructs.
// Assessment scores, NLP-inferred scores, and behavioral metrics are all normalized
// to 0-10 before calling UpdateEMA.
//
// See: PAIRING_SCIENCE.md §0 (Trait vs. State Separation)
func (dt *DualTrackDimension) UpdateEMA(dim string, value float64) {
	lambdaSlow := 0.02  // trait half-life ≈ 35 observations
	lambdaFast := 0.15  // state half-life ≈ 4 observations

	dt.Observations[dim]++

	if dt.Trait[dim] == nil {
		// First observation: initialize both trait and state to observed value
		v := value
		dt.Trait[dim] = &v
		s := value
		dt.State[dim] = &s
	} else {
		// Trait EMA (slow — long-term stable tendency)
		// Barely moves even with unusual observations. 35 observations to half-update.
		t := *dt.Trait[dim]*(1-lambdaSlow) + value*lambdaSlow
		dt.Trait[dim] = &t

		// State EMA (fast — current operating mode)
		// Responsive to recent behavior. 4 observations to half-update.
		s := *dt.State[dim]*(1-lambdaFast) + value*lambdaFast
		dt.State[dim] = &s
	}

	// Update sigma: running estimate of trait variance (for drift normalization)
	// Uses EMA with λ=0.05 on absolute deviations from trait.
	// Floor of 0.1 prevents division-by-zero in drift formula.
	if dt.Observations[dim] > 2 {
		diff := value - *dt.Trait[dim]
		dt.SigmaTrait[dim] = dt.SigmaTrait[dim]*0.95 + math.Abs(diff)*0.05
		if dt.SigmaTrait[dim] < 0.1 {
			dt.SigmaTrait[dim] = 0.1
		}
	}

	// Compute drift: how far state has diverged from trait, in σ units
	//   drift < 1.0:  normal variance (no action)
	//   drift 1.0-2.0: mild shift (α adjusts, logged)
	//   drift ≥ 2.0:  significant shift (DriftEvent recorded, context may activate)
	if dt.Trait[dim] != nil && dt.State[dim] != nil {
		sigma := dt.SigmaTrait[dim]
		if sigma < 0.1 {
			sigma = 0.1
		}
		dt.Drift[dim] = math.Abs(*dt.State[dim]-*dt.Trait[dim]) / sigma
	}

	// Compute alpha: blend coefficient that shifts trust between trait and state
	//   stability = 1/(1+drift): high when stable, low when drifting
	//   α = 0.30 + 0.40 × stability: range [0.30, 0.70]
	//   0.70 = mostly trait (stable periods)
	//   0.30 = mostly state (turbulent periods, high drift)
	drift := dt.Drift[dim]
	stability := 1.0 / (1.0 + drift)
	dt.Alpha[dim] = 0.30 + 0.40*stability

	// Compute effective: the blended score that all downstream systems read
	//   effective = α × trait + (1-α) × state
	if dt.Trait[dim] != nil && dt.State[dim] != nil {
		a := dt.Alpha[dim]
		e := a*(*dt.Trait[dim]) + (1-a)*(*dt.State[dim])
		dt.Effective[dim] = &e
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// CONTEXT WINDOWS — Auto-detected operating modes
//
// Context windows represent detected environmental conditions that modulate
// how Wirebot interprets behavior and calibrates its responses. They activate
// when enough signals accumulate and decay exponentially when signals stop.
//
// Multiple windows can be active simultaneously. Their calibration overrides
// compose additively (clamped to valid ranges).
//
// See: PAIRING_SCIENCE.md §0 (Seasonal & Contextual Modulation)
// ═══════════════════════════════════════════════════════════════════════════════

// ContextWindowType enumerates the 7 detectable operating modes.
type ContextWindowType string

const (
	CtxFinancialPressure ContextWindowType = "FINANCIAL_PRESSURE"  // Revenue drop, debt keywords, Stripe failures. τ=72h.
	CtxShippingSprint    ContextWindowType = "SHIPPING_SPRINT"    // 5+ ships in 72h. τ=48h. Calibration: reduce nudges.
	CtxRecoveryPeriod    ContextWindowType = "RECOVERY_PERIOD"    // Ship after stall. τ=72h. Calibration: protect, don't push.
	CtxContextExplosion  ContextWindowType = "CONTEXT_EXPLOSION"  // 5+ unique projects/day. τ=48h. Calibration: focus prompts.
	CtxStall             ContextWindowType = "STALL"              // 24h+ since last ship. τ=24h. Calibration: increase nudges.
	CtxCelebration       ContextWindowType = "CELEBRATION"        // Major revenue or product launch. τ=24h. Amplify win.
	CtxLifeEvent         ContextWindowType = "LIFE_EVENT"         // Health/family/moving keywords. τ=168h (7d). Full protection.
)

// ContextWindow tracks the activation level of a detected operating mode.
//
// Lifecycle: Inactive(0) → Activating(<0.3) → Active(≥0.3) → Decaying → Deactivated(0)
//
// When Activation ≥ 0.3, the window's calibration overrides are applied to Wirebot's
// behavior parameters (nudge frequency, risk framing, intervention timing, etc.).
//
// Fields:
//   - Activation:    0.0 to ~1.0 (sigmoid-bounded). Strength of the detected context.
//   - DecayTauHours: exponential decay time constant. After τ hours without new signals,
//     activation drops to ~37% of its value. Default varies by window type (24h-168h).
type ContextWindow struct {
	Name         ContextWindowType `json:"name"`
	Activation   float64           `json:"activation"`
	ActivatedAt  *time.Time        `json:"activated_at,omitempty"`
	LastSignal   *time.Time        `json:"last_signal,omitempty"`
	SignalCount  int               `json:"signal_count"`
	DecayTauHours float64          `json:"decay_tau_hours"`
}

// AddSignal pushes the window's activation up via sigmoid compression.
// Multiple signals compound: 3 signals with strength 0.5 → activation ≈ 0.72.
// Sigmoid prevents activation from exceeding 1.0 regardless of signal count.
func (cw *ContextWindow) AddSignal(strength float64) {
	cw.SignalCount++
	now := time.Now()
	cw.LastSignal = &now
	if cw.ActivatedAt == nil {
		cw.ActivatedAt = &now
	}
	cw.Activation = sigmoid(cw.Activation + strength*0.3)
}

// Decay applies exponential decay: activation *= e^(-hours/τ).
// When activation drops below 0.05, the window fully deactivates (reset to zero).
// Called every 5 minutes by the periodic tasks goroutine.
func (cw *ContextWindow) Decay(now time.Time) {
	if cw.LastSignal == nil {
		return
	}
	hours := now.Sub(*cw.LastSignal).Hours()
	if cw.DecayTauHours <= 0 {
		cw.DecayTauHours = 72 // default 3 days
	}
	cw.Activation *= math.Exp(-hours / cw.DecayTauHours)
	if cw.Activation < 0.05 {
		cw.Activation = 0
		cw.ActivatedAt = nil
		cw.SignalCount = 0
	}
}

// sigmoid bounds a value between 0 and 1: 1/(1+e^(-x)).
// Used for context window activation to prevent unbounded growth.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// ═══════════════════════════════════════════════════════════════════════════════
// OVERRIDE SYSTEM — Founder challenges algorithmic scores
//
// The founder can manually correct any score they believe is wrong. Overrides
// are treated as a high-confidence single observation that decays over 30 days.
//
// If behavioral data CONFIRMS the override (observed score within ±1.0), the
// override becomes permanent (weight = 0.15 constant).
// If behavioral data CONTRADICTS it (delta > 2.0), it decays normally.
//
// See: PAIRING_V2.md (Override Tab)
// ═══════════════════════════════════════════════════════════════════════════════

// ProfileOverride records a founder's manual correction to a profile score.
// Submitted via POST /v1/pairing/override or `wb override <trait> <dim> <value>`.
type ProfileOverride struct {
	ID          int64     `json:"id"`
	Trait       string    `json:"trait"`
	Dimension   string    `json:"dimension"`
	Value       float64   `json:"value"`
	Reason      string    `json:"reason"`
	CreatedAt   time.Time `json:"created_at"`
	Weight      float64   `json:"weight"`
	Confirmed   bool      `json:"confirmed"`
	Contradicted bool     `json:"contradicted"`
}

// CurrentWeight returns the override's current influence on the composite score.
// Fresh override: 0.30. Decays exponentially with τ=30 days.
// Confirmed overrides get permanent weight of 0.15 (behavioral data matched).
func (o *ProfileOverride) CurrentWeight() float64 {
	if o.Confirmed {
		return 0.15 // permanent — behavioral data confirmed the founder's correction
	}
	// Exponential decay: 0.30 × e^(-days/30)
	// Day 7: 0.24, Day 14: 0.19, Day 30: 0.10, Day 60: 0.04
	days := time.Since(o.CreatedAt).Hours() / 24
	return 0.30 * math.Exp(-days/30.0)
}

// ═══════════════════════════════════════════════════════════════════════════════
// COMPLEMENT VECTOR — Where Wirebot allocates effort
//
// The complement vector answers: "What should Wirebot do more of?"
// It's the INVERSE of the founder's effective scores:
//   gap(dim) = max(0, 10 - effective(dim))
//   complement(dim) = gap(dim) / Σ(all gaps)
//
// A founder scoring 9/10 on Quick Start gets almost no QS supplementation.
// A founder scoring 1/10 on Tenacity gets heavy Tenacity support.
//
// The vector sums to 1.0 and is recomputed after EVERY signal that changes
// any construct — so it shifts in real-time as the founder's state shifts.
//
// See: PAIRING_SCIENCE.md §0 (The Complement Shifts With the Founder)
// ═══════════════════════════════════════════════════════════════════════════════

// ComplementVector represents Wirebot's proportional effort allocation across
// 10 capability dimensions (4 action style + 6 energy topology). Each field
// is a proportion (0.0-1.0) that sums to 1.0 across all fields.
type ComplementVector struct {
	FactFinder   float64    `json:"fact_finder"`
	FollowThrough float64  `json:"follow_through"`
	QuickStart   float64    `json:"quick_start"`
	Implementor  float64    `json:"implementor"`
	Wonder       float64    `json:"wonder"`
	Invention    float64    `json:"invention"`
	Discernment  float64    `json:"discernment"`
	Galvanizing  float64    `json:"galvanizing"`
	Enablement   float64    `json:"enablement"`
	Tenacity     float64    `json:"tenacity"`
	LastRebalanced *time.Time `json:"last_rebalanced,omitempty"`
}

// Rebalance recalculates the complement vector from the founder's current
// effective scores. For each dimension: gap = max(0, 10 - effective).
// Then all gaps are normalized to sum to 1.0.
//
// The effect: founder's weaknesses get the most Wirebot attention.
// Founder's strengths get minimal supplementation.
//
// Reads from ActionStyle.Effective (FF, FT, QS, IM) and
// EnergyTopology.Effective (W, N, D_disc, G, E, T).
func (cv *ComplementVector) Rebalance(actionEffective map[string]*float64, energyEffective map[string]*float64) {
	gaps := make(map[string]float64)
	total := 0.0

	// Action style gaps (10 - effective = gap)
	for dim, eff := range actionEffective {
		if eff != nil {
			gap := math.Max(0, 10-*eff)
			gaps[dim] = gap
			total += gap
		}
	}
	// Energy topology gaps
	for dim, eff := range energyEffective {
		if eff != nil {
			gap := math.Max(0, 10-*eff)
			gaps[dim] = gap
			total += gap
		}
	}

	if total < 0.01 {
		total = 1.0
	}

	cv.FactFinder = gaps["FF"] / total
	cv.FollowThrough = gaps["FT"] / total
	cv.QuickStart = gaps["QS"] / total
	cv.Implementor = gaps["IM"] / total
	cv.Wonder = gaps["W"] / total
	cv.Invention = gaps["N"] / total
	cv.Discernment = gaps["D"] / total
	cv.Galvanizing = gaps["G"] / total
	cv.Enablement = gaps["E"] / total
	cv.Tenacity = gaps["T"] / total

	now := time.Now()
	cv.LastRebalanced = &now
}

// AdjustFromSecondary applies secondary construct data to modulate the complement
// vector. This ensures BIZ reality, temporal patterns, risk, and cognitive style
// influence HOW Wirebot complements (not just Action + Energy gaps).
func (cv *ComplementVector) AdjustFromSecondary(
	risk map[string]*float64,
	cognitive map[string]*float64,
	business map[string]*float64,
	temporal map[string]*float64,
) {
	// High debt pressure → boost Tenacity (founder needs someone who won't let up on revenue)
	if business != nil {
		if dp, ok := business["debt_pressure"]; ok && dp != nil && *dp >= 6 {
			cv.Tenacity = math.Min(1.0, cv.Tenacity*1.3)
		}
		// Solo operator → boost Enablement (founder has no one else to lean on)
		if ts, ok := business["team_size"]; ok && ts != nil && *ts <= 2 {
			cv.Enablement = math.Min(1.0, cv.Enablement*1.2)
		}
	}

	// Low risk tolerance → boost Quick Start (Wirebot pushes past fear)
	if risk != nil {
		if tol, ok := risk["tolerance"]; ok && tol != nil && *tol < 4 {
			cv.QuickStart = math.Min(1.0, cv.QuickStart*1.3)
		}
	}

	// Sequential thinker → boost Wonder (Wirebot injects big-picture thinking)
	if cognitive != nil {
		if seq, ok := cognitive["sequential"]; ok && seq != nil && *seq > 7 {
			cv.Wonder = math.Min(1.0, cv.Wonder*1.2)
		}
		// Abstract thinker → boost Implementor (ground the ideas)
		if abs, ok := cognitive["abstract"]; ok && abs != nil && *abs > 7 {
			cv.Implementor = math.Min(1.0, cv.Implementor*1.2)
		}
	}

	// High context switch cost → boost Follow Through (keep them on track)
	if temporal != nil {
		if sw, ok := temporal["context_switch_cost"]; ok && sw != nil && *sw >= 7 {
			cv.FollowThrough = math.Min(1.0, cv.FollowThrough*1.2)
		}
	}

	// Renormalize to sum=1.0
	total := cv.FactFinder + cv.FollowThrough + cv.QuickStart + cv.Implementor +
		cv.Wonder + cv.Invention + cv.Discernment + cv.Galvanizing + cv.Enablement + cv.Tenacity
	if total > 0 {
		cv.FactFinder /= total
		cv.FollowThrough /= total
		cv.QuickStart /= total
		cv.Implementor /= total
		cv.Wonder /= total
		cv.Invention /= total
		cv.Discernment /= total
		cv.Galvanizing /= total
		cv.Enablement /= total
		cv.Tenacity /= total
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// CALIBRATION PARAMETERS — How the profile translates into Wirebot behavior
//
// These are the concrete parameters that shape every Wirebot interaction:
// message length, tone, nudge frequency, recommendation style, proactive timing.
//
// They're recomputed after every signal from:
//   1. ObservedComm (directness, formality, emotion) → Communication params
//   2. DISC effective scores (D/I/S/C primary) → Lead-with style, question freq
//   3. Temporal patterns (peak_hour) → Standup hour
//   4. Active context windows → Override specific params (sprint=less nudging, etc.)
//
// See: PAIRING_SCIENCE.md §9 (Calibration Engine)
// ═══════════════════════════════════════════════════════════════════════════════

// CalibrationParams holds the concrete behavioral parameters that Wirebot uses
// for every interaction. Injected into chat context alongside the profile summary.
type CalibrationParams struct {
	Communication struct {
		MaxMessageWords    int     `json:"max_message_words"`
		LeadWith           string  `json:"lead_with"` // recommendation, data, vision, context
		ToneFormality      float64 `json:"tone_formality"`
		EmojiMirrorRatio   float64 `json:"emoji_mirror_ratio"`
		QuestionFrequency  string  `json:"question_frequency"`
		CelebrationIntensity float64 `json:"celebration_intensity"`
	} `json:"communication"`
	Accountability struct {
		NudgeFrequencyHours  float64 `json:"nudge_frequency_hours"`
		NudgeIntensity       float64 `json:"nudge_intensity"`
		DeadlinePressure     float64 `json:"deadline_pressure"`
		StreakEmphasis       float64 `json:"streak_emphasis"`
		StallInterventionH   float64 `json:"stall_intervention_hours"`
	} `json:"accountability"`
	Recommendations struct {
		OptionsPresented int     `json:"options_presented"`
		DataDensity      float64 `json:"data_density"`
		PlanningDepth    string  `json:"planning_depth"` // minimal, moderate, detailed
		RiskFraming      string  `json:"risk_framing"`   // optimistic, balanced, cautious
	} `json:"recommendations"`
	Proactive struct {
		StandupHour         int     `json:"standup_hour"`
		PeakTaskType        string  `json:"peak_task_type"`
		OffpeakTaskType     string  `json:"offpeak_task_type"`
		InterventionThreshH float64 `json:"intervention_threshold_hours"`
	} `json:"proactive"`
}

// defaultCalibration returns neutral/moderate defaults for all parameters.
// These are used before any profile data exists and are progressively
// overridden as the engine learns the founder's style.
func defaultCalibration() CalibrationParams {
	c := CalibrationParams{}
	c.Communication.MaxMessageWords = 300
	c.Communication.LeadWith = "recommendation"
	c.Communication.ToneFormality = 0.50
	c.Communication.EmojiMirrorRatio = 0.50
	c.Communication.QuestionFrequency = "moderate"
	c.Communication.CelebrationIntensity = 0.50
	c.Accountability.NudgeFrequencyHours = 8
	c.Accountability.NudgeIntensity = 0.50
	c.Accountability.DeadlinePressure = 0.50
	c.Accountability.StreakEmphasis = 0.50
	c.Accountability.StallInterventionH = 8
	c.Recommendations.OptionsPresented = 2
	c.Recommendations.DataDensity = 0.50
	c.Recommendations.PlanningDepth = "moderate"
	c.Recommendations.RiskFraming = "balanced"
	c.Proactive.StandupHour = 8
	c.Proactive.PeakTaskType = "genius_work"
	c.Proactive.OffpeakTaskType = "frustration_work"
	c.Proactive.InterventionThreshH = 8
	return c
}

// ═══════════════════════════════════════════════════════════════════════════════
// ACCURACY TRACKING — The system measures its own performance
//
// Accuracy is not assumed — it's computed from the convergence equation and
// tracked per-construct. The system knows how confident it is in each score
// and can tell the founder exactly what would improve accuracy next.
//
// See: PAIRING_SCIENCE.md §13 (Accuracy Convergence)
// See: PAIRING_SCIENCE.md §14 (The Convergence Equation)
// ═══════════════════════════════════════════════════════════════════════════════

// AccuracyLedger tracks the system's self-measured accuracy metrics.
// Displayed in the Accuracy Tab of the Profile Equalizer UI.
type AccuracyLedger struct {
	OverallAccuracy   float64                   `json:"overall_accuracy"`
	ByConstruct       map[string]ConstructAccuracy `json:"by_construct"`
	CalibrationLift   map[string]float64         `json:"calibration_lift"`
	DriftPatterns     DriftPatternSummary        `json:"drift_patterns"`
	ImprovementVsDay1 float64                    `json:"improvement_vs_day1"`
	LastUpdated       time.Time                  `json:"last_updated"`
}

type ConstructAccuracy struct {
	Confidence    float64 `json:"confidence"`
	Observations  int     `json:"observations"`
	BestPredictor string  `json:"best_predictor"`
	Improvement   float64 `json:"improvement_vs_day1"`
}

type DriftPatternSummary struct {
	TotalDetected   int     `json:"total_detected"`
	Anticipated     int     `json:"correctly_anticipated"`
	MostCommon      string  `json:"most_common_context"`
	MeanRecoveryErr float64 `json:"mean_recovery_prediction_error_days"`
}

// ─── Drift Event (historical) ────────────────────────────────────────────────

type DriftEvent struct {
	ID          int64             `json:"id"`
	Timestamp   time.Time         `json:"timestamp"`
	Construct   string            `json:"construct"`
	Dimension   string            `json:"dimension"`
	Magnitude   float64           `json:"magnitude"`
	Context     ContextWindowType `json:"context,omitempty"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
	RecoveryDays float64          `json:"recovery_days,omitempty"`
}

// ─── Assessment Answer ───────────────────────────────────────────────────────

type AssessmentAnswer struct {
	InstrumentID string      `json:"instrument_id"` // ASI-12, CSI-8, ETM-6, RDS-6, COG-8
	QuestionID   string      `json:"question_id"`
	Value        interface{} `json:"value"` // string for choice, float64 for slider, []string for sort
	AnsweredAt   time.Time   `json:"answered_at"`
}

// ─── Founder Profile v2 ─────────────────────────────────────────────────────

type FounderProfileV2 struct {
	Version    int    `json:"version"`
	ProfileID  string `json:"profile_id"`

	PairingScore struct {
		Composite  float64 `json:"composite"`
		Level      string  `json:"level"`
		Components map[string]float64 `json:"components"` // S1..S9
	} `json:"pairing_score"`

	// The 7 constructs with dual-track
	ActionStyle       *DualTrackDimension `json:"action_style"`       // Φ1: FF, FT, QS, IM
	CommunicationDNA  *DualTrackDimension `json:"communication_dna"`  // Φ2: D, I, S, C
	EnergyTopology    *DualTrackDimension `json:"energy_topology"`    // Φ3: W, N, D, G, E, T
	RiskDisposition   *DualTrackDimension `json:"risk_disposition"`   // Φ4: tolerance, speed, ambiguity, sunk_cost, loss_aversion, bias_to_action
	BusinessReality   *DualTrackDimension `json:"business_reality"`   // Φ5: stage, revenue, debt, focus
	TemporalPatterns  *DualTrackDimension `json:"temporal_patterns"`  // Φ6: chronotype, consistency, peak_hour
	CognitiveStyle    *DualTrackDimension `json:"cognitive_style"`    // Φ7: holistic, sequential, abstract, concrete

	// Observed communication style
	ObservedComm struct {
		Directness       float64 `json:"directness"`
		Formality        float64 `json:"formality"`
		DetailPreference float64 `json:"detail_preference"`
		EmotionExpr      float64 `json:"emotion_expression"`
		PacePreference   float64 `json:"pace_preference"`
		DecisionStyle    float64 `json:"decision_style"`
		MessagesAnalyzed int     `json:"messages_analyzed"`
		Confidence       float64 `json:"confidence"`
	} `json:"observed_comm"`

	// Self-perception deltas (self-report minus behavioral)
	SelfPerceptionDeltas map[string]float64 `json:"self_perception_deltas"`

	// Context windows
	ContextWindows map[ContextWindowType]*ContextWindow `json:"context_windows"`

	// Complement vector
	Complement ComplementVector `json:"complement"`

	// Calibration
	Calibration CalibrationParams `json:"calibration"`

	// Overrides
	Overrides []ProfileOverride `json:"overrides"`

	// Assessment answers
	Answers []AssessmentAnswer `json:"answers"`

	// Accuracy
	Accuracy AccuracyLedger `json:"accuracy"`

	// Meta
	Meta struct {
		CreatedAt          time.Time `json:"created_at"`
		LastAssessment     *time.Time `json:"last_assessment,omitempty"`
		LastRetest         *time.Time `json:"last_retest,omitempty"`
		LastInferenceUpdate *time.Time `json:"last_inference_update,omitempty"`
		LastBehavioralBatch *time.Time `json:"last_behavioral_batch,omitempty"`
		LastDriftCheck     *time.Time `json:"last_drift_check,omitempty"`
		LastComplementRebal *time.Time `json:"last_complement_rebalance,omitempty"`
		LastContextWindowEval *time.Time `json:"last_context_window_eval,omitempty"`
		TotalMessagesAnalyzed int     `json:"total_messages_analyzed"`
		TotalEventsAnalyzed   int     `json:"total_events_analyzed"`
		TotalDocsIngested     int     `json:"total_documents_ingested"`
		TotalStateShifts      int     `json:"total_state_shifts_detected"`
		SignalsProcessed      int     `json:"signals_processed"`
		ConnectedAccounts     []string `json:"connected_accounts"`
		EngineVersion         string   `json:"engine_version"`
	} `json:"meta"`
}

func NewFounderProfile() *FounderProfileV2 {
	p := &FounderProfileV2{
		Version:   2,
		ProfileID: generateID(),
	}

	p.ActionStyle = NewDualTrackDimension([]string{"FF", "FT", "QS", "IM"})
	p.CommunicationDNA = NewDualTrackDimension([]string{"D", "I", "S", "C"})
	p.EnergyTopology = NewDualTrackDimension([]string{"W", "N", "D_disc", "G", "E", "T"})
	p.RiskDisposition = NewDualTrackDimension([]string{"tolerance", "speed", "ambiguity", "sunk_cost", "loss_aversion", "bias_to_action"})
	p.BusinessReality = NewDualTrackDimension([]string{"focus", "revenue_maturity", "team_size", "bottleneck", "venture_age", "debt_pressure"})
	p.TemporalPatterns = NewDualTrackDimension([]string{"peak_hour", "planning_style", "stall_recovery", "work_intensity", "context_switch_cost", "planning_horizon"})
	p.CognitiveStyle = NewDualTrackDimension([]string{"holistic", "sequential", "abstract", "concrete"})

	p.ContextWindows = map[ContextWindowType]*ContextWindow{
		CtxFinancialPressure: {Name: CtxFinancialPressure, DecayTauHours: 72},
		CtxShippingSprint:    {Name: CtxShippingSprint, DecayTauHours: 48},
		CtxRecoveryPeriod:    {Name: CtxRecoveryPeriod, DecayTauHours: 72},
		CtxContextExplosion:  {Name: CtxContextExplosion, DecayTauHours: 48},
		CtxStall:             {Name: CtxStall, DecayTauHours: 24},
		CtxCelebration:       {Name: CtxCelebration, DecayTauHours: 24},
		CtxLifeEvent:         {Name: CtxLifeEvent, DecayTauHours: 168}, // 7 days
	}

	p.SelfPerceptionDeltas = make(map[string]float64)
	p.Calibration = defaultCalibration()
	p.PairingScore.Components = make(map[string]float64)

	p.Accuracy.ByConstruct = make(map[string]ConstructAccuracy)
	p.Accuracy.CalibrationLift = make(map[string]float64)

	p.Meta.CreatedAt = time.Now()
	p.Meta.EngineVersion = "2.0"

	return p
}

// ═══════════════════════════════════════════════════════════════════════════════
// PAIRING ENGINE — The central coordinator
//
// PairingEngine owns the profile, processes signals, and maintains all
// in-memory state. It's created once at server startup and lives for the
// entire process lifetime.
//
// Threading model:
//   - signalChan is written by any goroutine (HTTP handlers, pollers, hooks)
//   - processLoop goroutine reads signalChan serially (order matters for EMA)
//   - periodicTasks goroutine runs decay + score recompute every 5 minutes
//   - mu protects profile reads from concurrent API requests
//
// Behavioral accumulators are rolling windows that track recent activity
// patterns without storing individual events:
//   - recentShips: timestamps of last 100 ship events (for cadence analysis)
//   - recentProjects: project→last_seen map (for context switch detection)
//   - approvalLatencies: last 100 approval durations (for engagement tracking)
//   - dailyEventCounts: date→count map (for behavioral pattern scoring)
// ═══════════════════════════════════════════════════════════════════════════════

// PairingEngine is the central pairing system coordinator. Create with
// NewPairingEngine(path) and start processing with pe.Start().
type PairingEngine struct {
	mu         sync.RWMutex
	profile    *FounderProfileV2
	profilePath string
	signalChan chan Signal
	evidence   []EvidenceEntry
	predictions []PredictionEntry
	driftHistory []DriftEvent
	nlp        *NLPExtractor
	dirty      bool
	lastSave   time.Time
	running    bool

	// Behavioral accumulators (rolling windows)
	recentShips     []time.Time
	recentProjects  map[string]time.Time // project → last seen
	approvalLatencies []float64
	dailyEventCounts map[string]int // date string → count
}

func NewPairingEngine(profilePath string) *PairingEngine {
	pe := &PairingEngine{
		profilePath:     profilePath,
		signalChan:      make(chan Signal, 1000),
		evidence:        make([]EvidenceEntry, 0, 10000),
		predictions:     make([]PredictionEntry, 0, 1000),
		driftHistory:    make([]DriftEvent, 0, 100),
		nlp:             NewNLPExtractor(),
		recentShips:     make([]time.Time, 0, 100),
		recentProjects:  make(map[string]time.Time),
		approvalLatencies: make([]float64, 0, 100),
		dailyEventCounts: make(map[string]int),
	}

	// Load or create profile
	pe.profile = pe.loadOrCreate()
	return pe
}

func (pe *PairingEngine) loadOrCreate() *FounderProfileV2 {
	data, err := os.ReadFile(pe.profilePath)
	if err == nil {
		var p FounderProfileV2
		if json.Unmarshal(data, &p) == nil && p.Version == 2 {
			log.Printf("[pairing] Loaded profile v2: score=%.0f level=%s signals=%d",
				p.PairingScore.Composite, p.PairingScore.Level, p.Meta.SignalsProcessed)
			return &p
		}
	}
	log.Printf("[pairing] Creating new profile v2 at %s", pe.profilePath)
	p := NewFounderProfile()
	pe.dirty = true
	return p
}

func (pe *PairingEngine) Save() error {
	pe.mu.RLock()
	data, err := json.MarshalIndent(pe.profile, "", "  ")
	pe.mu.RUnlock()
	if err != nil {
		return err
	}
	pe.lastSave = time.Now()
	pe.dirty = false
	return os.WriteFile(pe.profilePath, data, 0644)
}

// syncToMemory pushes the current profile summary to Mem0 and Wirebot gateway
// so the AI has persistent memory of the founder's assessed traits.
func (pe *PairingEngine) syncToMemory() {
	log.Printf("[pairing] syncToMemory starting")
	summary := pe.GetChatContextSummary()

	// Push to Mem0 (if available) — endpoint is /v1/store
	// Escape for JSON: replace newlines and quotes
	safeSummary := strings.ReplaceAll(summary, "\\", "\\\\")
	safeSummary = strings.ReplaceAll(safeSummary, "\"", "'")
	safeSummary = strings.ReplaceAll(safeSummary, "\n", " | ")
	safeSummary = strings.ReplaceAll(safeSummary, "\r", "")
	mem0Payload := fmt.Sprintf(`{"messages":[{"role":"user","content":"Founder profile update: %s"}],"namespace":"wirebot_verious","category":"founder_profile"}`,
		safeSummary)
	req, err := http.NewRequest("POST", "http://localhost:8200/v1/store", strings.NewReader(mem0Payload))
	if err != nil {
		log.Printf("[pairing] Mem0 request build error: %v", err)
	} else {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Length", fmt.Sprintf("%d", len(mem0Payload)))
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[pairing] Mem0 sync error: %v", err)
		} else {
			body := make([]byte, 200)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			log.Printf("[pairing] Mem0 sync: status=%d body=%s", resp.StatusCode, string(body[:n]))
		}
	}

	// Push to Letta business_stage block (if available) — keep structured state current
	lettaAgentID := "agent-82610d14-ec65-4d10-9ec2-8c479848cea9"
	// Get current blocks to find business_stage block ID
	getReq, err := http.NewRequest("GET",
		fmt.Sprintf("http://localhost:8283/v1/agents/%s", lettaAgentID), nil)
	if err == nil {
		getReq.Header.Set("Authorization", "Bearer letta")
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(getReq)
		if err == nil {
			var agentData struct {
				Memory struct {
					Blocks []struct {
						ID    string `json:"id"`
						Label string `json:"label"`
					} `json:"blocks"`
				} `json:"memory"`
			}
			if json.NewDecoder(resp.Body).Decode(&agentData) == nil {
				for _, block := range agentData.Memory.Blocks {
					if block.Label == "business_stage" {
						// Update the block with current profile data
						stageUpdate := fmt.Sprintf("Operating Mode: Red-to-Black\nSeason 1 started: 2026-02-01\nCurrent Score: %.0f/100\nPairing Level: %s (%.0f%% accuracy)\nProfile: %s",
							pe.profile.PairingScore.Composite,
							pe.profile.PairingScore.Level,
							pe.computeAccuracy()*100,
							safeSummary)
						updatePayload, _ := json.Marshal(map[string]string{"value": stageUpdate})
						patchReq, _ := http.NewRequest("PATCH",
							fmt.Sprintf("http://localhost:8283/v1/blocks/%s", block.ID),
							strings.NewReader(string(updatePayload)))
						if patchReq != nil {
							patchReq.Header.Set("Authorization", "Bearer letta")
							patchReq.Header.Set("Content-Type", "application/json")
							patchResp, err := client.Do(patchReq)
							if err == nil {
								patchResp.Body.Close()
								log.Printf("[pairing] Letta business_stage block updated (score=%.0f)", pe.profile.PairingScore.Composite)
							}
						}
						break
					}
				}
			}
			resp.Body.Close()
		}
	}

	// Push to Wirebot gateway via wirebot_remember tool (if available)
	gatewayToken := os.Getenv("SCOREBOARD_TOKEN")
	if gatewayToken == "" {
		gatewayToken = "65b918ba-baf5-4996-8b53-6fb0f662a0c3"
	}
	rememberPayload := fmt.Sprintf(`{"tool":"wirebot_remember","args":{"fact":"PROFILE UPDATE: %s"}}`,
		safeSummary)
	req2, err := http.NewRequest("POST", "http://127.0.0.1:18789/tools/invoke", strings.NewReader(rememberPayload))
	if err != nil {
		log.Printf("[pairing] Gateway request build error: %v", err)
	} else {
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Authorization", "Bearer "+gatewayToken)
		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req2)
		if err != nil {
			log.Printf("[pairing] Gateway sync error: %v", err)
		} else {
			body := make([]byte, 200)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()
			log.Printf("[pairing] Gateway sync: status=%d body=%s", resp.StatusCode, string(body[:n]))
		}
	}
}

// Start begins the background signal processing goroutine
func (pe *PairingEngine) Start() {
	pe.running = true
	go pe.processLoop()
	go pe.periodicTasks()
	log.Printf("[pairing] Engine started — processing signals")
}

func (pe *PairingEngine) processLoop() {
	saveCounter := 0
	for sig := range pe.signalChan {
		pe.processSignal(sig)
		saveCounter++
		// Persist every 10 signals or if 60s since last save
		if saveCounter >= 10 || time.Since(pe.lastSave) > 60*time.Second {
			if pe.dirty {
				if err := pe.Save(); err != nil {
					log.Printf("[pairing] Save error: %v", err)
				}
			}
			saveCounter = 0
		}
	}
}

func (pe *PairingEngine) periodicTasks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if !pe.running {
			return
		}
		pe.mu.Lock()
		// Decay context windows
		now := time.Now()
		for _, cw := range pe.profile.ContextWindows {
			cw.Decay(now)
		}
		pe.mu.Unlock()

		// Recompute pairing score
		pe.recomputePairingScore()

		// Save if dirty
		if pe.dirty {
			pe.Save()
		}
	}
}

// Ingest is the public entry point — non-blocking
func (pe *PairingEngine) Ingest(sig Signal) {
	select {
	case pe.signalChan <- sig:
	default:
		log.Printf("[pairing] Signal channel full, dropping: %s/%s", sig.Type, sig.Source)
	}
}

// processSignal is the core pipeline — runs serially in the processing goroutine
func (pe *PairingEngine) processSignal(sig Signal) {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	startTime := time.Now()

	// 1. Extract features based on signal type
	sig.Features = pe.extractFeatures(sig)

	// 2. Build evidence entry
	ev := EvidenceEntry{
		ID:             int64(len(pe.evidence) + 1),
		Timestamp:      sig.Timestamp,
		SignalType:     sig.Type,
		Source:         sig.Source,
		FeaturesExtracted: sig.Features,
		ProfileImpact:  make(map[string]float64),
	}

	// 3. Update constructs based on signal type
	switch sig.Type {
	case SignalMessage:
		pe.processMessage(sig, &ev)
	case SignalEvent:
		pe.processEvent(sig, &ev)
	case SignalAssessment:
		pe.processAssessment(sig, &ev)
	case SignalApproval:
		pe.processApproval(sig, &ev)
	case SignalDocument:
		pe.processDocument(sig, &ev)
	case SignalAccount:
		pe.processAccountData(sig, &ev)
	}

	// 4. Detect drift across all constructs
	pe.detectDrift(&ev)

	// 5. Evaluate context windows
	pe.evaluateContextWindows(sig, &ev)

	// 6. Recompute complement if anything changed
	pe.recomputeComplement()

	// 7. Update calibration parameters
	pe.updateCalibration()

	// 8. Compute evidence summary
	ev.Summary = pe.summarizeEvidence(sig, &ev)
	pe.evidence = append(pe.evidence, ev)
	// Keep last 10000
	if len(pe.evidence) > 10000 {
		pe.evidence = pe.evidence[len(pe.evidence)-10000:]
	}

	// 9. Update meta
	pe.profile.Meta.SignalsProcessed++
	pe.dirty = true

	// 10. Push profile facts to Mem0 (async, non-blocking)
	//     Only on assessment signals (not every chat message)
	if sig.Type == SignalAssessment {
		go pe.syncToMemory()
	}

	elapsed := time.Since(startTime)
	if elapsed > 50*time.Millisecond {
		log.Printf("[pairing] Slow signal processing: %s/%s took %v", sig.Type, sig.Source, elapsed)
	}
}

// ─── Signal Processors ──────────────────────────────────────────────────────

func (pe *PairingEngine) processMessage(sig Signal, ev *EvidenceEntry) {
	if sig.Content == "" {
		return
	}
	pe.profile.Meta.TotalMessagesAnalyzed++
	now := time.Now()
	pe.profile.Meta.LastInferenceUpdate = &now

	features := sig.Features

	// Update DISC from NLP
	if d, ok := features["disc_D"]; ok {
		pe.profile.CommunicationDNA.UpdateEMA("D", d*10)
	}
	if i, ok := features["disc_I"]; ok {
		pe.profile.CommunicationDNA.UpdateEMA("I", i*10)
	}
	if s, ok := features["disc_S"]; ok {
		pe.profile.CommunicationDNA.UpdateEMA("S", s*10)
	}
	if c, ok := features["disc_C"]; ok {
		pe.profile.CommunicationDNA.UpdateEMA("C", c*10)
	}

	// Update observed communication style
	oc := &pe.profile.ObservedComm
	oc.MessagesAnalyzed++
	lambda := 0.10 // smoothing for observed style
	if v, ok := features["directness"]; ok {
		oc.Directness = oc.Directness*(1-lambda) + v*lambda
	}
	if v, ok := features["formality"]; ok {
		oc.Formality = oc.Formality*(1-lambda) + v*lambda
	}
	if v, ok := features["detail_preference"]; ok {
		oc.DetailPreference = oc.DetailPreference*(1-lambda) + v*lambda
	}
	if v, ok := features["emotion_expression"]; ok {
		oc.EmotionExpr = oc.EmotionExpr*(1-lambda) + v*lambda
	}
	if v, ok := features["pace_preference"]; ok {
		oc.PacePreference = oc.PacePreference*(1-lambda) + v*lambda
	}
	if v, ok := features["decision_style"]; ok {
		oc.DecisionStyle = oc.DecisionStyle*(1-lambda) + v*lambda
	}
	oc.Confidence = math.Min(1.0, float64(oc.MessagesAnalyzed)/200.0)

	// Update cognitive style from message features
	if v, ok := features["holistic_vs_sequential"]; ok {
		pe.profile.CognitiveStyle.UpdateEMA("holistic", v*10)
		pe.profile.CognitiveStyle.UpdateEMA("sequential", (1-v)*10)
	}
	if v, ok := features["abstract_vs_concrete"]; ok {
		pe.profile.CognitiveStyle.UpdateEMA("abstract", v*10)
		pe.profile.CognitiveStyle.UpdateEMA("concrete", (1-v)*10)
	}

	// Check for financial pressure keywords
	if v, ok := features["financial_pressure"]; ok && v > 0.3 {
		pe.profile.ContextWindows[CtxFinancialPressure].AddSignal(v)
	}

	// Check for life event signals
	if v, ok := features["life_event"]; ok && v > 0.3 {
		pe.profile.ContextWindows[CtxLifeEvent].AddSignal(v)
	}

	ev.ConstructsAffected = append(ev.ConstructsAffected, "communication_dna", "cognitive_style")
}

func (pe *PairingEngine) processEvent(sig Signal, ev *EvidenceEntry) {
	pe.profile.Meta.TotalEventsAnalyzed++
	now := time.Now()
	pe.profile.Meta.LastBehavioralBatch = &now

	md := sig.Metadata
	eventType, _ := md["event_type"].(string)
	lane, _ := md["lane"].(string)
	project, _ := md["project"].(string)

	// Track ships
	isShip := eventType == "TASK_COMPLETED" || eventType == "PRODUCT_RELEASE" ||
		eventType == "FEATURE_SHIPPED" || eventType == "CODE_PUBLISHED" ||
		eventType == "EXTENSION_PUBLISHED" || eventType == "DOCS_PUBLISHED"
	if isShip {
		pe.recentShips = append(pe.recentShips, sig.Timestamp)
		// Keep last 100
		if len(pe.recentShips) > 100 {
			pe.recentShips = pe.recentShips[1:]
		}
	}

	// Track projects
	if project != "" {
		pe.recentProjects[project] = sig.Timestamp
	}

	// Daily event count
	dateKey := sig.Timestamp.Format("2006-01-02")
	pe.dailyEventCounts[dateKey]++

	// Temporal pattern: hour of activity
	hour := float64(sig.Timestamp.Hour())
	pe.profile.TemporalPatterns.UpdateEMA("peak_hour", hour)

	// Shipping cadence → Quick Start signal
	if isShip && len(pe.recentShips) >= 3 {
		// Count ships in last 3 days
		cutoff := time.Now().Add(-72 * time.Hour)
		recentCount := 0
		for _, t := range pe.recentShips {
			if t.After(cutoff) {
				recentCount++
			}
		}
		if recentCount >= 5 {
			pe.profile.ContextWindows[CtxShippingSprint].AddSignal(0.5)
		}

		// Action style signal: shipping → QS + IM
		pe.profile.ActionStyle.UpdateEMA("QS", math.Min(10, float64(recentCount)*1.5))
		ev.ConstructsAffected = append(ev.ConstructsAffected, "action_style")
	}

	// Context switch detection
	today := sig.Timestamp.Format("2006-01-02")
	uniqueToday := 0
	for _, t := range pe.recentProjects {
		if t.Format("2006-01-02") == today {
			uniqueToday++
		}
	}
	if uniqueToday >= 5 {
		pe.profile.ContextWindows[CtxContextExplosion].AddSignal(0.4)
	}

	// Revenue signal → Business Reality
	if lane == "revenue" {
		if amt, ok := md["amount"].(float64); ok && amt > 0 {
			pe.profile.BusinessReality.UpdateEMA("revenue", math.Min(10, amt/100))
			ev.ConstructsAffected = append(ev.ConstructsAffected, "business_reality")
		}
	}

	// Stall detection
	if len(pe.recentShips) > 0 {
		lastShip := pe.recentShips[len(pe.recentShips)-1]
		hoursSince := time.Since(lastShip).Hours()
		if hoursSince > 24 {
			pe.profile.ContextWindows[CtxStall].AddSignal(math.Min(1.0, hoursSince/48))
		} else {
			// Ship after stall = recovery
			if pe.profile.ContextWindows[CtxStall].Activation > 0.3 {
				pe.profile.ContextWindows[CtxRecoveryPeriod].AddSignal(0.5)
				pe.profile.ContextWindows[CtxStall].Activation *= 0.3 // rapid decay on recovery
			}
		}
	}

	// Celebration detection
	if eventType == "PAYOUT_RECEIVED" || eventType == "PRODUCT_RELEASE" {
		pe.profile.ContextWindows[CtxCelebration].AddSignal(0.6)
	}

	ev.ConstructsAffected = append(ev.ConstructsAffected, "temporal_patterns")
}

func (pe *PairingEngine) processAssessment(sig Signal, ev *EvidenceEntry) {
	now := time.Now()
	pe.profile.Meta.LastAssessment = &now

	answers, ok := sig.Metadata["answers"].([]interface{})
	if !ok {
		log.Printf("[pairing] processAssessment: answers not []interface{}, type=%T", sig.Metadata["answers"])
		return
	}
	log.Printf("[pairing] processAssessment: processing %d answers", len(answers))

	for _, raw := range answers {
		a, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		inst, _ := a["instrument_id"].(string)
		qid, _ := a["question_id"].(string)
		val := a["value"]

		answer := AssessmentAnswer{
			InstrumentID: inst,
			QuestionID:   qid,
			Value:        val,
			AnsweredAt:   now,
		}
		pe.profile.Answers = append(pe.profile.Answers, answer)

		// Route to scoring — match by instrument_id OR by question_id prefix
		switch {
		case inst == "ASI-12" || strings.HasPrefix(qid, "ASI-"):
			pe.scoreASI(qid, val, ev)
		case inst == "CSI-8" || strings.HasPrefix(qid, "CSI-"):
			pe.scoreCSI(qid, val, ev)
		case inst == "ETM-6" || strings.HasPrefix(qid, "ETM-"):
			pe.scoreETM(val, ev)
		case inst == "RDS-6" || strings.HasPrefix(qid, "RDS-"):
			pe.scoreRDS(qid, val, ev)
		case inst == "COG-8" || strings.HasPrefix(qid, "COG-"):
			pe.scoreCOG(qid, val, ev)
		case inst == "BIZ-6" || strings.HasPrefix(qid, "BIZ-"):
			pe.scoreBIZ(qid, val, ev)
		case inst == "TIME-6" || strings.HasPrefix(qid, "TIME-"):
			pe.scoreTIME(qid, val, ev)
		default:
			log.Printf("[pairing] Unknown instrument: inst=%s qid=%s", inst, qid)
		}
	}
}

func (pe *PairingEngine) processApproval(sig Signal, ev *EvidenceEntry) {
	latencyS, _ := sig.Metadata["latency_seconds"].(float64)
	action, _ := sig.Metadata["action"].(string)

	if latencyS > 0 {
		pe.approvalLatencies = append(pe.approvalLatencies, latencyS)
		if len(pe.approvalLatencies) > 100 {
			pe.approvalLatencies = pe.approvalLatencies[1:]
		}
	}

	// Fast approval = high engagement
	if action == "approve" && latencyS < 300 { // < 5 min
		pe.profile.ActionStyle.UpdateEMA("QS", 8) // quick decision = QS signal
	}

	// Selective approval (approves some, rejects others) = Discernment
	if action == "reject" {
		pe.profile.EnergyTopology.UpdateEMA("D_disc", 8) // using discernment
	}

	ev.ConstructsAffected = append(ev.ConstructsAffected, "action_style", "energy_topology")
}

func (pe *PairingEngine) processDocument(sig Signal, ev *EvidenceEntry) {
	pe.profile.Meta.TotalDocsIngested++

	if sig.Content != "" {
		// Extract NLP features from document
		features := pe.nlp.ExtractFeatures(sig.Content)
		for k, v := range features {
			sig.Features[k] = v
		}

		// Document style → cognitive style
		if v, ok := features["vocabulary_richness"]; ok {
			pe.profile.CognitiveStyle.UpdateEMA("abstract", v*10)
		}
		if v, ok := features["list_usage"]; ok {
			pe.profile.CognitiveStyle.UpdateEMA("sequential", v*10)
		}
	}

	ev.ConstructsAffected = append(ev.ConstructsAffected, "cognitive_style")
}

func (pe *PairingEngine) processAccountData(sig Signal, ev *EvidenceEntry) {
	provider, _ := sig.Metadata["provider"].(string)

	// Track connected accounts
	found := false
	for _, a := range pe.profile.Meta.ConnectedAccounts {
		if a == provider {
			found = true
			break
		}
	}
	if !found {
		pe.profile.Meta.ConnectedAccounts = append(pe.profile.Meta.ConnectedAccounts, provider)
	}

	// Provider-specific signals
	switch provider {
	case "stripe":
		if rev, ok := sig.Metadata["monthly_revenue"].(float64); ok {
			pe.profile.BusinessReality.UpdateEMA("revenue", math.Min(10, rev/1000))
		}
	case "github":
		if commits, ok := sig.Metadata["weekly_commits"].(float64); ok {
			pe.profile.ActionStyle.UpdateEMA("IM", math.Min(10, commits/5))
		}
	}

	ev.ConstructsAffected = append(ev.ConstructsAffected, "business_reality")
}

// ─── Assessment Scorers ──────────────────────────────────────────────────────

// ASI-12: Action Style Inventory — forced-choice pairs
func (pe *PairingEngine) scoreASI(qid string, val interface{}, ev *EvidenceEntry) {
	choice, ok := val.(string)
	if !ok {
		log.Printf("[pairing] scoreASI: val not string, type=%T val=%v", val, val)
		return
	}
	scores := asiScoring[qid]
	if scores == nil {
		log.Printf("[pairing] scoreASI: no scoring table for qid=%s", qid)
		return
	}
	if s, ok := scores[choice]; ok {
		pe.profile.ActionStyle.UpdateEMA(s.Dim, s.Value)
		ev.ProfileImpact["action_style."+s.Dim] = s.Value
		log.Printf("[pairing] scoreASI: %s choice=%s → %s=%.0f", qid, choice, s.Dim, s.Value)
	} else {
		log.Printf("[pairing] scoreASI: %s choice=%s not in scoring table (keys: %v)", qid, choice, scores)
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "action_style")
}

// CSI-8: Communication Style Inventory — scenario picks
func (pe *PairingEngine) scoreCSI(qid string, val interface{}, ev *EvidenceEntry) {
	choice, ok := val.(string)
	if !ok {
		return
	}
	scores := csiScoring[qid]
	if s, ok := scores[choice]; ok {
		pe.profile.CommunicationDNA.UpdateEMA(s.Dim, s.Value)
		ev.ProfileImpact["communication_dna."+s.Dim] = s.Value
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "communication_dna")
}

// ETM-6: Energy Topology Map — drag-to-sort ranking
func (pe *PairingEngine) scoreETM(val interface{}, ev *EvidenceEntry) {
	ranking, ok := val.([]interface{})
	if !ok {
		return
	}
	// Position → score: 1st=10, 2nd=8, 3rd=6, 4th=4, 5th=2, 6th=0
	posScores := []float64{10, 8, 6, 4, 2, 0}
	dims := []string{"W", "N", "D_disc", "G", "E", "T"}
	for i, item := range ranking {
		if i >= 6 {
			break
		}
		dimName, ok := item.(string)
		if !ok {
			continue
		}
		// Find matching dimension
		for _, d := range dims {
			if d == dimName || strings.EqualFold(d, dimName) {
				pe.profile.EnergyTopology.UpdateEMA(d, posScores[i])
				ev.ProfileImpact["energy_topology."+d] = posScores[i]
				break
			}
		}
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "energy_topology")
}

// RDS-6: Risk Disposition Scale — sliders (0-100)
func (pe *PairingEngine) scoreRDS(qid string, val interface{}, ev *EvidenceEntry) {
	v, ok := toFloat64(val)
	if !ok {
		return
	}
	dim := rdsMapping[qid]
	if dim != "" {
		pe.profile.RiskDisposition.UpdateEMA(dim, v/10) // normalize 0-100 → 0-10
		ev.ProfileImpact["risk_disposition."+dim] = v / 10
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "risk_disposition")
}

// COG-8: Cognitive Style Inventory
func (pe *PairingEngine) scoreCOG(qid string, val interface{}, ev *EvidenceEntry) {
	choice, ok := val.(string)
	if !ok {
		return
	}
	scores := cogScoring[qid]
	if s, ok := scores[choice]; ok {
		pe.profile.CognitiveStyle.UpdateEMA(s.Dim, s.Value)
		ev.ProfileImpact["cognitive_style."+s.Dim] = s.Value
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "cognitive_style")
}

// BIZ-6: Business Reality — stored as metadata on the profile, influences calibration
func (pe *PairingEngine) scoreBIZ(qid string, val interface{}, ev *EvidenceEntry) {
	choice, ok := val.(string)
	if !ok {
		return
	}
	// Business reality answers feed into BusinessReality construct
	// Map coded answers to dimension scores
	bizScores := map[string]map[string]DimScore{
		"BIZ-01": {"focus_single": {Dim: "focus", Value: 10}, "focus_dual": {Dim: "focus", Value: 6}, "focus_multi": {Dim: "focus", Value: 3}},
		"BIZ-02": {"rev_pre": {Dim: "revenue_maturity", Value: 1}, "rev_early": {Dim: "revenue_maturity", Value: 4}, "rev_sustain": {Dim: "revenue_maturity", Value: 7}, "rev_growing": {Dim: "revenue_maturity", Value: 10}},
		"BIZ-03": {"team_solo": {Dim: "team_size", Value: 1}, "team_contractors": {Dim: "team_size", Value: 4}, "team_small": {Dim: "team_size", Value: 7}, "team_growing": {Dim: "team_size", Value: 10}},
		"BIZ-04": {"bottle_ship": {Dim: "bottleneck", Value: 3}, "bottle_dist": {Dim: "bottleneck", Value: 5}, "bottle_rev": {Dim: "bottleneck", Value: 7}, "bottle_ops": {Dim: "bottleneck", Value: 9}},
		"BIZ-05": {"age_new": {Dim: "venture_age", Value: 2}, "age_early": {Dim: "venture_age", Value: 4}, "age_mid": {Dim: "venture_age", Value: 7}, "age_mature": {Dim: "venture_age", Value: 10}},
		"BIZ-06": {"debt_none": {Dim: "debt_pressure", Value: 0}, "debt_some": {Dim: "debt_pressure", Value: 3}, "debt_heavy": {Dim: "debt_pressure", Value: 7}, "debt_critical": {Dim: "debt_pressure", Value: 10}},
	}
	if scores, ok := bizScores[qid]; ok {
		if s, ok := scores[choice]; ok {
			pe.profile.BusinessReality.UpdateEMA(s.Dim, s.Value)
			ev.ProfileImpact["business_reality."+s.Dim] = s.Value
		}
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "business_reality")
}

// TIME-6: Temporal Patterns — work rhythms, planning style, stall behavior
func (pe *PairingEngine) scoreTIME(qid string, val interface{}, ev *EvidenceEntry) {
	choice, ok := val.(string)
	if !ok {
		return
	}
	timeScores := map[string]map[string]DimScore{
		"TIME-01": {"peak_early": {Dim: "peak_hour", Value: 2}, "peak_mid_am": {Dim: "peak_hour", Value: 5}, "peak_afternoon": {Dim: "peak_hour", Value: 7}, "peak_evening": {Dim: "peak_hour", Value: 9}},
		"TIME-02": {"plan_rigid": {Dim: "planning_style", Value: 10}, "plan_flex": {Dim: "planning_style", Value: 7}, "plan_reactive": {Dim: "planning_style", Value: 4}, "plan_flow": {Dim: "planning_style", Value: 1}},
		"TIME-03": {"stall_push": {Dim: "stall_recovery", Value: 9}, "stall_switch": {Dim: "stall_recovery", Value: 7}, "stall_break": {Dim: "stall_recovery", Value: 5}, "stall_ask": {Dim: "stall_recovery", Value: 3}},
		"TIME-04": {"hours_part": {Dim: "work_intensity", Value: 3}, "hours_standard": {Dim: "work_intensity", Value: 5}, "hours_heavy": {Dim: "work_intensity", Value: 8}, "hours_max": {Dim: "work_intensity", Value: 10}},
		"TIME-05": {"switch_easy": {Dim: "context_switch_cost", Value: 1}, "switch_mild": {Dim: "context_switch_cost", Value: 4}, "switch_hard": {Dim: "context_switch_cost", Value: 7}, "switch_critical": {Dim: "context_switch_cost", Value: 10}},
		"TIME-06": {"horizon_short": {Dim: "planning_horizon", Value: 2}, "horizon_mid": {Dim: "planning_horizon", Value: 5}, "horizon_long": {Dim: "planning_horizon", Value: 8}, "horizon_visionary": {Dim: "planning_horizon", Value: 10}},
	}
	if scores, ok := timeScores[qid]; ok {
		if s, ok := scores[choice]; ok {
			pe.profile.TemporalPatterns.UpdateEMA(s.Dim, s.Value)
			ev.ProfileImpact["temporal_patterns."+s.Dim] = s.Value
		}
	}
	ev.ConstructsAffected = append(ev.ConstructsAffected, "temporal_patterns")
}

// ─── Drift Detection ─────────────────────────────────────────────────────────

func (pe *PairingEngine) detectDrift(ev *EvidenceEntry) {
	now := time.Now()
	pe.profile.Meta.LastDriftCheck = &now

	constructs := map[string]*DualTrackDimension{
		"action_style":      pe.profile.ActionStyle,
		"communication_dna": pe.profile.CommunicationDNA,
		"energy_topology":   pe.profile.EnergyTopology,
		"risk_disposition":  pe.profile.RiskDisposition,
		"temporal_patterns": pe.profile.TemporalPatterns,
		"cognitive_style":   pe.profile.CognitiveStyle,
	}

	for cName, dt := range constructs {
		for dim, drift := range dt.Drift {
			if drift >= 2.0 {
				// Significant drift — record event
				de := DriftEvent{
					ID:        int64(len(pe.driftHistory) + 1),
					Timestamp: now,
					Construct: cName,
					Dimension: dim,
					Magnitude: drift,
				}
				// Check which context window is active
				for wType, cw := range pe.profile.ContextWindows {
					if cw.Activation > 0.3 {
						de.Context = wType
						break
					}
				}
				pe.driftHistory = append(pe.driftHistory, de)
				pe.profile.Meta.TotalStateShifts++
				ev.ProfileImpact["drift."+cName+"."+dim] = drift
			}
		}
	}
}

// ─── Context Window Evaluation ───────────────────────────────────────────────

func (pe *PairingEngine) evaluateContextWindows(sig Signal, ev *EvidenceEntry) {
	now := time.Now()
	pe.profile.Meta.LastContextWindowEval = &now

	// Decay all windows
	for _, cw := range pe.profile.ContextWindows {
		cw.Decay(now)
	}

	// Record active windows
	for wType, cw := range pe.profile.ContextWindows {
		if cw.Activation > 0.3 {
			ev.ProfileImpact["context."+string(wType)] = cw.Activation
		}
	}
}

// ─── Complement Rebalancer ───────────────────────────────────────────────────

func (pe *PairingEngine) recomputeComplement() {
	pe.profile.Complement.Rebalance(
		pe.profile.ActionStyle.Effective,
		pe.profile.EnergyTopology.Effective,
	)
	// Also factor in risk, cognitive, business, temporal for secondary adjustments
	// If founder has low risk tolerance, Wirebot should be bolder (inverse)
	// If founder has high context_switch_cost, Wirebot should batch suggestions
	pe.profile.Complement.AdjustFromSecondary(
		pe.profile.RiskDisposition.Effective,
		pe.profile.CognitiveStyle.Effective,
		pe.profile.BusinessReality.Effective,
		pe.profile.TemporalPatterns.Effective,
	)
	now := time.Now()
	pe.profile.Meta.LastComplementRebal = &now
}

// ─── Calibration Updater ─────────────────────────────────────────────────────

func (pe *PairingEngine) updateCalibration() {
	p := pe.profile
	cal := &p.Calibration

	// Communication style → calibration
	oc := &p.ObservedComm
	if oc.MessagesAnalyzed > 10 {
		// High directness → shorter messages, lead with recommendation
		if oc.Directness > 0.65 {
			cal.Communication.MaxMessageWords = 200
			cal.Communication.LeadWith = "recommendation"
		} else if oc.Directness < 0.35 {
			cal.Communication.MaxMessageWords = 500
			cal.Communication.LeadWith = "context"
		}

		cal.Communication.ToneFormality = oc.Formality
		cal.Communication.EmojiMirrorRatio = oc.EmotionExpr
		cal.Communication.CelebrationIntensity = oc.EmotionExpr
	}

	// DISC primary → communication lead
	if p.CommunicationDNA.Effective["D"] != nil && p.CommunicationDNA.Effective["C"] != nil {
		dEff := *p.CommunicationDNA.Effective["D"]
		cEff := *p.CommunicationDNA.Effective["C"]
		if dEff > cEff && dEff > 6 {
			cal.Communication.LeadWith = "recommendation"
			cal.Communication.QuestionFrequency = "low"
		} else if cEff > dEff && cEff > 6 {
			cal.Communication.LeadWith = "data"
			cal.Communication.QuestionFrequency = "moderate"
		}
	}

	// Temporal → standup hour
	if p.TemporalPatterns.Effective["peak_hour"] != nil {
		peak := *p.TemporalPatterns.Effective["peak_hour"]
		if peak >= 20 || peak <= 4 { // night owl
			cal.Proactive.StandupHour = 11
		} else if peak >= 5 && peak <= 8 { // early bird
			cal.Proactive.StandupHour = 7
		} else {
			cal.Proactive.StandupHour = 9
		}
	}

	// Business Reality → calibration
	if p.BusinessReality.Effective["debt_pressure"] != nil {
		debt := *p.BusinessReality.Effective["debt_pressure"]
		if debt >= 7 { // heavy/critical debt
			cal.Recommendations.RiskFraming = "cautious"
			cal.Accountability.NudgeIntensity = math.Min(1.0, cal.Accountability.NudgeIntensity+0.1)
		}
	}
	if p.BusinessReality.Effective["team_size"] != nil {
		team := *p.BusinessReality.Effective["team_size"]
		if team <= 2 { // solo
			cal.Recommendations.OptionsPresented = 2 // fewer options, solo operator is overwhelmed
		} else {
			cal.Recommendations.OptionsPresented = 3
		}
	}
	if p.BusinessReality.Effective["bottleneck"] != nil {
		bottle := *p.BusinessReality.Effective["bottleneck"]
		if bottle <= 4 { // shipping bottleneck
			cal.Proactive.PeakTaskType = "shipping"
		} else if bottle <= 6 { // distribution bottleneck
			cal.Proactive.PeakTaskType = "distribution"
		} else { // revenue/ops bottleneck
			cal.Proactive.PeakTaskType = "revenue"
		}
	}

	// Temporal Patterns → calibration
	if p.TemporalPatterns.Effective["planning_style"] != nil {
		planStyle := *p.TemporalPatterns.Effective["planning_style"]
		if planStyle >= 8 { // rigid planner
			cal.Recommendations.PlanningDepth = "detailed"
		} else if planStyle <= 3 { // flow state
			cal.Recommendations.PlanningDepth = "minimal"
		} else {
			cal.Recommendations.PlanningDepth = "moderate"
		}
	}
	if p.TemporalPatterns.Effective["work_intensity"] != nil {
		intensity := *p.TemporalPatterns.Effective["work_intensity"]
		if intensity >= 8 { // heavy worker: less nudging, they're already on it
			cal.Accountability.NudgeFrequencyHours = math.Max(cal.Accountability.NudgeFrequencyHours, 10)
		} else if intensity <= 3 { // part-time: more nudging
			cal.Accountability.NudgeFrequencyHours = math.Min(cal.Accountability.NudgeFrequencyHours, 6)
		}
	}
	if p.TemporalPatterns.Effective["context_switch_cost"] != nil {
		switchCost := *p.TemporalPatterns.Effective["context_switch_cost"]
		if switchCost >= 7 { // expensive context switches
			cal.Accountability.StallInterventionH = math.Max(cal.Accountability.StallInterventionH, 6)
		}
	}
	if p.TemporalPatterns.Effective["stall_recovery"] != nil {
		recovery := *p.TemporalPatterns.Effective["stall_recovery"]
		if recovery >= 8 { // pushes through stalls
			cal.Accountability.NudgeIntensity = math.Max(0, cal.Accountability.NudgeIntensity-0.1)
		} else if recovery <= 3 { // asks for help
			cal.Communication.QuestionFrequency = "high" // ask probing questions to unstick them
		}
	}

	// Context window overrides (these override everything above when active)
	if cw := p.ContextWindows[CtxFinancialPressure]; cw.Activation > 0.5 {
		cal.Recommendations.RiskFraming = "cautious"
		cal.Recommendations.DataDensity = math.Min(1.0, cal.Recommendations.DataDensity+0.2)
	}
	if cw := p.ContextWindows[CtxShippingSprint]; cw.Activation > 0.5 {
		cal.Accountability.NudgeFrequencyHours = math.Max(12, cal.Accountability.NudgeFrequencyHours)
		cal.Accountability.NudgeIntensity = math.Max(0, cal.Accountability.NudgeIntensity-0.2)
	}
	if cw := p.ContextWindows[CtxRecoveryPeriod]; cw.Activation > 0.5 {
		cal.Accountability.NudgeIntensity = math.Max(0, cal.Accountability.NudgeIntensity-0.3)
		cal.Accountability.StallInterventionH = 24
	}
	if cw := p.ContextWindows[CtxStall]; cw.Activation > 0.5 {
		cal.Accountability.NudgeIntensity = math.Min(1.0, cal.Accountability.NudgeIntensity+0.3)
		cal.Accountability.StallInterventionH = 4
	}
}

// ─── Pairing Score Computation ───────────────────────────────────────────────

func (pe *PairingEngine) recomputePairingScore() {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	p := pe.profile
	sc := &p.PairingScore

	// S1: Action Style (15%) — from assessment answers
	s1 := 0.0
	if countDims(p.ActionStyle) >= 3 {
		s1 = 1.0
	} else if countDims(p.ActionStyle) >= 1 {
		s1 = 0.5
	}
	sc.Components["S1_action_style"] = s1

	// S2: Communication DNA (10%) — from assessment
	s2 := 0.0
	if countDims(p.CommunicationDNA) >= 3 {
		s2 = 1.0
	} else if countDims(p.CommunicationDNA) >= 1 {
		s2 = 0.5
	}
	sc.Components["S2_communication"] = s2

	// S3: Working Genius / Energy (10%) — from assessment
	s3 := 0.0
	if countDims(p.EnergyTopology) >= 4 {
		s3 = 1.0
	} else if countDims(p.EnergyTopology) >= 1 {
		s3 = 0.5
	}
	sc.Components["S3_energy"] = s3

	// S4: Risk Profile (10%) — from assessment
	s4 := 0.0
	if countDims(p.RiskDisposition) >= 4 {
		s4 = 1.0
	} else if countDims(p.RiskDisposition) >= 1 {
		s4 = 0.5
	}
	sc.Components["S4_risk"] = s4

	// S5: Business Reality declared (15%)
	s5 := 0.0
	if countDims(p.BusinessReality) >= 2 {
		s5 = 1.0
	} else if countDims(p.BusinessReality) >= 1 {
		s5 = 0.5
	}
	sc.Components["S5_business_declared"] = s5

	// S6: Business Reality verified (10%) — from connected accounts
	s6 := math.Min(1.0, float64(len(p.Meta.ConnectedAccounts))/3.0)
	sc.Components["S6_business_verified"] = s6

	// S7: Communication style inferred (15%) — needs 50+ messages
	s7 := math.Min(1.0, float64(p.ObservedComm.MessagesAnalyzed)/50.0)
	sc.Components["S7_comm_inferred"] = s7

	// S8: Behavioral patterns (10%) — needs 7+ days
	days := time.Since(p.Meta.CreatedAt).Hours() / 24
	eventDays := float64(len(pe.dailyEventCounts))
	s8 := math.Min(1.0, eventDays/7.0)
	_ = days
	sc.Components["S8_behavioral"] = s8

	// S9: Continuous inference (5%) — always accumulating
	s9 := math.Min(1.0, float64(p.Meta.SignalsProcessed)/500.0)
	sc.Components["S9_continuous"] = s9

	// Weighted composite
	weights := map[string]float64{
		"S1_action_style":     0.15,
		"S2_communication":    0.10,
		"S3_energy":           0.10,
		"S4_risk":             0.10,
		"S5_business_declared": 0.15,
		"S6_business_verified": 0.10,
		"S7_comm_inferred":    0.15,
		"S8_behavioral":       0.10,
		"S9_continuous":        0.05,
	}

	composite := 0.0
	for k, w := range weights {
		composite += sc.Components[k] * w
	}
	sc.Composite = composite * 100

	// Hard caps
	if p.ObservedComm.MessagesAnalyzed < 50 && sc.Composite > 60 {
		sc.Composite = 60 // Can't exceed 60 without comm scanning
	}
	if eventDays < 30 && sc.Composite > 80 {
		sc.Composite = 80 // Can't exceed 80 without 30 days behavioral
	}

	// Level
	switch {
	case sc.Composite >= 81:
		sc.Level = "Bonded"
	case sc.Composite >= 61:
		sc.Level = "Trusted"
	case sc.Composite >= 36:
		sc.Level = "Partner"
	case sc.Composite >= 16:
		sc.Level = "Acquaintance"
	default:
		sc.Level = "Stranger"
	}

	pe.dirty = true
}

// ─── Feature Extraction Router ───────────────────────────────────────────────

func (pe *PairingEngine) extractFeatures(sig Signal) map[string]float64 {
	if sig.Features == nil {
		sig.Features = make(map[string]float64)
	}

	switch sig.Type {
	case SignalMessage:
		if sig.Content != "" {
			nlpFeatures := pe.nlp.ExtractFeatures(sig.Content)
			for k, v := range nlpFeatures {
				sig.Features[k] = v
			}
			// DISC inference
			disc := pe.nlp.InferDISC(sig.Content)
			for k, v := range disc {
				sig.Features["disc_"+k] = v
			}
		}
	case SignalEvent:
		// Event features come from metadata already
	}

	return sig.Features
}

// ─── Evidence Summary ────────────────────────────────────────────────────────

func (pe *PairingEngine) summarizeEvidence(sig Signal, ev *EvidenceEntry) string {
	switch sig.Type {
	case SignalMessage:
		words := len(strings.Fields(sig.Content))
		return fmt.Sprintf("Chat message (%d words)", words)
	case SignalEvent:
		et, _ := sig.Metadata["event_type"].(string)
		return fmt.Sprintf("Scoreboard event: %s", et)
	case SignalAssessment:
		return "Assessment answers submitted"
	case SignalApproval:
		action, _ := sig.Metadata["action"].(string)
		return fmt.Sprintf("Event %s", action)
	case SignalDocument:
		return fmt.Sprintf("Document ingested (%d chars)", len(sig.Content))
	case SignalAccount:
		provider, _ := sig.Metadata["provider"].(string)
		return fmt.Sprintf("Account data from %s", provider)
	}
	return "Signal processed"
}

// ─── Effective Profile (for chat context injection) ──────────────────────────

type EffectiveProfile struct {
	ActionStyle    map[string]float64  `json:"action_style"`
	DISC           map[string]float64  `json:"disc"`
	Energy         map[string]float64  `json:"energy"`
	Risk           map[string]float64  `json:"risk"`
	Cognitive      map[string]float64  `json:"cognitive"`
	Business       map[string]float64  `json:"business"`
	Temporal       map[string]float64  `json:"temporal"`
	Complement     ComplementVector    `json:"complement"`
	Calibration    CalibrationParams   `json:"calibration"`
	ActiveContexts []string            `json:"active_contexts"`
	PairingScore   float64             `json:"pairing_score"`
	Level          string              `json:"level"`
	Accuracy       float64             `json:"accuracy"`
}

func (pe *PairingEngine) GetEffectiveProfile() EffectiveProfile {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	p := pe.profile
	eff := EffectiveProfile{
		ActionStyle:  extractEffective(p.ActionStyle),
		DISC:         extractEffective(p.CommunicationDNA),
		Energy:       extractEffective(p.EnergyTopology),
		Risk:         extractEffective(p.RiskDisposition),
		Cognitive:    extractEffective(p.CognitiveStyle),
		Business:     extractEffective(p.BusinessReality),
		Temporal:     extractEffective(p.TemporalPatterns),
		Complement:   p.Complement,
		Calibration:  p.Calibration,
		PairingScore: p.PairingScore.Composite,
		Level:        p.PairingScore.Level,
		Accuracy:     pe.computeAccuracy(),
	}

	for wType, cw := range p.ContextWindows {
		if cw.Activation > 0.3 {
			eff.ActiveContexts = append(eff.ActiveContexts, string(wType))
		}
	}

	return eff
}

// GetChatContextSummary returns a concise 3-5 line summary for injection into chat system message
func (pe *PairingEngine) GetChatContextSummary() string {
	eff := pe.GetEffectiveProfile()

	if eff.PairingScore < 5 {
		return "Founder profile: Not yet calibrated. Run pairing assessment first."
	}

	var lines []string

	// Line 1: DISC + Action Style
	discPrimary := "unknown"
	discMax := 0.0
	for d, v := range eff.DISC {
		if v > discMax {
			discMax = v
			discPrimary = d
		}
	}
	discNames := map[string]string{"D": "Driver", "I": "Influencer", "S": "Steady", "C": "Analytical"}
	if name, ok := discNames[discPrimary]; ok {
		discPrimary = name
	}

	actionDesc := ""
	if qs, ok := eff.ActionStyle["QS"]; ok && qs > 6 {
		actionDesc = "high Quick Start"
	} else if ff, ok := eff.ActionStyle["FF"]; ok && ff > 6 {
		actionDesc = "high Fact Finder"
	}
	lines = append(lines, fmt.Sprintf("Founder: %s-primary (%.0f%%), %s", discPrimary, discMax*10, actionDesc))

	// Line 2: Communication calibration
	cal := eff.Calibration.Communication
	lines = append(lines, fmt.Sprintf("Communication: %s first, %d-word max, formality=%.0f%%",
		cal.LeadWith, cal.MaxMessageWords, cal.ToneFormality*100))

	// Line 3: Business reality
	bizParts := []string{}
	if v, ok := eff.Business["debt_pressure"]; ok && v > 0 {
		if v >= 7 {
			bizParts = append(bizParts, "heavy debt pressure")
		} else if v >= 4 {
			bizParts = append(bizParts, "some debt")
		}
	}
	if v, ok := eff.Business["team_size"]; ok && v > 0 {
		if v <= 2 {
			bizParts = append(bizParts, "solo operator")
		} else {
			bizParts = append(bizParts, "has team")
		}
	}
	if v, ok := eff.Business["bottleneck"]; ok && v > 0 {
		if v <= 4 {
			bizParts = append(bizParts, "bottleneck=shipping")
		} else if v <= 6 {
			bizParts = append(bizParts, "bottleneck=distribution")
		} else {
			bizParts = append(bizParts, "bottleneck=revenue")
		}
	}
	if len(bizParts) > 0 {
		lines = append(lines, fmt.Sprintf("Business: %s", strings.Join(bizParts, ", ")))
	}

	// Line 4: Complement with behavioral instructions
	top := pe.topComplements(3)
	if len(top) > 0 {
		// Map complement areas to concrete Wirebot behaviors
		complementActions := map[string]string{
			"Tenacity":       "provide persistent follow-up, hold to commitments, don't let things slide",
			"Enablement":     "proactively offer help, remove blockers, connect dots across projects",
			"Galvanizing":    "inject energy, celebrate wins, rally momentum when stalled",
			"Follow Through": "track details, ensure nothing falls through cracks, remind about loose ends",
			"Quick Start":    "suggest bold moves, prototype ideas, push past analysis paralysis",
			"Fact Finder":    "surface research, data, evidence before decisions",
			"Implementor":    "provide concrete steps, blueprints, hands-on action items",
			"Wonder":         "ask big-picture questions, explore new possibilities, brainstorm",
			"Invention":      "suggest novel solutions, creative approaches, unconventional paths",
			"Discernment":    "evaluate tradeoffs, sense what feels right, trust pattern recognition",
		}
		var topActions []string
		for _, t := range top {
			name := strings.Split(t, " (")[0] // strip percentage
			if action, ok := complementActions[name]; ok {
				topActions = append(topActions, name+": "+action)
			}
		}
		if len(topActions) > 0 {
			lines = append(lines, fmt.Sprintf("Wirebot complement priorities: %s", strings.Join(topActions, "; ")))
		} else {
			lines = append(lines, fmt.Sprintf("Wirebot complement focus: %s", strings.Join(top, ", ")))
		}
	}

	// Line 5: What calibration produces
	fullCal := eff.Calibration
	lines = append(lines, fmt.Sprintf("Calibration: %s first, %d-word max, nudge every %.0fh, %s planning",
		fullCal.Communication.LeadWith, fullCal.Communication.MaxMessageWords,
		fullCal.Accountability.NudgeFrequencyHours, fullCal.Recommendations.PlanningDepth))

	// Line 6: Active contexts
	if len(eff.ActiveContexts) > 0 {
		lines = append(lines, fmt.Sprintf("Active contexts: %s", strings.Join(eff.ActiveContexts, ", ")))
	}

	// Line 7: Pairing level
	lines = append(lines, fmt.Sprintf("Pairing: %.0f/100 (%s) | Accuracy: %.0f%%", eff.PairingScore, eff.Level, eff.Accuracy*100))

	return strings.Join(lines, "\n")
}

func (pe *PairingEngine) topComplements(n int) []string {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	c := pe.profile.Complement
	items := []struct {
		Name  string
		Value float64
	}{
		{"Fact Finder", c.FactFinder}, {"Follow Through", c.FollowThrough},
		{"Quick Start", c.QuickStart}, {"Implementor", c.Implementor},
		{"Wonder", c.Wonder}, {"Invention", c.Invention},
		{"Discernment", c.Discernment}, {"Galvanizing", c.Galvanizing},
		{"Enablement", c.Enablement}, {"Tenacity", c.Tenacity},
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Value > items[j].Value })
	var result []string
	for i := 0; i < n && i < len(items); i++ {
		if items[i].Value > 0.01 {
			result = append(result, fmt.Sprintf("%s (%.0f%%)", items[i].Name, items[i].Value*100))
		}
	}
	return result
}

// ─── Accuracy Computation ────────────────────────────────────────────────────

func (pe *PairingEngine) computeAccuracy() float64 {
	p := pe.profile
	a0 := 0.35 // initial accuracy from assessment alone
	tau := 30.0 // days

	days := time.Since(p.Meta.CreatedAt).Hours() / 24
	msgs := float64(p.ObservedComm.MessagesAnalyzed)
	events := float64(p.Meta.TotalEventsAnalyzed)
	docs := float64(p.Meta.TotalDocsIngested)
	accounts := float64(len(p.Meta.ConnectedAccounts))

	// Convergence equation: A(t) = 1 - (1-A₀) × e^(-t/τ) × Π(1-Δᵢ)
	deltaChat := 0.15 * (1 - math.Exp(-msgs/100))
	deltaEvents := 0.12 * (1 - math.Exp(-events/500))
	deltaDocs := 0.08 * math.Min(1.0, docs/5)
	deltaAccounts := 0.10 * math.Min(1.0, accounts/3)
	deltaRetest := 0.0 // no retests yet
	deltaDrift := 0.05 * math.Min(1.0, float64(p.Meta.TotalStateShifts)/5)

	product := (1 - deltaChat) * (1 - deltaEvents) * (1 - deltaDocs) *
		(1 - deltaAccounts) * (1 - deltaRetest) * (1 - deltaDrift)

	accuracy := 1 - (1-a0)*math.Exp(-days/tau)*product
	return math.Min(0.97, math.Max(a0, accuracy))
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func countDims(dt *DualTrackDimension) int {
	count := 0
	for _, v := range dt.Effective {
		if v != nil {
			count++
		}
	}
	return count
}

func extractEffective(dt *DualTrackDimension) map[string]float64 {
	result := make(map[string]float64)
	for k, v := range dt.Effective {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ─── Assessment Scoring Tables ───────────────────────────────────────────────

type DimScore struct {
	Dim   string
	Value float64
}

// ASI-12: 12 forced-choice pairs
// Each question has 2 choices mapping to dimensions
var asiScoring = map[string]map[string]DimScore{
	"ASI-01": {"A": {Dim: "QS", Value: 9}, "B": {Dim: "FF", Value: 8}},
	"ASI-02": {"A": {Dim: "QS", Value: 8}, "B": {Dim: "FT", Value: 8}},
	"ASI-03": {"A": {Dim: "IM", Value: 9}, "B": {Dim: "FF", Value: 7}},
	"ASI-04": {"A": {Dim: "FT", Value: 9}, "B": {Dim: "QS", Value: 7}},
	"ASI-05": {"A": {Dim: "FF", Value: 9}, "B": {Dim: "IM", Value: 8}},
	"ASI-06": {"A": {Dim: "QS", Value: 8}, "B": {Dim: "IM", Value: 9}},
	"ASI-07": {"A": {Dim: "FT", Value: 8}, "B": {Dim: "FF", Value: 9}},
	"ASI-08": {"A": {Dim: "IM", Value: 8}, "B": {Dim: "FT", Value: 7}},
	"ASI-09": {"A": {Dim: "QS", Value: 9}, "B": {Dim: "FT", Value: 9}},
	"ASI-10": {"A": {Dim: "FF", Value: 8}, "B": {Dim: "QS", Value: 8}},
	"ASI-11": {"A": {Dim: "IM", Value: 7}, "B": {Dim: "FT", Value: 8}},
	"ASI-12": {"A": {Dim: "FF", Value: 7}, "B": {Dim: "IM", Value: 8}},
}

// CSI-8: 8 scenario cards → D/I/S/C
var csiScoring = map[string]map[string]DimScore{
	"CSI-01": {"D": {Dim: "D", Value: 9}, "I": {Dim: "I", Value: 9}, "S": {Dim: "S", Value: 8}, "C": {Dim: "C", Value: 8}},
	"CSI-02": {"D": {Dim: "D", Value: 8}, "I": {Dim: "I", Value: 8}, "S": {Dim: "S", Value: 9}, "C": {Dim: "C", Value: 9}},
	"CSI-03": {"D": {Dim: "D", Value: 9}, "I": {Dim: "I", Value: 7}, "S": {Dim: "S", Value: 8}, "C": {Dim: "C", Value: 8}},
	"CSI-04": {"D": {Dim: "D", Value: 7}, "I": {Dim: "I", Value: 9}, "S": {Dim: "S", Value: 7}, "C": {Dim: "C", Value: 9}},
	"CSI-05": {"D": {Dim: "D", Value: 8}, "I": {Dim: "I", Value: 8}, "S": {Dim: "S", Value: 9}, "C": {Dim: "C", Value: 7}},
	"CSI-06": {"D": {Dim: "D", Value: 9}, "I": {Dim: "I", Value: 7}, "S": {Dim: "S", Value: 7}, "C": {Dim: "C", Value: 9}},
	"CSI-07": {"D": {Dim: "D", Value: 8}, "I": {Dim: "I", Value: 9}, "S": {Dim: "S", Value: 8}, "C": {Dim: "C", Value: 7}},
	"CSI-08": {"D": {Dim: "D", Value: 7}, "I": {Dim: "I", Value: 8}, "S": {Dim: "S", Value: 9}, "C": {Dim: "C", Value: 8}},
}

// RDS-6: slider question → dimension
var rdsMapping = map[string]string{
	"RDS-01": "tolerance",
	"RDS-02": "ambiguity",
	"RDS-03": "sunk_cost",
	"RDS-04": "loss_aversion",
	"RDS-05": "speed",
	"RDS-06": "bias_to_action",
}

// COG-8: forced-choice → cognitive dimension
var cogScoring = map[string]map[string]DimScore{
	"COG-01": {"A": {Dim: "holistic", Value: 9}, "B": {Dim: "sequential", Value: 9}},
	"COG-02": {"A": {Dim: "abstract", Value: 9}, "B": {Dim: "concrete", Value: 9}},
	"COG-03": {"A": {Dim: "holistic", Value: 8}, "B": {Dim: "sequential", Value: 8}},
	"COG-04": {"A": {Dim: "abstract", Value: 8}, "B": {Dim: "concrete", Value: 8}},
	"COG-05": {"A": {Dim: "holistic", Value: 7}, "B": {Dim: "concrete", Value: 7}},
	"COG-06": {"A": {Dim: "sequential", Value: 8}, "B": {Dim: "abstract", Value: 7}},
	"COG-07": {"A": {Dim: "holistic", Value: 8}, "B": {Dim: "sequential", Value: 7}},
	"COG-08": {"A": {Dim: "concrete", Value: 8}, "B": {Dim: "abstract", Value: 8}},
}
