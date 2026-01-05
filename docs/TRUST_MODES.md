# Wirebot Trust Modes

> **System-level primitives, not UI toggles.**

Trust modes control **capability × risk** at the system level. They are enforced by both WordPress (JWT ceiling) and Gateway (runtime checks).

---

## Mode Overview

| Mode | Name | Who | Capabilities | Guards |
|------|------|-----|--------------|--------|
| 0 | Public / Demo | Anyone | None | Maximum |
| 1 | Standard Founder | Paid users | Core features | On |
| 2 | Advanced Trusted | Vetted power users | Extended features | Partial |
| 3 | Sovereign | Owner only | Full access | Internal off, External max |

---

## Mode 0 — Public / Demo

### Purpose
- Showcase capability
- Build trust
- Convert visitors

### Capabilities
- Demo flows only
- Sample reflections
- Preview of frameworks
- No personalization
- No memory
- No tools

### Guards
- Highest guardrails
- No persistence
- Rate limited
- Heavily filtered output

### Surfaces
- Public wirebot.chat (not logged in)
- Marketing pages

---

## Mode 1 — Standard Founder (Default Product)

### Who
- Most paid users
- LinkedIn-style founders
- Standard membership tier

### Capabilities
- Business context
- Stage tracking (Idea → Launch → Growth)
- Accountability engine
- Checklist progress + priority scoring
- "Next 3 actions" guidance
- Daily / weekly reflection
- Structured reasoning

### Guards ON
- No raw chain-of-thought
- No experimental tools
- Scoped memory (workspace-level)
- Strong prompt-injection defense
- No cross-workspace leakage

### Surfaces
- wirebot.chat (logged in)
- Chrome extension (authenticated)
- SMS (verified phone)

---

## Mode 2 — Advanced Trusted (Invite-Only)

### Who
- Vetted power users
- Technical founders
- Inner circle operators
- Requires explicit approval

### Capabilities
Everything in Mode 1, plus:
- Deeper memory (longer context window)
- Early/beta features
- Tool chaining
- Higher autonomy suggestions
- More direct feedback
- Pattern analysis across longer timeframes

### Guards PARTIALLY OFF
- Expanded context limits
- More permissive tool access
- Still isolated from public surfaces
- Still no cross-user data access

### Surfaces
- wirebot.chat (logged in, advanced flag)
- Chrome extension (authenticated)
- SMS (verified phone)
- Premium Discord (limited)

---

## Mode 3 — Sovereign Mode (Owner Only)

### Who
- System owner only (you)
- Never user-accessible
- Hard allowlist by identity

### Purpose
- Deep personal assistant
- Long-range thinking
- Sensitive data handling
- Experimentation and incubation
- Future feature testing

### Capabilities
- Maximum memory depth
- Freeform exploration
- Internal drafts
- Hypothesis testing
- Potentially agentic behavior
- Access to experimental tools
- Cross-workspace synthesis (owner's workspaces only)

### Guards
- **Internal guards OFF** — Full capability access
- **External guards MAX** — Completely isolated

### Isolation Requirements
- Separate container instance
- Separate database/schema
- Separate encryption keys
- Separate credentials
- No public ingress (localhost + VPN/SSH only)
- No community surfaces
- Possibly single-tenant or local inference

### Surfaces
- Dedicated admin interface
- SSH tunnel access
- Never exposed to public DNS

---

## Trust Escalation Rules

1. **Default is Mode 1** — All new paid users start here
2. **Mode 2 requires invitation** — Manual approval only
3. **Mode 3 is never user-accessible** — Owner identity only
4. **Downgrade is automatic** — Session timeout, suspicious activity
5. **Upgrade requires re-auth** — Fresh JWT with elevated scope

---

## JWT Trust Ceiling

The WordPress plugin embeds `trust_mode_max` in the JWT:

```json
{
  "user_id": "founder_123",
  "workspace_id": "ws_456",
  "trust_mode_max": 1,
  "scopes": ["chat", "checklist", "sms"],
  "surfaces": ["web", "extension"],
  "exp": 1704067200
}
```

The Gateway:
1. Validates JWT signature
2. Checks `trust_mode_max` against requested operation
3. Enforces scope restrictions
4. Logs access attempts

---

## Implementation Notes

### Mode 0-2: Same Gateway Instance
- Namespace isolation via database prefix
- Redis keyspace separation: `wb:m0:`, `wb:m1:`, `wb:m2:`
- Same encryption key (different per workspace)

### Mode 3: Separate Instance
- Container: `wirebot-sovereign`
- Database: `wirebot_sovereign`
- Redis keyspace: `wbs:`
- Encryption key: `/root/.credentials/wirebot-sovereign.key`
- Network: `localhost:8101` (no public binding)

---

## Security Principles

1. **Trust is earned, not configured** — Modes are system-assigned
2. **Ceiling, not floor** — JWT sets maximum, runtime can restrict further
3. **Isolation by default** — Mode 3 is physically separated
4. **No escalation paths** — Can't request higher mode than assigned
5. **Audit everything** — All mode transitions logged
