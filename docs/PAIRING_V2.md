# Wirebot Pairing Protocol v2 — Deep Calibration Engine

> Not a signup form. Not a chatbot questionnaire. A **psychometric-grade calibration system**
> that builds a behavioral model of the founder — how they think, act, communicate, decide,
> and where they need amplification vs. gap-filling.

---

## Design Principles

1. **Mixed-modal input** — Tap cards, sliders, quick-pick, ranking, free-text, voice (future). Never just "question → textbox."
2. **Psychometric rigor** — Real frameworks (Big Five, DISC, Kolbe-inspired, Working Genius) adapted for business context. Not horoscope. Science.
3. **Progressive disclosure** — Start easy (taps), go deep (conversation). The founder doesn't feel interrogated — they feel *understood*.
4. **Inference > Declaration** — What the founder *does* matters more than what they *say*. Communication scanning, behavioral patterns, and usage data outweigh self-reported answers.
5. **Living calibration** — Pairing is never "done." The model refines continuously. Day 1 score is provisional. Day 90 score is real.
6. **Embeddable UI** — Works in wins.wirebot.chat, Chrome extension, Connect overlay, and any white-label surface. Same component, different skin.

---

## The Founder Profile Model

Pairing builds a **Founder Profile** — a multi-dimensional model that Wirebot uses for every interaction.

### Dimension 1: Action Style (Kolbe-Inspired)

How the founder naturally takes action under pressure. Not personality — **conation** (instinct).

| Mode | Low (1-3) | Mid (4-6) | High (7-10) |
|------|-----------|-----------|-------------|
| **Fact Finder** | Acts on gut, minimal research | Balanced research | Deep researcher, needs data before moving |
| **Follow Through** | Improviser, hates rigid process | Flexible systems | Systematic, needs structure and checklists |
| **Quick Start** | Cautious, plans extensively | Balanced | Risk-taker, starts before planning, iterates fast |
| **Implementor** | Abstract thinker, delegates physical work | Balanced | Hands-on builder, prototypes, tangible output |

**Why this matters:** If the founder is a high Quick Start / low Follow Through, Wirebot needs to be the structure. If they're high Fact Finder, Wirebot surfaces data before recommending action. Complementary, not matching.

**Assessment method:** Forced-choice pairs (tap A or B):
```
When starting something new, do you prefer to:
  [A] Jump in and figure it out    [B] Research first, then start

When a plan falls apart, do you:
  [A] Improvise on the spot        [B] Regroup and make a new plan

When building something, do you prefer:
  [A] Drawing it out / designing   [B] Just start building it
```

12 forced-choice pairs → 4 scores (1-10 each). Takes ~2 minutes.

---

### Dimension 2: Communication DNA (DISC-Adapted)

How the founder communicates, processes information, and makes decisions.

| Style | Characteristics | Wirebot Adapts |
|-------|-----------------|----------------|
| **D — Driver** | Direct, results-oriented, impatient with detail | Give bottom-line first, then supporting data only if asked |
| **I — Influencer** | Enthusiastic, big-picture, relationship-focused | Lead with vision, celebrate momentum, keep energy up |
| **S — Steady** | Thoughtful, process-oriented, dislikes sudden change | Explain reasoning, give notice before pivots, be patient |
| **C — Analytical** | Precise, data-driven, skeptical of hype | Show evidence, provide options with tradeoffs, be specific |

**Assessment method:** Scenario cards (tap which resonates):
```
Your product just got featured on a major blog. What's your first thought?

  [🚀] "Let's capitalize! What's the next move?"          → D
  [🎉] "This is amazing! Let me share this everywhere!"   → I
  [🤔] "OK, let me make sure we can handle the traffic"   → S
  [📊] "How many actual signups did this generate?"        → C
```

8 scenario cards → DISC profile with percentages. Takes ~90 seconds.

**Critical: This is PROVISIONAL.** The real DISC profile comes from scanning actual communications (Phase 6).

---

### Dimension 3: Working Genius (Lencioni-Adapted)

What gives the founder energy vs. what drains them in business work.

