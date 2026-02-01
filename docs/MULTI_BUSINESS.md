# Multi-Business Architecture

> Wirebot knows **the operator**, not a business. Businesses are what the operator juggles.

---

## The Problem

Founders rarely have one clean business. They have:
- A main business that's running (or trying to)
- A side project that might become the main thing
- A freelance gig that pays the bills while the main thing gets built
- An investment or partnership they're involved in
- Infrastructure (DevOps, hosting) that supports everything

**Example — Verious:**

| Business | Stage | Role | Revenue | Priority |
|----------|-------|------|---------|----------|
| Startempire Wire (community) | Launch | Founder | Pre-revenue | Primary |
| Startempire Wire Network (software) | Idea | Founder | Pre-revenue | Primary |
| Wirebot (AI platform) | Idea | Founder | Pre-revenue | Primary |
| Philoveracity (umbrella/DevOps) | Growth | Owner | Active | Supporting |
| MainWP Fleet (46 client sites) | Growth | Operator | Active | Income |

A single-business checklist that says "Choose Business Name" is useless here. Wirebot needs to see the **whole picture**, then calmly sort out what matters most *right now*.

---

## Core Design: Operator-Centric, Not Business-Centric

```
                    ┌──────────────────┐
                    │    OPERATOR      │
                    │  (the human)     │
                    │                  │
                    │  Identity        │
                    │  Energy/Time     │
                    │  Financial State │
                    │  Priorities      │
                    └────────┬─────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼────┐   ┌────▼────┐   ┌────▼────┐
         │ Biz A   │   │ Biz B   │   │ Biz C   │
         │ Stage   │   │ Stage   │   │ Stage   │
         │ Tasks   │   │ Tasks   │   │ Tasks   │
         │ Revenue │   │ Revenue │   │ Revenue │
         │ Health  │   │ Health  │   │ Health  │
         └─────────┘   └─────────┘   └─────────┘
```

**Wirebot operates at the operator level:**
- Sees all businesses simultaneously
- Understands dependencies between them (Philoveracity hosts Startempire Wire)
- Identifies which business needs attention RIGHT NOW
- Advises when to focus vs. when to diversify
- Protects the operator from overextension (Pillar 9: Sustainability)
- Knows which businesses generate income vs. which consume it

---

## Data Model Changes

### Business Entity (NEW)

```typescript
interface Business {
  id: string;                     // UUID
  name: string;                   // "Startempire Wire"
  shortName: string;              // "SEW" — for CLI display
  description: string;            // One-liner
  stage: BusinessStage;           // idea | launch | growth | mature | sunset
  role: OperatorRole;             // founder | cofounder | operator | advisor | investor
  revenueStatus: RevenueStatus;   // pre-revenue | active | declining | paused
  monthlyRevenue?: number;        // Current MRR if known
  domain?: string;                // startempirewire.com
  priority: BusinessPriority;     // primary | secondary | supporting | passive
  relatedTo?: string[];           // Business IDs this depends on or supports
  tags?: string[];                // ["saas", "community", "marketplace"]
  createdAt: string;
  updatedAt: string;
}

type BusinessStage = "idea" | "launch" | "growth" | "mature" | "sunset";
type OperatorRole = "founder" | "cofounder" | "operator" | "advisor" | "investor";
type RevenueStatus = "pre-revenue" | "active" | "declining" | "paused";
type BusinessPriority = "primary" | "secondary" | "supporting" | "passive";
```

### Updated ChecklistState

```typescript
interface ChecklistState {
  version: 2;
  operatorId: string;              // "verious"
  businesses: Business[];          // Multiple businesses
  activeBusiness: string;          // Business ID currently focused on
  tasks: Task[];                   // All tasks, each tagged with businessId
  categories: TaskCategory[];
  crossCuttingTasks: Task[];       // Tasks that span businesses
  createdAt: string;
  updatedAt: string;
}

// Task now includes:
interface Task {
  // ... existing fields ...
  businessId: string;              // Which business this belongs to
  crossCutting?: boolean;          // Spans multiple businesses
}
```

