# Wirebot Launch Order

> **What to build first, what to defer.**

---

## Guiding Principles

1. **Value first** — Ship something useful before it's complete
2. **Trust later** — Mode 1 before Mode 2, Mode 2 before Mode 3
3. **Surfaces incrementally** — Web first, then extension, then SMS
4. **Memory grows** — Start structured, add vector later
5. **Modules unlock** — Core always, others by tier

---

## Phase 1: MVP (Foundation)

**Goal:** A working private founder experience on web.

### Deliverables

| Component | Description | Priority |
|-----------|-------------|----------|
| **Gateway Core** | HTTP API, session management, basic chat | P0 |
| **MariaDB Schema** | Workspaces, checklists, goals, summaries | P0 |
| **Redis Sessions** | Session state, rate limiting | P0 |
| **WordPress Plugin (Basic)** | JWT issuer, settings UI, trust mode | P0 |
| **wirebot.chat UI** | Simple chat interface (logged in) | P0 |
| **Mode 0 Demo** | Public demo flow (not logged in) | P1 |
| **Mode 1 Features** | Business context, stage, basic memory | P0 |
| **Accountability Engine** | Daily standup, EOD reflection | P1 |

### Trust Modes
- Mode 0: ✅ Implemented (demo)
- Mode 1: ✅ Implemented (full)
- Mode 2: ❌ Not yet
- Mode 3: ❌ Not yet

### Surfaces
- Web (logged in): ✅
- Public web: ✅ (Mode 0)
- Extension: ❌
- SMS: ❌
- Discord: ❌

### Modules
- core: ✅
- accountability: ✅ (basic)
- checklist: ✅
- goals: ✅
- sms: ❌
- discord: ❌
- advanced: ❌

### Infrastructure
- Gateway container (Podman): ✅
- LiteSpeed proxy: ✅
- MariaDB database: ✅
- Redis: ✅ (existing)
- DNS (api.wirebot.chat): ✅

### Exit Criteria
- [ ] Founder can log in and chat with Wirebot
- [ ] Business context persists across sessions
- [ ] Stage awareness works (Idea/Launch/Growth)
- [ ] Checklist progress tracked
- [ ] Daily standup prompts working
- [ ] Demo mode functional for visitors

---

## Phase 2: Extension + SMS

**Goal:** Multi-surface presence, accountability reinforcement.

### Deliverables

| Component | Description | Priority |
|-----------|-------------|----------|
| **Chrome Extension Auth** | JWT flow via Connect plugin | P0 |
| **Extension UI** | Chat panel, quick actions | P0 |
| **Twilio Integration** | Inbound/outbound SMS | P0 |
| **SMS Verification** | Phone number verification flow | P0 |
| **SMS Prompts** | Standup/reflection via SMS | P1 |
| **Weekly Planning** | Scheduled weekly prompts | P1 |
| **Monthly Recalibration** | Scheduled monthly prompts | P2 |

### Trust Modes
- Mode 0: ✅
- Mode 1: ✅ (enhanced)
- Mode 2: ❌ Not yet
- Mode 3: ❌ Not yet

### Surfaces
- Web: ✅
- Extension: ✅
- SMS: ✅
- Discord: ❌

### Modules
- sms: ✅
- accountability: ✅ (full)

### Exit Criteria
- [ ] Extension authenticates and chats
- [ ] SMS verification working
- [ ] Standup via SMS working
- [ ] Weekly planning prompts active
- [ ] Multi-surface session continuity

---

## Phase 3: Advanced Mode (Mode 2)

**Goal:** Deeper capabilities for vetted power users.

### Deliverables

| Component | Description | Priority |
|-----------|-------------|----------|
| **Mode 2 Trust Logic** | Invitation system, trust escalation | P0 |
| **Extended Memory** | Longer context, pattern detection | P0 |
| **Tool Chaining** | Sequential tool execution | P1 |
| **Beta Features Flag** | Feature flag system | P1 |
| **Vector Store** | Qdrant integration for semantic recall | P2 |
| **Calendar Integration** | Read/write calendar events | P2 |

### Trust Modes
- Mode 2: ✅ Implemented

### Modules
- advanced-memory: ✅
- tools: ✅
- experimental: ✅

### Exit Criteria
- [ ] Mode 2 invitation flow working
- [ ] Extended memory functioning
- [ ] Pattern detection alerts active
- [ ] Tool chaining operational
- [ ] Vector retrieval enhancing responses

