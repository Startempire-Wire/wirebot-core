package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
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
	"regexp"
	"sort"
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
	rlJWTSecret    = envOr("RL_JWT_SECRET", "")         // Ring Leader JWT secret (HMAC-SHA256)
	stripeKey      = envOr("STRIPE_SECRET_KEY", "")    // Stripe live secret key
	stripeWHSecret = envOr("STRIPE_WEBHOOK_SECRET", "") // Stripe webhook signing secret
	plaidClientID  = envOr("PLAID_CLIENT_ID", "")       // Plaid client_id
	plaidSecret    = envOr("PLAID_SECRET", "")           // Plaid secret (sandbox/development/production)
	plaidEnv       = envOr("PLAID_ENV", "sandbox")       // sandbox | development | production
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
	Detail            string  `json:"detail,omitempty"`
	Confidence        float64 `json:"confidence"`
	Verification      string  `json:"verification,omitempty"`     // PROVIDER_API, WEBHOOK, SELF_REPORTED, etc.
	Verifiers         string  `json:"verifiers,omitempty"`
	VerificationLevel string  `json:"verification_level,omitempty"` // STRONG, MEDIUM, WEAK, SELF_REPORTED, UNVERIFIED
	ScoreDelta        int     `json:"score_delta"`
	BusinessID        string  `json:"business_id,omitempty"`
	ExternalID        string  `json:"external_id,omitempty"` // provider-specific ID for dedup
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
	BusinessID       string `json:"business_id,omitempty"`
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
	BusinessID string  `json:"business_id,omitempty"`
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
	pairing  *PairingEngine // Living profile engine (pairing.go)
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

	// Each tenant gets their own pairing engine with isolated profile
	// No Letta/Mem0/Gateway by default — tenant configures their own memory stack
	profilePath := tenantDir + "/profile.json"
	os.MkdirAll(tenantDir, 0750)
	s.pairing = NewPairingEngine(profilePath, db, PairingConfig{})
	s.pairing.Start()

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

	// Initialize and start the Pairing Engine
	os.MkdirAll("/data/wirebot/pairing", 0750)
	s.pairing = NewPairingEngine("/data/wirebot/pairing/profile.json", s.db, PairingConfig{
		LettaAgentID: envOr("LETTA_AGENT_ID", "agent-82610d14-ec65-4d10-9ec2-8c479848cea9"),
		LettaURL:     envOr("LETTA_URL", "http://localhost:8283"),
		Mem0Namespace: envOr("MEM0_NAMESPACE", "wirebot_verious"),
		Mem0URL:       envOr("MEM0_URL", "http://localhost:8200"),
		GatewayToken:  envOr("GATEWAY_TOKEN", authToken),
		GatewayURL:    envOr("GATEWAY_URL", "http://127.0.0.1:18789"),
	})
	s.pairing.Start()

	mux := http.NewServeMux()

	// Public endpoints
	// Health (always public)
	mux.HandleFunc("/health", s.handleHealth)

	// ALL data endpoints require authentication — no public data when logged out
	mux.HandleFunc("/v1/scoreboard", s.authMember(s.handleScoreboard))
	mux.HandleFunc("/v1/events", s.auth(s.handleEvents))
	mux.HandleFunc("/v1/events/batch", s.auth(s.handleEventsBatch))
	mux.HandleFunc("/v1/score", s.auth(s.handleScore))
	mux.HandleFunc("/v1/feed", s.authMember(s.handleFeed))
	mux.HandleFunc("/v1/season", s.authMember(s.handleSeason))
	mux.HandleFunc("/v1/season/wrapped", s.authMember(s.handleWrapped))
	mux.HandleFunc("/v1/history", s.authMember(s.handleHistory))
	mux.HandleFunc("/v1/intent", s.auth(s.handleIntent))
	mux.HandleFunc("/v1/audit", s.auth(s.handleAudit))

	// Gated events (pending/approve/reject)
	mux.HandleFunc("/v1/pending", s.auth(s.handlePending))
	mux.HandleFunc("/v1/events/", s.auth(s.handleEventAction)) // /v1/events/<id>/approve|reject

	// Project-level approval
	mux.HandleFunc("/v1/projects", s.authMember(s.handleProjects))
	mux.HandleFunc("/v1/projects/", s.auth(s.handleProjectAction)) // POST .../approve|reject

	// Social share cards — intentionally public (OG embeds for Twitter/Discord/etc)
	// Cards show minimal info (score + streak) and only when user explicitly shares a link
	mux.HandleFunc("/v1/card/daily", s.handleCard)
	mux.HandleFunc("/v1/card/weekly", s.handleCard)
	mux.HandleFunc("/v1/card/season", s.handleCard)

	// EOD score lock
	mux.HandleFunc("/v1/lock", s.auth(s.handleLock))

	// Wirebot chat proxy — full conversations with memory retention
	mux.HandleFunc("/v1/chat", s.auth(s.handleChat))
	mux.HandleFunc("/v1/chat/sessions", s.auth(s.handleChatSessions))
	mux.HandleFunc("/v1/chat/sessions/", s.auth(s.handleChatSession))
	mux.HandleFunc("/v1/pairing/status", s.authMember(s.handlePairingStatus))
	s.registerPairingRoutes(mux)

	// Integrations management
	mux.HandleFunc("/v1/integrations", s.auth(s.handleIntegrations))
	mux.HandleFunc("/v1/integrations/", s.auth(s.handleIntegrationConfig))
	mux.HandleFunc("/v1/network/members", s.auth(s.handleNetworkMembers)) // Real members from startempirewire.com
	mux.HandleFunc("/v1/oauth/config", s.auth(s.handleOAuthConfig))       // GET=status, POST=store credentials
	mux.HandleFunc("/v1/oauth/setup/github", s.auth(s.handleGitHubSetup)) // Manifest flow: redirect to GitHub
	mux.HandleFunc("/v1/oauth/setup/github/callback", s.handleGitHubSetupCallback) // GitHub returns here with code
	mux.HandleFunc("/v1/oauth/setup/stripe", s.auth(s.handleStripeSetup))
	mux.HandleFunc("/v1/oauth/setup/freshbooks", s.auth(s.handleFreshBooksSetup))
	mux.HandleFunc("/v1/oauth/setup/hubspot", s.auth(s.handleHubSpotSetup))

	// Webhook receivers (use their own verification, not bearer auth)
	mux.HandleFunc("/v1/webhooks/github", s.auth(s.handleGitHubWebhook))
	mux.HandleFunc("/v1/webhooks/stripe", s.handleStripeWebhook) // Stripe signs its own webhooks
	mux.HandleFunc("/v1/financial/snapshot", s.auth(s.handleFinancialSnapshot))

	// Transaction reconciliation
	mux.HandleFunc("/v1/reconcile", s.auth(s.handleReconcile))
	mux.HandleFunc("/v1/reconcile/test-transactions", s.auth(s.handleTestTransactions))

	// Plaid (bank account connections)
	mux.HandleFunc("/v1/plaid/link-token", s.auth(s.handlePlaidLinkToken))
	mux.HandleFunc("/v1/plaid/exchange", s.auth(s.handlePlaidExchange))

	// OAuth flows (provider authorization + callbacks)
	mux.HandleFunc("/v1/oauth/stripe/authorize", s.auth(s.handleOAuthStart))
	mux.HandleFunc("/v1/oauth/github/authorize", s.auth(s.handleOAuthStart))
	mux.HandleFunc("/v1/oauth/google/authorize", s.auth(s.handleOAuthStart))
	mux.HandleFunc("/v1/oauth/freshbooks/authorize", s.auth(s.handleOAuthStart))
	mux.HandleFunc("/v1/oauth/hubspot/authorize", s.auth(s.handleOAuthStart))
	mux.HandleFunc("/v1/oauth/callback", s.handleOAuthCallback) // Provider redirects back here

	// Checklist data for Dashboard view
	mux.HandleFunc("/v1/checklist", s.auth(s.handleChecklist))

	// SSO callback — receives JWT from Connect Plugin redirect
	mux.HandleFunc("/auth/callback", s.handleSSOCallback)

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
	mux.HandleFunc("/v1/projects", s.handleProjects)
	mux.HandleFunc("/v1/projects/", s.auth(s.handleProjectAction))
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
		// Migrations
		`ALTER TABLE events ADD COLUMN status TEXT DEFAULT 'approved'`,
		`ALTER TABLE events ADD COLUMN verification_level TEXT DEFAULT 'SELF_REPORTED'`,
		`ALTER TABLE events ADD COLUMN external_id TEXT DEFAULT ''`,
		`ALTER TABLE events ADD COLUMN detail TEXT DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_events_external ON events(external_id)`,
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
			business_id TEXT DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_integrations_user ON integrations(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_integrations_provider ON integrations(user_id, provider)`,
		// Verification level migration on events
		`ALTER TABLE events ADD COLUMN verification_level TEXT DEFAULT 'SELF_REPORTED'`,
		// Project-level approval: approve/reject entire repos, remembered across sessions
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL DEFAULT 'operator',
			title TEXT NOT NULL DEFAULT 'New Chat',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			message_count INTEGER DEFAULT 0,
			pinned BOOLEAN DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (session_id) REFERENCES chat_sessions(id)
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			name TEXT PRIMARY KEY,
			path TEXT NOT NULL DEFAULT '',
			business TEXT NOT NULL DEFAULT '',
			github TEXT DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			auto_approve BOOLEAN DEFAULT 0,
			approved_at TEXT DEFAULT '',
			total_events INTEGER DEFAULT 0,
			approved_events INTEGER DEFAULT 0,
			rejected_events INTEGER DEFAULT 0,
			notes TEXT DEFAULT '',
			created_at TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS trusted_sources (
			source TEXT PRIMARY KEY,
			approved_at TEXT NOT NULL,
			approved_count INTEGER DEFAULT 1,
			notes TEXT DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS pairing_evidence (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			signal_type TEXT NOT NULL,
			source TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			features TEXT DEFAULT '{}',
			profile_impact TEXT DEFAULT '{}',
			constructs TEXT DEFAULT '[]',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			// Ignore ALTER TABLE errors (column may already exist)
			if !strings.Contains(err.Error(), "duplicate column") {
				log.Fatalf("initDB: %v", err)
			}
		}
	}

	// Init reconciliation tables
	s.initReconciliation()

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

// ─── Auth Per bigpicture.mdx ─────────────────────────────────────────────────
// Auth flow (faithful to blueprint):
//   ├─ WordPress Admin / Operator Token? → Allow All (tier_level=99)
//   ├─ Valid Ring Leader JWT? → Trust tier assignment (tier_level 0-3)
//   ├─ Valid API Key? → Apply tier limits
//   └─ No Auth → Free tier (tier_level=0, read-only)
//
// Tier levels per TRUST_MODES.md:
//   0 = Free (Mode 0: demo, no write)
//   1 = FreeWire (Mode 1: core skills)
//   2 = Wire (Mode 2: extended)
//   3 = ExtraWire (Mode 3: sovereign)
//  99 = Operator (admin override)

type AuthContext struct {
	Authenticated bool
	UserID        int
	Username      string
	Email         string
	Tier          string // "free", "freewire", "wire", "extrawire", "operator"
	TierLevel     int    // 0-3, or 99 for operator
	IsAdmin       bool   // WordPress administrator per bigpicture.mdx
	Roles         []string
}

func contextKey(key string) string { return "auth." + key }

// resolveAuth extracts auth context from request without rejecting.
func resolveAuth(r *http.Request) AuthContext {
	token := ""
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token = strings.TrimPrefix(auth, "Bearer ")
	}
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	if token == "" {
		token = r.URL.Query().Get("key")
	}

	// 1. Operator token — full admin access
	if token != "" && token == authToken {
		return AuthContext{Authenticated: true, Tier: "operator", TierLevel: 99}
	}

	// 2. Ring Leader JWT — verify HMAC-SHA256 signature
	if token != "" && rlJWTSecret != "" {
		if ac, ok := verifyRingLeaderJWT(token); ok {
			return ac
		}
	}

	// 3. No auth — free tier (read-only public)
	return AuthContext{Authenticated: false, Tier: "free", TierLevel: 0}
}

// verifyRingLeaderJWT verifies a Ring Leader JWT (HMAC-SHA256).
// JWT payload: { iss, iat, exp, data: { user_id, email, tier, tier_level } }
func verifyRingLeaderJWT(token string) (AuthContext, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return AuthContext{}, false
	}

	headerPayload := parts[0] + "." + parts[1]
	expectedSig := base64URLEncode(hmacSHA256([]byte(headerPayload), []byte(rlJWTSecret)))

	if !hmacEqual(expectedSig, parts[2]) {
		return AuthContext{}, false
	}

	// Decode payload
	payloadBytes, err := base64URLDecode(parts[1])
	if err != nil {
		return AuthContext{}, false
	}

	var payload struct {
		Exp  int64 `json:"exp"`
		Data struct {
			UserID    int      `json:"user_id"`
			Username  string   `json:"username"`
			Email     string   `json:"email"`
			Tier      string   `json:"tier"`
			TierLevel int      `json:"tier_level"`
			IsAdmin   bool     `json:"is_admin"`
			Roles     []string `json:"roles"`
		} `json:"data"`
	}
	if json.Unmarshal(payloadBytes, &payload) != nil {
		return AuthContext{}, false
	}

	// Check expiry
	if payload.Exp > 0 && payload.Exp < time.Now().Unix() {
		return AuthContext{}, false
	}

	tier := payload.Data.Tier
	if tier == "" {
		tier = "free"
	}
	tierLevel := payload.Data.TierLevel

	// Per bigpicture.mdx: "WordPress Admin? → Allow All Access"
	// Admin gets operator-equivalent tier level
	if payload.Data.IsAdmin {
		tierLevel = 99
	}

	return AuthContext{
		Authenticated: true,
		UserID:        payload.Data.UserID,
		Username:      payload.Data.Username,
		Email:         payload.Data.Email,
		Tier:          tier,
		TierLevel:     tierLevel,
		IsAdmin:       payload.Data.IsAdmin,
		Roles:         payload.Data.Roles,
	}, true
}

func hmacSHA256(data, key []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func base64URLEncode(data []byte) string {
	s := base64.RawURLEncoding.EncodeToString(data)
	return s
}

func base64URLDecode(s string) ([]byte, error) {
	// Add padding if needed
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	return base64.URLEncoding.DecodeString(s)
}

func hmacEqual(a, b string) bool {
	// Constant-time comparison
	if len(a) != len(b) {
		return false
	}
	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}
	return result == 0
}

// auth requires operator-level access (admin token or extrawire tier).
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" { return } // CORS preflight passthrough
		ac := resolveAuth(r)
		if !ac.Authenticated || ac.TierLevel < 3 {
			http.Error(w, `{"error":"unauthorized","hint":"Requires operator token or ExtraWire+ Ring Leader JWT"}`, 401)
			return
		}
		next(w, r)
	}
}

// authMember requires any authenticated user (any tier).
func (s *Server) authMember(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cors(w)
		if r.Method == "OPTIONS" { return } // CORS preflight passthrough
		ac := resolveAuth(r)
		if !ac.Authenticated {
			http.Error(w, `{"error":"unauthorized","hint":"Login via startempirewire.com or provide Ring Leader JWT"}`, 401)
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
		ScoreDelta        int             `json:"score_delta"`
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

	// Events start pending UNLESS their source has been approved before.
	// Approve once → trusted forever.
	status := "pending"
	var trustedCount int
	s.db.QueryRow(`SELECT approved_count FROM trusted_sources WHERE source=?`, evt.Source).Scan(&trustedCount)
	if trustedCount > 0 {
		status = "approved"
	}

	// git-discovery: check if the project is approved (auto-approve)
	if evt.Source == "git-discovery" && evt.Status == "approved" {
		// Discovery engine already checked project approval status
		// Verify: project must exist in projects table as approved
		var projStatus string
		var repo string
		if evt.Metadata != nil {
			var meta map[string]interface{}
			json.Unmarshal(evt.Metadata, &meta)
			if r, ok := meta["repo"].(string); ok {
				repo = r
			}
		}
		if repo != "" {
			s.db.QueryRow("SELECT status FROM projects WHERE name=? AND auto_approve=1", repo).Scan(&projStatus)
			if projStatus == "approved" {
				status = "approved"
			}
		}
	}

	// Explicit override: caller can request pending even for trusted sources
	if evt.Status == "pending" {
		status = "pending"
	}

	scoreDelta := evt.ScoreDelta
	if scoreDelta == 0 {
		// No explicit score — calculate from event type
		scoreDelta = calcScoreDelta(evt.Lane, evt.EventType, evt.Confidence)
	}
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

	// Feed event to pairing engine
	if s.pairing != nil {
		s.pairing.Ingest(Signal{
			Type:      SignalEvent,
			Source:    evt.Source,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"event_type": evt.EventType,
				"lane":       evt.Lane,
				"project":    evt.ArtifactTitle,
				"status":     status,
			},
		})
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

	// Update live Drift Score
	if s.pairing != nil {
		s.pairing.UpdateDrift(s.db)
	}

	// Include Drift in response
	var drift interface{}
	if s.pairing != nil {
		s.pairing.mu.RLock()
		drift = s.pairing.profile.Drift
		s.pairing.mu.RUnlock()
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"date": date, "score": daily, "streak": streak, "season": s.season,
		"drift": drift,
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

	// Dashboard mode: include today's feed + checklist summary
	if mode == "dashboard" || mode == "mobile" {
		feed := s.getFeedItems(20, today, "", "approved", "")
		resp := map[string]interface{}{
			"scoreboard": view, "feed": feed,
		}
		// Add checklist summary if available
		if data, err := os.ReadFile(checklistPath); err == nil {
			var cl map[string]interface{}
			if json.Unmarshal(data, &cl) == nil {
				tasks, _ := cl["tasks"].([]interface{})
				total := len(tasks)
				completed := 0
				nextTask := ""
				for _, t := range tasks {
					tm, _ := t.(map[string]interface{})
					status, _ := tm["status"].(string)
					if status == "completed" || status == "done" {
						completed++
					} else if nextTask == "" && status != "skipped" {
						nextTask, _ = tm["title"].(string)
					}
				}
				pct := 0
				if total > 0 {
					pct = completed * 100 / total
				}
				resp["checklist"] = map[string]interface{}{
					"total":     total,
					"completed": completed,
					"percent":   pct,
					"next_task": nextTask,
				}
			}
		}
		json.NewEncoder(w).Encode(resp)
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
	bizFilter := r.URL.Query().Get("business")
	limitStr := r.URL.Query().Get("limit")
	limit := 30
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
		limit = l
	}

	items := s.getFeedItems(limit, date, lane, status, bizFilter)

	// Business counts for filter pills
	bizCounts := map[string]int{}
	bcRows, _ := s.db.Query("SELECT business_id, COUNT(*) FROM events WHERE status='approved' GROUP BY business_id")
	if bcRows != nil {
		defer bcRows.Close()
		for bcRows.Next() {
			var bid string
			var cnt int
			bcRows.Scan(&bid, &cnt)
			if bid != "" {
				bizCounts[bid] = cnt
			}
		}
	}

	// Also return pending count for badge display
	var pendingCount int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE status='pending'").Scan(&pendingCount)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"items":           items,
		"count":           len(items),
		"pending_count":   pendingCount,
		"business_counts": bizCounts,
	})
}

func (s *Server) getFeedItems(limit int, date, lane, status, businessID string) []FeedItem {
	query := `SELECT id, event_type, lane, source, timestamp, artifact_title, artifact_url, score_delta, confidence, status, business_id
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
	if businessID != "" {
		query += " AND business_id = ?"
		args = append(args, businessID)
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
		rows.Scan(&f.ID, &f.Type, &f.Lane, &f.Source, &f.Timestamp, &f.Title, &f.URL, &f.Delta, &f.Confidence, &f.Status, &f.BusinessID)
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

// ─── Project Inference ──────────────────────────────────────────────────────
// Infer project name from whatever signals an event has: metadata, title, URL, source.
// This prevents "unknown" buckets when metadata is missing or sparse.

// inferProject determines the project name for an event using all available signals.
func inferProject(metadata, title, artifactURL, source string) string {
	// 1. Best: explicit repo in metadata
	if metadata != "" && metadata != "{}" {
		var meta map[string]interface{}
		if json.Unmarshal([]byte(metadata), &meta) == nil {
			if repo, ok := meta["repo"].(string); ok && repo != "" {
				return repo
			}
		}
	}

	// 2. Title prefix pattern: "[project-name] commit message"
	if len(title) > 2 && title[0] == '[' {
		end := strings.Index(title, "]")
		if end > 1 {
			return title[1:end]
		}
	}

	// 3. GitHub URL patterns
	if strings.Contains(artifactURL, "github.com/") {
		// https://github.com/Org/Repo/commit/sha or /releases/tag/v1
		parts := strings.Split(artifactURL, "/")
		for i, p := range parts {
			if p == "github.com" && i+2 < len(parts) {
				repo := parts[i+2] // Org/Repo — take repo name
				// Clean common suffixes
				repo = strings.TrimSuffix(repo, ".git")
				// Map known GitHub repos to short names
				return inferGitHubShortName(parts[i+1], repo)
			}
		}
	}

	// 4. URL domain patterns
	if strings.Contains(artifactURL, "startempirewire.com") {
		return "startempirewire.com"
	}
	if strings.Contains(artifactURL, "startempirewire.network") {
		return "startempirewire.network"
	}
	if strings.Contains(artifactURL, "wirebot.chat") {
		return "wirebot"
	}

	// 5. Source-based fallback
	switch source {
	case "github-webhook":
		return "github" // generic but better than unknown
	case "rss-poller":
		return "rss-content"
	case "stripe-webhook":
		return "stripe"
	case "youtube-poller":
		return "youtube"
	}

	return "other"
}

// inferGitHubShortName maps GitHub org/repo to a human-friendly project name.
func inferGitHubShortName(org, repo string) string {
	// Known mappings
	knownRepos := map[string]string{
		"wirebot-core":                              "wirebot-core",
		"focusa":                                    "focusa",
		"Startempire-Wire-Network":                  "chrome-extension",
		"Startempire-Wire-Network-Ring-Leader":      "ring-leader",
		"Startempire-Wire-Network-Connect":          "connect-plugin",
		"Startempire-Wire-Network-Parent-Core":      "parent-core",
		"Startempire-Wire-Network-Websockets":       "websockets",
		"Startempire-Wire-Network-Screenshots":      "screenshots",
	}
	if short, ok := knownRepos[repo]; ok {
		return short
	}
	// Fallback: use repo name as-is (lowercase)
	return strings.ToLower(repo)
}

// ─── Project-Level Approval ──────────────────────────────────────────────────
// Projects group events by repo. Approving a project bulk-approves all its
// pending events AND auto-approves future discoveries for that project.

func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	cors(w)

	switch r.Method {
	case "GET":
		// List projects using inferProject for intelligent grouping.
		// Scans ALL events so webhooks, RSS, etc. also group properly.
		rows, err := s.db.Query(`SELECT metadata, artifact_title, artifact_url, source, status, lane FROM events`)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
			return
		}
		defer rows.Close()

		type ProjectView struct {
			Name        string   `json:"name"`
			Path        string   `json:"path"`
			Business    string   `json:"business"`
			GitHub      string   `json:"github"`
			Status      string   `json:"status"`
			AutoApprove bool     `json:"auto_approve"`
			Total       int      `json:"total_events"`
			Pending     int      `json:"pending"`
			Approved    int      `json:"approved"`
			Rejected    int      `json:"rejected"`
			Lane        string   `json:"primary_lane"`
			Sources     []string `json:"sources,omitempty"`
		}

		projMap := map[string]*ProjectView{}
		srcMap := map[string]map[string]bool{}

		for rows.Next() {
			var metadata, title, url, source, status, lane string
			rows.Scan(&metadata, &title, &url, &source, &status, &lane)

			projName := inferProject(metadata, title, url, source)

			pv, exists := projMap[projName]
			if !exists {
				pv = &ProjectView{Name: projName, Lane: lane}
				projMap[projName] = pv
				srcMap[projName] = map[string]bool{}
			}
			pv.Total++
			srcMap[projName][source] = true
			switch status {
			case "pending":
				pv.Pending++
			case "approved":
				pv.Approved++
			case "rejected":
				pv.Rejected++
			}
		}

		// Enrich from projects table + build final list
		var projects []ProjectView
		for name, pv := range projMap {
			var dbStatus string
			var autoApprove bool
			var business, github, path string
			err := s.db.QueryRow("SELECT status, auto_approve, business, github, path FROM projects WHERE name=?", name).Scan(&dbStatus, &autoApprove, &business, &github, &path)
			if err == nil {
				pv.Status = dbStatus
				pv.AutoApprove = autoApprove
				pv.Business = business
				pv.GitHub = github
				pv.Path = path
			} else {
				if pv.Pending > 0 {
					pv.Status = "pending"
				} else if pv.Approved > 0 {
					pv.Status = "inferred"
				}
			}
			var sources []string
			for src := range srcMap[name] {
				sources = append(sources, src)
			}
			pv.Sources = sources
			projects = append(projects, *pv)
		}

		sort.Slice(projects, func(i, j int) bool {
			if projects[i].Pending != projects[j].Pending {
				return projects[i].Pending > projects[j].Pending
			}
			return projects[i].Total > projects[j].Total
		})

		var pendingCount int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE status='pending'").Scan(&pendingCount)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"projects":      projects,
			"count":         len(projects),
			"pending_total": pendingCount,
		})

	case "OPTIONS":
		w.WriteHeader(200)

	default:
		http.Error(w, `{"error":"GET only"}`, 405)
	}
}

