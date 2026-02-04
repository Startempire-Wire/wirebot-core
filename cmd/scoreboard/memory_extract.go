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

// ExtractMemoriesFromDocument uses LLM to extract personal facts with real quoted context
func (s *Server) ExtractMemoriesFromDocument(docPath, content string) ([]MemoryExtraction, error) {
	// Don't process tiny files
	if len(content) < 100 {
		return nil, nil
	}

	// Truncate very long docs to first 8000 chars for extraction
	extractContent := content
	if len(extractContent) > 8000 {
		extractContent = extractContent[:8000]
	}

	// Build LLM prompt for extraction — includes entity disambiguation, category, reasoning
	prompt := fmt.Sprintf(`You are extracting PERSONAL FACTS about the document author from their writing.

DOCUMENT: %s
---
%s
---

RULES:
1. Extract facts about the AUTHOR — inference OK if supported by quoted text
2. Each fact MUST have a direct quote (2-4 sentences) from the document as evidence
3. Confidence scoring:
   - 0.9+ explicit statement ("I live in X", "my name is Y")
   - 0.7-0.9 supported inference ("moved to Corona last year" → lives in Corona)
   - 0.5-0.7 weak inference (mentioned but unclear if about the author)
   - Below 0.5: do not extract
4. Focus on: location, work, preferences, relationships, goals, habits, beliefs, business
5. Do NOT extract general knowledge or opinions about external topics
6. NO GUESSING — if you can't quote evidence, don't extract it
7. FLAG ENTITIES that need disambiguation (e.g. "Providence" = hospital? city? organization?)

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

CATEGORIES: identity, preference, fact, relationship, business, temporal, habit
ENTITIES: List any person names, locations, organizations, or products mentioned.
REASONING: One sentence explaining why you assigned that confidence level.

If no personal facts can be extracted with real quotes, return: []`, docPath, extractContent)

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
		if !containsFuzzy(content, e.Quote) {
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
		memories = append(memories, MemoryExtraction{
			MemoryText:    e.Memory,
			SourceType:    "obsidian",
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
1. Extract facts about the USER - inference OK if supported by their actual words
2. Include the actual user message as the quote (evidence)
3. Confidence 0.9+ for direct statements ("I live in X"), 0.7-0.9 for supported inference
4. Do NOT extract what the assistant said or assumed - only what the USER stated
5. NO GUESSING - if the user didn't say it, don't extract it

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
	// Dedup: check if this exact text already exists in queue
	var existing int
	s.db.QueryRow(`SELECT COUNT(*) FROM memory_queue WHERE memory_text = ?`, m.MemoryText).Scan(&existing)
	if existing > 0 {
		return nil // already queued, skip silently
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
	isBulkScan := m.SourceType == "obsidian" || m.SourceType == "gdrive" || m.SourceType == "dropbox"

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

	// If auto-approved, fan out to all memory layers
	if status == "approved" {
		go s.writebackApprovedMemory(m.MemoryText)
		log.Printf("[memory-queue] Auto-approved (conf=%.2f, type=%s): %s", m.Confidence, m.SourceType, m.MemoryText[:min(60, len(m.MemoryText))])
	}

	return err
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
		"model": "claude-sonnet-4-20250514",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 2000,
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

	writeJSON(w, map[string]interface{}{
		"message": "Memory extraction started in background",
		"path":    vaultPath,
		"limit":   limit,
	})

	go func() {
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

			for _, m := range memories {
				if err := s.QueueMemoryForApproval(m); err == nil {
					queued++
				}
				extracted++
			}

			// Mark file as processed in watermark
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

func (s *Server) saveVaultWatermark(path string, processed map[string]bool) {
	files := make([]string, 0, len(processed))
	for f := range processed {
		files = append(files, f)
	}
	data, _ := json.Marshal(files)
	os.WriteFile(path, data, 0644)
}