---

## Phase 4: Discord + Network

**Goal:** Community presence, network integration.

### Deliverables

| Component | Description | Priority |
|-----------|-------------|----------|
| **Discord Bot** | Basic presence, mention responses | P0 |
| **Premium Discord Features** | Weekly summaries, milestones | P1 |
| **Network Integration** | Ring Leader sync | P1 |
| **Network Stats** | Cross-network insights | P2 |

### Surfaces
- Discord (Free): ✅ (Mode 0 behavior)
- Discord (Premium): ✅ (Mode 1 features)

### Modules
- discord: ✅
- network: ✅

### Exit Criteria
- [ ] Bot responds in Discord when mentioned
- [ ] Premium members get weekly summaries
- [ ] Milestone shoutouts working (opt-in)
- [ ] Network stats accessible

---

## Phase 5: Sovereign Mode (Mode 3)

**Goal:** Owner-only deep assistant with full access.

### Deliverables

| Component | Description | Priority |
|-----------|-------------|----------|
| **Separate Gateway Instance** | Isolated container | P0 |
| **Separate Database** | wirebot_sovereign | P0 |
| **Separate Keys** | Encryption isolation | P0 |
| **Admin Interface** | Mode 3 access UI | P1 |
| **Agentic Features** | Autonomous task execution | P2 |
| **Local Inference** | Optional local model | P3 |

### Trust Modes
- Mode 3: ✅ Implemented

### Surfaces
- Admin (localhost/VPN): ✅

### Exit Criteria
- [ ] Sovereign container running isolated
- [ ] Separate database operational
- [ ] Full capability access working
- [ ] Admin interface functional
- [ ] Cross-workspace synthesis available

---

## Deferred (Not in Initial Roadmap)

| Feature | Reason | Revisit When |
|---------|--------|--------------|
| Voice (Twilio) | Complexity | After SMS stable |
| Multi-agent | Not aligned with vision | Never (probably) |
| Public API | Security surface | After Mode 2 stable |
| Mobile App | Extension sufficient | User demand |
| Local Inference | Complexity | Mode 3 mature |
| White-label | Business model TBD | After product-market fit |

---

## Technical Debt Allowances

| Phase | Allowed Debt | Must Fix By |
|-------|--------------|-------------|
| MVP | Minimal tests, basic logging | Phase 2 |
| Phase 2 | Limited error handling | Phase 3 |
| Phase 3 | Manual Mode 2 approval | Phase 4 |
| Phase 4 | Discord rate limit workarounds | Phase 5 |

---

## Success Metrics

### Phase 1 (MVP)
- 10 active founders using weekly
- 80% standup completion rate
- Zero critical bugs

### Phase 2 (Extension + SMS)
- 50% of users on extension
- 30% SMS opt-in rate
- Multi-surface usage confirmed

### Phase 3 (Advanced)
- 10 Mode 2 users active
- Pattern detection delivering value
- Tool usage increasing

### Phase 4 (Discord)
- Discord bot stable
- Premium feature adoption
- Community engagement up

### Phase 5 (Sovereign)
- Mode 3 operational for owner
- No security incidents
- Experimental features incubating

---

## Resource Requirements

| Phase | Dev Time | Infrastructure |
|-------|----------|----------------|
| MVP | 4-6 weeks | Existing VPS |
| Phase 2 | 3-4 weeks | + Twilio account |
| Phase 3 | 4-6 weeks | + Qdrant container |
| Phase 4 | 3-4 weeks | + Discord bot hosting |
| Phase 5 | 2-3 weeks | + Sovereign container |

---

## Decision Points

### End of Phase 1
- Is the core value proposition validated?
- Continue or pivot?

### End of Phase 2
- Is multi-surface adding value?
- Which surface to prioritize?

### End of Phase 3
- Is Mode 2 worth the complexity?
- Scale invitations or keep small?

### End of Phase 4
- Is Discord presence worth maintaining?
- Community value vs. distraction?

---

## Summary

**Build order:**
1. Gateway + Plugin + Web UI (MVP)
2. Extension + SMS (multi-surface)
3. Mode 2 + Advanced features
4. Discord + Network
5. Sovereign Mode

**Ship early. Learn fast. Scale trust deliberately.**
