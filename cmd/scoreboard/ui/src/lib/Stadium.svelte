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

<div class="s">
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

  <!-- Stats Grid -->
  <div class="stats">
    <div class="st"><span class="st-v">🔥 {data.streak?.current || 0}</span><span class="st-l">STREAK</span></div>
    <div class="st"><span class="st-v">🏆 {data.streak?.best || 0}</span><span class="st-l">BEST</span></div>
    <div class="st"><span class="st-v">{data.record || '0-0'}</span><span class="st-l">W-L</span></div>
    <div class="st"><span class="st-v">🚀 {data.ship_today || 0}</span><span class="st-l">SHIPS</span></div>
  </div>

  <!-- Possession -->
  <div class="pos">⚡ <strong>{data.possession || '—'}</strong></div>

  <!-- Lanes (CSS bars, not text) -->
  <div class="lanes">
    {#each [
      ['SHIPPING', data.lanes?.shipping || 0, data.lanes?.shipping_max || 40],
      ['DISTRIB', data.lanes?.distribution || 0, data.lanes?.distribution_max || 25],
      ['REVENUE', data.lanes?.revenue || 0, data.lanes?.revenue_max || 20],
      ['SYSTEMS', data.lanes?.systems || 0, data.lanes?.systems_max || 15],
    ] as [name, val, max]}
      <div class="ln">
        <span class="ln-n">{name}</span>
        <div class="ln-track">
          <div class="ln-fill" style="width:{lanePct(val, max)}%"></div>
        </div>
        <span class="ln-v">{val}/{max}</span>
      </div>
    {/each}
  </div>

  <!-- Last Ship -->
  {#if data.last_ship}
    <div class="ls">LAST SHIP: {data.last_ship}</div>
  {/if}

  <!-- Clocks -->
  <div class="clk">
    {#each [
      ['DAY', data.clock?.day_progress],
      ['WEEK', data.clock?.week_progress],
      ['SEASON', data.clock?.season_progress],
    ] as [label, progress], i}
      <div class="ck">
        <span class="ck-l">{label}</span>
        <div class="ck-track">
          <div class="ck-fill {i === 2 ? 'ck-season' : ''}" style="width:{pct(progress)}%"></div>
        </div>
        <span class="ck-p">{pct(progress)}%</span>
      </div>
    {/each}
  </div>

  <div class="ft">{lastUpdate}</div>
</div>

<style>
  /* ── Base: Mobile (320px+) ── */
  .s {
    display: flex;
    flex-direction: column;
    min-height: 100dvh;
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    padding-bottom: max(12px, env(safe-area-inset-bottom));
    box-sizing: border-box;
    gap: 10px;
    background: #0a0a1a;
    color: #ddd;
    font-family: system-ui, -apple-system, sans-serif;
  }

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
  .sc {
    text-align: center;
    padding: 12px 0;
    border-radius: 12px;
  }
  .sc-lbl { font-size: 10px; letter-spacing: .3em; opacity: .5; }
  .sc-num {
    font-size: 80px;
    font-weight: 900;
    line-height: 1;
    font-variant-numeric: tabular-nums;
    font-family: system-ui, -apple-system, sans-serif;
  }
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

  /* Possession */
  .pos {
    text-align: center;
    font-size: 12px;
    color: #888;
    letter-spacing: .05em;
  }
  .pos strong { color: #7c7cff; font-size: 16px; }

  /* Lanes — CSS progress bars */
  .lanes { display: flex; flex-direction: column; gap: 8px; }
  .ln { display: flex; align-items: center; gap: 8px; }
  .ln-n { font-size: 10px; width: 56px; letter-spacing: .05em; opacity: .45; flex-shrink: 0; }
  .ln-track {
    flex: 1;
    height: 10px;
    background: #1a1a2e;
    border-radius: 5px;
    overflow: hidden;
  }
  .ln-fill {
    height: 100%;
    background: linear-gradient(90deg, #4a9eff, #7c7cff);
    border-radius: 5px;
    transition: width .8s ease;
    min-width: 2px;
  }
  .ln-v {
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    opacity: .6;
    min-width: 40px;
    text-align: right;
    flex-shrink: 0;
  }

  /* Last Ship */
  .ls {
    text-align: center;
    font-size: 11px;
    color: #4a9eff;
    opacity: .65;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  /* Clocks */
  .clk {
    display: flex;
    flex-direction: column;
    gap: 5px;
    margin-top: auto;
    padding-top: 8px;
    border-top: 1px solid #1e1e30;
  }
  .ck { display: flex; align-items: center; gap: 6px; }
  .ck-l { font-size: 9px; width: 44px; opacity: .35; letter-spacing: .1em; flex-shrink: 0; }
  .ck-track { flex: 1; height: 5px; background: #1a1a2e; border-radius: 3px; overflow: hidden; }
  .ck-fill { height: 100%; background: linear-gradient(90deg, #4a9eff, #7c7cff); border-radius: 3px; transition: width 1s ease; }
  .ck-season { background: linear-gradient(90deg, #ff6b4a, #ff4a9e); }
  .ck-p { font-size: 10px; opacity: .4; min-width: 28px; text-align: right; font-variant-numeric: tabular-nums; flex-shrink: 0; }

  /* Footer */
  .ft { text-align: center; font-size: 9px; opacity: .2; }

  /* ── Tablet (600px+) ── */
  @media (min-width: 600px) {
    .s { padding: 20px 32px; gap: 14px; }
    .sn { font-size: 14px; }
    .sc-num { font-size: 120px; }
    .sc-sub { font-size: 16px; }
    .st-v { font-size: 18px; }
    .st-l { font-size: 10px; }
    .pos strong { font-size: 20px; }
    .ln-n { font-size: 12px; width: 70px; }
    .ln-track { height: 14px; border-radius: 7px; }
    .ln-fill { border-radius: 7px; }
    .ln-v { font-size: 14px; }
    .ls { font-size: 13px; }
  }

  /* ── TV / Stadium (1024px+) ── */
  @media (min-width: 1024px) {
    .s {
      padding: 3vh 5vw;
      gap: 2vh;
      height: 100vh;
      min-height: auto;
      overflow: hidden;
      font-family: 'SF Mono', 'JetBrains Mono', 'Fira Code', monospace;
    }
    .sn { font-size: 2.2vh; }
    .sd { font-size: 1.8vh; }
    .sc { padding: 2.5vh 0; }
    .sc-lbl { font-size: 1.6vh; }
    .sc-num { font-size: 16vh; }
    .sc-sub { font-size: 2.5vh; }
    .stats { gap: 2vw; padding: 1.5vh 0; }
    .st-v { font-size: 3vh; }
    .st-l { font-size: 1.3vh; }
    .pos strong { font-size: 3vh; }
    .lanes { gap: 1.5vh; }
    .ln-n { font-size: 1.8vh; width: 8vw; }
    .ln-track { height: 1.5vh; border-radius: 1vh; }
    .ln-fill { border-radius: 1vh; }
    .ln-v { font-size: 1.8vh; min-width: 6vw; }
    .ls { font-size: 1.8vh; }
    .ck-l { font-size: 1.3vh; width: 6vw; }
    .ck-track { height: .8vh; }
    .ck-p { font-size: 1.3vh; }
    .ft { font-size: 1vh; }
  }
</style>
