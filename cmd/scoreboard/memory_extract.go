package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MemoryExtraction represents a fact extracted from a document with context
type MemoryExtraction struct {
	MemoryText    string  `json:"memory_text"`
	SourceType    string  `json:"source_type"`    // obsidian, conversation, pairing, blog
	SourceFile    string  `json:"source_file"`    // filename or conversation ID
	SourceSection string  `json:"source_section"` // header or section name
	SourceContext string  `json:"source_context"` // 3-4 sentences of actual quoted text
	Confidence    float64 `json:"confidence"`
}

// classifyDocument determines the extraction strategy based on file path and frontmatter.
// Returns: docType ("journal", "client_note", "ai_chat", "book", "note", "freewriting")
// and any metadata extracted from YAML frontmatter.
func classifyDocument(docPath, content string) (docType string, meta map[string]string) {
	meta = make(map[string]string)
	lower := strings.ToLower(docPath)

	// Parse YAML frontmatter (between --- delimiters)
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			fm := parts[1]
			for _, line := range strings.Split(fm, "\n") {
				line = strings.TrimSpace(line)
				if idx := strings.Index(line, ":"); idx > 0 {
					key := strings.TrimSpace(line[:idx])
					val := strings.TrimSpace(strings.Trim(line[idx+1:], "\"' "))
					if val != "" {
						meta[key] = val
					}
				}
			}
		}
	}

	// Classify by path
	switch {
	case strings.Contains(lower, "daily journal") || strings.Contains(lower, "daily thoughts"):
		docType = "journal"
	case strings.Contains(lower, "client notes") || strings.Contains(lower, "philoveracity notes"):
		docType = "client_note"
	case strings.Contains(lower, "ai chats") || strings.Contains(lower, "chatgpt") || strings.Contains(lower, "grok"):
		docType = "ai_chat"
	case strings.Contains(lower, "/book") || strings.HasPrefix(lower, "book"):
		docType = "book"
	case strings.Contains(lower, "freewriting"):
		docType = "freewriting"
	default:
		docType = "note"
	}

	return docType, meta
}

