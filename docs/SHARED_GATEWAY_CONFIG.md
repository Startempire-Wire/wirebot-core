# Sample Config: Shared Gateway

```json5
{
  gateway: {
    port: 18789,
    bind: "loopback",
    auth: { mode: "token", token: "<shared-token>" }
  },
  skills: {
    load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] }
  },
  plugins: {
    load: { paths: ["/home/wirebot/wirebot-core/plugins"] },
    slots: { memory: "memory-mem0" }
  },
  agents: {
    list: [
      { id: "user_1", name: "Wirebot: user_1" },
      { id: "user_2", name: "Wirebot: user_2" }
    ]
  },
  bindings: [
    { agentId: "user_1", match: { channel: "discord", peer: { kind: "dm", id: "123" } } },
    { agentId: "user_2", match: { channel: "telegram", peer: { kind: "dm", id: "456" } } }
  ],
  channels: {
    discord: { dm: { policy: "pairing" } },
    telegram: { dmPolicy: "pairing" }
  }
}
```

## Notes

- `bindings` must include channel + peer id.
- Default DM policy is pairing; approve via `clawdbot pairing approve`.

---

## See Also

- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [DEDICATED_GATEWAY_CONFIG.md](./DEDICATED_GATEWAY_CONFIG.md) — Per-user config
- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Infrastructure models
- [CAPABILITIES.md](./CAPABILITIES.md) — Channel availability per tier
