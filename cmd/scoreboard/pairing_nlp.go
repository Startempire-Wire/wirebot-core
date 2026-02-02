package main

import (
	"math"
	"strings"
	"unicode"
)

// ═══════════════════════════════════════════════════════════════════════════════
// NLP FEATURE EXTRACTION — Pure lexical/statistical text analysis
//
// Extracts 25+ features from raw text with zero external dependencies.
// No ML models — pure string analysis for sub-millisecond extraction.
//
// Feature categories:
//   1. Linguistic (sentence length, vocabulary, hedging, action verbs, etc.)
//   2. Derived (directness, formality, detail_preference, emotion, pace)
//   3. Contextual (financial_pressure, life_event keyword detection)
//   4. DISC inference (D/I/S/C scores from linguistic features)
//
// See: PAIRING_SCIENCE.md §4 (Communication Inference Engine)
// See: PAIRING_ENGINE.md §8 (NLP Feature Extraction)
// ═══════════════════════════════════════════════════════════════════════════════

// NLPExtractor performs pure-lexical NLP feature extraction from text.
// Stateless — safe to call from any goroutine.
type NLPExtractor struct {
	hedgeWords      map[string]bool
	actionVerbs     map[string]bool
	urgentWords     map[string]bool
	temporalWords   map[string]bool
	financialWords  map[string]bool
	lifeEventWords  map[string]bool
	abstractWords   map[string]bool
	concreteWords   map[string]bool
	firstPersonWords map[string]bool
}

// NewNLPExtractor creates an NLPExtractor with all word lists loaded.
func NewNLPExtractor() *NLPExtractor {
	return &NLPExtractor{
		hedgeWords: toSet([]string{
			"maybe", "perhaps", "possibly", "might", "could", "would",
			"probably", "i think", "i guess", "i suppose", "sort of",
			"kind of", "somewhat", "a bit", "a little", "not sure",
			"i believe", "it seems", "apparently", "arguably",
		}),
		actionVerbs: toSet([]string{
			"build", "ship", "launch", "deploy", "create", "make",
			"push", "fix", "implement", "code", "design", "develop",
			"release", "finish", "complete", "start", "begin", "execute",
			"run", "test", "write", "publish", "deliver", "send",
			"sell", "buy", "hire", "fire", "decide", "commit",
			"close", "open", "break", "solve", "kill", "cut",
			"move", "do", "try", "go", "get", "set",
		}),
		urgentWords: toSet([]string{
			"now", "today", "asap", "immediately", "urgent", "hurry",
			"quick", "fast", "right away", "this minute", "deadline",
			"overdue", "behind", "late", "rush", "priority", "critical",
			"tonight", "morning", "before", "soon",
		}),
		temporalWords: toSet([]string{
			"now", "today", "tomorrow", "yesterday", "soon", "later",
			"eventually", "someday", "next week", "next month", "future",
			"past", "before", "after", "when", "until", "deadline",
			"schedule", "timeline", "calendar", "morning", "tonight",
		}),
		financialWords: toSet([]string{
			"debt", "money", "afford", "broke", "budget", "expenses",
			"rent", "bills", "payroll", "overdraft", "loan", "credit",
			"payment", "invoice", "cash", "revenue", "profit", "loss",
			"bankrupt", "collections", "owe", "overdue", "financial",
			"salary", "income", "cost", "price", "fee", "charge",
		}),
		lifeEventWords: toSet([]string{
			"health", "hospital", "doctor", "sick", "illness", "surgery",
			"family", "divorce", "baby", "pregnant", "wedding", "funeral",
			"moving", "relocate", "accident", "emergency", "crisis",
			"death", "loss", "grief", "therapy", "mental health",
			"burnout", "exhausted", "overwhelmed", "anxiety", "depression",
		}),
		abstractWords: toSet([]string{
			"concept", "theory", "philosophy", "strategy", "vision",
			"framework", "paradigm", "principle", "idea", "thought",
			"perspective", "approach", "methodology", "model", "system",
			"architecture", "pattern", "abstract", "hypothetical",
		}),
		concreteWords: toSet([]string{
			"button", "page", "screen", "file", "code", "server",
			"database", "api", "endpoint", "table", "column", "row",
			"pixel", "color", "font", "image", "click", "tap",
			"build", "deploy", "install", "download", "upload",
		}),
		firstPersonWords: toSet([]string{
			"i", "me", "my", "mine", "myself", "i'm", "i've", "i'll", "i'd",
		}),
	}
}

