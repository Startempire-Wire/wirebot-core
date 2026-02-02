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

## The Profile Equalizer — Transparent Calibration UI

### Philosophy

No black box. The founder sees **exactly** how Wirebot sees them — every parameter,
every score, every piece of evidence that caused it, and the formula behind it.
Like a music equalizer where every band is visible, adjustable, and explained.

**Trust requires transparency.** If Wirebot is going to act as a co-founder based on a
psychometric profile, the founder has the right to see every number, challenge any score,
and understand why Wirebot behaves the way it does.

---

### Layout: The Equalizer View

Accessible from scoreboard Settings → "Profile" or dedicated route `/profile`.

```
┌──────────────────────────────────────────────────────────────┐
│  ⚡ FOUNDER PROFILE                    Score: 67 (Trusted)   │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ 67% ━━━━━                │
│  Accuracy: 87% ↑  |  Signals: 1,247  |  Days: 42            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─── ACTION STYLE ──────────────────────────────────────┐   │
│  │                                                        │   │
│  │  Fact Finder    ━━━●━━━━━━━━━━━  4/10   ⓘ            │   │
│  │  Follow Through ━━●━━━━━━━━━━━━  3/10   ⓘ  ⚠ drift   │   │
│  │  Quick Start    ━━━━━━━━━●━━━━━  8/10   ⓘ            │   │
│  │  Implementor    ━━━━━━●━━━━━━━━  6/10   ⓘ            │   │
│  │                                                        │   │
│  │  🔬 Sources: Assessment (35%) + Behavioral (45%)       │   │
│  │            + Chat inference (20%)                      │   │
│  │  📅 Last updated: 2h ago  |  Confidence: 88%          │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── COMMUNICATION DNA ─────────────────────────────────┐   │
│  │                                                        │   │
│  │  D (Driver)     ━━━━━━━━━●━━━━  72%     PRIMARY       │   │
│  │  I (Influencer) ━━━━━●━━━━━━━━  45%                   │   │
│  │  S (Steady)     ━━━●━━━━━━━━━━  28%                   │   │
│  │  C (Analytical)  ━━━━━━●━━━━━━  55%     secondary     │   │
│  │                                                        │   │
│  │  Observed style:                                       │   │
│  │  Directness     ━━━━━━━━●━━━━━  78%                   │   │
│  │  Formality      ━━━●━━━━━━━━━━  35%                   │   │
│  │  Emotion        ━━━━━━━━●━━━━━  71%                   │   │
│  │  Pace           ━━━━━━━━━━●━━━  85%                   │   │
│  │                                                        │   │
│  │  📨 312 messages analyzed  |  Confidence: 94%          │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── ENERGY MAP ────────────────────────────────────────┐   │
│  │                                                        │   │
│  │  ⚡ GENIUS      🔮 Wonder (8)  |  🛠️ Invention (10)   │   │
│  │  ✅ COMPETENT   🎯 Discern (6) |  📣 Galvanize (4)    │   │
│  │  😤 FRUSTRATION 🤝 Enable (2)  |  🏁 Tenacity (0)     │   │
│  │                                                        │   │
│  │  Wirebot compensates: Tenacity 33% | Enable 22%       │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── RISK PROFILE ─────────────────────────────────────┐    │
│  │  Risk tolerance   ━━━━━━━━●━━━  70%   ⓘ             │    │
│  │  Decision speed   ━━━━━━━━━●━━  82%   ⓘ             │    │
│  │  Ambiguity OK     ━━━━━━━●━━━━  65%   ⓘ             │    │
│  │  Sunk-cost trap   ━━━━━━━━━●━━  80%   ⚠ high        │    │
│  │  Loss aversion    ━━━●━━━━━━━━  30%   ⓘ             │    │
│  │  Bias to action   ━━━━━━━━━●━━  78%   ⓘ             │    │
│  └───────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─── CONTEXT WINDOWS (active) ──────────────────────────┐   │
│  │  🟢 SHIPPING_SPRINT   strength: 0.82  |  3 days       │   │
│  │  🟡 FINANCIAL_PRESSURE strength: 0.45  |  fading       │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── TRAIT vs STATE ────────────────────────────────────┐   │
│  │  Quick Start:  trait=8  state=9  drift=+1.0  α=0.55   │   │
│  │  Follow Thru:  trait=3  state=5  drift=+1.5  α=0.42   │   │
│  │  ⓘ State elevated → you're in execution mode           │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  [📊 Evidence Log]  [🧮 Formulas]  [📈 Accuracy]  [⚙️ Override] │
└──────────────────────────────────────────────────────────────┘
```

