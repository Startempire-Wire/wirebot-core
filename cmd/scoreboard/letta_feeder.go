package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// lettaStateFeeder polls for new approved memories and scoreboard events,
// then sends each to the Letta agent as an async message. Letta's internal
// LLM decides which blocks to update. This is a data pipeline, not an agent loop.
//
// Rate limited to 10 messages per 5-minute window. Watermarks survive restarts
// via the letta_feeder_state table.
func (s *Server) lettaStateFeeder() {
	if s.lettaAgentID == "" {
		log.Println("[letta-feeder] No LETTA_AGENT_ID configured, feeder disabled")
		return
	}

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	var msgCount int
	windowStart := time.Now()

	// Load watermarks from DB (survive restarts)
	lastApprovedRowID := s.feederGetWatermark("approved_rowid")
	lastEventRowID := s.feederGetWatermark("event_rowid")

	log.Printf("[letta-feeder] Started (agent=%s, approved_wm=%d, event_wm=%d)",
		s.lettaAgentID[:20]+"…", lastApprovedRowID, lastEventRowID)

	for range ticker.C {
		// Rate limit: max 10 messages per 5-minute window
		if time.Since(windowStart) > 5*time.Minute {
			msgCount = 0
			windowStart = time.Now()
		}
		if msgCount >= 10 {
			continue
		}

		// ── 1. Approved memories → Letta ─────────────────────────────────
		rows, err := s.db.Query(`
			SELECT rowid, memory_text, COALESCE(correction, '') 
			FROM memory_queue
			WHERE status='approved' AND rowid > ?
			ORDER BY rowid LIMIT 1`, lastApprovedRowID)
		if err == nil {
			for rows.Next() {
				var rowid int64
				var text, correction string
				if rows.Scan(&rowid, &text, &correction) != nil {
					continue
				}
				// Use correction if operator edited, otherwise original
				fact := text
				if correction != "" {
					fact = correction
				}
				msg := fmt.Sprintf(
					"Approved memory from operator review:\n%s\n\nUpdate your human, goals, kpis, or business_stage blocks if this is relevant to the business.",
					fact)
				if !s.lettaSendAsync(msg) {
					break // Letta down — stop, retry next tick
				}
				msgCount++
				lastApprovedRowID = rowid
				log.Printf("[letta-feeder] Sent approved memory (rowid=%d, %d chars)", rowid, len(fact))
				if msgCount >= 10 {
					break
				}
			}
			rows.Close()
			s.feederSetWatermark("approved_rowid", lastApprovedRowID)
		}

		if msgCount >= 10 {
			continue
		}

		// ── 2. Scoreboard events → Letta ─────────────────────────────────
		rows, err = s.db.Query(`
			SELECT rowid, event_type, lane, source, artifact_title
			FROM events
			WHERE rowid > ?
			ORDER BY rowid LIMIT 1`, lastEventRowID)
		if err == nil {
			for rows.Next() {
				var rowid int64
				var eventType, lane, source, title string
				if rows.Scan(&rowid, &eventType, &lane, &source, &title) != nil {
					continue
				}
				msg := fmt.Sprintf(
					"Scoreboard event [%s/%s] from %s: %s\n\nUpdate KPIs or goals blocks if this affects business metrics.",
					lane, eventType, source, title)
				if !s.lettaSendAsync(msg) {
					break // Letta down — stop, retry next tick
				}
				msgCount++
				lastEventRowID = rowid
				log.Printf("[letta-feeder] Sent event (rowid=%d, %s/%s)", rowid, lane, eventType)
				if msgCount >= 10 {
					break
				}
			}
			rows.Close()
			s.feederSetWatermark("event_rowid", lastEventRowID)
		}
	}
}

// lettaSendAsync sends a message to the Letta agent. Returns true on success.
func (s *Server) lettaSendAsync(message string) bool {
	payload, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]string{{"role": "user", "content": message}},
	})
	req, err := http.NewRequest("POST",
		fmt.Sprintf("%s/v1/agents/%s/messages", s.lettaURL, s.lettaAgentID),
		bytes.NewReader(payload))
	if err != nil {
		log.Printf("[letta-feeder] Request build error: %v", err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[letta-feeder] Send error: %v", err)
		return false
	}
	resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 400
}

// feederGetWatermark reads a watermark value from the DB.
func (s *Server) feederGetWatermark(key string) int64 {
	var val int64
	s.db.QueryRow("SELECT value FROM letta_feeder_state WHERE key=?", key).Scan(&val)
	return val // 0 if not found
}

// feederSetWatermark persists a watermark to the DB.
func (s *Server) feederSetWatermark(key string, val int64) {
	s.db.Exec(`INSERT OR REPLACE INTO letta_feeder_state (key, value, updated_at) VALUES (?, ?, datetime('now'))`,
		key, val)
}