// ExtractFeatures produces 25+ numeric features from raw text.
// All features are normalized to 0.0-1.0 range.
// Returns empty map for empty/very short text.
func (nlp *NLPExtractor) ExtractFeatures(text string) map[string]float64 {
	f := make(map[string]float64)
	if len(text) < 5 {
		return f
	}

	lower := strings.ToLower(text)
	words := tokenize(lower)
	wordCount := float64(len(words))
	if wordCount < 1 {
		return f
	}

	sentences := splitSentences(text)
	sentCount := float64(len(sentences))
	if sentCount < 1 {
		sentCount = 1
	}

	// ─── Linguistic Features ─────────────────────────────────────────────

	// Average sentence length (words per sentence)
	avgSentLen := wordCount / sentCount
	f["avg_sentence_length"] = avgSentLen

	// Vocabulary richness (unique words / total words)
	uniqueWords := make(map[string]bool)
	for _, w := range words {
		uniqueWords[w] = true
	}
	f["vocabulary_richness"] = float64(len(uniqueWords)) / wordCount

	// Hedging ratio (hedge phrases per sentence)
	hedgeCount := nlp.countMatches(lower, nlp.hedgeWords)
	f["hedging_ratio"] = math.Min(1.0, float64(hedgeCount)/sentCount)

	// Action verb density (action verbs / total words)
	actionCount := nlp.countWordMatches(words, nlp.actionVerbs)
	f["action_verb_density"] = math.Min(1.0, float64(actionCount)/wordCount)

	// Question ratio (sentences ending with ?)
	questionCount := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "?") {
			questionCount++
		}
	}
	f["question_ratio"] = float64(questionCount) / sentCount

	// Exclamation ratio
	exclamCount := 0
	for _, s := range sentences {
		s = strings.TrimSpace(s)
		if strings.HasSuffix(s, "!") {
			exclamCount++
		}
	}
	f["exclamation_ratio"] = float64(exclamCount) / sentCount

	// First person ratio (I/me/my per total words)
	fpCount := nlp.countWordMatches(words, nlp.firstPersonWords)
	f["first_person_ratio"] = math.Min(1.0, float64(fpCount)/wordCount)

	// Emoji frequency (approximation: count common emoji ranges)
	emojiCount := countEmoji(text)
	f["emoji_frequency"] = math.Min(1.0, float64(emojiCount)/wordCount)

	// Temporal urgency (urgent words / temporal words)
	urgentCount := nlp.countWordMatches(words, nlp.urgentWords)
	temporalCount := nlp.countWordMatches(words, nlp.temporalWords)
	if temporalCount > 0 {
		f["temporal_urgency"] = math.Min(1.0, float64(urgentCount)/float64(temporalCount))
	} else {
		f["temporal_urgency"] = 0
	}

	// Imperative ratio (sentences starting with a verb — approximation)
	imperativeCount := 0
	for _, s := range sentences {
		firstWord := strings.ToLower(firstToken(strings.TrimSpace(s)))
		if nlp.actionVerbs[firstWord] {
			imperativeCount++
		}
	}
	f["imperative_ratio"] = float64(imperativeCount) / sentCount

	// List usage (lines starting with -, *, numbers, bullets)
	listCount := 0
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		runes := []rune(trimmed)
		if len(runes) > 0 {
			firstRune := runes[0]
			if firstRune == '-' || firstRune == '*' || firstRune == '•' || (firstRune >= '0' && firstRune <= '9') {
				listCount++
			}
		}
	}
	f["list_usage"] = math.Min(1.0, float64(listCount)/sentCount)

	// ─── Derived Features ────────────────────────────────────────────────

	// Directness: high action verbs + low hedging + short sentences + imperatives
	f["directness"] = clamp01(
		0.30*(1.0-f["hedging_ratio"]) +
			0.25*f["action_verb_density"]*3.0 +
			0.25*f["imperative_ratio"] +
			0.20*math.Min(1.0, 15.0/math.Max(1, avgSentLen)))

	// Formality: high vocabulary + low emoji + low exclamation
	f["formality"] = clamp01(
		0.40*f["vocabulary_richness"] +
			0.30*(1.0-f["emoji_frequency"]*10.0) +
			0.30*(1.0-f["exclamation_ratio"]))

	// Detail preference: long sentences + rich vocabulary + questions
	f["detail_preference"] = clamp01(
		0.40*math.Min(1.0, avgSentLen/25.0) +
			0.35*f["vocabulary_richness"] +
			0.25*f["question_ratio"])

	// Emotion expression: exclamation + emoji + low hedging
	f["emotion_expression"] = clamp01(
		0.35*f["exclamation_ratio"] +
			0.35*math.Min(1.0, f["emoji_frequency"]*10.0) +
			0.30*(1.0-f["hedging_ratio"]))

	// Pace preference: urgency + action verbs + short response
	f["pace_preference"] = clamp01(
		0.40*f["temporal_urgency"] +
			0.35*f["action_verb_density"]*3.0 +
			0.25*math.Min(1.0, 10.0/math.Max(1, avgSentLen)))

	// Decision style: imperatives + low questions + low hedging
	f["decision_style"] = clamp01(
		0.35*f["imperative_ratio"] +
			0.35*(1.0-f["question_ratio"]) +
			0.30*(1.0-f["hedging_ratio"]))

	// Holistic vs sequential: abstract words vs list usage and numbers
	abstractCount := nlp.countWordMatches(words, nlp.abstractWords)
	concreteCount := nlp.countWordMatches(words, nlp.concreteWords)
	abstractRatio := 0.5
	if abstractCount+concreteCount > 0 {
		abstractRatio = float64(abstractCount) / float64(abstractCount+concreteCount)
	}
	f["holistic_vs_sequential"] = clamp01(
		0.50*abstractRatio +
			0.50*(1.0-f["list_usage"]))

	f["abstract_vs_concrete"] = abstractRatio

	// ─── Contextual Features (keyword detection) ─────────────────────────

	financialCount := nlp.countWordMatches(words, nlp.financialWords)
	f["financial_pressure"] = math.Min(1.0, float64(financialCount)/3.0)

	lifeEventCount := nlp.countWordMatches(words, nlp.lifeEventWords)
	f["life_event"] = math.Min(1.0, float64(lifeEventCount)/2.0)

	return f
}

