package main

import (
	"math"
	"regexp"
	"strings"
	"time"
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

// ═══════════════════════════════════════════════════════════════════════════════
// VAULT DEEP ANALYSIS — Extended NLP for Obsidian vault / personal writing
//
// Extracts deep personal signals across 10 dimensions:
//   1. Values & Faith — spiritual beliefs, moral framework, guiding principles
//   2. Emotional Patterns — recurring emotional states, coping mechanisms
//   3. Energy Cycles — productivity patterns, time-of-day preferences
//   4. Financial Reality — debt language, money relationship, earning patterns
//   5. Relationship Patterns — social dynamics, loneliness, community
//   6. Health & Body — physical habits, substances, fitness, sleep
//   7. Creative Output — writing patterns, idea generation, project sprawl
//   8. Distraction & Stall — procrastination triggers, avoidance behaviors
//   9. Purpose & Identity — self-concept, calling, mission, legacy
//  10. Productivity Style — planning vs doing, tools obsession, system hopping
// ═══════════════════════════════════════════════════════════════════════════════

// VaultInsight represents aggregated deep analysis of personal writing.
type VaultInsight struct {
	// Core dimensions (0.0-1.0)
	FaithStrength       float64 `json:"faith_strength"`       // Spiritual conviction and practice
	EmotionalVolatility float64 `json:"emotional_volatility"` // Range between emotional highs/lows
	FinancialStress     float64 `json:"financial_stress"`     // Frequency and intensity of money worry
	SocialIsolation     float64 `json:"social_isolation"`     // Loneliness, lack of community signals
	HealthAwareness     float64 `json:"health_awareness"`     // Attention to physical wellbeing
	CreativeEnergy      float64 `json:"creative_energy"`      // Idea generation rate, project creation
	DistractionRisk     float64 `json:"distraction_risk"`     // Susceptibility to tangents, tool hopping
	PurposeClarity      float64 `json:"purpose_clarity"`      // Clear sense of mission and calling
	ProductivityStyle   float64 `json:"productivity_style"`   // 0=planner, 1=doer
	ProjectSprawl       float64 `json:"project_sprawl"`       // Too many projects, finishing rate

	// Derived behavioral signals
	LateNightWorker   bool     `json:"late_night_worker"`
	MorningPerson     bool     `json:"morning_person"`
	JournalingHabit   float64  `json:"journaling_habit"`   // Consistency 0-1
	SelfAwareness     float64  `json:"self_awareness"`     // Introspective depth 0-1
	GrowthMindset     float64  `json:"growth_mindset"`     // Focus on improvement 0-1
	PerfectionismRisk float64  `json:"perfectionism_risk"` // Over-planning, under-shipping

	// Evidence
	TopValues       []string `json:"top_values"`       // Extracted values (faith, family, impact, etc.)
	RecurringThemes []string `json:"recurring_themes"` // Most frequent themes across docs
	EmotionalWords  []string `json:"emotional_words"`  // Top emotional language used
	StallTriggers   []string `json:"stall_triggers"`   // What causes stalls
	TimePatterns    []string `json:"time_patterns"`    // When they write/work
	DocumentCount   int      `json:"document_count"`   // Total docs processed
	WordCount       int      `json:"word_count"`       // Total words analyzed
	DateRange       string   `json:"date_range"`       // Earliest to latest date found
}

// vaultWordLists contains domain-specific word lists for vault analysis.
var vaultWordLists = struct {
	faith        map[string]bool
	struggle     map[string]bool
	purpose      map[string]bool
	distraction  map[string]bool
	health       map[string]bool
	relationship map[string]bool
	creative     map[string]bool
	stall        map[string]bool
	growth       map[string]bool
	substance    map[string]bool
	planning     map[string]bool
	shipping     map[string]bool
}{
	faith: toSet([]string{
		"god", "lord", "jesus", "christ", "prayer", "pray", "prayed", "praying",
		"faith", "spirit", "holy", "scripture", "bible", "church", "worship",
		"blessed", "blessing", "grace", "mercy", "salvation", "sin", "righteous",
		"purity", "repent", "forgive", "kingdom", "heaven", "soul", "anointing",
		"testimony", "minister", "ministry", "calling", "servant", "humble",
		"psalm", "proverbs", "gospel", "disciple", "christian", "believe",
	}),
	struggle: toSet([]string{
		"struggle", "struggling", "hard", "difficult", "pain", "painful",
		"frustrated", "frustrating", "overwhelmed", "hopeless", "helpless",
		"stuck", "trapped", "failing", "failed", "lost", "confused",
		"anxious", "anxiety", "depressed", "depression", "lonely", "alone",
		"afraid", "fear", "worry", "worried", "desperate", "despair",
		"exhausted", "tired", "burnout", "burned out", "broke", "broken",
		"shame", "guilt", "regret", "hatred", "hate", "anger", "angry",
	}),
	purpose: toSet([]string{
		"purpose", "mission", "calling", "destiny", "impact", "legacy",
		"vision", "dream", "goal", "goals", "aspire", "aspiration",
		"meaningful", "meaning", "why", "passion", "passionate",
		"created for", "born to", "designed to", "gifted", "talent",
		"potential", "greatness", "excellence", "succeed", "success",
	}),
	distraction: toSet([]string{
		"distracted", "distraction", "procrastinate", "procrastination",
		"twitter", "x.com", "social media", "surfing", "scrolling",
		"youtube", "rabbit hole", "tangent", "sidetrack", "wasted time",
		"got distracted", "off track", "lost focus", "couldn't focus",
		"shiny object", "new idea", "another idea", "tool", "setup",
		"configuring", "customizing", "tweaking", "researching",
		"forgot", "forgot to", "didn't log", "forgot to journal",
	}),
	health: toSet([]string{
		"health", "healthy", "exercise", "workout", "gym", "run", "running",
		"sleep", "sleeping", "insomnia", "diet", "eating", "nutrition",
		"vitamins", "supplement", "testosterone", "body", "weight",
		"fitness", "muscle", "cardio", "energy", "fatigue", "tired",
		"doctor", "hospital", "sick", "illness", "pain", "injury",
	}),
	relationship: toSet([]string{
		"friend", "friends", "friendship", "family", "wife", "husband",
		"girlfriend", "boyfriend", "dating", "relationship", "married",
		"marriage", "divorce", "children", "kids", "son", "daughter",
		"community", "fellowship", "team", "partner", "accountability",
		"mentor", "mentoring", "lonely", "loneliness", "alone", "isolated",
	}),
	creative: toSet([]string{
		"idea", "ideas", "create", "creating", "creative", "creativity",
		"build", "building", "design", "designing", "write", "writing",
		"book", "podcast", "blog", "project", "projects", "product",
		"launch", "launching", "startup", "business", "app", "website",
		"plugin", "invention", "innovate", "innovation", "brainstorm",
	}),
	stall: toSet([]string{
		"stall", "stalled", "stuck", "blocked", "paralyzed", "frozen",
		"can't start", "haven't started", "dragging", "dragging feet",
		"putting off", "delay", "delayed", "postpone", "postponed",
		"procrastinate", "not shipping", "not finishing", "abandoned",
		"gave up", "quit", "stopped", "paused", "on hold", "backlog",
	}),
	growth: toSet([]string{
		"learn", "learning", "grow", "growing", "growth", "improve",
		"improving", "better", "progress", "evolve", "develop",
		"developing", "skill", "skills", "practice", "training",
		"study", "studying", "read", "reading", "course", "class",
		"mentor", "feedback", "reflect", "reflection", "journal",
	}),
	substance: toSet([]string{
		"weed", "marijuana", "cannabis", "smoking", "smoke", "smoked",
		"alcohol", "drinking", "drunk", "beer", "wine", "liquor",
		"sober", "sobriety", "clean", "abstain", "addiction", "addict",
		"habit", "vice", "temptation", "flesh", "indulgence",
	}),
	planning: toSet([]string{
		"plan", "planning", "blueprint", "roadmap", "strategy", "organize",
		"organizing", "todo", "to-do", "checklist", "list", "outline",
		"schedule", "calendar", "timeline", "milestone", "phase",
		"step 1", "step 2", "step 3", "next step", "workflow",
	}),
	shipping: toSet([]string{
		"ship", "shipped", "shipping", "deploy", "deployed", "launch",
		"launched", "live", "published", "publish", "released", "release",
		"done", "finished", "completed", "delivered", "built", "made",
		"pushed", "merged", "committed", "went live", "in production",
	}),
}

// timePattern matches timestamps like "20:59", "06:51", "15:35" in journal entries
var timePattern = regexp.MustCompile(`\b(\d{1,2}):(\d{2})\s*[-–—]`)

// datePattern matches dates in filenames/content
var datePattern = regexp.MustCompile(`\b(20\d{2})-(\d{2})-(\d{2})\b`)

// AnalyzeVaultDocument extracts deep features from a single vault document.
// Returns feature map + detected themes. The filename/path provides metadata context.
func (nlp *NLPExtractor) AnalyzeVaultDocument(content string, filename string) (map[string]float64, []string) {
	f := nlp.ExtractFeatures(content) // base NLP features
	lower := strings.ToLower(content)
	words := tokenize(lower)
	wordCount := float64(len(words))
	if wordCount < 5 {
		return f, nil
	}

	var themes []string

	// Faith dimension
	faithCount := countWordMatchesStatic(words, vaultWordLists.faith)
	f["vault_faith"] = math.Min(1.0, float64(faithCount)/(wordCount*0.03+1))
	if faithCount > 3 {
		themes = append(themes, "faith")
	}

	// Struggle / emotional pain
	struggleCount := countWordMatchesStatic(words, vaultWordLists.struggle)
	f["vault_struggle"] = math.Min(1.0, float64(struggleCount)/(wordCount*0.02+1))
	if struggleCount > 2 {
		themes = append(themes, "struggle")
	}

	// Purpose & identity
	purposeCount := countWordMatchesStatic(words, vaultWordLists.purpose)
	f["vault_purpose"] = math.Min(1.0, float64(purposeCount)/(wordCount*0.02+1))
	if purposeCount > 2 {
		themes = append(themes, "purpose")
	}

	// Distraction signals
	distractCount := countWordMatchesStatic(words, vaultWordLists.distraction)
	f["vault_distraction"] = math.Min(1.0, float64(distractCount)/(wordCount*0.015+1))
	if distractCount > 1 {
		themes = append(themes, "distraction")
	}

	// Health awareness
	healthCount := countWordMatchesStatic(words, vaultWordLists.health)
	f["vault_health"] = math.Min(1.0, float64(healthCount)/(wordCount*0.02+1))
	if healthCount > 2 {
		themes = append(themes, "health")
	}

	// Relationship/community
	relCount := countWordMatchesStatic(words, vaultWordLists.relationship)
	f["vault_relationship"] = math.Min(1.0, float64(relCount)/(wordCount*0.02+1))
	if relCount > 2 {
		themes = append(themes, "relationship")
	}

	// Creative energy
	createCount := countWordMatchesStatic(words, vaultWordLists.creative)
	f["vault_creative"] = math.Min(1.0, float64(createCount)/(wordCount*0.025+1))
	if createCount > 2 {
		themes = append(themes, "creative")
	}

	// Stall patterns
	stallCount := countWordMatchesStatic(words, vaultWordLists.stall)
	f["vault_stall"] = math.Min(1.0, float64(stallCount)/(wordCount*0.015+1))
	if stallCount > 1 {
		themes = append(themes, "stall")
	}

	// Growth mindset
	growthCount := countWordMatchesStatic(words, vaultWordLists.growth)
	f["vault_growth"] = math.Min(1.0, float64(growthCount)/(wordCount*0.02+1))
	if growthCount > 2 {
		themes = append(themes, "growth")
	}

	// Substance use signals
	subCount := countWordMatchesStatic(words, vaultWordLists.substance)
	f["vault_substance"] = math.Min(1.0, float64(subCount)/(wordCount*0.01+1))

	// Planning vs shipping ratio
	planCount := countWordMatchesStatic(words, vaultWordLists.planning)
	shipCount := countWordMatchesStatic(words, vaultWordLists.shipping)
	if planCount+shipCount > 0 {
		f["vault_plan_vs_ship"] = float64(planCount) / float64(planCount+shipCount) // 1.0=all planning, 0.0=all shipping
	} else {
		f["vault_plan_vs_ship"] = 0.5
	}

	// Time-of-day patterns from journal timestamps
	lateNight := 0
	earlyMorning := 0
	for _, match := range timePattern.FindAllStringSubmatch(content, -1) {
		if len(match) >= 2 {
			hour := 0
			for _, c := range match[1] {
				hour = hour*10 + int(c-'0')
			}
			if hour >= 22 || hour <= 3 {
				lateNight++
			}
			if hour >= 5 && hour <= 8 {
				earlyMorning++
			}
		}
	}
	f["vault_late_night"] = math.Min(1.0, float64(lateNight)/3.0)
	f["vault_early_morning"] = math.Min(1.0, float64(earlyMorning)/3.0)

	// Self-awareness (first person + reflection language)
	reflectWords := []string{"realize", "realized", "aware", "awareness", "reflect", "reflecting",
		"honest", "honestly", "admit", "admitting", "acknowledge", "truth", "truthfully",
		"i need to", "i should", "i must", "i have been", "i know"}
	reflectCount := 0
	for _, rw := range reflectWords {
		reflectCount += strings.Count(lower, rw)
	}
	f["vault_self_awareness"] = math.Min(1.0, float64(reflectCount)/(wordCount*0.01+1))

	return f, themes
}

// AggregateVaultInsights combines features from many documents into a VaultInsight.
func AggregateVaultInsights(allFeatures []map[string]float64, allThemes [][]string, totalWords int, docCount int, dateRange string) VaultInsight {
	if len(allFeatures) == 0 {
		return VaultInsight{}
	}

	// Average each feature across all documents
	avg := make(map[string]float64)
	for _, f := range allFeatures {
		for k, v := range f {
			avg[k] += v
		}
	}
	n := float64(len(allFeatures))
	for k := range avg {
		avg[k] /= n
	}

	// Count theme frequencies
	themeCounts := make(map[string]int)
	for _, themes := range allThemes {
		for _, t := range themes {
			themeCounts[t]++
		}
	}

	// Sort themes by frequency
	type themeFreq struct {
		theme string
		count int
	}
	var sorted []themeFreq
	for t, c := range themeCounts {
		sorted = append(sorted, themeFreq{t, c})
	}
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].count > sorted[i].count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	var topThemes []string
	for i, tf := range sorted {
		if i >= 8 {
			break
		}
		topThemes = append(topThemes, tf.theme)
	}

	// Detect stall triggers from content patterns
	var stallTriggers []string
	if avg["vault_distraction"] > 0.3 {
		stallTriggers = append(stallTriggers, "social media / tool hopping")
	}
	if avg["vault_substance"] > 0.1 {
		stallTriggers = append(stallTriggers, "substance use affecting focus")
	}
	if avg["vault_plan_vs_ship"] > 0.7 {
		stallTriggers = append(stallTriggers, "over-planning without shipping")
	}
	if avg["vault_stall"] > 0.2 {
		stallTriggers = append(stallTriggers, "explicit stall/procrastination patterns")
	}
	if avg["financial_pressure"] > 0.3 {
		stallTriggers = append(stallTriggers, "financial pressure causing paralysis")
	}

	// Detect time patterns
	var timePatterns []string
	if avg["vault_late_night"] > 0.3 {
		timePatterns = append(timePatterns, "frequently works late night (10pm-3am)")
	}
	if avg["vault_early_morning"] > 0.2 {
		timePatterns = append(timePatterns, "sometimes works early morning (5-8am)")
	}

	// Extract top values
	var topValues []string
	if avg["vault_faith"] > 0.15 {
		topValues = append(topValues, "faith/spirituality")
	}
	if avg["vault_purpose"] > 0.15 {
		topValues = append(topValues, "purpose/impact")
	}
	if avg["vault_creative"] > 0.15 {
		topValues = append(topValues, "creative expression")
	}
	if avg["vault_growth"] > 0.15 {
		topValues = append(topValues, "personal growth")
	}
	if avg["vault_relationship"] > 0.15 {
		topValues = append(topValues, "community/relationships")
	}
	if avg["vault_health"] > 0.1 {
		topValues = append(topValues, "health/body")
	}

	vi := VaultInsight{
		FaithStrength:       avg["vault_faith"],
		EmotionalVolatility: avg["vault_struggle"] * 0.6 + avg["emotion_expression"] * 0.4,
		FinancialStress:     avg["financial_pressure"],
		SocialIsolation:     math.Max(0, 0.5-avg["vault_relationship"]),
		HealthAwareness:     avg["vault_health"],
		CreativeEnergy:      avg["vault_creative"],
		DistractionRisk:     avg["vault_distraction"],
		PurposeClarity:      avg["vault_purpose"],
		ProductivityStyle:   1.0 - avg["vault_plan_vs_ship"], // invert: 0=planner, 1=doer
		ProjectSprawl:       math.Min(1.0, avg["vault_creative"]*0.5+avg["vault_stall"]*0.5),

		LateNightWorker:   avg["vault_late_night"] > 0.3,
		MorningPerson:     avg["vault_early_morning"] > avg["vault_late_night"],
		JournalingHabit:   math.Min(1.0, float64(docCount)/50.0), // normalized to 50 entries
		SelfAwareness:     avg["vault_self_awareness"],
		GrowthMindset:     avg["vault_growth"],
		PerfectionismRisk: avg["vault_plan_vs_ship"],

		TopValues:       topValues,
		RecurringThemes: topThemes,
		StallTriggers:   stallTriggers,
		TimePatterns:    timePatterns,
		DocumentCount:   docCount,
		WordCount:       totalWords,
		DateRange:       dateRange,
	}

	return vi
}

// ExtractDatesFromFilename parses dates from vault document filenames.
func ExtractDatesFromFilename(filename string) *time.Time {
	matches := datePattern.FindStringSubmatch(filename)
	if len(matches) >= 4 {
		t, err := time.Parse("2006-01-02", matches[0])
		if err == nil {
			return &t
		}
	}
	return nil
}

// countWordMatchesStatic is a static version (no NLPExtractor receiver needed)
func countWordMatchesStatic(words []string, set map[string]bool) int {
	count := 0
	for _, w := range words {
		if set[w] {
			count++
		}
	}
	return count
}
