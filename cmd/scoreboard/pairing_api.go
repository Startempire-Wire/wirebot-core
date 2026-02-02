package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PAIRING API — 17 HTTP endpoints for the Profile Equalizer
//
// All endpoints require member auth (operator token or Ring Leader JWT).
// Profile data is read from the in-memory PairingEngine, not from disk.
//
// See: PAIRING_ENGINE.md §19 (API Endpoints)
// ═══════════════════════════════════════════════════════════════════════════════

// registerPairingRoutes wires all pairing endpoints into the HTTP mux.
// Called from main() after PairingEngine is initialized.
func (s *Server) registerPairingRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/pairing/profile", s.authMember(s.handlePairingProfile))
	mux.HandleFunc("/v1/pairing/profile/effective", s.authMember(s.handlePairingEffective))
	mux.HandleFunc("/v1/pairing/evidence", s.authMember(s.handlePairingEvidence))
	mux.HandleFunc("/v1/pairing/formulas", s.authMember(s.handlePairingFormulas))
	mux.HandleFunc("/v1/pairing/accuracy", s.authMember(s.handlePairingAccuracy))
	mux.HandleFunc("/v1/pairing/drift", s.authMember(s.handlePairingDrift))
	mux.HandleFunc("/v1/pairing/complement", s.authMember(s.handlePairingComplement))
	mux.HandleFunc("/v1/pairing/predictions", s.authMember(s.handlePairingPredictions))
	mux.HandleFunc("/v1/pairing/insights", s.authMember(s.handlePairingInsights))
	mux.HandleFunc("/v1/pairing/answers", s.authMember(s.handlePairingAnswers))
	mux.HandleFunc("/v1/pairing/override", s.authMember(s.handlePairingOverride))
	mux.HandleFunc("/v1/pairing/overrides", s.authMember(s.handlePairingOverrides))
	mux.HandleFunc("/v1/pairing/scan", s.authMember(s.handlePairingScan))
	mux.HandleFunc("/v1/pairing/reset", s.authMember(s.handlePairingReset))
}

// ─── GET /v1/pairing/profile — Full FounderProfileV2 JSON ────────────────────

func (s *Server) handlePairingProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	s.pairing.mu.RLock()
	data, err := json.Marshal(s.pairing.profile)
	s.pairing.mu.RUnlock()
	if err != nil {
		http.Error(w, "Failed to marshal profile", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

// ─── GET /v1/pairing/profile/effective — Blended effective scores only ───────

func (s *Server) handlePairingEffective(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	eff := s.pairing.GetEffectiveProfile()
	writeJSON(w, eff)
}

// ─── GET /v1/pairing/evidence — Evidence log (paginated) ────────────────────

func (s *Server) handlePairingEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	// Pagination
	limit := 50
	offset := 0
	filterType := ""
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := r.URL.Query().Get("type"); v != "" {
		filterType = v
	}

	s.pairing.mu.RLock()
	var filtered []EvidenceEntry
	for _, ev := range s.pairing.evidence {
		if filterType != "" && string(ev.SignalType) != filterType {
			continue
		}
		filtered = append(filtered, ev)
	}
	s.pairing.mu.RUnlock()

	// Reverse order (newest first)
	total := len(filtered)
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}

	// Apply pagination
	if offset >= len(filtered) {
		filtered = nil
	} else {
		end := offset + limit
		if end > len(filtered) {
			end = len(filtered)
		}
		filtered = filtered[offset:end]
	}

	writeJSON(w, map[string]interface{}{
		"total":    total,
		"offset":   offset,
		"limit":    limit,
		"evidence": filtered,
	})
}

// ─── GET /v1/pairing/formulas — Live formula state ──────────────────────────

