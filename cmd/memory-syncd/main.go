// wirebot-memory-syncd — bidirectional memory sync daemon
//
// Watches workspace files (inotify) and polls Mem0/Letta for changes.
// Syncs bidirectionally with debouncing and conflict prevention.
//
// Flows:
//   Workspace → External:
//     MEMORY.md edit     → extract new facts → POST Mem0
//     BUSINESS_STATE.md  → parse → PATCH Letta blocks
//   External → Workspace:
//     Mem0 new facts     → dedup → append MEMORY.md
//     Letta block change → snapshot BUSINESS_STATE.md
//
// Health: GET :8201/health
// Trigger: POST :8201/sync

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ============================================================================
// Config
// ============================================================================

type Config struct {
	Workspace    string
	Mem0URL      string
	LettaURL     string
	LettaAgentID string
	Mem0Namespace string
	ListenAddr   string
	PollInterval time.Duration
	Debounce     time.Duration
	LockFile     string
}

// ============================================================================
// State
// ============================================================================

type SyncState struct {
	mu              sync.Mutex
	lastMemoryHash  string // hash of MEMORY.md at last sync
	lastBizHash     string // hash of BUSINESS_STATE.md at last sync
	lastMem0Sync    time.Time
	lastLettaSync   time.Time
	syncsTotal      int64
	syncErrors      int64
	factsWritten    int64
	blocksWritten   int64
	startedAt       time.Time
	writing         bool // true when daemon is writing files (suppress inotify)

	// Hot cache
	cachedFacts     []Mem0Fact
	cachedBlocks    []LettaBlock
	cacheHits       int64
	cacheUpdatedAt  time.Time
}

// ============================================================================
// Mem0 client
// ============================================================================

type Mem0Fact struct {
	ID     string `json:"id"`
	Memory string `json:"memory"`
}

type Mem0ListResponse struct {
	Results []Mem0Fact `json:"results"`
}

func mem0List(baseURL, namespace string) ([]Mem0Fact, error) {
	body, _ := json.Marshal(map[string]string{"namespace": namespace})
	req, err := http.NewRequest("POST", baseURL+"/v1/list", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("mem0 list: status %d", resp.StatusCode)
	}

	var result Mem0ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Results, nil
}

func mem0Store(baseURL, namespace, text string) error {
	body, _ := json.Marshal(map[string]string{
		"text":      text,
		"namespace": namespace,
		"category":  "workspace",
	})
	req, err := http.NewRequest("POST", baseURL+"/v1/store", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("mem0 store: status %d", resp.StatusCode)
	}
	return nil
}

// ============================================================================
// Letta client
// ============================================================================

type LettaBlock struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

func lettaGetBlocks(baseURL, agentID string) ([]LettaBlock, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/core-memory/blocks", baseURL, agentID)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("letta blocks: status %d", resp.StatusCode)
	}

	var blocks []LettaBlock
	if err := json.NewDecoder(resp.Body).Decode(&blocks); err != nil {
		return nil, err
	}
	return blocks, nil
}

// lettaUpdateBlock sends a message to the Letta agent asking it to self-edit a block.
// This routes through the agent's reasoning rather than direct PATCH, preserving
// Letta's core value: the agent decides what to persist using memory_replace/memory_insert.
func lettaUpdateBlock(baseURL, agentID, label, value string) error {
	// Truncate value to avoid huge payloads
	truncated := value
	if len(truncated) > 2000 {
		truncated = truncated[:2000] + "..."
	}

	msg := fmt.Sprintf("Workspace sync update for your [%s] block. Here is the current workspace state:\n\n%s\n\nPlease update your %s block to reflect this information. Use memory_replace to make targeted edits rather than overwriting everything.",
		label, truncated, label)

	payload, _ := json.Marshal(map[string]interface{}{
		"messages": []map[string]string{{"role": "user", "content": msg}},
	})

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v1/agents/%s/messages/", baseURL, agentID), bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer letta")

	client := &http.Client{Timeout: 60 * time.Second} // Agent reasoning takes time
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("letta agent message for block %s: status %d: %s", label, resp.StatusCode, string(body[:min(200, len(body))]))
	}
	return nil
}