---

### Drill-Down: Tapping Any Parameter

Every bar, every number, every ⓘ icon is tappable. Tapping opens a **detail drawer**
that shows three layers:

**Layer 1: Human explanation**
**Layer 2: Evidence trail**
**Layer 3: The formula**

Example — tapping "Quick Start: 8/10":

```
┌──────────────────────────────────────────────────────────────┐
│  ← Quick Start                                    8/10       │
│                                                              │
│  ┌─── WHAT THIS MEANS ───────────────────────────────────┐   │
│  │  You naturally start things fast. You prefer action    │   │
│  │  over planning, learning by doing, and iterating       │   │
│  │  quickly. This is a STRENGTH when paired with          │   │
│  │  structure — which Wirebot provides.                   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── EVIDENCE (what caused this score) ─────────────────┐   │
│  │                                                        │   │
│  │  📋 Assessment (weight: 35%)              score: 9     │   │
│  │     Q1: "Jump in and figure it out" ✓                  │   │
│  │     Q2: "Keep options open" ✓                          │   │
│  │     Q8: "Skip unnecessary steps" ✓                     │   │
│  │     Consistency: 100% (3/3 items aligned)              │   │
│  │                                                        │   │
│  │  📊 Behavioral (weight: 45%)              score: 7.2   │   │
│  │     Shipping cadence CV: 2.1 (burst pattern)           │   │
│  │     New projects started/week: 3.2 (high)              │   │
│  │     Avg time from idea→first commit: 2.4h (fast)       │   │
│  │     Based on 847 events over 42 days                   │   │
│  │                                                        │   │
│  │  💬 Chat inference (weight: 20%)          score: 7.8   │   │
│  │     Action verb density: 0.34 (high)                   │   │
│  │     Urgency language: 0.71 (high)                      │   │
│  │     Hedging ratio: 0.08 (low — decisive)               │   │
│  │     Based on 312 messages                              │   │
│  │                                                        │   │
│  │  ⚖️ Self-perception delta: -1.8                        │   │
│  │     You rate yourself 9. Your behavior says 7.2.       │   │
│  │     This is normal — you START like a 9 but sustain    │   │
│  │     like a 7. Wirebot calibrates to the 7.2.           │   │
│  │                                                        │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── FORMULA ───────────────────────────────────────────┐   │
│  │                                                        │   │
│  │  effective = α × trait + (1-α) × state                 │   │
│  │           = 0.55 × 8.0 + 0.45 × 9.0                   │   │
│  │           = 4.4 + 4.05 = 8.45 → rounded: 8            │   │
│  │                                                        │   │
│  │  where:                                                │   │
│  │    trait = Σ(wᵢ × scoreᵢ)                             │   │
│  │         = 0.35×9 + 0.45×7.2 + 0.20×7.8                │   │
│  │         = 3.15 + 3.24 + 1.56 = 7.95 → 8.0             │   │
│  │                                                        │   │
│  │    state = recent EMA (λ=0.15, last 10 signals)        │   │
│  │         = 9.0 (elevated — shipping sprint active)      │   │
│  │                                                        │   │
│  │    α = 0.30 + 0.40 × stability                        │   │
│  │      = 0.30 + 0.40 × (1/(1+1.0))                      │   │
│  │      = 0.30 + 0.20 = 0.50 → 0.55 (smoothed)           │   │
│  │                                                        │   │
│  │    drift = |state - trait| / σ_trait                    │   │
│  │         = |9.0 - 8.0| / 1.0 = 1.0 (mild elevation)    │   │
│  │                                                        │   │
│  │  Confidence: 88%                                       │   │
│  │  CI 95%: [6.4, 9.6]                                    │   │
│  │  Next update: on next message or event                 │   │
│  │                                                        │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── HISTORY ───────────────────────────────────────────┐   │
│  │  ·····●●●●●●●●●●●●●·····●●●●●●●●●●●●                 │   │
│  │  Feb 1          Feb 15          Mar 1                  │   │
│  │  Score: 8→8→8→8→8→9→9→9→8→8→8→9→9→9→9                 │   │
│  │  ↑ sprint started here                                 │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  [🔄 Retest this trait]  [✏️ I think this is wrong]          │
└──────────────────────────────────────────────────────────────┘
```