func (s *Server) handleProjectAction(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	// Parse: /v1/projects/{name}/approve or /v1/projects/{name}/reject
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/projects/"), "/")
	if len(parts) < 2 {
		http.Error(w, `{"error":"use /v1/projects/{name}/approve or /reject"}`, 400)
		return
	}
	projectName := parts[0]
	action := parts[1]

	// ── Rename ──
	if action == "rename" {
		var body struct {
			NewName string `json:"new_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NewName == "" {
			http.Error(w, `{"error":"new_name required"}`, 400)
			return
		}
		now := time.Now().UTC().Format(time.RFC3339)

		s.mu.Lock()
		// Update projects table
		s.db.Exec(`UPDATE projects SET name=? WHERE name=?`, body.NewName, projectName)
		s.db.Exec(`INSERT OR IGNORE INTO projects (name, created_at) VALUES (?, ?)`, body.NewName, now)

		// Find all events that inferProject maps to old name, patch their metadata
		rows, _ := s.db.Query(`SELECT id, metadata, artifact_title, artifact_url, source FROM events`)
		var updates []string
		if rows != nil {
			for rows.Next() {
				var id, metadata, title, url, source string
				rows.Scan(&id, &metadata, &title, &url, &source)
				if inferProject(metadata, title, url, source) == projectName {
					updates = append(updates, id)
				}
			}
			rows.Close()
		}
		for _, id := range updates {
			var existing string
			s.db.QueryRow("SELECT metadata FROM events WHERE id=?", id).Scan(&existing)
			var meta map[string]interface{}
			if json.Unmarshal([]byte(existing), &meta) != nil || meta == nil {
				meta = map[string]interface{}{}
			}
			meta["repo"] = body.NewName
			meta["renamed_from"] = projectName
			newMeta, _ := json.Marshal(meta)
			s.db.Exec("UPDATE events SET metadata=? WHERE id=?", string(newMeta), id)
		}
		s.mu.Unlock()

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "project": projectName, "new_name": body.NewName, "events_updated": len(updates),
		})
		log.Printf("Project renamed: %s → %s (%d events)", projectName, body.NewName, len(updates))
		return
	}

	if action != "approve" && action != "reject" {
		http.Error(w, `{"error":"action must be approve, reject, or rename"}`, 400)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	newStatus := "approved"
	if action == "reject" {
		newStatus = "rejected"
	}

	// Upsert project status
	s.mu.Lock()
	s.db.Exec(`INSERT INTO projects (name, status, auto_approve, approved_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET status=excluded.status, auto_approve=excluded.auto_approve, approved_at=excluded.approved_at`,
		projectName, newStatus, action == "approve", now, now)

	// Bulk update all pending events for this project
	result, err := s.db.Exec(`UPDATE events SET status=?, score_delta=CASE WHEN ?='approved' THEN 
		CAST(json_extract(metadata, '$.effective_score') AS INTEGER) ELSE 0 END
		WHERE status='pending' AND json_extract(metadata, '$.repo')=?`,
		newStatus, newStatus, projectName)
	s.mu.Unlock()

	affected := int64(0)
	if err == nil {
		affected, _ = result.RowsAffected()
	}

	// Recalculate daily score
	today := operatorToday()
	s.updateDailyScore(today)
	s.updateStreak(today, "")
	s.recalcSeason()

	// Get updated counts
	var total, pending, approved, rejected int
	s.db.QueryRow(`SELECT COUNT(*), SUM(CASE WHEN status='pending' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status='approved' THEN 1 ELSE 0 END),
		SUM(CASE WHEN status='rejected' THEN 1 ELSE 0 END)
		FROM events WHERE json_extract(metadata, '$.repo')=?`, projectName).Scan(&total, &pending, &approved, &rejected)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":            true,
		"project":       projectName,
		"action":        action,
		"events_affected": affected,
		"total":         total,
		"pending":       pending,
		"approved":      approved,
		"rejected":      rejected,
	})

	log.Printf("Project %s: %s (%d events affected)", projectName, action, affected)
}

// ─── Webhook: Stripe ────────────────────────────────────────────────────────
// Stripe webhook: real financial events. Sovereign operator sees real money.
// Verified via Stripe webhook signature (not bearer auth).

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		http.Error(w, `{"error":"read failed"}`, 400)
		return
	}

	// Verify Stripe webhook signature if secret is configured
	if stripeWHSecret != "" {
		sigHeader := r.Header.Get("Stripe-Signature")
		if !verifyStripeSignature(body, sigHeader, stripeWHSecret) {
			log.Printf("Stripe webhook: invalid signature")
			http.Error(w, `{"error":"invalid signature"}`, 401)
			return
		}
	}

	var payload map[string]interface{}
	if json.Unmarshal(body, &payload) != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	evtTypeStripe, _ := payload["type"].(string)
	data, _ := payload["data"].(map[string]interface{})
	obj, _ := data["object"].(map[string]interface{})

	// Track which Stripe account sent this event (for multi-account support)
	stripeAccount, _ := payload["account"].(string) // Connect account ID, empty for direct
	if stripeAccount == "" {
		// Direct account — use the account from the charge object
		if acct, ok := obj["account"].(string); ok {
			stripeAccount = acct
		}
	}

	var evtType, lane, title string
	var amount float64
	var scoreDelta int
	metadata := map[string]interface{}{"stripe_event": evtTypeStripe, "currency": "usd", "stripe_account": stripeAccount}

	switch evtTypeStripe {
	case "charge.succeeded":
		evtType = "PAYMENT_RECEIVED"
		lane = "revenue"
		amount, _ = obj["amount"].(float64)
		desc, _ := obj["description"].(string)
		title = fmt.Sprintf("💰 Payment: $%.2f — %s", amount/100, truncate(desc, 60))
		metadata["amount"] = amount
		metadata["customer"], _ = obj["customer"].(string)
		scoreDelta = 5

	case "charge.failed":
		evtType = "PAYMENT_FAILED"
		lane = "revenue"
		amount, _ = obj["amount"].(float64)
		title = fmt.Sprintf("❌ Failed charge: $%.2f", amount/100)
		metadata["amount"] = amount
		metadata["failure_message"], _ = obj["failure_message"].(string)
		scoreDelta = 0

	case "charge.refunded":
		evtType = "REFUND_ISSUED"
		lane = "revenue"
		amount, _ = obj["amount_refunded"].(float64)
		title = fmt.Sprintf("↩️ Refund: $%.2f", amount/100)
		metadata["amount"] = amount
		scoreDelta = -2

	case "invoice.paid":
		evtType = "INVOICE_PAID"
		lane = "revenue"
		amount, _ = obj["amount_paid"].(float64)
		title = fmt.Sprintf("📄 Invoice paid: $%.2f", amount/100)
		metadata["amount"] = amount
		metadata["subscription"], _ = obj["subscription"].(string)
		scoreDelta = 5

	case "invoice.payment_failed":
		evtType = "INVOICE_FAILED"
		lane = "revenue"
		amount, _ = obj["amount_due"].(float64)
		title = fmt.Sprintf("⚠️ Invoice payment failed: $%.2f", amount/100)
		metadata["amount"] = amount
		scoreDelta = 0

	case "payout.paid":
		evtType = "PAYOUT_RECEIVED"
		lane = "revenue"
		amount, _ = obj["amount"].(float64)
		title = fmt.Sprintf("🏦 Payout to bank: $%.2f", amount/100)
		metadata["amount"] = amount
		scoreDelta = 3

	case "customer.subscription.created":
		evtType = "SUBSCRIPTION_CREATED"
		lane = "revenue"
		plan, _ := obj["plan"].(map[string]interface{})
		planAmt, _ := plan["amount"].(float64)
		interval, _ := plan["interval"].(string)
		title = fmt.Sprintf("🆕 New subscription: $%.2f/%s", planAmt/100, interval)
		metadata["plan_amount"] = planAmt
		metadata["interval"] = interval
		scoreDelta = 8

	case "customer.subscription.deleted":
		evtType = "SUBSCRIPTION_CANCELED"
		lane = "revenue"
		title = "📉 Subscription canceled"
		scoreDelta = -3

	case "customer.subscription.updated":
		evtType = "SUBSCRIPTION_UPDATED"
		lane = "revenue"
		title = "🔄 Subscription updated"
		scoreDelta = 0

	default:
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "skipped": evtTypeStripe})
		return
	}

	metaJSON, _ := json.Marshal(metadata)
	id := fmt.Sprintf("evt-stripe-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)

	// Resolve business_id from Stripe account → integration mapping
	businessID := ""
	if stripeAccount != "" {
		s.db.QueryRow(`SELECT json_extract(config, '$.business_id') FROM integrations
			WHERE provider='stripe' AND status='active' AND
			json_extract(config, '$.stripe_account_id')=? LIMIT 1`, stripeAccount).Scan(&businessID)
	}

	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, business_id, created_at, status)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, evtType, lane, "stripe-webhook", now,
		title, 0.99, `["stripe_webhook"]`, "STRONG",
		scoreDelta, string(metaJSON), businessID, now, "approved")
	s.mu.Unlock()

	today := operatorToday()
	s.updateDailyScore(today)
	s.recalcSeason()

	// Feed Stripe data into pairing engine for business reality calibration
	s.pairing.Ingest(Signal{
		Type:      SignalAccount,
		Source:    "stripe",
		Timestamp: time.Now(),
		Content:   title,
		Features:  map[string]float64{},
		Metadata: map[string]interface{}{
			"provider":        "stripe",
			"event_type":      evtType,
			"monthly_revenue": float64(scoreDelta) * 100, // approximate
		},
	})

	log.Printf("Stripe: %s → %s (score_delta=%d)", evtTypeStripe, evtType, scoreDelta)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": true, "event_id": id, "event_type": evtType, "score_delta": scoreDelta,
	})
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// verifyStripeSignature checks Stripe webhook HMAC-SHA256 signature.
func verifyStripeSignature(payload []byte, sigHeader, secret string) bool {
	if sigHeader == "" {
		return false
	}
	// Parse t=timestamp,v1=signature from Stripe-Signature header
	var timestamp, sig string
	for _, part := range strings.Split(sigHeader, ",") {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		switch kv[0] {
		case "t":
			timestamp = kv[1]
		case "v1":
			sig = kv[1]
		}
	}
	if timestamp == "" || sig == "" {
		return false
	}

	// Compute expected signature: HMAC-SHA256(secret, timestamp.payload)
	message := timestamp + "." + string(payload)
	expected := fmt.Sprintf("%x", hmacSHA256([]byte(message), []byte(secret)))
	return hmacEqual(expected, sig)
}

// ─── Financial Snapshot (Operator + Wirebot reasoning) ──────────────────────
// Returns real-time financial state from Stripe for AI reasoning.

func (s *Server) handleFinancialSnapshot(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	if stripeKey == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Stripe not configured"})
		return
	}

	snapshot := map[string]interface{}{}

	// Revenue from scored events (last 30/90 days)
	var rev30, rev90 float64
	var charges30, charges90 int
	s.db.QueryRow(`SELECT COALESCE(SUM(json_extract(metadata,'$.amount')),0), COUNT(*)
		FROM events WHERE source='stripe-webhook' AND event_type='PAYMENT_RECEIVED'
		AND timestamp > datetime('now','-30 days')`).Scan(&rev30, &charges30)
	s.db.QueryRow(`SELECT COALESCE(SUM(json_extract(metadata,'$.amount')),0), COUNT(*)
		FROM events WHERE source='stripe-webhook' AND event_type='PAYMENT_RECEIVED'
		AND timestamp > datetime('now','-90 days')`).Scan(&rev90, &charges90)

	snapshot["revenue_30d"] = rev30 / 100
	snapshot["revenue_90d"] = rev90 / 100
	snapshot["charges_30d"] = charges30
	snapshot["charges_90d"] = charges90
	snapshot["mrr_estimate"] = rev30 / 100 // Rough MRR from last 30 days

	// Recent events
	var recentEvents []map[string]interface{}
	rows, _ := s.db.Query(`SELECT event_type, artifact_title, timestamp, score_delta, metadata
		FROM events WHERE source='stripe-webhook' ORDER BY timestamp DESC LIMIT 10`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var et, title, ts, meta string
			var sd int
			rows.Scan(&et, &title, &ts, &sd, &meta)
			recentEvents = append(recentEvents, map[string]interface{}{
				"type": et, "title": title, "timestamp": ts, "score_delta": sd,
			})
		}
	}
	snapshot["recent_events"] = recentEvents

	// Churn signals
	var failedCharges int
	s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE source='stripe-webhook'
		AND event_type IN ('PAYMENT_FAILED','INVOICE_FAILED')
		AND timestamp > datetime('now','-30 days')`).Scan(&failedCharges)
	snapshot["failed_charges_30d"] = failedCharges

	var cancellations int
	s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE source='stripe-webhook'
		AND event_type='SUBSCRIPTION_CANCELED'
		AND timestamp > datetime('now','-30 days')`).Scan(&cancellations)
	snapshot["cancellations_30d"] = cancellations

	json.NewEncoder(w).Encode(snapshot)
}

// ─── SSO Callback ───────────────────────────────────────────────────────────
// Receives JWT via URL fragment from Connect Plugin SSO redirect.
// Serves a tiny HTML page that reads the fragment and stores in localStorage.

// ─── Plaid (Bank Account Connections) ────────────────────────────────────

func plaidBaseURL() string {
	switch plaidEnv {
	case "production":
		return "https://production.plaid.com"
	case "development":
		return "https://development.plaid.com"
	default:
		return "https://sandbox.plaid.com"
	}
}

func plaidRequest(endpoint string, body interface{}) (map[string]interface{}, error) {
	payload, _ := json.Marshal(body)
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("POST", plaidBaseURL()+endpoint, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	if errMsg, ok := result["error_message"].(string); ok && errMsg != "" {
		return result, fmt.Errorf("plaid: %s", errMsg)
	}
	return result, nil
}

// POST /v1/plaid/link-token — create a Plaid Link token for the frontend widget
func (s *Server) handlePlaidLinkToken(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	if plaidClientID == "" || plaidSecret == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "Plaid not configured. Set PLAID_CLIENT_ID and PLAID_SECRET.",
		})
		return
	}

	// Get auth context for user ID
	auth := resolveAuth(r)
	userID := "user-default"
	if auth.Username != "" {
		userID = fmt.Sprintf("user-%s", auth.Username)
	}

	var body struct {
		Products string `json:"products"` // "transactions" or "transactions,investments"
	}
	json.NewDecoder(r.Body).Decode(&body)
	products := []string{"transactions"}
	if body.Products != "" {
		products = strings.Split(body.Products, ",")
	}

	result, err := plaidRequest("/link/token/create", map[string]interface{}{
		"client_id":    plaidClientID,
		"secret":       plaidSecret,
		"user":         map[string]string{"client_user_id": userID},
		"client_name":  "Wirebot Scoreboard",
		"products":     products,
		"country_codes": []string{"US"},
		"language":     "en",
		"redirect_uri": fmt.Sprintf("%s/v1/plaid/oauth-redirect", oauthCallbackBase),
	})
	if err != nil {
		log.Printf("Plaid link token error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	linkToken, _ := result["link_token"].(string)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"link_token": linkToken,
		"expiration": result["expiration"],
	})
}

// POST /v1/plaid/exchange — exchange public_token for access_token, store as integration
func (s *Server) handlePlaidExchange(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var body struct {
		PublicToken string `json:"public_token"`
		Institution struct {
			ID   string `json:"institution_id"`
			Name string `json:"name"`
		} `json:"institution"`
		Accounts []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Mask    string `json:"mask"`
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
		} `json:"accounts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	if body.PublicToken == "" {
		http.Error(w, `{"error":"public_token required"}`, 400)
		return
	}

	// Exchange public_token → access_token
	result, err := plaidRequest("/item/public_token/exchange", map[string]interface{}{
		"client_id":    plaidClientID,
		"secret":       plaidSecret,
		"public_token": body.PublicToken,
	})
	if err != nil {
		log.Printf("Plaid exchange error: %v", err)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	accessToken, _ := result["access_token"].(string)
	itemID, _ := result["item_id"].(string)

	if accessToken == "" {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "No access token received"})
		return
	}

	// Build display name from institution + accounts
	displayName := body.Institution.Name
	if len(body.Accounts) > 0 {
		acctNames := []string{}
		for _, a := range body.Accounts {
			name := a.Name
			if a.Mask != "" {
				name = fmt.Sprintf("%s ••%s", a.Name, a.Mask)
			}
			acctNames = append(acctNames, name)
		}
		displayName = fmt.Sprintf("%s (%s)", body.Institution.Name, strings.Join(acctNames, ", "))
	}

	// Build config with account IDs for polling
	acctIDs := []string{}
	for _, a := range body.Accounts {
		acctIDs = append(acctIDs, a.ID)
	}
	config := map[string]interface{}{
		"institution_id":   body.Institution.ID,
		"institution_name": body.Institution.Name,
		"account_ids":      acctIDs,
		"item_id":          itemID,
	}
	configJSON, _ := json.Marshal(config)

	// Encrypt access_token
	tokenJSON, _ := json.Marshal(map[string]string{"access_token": accessToken, "item_id": itemID})
	encrypted, nonce, err := s.encryptCredential(tokenJSON)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Encryption failed"})
		return
	}

	// Store integration
	id := fmt.Sprintf("int-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	nextPoll := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)

	s.mu.Lock()
	s.db.Exec(`INSERT INTO integrations (id, user_id, provider, auth_type, encrypted_data, nonce,
		display_name, poll_interval_seconds, config, created_at, updated_at, next_poll_at, sensitivity)
		VALUES (?, 'default', 'plaid', 'plaid', ?, ?, ?, 1800, ?, ?, ?, ?, 'sensitive')`,
		id, encrypted, nonce, displayName, string(configJSON), now, now, nextPoll)
	s.mu.Unlock()

	log.Printf("Plaid: connected %s (%s), %d accounts, integration %s", body.Institution.Name, itemID, len(body.Accounts), id)

	// Immediately poll for recent transactions
	go s.pollPlaid(id, accessToken, string(configJSON), "")

	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":          true,
		"id":          id,
		"institution": body.Institution.Name,
		"accounts":    len(body.Accounts),
	})
}