// ============================================================================
// File helpers
// ============================================================================

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func quickHash(s string) string {
	// Simple fast hash for change detection — not crypto
	var h uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		h ^= uint64(s[i])
		h *= 1099511628211
	}
	return fmt.Sprintf("%016x", h)
}

// ============================================================================
// Sync: External → Workspace
// ============================================================================

func syncMem0ToWorkspace(cfg *Config, state *SyncState) {
	facts, err := mem0List(cfg.Mem0URL, cfg.Mem0Namespace)
	if err != nil {
		log.Printf("[mem0→ws] error listing facts: %v", err)
		state.mu.Lock()
		state.syncErrors++
		state.mu.Unlock()
		return
	}

	memoryPath := filepath.Join(cfg.Workspace, "MEMORY.md")
	content, err := readFile(memoryPath)
	if err != nil {
		log.Printf("[mem0→ws] error reading MEMORY.md: %v", err)
		return
	}

	// Find new facts not already in the file
	var newFacts []string
	for _, f := range facts {
		if f.Memory != "" && !strings.Contains(content, f.Memory) {
			newFacts = append(newFacts, f.Memory)
		}
	}

	// Always update cache
	state.mu.Lock()
	state.cachedFacts = facts
	state.cacheUpdatedAt = time.Now()
	state.mu.Unlock()

	if len(newFacts) == 0 {
		state.mu.Lock()
		state.lastMem0Sync = time.Now()
		state.mu.Unlock()
		return
	}

	// Ensure sync section exists
	if !strings.Contains(content, "## Synced Facts") {
		content += "\n## Synced Facts\n\n"
	}

	// Append
	var buf strings.Builder
	buf.WriteString(content)
	for _, fact := range newFacts {
		buf.WriteString("- ")
		buf.WriteString(fact)
		buf.WriteString("\n")
	}

	state.mu.Lock()
	state.writing = true
	state.mu.Unlock()

	if err := os.WriteFile(memoryPath, []byte(buf.String()), 0644); err != nil {
		log.Printf("[mem0→ws] error writing MEMORY.md: %v", err)
		state.mu.Lock()
		state.writing = false
		state.syncErrors++
		state.mu.Unlock()
		return
	}

	// Brief pause for inotify suppression
	time.AfterFunc(2*time.Second, func() {
		state.mu.Lock()
		state.writing = false
		state.lastMemoryHash = quickHash(buf.String())
		state.mu.Unlock()
	})

	state.mu.Lock()
	state.lastMem0Sync = time.Now()
	state.factsWritten += int64(len(newFacts))
	state.syncsTotal++
	state.mu.Unlock()

	log.Printf("[mem0→ws] appended %d new facts to MEMORY.md", len(newFacts))
}

func syncLettaToWorkspace(cfg *Config, state *SyncState) {
	blocks, err := lettaGetBlocks(cfg.LettaURL, cfg.LettaAgentID)
	if err != nil {
		log.Printf("[letta→ws] error getting blocks: %v", err)
		state.mu.Lock()
		state.syncErrors++
		state.mu.Unlock()
		return
	}

	if len(blocks) == 0 {
		return
	}

	// Build snapshot
	var buf strings.Builder
	buf.WriteString("# Business State\n\n")
	buf.WriteString(fmt.Sprintf("> Last synced: %s\n\n", time.Now().Format("2006-01-02 15:04 MST")))
	for _, b := range blocks {
		buf.WriteString("## ")
		buf.WriteString(b.Label)
		buf.WriteString("\n\n")
		buf.WriteString(b.Value)
		buf.WriteString("\n\n")
	}

	snapshot := buf.String()
	newHash := quickHash(snapshot)

	// Always update cache
	state.mu.Lock()
	state.cachedBlocks = blocks
	state.cacheUpdatedAt = time.Now()
	state.mu.Unlock()

	state.mu.Lock()
	if newHash == state.lastBizHash {
		state.lastLettaSync = time.Now()
		state.mu.Unlock()
		return
	}
	state.writing = true
	state.mu.Unlock()

	bizPath := filepath.Join(cfg.Workspace, "BUSINESS_STATE.md")
	if err := os.WriteFile(bizPath, []byte(snapshot), 0644); err != nil {
		log.Printf("[letta→ws] error writing BUSINESS_STATE.md: %v", err)
		state.mu.Lock()
		state.writing = false
		state.syncErrors++
		state.mu.Unlock()
		return
	}

	time.AfterFunc(2*time.Second, func() {
		state.mu.Lock()
		state.writing = false
		state.lastBizHash = newHash
		state.mu.Unlock()
	})

	state.mu.Lock()
	state.lastLettaSync = time.Now()
	state.blocksWritten++
	state.syncsTotal++
	state.mu.Unlock()

	log.Printf("[letta→ws] updated BUSINESS_STATE.md (%d blocks)", len(blocks))
}