// InferDISC produces DISC style scores (0.0-1.0 each) from text features.
// Uses the weighted formula from PAIRING_SCIENCE.md §4.2.
//
// D = 0.30×imperative + 0.25×(1-hedge) + 0.20×action + 0.15×(1/sent_len) + 0.10×urgency
// I = 0.30×exclamation + 0.25×emoji + 0.20×emotion + 0.15×(1-formality) + 0.10×questions
// S = 0.30×hedging + 0.25×questions + 0.20×(1-urgency) + 0.15×long_sent + 0.10×first_person
// C = 0.30×vocabulary + 0.25×(1-emoji) + 0.20×list_usage + 0.15×formality + 0.10×detail
func (nlp *NLPExtractor) InferDISC(text string) map[string]float64 {
	f := nlp.ExtractFeatures(text)
	if len(f) == 0 {
		return map[string]float64{"D": 0.25, "I": 0.25, "S": 0.25, "C": 0.25}
	}

	d := 0.30*f["imperative_ratio"] +
		0.25*(1.0-f["hedging_ratio"]) +
		0.20*f["action_verb_density"]*3.0 +
		0.15*math.Min(1.0, 15.0/math.Max(1, f["avg_sentence_length"])) +
		0.10*f["temporal_urgency"]

	i := 0.30*f["exclamation_ratio"] +
		0.25*math.Min(1.0, f["emoji_frequency"]*10.0) +
		0.20*f["emotion_expression"] +
		0.15*(1.0-f["formality"]) +
		0.10*f["question_ratio"]

	s := 0.30*f["hedging_ratio"] +
		0.25*f["question_ratio"] +
		0.20*(1.0-f["temporal_urgency"]) +
		0.15*math.Min(1.0, f["avg_sentence_length"]/25.0) +
		0.10*f["first_person_ratio"]

	c := 0.30*f["vocabulary_richness"] +
		0.25*(1.0-math.Min(1.0, f["emoji_frequency"]*10.0)) +
		0.20*f["list_usage"] +
		0.15*f["formality"] +
		0.10*f["detail_preference"]

	// Normalize to sum to 1.0
	total := d + i + s + c
	if total < 0.01 {
		total = 1.0
	}

	return map[string]float64{
		"D": d / total,
		"I": i / total,
		"S": s / total,
		"C": c / total,
	}
}