// pollPlaid fetches recent transactions and creates scoreboard events
func (s *Server) pollPlaid(integrationID, accessToken, configJSON, lastPoll string) error {
	if plaidClientID == "" || plaidSecret == "" || accessToken == "" {
		return fmt.Errorf("plaid not configured or no access token")
	}

	// Date range: last poll to now (or last 30 days on first poll)
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02") // Last 7 days
	if lastPoll != "" {
		if t, err := time.Parse(time.RFC3339, lastPoll); err == nil {
			startDate = t.Format("2006-01-02")
		}
	}

	result, err := plaidRequest("/transactions/get", map[string]interface{}{
		"client_id":    plaidClientID,
		"secret":       plaidSecret,
		"access_token": accessToken,
		"start_date":   startDate,
		"end_date":     endDate,
		"options":      map[string]int{"count": 100, "offset": 0},
	})
	if err != nil {
		return fmt.Errorf("plaid transactions: %w", err)
	}

	// Parse accounts for balance info
	accounts, _ := result["accounts"].([]interface{})
	for _, acctRaw := range accounts {
		acct, ok := acctRaw.(map[string]interface{})
		if !ok {
			continue
		}
		balances, _ := acct["balances"].(map[string]interface{})
		current, _ := balances["current"].(float64)
		available, _ := balances["available"].(float64)
		name, _ := acct["name"].(string)
		mask, _ := acct["mask"].(string)

		// Store balance snapshot as a metadata update (not a scored event)
		balMeta, _ := json.Marshal(map[string]interface{}{
			"account":   name,
			"mask":      mask,
			"current":   current,
			"available": available,
			"source":    "plaid",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		_ = balMeta // balance metadata logged above
		s.mu.Lock()
		s.db.Exec(`UPDATE integrations SET last_used_at=?,
			config=json_set(config, '$.balance_current', ?, '$.balance_available', ?, '$.balance_account', ?)
			WHERE id=?`,
			time.Now().UTC().Format(time.RFC3339), current, available, name+" ••"+mask, integrationID)
		s.mu.Unlock()

		log.Printf("Plaid: %s ••%s balance: $%.2f current, $%.2f available", name, mask, current, available)
	}

	// Parse transactions → revenue events
	transactions, _ := result["transactions"].([]interface{})
	newCount := 0
	for _, txnRaw := range transactions {
		txn, ok := txnRaw.(map[string]interface{})
		if !ok {
			continue
		}

		amount, _ := txn["amount"].(float64)
		txnName, _ := txn["name"].(string)
		txnDate, _ := txn["date"].(string)
		txnID, _ := txn["transaction_id"].(string)
		merchantName, _ := txn["merchant_name"].(string)
		category, _ := txn["category"].([]interface{})

		// Plaid amounts: positive = money leaving (expense), negative = money coming in (income)
		// We want income events (negative amounts = deposits/payments received)
		isIncome := amount < 0
		absAmount := math.Abs(amount)

		// Skip tiny transactions
		if absAmount < 1.0 {
			continue
		}

		// Check for duplicate
		evtID := fmt.Sprintf("evt-plaid-%s", txnID)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
		if exists > 0 {
			continue
		}

		var evtType, lane, title string
		var scoreDelta int
		catStr := ""
		if len(category) > 0 {
			cats := []string{}
			for _, c := range category {
				if cs, ok := c.(string); ok {
					cats = append(cats, cs)
				}
			}
			catStr = strings.Join(cats, "/")
		}

		displayName := txnName
		if merchantName != "" {
			displayName = merchantName
		}

		if isIncome {
			evtType = "PAYMENT_RECEIVED"
			lane = "revenue"
			title = fmt.Sprintf("🏦 Deposit: $%.2f — %s", absAmount, displayName)
			scoreDelta = 5
			if absAmount >= 500 {
				scoreDelta = 8
			}
			if absAmount >= 2000 {
				scoreDelta = 10
			}
		} else {
			// Expense — track but don't score positively
			evtType = "EXPENSE"
			lane = "revenue"
			title = fmt.Sprintf("💸 Expense: $%.2f — %s", absAmount, displayName)
			scoreDelta = 0 // Expenses don't add score, but are visible
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"amount":        absAmount,
			"is_income":     isIncome,
			"merchant":      merchantName,
			"category":      catStr,
			"transaction_id": txnID,
			"plaid_source":  true,
		})

		now := time.Now().UTC().Format(time.RFC3339)
		txnTimestamp := txnDate + "T12:00:00Z" // Plaid dates are date-only

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_title, confidence, verifiers, verification_level,
			score_delta, metadata, created_at, status)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			evtID, evtType, lane, "plaid", txnTimestamp,
			title, 0.95, `["plaid_api"]`, "STRONG",
			scoreDelta, string(meta), now, "approved")
		s.mu.Unlock()

		pubTime, _ := time.Parse("2006-01-02", txnDate)
		if !pubTime.IsZero() {
			s.updateDailyScore(pubTime.Format("2006-01-02"))
		}
		newCount++
	}

	if newCount > 0 {
		log.Printf("Plaid: %d new transactions from integration %s", newCount, integrationID)
	}
	return nil
}

// ─── OAuth Flow (Provider Authorization) ────────────────────────────────
// Handles OAuth initiation and callback for Stripe, GitHub, Google/YouTube.
// OAuth app credentials stored as env vars:
//   OAUTH_STRIPE_CLIENT_ID, OAUTH_STRIPE_CLIENT_SECRET
//   OAUTH_GITHUB_CLIENT_ID, OAUTH_GITHUB_CLIENT_SECRET
//   OAUTH_GOOGLE_CLIENT_ID, OAUTH_GOOGLE_CLIENT_SECRET

var (
	oauthStripeClientID         = os.Getenv("OAUTH_STRIPE_CLIENT_ID")
	oauthStripeClientSecret     = os.Getenv("OAUTH_STRIPE_CLIENT_SECRET")
	oauthGitHubClientID         = os.Getenv("OAUTH_GITHUB_CLIENT_ID")
	oauthGitHubClientSecret     = os.Getenv("OAUTH_GITHUB_CLIENT_SECRET")
	oauthGoogleClientID         = os.Getenv("OAUTH_GOOGLE_CLIENT_ID")
	oauthGoogleClientSecret     = os.Getenv("OAUTH_GOOGLE_CLIENT_SECRET")
	oauthFreshBooksClientID     = os.Getenv("OAUTH_FRESHBOOKS_CLIENT_ID")
	oauthFreshBooksClientSecret = os.Getenv("OAUTH_FRESHBOOKS_CLIENT_SECRET")
	oauthHubSpotClientID        = os.Getenv("OAUTH_HUBSPOT_CLIENT_ID")
	oauthHubSpotClientSecret    = os.Getenv("OAUTH_HUBSPOT_CLIENT_SECRET")
	oauthCallbackBase           = envOr("OAUTH_CALLBACK_BASE", "https://wins.wirebot.chat")
)

type oauthProviderConfig struct {
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	Scopes       string
	Provider     string
}

func getOAuthConfig(provider string) *oauthProviderConfig {
	switch provider {
	case "stripe":
		if oauthStripeClientID == "" {
			return nil
		}
		return &oauthProviderConfig{
			ClientID: oauthStripeClientID, ClientSecret: oauthStripeClientSecret,
			AuthURL: "https://connect.stripe.com/oauth/authorize", TokenURL: "https://connect.stripe.com/oauth/token",
			Scopes: "read_write", Provider: "stripe",
		}
	case "github":
		if oauthGitHubClientID == "" {
			return nil
		}
		return &oauthProviderConfig{
			ClientID: oauthGitHubClientID, ClientSecret: oauthGitHubClientSecret,
			AuthURL: "https://github.com/login/oauth/authorize", TokenURL: "https://github.com/login/oauth/access_token",
			Scopes: "repo,admin:repo_hook", Provider: "github",
		}
	case "google":
		if oauthGoogleClientID == "" {
			return nil
		}
		return &oauthProviderConfig{
			ClientID: oauthGoogleClientID, ClientSecret: oauthGoogleClientSecret,
			AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
			Scopes: "https://www.googleapis.com/auth/youtube.readonly", Provider: "google",
		}
	case "freshbooks":
		if oauthFreshBooksClientID == "" {
			return nil
		}
		return &oauthProviderConfig{
			ClientID: oauthFreshBooksClientID, ClientSecret: oauthFreshBooksClientSecret,
			AuthURL: "https://auth.freshbooks.com/oauth/authorize", TokenURL: "https://api.freshbooks.com/auth/oauth/token",
			Scopes: "", Provider: "freshbooks",
		}
	case "hubspot":
		if oauthHubSpotClientID == "" {
			return nil
		}
		return &oauthProviderConfig{
			ClientID: oauthHubSpotClientID, ClientSecret: oauthHubSpotClientSecret,
			AuthURL: "https://app.hubspot.com/oauth/authorize", TokenURL: "https://api.hubapi.com/oauth/v1/token",
			Scopes: "crm.objects.deals.read crm.objects.contacts.read", Provider: "hubspot",
		}
	}
	return nil
}

// handleOAuthConfig lets operator store OAuth app credentials (client_id/secret)
// via the UI instead of env vars. GET returns which providers are configured.
// POST stores new credentials.
// handleNetworkMembers returns the operator's actual connections from startempirewire.com.
// Uses BuddyBoss friends API (auth required) — only shows people you're connected to.
// Falls back to empty with a "connect" prompt if no friends yet.
func (s *Server) handleNetworkMembers(w http.ResponseWriter, r *http.Request) {
	cors(w)

	// Query BuddyBoss friends for the operator's WP user ID
	// TODO: resolve wp_user_id from JWT or config; for now use configured ID
	wpUserID := envOr("OPERATOR_WP_USER_ID", "229") // Verious = 229

	client := &http.Client{Timeout: 10 * time.Second}

	// Try friends endpoint (requires auth — use application password if available)
	wpAppPass := os.Getenv("WP_APP_PASSWORD")
	friendsURL := fmt.Sprintf("https://startempirewire.com/wp-json/buddyboss/v1/friends?user_id=%s&per_page=20", wpUserID)
	req, _ := http.NewRequest("GET", friendsURL, nil)
	if wpAppPass != "" {
		req.SetBasicAuth("verious.smith", wpAppPass)
	}
	resp, err := client.Do(req)

	var friends []map[string]interface{}
	if err == nil && resp.StatusCode == 200 {
		defer resp.Body.Close()
		json.NewDecoder(resp.Body).Decode(&friends)
	} else {
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Map to clean member objects
	var members []map[string]interface{}
	for _, m := range friends {
		name, _ := m["name"].(string)
		id := m["id"]
		avatar := ""
		if avatarURLs, ok := m["avatar_urls"].(map[string]interface{}); ok {
			if full, ok := avatarURLs["full"].(string); ok {
				avatar = full
			}
		}
		link, _ := m["link"].(string)
		members = append(members, map[string]interface{}{
			"id": id, "name": name, "avatar": avatar, "link": link,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"members": members,
		"count":   len(members),
		"source":  "startempirewire.com",
	})
}

func (s *Server) handleOAuthConfig(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}

	if r.Method == "GET" {
		// Return which OAuth providers are configured
		providers := map[string]bool{
			"github":     oauthGitHubClientID != "",
			"stripe":     oauthStripeClientID != "",
			"google":     oauthGoogleClientID != "",
			"freshbooks": oauthFreshBooksClientID != "",
			"hubspot":    oauthHubSpotClientID != "",
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"providers": providers,
			"callback_url": oauthCallbackBase + "/v1/oauth/callback",
		})
		return
	}

	if r.Method != "POST" {
		http.Error(w, `{"error":"GET or POST only"}`, 405)
		return
	}

	var req struct {
		Provider     string `json:"provider"`
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, 400)
		return
	}

	if req.Provider == "" || req.ClientID == "" || req.ClientSecret == "" {
		http.Error(w, `{"error":"provider, client_id, client_secret required"}`, 400)
		return
	}

	// Store in runtime (takes effect immediately)
	switch req.Provider {
	case "github":
		oauthGitHubClientID = req.ClientID
		oauthGitHubClientSecret = req.ClientSecret
	case "stripe":
		oauthStripeClientID = req.ClientID
		oauthStripeClientSecret = req.ClientSecret
	case "google":
		oauthGoogleClientID = req.ClientID
		oauthGoogleClientSecret = req.ClientSecret
	case "freshbooks":
		oauthFreshBooksClientID = req.ClientID
		oauthFreshBooksClientSecret = req.ClientSecret
	case "hubspot":
		oauthHubSpotClientID = req.ClientID
		oauthHubSpotClientSecret = req.ClientSecret
	default:
		http.Error(w, `{"error":"unknown provider, use: github, stripe, google"}`, 400)
		return
	}

	// Persist to env file so it survives restart
	envPath := os.Getenv("SCOREBOARD_ENV_PATH")
	if envPath == "" {
		envPath = "/run/wirebot/scoreboard.env"
	}
	envData, _ := os.ReadFile(envPath)
	envStr := string(envData)

	prefix := fmt.Sprintf("OAUTH_%s_CLIENT_ID=", strings.ToUpper(req.Provider))
	secretPrefix := fmt.Sprintf("OAUTH_%s_CLIENT_SECRET=", strings.ToUpper(req.Provider))

	// Replace existing lines
	lines := strings.Split(envStr, "\n")
	var newLines []string
	idSet, secretSet := false, false
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			newLines = append(newLines, fmt.Sprintf("%s%s", prefix, req.ClientID))
			idSet = true
		} else if strings.HasPrefix(line, secretPrefix) {
			newLines = append(newLines, fmt.Sprintf(`%s"%s"`, secretPrefix, req.ClientSecret))
			secretSet = true
		} else {
			newLines = append(newLines, line)
		}
	}
	if !idSet {
		newLines = append(newLines, fmt.Sprintf("%s%s", prefix, req.ClientID))
	}
	if !secretSet {
		newLines = append(newLines, fmt.Sprintf(`%s"%s"`, secretPrefix, req.ClientSecret))
	}

	os.WriteFile(envPath, []byte(strings.Join(newLines, "\n")), 0600)

	log.Printf("OAuth: %s credentials configured via UI (client_id: %s...)", req.Provider, req.ClientID[:8])
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":       true,
		"provider": req.Provider,
		"message":  fmt.Sprintf("%s OAuth configured — users can now click Connect", req.Provider),
	})
}

