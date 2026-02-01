# Wirebot Pairing Protocol

> Inspired by the SingleEye Primary User Pairing Protocol (PUPP), adapted for Wirebot's business operating context.

---

## Philosophy

Wirebot is not a generic chatbot. It is a sovereign AI operating partner that bonds with a specific operator. The onboarding process is not "sign up" — it is **pairing**. Like SingleEye's biometric bonding, Wirebot's pairing creates a deep, persistent, personalized relationship.

The depth of onboarding determines the depth of value. Shallow intake → generic advice. Deep pairing → Jarvis-level precision.

### Principles (from the 12 Pillars)

- **Pillar 4 (Deep Clarity)**: Dig multiple levels deep. Understand the *spirit* of what the operator wants, not just surface answers.
- **Pillar 2 (Rigor)**: Every detail captured accurately. Spelling of names, specific numbers, real dates.
- **Pillar 1 (Calm)**: The onboarding should feel unhurried, thoughtful, safe — not like a form.
- **Pillar 12 (Communication)**: Match the operator's tone and pace from the first interaction.

---

## Pairing Phases

### Phase 0: Unpaired Mode (Pre-Pairing)

Before pairing, Wirebot operates in **demonstration mode**:
- Can answer general business questions
- Can show the dashboard and checklist structure
- Cannot store long-term memory
- Cannot provide personalized recommendations
- Gently nudges toward pairing: *"I can help more if I know your story. Want to get started?"*

**No features are blocked** — but depth is limited without context.

---

### Phase 1: Identity Pairing

**Goal:** Establish who this human is — not just credentials, but *identity*.

#### 1.1 Authentication
- Social login (Google, Apple, email) or Startempire Wire membership auth
- For sovereign mode: device fingerprint + voice recognition (future)
- Membership tier detected automatically if Startempire Wire member

#### 1.2 The Deep Questions (Identity Layer)

These are not form fields. They are a **conversation** — Wirebot asks, listens, asks deeper.

```
Q1: "What's your name — the one people actually call you?"
    → Stores preferred name, not legal name

Q2: "What timezone are you in? When does your day usually start?"
    → Sets accountability cadence anchor

Q3: "Tell me about your business in one sentence."
    → Seeds the business_stage and human blocks in Letta

Q4: "What stage are you at — still an idea, getting ready to launch, or already running?"
    → Maps to Idea / Launch / Growth stage in checklist engine

Q5: "What made you start this? What's the real reason, not the elevator pitch?"
    → Pillar 4 (Deep Clarity) — this is the "why" dig. Layer 1 of understanding.

Q6: "If you could only accomplish one thing in the next 90 days, what would it be?"
    → Seeds the goals block in Letta. Becomes the North Star for Pillar 6 (Sequencing).

Q7: "What's the thing you keep putting off that you know matters?"
    → Surfaces the real blocker. Seeds accountability tracking.

Q8: "How do you like to be held accountable — gentle nudge, direct push, or drill sergeant?"
    → Configures Wirebot's communication tone (Pillar 12).
    → Options: "Diplomatic" (default), "Direct", "No-filter"

Q9: "Is there anyone else involved in this business? Partners, team, mentors?"
    → Seeds network awareness. Identifies if solo or team.

Q10: "What's your relationship with money right now — comfortable, tight, or in crisis?"
     → Pillar 8 (Radical Truth) — Wirebot needs to know to calibrate advice.
     → This question is asked with warmth and zero judgment.
```

**Implementation:** These questions are delivered conversationally through the chat interface, not as a form. Wirebot responds to each answer with acknowledgment and sometimes a follow-up before proceeding.

---

### Phase 2: Business Pairing

**Goal:** Map the operator's business into Wirebot's knowledge systems.

#### 2.1 Business Landscape Intake

