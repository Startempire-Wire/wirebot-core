---
name: score
description: Query Business Performance Scoreboard for score, lanes, streak, and season stats.
---

# Score

Get current execution metrics from the Business Performance Scoreboard.

## Use Cases

- Check today's score
- See lane progress (Ship, Distribute, Revenue, Systems)
- Check shipping streak
- View season record

## Implementation

Call `wirebot_score` tool with action="dashboard":

```
wirebot_score --action dashboard
```

## Response Format

```
⚡ SCORE: {score}/100 | 🔥 {streak}-day streak

📊 Lanes:
• Ship: {ship}/40
• Dist: {dist}/25  
• Rev:  {rev}/20
• Sys:  {sys}/15

🏆 Season {season}: {wins}W-{losses}L
```
