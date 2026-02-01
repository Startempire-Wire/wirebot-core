<script>
  let { concept = '', position = 'below' } = $props();
  let show = $state(false);

  const tips = {
    score: "Your daily Execution Score (0–100). Sum of 4 lanes minus penalties plus streak bonus. ≥50 = Win.",
    shipping: "Shipping lane (40 pts max). Deploys, releases, features live, public artifacts. The biggest lane because shipping IS the game.",
    distribution: "Distribution lane (25 pts max). Blog posts, videos, emails, social content, outreach. Getting what you built in front of people.",
    revenue: "Revenue lane (20 pts max). Payments, subscriptions, deals closed, invoices. Money is the ultimate signal of value.",
    systems: "Systems lane (15 pts max). Automations, SOPs, tools, monitoring, delegation. Makes future work faster.",
    streak: "Consecutive days with shipping events. 3+ days = +5, 7+ = +10, 14+ = +15, 30+ = +20 bonus points.",
    record: "Season win/loss record. A Win = score ≥50 that day. A Loss = score <50. Like a sports team's record.",
    signal: "Color state: 🟢 Green (50+, winning) • 🟡 Yellow (30-49, pressure) • 🔴 Red (0-29, stalling)",
    intent: "Your declared priority for today. 'If today only had one win, what must ship?' Unfulfilled intent = -10 penalty.",
    stall: "24+ hours since your last shipping event. The longer the stall, the louder the warning. Ship something to clear it.",
    season: "90-day cycle with a theme and score reset. Creates urgency and fresh starts. Each season has its own record.",
    possession: "Which business you're focused on right now. Like ball possession in basketball — only one at a time.",
    ships: "Number of shipping events today. Each is a verified artifact that entered reality.",
    penalty: "Point deductions: Context switch after 2nd = -5 each. Commitment breach (unfulfilled intent) = -10.",
    bonus: "Streak bonus: Extra points for shipping consistently. Only applies on days you actually ship.",
    pending: "Events waiting for your approval. AI agents submit gated events that don't score until you say so.",
    clock_day: "How much of today has passed. Urgency signal — if it's 80% and score is low, time to ship NOW.",
    clock_week: "Week progress. Helps you see if you're front-loading (good) or back-loading (risky).",
    clock_season: "Season progress. Day X of 90. The closer to end, the more each day matters.",
    confidence: "How certain the system is an event is real (0-1). Webhooks = 0.95+. Agent reports = 0.70+. Higher = more points.",
    wrapped: "End-of-season retrospective. Record, patterns, top artifacts, best streaks. Like Spotify Wrapped for execution.",
    no_ship_cap: "If zero shipping events today, score capped at 30. You literally cannot win without shipping.",
  };
</script>

<span
  class="tooltip-trigger"
  role="button"
  tabindex="0"
  onclick={() => show = !show}
  onkeydown={(e) => e.key === 'Enter' && (show = !show)}
>
  <slot />
  {#if show && tips[concept]}
    <span class="tooltip {position}">
      <span class="tt-text">{tips[concept]}</span>
      <span class="tt-close" onclick={(e) => { e.stopPropagation(); show = false; }}>✕</span>
    </span>
  {/if}
</span>

<style>
  .tooltip-trigger {
    position: relative;
    cursor: help;
    display: inline;
    border-bottom: 1px dotted rgba(124,124,255,0.3);
    -webkit-tap-highlight-color: transparent;
  }

  .tooltip {
    position: absolute;
    left: 50%;
    transform: translateX(-50%);
    width: max(200px, min(280px, 70vw));
    background: #1a1a2e;
    border: 1px solid #2a2a4a;
    border-radius: 8px;
    padding: 10px 12px;
    font-size: 12px;
    line-height: 1.5;
    color: #bbb;
    z-index: 100;
    box-shadow: 0 4px 20px rgba(0,0,0,0.5);
    display: flex;
    align-items: flex-start;
    gap: 6px;
  }
  .tooltip.below { top: calc(100% + 6px); }
  .tooltip.above { bottom: calc(100% + 6px); }

  .tt-text { flex: 1; }
  .tt-close {
    font-size: 10px;
    opacity: 0.4;
    cursor: pointer;
    padding: 2px;
    flex-shrink: 0;
  }
</style>