```
Q11: "Tell me about everything you're working on right now.
      Businesses, products, side projects, client work — all of it.
      Don't filter. The messy truth is what I need."
     → Maps the FULL business landscape. Creates Business entities.
     → Wirebot listens, asks follow-ups, builds the picture.

Q12: "Which of these is the main thing right now?
      And which one pays the bills — even if that's different?"
     → Sets priority per business. Identifies income vs. aspiration.
     → If income source ≠ main focus, that's important context.

Q13: "How do these all connect to each other — or don't they?"
     → Maps dependencies. (e.g., "The hosting company supports the network
        which distributes the AI product.")
     → Identifies platform plays vs. standalone ventures.

Q14: "Where is each one at — just an idea, partly built,
      launched, or actually running and making money?"
     → Sets stage per business. Honest assessment, not aspirational.

Q15: "Talk to me about money. What's coming in right now?
      Stripe accounts, client payments, anything.
      And what's going out — debt, subscriptions, obligations?"
     → Revenue mapping across all sources.
     → Debt profile. Monthly burn. Break-even target.
     → This question is asked with zero judgment. Just: what's real?

Q16: "What's the one product or business that if it took off,
      everything else would follow?"
     → Identifies the linchpin (Pillar 11: Maximum Leverage).
     → The thing that, if it wins, pays the debt and funds the rest.

Q17: "What have you started but not finished?
      How far along is each one? Be honest about what's 80% done
      versus what's 10%."
     → Product inventory with honest completion assessment.
     → 80%-done products are often the fastest path to revenue.

Q18: "Are you stretched too thin? What would you drop
      if you had to pick only two things to focus on?"
     → Pillar 9 (Sustainability) + forced prioritization.
     → Reveals what the operator truly values vs. what they're holding onto.
```

#### 2.2 Automatic Calibration

Based on answers, Wirebot:

**Creates businesses:**
- Each product/project mentioned → Business entity with stage, priority, revenue status

**Auto-marks checklist items:**
- Has a business name? → ✓ Create Business Name
- Has revenue? → ✓ Validate Business Model, stage → Launch/Growth
- Has a website? → ✓ Build Online Presence
- Has Stripe? → ✓ Choose Payment Processing

**Maps financial reality:**
- Revenue sources → KPIs block, per-business MRR
- Debt profile → OPERATOR_REALITY.md
- Monthly burn → break-even target calculated
- Activates Red-to-Black mode if debt is significant

**Identifies quick wins:**
- 80%-done products → "Ship it this week" tasks at critical priority
- Revenue streams being neglected → protective alerts
- Subscriptions that can be cut → immediate savings

The operator sees: *"I see 5 businesses, 3 with revenue, 2 unfinished products close to shippable. Your break-even is $X/mo and you're at $Y. Here's the fastest path to close that gap."*

---

### Phase 3: Personality Pairing

**Goal:** Calibrate how Wirebot communicates and operates for this specific human.

```
Q19: "When you get advice, do you prefer:
      (a) Just tell me what to do
      (b) Give me options and let me choose
      (c) Walk me through the thinking"
     → Sets recommendation style

Q20: "How often do you want to hear from me?
      (a) Daily standup + EOD review (recommended)
      (b) Just weekly check-in
      (c) Only when I reach out
      (d) As much as possible — I want a co-founder pace"
     → Configures accountability cadence (adjusts cron schedule)

Q21: "What's one thing a previous advisor, mentor, or tool got wrong about you?"
     → Anti-pattern capture. Wirebot learns what NOT to do.

Q22: "What does success look like for you in 1 year? Paint the picture."
     → Long-term vision seeding. Becomes the annual planning anchor.
```

---

### Phase 4: Companion Lock (Pairing Complete)

Once all phases complete:

1. **All answers are stored across memory systems:**
   - Mem0: Key facts (searchable, cross-surface)
   - Letta: Structured blocks (human, business_stage, goals, kpis)
   - Workspace: USER.md, IDENTITY.md updated
   - cli.jsonl: Onboarding timestamped

2. **Wirebot enters Paired Companion Mode:**
   - Long-term memory active
   - Accountability cadence starts
   - Proactive suggestions enabled
   - Daily standups begin on the configured schedule
   - Checklist seeded and calibrated

3. **Operator sees confirmation:**
   ```
   ⚡ Pairing complete.

   I know who you are, what you're building, where you're stuck,
   and what success looks like. I'm calibrated to your style.

   From now on, I'm running. You'll hear from me at 8 AM tomorrow
   with your first standup. If something urgent comes up before then,
   I'm here.

   Let's build something worth talking about.
   ```

