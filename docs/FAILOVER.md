# Wirebot Failover Architecture

> **Goal:** Zero-downtime failover for sovereign client instances. Sub-minute data loss, <2 minute recovery.

---

## 1. Current Infrastructure (Primary)

```
┌─────────────────────────────────────────────────────────────┐
│  PRIMARY VPS (AlmaLinux, cPanel)                            │
│  IP: 199.167.200.52                                        │
│                                                             │
│  Services:                                                  │
│    openclaw-gateway.service    :18789  (AI gateway)         │
│    wirebot-scoreboard.service  :8100   (scoreboard API+PWA) │
│    mem0-wirebot.service        :8200   (fact memory)        │
│    wirebot-memory-syncd        :8201   (memory bridge)      │
│    letta-wirebot (podman)      :8283   (business state)     │
│    cloudflared-wirebot.service         (CF tunnel)          │
│                                                             │
│  Domains (via Cloudflare Tunnel):                           │
│    helm.wirebot.chat  → :18789                              │
│    wins.wirebot.chat  → :8100                               │
│    api.wirebot.chat   → :8100                               │
│    ai.CLIENT.com      → :18789  (future client instances)   │
│                                                             │
│  Data (~45MB total):                                        │
│    /data/wirebot/users/verious/memory/verious.sqlite  6.4MB │
│    /data/wirebot/scoreboard/events.db                 128KB │
│    /data/wirebot/scoreboard/tenants/*/events.db       528KB │
│    /data/wirebot/mem0/ (history + qdrant)             52KB  │
│    /data/wirebot/letta/                               680KB │
│    /home/wirebot/clawd/ (workspace)                   944KB │
│    /data/wirebot/users/verious/ (config)              6.6MB │
│    /data/wirebot/discovery/                           64KB  │
│    /data/wirebot/bin/ (binaries)                      24MB  │
│    /home/wirebot/wirebot-core/plugins/                6.2MB │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Standby VPS (Hetzner)

**Recommended:** Hetzner CAX11 (ARM64) — €3.79/mo (~$4/mo)
- 2 vCPU (Ampere), 4GB RAM, 40GB NVMe
- Ashburn, VA datacenter (same coast as primary)
- 99.9% uptime SLA

**Alternative:** Hetzner CX22 (x86_64) — €4.35/mo (~$5/mo)
- If ARM compatibility is a concern (Go binaries need GOARCH=arm64 build)

**OS:** AlmaLinux 8 or Ubuntu 22.04 (minimal, no cPanel needed)

```
┌─────────────────────────────────────────────────────────────┐
│  STANDBY VPS (Hetzner)                                      │
│                                                             │
│  State: WARM STANDBY                                        │
│  - Services installed but STOPPED                           │
│  - Data continuously replicated from primary                │
│  - cloudflared installed, config ready, tunnel DISCONNECTED │
│  - Activates only on failover trigger                       │
│                                                             │
│  Replicated data (real-time):                               │
│    SQLite DBs via Litestream → local replicas               │
│    Workspace + config via sync daemon → local mirror        │
│    Letta PG via pg_dump cron → hourly snapshot              │
│                                                             │
│  Pre-installed:                                             │
│    - Node.js 22 + OpenClaw (npm)                            │
│    - wirebot-scoreboard binary (Go, built for target arch)  │
│    - wirebot-memory-syncd binary                            │
│    - mem0-server.py + Python deps                           │
│    - Letta container image (podman)                         │
│    - cloudflared                                            │
│    - systemd units (identical, but disabled)                │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Sync Daemon: `wirebot-replicator`

A single Go daemon running on the **primary** VPS that handles all replication.

### Architecture

