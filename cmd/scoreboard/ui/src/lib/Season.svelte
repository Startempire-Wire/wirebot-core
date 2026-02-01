<script>
  let { season, history, streak } = $props();

  function pct(v) { return Math.round((v || 0) * 100); }

  function dayColor(day) {
    if (!day) return '#1a1a2e';
    if (day.won) return '#00ff64';
    if (day.score > 30) return '#ffc800';
    if (day.score > 0) return '#ff3232';
    return '#1a1a2e';
  }

  // Build calendar grid for current month
  function buildCalendar() {
    const now = new Date();
    const year = now.getFullYear();
    const month = now.getMonth();
    const firstDay = new Date(year, month, 1).getDay();
    const daysInMonth = new Date(year, month + 1, 0).getDate();

    const dayMap = {};
    (history || []).forEach(d => { dayMap[d.date] = d; });

    const cells = [];
    for (let i = 0; i < firstDay; i++) cells.push(null);
    for (let d = 1; d <= daysInMonth; d++) {
      const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
      cells.push({ day: d, date: dateStr, data: dayMap[dateStr] || null });
    }
    return cells;
  }

  let calendar = $derived(buildCalendar());

  const monthNames = ['January','February','March','April','May','June','July','August','September','October','November','December'];
  const now = new Date();
</script>

