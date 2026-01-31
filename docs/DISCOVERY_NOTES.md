# Discovery Notes — Wirebot in the Startempire Wire Ecosystem

> **Date:** 2026-01-31
> **Source:** Figma mockup (`Wire Bot`), `wirebot_chat_gpt_brainstorm.mdx`, `bigpicture.mdx`, existing wirebot-core docs
> **Purpose:** Integrate discoveries into the bigger picture before building the client-facing frontend

---

## 1. What the Figma Mockup Reveals

The Wire-Bot Figma design is a **mobile-first business operating dashboard** — NOT a chat app.

### Screens (4 frames, all mobile viewport)

| Frame | Purpose |
|-------|---------|
| **Home Overview** | Main dashboard — welcome, progress, tasks, network, AI input |
| **Task List Total** | Aggregated task view (20 tasks completed state) |
| **Single Task** | Individual task detail with actions |
| **Profile Information** | User profile + same dashboard metrics |

### UI Components Identified

1. **Welcome Header** — "Welcome, Verious!" + user avatar + settings gear
2. **Business Setup Score** — Progress bar (START → COMPLETED) with percentage: "BUSINESS SETUP TASKS - 15%"
3. **Task Counter** — "20 TASKS COMPLETED" (large, prominent)
4. **Next Task Prompt** — "NEXT TASK: Create Mission Statement" with action arrow
5. **Onboarding Progress** — "FINISH ONBOARDING" section with step cards (scrollable horizontal)
6. **Network Growth Partners** — Avatar circles + "CONNECT" button — social/mentorship network
7. **Journey Stage Tabs** — **Idea** (seedling 🌱) | **Launch** (lightbulb 💡) | **Growth** (rocket 🚀) — color-coded pipeline
8. **Daily Stand Up Tasks** — Checklist items ("Create Mission Statement") with flag/share/priority icons
9. **Business Set Up Tasks** — Separate checklist section
10. **Wire Bot Intelligent Suggestions** — Horizontal scroll cards at bottom
11. **"Ask Wire Bot A Question..."** — Input bar at very bottom with robot icon — the AI interface is *minimal*, embedded in the dashboard

### Top-Level Text Labels (Design Specs / Feature Areas)

- Product / Service Scalability Analyzer
- AI Powered Analysis
- Quality, Service, Cost Analyzer
- Action Steps
- Million Dollar Formula

### Layer Structure