// handleGitHubSetup uses the GitHub App Manifest Flow to create an OAuth app
// with zero copy-paste. Operator clicks → GitHub creates app → credentials returned.
func (s *Server) handleGitHubSetup(w http.ResponseWriter, r *http.Request) {
	callbackURL := oauthCallbackBase + "/v1/oauth/callback"
	_ = oauthCallbackBase + "/v1/oauth/setup/github/callback" // available for future use

	manifest := map[string]interface{}{
		"name":               "Wirebot Scoreboard",
		"url":                oauthCallbackBase,
		"redirect_url":       callbackURL,
		"callback_urls":      []string{callbackURL},
		"setup_url":          oauthCallbackBase,
		"hook_attributes":    map[string]interface{}{"url": oauthCallbackBase + "/v1/webhooks/github", "active": true},
		"public":             true,
		"default_permissions": map[string]string{"contents": "read", "metadata": "read", "pull_requests": "read"},
		"default_events":     []string{"push", "pull_request", "release"},
	}

	manifestJSON, _ := json.Marshal(manifest)

	// Generate state for CSRF
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	http.SetCookie(w, &http.Cookie{
		Name: "github_setup_state", Value: state,
		Path: "/", MaxAge: 600, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})

	// Render a form that auto-submits to GitHub
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Setting up GitHub...</title>
<style>body{background:#0a0a12;color:#ddd;font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100dvh;margin:0}
.card{text-align:center;padding:32px}.spinner{width:32px;height:32px;border:3px solid #222;border-top-color:#7c7cff;border-radius:50%%;animation:spin .8s linear infinite;margin:0 auto 16px}
@keyframes spin{to{transform:rotate(360deg)}}</style></head>
<body><div class="card"><div class="spinner"></div><h2>Redirecting to GitHub...</h2><p>Creating Wirebot app for your account</p></div>
<form id="f" method="post" action="https://github.com/settings/apps/new?state=%s">
<input type="hidden" name="manifest" value='%s' />
</form>
<script>document.getElementById('f').submit();</script>
</body></html>`, state, string(manifestJSON))
}

// handleGitHubSetupCallback receives the code from GitHub after app creation
func (s *Server) handleGitHubSetupCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Redirect(w, r, "/?oauth=github&oauth_status=fail&error=No+code+from+GitHub", 302)
		return
	}

	// Exchange code for app credentials
	client := &http.Client{Timeout: 15 * time.Second}
	convURL := fmt.Sprintf("https://api.github.com/app-manifests/%s/conversions", code)
	req, _ := http.NewRequest("POST", convURL, nil)
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		http.Redirect(w, r, "/?oauth=github&oauth_status=fail&error=GitHub+API+error", 302)
		return
	}
	defer resp.Body.Close()

	var appData map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&appData)

	if appData["client_id"] == nil {
		errMsg := "Unknown error"
		if msg, ok := appData["message"].(string); ok {
			errMsg = msg
		}
		http.Redirect(w, r, fmt.Sprintf("/?oauth=github&oauth_status=fail&error=%s", errMsg), 302)
		return
	}

	clientID, _ := appData["client_id"].(string)
	clientSecret, _ := appData["client_secret"].(string)
	appName, _ := appData["name"].(string)

	// Store credentials
	oauthGitHubClientID = clientID
	oauthGitHubClientSecret = clientSecret

	// Persist to env file
	envPath := os.Getenv("SCOREBOARD_ENV_PATH")
	if envPath == "" {
		envPath = "/run/wirebot/scoreboard.env"
	}
	envData, _ := os.ReadFile(envPath)
	envStr := string(envData)
	lines := strings.Split(envStr, "\n")
	var newLines []string
	idSet, secretSet := false, false
	for _, line := range lines {
		if strings.HasPrefix(line, "OAUTH_GITHUB_CLIENT_ID=") {
			newLines = append(newLines, "OAUTH_GITHUB_CLIENT_ID="+clientID)
			idSet = true
		} else if strings.HasPrefix(line, "OAUTH_GITHUB_CLIENT_SECRET=") {
			newLines = append(newLines, fmt.Sprintf(`OAUTH_GITHUB_CLIENT_SECRET="%s"`, clientSecret))
			secretSet = true
		} else {
			newLines = append(newLines, line)
		}
	}
	if !idSet {
		newLines = append(newLines, "OAUTH_GITHUB_CLIENT_ID="+clientID)
	}
	if !secretSet {
		newLines = append(newLines, fmt.Sprintf(`OAUTH_GITHUB_CLIENT_SECRET="%s"`, clientSecret))
	}
	os.WriteFile(envPath, []byte(strings.Join(newLines, "\n")), 0600)

	log.Printf("GitHub App created via manifest: %s (client_id: %s)", appName, clientID)
	http.Redirect(w, r, "/?oauth=github&oauth_status=ok&message=GitHub+app+created+successfully", 302)
}

// handleStripeSetup starts Stripe Connect Express onboarding
// Operator clicks → redirected to Stripe → account connected
func (s *Server) handleStripeSetup(w http.ResponseWriter, r *http.Request) {
	// If already configured, just redirect to the OAuth flow
	if oauthStripeClientID != "" {
		http.Redirect(w, r, "/v1/oauth/stripe/authorize", 302)
		return
	}
	// For Stripe, we need to check if they have an existing account
	// Stripe doesn't have a manifest flow, but we can use Stripe Connect
	// Standard account linking which works like OAuth once the platform is set up
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Stripe Setup</title>
<style>body{background:#0a0a12;color:#ddd;font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100dvh;margin:0}
.card{text-align:center;padding:32px;max-width:400px}h2{margin:0 0 12px}p{color:#888;font-size:14px;line-height:1.6}
a{color:#7c7cff;text-decoration:none;padding:12px 24px;border:1px solid #7c7cff;border-radius:8px;display:inline-block;margin-top:16px}
a:hover{background:#7c7cff20}</style></head>
<body><div class="card">
<h2>💳 Connect Stripe</h2>
<p>Stripe requires creating a Connect platform in your dashboard. This takes about 60 seconds.</p>
<a href="https://dashboard.stripe.com/settings/connect" target="_blank">Open Stripe Settings →</a>
<p style="font-size:12px;margin-top:24px;color:#555">After enabling Connect, your account's client_id will appear in the Connect settings. Come back here to finish setup.</p>
</div></body></html>`)
}


func (s *Server) handleFreshBooksSetup(w http.ResponseWriter, r *http.Request) {
	if oauthFreshBooksClientID != "" {
		http.Redirect(w, r, "/v1/oauth/freshbooks/authorize", 302)
		return
	}
	callbackURL := oauthCallbackBase + "/v1/oauth/callback"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Set Up FreshBooks</title>
<style>body{background:#0a0a12;color:#ddd;font-family:system-ui;margin:0;padding:24px}
.card{max-width:480px;margin:40px auto;text-align:center}h2{margin:0 0 8px}
.sub{color:#888;font-size:14px;margin:0 0 24px}
.steps{text-align:left;background:#111118;border:1px solid #1e1e30;border-radius:12px;padding:20px;margin:0 0 20px}
.step{display:flex;gap:10px;margin-bottom:14px;font-size:13px;color:#aaa;line-height:1.5}.step:last-child{margin-bottom:0}
.num{width:24px;height:24px;border-radius:50%%;background:#7c7cff15;color:#7c7cff;font-size:11px;font-weight:700;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.cb{background:#0a0a15;border:1px solid #2a2a40;border-radius:8px;padding:10px 12px;font-family:monospace;font-size:12px;color:#7c7cff;word-break:break-all;user-select:all;cursor:pointer;margin:8px 0}
.cb:hover{border-color:#7c7cff}
.fields{display:flex;flex-direction:column;gap:8px;margin-top:16px}
.fields input{padding:10px 12px;background:#0a0a15;border:1px solid #2a2a40;border-radius:8px;color:#ddd;font-size:13px;outline:none}
.fields input:focus{border-color:#7c7cff}
.save{width:100%%;padding:10px;background:#7c7cff;color:white;border:none;border-radius:8px;font-weight:600;font-size:13px;cursor:pointer;margin-top:4px}
.save:hover{background:#6c6cee}.save:disabled{opacity:.4}
.back{display:block;margin-top:20px;color:#555;font-size:12px;text-decoration:none}.back:hover{color:#888}
</style></head>
<body><div class="card">
<h2>📗 Set Up FreshBooks</h2>
<p class="sub">One-time setup — takes about 60 seconds</p>
<div class="steps">
<div class="step"><span class="num">1</span><span>Go to <a href="https://my.freshbooks.com/#/developer" target="_blank" style="color:#7c7cff">FreshBooks Developer Portal</a> → Create App</span></div>
<div class="step"><span class="num">2</span><span>Set the redirect URI to:</span></div>
<div class="cb" onclick="navigator.clipboard.writeText('%s');this.style.borderColor='#2ecc71'">%s</div>
<div class="step"><span class="num">3</span><span>Copy the Client ID and Client Secret below</span></div>
</div>
<div class="fields">
<input type="text" id="cid" placeholder="Client ID" />
<input type="password" id="csec" placeholder="Client Secret" />
<button class="save" id="btn" onclick="
  var c=document.getElementById('cid').value,s=document.getElementById('csec').value;
  if(!c||!s)return;this.disabled=true;this.textContent='Saving...';
  fetch('/v1/oauth/config',{method:'POST',headers:{'Content-Type':'application/json','Authorization':localStorage.getItem('token')?'Bearer '+localStorage.getItem('token'):''},body:JSON.stringify({provider:'freshbooks',client_id:c,client_secret:s})}).then(function(r){return r.json()}).then(function(d){if(d.ok)window.location='/?oauth=freshbooks&oauth_status=ok';else{document.getElementById('btn').disabled=false;document.getElementById('btn').textContent=d.error||'Error'}})
">Save & Enable Connect</button>
</div>
<a class="back" href="/">← Back to scoreboard</a>
</div></body></html>`, callbackURL, callbackURL)
}

func (s *Server) handleHubSpotSetup(w http.ResponseWriter, r *http.Request) {
	if oauthHubSpotClientID != "" {
		http.Redirect(w, r, "/v1/oauth/hubspot/authorize", 302)
		return
	}
	callbackURL := oauthCallbackBase + "/v1/oauth/callback"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Set Up HubSpot</title>
<style>body{background:#0a0a12;color:#ddd;font-family:system-ui;margin:0;padding:24px}
.card{max-width:480px;margin:40px auto;text-align:center}h2{margin:0 0 8px}
.sub{color:#888;font-size:14px;margin:0 0 24px}
.steps{text-align:left;background:#111118;border:1px solid #1e1e30;border-radius:12px;padding:20px;margin:0 0 20px}
.step{display:flex;gap:10px;margin-bottom:14px;font-size:13px;color:#aaa;line-height:1.5}.step:last-child{margin-bottom:0}
.num{width:24px;height:24px;border-radius:50%%;background:#7c7cff15;color:#7c7cff;font-size:11px;font-weight:700;display:flex;align-items:center;justify-content:center;flex-shrink:0}
.cb{background:#0a0a15;border:1px solid #2a2a40;border-radius:8px;padding:10px 12px;font-family:monospace;font-size:12px;color:#7c7cff;word-break:break-all;user-select:all;cursor:pointer;margin:8px 0}
.cb:hover{border-color:#7c7cff}
.fields{display:flex;flex-direction:column;gap:8px;margin-top:16px}
.fields input{padding:10px 12px;background:#0a0a15;border:1px solid #2a2a40;border-radius:8px;color:#ddd;font-size:13px;outline:none}
.fields input:focus{border-color:#7c7cff}
.save{width:100%%;padding:10px;background:#7c7cff;color:white;border:none;border-radius:8px;font-weight:600;font-size:13px;cursor:pointer;margin-top:4px}
.save:hover{background:#6c6cee}.save:disabled{opacity:.4}
.back{display:block;margin-top:20px;color:#555;font-size:12px;text-decoration:none}.back:hover{color:#888}
</style></head>
<body><div class="card">
<h2>🔶 Set Up HubSpot</h2>
<p class="sub">One-time setup — takes about 60 seconds</p>
<div class="steps">
<div class="step"><span class="num">1</span><span>Go to <a href="https://app.hubspot.com/developer" target="_blank" style="color:#7c7cff">HubSpot Developer Portal</a> → Create App</span></div>
<div class="step"><span class="num">2</span><span>Under Auth tab, set redirect URI to:</span></div>
<div class="cb" onclick="navigator.clipboard.writeText('%s');this.style.borderColor='#2ecc71'">%s</div>
<div class="step"><span class="num">3</span><span>Copy the Client ID and Client Secret below</span></div>
</div>
<div class="fields">
<input type="text" id="cid" placeholder="Client ID" />
<input type="password" id="csec" placeholder="Client Secret" />
<button class="save" id="btn" onclick="
  var c=document.getElementById('cid').value,s=document.getElementById('csec').value;
  if(!c||!s)return;this.disabled=true;this.textContent='Saving...';
  fetch('/v1/oauth/config',{method:'POST',headers:{'Content-Type':'application/json','Authorization':localStorage.getItem('token')?'Bearer '+localStorage.getItem('token'):''},body:JSON.stringify({provider:'hubspot',client_id:c,client_secret:s})}).then(function(r){return r.json()}).then(function(d){if(d.ok)window.location='/?oauth=hubspot&oauth_status=ok';else{document.getElementById('btn').disabled=false;document.getElementById('btn').textContent=d.error||'Error'}})
">Save & Enable Connect</button>
</div>
<a class="back" href="/">← Back to scoreboard</a>
</div></body></html>`, callbackURL, callbackURL)
}

func (s *Server) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	// Determine provider from URL path: /v1/oauth/{provider}/authorize
	parts := strings.Split(r.URL.Path, "/")
	var provider string
	for i, p := range parts {
		if p == "oauth" && i+1 < len(parts) {
			provider = parts[i+1]
			break
		}
	}

	cfg := getOAuthConfig(provider)
	if cfg == nil {
		// OAuth not configured yet — redirect back with helpful error
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=OAuth+app+not+configured+yet.+Contact+your+operator.", provider), 302)
		return
	}

	// Generate state token for CSRF protection
	stateBytes := make([]byte, 16)
	rand.Read(stateBytes)
	state := hex.EncodeToString(stateBytes)

	// Store state in a short-lived cookie (5 minutes)
	http.SetCookie(w, &http.Cookie{
		Name: "oauth_state", Value: state,
		Path: "/", MaxAge: 300, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})

	callbackURL := fmt.Sprintf("%s/v1/oauth/callback", oauthCallbackBase)

	var authURL string
	if provider == "stripe" {
		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&state=%s",
			cfg.AuthURL, cfg.ClientID, cfg.Scopes, callbackURL, state)
	} else if provider == "google" {
		scope := r.URL.Query().Get("scope")
		if scope == "youtube" {
			scope = "https://www.googleapis.com/auth/youtube.readonly"
		} else {
			scope = cfg.Scopes
		}
		authURL = fmt.Sprintf("%s?response_type=code&client_id=%s&scope=%s&redirect_uri=%s&state=%s&access_type=offline&prompt=consent",
			cfg.AuthURL, cfg.ClientID, scope, callbackURL, state)
	} else {
		authURL = fmt.Sprintf("%s?client_id=%s&scope=%s&redirect_uri=%s&state=%s",
			cfg.AuthURL, cfg.ClientID, cfg.Scopes, callbackURL, state)
	}

	// Store provider in state cookie so callback knows which provider
	http.SetCookie(w, &http.Cookie{
		Name: "oauth_provider", Value: provider,
		Path: "/", MaxAge: 300, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, authURL, 302)
}

func (s *Server) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errParam := r.URL.Query().Get("error")

	// Get provider from cookie
	providerCookie, _ := r.Cookie("oauth_provider")
	provider := ""
	if providerCookie != nil {
		provider = providerCookie.Value
	}

	// Clear cookies
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/", MaxAge: -1})
	http.SetCookie(w, &http.Cookie{Name: "oauth_provider", Path: "/", MaxAge: -1})

	if errParam != "" {
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=%s", provider, errParam), 302)
		return
	}

	// Verify state
	stateCookie, _ := r.Cookie("oauth_state")
	if stateCookie == nil || stateCookie.Value != state {
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=Invalid+state+token", provider), 302)
		return
	}

	cfg := getOAuthConfig(provider)
	if cfg == nil {
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=Provider+not+configured", provider), 302)
		return
	}

	// Exchange code for token
	callbackURL := fmt.Sprintf("%s/v1/oauth/callback", oauthCallbackBase)
	tokenBody := fmt.Sprintf("grant_type=authorization_code&code=%s&redirect_uri=%s&client_id=%s&client_secret=%s",
		code, callbackURL, cfg.ClientID, cfg.ClientSecret)

	client := &http.Client{Timeout: 15 * time.Second}
	tokenReq, _ := http.NewRequest("POST", cfg.TokenURL, strings.NewReader(tokenBody))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if provider == "github" {
		tokenReq.Header.Set("Accept", "application/json")
	}

	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=Token+exchange+failed", provider), 302)
		return
	}
	defer tokenResp.Body.Close()

	var tokenData map[string]interface{}
	json.NewDecoder(tokenResp.Body).Decode(&tokenData)

	// Check for errors in token response
	if tokenData["error"] != nil {
		errMsg, _ := tokenData["error_description"].(string)
		if errMsg == "" {
			errMsg, _ = tokenData["error"].(string)
		}
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=%s", provider, errMsg), 302)
		return
	}

	// Store the token as an encrypted integration
	tokenJSON, _ := json.Marshal(tokenData)
	encrypted, nonce, err := s.encryptCredential(tokenJSON)
	if err != nil {
		http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=fail&error=Encryption+failed", provider), 302)
		return
	}

	// Determine display name from token data
	displayName := provider
	switch provider {
	case "stripe":
		if acct, ok := tokenData["stripe_user_id"].(string); ok {
			displayName = fmt.Sprintf("Stripe (%s)", acct)
		}
	case "github":
		// Fetch user info
		if token, ok := tokenData["access_token"].(string); ok {
			userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
			userReq.Header.Set("Authorization", "Bearer "+token)
			if userResp, err := client.Do(userReq); err == nil {
				var user map[string]interface{}
				json.NewDecoder(userResp.Body).Decode(&user)
				userResp.Body.Close()
				if login, ok := user["login"].(string); ok {
					displayName = fmt.Sprintf("GitHub (@%s)", login)
				}
			}
		}
	case "google":
		displayName = "YouTube"
	case "freshbooks":
		if token, ok := tokenData["access_token"].(string); ok {
			req, _ := http.NewRequest("GET", "https://api.freshbooks.com/auth/api/v1/users/me", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			if resp, err := client.Do(req); err == nil {
				var me map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&me)
				resp.Body.Close()
				if r, ok := me["response"].(map[string]interface{}); ok {
					if fn, ok := r["first_name"].(string); ok {
						ln, _ := r["last_name"].(string)
						displayName = fmt.Sprintf("FreshBooks (%s %s)", fn, ln)
					}
				}
			}
		}
	case "hubspot":
		if token, ok := tokenData["access_token"].(string); ok {
			req, _ := http.NewRequest("GET", "https://api.hubapi.com/oauth/v1/access-tokens/"+token, nil)
			if resp, err := client.Do(req); err == nil {
				var info map[string]interface{}
				json.NewDecoder(resp.Body).Decode(&info)
				resp.Body.Close()
				if name, ok := info["hub_domain"].(string); ok {
					displayName = fmt.Sprintf("HubSpot (%s)", name)
				}
			}
		}
	}

	id := fmt.Sprintf("int-%d", time.Now().UnixNano())
	now := time.Now().UTC().Format(time.RFC3339)
	nextPoll := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)

	// Map provider to scoreboard provider ID
	scoreProvider := provider
	if provider == "google" {
		scoreProvider = "youtube"
	}

	s.mu.Lock()
	s.db.Exec(`INSERT INTO integrations (id, user_id, provider, auth_type, encrypted_data, nonce,
		display_name, poll_interval_seconds, created_at, updated_at, next_poll_at, scopes)
		VALUES (?, 'default', ?, 'oauth2', ?, ?, ?, 1800, ?, ?, ?, ?)`,
		id, scoreProvider, encrypted, nonce, displayName, now, now, nextPoll, cfg.Scopes)
	s.mu.Unlock()

	// For GitHub: auto-create webhooks on user's repos
	if provider == "github" {
		go s.setupGitHubWebhooks(tokenData)
	}

	log.Printf("OAuth: %s connected as %s (integration %s)", provider, displayName, id)
	http.Redirect(w, r, fmt.Sprintf("/?oauth=%s&oauth_status=ok", scoreProvider), 302)
}

// setupGitHubWebhooks creates webhook on connected user's repos after OAuth
func (s *Server) setupGitHubWebhooks(tokenData map[string]interface{}) {
	token, ok := tokenData["access_token"].(string)
	if !ok || token == "" {
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	webhookURL := fmt.Sprintf("%s/v1/webhooks/github", oauthCallbackBase)

	// List user's repos
	req, _ := http.NewRequest("GET", "https://api.github.com/user/repos?per_page=100&sort=pushed", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("GitHub OAuth: failed to list repos: %v", err)
		return
	}
	defer resp.Body.Close()

	var repos []struct {
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		Permissions struct{ Admin bool `json:"admin"` } `json:"permissions"`
	}
	json.NewDecoder(resp.Body).Decode(&repos)

	created := 0
	for _, repo := range repos {
		if !repo.Permissions.Admin {
			continue // Can't create webhooks without admin
		}

		// Create webhook
		hookBody, _ := json.Marshal(map[string]interface{}{
			"name":   "web",
			"active": true,
			"events": []string{"push", "pull_request", "release"},
			"config": map[string]string{
				"url":          webhookURL,
				"content_type": "json",
			},
		})

		hookReq, _ := http.NewRequest("POST", fmt.Sprintf("https://api.github.com/repos/%s/hooks", repo.FullName), strings.NewReader(string(hookBody)))
		hookReq.Header.Set("Authorization", "Bearer "+token)
		hookReq.Header.Set("Content-Type", "application/json")
		hookResp, err := client.Do(hookReq)
		if err != nil {
			continue
		}
		hookResp.Body.Close()
		if hookResp.StatusCode == 201 {
			created++
			log.Printf("GitHub OAuth: webhook created on %s", repo.FullName)
		}
	}
	log.Printf("GitHub OAuth: created webhooks on %d repos", created)
}

func (s *Server) handleSSOCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`<!DOCTYPE html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Signing in...</title>
<style>
body{background:#0a0a12;color:#ddd;font-family:system-ui;display:flex;justify-content:center;align-items:center;height:100dvh;margin:0}
.card{text-align:center;padding:32px}
.spinner{width:32px;height:32px;border:3px solid #222;border-top-color:#7c7cff;border-radius:50%;animation:spin .8s linear infinite;margin:0 auto 16px}
@keyframes spin{to{transform:rotate(360deg)}}
h2{font-size:18px;margin:0 0 8px}
p{font-size:13px;color:#666;margin:0}
.ok{color:#2ecc71} .fail{color:#ff4444}
</style></head><body>
<div class="card" id="card">
<div class="spinner" id="spinner"></div>
<h2 id="msg">Signing in...</h2>
<p id="detail">Connecting to Startempire Wire</p>
</div>
<script>
(function(){
  var hash = window.location.hash.substring(1);
  var params = new URLSearchParams(hash);
  var token = params.get('token');
  var userStr = params.get('user');

  if (!token) {
    document.getElementById('spinner').style.display='none';
    document.getElementById('msg').className='fail';
    document.getElementById('msg').textContent='No token received';
    document.getElementById('detail').textContent='Try signing in again from startempirewire.com';
    return;
  }

  // Store JWT + user data
  localStorage.setItem('wb_token', token);
  if (userStr) {
    try { localStorage.setItem('wb_user', decodeURIComponent(userStr)); } catch(e) {}
  }
  // Set expiry (24h)
  localStorage.setItem('wb_token_exp', String(Date.now() + 86400000));

  // Clean URL and redirect
  document.getElementById('msg').className='ok';
  document.getElementById('msg').textContent='✓ Signed in';
  document.getElementById('detail').textContent='Redirecting to scoreboard...';
  document.getElementById('spinner').style.display='none';

  setTimeout(function(){ window.location.href = '/'; }, 800);
})();
</script></body></html>`))
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
			"DOCS_PUBLISHED": 4, "EXTENSION_PUBLISHED": 5, "CODE_PUBLISHED": 3,
			"CAMPAIGN_SENT": 5, "DEPLOY": 3, "EMAIL_HEALTH": 1,
		},
		"revenue": {
			"PAYMENT_RECEIVED": 10, "SUBSCRIPTION_CREATED": 12, "DEAL_CLOSED": 8,
			"PROPOSAL_SENT": 4, "INVOICE_PAID": 8, "PAYOUT_RECEIVED": 2,
			"PAYMENT_FAILED": 0, "REFUND_ISSUED": -2, "EXPENSE_RECORDED": 0,
		},
		"systems": {
			"AUTOMATION_DEPLOYED": 6, "SOP_DOCUMENTED": 4, "TOOL_INTEGRATED": 5,
			"DELEGATION_COMPLETED": 6, "MONITORING_ENABLED": 4,
			"DEPLOY": 3, "SERVICE_HEALTH": 2, "INFRA_CHECK": 2,
			"MEMORY_FIX": 5, "FEATURE_SHIPPED": 5,
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
	var currentStatus, lane, evtType, title, source string
	var confidence float64
	err := s.db.QueryRow("SELECT status, lane, event_type, artifact_title, confidence, source FROM events WHERE id=?", eventID).
		Scan(&currentStatus, &lane, &evtType, &title, &confidence, &source)
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

		// Trust this source forever — approve once, in forever
		now := time.Now().UTC().Format(time.RFC3339)
		s.db.Exec(`INSERT INTO trusted_sources (source, approved_at, approved_count)
			VALUES (?, ?, 1)
			ON CONFLICT(source) DO UPDATE SET approved_count = approved_count + 1`, source, now)
		s.mu.Unlock()

		// Recalculate daily score
		var ts string
		s.db.QueryRow("SELECT timestamp FROM events WHERE id=?", eventID).Scan(&ts)
		date := ts[:10] // YYYY-MM-DD
		s.updateDailyScore(date)
		s.updateStreak(date, title)
		s.recalcSeason()

		daily := s.getDailyScore(date)

		// Feed approval to pairing engine
		if s.pairing != nil {
			var createdAt string
			s.db.QueryRow("SELECT created_at FROM events WHERE id=?", eventID).Scan(&createdAt)
			latency := 0.0
			if ct, err := time.Parse(time.RFC3339, createdAt); err == nil {
				latency = time.Since(ct).Seconds()
			}
			s.pairing.Ingest(Signal{
				Type:      SignalApproval,
				Source:    "scoreboard",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"action":          "approve",
					"event_id":        eventID,
					"latency_seconds": latency,
				},
			})
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok": true, "event_id": eventID, "action": "approved",
			"score_delta": scoreDelta, "new_daily_score": daily.ExecutionScore,
		})
	} else {
		s.db.Exec("UPDATE events SET status='rejected', score_delta=0 WHERE id=?", eventID)
		s.mu.Unlock()

		// Feed rejection to pairing engine
		if s.pairing != nil {
			s.pairing.Ingest(Signal{
				Type:      SignalApproval,
				Source:    "scoreboard",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"action":   "reject",
					"event_id": eventID,
				},
			})
		}

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

// ─── POST /v1/chat — Wirebot Chat Proxy ─────────────────────────────────
// Proxies to OpenClaw gateway with session stickiness.
// The "user" field ensures persistent sessions with full memory retention.

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	// Auth already enforced by s.auth() wrapper (tier >= 3)
	ac := resolveAuth(r)
	userID := "operator"
	if ac.UserID > 0 {
		userID = fmt.Sprintf("member-%d", ac.UserID)
	}

	// Parse incoming request
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
		SessionID string `json:"session_id,omitempty"`
		Stream    bool   `json:"stream,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, 400)
		return
	}
	if len(req.Messages) == 0 {
		http.Error(w, `{"error":"messages required"}`, 400)
		return
	}

	// Persist session + save user message
	sessionID := s.getOrCreateSession(userID, req.SessionID)
	lastMsg := req.Messages[len(req.Messages)-1]
	if lastMsg.Role == "user" {
		s.saveMessage(sessionID, "user", lastMsg.Content)
		// Feed message to pairing engine for NLP analysis
		if s.pairing != nil {
			s.pairing.Ingest(Signal{
				Type:      SignalMessage,
				Source:    "chat",
				Timestamp: time.Now(),
				Content:   lastMsg.Content,
			})
		}
	}

	// Inject context — pairing protocol if incomplete, scoreboard state always
	contextMessages := s.buildChatContext(req.Messages)

	// Build proxy request to OpenClaw
	// "user" field = stable session key → persistent conversation with memory
	gatewayURL := "http://127.0.0.1:18789/v1/chat/completions"
	gatewayToken := os.Getenv("SCOREBOARD_TOKEN")
	if gatewayToken == "" {
		gatewayToken = "65b918ba-baf5-4996-8b53-6fb0f662a0c3"
	}

	proxyBody := map[string]interface{}{
		"messages":   contextMessages,
		"user":       "scoreboard:" + userID + ":" + sessionID, // stable per-session key
		"max_tokens": 2048,
	}
	if req.Stream {
		proxyBody["stream"] = true
	}

	bodyBytes, _ := json.Marshal(proxyBody)
	proxyReq, err := http.NewRequest("POST", gatewayURL, bytes.NewReader(bodyBytes))
	if err != nil {
		http.Error(w, `{"error":"Failed to create proxy request"}`, 502)
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("Authorization", "Bearer "+gatewayToken)
	proxyReq.Header.Set("x-openclaw-agent-id", "verious") // Use the Wirebot agent, not "main"

	client := &http.Client{Timeout: 120 * time.Second}

	if req.Stream {
		// SSE streaming — pipe through and capture full response for persistence
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, `{"error":"Gateway unavailable"}`, 502)
			return
		}
		defer proxyResp.Body.Close()

		// Prepend session_id as first SSE event
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(proxyResp.StatusCode)

		flusher, ok := w.(http.Flusher)
		// Send session_id as custom event
		fmt.Fprintf(w, "event: session\ndata: %s\n\n", sessionID)
		if ok { flusher.Flush() }

		var fullResponse strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := proxyResp.Body.Read(buf)
			if n > 0 {
				chunk := string(buf[:n])
				w.Write(buf[:n])
				if ok { flusher.Flush() }

				// Extract content deltas for persistence
				for _, line := range strings.Split(chunk, "\n") {
					if !strings.HasPrefix(line, "data: ") { continue }
					payload := strings.TrimPrefix(line, "data: ")
					if payload == "[DONE]" { continue }
					var sse struct {
						Choices []struct {
							Delta struct { Content string `json:"content"` } `json:"delta"`
						} `json:"choices"`
					}
					if json.Unmarshal([]byte(payload), &sse) == nil && len(sse.Choices) > 0 {
						fullResponse.WriteString(sse.Choices[0].Delta.Content)
					}
				}
			}
			if readErr != nil { break }
		}

		// Save assistant response
		if resp := fullResponse.String(); resp != "" {
			s.saveMessage(sessionID, "assistant", resp)
		}
	} else {
		// Non-streaming — simple proxy
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			http.Error(w, `{"error":"Gateway unavailable"}`, 502)
			return
		}
		defer proxyResp.Body.Close()

		respBody, _ := io.ReadAll(proxyResp.Body)

		// Save assistant response
		var result struct {
			Choices []struct {
				Message struct { Content string `json:"content"` } `json:"message"`
			} `json:"choices"`
		}
		if json.Unmarshal(respBody, &result) == nil && len(result.Choices) > 0 {
			s.saveMessage(sessionID, "assistant", result.Choices[0].Message.Content)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(proxyResp.StatusCode)
		w.Write(respBody)
	}
}

// ─── Chat Context Injection ──────────────────────────────────────────────

// buildChatContext prepends context system message before user messages.
// OpenClaw already injects agent identity from workspace (IDENTITY.md, SOUL.md).
// We only inject what the agent doesn't already know: pairing state + live scoreboard.
func (s *Server) buildChatContext(userMessages []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}) []map[string]string {
	var result []map[string]string

	var contextParts []string

	// --- Live scoreboard snapshot ---
	// Give Wirebot the current score so it doesn't have to call a tool for basic questions
	scoreData := s.getScoreSnapshot()
	if scoreData != "" {
		contextParts = append(contextParts, "LIVE SCOREBOARD (as of right now):\n"+scoreData)
	}

	// --- Pairing profile (v2 engine) ---
	if s.pairing != nil {
		pairingSummary := s.pairing.GetChatContextSummary()
		contextParts = append(contextParts, "FOUNDER PROFILE:\n"+pairingSummary)

		// If pairing is early (score < 20), also inject conversational guidance
		eff := s.pairing.GetEffectiveProfile()
		if eff.PairingScore < 20 {
			contextParts = append(contextParts, `PAIRING — EARLY STAGE:
The founder profile is still being built. When the operator mentions pairing, onboarding, or "getting to know you" — guide them to the assessment at /profile in the scoreboard app. You can also ask calibration questions conversationally:
- "What gives you energy in your business? What drains you?"
- "When you're under pressure, do you speed up or slow down?"
- "How do you prefer to get advice — bottom-line first or full context?"
After EACH answer, call wirebot_remember to persist the fact.`)
		}
	} else {
		// Fallback: read old v1 pairing.json
		pairingData, err := os.ReadFile("/home/wirebot/clawd/pairing.json")
		if err == nil {
			var p struct {
				Completed bool                   `json:"completed"`
				Answers   map[string]interface{} `json:"answers"`
			}
			if json.Unmarshal(pairingData, &p) == nil && !p.Completed {
				contextParts = append(contextParts, "Pairing incomplete. Guide operator to /profile for assessment.")
			}
		}
	}

	if len(contextParts) > 0 {
		result = append(result, map[string]string{
			"role":    "system",
			"content": strings.Join(contextParts, "\n\n---\n\n"),
		})
	}

	// Append user's messages
	for _, m := range userMessages {
		result = append(result, map[string]string{"role": m.Role, "content": m.Content})
	}

	return result
}

// getScoreSnapshot returns a compact text summary of current scoreboard state.
func (s *Server) getScoreSnapshot() string {
	today := operatorToday()

	var score, shipping, distribution, revenue, systems int
	err := s.db.QueryRow(`SELECT COALESCE(execution_score,0), COALESCE(shipping_score,0),
		COALESCE(distribution_score,0), COALESCE(revenue_score,0), COALESCE(systems_score,0)
		FROM daily_scores WHERE date = ?`, today).Scan(&score, &shipping, &distribution, &revenue, &systems)
	if err != nil {
		return ""
	}

	var streak int
	s.db.QueryRow("SELECT COALESCE(current,0) FROM streaks LIMIT 1").Scan(&streak)

	var intent string
	s.db.QueryRow("SELECT COALESCE(intent,'') FROM daily_scores WHERE date = ?", today).Scan(&intent)

	return fmt.Sprintf("Score: %d/100 | Shipping: %d/40, Distribution: %d/25, Revenue: %d/20, Systems: %d/15 | Streak: %d days | Intent: %s",
		score, shipping, distribution, revenue, systems, streak, intent)
}

// ─── Chat Session Persistence ───────────────────────────────────────────

func generateSessionID() string {
	b := make([]byte, 12)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// GET /v1/chat/sessions — list sessions, POST — create
func (s *Server) handleChatSessions(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" { return }

	ac := resolveAuth(r)
	userID := "operator"
	if ac.UserID > 0 {
		userID = fmt.Sprintf("member-%d", ac.UserID)
	}

	if r.Method == "GET" {
		rows, err := s.db.Query(`SELECT id, title, created_at, updated_at, message_count, pinned
			FROM chat_sessions WHERE user_id = ? ORDER BY updated_at DESC LIMIT 50`, userID)
		if err != nil {
			http.Error(w, `{"error":"db error"}`, 500)
			return
		}
		defer rows.Close()

		type SessionSummary struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			Created  string `json:"created_at"`
			Updated  string `json:"updated_at"`
			Messages int    `json:"message_count"`
			Pinned   bool   `json:"pinned"`
		}
		var sessions []SessionSummary
		for rows.Next() {
			var ss SessionSummary
			rows.Scan(&ss.ID, &ss.Title, &ss.Created, &ss.Updated, &ss.Messages, &ss.Pinned)
			sessions = append(sessions, ss)
		}
		if sessions == nil {
			sessions = []SessionSummary{}
		}
		json.NewEncoder(w).Encode(sessions)
		return
	}

	if r.Method == "POST" {
		id := generateSessionID()
		now := time.Now().UTC().Format(time.RFC3339)
		s.db.Exec(`INSERT INTO chat_sessions (id, user_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
			id, userID, "New Chat", now, now)
		json.NewEncoder(w).Encode(map[string]string{"id": id})
		return
	}

	http.Error(w, `{"error":"method not allowed"}`, 405)
}

// GET /v1/chat/sessions/{id} — load messages, DELETE — remove
func (s *Server) handleChatSession(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" { return }

	// Extract session ID from path
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/v1/chat/sessions/"), "/")
	sessionID := parts[0]
	if sessionID == "" {
		http.Error(w, `{"error":"session id required"}`, 400)
		return
	}

	if r.Method == "GET" {
		type Msg struct {
			Role    string `json:"role"`
			Content string `json:"content"`
			Time    string `json:"created_at"`
		}
		rows, err := s.db.Query(`SELECT role, content, created_at FROM chat_messages
			WHERE session_id = ? ORDER BY id ASC`, sessionID)
		if err != nil {
			http.Error(w, `{"error":"db error"}`, 500)
			return
		}
		defer rows.Close()

		var msgs []Msg
		for rows.Next() {
			var m Msg
			rows.Scan(&m.Role, &m.Content, &m.Time)
			msgs = append(msgs, m)
		}
		if msgs == nil {
			msgs = []Msg{}
		}

		// Get session metadata
		var title string
		var pinned bool
		s.db.QueryRow(`SELECT title, pinned FROM chat_sessions WHERE id = ?`, sessionID).Scan(&title, &pinned)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":       sessionID,
			"title":    title,
			"pinned":   pinned,
			"messages": msgs,
		})
		return
	}

	if r.Method == "DELETE" {
		s.db.Exec("DELETE FROM chat_messages WHERE session_id = ?", sessionID)
		s.db.Exec("DELETE FROM chat_sessions WHERE id = ?", sessionID)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
		return
	}

	if r.Method == "PATCH" {
		var body struct {
			Title  *string `json:"title,omitempty"`
			Pinned *bool   `json:"pinned,omitempty"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Title != nil {
			s.db.Exec("UPDATE chat_sessions SET title = ? WHERE id = ?", *body.Title, sessionID)
		}
		if body.Pinned != nil {
			s.db.Exec("UPDATE chat_sessions SET pinned = ? WHERE id = ?", *body.Pinned, sessionID)
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
		return
	}

	http.Error(w, `{"error":"method not allowed"}`, 405)
}

// saveMessage persists a chat message and auto-titles the session.
func (s *Server) saveMessage(sessionID, role, content string) {
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec(`INSERT INTO chat_messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`,
		sessionID, role, content, now)
	s.db.Exec(`UPDATE chat_sessions SET updated_at = ?, message_count = message_count + 1 WHERE id = ?`, now, sessionID)

	// Auto-title from first user message
	if role == "user" {
		var count int
		s.db.QueryRow("SELECT message_count FROM chat_sessions WHERE id = ?", sessionID).Scan(&count)
		if count <= 1 {
			title := content
			if len(title) > 60 {
				title = title[:57] + "..."
			}
			s.db.Exec("UPDATE chat_sessions SET title = ? WHERE id = ?", title, sessionID)
		}
	}
}

// getOrCreateSession gets existing session or creates new one for a user.
func (s *Server) getOrCreateSession(userID, sessionID string) string {
	if sessionID != "" {
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM chat_sessions WHERE id = ?", sessionID).Scan(&exists)
		if exists > 0 {
			return sessionID
		}
	}
	// Create new
	id := generateSessionID()
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec(`INSERT INTO chat_sessions (id, user_id, title, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, "New Chat", now, now)
	return id
}

// ─── GET /v1/pairing/status — Check if pairing is complete ──────────────

func (s *Server) handlePairingStatus(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" { return }
	if r.Method != "GET" {
		http.Error(w, `{"error":"GET only"}`, 405)
		return
	}

	// Use v2 pairing engine if available
	if s.pairing != nil {
		s.pairing.mu.RLock()
		p := s.pairing.profile
		score := p.PairingScore.Composite
		completed := score >= 60
		answered := len(p.Answers)
		level := p.PairingScore.Level
		s.pairing.mu.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"completed": completed,
			"score":     score,
			"level":     level,
			"answered":  answered,
			"total":     40, // 12+8+6+6+8 = 40 assessment items across 5 instruments
		})
		return
	}

	// Fallback: old v1 pairing.json (shouldn't reach here normally)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"completed": false,
		"score":     0,
		"answered":  0,
		"total":     40,
	})
}

