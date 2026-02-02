# Wirebot Resilience Architecture

> **Philosophy:** Local-first. Every surface owns its data. The hub aggregates and provides AI — it is never the sole copy of anything that matters.

---

## 1. The Principle

```
OLD (hub-centric):
  Client → thin window → HUB HAS ALL DATA → hub dies → client has NOTHING

NEW (local-first):
  Client → holds OWN data → syncs TO hub → hub dies → client keeps working
  Hub recovers → client syncs back → zero data lost
```

**Every surface is a first-class data citizen.** The browser, the extension, the CLI, the Connect Plugin overlay — each one stores what it creates and what it receives. The hub is the intelligence layer (AI inference, memory search, cross-client aggregation), not the storage layer.

The standby VPS exists for one reason: **AI availability.** Not data preservation. Local-first handles data.

---

## 2. Local-First by Surface

### Scoreboard PWA (wins.wirebot.chat)

**Storage:** IndexedDB via `idb-keyval` or Dexie.js

```
IndexedDB: wirebot-scoreboard
├── events        (all scored events, append-only)
├── feed          (activity feed cache)
├── season        (current season state)
├── projects      (project list + approval status)
├── profile       (user identity, tier, preferences)
└── sync_queue    (events created offline, pending upload)
```

**Behavior:**
- On load: render from IndexedDB immediately (instant paint, no spinner)
- Background: fetch latest from hub API, merge into IndexedDB
- On event creation (ship, intent, submit): write to IndexedDB first, queue sync
- If hub is down: full read access to all historical data, write to queue
- When hub returns: drain sync_queue, reconcile

**Conflict resolution:** Events are append-only with UUIDs. No conflicts possible — hub deduplicates by event ID.

### Chrome Extension

**Storage:** `chrome.storage.local`

```
chrome.storage.local:
├── sewn_auth      (JWT, user profile, tier, expiry)
├── wb_events      (recent scoreboard events)
├── wb_score       (current score snapshot)
├── wb_checklist   (business setup progress)
├── wb_feed        (cached activity feed)
├── wb_sync_queue  (offline actions pending sync)
└── wb_wirebot     (last Wirebot conversation context)
```

**Behavior:**
- Wirebot tab works offline with cached checklist + score
- "Ask Wirebot" queues message locally if hub is down, sends when available
- Network tab shows cached stats, marks as stale with timestamp
- Score badge updates from local data, no hub needed

### Connect Plugin Overlay (member websites)

**Storage:** `localStorage` + `sessionStorage`

```
localStorage:
├── sewn_auth        (Ring Leader JWT)
├── sewn_profile     (member profile cache)
├── sewn_network     (cached network stats, content feed)
├── sewn_scoreboard  (member's score snapshot)
└── sewn_sync_queue  (queued interactions)
```

**Behavior:**
- Overlay renders from localStorage on open (no loading state)
- Background sync to Ring Leader for fresh content
- Member interactions (content clicks, profile views) queued for analytics
- Hub down: overlay still shows cached content, member profile, score

### White-Label Client Frontend (ai.clientbiz.com)

**Storage:** IndexedDB

```
IndexedDB: wirebot-client
├── conversations  (full chat history, all sessions)
├── sessions       (session list + metadata)
├── workspace      (cached workspace state — checklist, identity)
├── skills         (skill status cache)
├── cron           (scheduled job list + last run)
├── sync_queue     (messages queued while offline)
└── config         (brand.json + runtime state)
```

**Behavior:**
- Chat history is fully local — scrollable offline
- New message: write to local, send to hub, show immediately with pending indicator
- Hub down: read all history, see workspace state, view skills/cron status
- Hub returns: queued messages send, responses stream back, local DB updates
- **Sovereign clients have COMPLETE local copy of everything**

### wb CLI (operator terminal)

**Storage:** Local filesystem (already there)

```
/data/wirebot/
├── scoreboard/events.db     (SQLite — canonical on this VPS)
├── discovery/               (project registry, watermarks)
├── users/verious/           (gateway config, memory)
└── local-journal.jsonl      (all CLI actions, always written locally first)
```