<div class="season-view">
  <div class="s-hdr">
    <h2>{season?.name?.toUpperCase() || 'SEASON'}</h2>
    <span class="s-num">Season {season?.number || 1}</span>
  </div>

  <!-- Season Progress -->
  <div class="s-progress">
    <div class="sp-bar">
      <div class="sp-fill" style="width:{pct(season?.days_elapsed / (season?.days_elapsed + season?.days_remaining))}%"></div>
    </div>
    <div class="sp-labels">
      <span>{season?.start_date}</span>
      <span>Day {season?.days_elapsed} of {(season?.days_elapsed || 0) + (season?.days_remaining || 0)}</span>
      <span>{season?.end_date}</span>
    </div>
  </div>

  <!-- Big Stats -->
  <div class="big-stats">
    <div class="bs">
      <div class="bs-val">{season?.record || '0W-0L'}</div>
      <div class="bs-lbl">RECORD</div>
    </div>
    <div class="bs">
      <div class="bs-val">{season?.avg_score || 0}</div>
      <div class="bs-lbl">AVG SCORE</div>
    </div>
    <div class="bs">
      <div class="bs-val">🔥 {streak?.best || 0}</div>
      <div class="bs-lbl">BEST STREAK</div>
    </div>
  </div>

  <!-- Calendar Heatmap -->
  <div class="cal">
    <div class="cal-title">{monthNames[now.getMonth()]} {now.getFullYear()}</div>
    <div class="cal-dow">
      {#each ['S','M','T','W','T','F','S'] as d}
        <span>{d}</span>
      {/each}
    </div>
    <div class="cal-grid">
      {#each calendar as cell}
        {#if cell === null}
          <div class="cal-cell empty"></div>
        {:else}
          <div
            class="cal-cell {cell.data?.won ? 'won' : cell.data?.score > 0 ? 'played' : ''}"
            style="background:{dayColor(cell.data)}"
            title="{cell.date}: {cell.data?.score || 0} pts{cell.data?.won ? ' ✓' : ''}"
          >
            <span class="cal-day">{cell.day}</span>
          </div>
        {/if}
      {/each}
    </div>
    <div class="cal-legend">
      <span class="leg"><span class="leg-dot" style="background:#1a1a2e"></span> No data</span>
      <span class="leg"><span class="leg-dot" style="background:#ff3232"></span> Loss</span>
      <span class="leg"><span class="leg-dot" style="background:#ffc800"></span> Close</span>
      <span class="leg"><span class="leg-dot" style="background:#00ff64"></span> Win</span>
    </div>
  </div>

  <!-- Theme -->
  {#if season?.theme}
    <div class="theme">"{season.theme}"</div>
  {/if}

  <!-- Daily History -->
  {#if history.length > 0}
    <div class="hist">
      <div class="hist-title">Daily Scores</div>
      {#each [...history].reverse().slice(0, 14) as day}
        <div class="hist-row">
          <span class="hr-date">{day.date.slice(5)}</span>
          <div class="hr-bar-track">
            <div class="hr-bar-fill {day.won ? 'won' : day.score > 30 ? 'mid' : 'low'}"
                 style="width:{day.score}%"></div>
          </div>
          <span class="hr-score">{day.score}</span>
          <span class="hr-result">{day.won ? '✅' : '❌'}</span>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .season-view {
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    min-height: calc(100dvh - 56px);
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .s-hdr {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid #1e1e30;
    padding-bottom: 6px;
  }
  .s-hdr h2 { font-size: 16px; font-weight: 700; color: #7c7cff; }
  .s-num { font-size: 12px; color: #555; }

  /* Progress */
  .s-progress { margin: 4px 0; }
  .sp-bar { height: 8px; background: #1a1a2e; border-radius: 4px; overflow: hidden; }
  .sp-fill { height: 100%; background: linear-gradient(90deg, #ff6b4a, #ff4a9e); border-radius: 4px; transition: width 1s; }
  .sp-labels { display: flex; justify-content: space-between; font-size: 10px; opacity: 0.35; margin-top: 4px; }

  /* Big Stats */
  .big-stats {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
    padding: 12px 0;
    border-top: 1px solid #1e1e30;
    border-bottom: 1px solid #1e1e30;
  }
  .bs { text-align: center; }
  .bs-val { font-size: 20px; font-weight: 800; }
  .bs-lbl { font-size: 9px; opacity: 0.35; letter-spacing: 0.1em; margin-top: 2px; }

  /* Calendar */
  .cal { background: #111118; border-radius: 12px; padding: 12px; }
  .cal-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; text-align: center; }
  .cal-dow {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    text-align: center;
    font-size: 10px;
    opacity: 0.3;
    margin-bottom: 4px;
  }
  .cal-grid {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 3px;
  }
  .cal-cell {
    aspect-ratio: 1;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 10px;
    position: relative;
  }
  .cal-cell.empty { background: transparent; }
  .cal-day { opacity: 0.7; }
  .cal-cell.won .cal-day { color: #000; font-weight: 700; }

  .cal-legend {
    display: flex;
    justify-content: center;
    gap: 12px;
    margin-top: 8px;
  }
  .leg { display: flex; align-items: center; gap: 4px; font-size: 10px; opacity: 0.5; }
  .leg-dot { width: 10px; height: 10px; border-radius: 2px; display: inline-block; }

  .theme {
    text-align: center;
    font-size: 12px;
    font-style: italic;
    opacity: 0.3;
  }

  /* History bars */
  .hist { margin-top: 4px; }
  .hist-title { font-size: 13px; font-weight: 600; margin-bottom: 8px; }
  .hist-row {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 0;
  }
  .hr-date { font-size: 11px; opacity: 0.4; width: 38px; flex-shrink: 0; font-variant-numeric: tabular-nums; }
  .hr-bar-track { flex: 1; height: 8px; background: #1a1a2e; border-radius: 4px; overflow: hidden; }
  .hr-bar-fill { height: 100%; border-radius: 4px; transition: width 0.5s; }
  .hr-bar-fill.won { background: #00ff64; }
  .hr-bar-fill.mid { background: #ffc800; }
  .hr-bar-fill.low { background: #ff3232; }
  .hr-score { font-size: 12px; font-weight: 600; width: 24px; text-align: right; font-variant-numeric: tabular-nums; flex-shrink: 0; }
  .hr-result { font-size: 12px; flex-shrink: 0; }
</style>