// ─── POST /v1/lock — EOD Score Lock ─────────────────────────────────────

// handleChecklist serves task data for the Dashboard view.
// Reads from the checklist.json file maintained by the gateway plugin.
func (s *Server) handleChecklist(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == "OPTIONS" {
		return
	}

	action := r.URL.Query().Get("action")
	stageFilter := r.URL.Query().Get("stage")

	data, err := os.ReadFile(checklistPath)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": []interface{}{}, "total": 0, "completed": 0, "percent": 0, "stage": "launch",
		})
		return
	}

	var cl map[string]interface{}
	if json.Unmarshal(data, &cl) != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": []interface{}{}, "total": 0, "completed": 0, "percent": 0, "stage": "launch",
		})
		return
	}

	tasks, _ := cl["tasks"].([]interface{})
	currentStage, _ := cl["stage"].(string)
	if currentStage == "" {
		currentStage = "launch"
	}

	// Filter and count
	var filtered []interface{}
	total := 0
	completed := 0
	var nextTask interface{}

	for _, t := range tasks {
		tm, ok := t.(map[string]interface{})
		if !ok {
			continue
		}

		status, _ := tm["status"].(string)
		taskStage, _ := tm["stage"].(string)

		// Apply stage filter
		if stageFilter != "" && taskStage != stageFilter {
			continue
		}

		total++
		if status == "completed" || status == "done" {
			completed++
		} else if nextTask == nil && status != "skipped" {
			nextTask = tm
		}

		filtered = append(filtered, tm)
	}

	pct := 0
	if total > 0 {
		pct = completed * 100 / total
	}

	switch action {
	case "summary":
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":     total,
			"completed": completed,
			"percent":   pct,
			"stage":     currentStage,
			"next_task": nextTask,
		})
	case "daily":
		// Return uncompleted tasks for today (up to 5)
		var daily []interface{}
		for _, t := range filtered {
			tm, _ := t.(map[string]interface{})
			status, _ := tm["status"].(string)
			if status != "completed" && status != "done" && status != "skipped" {
				daily = append(daily, tm)
				if len(daily) >= 5 {
					break
				}
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks": daily,
			"total": len(daily),
		})
	case "complete":
		// Mark task as done
		taskID := r.URL.Query().Get("id")
		if taskID == "" || r.Method != "POST" {
			http.Error(w, `{"error":"POST with id required"}`, 400)
			return
		}
		for _, t := range tasks {
			tm, _ := t.(map[string]interface{})
			if id, _ := tm["id"].(string); id == taskID {
				tm["status"] = "completed"
			}
		}
		cl["tasks"] = tasks
		updated, _ := json.MarshalIndent(cl, "", "  ")
		os.WriteFile(checklistPath, updated, 0644)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":   true,
			"id":   taskID,
			"note": "Task marked complete",
		})
	case "grouped":
		// Return tasks grouped by category with per-category progress
		catMap := map[string]map[string]interface{}{}
		catOrder := []string{}
		catLabels := map[string]string{
			"idea-identity": "Business Identity", "idea-research": "Market Research",
			"idea-planning": "Business Planning", "idea-finance": "Financial Planning",
			"idea-legal": "Legal Foundation", "launch-brand": "Brand & Marketing",
			"launch-digital": "Digital Presence", "launch-ops": "Operations Setup",
			"launch-product": "Product/Service Ready", "launch-sales": "Sales Pipeline",
			"growth-scale": "Scaling Operations", "growth-team": "Team Building",
			"growth-revenue": "Revenue Optimization", "growth-systems": "Systems & Automation",
			"growth-network": "Network & Partnerships",
		}
		catIcons := map[string]string{
			"idea-identity": "🎯", "idea-research": "🔍", "idea-planning": "📋",
			"idea-finance": "💰", "idea-legal": "⚖️", "launch-brand": "🎨",
			"launch-digital": "🌐", "launch-ops": "⚙️", "launch-product": "📦",
			"launch-sales": "📈", "growth-scale": "🚀", "growth-team": "👥",
			"growth-revenue": "💎", "growth-systems": "🔧", "growth-network": "🤝",
		}

		for _, t := range filtered {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			cat, _ := tm["category"].(string)
			if cat == "" {
				cat = "uncategorized"
			}
			if _, exists := catMap[cat]; !exists {
				catMap[cat] = map[string]interface{}{
					"id": cat, "label": catLabels[cat], "icon": catIcons[cat],
					"tasks": []interface{}{}, "total": 0, "completed": 0,
				}
				if catMap[cat]["label"] == nil || catMap[cat]["label"] == "" {
					catMap[cat]["label"] = cat
				}
				if catMap[cat]["icon"] == nil || catMap[cat]["icon"] == "" {
					catMap[cat]["icon"] = "📌"
				}
				catOrder = append(catOrder, cat)
			}
			entry := catMap[cat]
			tasks := entry["tasks"].([]interface{})
			entry["tasks"] = append(tasks, tm)
			entry["total"] = entry["total"].(int) + 1
			st, _ := tm["status"].(string)
			if st == "completed" || st == "done" {
				entry["completed"] = entry["completed"].(int) + 1
			}
		}

		// Build ordered result
		var groups []interface{}
		for _, cat := range catOrder {
			entry := catMap[cat]
			t := entry["total"].(int)
			c := entry["completed"].(int)
			p := 0
			if t > 0 {
				p = c * 100 / t
			}
			entry["percent"] = p
			groups = append(groups, entry)
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"categories": groups,
			"total":      total,
			"completed":  completed,
			"percent":    pct,
			"stage":      currentStage,
			"next_task":  nextTask,
		})
	default: // "list" or empty
		json.NewEncoder(w).Encode(map[string]interface{}{
			"tasks":     filtered,
			"total":     total,
			"completed": completed,
			"percent":   pct,
			"stage":     currentStage,
			"next_task": nextTask,
		})
	}
}

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
			poll_interval_seconds, last_used_at, last_error, business_id, created_at FROM integrations ORDER BY created_at`)
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
				&i.PollInterval, &i.LastUsedAt, &i.LastError, &i.BusinessID, &i.CreatedAt)
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
			BusinessID  string `json:"business_id"` // optional: which business this account belongs to
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
			display_name, sensitivity, poll_interval_seconds, config, business_id, created_at, updated_at, next_poll_at)
			VALUES (?, 'default', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			id, body.Provider, body.AuthType, encrypted, nonce,
			body.DisplayName, sensitivity, pollInterval, config, body.BusinessID, now, now, nextPoll)
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
	// Integration poller: immediately then every 60 seconds
	go func() {
		time.Sleep(5 * time.Second)
		s.pollDueIntegrations()
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			s.pollDueIntegrations()
		}
	}()

	// Systems health check: on startup then every 6 hours
	go func() {
		time.Sleep(10 * time.Second)
		s.checkSystemsHealth()
		ticker := time.NewTicker(6 * time.Hour)
		for range ticker.C {
			s.checkSystemsHealth()
		}
	}()
}

func (s *Server) checkSystemsHealth() {
	today := time.Now().Format("2006-01-02")

	// Only emit one health event per day
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE event_type='SERVICE_HEALTH' AND date(timestamp)=?`, today).Scan(&count)
	if count > 0 {
		return
	}

	// Check services
	services := []struct{ name, check string }{
		{"gateway", "http://127.0.0.1:18789/__openclaw__/health"},
		{"mem0", "http://127.0.0.1:8200/health"},
		{"syncd", "http://127.0.0.1:8201/health"},
		{"scoreboard", "http://127.0.0.1:8100/v1/score?token=" + os.Getenv("SCOREBOARD_TOKEN")},
	}

	up := 0
	total := len(services)
	client := &http.Client{Timeout: 3 * time.Second}
	for _, svc := range services {
		resp, err := client.Get(svc.check)
		if err == nil && resp.StatusCode < 400 {
			up++
			resp.Body.Close()
		}
	}

	delta := 0
	if up == total {
		delta = 2 // All services healthy
	} else if up > 0 {
		delta = 1 // Partial
	}

	title := fmt.Sprintf("Systems health: %d/%d services up", up, total)
	id := fmt.Sprintf("evt-health-%s", today)
	now := time.Now().UTC().Format(time.RFC3339)

	s.db.Exec(`INSERT OR IGNORE INTO events (id, event_type, lane, source, timestamp, artifact_title, score_delta, status, created_at)
		VALUES (?, 'SERVICE_HEALTH', 'systems', 'automated', ?, ?, ?, 'pending', ?)`,
		id, now, title, delta, now)

	log.Printf("Systems health: %d/%d up, +%d pts", up, total, delta)
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
		case "youtube", "youtube_key":
			pollErr = s.pollYouTube(id, credential, config, lastPoll)
		case "plaid":
			var creds map[string]string
			json.Unmarshal([]byte(credential), &creds)
			pollErr = s.pollPlaid(id, creds["access_token"], config, lastPoll)
		case "posthog":
			pollErr = s.pollPostHog(id, credential, config, lastPoll)
		case "uptimerobot":
			pollErr = s.pollUptimeRobot(id, credential, lastPoll)
		case "rescuetime":
			pollErr = s.pollRescueTime(id, credential, lastPoll)
		case "woocommerce":
			pollErr = s.pollWooCommerce(id, credential, config, lastPoll)
		case "cloudflare":
			pollErr = s.pollCloudflare(id, credential, config, lastPoll)
		case "hubspot":
			pollErr = s.pollHubSpot(id, credential, lastPoll)
		case "discord_webhook":
			pollErr = s.pollDiscord(id, credential, lastPoll)
		case "sendy":
			pollErr = s.pollSendy(id, credential, config, lastPoll)
		case "freshbooks":
			pollErr = s.pollFreshBooks(id, credential, config, lastPoll)
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
			// Feed integration data into pairing engine
			s.pairing.Ingest(Signal{
				Type:      SignalAccount,
				Source:    provider,
				Timestamp: time.Now(),
				Content:   "",
				Features:  map[string]float64{},
				Metadata: map[string]interface{}{
					"provider":       provider,
					"integration_id": id,
					"status":         "active",
				},
			})
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