// buildExtractionPrompt generates a type-specific LLM prompt for document memory extraction.
// Different document types get different prompts tuned for their content structure.
func buildExtractionPrompt(docPath, content, docType string, meta map[string]string) string {
	// Inject frontmatter context if available
	var metaContext string
	if author := meta["author"]; author != "" {
		metaContext += fmt.Sprintf("Author: %s\n", author)
	}
	if created := meta["created"]; created != "" {
		metaContext += fmt.Sprintf("Created: %s\n", created)
	}
	if loc := meta["location"]; loc != "" {
		metaContext += fmt.Sprintf("GPS: %s\n", loc)
	}
	if title := meta["title"]; title != "" {
		metaContext += fmt.Sprintf("Title: %s\n", title)
	}
	if platform := meta["platform"]; platform != "" {
		metaContext += fmt.Sprintf("Platform: %s\n", platform)
	}

	// Type-specific extraction instructions
	var typeRules string
	switch docType {
	case "journal":
		typeRules = `This is a DAILY JOURNAL entry. Extract:
- Habits and routines (wake time, meals, exercise, work patterns)
- Emotional state, stress levels, energy
- Projects being worked on and progress
- People mentioned and relationships
- Goals set or referenced
- Health notes (vitamins, sleep, food)
- Location and environment clues
Weight recent patterns over one-off mentions. Habits with timestamps are high-value.`

	case "client_note":
		typeRules = `This is a CLIENT/BUSINESS NOTE. Extract:
- Client name and project relationship
- Pricing, hourly rates, project costs mentioned
- Technical skills demonstrated (WordPress, WooCommerce, etc.)
- Business practices (rush fees, estimation methods)
- Client industries served
- Tools and platforms used
DO extract business relationships even if old — they reveal skills and experience.`

	case "ai_chat":
		typeRules = `This is an AI CHAT EXPORT. Extract ONLY facts from the USER's messages (👤 You):
- Questions reveal what the user is working on or curious about
- Stated goals, plans, strategies
- Business context (startup stage, revenue model, products)
- Technical skills and tools mentioned
IGNORE the AI assistant's responses entirely — only the user's own words matter.
The chat title often reveals the user's current focus area.`

	case "book":
		typeRules = `This is the author's BOOK MANUSCRIPT or notes. Extract:
- The author is WRITING this book — that itself is a key fact
- Topics and expertise areas the book covers
- Personal stories or anecdotes embedded in the writing
- Author's beliefs, philosophy, or worldview expressed
- Target audience mentioned
DO NOT extract facts from quoted external sources within the book.`

	case "freewriting":
		typeRules = `This is FREEWRITING / creative expression. Extract:
- Emotional states and what triggered them
- Life events and circumstances described
- Relationships and people mentioned
- Goals, dreams, aspirations expressed
- Struggles or challenges mentioned
Be careful with metaphor vs. literal statements — confidence should be lower for figurative language.`

	default: // "note"
		typeRules = `This is a general NOTE. Extract any personal facts about the author.
Look for: ideas the author had, skills demonstrated, tools used, projects planned,
people referenced, locations mentioned, and business concepts explored.`
	}

	return fmt.Sprintf(`You are extracting PERSONAL FACTS about the document author from their writing.

DOCUMENT TYPE: %s
FILE: %s
%s---
%s
---

TYPE-SPECIFIC INSTRUCTIONS:
%s

GENERAL RULES:
1. Extract facts about the AUTHOR — inference OK if supported by quoted text
2. Each fact MUST have a direct quote (2-4 sentences) from the document as evidence
3. Confidence scoring:
   - 0.9+ explicit first-person statement ("I live in X", "my name is Y")
   - 0.7-0.9 supported inference ("moved to Corona last year" → lives in Corona)
   - 0.5-0.7 weak inference (mentioned but unclear if about the author)
   - Below 0.5: do not extract
4. Focus on: location, work, preferences, relationships, goals, habits, beliefs, business, skills, tools
5. Do NOT extract general knowledge or opinions about external topics
6. NO GUESSING — if you can't quote evidence, don't extract it
7. FLAG ENTITIES that need disambiguation (e.g. "Providence" = hospital? city? organization?)
8. TEMPORAL CONTEXT: note when facts are dated — "in 2016" differs from "currently"
9. RELATIONSHIPS: extract connections between people and projects (e.g. "Client X hired author for Y")
10. SKILLS: programming languages, tools, platforms the author uses or has used

OUTPUT FORMAT (JSON array):
[
  {
    "memory": "The author lives in Corona, California",
    "quote": "I've been living in Corona for the past 5 years...",
    "section": "About Me",
    "confidence": 0.95,
    "category": "identity",
    "entities": ["Corona, California"],
    "reasoning": "Explicit first-person statement about current residence"
  }
]

CATEGORIES: identity, preference, fact, relationship, business, temporal, habit, skill, health, creative
ENTITIES: List any person names, locations, organizations, or products mentioned.
REASONING: One sentence explaining why you assigned that confidence level.

Extract as many facts as the evidence supports — do not artificially limit to 3-5. Rich documents may yield 10+ facts.
If no personal facts can be extracted with real quotes, return: []`,
		docType, docPath,
		func() string {
			if metaContext != "" {
				return "METADATA:\n" + metaContext
			}
			return ""
		}(),
		content, typeRules)
}

