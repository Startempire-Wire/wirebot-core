# Scoreboard Integration Architecture — Secrets, Connections & Verification

> Decisions locked: Hybrid OAuth/API keys. Full financial access (anonymizable after). 
> Encrypted credential store. Configurable Wirebot visibility with effectiveness warnings.

---

## Design Principles

1. **Doctor Model**: Wirebot is the operator's AI doctor. Keeping secrets from it reduces its effectiveness — but the operator always controls what's disclosed.
2. **Full access, anonymizable after**: Scoreboard ingests everything. Anonymization/redaction is a *display* layer, not an *ingestion* layer. You can always un-anonymize your own data.
3. **Encrypt at rest, decrypt at use**: Credentials stored encrypted in the database. Decrypted only in-memory at poll/request time. Never written to disk in plaintext.
4. **Scoped minimum**: OAuth tokens request the minimum scopes needed. API keys should be restricted/read-only where possible.
5. **Multi-tenant from day one**: Even for single-user now, the credential model is per-user. When Wirebot deploys to other operators, their credentials are isolated.

---

## Credential Storage Architecture

### Why Encrypted Database (The n8n/Zapier Pattern)

This is how every major integration platform handles it:
- **n8n**: Credentials encrypted in database with `N8N_ENCRYPTION_KEY`
- **Zapier**: OAuth tokens encrypted at rest in their DB
- **Plaid**: User credentials encrypted with per-tenant keys
- **Supabase Vault**: Encrypted secrets table with `pgsodium`