```
wirebot-replicator (Go daemon, runs on primary)
│
├── Litestream Manager
│   ├── memory-core.sqlite  → standby:/data/wirebot/.../verious.sqlite
│   ├── events.db           → standby:/data/wirebot/scoreboard/events.db
│   ├── tenant/*.db         → standby:/data/wirebot/scoreboard/tenants/*/events.db
│   ├── mem0/history.db     → standby:/data/wirebot/mem0/history.db
│   └── mem0/qdrant/*.sqlite→ standby:/data/wirebot/mem0/qdrant/...
│
├── File Sync (inotify + rsync)
│   ├── /home/wirebot/clawd/        → standby (workspace files)
│   ├── /data/wirebot/users/        → standby (config, auth-profiles)
│   ├── /data/wirebot/discovery/    → standby (projects, watermarks)
│   ├── /data/wirebot/bin/          → standby (binaries)
│   └── /home/wirebot/wirebot-core/plugins/ → standby (bridge plugin)
│
├── Letta Snapshot (cron, hourly)
│   └── pg_dump via podman exec → standby:/data/wirebot/letta/pg_dump.sql
│
├── Health Reporter
│   └── POST standby:8300/health every 30s
│       (standby knows primary is alive)
│
└── HTTP API (:8300 on primary)
    ├── GET  /status          → replication lag, last sync times
    ├── GET  /health          → daemon health
    └── POST /trigger-sync    → force immediate full sync
```

### Replication Methods

#### A. SQLite — Litestream (real-time WAL streaming)

Litestream continuously replicates SQLite WAL changes over SFTP to the standby.
- **Lag:** sub-second (WAL frames streamed as written)
- **Restore:** `litestream restore` on standby recovers to latest frame
- **DBs replicated:** 5-8 files, ~7MB total

```yaml
# /etc/litestream.yml (on primary)
dbs:
  - path: /data/wirebot/users/verious/memory/verious.sqlite
    replicas:
      - type: sftp
        host: STANDBY_IP
        path: /data/wirebot/users/verious/memory/verious.sqlite
        
  - path: /data/wirebot/scoreboard/events.db
    replicas:
      - type: sftp
        host: STANDBY_IP
        path: /data/wirebot/scoreboard/events.db

  - path: /data/wirebot/mem0/history.db
    replicas:
      - type: sftp
        host: STANDBY_IP
        path: /data/wirebot/mem0/history.db

  # Tenant DBs added dynamically when provisioned
```

**Cost:** 0. Litestream is open source. SFTP bandwidth is negligible (<1MB/day for WAL diffs).

#### B. Workspace + Config — inotify + rsync (near-real-time)

Daemon watches directories with inotify. On change, debounces 5s, then rsync diffs.

```
Watched paths:
  /home/wirebot/clawd/                    → workspace identity, checklist, memory
  /data/wirebot/users/verious/            → gateway config, auth-profiles
  /data/wirebot/discovery/                → project registry, watermarks
  /data/wirebot/bin/                      → compiled binaries
  /home/wirebot/wirebot-core/plugins/     → bridge plugin source
  /run/wirebot/                           → runtime secrets (scoreboard.env)
```

- **Lag:** 5-10 seconds after file change
- **Bandwidth:** <100KB per sync (rsync diffs only)
- **Fallback:** Full rsync every 15 minutes regardless of inotify events

#### C. Letta PostgreSQL — pg_dump hourly

```bash
# Runs hourly via daemon's internal cron
podman exec letta-wirebot pg_dumpall -U letta > /tmp/letta-dump.sql
rsync -az /tmp/letta-dump.sql standby:/data/wirebot/letta/pg_dump.sql
```

- **Lag:** up to 1 hour
- **Acceptable:** Letta holds structured business state blocks that change slowly.
  The 4 blocks (identity, soul, user, business) only update on major events.
- **Restore:** `psql < pg_dump.sql` on standby Letta container

#### D. Health Heartbeat

Primary sends heartbeat to standby every 30 seconds:
```
POST standby:8300/health
Body: { "timestamp": "...", "services": {...}, "replication_lag": {...} }
```

Standby tracks heartbeat. If **3 consecutive misses (90 seconds)**, standby
can auto-activate (optional) or send alert.

---

## 4. Failover Procedure

### Automatic (recommended for sovereign clients)