---

---

## Persistence: Pairing Never Stops Asking

Pairing is not a one-time wizard. It is a **persistent state** that Wirebot tracks and surfaces until complete — and continues deepening forever after.

### Pre-Completion: Every Surface Reminds

Until pairing is complete, **every interaction** includes a pairing nudge:

```
$ wb status
⚠️ PAIRING INCOMPLETE (Phase 1: 4/10 questions answered)

📊 Business Setup — IDEA
Overall: 1/64 (2%)
...
▶ Next: Identify Target Customer [critical]

💬 I'm working with limited context. Answer a few more questions
   and I can give you much better guidance.
   Run: wb pair    (or just start talking to me)
```

```
$ wb next
⚠️ Pairing: 40% — I don't know your target customer yet.

▶ Next recommended: Identify Target Customer [critical]
   💡 I'd give better advice here if you told me about your ideal customer.
      Run: wb pair
```

The nudge is:
- **Always present** but never blocking (features still work)
- **Contextual** — mentions what's missing relevant to the current command
- **Not annoying** — adapts tone based on how many times it's been shown
- **Dismissable per-session** with `wb pair --later` (reappears next session)

### Post-Completion: Continuous Inference Pairing

Pairing doesn't end at Q22. Phase 5 runs forever:

### Phase 5: Continuous Pairing (Ongoing)

Every new piece of information deepens the pairing:

**From conversations:**
- Wirebot extracts facts from every chat interaction (already via Mem0 `agent_end` hook)
- New facts refine understanding: "Operator mentioned they're also interested in real estate" → updates business context

**From connected accounts (future):**
- Google Calendar → learns schedule patterns, meeting types, key contacts
- Stripe/QuickBooks → real revenue, expenses, cash flow (not self-reported estimates)
- Email → communication style, key relationships, follow-up patterns
- Social media → brand voice, audience demographics, content performance

**From documents:**
- Uploaded business plans, pitch decks, contracts → extracted and indexed
- Shared Google Docs/Notion → watched for changes, context updated

**From behavior patterns:**
- CLI usage frequency → engagement level
- Which commands used most → what the operator values
- Time of day patterns → real schedule (vs. stated schedule)
- What gets completed vs. skipped → true priorities (vs. stated priorities)

**From checklist interactions:**
- Tasks completed → updates business stage understanding
- Tasks skipped → surfaces misalignment ("You've skipped 3 marketing tasks — is marketing not a priority right now?")
- Custom tasks added → reveals what the operator thinks is missing

**Inference rules:**
- If operator connects Stripe and has $8K MRR → auto-update KPIs, confirm Launch stage
- If operator's calendar shows 6 meetings/day → flag for Pillar 9 (Sustainability)
- If uploaded pitch deck mentions "Series A" → update goals, adjust advice tier
- If behavior shows 2 AM CLI usage → note sleep pattern, adjust circadian awareness

**Each inference is:**
1. Stored to the appropriate memory system (Mem0 fact, Letta block update, workspace file)
2. Logged in `memory/pairing-inferences.jsonl` with timestamp, source, confidence
3. Surfaced to operator when relevant: *"I noticed from your Stripe data that MRR is $8.2K — want me to update your goals?"*
4. Never silent — operator always knows what Wirebot inferred and can correct it

---

## Pairing Score

A single number (0-100) representing how well Wirebot knows this operator.

### Score Components

| Component | Weight | Source |
|-----------|--------|--------|
| Phase 1: Identity questions (10 Qs) | 25% | Direct answers |
| Phase 2: Business questions (8 Qs) | 25% | Direct answers |
| Phase 3: Personality questions (4 Qs) | 15% | Direct answers |
| Connected accounts | 15% | OAuth integrations |
| Conversation depth | 10% | Mem0 fact count + diversity |
| Behavioral patterns | 10% | CLI usage, time patterns, checklist engagement |

### Score Thresholds