---

### Evidence Log Tab

A chronological feed of every signal that changed the profile:

```
┌──────────────────────────────────────────────────────────────┐
│  📊 EVIDENCE LOG                          Filter: [All ▾]    │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  2 min ago · 💬 Chat message                           │  │
│  │  "Let's ship this today and iterate" (14 words)        │  │
│  │  ┌ Features extracted:                                 │  │
│  │  │  action_verb_density: 0.21 → QS signal (+0.02)     │  │
│  │  │  urgency: 0.14 → pace_preference (+0.01)           │  │
│  │  │  sentence_length: 7 → directness (+0.01)           │  │
│  │  └ Profile impact: QS state 8.9→9.0, D signal +0.003  │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  18 min ago · 📦 Scoreboard event                      │  │
│  │  PRODUCT_RELEASE: "Extension v0.2.2" (+3 pts, shipping)│  │
│  │  ┌ Signals:                                            │  │
│  │  │  temporal: 08:18 PT → morning activity (+0.04)      │  │
│  │  │  action: QS+IM signal (product release = build+ship)│  │
│  │  │  momentum: 2nd ship today → sprint_strength +0.05   │  │
│  │  │  focus: wirebot-core (same project) → low switch    │  │
│  │  └ Profile impact: chronotype stable, sprint confirmed │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  1 hour ago · ✅ Event approved                         │  │
│  │  Approved "Sendy campaign" in 12 min (fast)            │  │
│  │  ┌ Signals:                                            │  │
│  │  │  approval_latency: 12min → engagement HIGH          │  │
│  │  │  selective: approved 3, skipped 2 → Discernment sig │  │
│  │  └ Profile impact: engagement_level +0.02              │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  3 hours ago · 📄 Document ingested                    │  │
│  │  PAIRING_SCIENCE.md (2379 lines, technical spec)       │  │
│  │  ┌ Signals:                                            │  │
│  │  │  vocabulary_richness: 0.82 → Openness +0.03         │  │
│  │  │  list_usage: 0.45 → Conscientiousness +0.02         │  │
│  │  │  doc_type: technical_spec → FF+Analytical signal    │  │
│  │  │  info_gain: 0.12 (confirmed existing profile)       │  │
│  │  └ Profile impact: Big Five refined, minor             │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Yesterday · 💳 Connected account data                 │  │
│  │  Stripe: $50 payment received                          │  │
│  │  ┌ Signals:                                            │  │
│  │  │  verified_revenue: $50 → Φ5 ground truth update     │  │
│  │  │  revenue_bracket: confirmed "$1-5K/mo"              │  │
│  │  │  override: self-reported matched ✓ (no correction)  │  │
│  │  └ Profile impact: Φ5 confidence +0.05                 │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  [Load more...]                                              │
└──────────────────────────────────────────────────────────────┘
```

---

### Formulas Tab

A reference page showing every active formula, current inputs, and outputs:

```
┌──────────────────────────────────────────────────────────────┐
│  🧮 ACTIVE FORMULAS                                          │
│                                                              │
│  ┌─── TRAIT/STATE BLEND ─────────────────────────────────┐   │
│  │  effective(M) = α × trait(M) + (1-α) × state(M)       │   │
│  │  α = 0.30 + 0.40 × (1 / (1 + drift))                  │   │
│  │                                                        │   │
│  │  Currently:                                            │   │
│  │  QS: 0.55 × 8.0 + 0.45 × 9.0 = 8.45                  │   │
│  │  FT: 0.42 × 3.0 + 0.58 × 5.0 = 4.16                  │   │
│  │  FF: 0.68 × 4.0 + 0.32 × 4.2 = 4.06                  │   │
│  │  IM: 0.70 × 6.0 + 0.30 × 6.0 = 6.00                  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── DISC INFERENCE ────────────────────────────────────┐   │
│  │  D = 0.30×imperative + 0.25×(1-hedge) + 0.20×action   │   │
│  │    + 0.15×(1/sent_len) + 0.10×urgency                 │   │
│  │                                                        │   │
│  │  Currently:                                            │   │
│  │  D = 0.30×0.12 + 0.25×0.92 + 0.20×0.34               │   │
│  │    + 0.15×0.08 + 0.10×0.71 = 0.449                    │   │
│  │  Normalized: D=0.38, I=0.22, S=0.12, C=0.28           │   │
│  │                                                        │   │
│  │  Source weights (current):                             │   │
│  │    Assessment: 28%  Chat: 52%  Behavioral: 20%         │   │
│  │    ↑ Chat weight rose from 20%→52% as messages grew    │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── COMPLEMENT VECTOR ─────────────────────────────────┐   │
│  │  wirebot_effort(A) = (10 - state(A)) / Σ(10 - all)    │   │
│  │                                                        │   │
│  │  Current allocation:                                   │   │
│  │  Tenacity    ████████████████████  33% (your gap)      │   │
│  │  Enablement  █████████████        22%                  │   │
│  │  Galvanizing ████████             13%                  │   │
│  │  Fact Finder ██████               10%                  │   │
│  │  Follow Thru ██████               10%                  │   │
│  │  Discernment █████                 8%                  │   │
│  │  Quick Start ██                    3% (your strength)  │   │
│  │  Implementor ██                    1%                  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── CONVERGENCE EQUATION ──────────────────────────────┐   │
│  │  A(t) = 1 - (1-A₀) × e^(-t/τ) × Π(1-Δᵢ)             │   │
│  │                                                        │   │
│  │  Your current accuracy: 87%                            │   │
│  │  Day 1 accuracy: 35%                                   │   │
│  │  Improvement: +148%                                    │   │
│  │                                                        │   │
│  │  ●━━━━━━━━━━━━━━━━━●━━━━━━━━━━━○━━━━━━━━━━━○          │   │
│  │  35%            87%           94%           97%         │   │
│  │  Day 1      Day 42 (today)  Day 90 (est)  Day 365      │   │
│  │                                                        │   │
│  │  Breakdown:                                            │   │
│  │    Δ_chat = 0.15 × (1 - e^(-312/100)) = 0.144         │   │
│  │    Δ_events = 0.12 × (1 - e^(-847/500)) = 0.097       │   │
│  │    Δ_documents = 0.08 × min(1, 3/5) = 0.048           │   │
│  │    Δ_accounts = 0.10 × min(1, 2/3) = 0.067            │   │
│  │    Δ_retest = 0.05 × min(1, 1/2) = 0.025              │   │
│  │    Δ_drift = 0.05 × min(1, 3/5) = 0.030               │   │
│  │                                                        │   │
│  │  📈 To reach 90%: ~20 more days + 1 more account       │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── CONTEXT WINDOW FORMULAS ───────────────────────────┐   │
│  │  window(W) = sigmoid(Σ signals - threshold)            │   │
│  │  decay: activation × e^(-(t-t_last)/τ_decay)           │   │
│  │                                                        │   │
│  │  SHIPPING_SPRINT (active, 0.82):                       │   │
│  │    signals: 4 ships today + CV=2.1 + consecutive days  │   │
│  │    threshold: 0.60 → exceeded                          │   │
│  │    decay τ: 72h (will fade if shipping stops for 3 days)│  │
│  │    calibration: ↓nudge_freq, ↑next_task_supply         │   │
│  │                                                        │   │
│  │  FINANCIAL_PRESSURE (fading, 0.45):                    │   │
│  │    signals: "debt" mention 5d ago (decayed)            │   │
│  │    threshold: 0.60 → below (deactivating)              │   │
│  │    calibration: revenue-first recs (partial)           │   │
│  └────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

### Accuracy Tab

Shows the system's self-measured accuracy over time:

```
┌──────────────────────────────────────────────────────────────┐
│  📈 ACCURACY REPORT                       Updated: 2h ago    │
│                                                              │
│  Overall: 87% (+3% this week)                                │
│  ████████████████████████████████████████░░░░░ 87%            │
│                                                              │
│  ┌─── BY CONSTRUCT ──────────────────────────────────────┐   │
│  │  Φ1 Action Style     88%  ████████░░  312 observations│   │
│  │  Φ2 Communication    94%  █████████░  312 messages     │   │
│  │  Φ3 Energy Topology  71%  ███████░░░  needs more data  │   │
│  │  Φ4 Risk Disposition 82%  ████████░░  6 decisions obs  │   │
│  │  Φ5 Business Reality 78%  ████████░░  2 accounts       │   │
│  │  Φ6 Temporal         91%  █████████░  42 days          │   │
│  │  Φ7 Cognitive Style  76%  ████████░░  low item count   │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── ACCURACY OVER TIME ────────────────────────────────┐   │
│  │  100% ┤                                                │   │
│  │       │                                     ●●●87%     │   │
│  │   80% ┤                          ●●●●●●●●●●           │   │
│  │       │                   ●●●●●●●                      │   │
│  │   60% ┤            ●●●●●●                              │   │
│  │       │       ●●●●●                                    │   │
│  │   40% ┤  ●●●●●                                        │   │
│  │       │ ●                                              │   │
│  │   35% ● (day 1)                                        │   │
│  │       └───────────────────────────────────────────     │   │
│  │       Feb 1    Feb 8    Feb 15   Feb 22   Mar 1       │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── WHAT WOULD IMPROVE ACCURACY ───────────────────────┐   │
│  │                                                        │   │
│  │  +5% → Connect 1 more account (GitHub recommended)     │   │
│  │  +3% → 50 more chat messages (~3 days at current pace) │   │
│  │  +2% → Complete Energy Topology retest (30 sec)        │   │
│  │  +2% → Upload a business plan or pitch deck            │   │
│  │                                                        │   │
│  │  🎯 Next milestone: 90% (est. 20 days)                 │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── SELF-PERCEPTION GAPS ──────────────────────────────┐   │
│  │                                                        │   │
│  │  Quick Start: You say 9, behavior says 7.2 (-1.8)      │   │
│  │    → You start fast but don't always sustain the pace   │   │
│  │    → Wirebot calibrates to the behavioral score         │   │
│  │                                                        │   │
│  │  Follow Through: You say 3, behavior says 4.2 (+1.2)   │   │
│  │    → You're more structured than you think              │   │
│  │    → This gap is narrowing (was +2.1 two weeks ago)     │   │
│  │                                                        │   │
│  │  These gaps are NORMAL and informative. Wirebot uses    │   │
│  │  the behavioral score for calibration but tracks both.  │   │
│  └────────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌─── PREDICTION TRACK RECORD ───────────────────────────┐   │
│  │                                                        │   │
│  │  Last 30 predictions:  26 correct (87%)                │   │
│  │                                                        │   │
│  │  ✅ "Night owl peak at 10 PM" — confirmed              │   │
│  │  ✅ "Sprint recovery in 2 days" — actual: 1.5 days     │   │
│  │  ✅ "Revenue response: doubled down" — confirmed       │   │
│  │  ❌ "Would approve in <1h" — took 4h (was busy)        │   │
│  │  ✅ "Context switch imminent" — 2 new projects started  │   │
│  │  ...                                                   │   │
│  └────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────┘
```

---

### Override Tab

The founder can challenge or correct any score:

```
┌──────────────────────────────────────────────────────────────┐
│  ⚙️ PROFILE OVERRIDES                                        │
│                                                              │
│  You can adjust any score. Wirebot will factor your          │
│  correction in alongside the algorithmic estimate.           │
│  Overrides decay over time as new evidence accumulates.      │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │  Quick Start                                           │  │
│  │  Algorithm says: 8/10                                  │  │
│  │  You say:  ━━━━━━━━━━●━━  9/10                         │  │
│  │                                                        │  │
│  │  💬 Why? (optional)                                    │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │ I know I start fast — the behavioral dip is      │  │  │
│  │  │ because I was stuck on infrastructure, not slow  │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  │                                                        │  │
│  │  [Apply Override]                                      │  │
│  │                                                        │  │
│  │  ⓘ Override weight: 30% now, decays to 0% over 30d    │  │
│  │    unless behavioral evidence confirms your correction │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
│  Active overrides:                                           │
│  • Quick Start: +1 (applied 3d ago, weight: 24%, decaying)   │
│  • Chronotype: "night owl" → "flexible" (applied 7d ago, 18%)│
│                                                              │
│  ⓘ Overrides are treated as a new evidence source.           │
│    They're factored in with weight proportional to recency.  │
│    If your override is confirmed by behavioral data,         │
│    it becomes permanent. If contradicted, it fades out.      │
│    Either way, the system converges to truth.                │
└──────────────────────────────────────────────────────────────┘
```

**Override formula:**

```
// Override treated as a high-confidence single observation:
override_weight(t) = 0.30 × e^(-(t - t_override) / τ_override)
  where τ_override = 30 days

