<script>
  let { data, lastUpdate } = $props();

  function bar(value, max, width = 10) {
    const filled = Math.round((value / max) * width);
    return '█'.repeat(filled) + '░'.repeat(width - filled);
  }

  function signalClass(signal) {
    return signal === 'green' ? 'signal-green' : signal === 'yellow' ? 'signal-yellow' : 'signal-red';
  }

  function pct(v) { return Math.round(v * 100); }

  function signalLabel(signal) {
    return signal === 'green' ? 'WINNING' : signal === 'yellow' ? 'PRESSURE' : 'STALLING';
  }
</script>

<div class="stadium">
  <!-- Header -->
  <header>
    <span class="season-name">{data.season?.name?.toUpperCase() || 'SCOREBOARD'}</span>
    <span class="season-day">{data.season_day || ''}</span>
  </header>

  <!-- Main Score -->
  <div class="score-zone {signalClass(data.signal)}">
    <div class="score-label">EXECUTION SCORE</div>
    <div class="score-number">{data.score}</div>
    <div class="score-sub">{signalLabel(data.signal)}</div>
  </div>

  <!-- Stats -->
  <div class="stats">
    <div class="stat">
      <div class="stat-val">🔥 {data.streak?.current || 0}</div>
      <div class="stat-lbl">STREAK</div>
    </div>
    <div class="stat">
      <div class="stat-val">🏆 {data.streak?.best || 0}</div>
      <div class="stat-lbl">BEST</div>
    </div>
    <div class="stat">
      <div class="stat-val">📊 {data.record || '0W-0L'}</div>
      <div class="stat-lbl">RECORD</div>
    </div>
    <div class="stat">
      <div class="stat-val">🚀 {data.ship_today || 0}</div>
      <div class="stat-lbl">SHIPS</div>
    </div>
  </div>

  <!-- Possession -->
  <div class="possession">
    ⚡ <span class="pos-value">{data.possession || '—'}</span>
  </div>

  <!-- Lanes -->
  <div class="lanes">
    <div class="lane">
      <span class="lane-name">SHIP</span>
      <span class="lane-bar">{bar(data.lanes?.shipping || 0, data.lanes?.shipping_max || 40)}</span>
      <span class="lane-pts">{data.lanes?.shipping || 0}<span class="lane-max">/{data.lanes?.shipping_max || 40}</span></span>
    </div>
    <div class="lane">
      <span class="lane-name">DIST</span>
      <span class="lane-bar">{bar(data.lanes?.distribution || 0, data.lanes?.distribution_max || 25)}</span>
      <span class="lane-pts">{data.lanes?.distribution || 0}<span class="lane-max">/{data.lanes?.distribution_max || 25}</span></span>
    </div>
    <div class="lane">
      <span class="lane-name">REV</span>
      <span class="lane-bar">{bar(data.lanes?.revenue || 0, data.lanes?.revenue_max || 20)}</span>
      <span class="lane-pts">{data.lanes?.revenue || 0}<span class="lane-max">/{data.lanes?.revenue_max || 20}</span></span>
    </div>
    <div class="lane">
      <span class="lane-name">SYS</span>
      <span class="lane-bar">{bar(data.lanes?.systems || 0, data.lanes?.systems_max || 15)}</span>
      <span class="lane-pts">{data.lanes?.systems || 0}<span class="lane-max">/{data.lanes?.systems_max || 15}</span></span>
    </div>
  </div>

  <!-- Last Ship -->
  {#if data.last_ship}
    <div class="last-ship">
      LAST: {data.last_ship}
    </div>
  {/if}

  <!-- Clock -->
  <div class="clocks">
    <div class="clock-row">
      <span class="clock-lbl">DAY</span>
      <div class="clock-track"><div class="clock-fill" style="width:{pct(data.clock?.day_progress||0)}%"></div></div>
      <span class="clock-pct">{pct(data.clock?.day_progress||0)}%</span>
    </div>
    <div class="clock-row">
      <span class="clock-lbl">WEEK</span>
      <div class="clock-track"><div class="clock-fill" style="width:{pct(data.clock?.week_progress||0)}%"></div></div>
      <span class="clock-pct">{pct(data.clock?.week_progress||0)}%</span>
    </div>
    <div class="clock-row">
      <span class="clock-lbl">SEASON</span>
      <div class="clock-track"><div class="clock-fill season-bar" style="width:{pct(data.clock?.season_progress||0)}%"></div></div>
      <span class="clock-pct">{pct(data.clock?.season_progress||0)}%</span>
    </div>
  </div>

  <footer>{data.season?.theme || ''} · Updated {lastUpdate}</footer>
</div>

<style>
  .stadium {
    width: 100%;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    padding: env(safe-area-inset-top, 12px) 16px env(safe-area-inset-bottom, 12px);
    box-sizing: border-box;
    background: linear-gradient(180deg, #0a0a1a 0%, #0d0d20 50%, #0a0a1a 100%);
    gap: 12px;
    font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', 'Courier New', monospace;
  }

  header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid #1a1a2e;
    padding-bottom: 8px;
  }
  .season-name { font-size: 14px; font-weight: 700; letter-spacing: 0.2em; color: #7c7cff; }
  .season-day { font-size: 13px; color: #666; }

  /* Score */
  .score-zone {
    text-align: center;
    padding: 16px 0;
    border-radius: 12px;
    transition: all 0.5s ease;
  }
  .score-label { font-size: 11px; letter-spacing: 0.4em; opacity: 0.5; }
  .score-number { font-size: 96px; font-weight: 900; line-height: 1; font-variant-numeric: tabular-nums; }
  .score-sub { font-size: 14px; letter-spacing: 0.3em; margin-top: 4px; }

  .signal-green { background: rgba(0,255,100,0.06); }
  .signal-green .score-number { color: #00ff64; text-shadow: 0 0 30px rgba(0,255,100,0.25); }
  .signal-green .score-sub { color: #00cc50; }

  .signal-yellow { background: rgba(255,200,0,0.06); }
  .signal-yellow .score-number { color: #ffc800; text-shadow: 0 0 30px rgba(255,200,0,0.25); }
  .signal-yellow .score-sub { color: #cc9900; }

  .signal-red { background: rgba(255,50,50,0.06); }
  .signal-red .score-number { color: #ff3232; text-shadow: 0 0 30px rgba(255,50,50,0.25); animation: pulse 2s infinite; }
  .signal-red .score-sub { color: #cc2020; }

  @keyframes pulse { 0%,100% { opacity:1; } 50% { opacity:0.6; } }

  /* Stats */
  .stats {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: 8px;
    padding: 8px 0;
    border-top: 1px solid #1a1a2e;
    border-bottom: 1px solid #1a1a2e;
  }
  .stat { text-align: center; }
  .stat-val { font-size: 16px; font-weight: 700; }
  .stat-lbl { font-size: 9px; letter-spacing: 0.15em; opacity: 0.4; margin-top: 2px; }

  /* Possession */
  .possession {
    text-align: center;
    font-size: 13px;
    color: #888;
    letter-spacing: 0.1em;
  }
  .pos-value { font-size: 18px; font-weight: 700; color: #7c7cff; }

  /* Lanes */
  .lanes { display: flex; flex-direction: column; gap: 6px; padding: 4px 0; }
  .lane { display: flex; align-items: center; gap: 8px; }
  .lane-name { font-size: 11px; width: 36px; opacity: 0.5; letter-spacing: 0.05em; }
  .lane-bar { font-size: 14px; color: #4a9eff; letter-spacing: 0.02em; flex: 1; }
  .lane-pts { font-size: 13px; font-variant-numeric: tabular-nums; min-width: 48px; text-align: right; }
  .lane-max { opacity: 0.4; font-size: 11px; }

  /* Last Ship */
  .last-ship {
    text-align: center;
    font-size: 12px;
    color: #4a9eff;
    opacity: 0.7;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    padding: 4px 0;
  }

  /* Clocks */
  .clocks { display: flex; flex-direction: column; gap: 6px; margin-top: auto; padding-top: 8px; border-top: 1px solid #1a1a2e; }
  .clock-row { display: flex; align-items: center; gap: 8px; }
  .clock-lbl { font-size: 10px; width: 50px; opacity: 0.4; letter-spacing: 0.15em; }
  .clock-track { flex: 1; height: 6px; background: #1a1a2e; border-radius: 3px; overflow: hidden; }
  .clock-fill { height: 100%; background: linear-gradient(90deg, #4a9eff, #7c7cff); border-radius: 3px; transition: width 1s ease; }
  .season-bar { background: linear-gradient(90deg, #ff6b4a, #ff4a9e); }
  .clock-pct { font-size: 11px; opacity: 0.5; min-width: 32px; text-align: right; font-variant-numeric: tabular-nums; }

  footer {
    text-align: center;
    font-size: 10px;
    opacity: 0.25;
    padding: 4px 0;
  }

  /* ── TV / Large screen overrides ── */
  @media (min-width: 1024px) {
    .stadium { padding: 3vh 4vw; gap: 2vh; }
    .season-name { font-size: 2.5vh; }
    .season-day { font-size: 2vh; }
    .score-number { font-size: 18vh; }
    .score-label { font-size: 1.8vh; }
    .score-sub { font-size: 2.5vh; }
    .stats { gap: 2vw; }
    .stat-val { font-size: 3vh; }
    .stat-lbl { font-size: 1.5vh; }
    .pos-value { font-size: 3.5vh; }
    .lane-name { font-size: 2vh; width: 8vw; }
    .lane-bar { font-size: 2.5vh; }
    .lane-pts { font-size: 2vh; }
    .lanes { gap: 1.5vh; }
    .clock-lbl { font-size: 1.5vh; }
    .clock-track { height: 1vh; }
    .clock-pct { font-size: 1.5vh; }
    .last-ship { font-size: 2vh; }
    footer { font-size: 1.2vh; }
  }
</style>
