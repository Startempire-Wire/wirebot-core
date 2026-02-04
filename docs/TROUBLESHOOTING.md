# Wirebot Troubleshooting

> **Common failures, diagnosis, and fixes for the OpenClaw gateway and tunnel.**

---

## Quick Diagnosis

```bash
# 1. Is the service running?
systemctl status openclaw-gateway

# 2. Is the port listening?
ss -tlnp | grep 18789

# 3. Can we probe the gateway?
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export OPENCLAW_STATE_DIR=/data/wirebot/users/verious \
  OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json; \
  openclaw gateway probe'

# 4. Is the tunnel running?
systemctl status cloudflared-wirebot

# 5. Does the public URL respond?
curl -sI https://helm.wirebot.chat
```

---

## Common Issues

### 1. Gateway Not Listening (Connection Refused on 18789)

**Symptom:** `curl http://127.0.0.1:18789/` returns "Connection refused"

**Causes & Fixes:**

| Cause | Diagnosis | Fix |
|-------|-----------|-----|
| Service not running | `systemctl status openclaw-gateway` shows inactive | `systemctl start openclaw-gateway` |
| Service crashed (OOM) | Log shows `FATAL ERROR: Reached heap limit` | Increase `--max-old-space-size` in launcher |
| Service still starting | Service active but port not yet open | Wait 15–25 seconds (see [OPERATIONS.md](./OPERATIONS.md#startup-timeline)) |
| Port conflict | `ss -tlnp \| grep 18789` shows different process | Kill conflicting process, restart service |
| Config validation failure | Gateway refuses to start, log shows schema error | Run `openclaw doctor` to diagnose |

### 2. OOM (Out of Memory) Crash

**Symptom:** Log shows `FATAL ERROR: Reached heap limit Allocation failed - JavaScript heap out of memory`

**Fix:**

Edit `/data/wirebot/bin/openclaw-gateway.sh`:

```bash
# Increase heap size (current: 1024MB, increase if needed)
export NODE_OPTIONS="--max-old-space-size=1536"
```

Then restart: `systemctl restart openclaw-gateway`

**Prevention:** Monitor RSS with `ps aux | grep openclaw-gateway | awk '{print $6/1024 "MB"}'`

### 3. Anthropic Auth Error ("No API key found")

**Symptom:** Log shows `No API key found for provider "anthropic". Auth store: /data/wirebot/users/verious/agents/verious/agent/auth-profiles.json`

**Causes:**

| Cause | Fix |
|-------|-----|
| `auth-profiles.json` missing | Create it (see [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md#auth-profiles-openclaw-provider-auth)) |
| Wrong agent dir | Check which agent dir the gateway resolves (may be `main` not `verious`) |
| OAuth token expired + refresh failed | Re-copy from Claude Code: see below |
| File permissions wrong | `chmod 600 auth-profiles.json && chown wirebot:wirebot auth-profiles.json` |

**Re-copying Claude Code OAuth tokens:**

```bash
# 1. Check Claude Code credentials
cat /root/.claude/.credentials.json | jq '.claudeAiOauth | {expiresAt: (.expiresAt / 1000 | strftime("%Y-%m-%d %H:%M UTC"))}'

# 2. If expired, re-login
claude login

# 3. Extract and rebuild auth-profiles.json
ACCESS=$(jq -r '.claudeAiOauth.accessToken' /root/.claude/.credentials.json)
REFRESH=$(jq -r '.claudeAiOauth.refreshToken' /root/.claude/.credentials.json)
EXPIRES=$(jq -r '.claudeAiOauth.expiresAt' /root/.claude/.credentials.json)
OR_KEY=$(~/.cargo/bin/rbw get "openrouter.ai" --field "Cursor Code Editor API Key")

cat > /tmp/auth-profiles.json << EOF
{
  "profiles": {
    "anthropic:claude-cli": {
      "type": "oauth",
      "provider": "anthropic",
      "access": "${ACCESS}",
      "refresh": "${REFRESH}",
      "expires": ${EXPIRES}
    },
    "openrouter:default": {
      "type": "api_key",
      "provider": "openrouter",
      "key": "${OR_KEY}"
    }
  },
  "usageStats": {}
}
EOF

# 4. Install for both agents
for agent in main verious; do
  cp /tmp/auth-profiles.json /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
  chown wirebot:wirebot /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
  chmod 600 /data/wirebot/users/verious/agents/$agent/agent/auth-profiles.json
done
rm /tmp/auth-profiles.json

# 5. Restart gateway
systemctl restart openclaw-gateway
```

### 4. OpenRouter Auth Error

**Symptom:** OpenRouter requests fail with 401

**Fix:**

```bash
# Verify the key is being injected
cat /run/wirebot/gateway.env | grep OPENROUTER  # Should show the key

# If missing, check rbw
~/.cargo/bin/rbw get "openrouter.ai" --field "Cursor Code Editor API Key"

# If rbw fails, check agent status
~/.cargo/bin/rbw unlock
~/.cargo/bin/rbw sync

# Re-inject
/data/wirebot/bin/inject-gateway-secrets.sh
systemctl restart openclaw-gateway
```

### 5. Cloudflare Tunnel Down (helm.wirebot.chat unreachable)

**Symptom:** `curl https://helm.wirebot.chat` returns 502/503 or timeout

**Check:**

```bash
# Is the tunnel service running?
systemctl status cloudflared-wirebot

# Is the gateway actually listening?
ss -tlnp | grep 18789

# Restart tunnel
systemctl restart cloudflared-wirebot
```

**If both running but still 502:** The tunnel connects to `127.0.0.1:18789`. If the gateway is starting up (15–25s), the tunnel will get connection refused until it's ready.

### 6. ExecStartPre Fails (Secret Injection)

**Symptom:** `systemctl status` shows `ExecStartPre` failed with exit code

| Error | Cause | Fix |
|-------|-------|-----|
| `status=203/EXEC` | Permission denied on script | Check `+` prefix in unit file; script must be executable |
| `status=1` | rbw failed | Check `rbw unlock`, `pgrep rbw-agent`; may need `rbw login` |
| Script hangs | rbw-agent not responding | `rbw stop-agent && rbw unlock` |

### 7. Config Validation Error (Gateway Won't Start)

**Symptom:** Gateway refuses to boot, log shows validation errors

**Fix:**

```bash
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export OPENCLAW_STATE_DIR=/data/wirebot/users/verious \
  OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json; \
  openclaw doctor'

# Auto-fix if safe
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export OPENCLAW_STATE_DIR=/data/wirebot/users/verious \
  OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json; \
  openclaw doctor --fix'
```

---

## Log Analysis

### Key Log Patterns

| Pattern | Meaning |
|---------|---------|
| `[gateway] listening on ws://127.0.0.1:18789` | ✅ Gateway started successfully |
| `[gateway] signal SIGTERM received` | Service is shutting down (expected on restart) |
| `[diagnostic] lane task error` | Agent task failed (check auth, model) |
| `FATAL ERROR: Reached heap limit` | OOM crash — increase heap |
| `No API key found for provider` | Missing auth-profiles.json |
| `[heartbeat] started` | ✅ Health monitoring active |
| `Config validation failed` | Bad config — run `openclaw doctor` |

### Tailing Logs

```bash
# Live gateway log
tail -f /home/wirebot/logs/openclaw-gateway.log

# Systemd journal (includes service events)
journalctl -u openclaw-gateway -f

# Filter for errors only
tail -f /home/wirebot/logs/openclaw-gateway.log | grep -i -E "error|fatal|fail|diagnostic"
```

---

## Recovery Checklist (After Outage)

1. `systemctl status openclaw-gateway` — is the service running?
2. `ss -tlnp | grep 18789` — is the port listening?
3. `systemctl status cloudflared-wirebot` — is the tunnel up?
4. `curl -s http://127.0.0.1:18789/` — does the gateway respond?
5. `curl -sI https://helm.wirebot.chat` — does the public URL work?
6. Check auth: `openclaw models status --probe` — are providers working?
7. Check logs: `tail -50 /home/wirebot/logs/openclaw-gateway.log` — any errors?

---

## See Also

- [OPERATIONS.md](./OPERATIONS.md) — Service management
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Auth profile details
- [MONITORING.md](./MONITORING.md) — Automated health checks
- [CURRENT_STATE.md](./CURRENT_STATE.md) — What should be running
