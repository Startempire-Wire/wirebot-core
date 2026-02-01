<script>
  let { wrapped } = $props();
</script>

<div class="wrapped-view">
  <div class="w-header">
    <h2>🎬 Season Wrapped</h2>
    <span class="w-season">{wrapped?.season || 'Season'}</span>
  </div>

  {#if !wrapped || !wrapped.days_played}
    <div class="empty">
      <div class="empty-icon">🎬</div>
      <p>Not enough data yet</p>
      <p class="hint">Play a few days to see your retrospective</p>
    </div>
  {:else}
    <!-- Big stats cards -->
    <div class="cards">
      <div class="card hero">
        <div class="card-val">{wrapped.record || '0W-0L'}</div>
        <div class="card-lbl">SEASON RECORD</div>
      </div>

      <div class="card-row">
        <div class="card">
          <div class="card-val">{wrapped.total_ships || 0}</div>
          <div class="card-lbl">TOTAL SHIPS</div>
        </div>
        <div class="card">
          <div class="card-val">{wrapped.best_streak || 0}</div>
          <div class="card-lbl">BEST STREAK</div>
        </div>
      </div>

      <div class="card-row">
        <div class="card">
          <div class="card-val">{wrapped.avg_score || 0}</div>
          <div class="card-lbl">AVG SCORE</div>
        </div>
        <div class="card">
          <div class="card-val">{wrapped.revenue_events || 0}</div>
          <div class="card-lbl">REVENUE EVENTS</div>
        </div>
      </div>

      <div class="card-row">
        <div class="card">
          <div class="card-val">{wrapped.days_won || 0}</div>
          <div class="card-lbl">DAYS WON</div>
        </div>
        <div class="card">
          <div class="card-val">{wrapped.days_played || 0}</div>
          <div class="card-lbl">DAYS PLAYED</div>
        </div>
      </div>
    </div>

    <!-- Patterns -->
    {#if wrapped.patterns}
      <div class="patterns">
        <h3>Patterns</h3>
        {#if wrapped.patterns.best_day_of_week}
          <div class="pat">📅 Best day: <strong>{wrapped.patterns.best_day_of_week}</strong></div>
        {/if}
        {#if wrapped.patterns.best_lane}
          <div class="pat">🎯 Best lane: <strong>{wrapped.patterns.best_lane}</strong></div>
        {/if}
        {#if wrapped.patterns.avg_score_trend}
          <div class="pat">📈 Trend: <strong>{wrapped.patterns.avg_score_trend === '↑' ? 'Improving' : wrapped.patterns.avg_score_trend === '↓' ? 'Declining' : 'Steady'}</strong></div>
        {/if}
      </div>
    {/if}

    <!-- Top artifacts -->
    {#if wrapped.top_artifacts?.length}
      <div class="artifacts">
        <h3>Top Artifacts</h3>
        {#each wrapped.top_artifacts as a, i}
          <div class="artifact">
            <span class="a-rank">#{i + 1}</span>
            <span class="a-title">{a.title || a.event_type}</span>
            <span class="a-pts">+{a.score_delta}</span>
          </div>
        {/each}
      </div>
    {/if}

    <!-- Share card link -->
    <div class="share">
      <a href="/v1/card/season" target="_blank">📤 Download Share Card (SVG)</a>
    </div>
  {/if}
</div>

<style>
  .wrapped-view {
    padding: 12px 16px;
    padding-top: 0;
    min-height: calc(100dvh - 56px);
    display: flex;
    flex-direction: column;
    gap: 14px;
  }

  .w-header {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    border-bottom: 1px solid #1e1e30;
    padding-bottom: 6px;
  }
  .w-header h2 { font-size: 16px; font-weight: 700; color: #ff4a9e; }
  .w-season { font-size: 12px; color: #555; }

  .empty { text-align: center; padding: 40px 0; }
  .empty-icon { font-size: 40px; margin-bottom: 8px; }
  .empty p { font-size: 14px; opacity: 0.5; }
  .hint { margin-top: 4px; font-size: 12px; }

  .cards { display: flex; flex-direction: column; gap: 8px; }

  .card {
    background: #111118;
    border-radius: 12px;
    padding: 14px;
    text-align: center;
  }
  .card.hero {
    background: linear-gradient(135deg, #1a1a30, #111118);
    border: 1px solid #2a2a4a;
    padding: 20px;
  }
  .card-val { font-size: 28px; font-weight: 900; }
  .hero .card-val { font-size: 40px; color: #ff4a9e; }
  .card-lbl { font-size: 10px; opacity: 0.35; letter-spacing: 0.1em; margin-top: 4px; }

  .card-row { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }

  .patterns, .artifacts {
    background: #111118;
    border-radius: 12px;
    padding: 14px;
  }
  .patterns h3, .artifacts h3 { font-size: 13px; font-weight: 600; margin-bottom: 8px; color: #7c7cff; }
  .pat { font-size: 13px; padding: 4px 0; opacity: 0.7; }
  .pat strong { color: #ddd; opacity: 1; }

  .artifact {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    border-bottom: 1px solid #1a1a25;
    font-size: 13px;
  }
  .a-rank { opacity: 0.4; font-weight: 700; width: 24px; }
  .a-title { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .a-pts { color: #2ecc71; font-weight: 700; }

  .share {
    text-align: center;
    padding: 12px 0;
  }
  .share a {
    color: #7c7cff;
    text-decoration: none;
    font-size: 14px;
    padding: 8px 16px;
    border: 1px solid #7c7cff;
    border-radius: 8px;
    display: inline-block;
  }
</style>