**Behavior:**
- `wb ship`, `wb intent`, `wb complete` write to local events.db AND queue to hub API
- If hub API fails, event is still in local DB — retry on next command
- `wb score` reads from local DB, no hub needed
- Discovery engine writes locally, syncs to scoreboard API asynchronously

---

## 3. Sync Protocol

### Event Sync (all surfaces → hub)

Events use **append-only log with UUIDs.** No conflicts by design.

```
Surface creates event:
  1. Generate UUID v4
  2. Write to local store (IndexedDB / chrome.storage / SQLite)
  3. POST to hub API (async, non-blocking)
  4. Hub acknowledges → mark synced locally
  5. Hub unreachable → stays in sync_queue → retry with exponential backoff

Hub processes event:
  1. Check UUID — if exists, skip (idempotent)
  2. Score the event (calcScoreDelta)
  3. Store in canonical DB
  4. Broadcast to other connected surfaces (WebSocket)
```

### State Sync (hub → surfaces)

State snapshots (score, season, projects) flow from hub to surfaces:

```
Surface requests state:
  1. Read from local store (instant)
  2. Fetch from hub API (background)
  3. If hub response is newer → update local store
  4. If hub is down → show local data with "last synced: X ago" indicator

Hub pushes state:
  1. On event scored → broadcast updated score via WebSocket
  2. Connected surfaces update local store in real-time
  3. Disconnected surfaces catch up on next fetch
```

### Conversation Sync (client frontend ↔ hub)

Chat is the one interaction that REQUIRES the hub (AI inference). Local-first handles it gracefully:

```
User sends message:
  1. Append to local conversations (IndexedDB)
  2. Show in UI immediately with "sending..." state
  3. Send to hub via WebSocket (or HTTP fallback)
  4. Hub processes → streams response tokens
  5. Each token appended to local store in real-time
  6. If hub is down:
     - Message stays in sync_queue with "queued" indicator
     - User sees full conversation history (local)
     - User sees "AI is temporarily unavailable" banner
     - When hub returns → message sends → response streams
```

### Conflict Resolution Rules

| Data Type | Strategy | Rationale |
|-----------|----------|-----------|
| Events | UUID dedup (append-only) | Events are immutable facts — no conflicts |
| Score | Hub authoritative | Score computation is server-side |
| Projects | Hub authoritative | Approval state is operator-controlled |
| Conversations | Ordered log, local-first | Messages have timestamps + sequence numbers |
| Profile/Auth | Hub authoritative (JWT) | Identity comes from Ring Leader |
| Checklist | Last-write-wins with timestamp | Single operator per business |
| Workspace files | Last-write-wins with timestamp | Files have mtime |

---

## 4. Hub Architecture (Primary VPS)

The hub's role shrinks to three functions:

1. **AI Inference** — OpenClaw gateway, model routing, tool execution
2. **Aggregation** — Cross-surface event collection, scoring, analytics
3. **Coordination** — Memory search, fact extraction, cross-client state