The pattern works because:
- Single source of truth (no sync between vault and DB)
- Per-credential encryption (compromise of one doesn't leak all)
- Database-level isolation per user/tenant
- Backup and migration just works (encrypted blob travels with data)
- No external dependency (no Vault server to go down)

### Storage Model

```sql
-- Credential store (encrypted at rest)
CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,               -- operator who owns this
    provider TEXT NOT NULL,              -- 'github', 'stripe', 'youtube', etc.
    auth_type TEXT NOT NULL,             -- 'oauth2', 'api_key', 'webhook_secret', 'rss_url'
    
    -- Encrypted blob (JSON containing tokens/keys)
    -- Encrypted with AES-256-GCM using per-credential nonce
    encrypted_data BLOB NOT NULL,
    nonce BLOB NOT NULL,                 -- 12-byte nonce for AES-GCM
    
    -- Metadata (NOT encrypted — needed for management)
    display_name TEXT,                   -- "Startempire Wire GitHub"
    scopes TEXT,                         -- JSON array of granted scopes
    status TEXT DEFAULT 'active',        -- active, expired, revoked, error
    last_used_at TEXT,
    last_error TEXT,
    expires_at TEXT,                     -- OAuth token expiry
    
    -- Sensitivity & visibility
    sensitivity TEXT DEFAULT 'standard', -- 'public', 'standard', 'sensitive', 'financial'
    wirebot_visible BOOLEAN DEFAULT 1,  -- can Wirebot see data from this integration?
    wirebot_detail_level TEXT DEFAULT 'full', -- 'full', 'summary', 'binary', 'none'
    share_level TEXT DEFAULT 'private',  -- 'private', 'anonymized', 'shared', 'public'
    
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_creds_user ON credentials(user_id);
CREATE INDEX idx_creds_provider ON credentials(user_id, provider);
```

### Encryption Details

```
Master Key:     Retrieved from rbw at service start: rbw get "Wirebot Scoreboard Master Key"
                256-bit AES key. Never written to disk. Lives only in process memory.
                For multi-tenant: each tenant gets their own derived key via HKDF.

Per-Credential: Each credential row has its own 12-byte random nonce.
                AES-256-GCM encrypt(master_key, nonce, plaintext_json) → encrypted_data

Decrypted JSON example (what's inside encrypted_data):
{
    "access_token": "ghp_xxxxxxxxxxxx",
    "refresh_token": "ghr_xxxxxxxxxxxx",
    "token_type": "bearer",
    "scope": "repo:read,admin:repo_hook",
    "raw_response": { ... }  // original OAuth response
}

Or for API keys:
{
    "api_key": "sk_live_xxxxxxxxxxxx",
    "key_type": "restricted",
    "permissions": ["read_events", "read_charges"]
}

Or for simple URLs:
{
    "url": "https://startempirewire.com/feed/",
    "type": "rss"
}
```

### Multi-Tenant Key Derivation (Future)

```
Platform Master Key (in HSM/Vault when at scale)
    └── HKDF(master, "wirebot-cred-v1", user_id) → per-user key
        └── AES-GCM(per-user-key, nonce, credential_json)
```

For now (single operator): one master key from rbw. 
For scale: derive per-user keys from a platform master.

---

## Integration Registry

### Tier 1: OAuth (Browser Authorization Flow)

| Provider | OAuth Type | Scopes Needed | Sensitivity | Events Detected |
|----------|-----------|---------------|-------------|-----------------|
| **GitHub** | OAuth App | `repo:read`, `admin:repo_hook` | standard | Releases, PRs merged, deploys, repo activity |
| **Google/YouTube** | Google OAuth 2.0 | `youtube.readonly` | standard | Video published, channel stats |
| **Stripe** | Stripe Connect (read-only) | `read_only` | financial | Payments, subscriptions, invoices |
| **LinkedIn** | OAuth 2.0 | `r_liteprofile`, `r_organization_social` | standard | Posts, engagement on business content |
| **Google Analytics** | Google OAuth 2.0 | `analytics.readonly` | standard | Goal completions, traffic events |

#### OAuth Flow (How It Works)

```
1. Operator clicks "Connect GitHub" in Settings
2. Scoreboard redirects to GitHub authorization URL:
   https://github.com/login/oauth/authorize?
     client_id=WIREBOT_GITHUB_CLIENT_ID&
     redirect_uri=https://wins.wirebot.chat/auth/callback/github&
     scope=repo:status,admin:repo_hook&
     state=<encrypted_csrf_token>

3. Operator authorizes in their browser
4. GitHub redirects back with authorization code
5. Scoreboard exchanges code for tokens (server-side)
6. Tokens encrypted and stored in credentials table
7. Scoreboard registers webhooks automatically (GitHub)
8. Integration status shows "✅ Connected" in Settings

Token refresh handled automatically:
- Background job checks expires_at
- Refreshes tokens before expiry
- If refresh fails → status = 'expired', operator notified
```

### Tier 2: API Keys (Paste in Settings)

| Provider | Key Type | Sensitivity | Events Detected |
|----------|---------|-------------|-----------------|
| **Stripe** (alt) | Restricted API key (read-only) | financial | Same as OAuth — for users who prefer keys over OAuth |
| **PostHog** | Project API key | standard | Custom events, feature flags |
| **ConvertKit/Mailchimp** | API key | standard | Email campaigns sent, subscribers |
| **Vercel/Netlify** | Deploy hook + API token | standard | Deploy success/failure |
| **Docker Hub** | Read-only token | standard | Image pushes |
| **Product Hunt** | API token | standard | Launch events |
| **Custom webhook** | Webhook secret | varies | Any POST to /v1/webhooks/custom |

### Tier 3: No Auth Needed (URLs Only)

| Source | What's Stored | Events Detected |
|--------|--------------|-----------------|
| **Blog RSS** | Feed URL | New posts published |
| **Podcast RSS** | Feed URL | New episodes |
| **Sitemap** | Sitemap URL | New pages published |
| **Public GitHub** | Repo URL (no auth) | Public releases (via API, no token needed) |
| **YouTube Channel** | Channel ID | Public videos (via YouTube Data API v3 with API key) |

---

## Sensitivity Tiers

Every integration is classified:

| Tier | Label | Examples | Default Wirebot Access | Default Share Level |
|------|-------|---------|----------------------|-------------------|
| **public** | Public Info | Blog RSS, public repos | full | public |
| **standard** | Business Activity | GitHub (private repos), email campaigns, social posts | full | private |
| **sensitive** | Business Intelligence | Analytics, CRM pipeline, conversion data | summary | private |
| **financial** | Financial Data | Stripe, bank, revenue, payments | full (doctor model) | private |

### Anonymization Layers

Data flows through these layers:

```
RAW INGESTION (always full detail)
    ↓
STORAGE (encrypted, full detail, never deleted)
    ↓
WIREBOT VIEW (filtered by wirebot_detail_level per credential)
    ↓
SCOREBOARD VIEW (filtered by share_level per credential)
    ↓
NETWORK/LEAGUE VIEW (anonymized: amounts rounded, names stripped, only patterns)
    ↓
GLOBAL BENCHMARK (fully anonymous: only aggregate statistics)
```

### Wirebot Detail Levels (Configurable Per Integration)

| Level | What Wirebot Sees | Effectiveness Warning |
|-------|------------------|----------------------|
| **full** | Everything: amounts, names, timestamps, raw data | ✅ "Full operating picture — maximum effectiveness" |
| **summary** | Aggregates: "3 payments this week totaling $X" | ⚠️ "Wirebot can see patterns but not details. Coaching may be less specific." |
| **binary** | Events happened: "payment received", "deploy succeeded" | ⚠️ "Wirebot knows what happened but not the magnitude. Revenue coaching will be generic." |
| **none** | Only the score points. No event details. | 🔴 "Wirebot is flying blind on this integration. Like a doctor you won't share test results with." |

### Effectiveness Warnings in UI

When operator sets low visibility on a financial integration:

```
┌──────────────────────────────────────────────────┐
│ ⚠️  Stripe visibility set to "binary"            │
│                                                    │
│ Wirebot will know payments happened but not       │
│ amounts. This means:                              │
│                                                    │
│ ❌ Cannot calculate break-even progress           │
│ ❌ Cannot flag revenue decline trends             │
│ ❌ Cannot advise on pricing decisions             │
│ ✅ Can still count revenue events for scoring     │
│                                                    │
│ Wirebot works best with full financial access —   │
│ like a doctor who needs your test results to      │
│ give accurate advice.                             │
│                                                    │
│ Your data is encrypted, never shared with others  │
│ unless you explicitly enable network sharing.     │
│                                                    │
│ [Keep Binary] [Switch to Full]                    │
└──────────────────────────────────────────────────┘
```

---

## Polling Architecture

### How Integrations Are Checked

Three patterns:

#### 1. Webhooks (Real-Time, Preferred)
```
External Service → POST /v1/webhooks/<provider> → Scoreboard
```
- GitHub: auto-registered via API when OAuth connected
- Stripe: configured in Stripe dashboard or via API
- Custom: operator gets a unique webhook URL

#### 2. Polling (Periodic, Fallback)
```
Scoreboard Poller (cron) → GET external API → Process → Store Event
```
- YouTube: poll channel every 30 min for new videos
- Blog RSS: poll feed every 15 min for new posts  
- LinkedIn: poll posts every 1 hour
- Analytics: poll daily

#### 3. Hybrid (Webhook + Polling for Verification)
```
Webhook fires → immediate event created
Poller runs → verifies webhook data matches reality
```
- Stripe: webhook for immediate, poll for reconciliation
- GitHub: webhook for PRs, poll for release verification

### Poller Implementation

```go
// New table for poll schedules
CREATE TABLE poll_schedules (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    credential_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    poll_type TEXT NOT NULL,        -- 'rss', 'api', 'scrape'
    interval_seconds INTEGER,       -- 900 = 15min, 1800 = 30min, 3600 = 1hr
    last_poll_at TEXT,
    last_success_at TEXT,
    last_error TEXT,
    next_poll_at TEXT,
    enabled BOOLEAN DEFAULT 1
);

// Poller goroutine runs every minute, checks what's due
func (s *Server) startPoller() {
    ticker := time.NewTicker(60 * time.Second)
    for range ticker.C {
        s.pollDueIntegrations()
    }
}
```

### Per-Provider Poller Logic

#### GitHub (OAuth)
```
Poll: GET /repos/{owner}/{repo}/releases?per_page=5
      GET /repos/{owner}/{repo}/pulls?state=closed&sort=updated&per_page=10
For each new release/merged PR since last_poll:
  → Create event: PRODUCT_RELEASE / FEATURE_SHIPPED
  → confidence: 0.95 (API-verified)
  → verification_level: STRONG
  → artifact_url: release/PR URL
```

#### Stripe (OAuth or API Key)
```
Webhook (preferred): POST /v1/webhooks/stripe
  Handles: charge.succeeded, customer.subscription.created, invoice.paid
Poll (reconciliation): GET /v1/charges?created[gt]=last_poll_timestamp&limit=20
For each new charge/subscription:
  → Create event: PAYMENT_RECEIVED / SUBSCRIPTION_CREATED / INVOICE_PAID
  → confidence: 0.99 (payment processor = ground truth)
  → verification_level: STRONG
  → Store: amount (encrypted), currency, customer_id (encrypted)
  → Wirebot view: filtered by wirebot_detail_level
```

#### YouTube (Google OAuth or API Key)
```
Poll: GET /youtube/v3/search?channelId={id}&order=date&publishedAfter={last_poll}&type=video
For each new video:
  → Create event: VIDEO_PUBLISHED
  → confidence: 0.95
  → verification_level: STRONG
  → artifact_url: https://youtube.com/watch?v={id}
  → artifact_title: video title
```

#### Blog RSS (No Auth)
```
Poll: GET {feed_url}
Parse: RSS/Atom items, check pubDate > last_poll
For each new post:
  → Create event: BLOG_PUBLISHED
  → confidence: 0.90 (RSS can lag)
  → verification_level: MEDIUM
  → artifact_url: post URL
  → artifact_title: post title
```

#### LinkedIn (OAuth)
```
Poll: GET /v2/ugcPosts?q=authors&authors=urn:li:person:{id}&sortBy=LAST_MODIFIED
For each new post since last_poll:
  → AI classification: is this a business post?
  → If business-relevant (AI score > 0.6):
    → Create event: SOCIAL_POST_BUSINESS
    → confidence: 0.70–0.90 (AI-scored)
    → verification_level: MEDIUM
    → Points scaled by AI quality score
```

---

## Verification Levels

Every event gets a verification level:

| Level | Meaning | Point Multiplier | Examples |
|-------|---------|-----------------|---------|
| **STRONG** | External system confirmed via API/webhook | 1.0x (full points) | Stripe webhook, GitHub API, Deploy platform |
| **MEDIUM** | Detected by polling, URL exists, AI-verified | 0.85x | RSS feed, YouTube API, LinkedIn post |
| **WEAK** | Agent-reported or low-confidence detection | 0.70x | AI agent observation, social scrape |
| **SELF_REPORTED** | User typed it (wb ship, wb complete) | 0.80x | CLI commands (operator's word) |
| **UNVERIFIED** | Submitted but not verified | 0.50x, auto-gated to pending | Low-confidence agent events |

### Verification in Score Engine

```go
func (s *Server) calculateEventScore(eventType, lane string, confidence float64, verificationLevel string) int {
    baseScore := s.getBaseScore(eventType, lane)
    
    // Apply confidence
    score := int(float64(baseScore) * confidence)
    
    // Apply verification multiplier
    switch verificationLevel {
    case "STRONG":
        // full points
    case "MEDIUM":
        score = int(float64(score) * 0.85)
    case "WEAK":
        score = int(float64(score) * 0.70)
    case "SELF_REPORTED":
        score = int(float64(score) * 0.80)
    case "UNVERIFIED":
        score = int(float64(score) * 0.50)
    }
    
    return score
}
```

---

## Network Sharing & Anonymous Benchmarks

### Share Levels (Per User)

| Level | What's Visible to Others | Default |
|-------|-------------------------|---------|
| **private** | Nothing. Fully local. | ✅ Default |
| **anonymized** | Score, streak, W/L record. No business names, no amounts, no event details. Identity = hash. | — |
| **shared** | Score + event types + lane breakdown. Business name visible. No financial amounts. | — |
| **public** | Full scoreboard visible. Social cards show real data. League participation. | — |

### Global Benchmark Aggregation

Even fully private users contribute to **anonymous aggregate statistics** (opt-out available):

```json
{
    "global_benchmarks": {
        "division": "pre_revenue",
        "users_in_division": 847,
        "median_daily_score": 42,
        "median_ships_per_week": 3.2,
        "p25_score": 28,
        "p75_score": 61,
        "avg_streak": 4.1,
        "top_event_types": ["FEATURE_SHIPPED", "BLOG_PUBLISHED", "DEPLOY_SUCCESS"]
    }
}
```

No individual data. Just statistics. Users see: "Your score is in the 72nd percentile of pre-revenue founders."

### Data Anonymization Pipeline

```
User's raw event:
{
    "event_type": "PAYMENT_RECEIVED",
    "artifact_title": "Startempire Wire Pro subscription",
    "amount": 49.00,
    "customer": "john@example.com",
    "business": "Startempire Wire"
}

After anonymization for network:
{
    "event_type": "PAYMENT_RECEIVED",
    "amount_bucket": "$25-$99",     // bucketed, not exact
    "business": null,                // stripped
    "customer": null,                // stripped
    "artifact_title": null           // stripped
}

After anonymization for global benchmark:
{
    "event_type": "PAYMENT_RECEIVED",
    "lane": "revenue"
    // nothing else
}
```

---

## Integration Settings UI

### Settings → Integrations View

```
┌──────────────────────────────────────────────────┐
│ 🔌 Connected Integrations                        │
│                                                    │
│ ✅ GitHub (OAuth)              [Configure] [⚙️]  │
│    Startempire-Wire org · 10 repos · Last: 2m ago │
│    Wirebot: full · Share: private                  │
│                                                    │
│ ✅ Stripe (API Key)            [Configure] [⚙️]  │
│    Live mode · 3 events today · Last: 5m ago       │
│    Wirebot: full · Share: private · 💰 Financial   │
│                                                    │
│ ✅ Blog RSS                    [Configure] [⚙️]  │
│    startempirewire.com/feed · Last: 12m ago        │
│    Wirebot: full · Share: public                   │
│                                                    │
│ ⏸ YouTube (OAuth)              [Reconnect] [⚙️]  │
│    Token expired · Last success: 2d ago            │
│                                                    │
│ ─── Available ─────────────────────────────────── │
│                                                    │
│ [+ GitHub]  [+ Stripe]  [+ YouTube]  [+ LinkedIn] │
│ [+ Blog RSS]  [+ Podcast RSS]  [+ Analytics]      │
│ [+ Custom Webhook]  [+ Vercel]  [+ PostHog]       │
└──────────────────────────────────────────────────┘
```

### Per-Integration Configure Modal

```
┌──────────────────────────────────────────────────┐
│ ⚙️ Stripe Configuration                          │
│                                                    │
│ Connection: ✅ Live mode (restricted key)          │
│ Last poll: 5 minutes ago                           │
│ Events today: 3                                    │
│                                                    │
│ ─── Wirebot Access ──────────────────────────────│
│ Detail level:  ● Full  ○ Summary  ○ Binary  ○ None│
│                                                    │
│ ℹ️ Full: Wirebot sees payment amounts, can advise │
│    on pricing, flag revenue trends, calculate      │
│    break-even progress.                            │
│                                                    │
│ ─── Network Sharing ─────────────────────────────│
│ Share level:   ○ Private  ○ Anonymized  ○ Shared  │
│                                                    │
│ ℹ️ Private: Only you and Wirebot see Stripe data.  │
│    No financial data leaves your instance.         │
│                                                    │
│ ─── Scoring ─────────────────────────────────────│
│ Verification: STRONG (webhook + API reconciliation)│
│ Point multiplier: 1.0x                             │
│ Event types: payment, subscription, invoice        │
│                                                    │
│ [Test Connection]  [Disconnect]  [Save]            │
└──────────────────────────────────────────────────┘
```

---

## Wirebot as First-Class Integration Consumer

### New Gateway Tool: `wirebot_score`

Added to the memory bridge plugin alongside recall/remember/business_state/checklist:

```typescript
// wirebot_score tool
{
    name: "wirebot_score",
    description: "Get current execution score, lane breakdown, streak, season, and recent events. Use to understand operator's current state before advising.",
    parameters: {
        date: { type: "string", description: "YYYY-MM-DD, defaults to today" },
        include_feed: { type: "boolean", description: "Include last 10 events" },
        include_integrations: { type: "boolean", description: "Include integration health status" }
    }
}

// Returns (respecting wirebot_detail_level per integration):
{
    score: 52,
    signal: "green",
    lanes: { shipping: 32, distribution: 12, revenue: 8, systems: 0 },
    streak: { current: 3, best: 7 },
    season: { name: "Red-to-Black", day: 15, remaining: 75, record: "9W-6L" },
    intent: "Ship the checkout page",
    stall_hours: 0.5,
    penalties: 0,
    streak_bonus: 5,
    recent_events: [
        { type: "PAYMENT_RECEIVED", lane: "revenue", title: "Pro subscription", points: 10, 
          detail: "$49.00 from john@..." },  // or "$25-$99 range" or "payment received" per detail level
        ...
    ],
    integration_health: {
        github: { status: "active", last: "2m ago" },
        stripe: { status: "active", last: "5m ago" },
        youtube: { status: "expired", last: "2d ago" }
    }
}
```

### Scorecard Injection (Every Conversation)

Wirebot's system prompt gets a live scorecard injected:

```
OPERATOR SCORECARD (live):
Score: 52 (🟢 WINNING) | Streak: 🔥 3 days | Season: Red-to-Black Day 15 (9W-6L)
Lanes: SHIP 32/40 | DIST 12/25 | REV 8/20 | SYS 0/15
Intent: "Ship the checkout page"
Last ship: 45 min ago — "Checkout page deployed to production"
⚠️ Systems lane empty today. Revenue lane has first real signal.

When advising, factor in:
- Score is green → reinforce momentum, don't redirect
- Revenue lane active for first time → celebrate and protect this signal
- Systems lane empty → if operator asks about infra, connect it to shipping
```

### Wirebot Auto-Push Events

When Wirebot itself creates verifiable artifacts, it auto-pushes:

```
Wirebot deploys code → DEPLOY_SUCCESS (pending, agent source)
Wirebot creates systemd service → INFRASTRUCTURE_ACTIVATED (pending)
Wirebot publishes docs to GitHub → PUBLIC_ARTIFACT (pending)
Wirebot triggers a webhook → AUTOMATION_DEPLOYED (pending)
```

All agent-pushed events remain gated (pending approval) per existing design.

---

## OAuth App Registration Checklist

Before any OAuth integration works, we need app registrations:

### GitHub OAuth App
- [ ] Register at: https://github.com/settings/applications/new
- [ ] App name: "Wirebot Scoreboard"
- [ ] Homepage URL: https://wirebot.chat
- [ ] Callback URL: https://wins.wirebot.chat/auth/callback/github
- [ ] Store client_id + client_secret in rbw as "Wirebot GitHub OAuth"
- [ ] Scopes: `repo:status`, `admin:repo_hook`, `read:org`

### Google OAuth (YouTube + Analytics)
- [ ] Create project in Google Cloud Console
- [ ] Enable YouTube Data API v3 + Google Analytics Data API
- [ ] Create OAuth 2.0 credentials (Web application)
- [ ] Redirect URI: https://wins.wirebot.chat/auth/callback/google
- [ ] Store client_id + client_secret in rbw as "Wirebot Google OAuth"
- [ ] Scopes: `youtube.readonly`, `analytics.readonly`

### Stripe Connect (Read-Only)
- [ ] Already have Stripe CLI authenticated for Startempire
- [ ] Create restricted API key with read-only permissions
- [ ] Configure webhook endpoint in Stripe dashboard
- [ ] Or: use Stripe Connect OAuth for multi-tenant later
- [ ] Store key in rbw as "Wirebot Stripe Integration"

### LinkedIn OAuth
- [ ] Register app at https://www.linkedin.com/developers/
- [ ] Add redirect URI: https://wins.wirebot.chat/auth/callback/linkedin
- [ ] Request `r_liteprofile`, `r_organization_social` scopes
- [ ] Store client_id + client_secret in rbw as "Wirebot LinkedIn OAuth"

---

## Implementation Order

### Phase 3A: Immediate (Wire What We Have)

These require NO OAuth registration — just point existing services:

1. **GitHub Webhooks** — `gh` CLI is already authenticated. Use it to register webhooks on Startempire-Wire repos pointing at `wins.wirebot.chat/v1/webhooks/github`.
2. **Stripe Webhook** — Stripe CLI already authenticated with live key. Register webhook endpoint.
3. **Blog RSS** — Just store `https://startempirewire.com/feed/` as a credential. Build RSS poller.
4. **`wirebot_score` tool** — Add to memory bridge plugin. No external deps.
5. **Score injection into conversations** — Update gateway config.

### Phase 3B: OAuth Registration (1-2 days)

6. Register GitHub OAuth app
7. Register Google OAuth app (YouTube)
8. Build OAuth callback handler in Go server
9. Build credential encryption/decryption
10. Build Settings → Integrations UI in Svelte

### Phase 3C: Pollers (3-5 days)

11. RSS poller goroutine
12. YouTube poller
13. GitHub release/PR poller (supplements webhooks)
14. Stripe reconciliation poller
15. Verification level enforcement in score engine

### Phase 3D: Wirebot Deep Integration (2-3 days)

16. Scorecard injection into system prompt
17. Wirebot auto-push for its own verified work
18. Morning standup reads score + integration health
19. EOD review auto-locks and reports through Wirebot
20. Pending event approval via Wirebot conversation

### Phase 3E: Network Foundation (1 week)

21. Credential table with per-user isolation
22. Share level controls in Settings UI
23. Anonymization pipeline
24. Global benchmark aggregation endpoint
25. Division assignment logic

---

## Security Checklist

- [ ] Master encryption key generated and stored in rbw
- [ ] AES-256-GCM encryption implementation (Go `crypto/aes` + `crypto/cipher`)
- [ ] OAuth state parameter includes CSRF token
- [ ] Redirect URIs whitelist enforced
- [ ] All OAuth tokens have expiry tracking
- [ ] Refresh token rotation
- [ ] Rate limiting on OAuth callback endpoint
- [ ] Webhook signature verification (GitHub: HMAC-SHA256, Stripe: webhook signing secret)
- [ ] No credentials in logs (redact before logging)
- [ ] Credential access audit trail
- [ ] "Disconnect" fully deletes encrypted credentials
- [ ] Integration health endpoint doesn't leak tokens

---

## See Also

- [SCOREBOARD_GAP_ANALYSIS.md](./SCOREBOARD_GAP_ANALYSIS.md) — Full gap analysis
- [SCOREBOARD_PRODUCT.md](./SCOREBOARD_PRODUCT.md) — Product spec
- [SCOREBOARD.md](./SCOREBOARD.md) — Integration concepts from brainstorm
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Server-level secrets architecture
