<script>
  import Tooltip from './Tooltip.svelte';
  import AnimatedCounter from './AnimatedCounter.svelte';
  let { data, lastUpdate, onHelp, user = null, onPairing = null, onShare = null, canShare = false } = $props();

  let editingIntent = $state(false);
  let intentDraft = $state('');
  let intentSaving = $state(false);

  function signalClass(s) {
    return s === 'green' ? 'sig-g' : s === 'yellow' ? 'sig-y' : 'sig-r';
  }
  function signalLabel(s) {
    return s === 'green' ? 'WINNING' : s === 'yellow' ? 'PRESSURE' : 'STALLING';
  }
  function pct(v) { return Math.round((v || 0) * 100); }
  function lanePct(v, max) { return max ? Math.round((v / max) * 100) : 0; }

  function startEditIntent() {
    intentDraft = data.intent || '';
    editingIntent = true;
    setTimeout(() => {
      const el = document.querySelector('.intent-input');
      if (el) { el.focus(); el.select(); }
    }, 50);
  }

  async function saveIntent() {
    const text = intentDraft.trim();
    if (!text) return;
    intentSaving = true;
    try {
      const token = localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || '';
      const resp = await fetch('/v1/intent', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? (token.startsWith('Bearer ') ? token : `Bearer ${token}`) : '',
        },
        body: JSON.stringify({ intent: text }),
      });
      if (resp.ok) {
        data.intent = text;
        editingIntent = false;
      }
    } catch { /* silent */ }
    intentSaving = false;
  }

  function intentKeydown(e) {
    if (e.key === 'Enter') { e.preventDefault(); saveIntent(); }
    if (e.key === 'Escape') { editingIntent = false; }
  }
</script>

