<script>
  import { createEventDispatcher } from 'svelte';
  import PairingFlow from './PairingFlow.svelte';
  const dispatch = createEventDispatcher();

  let { apiBase = '', token = '', onAssess = null } = $props();
  let loading = $state(true);
  let error = $state('');
  let effective = $state(null);
  let accuracy = $state(null);
  let evidence = $state(null);
  let complement = $state(null);
  let insights = $state(null);
  let showFlow = $state(false);
  let flowInstrument = $state('');

  async function fetchAPI(path) {
    const res = await fetch(apiBase + path, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) throw new Error(`${res.status}`);
    return res.json();
  }

  async function load() {
    loading = true;
    error = '';
    try {
      [effective, accuracy, complement, evidence, insights] = await Promise.all([
        fetchAPI('/v1/pairing/profile/effective'),
        fetchAPI('/v1/pairing/accuracy'),
        fetchAPI('/v1/pairing/complement'),
        fetchAPI('/v1/pairing/evidence?limit=10'),
        fetchAPI('/v1/pairing/insights'),
      ]);
    } catch (e) {
      error = e.message;
    }
    loading = false;
  }

  $effect(() => { if (token) load(); });

  // Construct display config
  const constructs = [
    { key: 'action_style', label: 'Action Style', icon: '⚡', color: '#7c7cff',
      dims: [
        { code: 'FF', name: 'Fact Finder' },
        { code: 'FT', name: 'Follow Through' },
        { code: 'QS', name: 'Quick Start' },
        { code: 'IM', name: 'Implementor' },
      ],
      instrument: 'ASI-12', desc: '12 forced-choice pairs' },
    { key: 'disc', label: 'Communication', icon: '💬', color: '#ff7c7c',
      dims: [
        { code: 'D', name: 'Dominance' },
        { code: 'I', name: 'Influence' },
        { code: 'S', name: 'Steadiness' },
        { code: 'C', name: 'Conscientiousness' },
      ],
      instrument: 'CSI-8', desc: '8 scenario picks' },
    { key: 'energy', label: 'Energy Topology', icon: '🔋', color: '#7cff7c',
      dims: [
        { code: 'W', name: 'Wonder' },
        { code: 'N', name: 'Invention' },
        { code: 'D_disc', name: 'Discernment' },
        { code: 'G', name: 'Galvanizing' },
        { code: 'E', name: 'Enablement' },
        { code: 'T', name: 'Tenacity' },
      ],
      instrument: 'ETM-6', desc: 'Drag to sort by energy' },
    { key: 'risk', label: 'Risk Disposition', icon: '🎲', color: '#ffd700',
      dims: [
        { code: 'tolerance', name: 'Risk Tolerance' },
        { code: 'speed', name: 'Decision Speed' },
        { code: 'loss_aversion', name: 'Loss Aversion' },
        { code: 'ambiguity', name: 'Ambiguity Comfort' },
        { code: 'bias_to_action', name: 'Bias to Action' },
        { code: 'sunk_cost', name: 'Sunk Cost Immunity' },
      ],
      instrument: 'RDS-6', desc: '6 sliders' },
    { key: 'cognitive', label: 'Cognitive Style', icon: '🧠', color: '#ff7cff',
      dims: [
        { code: 'holistic', name: 'Holistic' },
        { code: 'sequential', name: 'Sequential' },
        { code: 'abstract', name: 'Abstract' },
        { code: 'concrete', name: 'Concrete' },
      ],
      instrument: 'COG-8', desc: '8 scenario picks' },
    { key: 'business', label: 'Business Reality', icon: '🏢', color: '#ff9f43',
      dims: [
        { code: 'focus', name: 'Focus Level' },
        { code: 'revenue_maturity', name: 'Revenue Maturity' },
        { code: 'team_size', name: 'Team Size' },
        { code: 'bottleneck', name: 'Bottleneck Area' },
        { code: 'venture_age', name: 'Venture Age' },
        { code: 'debt_pressure', name: 'Debt Pressure' },
      ],
      instrument: 'BIZ-6', desc: '6 context questions' },
    { key: 'temporal', label: 'Temporal Patterns', icon: '⏰', color: '#54a0ff',
      dims: [
        { code: 'peak_hour', name: 'Peak Hours' },
        { code: 'planning_style', name: 'Planning Style' },
        { code: 'stall_recovery', name: 'Stall Recovery' },
        { code: 'work_intensity', name: 'Work Intensity' },
        { code: 'context_switch_cost', name: 'Context Switch Cost' },
        { code: 'planning_horizon', name: 'Planning Horizon' },
      ],
      instrument: 'TIME-6', desc: '6 rhythm questions' },
  ];

  function dimValue(construct, code) {
    if (!effective || !effective[construct]) return 0;
    return effective[construct][code] || 0;
  }

  function hasDimData(construct) {
    if (!effective || !effective[construct]) return false;
    return Object.values(effective[construct]).some(v => v > 0);
  }

  function startAssessment(instrument) {
    if (onAssess) {
      onAssess(instrument);
    } else {
      flowInstrument = instrument;
      showFlow = true;
    }
  }

  function complementTop() {
    if (!complement?.sorted) return [];
    return complement.sorted.filter(s => s.allocation > 0).sort((a,b) => b.allocation - a.allocation).slice(0, 5);
  }

  function timeAgo(ts) {
    const d = new Date(ts);
    const now = Date.now();
    const s = Math.floor((now - d.getTime()) / 1000);
    if (s < 60) return 'now';
    if (s < 3600) return Math.floor(s/60) + 'm';
    if (s < 86400) return Math.floor(s/3600) + 'h';
    return Math.floor(s/86400) + 'd';
  }
