# Authentication & Secret Management

> **All secrets via rbw (Bitwarden vault). No plaintext on disk. OAuth auto-refresh.**

---

## Principles

1. **No plaintext secrets on disk** — all new secrets stored in Bitwarden vault, retrieved via `rbw`
2. **Runtime injection only** — secrets written to tmpfs (`/run/wirebot/`) at service start, cleared on reboot
3. **Auth profiles for Clawdbot** — provider credentials stored in `auth-profiles.json` per agent
4. **OAuth bidirectional sync** — Clawdbot auto-refreshes Anthropic tokens and writes back to Claude Code

---

## Secret Storage: rbw (Bitwarden CLI)

Wirebot follows the server-wide secret storage policy: all secrets come from `rbw` (unofficial Bitwarden CLI with background agent).

### How It Works

```
Service starts (systemd)
    │
    ▼
ExecStartPre (root) ──→ rbw vault ──→ /run/wirebot/gateway.env (tmpfs, mode 600)
    │
    ▼
Main process (wirebot) sources /run/wirebot/gateway.env
    │
    ▼
Secrets available as env vars (OPENROUTER_API_KEY, etc.)
```

### Current Vault Entries Used

| Vault Entry | Field | Variable | Verified |
|-------------|-------|----------|----------|
| `openrouter.ai` | `Cursor Code Editor API Key` | `OPENROUTER_API_KEY` | ✅ |

### Adding New Secrets

1. Add entry to Bitwarden vault (web UI or `rbw add`)
2. Add retrieval line to `/data/wirebot/bin/inject-gateway-secrets.sh`
3. Add env var reference to `/data/wirebot/bin/clawdbot-gateway.sh` or `clawdbot.json` env block
4. Restart service: `systemctl restart clawdbot-gateway`

```bash
# Example: adding a Brave Search API key
# In inject-gateway-secrets.sh:
BRAVE_KEY=$("$RBW" get "Brave Search API" --raw 2>/dev/null | jq -r '.notes' || true)

# In gateway.env output:
BRAVE_SEARCH_API_KEY=${BRAVE_KEY}
```

---

## Auth Profiles (Clawdbot Provider Auth)

Clawdbot uses **auth profiles** for API keys and OAuth tokens. These live per-agent:

```
/data/wirebot/users/verious/agents/<agentId>/agent/auth-profiles.json
```

### Current Agents

| Agent ID | Auth Profiles | Purpose |
|----------|--------------|---------|
| `main` | `anthropic:claude-cli` (OAuth), `openrouter:default` (API key) | Default CLI agent |
| `verious` | Same (copied) | Named agent from config |

### Profile Format

```json
{
  "profiles": {
    "anthropic:claude-cli": {
      "type": "oauth",
      "provider": "anthropic",
      "access": "<access-token>",
      "refresh": "<refresh-token>",
      "expires": 1768531894752
    },
    "openrouter:default": {
      "type": "api_key",
      "provider": "openrouter",
      "key": "<api-key>"
    }
  },
  "usageStats": {}
}
```

### Profile Types

| Type | Fields | Use Case |
|------|--------|----------|
| `api_key` | `provider`, `key` | OpenRouter, Groq, etc. |
| `oauth` | `provider`, `access`, `refresh`, `expires` | Anthropic (Claude Pro/Max), OpenAI Codex |

---

## Anthropic OAuth (Claude Code Sync)

The Anthropic provider uses OAuth tokens synced from Claude Code credentials on this server.

### How It Works

1. Claude Code (`/root/.claude/.credentials.json`) holds OAuth tokens (access + refresh)
2. Tokens were copied into Clawdbot's `auth-profiles.json` as `anthropic:claude-cli`
3. Clawdbot auto-refreshes expired tokens using the refresh token
4. Clawdbot writes refreshed tokens back to Claude Code credentials (bidirectional sync)

### Token Lifecycle

```
Claude Code login (manual, infrequent)
    │
    ▼
/root/.claude/.credentials.json
    │
    ▼ (copied once during setup)
auth-profiles.json → anthropic:claude-cli
    │
    ▼ (auto-refresh on expiry)
Clawdbot runtime refreshes → updates auth-profiles.json
    │
    ▼ (bidirectional write-back)
Claude Code credentials updated
```

### Current Subscription

- **Type:** Claude Max 5x
- **Rate limit tier:** `default_claude_max_5x`
- **Token refresh:** Automatic (every ~8 hours)

### Refreshing Manually

If the refresh token itself expires or gets invalidated:

```bash
# On the server, re-login to Claude Code
claude login

# Then copy fresh credentials to Clawdbot
# (see TROUBLESHOOTING.md for the procedure)
```

---

## Gateway Auth (Token)

The Clawdbot Gateway itself uses token auth for WebSocket connections:

```json5
{
  gateway: {
    auth: {
      mode: "token",
      token: "<uuid>",
      allowTailscale: true
    }
  }
}
```

- **Token location:** `gateway.auth.token` in `/data/wirebot/users/verious/clawdbot.json`
- **Who uses it:** Control UI, WebChat, CLI probe commands, WordPress plugin (server-side proxy)
- **Never expose** the gateway token to client-side JavaScript

### Trusted Proxies

The Cloudflare tunnel connects from localhost. Config:

```json5
{
  gateway: {
    trustedProxies: ["127.0.0.1"]
  }
}
```

---

## Model Failover

Clawdbot rotates auth profiles and falls back across models:

1. **Auth profile rotation** — within the current provider (round-robin, cooldowns)
2. **Model fallback** — to next model in `agents.defaults.model.fallbacks`

### Cooldown Behavior

- Auth errors → exponential backoff (1m → 5m → 25m → 1h cap)
- Billing failures → longer backoff (5h start, doubles, 24h cap)
- Session stickiness → pinned profile per session for cache warmth

### Checking Auth Status

```bash
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export CLAWDBOT_STATE_DIR=/data/wirebot/users/verious \
  CLAWDBOT_CONFIG_PATH=/data/wirebot/users/verious/clawdbot.json; \
  clawdbot models status --probe'
```

---

## File Permissions (Security)

| Path | Mode | Owner | Contains |
|------|------|-------|----------|
| `auth-profiles.json` | `600` | wirebot | OAuth tokens + API keys |
| `clawdbot.json` | `600` | wirebot | Gateway token |
| `/run/wirebot/gateway.env` | `600` | wirebot | Runtime env secrets |
| `/run/wirebot/` | `700` | wirebot | Runtime secret dir (tmpfs) |
| `inject-gateway-secrets.sh` | `700` | root | rbw access script |

---

## See Also

- [OPERATIONS.md](./OPERATIONS.md) — Service lifecycle + launcher details
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — Auth error diagnosis
- [MONITORING.md](./MONITORING.md) — Auth probe checks
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Current auth status
- Server policy: `/root/.agent-kb/BITWARDEN_RBW.md` — rbw reference
- Server policy: `/root/.agent-kb/SAFETY_RULES.md` — Secret storage rules
- Clawdbot docs: `concepts/model-failover` — Full failover mechanics
- Clawdbot docs: `concepts/oauth` — OAuth exchange + sync details
