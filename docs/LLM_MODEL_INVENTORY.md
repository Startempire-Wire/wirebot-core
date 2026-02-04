# LLM Model Inventory

> Every hardcoded or configured LLM model reference across the Wirebot stack.
> Use this to audit costs, change providers, or debug routing.
>
> Last updated: 2026-02-04

---

## Active Model References

### 1. Gateway — Primary Chat (Discord, tools, agent)

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/users/verious/openclaw.json` |
| **Path** | `agents.defaults.model` |
| **Model** | `kimi-coding/k2p5` (primary) → `zai/glm-4.7` (fallback) |
| **Cost** | Free (Kimi free tier, GLM free tier) |
| **Provider** | Kimi API, ZhipuAI API (keys in `/run/wirebot/gateway.env`) |

### 2. Gateway — Heartbeat / Background Tasks

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/users/verious/openclaw.json` |
| **Path** | `agents.defaults.heartbeat.model` |
| **Model** | `kimi/moonshot-v1-auto` |
| **Cost** | Free |

### 3. Gateway — Memory Search Embeddings

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/users/verious/openclaw.json` |
| **Path** | `agents.defaults.memorySearch.remote` |
| **Model** | `openai/text-embedding-3-small` (via gateway proxy) |
| **Base URL** | `http://127.0.0.1:18789/v1/` |
| **Cost** | Routed through gateway (provider-dependent) |

### 4. Scoreboard — Memory Extraction (vault + conversation)

| Field | Value |
|-------|-------|
| **File** | `/home/wirebot/wirebot-core/cmd/scoreboard/memory_extract.go` |
| **Function** | `callLLMForExtraction()` (line ~624) |
| **Model** | `envOr("EXTRACTION_MODEL", "kimi-coding/k2p5")` |
| **Route** | Local gateway `http://127.0.0.1:18789/v1/chat/completions` |
| **Auth** | `GATEWAY_TOKEN` env → fallback `SCOREBOARD_TOKEN` |
| **Cost** | Free (Kimi) |
| **Override** | Set `EXTRACTION_MODEL` env var to change |

### 5. Scoreboard — Draft Generation (defer action engine)

| Field | Value |
|-------|-------|
| **File** | `/home/wirebot/wirebot-core/cmd/scoreboard/main.go` |
| **Function** | `generateDraftForTask()` (line ~6422) |
| **Model** | `kimi-coding/k2p5` (hardcoded) |
| **Route** | Local gateway `http://127.0.0.1:18789/v1/chat/completions` |
| **Auth** | `GATEWAY_TOKEN` → fallback `SCOREBOARD_TOKEN` |
| **Cost** | Free |
| **⚠️ Note** | Hardcoded — should use `envOr()` like extraction does |

### 6. Mem0 — Fact Extraction LLM

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/bin/mem0-server.py` |
| **Section** | `CONFIG.llm` (line ~33) |
| **Model** | `kimi/moonshot-v1-auto` |
| **Base URL** | `http://127.0.0.1:18789/v1` (local gateway) |
| **Auth** | `GATEWAY_TOKEN` env (default `88f4cdab-...`) |
| **Cost** | Free |
| **⚠️ Note** | Mem0 Python lib checks `OPENROUTER_API_KEY` env first and ignores config if set. Script pops that var before import to force local gateway usage. |

### 7. Mem0 — Embedding Model

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/bin/mem0-server.py` |
| **Section** | `CONFIG.embedder` (line ~37) |
| **Model** | `BAAI/bge-base-en-v1.5` via fastembed |
| **Dimensions** | 768 |
| **Cost** | Free (runs locally, no API calls) |

### 8. Letta Agent — LLM ✅ Fixed

| Field | Value |
|-------|-------|
| **Location** | PostgreSQL `letta` database → `agents` table → `llm_config` column |
| **Agent** | `wirebot_verious` |
| **Model** | `kimi-coding/k2p5` |
| **Endpoint** | `http://127.0.0.1:18789/v1` |
| **Cost** | **BROKEN** — OpenRouter balance is $0 |
| **Status** | ✅ Active — routing through local gateway at $0/month |
| **Fix needed** | Update agent config in PostgreSQL to use local gateway, or switch model |

### 9. Letta Agent — Embeddings ✅ Fixed

| Field | Value |
|-------|-------|
| **Location** | PostgreSQL `letta` database → `agents` table → `embedding_config` column |
| **Agent** | `wirebot_verious` |
| **Model** | `letta/letta-free` |
| **Endpoint** | `http://127.0.0.1:18789/v1` |
| **Dimensions** | 1536 |
| **Cost** | **BROKEN** — OpenRouter balance is $0 |
| **Fix needed** | Switch to local fastembed or route through gateway |

---

## Startup Scripts with Model/Provider References

### 10. Letta Container Launch

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/bin/letta-container.sh` |
| **Line** | 15 |
| **Sets** | `OPENAI_API_KEY=${OPENROUTER_API_KEY}`, `OPENAI_BASE_URL=https://openrouter.ai/api/v1` |
| **⚠️ Status** | Points to OpenRouter ($0 balance). Needs update to local gateway. |

### 11. Letta Server Launch

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/bin/letta-server.sh` |
| **Line** | 12-13 |
| **Sets** | `OPENAI_API_KEY="${OPENROUTER_API_KEY}"`, `OPENAI_BASE_URL="https://openrouter.ai/api/v1"` |
| **⚠️ Status** | Same as above — OpenRouter dead. |

### 12. Secret Injector

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/bin/inject-gateway-secrets.sh` |
| **Purpose** | Pulls API keys from `rbw` vault and writes `/run/wirebot/gateway.env` |
| **Keys injected** | `OPENROUTER_API_KEY`, `KIMI_API_KEY`, `ZAI_API_KEY` |
| **Note** | OpenRouter key is still injected (used as last-resort fallback by gateway). Not harmful since gateway primary/fallback models don't route through OpenRouter anymore. |