```
Primary dies
    │
    ▼ (90 seconds: 3 missed heartbeats)
Standby detects failure
    │
    ▼ (10 seconds)
Standby activates:
    1. Litestream restore (latest SQLite frames)
    2. Start all systemd services
    3. Connect cloudflared tunnel (standby has same tunnel config)
    │
    ▼ (30-60 seconds)
Cloudflare routes traffic to standby
    │
    ▼
Client never notices (CF tunnel handles DNS, no IP change needed)
```

**Total downtime: ~2 minutes**

### Manual (operator-triggered)

```bash
# On standby VPS:
ssh standby
wirebot-failover activate

# This runs:
# 1. litestream restore (all DBs)
# 2. systemctl start openclaw-gateway wirebot-scoreboard mem0-wirebot ...
# 3. cloudflared tunnel run wirebot
```

### Recovery (primary comes back)

```
Primary restored
    │
    ▼
STOP standby services (prevent split-brain)
    │
    ▼
Merge standby writes back to primary:
    - rsync standby SQLite DBs → primary (if standby took new writes)
    - Compare and merge scoreboard events (dedup by event ID)
    │
    ▼
Restart primary services
    │
    ▼
Reconnect primary tunnel (standby disconnects)
    │
    ▼
Resume normal replication
```

---

## 5. Cloudflare Tunnel Strategy

### Option A: Same tunnel, different connectors (recommended)

Both VPS boxes run `cloudflared` with the **same tunnel ID** and credentials.
Only ONE is connected at a time.

- Primary: `cloudflared-wirebot.service` (enabled, running)
- Standby: `cloudflared-wirebot.service` (installed, stopped)

On failover: start standby connector, CF routes to it.
On recovery: stop standby, primary reconnects.

**Requirement:** Copy tunnel credentials from primary to standby:
```
/root/.cloudflared/57df17a8-b9d1-4790-bab9-8157ac51641b.json
/etc/cloudflared/wirebot.yml
```

### Option B: Cloudflare Load Balancer (better but costs)

CF Load Balancer with health checks on both origins.
- Automatic failover, no manual tunnel switching
- Cost: $5/mo for CF LB

### Option C: Two tunnels, DNS failover

Two separate tunnels. Primary's DNS active. On failure, update DNS to standby.
- Slower (DNS propagation 30-300 seconds)
- No cost

**Recommendation:** Option A for now. Zero cost, <60 second failover.

---

## 6. Data Loss Matrix

| Data Type | Sync Method | Max Data Loss | Impact |
|-----------|-------------|---------------|--------|
| Scoreboard events | Litestream | ~1 second | Negligible |
| Memory-core (embeddings) | Litestream | ~1 second | Negligible |
| Mem0 facts | Litestream | ~1 second | Negligible |
| Workspace files | inotify+rsync | ~10 seconds | Low |
| Gateway config | inotify+rsync | ~10 seconds | Low |
| Discovery state | inotify+rsync | ~10 seconds | Low |
| Letta business state | pg_dump hourly | ~1 hour | Medium* |
| Active conversations | Not synced | Session lost | Low** |
| In-flight webhooks | Not synced | Events lost | Low*** |

\* Letta blocks change slowly. Hourly is adequate.
\** Client starts new conversation. No history lost (sessions synced via Litestream).
\*** Stripe/GitHub will retry failed webhooks automatically.

---

## 7. Client Instance Considerations

For sovereign client instances (`ai.clientbiz.com`):

- **Client's agent workspace** synced same as operator workspace
- **Client's scoreboard tenant DB** covered by Litestream (auto-discovered)
- **Client's domain** routes via same CF tunnel — failover is transparent
- **Client-specific secrets** in `/run/wirebot/` — synced via file sync

### Multi-client scaling

Each new client adds:
- ~1MB SQLite (tenant scoreboard)
- ~5MB workspace
- ~100KB config
- One agent entry in `openclaw.json`

The standby box comfortably handles 50+ clients before needing an upgrade.