// ============================================================================
// Sync: Workspace → External
// ============================================================================

func syncMemoryToMem0(cfg *Config, state *SyncState) {
	memoryPath := filepath.Join(cfg.Workspace, "MEMORY.md")
	content, err := readFile(memoryPath)
	if err != nil {
		return
	}

	newHash := quickHash(content)
	state.mu.Lock()
	if newHash == state.lastMemoryHash {
		state.mu.Unlock()
		return
	}
	state.lastMemoryHash = newHash
	state.mu.Unlock()

	// Extract bullet points that look like facts
	var facts []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "- ") && len(line) > 10 && len(line) < 500 {
			fact := strings.TrimPrefix(line, "- ")
			facts = append(facts, fact)
		}
	}

	// Get existing Mem0 facts to dedup
	existing, err := mem0List(cfg.Mem0URL, cfg.Mem0Namespace)
	if err != nil {
		log.Printf("[ws→mem0] error listing existing facts: %v", err)
		return
	}

	existingSet := make(map[string]bool)
	for _, f := range existing {
		existingSet[f.Memory] = true
	}

	stored := 0
	for _, fact := range facts {
		if existingSet[fact] {
			continue
		}
		if err := mem0Store(cfg.Mem0URL, cfg.Mem0Namespace, fact); err != nil {
			log.Printf("[ws→mem0] error storing fact: %v", err)
			state.mu.Lock()
			state.syncErrors++
			state.mu.Unlock()
			continue
		}
		stored++
		// Rate limit: don't slam Mem0 with LLM calls
		time.Sleep(500 * time.Millisecond)
	}

	if stored > 0 {
		state.mu.Lock()
		state.factsWritten += int64(stored)
		state.syncsTotal++
		state.mu.Unlock()
		log.Printf("[ws→mem0] pushed %d new facts to Mem0", stored)
	}
}

func syncBizStateToLetta(cfg *Config, state *SyncState) {
	bizPath := filepath.Join(cfg.Workspace, "BUSINESS_STATE.md")
	content, err := readFile(bizPath)
	if err != nil {
		return
	}

	newHash := quickHash(content)
	state.mu.Lock()
	if newHash == state.lastBizHash {
		state.mu.Unlock()
		return
	}
	state.lastBizHash = newHash
	state.mu.Unlock()

	// Parse markdown blocks: ## label\n\nvalue\n\n
	blocks := parseBusinessState(content)
	if len(blocks) == 0 {
		return
	}

	updated := 0
	for label, value := range blocks {
		if err := lettaUpdateBlock(cfg.LettaURL, cfg.LettaAgentID, label, value); err != nil {
			log.Printf("[ws→letta] error updating block %s: %v", label, err)
			state.mu.Lock()
			state.syncErrors++
			state.mu.Unlock()
			continue
		}
		updated++
	}

	if updated > 0 {
		state.mu.Lock()
		state.blocksWritten += int64(updated)
		state.syncsTotal++
		state.mu.Unlock()
		log.Printf("[ws→letta] updated %d Letta blocks", updated)
	}
}

func parseBusinessState(content string) map[string]string {
	blocks := make(map[string]string)
	lines := strings.Split(content, "\n")
	var currentLabel string
	var currentValue strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Flush previous
			if currentLabel != "" {
				blocks[currentLabel] = strings.TrimSpace(currentValue.String())
			}
			currentLabel = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentValue.Reset()
		} else if currentLabel != "" && !strings.HasPrefix(line, "# ") && !strings.HasPrefix(line, "> ") {
			currentValue.WriteString(line)
			currentValue.WriteString("\n")
		}
	}
	// Flush last
	if currentLabel != "" {
		blocks[currentLabel] = strings.TrimSpace(currentValue.String())
	}
	return blocks
}

