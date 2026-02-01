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

#### 2.1 Business Profile Intake

```
Q11: "What do you sell — or what will you sell?"
     → Core product offer identification

Q12: "Who is your ideal customer? Describe them like you're pointing them out in a crowd."
     → Target market seeding (used for marketing guidance later)

Q13: "Do you have revenue yet? If so, roughly how much per month?"
     → Seeds KPIs block. Determines if Idea, Launch, or Growth is accurate.

Q14: "What's your biggest expense right now?"
     → Financial awareness for Pillar 7 (Protect What's Built)

Q15: "Do you have a website? Social media? Any online presence?"
     → Inventory of existing assets

Q16: "What tools do you already use? (e.g., Stripe, QuickBooks, Gmail, Notion, etc.)"
     → Integration opportunity mapping

Q17: "What's working right now in your business?"
     → Identifies strengths to protect and leverage (Pillar 5, 7, 11)

Q18: "What's NOT working?"
     → Surfaces pain points for immediate action (Pillar 8 — truth)
```

#### 2.2 Checklist Calibration

Based on answers, Wirebot auto-marks checklist items as complete:
- Has a business name? → ✓ Create Business Name
- Has revenue? → ✓ Validate Business Model, likely Launch stage
- Has a website? → ✓ Build Online Presence
- Etc.

The operator sees: *"Based on what you've told me, you're at 34% setup. Here's what's next."*

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
wb onboard              # Start/resume onboarding (interactive)
wb onboard status       # Show pairing completion %
wb onboard reset        # Revoke pairing (requires confirmation)
```

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
