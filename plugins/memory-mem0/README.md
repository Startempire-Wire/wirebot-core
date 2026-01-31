# Memory (Mem0) Plugin — Skeleton

Clawdbot memory plugin that proxies to a Mem0 server.

**Status:** skeleton — endpoints must be configured.

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

## Tools

- `memory_recall`
- `memory_store`
- `memory_forget`

Each tool sends POST requests to configured endpoints.

## Notes

- Endpoints are **placeholders**. Set to Mem0 server API paths.
- Namespace uses `wirebot_<agentId>` by default.