```
┌─────────────────────────────────────────────────────────────┐
│  PRIMARY VPS (AlmaLinux, cPanel)                            │
│  IP: 199.167.200.52                                        │
│                                                             │
│  AI Layer:                                                  │
│    openclaw-gateway.service    :18789  (inference + tools)   │
│    mem0-wirebot.service        :8200   (fact memory)        │
│    wirebot-memory-syncd        :8201   (memory bridge)      │
│    letta-wirebot (podman)      :8283   (business state)     │
│                                                             │
│  Aggregation Layer:                                         │
│    wirebot-scoreboard.service  :8100   (event store + API)  │
│                                                             │
│  Network Layer:                                             │
│    cloudflared-wirebot.service         (CF tunnel)          │
│                                                             │
│  Domains:                                                   │
│    helm.wirebot.chat  → :18789                              │
│    wins.wirebot.chat  → :8100                               │
│    api.wirebot.chat   → :8100                               │
│    ai.CLIENT.com      → :18789  (client instances)          │
│                                                             │
│  Canonical Data (~45MB):                                    │
│    memory-core.sqlite       6.4MB  (embeddings + BM25)      │
│    scoreboard events.db     128KB  (aggregated events)       │
│    tenant DBs               528KB  (client scoreboards)      │
│    mem0                     52KB   (conversation facts)      │
│    letta                    680KB  (business state blocks)   │
│    workspace                944KB  (agent identity files)    │
│    config                   6.6MB  (gateway + auth)          │
│    binaries + plugins       30MB   (runtime)                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 5. Standby VPS (AI Availability Only)

With local-first, the standby's job is simple: **keep the AI online.**

Surfaces have their data. They just need somewhere to send chat messages and receive AI responses. The standby doesn't need perfect data — it needs a working OpenClaw gateway with the right model keys and agent config.

### Standby Spec

**Hetzner CAX11 (ARM64):** €3.79/mo (~$4/mo)
- 2 vCPU (Ampere), 4GB RAM, 40GB NVMe
- Ashburn, VA datacenter

**What it runs on activation:**
- OpenClaw gateway (AI inference — the critical path)
- Scoreboard API (event ingestion — so surfaces can sync)
- cloudflared (tunnel connector)

**What it does NOT need to run:**
- Mem0 (facts are cached locally on surfaces, AI can work without history temporarily)
- Letta (business state blocks change slowly, gateway works without them)
- memory-syncd (no memory bridge needed in degraded mode)

This means the standby is **much lighter** — just the gateway + scoreboard + tunnel.

### Replication (hub → standby)

Since local-first handles client data, the standby only needs enough to run the AI:

```
wirebot-replicator (Go daemon on primary)
│
├── Litestream (real-time SQLite WAL streaming)
│   ├── events.db           → standby  (so scoreboard API can ingest)
│   ├── tenant/*.db         → standby  (client scoreboards)
│   └── memory-core.sqlite  → standby  (AI memory search)
│
├── File Sync (inotify + rsync, ~10s lag)
│   ├── gateway config      → standby  (openclaw.json, auth-profiles)
│   ├── workspace files     → standby  (agent identity)
│   ├── binaries            → standby  (scoreboard binary)
│   └── runtime secrets     → standby  (API keys)
│
└── Health Heartbeat (every 30s)
    └── POST standby:8300/heartbeat
        3 missed → standby self-activates
```

**Data loss on failover (with local-first):**

| Data Type | Loss | Why |
|-----------|------|-----|
| Client's own events | **Zero** | In their IndexedDB, sync_queue retries |
| Client's conversation history | **Zero** | In their IndexedDB |
| Client's score/profile | **Zero** | In their local store |
| AI memory (Mem0 facts) | ~1 second | Litestream |
| AI memory (Letta blocks) | **Skipped** | Not on standby, AI works without |
| Workspace files | ~10 seconds | rsync |

**vs. old hub-centric model:** every row was "up to 24 hours" or worse.

---

## 6. Failover Timeline

```
Primary dies
    │
    ▼  IMMEDIATELY
Surfaces detect API failure:
    - Show "offline" indicator
    - Continue rendering from local data
    - Queue new writes to sync_queue
    - User keeps working (read + write locally)
    │
    ▼  90 seconds (3 missed heartbeats)
Standby detects failure:
    1. Litestream restore (latest SQLite frames, ~5s)
    2. Start openclaw-gateway + scoreboard (~15s)
    3. Connect cloudflared tunnel (~10s)
    │
    ▼  ~2 minutes
Standby is live:
    - AI inference available
    - Scoreboard API accepting events
    - CF tunnel routes traffic to standby
    │
    ▼
Surfaces auto-reconnect:
    - Drain sync_queue (queued events upload)
    - Resume real-time AI chat
    - "Online" indicator returns
    - User never lost data, only waited ~2 min for AI
```

**Client experience during outage:**
- Scoreboard: fully usable, all data visible, new events queued
- Chat: "AI temporarily unavailable" banner, history visible
- Extension: cached data shown, actions queued
- After recovery: everything syncs seamlessly

---

## 7. Cloudflare Tunnel Strategy

**Option A: Same tunnel, different connectors (recommended)**

Both VPS boxes have the **same tunnel ID** and credentials.
Only ONE connector is active at a time.

- Primary: `cloudflared-wirebot.service` (enabled, running)
- Standby: `cloudflared-wirebot.service` (installed, stopped until failover)

On failover: standby starts its connector → CF routes to it.
On recovery: stop standby connector → primary reconnects.

**Zero DNS propagation.** Client's domain doesn't change.

Credentials to copy:
```
/root/.cloudflared/57df17a8-b9d1-4790-bab9-8157ac51641b.json
/etc/cloudflared/wirebot.yml
```

---

## 8. Recovery (Primary Returns)

```
Primary restored
    │
    ▼
Stop standby services + tunnel
    │
    ▼
Merge standby writes → primary:
    - Scoreboard events: dedup by UUID (append-only, no conflict)
    - Any new tenant DBs: copy over
    │
    ▼
Start primary services + tunnel
    │
    ▼
Resume normal replication (primary → standby)
    │
    ▼
Surfaces auto-reconnect to primary (CF tunnel handles routing)
    - Any remaining sync_queue items drain to primary
```

No split-brain risk because:
1. Events are UUID-keyed and append-only
2. Only one tunnel connector is active at a time
3. Surfaces sync to whichever hub is reachable — the data is the same

---

## 9. Implementation Per Surface

### Scoreboard PWA Changes

```javascript
// lib/localStore.js — new module
import { openDB } from 'idb';

const db = await openDB('wirebot-scoreboard', 1, {
  upgrade(db) {
    db.createObjectStore('events', { keyPath: 'id' });
    db.createObjectStore('state',  { keyPath: 'key' });
    db.createObjectStore('sync_queue', { keyPath: 'id', autoIncrement: true });
  }
});

// Write-local-first pattern
export async function submitEvent(event) {
  event.id = event.id || crypto.randomUUID();
  event.synced = false;
  await db.put('events', event);

  try {
    await fetch('/v1/events', {
      method: 'POST',
      headers: { ...authHeaders(), 'Content-Type': 'application/json' },
      body: JSON.stringify(event)
    });
    event.synced = true;
    await db.put('events', event);
  } catch (e) {
    // Hub down — event is safe in IndexedDB, will retry
    await db.put('sync_queue', { eventId: event.id, payload: event });
  }
}

// Background sync drain
export async function drainSyncQueue() {
  const queue = await db.getAll('sync_queue');
  for (const item of queue) {
    try {
      await fetch('/v1/events', {
        method: 'POST',
        headers: { ...authHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(item.payload)
      });
      await db.delete('sync_queue', item.id);
    } catch (e) {
      break; // Hub still down, stop draining
    }
  }
}

// Read-local-first pattern
export async function getScore() {
  const local = await db.get('state', 'score');

  try {
    const res = await fetch('/v1/scoreboard', { headers: authHeaders() });
    const remote = await res.json();
    await db.put('state', { key: 'score', ...remote, lastSync: Date.now() });
    return { data: remote, source: 'live' };
  } catch (e) {
    return { data: local, source: 'cache', stale: true };
  }
}
```

### Chrome Extension Changes

```javascript
// services/localCache.js — new module
const KEYS = ['wb_events', 'wb_score', 'wb_checklist', 'wb_feed', 'wb_sync_queue'];

export async function cacheScore(score) {
  await chrome.storage.local.set({ wb_score: { ...score, cachedAt: Date.now() } });
}

export async function getCachedScore() {
  const { wb_score } = await chrome.storage.local.get('wb_score');
  return wb_score;
}

export async function queueAction(action) {
  const { wb_sync_queue = [] } = await chrome.storage.local.get('wb_sync_queue');
  wb_sync_queue.push({ ...action, id: crypto.randomUUID(), queuedAt: Date.now() });
  await chrome.storage.local.set({ wb_sync_queue });
}

// Background script: periodic sync drain
chrome.alarms.create('sync-drain', { periodInMinutes: 1 });
chrome.alarms.onAlarm.addListener(async (alarm) => {
  if (alarm.name === 'sync-drain') {
    const { wb_sync_queue = [] } = await chrome.storage.local.get('wb_sync_queue');
    // ... drain to hub API, remove succeeded items
  }
});
```

### White-Label Client Frontend

```javascript
// chat.js — local-first conversation
async function sendMessage(text) {
  const msg = {
    id: crypto.randomUUID(),
    role: 'user',
    content: text,
    timestamp: Date.now(),
    synced: false
  };

  // 1. Write locally (instant)
  await db.put('conversations', msg);
  renderMessage(msg);

  // 2. Send to hub
  try {
    const ws = getWebSocket();
    ws.send(JSON.stringify({ method: 'chat.send', params: { message: text } }));
    msg.synced = true;
    await db.put('conversations', msg);
  } catch (e) {
    // Hub down — show queued indicator
    showOfflineBanner();
    await db.put('sync_queue', { msgId: msg.id, payload: msg });
  }
}

// On page load: render from local, then background refresh
async function init() {
  const localHistory = await db.getAll('conversations');
  renderConversation(localHistory); // instant, no spinner

  try {
    // Fetch latest from hub to catch up
    const remote = await rpc('chat.history');
    await mergeConversation(localHistory, remote);
  } catch (e) {
    showStaleIndicator();
  }
}
```

---

## 10. Offline UX Patterns

Every surface follows the same visual language:

| State | Indicator | Behavior |
|-------|-----------|----------|
| **Online** | Green dot / no indicator | Real-time sync, live AI |
| **Syncing** | Amber pulse | Draining queue, catching up |
| **Offline** | Red dot + "Last synced: X ago" | Full local data, writes queued |
| **AI unavailable** | Banner: "AI is temporarily offline" | History visible, new messages queued |
| **Reconnected** | Brief green flash + "Back online" | Queue drained, indicators clear |

Queued events show a subtle clock icon. When synced, the icon disappears. No user action required.

---

## 11. Standby Setup

### Provisioning

```bash
# 1. Hetzner CAX11, Ashburn datacenter (~$4/mo)

# 2. Minimal OS setup (no cPanel needed)
apt update && apt install -y rsync nodejs cloudflared

# 3. Install OpenClaw
npm install -g openclaw

# 4. Copy tunnel credentials
scp /root/.cloudflared/57df17a8-*.json standby:/root/.cloudflared/
scp /etc/cloudflared/wirebot.yml standby:/etc/cloudflared/

# 5. Copy systemd units (gateway + scoreboard only)
scp /etc/systemd/system/openclaw-gateway.service standby:/etc/systemd/system/
scp /etc/systemd/system/wirebot-scoreboard.service standby:/etc/systemd/system/

# 6. Create directory structure
ssh standby 'mkdir -p /data/wirebot/{users/verious/memory,scoreboard/tenants,bin}'
ssh standby 'mkdir -p /home/wirebot/clawd'

# 7. Initial full sync
rsync -avz /data/wirebot/ standby:/data/wirebot/
rsync -avz /home/wirebot/clawd/ standby:/home/wirebot/clawd/

# 8. Install Litestream on primary
curl -L https://github.com/benbjohnson/litestream/releases/latest/... | tar xz
mv litestream /usr/local/bin/

# 9. Start replicator on primary
systemctl enable --now wirebot-replicator
```

### Lighter than before

With local-first, the standby box doesn't need:
- ❌ Mem0 (facts cached on surfaces)
- ❌ Letta + PostgreSQL (business state is non-critical for degraded mode)
- ❌ memory-syncd (no memory bridge in failover)
- ❌ Full data replication (surfaces hold their own data)

Just: **OpenClaw gateway + Scoreboard API + cloudflared.** That's it.

RAM requirement drops from ~4GB to ~1-2GB. The cheapest Hetzner box is overkill.

---

## 12. Replicator Daemon Spec

```
wirebot-replicator (Go binary, ~10MB, runs on primary)
│
├── Litestream Manager
│   ├── Wraps litestream process lifecycle
│   ├── Watches /data/wirebot/scoreboard/tenants/ for new tenant DBs
│   ├── Auto-adds new DBs to replication
│   └── Reports lag per DB
│
├── File Sync
│   ├── inotify on watched paths
│   ├── Debounce 5s → rsync diff to standby
│   ├── Full sync every 15 min (safety net)
│   └── Paths:
│       ├── /data/wirebot/users/verious/  (config)
│       ├── /home/wirebot/clawd/          (workspace)
│       ├── /data/wirebot/bin/            (binaries)
│       └── /run/wirebot/                 (secrets)
│
├── Health Heartbeat
│   ├── POST standby:8300/heartbeat every 30s
│   └── Includes: timestamp, service status, replication lag
│
├── HTTP API (:8300)
│   ├── GET  /status         (replication state)
│   ├── GET  /health         (daemon health)
│   ├── POST /trigger-sync   (force full sync now)
│   └── POST /drill          (failover fire drill)
│
└── Config: /etc/wirebot/replicator.yml
    standby:
      host: <hetzner-ip>
      user: root
      ssh_key: /root/.ssh/wirebot-standby
    litestream:
      dbs:
        - /data/wirebot/users/verious/memory/verious.sqlite
        - /data/wirebot/scoreboard/events.db
        - /data/wirebot/mem0/history.db
      tenant_dir: /data/wirebot/scoreboard/tenants
    file_sync:
      paths:
        - /data/wirebot/users/verious
        - /home/wirebot/clawd
        - /data/wirebot/bin
        - /run/wirebot
      debounce_seconds: 5
      full_sync_minutes: 15
    heartbeat:
      interval_seconds: 30
      miss_threshold: 3
```

---

## 13. Data Loss Summary (Local-First vs Hub-Centric)

| Data Type | Hub-Centric Loss | Local-First Loss |
|-----------|-----------------|------------------|
| Client's scoreboard events | Up to 24 hours | **Zero** |
| Client's conversation history | Up to 24 hours | **Zero** |
| Client's score/streak | Up to 24 hours | **Zero** |
| Client's workspace state | Up to 10 seconds | **Zero** |
| AI memory (embeddings) | ~1 second | ~1 second (hub-side only) |
| AI memory (facts) | ~1 second | ~1 second (hub-side only) |
| AI business state (Letta) | ~1 hour | **Skipped** (not on standby) |
| Active AI response | Lost | **Queued** (resends on recovery) |

Local-first turns "data loss" into "AI pause." The client's data is always safe. They just wait ~2 minutes for the AI to come back.

---

## 14. Monthly Fire Drill

First Sunday, 3am PT. Automated via `wirebot-replicator drill`:

1. Stop primary tunnel connector
2. Verify surfaces show "offline" indicator (not error)
3. Activate standby (gateway + scoreboard + tunnel)
4. Verify all endpoints respond
5. Send test chat message → receive AI response
6. Submit test event → verify in scoreboard
7. Deactivate standby
8. Reconnect primary
9. Verify surfaces reconnect and drain queues
10. Log results to `/var/log/wirebot/fire-drill.log`
11. Alert operator with summary (Discord webhook)

---

## 15. Cost Summary

| Item | Monthly Cost |
|------|-------------|
| Hetzner CAX11 (standby) | ~$4 |
| Litestream | $0 (open source) |
| Bandwidth | $0 (included, <1MB/day) |
| Cloudflare tunnel | $0 (free tier) |
| **Total** | **~$4/mo** |

Client revenue: $2,500/mo → **0.16% on infrastructure redundancy**

Client data loss on worst-case failure: **Zero.**
AI downtime on worst-case failure: **~2 minutes.**

---

## See Also

- [ARCHITECTURE.md](./ARCHITECTURE.md) — Hub-and-spoke model
- [WHITE_LABEL.md](./WHITE_LABEL.md) — Client instance frontend
- [PROVISIONING.md](./PROVISIONING.md) — Client provisioning
- [OPERATIONS.md](./OPERATIONS.md) — Gateway management
- [MONITORING.md](./MONITORING.md) — Health checks
