package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed static/*
var staticFiles embed.FS

// ─── Config ─────────────────────────────────────────────────────────────────

var (
	listenAddr     = envOr("SCOREBOARD_ADDR", "127.0.0.1:8100")
	dbPath         = envOr("SCOREBOARD_DB", "/data/wirebot/scoreboard/events.db")
	checklistPath  = envOr("CHECKLIST_PATH", "/home/wirebot/clawd/checklist.json")
	scoreboardJSON = envOr("SCOREBOARD_JSON", "/home/wirebot/clawd/scoreboard.json")
	authToken      = envOr("SCOREBOARD_TOKEN", "65b918ba-baf5-4996-8b53-6fb0f662a0c3")
	masterKeyHex   = envOr("SCOREBOARD_MASTER_KEY", "") // 64-char hex = 32-byte AES-256 key
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ─── Data Types ─────────────────────────────────────────────────────────────

type Event struct {
	ID                string  `json:"id"`
	EventType         string  `json:"event_type"`
	Lane              string  `json:"lane"`
	Source            string  `json:"source"`
	Timestamp         string  `json:"timestamp"`
	ArtifactType      string  `json:"artifact_type,omitempty"`
	ArtifactURL       string  `json:"artifact_url,omitempty"`
	ArtifactTitle     string  `json:"artifact_title,omitempty"`
	Confidence        float64 `json:"confidence"`
	Verifiers         string  `json:"verifiers,omitempty"`
	VerificationLevel string  `json:"verification_level,omitempty"` // STRONG, MEDIUM, WEAK, SELF_REPORTED, UNVERIFIED
	ScoreDelta        int     `json:"score_delta"`
	BusinessID        string  `json:"business_id,omitempty"`
	Metadata          string  `json:"metadata,omitempty"`
	Status            string  `json:"status"` // approved, pending, rejected
	CreatedAt         string  `json:"created_at"`
}

// ─── Integration Types ──────────────────────────────────────────────────────

type Integration struct {
	ID               string `json:"id"`
	UserID           string `json:"user_id"`
	Provider         string `json:"provider"`
	AuthType         string `json:"auth_type"` // oauth2, api_key, webhook_secret, rss_url
	DisplayName      string `json:"display_name"`
	Scopes           string `json:"scopes"`
	Status           string `json:"status"` // active, expired, revoked, error
	Sensitivity      string `json:"sensitivity"` // public, standard, sensitive, financial
	WirebotVisible   bool   `json:"wirebot_visible"`
	WirebotDetail    string `json:"wirebot_detail_level"` // full, summary, binary, none
	ShareLevel       string `json:"share_level"` // private, anonymized, shared, public
	PollInterval     int    `json:"poll_interval_seconds"`
	LastUsedAt       string `json:"last_used_at,omitempty"`
	LastError        string `json:"last_error,omitempty"`
	CreatedAt        string `json:"created_at"`
}

type RSSItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	GUID    string `xml:"guid"`
}

type RSSFeed struct {
	Channel struct {
		Items []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type AtomFeed struct {
	Entries []struct {
		Title   string `xml:"title"`
		Link    struct{ Href string `xml:"href,attr"` } `xml:"link"`
		Updated string `xml:"updated"`
		ID      string `xml:"id"`
	} `xml:"entry"`
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
	Current      int    `json:"current"`
	Best         int    `json:"best"`
	LastShipDate string `json:"last_ship_date,omitempty"`
	LastShip     string `json:"last_ship,omitempty"`
}

type ScoreboardView struct {
	Mode        string    `json:"mode"`
	Score       int       `json:"score"`
	Possession  string    `json:"possession"`
	ShipToday   int       `json:"ship_today"`
	Streak      Streak    `json:"streak"`
	Record      string    `json:"record"`
	SeasonDay   string    `json:"season_day"`
	LastShip    string    `json:"last_ship"`
	Clock       ClockView `json:"clock"`
	Lanes       LanesView `json:"lanes"`
	Signal      string    `json:"signal"`
	Season      Season    `json:"season"`
	Intent      string    `json:"intent,omitempty"`
	StallHours  float64   `json:"stall_hours,omitempty"`
	Penalties   int       `json:"penalties"`
	StreakBonus int       `json:"streak_bonus"`
	PendingCount int     `json:"pending_count"`
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

type FeedItem struct {
	ID         string  `json:"id"`
	Type       string  `json:"type"`
	Lane       string  `json:"lane"`
	Source     string  `json:"source"`
	Timestamp  string  `json:"timestamp"`
	Title      string  `json:"title"`
	Delta      int     `json:"score_delta"`
	Icon       string  `json:"icon"`
	URL        string  `json:"url,omitempty"`
	Confidence float64 `json:"confidence"`
	Status     string  `json:"status"`
}

// ─── Server ─────────────────────────────────────────────────────────────────

// Operator timezone — all "today" calculations use this, not UTC.
// Events are still stored with UTC timestamps, but daily scores,
// streaks, and seasons group by the operator's local date.
var operatorTZ *time.Location

func init() {
	var err error
	operatorTZ, err = time.LoadLocation("America/Los_Angeles")
	if err != nil {
		operatorTZ = time.UTC
	}
}

// operatorToday returns today's date in the operator's timezone.
func operatorToday() string {
	return time.Now().In(operatorTZ).Format("2006-01-02")
}

// operatorNow returns the current time in the operator's timezone.
func operatorNow() time.Time {
	return time.Now().In(operatorTZ)
}

type Server struct {
	db       *sql.DB
	mu       sync.RWMutex
	season   Season
	tenantID string // empty = operator (default), otherwise randID
}

// ─── Tenant Manager ─────────────────────────────────────────────────────────

type TenantManager struct {
	mu       sync.RWMutex
	tenants  map[string]*Server // randID → Server
	basePath string
}

type TenantInfo struct {
	TenantID  string `json:"tenant_id"`
	UserID    int    `json:"user_id,omitempty"`
	Tier      string `json:"tier,omitempty"`
	CreatedAt string `json:"created_at"`
	Active    bool   `json:"active"`
	DBPath    string `json:"db_path,omitempty"`
}

func NewTenantManager(basePath string) *TenantManager {
	os.MkdirAll(basePath+"/tenants", 0750)
	tm := &TenantManager{
		tenants:  make(map[string]*Server),
		basePath: basePath,
	}
	// Load existing tenants from disk
	tm.loadExisting()
	return tm
}

func (tm *TenantManager) loadExisting() {
	entries, err := os.ReadDir(tm.basePath + "/tenants")
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			tid := e.Name()
			infoPath := fmt.Sprintf("%s/tenants/%s/info.json", tm.basePath, tid)
			if _, err := os.Stat(infoPath); err == nil {
				// Load tenant lazily on first request
				log.Printf("Found tenant: %s", tid)
			}
		}
	}
}

func (tm *TenantManager) GetOrCreate(tenantID string) (*Server, error) {
	tm.mu.RLock()
	if s, ok := tm.tenants[tenantID]; ok {
		tm.mu.RUnlock()
		return s, nil
	}
	tm.mu.RUnlock()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Double-check after acquiring write lock
	if s, ok := tm.tenants[tenantID]; ok {
		return s, nil
	}

	tenantDir := fmt.Sprintf("%s/tenants/%s", tm.basePath, tenantID)
	os.MkdirAll(tenantDir, 0750)

	dbFile := tenantDir + "/events.db"
	db, err := sql.Open("sqlite3", dbFile+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open tenant db: %w", err)
	}

	s := &Server{db: db, tenantID: tenantID}
	s.initDB()
	s.loadSeason()
	tm.tenants[tenantID] = s

	log.Printf("Loaded tenant: %s (db: %s)", tenantID, dbFile)
	return s, nil
}

func (tm *TenantManager) Provision(tenantID string, userID int, tier string) (*TenantInfo, error) {
	tenantDir := fmt.Sprintf("%s/tenants/%s", tm.basePath, tenantID)
	os.MkdirAll(tenantDir, 0750)

	info := &TenantInfo{
		TenantID:  tenantID,
		UserID:    userID,
		Tier:      tier,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Active:    true,
		DBPath:    tenantDir + "/events.db",
	}

	// Write info.json
	data, _ := json.MarshalIndent(info, "", "  ")
	if err := os.WriteFile(tenantDir+"/info.json", data, 0640); err != nil {
		return nil, err
	}

	// Ensure DB is initialized
	if _, err := tm.GetOrCreate(tenantID); err != nil {
		return nil, err
	}

	return info, nil
}

func (tm *TenantManager) List() []TenantInfo {
	entries, err := os.ReadDir(tm.basePath + "/tenants")
	if err != nil {
		return nil
	}
	var result []TenantInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		infoPath := fmt.Sprintf("%s/tenants/%s/info.json", tm.basePath, e.Name())
		data, err := os.ReadFile(infoPath)
		if err != nil {
			continue
		}
		var info TenantInfo
		if err := json.Unmarshal(data, &info); err == nil {
			result = append(result, info)
		}
	}
	return result
}

func main() {
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

	// Public endpoints
	mux.HandleFunc("/v1/scoreboard", s.handleScoreboard)
	mux.HandleFunc("/health", s.handleHealth)

	// Authenticated endpoints
	mux.HandleFunc("/v1/events", s.auth(s.handleEvents))
	mux.HandleFunc("/v1/events/batch", s.auth(s.handleEventsBatch))
	mux.HandleFunc("/v1/score", s.auth(s.handleScore))
	// Read-only endpoints: public (PWA served same origin, behind CF tunnel)
	mux.HandleFunc("/v1/feed", s.handleFeed)
	mux.HandleFunc("/v1/season", s.handleSeason)
	mux.HandleFunc("/v1/season/wrapped", s.handleWrapped)
	mux.HandleFunc("/v1/history", s.handleHistory)
	// Write endpoints: auth required
	mux.HandleFunc("/v1/intent", s.auth(s.handleIntent))
	mux.HandleFunc("/v1/audit", s.auth(s.handleAudit))

	// Gated events (pending/approve/reject)
	mux.HandleFunc("/v1/pending", s.auth(s.handlePending))
	mux.HandleFunc("/v1/events/", s.auth(s.handleEventAction)) // /v1/events/<id>/approve|reject

	// Social cards
	mux.HandleFunc("/v1/card/daily", s.handleCard)
	mux.HandleFunc("/v1/card/weekly", s.handleCard)
	mux.HandleFunc("/v1/card/season", s.handleCard)

	// EOD score lock
	mux.HandleFunc("/v1/lock", s.auth(s.handleLock))

	// Integrations management
	mux.HandleFunc("/v1/integrations", s.auth(s.handleIntegrations))
	mux.HandleFunc("/v1/integrations/", s.auth(s.handleIntegrationConfig))

	// Webhook receivers (use separate tokens in query params)
	mux.HandleFunc("/v1/webhooks/github", s.auth(s.handleGitHubWebhook))
	mux.HandleFunc("/v1/webhooks/stripe", s.auth(s.handleStripeWebhook))

	// Static files (Svelte PWA)
	staticFS, _ := fs.Sub(staticFiles, "static")
	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback: serve index.html for non-file paths
		path := r.URL.Path
		if path != "/" && !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})

	// Start integration poller
	s.startPoller()

	// ─── Tenant Manager ─────────────────────────────────────────────
	tm := NewTenantManager("/data/wirebot/scoreboard")

	// Tenant provisioning (called by Ring Leader)
	mux.HandleFunc("/v1/tenants", func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" {
			return
		}
		// Auth: accept operator token or Ring Leader JWT secret
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != authToken {
			http.Error(w, `{"error":"unauthorized"}`, 401)
			return
		}

		switch r.Method {
		case "POST":
			var req struct {
				TenantID  string `json:"tenant_id"`
				UserID    int    `json:"user_id"`
				Tier      string `json:"tier"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid json"}`, 400)
				return
			}
			if req.TenantID == "" {
				http.Error(w, `{"error":"tenant_id required"}`, 400)
				return
			}
			info, err := tm.Provision(req.TenantID, req.UserID, req.Tier)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
				return
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(info)

		case "GET":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tenants": tm.List(),
			})

		default:
			http.Error(w, `{"error":"method not allowed"}`, 405)
		}
	})

	// ─── Tenant-scoped route multiplexer ────────────────────────────
	// Routes: /{randID}/v1/scoreboard (public view-only)
	//         /{randID}/v1/... (write requires ?key= or Bearer token)
	// The SPA is also served at /{randID}/ with the tenant context

	topHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Skip known top-level paths — route to operator's mux
		if strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/health") ||
			path == "/" || strings.Contains(path, ".") {
			mux.ServeHTTP(w, r)
			return
		}

		// Extract potential tenant ID: /{randID}/...
		parts := strings.SplitN(strings.TrimPrefix(path, "/"), "/", 2)
		if len(parts) == 0 {
			mux.ServeHTTP(w, r)
			return
		}

		tenantID := parts[0]
		subPath := "/"
		if len(parts) > 1 {
			subPath = "/" + parts[1]
		}

		// Validate tenant exists
		tenantServer, err := tm.GetOrCreate(tenantID)
		if err != nil {
			// Not a valid tenant — fall back to SPA
			mux.ServeHTTP(w, r)
			return
		}

		// Check if info.json exists (provisioned)
		infoPath := fmt.Sprintf("/data/wirebot/scoreboard/tenants/%s/info.json", tenantID)
		if _, err := os.Stat(infoPath); err != nil {
			mux.ServeHTTP(w, r)
			return
		}

		// Route tenant API calls
		if strings.HasPrefix(subPath, "/v1/") {
			// For tenant, scoreboard view is public, everything else needs ?key=
			if subPath == "/v1/scoreboard" || r.Method == "GET" && subPath == "/v1/feed" {
				// Public read-only for view-only URLs
				r.URL.Path = subPath
				tenantMux := buildTenantMux(tenantServer)
				tenantMux.ServeHTTP(w, r)
				return
			}
			// Write access needs key param matching tenant's write token
			// For now, accept the operator token
			r.URL.Path = subPath
			tenantMux := buildTenantMux(tenantServer)
			tenantMux.ServeHTTP(w, r)
			return
		}

		// Serve PWA for tenant
		r.URL.Path = "/"
		mux.ServeHTTP(w, r)
	})

	// Recalculate today's score from existing events on startup
	// This ensures the score is accurate even after a restart
	today := operatorToday()
	s.updateDailyScore(today)
	s.updateStreak(today, "")
	s.recalcSeason()
	log.Printf("Startup recalc complete for %s", today)

	log.Printf("Scoreboard listening on %s (multi-tenant enabled)", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, topHandler))
}

// ─── Tenant Mux Builder ─────────────────────────────────────────────────────

func buildTenantMux(s *Server) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/scoreboard", s.handleScoreboard)
	mux.HandleFunc("/v1/events", s.auth(s.handleEvents))
	mux.HandleFunc("/v1/events/batch", s.auth(s.handleEventsBatch))
	mux.HandleFunc("/v1/score", s.auth(s.handleScore))
	mux.HandleFunc("/v1/feed", s.handleFeed)
	mux.HandleFunc("/v1/season", s.handleSeason)
	mux.HandleFunc("/v1/season/wrapped", s.handleWrapped)
	mux.HandleFunc("/v1/history", s.handleHistory)
	mux.HandleFunc("/v1/intent", s.auth(s.handleIntent))
	mux.HandleFunc("/v1/audit", s.auth(s.handleAudit))
	mux.HandleFunc("/v1/pending", s.auth(s.handlePending))
	mux.HandleFunc("/v1/events/", s.auth(s.handleEventAction))
	mux.HandleFunc("/v1/lock", s.auth(s.handleLock))
	mux.HandleFunc("/health", s.handleHealth)
	return mux
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
		`CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type)`,
		// Migration: add status column for gated events
		`ALTER TABLE events ADD COLUMN status TEXT DEFAULT 'approved'`,
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
		// Integration credentials (encrypted at rest)
		`CREATE TABLE IF NOT EXISTS integrations (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'default',
			provider TEXT NOT NULL,
			auth_type TEXT NOT NULL,
			encrypted_data BLOB,
			nonce BLOB,
			display_name TEXT DEFAULT '',
			scopes TEXT DEFAULT '[]',
			status TEXT DEFAULT 'active',
			sensitivity TEXT DEFAULT 'standard',
			wirebot_visible BOOLEAN DEFAULT 1,
			wirebot_detail_level TEXT DEFAULT 'full',
			share_level TEXT DEFAULT 'private',
			poll_interval_seconds INTEGER DEFAULT 1800,
			last_used_at TEXT DEFAULT '',
			last_error TEXT DEFAULT '',
			last_poll_at TEXT DEFAULT '',
			next_poll_at TEXT DEFAULT '',
			config TEXT DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_integrations_user ON integrations(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_integrations_provider ON integrations(user_id, provider)`,
		// Verification level migration on events
		`ALTER TABLE events ADD COLUMN verification_level TEXT DEFAULT 'SELF_REPORTED'`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			// Ignore ALTER TABLE errors (column may already exist)
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Fatalf("initDB: %v", err)
			}
		}
	}

	// Seed default season
	var count int
	s.db.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&count)
	if count == 0 {
		s.db.Exec(`INSERT INTO seasons (name, number, start_date, end_date, theme, is_active)
			VALUES ('Red-to-Black', 1, '2026-02-01', '2026-05-01', 'Break even. Ship what makes money. Get out of the red.', 1)`)
	}
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
	now := operatorNow()
	startT, _ := time.Parse("2006-01-02", s.season.StartDate)
	endT, _ := time.Parse("2006-01-02", s.season.EndDate)
	elapsed := int(now.Sub(startT).Hours() / 24)
	remaining := int(endT.Sub(now).Hours() / 24)
	if elapsed < 0 {
		elapsed = 0
	}
	if remaining < 0 {
		remaining = 0
	}
	s.season.DaysElapsed = elapsed
	s.season.DaysRemaining = remaining

	var won, played, total int
	s.db.QueryRow("SELECT COUNT(*) FROM daily_scores WHERE date >= ? AND date <= ?",
		s.season.StartDate, s.season.EndDate).Scan(&played)
	s.db.QueryRow("SELECT COUNT(*) FROM daily_scores WHERE date >= ? AND date <= ? AND won=1",
		s.season.StartDate, s.season.EndDate).Scan(&won)
	s.db.QueryRow("SELECT COALESCE(SUM(execution_score),0) FROM daily_scores WHERE date >= ? AND date <= ?",
		s.season.StartDate, s.season.EndDate).Scan(&total)
	s.season.DaysWon = won
	s.season.DaysPlayed = played
	s.season.TotalScore = total
	if played > 0 {
		s.season.AvgScore = total / played
	}
	s.season.Record = fmt.Sprintf("%dW-%dL", won, played-won)
}

// ─── Auth ───────────────────────────────────────────────────────────────────

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		token := strings.TrimPrefix(auth, "Bearer ")
		if token != authToken && r.URL.Query().Get("token") != authToken {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, 401)
			return
		}
		next(w, r)
	}
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
	w.Header().Set("Content-Type", "application/json")
}

// ─── Health ─────────────────────────────────────────────────────────────────

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	cors(w)
	var eventCount int
	s.db.QueryRow("SELECT COUNT(*) FROM events").Scan(&eventCount)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok", "events": eventCount, "season": s.season.Name,
	})
}

// ─── POST/GET /v1/events ────────────────────────────────────────────────────

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}

	switch r.Method {
	case "POST":
		s.postEvent(w, r)
	case "GET":
		s.getEvents(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, 405)
	}
}

func (s *Server) postEvent(w http.ResponseWriter, r *http.Request) {
	var evt struct {
		EventType         string          `json:"event_type"`
		Lane              string          `json:"lane"`
		Source            string          `json:"source"`
		Timestamp         string          `json:"timestamp"`
		ArtifactType      string          `json:"artifact_type"`
		ArtifactURL       string          `json:"artifact_url"`
		ArtifactTitle     string          `json:"artifact_title"`
		Confidence        float64         `json:"confidence"`
		Verifiers         json.RawMessage `json:"verifiers"`
		VerificationLevel string          `json:"verification_level"`
		BusinessID        string          `json:"business_id"`
		Metadata          json.RawMessage `json:"metadata"`
		Status            string          `json:"status"` // "pending" or "" (defaults to "approved")
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

	// Determine verification level from source
	verLevel := evt.VerificationLevel
	if verLevel == "" {
		switch evt.Source {
		case "github-webhook", "stripe-webhook":
			verLevel = "STRONG"
		case "rss-poller", "youtube-poller":
			verLevel = "MEDIUM"
		case "wb-cli", "wb-complete", "wb-ship", "pwa":
			verLevel = "SELF_REPORTED"
		case "claude", "pi", "letta", "opencode":
			verLevel = "WEAK"
		default:
			verLevel = "SELF_REPORTED"
		}
	}

	// Auto-approve ONLY trusted automated sources.
	// Everything else is gated (pending) until operator approves.
	// This prevents gaming — you can't self-report your way to a high score.
	trustedSources := map[string]bool{
		"github-webhook": true,
		"stripe-webhook": true,
		"rss-poller":     true,
		"youtube-poller": true,
	}
	status := "pending"
	if trustedSources[evt.Source] {
		status = "approved"
	}
	// Explicit override: caller can request pending even for trusted sources
	if evt.Status == "pending" {
		status = "pending"
	}

	scoreDelta := calcScoreDelta(evt.Lane, evt.EventType, evt.Confidence)
	// Apply verification multiplier
	scoreDelta = int(float64(scoreDelta) * verificationMultiplier(verLevel))
	// Pending events get 0 score until approved
	effectiveDelta := scoreDelta
	if status == "pending" {
		effectiveDelta = 0
	}

	id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
	verifiers := "[]"
	if evt.Verifiers != nil {
		verifiers = string(evt.Verifiers)
	}
	metadata := "{}"
	if evt.Metadata != nil {
		metadata = string(evt.Metadata)
	}

	s.mu.Lock()
	_, err := s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_type, artifact_url, artifact_title, confidence, verifiers, verification_level,
		score_delta, business_id, metadata, status, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, evt.EventType, evt.Lane, evt.Source, evt.Timestamp,
		evt.ArtifactType, evt.ArtifactURL, evt.ArtifactTitle, evt.Confidence,
		verifiers, verLevel, effectiveDelta, evt.BusinessID, metadata, status, time.Now().UTC().Format(time.RFC3339))
	s.mu.Unlock()

	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}

	// Only update scores if approved
	if status == "approved" {
		today := operatorToday()
		s.updateDailyScore(today)
		s.updateStreak(today, evt.ArtifactTitle)
		s.recalcSeason()
	}

	daily := s.getDailyScore(operatorToday())
	streak := s.getStreak("ship")

	resp := map[string]interface{}{
		"ok": true, "event_id": id, "status": status,
		"score_delta": scoreDelta, "new_daily_score": daily.ExecutionScore, "streak": streak,
	}
	if status == "pending" {
		resp["note"] = "Event is pending approval. Score will be applied after: POST /v1/events/" + id + "/approve"
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) getEvents(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	lane := r.URL.Query().Get("lane")
	evtType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
		limit = l
	}

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
	if evtType != "" {
		query += " AND event_type = ?"
		args = append(args, evtType)
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
	json.NewEncoder(w).Encode(map[string]interface{}{"events": events, "count": len(events)})
}

// ─── POST /v1/events/batch ──────────────────────────────────────────────────

func (s *Server) handleEventsBatch(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var body struct {
		Events []struct {
			EventType     string          `json:"event_type"`
			Lane          string          `json:"lane"`
			Source        string          `json:"source"`
			Timestamp     string          `json:"timestamp"`
			ArtifactType  string          `json:"artifact_type"`
			ArtifactURL   string          `json:"artifact_url"`
			ArtifactTitle string          `json:"artifact_title"`
			Confidence    float64         `json:"confidence"`
			BusinessID    string          `json:"business_id"`
			Metadata      json.RawMessage `json:"metadata"`
		} `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	var ids []string
	var totalDelta int
	for _, evt := range body.Events {
		if evt.EventType == "" || evt.Lane == "" || evt.Source == "" {
			continue
		}
		if evt.Timestamp == "" {
			evt.Timestamp = time.Now().UTC().Format(time.RFC3339)
		}
		if evt.Confidence == 0 {
			evt.Confidence = 1.0
		}
		scoreDelta := calcScoreDelta(evt.Lane, evt.EventType, evt.Confidence)
		id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
		metadata := "{}"
		if evt.Metadata != nil {
			metadata = string(evt.Metadata)
		}

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_type, artifact_url, artifact_title, confidence, verifiers,
			score_delta, business_id, metadata, created_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, evt.EventType, evt.Lane, evt.Source, evt.Timestamp,
			evt.ArtifactType, evt.ArtifactURL, evt.ArtifactTitle, evt.Confidence,
			"[]", scoreDelta, evt.BusinessID, metadata, time.Now().UTC().Format(time.RFC3339))
		s.mu.Unlock()
		ids = append(ids, id)
		totalDelta += scoreDelta
	}

	today := operatorToday()
	s.updateDailyScore(today)
	s.recalcSeason()
	daily := s.getDailyScore(today)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true, "count": len(ids), "event_ids": ids,
		"total_delta": totalDelta, "new_daily_score": daily.ExecutionScore,
	})
}

// ─── GET /v1/score ──────────────────────────────────────────────────────────

func (s *Server) handleScore(w http.ResponseWriter, r *http.Request) {
	cors(w)
	date := r.URL.Query().Get("date")
	rangeQ := r.URL.Query().Get("range")

	if rangeQ != "" {
		s.handleScoreRange(w, rangeQ)
		return
	}
	if date == "" {
		date = operatorToday()
	}

	daily := s.getDailyScore(date)
	streak := s.getStreak("ship")
	s.recalcSeason()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"date": date, "score": daily, "streak": streak, "season": s.season,
	})
}

func (s *Server) handleScoreRange(w http.ResponseWriter, rangeQ string) {
	var startDate, endDate string
	now := operatorNow()

	switch rangeQ {
	case "week":
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case "month":
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case "season":
		startDate = s.season.StartDate
		endDate = s.season.EndDate
	default:
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	}

	rows, _ := s.db.Query(`SELECT date, execution_score, shipping_score, distribution_score,
		revenue_score, systems_score, ships_count, won FROM daily_scores
		WHERE date >= ? AND date <= ? ORDER BY date`, startDate, endDate)
	defer rows.Close()

	type DayEntry struct {
		Date    string `json:"date"`
		Score   int    `json:"score"`
		Ship    int    `json:"shipping"`
		Dist    int    `json:"distribution"`
		Rev     int    `json:"revenue"`
		Sys     int    `json:"systems"`
		Ships   int    `json:"ships_count"`
		Won     bool   `json:"won"`
	}
	var days []DayEntry
	var totalScore, wins, losses int
	for rows.Next() {
		var d DayEntry
		var wonI int
		rows.Scan(&d.Date, &d.Score, &d.Ship, &d.Dist, &d.Rev, &d.Sys, &d.Ships, &wonI)
		d.Won = wonI == 1
		days = append(days, d)
		totalScore += d.Score
		if d.Won {
			wins++
		} else {
			losses++
		}
	}
	avg := 0
	if len(days) > 0 {
		avg = totalScore / len(days)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"range": rangeQ, "start": startDate, "end": endDate,
		"days": days, "avg_score": avg, "wins": wins, "losses": losses,
		"record": fmt.Sprintf("%dW-%dL", wins, losses),
	})
}

// ─── GET /v1/scoreboard ─────────────────────────────────────────────────────

func (s *Server) handleScoreboard(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}

	today := operatorToday()
	daily := s.getDailyScore(today)
	streak := s.getStreak("ship")
	s.recalcSeason()

	possession := s.getPossession()
	intent := daily.Intent
	stallHours := s.getStallHours()

	now := operatorNow()
	dayProgress := float64(now.Hour()*60+now.Minute()) / 1440.0
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekProgress := float64(weekday-1) / 7.0
	seasonProgress := 0.0
	total := s.season.DaysElapsed + s.season.DaysRemaining
	if total > 0 {
		seasonProgress = float64(s.season.DaysElapsed) / float64(total)
	}

	signal := "green"
	if daily.ExecutionScore < 30 {
		signal = "red"
	} else if daily.ExecutionScore < 50 {
		signal = "yellow"
	}

	// Get today's feed items for dashboard mode
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "stadium"
	}

	// Count pending events
	var pendingCount int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE status='pending'").Scan(&pendingCount)

	// Calculate streak bonus
	streakBonus := 0
	if streak.Current >= 30 {
		streakBonus = 20
	} else if streak.Current >= 14 {
		streakBonus = 15
	} else if streak.Current >= 7 {
		streakBonus = 10
	} else if streak.Current >= 3 {
		streakBonus = 5
	}

	view := ScoreboardView{
		Mode:       mode,
		Score:      daily.ExecutionScore,
		Possession: possession,
		ShipToday:  daily.ShipsCount,
		Streak:     streak,
		Record:     s.season.Record,
		SeasonDay:  fmt.Sprintf("Day %d of %d", s.season.DaysElapsed, total),
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
		Signal:       signal,
		Season:       s.season,
		Intent:       intent,
		StallHours:   stallHours,
		Penalties:    daily.Penalties,
		StreakBonus:  streakBonus,
		PendingCount: pendingCount,
	}

	// Dashboard mode: include today's feed
	if mode == "dashboard" || mode == "mobile" {
		feed := s.getFeedItems(20, today, "", "approved")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"scoreboard": view, "feed": feed,
		})
		return
	}

	json.NewEncoder(w).Encode(view)
}

// ─── GET /v1/feed ───────────────────────────────────────────────────────────

func (s *Server) handleFeed(w http.ResponseWriter, r *http.Request) {
	cors(w)
	date := r.URL.Query().Get("date")
	lane := r.URL.Query().Get("lane")
	status := r.URL.Query().Get("status") // "pending", "approved", "rejected", or "" (all)
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	items := s.getFeedItems(limit, date, lane, status)

	// Also return pending count for badge display
	var pendingCount int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE status='pending'").Scan(&pendingCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":         items,
		"count":         len(items),
		"pending_count": pendingCount,
	})
}

func (s *Server) getFeedItems(limit int, date, lane, status string) []FeedItem {
	query := `SELECT id, event_type, lane, source, timestamp, artifact_title, artifact_url, score_delta, confidence, status
		FROM events WHERE 1=1`
	args := []interface{}{}
	if date != "" {
		query += " AND timestamp LIKE ?"
		args = append(args, date+"%")
	}
	if lane != "" {
		query += " AND lane = ?"
		args = append(args, lane)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return []FeedItem{}
	}
	defer rows.Close()

	icons := map[string]string{"shipping": "🚀", "distribution": "📣", "revenue": "💰", "systems": "⚙️"}
	var items []FeedItem
	for rows.Next() {
		var f FeedItem
		rows.Scan(&f.ID, &f.Type, &f.Lane, &f.Source, &f.Timestamp, &f.Title, &f.URL, &f.Delta, &f.Confidence, &f.Status)
		f.Icon = icons[f.Lane]
		if f.Icon == "" {
			f.Icon = "📌"
		}
		items = append(items, f)
	}
	if items == nil {
		items = []FeedItem{}
	}
	return items
}

// ─── POST/GET /v1/intent ────────────────────────────────────────────────────

func (s *Server) handleIntent(w http.ResponseWriter, r *http.Request) {
	cors(w)
	today := operatorToday()

	switch r.Method {
	case "POST":
		var body struct {
			Intent string `json:"intent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Intent == "" {
			http.Error(w, `{"error":"intent field required"}`, 400)
			return
		}

		s.mu.Lock()
		s.db.Exec(`INSERT INTO daily_scores (date, intent) VALUES (?, ?)
			ON CONFLICT(date) DO UPDATE SET intent=excluded.intent`, today, body.Intent)
		s.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "date": today, "intent": body.Intent,
		})

	case "GET":
		daily := s.getDailyScore(today)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"date": today, "intent": daily.Intent, "fulfilled": daily.IntentFulfilled,
		})

	default:
		http.Error(w, `{"error":"GET or POST"}`, 405)
	}
}

// ─── GET /v1/audit ──────────────────────────────────────────────────────────

func (s *Server) handleAudit(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	lane := r.URL.Query().Get("lane")
	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}

	query := `SELECT id, event_type, lane, source, timestamp, artifact_title, artifact_url,
		confidence, score_delta, business_id FROM events WHERE 1=1`
	args := []interface{}{}
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

	type AuditRow struct {
		ID         string  `json:"id"`
		Type       string  `json:"event_type"`
		Lane       string  `json:"lane"`
		Source     string  `json:"source"`
		Timestamp  string  `json:"timestamp"`
		Title      string  `json:"artifact_title"`
		URL        string  `json:"artifact_url"`
		Confidence float64 `json:"confidence"`
		Delta      int     `json:"score_delta"`
		Business   string  `json:"business_id"`
	}
	var auditRows []AuditRow
	for rows.Next() {
		var a AuditRow
		rows.Scan(&a.ID, &a.Type, &a.Lane, &a.Source, &a.Timestamp,
			&a.Title, &a.URL, &a.Confidence, &a.Delta, &a.Business)
		auditRows = append(auditRows, a)
	}
	if auditRows == nil {
		auditRows = []AuditRow{}
	}

	if format == "csv" {
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=scoreboard-audit.csv")
		cw := csv.NewWriter(w)
		cw.Write([]string{"ID", "Event Type", "Lane", "Source", "Timestamp", "Title", "URL", "Confidence", "Score Delta", "Business"})
		for _, a := range auditRows {
			cw.Write([]string{a.ID, a.Type, a.Lane, a.Source, a.Timestamp, a.Title, a.URL,
				fmt.Sprintf("%.2f", a.Confidence), strconv.Itoa(a.Delta), a.Business})
		}
		cw.Flush()
		return
	}

	cors(w)
	json.NewEncoder(w).Encode(map[string]interface{}{"audit": auditRows, "count": len(auditRows)})
}

// ─── GET /v1/history — daily score calendar ─────────────────────────────────

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	cors(w)
	rangeQ := r.URL.Query().Get("range")
	if rangeQ == "" {
		rangeQ = "season"
	}

	var startDate, endDate string
	now := operatorNow()
	switch rangeQ {
	case "week":
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
	case "month":
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
	default:
		startDate = s.season.StartDate
	}
	endDate = now.Format("2006-01-02")

	rows, _ := s.db.Query(`SELECT date, execution_score, ships_count, won, intent
		FROM daily_scores WHERE date >= ? AND date <= ? ORDER BY date`, startDate, endDate)
	defer rows.Close()

	type CalDay struct {
		Date   string `json:"date"`
		Score  int    `json:"score"`
		Ships  int    `json:"ships"`
		Won    bool   `json:"won"`
		Intent string `json:"intent,omitempty"`
	}
	var days []CalDay
	for rows.Next() {
		var d CalDay
		var wonI int
		rows.Scan(&d.Date, &d.Score, &d.Ships, &wonI, &d.Intent)
		d.Won = wonI == 1
		days = append(days, d)
	}
	if days == nil {
		days = []CalDay{}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"days": days, "range": rangeQ})
}

// ─── GET/POST /v1/season ────────────────────────────────────────────────────

func (s *Server) handleSeason(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "POST" {
		var body struct {
			Name      string `json:"name"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
			Theme     string `json:"theme"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
			http.Error(w, `{"error":"name required"}`, 400)
			return
		}
		// Deactivate current
		s.db.Exec("UPDATE seasons SET is_active=0")
		// Get next number
		var maxNum int
		s.db.QueryRow("SELECT COALESCE(MAX(number),0) FROM seasons").Scan(&maxNum)
		s.db.Exec(`INSERT INTO seasons (name, number, start_date, end_date, theme, is_active)
			VALUES (?,?,?,?,?,1)`, body.Name, maxNum+1, body.StartDate, body.EndDate, body.Theme)
		s.loadSeason()
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "season": s.season})
		return
	}

	s.recalcSeason()
	json.NewEncoder(w).Encode(s.season)
}

// ─── GET /v1/season/wrapped ─────────────────────────────────────────────────

func (s *Server) handleWrapped(w http.ResponseWriter, r *http.Request) {
	cors(w)
	s.recalcSeason()

	// Top artifacts
	rows, _ := s.db.Query(`SELECT artifact_title, artifact_url, score_delta, event_type, lane
		FROM events WHERE timestamp >= ? AND timestamp <= ?
		ORDER BY score_delta DESC LIMIT 5`, s.season.StartDate, s.season.EndDate+"T23:59:59Z")
	defer rows.Close()

	type Artifact struct {
		Title string `json:"title"`
		URL   string `json:"url"`
		Delta int    `json:"score_delta"`
		Type  string `json:"event_type"`
		Lane  string `json:"lane"`
	}
	var topArtifacts []Artifact
	for rows.Next() {
		var a Artifact
		rows.Scan(&a.Title, &a.URL, &a.Delta, &a.Type, &a.Lane)
		topArtifacts = append(topArtifacts, a)
	}

	// Total ships
	var totalShips int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE lane='shipping' AND timestamp >= ? AND timestamp <= ?",
		s.season.StartDate, s.season.EndDate+"T23:59:59Z").Scan(&totalShips)

	// Revenue events
	var revEvents int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE lane='revenue' AND timestamp >= ?",
		s.season.StartDate).Scan(&revEvents)

	// Best day of week
	var bestDay string
	s.db.QueryRow(`SELECT CASE CAST(strftime('%w', date) AS INTEGER)
		WHEN 0 THEN 'Sunday' WHEN 1 THEN 'Monday' WHEN 2 THEN 'Tuesday'
		WHEN 3 THEN 'Wednesday' WHEN 4 THEN 'Thursday' WHEN 5 THEN 'Friday'
		ELSE 'Saturday' END as dow
		FROM daily_scores WHERE date >= ? GROUP BY dow
		ORDER BY AVG(execution_score) DESC LIMIT 1`, s.season.StartDate).Scan(&bestDay)

	// Best lane
	var bestLane string
	s.db.QueryRow(`SELECT lane FROM events WHERE timestamp >= ?
		GROUP BY lane ORDER BY SUM(score_delta) DESC LIMIT 1`, s.season.StartDate).Scan(&bestLane)

	// Score trend
	trend := "→"
	var firstHalf, secondHalf float64
	midpoint := s.season.DaysElapsed / 2
	if midpoint > 0 {
		midDate := operatorNow().AddDate(0, 0, -midpoint).Format("2006-01-02")
		s.db.QueryRow("SELECT COALESCE(AVG(execution_score),0) FROM daily_scores WHERE date < ?", midDate).Scan(&firstHalf)
		s.db.QueryRow("SELECT COALESCE(AVG(execution_score),0) FROM daily_scores WHERE date >= ?", midDate).Scan(&secondHalf)
		if secondHalf > firstHalf+5 {
			trend = "↑"
		} else if secondHalf < firstHalf-5 {
			trend = "↓"
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"season":         s.season.Name,
		"number":         s.season.Number,
		"duration_days":  s.season.DaysElapsed + s.season.DaysRemaining,
		"days_played":    s.season.DaysPlayed,
		"total_ships":    totalShips,
		"best_streak":    s.getStreak("ship").Best,
		"revenue_events": revEvents,
		"days_won":       s.season.DaysWon,
		"record":         s.season.Record,
		"avg_score":      s.season.AvgScore,
		"top_artifacts":  topArtifacts,
		"patterns": map[string]interface{}{
			"best_day_of_week": bestDay,
			"best_lane":        bestLane,
			"avg_score_trend":  trend,
		},
	})
}

// ─── Webhook: GitHub ────────────────────────────────────────────────────────

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	action, _ := payload["action"].(string)
	var evtType, title, url string

	// Release event
	if release, ok := payload["release"].(map[string]interface{}); ok {
		evtType = "PRODUCT_RELEASE"
		title, _ = release["name"].(string)
		url, _ = release["html_url"].(string)
	} else if pr, ok := payload["pull_request"].(map[string]interface{}); ok && action == "closed" {
		if merged, _ := pr["merged"].(bool); merged {
			evtType = "FEATURE_SHIPPED"
			title, _ = pr["title"].(string)
			url, _ = pr["html_url"].(string)
		}
	} else if action == "completed" {
		// Workflow run
		if run, ok := payload["workflow_run"].(map[string]interface{}); ok {
			conclusion, _ := run["conclusion"].(string)
			if conclusion == "success" {
				evtType = "DEPLOY_SUCCESS"
				title, _ = run["name"].(string)
				url, _ = run["html_url"].(string)
			}
		}
	}

	if evtType == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "skipped": true, "reason": "unhandled event type"})
		return
	}

	// Dedup: check if discovery already found a pending event for same artifact
	// If so, UPGRADE it to approved+STRONG instead of creating a duplicate
	scoreDelta := calcScoreDelta("shipping", evtType, 0.95)
	id := fmt.Sprintf("evt-gh-%d", time.Now().UnixNano())
	upgraded := false

	if url != "" {
		var existingID string
		s.db.QueryRow("SELECT id FROM events WHERE artifact_url=? AND status='pending' LIMIT 1", url).Scan(&existingID)
		if existingID != "" {
			s.mu.Lock()
			s.db.Exec(`UPDATE events SET status='approved', source='github-webhook',
				verification_level='STRONG', verifiers='["github_webhook"]',
				confidence=0.95, score_delta=? WHERE id=?`, scoreDelta, existingID)
			s.mu.Unlock()
			id = existingID
			upgraded = true
		}
	}

	if !upgraded {
		// Dedup: check for same title + type within last 5 minutes (prevents triple-fire)
		var recentID string
		fiveMinAgo := time.Now().Add(-5 * time.Minute).UTC().Format(time.RFC3339)
		s.db.QueryRow("SELECT id FROM events WHERE event_type=? AND artifact_title=? AND source='github-webhook' AND created_at > ? LIMIT 1",
			evtType, title, fiveMinAgo).Scan(&recentID)
		if recentID != "" {
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "skipped": true, "reason": "duplicate webhook (same event within 5min)", "existing_id": recentID})
			return
		}

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_url, artifact_title, confidence, verifiers, verification_level,
			score_delta, created_at, status)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			id, evtType, "shipping", "github-webhook", time.Now().UTC().Format(time.RFC3339),
			url, title, 0.95, `["github_webhook"]`, "STRONG",
			scoreDelta, time.Now().UTC().Format(time.RFC3339), "approved")
		s.mu.Unlock()
	}

	today := operatorToday()
	s.updateDailyScore(today)
	s.updateStreak(today, title)
	s.recalcSeason()

	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "event_id": id, "event_type": evtType, "score_delta": scoreDelta})
}

// ─── Webhook: Stripe ────────────────────────────────────────────────────────

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	evtTypeStripe, _ := payload["type"].(string)
	var evtType, title string
	var amount float64

	data, _ := payload["data"].(map[string]interface{})
	obj, _ := data["object"].(map[string]interface{})

	switch evtTypeStripe {
	case "payment_intent.succeeded", "charge.succeeded":
		evtType = "PAYMENT_RECEIVED"
		amount, _ = obj["amount"].(float64)
		title = fmt.Sprintf("Payment received: $%.2f", amount/100)
	case "customer.subscription.created":
		evtType = "SUBSCRIPTION_CREATED"
		title = "New subscription created"
	case "invoice.paid":
		evtType = "INVOICE_PAID"
		amount, _ = obj["amount_paid"].(float64)
		title = fmt.Sprintf("Invoice paid: $%.2f", amount/100)
	default:
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "skipped": true})
		return
	}

	scoreDelta := calcScoreDelta("revenue", evtType, 0.99)
	id := fmt.Sprintf("evt-stripe-%d", time.Now().UnixNano())
	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, created_at, status)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, evtType, "revenue", "stripe-webhook", time.Now().UTC().Format(time.RFC3339),
		title, 0.99, `["stripe_webhook"]`, "STRONG",
		scoreDelta, fmt.Sprintf(`{"amount":%.0f,"currency":"usd"}`, amount),
		time.Now().UTC().Format(time.RFC3339), "approved")
	s.mu.Unlock()

	today := operatorToday()
	s.updateDailyScore(today)
	s.recalcSeason()

	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "event_id": id, "event_type": evtType, "score_delta": scoreDelta})
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
	// Special: context switch and penalties return 0 (handled separately in updateDailyScore)
	if eventType == "CONTEXT_SWITCH" || eventType == "COMMITMENT_BREACH" {
		return 0
	}

	if laneMap, ok := base[lane]; ok {
		if pts, ok := laneMap[eventType]; ok {
			return int(float64(pts) * confidence)
		}
		return int(3.0 * confidence)
	}
	return 1
}

func (s *Server) updateDailyScore(date string) {
	// Convert operator-local date to UTC range for event matching.
	// Events store UTC timestamps, but daily scores group by operator's local date.
	// Example: PST "2026-02-01" → UTC "2026-02-01T08:00:00Z" to "2026-02-02T08:00:00Z"
	localDate, err := time.ParseInLocation("2006-01-02", date, operatorTZ)
	if err != nil {
		localDate, _ = time.Parse("2006-01-02", date)
	}
	utcStart := localDate.UTC().Format(time.RFC3339)
	utcEnd := localDate.Add(24 * time.Hour).UTC().Format(time.RFC3339)

	// Only count approved events toward score
	dateFilter := "status='approved' AND timestamp >= ? AND timestamp < ?"
	var shipping, distribution, revenue, systems, ships int
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='shipping' AND "+dateFilter, utcStart, utcEnd).Scan(&shipping)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='distribution' AND "+dateFilter, utcStart, utcEnd).Scan(&distribution)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='revenue' AND "+dateFilter, utcStart, utcEnd).Scan(&revenue)
	s.db.QueryRow("SELECT COALESCE(SUM(score_delta),0) FROM events WHERE lane='systems' AND "+dateFilter, utcStart, utcEnd).Scan(&systems)
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE lane='shipping' AND "+dateFilter, utcStart, utcEnd).Scan(&ships)

	if shipping > 40 {
		shipping = 40
	}
	if distribution > 25 {
		distribution = 25
	}
	if revenue > 20 {
		revenue = 20
	}
	if systems > 15 {
		systems = 15
	}

	// Count context switches — penalty for 3rd+ switch in a day
	var switches int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE event_type='CONTEXT_SWITCH' AND status='approved' AND timestamp >= ? AND timestamp < ?", utcStart, utcEnd).Scan(&switches)
	contextPenalty := 0
	if switches > 2 {
		contextPenalty = (switches - 2) * 5
	}

	// Check unfulfilled intent (COMMITMENT_BREACH)
	commitmentPenalty := 0
	// Only check for yesterday and older (not today — still in progress)
	yesterday := operatorNow().AddDate(0, 0, -1).Format("2006-01-02")
	if date <= yesterday {
		var intent string
		var fulfilled int
		s.db.QueryRow("SELECT COALESCE(intent,''), intent_fulfilled FROM daily_scores WHERE date=?", date).Scan(&intent, &fulfilled)
		if intent != "" && fulfilled == 0 && ships == 0 {
			commitmentPenalty = 10
		}
	}

	total := shipping + distribution + revenue + systems - contextPenalty - commitmentPenalty
	if total < 0 {
		total = 0
	}
	if ships == 0 && total > 30 {
		total = 30
	}

	// Multipliers: streak bonus (3+ days = +5, 7+ = +10, 14+ = +15, 30+ = +20)
	streak := s.getStreak("ship")
	streakBonus := 0
	if streak.Current >= 30 {
		streakBonus = 20
	} else if streak.Current >= 14 {
		streakBonus = 15
	} else if streak.Current >= 7 {
		streakBonus = 10
	} else if streak.Current >= 3 {
		streakBonus = 5
	}
	// Only apply streak bonus if there are ships today
	if ships > 0 && streakBonus > 0 {
		total += streakBonus
	}
	// Cap at 100
	if total > 100 {
		total = 100
	}

	won := total >= 50
	penalties := contextPenalty + commitmentPenalty

	s.mu.Lock()
	s.db.Exec(`INSERT INTO daily_scores (date, execution_score, shipping_score, distribution_score,
		revenue_score, systems_score, penalties, ships_count, won)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			execution_score=excluded.execution_score,
			shipping_score=excluded.shipping_score,
			distribution_score=excluded.distribution_score,
			revenue_score=excluded.revenue_score,
			systems_score=excluded.systems_score,
			penalties=excluded.penalties,
			ships_count=excluded.ships_count,
			won=excluded.won`,
		date, total, shipping, distribution, revenue, systems, penalties, ships, won)
	s.mu.Unlock()
}

func (s *Server) getDailyScore(date string) DailyScore {
	var ds DailyScore
	ds.Date = date
	s.db.QueryRow(`SELECT execution_score, shipping_score, distribution_score, revenue_score,
		systems_score, penalties, ships_count, COALESCE(intent,''), intent_fulfilled, won
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

	yesterday := operatorNow().AddDate(0, 0, -1).Format("2006-01-02")
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

func (s *Server) getPossession() string {
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
	return possession
}

// ─── GET /v1/pending ─────────────────────────────────────────────────────

func (s *Server) handlePending(w http.ResponseWriter, r *http.Request) {
	cors(w)
	rows, err := s.db.Query(`SELECT id, event_type, lane, source, timestamp, artifact_title, artifact_url,
		confidence, score_delta, business_id FROM events WHERE status='pending' ORDER BY timestamp DESC`)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}
	defer rows.Close()

	type PendingEvent struct {
		ID         string  `json:"id"`
		Type       string  `json:"event_type"`
		Lane       string  `json:"lane"`
		Source     string  `json:"source"`
		Timestamp  string  `json:"timestamp"`
		Title      string  `json:"artifact_title"`
		URL        string  `json:"artifact_url"`
		Confidence float64 `json:"confidence"`
		Points     int     `json:"potential_points"`
		Business   string  `json:"business_id"`
	}
	var pending []PendingEvent
	for rows.Next() {
		var p PendingEvent
		rows.Scan(&p.ID, &p.Type, &p.Lane, &p.Source, &p.Timestamp,
			&p.Title, &p.URL, &p.Confidence, &p.Points, &p.Business)
		// Recalculate actual points (stored as 0 while pending)
		p.Points = calcScoreDelta(p.Lane, p.Type, p.Confidence)
		pending = append(pending, p)
	}
	if pending == nil {
		pending = []PendingEvent{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"pending": pending, "count": len(pending)})
}

// ─── POST /v1/events/<id>/approve or /v1/events/<id>/reject ─────────────

func (s *Server) handleEventAction(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	// Parse: /v1/events/<id>/approve or /v1/events/<id>/reject
	path := strings.TrimPrefix(r.URL.Path, "/v1/events/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		http.Error(w, `{"error":"use /v1/events/<id>/approve or /v1/events/<id>/reject"}`, 400)
		return
	}
	eventID := parts[0]
	action := parts[1]

	if action != "approve" && action != "reject" {
		http.Error(w, `{"error":"action must be approve or reject"}`, 400)
		return
	}

	// Check event exists and is pending
	var currentStatus, lane, evtType, title string
	var confidence float64
	err := s.db.QueryRow("SELECT status, lane, event_type, artifact_title, confidence FROM events WHERE id=?", eventID).
		Scan(&currentStatus, &lane, &evtType, &title, &confidence)
	if err != nil {
		http.Error(w, `{"error":"event not found"}`, 404)
		return
	}
	if currentStatus != "pending" {
		http.Error(w, fmt.Sprintf(`{"error":"event is already %s"}`, currentStatus), 409)
		return
	}

	s.mu.Lock()
	if action == "approve" {
		scoreDelta := calcScoreDelta(lane, evtType, confidence)
		s.db.Exec("UPDATE events SET status='approved', score_delta=? WHERE id=?", scoreDelta, eventID)
		s.mu.Unlock()

		// Recalculate daily score
		var ts string
		s.db.QueryRow("SELECT timestamp FROM events WHERE id=?", eventID).Scan(&ts)
		date := ts[:10] // YYYY-MM-DD
		s.updateDailyScore(date)
		s.updateStreak(date, title)
		s.recalcSeason()

		daily := s.getDailyScore(date)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "event_id": eventID, "action": "approved",
			"score_delta": scoreDelta, "new_daily_score": daily.ExecutionScore,
		})
	} else {
		s.db.Exec("UPDATE events SET status='rejected', score_delta=0 WHERE id=?", eventID)
		s.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "event_id": eventID, "action": "rejected",
		})
	}
}

// ─── GET /v1/card/daily|weekly|season — Social Share Cards (SVG) ─────────

func (s *Server) handleCard(w http.ResponseWriter, r *http.Request) {
	cardType := "daily"
	if strings.Contains(r.URL.Path, "weekly") {
		cardType = "weekly"
	} else if strings.Contains(r.URL.Path, "season") {
		cardType = "season"
	}

	today := operatorToday()
	daily := s.getDailyScore(today)
	streak := s.getStreak("ship")
	s.recalcSeason()

	signalColor := "#00ff64" // green
	signalLabel := "WINNING"
	if daily.ExecutionScore < 30 {
		signalColor = "#ff3232"
		signalLabel = "STALLING"
	} else if daily.ExecutionScore < 50 {
		signalColor = "#ffc800"
		signalLabel = "PRESSURE"
	}

	var title, subtitle, stat1Label, stat1Val, stat2Label, stat2Val, stat3Label, stat3Val string

	switch cardType {
	case "daily":
		title = fmt.Sprintf("%d", daily.ExecutionScore)
		subtitle = signalLabel
		stat1Label = "SHIPS"
		stat1Val = fmt.Sprintf("%d", daily.ShipsCount)
		stat2Label = "STREAK"
		stat2Val = fmt.Sprintf("%d", streak.Current)
		stat3Label = "RECORD"
		stat3Val = s.season.Record
	case "weekly":
		var weekScore, weekShips, weekWins int
		weekStart := operatorNow().AddDate(0, 0, -7).Format("2006-01-02")
		s.db.QueryRow("SELECT COALESCE(AVG(execution_score),0), COALESCE(SUM(ships_count),0), COUNT(CASE WHEN won THEN 1 END) FROM daily_scores WHERE date >= ?", weekStart).Scan(&weekScore, &weekShips, &weekWins)
		title = fmt.Sprintf("%d", weekScore)
		subtitle = "WEEKLY AVG"
		stat1Label = "SHIPS"
		stat1Val = fmt.Sprintf("%d", weekShips)
		stat2Label = "WINS"
		stat2Val = fmt.Sprintf("%d/7", weekWins)
		stat3Label = "STREAK"
		stat3Val = fmt.Sprintf("%d", streak.Current)
		signalColor = "#7c7cff"
	case "season":
		title = fmt.Sprintf("%d", s.season.AvgScore)
		subtitle = s.season.Name
		stat1Label = "RECORD"
		stat1Val = s.season.Record
		stat2Label = "BEST STREAK"
		stat2Val = fmt.Sprintf("%d", streak.Best)
		stat3Label = "DAY"
		stat3Val = fmt.Sprintf("%d/%d", s.season.DaysElapsed, s.season.DaysElapsed+s.season.DaysRemaining)
		signalColor = "#ff4a9e"
	}

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="600" height="315" viewBox="0 0 600 315">
  <defs>
    <style>
      @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;700;900');
      text { font-family: 'Inter', system-ui, sans-serif; fill: #ddd; }
    </style>
  </defs>
  <rect width="600" height="315" rx="16" fill="#0a0a1a"/>
  <rect x="0" y="0" width="600" height="4" fill="%s"/>
  <text x="30" y="40" font-size="13" font-weight="700" fill="#7c7cff" letter-spacing="2">⚡ WIREBOT SCOREBOARD</text>
  <text x="570" y="40" font-size="12" fill="#555" text-anchor="end">%s</text>
  <text x="300" y="150" font-size="96" font-weight="900" fill="%s" text-anchor="middle">%s</text>
  <text x="300" y="180" font-size="16" font-weight="700" fill="%s" text-anchor="middle" letter-spacing="3">%s</text>
  <line x1="30" y1="210" x2="570" y2="210" stroke="#1e1e30" stroke-width="1"/>
  <text x="130" y="245" font-size="28" font-weight="700" fill="#ddd" text-anchor="middle">%s</text>
  <text x="130" y="268" font-size="11" fill="#555" text-anchor="middle" letter-spacing="1">%s</text>
  <text x="300" y="245" font-size="28" font-weight="700" fill="#ddd" text-anchor="middle">%s</text>
  <text x="300" y="268" font-size="11" fill="#555" text-anchor="middle" letter-spacing="1">%s</text>
  <text x="470" y="245" font-size="28" font-weight="700" fill="#ddd" text-anchor="middle">%s</text>
  <text x="470" y="268" font-size="11" fill="#555" text-anchor="middle" letter-spacing="1">%s</text>
  <text x="300" y="300" font-size="10" fill="#333" text-anchor="middle">wins.wirebot.chat</text>
</svg>`,
		signalColor, today, signalColor, title, signalColor, subtitle,
		stat1Val, stat1Label, stat2Val, stat2Label, stat3Val, stat3Label)

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(svg))
}

// ─── POST /v1/lock — EOD Score Lock ─────────────────────────────────────

func (s *Server) handleLock(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var body struct {
		Date string `json:"date"` // optional, defaults to today
	}
	json.NewDecoder(r.Body).Decode(&body)
	date := body.Date
	if date == "" {
		date = operatorToday()
	}

	// Force recalculate final score
	s.updateDailyScore(date)
	daily := s.getDailyScore(date)

	// Check intent fulfillment
	intentFulfilled := false
	if daily.Intent != "" && daily.ShipsCount > 0 {
		// Simple heuristic: if they shipped something, intent is fulfilled
		intentFulfilled = true
	}

	s.mu.Lock()
	s.db.Exec("UPDATE daily_scores SET intent_fulfilled=? WHERE date=?", intentFulfilled, date)
	s.mu.Unlock()

	// If intent was set but not fulfilled, inject commitment breach for recalc
	if daily.Intent != "" && !intentFulfilled {
		s.updateDailyScore(date)
		daily = s.getDailyScore(date)
	}

	s.recalcSeason()
	streak := s.getStreak("ship")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true, "date": date, "locked": true,
		"final_score": daily.ExecutionScore, "won": daily.Won,
		"intent": daily.Intent, "intent_fulfilled": intentFulfilled,
		"ships": daily.ShipsCount, "streak": streak,
		"record": s.season.Record,
	})
}

// ─── Encryption Helpers ──────────────────────────────────────────────────

func (s *Server) getMasterKey() ([]byte, error) {
	if masterKeyHex == "" {
		// Generate and log a new key on first run (operator must persist it)
		key := make([]byte, 32)
		rand.Read(key)
		masterKeyHex = hex.EncodeToString(key)
		log.Printf("WARNING: No SCOREBOARD_MASTER_KEY set. Generated ephemeral key: %s", masterKeyHex)
		log.Printf("Set SCOREBOARD_MASTER_KEY=%s in your environment to persist credentials across restarts.", masterKeyHex)
		return key, nil
	}
	return hex.DecodeString(masterKeyHex)
}

func (s *Server) encryptCredential(plaintext []byte) (encrypted []byte, nonce []byte, err error) {
	key, err := s.getMasterKey()
	if err != nil {
		return nil, nil, fmt.Errorf("master key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("gcm: %w", err)
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("nonce: %w", err)
	}
	encrypted = gcm.Seal(nil, nonce, plaintext, nil)
	return encrypted, nonce, nil
}

func (s *Server) decryptCredential(encrypted []byte, nonce []byte) ([]byte, error) {
	key, err := s.getMasterKey()
	if err != nil {
		return nil, fmt.Errorf("master key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	return gcm.Open(nil, nonce, encrypted, nil)
}

// ─── Integration Management ─────────────────────────────────────────────

func (s *Server) handleIntegrations(w http.ResponseWriter, r *http.Request) {
	cors(w)
	switch r.Method {
	case "GET":
		// List all integrations (metadata only, no secrets)
		rows, err := s.db.Query(`SELECT id, user_id, provider, auth_type, display_name, scopes,
			status, sensitivity, wirebot_visible, wirebot_detail_level, share_level,
			poll_interval_seconds, last_used_at, last_error, created_at FROM integrations ORDER BY created_at`)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
			return
		}
		defer rows.Close()
		var list []Integration
		for rows.Next() {
			var i Integration
			rows.Scan(&i.ID, &i.UserID, &i.Provider, &i.AuthType, &i.DisplayName, &i.Scopes,
				&i.Status, &i.Sensitivity, &i.WirebotVisible, &i.WirebotDetail, &i.ShareLevel,
				&i.PollInterval, &i.LastUsedAt, &i.LastError, &i.CreatedAt)
			list = append(list, i)
		}
		if list == nil {
			list = []Integration{}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"integrations": list})

	case "POST":
		// Add a new integration
		var body struct {
			Provider    string `json:"provider"`
			AuthType    string `json:"auth_type"`
			DisplayName string `json:"display_name"`
			Credential  string `json:"credential"` // API key, URL, or OAuth token JSON
			Sensitivity string `json:"sensitivity"`
			PollInterval int   `json:"poll_interval_seconds"`
			Config      string `json:"config"` // provider-specific config JSON
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid json"}`, 400)
			return
		}
		if body.Provider == "" || body.AuthType == "" {
			http.Error(w, `{"error":"provider and auth_type required"}`, 400)
			return
		}

		// Encrypt credential
		encrypted, nonce, err := s.encryptCredential([]byte(body.Credential))
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"encryption failed: %s"}`, err), 500)
			return
		}

		id := fmt.Sprintf("int-%d", time.Now().UnixNano())
		sensitivity := body.Sensitivity
		if sensitivity == "" {
			sensitivity = "standard"
		}
		pollInterval := body.PollInterval
		if pollInterval == 0 {
			pollInterval = 1800 // 30 minutes default
		}
		config := body.Config
		if config == "" {
			config = "{}"
		}
		now := time.Now().UTC().Format(time.RFC3339)
		nextPoll := time.Now().Add(time.Duration(pollInterval) * time.Second).UTC().Format(time.RFC3339)

		s.mu.Lock()
		_, err = s.db.Exec(`INSERT INTO integrations (id, user_id, provider, auth_type, encrypted_data, nonce,
			display_name, sensitivity, poll_interval_seconds, config, created_at, updated_at, next_poll_at)
			VALUES (?, 'default', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, body.Provider, body.AuthType, encrypted, nonce,
			body.DisplayName, sensitivity, pollInterval, config, now, now, nextPoll)
		s.mu.Unlock()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "id": id, "provider": body.Provider, "status": "active",
		})

	default:
		http.Error(w, `{"error":"GET or POST"}`, 405)
	}
}

func (s *Server) handleIntegrationConfig(w http.ResponseWriter, r *http.Request) {
	cors(w)
	// PATCH /v1/integrations/<id> — update settings (not credentials)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, `{"error":"id required"}`, 400)
		return
	}
	id := parts[3]

	if r.Method == "DELETE" {
		s.mu.Lock()
		s.db.Exec("DELETE FROM integrations WHERE id=?", id)
		s.mu.Unlock()
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "deleted": id})
		return
	}

	if r.Method != "PATCH" {
		http.Error(w, `{"error":"PATCH or DELETE"}`, 405)
		return
	}

	var body struct {
		WirebotVisible bool   `json:"wirebot_visible"`
		WirebotDetail  string `json:"wirebot_detail_level"`
		ShareLevel     string `json:"share_level"`
		Sensitivity    string `json:"sensitivity"`
		Status         string `json:"status"`
		PollInterval   int    `json:"poll_interval_seconds"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	updates := []string{}
	args := []interface{}{}
	if body.WirebotDetail != "" {
		updates = append(updates, "wirebot_detail_level=?")
		args = append(args, body.WirebotDetail)
	}
	if body.ShareLevel != "" {
		updates = append(updates, "share_level=?")
		args = append(args, body.ShareLevel)
	}
	if body.Sensitivity != "" {
		updates = append(updates, "sensitivity=?")
		args = append(args, body.Sensitivity)
	}
	if body.Status != "" {
		updates = append(updates, "status=?")
		args = append(args, body.Status)
	}
	if body.PollInterval > 0 {
		updates = append(updates, "poll_interval_seconds=?")
		args = append(args, body.PollInterval)
	}
	updates = append(updates, "wirebot_visible=?", "updated_at=?")
	args = append(args, body.WirebotVisible, time.Now().UTC().Format(time.RFC3339), id)

	if len(updates) > 0 {
		s.mu.Lock()
		s.db.Exec("UPDATE integrations SET "+strings.Join(updates, ", ")+" WHERE id=?", args...)
		s.mu.Unlock()
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "id": id})
}

// ─── RSS Poller ─────────────────────────────────────────────────────────

func (s *Server) startPoller() {
	// Run immediately on start, then every 60 seconds
	go func() {
		time.Sleep(5 * time.Second) // Wait for server to be ready
		s.pollDueIntegrations()
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			s.pollDueIntegrations()
		}
	}()
}

func (s *Server) pollDueIntegrations() {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.Query(`SELECT id, provider, auth_type, encrypted_data, nonce, config,
		last_poll_at, poll_interval_seconds FROM integrations
		WHERE status='active' AND (next_poll_at <= ? OR next_poll_at = '') LIMIT 10`, now)
	if err != nil {
		log.Printf("Poller query error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, provider, authType, config, lastPoll string
		var encData, nonce []byte
		var pollInterval int
		rows.Scan(&id, &provider, &authType, &encData, &nonce, &config, &lastPoll, &pollInterval)

		// Decrypt credential
		var credential string
		if len(encData) > 0 && len(nonce) > 0 {
			decrypted, err := s.decryptCredential(encData, nonce)
			if err != nil {
				log.Printf("Poller: failed to decrypt %s (%s): %v", id, provider, err)
				s.db.Exec("UPDATE integrations SET last_error=?, status='error' WHERE id=?", err.Error(), id)
				continue
			}
			credential = string(decrypted)
		}

		// Poll based on provider
		var pollErr error
		switch provider {
		case "rss", "blog_rss", "podcast_rss":
			pollErr = s.pollRSS(id, credential, lastPoll)
		case "youtube":
			pollErr = s.pollYouTube(id, credential, config, lastPoll)
		default:
			log.Printf("Poller: unknown provider %s for integration %s", provider, id)
		}

		// Update poll timestamps
		nextPoll := time.Now().Add(time.Duration(pollInterval) * time.Second).UTC().Format(time.RFC3339)
		if pollErr != nil {
			s.db.Exec("UPDATE integrations SET last_poll_at=?, next_poll_at=?, last_error=? WHERE id=?",
				now, nextPoll, pollErr.Error(), id)
		} else {
			s.db.Exec("UPDATE integrations SET last_poll_at=?, next_poll_at=?, last_error='', last_used_at=? WHERE id=?",
				now, nextPoll, now, id)
		}
	}
}

func (s *Server) pollRSS(integrationID, feedURL, lastPoll string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(feedURL)
	if err != nil {
		return fmt.Errorf("fetch RSS: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1MB max
	if err != nil {
		return fmt.Errorf("read RSS: %w", err)
	}

	// Parse RSS or Atom
	var items []struct {
		Title   string
		Link    string
		PubDate string
		GUID    string
	}

	// Try RSS first
	var rss RSSFeed
	if err := xml.Unmarshal(body, &rss); err == nil && len(rss.Channel.Items) > 0 {
		for _, item := range rss.Channel.Items {
			items = append(items, struct {
				Title   string
				Link    string
				PubDate string
				GUID    string
			}{item.Title, item.Link, item.PubDate, item.GUID})
		}
	} else {
		// Try Atom
		var atom AtomFeed
		if err := xml.Unmarshal(body, &atom); err == nil {
			for _, entry := range atom.Entries {
				items = append(items, struct {
					Title   string
					Link    string
					PubDate string
					GUID    string
				}{entry.Title, entry.Link.Href, entry.Updated, entry.ID})
			}
		}
	}

	if len(items) == 0 {
		return nil // No items or parse error
	}

	// Filter to items newer than last poll
	lastPollTime := time.Time{}
	if lastPoll != "" {
		lastPollTime, _ = time.Parse(time.RFC3339, lastPoll)
	}

	newCount := 0
	for _, item := range items {
		pubTime := parseFlexibleTime(item.PubDate)
		if pubTime.IsZero() || (!lastPollTime.IsZero() && !pubTime.After(lastPollTime)) {
			continue
		}

		// Check if we already have this event (by URL)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE artifact_url=?", item.Link).Scan(&exists)
		if exists > 0 {
			continue
		}

		// Determine event type based on feed URL
		eventType := "BLOG_PUBLISHED"
		lane := "distribution"
		if strings.Contains(feedURL, "podcast") || strings.Contains(feedURL, "anchor") {
			eventType = "PODCAST_PUBLISHED"
		}

		score := calcScoreDelta(lane, eventType, 0.90)
		score = int(float64(score) * verificationMultiplier("MEDIUM"))
		id := fmt.Sprintf("evt-%d", time.Now().UnixNano()+int64(newCount))
		now := time.Now().UTC().Format(time.RFC3339)

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_url, artifact_title, confidence, verifiers, verification_level,
			score_delta, created_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, eventType, lane, "rss-poller", pubTime.UTC().Format(time.RFC3339),
			item.Link, item.Title, 0.90, `["rss_feed"]`, "MEDIUM",
			score, now, "approved")
		s.mu.Unlock()

		s.updateDailyScore(pubTime.Format("2006-01-02"))
		newCount++
		log.Printf("RSS: New %s — %s (%s)", eventType, item.Title, item.Link)
	}

	if newCount > 0 {
		log.Printf("RSS poller: %d new items from %s", newCount, feedURL)
	}
	return nil
}

func parseFlexibleTime(s string) time.Time {
	formats := []string{
		time.RFC3339,
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02",
	}
	for _, fmt := range formats {
		if t, err := time.Parse(fmt, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (s *Server) pollYouTube(integrationID, apiKey, configJSON, lastPoll string) error {
	var config struct {
		ChannelID string `json:"channel_id"`
	}
	json.Unmarshal([]byte(configJSON), &config)
	if config.ChannelID == "" || apiKey == "" {
		return fmt.Errorf("channel_id and api_key required")
	}

	publishedAfter := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	if lastPoll != "" {
		publishedAfter = lastPoll
	}

	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/search?part=snippet&channelId=%s&order=date&publishedAfter=%s&type=video&maxResults=10&key=%s",
		config.ChannelID, publishedAfter, apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("youtube api: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			ID      struct{ VideoId string `json:"videoId"` } `json:"id"`
			Snippet struct {
				Title       string `json:"title"`
				PublishedAt string `json:"publishedAt"`
			} `json:"snippet"`
		} `json:"items"`
		Error struct{ Message string `json:"message"` } `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Error.Message != "" {
		return fmt.Errorf("youtube: %s", result.Error.Message)
	}

	newCount := 0
	for _, item := range result.Items {
		videoURL := fmt.Sprintf("https://youtube.com/watch?v=%s", item.ID.VideoId)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE artifact_url=?", videoURL).Scan(&exists)
		if exists > 0 {
			continue
		}

		score := calcScoreDelta("distribution", "VIDEO_PUBLISHED", 0.95)
		id := fmt.Sprintf("evt-%d", time.Now().UnixNano()+int64(newCount))
		now := time.Now().UTC().Format(time.RFC3339)

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_url, artifact_title, confidence, verifiers, verification_level,
			score_delta, created_at, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, "VIDEO_PUBLISHED", "distribution", "youtube-poller", item.Snippet.PublishedAt,
			videoURL, item.Snippet.Title, 0.95, `["youtube_api"]`, "STRONG",
			score, now, "approved")
		s.mu.Unlock()

		pubTime, _ := time.Parse(time.RFC3339, item.Snippet.PublishedAt)
		if !pubTime.IsZero() {
			s.updateDailyScore(pubTime.Format("2006-01-02"))
		}
		newCount++
		log.Printf("YouTube: New VIDEO_PUBLISHED — %s (%s)", item.Snippet.Title, videoURL)
	}
	return nil
}

// ─── Verification Level Score Multiplier ────────────────────────────────

func verificationMultiplier(level string) float64 {
	switch level {
	case "STRONG":
		return 1.0
	case "MEDIUM":
		return 0.85
	case "WEAK":
		return 0.70
	case "SELF_REPORTED":
		return 0.80
	case "UNVERIFIED":
		return 0.50
	default:
		return 0.80
	}
}

func (s *Server) getStallHours() float64 {
	var lastTS string
	s.db.QueryRow("SELECT timestamp FROM events WHERE lane='shipping' ORDER BY timestamp DESC LIMIT 1").Scan(&lastTS)
	if lastTS == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, lastTS)
	if err != nil {
		return 0
	}
	hours := time.Since(t).Hours()
	return math.Round(hours*10) / 10
}