| Genius | Description | If Strong | If Weak (Gap) |
|--------|-------------|-----------|---------------|
| **Wonder** | Pondering, questioning, seeing what could be | Visionary thinker | Wirebot provides the "what if" prompts |
| **Invention** | Creating solutions to problems | Natural innovator | Wirebot surfaces proven solutions |
| **Discernment** | Evaluating ideas, gut instinct for quality | Good taste, knows what works | Wirebot provides decision frameworks |
| **Galvanizing** | Rallying people, creating momentum | Natural leader/seller | Wirebot drafts outreach, creates urgency |
| **Enablement** | Supporting others, making things happen | Great executor for others | Wirebot handles coordination/admin |
| **Tenacity** | Pushing through to completion | Finisher, ships | Wirebot adds accountability, deadline pressure |

**Assessment method:** Energy ranking (drag to sort):
```
Rank these from "gives me energy" to "drains me":

  🔮 Brainstorming new ideas
  🛠️ Building the actual product
  🎯 Deciding which idea is best
  📣 Convincing people to buy/join
  🤝 Helping team members succeed
  🏁 Grinding through the last 20% to ship
```

One drag-to-sort interaction → 6 scores. Takes ~30 seconds.

**Why this matters:** Wirebot becomes the **complement**. Founder weak at Tenacity? Wirebot becomes the relentless accountability engine. Founder weak at Galvanizing? Wirebot drafts the pitch emails and sales pages.

---

### Dimension 4: Risk & Decision Profile

How the founder handles uncertainty, loss, and irreversible choices.

**Assessment method:** Slider scales (0-100):
```
━━━━━━━━━━●━━━━━━━ 70%
"I'd rather move fast and fix mistakes than move slow and avoid them"

━━━━●━━━━━━━━━━━━━ 30%
"I'm comfortable making decisions with incomplete information"

━━━━━━━━━━━━●━━━━━ 80%
"When I commit to something, I find it very hard to quit even when I should"

━━━━━━━●━━━━━━━━━━ 50%
"I think about worst-case scenarios before acting"
```

6 sliders → Risk tolerance, decision speed, sunk-cost sensitivity, loss aversion. Takes ~45 seconds.

---

### Dimension 5: Business Reality Scan

The operator's actual situation — not from questions but from **evidence**.

**Assessment method:** Connected accounts + direct questions (mixed):

**Quick taps:**
```
What stage is your main business?

  [💡 Idea]  [🔨 Building]  [🚀 Launched]  [📈 Growing]  [🔥 Scaling]

How many businesses/projects are you actively running?

  [1]  [2-3]  [4-5]  [6+]

Revenue situation right now:

  [❌ $0]  [🌱 <$1K/mo]  [💰 $1-5K/mo]  [🚀 $5-20K/mo]  [🏦 $20K+/mo]

Debt situation:

  [✅ None]  [📋 Manageable]  [⚠️ Significant]  [🔴 Critical]
```

**Deep conversation (after taps):**
```
Q: "Tell me about ALL the businesses and projects. Don't filter — the messy truth."
Q: "Which one pays the bills right now? Even if it's not the one you love."
Q: "What have you started but not finished? How far along is each?"
Q: "What's the one thing that if it took off, everything else follows?"
```

**Auto-calibration from connected accounts (Phase 6):**
- Stripe → Real revenue, not self-reported
- GitHub → Real shipping velocity
- Calendar → Real time allocation
- Bank (Plaid, future) → Real burn rate

---

### Dimension 6: Communication Style Inference (The Scanner)

> **Pairing is NOT complete without this.**
> Self-reported communication style is unreliable. Wirebot must OBSERVE.

**What gets scanned:**

| Source | What's Extracted |
|--------|-----------------|
| **Chat history** (Wirebot conversations) | Sentence length, vocabulary complexity, emoji usage, question vs. statement ratio, response latency, topic switching frequency |
| **Email** (future: IMAP/Gmail) | Formality level, average email length, response time patterns, sign-off style, thread depth |
| **Git commits** (already connected) | Commit message style, frequency patterns, burst vs. steady cadence |
| **Sendy campaigns** (already connected) | Marketing voice, subject line patterns, CTA style |
| **Blog posts** (already connected via RSS) | Writing style, topic patterns, publishing cadence |
| **Discord/Slack** (future) | Casual vs. professional register, emoji patterns, reaction patterns |

**Inference algorithms:**