// Factored into composite like any other source:
composite(M) = w_sr × sr + w_beh × beh + w_inf × inf + w_override(t) × override_value
             / (w_sr + w_beh + w_inf + w_override(t))

// If behavioral data CONFIRMS the override (within ±1.0):
  → Override becomes permanent (τ_override → ∞, weight = 0.15 constant)
  → Log: "Your correction was confirmed by behavior ✓"

// If behavioral data CONTRADICTS the override (delta > 2.0):
  → Override decays normally (30-day half-life)
  → Log: "Your correction didn't match observed patterns. Algorithm reverting."
  → Founder sees this transparently in the Evidence Log
```

---

### Real-Time Update Indicator

Every parameter in the equalizer has a subtle **pulse animation** when it receives new data:

```css
/* Parameter just updated — subtle glow */
.param-updated {
  animation: pulse-update 0.6s ease-out;
}

@keyframes pulse-update {
  0%   { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0.4); }
  70%  { box-shadow: 0 0 0 6px rgba(99, 102, 241, 0); }
  100% { box-shadow: 0 0 0 0 rgba(99, 102, 241, 0); }
}

/* Drift warning — amber pulse */
.param-drifting {
  border-left: 3px solid #f59e0b;
}

/* Context window active — green glow */
.context-active {
  border-left: 3px solid #10b981;
}
```

If the founder is watching the profile while chatting, they can literally **see** each
message update the scores in real-time. The equalizer bars shift, drift indicators appear
or disappear, context windows activate — all live.

---

### Mobile Layout

On mobile (< 768px), the equalizer stacks vertically with collapsible sections:

```
┌────────────────────────────┐
│ ⚡ PROFILE    67 (Trusted) │
│ ███████████████░░░░░ 87%   │
│ 1,247 signals · 42 days   │
├────────────────────────────┤
│ ▸ Action Style         8ⓘ │
│ ▸ Communication DNA   Dⓘ  │
│ ▸ Energy Map          ⚡🛠️ │
│ ▸ Risk Profile        70% │
│ ▸ Cognitive Style     H-I │
│ ▾ Trait vs State           │
│   QS: 8→9 drift=1.0 α=.55│
│   FT: 3→5 drift=1.5 α=.42│
│ ▸ Context: SPRINT (0.82)  │
├────────────────────────────┤
│ 📊 Evidence  🧮 Formulas   │
│ 📈 Accuracy  ⚙️ Override   │
└────────────────────────────┘
```

Tapping any row expands the full detail drawer (same content as desktop, just full-width).

---

### White-Label Theming

The equalizer respects a theme object for white-label deployments:

```typescript
interface ProfileEqualizerTheme {
  background: string;        // card background
  surface: string;           // inner card surface
  primary: string;           // active bars, accent
  secondary: string;         // secondary bars
  warning: string;           // drift warnings
  success: string;           // context active, confirmations
  text: string;              // primary text
  textMuted: string;         // labels, descriptions
  barFilled: string;         // filled portion of EQ bars
  barEmpty: string;          // empty portion of EQ bars
  fontFamily: string;        // override font
  borderRadius: string;      // card rounding
  showFormulas: boolean;     // hide formulas for simplified client view
  showEvidence: boolean;     // hide evidence for simplified client view
  showOverride: boolean;     // allow overrides in client view
  brandLogo?: string;        // replace ⚡ with client logo
  brandName?: string;        // replace "Wirebot" with client name
}
```

**Operator (sovereign):** Full transparency — all tabs, all formulas, all evidence, overrides enabled.

**Client (white-label simplified):** `showFormulas: false` hides the math. Shows scores + descriptions + evidence (what caused it) but not the raw equations. Still transparent about WHAT, just not HOW at the formula level.

---

## API Endpoints

```
POST   /v1/pairing/answers     — Submit assessment answers (batch)
GET    /v1/pairing/status       — Current score + profile summary
GET    /v1/pairing/profile      — Full Founder Profile JSON
GET    /v1/pairing/profile/effective — Effective scores (trait×α + state×(1-α))
GET    /v1/pairing/evidence     — Evidence log (paginated, filterable)
GET    /v1/pairing/evidence/:id — Single evidence entry with full features
GET    /v1/pairing/formulas     — Current formula state (all inputs + outputs)
GET    /v1/pairing/accuracy     — Accuracy metrics + convergence curve
GET    /v1/pairing/drift        — Current drift readings + context windows
GET    /v1/pairing/complement   — Current complement vector + allocation %
POST   /v1/pairing/override     — Submit manual override { trait, value, reason }
GET    /v1/pairing/overrides    — List active overrides + decay status
DELETE /v1/pairing/overrides/:id— Remove an override
POST   /v1/pairing/scan         — Trigger communication scan
GET    /v1/pairing/insights     — Latest inferences + deltas + predictions
GET    /v1/pairing/predictions  — Prediction log + accuracy track record
PATCH  /v1/pairing/profile      — Manual profile corrections (admin)
DELETE /v1/pairing/reset        — Full reset (requires confirmation)
```

---

## Implementation Order

1. **Founder Profile schema + storage** (profile.json v2 with dual-track)
2. **Assessment cards UI** (Svelte component, embeddable)
3. **Scoring algorithms** (Go server endpoints)
4. **Profile Equalizer UI** (Svelte component — the transparent dashboard)
5. **Evidence Log system** (every signal logged with features + impact)
6. **Communication scanner** (analyze chat history in Go, feed evidence log)
7. **Behavioral pattern detector** (analyze scoreboard events, feed evidence log)
8. **Formulas Tab** (live formula display with current values)
9. **Accuracy Tab** (convergence curve, self-measurement, predictions)
10. **Override system** (submit, decay, confirm/contradict)
11. **Wirebot calibration engine** (apply effective profile to chat context)
12. **Background daemon** (continuous signal processing, drift detection, context windows)
13. **White-label theming API**

---

## See Also

- [PAIRING.md](./PAIRING.md) — Original v1 protocol (22 questions)
- [PAIRING_SCIENCE.md](./PAIRING_SCIENCE.md) — Full mathematical specification
- [SOUL.md](/home/wirebot/clawd/SOUL.md) — 12 Pillars that shape Wirebot's behavior
- [OPERATOR_REALITY.md](/home/wirebot/clawd/OPERATOR_REALITY.md) — Current state of the operator
- [SCOREBOARD_PRODUCT.md](./SCOREBOARD_PRODUCT.md) — Scoring system that feeds behavioral data

---

*Pairing v2 is not a feature. It's the foundation of everything Wirebot does.
Without deep calibration, Wirebot is a chatbot. With it, Wirebot is a co-founder.
And the founder sees everything — every number, every reason, every formula.
No black box. Full trust.*