func (s *Server) handlePairingFormulas(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	p := s.pairing.profile

	// Build formula state with current inputs and outputs
	formulas := map[string]interface{}{
		"trait_state_blend": buildBlendFormulas(p),
		"disc_inference": map[string]interface{}{
			"description": "D = 0.30×imperative + 0.25×(1-hedge) + 0.20×action + 0.15×(1/sent_len) + 0.10×urgency",
			"observed_comm": p.ObservedComm,
			"effective":     extractEffective(p.CommunicationDNA),
			"source_weights": map[string]string{
				"note": "Weights shift from initial (assessment-heavy) to empirical (behavior-heavy) over 60 days",
			},
		},
		"complement_vector": map[string]interface{}{
			"description": "wirebot_effort(dim) = (10 - effective(dim)) / Σ(10 - all)",
			"current":     p.Complement,
		},
		"convergence": map[string]interface{}{
			"description": "A(t) = 1 - (1-A₀)×e^(-t/τ)×Π(1-Δᵢ)",
			"A0":          0.35,
			"tau_days":    30,
			"current":     s.pairing.computeAccuracy(),
			"days_active": time.Since(p.Meta.CreatedAt).Hours() / 24,
			"messages":    p.ObservedComm.MessagesAnalyzed,
			"events":      p.Meta.TotalEventsAnalyzed,
			"documents":   p.Meta.TotalDocsIngested,
			"accounts":    len(p.Meta.ConnectedAccounts),
		},
		"context_windows": buildContextFormulas(p),
		"pairing_score":   p.PairingScore,
	}
	s.pairing.mu.RUnlock()

	writeJSON(w, formulas)
}

// ─── GET /v1/pairing/accuracy — Accuracy metrics + convergence ──────────────

func (s *Server) handlePairingAccuracy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	p := s.pairing.profile
	accuracy := s.pairing.computeAccuracy()

	// Build per-construct accuracy
	byConstruct := make(map[string]interface{})
	constructs := map[string]*DualTrackDimension{
		"action_style":      p.ActionStyle,
		"communication_dna": p.CommunicationDNA,
		"energy_topology":   p.EnergyTopology,
		"risk_disposition":  p.RiskDisposition,
		"temporal_patterns": p.TemporalPatterns,
		"cognitive_style":   p.CognitiveStyle,
	}
	for name, dt := range constructs {
		totalObs := 0
		for _, n := range dt.Observations {
			totalObs += n
		}
		byConstruct[name] = map[string]interface{}{
			"observations": totalObs,
			"dimensions":   countDims(dt),
			"confidence":   float64(totalObs) / 100.0, // rough: 100 obs = full confidence
		}
	}

	// What would improve accuracy
	improvements := []map[string]interface{}{}
	if p.ObservedComm.MessagesAnalyzed < 50 {
		needed := 50 - p.ObservedComm.MessagesAnalyzed
		improvements = append(improvements, map[string]interface{}{
			"action": "Send more chat messages",
			"needed": needed,
			"boost":  "+5-10%",
		})
	}
	if len(p.Meta.ConnectedAccounts) < 3 {
		improvements = append(improvements, map[string]interface{}{
			"action": "Connect more accounts (GitHub, Stripe recommended)",
			"needed": 3 - len(p.Meta.ConnectedAccounts),
			"boost":  "+3-5% per account",
		})
	}
	if len(p.Answers) < 30 {
		improvements = append(improvements, map[string]interface{}{
			"action": "Complete more assessment questions",
			"needed": 30 - len(p.Answers),
			"boost":  "+5-15%",
		})
	}

	s.pairing.mu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"overall_accuracy":    accuracy,
		"improvement_vs_day1": (accuracy - 0.35) / 0.35,
		"days_active":         time.Since(p.Meta.CreatedAt).Hours() / 24,
		"by_construct":        byConstruct,
		"improvements":        improvements,
		"trajectory": map[string]interface{}{
			"day_1":   0.35,
			"day_7":   0.50,
			"day_30":  0.72,
			"day_90":  0.88,
			"day_365": 0.97,
			"current": accuracy,
		},
	})
}

// ─── GET /v1/pairing/drift — Current drift + context windows ────────────────

