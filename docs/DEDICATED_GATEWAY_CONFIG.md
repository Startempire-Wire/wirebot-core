# Sample Config: Dedicated Gateway (Top Tier)

```json5
{
  gateway: {
    port: 18789,
    bind: "loopback",
    auth: { mode: "token", token: "<user-token>" }
  },
  skills: {
    load: { extraDirs: ["/home/wirebot/wirebot-core/skills"] }
  },
  plugins: {
    load: { paths: ["/home/wirebot/wirebot-core/plugins"] },
    slots: { memory: "memory-mem0" }
  },
  channels: {
    discord: { dm: { policy: "pairing" } },
    telegram: { dmPolicy: "pairing" },
    whatsapp: { enabled: true }
  }
}
```

## Notes

- Dedicated container uses its own `OPENCLAW_STATE_DIR`.
- Ports must be unique per user.

---

## See Also

- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [SHARED_GATEWAY_CONFIG.md](./SHARED_GATEWAY_CONFIG.md) — Multi-tenant config
- [PROVISIONING.md](./PROVISIONING.md) — Provisioning steps
- [OPERATIONS.md](./OPERATIONS.md) — Service management
- [AUTH_AND_SECRETS.md](./AUTH_AND_SECRETS.md) — Per-user auth profiles