// ============================================================================
// File watcher
// ============================================================================

func watchWorkspace(ctx context.Context, cfg *Config, state *SyncState) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("[watch] failed to create watcher: %v", err)
	}
	defer watcher.Close()

	if err := watcher.Add(cfg.Workspace); err != nil {
		log.Fatalf("[watch] failed to watch %s: %v", cfg.Workspace, err)
	}
	log.Printf("[watch] watching %s", cfg.Workspace)

	// Debounce timers
	var memoryTimer *time.Timer
	var bizTimer *time.Timer

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Skip if we're the ones writing
			state.mu.Lock()
			writing := state.writing
			state.mu.Unlock()
			if writing {
				continue
			}

			base := filepath.Base(event.Name)
			isWrite := event.Op&(fsnotify.Write|fsnotify.Create) != 0
			if !isWrite {
				continue
			}

			switch base {
			case "MEMORY.md":
				if memoryTimer != nil {
					memoryTimer.Stop()
				}
				memoryTimer = time.AfterFunc(cfg.Debounce, func() {
					log.Printf("[watch] MEMORY.md changed, syncing to Mem0")
					syncMemoryToMem0(cfg, state)
				})

			case "BUSINESS_STATE.md":
				if bizTimer != nil {
					bizTimer.Stop()
				}
				bizTimer = time.AfterFunc(cfg.Debounce, func() {
					log.Printf("[watch] BUSINESS_STATE.md changed, syncing to Letta")
					syncBizStateToLetta(cfg, state)
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[watch] error: %v", err)
		}
	}
}

// ============================================================================
// Poller
// ============================================================================

func pollLoop(ctx context.Context, cfg *Config, state *SyncState) {
	ticker := time.NewTicker(cfg.PollInterval)
	defer ticker.Stop()

	// Initial sync
	syncMem0ToWorkspace(cfg, state)
	syncLettaToWorkspace(cfg, state)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncMem0ToWorkspace(cfg, state)
			syncLettaToWorkspace(cfg, state)
		}
	}
}

// ============================================================================
// HTTP health/trigger server
// ============================================================================