// ─── PostHog Poller ──────────────────────────────────────────────────────

func (s *Server) pollPostHog(integrationID, apiKey, configJSON, lastPoll string) error {
	var cfg struct {
		Host string `json:"host"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	host := cfg.Host
	if host == "" {
		host = "https://data.philoveracity.com"
	}

	// Get event count and active users for the last period
	since := time.Now().Add(-24 * time.Hour).Format("2006-01-02")
	if lastPoll != "" {
		if t, err := time.Parse(time.RFC3339, lastPoll); err == nil {
			since = t.Format("2006-01-02")
		}
	}

	// Insights query: pageviews + unique users
	client := &http.Client{Timeout: 15 * time.Second}
	url := fmt.Sprintf("%s/api/projects/@current/insights/trend/?events=[{\"id\":\"$pageview\"}]&date_from=%s&date_to=now", host, since)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("posthog api: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Extract total pageviews from results
	totalPageviews := 0
	if results, ok := result["result"].([]interface{}); ok && len(results) > 0 {
		if first, ok := results[0].(map[string]interface{}); ok {
			if counts, ok := first["data"].([]interface{}); ok {
				for _, c := range counts {
					if v, ok := c.(float64); ok {
						totalPageviews += int(v)
					}
				}
			}
		}
	}

	// Store as a systems health snapshot (not individually scored per-pageview)
	now := time.Now().UTC().Format(time.RFC3339)
	today := operatorToday()
	evtID := fmt.Sprintf("evt-posthog-%s-%s", integrationID[:15], today)

	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
	if exists > 0 {
		return nil // Already logged today
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"pageviews": totalPageviews,
		"period":    since + " to " + time.Now().Format("2006-01-02"),
		"host":      host,
	})

	scoreDelta := 0
	title := fmt.Sprintf("📊 Analytics: %d pageviews", totalPageviews)
	if totalPageviews > 0 {
		scoreDelta = 2 // Systems lane: site is alive and getting traffic
		if totalPageviews > 100 {
			scoreDelta = 3
		}
	}

	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evtID, "ANALYTICS_SNAPSHOT", "systems", "posthog-poller", now,
		title, 0.9, `["posthog_api"]`, "STRONG",
		scoreDelta, string(meta), now, "approved")
	s.mu.Unlock()

	s.updateDailyScore(today)
	log.Printf("PostHog: %d pageviews since %s", totalPageviews, since)
	return nil
}

// ─── UptimeRobot Poller ─────────────────────────────────────────────────

func (s *Server) pollUptimeRobot(integrationID, apiKey, lastPoll string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	body := strings.NewReader(fmt.Sprintf("api_key=%s&format=json&all_time_uptime_ratio=1&custom_uptime_ratios=1-7-30", apiKey))
	req, _ := http.NewRequest("POST", "https://api.uptimerobot.com/v2/getMonitors", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("uptimerobot api: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Stat     string `json:"stat"`
		Monitors []struct {
			ID                 int    `json:"id"`
			FriendlyName       string `json:"friendly_name"`
			URL                string `json:"url"`
			Status             int    `json:"status"`
			AllTimeUptimeRatio string `json:"all_time_uptime_ratio"`
			CustomUptimeRatio  string `json:"custom_uptime_ratio"`
		} `json:"monitors"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Stat != "ok" {
		return fmt.Errorf("uptimerobot: stat=%s", result.Stat)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := operatorToday()
	evtID := fmt.Sprintf("evt-uptime-%s-%s", integrationID[:15], today)

	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
	if exists > 0 {
		return nil
	}

	totalMonitors := len(result.Monitors)
	upCount := 0
	downCount := 0
	monitorNames := []string{}
	for _, m := range result.Monitors {
		if m.Status == 2 { // 2 = up
			upCount++
		} else if m.Status == 9 { // 9 = down
			downCount++
		}
		monitorNames = append(monitorNames, m.FriendlyName)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"total_monitors": totalMonitors,
		"up":             upCount,
		"down":           downCount,
		"monitors":       monitorNames,
	})

	scoreDelta := 0
	title := fmt.Sprintf("🟢 Uptime: %d/%d monitors up", upCount, totalMonitors)
	if downCount > 0 {
		title = fmt.Sprintf("🔴 Uptime: %d DOWN, %d up of %d", downCount, upCount, totalMonitors)
		scoreDelta = -2 // Penalty for downtime
	} else if upCount > 0 {
		scoreDelta = 2 // All systems healthy
	}

	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evtID, "UPTIME_CHECK", "systems", "uptimerobot-poller", now,
		title, 0.99, `["uptimerobot_api"]`, "STRONG",
		scoreDelta, string(meta), now, "approved")
	s.mu.Unlock()

	s.updateDailyScore(today)
	log.Printf("UptimeRobot: %d up, %d down of %d monitors", upCount, downCount, totalMonitors)
	return nil
}

// ─── RescueTime Poller ──────────────────────────────────────────────────