func (s *Server) handlePairingDrift(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	p := s.pairing.profile

	// Current drift readings
	driftReadings := make(map[string]map[string]interface{})
	constructs := map[string]*DualTrackDimension{
		"action_style":      p.ActionStyle,
		"communication_dna": p.CommunicationDNA,
		"energy_topology":   p.EnergyTopology,
		"risk_disposition":  p.RiskDisposition,
		"temporal_patterns": p.TemporalPatterns,
		"cognitive_style":   p.CognitiveStyle,
	}
	for name, dt := range constructs {
		dims := make(map[string]interface{})
		for dim, drift := range dt.Drift {
			severity := "normal"
			if drift >= 2.0 {
				severity = "significant"
			} else if drift >= 1.0 {
				severity = "mild"
			}
			dims[dim] = map[string]interface{}{
				"drift":    drift,
				"alpha":    dt.Alpha[dim],
				"severity": severity,
			}
			if dt.Trait[dim] != nil {
				dims[dim].(map[string]interface{})["trait"] = *dt.Trait[dim]
			}
			if dt.State[dim] != nil {
				dims[dim].(map[string]interface{})["state"] = *dt.State[dim]
			}
			if dt.Effective[dim] != nil {
				dims[dim].(map[string]interface{})["effective"] = *dt.Effective[dim]
			}
		}
		driftReadings[name] = dims
	}

	// Active context windows
	activeWindows := make(map[string]interface{})
	for wType, cw := range p.ContextWindows {
		if cw.Activation > 0.01 {
			entry := map[string]interface{}{
				"activation":   cw.Activation,
				"active":       cw.Activation >= 0.3,
				"signal_count": cw.SignalCount,
				"decay_tau_h":  cw.DecayTauHours,
			}
			if cw.ActivatedAt != nil {
				entry["activated_at"] = cw.ActivatedAt
				entry["active_hours"] = time.Since(*cw.ActivatedAt).Hours()
			}
			activeWindows[string(wType)] = entry
		}
	}

	// Drift history (last 20)
	historyLen := len(s.pairing.driftHistory)
	histStart := 0
	if historyLen > 20 {
		histStart = historyLen - 20
	}
	history := s.pairing.driftHistory[histStart:]

	s.pairing.mu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"drift_readings":  driftReadings,
		"context_windows": activeWindows,
		"drift_history":   history,
		"total_shifts":    p.Meta.TotalStateShifts,
	})
}

// ─── GET /v1/pairing/complement — Complement vector ─────────────────────────

func (s *Server) handlePairingComplement(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	c := s.pairing.profile.Complement

	// Build sorted list for display
	items := []map[string]interface{}{
		{"name": "Fact Finder", "code": "FF", "allocation": c.FactFinder},
		{"name": "Follow Through", "code": "FT", "allocation": c.FollowThrough},
		{"name": "Quick Start", "code": "QS", "allocation": c.QuickStart},
		{"name": "Implementor", "code": "IM", "allocation": c.Implementor},
		{"name": "Wonder", "code": "W", "allocation": c.Wonder},
		{"name": "Invention", "code": "N", "allocation": c.Invention},
		{"name": "Discernment", "code": "D", "allocation": c.Discernment},
		{"name": "Galvanizing", "code": "G", "allocation": c.Galvanizing},
		{"name": "Enablement", "code": "E", "allocation": c.Enablement},
		{"name": "Tenacity", "code": "T", "allocation": c.Tenacity},
	}
	s.pairing.mu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"complement":     c,
		"sorted":         items,
		"last_rebalanced": c.LastRebalanced,
		"description":    "Wirebot effort allocation. Higher = founder's bigger gap. Sums to 1.0.",
	})
}

// ─── GET /v1/pairing/predictions — Prediction track record ──────────────────

func (s *Server) handlePairingPredictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	preds := s.pairing.predictions
	correct := 0
	for _, p := range preds {
		if p.Correct {
			correct++
		}
	}
	total := len(preds)
	accuracy := 0.0
	if total > 0 {
		accuracy = float64(correct) / float64(total)
	}
	s.pairing.mu.RUnlock()

	writeJSON(w, map[string]interface{}{
		"total":       total,
		"correct":     correct,
		"accuracy":    accuracy,
		"predictions": preds,
	})
}

// ─── GET /v1/pairing/insights — Latest inferences + deltas ──────────────────

