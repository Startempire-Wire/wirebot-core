<script>
  let { items } = $props();

  function timeAgo(ts) {
    const d = new Date(ts);
    const now = new Date();
    const mins = Math.floor((now - d) / 60000);
    if (mins < 1) return 'now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    const days = Math.floor(hrs / 24);
    return `${days}d ago`;
  }

  function laneColor(lane) {
    const colors = { shipping: '#4a9eff', distribution: '#9b59b6', revenue: '#2ecc71', systems: '#e67e22' };
    return colors[lane] || '#888';
  }
</script>

<div class="feed-view">
  <div class="feed-hdr">
    <h2>Activity Feed</h2>
    <span class="feed-count">{items.length} events</span>
  </div>

  {#if items.length === 0}
    <div class="empty">
      <div class="empty-icon">📋</div>
      <p>No events yet</p>
      <p class="hint">Push events via <code>wb ship</code> or the API</p>
    </div>
  {:else}
    <div class="feed-list">
      {#each items as item}
        <div class="feed-item">
          <div class="fi-icon">{item.icon || '📌'}</div>
          <div class="fi-body">
            <div class="fi-title">{item.title || item.type}</div>
            <div class="fi-meta">
              <span class="fi-lane" style="color:{laneColor(item.lane)}">{item.lane}</span>
              <span class="fi-sep">·</span>
              <span class="fi-source">{item.source}</span>
              <span class="fi-sep">·</span>
              <span class="fi-time">{timeAgo(item.timestamp)}</span>
            </div>
          </div>
          <div class="fi-delta" class:positive={item.score_delta > 0}>
            +{item.score_delta}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<style>
  .feed-view {
    padding: 12px 16px;
    padding-top: 0;
    min-height: calc(100dvh - 56px);
  }

  .feed-hdr {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 12px;
    border-bottom: 1px solid #1e1e30;
    padding-bottom: 8px;
  }
  .feed-hdr h2 { font-size: 16px; font-weight: 700; color: #7c7cff; }
  .feed-count { font-size: 12px; color: #555; }

  .empty {
    text-align: center;
    padding: 40px 0;
  }
  .empty-icon { font-size: 40px; margin-bottom: 8px; }
  .empty p { font-size: 14px; opacity: 0.5; }
  .hint { margin-top: 4px; }
  .hint code { font-family: monospace; color: #7c7cff; }

  .feed-list {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .feed-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 0;
    border-bottom: 1px solid #1a1a25;
  }

  .fi-icon { font-size: 20px; flex-shrink: 0; }

  .fi-body { flex: 1; min-width: 0; }

  .fi-title {
    font-size: 13px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .fi-meta {
    font-size: 11px;
    opacity: 0.4;
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 2px;
  }

  .fi-lane { font-weight: 600; opacity: 1; }
  .fi-sep { opacity: 0.3; }

  .fi-delta {
    font-size: 14px;
    font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: #555;
    flex-shrink: 0;
  }
  .fi-delta.positive { color: #2ecc71; }
</style>