**1. Linguistic Style Analysis**
```
Metrics extracted from text:
- Average sentence length (short=direct, long=analytical)
- Hedging language frequency ("maybe", "perhaps", "I think") → confidence level
- Action verb density ("build", "ship", "launch" vs. "consider", "explore", "plan")
- Question ratio (high = collaborative/uncertain, low = directive)
- Exclamation frequency (high = enthusiastic/I-style, low = reserved/C-style)
- First-person vs. second-person ratio (self-focused vs. other-focused)
- Temporal language ("right now", "today" vs. "eventually", "someday") → urgency orientation
```

**2. Behavioral Pattern Detection**
```
From usage data:
- Time-of-day activity → chronotype (early bird, night owl, erratic)
- Burst vs. steady patterns → Quick Start score validation
- Task completion rate → Tenacity score validation
- Context switch frequency → Focus capacity measurement
- Response latency to Wirebot → engagement/priority signal
```

**3. Communication Style Synthesis**
```
All signals feed into a Communication DNA profile:
{
  "directness": 0.78,        // 0=indirect, 1=blunt
  "formality": 0.35,         // 0=casual, 1=formal
  "detail_preference": 0.62, // 0=big-picture, 1=granular
  "emotion_expression": 0.71,// 0=reserved, 1=expressive
  "pace_preference": 0.85,   // 0=methodical, 1=fast
  "decision_style": 0.60,    // 0=consensus, 1=unilateral
  "confidence": {
    "self_reported": 0.4,    // From DISC assessment
    "observed": 0.72,        // From actual communication
    "delta": 0.32            // Gap = self-awareness insight
  }
}
```

**The delta between self-reported and observed is itself a signal.** If someone says they're analytical but writes with high emotion and short bursts, Wirebot knows they *aspire* to be analytical but *operate* as an Influencer. Wirebot serves who they ARE, not who they think they are.

**Minimum scan threshold:** Pairing cannot reach "Bonded" (81+) without at least 50 messages analyzed + 7 days of behavioral data.

---

## Pairing Score v2

### Weighted Components

| Component | Weight | Source | Can Max Without Conversation? |
|-----------|--------|--------|-------------------------------|
| Action Style (Kolbe) | 15% | 12 forced-choice pairs | No |
| Communication DNA (DISC) | 10% | 8 scenario cards | No |
| Working Genius | 10% | 1 drag-to-sort | No |
| Risk & Decision Profile | 10% | 6 sliders | No |
| Business Reality (declared) | 15% | Quick taps + conversation | No |
| Business Reality (verified) | 10% | Connected accounts | Yes (auto-scans) |
| Communication Style (inferred) | 15% | Scanner (50+ messages) | Yes (passive) |
| Behavioral Patterns | 10% | 7+ days usage data | Yes (passive) |
| Continuous Inference | 5% | Ongoing fact extraction | Yes (passive) |

### Score Thresholds

| Score | Level | What Unlocks |
|-------|-------|-------------|
| 0-15 | **Stranger** | Generic responses. Heavy nudges. UI shows pairing card. |
| 16-35 | **Acquaintance** | Basic personalization. Knows name, stage, tone preference. |
| 36-60 | **Partner** | Solid model. Personalized recommendations. Accountability active. Complementary gap-filling starts. |
| 61-80 | **Trusted** | Deep model. Proactive suggestions. Pattern recognition. Communication style matched. Auto-sequencing. |
| 81-100 | **Bonded** | Full sovereign mode. Anticipates needs. Acts autonomously within trust bounds. Communication scanner validated. Founder Profile stable. |

**Key rule:** Score cannot exceed 60 without communication scanning data. Cannot exceed 80 without 30+ days of behavioral data. Self-report alone is never enough.

---

## UI Component: `<PairingFlow />`

### Design Language

- **Dark theme** (matches scoreboard)
- **Card-based** — one interaction per card, swipe or tap to advance
- **Progress ring** — circular progress indicator, not a linear bar
- **Micro-animations** — cards slide, scores animate up, checkmarks pop
- **Haptic feedback** (mobile) — subtle vibration on selection
- **Ambient score** — pairing score visible and updating in real-time as you answer

### Card Types

