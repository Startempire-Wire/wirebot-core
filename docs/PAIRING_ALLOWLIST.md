# Pairing + Allowlist

> **OpenClaw default DM policy = pairing.**

---

## Where Pairing Lives

Pairing + allowlist files are stored in the credentials dir:

```
~/.openclaw/credentials/
  telegram-pairing.json
  telegram-allowFrom.json
  discord-pairing.json
  discord-allowFrom.json
  imessage-pairing.json
  imessage-allowFrom.json
```

Path resolves via:
- `OPENCLAW_STATE_DIR/credentials` (preferred)
- `~/.openclaw/credentials`

---

## CLI Commands

```bash
# list pending pairing
openclaw pairing list --channel telegram

# approve
openclaw pairing approve telegram ABCD1234

# reject
openclaw pairing reject telegram ABCD1234
```

---

## Open Policy (Use Carefully)

To bypass pairing (not recommended for public bots):

```json5
channels: {
  telegram: {
    dmPolicy: "open",
    allowFrom: ["*"]
  }
}
```

---

## SMB Onboarding Pattern

Recommended:
- Keep `pairing` default
- On signup, auto‑approve via CLI or write allowFrom file

Example (server‑side):
```bash
openclaw pairing approve telegram <code>
```

### Allowlist File Write (server‑side)

```bash
# Example: add user to allowFrom
cat > "$OPENCLAW_STATE_DIR/credentials/telegram-allowFrom.json" <<'EOF'
{
  "version": 1,
  "allowFrom": ["123456789", "*"]
}
EOF
```

---

## See Also

- [PROVISIONING.md](./PROVISIONING.md) — User provisioning
- [WP_PAIRING_FLOW.md](./WP_PAIRING_FLOW.md) — WordPress auto-approve
- [GATEWAY.md](./GATEWAY.md) — Gateway config reference
- [OPERATIONS.md](./OPERATIONS.md) — Where pairing files live
- [TRUST_MODES.md](./TRUST_MODES.md) — DM policy by tier
