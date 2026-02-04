package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// lettaBlock mirrors Letta's block response.
type lettaBlock struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// alert represents a generated alert from Letta state analysis.
type alert struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`       // goal_deadline, goal_overdue, kpi_stale
	Title     string `json:"title"`
	Detail    string `json:"detail"`
	Severity  string `json:"severity"`   // info, warning, critical
	CreatedAt string `json:"created_at"`
	DismissedAt string `json:"dismissed_at,omitempty"`
}

// parsedGoal is a goal extracted from the goals block.
type parsedGoal struct {
	Title    string
	Due      time.Time
	HasDue   bool
	Complete bool
}

// initAlerts creates the alerts table.
func (s *Server) initAlerts() {
	s.db.Exec(`CREATE TABLE IF NOT EXISTS alerts (
		id TEXT PRIMARY KEY,
		kind TEXT NOT NULL,
		title TEXT NOT NULL,
		detail TEXT DEFAULT '',
		severity TEXT DEFAULT 'info',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		dismissed_at DATETIME
	)`)
}

// lettaAlertChecker runs every hour, reads Letta blocks, generates alerts.
func (s *Server) lettaAlertChecker() {
	if s.lettaAgentID == "" {
		return
	}

	// First check after 2 minutes (let feeder warm up)
	time.Sleep(2 * time.Minute)
	s.checkLettaAlerts()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		s.checkLettaAlerts()
	}
}

func (s *Server) checkLettaAlerts() {
	blocks, err := s.fetchLettaBlocks()
	if err != nil {
		log.Printf("[letta-alerts] Failed to fetch blocks: %v", err)
		return
	}

	now := operatorNow()
	var generated int

	// ── Goal deadline alerts ─────────────────────────────────────────
	for _, b := range blocks {
		if b.Label != "goals" {
			continue
		}
		goals := parseGoals(b.Value)
		for _, g := range goals {
			if g.Complete || !g.HasDue {
				continue
			}
			daysLeft := int(g.Due.Sub(now).Hours() / 24)

			if daysLeft < 0 {
				// Overdue
				id := fmt.Sprintf("overdue_%s_%s", g.Due.Format("2006-01-02"), sanitizeID(g.Title))
				if s.insertAlert(id, "goal_overdue", g.Title,
					fmt.Sprintf("Overdue by %d days", -daysLeft), "critical") {
					generated++
				}
			} else if daysLeft <= 3 {
				// Approaching deadline
				id := fmt.Sprintf("deadline_%s_%s", g.Due.Format("2006-01-02"), sanitizeID(g.Title))
				if s.insertAlert(id, "goal_deadline", g.Title,
					fmt.Sprintf("%d days remaining", daysLeft), "warning") {
					generated++
				}
			}
		}
	}

	if generated > 0 {
		log.Printf("[letta-alerts] Generated %d new alerts", generated)
	}
}

// fetchLettaBlocks reads all blocks from the Letta agent.
func (s *Server) fetchLettaBlocks() ([]lettaBlock, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/core-memory/blocks", s.lettaURL, s.lettaAgentID)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var blocks []lettaBlock
	if err := json.Unmarshal(body, &blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// dateRe matches "Due YYYY-MM-DD" or "Due ~YYYY-MM-DD" patterns.
var dateRe = regexp.MustCompile(`Due\s+~?(\d{4}-\d{2}-\d{2})`)

// completeRe matches "COMPLETE" or "✅" markers.
var completeRe = regexp.MustCompile(`(?i)COMPLETE|✅`)

// goalLineRe matches numbered goal lines: "N. Title — ..."
var goalLineRe = regexp.MustCompile(`^\d+\.\s+(.+?)(?:\s+—\s+|\s*$)`)

// parseGoals extracts goals from the goals block text.
func parseGoals(text string) []parsedGoal {
	var goals []parsedGoal
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		m := goalLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		g := parsedGoal{
			Title:    m[1],
			Complete: completeRe.MatchString(line),
		}

		if dm := dateRe.FindStringSubmatch(line); dm != nil {
			if t, err := time.Parse("2006-01-02", dm[1]); err == nil {
				g.Due = t
				g.HasDue = true
			}
		}

		goals = append(goals, g)
	}
	return goals
}

// insertAlert creates an alert if it doesn't already exist. Returns true if new.
func (s *Server) insertAlert(id, kind, title, detail, severity string) bool {
	res, err := s.db.Exec(`INSERT OR IGNORE INTO alerts (id, kind, title, detail, severity) VALUES (?, ?, ?, ?, ?)`,
		id, kind, title, detail, severity)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// handleAlerts serves GET /v1/alerts — active (undismissed) alerts.
func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		s.handleDismissAlert(w, r)
		return
	}

	rows, err := s.db.Query(`
		SELECT id, kind, title, detail, severity, created_at
		FROM alerts
		WHERE dismissed_at IS NULL
		ORDER BY 
			CASE severity WHEN 'critical' THEN 0 WHEN 'warning' THEN 1 ELSE 2 END,
			created_at DESC
		LIMIT 20`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	var alerts []alert
	for rows.Next() {
		var a alert
		rows.Scan(&a.ID, &a.Kind, &a.Title, &a.Detail, &a.Severity, &a.CreatedAt)
		alerts = append(alerts, a)
	}
	if alerts == nil {
		alerts = []alert{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alerts)
}

// handleDismissAlert handles POST /v1/alerts with {"id":"..."} to dismiss.
func (s *Server) handleDismissAlert(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		http.Error(w, "id required", 400)
		return
	}
	s.db.Exec(`UPDATE alerts SET dismissed_at=CURRENT_TIMESTAMP WHERE id=?`, body.ID)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"ok":true}`))
}

// sanitizeID creates a stable short ID from a title string.
func sanitizeID(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, s)
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
