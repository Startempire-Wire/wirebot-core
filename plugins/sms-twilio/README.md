# SMS (Twilio) Plugin — Skeleton

Outbound-only SMS tool for Clawdbot.

**Status:** skeleton — inbound webhook not implemented.

## Compliance note

Twilio US long‑code SMS requires A2P 10DLC registration. Consider:
- Toll‑free numbers (less 10DLC burden)
- Non‑US numbers (local rules apply)
- Alternate providers (custom plugin)

## Config (clawdbot.json)

```json5
plugins: {
  load: { paths: ["/home/wirebot/wirebot-core/plugins"] },
  entries: {
    "sms-twilio": {
      config: {
        accountSid: "${TWILIO_ACCOUNT_SID}",
        authToken: "${TWILIO_AUTH_TOKEN}",
        fromNumber: "+15551234567",
        statusCallback: "https://example.com/twilio/status"
      }
    }
  }
}
```

## Tool

- `sms_send` — sends outbound SMS

## TODO

- Inbound webhook handler
- Channel plugin integration
- Delivery status reconciliation