**1. Forced Choice Card**
```
┌─────────────────────────────────┐
│                                 │
│   When starting something new   │
│   I usually...                  │
│                                 │
│  ┌─────────────┐ ┌────────────┐ │
│  │   🚀        │ │   📋       │ │
│  │  Jump in    │ │ Research   │ │
│  │  and figure │ │ first,     │ │
│  │  it out     │ │ then start │ │
│  └─────────────┘ └────────────┘ │
│                                 │
│         ○ ○ ● ○ ○ ○            │
└─────────────────────────────────┘
```

**2. Scenario Card (DISC)**
```
┌─────────────────────────────────┐
│                                 │
│  Your biggest client just       │
│  asked for something you've     │
│  never done before.             │
│                                 │
│  ┌─────────────────────────────┐│
│  │ 🚀 "Yes! I'll figure it    ││
│  │     out as I go"            ││
│  └─────────────────────────────┘│
│  ┌─────────────────────────────┐│
│  │ 🎯 "Let me scope this and  ││
│  │     give you a real answer" ││
│  └─────────────────────────────┘│
│  ┌─────────────────────────────┐│
│  │ 🤝 "Let me find someone    ││
│  │     who's done this before" ││
│  └─────────────────────────────┘│
│  ┌─────────────────────────────┐│
│  │ 📊 "I need to analyze the  ││
│  │     ROI before committing"  ││
│  └─────────────────────────────┘│
│                                 │
└─────────────────────────────────┘
```

**3. Slider Card**
```
┌─────────────────────────────────┐
│                                 │
│  I'd rather move fast and fix   │
│  mistakes than move slow and    │
│  avoid them                     │
│                                 │
│  Disagree ━━━━━━━━●━━━ Agree   │
│                   72%           │
│                                 │
│         ○ ○ ○ ● ○ ○            │
└─────────────────────────────────┘
```

**4. Drag-to-Sort Card (Working Genius)**
```
┌─────────────────────────────────┐
│                                 │
│  Drag to rank: what gives you   │
│  ENERGY vs. what DRAINS you     │
│                                 │
│  ⚡ Energizes                   │
│  ┌─────────────────────────────┐│
│  │ 🔮 Brainstorming new ideas ││
│  │ 🛠️ Building the product     ││
│  │ 🏁 Grinding to ship         ││
│  └─────────────────────────────┘│
│  😴 Drains                      │
│  ┌─────────────────────────────┐│
│  │ 📣 Convincing people        ││
│  │ 🤝 Supporting team          ││
│  │ 🎯 Deciding which idea      ││
│  └─────────────────────────────┘│
│                                 │
└─────────────────────────────────┘
```

**5. Quick Tap Grid**
```
┌─────────────────────────────────┐
│                                 │
│  What stage is your main        │
│  business?                      │
│                                 │
│  ┌──────┐ ┌──────┐ ┌──────┐   │
│  │  💡  │ │  🔨  │ │  🚀  │   │
│  │ Idea │ │Build │ │Launch│   │
│  └──────┘ └──────┘ └──────┘   │
│  ┌──────┐ ┌──────┐             │
│  │  📈  │ │  🔥  │             │
│  │ Grow │ │Scale │             │
│  └──────┘ └──────┘             │
│                                 │
└─────────────────────────────────┘
```

**6. Conversation Card (deep questions)**
```
┌─────────────────────────────────┐
│                                 │
│  💬 Let's go deeper             │
│                                 │
│  Tell me about ALL the          │
│  businesses and projects.       │
│  Don't filter — the messy       │
│  truth is what I need.          │
│                                 │
│  ┌─────────────────────────────┐│
│  │                             ││
│  │  (expandable text area)     ││
│  │                             ││
│  └─────────────────────────────┘│
│             [Continue →]        │
│                                 │
│  💡 Or tap to talk:             │
│     [🎤 Voice note]            │
│                                 │
└─────────────────────────────────┘
```

### Flow Architecture

