# Wirebot Capability Matrix

> **Trust Modes × Modules × Surfaces**

---

## Overview

This matrix defines what capabilities are available based on:
- **Trust Mode** (0-3)
- **Module** (feature set)
- **Surface** (interaction channel)

---

## Trust Mode × Capability Matrix

| Capability | Mode 0 | Mode 1 | Mode 2 | Mode 3 |
|------------|--------|--------|--------|--------|
| **Basic Chat** | ✅ Demo only | ✅ | ✅ | ✅ |
| **Business Context** | ❌ | ✅ | ✅ | ✅ |
| **Stage Awareness** | ❌ | ✅ | ✅ | ✅ |
| **Checklist Tracking** | ❌ | ✅ | ✅ | ✅ |
| **Goal Tracking** | ❌ | ✅ | ✅ | ✅ |
| **Daily Standup** | ❌ | ✅ | ✅ | ✅ |
| **EOD Reflection** | ❌ | ✅ | ✅ | ✅ |
| **Weekly Planning** | ❌ | ✅ | ✅ | ✅ |
| **Monthly Recalibration** | ❌ | ✅ | ✅ | ✅ |
| **Next 3 Actions** | ❌ | ✅ | ✅ | ✅ |
| **Pattern Detection** | ❌ | ❌ | ✅ | ✅ |
| **Extended Memory** | ❌ | ❌ | ✅ | ✅ |
| **Tool Chaining** | ❌ | ❌ | ✅ | ✅ |
| **Beta Features** | ❌ | ❌ | ✅ | ✅ |
| **Cross-Workspace Synthesis** | ❌ | ❌ | ❌ | ✅ |
| **Agentic Behavior** | ❌ | ❌ | ❌ | ✅ |
| **Raw Chain-of-Thought** | ❌ | ❌ | ❌ | ✅ |
| **Experimental Tools** | ❌ | ❌ | ❌ | ✅ |

---

## Module × Trust Mode Requirements

| Module | Min Trust | Min Membership | Description |
|--------|-----------|----------------|-------------|
| `core` | 1 | Basic | Basic chat, context, workspace |
| `accountability` | 1 | Basic | Standups, reflections, planning |
| `checklist` | 1 | Basic | Task and checklist management |
| `goals` | 1 | Basic | Goal and KPI tracking |
| `sms` | 1 | Basic | SMS check-ins and prompts |
| `discord` | 1 | Premium | Discord presence features |
| `network` | 1 | Premium | Startempire Wire network features |
| `advanced-memory` | 2 | Premium | Extended context, pattern detection |
| `tools` | 2 | Premium | Tool access and chaining |
| `experimental` | 2 | Premium | Beta features, early access |
| `sovereign` | 3 | Owner | Full access, no restrictions |

---

## Surface × Capability Matrix

| Capability | Web | Extension | SMS | Discord (Free) | Discord (Premium) |
|------------|-----|-----------|-----|----------------|-------------------|
| **Full Chat** | ✅ | ✅ | ✅ | ❌ | ⚠️ Limited |
| **Business Context** | ✅ | ✅ | ✅ | ❌ | ❌ |
| **Checklist Access** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Goal Access** | ✅ | ✅ | ❌ | ❌ | ❌ |
| **Standup Prompts** | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Reflection Prompts** | ✅ | ✅ | ✅ | ❌ | ✅ |
| **Weekly Summaries** | ✅ | ✅ | ❌ | ❌ | ✅ |
| **Network Stats** | ✅ | ✅ | ❌ | ✅ | ✅ |
| **Generic Frameworks** | ✅ | ✅ | ❌ | ✅ | ✅ |
| **Memory** | ✅ | ✅ | ⚠️ Session | ❌ | ❌ |
| **Personalization** | ✅ | ✅ | ✅ | ❌ | ❌ |

**Legend:**
- ✅ Full access
- ⚠️ Limited/restricted
- ❌ Not available

---

## Surface × Trust Mode Access

| Surface | Mode 0 | Mode 1 | Mode 2 | Mode 3 |
|---------|--------|--------|--------|--------|
| **Public Web** (not logged in) | ✅ | ❌ | ❌ | ❌ |
| **Web** (logged in) | ❌ | ✅ | ✅ | ❌ |
| **Chrome Extension** | ❌ | ✅ | ✅ | ❌ |
| **SMS** | ❌ | ✅ | ✅ | ❌ |
| **Discord (Free)** | ✅ | ❌ | ❌ | ❌ |
| **Discord (Premium)** | ❌ | ✅ | ✅ | ❌ |
| **Admin Interface** | ❌ | ❌ | ❌ | ✅ |