// ExtractMemoriesFromDocument uses LLM to extract personal facts with real quoted context.
// Uses document classification for type-specific extraction prompts.
// Long documents are chunked to avoid truncation loss.
func (s *Server) ExtractMemoriesFromDocument(docPath, content string) ([]MemoryExtraction, error) {
	// Don't process tiny files
	if len(content) < 100 {
		return nil, nil
	}

	// Classify document for type-specific extraction
	docType, meta := classifyDocument(docPath, content)

	// Strip YAML frontmatter from content before sending to LLM
	extractContent := content
	if strings.HasPrefix(strings.TrimSpace(content), "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			extractContent = parts[2]
		}
	}

	// Chunk strategy: for long docs, extract from multiple chunks instead of truncating
	var allMemories []MemoryExtraction
	chunkSize := 6000
	overlap := 500

	if len(extractContent) <= chunkSize+overlap {
		// Single chunk — fast path
		prompt := buildExtractionPrompt(docPath, extractContent, docType, meta)
		memories, err := s.extractFromPrompt(docPath, content, prompt, docType)
		if err != nil {
			return nil, err
		}
		return memories, nil
	}

	// Multi-chunk: slide through document with overlap
	chunkNum := 0
	for offset := 0; offset < len(extractContent); offset += chunkSize {
		end := offset + chunkSize + overlap
		if end > len(extractContent) {
			end = len(extractContent)
		}
		chunk := extractContent[offset:end]
		chunkNum++

		chunkPath := fmt.Sprintf("%s [chunk %d]", docPath, chunkNum)
		prompt := buildExtractionPrompt(chunkPath, chunk, docType, meta)
		memories, err := s.extractFromPrompt(docPath, content, prompt, docType)
		if err != nil {
			log.Printf("[memory-extract] Chunk %d error for %s: %v", chunkNum, docPath, err)
			continue
		}
		allMemories = append(allMemories, memories...)

		// Max 3 chunks per document to bound LLM costs
		if chunkNum >= 3 {
			break
		}

		// Rate limit between chunks
		time.Sleep(1 * time.Second)
	}

	// Dedup memories within the same document (multi-chunk can produce overlaps)
	seen := map[string]bool{}
	var deduped []MemoryExtraction
	for _, m := range allMemories {
		key := strings.ToLower(strings.Join(strings.Fields(m.MemoryText), " "))
		if !seen[key] {
			seen[key] = true
			deduped = append(deduped, m)
		}
	}

	return deduped, nil
}