```
Phase 1: Quick Calibration (~4 min)
  ├── Welcome card (animated, sets tone)
  ├── Name + timezone (2 taps + picker)
  ├── Action Style: 12 forced-choice pairs (~2 min)
  ├── Working Genius: 1 drag-to-sort (~30 sec)
  ├── Risk Profile: 6 sliders (~45 sec)
  └── 🎯 Score: ~25/100

Phase 2: Communication & Style (~2 min)
  ├── DISC scenarios: 8 scenario cards (~90 sec)
  ├── Accountability preference (3-way tap)
  ├── Advice style preference (3-way tap)
  ├── Check-in frequency (4-way tap)
  └── 🎯 Score: ~45/100

Phase 3: Business Reality (~5 min)
  ├── Stage tap (5-way)
  ├── Business count tap (4-way)
  ├── Revenue tap (5-way)
  ├── Debt tap (4-way)
  ├── Conversation: "Tell me about everything you're working on"
  ├── Conversation: "Which one pays the bills?"
  ├── Conversation: "What's 80% done that could ship?"
  ├── Conversation: "What's the one thing — if it worked, everything follows?"
  └── 🎯 Score: ~60/100

Phase 4: Passive Calibration (ongoing, no user effort)
  ├── Communication scanner (50+ messages) → +15%
  ├── Behavioral patterns (7+ days) → +10%
  ├── Connected accounts verification → +10%
  └── Continuous inference → +5%
  └── 🎯 Score: up to 100/100

Phase 5: Companion Lock
  └── Triggered when score crosses 80
  └── Ceremony screen + mode transition
```

### Embeddability

The `<PairingFlow />` component is self-contained:

```html
<!-- In scoreboard PWA -->
<PairingFlow
  apiUrl="https://wins.wirebot.chat"
  token="{jwt}"
  theme="dark"
  onComplete={(profile) => ...}
/>

<!-- In Chrome extension -->
<PairingFlow
  apiUrl="https://wins.wirebot.chat"
  token="{jwt}"
  theme="extension"
  compact={true}
/>

<!-- In white-label app -->
<PairingFlow
  apiUrl="https://client.wirebot.chat"
  token="{jwt}"
  theme={clientTheme}
  branding={clientBranding}
/>
```

---

## Scientific Algorithms

### 1. Kolbe-Style Action Mode Scoring

Each forced-choice maps to one or two modes. Scoring uses **ipsative measurement** (forced ranking, not absolute):

```
score[mode] = (selections_for_mode / total_pairs_involving_mode) * 10

// Normalized so modes sum to a constant (prevents all-high gaming)
// Result: 4 scores, each 1-10, sum = ~20
```

**Validation:** Cross-reference with behavioral data after 7 days. If self-reported Quick Start = 9 but actual shipping velocity is low, adjust the effective score (not the declared score — the delta is informative).

### 2. DISC Composite Scoring

Each scenario has 4 responses mapped to D/I/S/C. Selection adds weight:

```
disc[style] = Σ(weight_per_scenario) / max_possible

// Primary style: highest score
// Secondary style: second highest
// Stress style: lowest score (what they avoid under pressure)
```

**Output:** `{ D: 0.72, I: 0.45, S: 0.28, C: 0.55 }` → Primary: D, Secondary: C

### 3. Working Genius Energy Map

Rank position → score:
```
Position 1 (top) → 10 points (Working Genius)
Position 2 → 8 points (Working Genius)
Position 3 → 6 points (Working Competency)
Position 4 → 4 points (Working Competency)
Position 5 → 2 points (Working Frustration)
Position 6 → 0 points (Working Frustration)
```

**Wirebot complement rule:**
```
For each Frustration (score ≤ 2):
  → Wirebot amplifies this capability
  → E.g., Founder frustration = Tenacity → Wirebot becomes the relentless closer

For each Genius (score ≥ 8):
  → Wirebot supports and feeds this
  → E.g., Founder genius = Invention → Wirebot surfaces problems worth solving
```

### 4. Communication Style Inference (NLP)

**Text metrics extracted per message:**
```python
{
  "avg_sentence_length": 12.4,
  "vocabulary_richness": 0.68,      # unique_words / total_words
  "hedging_ratio": 0.12,            # hedging_phrases / total_sentences
  "action_verb_density": 0.34,      # action_verbs / total_verbs
  "question_ratio": 0.15,           # questions / total_sentences
  "exclamation_ratio": 0.08,        # exclamations / total_sentences
  "first_person_ratio": 0.22,       # "I/me/my" / total_words
  "emoji_frequency": 0.03,          # emojis / total_words
  "avg_response_time_seconds": 45,  # time between receive and reply
  "temporal_urgency": 0.71,         # urgent_words / temporal_words
}
```

