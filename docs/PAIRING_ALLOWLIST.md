# Pairing + Allowlist

> **Clawdbot default DM policy = pairing.**

---

## Where Pairing Lives

Pairing + allowlist files are stored in the credentials dir:

```
~/.clawdbot/credentials/
  telegram-pairing.json
  telegram-allowFrom.json
  discord-pairing.json
  discord-allowFrom.json
  imessage-pairing.json
  imessage-allowFrom.json
```

Path resolves via:
- `CLAWDBOT_STATE_DIR/credentials` (preferred)
- `~/.clawdbot/credentials`

---

## CLI Commands

```bash
# list pending pairing
clawdbot pairing list --channel telegram

# approve
clawdbot pairing approve telegram ABCD1234

# reject
clawdbot pairing reject telegram ABCD1234
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
clawdbot pairing approve telegram <code>
```

### Allowlist File Write (server‑side)

```bash
# Example: add user to allowFrom
cat > "$CLAWDBOT_STATE_DIR/credentials/telegram-allowFrom.json" <<'EOF'
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