// extractFromPrompt handles the LLM call and response parsing for a single extraction prompt.
func (s *Server) extractFromPrompt(docPath, fullContent, prompt, docType string) ([]MemoryExtraction, error) {

	// Call gateway for extraction
	result, err := s.callLLMForExtraction(prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	var extractions []struct {
		Memory     string   `json:"memory"`
		Quote      string   `json:"quote"`
		Section    string   `json:"section"`
		Confidence float64  `json:"confidence"`
		Category   string   `json:"category"`
		Entities   []string `json:"entities"`
		Reasoning  string   `json:"reasoning"`
	}

	// Find JSON array in response
	start := strings.Index(result, "[")
	end := strings.LastIndex(result, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(result[start:end+1]), &extractions); err != nil {
		log.Printf("[memory-extract] JSON parse error: %v", err)
		return nil, nil
	}

	// Convert to MemoryExtraction
	var memories []MemoryExtraction
	for _, e := range extractions {
		if e.Confidence < 0.5 || e.Memory == "" || e.Quote == "" {
			continue
		}
		// Verify quote actually exists in content (fuzzy match)
		if !containsFuzzy(fullContent, e.Quote) {
			log.Printf("[memory-extract] Quote not found in doc, skipping: %s", e.Quote[:min(50, len(e.Quote))])
			continue
		}
		// Build context with reasoning and entities
		context := e.Quote
		if e.Reasoning != "" {
			context += "\n\n[Reasoning: " + e.Reasoning + "]"
		}
		if len(e.Entities) > 0 {
			context += "\n[Entities: " + strings.Join(e.Entities, ", ") + "]"
		}
		// Use classified doc type as source_type for richer filtering
		sourceType := "obsidian"
		if docType == "ai_chat" {
			sourceType = "ai_chat"
		} else if docType == "journal" {
			sourceType = "journal"
		} else if docType == "client_note" {
			sourceType = "client_note"
		}

		memories = append(memories, MemoryExtraction{
			MemoryText:    e.Memory,
			SourceType:    sourceType,
			SourceFile:    docPath,
			SourceSection: e.Section,
			SourceContext: context,
			Confidence:    e.Confidence,
		})
	}

	return memories, nil
}

// containsFuzzy checks if quote exists in content using token overlap scoring.
// Returns true if enough tokens from the quote appear in the content.
func containsFuzzy(content, quote string) bool {
	// Normalize both
	normContent := strings.ToLower(strings.Join(strings.Fields(content), " "))
	normQuote := strings.ToLower(strings.Join(strings.Fields(quote), " "))

	// First try: exact substring match on first 80 chars (fast path)
	checkLen := len(normQuote)
	if checkLen > 80 {
		checkLen = 80
	}
	if strings.Contains(normContent, normQuote[:checkLen]) {
		return true
	}

	// Second try: token overlap — if 60%+ of significant quote tokens appear in content, accept
	quoteTokens := strings.Fields(normQuote)
	if len(quoteTokens) < 3 {
		return false
	}
	contentSet := make(map[string]bool)
	for _, t := range strings.Fields(normContent) {
		contentSet[t] = true
	}
	matches := 0
	significant := 0
	for _, t := range quoteTokens {
		if len(t) < 3 {
			continue // skip tiny words (a, an, the, is) — don't count in numerator or denominator
		}
		significant++
		if contentSet[t] {
			matches++
		}
	}
	if significant < 3 {
		return false
	}
	ratio := float64(matches) / float64(significant)
	return ratio >= 0.6
}

// ExtractMemoriesFromConversation extracts facts from chat messages with context
func (s *Server) ExtractMemoriesFromConversation(messages []map[string]string) ([]MemoryExtraction, error) {
	if len(messages) == 0 {
		return nil, nil
	}

	// Build conversation text
	var convoBuilder strings.Builder
	for _, m := range messages {
		role := m["role"]
		content := m["content"]
		convoBuilder.WriteString(fmt.Sprintf("%s: %s\n", role, content))
	}
	convoText := convoBuilder.String()

	prompt := fmt.Sprintf(`Extract PERSONAL FACTS about the USER from this conversation.

CONVERSATION:
---
%s
---

RULES:
1. Extract facts about the USER — inference OK if supported by their actual words
2. Include the actual user message as the quote (evidence)
3. Confidence 0.9+ for direct statements ("I live in X"), 0.7-0.9 for supported inference
4. Do NOT extract what the assistant said or assumed — only what the USER stated or confirmed
5. NO GUESSING — if the user didn't say it, don't extract it
6. Look for: preferences, decisions made, goals stated, projects discussed, emotions expressed
7. Business context: tools chosen, strategies discussed, pricing decisions, product direction
8. Relationships: people mentioned, collaborators, clients, partners
9. Extract ACTIONS taken ("I just shipped X", "I deployed Y") as temporal facts

OUTPUT FORMAT (JSON array):
[
  {
    "memory": "User prefers morning meetings",
    "quote": "user: I'm definitely a morning person, so let's schedule calls before noon",
    "confidence": 0.9
  }
]

If no personal facts found, return: []`, convoText)

	result, err := s.callLLMForExtraction(prompt)
	if err != nil {
		return nil, err
	}

	var extractions []struct {
		Memory     string  `json:"memory"`
		Quote      string  `json:"quote"`
		Confidence float64 `json:"confidence"`
	}

	start := strings.Index(result, "[")
	end := strings.LastIndex(result, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(result[start:end+1]), &extractions); err != nil {
		log.Printf("[memory-extract] Conversation JSON parse error: %v", err)
		return nil, nil
	}

	var memories []MemoryExtraction
	for _, e := range extractions {
		if e.Confidence < 0.7 || e.Memory == "" {
			continue
		}
		memories = append(memories, MemoryExtraction{
			MemoryText:    e.Memory,
			SourceType:    "conversation",
			SourceFile:    time.Now().Format("2006-01-02"),
			SourceContext: e.Quote,
			Confidence:    e.Confidence,
		})
	}

	return memories, nil
}

// ExtractMemoryFromPairing creates a memory from a pairing question answer
func ExtractMemoryFromPairing(questionID, questionText, answerText string) MemoryExtraction {
	return MemoryExtraction{
		MemoryText:    answerText,
		SourceType:    "pairing",
		SourceFile:    questionID,
		SourceSection: "Onboarding Questionnaire",
		SourceContext: fmt.Sprintf("Question: %s\n\nYour answer: %s", questionText, answerText),
		Confidence:    0.95, // Direct user input = high confidence
	}
}

// QueueMemoryForApproval adds an extraction to the approval queue with dedup.
// Skips if an identical memory_text already exists (any status).
// Auto-approves high-confidence items per MEMORY_APPROVAL_SYSTEM.md rules.
func (s *Server) QueueMemoryForApproval(m MemoryExtraction) error {
	// Dedup: check exact match first, then fuzzy match against recent items
	var existing int
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_queue WHERE memory_text = ?`, m.MemoryText).Scan(&existing)
	if existing > 0 {
		return nil // exact match, skip silently
	}

	// Fuzzy dedup: check if a semantically similar memory already exists.
	// Normalize current memory to token set, compare against recent queue items.
	normNew := strings.ToLower(strings.Join(strings.Fields(m.MemoryText), " "))
	newTokens := make(map[string]bool)
	for _, t := range strings.Fields(normNew) {
		if len(t) >= 3 {
			newTokens[t] = true
		}
	}
	if len(newTokens) >= 3 {
		// Check against items from same source file (most likely duplicates)
		isDupe := false
		rows, err := s.db.Query(`SELECT memory_text FROM memory_queue WHERE source_file LIKE ? LIMIT 100`,
			strings.Split(m.SourceFile, " [")[0]+"%")
		if err == nil {
			for rows.Next() {
				var existingText string
				rows.Scan(&existingText)
				normExisting := strings.ToLower(strings.Join(strings.Fields(existingText), " "))
				existingTokens := make(map[string]bool)
				for _, t := range strings.Fields(normExisting) {
					if len(t) >= 3 {
						existingTokens[t] = true
					}
				}
				overlap := 0
				for t := range newTokens {
					if existingTokens[t] {
						overlap++
					}
				}
				if len(existingTokens) > 0 {
					fwd := float64(overlap) / float64(len(newTokens))
					rev := float64(overlap) / float64(len(existingTokens))
					if fwd >= 0.8 && rev >= 0.8 {
						isDupe = true
						break
					}
				}
			}
			rows.Close()
		}
		if isDupe {
			return nil // near-duplicate, skip
		}
	}

	id := fmt.Sprintf("mem-%d-%04x", time.Now().UnixNano(), rand.Intn(0xFFFF))
	sourceFile := m.SourceFile
	if m.SourceSection != "" {
		sourceFile = fmt.Sprintf("%s [%s]", m.SourceFile, m.SourceSection)
	}

	// Auto-approve rules (from MEMORY_APPROVAL_SYSTEM.md):
	//   - conf >= 0.95 AND source = direct_statement/pairing → auto-approve
	//   - conf >= 0.95 AND no ambiguous entities → auto-approve
	//   - conf < 0.7 → always queue for review
	//   - source = bulk_document_scan → always queue for review
	status := "pending"
	hasAmbiguousEntities := containsAmbiguousEntities(m.MemoryText)

	// Bulk document scans always queue for review regardless of confidence
	isBulkScan := m.SourceType == "obsidian" || m.SourceType == "gdrive" || m.SourceType == "dropbox" ||
		m.SourceType == "ai_chat" || m.SourceType == "journal" || m.SourceType == "client_note"

	if m.Confidence >= 0.95 && m.SourceType == "pairing" {
		status = "approved"
	} else if m.Confidence >= 0.95 && !hasAmbiguousEntities && !isBulkScan {
		status = "approved"
	}
	// Everything else stays "pending" (including all bulk scans, <0.95 conf, or entity-bearing)

	_, err := s.db.Exec(`
		INSERT INTO memory_queue (id, memory_text, source_type, source_file, source_context, confidence, status, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CASE WHEN ?='approved' THEN CURRENT_TIMESTAMP ELSE NULL END)`,
		id, m.MemoryText, m.SourceType, sourceFile, m.SourceContext, m.Confidence, status, status)
	if err != nil {
		return err
	}

	// If auto-approved AND recorded in DB, fan out to all memory layers
	if status == "approved" {
		go s.writebackApprovedMemory(id, m.MemoryText)
		log.Printf("[memory-queue] Auto-approved (conf=%.2f, type=%s): %s", m.Confidence, m.SourceType, m.MemoryText[:min(60, len(m.MemoryText))])
	}

	return nil
}

// containsAmbiguousEntities checks for location/person/org names that need human disambiguation.
// Uses word-level matching to avoid false positives like "coronavirus" matching "corona".
func containsAmbiguousEntities(text string) bool {
	ambiguousLocations := map[string]bool{
		"providence": true, "corona": true, "riverside": true,
		"portland": true, "springfield": true,
	}
	for _, word := range strings.Fields(strings.ToLower(text)) {
		word = strings.Trim(word, ".,!?;:\"'()-")
		if ambiguousLocations[word] {
			return true
		}
	}
	return false
}

// callLLMForExtraction calls the gateway for memory extraction
func (s *Server) callLLMForExtraction(prompt string) (string, error) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://127.0.0.1:18789"
	}
	gwToken := gatewayToken
	if gwToken == "" {
		gwToken = authToken // fallback to scoreboard token
	}

	body := map[string]interface{}{
		"model": envOr("EXTRACTION_MODEL", "kimi-coding/k2p5"),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 4000,
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", gatewayURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if gwToken != "" {
		req.Header.Set("Authorization", "Bearer "+gwToken)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("gateway returned %d: %s", resp.StatusCode, string(respBody[:min(200, len(respBody))]))
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", err
	}
	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return chatResp.Choices[0].Message.Content, nil
}

// vaultExtractMu guards vaultExtractRunning to prevent concurrent vault extractions.
// Two simultaneous runs would race on the watermark file and double-extract every file.
var vaultExtractMu sync.Mutex
var vaultExtractRunning bool

// POST /v1/memory/extract-vault — Extract memories from vault with real context
func (s *Server) handleMemoryExtractVault(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}

	var body struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"` // max files to process
	}
	json.NewDecoder(r.Body).Decode(&body)
	
	vaultPath := body.Path
	if vaultPath == "" {
		vaultPath = "/data/wirebot/obsidian"
	}
	limit := body.Limit
	if limit == 0 {
		limit = 50
	}

	vaultExtractMu.Lock()
	if vaultExtractRunning {
		vaultExtractMu.Unlock()
		writeJSON(w, map[string]interface{}{
			"message": "Extraction already in progress — ignoring duplicate request",
			"path":    vaultPath,
			"limit":   limit,
		})
		return
	}
	vaultExtractRunning = true
	vaultExtractMu.Unlock()

	writeJSON(w, map[string]interface{}{
		"message": "Memory extraction started in background",
		"path":    vaultPath,
		"limit":   limit,
	})

	go func() {
		defer func() {
			vaultExtractMu.Lock()
			vaultExtractRunning = false
			vaultExtractMu.Unlock()
		}()
		processed := 0
		extracted := 0
		queued := 0
		skipped := 0

		// Load watermark: set of already-processed file paths
		watermarkPath := "/data/wirebot/scoreboard/vault_watermark.json"
		processedFiles := map[string]bool{}
		if data, err := os.ReadFile(watermarkPath); err == nil {
			var files []string
			if json.Unmarshal(data, &files) == nil {
				for _, f := range files {
					processedFiles[f] = true
				}
			}
		}

		filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if processed >= limit {
				return filepath.SkipAll
			}
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			relPath := strings.TrimPrefix(path, vaultPath+"/")

			// Skip already-processed files (watermark)
			if processedFiles[relPath] {
				skipped++
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			memories, err := s.ExtractMemoriesFromDocument(relPath, string(data))
			if err != nil {
				log.Printf("[memory-extract] Error extracting from %s: %v", relPath, err)
				return nil
			}

			queueOK := true
			for _, m := range memories {
				if err := s.QueueMemoryForApproval(m); err != nil {
					log.Printf("[memory-extract] Queue error for %s: %v", relPath, err)
					queueOK = false
				} else {
					queued++
				}
				extracted++
			}

			// Only watermark if all queue operations succeeded (or no memories found)
			if !queueOK {
				return nil // don't watermark — retry next run
			}
			processedFiles[relPath] = true
			processed++
			if processed%10 == 0 {
				log.Printf("[memory-extract] Processed %d files (skipped %d), extracted %d memories", processed, skipped, extracted)
				// Save watermark periodically so crashes resume from here
				s.saveVaultWatermark(watermarkPath, processedFiles)
			}

			// Rate limit to avoid hammering the LLM
			time.Sleep(2 * time.Second)
			return nil
		})

		// Final watermark save
		s.saveVaultWatermark(watermarkPath, processedFiles)
		log.Printf("[memory-extract] Complete: %d files processed, %d skipped, %d memories extracted, %d queued", processed, skipped, extracted, queued)
	}()
}

