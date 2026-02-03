package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
func (s *Server) ExtractMemoriesFromDocument(filepath, content string) ([]MemoryExtraction, error) {
	// Don't process tiny files
	if len(content) < 100 {
		return nil, nil
	}

	// Truncate very long docs to first 8000 chars for extraction
	extractContent := content
	if len(extractContent) > 8000 {
		extractContent = extractContent[:8000]
	}

	// Build LLM prompt for extraction
	prompt := fmt.Sprintf(`You are extracting PERSONAL FACTS about the document author from their writing.

DOCUMENT: %s
---
%s
---

RULES:
1. Extract ONLY facts that are explicitly stated, NOT inferred
2. Each fact must have a direct quote from the document as context
3. Confidence: 0.9+ for explicit statements, 0.7-0.9 for clear implications, skip anything below 0.7
4. Focus on: location, work, preferences, relationships, goals, habits, beliefs
5. Do NOT extract general knowledge or opinions about external topics
6. The quote must be 2-4 actual sentences from the document

OUTPUT FORMAT (JSON array):
[
  {
    "memory": "The author lives in Corona, California",
    "quote": "I've been living in Corona for the past 5 years. The Inland Empire heat takes some getting used to, but I love the community here.",
    "section": "About Me",
    "confidence": 0.95
  }
]

If no personal facts can be extracted with real quotes, return: []`, filepath, extractContent)

	// Call gateway for extraction
	result, err := s.callLLMForExtraction(prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	var extractions []struct {
		Memory     string  `json:"memory"`
		Quote      string  `json:"quote"`
		Section    string  `json:"section"`
		Confidence float64 `json:"confidence"`
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
		if e.Confidence < 0.7 || e.Memory == "" || e.Quote == "" {
			continue
		}
		// Verify quote actually exists in content (fuzzy match)
		if !containsFuzzy(content, e.Quote) {
			log.Printf("[memory-extract] Quote not found in doc, skipping: %s", e.Quote[:min(50, len(e.Quote))])
			continue
		}
		memories = append(memories, MemoryExtraction{
			MemoryText:    e.Memory,
			SourceType:    "obsidian",
			SourceFile:    filepath,
			SourceSection: e.Section,
			SourceContext: e.Quote,
			Confidence:    e.Confidence,
		})
	}

	return memories, nil
}

// containsFuzzy checks if quote exists in content (allows minor differences)
func containsFuzzy(content, quote string) bool {
	// Normalize both
	normContent := strings.ToLower(strings.Join(strings.Fields(content), " "))
	normQuote := strings.ToLower(strings.Join(strings.Fields(quote), " "))
	
	// Check first 50 chars of quote
	if len(normQuote) > 50 {
		normQuote = normQuote[:50]
	}
	
	return strings.Contains(normContent, normQuote)
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
1. Only extract facts the USER explicitly stated about themselves
2. Include the actual user message as the quote
3. Confidence 0.9+ for direct statements ("I live in X"), 0.7-0.9 for clear context
4. Do NOT extract what the assistant said or assumed

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
	if start == -1 || end == -1 {
		return nil, nil
	}

	if err := json.Unmarshal([]byte(result[start:end+1]), &extractions); err != nil {
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

// QueueMemoryForApproval adds an extraction to the approval queue
func (s *Server) QueueMemoryForApproval(m MemoryExtraction) error {
	id := fmt.Sprintf("mem-%d", time.Now().UnixNano())
	_, err := s.db.Exec(`
		INSERT INTO memory_queue (id, memory_text, source_type, source_file, source_context, confidence)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, m.MemoryText, m.SourceType, 
		fmt.Sprintf("%s [%s]", m.SourceFile, m.SourceSection),
		m.SourceContext, m.Confidence)
	return err
}

// callLLMForExtraction calls the gateway for memory extraction
func (s *Server) callLLMForExtraction(prompt string) (string, error) {
	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://127.0.0.1:5005"
	}
	gatewayToken := os.Getenv("GATEWAY_TOKEN")

	body := map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 2000,
	}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", gatewayURL+"/v1/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if gatewayToken != "" {
		req.Header.Set("Authorization", "Bearer "+gatewayToken)
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	
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

		filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || processed >= limit {
				return nil
			}
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			relPath := strings.TrimPrefix(path, vaultPath+"/")
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

			processed++
			if processed%10 == 0 {
				log.Printf("[memory-extract] Processed %d files, extracted %d memories", processed, extracted)
			}
			
			// Rate limit to avoid hammering the LLM
			time.Sleep(2 * time.Second)
			return nil
		})

		log.Printf("[memory-extract] Complete: %d files, %d memories extracted, %d queued", processed, extracted, queued)
	}()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