func (s *Server) pollRescueTime(integrationID, apiKey, lastPoll string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	_ = operatorToday()

	// Daily summary: productive hours, productivity pulse
	url := fmt.Sprintf("https://www.rescuetime.com/anapi/daily_summary_feed?key=%s&format=json", apiKey)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("rescuetime api: %w", err)
	}
	defer resp.Body.Close()

	var days []struct {
		Date                string  `json:"date"`
		ProductivityPulse   float64 `json:"productivity_pulse"`
		TotalHours          float64 `json:"total_hours"`
		VeryProductiveHours float64 `json:"very_productive_hours"`
		ProductiveHours     float64 `json:"productive_hours"`
		NeutralHours        float64 `json:"neutral_hours"`
		DistractingHours    float64 `json:"distracting_hours"`
		VeryDistractingHours float64 `json:"very_distracting_hours"`
		TotalDurationFormatted string `json:"total_duration_formatted"`
	}
	json.NewDecoder(resp.Body).Decode(&days)

	if len(days) == 0 {
		return nil // No data yet today
	}

	now := time.Now().UTC().Format(time.RFC3339)
	newCount := 0

	for _, day := range days {
		// Only process recent days (since last poll)
		if lastPoll != "" {
			if lp, err := time.Parse(time.RFC3339, lastPoll); err == nil {
				dp, _ := time.Parse("2006-01-02", day.Date)
				if dp.Before(lp) {
					continue
				}
			}
		}

		evtID := fmt.Sprintf("evt-rt-%s-%s", integrationID[:15], day.Date)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
		if exists > 0 {
			continue
		}

		focusHours := day.VeryProductiveHours + day.ProductiveHours
		meta, _ := json.Marshal(map[string]interface{}{
			"productivity_pulse": day.ProductivityPulse,
			"total_hours":       day.TotalHours,
			"focus_hours":       focusHours,
			"distracting_hours": day.DistractingHours + day.VeryDistractingHours,
			"date":              day.Date,
		})

		// Score based on focus hours
		scoreDelta := 0
		title := fmt.Sprintf("⏱️ Focus: %.1fh productive (pulse: %.0f%%)", focusHours, day.ProductivityPulse)
		if focusHours >= 6 {
			scoreDelta = 3 // Outstanding focus day
		} else if focusHours >= 4 {
			scoreDelta = 2 // Good focus day
		} else if focusHours >= 2 {
			scoreDelta = 1 // Minimal focus
		}
		// No negative scoring for low-focus days — that's wellness, not punishment

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_title, confidence, verifiers, verification_level,
			score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			evtID, "FOCUS_REPORT", "systems", "rescuetime-poller", now,
			title, 0.95, `["rescuetime_api"]`, "STRONG",
			scoreDelta, string(meta), now, "approved")
		s.mu.Unlock()

		s.updateDailyScore(day.Date)
		newCount++
	}

	if newCount > 0 {
		log.Printf("RescueTime: %d daily reports", newCount)
	}
	return nil
}

// ─── WooCommerce Poller ─────────────────────────────────────────────────

func (s *Server) pollWooCommerce(integrationID, consumerKey, configJSON, lastPoll string) error {
	var cfg struct {
		ConsumerSecret string `json:"consumer_secret"`
		StoreURL       string `json:"store_url"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.StoreURL == "" || cfg.ConsumerSecret == "" {
		return fmt.Errorf("store_url and consumer_secret required")
	}

	// Fetch recent orders
	after := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	if lastPoll != "" {
		after = lastPoll
	}

	url := fmt.Sprintf("%s/wp-json/wc/v3/orders?after=%s&per_page=50&orderby=date&order=desc",
		strings.TrimRight(cfg.StoreURL, "/"), after)

	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(consumerKey, cfg.ConsumerSecret)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("woocommerce api: %w", err)
	}
	defer resp.Body.Close()

	var orders []struct {
		ID          int    `json:"id"`
		Status      string `json:"status"`
		Total       string `json:"total"`
		Currency    string `json:"currency"`
		DateCreated string `json:"date_created"`
		Billing     struct {
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"billing"`
		LineItems []struct {
			Name string `json:"name"`
		} `json:"line_items"`
	}
	json.NewDecoder(resp.Body).Decode(&orders)

	newCount := 0
	for _, order := range orders {
		evtID := fmt.Sprintf("evt-woo-%d", order.ID)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
		if exists > 0 {
			continue
		}

		// Only score completed/processing orders
		if order.Status != "completed" && order.Status != "processing" {
			continue
		}

		totalF := 0.0
		fmt.Sscanf(order.Total, "%f", &totalF)

		items := []string{}
		for _, li := range order.LineItems {
			items = append(items, li.Name)
		}

		meta, _ := json.Marshal(map[string]interface{}{
			"order_id":  order.ID,
			"status":    order.Status,
			"total":     totalF,
			"currency":  order.Currency,
			"items":     strings.Join(items, ", "),
			"customer":  order.Billing.FirstName,
		})

		scoreDelta := 5
		if totalF >= 100 {
			scoreDelta = 8
		}
		if totalF >= 500 {
			scoreDelta = 10
		}

		title := fmt.Sprintf("🛒 Order #%d: $%.2f — %s", order.ID, totalF, strings.Join(items, ", "))
		now := time.Now().UTC().Format(time.RFC3339)

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_title, confidence, verifiers, verification_level,
			score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			evtID, "PAYMENT_RECEIVED", "revenue", "woocommerce-poller", order.DateCreated,
			title, 0.95, `["woocommerce_api"]`, "STRONG",
			scoreDelta, string(meta), now, "approved")
		s.mu.Unlock()

		orderTime, _ := time.Parse("2006-01-02T15:04:05", order.DateCreated)
		if !orderTime.IsZero() {
			s.updateDailyScore(orderTime.Format("2006-01-02"))
		}
		newCount++
	}

	if newCount > 0 {
		log.Printf("WooCommerce: %d new orders from %s", newCount, cfg.StoreURL)
	}
	return nil
}

// ─── Cloudflare Poller ──────────────────────────────────────────────────

func (s *Server) pollCloudflare(integrationID, apiToken, configJSON, lastPoll string) error {
	var cfg struct {
		AccountID string `json:"account_id"`
		Email     string `json:"email"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)

	client := &http.Client{Timeout: 15 * time.Second}

	// List zones for this account
	zonesURL := "https://api.cloudflare.com/client/v4/zones?per_page=50"
	if cfg.AccountID != "" {
		zonesURL += "&account.id=" + cfg.AccountID
	}
	req, _ := http.NewRequest("GET", zonesURL, nil)
	// Support both Bearer token and Global API Key auth
	if cfg.Email != "" {
		req.Header.Set("X-Auth-Email", cfg.Email)
		req.Header.Set("X-Auth-Key", apiToken)
	} else {
		req.Header.Set("Authorization", "Bearer "+apiToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cloudflare api: %w", err)
	}
	defer resp.Body.Close()

	var zonesResult struct {
		Success bool `json:"success"`
		Result  []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&zonesResult)

	if !zonesResult.Success {
		return fmt.Errorf("cloudflare zones: request failed")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	today := operatorToday()
	evtID := fmt.Sprintf("evt-cf-%s-%s", integrationID[:15], today)

	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
	if exists > 0 {
		return nil
	}

	activeZones := 0
	zoneNames := []string{}
	for _, z := range zonesResult.Result {
		if z.Status == "active" {
			activeZones++
		}
		zoneNames = append(zoneNames, z.Name)
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"total_zones":  len(zonesResult.Result),
		"active_zones": activeZones,
		"zones":        zoneNames,
	})

	scoreDelta := 0
	if activeZones > 0 {
		scoreDelta = 1 // Infra is running
	}
	title := fmt.Sprintf("🔥 Cloudflare: %d active zones — %s", activeZones, strings.Join(zoneNames, ", "))

	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evtID, "INFRA_CHECK", "systems", "cloudflare-poller", now,
		title, 0.9, `["cloudflare_api"]`, "STRONG",
		scoreDelta, string(meta), now, "approved")
	s.mu.Unlock()

	s.updateDailyScore(today)
	log.Printf("Cloudflare: %d active zones, %d total", activeZones, len(zonesResult.Result))
	return nil
}

// ─── HubSpot Poller ─────────────────────────────────────────────────────

func (s *Server) pollHubSpot(integrationID, apiToken, lastPoll string) error {
	client := &http.Client{Timeout: 15 * time.Second}

	// Get recent deals (pipeline)
	url := "https://api.hubapi.com/crm/v3/objects/deals?limit=20&properties=dealname,amount,dealstage,closedate,createdate&sorts=-createdate"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+apiToken)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("hubspot api: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			ID         string `json:"id"`
			Properties struct {
				DealName  string `json:"dealname"`
				Amount    string `json:"amount"`
				DealStage string `json:"dealstage"`
				CloseDate string `json:"closedate"`
				CreateDate string `json:"createdate"`
			} `json:"properties"`
		} `json:"results"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	newCount := 0
	for _, deal := range result.Results {
		evtID := fmt.Sprintf("evt-hs-deal-%s", deal.ID)
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
		if exists > 0 {
			continue
		}

		// Only track deals created after last poll
		if lastPoll != "" && deal.Properties.CreateDate != "" {
			lp, _ := time.Parse(time.RFC3339, lastPoll)
			cd, _ := time.Parse("2006-01-02T15:04:05.000Z", deal.Properties.CreateDate)
			if cd.Before(lp) {
				continue
			}
		}

		amount := 0.0
		fmt.Sscanf(deal.Properties.Amount, "%f", &amount)

		meta, _ := json.Marshal(map[string]interface{}{
			"deal_id":    deal.ID,
			"deal_stage": deal.Properties.DealStage,
			"amount":     amount,
		})

		scoreDelta := 3 // New pipeline deal
		title := fmt.Sprintf("🔶 Deal: %s", deal.Properties.DealName)
		if amount > 0 {
			title = fmt.Sprintf("🔶 Deal: %s ($%.0f)", deal.Properties.DealName, amount)
			if amount >= 1000 {
				scoreDelta = 5
			}
		}
		evtType := "DEAL_CREATED"
		if deal.Properties.DealStage == "closedwon" {
			evtType = "DEAL_WON"
			scoreDelta = 8
		}

		now := time.Now().UTC().Format(time.RFC3339)

		s.mu.Lock()
		s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
			artifact_title, confidence, verifiers, verification_level,
			score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			evtID, evtType, "revenue", "hubspot-poller", deal.Properties.CreateDate,
			title, 0.9, `["hubspot_api"]`, "STRONG",
			scoreDelta, string(meta), now, "approved")
		s.mu.Unlock()

		newCount++
	}

	if newCount > 0 {
		log.Printf("HubSpot: %d new deals", newCount)
	}
	return nil
}

// ─── Discord Webhook Poller ─────────────────────────────────────────────
// Note: Discord "poller" doesn't poll Discord — it checks if our outgoing
// webhook is still valid. Actual Discord events come via bot/webhook push.
// This just verifies the connection is alive.

func (s *Server) pollDiscord(integrationID, webhookURL, lastPoll string) error {
	if webhookURL == "" {
		return fmt.Errorf("webhook URL required")
	}

	// Validate webhook is still alive (GET returns webhook info)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(webhookURL)
	if err != nil {
		return fmt.Errorf("discord webhook check: %w", err)
	}
	defer resp.Body.Close()

	var wh struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
	}
	json.NewDecoder(resp.Body).Decode(&wh)

	if wh.ID == "" {
		return fmt.Errorf("discord webhook invalid or expired")
	}

	// Update integration config with webhook metadata
	now := time.Now().UTC().Format(time.RFC3339)
	s.db.Exec(`UPDATE integrations SET config=json_set(COALESCE(config,'{}'),
		'$.webhook_name', ?, '$.guild_id', ?, '$.channel_id', ?),
		last_used_at=? WHERE id=?`,
		wh.Name, wh.GuildID, wh.ChannelID, now, integrationID)

	log.Printf("Discord: webhook %s (%s) verified", wh.Name, wh.ID)
	return nil
}

// ─── Sendy Poller ────────────────────────────────────────────────────────