</script>

{#if loading}
  <div class="profile-loading">
    <div class="pulse">🧬</div>
    <p>Loading profile...</p>
  </div>
{:else if error}
  <div class="profile-error">
    <p>Failed to load profile</p>
    <button onclick={load}>Retry</button>
  </div>
{:else if showFlow}
  <PairingFlow
    instrument={flowInstrument}
    {apiBase}
    {token}
    onComplete={() => { showFlow = false; load(); }}
    onBack={() => showFlow = false}
  />
{:else}
  <div class="profile">
    <div class="p-header">
      <h2>🧬 Founder Profile</h2>
      <div class="p-meta">
        {#if accuracy}
          <span class="accuracy" title="Model accuracy">
            {(accuracy.overall_accuracy * 100).toFixed(0)}% accurate
          </span>
          <span class="signals">
            {accuracy.days_active?.toFixed(0) || 0}d active
          </span>
        {/if}
      </div>
    </div>

    <!-- Pairing Score -->
    {#if effective}
      <div class="score-card">
        <div class="sc-ring" style="--pct:{effective.pairing_score}">
          <span class="sc-num">{effective.pairing_score}</span>
        </div>
        <div class="sc-info">
          <div class="sc-label">Pairing Score</div>
          <div class="sc-level">{effective.level || 'Initializing...'}</div>
          {#if effective.pairing_score < 20}
            <div class="sc-hint">Complete assessments below to calibrate</div>
          {:else if effective.pairing_score < 60}
            <div class="sc-hint">Send 50+ chat messages for behavioral data</div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Constructs (Equalizer bars) -->
    <div class="constructs">
      {#each constructs as c}
        <div class="construct" style="--accent:{c.color}">
          <div class="c-header">
            <span class="c-icon">{c.icon}</span>
            <span class="c-label">{c.label}</span>
            {#if !hasDimData(c.key)}
              <button class="c-assess" onclick={() => startAssessment(c.instrument)}>
                Assess →
              </button>
            {/if}
          </div>
          <div class="c-bars">
            {#each c.dims as dim}
              {@const val = dimValue(c.key, dim.code)}
              <div class="dim-row">
                <span class="dim-name">{dim.name}</span>
                <div class="dim-bar">
                  <div class="dim-fill" style="width:{val*10}%"></div>
                </div>
                <span class="dim-val">{val > 0 ? val.toFixed(1) : '—'}</span>
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>

    <!-- Complement Vector -->
    {#if complement}
      <div class="complement-section">
        <h3>🤖 Wirebot Effort Allocation</h3>
        <p class="comp-desc">Where Wirebot focuses to complement your gaps</p>
        <div class="comp-bars">
          {#each complementTop() as item}
            <div class="comp-row">
              <span class="comp-name">{item.name}</span>
              <div class="comp-bar">
                <div class="comp-fill" style="width:{item.allocation * 100}%"></div>
              </div>
              <span class="comp-pct">{(item.allocation * 100).toFixed(0)}%</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Self-Perception Gaps -->
    {#if insights?.self_perception_gaps && Object.keys(insights.self_perception_gaps).length > 0}
      <div class="gaps-section">
        <h3>🪞 Self-Perception Gaps</h3>
        <p class="gaps-desc">Where your self-assessment differs from observed behavior</p>
        {#each Object.entries(insights.self_perception_gaps) as [dim, info]}
          <div class="gap-item" class:overestimate={info.delta > 0} class:underestimate={info.delta < 0}>
            <span class="gap-dim">{dim.replace('_', ' ')}</span>
            <span class="gap-delta">{info.delta > 0 ? '+' : ''}{info.delta.toFixed(1)}</span>
            <span class="gap-interp">{info.interpretation}</span>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Active Contexts -->
    {#if insights?.active_contexts?.length > 0}
      <div class="contexts-section">
        <h3>🎯 Active Contexts</h3>
        {#each insights.active_contexts as ctx}
          <div class="ctx-item">
            <span class="ctx-name">{ctx.window.replace('_', ' ')}</span>
            <div class="ctx-bar">
              <div class="ctx-fill" style="width: {ctx.activation * 100}%"></div>
            </div>
            <span class="ctx-desc">{ctx.description}</span>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Recent Evidence -->
    {#if evidence?.evidence?.length > 0}
      <div class="evidence-section">
        <h3>📊 Recent Signals</h3>
        <div class="evidence-list">
          {#each evidence.evidence as ev}
            <div class="ev-item">
              <span class="ev-icon">
                {ev.signal_type === 'message' ? '💬' :
                 ev.signal_type === 'event' ? '📦' :
                 ev.signal_type === 'assessment' ? '📝' :
                 ev.signal_type === 'approval' ? '✅' : '🔹'}
              </span>
              <span class="ev-summary">{ev.summary}</span>
              <span class="ev-time">{timeAgo(ev.timestamp)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Accuracy Trajectory -->
    {#if accuracy}
      <div class="accuracy-section">
        <h3>📈 Accuracy Trajectory</h3>
        <div class="traj-grid">
          <div class="traj-item">
            <span class="traj-label">Day 1</span>
            <span class="traj-val">35%</span>
          </div>
          <div class="traj-item">
            <span class="traj-label">Day 7</span>
            <span class="traj-val">50%</span>
          </div>
          <div class="traj-item current">
            <span class="traj-label">Now</span>
            <span class="traj-val">{(accuracy.overall_accuracy * 100).toFixed(0)}%</span>
          </div>
          <div class="traj-item">
            <span class="traj-label">Day 30</span>
            <span class="traj-val">72%</span>
          </div>
          <div class="traj-item">
            <span class="traj-label">Day 90</span>
            <span class="traj-val">88%</span>
          </div>
        </div>
        {#if accuracy.improvements?.length > 0}
          <div class="improve-list">
            <h4>🚀 Boost accuracy:</h4>
            {#each accuracy.improvements as imp}
              <div class="improve-item">
                <span>{imp.action}</span>
                <span class="imp-boost">{imp.boost}</span>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}

    <!-- Active Contexts -->
    {#if effective?.active_contexts?.length > 0}
      <div class="contexts-section">
        <h3>🎯 Active Contexts</h3>
        {#each effective.active_contexts as ctx}
          <div class="ctx-item">
            <div class="ctx-name">{ctx.window}</div>
            <div class="ctx-desc">{ctx.description}</div>
            <div class="ctx-bar">
              <div class="ctx-fill" style="width:{ctx.activation * 100}%"></div>
            </div>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Calibration Preview -->
    {#if effective?.calibration}
      <div class="calibration-section">
        <h3>🎛️ Calibration</h3>
        <div class="cal-grid">
          <div class="cal-item">
            <span class="cal-k">Lead with</span>
            <span class="cal-v">{effective.calibration.communication.lead_with}</span>
          </div>
          <div class="cal-item">
            <span class="cal-k">Tone</span>
            <span class="cal-v">{effective.calibration.communication.tone_formality < 0.3 ? 'Casual' :
              effective.calibration.communication.tone_formality > 0.7 ? 'Formal' : 'Balanced'}</span>
          </div>
          <div class="cal-item">
            <span class="cal-k">Nudge every</span>
            <span class="cal-v">{effective.calibration.accountability.nudge_frequency_hours}h</span>
          </div>
          <div class="cal-item">
            <span class="cal-k">Options shown</span>
            <span class="cal-v">{effective.calibration.recommendations.options_presented}</span>
          </div>
          <div class="cal-item">
            <span class="cal-k">Peak task</span>
            <span class="cal-v">{effective.calibration.proactive.peak_task_type.replace('_',' ')}</span>
          </div>
          <div class="cal-item">
            <span class="cal-k">Standup</span>
            <span class="cal-v">{effective.calibration.proactive.standup_hour}:00</span>
          </div>
        </div>
      </div>
    {/if}
  </div>
{/if}



<style>
  .profile { padding: 16px 16px 80px; max-width: 600px; margin: 0 auto; }
  .profile-loading, .profile-error { padding: 40px; text-align: center; color: var(--text-secondary); }
  .pulse { font-size: 40px; animation: pulse 1.5s ease-in-out infinite; }
  @keyframes pulse { 0%,100% { opacity: 0.5; } 50% { opacity: 1; } }
  .profile-error button { margin-top: 12px; padding: 8px 20px; background: var(--bg-elevated); color: var(--text); border: 1px solid var(--border-light); border-radius: 8px; }

  .p-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
  .p-header h2 { font-size: 18px; margin: 0; color: var(--text); }
  .p-meta { display: flex; gap: 10px; font-size: 11px; color: var(--text-secondary); }
  .accuracy { color: var(--accent); }

  /* Pairing Score Ring */
  .score-card {
    display: flex; align-items: center; gap: 16px;
    padding: 16px; margin-bottom: 20px;
    background: linear-gradient(135deg, rgba(124,124,255,0.1), rgba(124,124,255,0.05));
    border: 1px solid rgba(124,124,255,0.2); border-radius: 12px;
  }
  .sc-ring {
    width: 64px; height: 64px; border-radius: 50%;
    background: conic-gradient(#7c7cff calc(var(--pct) * 1%), #222 0);
    display: flex; align-items: center; justify-content: center;
    position: relative;
  }
  .sc-ring::before {
    content: ''; position: absolute; inset: 6px; border-radius: 50%; background: var(--bg-card);
  }
  .sc-num { position: relative; z-index: 1; font-size: 20px; font-weight: 700; color: var(--text); }
  .sc-info { flex: 1; }
  .sc-label { font-size: 12px; color: var(--text-secondary); margin-bottom: 2px; }
  .sc-level { font-size: 16px; font-weight: 600; color: var(--text); }
  .sc-hint { font-size: 11px; color: var(--accent); margin-top: 4px; }

  /* Constructs */
  .constructs { display: flex; flex-direction: column; gap: 16px; margin-bottom: 20px; }
  .construct {
    background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.08);
    border-radius: 12px; padding: 12px;
  }
  .c-header { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
  .c-icon { font-size: 16px; }
  .c-label { font-size: 14px; font-weight: 600; color: var(--text); flex: 1; }
  .c-assess {
    font-size: 11px; padding: 4px 10px; border-radius: 6px;
    background: rgba(124,124,255,0.15); color: var(--accent);
    border: 1px solid rgba(124,124,255,0.3); cursor: pointer;
  }

  .c-bars { display: flex; flex-direction: column; gap: 6px; }
  .dim-row { display: flex; align-items: center; gap: 8px; }
  .dim-name { font-size: 11px; color: var(--text-secondary); width: 90px; flex-shrink: 0; }
  .dim-bar { flex: 1; height: 8px; background: rgba(255,255,255,0.06); border-radius: 4px; overflow: hidden; }
  .dim-fill { height: 100%; background: var(--accent); border-radius: 4px; transition: width 0.6s ease; }
  .dim-val { font-size: 11px; color: var(--text); width: 28px; text-align: right; font-variant-numeric: tabular-nums; }

  /* Complement */
  .complement-section { margin-bottom: 20px; }
  .complement-section h3 { font-size: 14px; color: var(--text); margin: 0 0 4px; }
  .comp-desc { font-size: 11px; color: var(--text-secondary); margin: 0 0 10px; }
  .comp-bars { display: flex; flex-direction: column; gap: 6px; }
  .comp-row { display: flex; align-items: center; gap: 8px; }
  .comp-name { font-size: 11px; color: var(--text-secondary); width: 100px; flex-shrink: 0; }
  .comp-bar { flex: 1; height: 8px; background: rgba(255,255,255,0.06); border-radius: 4px; overflow: hidden; }
  .comp-fill { height: 100%; background: linear-gradient(90deg, #7c7cff, #ff7cff); border-radius: 4px; transition: width 0.6s ease; }
  .comp-pct { font-size: 11px; color: var(--text); width: 28px; text-align: right; }

  /* Evidence */
  .evidence-section { margin-bottom: 20px; }
  .evidence-section h3 { font-size: 14px; color: var(--text); margin: 0 0 10px; }
  .evidence-list { display: flex; flex-direction: column; gap: 4px; }
  .ev-item { display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,0.05); }
  .ev-icon { font-size: 14px; }
  .ev-summary { flex: 1; font-size: 12px; color: var(--text); }
  .ev-time { font-size: 10px; color: #666; }

  /* Self-Perception Gaps */
  .gaps-section { margin-bottom: 20px; }
  .gaps-section h3 { font-size: 14px; color: var(--text); margin: 0 0 4px; }
  .gaps-desc { font-size: 11px; color: var(--text-secondary); margin: 0 0 8px; }
  .gap-item { display: flex; align-items: center; gap: 8px; padding: 6px 0; border-bottom: 1px solid #222; }
  .gap-dim { font-size: 12px; color: var(--text); min-width: 100px; }
  .gap-delta { font-size: 14px; font-weight: 600; min-width: 40px; text-align: right; }
  .overestimate .gap-delta { color: #ff6b6b; }
  .underestimate .gap-delta { color: #51cf66; }
  .gap-interp { font-size: 11px; color: var(--text-secondary); }

  /* Active Contexts */
  .contexts-section { margin-bottom: 20px; }
  .contexts-section h3 { font-size: 14px; color: var(--text); margin: 0 0 8px; }
  .ctx-item { margin-bottom: 10px; }
  .ctx-name { font-size: 11px; color: #aaa; text-transform: uppercase; letter-spacing: 0.5px; }
  .ctx-bar { height: 6px; background: #222; border-radius: 3px; margin: 4px 0; overflow: hidden; }
  .ctx-fill { height: 100%; background: linear-gradient(90deg, #7c7cff, #ff7cff); border-radius: 3px; transition: width 0.3s; }
  .ctx-desc { font-size: 11px; color: #666; }

  /* Accuracy */
  .accuracy-section { margin-bottom: 20px; }
  .accuracy-section h3 { font-size: 14px; color: var(--text); margin: 0 0 10px; }
  .traj-grid { display: flex; gap: 0; margin-bottom: 12px; }
  .traj-item {
    flex: 1; text-align: center; padding: 8px 4px;
    border-bottom: 2px solid rgba(255,255,255,0.08);
  }
  .traj-item.current { border-bottom-color: var(--accent); }
  .traj-label { display: block; font-size: 10px; color: #666; }
  .traj-val { display: block; font-size: 14px; font-weight: 600; color: var(--text); }
  .traj-item.current .traj-val { color: var(--accent); }

  .improve-list h4 { font-size: 12px; color: var(--text-secondary); margin: 0 0 6px; }
  .improve-item { display: flex; justify-content: space-between; font-size: 12px; color: var(--text); padding: 4px 0; }
  .imp-boost { color: #7cff7c; }

  /* Contexts */
  .contexts-section { margin-bottom: 20px; }
  .contexts-section h3 { font-size: 14px; color: var(--text); margin: 0 0 10px; }
  .ctx-item { margin-bottom: 8px; }
  .ctx-name { font-size: 12px; font-weight: 600; color: #ffd700; }
  .ctx-desc { font-size: 11px; color: var(--text-secondary); margin: 2px 0 4px; }
  .ctx-bar { height: 4px; background: rgba(255,255,255,0.06); border-radius: 2px; overflow: hidden; }
  .ctx-fill { height: 100%; background: #ffd700; border-radius: 2px; }

  /* Calibration */
  .calibration-section { margin-bottom: 20px; }
  .calibration-section h3 { font-size: 14px; color: var(--text); margin: 0 0 10px; }
  .cal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
  .cal-item { display: flex; justify-content: space-between; padding: 6px 8px; background: rgba(255,255,255,0.03); border-radius: 6px; }
  .cal-k { font-size: 11px; color: var(--text-secondary); }
  .cal-v { font-size: 11px; color: var(--text); font-weight: 500; }

  /* Flow placeholder */
  .flow-placeholder { padding: 40px; text-align: center; color: var(--text-secondary); }
  .back-btn { background: none; border: none; color: var(--accent); font-size: 14px; cursor: pointer; margin-bottom: 20px; }
</style>