func startHTTP(cfg *Config, state *SyncState) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		data := map[string]interface{}{
			"status":         "ok",
			"uptime_seconds": int(time.Since(state.startedAt).Seconds()),
			"last_mem0_sync": state.lastMem0Sync.Format(time.RFC3339),
			"last_letta_sync": state.lastLettaSync.Format(time.RFC3339),
			"syncs_total":    state.syncsTotal,
			"sync_errors":    state.syncErrors,
			"facts_written":  state.factsWritten,
			"blocks_written": state.blocksWritten,
			"writing":        state.writing,
			"cache_facts":    len(state.cachedFacts),
			"cache_blocks":   len(state.cachedBlocks),
			"cache_hits":     state.cacheHits,
			"cache_age_ms":   int(time.Since(state.cacheUpdatedAt).Milliseconds()),
		}
		state.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	})

	// Hot cache: return facts matching substring (no embedding needed)
	mux.HandleFunc("/cache/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, `{"error":"missing ?q="}`, 400)
			return
		}
		queryLower := strings.ToLower(query)

		state.mu.Lock()
		facts := state.cachedFacts
		blocks := state.cachedBlocks
		state.cacheHits++
		state.mu.Unlock()

		type result struct {
			Source string `json:"source"`
			Text   string `json:"text"`
		}
		var results []result

		// Search facts
		for _, f := range facts {
			if strings.Contains(strings.ToLower(f.Memory), queryLower) {
				results = append(results, result{Source: "mem0", Text: f.Memory})
			}
		}

		// Search blocks
		for _, b := range blocks {
			if strings.Contains(strings.ToLower(b.Value), queryLower) ||
				strings.Contains(strings.ToLower(b.Label), queryLower) {
				results = append(results, result{Source: "letta:" + b.Label, Text: b.Value})
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"query":   query,
			"results": results,
			"cached":  true,
			"age_ms":  int(time.Since(state.cacheUpdatedAt).Milliseconds()),
		})
	})

	// Hot cache: return all cached state
	mux.HandleFunc("/cache/state", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		facts := state.cachedFacts
		blocks := state.cachedBlocks
		state.cacheHits++
		age := time.Since(state.cacheUpdatedAt).Milliseconds()
		state.mu.Unlock()

		factTexts := make([]string, len(facts))
		for i, f := range facts {
			factTexts[i] = f.Memory
		}

		blockMap := make(map[string]string)
		for _, b := range blocks {
			blockMap[b.Label] = b.Value
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"facts":  factTexts,
			"blocks": blockMap,
			"age_ms": int(age),
		})
	})

	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "POST only", 405)
			return
		}

		go func() {
			syncMem0ToWorkspace(cfg, state)
			syncLettaToWorkspace(cfg, state)
		}()

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"triggered":true}`)
	})

	srv := &http.Server{Addr: cfg.ListenAddr, Handler: mux}
	go func() {
		log.Printf("[http] listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("[http] error: %v", err)
		}
	}()
	return srv
}

// ============================================================================
// Main
// ============================================================================

func main() {
	cfg := Config{}
	flag.StringVar(&cfg.Workspace, "workspace", "/home/wirebot/clawd", "Workspace directory")
	flag.StringVar(&cfg.Mem0URL, "mem0-url", "http://127.0.0.1:8200", "Mem0 server URL")
	flag.StringVar(&cfg.LettaURL, "letta-url", "http://127.0.0.1:8283", "Letta server URL")
	flag.StringVar(&cfg.LettaAgentID, "letta-agent", "agent-82610d14-ec65-4d10-9ec2-8c479848cea9", "Letta agent ID")
	flag.StringVar(&cfg.Mem0Namespace, "mem0-namespace", "wirebot_verious", "Mem0 namespace")
	flag.StringVar(&cfg.ListenAddr, "listen", "127.0.0.1:8201", "Health/trigger HTTP address")
	flag.DurationVar(&cfg.PollInterval, "poll", 60*time.Second, "Poll interval for Mem0/Letta")
	flag.DurationVar(&cfg.Debounce, "debounce", 5*time.Second, "File change debounce")
	flag.StringVar(&cfg.LockFile, "lock", "/run/wirebot/sync.lock", "Lock file path")
	flag.Parse()

	log.SetFlags(log.Ldate | log.Ltime | log.Lmsgprefix)
	log.SetPrefix("[syncd] ")

	log.Printf("starting wirebot-memory-syncd")
	log.Printf("  workspace:  %s", cfg.Workspace)
	log.Printf("  mem0:       %s (ns: %s)", cfg.Mem0URL, cfg.Mem0Namespace)
	log.Printf("  letta:      %s (agent: %s)", cfg.LettaURL, cfg.LettaAgentID)
	log.Printf("  listen:     %s", cfg.ListenAddr)
	log.Printf("  poll:       %s", cfg.PollInterval)
	log.Printf("  debounce:   %s", cfg.Debounce)

	state := &SyncState{startedAt: time.Now()}

	// Seed hashes
	if content, err := readFile(filepath.Join(cfg.Workspace, "MEMORY.md")); err == nil {
		state.lastMemoryHash = quickHash(content)
	}
	if content, err := readFile(filepath.Join(cfg.Workspace, "BUSINESS_STATE.md")); err == nil {
		state.lastBizHash = quickHash(content)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start HTTP health server
	srv := startHTTP(&cfg, state)

	// Start file watcher
	go watchWorkspace(ctx, &cfg, state)

	// Start poll loop
	go pollLoop(ctx, &cfg, state)

	// Wait for signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("received %v, shutting down", sig)

	cancel()
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	srv.Shutdown(shutCtx)

	log.Printf("stopped (syncs: %d, errors: %d, facts: %d, blocks: %d)",
		state.syncsTotal, state.syncErrors, state.factsWritten, state.blocksWritten)
}