### Cross-Cutting Tasks

Some tasks don't belong to a single business:
- "Update personal brand website" (touches Philoveracity + SEW + Wirebot)
- "File quarterly taxes" (all businesses)
- "Review cash flow" (operator-level)
- "Schedule vacation" (sustainability, all businesses)

These are tagged `crossCutting: true` and shown at the operator level.

---

## How Wirebot Decides Priority

### The Calm Sort Algorithm

When the operator asks "what should I work on?", Wirebot evaluates:

1. **What's on fire?** (urgent/critical across all businesses)
2. **What generates income?** (protect revenue-generating businesses first — Pillar 7)
3. **What unblocks the most?** (dependencies between businesses — Pillar 11)
4. **What's the operator's stated priority?** (respect intent — Pillar 4)
5. **What stage needs attention?** (launch businesses need more time than mature ones)
6. **What hasn't been touched?** (detect neglect, surface it gently)
7. **Operator energy/time budget** (Pillar 9 — don't assign 12 hours of work)

### Business Health Score

Each business gets a health score (0-100):

| Factor | Weight | Signal |
|--------|--------|--------|
| Checklist progress | 20% | % complete for current stage |
| Revenue trajectory | 25% | Growing, flat, declining, or n/a |
| Operator attention | 15% | Days since last interaction |
| Blocker count | 20% | Number of critical blocked tasks |
| Stage alignment | 10% | Is the stage accurate? |
| Dependency health | 10% | Are businesses it depends on healthy? |

**Dashboard shows all businesses ranked by health:**

```
⚡ Your Businesses

🔴 Startempire Wire Network    [Idea]  Health: 23/100  ⚠️ No progress in 14 days
🟡 Startempire Wire            [Launch] Health: 45/100  3 critical tasks blocked
🟡 Wirebot                     [Idea]  Health: 52/100  Active development
🟢 Philoveracity               [Growth] Health: 78/100  Stable, income-generating
🟢 MainWP Fleet                [Growth] Health: 85/100  Operating smoothly

💡 Recommendation: SEW Network needs attention — it's your distribution
   channel and it's stalling. 30 minutes on Ring Leader integration
   would unblock Wirebot AND Network simultaneously.
```

---

## CLI Changes

### `wb status` — Now Multi-Business

```
$ wb status

⚡ Operator: Verious | Pairing: 67% (Trusted)
   5 businesses | 2 primary | 1 needs attention

🔴 SEW Network     [Idea]   ██░░░░░░░░ 12%  ⚠️ stale
🟡 SEW Community   [Launch] █████░░░░░ 45%  3 blocked
🟡 Wirebot         [Idea]   ██████░░░░ 52%  active
🟢 Philoveracity   [Growth] ████████░░ 78%  stable
🟢 MainWP          [Growth] █████████░ 85%  healthy

▶ Focus: SEW Network — Ring Leader Plugin (unblocks 2 businesses)
```

### `wb focus <business>` — Switch Active Business

```
$ wb focus wirebot
⚡ Active business: Wirebot [Idea] — 52% health
   14/64 tasks complete | Next: Build Dashboard Frontend [critical]
```

### `wb businesses` — List All

```
$ wb businesses
NAME                 STAGE    PRIORITY    REVENUE     HEALTH
SEW Network          Idea     Primary     Pre-rev     23/100
SEW Community        Launch   Primary     Pre-rev     45/100
Wirebot              Idea     Primary     Pre-rev     52/100
Philoveracity        Growth   Supporting  Active      78/100
MainWP Fleet         Growth   Supporting  Active      85/100

Add: wb add-business "Name"
```

### `wb overview` — Operator-Level View

```
$ wb overview

⚡ Verious — Sunday, Feb 1, 2026 12:30 PM PST

INCOME         $X,XXX/mo (MainWP + Philoveracity)
PRIMARY FOCUS  Wirebot + SEW ecosystem
ENERGY         3 businesses in active development ⚠️

THIS WEEK
  ▶ Ring Leader integration (unblocks SEW Network + Wirebot)
  ▶ Wirebot dashboard prototype (beta tester target)
  ▶ 2 MainWP client renewals due

NEGLECTED
  ⚠️ SEW Network — 14 days no activity
  💡 Even 30 min on Ring Leader would move this forward

CROSS-CUTTING
  □ Update rbw vault (stale OpenRouter key)
  □ Quarterly tax prep (all entities)
```

---

## Pairing Impact

Multi-business awareness deepens pairing significantly:

### New Pairing Questions (Replacing Single-Business Q11-Q18)

```
Q11: "Tell me about everything you're working on right now.
      Businesses, projects, side gigs — all of it."
     → Maps the full business landscape

Q12: "Which of these is the main thing? Which pays the bills?"
     → Sets priority + identifies income source

Q13: "How do these businesses relate to each other?"
     → Maps dependencies (SEW Network distributes Wirebot; Philoveracity hosts SEW)

Q14: "For each one — where is it? Just started, trying to launch, or already running?"
     → Sets stage per business

Q15: "Where do you spend most of your time? Is that where you WANT to spend it?"
     → Surfaces misalignment between effort and priority

Q16: "What's generating revenue right now? How much, roughly?"
     → Maps income sources

Q17: "What's the one business that if it took off, everything else would follow?"
     → Identifies the linchpin (Pillar 11: Maximum Leverage)

Q18: "Are you stretched too thin? Be honest."
     → Pillar 9 (Sustainability) check. Opens conversation about focus vs. diversification.
```

### Continuous Inference — Multi-Business

- Operator completes tasks in Business A but ignores Business B for 2 weeks → Wirebot notices and surfaces it
- Revenue in Business C drops → Wirebot flags and asks if it needs attention
- Operator adds tasks to a new project → Wirebot asks if it's a new business or part of existing one
- Operator's schedule shows all meetings are for one business → flags potential neglect of others

---

## Migration Path

### v1 → v2 Checklist State

```typescript
function migrateV1toV2(v1: ChecklistStateV1): ChecklistState {
  // Create a default business from existing state
  const defaultBusiness: Business = {
    id: randomUUID(),
    name: v1.businessName || "My Business",
    shortName: v1.businessName?.slice(0, 3).toUpperCase() || "BIZ",
    description: "",
    stage: v1.currentStage,
    role: "founder",
    revenueStatus: "pre-revenue",
    priority: "primary",
    createdAt: v1.createdAt,
    updatedAt: v1.updatedAt,
  };

  // Tag all existing tasks with the default business
  const tasks = v1.tasks.map(t => ({
    ...t,
    businessId: defaultBusiness.id,
  }));

  return {
    version: 2,
    operatorId: v1.userId,
    businesses: [defaultBusiness],
    activeBusiness: defaultBusiness.id,
    tasks,
    categories: v1.categories,
    crossCuttingTasks: [],
    createdAt: v1.createdAt,
    updatedAt: new Date().toISOString(),
  };
}
```

---

## Implementation Order

1. **Update PAIRING.md** — Replace single-business questions with multi-business
2. **Update schema.ts** — Add Business entity, version 2, migration
3. **Update engine.ts** — Multi-business filtering, health scoring, cross-cutting
4. **Update tool.ts** — New actions: `businesses`, `focus`, `overview`, `add-business`
5. **Update `wb` CLI** — New commands matching above
6. **Rebuild gateway plugin** — Load updated tool
7. **Seed Verious's businesses** — From pairing data already known

---

## See Also

- [PAIRING.md](./PAIRING.md) — Pairing protocol (updated for multi-business)
- [SINGLEEYE_CONCEPTS.md](./SINGLEEYE_CONCEPTS.md) — Background processing for multi-business monitoring
- [VISION.md](./VISION.md) — Sovereign mode = operator-level, not business-level
