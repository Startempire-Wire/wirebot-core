# Mem0 Plugin (memory-mem0)

> **Optional memory slot for Mem0 (does not replace Clawdbot memory by default).**

---

## Role

Clawdbot already includes a full memory system (Markdown + hybrid search). The Mem0
plugin is **additive** and is mainly for:

- Browser sync (OpenMemory → Mem0 → Wirebot)
- Cross‑surface memory sharing
- Vector/graph memory outside the gateway

If you want Clawdbot’s memory tools to still use its built‑in memory, keep
`memory-core` as the slot and call Mem0 via Wirebot skills.

---

## Status

Skeleton plugin added:

```
/home/wirebot/wirebot-core/plugins/memory-mem0
```

Tools:
- `memory_recall`
- `memory_store`
- `memory_forget`

---

## Config (clawdbot.json)

```json5
plugins: {
  load: { paths: ["/home/wirebot/wirebot-core/plugins"] },
  slots: { memory: "memory-mem0" },
  entries: {
    "memory-mem0": {
      config: {
        baseUrl: "http://localhost:8080",
        apiKey: "${MEM0_API_KEY}",
        namespacePrefix: "wirebot_",
        endpoints: {
          search: "/v1/search",
          store: "/v1/store",
          delete: "/v1/delete"
        }
      }
    }
  }
}
```

**Default recommendation:** keep `memory-core` and use Mem0 via skills unless you
explicitly want memory tools to point at Mem0.

---

## Expected Mem0 API Contract

```http
POST /v1/search
{ "query": "...", "namespace": "wirebot_user_123", "limit": 5 }

POST /v1/store
{ "text": "...", "category": "decision", "namespace": "wirebot_user_123" }

POST /v1/delete
{ "id": "<memory-id>", "namespace": "wirebot_user_123" }
```

---

## Browser Sync Flow (Primary Use‑Case)

```
ChatGPT/Claude (browser)
  → OpenMemory extension
  → Mem0 server
  → Wirebot skill (fetch + store)
  → Clawdbot agent context
```

---

## TODO

- Auto‑capture hook
- Auto‑recall hook
- Result formatting

---

## See Also

- [MEMORY.md](./MEMORY.md) — Full memory stack
- [ARCHITECTURE.md](./ARCHITECTURE.md) — System architecture
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Mem0 deployment status (not yet deployed)
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — When Mem0 is needed (Phase 0 remaining)
- [LETTA_INTEGRATION.md](./LETTA_INTEGRATION.md) — Structured state companion
