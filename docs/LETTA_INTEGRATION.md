# Letta Integration

> **Letta = structured business state + agent runtime.**

---

## Role

Letta stores authoritative business context:
- Stage (Idea → Launch → Growth)
- Goals + KPIs
- Checklists
- Preferences

Wirebot skills read/write Letta data per user.

---

## Per‑User Agents

Create one Letta agent per user:

```
agent_<user_id>
```

Store agent ID in WordPress user meta or provisioning registry.

### Agent Registry (example)

| user_id | letta_agent_id | mem0_namespace |
|--------|-----------------|----------------|
| user_123 | agent_user_123 | memory_user_123 |

### Webhook Payload (WP → Provisioning)

```json
{
  "user_id": "user_123",
  "email": "user@example.com",
  "tier": "power",
  "letta_agent_id": null,
  "mem0_namespace": null
}
```

---

## API Usage (example)

```bash
# create agent
curl -X POST http://letta:8283/v1/agents \
  -H "Authorization: Bearer $LETTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name":"agent_user_123","metadata":{"user_id":"user_123"}}'
```

```bash
# send message
curl -X POST http://letta:8283/v1/agents/agent_user_123/messages \
  -H "Authorization: Bearer $LETTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"role":"user","content":"Daily standup"}'
```

---

## Integration Points

- Wirebot skills call Letta for state
- Shared gateway can proxy calls
- Dedicated containers call Letta directly

---

## See Also

- [MEMORY.md](./MEMORY.md) — Full memory stack
- [PROVISIONING.md](./PROVISIONING.md) — Per-user Letta agent creation
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Where Letta fits
- [CURRENT_STATE.md](./CURRENT_STATE.md) — Letta deployment status (not yet deployed)
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — When Letta is needed (Phase 0 remaining)
