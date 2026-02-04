# Wirebot Monitoring

> **Health checks, probes, log monitoring, and alerting patterns.**

---

## Quick Health Check

```bash
# One-liner: check everything
systemctl is-active openclaw-gateway && ss -tlnp | grep -q 18789 && echo "Gateway: UP" || echo "Gateway: DOWN"
systemctl is-active cloudflared-wirebot && echo "Tunnel: UP" || echo "Tunnel: DOWN"
```

---

## Gateway Health Probes

### Basic Probe (Port Check)

```bash
ss -tlnp | grep 18789
# Expected: LISTEN ... 127.0.0.1:18789 ... openclaw-gatewa
```

### HTTP Probe (Control UI)

```bash
curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1:18789/
# Expected: 200
```

### RPC Probe (Deep Health)

```bash
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export OPENCLAW_STATE_DIR=/data/wirebot/users/verious \
  OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json; \
  openclaw gateway probe'
# Expected: "Connect: ok ... RPC: ok"
```

### Auth Probe (Provider Status)

```bash
as-user wirebot 'source ~/.nvm/nvm.sh && \
  export OPENCLAW_STATE_DIR=/data/wirebot/users/verious \
  OPENCLAW_CONFIG_PATH=/data/wirebot/users/verious/openclaw.json; \
  openclaw models status --probe'
# Expected: anthropic → ok, openrouter → ok
```

### Public URL Probe (End-to-End)

```bash
curl -s -o /dev/null -w "%{http_code}" https://helm.wirebot.chat
# Expected: 200
```

---

## Systemd Monitoring

### Service Status

```bash
systemctl status openclaw-gateway
systemctl status cloudflared-wirebot
```

### Restart Count

```bash
systemctl show openclaw-gateway --property=NRestarts
# NRestarts=0 is ideal; >0 means crashes occurred
```

### Uptime

```bash
systemctl show openclaw-gateway --property=ActiveEnterTimestamp
```

---

## Log Monitoring

### Live Error Watch

```bash
# Errors only
tail -f /home/wirebot/logs/openclaw-gateway.log | grep -i -E "error|fatal|fail|diagnostic"

# Auth issues
tail -f /home/wirebot/logs/openclaw-gateway.log | grep -i -E "auth|api.key|cooldown|expired"

# Connection events
tail -f /home/wirebot/logs/openclaw-gateway.log | grep -i -E "connected|disconnected|listening"
```

### Log Size Check

```bash
du -sh /home/wirebot/logs/openclaw-gateway.log
# Alert if >100MB — consider rotation
```

---

## Resource Monitoring

### Memory Usage

```bash
# RSS of gateway process
ps aux | grep openclaw-gateway | grep -v grep | awk '{printf "%.0fMB\n", $6/1024}'
# Warning threshold: >800MB (heap limit is 1024MB)
```

### Process Check

```bash
pgrep -f openclaw-gateway > /dev/null && echo "Process: running" || echo "Process: NOT running"
```

---

## Cron-Based Monitoring (Recommended)

### Simple Watchdog Script

```bash
#!/bin/bash
# /data/wirebot/bin/watchdog.sh
# Run via cron: */5 * * * * /data/wirebot/bin/watchdog.sh

LOG="/home/wirebot/logs/watchdog.log"

# Check gateway
if ! ss -tlnp | grep -q 18789; then
    echo "$(date -Iseconds) ALERT: Gateway not listening on 18789" >> "$LOG"
    systemctl restart openclaw-gateway
    echo "$(date -Iseconds) ACTION: Restarted openclaw-gateway" >> "$LOG"
fi

# Check tunnel
if ! systemctl is-active --quiet cloudflared-wirebot; then
    echo "$(date -Iseconds) ALERT: Tunnel service not active" >> "$LOG"
    systemctl restart cloudflared-wirebot
    echo "$(date -Iseconds) ACTION: Restarted cloudflared-wirebot" >> "$LOG"
fi
```

### Auth Token Expiry Monitor

```bash
#!/bin/bash
# /data/wirebot/bin/check-auth-expiry.sh
# Run via cron: 0 */6 * * * /data/wirebot/bin/check-auth-expiry.sh

AUTH_FILE="/data/wirebot/users/verious/agents/main/agent/auth-profiles.json"
EXPIRES=$(jq -r '.profiles["anthropic:claude-cli"].expires // 0' "$AUTH_FILE")
NOW_MS=$(($(date +%s) * 1000))
REMAINING_H=$(( (EXPIRES - NOW_MS) / 3600000 ))

if [ "$REMAINING_H" -lt 2 ]; then
    echo "$(date -Iseconds) WARNING: Anthropic OAuth expires in ${REMAINING_H}h" >> /home/wirebot/logs/watchdog.log
fi
```

---

## Health Dashboard (Control UI)

The OpenClaw Control UI provides a built-in dashboard at:

- **Local:** `http://127.0.0.1:18789/`
- **Public:** `https://helm.wirebot.chat/` (via Cloudflare tunnel)

Dashboard shows:
- Gateway status + uptime
- Connected channels
- Active sessions
- Model status
- Skill status
- Live event log

**Auth required:** Paste the gateway token into Control UI settings.

---

## Alerting Patterns (Future)

| Check | Interval | Action |
|-------|----------|--------|
| Port 18789 listening | Every 5 min | Auto-restart service |
| Tunnel active | Every 5 min | Auto-restart tunnel |
| Gateway HTTP 200 | Every 5 min | Alert if failing |
| Auth probe | Every 6 hours | Alert if expired |
| Memory RSS | Every 15 min | Alert if >800MB |
| Log size | Daily | Rotate if >100MB |

---

## See Also

- [OPERATIONS.md](./OPERATIONS.md) — Service lifecycle
- [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) — When checks fail
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Expected running state
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Auth monitoring
