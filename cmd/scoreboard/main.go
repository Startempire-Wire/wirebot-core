package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed static/*
var staticFiles embed.FS

// ─── Config ─────────────────────────────────────────────────────────────────

var (
	listenAddr    = envOr("SCOREBOARD_ADDR", "127.0.0.1:8100")
	dbPath        = envOr("SCOREBOARD_DB", "/data/wirebot/scoreboard/events.db")
	checklistPath = envOr("CHECKLIST_PATH", "/home/wirebot/clawd/checklist.json")
	scoreboardJSON = envOr("SCOREBOARD_JSON", "/home/wirebot/clawd/scoreboard.json")
	authToken     = envOr("SCOREBOARD_TOKEN", "65b918ba-baf5-4996-8b53-6fb0f662a0c3")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ─── Data Types ─────────────────────────────────────────────────────────────

type Event struct {
	ID            string  `json:"id"`
	EventType     string  `json:"event_type"`
	Lane          string  `json:"lane"`
	Source        string  `json:"source"`
	Timestamp     string  `json:"timestamp"`
	ArtifactType  string  `json:"artifact_type,omitempty"`
	ArtifactURL   string  `json:"artifact_url,omitempty"`
	ArtifactTitle string  `json:"artifact_title,omitempty"`
	Confidence    float64 `json:"confidence"`
	Verifiers     string  `json:"verifiers,omitempty"`
	ScoreDelta    int     `json:"score_delta"`
	BusinessID    string  `json:"business_id,omitempty"`
	Metadata      string  `json:"metadata,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type DailyScore struct {
	Date              string `json:"date"`
	ExecutionScore    int    `json:"execution_score"`
	ShippingScore     int    `json:"shipping_score"`
	DistributionScore int    `json:"distribution_score"`
	RevenueScore      int    `json:"revenue_score"`
	SystemsScore      int    `json:"systems_score"`
	Penalties         int    `json:"penalties"`
	ShipsCount        int    `json:"ships_count"`
	Intent            string `json:"intent,omitempty"`
	IntentFulfilled   bool   `json:"intent_fulfilled"`
	Won               bool   `json:"won"`
}

type Season struct {
	Name          string `json:"name"`
	Number        int    `json:"number"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	Theme         string `json:"theme"`
	DaysElapsed   int    `json:"days_elapsed"`
	DaysRemaining int    `json:"days_remaining"`
	TotalScore    int    `json:"total_score"`
	DaysWon       int    `json:"days_won"`
	DaysPlayed    int    `json:"days_played"`
	AvgScore      int    `json:"avg_score"`
	Record        string `json:"record"`
}

type Streak struct {
	Current     int    `json:"current"`
	Best        int    `json:"best"`
	LastShipDate string `json:"last_ship_date,omitempty"`
	LastShip     string `json:"last_ship,omitempty"`
}

type ScoreboardView struct {
	Mode       string     `json:"mode"`
	Score      int        `json:"score"`
	Possession string     `json:"possession"`
	ShipToday  int        `json:"ship_today"`
	Streak     Streak     `json:"streak"`
	Record     string     `json:"record"`
	SeasonDay  string     `json:"season_day"`
	LastShip   string     `json:"last_ship"`
	Clock      ClockView  `json:"clock"`
	Lanes      LanesView  `json:"lanes"`
	Signal     string     `json:"signal"`
	Season     Season     `json:"season"`
}

type ClockView struct {
	DayProgress    float64 `json:"day_progress"`
	WeekProgress   float64 `json:"week_progress"`
	SeasonProgress float64 `json:"season_progress"`
}

type LanesView struct {
	Shipping     int `json:"shipping"`
	ShippingMax  int `json:"shipping_max"`
	Distribution int `json:"distribution"`
	DistMax      int `json:"distribution_max"`
	Revenue      int `json:"revenue"`
	RevenueMax   int `json:"revenue_max"`
	Systems      int `json:"systems"`
	SystemsMax   int `json:"systems_max"`
}

// ─── Server ─────────────────────────────────────────────────────────────────

type Server struct {
	db     *sql.DB
	mu     sync.RWMutex
	season Season
}

func main() {
	// Ensure DB directory exists
	os.MkdirAll("/data/wirebot/scoreboard", 0750)

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	s := &Server{db: db}
	s.initDB()
	s.loadSeason()

	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/v1/events", s.authMiddleware(s.handleEvents))
	mux.HandleFunc("/v1/score", s.authMiddleware(s.handleScore))
	mux.HandleFunc("/v1/scoreboard", s.handleScoreboard) // Public — stadium mode
	mux.HandleFunc("/v1/feed", s.authMiddleware(s.handleFeed))
	mux.HandleFunc("/v1/season", s.authMiddleware(s.handleSeason))
	mux.HandleFunc("/health", s.handleHealth)

	// Static files (stadium mode UI)
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("Scoreboard listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

// ─── Database ───────────────────────────────────────────────────────────────

func (s *Server) initDB() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			lane TEXT NOT NULL,
			source TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			artifact_type TEXT DEFAULT '',
			artifact_url TEXT DEFAULT '',
			artifact_title TEXT DEFAULT '',
			confidence REAL DEFAULT 1.0,
			verifiers TEXT DEFAULT '[]',
			score_delta INTEGER DEFAULT 0,
			business_id TEXT DEFAULT '',
			metadata TEXT DEFAULT '{}',
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_date ON events(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_events_lane ON events(lane)`,
		`CREATE TABLE IF NOT EXISTS daily_scores (
			date TEXT PRIMARY KEY,
			execution_score INTEGER DEFAULT 0,
			shipping_score INTEGER DEFAULT 0,
			distribution_score INTEGER DEFAULT 0,
			revenue_score INTEGER DEFAULT 0,
			systems_score INTEGER DEFAULT 0,
			penalties INTEGER DEFAULT 0,
			ships_count INTEGER DEFAULT 0,
			intent TEXT DEFAULT '',
			intent_fulfilled BOOLEAN DEFAULT 0,
			won BOOLEAN DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY,
			name TEXT,
			number INTEGER,
			start_date TEXT,
			end_date TEXT,
			theme TEXT,
			is_active BOOLEAN DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS streaks (
			streak_type TEXT PRIMARY KEY,
			current_len INTEGER DEFAULT 0,
			best_len INTEGER DEFAULT 0,
			last_date TEXT DEFAULT '',
			last_artifact TEXT DEFAULT ''
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			log.Fatalf("initDB: %v", err)
		}
	}

	// Seed default season if none
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&count)
	if count == 0 {
		s.db.Exec(`INSERT INTO seasons (name, number, start_date, end_date, theme, is_active)
			VALUES ('Red-to-Black', 1, '2026-02-01', '2026-05-01', 'Break even. Ship what makes money. Get out of the red.', 1)`)
	}

	// Seed default streak rows
	for _, st := range []string{"ship", "no_zero"} {
		s.db.Exec("INSERT OR IGNORE INTO streaks (streak_type) VALUES (?)", st)
	}
}

func (s *Server) loadSeason() {
	row := s.db.QueryRow("SELECT name, number, start_date, end_date, theme FROM seasons WHERE is_active=1 LIMIT 1")
	var name, start, end, theme string
	var num int
	if err := row.Scan(&name, &num, &start, &end, &theme); err != nil {
		s.season = Season{Name: "Default", Number: 1, StartDate: "2026-02-01", EndDate: "2026-05-01"}
		return
	}
	s.season = Season{Name: name, Number: num, StartDate: start, EndDate: end, Theme: theme}
	s.recalcSeason()
}

func (s *Server) recalcSeason() {
	now := time.Now()
	startT, _ := time.Parse("2006-01-02", s.season.StartDate)
	endT, _ := time.Parse("2006-01-02", s.season.EndDate)
	elapsed := int(now.Sub(startT).Hours() / 24)
	remaining := int(endT.Sub(now).Hours() / 24)
	if elapsed < 0 { elapsed = 0 }
	if remaining < 0 { remaining = 0 }
	s.season.DaysElapsed = elapsed
	s.season.DaysRemaining = remaining

	// Count wins/played
	var won, played int
	s.db.QueryRow("SELECT COUNT(*) FROM daily_scores WHERE date >= ? AND date <= ?",
		s.season.StartDate, s.season.EndDate).Scan(&played)
	s.db.QueryRow("SELECT COUNT(*) FROM daily_scores WHERE date >= ? AND date <= ? AND won=1",
		s.season.StartDate, s.season.EndDate).Scan(&won)
	s.season.DaysWon = won
	s.season.DaysPlayed = played
	if played > 0 {
		var total int
		s.db.QueryRow("SELECT COALESCE(SUM(execution_score),0) FROM daily_scores WHERE date >= ? AND date <= ?",
			s.season.StartDate, s.season.EndDate).Scan(&total)
		s.season.TotalScore = total
		s.season.AvgScore = total / played
	}
	s.season.Record = fmt.Sprintf("%dW-%dL", won, played-won)
}

// ─── Auth ───────────────────────────────────────────────────────────────────

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != authToken && r.URL.Query().Get("token") != authToken {
			http.Error(w, `{"error":"unauthorized"}`, 401)
			return
		}
		next(w, r)
	}
}

// ─── Handlers ───────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	var eventCount int
	s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&eventCount)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"events": eventCount,
		"season": s.season.Name,
	})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "POST":
		var evt struct {
			EventType     string  `json:"event_type"`
			Lane          string  `json:"lane"`
			Source        string  `json:"source"`
			Timestamp     string  `json:"timestamp"`
			ArtifactType  string  `json:"artifact_type"`
			ArtifactURL   string  `json:"artifact_url"`
			ArtifactTitle string  `json:"artifact_title"`
			Confidence    float64 `json:"confidence"`
			Verifiers     json.RawMessage `json:"verifiers"`
			BusinessID    string  `json:"business_id"`
			Metadata      json.RawMessage `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
			http.Error(w, `{"error":"invalid json"}`, 400)
			return
		}

		if evt.EventType == "" || evt.Lane == "" || evt.Source == "" {
			http.Error(w, `{"error":"event_type, lane, source required"}`, 400)
			return
		}
		if evt.Timestamp == "" {
			evt.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
		if evt.Confidence == 0 {
			evt.Confidence = 1.0
		}

		// Calculate score delta based on lane + event type
		scoreDelta := calcScoreDelta(evt.Lane, evt.EventType, evt.Confidence)

		id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
		verifiers := "{}"
		if evt.Verifiers != nil {
			verifiers = string(evt.Verifiers)
		}
		metadata := "{}"
		if evt.Metadata != nil {
			metadata = string(evt.Metadata)
		}

		s.mu.Lock()
		_, err := s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_type, artifact_url, artifact_title, confidence, verifiers,
			score_delta, business_id, metadata, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, evt.EventType, evt.Lane, evt.Source, evt.Timestamp,
			evt.ArtifactType, evt.ArtifactURL, evt.ArtifactTitle, evt.Confidence,
			verifiers, scoreDelta, evt.BusinessID, metadata, time.Now().UTC().Format(time.RFC3339))
		s.mu.Unlock()

		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
			return
		}

		// Update daily score
		today := time.Now().Format("2006-01-02")
		s.updateDailyScore(today)
		s.updateStreak(today, evt.ArtifactTitle)
		s.recalcSeason()

		daily := s.getDailyScore(today)
		streak := s.getStreak("ship")

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":              true,
			"event_id":        id,
			"score_delta":     scoreDelta,
			"new_daily_score": daily.ExecutionScore,
			"streak":          streak,
		})

	case "GET":
		date := r.URL.Query().Get("date")
		lane := r.URL.Query().Get("lane")
		limit := 50

		query := "SELECT id, event_type, lane, source, timestamp, artifact_type, artifact_url, artifact_title, confidence, score_delta, business_id, created_at FROM events WHERE 1=1"
		args := []interface{}{}
		if date != "" {
			query += " AND timestamp LIKE ?"
			args = append(args, date+"%")
		}
		if lane != "" {
			query += " AND lane = ?"
			args = append(args, lane)
		}
		query += " ORDER BY timestamp DESC LIMIT ?"
		args = append(args, limit)

		rows, err := s.db.Query(query, args...)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
			return
		}
		defer rows.Close()

		var events []Event
		for rows.Next() {
			var e Event
			rows.Scan(&e.ID, &e.EventType, &e.Lane, &e.Source, &e.Timestamp,
				&e.ArtifactType, &e.ArtifactURL, &e.ArtifactTitle, &e.Confidence,
				&e.ScoreDelta, &e.BusinessID, &e.CreatedAt)
			events = append(events, e)
		}
		if events == nil {
			events = []Event{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"events": events})

	default:
		http.Error(w, `{"error":"method not allowed"}`, 405)
	}
}

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	daily := s.getDailyScore(date)
	streak := s.getStreak("ship")
	s.recalcSeason()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"date":   date,
		"score":  daily,
		"streak": streak,
		"season": s.season,
	})
}

func (s *Server) handleScoreboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	today := time.Now().Format("2006-01-02")
	daily := s.getDailyScore(today)
	streak := s.getStreak("ship")
	s.recalcSeason()

	// Get active business from checklist
	possession := "—"
	if data, err := os.ReadFile(checklistPath); err == nil {
		var cl map[string]interface{}
		if json.Unmarshal(data, &cl) == nil {
			if businesses, ok := cl["businesses"].([]interface{}); ok {
				activeID, _ := cl["activeBusiness"].(string)
				for _, b := range businesses {
					bm, _ := b.(map[string]interface{})
					if bm["id"] == activeID {
						possession, _ = bm["name"].(string)
						break
					}
				}
			}
		}
	}

	// Clock
	now := time.Now()
	dayProgress := float64(now.Hour()*60+now.Minute()) / 1440.0
	weekday := int(now.Weekday())
	if weekday == 0 { weekday = 7 }
	weekProgress := float64(weekday-1) / 7.0
	seasonProgress := 0.0
	if s.season.DaysElapsed+s.season.DaysRemaining > 0 {
		seasonProgress = float64(s.season.DaysElapsed) / float64(s.season.DaysElapsed+s.season.DaysRemaining)
	}

	// Signal
	signal := "green"
	if daily.ExecutionScore < 30 {
		signal = "red"
	} else if daily.ExecutionScore < 50 {
		signal = "yellow"
	}

	view := ScoreboardView{
		Mode:       "stadium",
		Score:      daily.ExecutionScore,
		Possession: possession,
		ShipToday:  daily.ShipsCount,
		Streak:     streak,
		Record:     s.season.Record,
		SeasonDay:  fmt.Sprintf("Day %d of %d", s.season.DaysElapsed, s.season.DaysElapsed+s.season.DaysRemaining),
		LastShip:   streak.LastShip,
		Clock: ClockView{
			DayProgress:    dayProgress,
			WeekProgress:   weekProgress,
			SeasonProgress: seasonProgress,
		},
		Lanes: LanesView{
			Shipping: daily.ShippingScore, ShippingMax: 40,
			Distribution: daily.DistributionScore, DistMax: 25,
			Revenue: daily.RevenueScore, RevenueMax: 20,
			Systems: daily.SystemsScore, SystemsMax: 15,
		},
		Signal: signal,
		Season: s.season,
	}

	json.NewEncoder(w).Encode(view)
}

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	limit := 20
	rows, err := s.db.Query(`SELECT id, event_type, lane, source, timestamp, artifact_title, score_delta
		FROM events ORDER BY timestamp DESC LIMIT ?`, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}
	defer rows.Close()

	type FeedItem struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Lane      string `json:"lane"`
		Source    string `json:"source"`
		Timestamp string `json:"timestamp"`
		Title     string `json:"title"`
		Delta     int    `json:"score_delta"`
		Icon      string `json:"icon"`
	}
	var items []FeedItem
	icons := map[string]string{"shipping": "🚀", "distribution": "📣", "revenue": "💰", "systems": "⚙️"}
	for rows.Next() {
		var f FeedItem
		rows.Scan(&f.ID, &f.Type, &f.Lane, &f.Source, &f.Timestamp, &f.Title, &f.Delta)
		f.Icon = icons[f.Lane]
		items = append(items, f)
	}
	if items == nil { items = []FeedItem{} }
	json.NewEncoder(w).Encode(map[string]interface{}{"items": items})
}

func (s *Server) handleSeason(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	s.recalcSeason()
	json.NewEncoder(w).Encode(s.season)
}

// ─── Score Engine ───────────────────────────────────────────────────────────

func calcScoreDelta(lane, eventType string, confidence float64) int {
	base := map[string]map[string]int{
		"shipping": {
			"PRODUCT_RELEASE": 10, "DEPLOY_SUCCESS": 8, "FEATURE_SHIPPED": 6,
			"APP_STORE_SUBMIT": 10, "PUBLIC_ARTIFACT": 5, "INFRASTRUCTURE_ACTIVATED": 7,
			"TASK_COMPLETED": 4, "CODE_PUSHED": 3,
		},
		"distribution": {
			"BLOG_PUBLISHED": 6, "VIDEO_PUBLISHED": 7, "EMAIL_CAMPAIGN_SENT": 5,
			"SOCIAL_POST_BUSINESS": 4, "COLD_OUTREACH": 4, "PODCAST_PUBLISHED": 6,
		},
		"revenue": {
			"PAYMENT_RECEIVED": 10, "SUBSCRIPTION_CREATED": 12, "DEAL_CLOSED": 8,
			"PROPOSAL_SENT": 4, "INVOICE_PAID": 8,
		},
		"systems": {
			"AUTOMATION_DEPLOYED": 6, "SOP_DOCUMENTED": 4, "TOOL_INTEGRATED": 5,
			"DELEGATION_COMPLETED": 6, "MONITORING_ENABLED": 4,
		},
	}

	if laneMap, ok := base[lane]; ok {
		if pts, ok := laneMap[eventType]; ok {
			return int(float64(pts) * confidence)
		}
		return 3 // Unknown event type in known lane
	}
	return 1 // Unknown lane
}

func (s *Server) updateDailyScore(date string) {
	// Sum by lane for today
	var shipping, distribution, revenue, systems, ships int
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='shipping' AND timestamp LIKE ?", date+"%").Scan(&shipping)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='distribution' AND timestamp LIKE ?", date+"%").Scan(&distribution)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='revenue' AND timestamp LIKE ?", date+"%").Scan(&revenue)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='systems' AND timestamp LIKE ?", date+"%").Scan(&systems)
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE lane='shipping' AND timestamp LIKE ?", date+"%").Scan(&ships)

	// Cap per lane
	if shipping > 40 { shipping = 40 }
	if distribution > 25 { distribution = 25 }
	if revenue > 20 { revenue = 20 }
	if systems > 15 { systems = 15 }

	total := shipping + distribution + revenue + systems
	// No ship penalty: can't exceed 30
	if ships == 0 && total > 30 {
		total = 30
	}
	won := total >= 50

	s.mu.Lock()
	s.db.Exec(`INSERT INTO daily_scores (date, execution_score, shipping_score, distribution_score, revenue_score, systems_score, ships_count, won)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			execution_score=excluded.execution_score,
			shipping_score=excluded.shipping_score,
			distribution_score=excluded.distribution_score,
			revenue_score=excluded.revenue_score,
			systems_score=excluded.systems_score,
			ships_count=excluded.ships_count,
			won=excluded.won`,
		date, total, shipping, distribution, revenue, systems, ships, won)
	s.mu.Unlock()
}

func (s *Server) getDailyScore(date string) DailyScore {
	var ds DailyScore
	ds.Date = date
	s.db.QueryRow(`SELECT execution_score, shipping_score, distribution_score, revenue_score,
		systems_score, penalties, ships_count, intent, intent_fulfilled, won
		FROM daily_scores WHERE date=?`, date).Scan(
		&ds.ExecutionScore, &ds.ShippingScore, &ds.DistributionScore,
		&ds.RevenueScore, &ds.SystemsScore, &ds.Penalties, &ds.ShipsCount,
		&ds.Intent, &ds.IntentFulfilled, &ds.Won)
	return ds
}

func (s *Server) updateStreak(date string, artifact string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var lastDate string
	var current, best int
	s.db.QueryRow("SELECT current_len, best_len, last_date FROM streaks WHERE streak_type='ship'").Scan(&current, &best, &lastDate)

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if lastDate == date {
		// Already counted today
	} else if lastDate == yesterday {
		current++
	} else {
		current = 1
	}
	if current > best {
		best = current
	}

	s.db.Exec("UPDATE streaks SET current_len=?, best_len=?, last_date=?, last_artifact=? WHERE streak_type='ship'",
		current, best, date, artifact)
}

func (s *Server) getStreak(stype string) Streak {
	var st Streak
	var current, best int
	var lastDate, lastArtifact string
	s.db.QueryRow("SELECT current_len, best_len, last_date, last_artifact FROM streaks WHERE streak_type=?", stype).Scan(
		&current, &best, &lastDate, &lastArtifact)
	st.Current = current
	st.Best = best
	st.LastShipDate = lastDate
	st.LastShip = lastArtifact
	return st
}