<div class="score-view">
  <!-- User Identity + Share -->
  <div class="top-bar">
    {#if user}
      <div class="user-bar">
        {#if user.avatar_url}<img class="ub-avatar" src={user.avatar_url} alt="" />{/if}
        <span class="ub-name">{user.display_name}</span>
        <span class="ub-tier tier-{user.tier}">{user.tier}</span>
        {#if user.is_admin}<span class="ub-admin">admin</span>{/if}
      </div>
    {:else}
      <div></div>
    {/if}
    {#if onShare}
      <button class="share-btn" onclick={onShare} title="Share your score">
        📤 {canShare ? 'Share' : 'Copy'}
      </button>
    {/if}
  </div>

  <!-- Stall Alert -->
  {#if data.stall_hours > 24}
    <div class="stall-alert">
      ⚠️ NO SHIP IN {Math.floor(data.stall_hours)}H — STALLING
    </div>
  {/if}

  <!-- Intent Bar (tap to edit) -->
  {#if editingIntent}
    <div class="intent editing">
      <span>🎯</span>
      <input
        class="intent-input"
        type="text"
        placeholder="What are you shipping today?"
        bind:value={intentDraft}
        onkeydown={intentKeydown}
        disabled={intentSaving}
      />
      <button class="intent-save" onclick={saveIntent} disabled={intentSaving || !intentDraft.trim()}>
        {intentSaving ? '...' : '✓'}
      </button>
      <button class="intent-cancel" onclick={() => editingIntent = false}>✕</button>
    </div>
  {:else if data.intent}
    <div class="intent" role="button" tabindex="0" onclick={startEditIntent} onkeydown={(e) => e.key === 'Enter' && startEditIntent()}>
      <Tooltip concept="intent">🎯</Tooltip> <span>{data.intent}</span>
      <span class="intent-edit-hint">tap to edit</span>
    </div>
  {:else}
    <div class="intent empty" role="button" tabindex="0" onclick={startEditIntent} onkeydown={(e) => e.key === 'Enter' && startEditIntent()}>
      <Tooltip concept="intent">🎯</Tooltip> <span>Tap to set today's intent</span>
    </div>
  {/if}

  <!-- Header -->
  <div class="hdr">
    <span class="sn">{data.season?.name?.toUpperCase() || 'SCOREBOARD'}</span>
    <span class="hdr-right">
      <span class="sd">{data.season_day || ''}</span>
      <button class="hdr-help" onclick={onHelp} title="How it works">?</button>
    </span>
  </div>

  <!-- Score Ring -->
  <div class="sc {signalClass(data.signal)}">
    <div class="sc-lbl"><Tooltip concept="score">EXECUTION SCORE</Tooltip></div>
    <div class="sc-ring-wrap">
      <svg class="sc-ring" viewBox="0 0 200 200">
        <defs>
          <linearGradient id="scoreGrad" x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" style="stop-color:{data.signal === 'green' ? '#00ff64' : data.signal === 'yellow' ? '#ffc800' : '#ff3232'}" />
            <stop offset="100%" style="stop-color:{data.signal === 'green' ? '#4a9eff' : data.signal === 'yellow' ? '#ff9500' : '#ff6666'}" />
          </linearGradient>
          <filter id="glow">
            <feGaussianBlur stdDeviation="3" result="blur"/>
            <feMerge><feMergeNode in="blur"/><feMergeNode in="SourceGraphic"/></feMerge>
          </filter>
        </defs>
        <circle cx="100" cy="100" r="88" fill="none" stroke="var(--border)" stroke-width="6"/>
        <circle cx="100" cy="100" r="88" fill="none" stroke="url(#scoreGrad)" stroke-width="6"
          stroke-linecap="round" filter="url(#glow)"
          stroke-dasharray="{553}" stroke-dashoffset="{553 - (553 * (data.score || 0) / 100)}"
          transform="rotate(-90 100 100)"
          class="sc-ring-fill"/>
      </svg>
      <div class="sc-ring-center">
        <div class="sc-num"><AnimatedCounter value={data.score || 0} duration={800} /></div>
        <div class="sc-sub"><Tooltip concept="signal">{signalLabel(data.signal)}</Tooltip></div>
      </div>
    </div>
  </div>

  <!-- Drift Indicator -->
  {#if data.drift}
    <div class="drift-bar" class:drift-deep={data.drift.signal === 'deep_sync'}
         class:drift-in={data.drift.signal === 'in_drift'}
         class:drift-weak={data.drift.signal === 'weak' || data.drift.signal === 'disconnected'}>
      <span class="drift-label">🧠 NEURAL DRIFT</span>
      <div class="drift-track">
        <div class="drift-fill" style="width:{data.drift.score || 0}%"></div>
      </div>
      <span class="drift-val">{data.drift.score || 0}%</span>
    </div>
    {#if data.drift.rabbit?.active}
      <div class="rabbit-alert">
        🐇 R.A.B.I.T. — {data.drift.rabbit.message}
      </div>
    {/if}
  {/if}

  <!-- Stats (glass cards) -->
  <div class="stats">
    <div class="st glass"><span class="st-v">🔥 <AnimatedCounter value={data.streak?.current || 0} duration={500} /></span><span class="st-l"><Tooltip concept="streak" position="below">STREAK</Tooltip></span></div>
    <div class="st glass"><span class="st-v">🏆 <AnimatedCounter value={data.streak?.best || 0} duration={500} /></span><span class="st-l">BEST</span></div>
    <div class="st glass"><span class="st-v">{data.record || '0-0'}</span><span class="st-l"><Tooltip concept="record" position="below">W-L</Tooltip></span></div>
    <div class="st glass"><span class="st-v">🚀 <AnimatedCounter value={data.ship_today || 0} duration={400} /></span><span class="st-l"><Tooltip concept="ships" position="below">SHIPS</Tooltip></span></div>
  </div>

  <!-- Possession -->
  <div class="pos"><Tooltip concept="possession">⚡</Tooltip> <strong>{data.possession || '—'}</strong></div>

  <!-- Lanes -->
  <div class="lanes">
    {#each [
      ['SHIPPING', data.lanes?.shipping || 0, data.lanes?.shipping_max || 40, '#4a9eff'],
      ['DISTRIB', data.lanes?.distribution || 0, data.lanes?.distribution_max || 25, '#9b59b6'],
      ['REVENUE', data.lanes?.revenue || 0, data.lanes?.revenue_max || 20, '#2ecc71'],
      ['SYSTEMS', data.lanes?.systems || 0, data.lanes?.systems_max || 15, '#e67e22'],
    ] as [name, val, max, color]}
      <div class="ln" class:ln-maxed={val >= max}>
        <span class="ln-n">{name}</span>
        <div class="ln-track">
          <div class="ln-fill" style="width:{lanePct(val, max)}%; background:{color};{val >= max ? `box-shadow: 0 0 12px ${color}40, 0 0 4px ${color}60;` : ''}"></div>
        </div>
        <span class="ln-v"><AnimatedCounter value={val} duration={600} />{#if val >= max}<span class="ln-max-badge">MAX</span>{:else}<span class="ln-max">/{max}</span>{/if}</span>
      </div>
    {/each}
  </div>

  <!-- Modifiers -->
  {#if data.streak_bonus > 0 || data.penalties > 0}
    <div class="mods">
      {#if data.streak_bonus > 0}
        <Tooltip concept="bonus"><span class="mod bonus">🔥 +{data.streak_bonus} streak bonus</span></Tooltip>
      {/if}
      {#if data.penalties > 0}
        <Tooltip concept="penalty"><span class="mod penalty">⚠️ -{data.penalties} penalties</span></Tooltip>
      {/if}
    </div>
  {/if}

  <!-- Pairing nudge (if profile score is low and callback is provided) -->
  {#if onPairing}
    <button class="pairing-cta" onclick={onPairing}>
      <span class="pc-icon">🧬</span>
      <span class="pc-text">Calibrate your profile</span>
      <span class="pc-arrow">→</span>
    </button>
  {/if}

  <!-- Last Ship -->
  {#if data.last_ship}
    <div class="ls">🚀 {data.last_ship}</div>
  {/if}

  <!-- Clocks -->
  <div class="clk">
    <div class="ck">
      <span class="ck-l"><Tooltip concept="clock_day" position="above">DAY</Tooltip></span>
      <div class="ck-track"><div class="ck-fill" style="width:{pct(data.clock?.day_progress)}%"></div></div>
      <span class="ck-p">{pct(data.clock?.day_progress)}%</span>
    </div>
    <div class="ck">
      <span class="ck-l"><Tooltip concept="clock_week" position="above">WEEK</Tooltip></span>
      <div class="ck-track"><div class="ck-fill" style="width:{pct(data.clock?.week_progress)}%"></div></div>
      <span class="ck-p">{pct(data.clock?.week_progress)}%</span>
    </div>
    <div class="ck">
      <span class="ck-l"><Tooltip concept="clock_season" position="above">SEASON</Tooltip></span>
      <div class="ck-track"><div class="ck-fill ck-season" style="width:{pct(data.clock?.season_progress)}%"></div></div>
      <span class="ck-p">{pct(data.clock?.season_progress)}%</span>
    </div>
  </div>

  <div class="ft">{data.season?.theme || ''}</div>
</div>

<style>
  /* Top bar with user + share */
  .top-bar {
    display: flex; justify-content: space-between; align-items: center;
    padding-bottom: 6px; border-bottom: 1px solid var(--border); margin-bottom: -4px;
  }
  .share-btn {
    background: rgba(124,124,255,0.1); border: 1px solid rgba(124,124,255,0.3);
    color: var(--accent); padding: 4px 10px; border-radius: 6px;
    font-size: 12px; font-weight: 600; cursor: pointer;
    transition: all 0.2s ease;
  }
  .share-btn:hover { background: rgba(124,124,255,0.2); }
  .share-btn:active { transform: scale(0.95); }

  /* User bar */
  .user-bar {
    display: flex; align-items: center; gap: 8px;
  }
  .ub-avatar { width: 24px; height: 24px; border-radius: 50%; }
  .ub-name { font-size: 13px; font-weight: 600; color: var(--text-secondary); }
  .ub-tier {
    font-size: 9px; font-weight: 700; padding: 1px 6px; border-radius: 3px;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .ub-tier.tier-free { background: #222; color: var(--text-secondary); }
  .ub-tier.tier-freewire { background: var(--badge-freewire-bg); color: var(--success); }
  .ub-tier.tier-wire { background: var(--bg-elevated); color: var(--accent); }
  .ub-tier.tier-extrawire { background: var(--badge-extrawire-bg); color: #d946ef; }
  .ub-admin { font-size: 9px; font-weight: 700; color: var(--badge-admin-text); background: var(--badge-admin-bg); padding: 1px 6px; border-radius: 3px; }

  .score-view {
    display: flex;
    flex-direction: column;
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    gap: 10px;
    min-height: calc(100dvh - 56px);
    background: var(--page-gradient);
    animation: fadeInUp 0.4s ease-out;
  }
  @keyframes fadeInUp {
    from { opacity: 0; transform: translateY(12px); }
    to { opacity: 1; transform: translateY(0); }
  }

  /* Stall Alert */
  .stall-alert {
    background: rgba(255,50,50,0.15);
    color: var(--error);
    text-align: center;
    padding: 8px;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0.1em;
    animation: blink 1.5s infinite;
  }
  @keyframes blink { 0%,100%{opacity:1} 50%{opacity:0.5} }

  /* Intent */
  .intent {
    background: rgba(124,124,255,0.08);
    border: 1px solid rgba(124,124,255,0.2);
    border-radius: 8px;
    padding: 8px 12px;
    font-size: 13px;
    color: var(--text-secondary);
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .intent span { flex: 1; }
  .intent:not(.editing) { cursor: pointer; -webkit-tap-highlight-color: transparent; }
  .intent:not(.editing):active { background: rgba(124,124,255,0.15); }
  .intent.empty { opacity: 0.5; border-style: dashed; }
  .intent-edit-hint { font-size: 10px; opacity: 0.3; flex: none; }
  .intent-input {
    flex: 1;
    background: transparent;
    border: none;
    outline: none;
    color: var(--text);
    font-size: 13px;
    padding: 0;
  }
  .intent-input::placeholder { color: var(--text-muted); }
  .intent-save, .intent-cancel {
    width: 28px; height: 28px;
    border-radius: 6px; border: none;
    font-size: 14px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }
  .intent-save { background: rgba(0,255,100,0.15); color: #00ff64; }
  .intent-save:disabled { opacity: 0.3; cursor: default; }
  .intent-cancel { background: rgba(255,50,50,0.1); color: var(--error); }

  /* Header */
  .hdr {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid var(--border);
    padding-bottom: 6px;
  }
  .sn { font-size: 12px; font-weight: 700; letter-spacing: .15em; color: var(--accent); }
  .hdr-right { display: flex; align-items: center; gap: 8px; }
  .sd { font-size: 12px; color: var(--text-muted); }
  .hdr-help {
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: rgba(124,124,255,0.1);
    border: 1px solid rgba(124,124,255,0.25);
    color: var(--accent);
    font-size: 12px;
    font-weight: 700;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    -webkit-tap-highlight-color: transparent;
    flex-shrink: 0;
  }

  /* Score Ring */
  .sc { text-align: center; padding: 8px 0; border-radius: 16px; position: relative; }
  .sc-lbl { font-size: 10px; letter-spacing: .3em; opacity: .5; margin-bottom: 4px; }
  .sc-ring-wrap { position: relative; width: 200px; height: 200px; margin: 0 auto; }
  .sc-ring { width: 100%; height: 100%; }
  .sc-ring-fill { transition: stroke-dashoffset 1.5s cubic-bezier(0.4, 0, 0.2, 1); }
  .sc-ring-center {
    position: absolute; inset: 0;
    display: flex; flex-direction: column;
    align-items: center; justify-content: center;
  }
  .sc-num { font-size: 64px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
  .sc-sub { font-size: 12px; letter-spacing: .25em; margin-top: 2px; }

  .sig-g { background: radial-gradient(ellipse at 50% 30%, rgba(0,255,100,.08) 0%, transparent 70%); }
  .sig-g .sc-num { color: #00ff64; text-shadow: 0 0 40px rgba(0,255,100,.3), 0 0 80px rgba(0,255,100,.1); }
  .sig-g .sc-sub { color: #00cc50; }
  .sig-y { background: radial-gradient(ellipse at 50% 30%, rgba(255,200,0,.08) 0%, transparent 70%); }
  .sig-y .sc-num { color: #ffc800; text-shadow: 0 0 40px rgba(255,200,0,.3), 0 0 80px rgba(255,200,0,.1); }
  .sig-y .sc-sub { color: #cc9900; }
  .sig-r { background: radial-gradient(ellipse at 50% 30%, rgba(255,50,50,.08) 0%, transparent 70%); }
  .sig-r .sc-num { color: #ff3232; text-shadow: 0 0 40px rgba(255,50,50,.3), 0 0 80px rgba(255,50,50,.1); animation: pulse 2s infinite; }
  .sig-r .sc-sub { color: #cc2020; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.6} }

  /* Drift Indicator */
  .drift-bar {
    display: flex; align-items: center; gap: 8px;
    padding: 6px 12px; border-radius: 10px;
    background: rgba(124,124,255,0.05);
    border: 1px solid rgba(124,124,255,0.12);
    transition: all 0.3s;
  }
  .drift-bar.drift-deep {
    background: rgba(0,255,100,0.06);
    border-color: rgba(0,255,100,0.2);
  }
  .drift-bar.drift-weak {
    background: rgba(255,50,50,0.06);
    border-color: rgba(255,50,50,0.15);
  }
  .drift-label {
    font-size: 9px; letter-spacing: 0.15em; opacity: 0.5;
    white-space: nowrap; flex-shrink: 0;
  }
  .drift-track {
    flex: 1; height: 6px; border-radius: 3px;
    background: rgba(255,255,255,0.05);
    overflow: hidden;
  }
  .drift-fill {
    height: 100%; border-radius: 3px;
    background: linear-gradient(90deg, #7c7cff, #00ff64);
    transition: width 1s ease;
  }
  .drift-deep .drift-fill { background: linear-gradient(90deg, #00ff64, #4aff9e); }
  .drift-weak .drift-fill { background: linear-gradient(90deg, #ff6666, #ff9500); }
  .drift-val {
    font-size: 12px; font-weight: 700; font-variant-numeric: tabular-nums;
    color: var(--accent); min-width: 32px; text-align: right; flex-shrink: 0;
  }
  .drift-deep .drift-val { color: #00ff64; }
  .drift-weak .drift-val { color: var(--error); }

  .rabbit-alert {
    background: rgba(255,180,0,0.1);
    border: 1px solid rgba(255,180,0,0.25);
    border-radius: 8px; padding: 8px 12px;
    font-size: 12px; color: #ffb400;
    animation: blink 1.5s infinite;
  }

  /* Stats — glassmorphism cards */
  .stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 6px;
    padding: 6px 0;
  }
  .st {
    text-align: center;
    padding: 8px 4px;
    border-radius: 12px;
    transition: transform 0.2s, box-shadow 0.3s;
  }
  .st.glass {
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.06);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
  }
  .st:active { transform: scale(0.95); }
  .st-v { display: block; font-size: 15px; font-weight: 700; white-space: nowrap; }
  .st-l { display: block; font-size: 8px; letter-spacing: .1em; opacity: .35; margin-top: 2px; }

  .pos { text-align: center; font-size: 12px; color: var(--text-secondary); }
  .pos strong { color: var(--accent); font-size: 16px; }

  /* Lanes */
  .lanes { display: flex; flex-direction: column; gap: 8px; }
  .ln {
    display: flex; align-items: center; gap: 8px;
    transition: transform 0.2s;
  }
  .ln:active { transform: translateX(4px); }
  .ln-n { font-size: 10px; width: 56px; opacity: .45; flex-shrink: 0; letter-spacing: 0.05em; }
  .ln-track {
    flex: 1; height: 10px; border-radius: 5px; overflow: hidden;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.03);
  }
  .ln-fill {
    height: 100%; border-radius: 5px;
    transition: width 1s cubic-bezier(0.4, 0, 0.2, 1);
    min-width: 2px;
  }
  .ln-v { font-size: 12px; font-variant-numeric: tabular-nums; opacity: .6; min-width: 40px; text-align: right; flex-shrink: 0; }
  .ln-max { opacity: .4; font-size: 10px; }
  .ln-maxed .ln-n { opacity: .8; font-weight: 600; }
  .ln-max-badge {
    font-size: 8px; font-weight: 800; letter-spacing: 0.1em;
    color: #00ff64; opacity: 0.8;
    margin-left: 2px;
    animation: maxGlow 2s infinite;
  }
  @keyframes maxGlow {
    0%, 100% { text-shadow: 0 0 4px rgba(0,255,100,0.4); }
    50% { text-shadow: 0 0 8px rgba(0,255,100,0.8); }
  }

  /* Modifiers */
  .mods { display: flex; justify-content: center; gap: 12px; font-size: 11px; }
  .mod { padding: 3px 8px; border-radius: 6px; }
  .bonus { background: rgba(0,255,100,.08); color: #00cc50; }
  .penalty { background: rgba(255,50,50,.08); color: var(--error); }

  .ls { text-align: center; font-size: 11px; color: var(--accent); opacity: .65; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .pairing-cta {
    display: flex; align-items: center; gap: 10px;
    width: 100%; padding: 12px 16px; margin: 8px 0;
    background: linear-gradient(135deg, rgba(124,124,255,0.08), rgba(255,124,255,0.04));
    border: 1px solid rgba(124,124,255,0.2);
    border-radius: 14px; cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    position: relative; overflow: hidden;
    transition: all 0.3s;
  }
  .pairing-cta::before {
    content: ''; position: absolute; inset: 0;
    background: linear-gradient(135deg, rgba(124,124,255,0.1), rgba(255,124,255,0.05));
    opacity: 0; transition: opacity 0.3s;
  }
  .pairing-cta:active::before { opacity: 1; }
  .pairing-cta:active { transform: scale(0.98); }
  .pc-icon { font-size: 18px; }
  .pc-text { flex: 1; font-size: 13px; color: var(--accent); text-align: left; font-weight: 600; letter-spacing: 0.02em; }
  .pc-arrow { font-size: 14px; color: var(--accent); opacity: 0.5; transition: transform 0.2s; }
  .pairing-cta:active .pc-arrow { transform: translateX(4px); }

  /* Clocks */
  .clk { display: flex; flex-direction: column; gap: 5px; margin-top: auto; padding-top: 8px; border-top: 1px solid var(--border); }
  .ck { display: flex; align-items: center; gap: 6px; }
  .ck-l { font-size: 9px; width: 44px; opacity: .35; letter-spacing: .1em; flex-shrink: 0; }
  .ck-track { flex: 1; height: 5px; background: var(--bg-elevated); border-radius: 3px; overflow: hidden; }
  .ck-fill { height: 100%; background: linear-gradient(90deg, #4a9eff, #7c7cff); border-radius: 3px; transition: width 1s ease; }
  .ck-season { background: linear-gradient(90deg, #ff6b4a, #ff4a9e); }
  .ck-p { font-size: 10px; opacity: .4; min-width: 28px; text-align: right; font-variant-numeric: tabular-nums; flex-shrink: 0; }

  .ft { text-align: center; font-size: 10px; opacity: .2; padding-bottom: 4px; }

  @media (min-width: 600px) {
    .score-view { padding: 20px 32px; gap: 14px; }
    .sc-ring-wrap { width: 260px; height: 260px; }
    .sc-num { font-size: 80px; }
    .st-v { font-size: 18px; }
    .ln-track { height: 14px; }
  }

  @media (min-width: 1024px) {
    .score-view { padding: 3vh 5vw; gap: 2vh; font-family: 'SF Mono', monospace; }
    .sc-ring-wrap { width: 320px; height: 320px; }
    .sc-num { font-size: 10vh; }
    .st-v { font-size: 3vh; }
  }
</style>
