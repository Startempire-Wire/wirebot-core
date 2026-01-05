# Wirebot Gateway Specification

> **The single brain that owns all intelligence.**

---

## Overview

The Wirebot Gateway is a containerized TypeScript service that:
- Owns all sessions and memory
- Manages scheduling and cron jobs
- Routes requests to AI providers
- Enforces trust modes at runtime
- Applies surface-aware policies

---

## Technology Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| Runtime | Node 20 (TypeScript) | Modern, async-native |
| Framework | Fastify or Hono | Fast, WS-friendly |
| Cache | Redis 7.2 | Sessions, rate limits |
| Database | MariaDB 10.6 | Structured state, transcripts |
| Container | Podman | Rootless, OCI-compliant |
| Reverse Proxy | LiteSpeed | Already configured |
| Vector Store | Qdrant (optional, later) | Semantic recall |

---

## API Surfaces

### HTTP Endpoints

```
POST /chat                    # Main chat endpoint
POST /sms/inbound             # Twilio webhook
POST /voice/inbound           # Twilio voice webhook
GET  /health                  # Health check
GET  /session/:id             # Session state (authenticated)
POST /workspace/:id/sync      # Force workspace sync
```

### WebSocket

```
ws://api.wirebot.chat/ws

Events:
- connect (auth via JWT)
- message (user input)
- response (streaming tokens)
- heartbeat
- disconnect
```

---

## Core Components

### 1. Session Manager

Owns conversation state and context windows.

```typescript
interface Session {
  id: string;
  founderId: string;
  workspaceId: string;
  trustMode: 0 | 1 | 2 | 3;
  surface: 'web' | 'extension' | 'sms' | 'discord';
  context: Message[];
  createdAt: Date;
  lastActiveAt: Date;
}
```

Storage: Redis (`wb:session:{id}`)
TTL: 24 hours (configurable)

### 2. Memory Store

Layered memory architecture.

| Layer | Storage | Purpose |
|-------|---------|---------|
| Structured State | MariaDB | Business profile, stage, goals |
| Rolling Summaries | MariaDB | Weekly ops, blockers, plan |
| Transcripts | MariaDB | Full conversation logs |
| Vector (optional) | Qdrant | Semantic recall |

### 3. Scheduler (Cron)

Handles automated prompts and maintenance.

```typescript
interface ScheduledJob {
  id: string;
  founderId: string;
  type: 'standup' | 'eod' | 'weekly' | 'maintenance';
  schedule: string; // cron expression
  enabled: boolean;
  lastRun: Date;
  nextRun: Date;
}
```

Jobs:
- Daily standup prompt (configurable time)
- End-of-day reflection
- Weekly planning prompt
- Monthly recalibration
- Rolling summary generation

### 4. Tool Registry

Managed tool access by trust mode.

```typescript
interface Tool {
  name: string;
  description: string;
  minTrustMode: 0 | 1 | 2 | 3;
  handler: (input: unknown) => Promise<unknown>;
}
```

Tools are registered and gated by trust level.

### 5. Provider Router

Routes to AI providers (Claude, etc).

- Primary: Anthropic Claude API
- Fallback: Configurable
- Rate limiting: Per-founder, per-surface
- Cost tracking: Per-workspace

### 6. Trust Enforcer

Runtime enforcement of trust boundaries.

- Validates JWT on every request
- Checks operation against `trust_mode_max`
- Applies surface-specific restrictions
- Logs all access attempts

---

## Authentication Flow

```
1. Request arrives with JWT in Authorization header
2. Gateway validates signature (shared secret or JWKS)
3. Gateway extracts claims:
   - user_id
   - workspace_id
   - trust_mode_max
   - scopes
   - surfaces
4. Gateway checks:
   - Token not expired
   - Surface allowed
   - Scope includes requested operation
5. Gateway creates/resumes session with trust ceiling
6. All operations checked against ceiling
```

---

## Database Schema (MariaDB)

```sql
-- Workspaces (business context)
CREATE TABLE workspaces (
  id VARCHAR(36) PRIMARY KEY,
  founder_id VARCHAR(36) NOT NULL,
  name VARCHAR(255),
  stage ENUM('idea', 'launch', 'growth'),
  profile JSON,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Checklists
CREATE TABLE checklists (
  id VARCHAR(36) PRIMARY KEY,
  workspace_id VARCHAR(36) NOT NULL,
  title VARCHAR(255),
  items JSON,
  priority INT,
  progress DECIMAL(5,2),
  due_date DATE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Goals
CREATE TABLE goals (
  id VARCHAR(36) PRIMARY KEY,
  workspace_id VARCHAR(36) NOT NULL,
  title VARCHAR(255),
  target_value DECIMAL(15,2),
  current_value DECIMAL(15,2),
  unit VARCHAR(50),
  deadline DATE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Summaries (rolling memory)
CREATE TABLE summaries (
  id VARCHAR(36) PRIMARY KEY,
  workspace_id VARCHAR(36) NOT NULL,
  type ENUM('weekly', 'blockers', 'plan', 'decisions'),
  content TEXT,
  generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Transcripts (audit log)
CREATE TABLE transcripts (
  id VARCHAR(36) PRIMARY KEY,
  session_id VARCHAR(36) NOT NULL,
  workspace_id VARCHAR(36) NOT NULL,
  surface VARCHAR(50),
  messages JSON,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Scheduled Jobs
CREATE TABLE scheduled_jobs (
  id VARCHAR(36) PRIMARY KEY,
  founder_id VARCHAR(36) NOT NULL,
  type VARCHAR(50),
  schedule VARCHAR(100),
  enabled BOOLEAN DEFAULT TRUE,
  last_run TIMESTAMP,
  next_run TIMESTAMP
);
```

---

## Redis Key Structure

```
wb:session:{session_id}         # Session state (JSON)
wb:rate:{founder_id}:{surface}  # Rate limit counter
wb:lock:{resource}              # Distributed locks
wb:cache:{key}                  # General cache
wbs:*                           # Mode 3 (Sovereign) namespace
```

---

## Container Configuration

### Dockerfile

```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --only=production
COPY dist ./dist
EXPOSE 8100
USER node
CMD ["node", "dist/index.js"]
```

### Podman Run

```bash
podman run -d \
  --name wirebot-gateway \
  --network host \
  -e DATABASE_URL="mysql://wirebot:pass@localhost/wirebot_prod" \
  -e REDIS_URL="redis://localhost:6379" \
  -e JWT_SECRET="..." \
  -e ANTHROPIC_API_KEY="..." \
  -p 127.0.0.1:8100:8100 \
  wirebot-gateway:latest
```

---

## Monitoring

- Health endpoint: `/health`
- Metrics: Prometheus-compatible (optional)
- Logs: Structured JSON to stdout
- Alerts: Via existing server monitoring

---

## Security Considerations

1. **No direct public access** — Always behind LiteSpeed proxy
2. **JWT validation on every request** — No exceptions
3. **Trust ceiling enforcement** — Runtime checks
4. **Input sanitization** — All user input validated
5. **Rate limiting** — Per-founder, per-surface
6. **Audit logging** — All operations logged
7. **Secrets management** — Environment variables, not config files