---

## 8. Standby Setup Checklist

### One-time provisioning

```bash
# 1. Provision Hetzner CAX11 (Ashburn)
# 2. OS setup
apt update && apt install -y rsync python3 python3-pip podman

# 3. Install Node.js 22
curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt install -y nodejs

# 4. Install OpenClaw
npm install -g openclaw

# 5. Install Litestream (on PRIMARY)
curl -L https://github.com/benbjohnson/litestream/releases/latest/download/litestream-linux-amd64.tar.gz | tar xz
mv litestream /usr/local/bin/

# 6. Install cloudflared (on standby)
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /usr/local/bin/cloudflared
chmod +x /usr/local/bin/cloudflared

# 7. Copy tunnel credentials
scp /root/.cloudflared/57df17a8-*.json standby:/root/.cloudflared/
scp /etc/cloudflared/wirebot.yml standby:/etc/cloudflared/

# 8. Copy systemd units
scp /etc/systemd/system/openclaw-gateway.service standby:/etc/systemd/system/
scp /etc/systemd/system/wirebot-scoreboard.service standby:/etc/systemd/system/
scp /etc/systemd/system/mem0-wirebot.service standby:/etc/systemd/system/
# ... all wirebot-* services

# 9. Create directory structure
ssh standby 'mkdir -p /data/wirebot/{users/verious/memory,scoreboard/tenants,mem0,bin,discovery,letta}'
ssh standby 'mkdir -p /home/wirebot/{clawd,wirebot-core/plugins}'

# 10. Initial full sync
rsync -avz /data/wirebot/ standby:/data/wirebot/
rsync -avz /home/wirebot/clawd/ standby:/home/wirebot/clawd/

# 11. Start replicator daemon on primary
systemctl enable --now wirebot-replicator
```

### Ongoing (automated by replicator)

- Litestream: continuous WAL streaming
- inotify+rsync: file changes within 10 seconds
- pg_dump: hourly Letta snapshots
- Heartbeat: every 30 seconds

---

## 9. Monitoring

The replicator daemon exposes metrics:

```
GET http://localhost:8300/status
{
  "primary": true,
  "standby_reachable": true,
  "last_heartbeat": "2026-02-02T12:00:30Z",
  "replication": {
    "litestream": { "status": "streaming", "lag_ms": 450 },
    "file_sync":  { "status": "idle", "last_sync": "2026-02-02T11:59:55Z", "pending": 0 },
    "letta_dump": { "status": "ok", "last_dump": "2026-02-02T11:00:03Z" }
  },
  "standby": {
    "ip": "...",
    "services": "stopped",
    "disk_free": "34GB",
    "last_restore_test": "2026-01-26T03:00:00Z"
  }
}
```

Guardian integration: replicator reports to Guardian health checks.
Alert channels: Discord webhook, SMS (via existing infra).

---

## 10. Monthly Fire Drill

Scheduled monthly (first Sunday, 3am PT):

1. Stop primary tunnel
2. Activate standby
3. Verify all endpoints respond
4. Run test query against gateway
5. Check scoreboard data integrity
6. Deactivate standby
7. Reconnect primary
8. Log results

Automated via `wirebot-replicator drill` command.

---

## 11. Cost Summary

| Item | Monthly Cost |
|------|-------------|
| Hetzner CAX11 (standby VPS) | ~$4 |
| Bandwidth (replication) | $0 (included) |
| Litestream | $0 (open source) |
| Cloudflare tunnel | $0 (free tier) |
| **Total** | **~$4/mo** |

vs. client revenue: $2,500/mo → **0.16% of revenue on infrastructure redundancy**

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Hub-and-spoke model
- [WHITE_LABEL.md](./WHITE_LABEL.md) — Client instance frontend
- [PROVISIONING.md](./PROVISIONING.md) — Client provisioning
- [OPERATIONS.md](./OPERATIONS.md) — Gateway management
- [MONITORING.md](./MONITORING.md) — Health checks