---

## Legacy / Inactive References

### 13. openclaw.json (DEAD — replaced by openclaw.json)

| Field | Value |
|-------|-------|
| **File** | `/data/wirebot/users/verious/openclaw.json` |
| **Status** | **Not loaded by any service.** Legacy config from openclaw era. |
| **Models** | `kimi-coding/k2p5` → `zai/glm-4.7` → `openrouter/openrouter/auto` |
| **memorySearch** | `https://openrouter.ai/api/v1/` |
| **Action** | Safe to delete or archive. |

### 14. MEMORY.md Stale Runtime Reference

| Field | Value |
|-------|-------|
| **File** | `/home/wirebot/clawd/MEMORY.md` |
| **Line** | 9 |
| **Content** | `Runtime: OpenClaw v2026.1.24-3 on Node v22.22.0` |
| **Action** | Update to `OpenClaw v2026.2.2-3` |

---

## Classification Logic (Not a Model, But Relevant)

### 15. Verification Level by Source

| Field | Value |
|-------|-------|
| **File** | `/home/wirebot/wirebot-core/cmd/scoreboard/main.go` |
| **Function** | `postEvent()` (line ~1162) |
| **Logic** | Agent names (`claude`, `pi`, `letta`, `opencode`) → verification level `WEAK` |
| **Note** | These are source labels, not model names. Used for anti-gaming trust levels. |

---

## Environment Variables That Control Models

| Var | Source | Used By | Current Value |
|-----|--------|---------|---------------|
| `EXTRACTION_MODEL` | scoreboard service env | `memory_extract.go` | unset → default `kimi-coding/k2p5` |
| `GATEWAY_TOKEN` | `/run/wirebot/gateway.env` or service env | mem0, scoreboard | `88f4cdab-357a-464f-b68d-ebec3ddd2531` |
| `GATEWAY_URL` | scoreboard service env | `memory_extract.go` | unset → default `http://127.0.0.1:18789` |
| `OPENROUTER_API_KEY` | `/run/wirebot/gateway.env` | gateway (last fallback), ⚠️ Mem0 (popped) | `sk-or-v1-a2b...` ($0 balance) |
| `KIMI_API_KEY` | `/run/wirebot/gateway.env` | gateway (Kimi provider) | Set (from rbw vault) |
| `ZAI_API_KEY` | `/run/wirebot/gateway.env` | gateway (GLM provider) | Set (from rbw vault) |
| `OPENAI_API_KEY` | letta scripts | Letta agent | Gateway token (local) ✅ |
| `OPENAI_BASE_URL` | letta scripts | Letta agent | `http://127.0.0.1:18789/v1` ✅ |

---

## Cost Summary

| Component | Model | Provider | Cost |
|-----------|-------|----------|------|
| Gateway chat | kimi-coding/k2p5 | Kimi | **Free** |
| Gateway fallback | zai/glm-4.7 | ZhipuAI | **Free** |
| Gateway heartbeat | kimi/moonshot-v1-auto | Kimi | **Free** |
| Mem0 fact extraction | kimi/moonshot-v1-auto | Kimi (via gateway) | **Free** |
| Mem0 embeddings | BAAI/bge-base-en-v1.5 | Local fastembed | **Free** |
| Scoreboard extraction | kimi-coding/k2p5 | Kimi (via gateway) | **Free** |
| Scoreboard drafts | kimi-coding/k2p5 | Kimi (via gateway) | **Free** |
| Letta LLM | kimi-coding/k2p5 | Gateway | ✅ $0 (free) |
| Letta embeddings | letta/letta-free | Local (built-in) | ✅ $0 (free) |
| **Total monthly** | | | **$0** (all services active) |

---

## Action Items

1. ~~**Fix Letta**~~ ✅ Done — kimi-coding/k2p5 via gateway, letta/letta-free for embeddings
2. ~~**Hardcoded model in `generateDraftForTask()`**~~ ✅ Done — `envOr("DRAFT_MODEL", "kimi-coding/k2p5")`
3. ~~**Update MEMORY.md**~~ ✅ Done — OpenClaw → OpenClaw throughout
4. **Archive openclaw.json** — No longer loaded by anything
5. **Monitor OpenRouter** — Key still injected by `inject-gateway-secrets.sh` but nothing actively uses it

---

## How to Change a Model

**Gateway primary/fallback:**
```bash
# Edit /data/wirebot/users/verious/openclaw.json
# Change agents.defaults.model.primary and .fallbacks[]
# Restart: systemctl restart openclaw-gateway
```

**Scoreboard extraction:**
```bash
# Set env var in service or /run/wirebot/scoreboard.env
EXTRACTION_MODEL=zai/glm-4.7
# Rebuild: cd wirebot-core/cmd/scoreboard && go build -o /data/wirebot/bin/wirebot-scoreboard .
# Restart: systemctl restart wirebot-scoreboard
```

**Mem0 LLM:**
```bash
# Edit /data/wirebot/bin/mem0-server.py → CONFIG.llm.config.model
# Restart: systemctl restart mem0-wirebot
```

**Letta agent (when fixed):**
```sql
-- In PostgreSQL letta database:
UPDATE agents SET llm_config = jsonb_set(llm_config, '{model}', '"new-model-name"') WHERE name = 'wirebot_verious';
UPDATE agents SET llm_config = jsonb_set(llm_config, '{model_endpoint}', '"http://127.0.0.1:18789/v1"') WHERE name = 'wirebot_verious';
```
