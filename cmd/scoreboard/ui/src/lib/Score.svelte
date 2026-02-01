<script>
  let { data, lastUpdate } = $props();

  function signalClass(s) {
    return s === 'green' ? 'sig-g' : s === 'yellow' ? 'sig-y' : 'sig-r';
  }
  function signalLabel(s) {
    return s === 'green' ? 'WINNING' : s === 'yellow' ? 'PRESSURE' : 'STALLING';
  }
  function pct(v) { return Math.round((v || 0) * 100); }
  function lanePct(v, max) { return max ? Math.round((v / max) * 100) : 0; }
</script>

<div class="score-view">
  <!-- Stall Alert -->
  {#if data.stall_hours > 24}
    <div class="stall-alert">
      ⚠️ NO SHIP IN {Math.floor(data.stall_hours)}H — STALLING
    </div>
  {/if}

  <!-- Intent Bar -->
  {#if data.intent}
    <div class="intent">
      🎯 <span>{data.intent}</span>
    </div>
  {:else}
    <div class="intent empty">
      🎯 <span>No intent declared — <code>wb intent "..."</code></span>
    </div>
  {/if}

  <!-- Header -->
  <div class="hdr">
    <span class="sn">{data.season?.name?.toUpperCase() || 'SCOREBOARD'}</span>
    <span class="sd">{data.season_day || ''}</span>
  </div>

  <!-- Score -->
  <div class="sc {signalClass(data.signal)}">
    <div class="sc-lbl">EXECUTION SCORE</div>
    <div class="sc-num">{data.score}</div>
    <div class="sc-sub">{signalLabel(data.signal)}</div>
  </div>

  <!-- Stats -->
  <div class="stats">
    <div class="st"><span class="st-v">🔥 {data.streak?.current || 0}</span><span class="st-l">STREAK</span></div>
    <div class="st"><span class="st-v">🏆 {data.streak?.best || 0}</span><span class="st-l">BEST</span></div>
    <div class="st"><span class="st-v">{data.record || '0-0'}</span><span class="st-l">W-L</span></div>
    <div class="st"><span class="st-v">🚀 {data.ship_today || 0}</span><span class="st-l">SHIPS</span></div>
  </div>

  <!-- Possession -->
  <div class="pos">⚡ <strong>{data.possession || '—'}</strong></div>

  <!-- Lanes -->
  <div class="lanes">
    {#each [
      ['SHIPPING', data.lanes?.shipping || 0, data.lanes?.shipping_max || 40, '#4a9eff'],
      ['DISTRIB', data.lanes?.distribution || 0, data.lanes?.distribution_max || 25, '#9b59b6'],
      ['REVENUE', data.lanes?.revenue || 0, data.lanes?.revenue_max || 20, '#2ecc71'],
      ['SYSTEMS', data.lanes?.systems || 0, data.lanes?.systems_max || 15, '#e67e22'],
    ] as [name, val, max, color]}
      <div class="ln">
        <span class="ln-n">{name}</span>
        <div class="ln-track">
          <div class="ln-fill" style="width:{lanePct(val, max)}%; background:{color}"></div>
        </div>
        <span class="ln-v">{val}<span class="ln-max">/{max}</span></span>
      </div>
    {/each}
  </div>

  <!-- Modifiers -->
  {#if data.streak_bonus > 0 || data.penalties > 0}
    <div class="mods">
      {#if data.streak_bonus > 0}
        <span class="mod bonus">🔥 +{data.streak_bonus} streak bonus</span>
      {/if}
      {#if data.penalties > 0}
        <span class="mod penalty">⚠️ -{data.penalties} penalties</span>
      {/if}
    </div>
  {/if}

  <!-- Last Ship -->
  {#if data.last_ship}
    <div class="ls">🚀 {data.last_ship}</div>
  {/if}

  <!-- Clocks -->
  <div class="clk">
    {#each [
      ['DAY', data.clock?.day_progress, false],
      ['WEEK', data.clock?.week_progress, false],
      ['SEASON', data.clock?.season_progress, true],
    ] as [label, progress, isSeason]}
      <div class="ck">
        <span class="ck-l">{label}</span>
        <div class="ck-track"><div class="ck-fill {isSeason ? 'ck-season' : ''}" style="width:{pct(progress)}%"></div></div>
        <span class="ck-p">{pct(progress)}%</span>
      </div>
    {/each}
  </div>

  <div class="ft">{data.season?.theme || ''}</div>
</div>

<style>
  .score-view {
    display: flex;
    flex-direction: column;
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    gap: 10px;
    min-height: calc(100dvh - 56px);
    background: linear-gradient(180deg, #0a0a1a 0%, #0d0d20 50%, #0a0a1a 100%);
  }

  /* Stall Alert */
  .stall-alert {
    background: rgba(255,50,50,0.15);
    color: #ff4444;
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
    color: #aaa;
    display: flex;
    align-items: center;
    gap: 6px;
  }
  .intent span { flex: 1; }
  .intent.empty { opacity: 0.5; border-style: dashed; }
  .intent code { font-family: monospace; font-size: 11px; color: #7c7cff; }

  /* Header */
  .hdr {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid #1e1e30;
    padding-bottom: 6px;
  }
  .sn { font-size: 12px; font-weight: 700; letter-spacing: .15em; color: #7c7cff; }
  .sd { font-size: 12px; color: #555; }

  /* Score */
  .sc { text-align: center; padding: 12px 0; border-radius: 12px; }
  .sc-lbl { font-size: 10px; letter-spacing: .3em; opacity: .5; }
  .sc-num { font-size: 80px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
  .sc-sub { font-size: 13px; letter-spacing: .25em; margin-top: 4px; }

  .sig-g { background: rgba(0,255,100,.06); }
  .sig-g .sc-num { color: #00ff64; text-shadow: 0 0 30px rgba(0,255,100,.2); }
  .sig-g .sc-sub { color: #00cc50; }
  .sig-y { background: rgba(255,200,0,.06); }
  .sig-y .sc-num { color: #ffc800; text-shadow: 0 0 30px rgba(255,200,0,.2); }
  .sig-y .sc-sub { color: #cc9900; }
  .sig-r { background: rgba(255,50,50,.06); }
  .sig-r .sc-num { color: #ff3232; text-shadow: 0 0 30px rgba(255,50,50,.2); animation: pulse 2s infinite; }
  .sig-r .sc-sub { color: #cc2020; }
  @keyframes pulse { 0%,100%{opacity:1} 50%{opacity:.6} }

  /* Stats */
  .stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 4px;
    border-top: 1px solid #1e1e30;
    border-bottom: 1px solid #1e1e30;
    padding: 8px 0;
  }
  .st { text-align: center; }
  .st-v { display: block; font-size: 15px; font-weight: 700; white-space: nowrap; }
  .st-l { display: block; font-size: 8px; letter-spacing: .1em; opacity: .35; margin-top: 1px; }

  .pos { text-align: center; font-size: 12px; color: #888; }
  .pos strong { color: #7c7cff; font-size: 16px; }

  /* Lanes */
  .lanes { display: flex; flex-direction: column; gap: 8px; }
  .ln { display: flex; align-items: center; gap: 8px; }
  .ln-n { font-size: 10px; width: 56px; opacity: .45; flex-shrink: 0; }
  .ln-track { flex: 1; height: 10px; background: #1a1a2e; border-radius: 5px; overflow: hidden; }
  .ln-fill { height: 100%; border-radius: 5px; transition: width .8s ease; min-width: 2px; }
  .ln-v { font-size: 12px; font-variant-numeric: tabular-nums; opacity: .6; min-width: 40px; text-align: right; flex-shrink: 0; }
  .ln-max { opacity: .4; font-size: 10px; }

  /* Modifiers */
  .mods { display: flex; justify-content: center; gap: 12px; font-size: 11px; }
  .mod { padding: 3px 8px; border-radius: 6px; }
  .bonus { background: rgba(0,255,100,.08); color: #00cc50; }
  .penalty { background: rgba(255,50,50,.08); color: #ff4444; }

  .ls { text-align: center; font-size: 11px; color: #4a9eff; opacity: .65; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  /* Clocks */
  .clk { display: flex; flex-direction: column; gap: 5px; margin-top: auto; padding-top: 8px; border-top: 1px solid #1e1e30; }
  .ck { display: flex; align-items: center; gap: 6px; }
  .ck-l { font-size: 9px; width: 44px; opacity: .35; letter-spacing: .1em; flex-shrink: 0; }
  .ck-track { flex: 1; height: 5px; background: #1a1a2e; border-radius: 3px; overflow: hidden; }
  .ck-fill { height: 100%; background: linear-gradient(90deg, #4a9eff, #7c7cff); border-radius: 3px; transition: width 1s ease; }
  .ck-season { background: linear-gradient(90deg, #ff6b4a, #ff4a9e); }
  .ck-p { font-size: 10px; opacity: .4; min-width: 28px; text-align: right; font-variant-numeric: tabular-nums; flex-shrink: 0; }

  .ft { text-align: center; font-size: 10px; opacity: .2; padding-bottom: 4px; }

  @media (min-width: 600px) {
    .score-view { padding: 20px 32px; gap: 14px; }
    .sc-num { font-size: 120px; }
    .st-v { font-size: 18px; }
    .ln-track { height: 14px; }
  }

  @media (min-width: 1024px) {
    .score-view { padding: 3vh 5vw; gap: 2vh; font-family: 'SF Mono', monospace; }
    .sc-num { font-size: 16vh; }
    .st-v { font-size: 3vh; }
  }
</style>