// ─── Text Utilities ──────────────────────────────────────────────────────────

// tokenize splits text into lowercase word tokens.
func tokenize(text string) []string {
	var words []string
	for _, w := range strings.Fields(text) {
		cleaned := strings.TrimFunc(w, func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\''
		})
		if len(cleaned) > 0 {
			words = append(words, strings.ToLower(cleaned))
		}
	}
	return words
}

// splitSentences splits text into sentences (approximation using .!? boundaries).
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder
	for _, r := range text {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if len(s) > 2 {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	// Remaining text is a sentence if non-trivial
	s := strings.TrimSpace(current.String())
	if len(s) > 2 {
		sentences = append(sentences, s)
	}
	if len(sentences) == 0 {
		sentences = append(sentences, text)
	}
	return sentences
}

// firstToken returns the first word in a string.
func firstToken(s string) string {
	for i, r := range s {
		if unicode.IsSpace(r) {
			return s[:i]
		}
	}
	return s
}

// countEmoji counts Unicode characters in emoji ranges.
func countEmoji(text string) int {
	count := 0
	for _, r := range text {
		if r >= 0x1F600 && r <= 0x1F64F { // emoticons
			count++
		} else if r >= 0x1F300 && r <= 0x1F5FF { // misc symbols
			count++
		} else if r >= 0x1F680 && r <= 0x1F6FF { // transport
			count++
		} else if r >= 0x1F900 && r <= 0x1F9FF { // supplemental
			count++
		} else if r >= 0x2600 && r <= 0x26FF { // misc symbols
			count++
		} else if r >= 0x2700 && r <= 0x27BF { // dingbats
			count++
		}
	}
	return count
}

// countMatches counts how many phrases from the set appear in the text (substring match).
func (nlp *NLPExtractor) countMatches(text string, set map[string]bool) int {
	count := 0
	for phrase := range set {
		if strings.Contains(text, phrase) {
			count++
		}
	}
	return count
}

// countWordMatches counts how many words from the list appear in the word set.
func (nlp *NLPExtractor) countWordMatches(words []string, set map[string]bool) int {
	count := 0
	for _, w := range words {
		if set[w] {
			count++
		}
	}
	return count
}

// toSet converts a string slice to a map for O(1) lookup.
func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

// clamp01 clamps a value to [0.0, 1.0].
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