---

## Accountability Features Matrix

| Feature | Trigger | Mode 1 | Mode 2 | Mode 3 |
|---------|---------|--------|--------|--------|
| **Daily Standup** | Scheduled (AM) | ✅ | ✅ | ✅ |
| **EOD Reflection** | Scheduled (PM) | ✅ | ✅ | ✅ |
| **Weekly Planning** | Scheduled (Sunday) | ✅ | ✅ | ✅ |
| **Monthly Recalibration** | Scheduled (1st) | ✅ | ✅ | ✅ |
| **Nudge Reminders** | Inactivity | ✅ | ✅ | ✅ |
| **Pattern Alerts** | Detection | ❌ | ✅ | ✅ |
| **Milestone Shoutouts** | Achievement | ✅ | ✅ | ✅ |
| **Avoidance Detection** | Analysis | ❌ | ✅ | ✅ |

---

## Membership × Feature Access

| Membership Tier | Trust Ceiling | Modules | Surfaces |
|-----------------|---------------|---------|----------|
| **Free** | 0 | Demo only | Public web, Discord (free) |
| **FreeWire** | 1 | core, accountability | Web, Extension |
| **Wire** | 1 | + sms, checklist, goals | + SMS |
| **ExtraWire** | 1-2 (invite) | + discord, network, advanced | + Discord Premium |
| **Owner** | 3 | All | All |

---

## Tool Access Matrix

| Tool | Mode 1 | Mode 2 | Mode 3 | Description |
|------|--------|--------|--------|-------------|
| **Checklist CRUD** | ✅ | ✅ | ✅ | Create/update/delete items |
| **Goal CRUD** | ✅ | ✅ | ✅ | Create/update/delete goals |
| **Calendar Read** | ✅ | ✅ | ✅ | View scheduled events |
| **Calendar Write** | ❌ | ✅ | ✅ | Create/modify events |
| **Email Draft** | ❌ | ✅ | ✅ | Draft emails (no send) |
| **Web Search** | ❌ | ✅ | ✅ | Search external sources |
| **Document Read** | ❌ | ✅ | ✅ | Read linked documents |
| **Document Write** | ❌ | ❌ | ✅ | Create/edit documents |
| **API Calls** | ❌ | ❌ | ✅ | External API access |
| **Code Execution** | ❌ | ❌ | ✅ | Sandboxed code running |

---

## Rate Limits

| Surface | Mode 0 | Mode 1 | Mode 2 | Mode 3 |
|---------|--------|--------|--------|--------|
| **Messages/hour** | 5 | 50 | 200 | Unlimited |
| **API calls/day** | 0 | 100 | 500 | Unlimited |
| **SMS/day** | 0 | 10 | 50 | Unlimited |
| **Concurrent sessions** | 1 | 3 | 10 | Unlimited |

---

## Discord Behavior Matrix

| Context | Mode 0 | Premium Mode 1+ |
|---------|--------|-----------------|
| **When mentioned** | Generic response | Acknowledge, point to private |
| **DM** | Not allowed | Limited, point to web |
| **Weekly summary** | ❌ | ✅ Opt-in |
| **Milestone shoutout** | ❌ | ✅ Opt-in |
| **Memory** | ❌ None | ❌ None (privacy) |
| **Personalization** | ❌ None | ❌ None (privacy) |

**Hard Rule:** Discord never replaces private founder work. Never exposes sensitive context.

---

## Capability Enforcement

### At WordPress Layer (JWT Generation)
- Membership tier → trust ceiling
- Module entitlements → scopes
- Surface permissions → surfaces array

### At Gateway Layer (Runtime)
- JWT validation → trust mode enforcement
- Scope checking → capability gating
- Surface verification → behavior adjustment
- Rate limiting → usage enforcement

---

## Quick Reference

**"What can a basic paid user do?"**
- Mode 1, core + accountability + sms modules
- Web, Extension, SMS surfaces
- Full chat with business context
- Standups, reflections, planning
- Checklist and goal tracking
- No pattern detection, no advanced tools

**"What's the upgrade path?"**
- Mode 1 → Mode 2: Invitation after demonstrated engagement
- Mode 2 → Mode 3: Never (owner only)