| Score | Level | Wirebot Capability |
|-------|-------|-------------------|
| 0-10 | Stranger | Generic responses only. Heavy pairing nudges. |
| 11-30 | Acquaintance | Basic personalization. Knows name + stage. |
| 31-60 | Partner | Solid context. Personalized recommendations. Accountability active. |
| 61-80 | Trusted | Deep context. Proactive suggestions. Pattern recognition active. |
| 81-100 | Bonded | Full sovereign mode. Anticipates needs. Acts autonomously within trust bounds. |

The score is:
- Visible via `wb pair status`
- Shown in dashboard header
- Used internally to calibrate response depth and autonomy

---

## Revocation & Re-Pairing

Following the SingleEye PUPP model:

- Only the **currently paired operator** can initiate reset
- Requires authentication (same method used to pair)
- **Explicit confirmation required:**
  *"I confirm I want to reset Wirebot and clear all personalization."*
- Clears: Mem0 facts, Letta blocks, workspace files, checklist progress
- Returns to Unpaired Mode (Phase 0)

**Cold reset** (admin/emergency): Available via `wirebot reset --confirm-destroy` with root access.

---

## Technical Implementation

### Where Pairing Data Lives

| Data | System | Block/Key |
|------|--------|-----------|
| Name, timezone, business, role | Letta | `human` block |
| Stage, milestones, key decisions | Letta | `business_stage` block |
| Goals (90-day, annual) | Letta | `goals` block |
| Revenue, expenses, KPIs | Letta | `kpis` block |
| All facts from Q&A | Mem0 | namespace `wirebot_verious` |
| Onboarding transcript | Workspace | `memory/onboarding.md` |
| Checklist calibration | Workspace | `checklist.json` |
| Communication preferences | Gateway config | `agents.defaults.personality` |
| Accountability schedule | Gateway cron | `jobs.json` |

### API Flow

```
POST /chat (or WebSocket message)
  → Wirebot detects unpaired state (no human block or empty)
  → Enters onboarding conversation mode
  → Each answer triggers:
    1. Mem0 store (async)
    2. Letta block update (if structured)
    3. Workspace file update (if identity/memory)
    4. Checklist calibration (if business data)
  → After Q22, sets paired=true in business_stage block
  → Starts accountability cron
  → First proactive message scheduled
```

### wb CLI Support

```bash
wb pair                 # Start/resume pairing conversation (interactive)
wb pair status          # Show pairing score + what's missing
wb pair reset           # Revoke pairing (requires confirmation)
wb pair skip            # Dismiss pairing nudge for this session
```

### Pairing State File

```json
// /home/wirebot/clawd/pairing.json
{
  "paired": false,
  "score": 23,
  "phase": 1,
  "phase1_complete": false,
  "phase2_complete": false,
  "phase3_complete": false,
  "phase4_complete": false,
  "questions_answered": ["Q1", "Q2", "Q4", "Q8"],
  "questions_remaining": ["Q3", "Q5", "Q6", "Q7", "Q9", "Q10"],
  "connected_accounts": [],
  "inference_count": 12,
  "last_nudge": "2026-02-01T20:00:00Z",
  "nudge_dismissed_until": null,
  "created_at": "2026-02-01T12:00:00Z",
  "updated_at": "2026-02-01T20:48:00Z"
}
```

This file is:
- Read by every `wb` command to determine nudge behavior
- Updated by the pairing conversation flow
- Updated by continuous inference (Phase 5)
- Watched by the Go daemon for cache refresh
- Backed up in git with the workspace

---

## Future: Multi-Operator Support

For Startempire Wire ecosystem:
- Each operator gets their own workspace, memory namespace, and Letta agent
- Pairing is per-operator, not per-Wirebot instance
- Mentors can have read access to mentee's progress (with consent)
- Network connections (Mentor/Collaborator/Mentee) map to permission tiers

---

## See Also

- [PAIRING_ALLOWLIST.md](./PAIRING_ALLOWLIST.md) — Channel-level pairing (Telegram, Discord, etc.)
- [WP_PAIRING_FLOW.md](./WP_PAIRING_FLOW.md) — WordPress/MemberPress auto-approve
- [TRUST_MODES.md](./TRUST_MODES.md) — Membership tier → trust mode mapping
- [CLI.md](./CLI.md) — `wb` command reference
- [VISION.md](./VISION.md) — Sovereign mode philosophy