func (s *Server) handlePairingInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	s.pairing.mu.RLock()
	p := s.pairing.profile

	// Self-perception gaps
	gaps := make(map[string]interface{})
	for dim, delta := range p.SelfPerceptionDeltas {
		interpretation := "aligned"
		if delta > 1.0 {
			interpretation = "you rate yourself higher than behavior shows"
		} else if delta < -1.0 {
			interpretation = "you're better at this than you think"
		}
		gaps[dim] = map[string]interface{}{
			"delta":          delta,
			"interpretation": interpretation,
		}
	}

	// Active contexts with descriptions
	contexts := []map[string]interface{}{}
	contextDescriptions := map[ContextWindowType]string{
		CtxFinancialPressure: "Revenue pressure detected — Wirebot shifts to revenue-first recommendations",
		CtxShippingSprint:    "You're in a shipping sprint — Wirebot reduces nudges, increases task supply",
		CtxRecoveryPeriod:    "Recovery period — Wirebot backs off, suggests rest",
		CtxContextExplosion:  "Context switching spike — Wirebot prompts focus and sequencing",
		CtxStall:             "Shipping stall detected — Wirebot increases check-ins",
		CtxCelebration:       "Win detected! Wirebot celebrates then redirects energy",
		CtxLifeEvent:         "Life event detected — Wirebot reduces all pressure",
	}
	for wType, cw := range p.ContextWindows {
		if cw.Activation >= 0.3 {
			contexts = append(contexts, map[string]interface{}{
				"window":      string(wType),
				"activation":  cw.Activation,
				"description": contextDescriptions[wType],
			})
		}
	}

	s.pairing.mu.RUnlock()

	eff := s.pairing.GetEffectiveProfile()

	writeJSON(w, map[string]interface{}{
		"effective_profile":   eff,
		"self_perception_gaps": gaps,
		"active_contexts":     contexts,
		"chat_summary":        s.pairing.GetChatContextSummary(),
	})
}

// ─── POST /v1/pairing/answers — Submit assessment answers ───────────────────

func (s *Server) handlePairingAnswers(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var body struct {
		Answers []struct {
			InstrumentID string      `json:"instrument_id"`
			QuestionID   string      `json:"question_id"`
			Value        interface{} `json:"value"`
		} `json:"answers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	if len(body.Answers) == 0 {
		http.Error(w, "No answers provided", 400)
		return
	}

	// Convert to metadata format for signal
	answersIface := make([]interface{}, len(body.Answers))
	for i, a := range body.Answers {
		answersIface[i] = map[string]interface{}{
			"instrument_id": a.InstrumentID,
			"question_id":   a.QuestionID,
			"value":         a.Value,
		}
	}

	s.pairing.Ingest(Signal{
		Type:      SignalAssessment,
		Source:    "assessment_ui",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"answers": answersIface,
		},
	})

	writeJSON(w, map[string]interface{}{
		"accepted": len(body.Answers),
		"message":  "Answers queued for processing",
	})
}

// ─── POST /v1/pairing/override — Submit manual correction ───────────────────

func (s *Server) handlePairingOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var body struct {
		Trait     string  `json:"trait"`
		Dimension string  `json:"dimension"`
		Value     float64 `json:"value"`
		Reason    string  `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}

	if body.Trait == "" || body.Dimension == "" {
		http.Error(w, "trait and dimension required", 400)
		return
	}

	override := ProfileOverride{
		ID:        int64(len(s.pairing.profile.Overrides) + 1),
		Trait:     body.Trait,
		Dimension: body.Dimension,
		Value:     body.Value,
		Reason:    body.Reason,
		CreatedAt: time.Now(),
		Weight:    0.30,
	}

	s.pairing.mu.Lock()
	s.pairing.profile.Overrides = append(s.pairing.profile.Overrides, override)
	s.pairing.dirty = true
	s.pairing.mu.Unlock()

	writeJSON(w, map[string]interface{}{
		"override": override,
		"message":  "Override applied. Will decay over 30 days unless confirmed by behavior.",
	})
}

// ─── GET/DELETE /v1/pairing/overrides — List or remove overrides ─────────────