**Mapping to profile dimensions:**
```
directness = f(avg_sentence_length⁻¹, hedging_ratio⁻¹, action_verb_density)
formality = f(vocabulary_richness, emoji_frequency⁻¹, exclamation_ratio⁻¹)
detail_preference = f(avg_sentence_length, vocabulary_richness, question_ratio)
emotion_expression = f(exclamation_ratio, emoji_frequency, hedging_ratio⁻¹)
pace_preference = f(avg_response_time⁻¹, temporal_urgency, action_verb_density)
```

Each `f()` is a weighted linear combination, calibrated against the DISC self-report as a soft prior. The weights shift as more data accumulates (Bayesian updating).

### 5. Behavioral Pattern Detection

**From scoreboard events:**
```
shipping_consistency = stddev(daily_ship_count) over 7 days
  → Low stddev = Steady (Follow Through validation)
  → High stddev = Burst (Quick Start validation)

context_switch_rate = unique_projects_per_day / total_events
  → High = scattered (flag for Pillar 6 sequencing)
  → Low = focused (support this)

time_of_day_distribution = histogram(event_hours, bins=24)
  → Bimodal = split schedule (morning + evening)
  → Single peak = clear productive window
  → Flat = no rhythm (Wirebot should help establish one)

completion_ratio = tasks_completed / tasks_created over 14 days
  → > 0.8 = strong finisher
  → < 0.4 = starting > finishing (Tenacity gap)
```

### 6. Founder Profile Synthesis

All dimensions merge into a single **Founder Profile** stored as JSON:

```json
{
  "version": 2,
  "pairing_score": 67,
  "level": "Trusted",
  "last_updated": "2026-02-15T08:00:00Z",

  "action_style": {
    "fact_finder": 4,
    "follow_through": 3,
    "quick_start": 8,
    "implementor": 6,
    "source": "assessment",
    "validated": true,
    "behavioral_delta": { "quick_start": -1.2 }
  },

  "communication_dna": {
    "primary": "D",
    "secondary": "C",
    "scores": { "D": 0.72, "I": 0.45, "S": 0.28, "C": 0.55 },
    "source": "assessment+inference",
    "observed": {
      "directness": 0.78,
      "formality": 0.35,
      "detail_preference": 0.62,
      "emotion_expression": 0.71,
      "pace_preference": 0.85,
      "messages_analyzed": 127,
      "confidence": 0.82
    }
  },

  "working_genius": {
    "genius": ["invention", "quick_start_analog"],
    "competency": ["discernment", "wonder"],
    "frustration": ["enablement", "tenacity"],
    "wirebot_complements": ["tenacity", "enablement"]
  },

  "risk_profile": {
    "risk_tolerance": 0.70,
    "decision_speed": 0.82,
    "sunk_cost_sensitivity": 0.80,
    "loss_aversion": 0.30,
    "worst_case_thinking": 0.50,
    "incomplete_info_comfort": 0.65
  },

  "business_reality": {
    "stage": "launched",
    "business_count": 4,
    "revenue_bracket": "$1-5K/mo",
    "debt_level": "significant",
    "verified_revenue": 678.00,
    "verified_mrr": null,
    "stripe_connected": true,
    "businesses": [
      { "name": "Startempire Wire", "stage": "launched", "revenue": true, "priority": 1 },
      { "name": "Wirebot", "stage": "building", "revenue": false, "priority": 2 },
      { "name": "Philoveracity", "stage": "launched", "revenue": true, "priority": 3 },
      { "name": "SEW Network", "stage": "building", "revenue": false, "priority": 4 }
    ]
  },

  "behavioral_patterns": {
    "chronotype": "night_owl",
    "peak_hours": [22, 23, 0, 1, 2],
    "shipping_style": "burst",
    "focus_capacity": 0.45,
    "completion_ratio": 0.62,
    "days_observed": 14,
    "confidence": 0.71
  },

  "wirebot_calibration": {
    "tone": "direct_diplomatic",
    "advice_style": "tell_me_what_to_do",
    "check_in_frequency": "daily",
    "accountability_level": "direct_push",
    "complement_focus": ["tenacity", "enablement", "follow_through"],
    "amplify_focus": ["invention", "quick_start", "risk_taking"]
  }
}
```

---

## How Wirebot Uses the Profile

### Complement Mode (Gap-Filling)

