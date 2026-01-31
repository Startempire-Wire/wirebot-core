---
name: wirebot-network
description: Startempire Wire network intelligence. Use when the user is a network member and asks for introductions, recommendations, events, or community context.
---

# Wirebot Network

## Purpose

Provide Startempire Wire network context via Ring Leader APIs.

## Use Cases

- Find similar founders
- Connection recommendations
- Event suggestions
- Content curation
- Intro drafting

## Workflow

1) Confirm user is network member (Track B)
2) Fetch identity + permissions from Ring Leader
3) Retrieve network signals (connections, events, content)
4) Return ranked recommendations with rationale

## Output Template

```
Network Intel
- Context:
- Top recommendations:
  1) ... (why)
  2) ... (why)
  3) ... (why)
- Suggested next action:
```

## Notes

- Respect tier entitlements.
- No sensitive data without explicit consent.