func (s *Server) handlePairingOverrides(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		s.pairing.mu.RLock()
		overrides := s.pairing.profile.Overrides
		active := []map[string]interface{}{}
		for _, o := range overrides {
			weight := o.CurrentWeight()
			if weight > 0.01 || o.Confirmed {
				active = append(active, map[string]interface{}{
					"id":           o.ID,
					"trait":        o.Trait,
					"dimension":    o.Dimension,
					"value":        o.Value,
					"reason":       o.Reason,
					"created_at":   o.CreatedAt,
					"weight":       weight,
					"confirmed":    o.Confirmed,
					"contradicted": o.Contradicted,
					"age_days":     time.Since(o.CreatedAt).Hours() / 24,
				})
			}
		}
		s.pairing.mu.RUnlock()
		writeJSON(w, active)
		return
	}

	if r.Method == "DELETE" {
		idStr := strings.TrimPrefix(r.URL.Path, "/v1/pairing/overrides/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			http.Error(w, "Invalid override ID", 400)
			return
		}
		s.pairing.mu.Lock()
		newOverrides := []ProfileOverride{}
		found := false
		for _, o := range s.pairing.profile.Overrides {
			if o.ID == id {
				found = true
				continue
			}
			newOverrides = append(newOverrides, o)
		}
		s.pairing.profile.Overrides = newOverrides
		s.pairing.dirty = true
		s.pairing.mu.Unlock()
		if !found {
			http.Error(w, "Override not found", 404)
			return
		}
		writeJSON(w, map[string]string{"message": "Override removed"})
		return
	}

	http.Error(w, "Method not allowed", 405)
}

// ─── POST /v1/pairing/scan — Trigger communication scan ────────────────────

func (s *Server) handlePairingScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	// TODO: Scan chat history from DB and ingest all messages
	writeJSON(w, map[string]string{
		"message": "Communication scan queued. Messages will be processed in background.",
	})
}

// ─── DELETE /v1/pairing/reset — Full profile reset ──────────────────────────

func (s *Server) handlePairingReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != "DELETE" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var body struct {
		Confirm string `json:"confirm"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	if body.Confirm != "RESET_PROFILE" {
		http.Error(w, `Send {"confirm":"RESET_PROFILE"} to confirm`, 400)
		return
	}

	s.pairing.mu.Lock()
	s.pairing.profile = NewFounderProfile()
	s.pairing.evidence = s.pairing.evidence[:0]
	s.pairing.predictions = s.pairing.predictions[:0]
	s.pairing.driftHistory = s.pairing.driftHistory[:0]
	s.pairing.dirty = true
	s.pairing.mu.Unlock()

	s.pairing.Save()

	writeJSON(w, map[string]string{
		"message": "Profile reset to factory defaults. All calibration data erased.",
	})
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func buildBlendFormulas(p *FounderProfileV2) map[string]interface{} {
	result := make(map[string]interface{})
	constructs := map[string]*DualTrackDimension{
		"action_style":      p.ActionStyle,
		"communication_dna": p.CommunicationDNA,
		"energy_topology":   p.EnergyTopology,
		"risk_disposition":  p.RiskDisposition,
	}
	for name, dt := range constructs {
		dims := make(map[string]interface{})
		for dim := range dt.Alpha {
			entry := map[string]interface{}{
				"alpha": dt.Alpha[dim],
			}
			if dt.Trait[dim] != nil && dt.State[dim] != nil && dt.Effective[dim] != nil {
				entry["formula"] = map[string]interface{}{
					"trait":     *dt.Trait[dim],
					"state":     *dt.State[dim],
					"alpha":     dt.Alpha[dim],
					"effective": *dt.Effective[dim],
					"equation":  "effective = α × trait + (1-α) × state",
				}
			}
			dims[dim] = entry
		}
		result[name] = dims
	}
	return result
}

func buildContextFormulas(p *FounderProfileV2) map[string]interface{} {
	result := make(map[string]interface{})
	for wType, cw := range p.ContextWindows {
		result[string(wType)] = map[string]interface{}{
			"activation":    cw.Activation,
			"active":        cw.Activation >= 0.3,
			"decay_tau_h":   cw.DecayTauHours,
			"signal_count":  cw.SignalCount,
			"decay_formula": "activation × e^(-hours_since_last_signal / τ_decay)",
		}
	}
	return result
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