For each Working Frustration:
```
tenacity (frustration) →
  Wirebot: More aggressive follow-ups, deadline pressure,
  "You started X 3 days ago — what's blocking the finish?"
  Accountability cron runs tighter cycles.

enablement (frustration) →
  Wirebot: Handles coordination, admin prep, documentation.
  Auto-drafts reports, summaries, handoff docs.
  "I've prepared the brief for your call with [contact]."

follow_through (low action style) →
  Wirebot: Provides the structure the founder won't create.
  Auto-generates checklists, sequences, dependencies.
  "Here's the 6-step plan. Step 1 is ready to execute."
```

### Amplification Mode (Strength-Feeding)

For each Working Genius:
```
invention (genius) →
  Wirebot: Surfaces problems worth solving, market gaps, customer pain.
  "3 members complained about X this week. Could be a product."
  Connects invention genius to revenue pipeline.

quick_start (high action style) →
  Wirebot: Matches pace. Doesn't slow down with excessive planning.
  Provides just-enough structure, then "Go."
  "Here's the minimum viable plan. Ship today?"
```

### Communication Adaptation

```
if profile.communication_dna.primary == "D":
  → Lead with the recommendation, not the reasoning
  → "Ship the extension today. Here's why it matters: [1 line]"
  → Keep messages under 3 sentences unless asked for more

if profile.communication_dna.primary == "C":
  → Lead with data, then recommendation
  → "Revenue is $678/mo, 3 products at >80% done. Shipping Product A
     has the highest ROI because [evidence]. Recommendation: ship by Friday."
  → Include numbers, comparisons, tradeoffs

if profile.behavioral_patterns.chronotype == "night_owl":
  → Shift standup from 8 AM to 11 AM
  → Heaviest nudges during peak hours (10 PM - 2 AM)
  → "It's 11 PM and you're in the zone — here's what's next"
```

### Proactive Multiplier

Wirebot doesn't wait to be asked. Based on the profile:

```
Daily:
  - Morning intent suggestion based on yesterday's momentum + profile strengths
  - "You're strongest at invention and Quick Start — today focus on
     the creative work. I'll handle the follow-through items."

Weekly:
  - Gap analysis: "This week you shipped 12 things (Quick Start ✓)
    but completed 0 follow-ups (Follow Through gap). Next week I'm
    scheduling 3 completion blocks."

Triggered:
  - Stall detected + Tenacity frustration → More aggressive nudge
  - Revenue drop + Driver DISC → Bottom-line alert, action-oriented
  - Context switching spike + low Follow Through → "You've touched 6
    projects today. Pick ONE for the next 2 hours. I'll hold the rest."
```

---

## API Endpoints

```
POST   /v1/pairing/answers     — Submit assessment answers (batch)
GET    /v1/pairing/status       — Current score + profile summary
GET    /v1/pairing/profile      — Full Founder Profile JSON
POST   /v1/pairing/scan         — Trigger communication scan
GET    /v1/pairing/insights     — Latest inferences + deltas
PATCH  /v1/pairing/profile      — Manual profile corrections
DELETE /v1/pairing/reset        — Full reset (requires confirmation)
```

---

## Implementation Order

1. **Founder Profile schema + storage** (pairing.json v2)
2. **Assessment cards UI** (Svelte component, embeddable)
3. **Scoring algorithms** (Go server endpoints)
4. **Communication scanner** (analyze chat history in Go)
5. **Behavioral pattern detector** (analyze scoreboard events)
6. **Wirebot calibration engine** (apply profile to chat context)
7. **Continuous inference loop** (background, always running)
8. **White-label theming API**

---

## See Also

- [PAIRING.md](./PAIRING.md) — Original v1 protocol (22 questions)
- [SOUL.md](/home/wirebot/clawd/SOUL.md) — 12 Pillars that shape Wirebot's behavior
- [OPERATOR_REALITY.md](/home/wirebot/clawd/OPERATOR_REALITY.md) — Current state of the operator
- [SCOREBOARD_PRODUCT.md](./SCOREBOARD_PRODUCT.md) — Scoring system that feeds behavioral data

---

*Pairing v2 is not a feature. It's the foundation of everything Wirebot does.
Without deep calibration, Wirebot is a chatbot. With it, Wirebot is a co-founder.*
