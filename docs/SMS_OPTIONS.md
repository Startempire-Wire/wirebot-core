# SMS Options (No A2P 10DLC)

> **Supported plan: Android node, iMessage SMS, email‑to‑SMS.**

---

## 1) Android Node (sms.send)

OpenClaw nodes can expose `sms.send` when an Android device has telephony.

**Pros:** reliable, no A2P 10DLC, direct SIM.
**Cons:** requires dedicated device.

**Flow:**
- Pair Android node
- Enable SMS permission
- Use tool: `sms.send`

**Pairing commands:**
```bash
openclaw node register
openclaw nodes list
```

**Example (CLI):**
```bash
# Send SMS via paired Android node
openclaw nodes invoke --node <idOrNameOrIp> \
  --command sms.send \
  --params '{"to":"+15555550123","message":"Hello from Wirebot"}'
```

---

## 2) iMessage SMS (macOS channel)

OpenClaw iMessage channel can send **SMS fallback** via a paired iPhone.

**Pros:** no A2P 10DLC, uses Apple stack.
**Cons:** requires Mac + iPhone.

**Config example (openclaw.json):**
```json5
{
  channels: {
    imessage: {
      enabled: true,
      service: "sms", // or "auto"
      dmPolicy: "allowlist",
      allowFrom: ["+15555550123", "*"],
      cliPath: "/usr/local/bin/imsg"
    }
  }
}
```

**CLI requirement:**
- `imsg` binary must be installed and accessible at `cliPath`.

---

## 3) Email‑to‑SMS (Carrier Gateways)

Use carrier email gateways (e.g., `number@vtext.com`).

**Pros:** no A2P 10DLC.
**Cons:** unreliable, often blocked, no delivery guarantees.

---

## 4) Optional: Toll‑Free SMS

Toll‑free numbers can bypass **A2P 10DLC**, but still require verification and filtering.

Use later if Android/iMessage are insufficient.

---

## Notes

- Twilio long‑code requires A2P 10DLC (avoid if possible).
- If future provider used, build a provider‑specific plugin.

---

## See Also

- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [CAPABILITIES.md](./CAPABILITIES.md) — Feature matrix (SMS per tier)
- [LAUNCH_ORDER.md](./LAUNCH_ORDER.md) — Rollout roadmap
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Node architecture for SMS
