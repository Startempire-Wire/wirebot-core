<script>
  let { visible = false } = $props();
  let activeSection = $state('overview');

  const sections = [
    { id: 'overview', label: '🎯 Overview', icon: '🎯' },
    { id: 'score', label: '⚡ Score', icon: '⚡' },
    { id: 'lanes', label: '🛤️ Lanes', icon: '🛤️' },
    { id: 'signals', label: '🚦 Signals', icon: '🚦' },
    { id: 'streaks', label: '🔥 Streaks', icon: '🔥' },
    { id: 'seasons', label: '🏆 Seasons', icon: '🏆' },
    { id: 'planes', label: '📐 Three Planes', icon: '📐' },
    { id: 'penalties', label: '⚠️ Penalties', icon: '⚠️' },
    { id: 'events', label: '📋 Events', icon: '📋' },
    { id: 'gated', label: '🔒 Gated Events', icon: '🔒' },
    { id: 'intent', label: '🎯 Intent', icon: '🎯' },
    { id: 'glossary', label: '📖 Glossary', icon: '📖' },
  ];
</script>

{#if visible}
<div class="hints-overlay">
  <div class="hints-panel">
    <div class="hints-hdr">
      <h2>📘 How Scoreboard Works</h2>
      <button class="close-btn" onclick={() => visible = false}>✕</button>
    </div>

    <!-- Section nav -->
    <div class="section-nav">
      {#each sections as s}
        <button
          class="sec-btn {activeSection === s.id ? 'active' : ''}"
          onclick={() => activeSection = s.id}
        >{s.icon} {s.label.split(' ').slice(1).join(' ')}</button>
      {/each}
    </div>

    <!-- Content -->
    <div class="hints-body">

      {#if activeSection === 'overview'}
        <div class="rubric">
          <h3>One Question</h3>
          <div class="big-question">"Am I winning today?"</div>
          <p>Not "am I busy?" Not "did I work?" But: <strong>did reality objectively change because I worked?</strong></p>
          <p>The scoreboard treats building a business like a competitive sport. Every day is a game. You either win or you don't. No judgment — just signal.</p>

          <div class="card-box">
            <div class="cb-title">What This Is</div>
            <ul>
              <li>An <strong>execution accountability surface</strong></li>
              <li>A sports scoreboard for building a business</li>
              <li>Reality-backed — only verified artifacts score</li>
              <li>Time-boxed — daily games, 90-day seasons</li>
            </ul>
          </div>

          <div class="card-box warn">
            <div class="cb-title">What This Is NOT</div>
            <ul>
              <li>A task manager (use your checklist for that)</li>
              <li>A habit tracker (we don't score routines)</li>
              <li>A productivity tool (hours worked don't count)</li>
              <li>A vanity dashboard (you can't game it)</li>
            </ul>
          </div>

          <div class="card-box accent">
            <div class="cb-title">The Core Rule</div>
            <p><strong>Only completed outputs score. Progress is only counted when something leaves your local machine and enters reality.</strong></p>
            <p>A "ship" is: code deployed, article published, feature live, proposal sent, payment received, infrastructure activated.</p>
          </div>
        </div>

      {:else if activeSection === 'score'}
        <div class="rubric">
          <h3>Execution Score (0–100)</h3>
          <p>Your daily score measures one thing: <strong>did verifiable things happen today?</strong></p>

          <div class="score-breakdown">
            <div class="sb-row">
              <div class="sb-bar" style="width:40%; background:#4a9eff"></div>
              <div class="sb-info"><strong>Shipping</strong> — 40 pts max</div>
            </div>
            <div class="sb-row">
              <div class="sb-bar" style="width:25%; background:#9b59b6"></div>
              <div class="sb-info"><strong>Distribution</strong> — 25 pts max</div>
            </div>
            <div class="sb-row">
              <div class="sb-bar" style="width:20%; background:#2ecc71"></div>
              <div class="sb-info"><strong>Revenue</strong> — 25 pts max</div>
            </div>
            <div class="sb-row">
              <div class="sb-bar" style="width:15%; background:#e67e22"></div>
              <div class="sb-info"><strong>Systems</strong> — 15 pts max</div>
            </div>
          </div>

          <div class="card-box">
            <div class="cb-title">How Score Is Calculated</div>
            <ol>
              <li>Each event adds points to its lane</li>
              <li>Each lane has a cap (can't score more than the max)</li>
              <li>Lane scores are summed (base: 0–100)</li>
              <li>Streak bonus adds up to +20 (see Streaks)</li>
              <li>Penalties subtract (see Penalties)</li>
              <li>If zero ships today → score capped at 30</li>
              <li>Score ≥ 50 = WIN. Score &lt; 50 = LOSS</li>
            </ol>
          </div>

          <div class="card-box accent">
            <div class="cb-title">Win Threshold: 50</div>
            <p>You need to score at least 50 to win the day. This means you need meaningful activity in at least 2 lanes, OR heavy shipping + something else. You literally cannot win by only doing systems work.</p>
          </div>
        </div>

      {:else if activeSection === 'lanes'}
        <div class="rubric">
          <h3>The Four Lanes</h3>
          <p>Every event falls into one of four lanes. The weights reflect what actually moves a business forward.</p>

          <div class="lane-card" style="border-left-color:#4a9eff">
            <div class="lc-hdr"><span class="lc-icon" style="background:#4a9eff">🚀</span> SHIPPING — 40 pts</div>
            <p>The biggest lane. <strong>Shipping is the whole game.</strong></p>
            <p>What counts: Deploys, releases, features merged, public artifacts live, infrastructure activated, apps submitted.</p>
            <div class="lc-examples">
              <span class="lc-ev">PRODUCT_RELEASE +10</span>
              <span class="lc-ev">DEPLOY_SUCCESS +8</span>
              <span class="lc-ev">FEATURE_SHIPPED +6</span>
              <span class="lc-ev">PUBLIC_ARTIFACT +5</span>
              <span class="lc-ev">INFRASTRUCTURE_ACTIVATED +7</span>
            </div>
            <p class="lc-note">If you shipped nothing today, your score is capped at 30. No matter what else happened.</p>
          </div>

          <div class="lane-card" style="border-left-color:#9b59b6">
            <div class="lc-hdr"><span class="lc-icon" style="background:#9b59b6">📢</span> DISTRIBUTION — 25 pts</div>
            <p>Getting what you built in front of people. Amplification.</p>
            <p>What counts: Blog posts published, videos uploaded, email campaigns sent, social posts with real content, outreach, podcasts.</p>
            <div class="lc-examples">
              <span class="lc-ev">VIDEO_PUBLISHED +7</span>
              <span class="lc-ev">BLOG_PUBLISHED +6</span>
              <span class="lc-ev">PODCAST_PUBLISHED +6</span>
              <span class="lc-ev">EMAIL_CAMPAIGN_SENT +5</span>
              <span class="lc-ev">SOCIAL_POST_BUSINESS +3-5</span>
              <span class="lc-ev">COLD_OUTREACH +4</span>
            </div>
          </div>

          <div class="lane-card" style="border-left-color:#2ecc71">
            <div class="lc-hdr"><span class="lc-icon" style="background:#2ecc71">💰</span> REVENUE — 20 pts</div>
            <p>Money. The ultimate signal that what you're building has value.</p>
            <p>What counts: Payments received, subscriptions created, deals closed, invoices paid, proposals sent.</p>
            <div class="lc-examples">
              <span class="lc-ev">SUBSCRIPTION_CREATED +12</span>
              <span class="lc-ev">PAYMENT_RECEIVED +10</span>
              <span class="lc-ev">DEAL_CLOSED +8</span>
              <span class="lc-ev">INVOICE_PAID +8</span>
              <span class="lc-ev">PROPOSAL_SENT +4</span>
            </div>
          </div>

          <div class="lane-card" style="border-left-color:#e67e22">
            <div class="lc-hdr"><span class="lc-icon" style="background:#e67e22">⚙️</span> SYSTEMS — 15 pts</div>
            <p>Infrastructure that makes future work faster or unnecessary.</p>
            <p>What counts: Automations deployed, SOPs documented, tools integrated, monitoring enabled, delegation completed.</p>
            <div class="lc-examples">
              <span class="lc-ev">AUTOMATION_DEPLOYED +6</span>
              <span class="lc-ev">DELEGATION_COMPLETED +6</span>
              <span class="lc-ev">TOOL_INTEGRATED +5</span>
              <span class="lc-ev">SOP_DOCUMENTED +4</span>
              <span class="lc-ev">MONITORING_ENABLED +4</span>
            </div>
            <p class="lc-note">Important but can't carry the day alone. Systems support shipping — they don't replace it.</p>
          </div>
        </div>

      {:else if activeSection === 'signals'}
        <div class="rubric">
          <h3>Signal Colors</h3>
          <p>The score number tells you the detail. The color tells you the state at a glance.</p>

          <div class="signal-card sig-green">
            <div class="sig-score">50–100</div>
            <div class="sig-name">🟢 WINNING</div>
            <p>You shipped. Reality changed. Multiple lanes active. Keep pushing.</p>
          </div>

          <div class="signal-card sig-yellow">
            <div class="sig-score">30–49</div>
            <div class="sig-name">🟡 PRESSURE</div>
            <p>Some activity, but not enough. You're on the edge. One more ship could flip this to a win. A common state when you're doing systems/distribution work without shipping.</p>
          </div>

          <div class="signal-card sig-red">
            <div class="sig-score">0–29</div>
            <div class="sig-name">🔴 STALLING</div>
            <p>Nothing verifiable happened. Either you didn't work, or you worked on things that don't score (planning, meetings, research). The stall alert flashes after 24h with no ship.</p>
          </div>

          <div class="card-box">
            <div class="cb-title">Reading the Signal</div>
            <ul>
              <li><strong>Green doesn't mean relax</strong> — it means "the machine is running"</li>
              <li><strong>Yellow is the most important color</strong> — it means you're close. Push.</li>
              <li><strong>Red isn't punishment</strong> — it's information. Maybe you needed a rest day. Maybe you got pulled into fires. The scoreboard just tells the truth.</li>
            </ul>
          </div>
        </div>

      {:else if activeSection === 'streaks'}
        <div class="rubric">
          <h3>Ship Streaks</h3>
          <p>A streak is consecutive days with at least one shipping event. Streaks reward consistency over bursts.</p>

          <div class="card-box accent">
            <div class="cb-title">Streak Bonuses</div>
            <table class="rubric-table">
              <thead><tr><th>Streak</th><th>Bonus</th><th>What It Means</th></tr></thead>
              <tbody>
                <tr><td>3+ days</td><td>+5</td><td>Building rhythm</td></tr>
                <tr><td>7+ days</td><td>+10</td><td>Real momentum</td></tr>
                <tr><td>14+ days</td><td>+15</td><td>Shipping machine</td></tr>
                <tr><td>30+ days</td><td>+20</td><td>Legendary consistency</td></tr>
              </tbody>
            </table>
            <p class="note">Bonus only applies on days you ship. A 7-day streak that breaks doesn't retroactively lose bonus from previous days.</p>
          </div>

          <div class="card-box">
            <div class="cb-title">How Streaks Work</div>
            <ul>
              <li>Counted by calendar day (in your timezone)</li>
              <li>A "ship" = any event in the shipping lane</li>
              <li>Miss one day → streak resets to 0</li>
              <li>Your best streak ever is tracked separately</li>
              <li>Streak bonus is added AFTER lane scores, BEFORE penalties</li>
            </ul>
          </div>

          <div class="card-box warn">
            <div class="cb-title">Why Not Score Streaks?</div>
            <p>We track streaks but don't penalize breaking them. A broken streak already hurts your average. Adding penalty on top would incentivize shipping garbage just to keep a streak alive. We want <strong>quality ships, not streak anxiety.</strong></p>
          </div>
        </div>

      {:else if activeSection === 'seasons'}
        <div class="rubric">
          <h3>Seasons</h3>
          <p>Business doesn't happen on an infinite timeline. It happens in focused, themed pushes. A season is 90 days with a specific mission.</p>

          <div class="card-box accent">
            <div class="cb-title">Season Structure</div>
            <ul>
              <li><strong>Duration:</strong> 90 days (configurable)</li>
              <li><strong>Theme:</strong> Each season has a focus ("Red-to-Black", "Audience First", "Revenue Proof")</li>
              <li><strong>Record:</strong> Wins vs Losses (days scored ≥50 vs &lt;50)</li>
              <li><strong>Average:</strong> Running average execution score</li>
              <li><strong>Reset:</strong> Score counters reset. History stays.</li>
            </ul>
          </div>

          <div class="card-box">
            <div class="cb-title">Why Seasons?</div>
            <ul>
              <li><strong>Urgency:</strong> "Day 45 of 90" hits different than "Day 412"</li>
              <li><strong>Fresh starts:</strong> Bad months don't haunt you forever</li>
              <li><strong>Focus:</strong> Theme forces one mission per season</li>
              <li><strong>Retrospective:</strong> Wrapped at end shows what actually happened</li>
              <li><strong>Prevents hoarding:</strong> Can't rest on past wins</li>
            </ul>
          </div>

          <div class="card-box">
            <div class="cb-title">Calendar Heatmap</div>
            <p>The Season view shows a month calendar colored by daily result:</p>
            <div class="color-legend">
              <span><span class="dot" style="background:#00ff64"></span> Win (≥50)</span>
              <span><span class="dot" style="background:#ffc800"></span> Close (30–49)</span>
              <span><span class="dot" style="background:#ff3232"></span> Loss (&lt;30)</span>
              <span><span class="dot" style="background:var(--border)"></span> No data</span>
            </div>
            <p>At a glance, you can see patterns: weekend gaps, streak runs, stall clusters.</p>
          </div>
        </div>

      {:else if activeSection === 'planes'}
        <div class="rubric">
          <h3>The Three Planes</h3>
          <p>This is the most important concept. <strong>Most productivity tools fail because they mix these planes.</strong> The scoreboard does not.</p>

          <div class="plane-card reality">
            <div class="pc-hdr">✅ Reality Plane — SCORED</div>
            <p>Things that objectively changed in the world because you worked.</p>
            <ul>
              <li>Code deployed and running in production</li>
              <li>Article published and accessible via URL</li>
              <li>Payment received in your bank account</li>
              <li>Feature live and usable by customers</li>
            </ul>
            <p class="pc-rule">Rule: If someone else can verify it without your explanation, it's Reality.</p>
          </div>

          <div class="plane-card behavior">
            <div class="pc-hdr">⚠️ Behavior Plane — TRACKED (penalties only)</div>
            <p>How you worked. Visible to you and your AI partner, but not scored positively.</p>
            <ul>
              <li>Focus sessions, time allocation</li>
              <li>Context switches between businesses</li>
              <li>Commitments made vs. fulfilled</li>
              <li>Distraction patterns</li>
            </ul>
            <p class="pc-rule">Rule: Behavior can only <em>deduct</em> points (penalties), never add them. Good behavior is its own reward — it should produce Reality events.</p>
          </div>

          <div class="plane-card reflection">
            <div class="pc-hdr">🔒 Reflection Plane — NEVER SCORED</div>
            <p>How you felt. Private. Between you and your AI partner. Never touches the score.</p>
            <ul>
              <li>Mood, energy level, stress</li>
              <li>Personal challenges or setbacks</li>
              <li>Clarity about direction</li>
              <li>Motivation, doubt, ambition</li>
            </ul>
            <p class="pc-rule">Rule: Scoring feelings would create perverse incentives ("pretend to be happy to score higher"). Reflection is tracked for sustainability — so your AI partner can notice burnout patterns — but never for scoring.</p>
          </div>
        </div>

      {:else if activeSection === 'penalties'}
        <div class="rubric">
          <h3>Penalties</h3>
          <p>The scoreboard has three penalty mechanics. They exist to keep scoring honest, not to punish.</p>

          <div class="card-box warn">
            <div class="cb-title">⚠️ Context Switch — -5 per switch (after 2nd)</div>
            <p>Tracked automatically from <code>wb focus</code> changes. Your first two focus switches per day are free — sometimes you need to check on another business. But the 3rd switch costs 5 points, 4th costs another 5, etc.</p>
            <p><strong>Why:</strong> Context switching is the #1 productivity killer for multi-business operators. The penalty isn't large enough to matter if you're shipping, but it makes the cost visible.</p>
          </div>

          <div class="card-box warn">
            <div class="cb-title">⚠️ Commitment Breach — -10</div>
            <p>If you declared an intent ("I will ship X today") and the day ended with zero shipping events, that's a breach. You said you would and didn't.</p>
            <p><strong>Why:</strong> Declaring intent creates accountability. Breaking it should sting a little. The fix is simple: either ship something, or don't declare intent if you're not confident.</p>
          </div>

          <div class="card-box">
            <div class="cb-title">📊 No-Ship Cap — Max 30</div>
            <p>Not technically a penalty, but a cap. If you have zero shipping events today, your score can't exceed 30 — no matter how much distribution, revenue, or systems work you did.</p>
            <p><strong>Why:</strong> The scoreboard has a shipping bias by design. You can have the best systems in the world, but if nothing ships, the needle doesn't move. 30 is "respectable work day" territory. 50+ requires shipping.</p>
          </div>
        </div>

      {:else if activeSection === 'events'}
        <div class="rubric">
          <h3>Events</h3>
          <p>Everything on the scoreboard traces back to events. An event is: "something verifiable happened."</p>

          <div class="card-box">
            <div class="cb-title">Event Anatomy</div>
            <ul>
              <li><strong>Type:</strong> What happened (FEATURE_SHIPPED, PAYMENT_RECEIVED, etc.)</li>
              <li><strong>Lane:</strong> Which lane it scores in (shipping, distribution, revenue, systems)</li>
              <li><strong>Source:</strong> Where it came from (github, stripe, wb-cli, agent)</li>
              <li><strong>Artifact:</strong> Title and URL of the evidence</li>
              <li><strong>Confidence:</strong> 0.0–1.0 — how certain the score engine should be</li>
              <li><strong>Score delta:</strong> How many points this event added</li>
            </ul>
          </div>

          <div class="card-box accent">
            <div class="cb-title">Confidence Levels</div>
            <table class="rubric-table">
              <thead><tr><th>Range</th><th>Meaning</th><th>Example</th></tr></thead>
              <tbody>
                <tr><td>0.95–1.0</td><td>Verified by external system</td><td>Stripe webhook, GitHub API</td></tr>
                <tr><td>0.85–0.94</td><td>Verified by agent</td><td>AI confirmed URL is live</td></tr>
                <tr><td>0.70–0.84</td><td>Reported by agent</td><td>AI detected activity, not verified</td></tr>
                <tr><td>&lt;0.70</td><td>Low confidence</td><td>Self-reported, should be gated</td></tr>
              </tbody>
            </table>
            <p class="note">Higher confidence → full point value. Lower confidence → scaled down. Events below 0.50 are auto-rejected.</p>
          </div>

          <div class="card-box">
            <div class="cb-title">Where Events Come From</div>
            <ul>
              <li><strong>CLI:</strong> <code>wb ship "..."</code>, <code>wb complete</code> — auto-approved</li>
              <li><strong>Webhooks:</strong> GitHub, Stripe — auto-approved at 0.95+ confidence</li>
              <li><strong>AI agents:</strong> Wirebot, Claude, Pi — submitted as pending (gated)</li>
              <li><strong>Quick-add:</strong> FAB button in the app — sent directly</li>
              <li><strong>Cron:</strong> EOD lock, morning standup — system events</li>
            </ul>
          </div>
        </div>

      {:else if activeSection === 'gated'}
        <div class="rubric">
          <h3>Gated Events</h3>
          <p>Some events need your approval before they score. This prevents AI agents or automated systems from inflating your score.</p>

          <div class="card-box accent">
            <div class="cb-title">How Gating Works</div>
            <ol>
              <li>An agent (or system) submits an event with <code>status: "pending"</code></li>
              <li>The event is stored but gets <strong>zero points</strong></li>
              <li>You see a "⏳ pending" badge in the app</li>
              <li>You review and either <strong>approve</strong> (scores points) or <strong>reject</strong> (disappears)</li>
            </ol>
          </div>

          <div class="card-box">
            <div class="cb-title">What Gets Auto-Approved</div>
            <ul>
              <li><code>wb ship</code>, <code>wb complete</code> — you typed it, you own it</li>
              <li>Stripe webhooks — money is the ultimate proof</li>
              <li>GitHub webhooks — system-verified deployment</li>
            </ul>
          </div>

          <div class="card-box warn">
            <div class="cb-title">What Gets Gated</div>
            <ul>
              <li>AI agent observations ("I noticed you deployed...")</li>
              <li>Automated scans with low confidence</li>
              <li>Any event with <code>confidence &lt; 0.85</code> from non-human sources</li>
              <li>Any event explicitly submitted with <code>status: "pending"</code></li>
            </ul>
          </div>

          <div class="card-box">
            <div class="cb-title">CLI Commands</div>
            <ul>
              <li><code>wb submit "description"</code> — create pending event</li>
              <li><code>wb pending</code> — see all pending events</li>
              <li><code>wb approve &lt;id&gt;</code> — approve and score</li>
              <li><code>wb reject &lt;id&gt;</code> — reject and discard</li>
            </ul>
          </div>
        </div>

      {:else if activeSection === 'intent'}
        <div class="rubric">
          <h3>Daily Intent</h3>
          <p>The single most powerful feature. Before you start your day, declare: <strong>"If today only had one win, what must ship?"</strong></p>

          <div class="card-box accent">
            <div class="cb-title">The Three Questions</div>
            <ol>
              <li><strong>What must ship?</strong> — If today only had one win, what is it?</li>
              <li><strong>What's the smallest version?</strong> — What's the MVP of that win?</li>
              <li><strong>What can you ignore?</strong> — What are you allowed to not touch until this ships?</li>
            </ol>
          </div>

          <div class="card-box">
            <div class="cb-title">How Intent Works</div>
            <ul>
              <li>Declare via <code>wb intent "Ship the checkout page"</code> or the app</li>
              <li>Shown at the top of the Score view all day</li>
              <li>At EOD lock, the system checks: did you ship?</li>
              <li>If intent + ships → <strong>intent fulfilled ✅</strong></li>
              <li>If intent + no ships → <strong>commitment breach ⚠️ -10 penalty</strong></li>
              <li>If no intent declared → no penalty either way</li>
            </ul>
          </div>

          <div class="card-box">
            <div class="cb-title">Good Intents</div>
            <ul>
              <li>✅ "Deploy the payment integration" — specific, shippable</li>
              <li>✅ "Publish the landing page" — verifiable output</li>
              <li>✅ "Send the first email campaign" — concrete action</li>
            </ul>
          </div>

          <div class="card-box warn">
            <div class="cb-title">Bad Intents</div>
            <ul>
              <li>❌ "Work on the app" — not specific enough</li>
              <li>❌ "Research competitors" — not shippable</li>
              <li>❌ "Plan the marketing strategy" — planning doesn't score</li>
              <li>❌ "Fix bugs" — which bugs? Be specific.</li>
            </ul>
          </div>
        </div>

      {:else if activeSection === 'glossary'}
        <div class="rubric">
          <h3>Glossary</h3>
          <dl class="glossary">
            <dt>Artifact</dt>
            <dd>A verifiable output — a URL, a deploy, a payment. The evidence that something happened.</dd>

            <dt>Confidence</dt>
            <dd>How certain we are an event is real (0.0–1.0). Higher = more points. Webhooks are 0.95+. Self-reported is lower.</dd>

            <dt>EOD Lock</dt>
            <dd>End-of-day score finalization. After lock, today's score can't change. Determines Win/Loss.</dd>

            <dt>Execution Score</dt>
            <dd>Your daily 0–100 score. Measures output across 4 lanes. ≥50 = Win.</dd>

            <dt>Gated Event</dt>
            <dd>An event submitted as "pending" that needs operator approval before it scores.</dd>

            <dt>Intent</dt>
            <dd>Your declared priority for the day. "The one thing that must ship." Creates accountability.</dd>

            <dt>Lane</dt>
            <dd>One of 4 scoring categories: Shipping (40), Distribution (25), Revenue (20), Systems (15).</dd>

            <dt>Possession</dt>
            <dd>Which business you're actively focused on. Like ball possession in basketball.</dd>

            <dt>Season</dt>
            <dd>A 90-day cycle with a theme, record, and score reset. Keeps things fresh and urgent.</dd>

            <dt>Ship</dt>
            <dd>A completed output that entered reality. Code deployed. Article published. Not a draft. Not a plan.</dd>

            <dt>Signal</dt>
            <dd>The color state: Green (winning, 50+), Yellow (pressure, 30-49), Red (stalling, 0-29).</dd>

            <dt>Stall</dt>
            <dd>24+ hours since last shipping event. Triggers a flashing alert in the Score view.</dd>

            <dt>Streak</dt>
            <dd>Consecutive days with at least one shipping event. Unlocks bonus points.</dd>

            <dt>Three Planes</dt>
            <dd>Reality (scored), Behavior (penalties only), Reflection (never scored). The fundamental architecture.</dd>

            <dt>Win/Loss</dt>
            <dd>Daily result. Score ≥50 = Win. &lt;50 = Loss. Season record tracks W-L.</dd>

            <dt>Wrapped</dt>
            <dd>End-of-season retrospective. Like Spotify Wrapped but for your business execution.</dd>
          </dl>
        </div>
      {/if}
    </div>
  </div>
</div>
{/if}

<style>
  .hints-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.85);
    z-index: 200;
    display: flex;
    align-items: flex-end;
    justify-content: center;
  }

  .hints-panel {
    background: var(--bg);
    width: 100%;
    max-width: 600px;
    max-height: 90dvh;
    border-radius: 16px 16px 0 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .hints-hdr {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 14px 16px;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
  }
  .hints-hdr h2 { font-size: 16px; font-weight: 700; }
  .close-btn {
    background: none;
    border: 1px solid #333;
    color: #888;
    width: 32px;
    height: 32px;
    border-radius: 50%;
    cursor: pointer;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .section-nav {
    display: flex;
    gap: 4px;
    padding: 8px 12px;
    overflow-x: auto;
    border-bottom: 1px solid var(--border);
    flex-shrink: 0;
    -webkit-overflow-scrolling: touch;
  }
  .section-nav::-webkit-scrollbar { display: none; }
  .sec-btn {
    padding: 6px 10px;
    border-radius: 16px;
    border: 1px solid var(--border-light);
    background: transparent;
    color: #888;
    font-size: 11px;
    white-space: nowrap;
    cursor: pointer;
    flex-shrink: 0;
  }
  .sec-btn.active { background: rgba(124,124,255,0.15); border-color: var(--accent); color: var(--accent); }

  .hints-body {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    -webkit-overflow-scrolling: touch;
  }

  .rubric h3 {
    font-size: 18px;
    font-weight: 800;
    margin-bottom: 8px;
    color: var(--text);
  }
  .rubric p {
    font-size: 13px;
    line-height: 1.6;
    color: #aaa;
    margin-bottom: 10px;
  }
  .rubric strong { color: var(--text); }
  .rubric code { font-family: monospace; font-size: 12px; color: var(--accent); background: rgba(124,124,255,0.1); padding: 1px 4px; border-radius: 3px; }
  .rubric ol, .rubric ul { padding-left: 20px; margin-bottom: 10px; }
  .rubric li { font-size: 13px; line-height: 1.6; color: #aaa; margin-bottom: 4px; }

  .big-question {
    font-size: 22px;
    font-weight: 900;
    color: var(--accent);
    text-align: center;
    padding: 16px 0;
  }

  /* Card boxes */
  .card-box {
    background: var(--bg-card);
    border-radius: 10px;
    padding: 12px;
    margin: 12px 0;
    border-left: 3px solid #333;
  }
  .card-box.accent { border-left-color: var(--accent); background: rgba(124,124,255,0.04); }
  .card-box.warn { border-left-color: #ffc800; background: rgba(255,200,0,0.04); }
  .cb-title { font-size: 13px; font-weight: 700; color: var(--text); margin-bottom: 6px; }
  .card-box p { margin-bottom: 6px; }
  .card-box .note { font-size: 11px; opacity: 0.5; margin-top: 6px; }

  /* Score breakdown */
  .score-breakdown { display: flex; flex-direction: column; gap: 8px; margin: 12px 0; }
  .sb-row { display: flex; align-items: center; gap: 8px; }
  .sb-bar { height: 16px; border-radius: 4px; min-width: 4px; }
  .sb-info { font-size: 12px; color: #aaa; }

  /* Lane cards */
  .lane-card {
    background: var(--bg-card);
    border-radius: 10px;
    padding: 12px;
    margin: 12px 0;
    border-left: 4px solid;
  }
  .lc-hdr { font-size: 14px; font-weight: 700; display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
  .lc-icon { display: inline-flex; align-items: center; justify-content: center; width: 24px; height: 24px; border-radius: 6px; font-size: 12px; }
  .lc-examples { display: flex; flex-wrap: wrap; gap: 4px; margin: 8px 0; }
  .lc-ev {
    font-size: 10px;
    padding: 3px 8px;
    border-radius: 12px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.08);
    color: #aaa;
    white-space: nowrap;
  }
  .lc-note { font-size: 11px; opacity: 0.5; font-style: italic; }

  /* Signal cards */
  .signal-card {
    border-radius: 10px;
    padding: 14px;
    margin: 10px 0;
    display: flex;
    flex-direction: column;
    gap: 4px;
  }
  .sig-green { background: rgba(0,255,100,0.06); border: 1px solid rgba(0,255,100,0.2); }
  .sig-yellow { background: rgba(255,200,0,0.06); border: 1px solid rgba(255,200,0,0.2); }
  .sig-red { background: rgba(255,50,50,0.06); border: 1px solid rgba(255,50,50,0.2); }
  .sig-score { font-size: 20px; font-weight: 900; }
  .sig-green .sig-score { color: #00ff64; }
  .sig-yellow .sig-score { color: #ffc800; }
  .sig-red .sig-score { color: #ff3232; }
  .sig-name { font-size: 13px; font-weight: 700; letter-spacing: 0.1em; }
  .sig-green .sig-name { color: #00cc50; }
  .sig-yellow .sig-name { color: #cc9900; }
  .sig-red .sig-name { color: #cc2020; }
  .signal-card p { font-size: 12px; color: #aaa; margin: 0; }

  /* Plane cards */
  .plane-card {
    border-radius: 10px;
    padding: 14px;
    margin: 10px 0;
    border-left: 4px solid;
  }
  .plane-card.reality { border-left-color: #00ff64; background: rgba(0,255,100,0.04); }
  .plane-card.behavior { border-left-color: #ffc800; background: rgba(255,200,0,0.04); }
  .plane-card.reflection { border-left-color: var(--accent); background: rgba(124,124,255,0.04); }
  .pc-hdr { font-size: 14px; font-weight: 700; margin-bottom: 6px; }
  .pc-rule { font-size: 11px; font-weight: 600; margin-top: 8px; padding: 6px 8px; background: rgba(255,255,255,0.03); border-radius: 6px; }

  /* Color legend */
  .color-legend {
    display: flex;
    flex-wrap: wrap;
    gap: 12px;
    margin: 8px 0;
  }
  .color-legend span { display: flex; align-items: center; gap: 4px; font-size: 11px; }
  .dot { width: 12px; height: 12px; border-radius: 3px; display: inline-block; }

  /* Rubric table */
  .rubric-table { width: 100%; border-collapse: collapse; margin: 8px 0; }
  .rubric-table th, .rubric-table td {
    padding: 6px 8px;
    font-size: 12px;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }
  .rubric-table th { color: #888; font-weight: 600; font-size: 10px; letter-spacing: 0.05em; }
  .rubric-table td { color: #aaa; }

  /* Glossary */
  .glossary { margin: 0; }
  .glossary dt {
    font-size: 14px;
    font-weight: 700;
    color: var(--accent);
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #1a1a25;
  }
  .glossary dt:first-child { border-top: none; margin-top: 0; padding-top: 0; }
  .glossary dd {
    font-size: 13px;
    color: #aaa;
    line-height: 1.6;
    margin: 4px 0 0 0;
  }
</style>