func (s *Server) pollSendy(integrationID, apiKey, configJSON, lastPoll string) error {
	var cfg struct {
		SendyURL string `json:"sendy_url"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.SendyURL == "" || apiKey == "" {
		return fmt.Errorf("sendy_url and api_key required")
	}
	baseURL := strings.TrimRight(cfg.SendyURL, "/")

	client := &http.Client{Timeout: 15 * time.Second}

	// 1. Get campaigns — Sendy doesn't have a campaign list API,
	//    but we can check subscriber count per brand/list
	//    and detect new campaigns via /api/campaigns/get.php (if available)

	// Get active subscriber count (needs list_id, but we can try brands)
	// Sendy /api/subscribers/active-subscriber-count.php
	// For now: poll the status endpoint to verify connection
	// and track subscriber counts

	// Try to get brand list (Sendy 6.x+)
	resp, err := client.PostForm(baseURL+"/api/brands/get-brands.php", map[string][]string{
		"api_key": {apiKey},
	})
	if err != nil {
		return fmt.Errorf("sendy brands: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	now := time.Now().UTC().Format(time.RFC3339)
	today := operatorToday()
	evtID := fmt.Sprintf("evt-sendy-%s-%s", integrationID[:15], today)

	var exists int
	s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", evtID).Scan(&exists)
	if exists > 0 {
		return nil
	}

	// Sendy returns brands as {"brand1": {...}, "brand2": {...}} — NOT an array
	var brandsMap map[string]struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	jsonErr := json.Unmarshal(body, &brandsMap)

	meta := map[string]interface{}{
		"sendy_url": baseURL,
	}

	if jsonErr == nil && len(brandsMap) > 0 {
		brandNames := []string{}
		for _, b := range brandsMap {
			brandNames = append(brandNames, b.Name)
		}
		meta["brands"] = brandNames
		meta["brand_count"] = len(brandsMap)
		log.Printf("Sendy: %d brands at %s — %s", len(brandsMap), baseURL, strings.Join(brandNames, ", "))
	} else {
		bodyStr := string(body)
		if strings.Contains(bodyStr, "No data passed") || strings.Contains(bodyStr, "API key") || strings.Contains(bodyStr, "Invalid") {
			meta["status"] = "api_responding"
			log.Printf("Sendy: API responding at %s (brands: %s)", baseURL, bodyStr[:min(len(bodyStr), 80)])
		} else {
			return fmt.Errorf("sendy: unexpected response: %s", bodyStr[:min(len(bodyStr), 100)])
		}
	}

	// Try to get campaigns (Sendy may not have this API in all versions)
	resp2, err2 := client.PostForm(baseURL+"/api/campaigns/get-campaigns.php", map[string][]string{
		"api_key": {apiKey},
	})
	if err2 == nil {
		defer resp2.Body.Close()
		body2, _ := io.ReadAll(resp2.Body)

		// Sendy campaigns may be {"campaign1":{...},"campaign2":{...}} or an array
		var campaignsMap map[string]struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			SentAt string `json:"sent_at"`
			Sent   interface{} `json:"sent"` // Could be timestamp int or string
			Opens  interface{} `json:"opens"`
			Clicks interface{} `json:"clicks"`
		}
		if json.Unmarshal(body2, &campaignsMap) == nil && len(campaignsMap) > 0 {
			meta["campaigns_total"] = len(campaignsMap)
			newCampaigns := 0

			for _, c := range campaignsMap {
				campID := c.ID
				if campID == "" {
					continue
				}
				campEvtID := fmt.Sprintf("evt-sendy-camp-%s", campID)
				var campExists int
				s.db.QueryRow("SELECT COUNT(*) FROM events WHERE id=?", campEvtID).Scan(&campExists)
				if campExists > 0 {
					continue
				}

				opens := 0
				clicks := 0
				switch v := c.Opens.(type) {
				case float64: opens = int(v)
				case string:  fmt.Sscanf(v, "%d", &opens)
				}
				switch v := c.Clicks.(type) {
				case float64: clicks = int(v)
				case string:  fmt.Sscanf(v, "%d", &clicks)
				}

				// Determine timestamp
				sentTime := now
				if c.SentAt != "" {
					sentTime = c.SentAt
				} else if ts, ok := c.Sent.(float64); ok && ts > 0 {
					sentTime = time.Unix(int64(ts), 0).UTC().Format(time.RFC3339)
				}

				campMeta, _ := json.Marshal(map[string]interface{}{
					"campaign_id": campID, "opens": opens, "clicks": clicks,
				})

				campTitle := fmt.Sprintf("📧 Campaign: %s", c.Title)
				scoreDelta := 5
				if opens > 100 { scoreDelta = 7 }

				s.mu.Lock()
				s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
					artifact_title, confidence, verifiers, verification_level,
					score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
					campEvtID, "CAMPAIGN_SENT", "distribution", "sendy-poller", sentTime,
					campTitle, 0.9, `["sendy_api"]`, "STRONG",
					scoreDelta, string(campMeta), now, "approved")
				s.mu.Unlock()
				newCampaigns++
			}

			if newCampaigns > 0 {
				log.Printf("Sendy: %d new campaigns discovered", newCampaigns)
			}
		}
	}

	// Daily health check event (only if we haven't scored campaigns today)
	metaJSON, _ := json.Marshal(meta)
	scoreDelta := 0 // Don't double-score if campaigns already scored

	s.mu.Lock()
	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, confidence, verifiers, verification_level,
		score_delta, metadata, created_at, status) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		evtID, "EMAIL_HEALTH", "distribution", "sendy-poller", now,
		"📧 Sendy: connected", 0.8, `["sendy_api"]`, "MEDIUM",
		scoreDelta, string(metaJSON), now, "approved")
	s.mu.Unlock()

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// isSourceTrusted checks if a source has been approved at least once (approve-once trust)
func (s *Server) isSourceTrusted(source string) bool {
	var count int
	s.db.QueryRow(`SELECT approved_count FROM trusted_sources WHERE source=?`, source).Scan(&count)
	return count > 0
}

// insertEventIfNew inserts an event only if external_id doesn't already exist.
// Used by pollers to avoid duplicate events on re-poll.
func (s *Server) insertEventIfNew(evt Event) {
	if evt.ExternalID != "" {
		var exists int
		s.db.QueryRow("SELECT COUNT(*) FROM events WHERE external_id=?", evt.ExternalID).Scan(&exists)
		if exists > 0 {
			return
		}
	}
	id := fmt.Sprintf("evt-%d", time.Now().UnixNano())
	ts := evt.Timestamp
	if ts == "" {
		ts = time.Now().UTC().Format(time.RFC3339)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	// Determine status based on trusted sources
	status := "pending"
	if s.isSourceTrusted(evt.Source) {
		status = "approved"
	}

	verLevel := evt.Verification
	if verLevel == "" {
		verLevel = "PROVIDER_API"
	}

	s.db.Exec(`INSERT INTO events (id, event_type, lane, source, timestamp,
		artifact_title, detail, score_delta, confidence, verification_level, external_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, evt.EventType, evt.Lane, evt.Source, ts,
		evt.ArtifactTitle, evt.Detail, evt.ScoreDelta, evt.Confidence, verLevel,
		evt.ExternalID, status, now)

	// Signal pairing engine
	if s.pairing != nil {
		s.pairing.Ingest(Signal{
			Type:    SignalEvent,
			Source:  evt.Source,
			Content: evt.ArtifactTitle,
			Features: map[string]float64{"score_delta": float64(evt.ScoreDelta)},
			Metadata: map[string]interface{}{
				"event_type": evt.EventType,
				"lane":       evt.Lane,
			},
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// FRESHBOOKS INTEGRATION — Invoices, Expenses, Payments, P&L
//
// Auth: Bearer token from FreshBooks Settings > Developer
// Config JSON: {"account_id": "XXXXX", "freshbooks_url": "https://api.freshbooks.com"}
// Polls: /accounting/account/{id}/invoices/invoices (paid/outstanding)
//        /accounting/account/{id}/expenses/expenses
//        /accounting/account/{id}/payments/payments
// ═══════════════════════════════════════════════════════════════════════════════

func (s *Server) pollFreshBooks(integrationID, token, configJSON, lastPoll string) error {
	var cfg struct {
		AccountID string `json:"account_id"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.AccountID == "" || token == "" {
		return fmt.Errorf("account_id and bearer token required")
	}

	baseURL := "https://api.freshbooks.com"
	client := &http.Client{Timeout: 30 * time.Second}
	eventsCreated := 0

	doGet := func(path string) (map[string]interface{}, error) {
		req, _ := http.NewRequest("GET", baseURL+path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Api-Version", "alpha")
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("freshbooks API %d", resp.StatusCode)
		}
		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		return result, nil
	}

	// Parse last poll time
	var since time.Time
	if lastPoll != "" {
		since, _ = time.Parse(time.RFC3339, lastPoll)
	}
	if since.IsZero() {
		since = time.Now().AddDate(0, -3, 0) // default: last 3 months
	}

	// ── Invoices ──────────────────────────────────────────────────────────
	invoiceData, err := doGet(fmt.Sprintf("/accounting/account/%s/invoices/invoices?include[]=lines&per_page=100&search[date_min]=%s",
		cfg.AccountID, since.Format("2006-01-02")))
	if err == nil {
		if resp, ok := invoiceData["response"].(map[string]interface{}); ok {
			if result, ok := resp["result"].(map[string]interface{}); ok {
				if invoices, ok := result["invoices"].([]interface{}); ok {
					for _, inv := range invoices {
						i, ok := inv.(map[string]interface{})
						if !ok {
							continue
						}
						status, _ := i["payment_status"].(string) // "paid", "unpaid", "partial"
						amount, _ := i["amount"].(map[string]interface{})
						amtStr, _ := amount["amount"].(string)
						invNum, _ := i["invoice_number"].(string)
						clientName := ""
						if org, ok := i["organization"].(string); ok && org != "" {
							clientName = org
						}
						updated, _ := i["updated"].(string)

						if status == "paid" {
							amtFloat := 0.0
							fmt.Sscanf(amtStr, "%f", &amtFloat)

							s.insertEventIfNew(Event{
								EventType:     "INVOICE_PAID",
								Lane:          "revenue",
								Source:        "freshbooks",
								ArtifactTitle: fmt.Sprintf("Invoice #%s — %s ($%s)", invNum, clientName, amtStr),
								Detail:        fmt.Sprintf("FreshBooks invoice paid: %s", clientName),
								ScoreDelta:    calcScoreDelta("revenue", "INVOICE_PAID", 1.0),
								Confidence:    1.0,
								Verification:  "PROVIDER_API",
								ExternalID:    fmt.Sprintf("fb-inv-%s", invNum),
								Timestamp:     updated,
							})
							eventsCreated++
						}
					}
				}
			}
		}
	}

	// ── Expenses ──────────────────────────────────────────────────────────
	expenseData, err := doGet(fmt.Sprintf("/accounting/account/%s/expenses/expenses?per_page=100&search[date_min]=%s",
		cfg.AccountID, since.Format("2006-01-02")))
	if err == nil {
		if resp, ok := expenseData["response"].(map[string]interface{}); ok {
			if result, ok := resp["result"].(map[string]interface{}); ok {
				if expenses, ok := result["expenses"].([]interface{}); ok {
					for _, exp := range expenses {
						e, ok := exp.(map[string]interface{})
						if !ok {
							continue
						}
						amt, _ := e["amount"].(map[string]interface{})
						amtStr, _ := amt["amount"].(string)
						vendor, _ := e["vendor"].(string)
						category, _ := e["category_name"].(string)
						date, _ := e["date"].(string)
						expID, _ := e["id"].(float64)

						amtFloat := 0.0
						fmt.Sscanf(amtStr, "%f", &amtFloat)

						s.insertEventIfNew(Event{
							EventType:     "EXPENSE_RECORDED",
							Lane:          "revenue",
							Source:        "freshbooks",
							ArtifactTitle: fmt.Sprintf("Expense: %s — %s ($%s)", vendor, category, amtStr),
							Detail:        fmt.Sprintf("FreshBooks expense: %s", vendor),
							ScoreDelta:    0, // expenses don't add score — tracked for P&L
							Confidence:    1.0,
							Verification:  "PROVIDER_API",
							ExternalID:    fmt.Sprintf("fb-exp-%d", int(expID)),
							Timestamp:     date + "T00:00:00Z",
						})
						eventsCreated++
					}
				}
			}
		}
	}

	// ── Payments ──────────────────────────────────────────────────────────
	paymentData, err := doGet(fmt.Sprintf("/accounting/account/%s/payments/payments?per_page=100&search[date_min]=%s",
		cfg.AccountID, since.Format("2006-01-02")))
	if err == nil {
		if resp, ok := paymentData["response"].(map[string]interface{}); ok {
			if result, ok := resp["result"].(map[string]interface{}); ok {
				if payments, ok := result["payments"].([]interface{}); ok {
					for _, pay := range payments {
						p, ok := pay.(map[string]interface{})
						if !ok {
							continue
						}
						amt, _ := p["amount"].(map[string]interface{})
						amtStr, _ := amt["amount"].(string)
						date, _ := p["date"].(string)
						payID, _ := p["id"].(float64)
						payType, _ := p["type"].(string) // "Credit", "Cash", etc.

						s.insertEventIfNew(Event{
							EventType:     "PAYMENT_RECEIVED",
							Lane:          "revenue",
							Source:        "freshbooks",
							ArtifactTitle: fmt.Sprintf("Payment received: $%s (%s)", amtStr, payType),
							Detail:        fmt.Sprintf("FreshBooks payment via %s", payType),
							ScoreDelta:    0, // don't double-count — invoice_paid already scored
							Confidence:    1.0,
							Verification:  "PROVIDER_API",
							ExternalID:    fmt.Sprintf("fb-pay-%d", int(payID)),
							Timestamp:     date + "T00:00:00Z",
						})
						eventsCreated++
					}
				}
			}
		}
	}

	if eventsCreated > 0 {
		log.Printf("[freshbooks] %d events from account %s", eventsCreated, cfg.AccountID)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// TRANSACTION RECONCILIATION ENGINE
//
// Problem: A single real-world transaction flows through multiple platforms:
//   FreshBooks invoice → Stripe charge → Bank deposit → WooCommerce order
//
// Without reconciliation, the same $500 appears 4 times in revenue.
//
// Solution: Fingerprint-based deduplication with fuzzy matching:
//   1. Extract canonical fields: amount, date (±3 days), description keywords
//   2. Generate fingerprint: hash(amount + date_bucket + normalized_description)
//   3. Group events by fingerprint into "transaction clusters"
//   4. Mark duplicates: keep highest-confidence event, mark others as "reconciled"
//   5. Test/fake detection: Stripe test mode keys, $0.50 charges, WooCommerce draft orders
//
// Rules:
//   - Amount must match within $0.50 (fees/rounding)
//   - Date must be within 3 calendar days
//   - Same amount + same date range + different sources = probable duplicate
//   - Stripe test mode (test_ prefix, amounts like $1.00) = always fake
//   - WooCommerce draft/pending orders without payment = not real revenue
//   - Manual override: operator can force-reconcile or force-keep
// ═══════════════════════════════════════════════════════════════════════════════

// ReconciliationResult represents a cluster of related transactions
type ReconciliationResult struct {
	ClusterID     string   `json:"cluster_id"`
	Amount        float64  `json:"amount"`
	DateRange     string   `json:"date_range"`
	Sources       []string `json:"sources"`       // e.g., ["stripe", "freshbooks", "woocommerce"]
	EventIDs      []string `json:"event_ids"`     // event IDs in cluster
	PrimaryID     string   `json:"primary_id"`    // highest-confidence event kept
	DuplicateIDs  []string `json:"duplicate_ids"`  // events marked as duplicates
	TestEvents    []string `json:"test_events"`    // events identified as test/fake
	Confidence    float64  `json:"confidence"`     // 0-1 how confident this is a real match
	Status        string   `json:"status"`         // "auto", "manual_confirmed", "manual_rejected"
}

func init() {
	// Ensure table exists — called from main init
}

func (s *Server) initReconciliation() {
	s.db.Exec(`CREATE TABLE IF NOT EXISTS reconciled_events (
		event_id TEXT PRIMARY KEY,
		cluster_id TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT 'duplicate',
		primary_event_id TEXT,
		reconciled_at TEXT NOT NULL
	)`)
	s.db.Exec(`CREATE TABLE IF NOT EXISTS test_transactions (
		event_id TEXT PRIMARY KEY,
		reason TEXT NOT NULL,
		detected_at TEXT NOT NULL
	)`)
}

// ReconcileRevenue runs the full reconciliation pipeline on revenue events
func (s *Server) ReconcileRevenue() []ReconciliationResult {
	// 1. Fetch all revenue events
	rows, err := s.db.Query(`SELECT id, event_type, source, artifact_title, score_delta, timestamp, external_id
		FROM events WHERE lane='revenue' AND status='approved'
		ORDER BY timestamp DESC LIMIT 1000`)
	if err != nil {
		log.Printf("[reconcile] query error: %v", err)
		return nil
	}
	defer rows.Close()

	type revEvent struct {
		ID        string
		EventType string
		Source    string
		Title     string
		Amount    float64
		Timestamp string
		ExtID     string
	}
	var events []revEvent
	for rows.Next() {
		var e revEvent
		rows.Scan(&e.ID, &e.EventType, &e.Source, &e.Title, &e.Amount, &e.Timestamp, &e.ExtID)
		// Extract dollar amount from title if score_delta is 0
		if e.Amount == 0 {
			e.Amount = extractAmountFromTitle(e.Title)
		}
		events = append(events, e)
	}

	// 2. Detect test/fake transactions first
	var testIDs []string
	for _, e := range events {
		if reason := detectTestTransaction(e.Source, e.Title, e.Amount, e.ExtID); reason != "" {
			testIDs = append(testIDs, e.ID)
			s.db.Exec(`INSERT OR REPLACE INTO test_transactions (event_id, reason, detected_at) VALUES (?, ?, ?)`,
				e.ID, reason, time.Now().UTC().Format(time.RFC3339))
		}
	}

	// 3. Build clusters by amount + date proximity
	type cluster struct {
		amount    float64
		dateStart time.Time
		dateEnd   time.Time
		events    []revEvent
	}
	var clusters []cluster

	isTest := make(map[string]bool)
	for _, id := range testIDs {
		isTest[id] = true
	}

	for _, e := range events {
		if isTest[e.ID] {
			continue // skip test transactions from clustering
		}
		if e.Amount < 0.01 {
			continue // skip zero-amount
		}

		eTime, _ := time.Parse(time.RFC3339, e.Timestamp)
		if eTime.IsZero() {
			eTime, _ = time.Parse("2006-01-02T15:04:05Z", e.Timestamp)
		}

		matched := false
		for i := range clusters {
			// Amount within $0.50 and date within 3 days
			if math.Abs(clusters[i].amount-e.Amount) <= 0.50 {
				if !eTime.IsZero() {
					dayDiff := math.Abs(eTime.Sub(clusters[i].dateStart).Hours() / 24)
					if dayDiff <= 3.0 {
						clusters[i].events = append(clusters[i].events, e)
						if eTime.Before(clusters[i].dateStart) {
							clusters[i].dateStart = eTime
						}
						if eTime.After(clusters[i].dateEnd) {
							clusters[i].dateEnd = eTime
						}
						matched = true
						break
					}
				}
			}
		}
		if !matched {
			clusters = append(clusters, cluster{
				amount:    e.Amount,
				dateStart: eTime,
				dateEnd:   eTime,
				events:    []revEvent{e},
			})
		}
	}

	// 4. Process clusters — multi-source clusters are probable duplicates
	var results []ReconciliationResult
	for idx, c := range clusters {
		if len(c.events) < 2 {
			continue // single event = no reconciliation needed
		}

		// Check if multiple sources
		sources := make(map[string]bool)
		for _, e := range c.events {
			sources[e.Source] = true
		}
		if len(sources) < 2 {
			continue // all from same source = not cross-platform duplicate
		}

		var sourceList []string
		for src := range sources {
			sourceList = append(sourceList, src)
		}

		// Pick primary: highest confidence source priority
		// stripe > freshbooks > woocommerce > memberpress > manual
		primaryIdx := 0
		primaryScore := sourcePriority(c.events[0].Source)
		for i, e := range c.events {
			p := sourcePriority(e.Source)
			if p > primaryScore {
				primaryScore = p
				primaryIdx = i
			}
		}

		var eventIDs, dupIDs []string
		for i, e := range c.events {
			eventIDs = append(eventIDs, e.ID)
			if i != primaryIdx {
				dupIDs = append(dupIDs, e.ID)
			}
		}

		// Confidence: more sources = higher confidence this is real
		conf := math.Min(1.0, 0.5+float64(len(sources))*0.2)

		clusterID := fmt.Sprintf("rc-%d-%d", idx, time.Now().Unix())
		results = append(results, ReconciliationResult{
			ClusterID:    clusterID,
			Amount:       c.amount,
			DateRange:    c.dateStart.Format("2006-01-02") + " → " + c.dateEnd.Format("2006-01-02"),
			Sources:      sourceList,
			EventIDs:     eventIDs,
			PrimaryID:    c.events[primaryIdx].ID,
			DuplicateIDs: dupIDs,
			TestEvents:   nil,
			Confidence:   conf,
			Status:       "auto",
		})

		// Mark duplicates in DB
		for _, dupID := range dupIDs {
			s.db.Exec(`INSERT OR REPLACE INTO reconciled_events (event_id, cluster_id, status, primary_event_id, reconciled_at)
				VALUES (?, ?, 'duplicate', ?, ?)`,
				dupID, clusterID, c.events[primaryIdx].ID, time.Now().UTC().Format(time.RFC3339))
			// Zero out the duplicate's score to prevent double-counting
			s.db.Exec(`UPDATE events SET score_delta = 0 WHERE id = ?`, dupID)
		}
	}

	if len(results) > 0 || len(testIDs) > 0 {
		log.Printf("[reconcile] processed %d clusters, %d duplicates zeroed, %d test transactions flagged",
			len(results),
			func() int { c := 0; for _, r := range results { c += len(r.DuplicateIDs) }; return c }(),
			len(testIDs))
	}

	return results
}

// detectTestTransaction identifies fake/test transactions
func detectTestTransaction(source, title string, amount float64, extID string) string {
	titleLower := strings.ToLower(title)
	extIDLower := strings.ToLower(extID)

	// Stripe test mode indicators
	if source == "stripe" {
		if strings.Contains(extIDLower, "test_") {
			return "stripe test mode (test_ prefix in ID)"
		}
		if strings.Contains(titleLower, "test") && amount <= 1.0 {
			return "stripe test charge (contains 'test' + small amount)"
		}
		// $0.50 Stripe test charges are extremely common
		if amount == 0.50 {
			return "stripe likely test charge ($0.50)"
		}
	}

	// WooCommerce test/draft orders
	if source == "woocommerce" {
		if strings.Contains(titleLower, "draft") || strings.Contains(titleLower, "pending") {
			return "woocommerce draft/pending order (not completed)"
		}
		if strings.Contains(titleLower, "test") {
			return "woocommerce test order"
		}
		if amount == 0 {
			return "woocommerce zero-amount order"
		}
	}

	// FreshBooks test
	if source == "freshbooks" {
		if strings.Contains(titleLower, "test") && amount <= 1.0 {
			return "freshbooks test invoice"
		}
	}

	// Generic test indicators
	if strings.Contains(titleLower, "test transaction") || strings.Contains(titleLower, "test payment") {
		return "title contains 'test transaction/payment'"
	}

	return "" // not a test transaction
}

// extractAmountFromTitle pulls dollar amounts from event titles like "Invoice #123 ($500.00)"
func extractAmountFromTitle(title string) float64 {
	amtPattern := regexp.MustCompile(`\$([0-9,]+\.?\d*)`)
	matches := amtPattern.FindStringSubmatch(title)
	if len(matches) >= 2 {
		cleaned := strings.ReplaceAll(matches[1], ",", "")
		var amt float64
		fmt.Sscanf(cleaned, "%f", &amt)
		return amt
	}
	return 0
}

// sourcePriority returns a score for source trustworthiness (higher = more authoritative)
func sourcePriority(source string) int {
	switch source {
	case "stripe":
		return 100 // payment processor = source of truth
	case "freshbooks":
		return 90 // accounting system
	case "plaid":
		return 85 // bank account
	case "woocommerce":
		return 70 // storefront
	case "memberpress":
		return 70 // membership
	case "manual":
		return 50 // manual entry
	default:
		return 60
	}
}

// ─── Reconciliation API endpoints ────────────────────────────────────────────

func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	cors(w)
	switch r.Method {
	case "GET":
		// Return current reconciliation state
		var recon []ReconciliationResult
		rows, err := s.db.Query(`SELECT DISTINCT cluster_id FROM reconciled_events ORDER BY reconciled_at DESC LIMIT 50`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var clusterID string
				rows.Scan(&clusterID)
				var r ReconciliationResult
				r.ClusterID = clusterID
				// Get events in cluster
				erows, _ := s.db.Query(`SELECT event_id, status, primary_event_id FROM reconciled_events WHERE cluster_id=?`, clusterID)
				if erows != nil {
					for erows.Next() {
						var eid, st, pid string
						erows.Scan(&eid, &st, &pid)
						r.EventIDs = append(r.EventIDs, eid)
						if st == "duplicate" {
							r.DuplicateIDs = append(r.DuplicateIDs, eid)
						}
						r.PrimaryID = pid
					}
					erows.Close()
				}
				r.Status = "auto"
				recon = append(recon, r)
			}
		}

		// Get test transactions
		var tests []map[string]string
		trows, err := s.db.Query(`SELECT event_id, reason, detected_at FROM test_transactions ORDER BY detected_at DESC LIMIT 50`)
		if err == nil {
			defer trows.Close()
			for trows.Next() {
				var eid, reason, det string
				trows.Scan(&eid, &reason, &det)
				tests = append(tests, map[string]string{"event_id": eid, "reason": reason, "detected_at": det})
			}
		}

		writeJSON(w, map[string]interface{}{
			"reconciled_clusters": recon,
			"test_transactions":   tests,
		})

	case "POST":
		// Run reconciliation now
		results := s.ReconcileRevenue()
		writeJSON(w, map[string]interface{}{
			"clusters_found": len(results),
			"results":        results,
		})

	default:
		http.Error(w, "GET or POST", 405)
	}
}

func (s *Server) handleTestTransactions(w http.ResponseWriter, r *http.Request) {
	cors(w)
	rows, err := s.db.Query(`SELECT t.event_id, t.reason, t.detected_at, e.artifact_title, e.source, e.score_delta
		FROM test_transactions t
		LEFT JOIN events e ON e.id = t.event_id
		ORDER BY t.detected_at DESC LIMIT 100`)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err), 500)
		return
	}
	defer rows.Close()

	var tests []map[string]interface{}
	for rows.Next() {
		var eid, reason, det, title, source string
		var delta float64
		rows.Scan(&eid, &reason, &det, &title, &source, &delta)
		tests = append(tests, map[string]interface{}{
			"event_id":    eid,
			"reason":      reason,
			"detected_at": det,
			"title":       title,
			"source":      source,
			"score_delta": delta,
		})
	}
	writeJSON(w, map[string]interface{}{"test_transactions": tests})
}