- Icons: "Seedling" (Idea), "Lightbulb" (Launch), "Rocket" (Growth)
- Calendar with Day Focus
- Chevron navigation arrows
- Grouped task items with check/flag/share actions
- Font: Inter, 12px regular, black (#000000) on light gray (#B0B0B0) cards

### Key Insight

**The AI is embedded in the workflow, not the other way around.** The dashboard IS the product. "Ask Wire Bot" is one component — the progress tracking, task accountability, network connections, and business stage visualization are the core UX.

---

## 2. What the Brainstorm Doc Reveals

### Core Concept

Wirebot = **AI-powered startup mentor** that:
- Guides entrepreneurs through Idea → Launch → Growth
- Holds them accountable via daily standups, EOD Q&As, weekly/monthly/yearly planning
- Tracks business setup progress as checklist items (priority 1-5)
- Uses collected business data as RAG context for personalized advice
- Calculates continuous progress toward 100% "Set Up" score
- After 100%: shifts to business model evaluation, ROI metrics, scaling to profitability

### Business Model Focus (Post-Setup)

The 4 business models Wirebot coaches toward:
1. **Lead Generation** offers
2. **Core Product** offers
3. **Premium Value** offers
4. **Forced Continuity** offers (subscriptions/recurring)

Goal: **Grow the business owner's MRR**

### Distribution Strategy

| Surface | Version | Notes |
|---------|---------|-------|
| **wirebot.chat** | Full standalone webapp | Premium, full features |
| **Chrome Extension** | Embedded tab in Startempire Wire Network extension | Possibly stripped "light" version |
| **Startempire Wire Network** | Integrated via membership | Network features (mentors, collaborators, mentees) unlock at membership tiers |

### Networking Features

- **Mentor / Collaborator / Mentee** role designation
- Shared accountability data between network connections
- Selective information sharing (user controls visibility)
- Tied to Startempire Wire membership levels (FreeWire, Wire, ExtraWire)

### Technical Ideas from Brainstorm

- SMS/MMS via Twilio (CORE)
- LLM connection (CORE) — ✅ already have via Clawdbot gateway
- TODO robustness like "Remember The Milk"
- TODO protocol: todo.txt format or p2ppsr protocol
- Multi-agent system behind unified Wirebot face
- Crypto token (Solana) for network incentives
- ActivityPub integration for decentralized social

---

## 3. What the Big Picture Doc Reveals

### Startempire Wire Ecosystem (Hub & Spoke)

```
┌─────────────────────────────────────────────────────────────────┐
│                    STARTEMPIRE WIRE ECOSYSTEM                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  startempirewire.com          startempirewire.network            │
│  (Parent Membership Hub)      (Software Distribution Hub)        │
│  - MemberPress + BuddyBoss    - Ring Leader Plugin               │
│  - Member directory            - Screenshots Plugin               │
│  - Content (articles,          - Software downloads               │
│    podcasts, events)           - Documentation                    │
│  - Auth source of truth        - API relay                       │
│       │                              │                            │
│       └──────── WordPress REST ──────┘                            │
│                      │                                            │
│              Ring Leader Plugin                                   │
│              (Central API Relay)                                  │
│                      │                                            │
│         ┌────────────┼────────────┐                               │
│         ▼            ▼            ▼                               │
│   Connect Plugin  Chrome Ext   WIREBOT                           │
│   (Member sites)  (Modern       (wirebot.chat)                   │
│                    Web Ring)                                      │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Membership Tiers (Auth Source: startempirewire.com)

| Tier | Level | Wirebot Access |
|------|-------|----------------|
| **Free (Non-Verified)** | Public | View-only, no membership |
| **FreeWire** | Free, moderated, approved | Basic Wirebot, network exposure |
| **Wire** | Paid, approved | Enhanced Wirebot features, secondary content priority |
| **ExtraWire** | Paid, approved | Full Wirebot features, primary content priority |

### Software Components

1. **Ring Leader Plugin** (WP) — Central API relay, auth handler, data distribution hub
2. **Connect Plugin** (WP/JS) — Distributed to member websites, creates overlay UI
3. **Screenshots Plugin** (WP) — Screenshot capture service for the network
4. **Chrome Extension** (Svelte) — Modern web ring UI, network content, embedded Wirebot tab
5. **Wirebot** (standalone) — wirebot.chat, full business coaching dashboard

### Auth Chain

```
startempirewire.com (MemberPress + WordPress Auth + BuddyBoss + Discord + OAuth/OIDC)
    → Ring Leader Plugin (API relay on startempirewire.network)
        → Chrome Extension (reads membership level)
        → Connect Plugin (reads membership level)
        → Wirebot (reads membership level for feature gating)
```

---

## 4. Reconciliation: Wirebot-Core Docs vs. Big Picture

### What Aligns

| Wirebot-Core Concept | Big Picture Equivalent |
|----------------------|----------------------|
| Trust Mode 0 (Demo) | Free (Non-Verified) |
| Trust Mode 1 (Standard) | FreeWire member |
| Trust Mode 2 (Advanced) | Wire member |
| Trust Mode 3 (Sovereign) | ExtraWire member |
| "Network Integration" (NETWORK_INTEGRATION.md) | Ring Leader Plugin connection |
| "WordPress Plugin" (PLUGIN.md) | Connect Plugin integration |
| "SMS channel" (SMS_OPTIONS.md) | Twilio SMS/MMS (CORE idea) |
| "Multi-agent behind single face" (ARCHITECTURE.md) | Brainstorm doc confirms this pattern |

### What's Missing in Wirebot-Core

1. **Business Setup Checklist Engine** — The core Idea→Launch→Growth task system with priority 1-5 items, progress percentage, "next 3 items" suggestions
2. **Accountability Cadence** — Daily standups, EOD Q&A, weekly planning, monthly reviews, yearly planning (the cron system exists but has zero jobs)
3. **Network Roles** — Mentor/Collaborator/Mentee designation and selective data sharing
4. **Business Model Coaching** — Lead gen, core product, premium value, forced continuity frameworks
5. **MRR Tracking** — The metric Wirebot ultimately optimizes for
6. **Ring Leader API Integration** — Connecting to startempirewire.com/network for auth + content
7. **Chrome Extension Embedding** — Wirebot as a tab in the Startempire Wire Network extension
8. **Membership-Gated Features** — Feature flags tied to FreeWire/Wire/ExtraWire tiers

### What's Extra in Wirebot-Core (Not in Big Picture)

1. **Clawdbot gateway infrastructure** — The big picture doesn't specify implementation; Clawdbot is our execution layer
2. **Letta memory integration** — Advanced memory stack beyond what brainstorm specified
3. **Multiple auth providers** — Anthropic OAuth + OpenRouter (implementation detail)
4. **Cloudflare tunnel architecture** — Deployment detail

---

## 5. Revised Frontend Architecture

The WHITE_LABEL.md concept was wrong. This is not a "white-label chat frontend." It's a **Business Operating Dashboard** powered by Wirebot AI.

### What to Build (wirebot.chat)

```
┌──────────────────────────────────────────────────────┐
│  WIREBOT — Business Operating Dashboard               │
├──────────────────────────────────────────────────────┤
│                                                        │
│  ┌─ HEADER ──────────────────────────────────────┐    │
│  │ Welcome, {Name}!        [avatar] [settings]   │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ BUSINESS SETUP SCORE ────────────────────────┐    │
│  │ ████████░░░░░░░░░░░░░░░░░░  15%               │    │
│  │ 20 TASKS COMPLETED                             │    │
│  │ NEXT TASK: Create Mission Statement     →      │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ ONBOARDING PROGRESS ─────────────────────────┐    │
│  │ [Card 1] [Card 2] [Card 3] [Card 4] →         │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ NETWORK GROWTH PARTNERS ─────────────────────┐    │
│  │ (○)(○)(○)(○)(○)(○)          [CONNECT]          │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ JOURNEY STAGES ──────────────────────────────┐    │
│  │ [🌱 Idea] [💡 Launch] [🚀 Growth]             │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ DAILY STAND UP TASKS ────────────────────────┐    │
│  │ ☐ Create Mission Statement      🚩 👤 ➤       │    │
│  │ ☐ Create Mission Statement      🚩 👤 ➤       │    │
│  │ ☐ Create Mission Statement      🚩 👤 ➤       │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ BUSINESS SET UP TASKS ───────────────────────┐    │
│  │ ☐ Create Mission Statement      🚩 👤 ➤       │    │
│  │ ☐ Create Mission Statement      🚩 👤 ➤       │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ WIRE BOT INTELLIGENT SUGGESTIONS ────────────┐    │
│  │ [Suggestion 1] [Suggestion 2] [Suggestion 3] → │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
│  ┌─ ASK WIRE BOT ───────────────────────────────┐    │
│  │ Ask Wire Bot A Question...          🤖 ➤      │    │
│  └───────────────────────────────────────────────┘    │
│                                                        │
└──────────────────────────────────────────────────────┘
```

### Feature Gating by Membership Level

| Feature | Free | FreeWire | Wire | ExtraWire |
|---------|------|----------|------|-----------|
| Business Setup Checklist | ✅ (limited) | ✅ | ✅ | ✅ |
| Progress Tracking (Idea/Launch/Growth) | ✅ (view) | ✅ | ✅ | ✅ |
| Daily Standups | ❌ | ✅ | ✅ | ✅ |
| EOD Q&A / Weekly Planning | ❌ | ❌ | ✅ | ✅ |
| Monthly/Yearly Planning | ❌ | ❌ | ✅ | ✅ |
| AI Business Advice | ❌ | Basic | Full | Full + Priority |
| Network Partners (Mentor/Collaborator/Mentee) | ❌ | ❌ | ✅ | ✅ |
| Intelligent Suggestions | ❌ | ❌ | ✅ | ✅ |
| Business Model Coaching (4 models) | ❌ | ❌ | ❌ | ✅ |
| MRR Tracking / Scalability Analysis | ❌ | ❌ | ❌ | ✅ |
| AI Powered Analysis | ❌ | ❌ | Basic | Full |
| Million Dollar Formula | ❌ | ❌ | ❌ | ✅ |
| Product/Service Scalability Analyzer | ❌ | ❌ | ❌ | ✅ |
| SMS/MMS Channel (Twilio) | ❌ | ❌ | ❌ | ✅ |

### How Clawdbot Powers This

The Clawdbot gateway is the AI engine underneath. The dashboard frontend calls:

| Dashboard Feature | Clawdbot API |
|-------------------|--------------|
| "Ask Wire Bot" input | `chat.send` (WebSocket) or `/v1/chat/completions` (HTTP) |
| Intelligent Suggestions | `chat.send` with system prompt for suggestions |
| Daily Standup prompts | `cron` jobs triggering `chat.send` |
| Planning sessions | Structured `chat.send` with business context |
| Task prioritization | Agent skill: wirebot-accountability |
| Business analysis | Agent skill: wirebot-core + RAG context |

The **business data** (tasks, progress, checklist state, network connections) lives in a **separate data layer** — not in Clawdbot. Clawdbot reads it for RAG context but doesn't own it.

### Data Architecture

```
┌─────────────────────────────────────────┐
│  wirebot.chat Frontend                   │
│  (Dashboard + AI input)                  │
├─────────────────┬───────────────────────┤
│  Business Data   │  AI Engine            │
│  (REST API)      │  (Clawdbot Gateway)   │
│  ┌─────────────┐ │  ┌─────────────────┐  │
│  │ Tasks/       │ │  │ chat.send       │  │
│  │ Checklists   │ │  │ (with business  │  │
│  │ Progress     │ │  │  context as     │  │
│  │ User profile │ │  │  RAG/system     │  │
│  │ Network      │ │  │  prompt)        │  │
│  │ connections  │ │  │                 │  │
│  │ Business     │ │  │ Skills:         │  │
│  │ metrics      │ │  │ - accountability│  │
│  └──────┬──────┘ │  │ - memory        │  │
│         │        │  │ - network       │  │
│         ▼        │  │ - core          │  │
│  PostgreSQL /    │  └────────┬────────┘  │
│  SQLite / WP API │           │           │
│  (TBD)           │    Clawdbot Gateway   │
│                  │    port 18789         │
└─────────────────┴───────────────────────┘
```

---

## 6. What This Means for the Beta Tester

The beta tester doesn't get a "chat app." They get:

1. **A business setup dashboard** at `ai.theirdomain.com` (or `wirebot.chat/user/`)
2. **Checklist tracking** through Idea → Launch → Growth
3. **Daily standup prompts** from Wirebot
4. **Progress visualization** (% complete, tasks remaining)
5. **AI business advice** contextualized to their collected business data
6. **Network connection** to you (as Mentor) for shared accountability

This is dramatically different from the chat-first approach in WHITE_LABEL.md.

---

## 7. Next Steps (Prioritized)

### Must Build (MVP)

1. **Business Setup Checklist Engine** — The data model for Idea/Launch/Growth tasks with priority levels, completion state, progress calculation
2. **Dashboard Frontend** — Mobile-first, matching the Figma mockup, connected to Clawdbot for AI features
3. **Accountability Cadence** — Daily standup cron jobs with structured prompts
4. **User Onboarding Flow** — Collect business info, create agent context

### Should Build (Phase 1)

5. **Ring Leader API integration** — Auth via startempirewire.com membership
6. **Network Roles** — Mentor/Collaborator/Mentee with selective data sharing
7. **Intelligent Suggestions Engine** — AI-powered "next 3 tasks" recommendation
8. **Business Model Frameworks** — Lead gen, core product, premium value, forced continuity templates

### Could Build (Phase 2)

9. **Chrome Extension tab** — Embedded Wirebot in Startempire Wire Network extension
10. **SMS/MMS channel** via Twilio
11. **Crypto token integration** (Solana)
12. **ActivityPub** for decentralized social features

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Hub-and-spoke model
- [WHITE_LABEL.md](./WHITE_LABEL.md) — Earlier (now superseded) chat-first approach
- [TRUST_MODES.md](./TRUST_MODES.md) — Maps to FreeWire/Wire/ExtraWire
- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix
- [NETWORK_INTEGRATION.md](./NETWORK_INTEGRATION.md) — Ring Leader connection
- [PLUGIN.md](./PLUGIN.md) — WordPress Connect Plugin integration
- [VISION.md](./VISION.md) — Core trajectory
- [Figma mockup](https://www.figma.com/design/w4HJlWiSspeHJe4OSorymV/Wire-Bot) — Source UI design
- [Brainstorm doc](https://github.com/Startempire-Wire/Startempire-Wire-Network/blob/main/docs/wirebot_chat_gpt_brainstorm.mdx)
- [Big Picture doc](https://github.com/Startempire-Wire/Startempire-Wire-Network/blob/main/docs/bigpicture.mdx)