// convoExtractMu is separate from chatExtractMu (main.go) because they guard
// different chat surfaces: this guards the API endpoint (Discord/agent via agent_end hook),
// while chatExtractMu guards the wins portal chat proxy (extractConversationToQueue).
var convoExtractMu sync.Mutex
var lastConvoExtractTime time.Time

// POST /v1/memory/extract-conversation — Extract memories from a conversation exchange.
// Called by the wirebot-memory-bridge plugin's agent_end hook to route Discord/agent
// conversations through the approval pipeline. Separate rate limiter from wins portal.
// Rate-limited to once per 2 minutes to avoid hammering the LLM.
func (s *Server) handleMemoryExtractConversation(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method != "POST" {
		http.Error(w, `{"error":"POST only"}`, 405)
		return
	}

	var body struct {
		UserMessage      string `json:"user_message"`
		AssistantMessage string `json:"assistant_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserMessage == "" {
		http.Error(w, `{"error":"user_message required"}`, 400)
		return
	}

	if len(body.UserMessage) < 20 {
		writeJSON(w, map[string]interface{}{"ok": true, "message": "Skipped (too short)"})
		return
	}

	// Rate limit: at most once per 2 minutes (separate from wins portal's chatExtractMu)
	convoExtractMu.Lock()
	if time.Since(lastConvoExtractTime) < 2*time.Minute {
		convoExtractMu.Unlock()
		writeJSON(w, map[string]interface{}{"ok": true, "message": "Skipped (rate limited)"})
		return
	}
	lastConvoExtractTime = time.Now()
	convoExtractMu.Unlock()

	go func() {
		messages := []map[string]string{
			{"role": "user", "content": body.UserMessage},
			{"role": "assistant", "content": body.AssistantMessage},
		}
		memories, err := s.ExtractMemoriesFromConversation(messages)
		if err != nil {
			log.Printf("[convo-extract-api] Extraction error: %v", err)
			return
		}
		for _, m := range memories {
			if err := s.QueueMemoryForApproval(m); err != nil {
				log.Printf("[convo-extract-api] Queue error: %v", err)
			}
		}
		if len(memories) > 0 {
			log.Printf("[convo-extract-api] Extracted %d memories from conversation", len(memories))
		}
	}()

	writeJSON(w, map[string]interface{}{"ok": true, "message": "Extraction queued"})
}

func (s *Server) saveVaultWatermark(path string, processed map[string]bool) {
	files := make([]string, 0, len(processed))
	for f := range processed {
		files = append(files, f)
	}
	data, _ := json.Marshal(files)
	os.WriteFile(path, data, 0644)
}
