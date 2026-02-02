<script>
  let { items, pendingCount = 0, onHelp } = $props();

  let tab = $state('all');         // 'all' | 'pending' | 'approved'
  let pendingItems = $state([]);
  let loadingPending = $state(false);
  let actionInFlight = $state(''); // event ID being acted on

  const API = '';  // same origin
  const TOKEN = new URLSearchParams(window.location.search).get('token') || 
                new URLSearchParams(window.location.search).get('key') || '';

  function authParam() {
    return TOKEN ? `token=${TOKEN}` : '';
  }

  function headers() {
    const h = { 'Content-Type': 'application/json' };
    if (TOKEN) h['Authorization'] = `Bearer ${TOKEN}`;
    return h;
  }

  async function loadPending() {
    loadingPending = true;
    try {
      const ap = authParam();
      const res = await fetch(`${API}/v1/feed?status=pending&limit=100${ap ? '&' + ap : ''}`, { headers: headers() });
      const data = await res.json();
      pendingItems = data.items || [];
    } catch(e) {
      console.error('Failed to load pending:', e);
    }
    loadingPending = false;
  }

  async function approveEvent(id) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const url = `${API}/v1/events/${id}/approve${ap ? '?' + ap : ''}`;
      const res = await fetch(url, { method: 'POST', headers: headers() });
      const data = await res.json();
      if (data.ok) {
        pendingItems = pendingItems.filter(i => i.id !== id);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function rejectEvent(id) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const url = `${API}/v1/events/${id}/reject${ap ? '?' + ap : ''}`;
      const res = await fetch(url, { method: 'POST', headers: headers() });
      const data = await res.json();
      if (data.ok) {
        pendingItems = pendingItems.filter(i => i.id !== id);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function approveAll() {
    for (const item of [...pendingItems]) {
      await approveEvent(item.id);
    }
  }

  function switchTab(t) {
    tab = t;
    if (t === 'pending' && pendingItems.length === 0) {
      loadPending();
    }
  }

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

  function classLabel(source) {
    if (source === 'github-webhook') return '✓ VERIFIED';
    if (source === 'rss-poller') return '⚡ AUTO';
    if (source === 'git-discovery') return '🔍 DISCOVERED';
    return source;
  }
</script>

<div class="feed-view">
  <div class="feed-hdr">
    <h2>Activity Feed</h2>
    <span class="feed-right">
      <span class="feed-count">{items.length} events</span>
      <button class="hdr-help" onclick={onHelp} title="How it works">?</button>
    </span>
  </div>

  <!-- Tab bar -->
  <div class="tab-bar">
    <button class="tab" class:active={tab === 'all'} onclick={() => switchTab('all')}>
      All
    </button>
    <button class="tab" class:active={tab === 'pending'} onclick={() => switchTab('pending')}>
      Pending
      {#if pendingCount > 0}
        <span class="badge">{pendingCount}</span>
      {/if}
    </button>
    <button class="tab" class:active={tab === 'approved'} onclick={() => switchTab('approved')}>
      Scored
    </button>
  </div>

  <!-- Pending tab -->
  {#if tab === 'pending'}
    {#if loadingPending}
      <div class="loading">Loading pending events...</div>
    {:else if pendingItems.length === 0}
      <div class="empty">
        <div class="empty-icon">✅</div>
        <p>No pending events</p>
        <p class="hint">New commits auto-discovered every 5 min</p>
      </div>
    {:else}
      <div class="pending-actions">
        <span class="pa-count">{pendingItems.length} awaiting review</span>
        <button class="btn-approve-all" onclick={approveAll}>Approve All</button>
      </div>
      <div class="feed-list">
        {#each pendingItems as item}
          <div class="feed-item pending-item">
            <div class="fi-icon">{item.icon || '📌'}</div>
            <div class="fi-body">
              <div class="fi-title">{item.title || item.type}</div>
              <div class="fi-meta">
                <span class="fi-lane" style="color:{laneColor(item.lane)}">{item.lane}</span>
                <span class="fi-sep">·</span>
                <span class="fi-source">{classLabel(item.source)}</span>
                <span class="fi-sep">·</span>
                <span class="fi-time">{timeAgo(item.timestamp)}</span>
              </div>
              {#if item.url}
                <a class="fi-link" href={item.url} target="_blank" rel="noopener">View commit →</a>
              {/if}
            </div>
            <div class="fi-actions">
              {#if actionInFlight === item.id}
                <span class="acting">...</span>
              {:else}
                <button class="btn-approve" onclick={() => approveEvent(item.id)} title="Approve">✓</button>
                <button class="btn-reject" onclick={() => rejectEvent(item.id)} title="Reject">✗</button>
              {/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}

  <!-- All / Approved tabs -->
  {:else}
    {#if items.length === 0}
      <div class="empty">
        <div class="empty-icon">📋</div>
        <p>No events yet</p>
        <p class="hint">Push events via <code>wb ship</code> or the API</p>
      </div>
    {:else}
      <div class="feed-list">
        {#each items.filter(i => tab === 'all' || i.status === 'approved') as item}
          <div class="feed-item" class:is-pending={item.status === 'pending'}>
            <div class="fi-icon">{item.icon || '📌'}</div>
            <div class="fi-body">
              <div class="fi-title">{item.title || item.type}</div>
              <div class="fi-meta">
                <span class="fi-lane" style="color:{laneColor(item.lane)}">{item.lane}</span>
                <span class="fi-sep">·</span>
                <span class="fi-source">{classLabel(item.source)}</span>
                <span class="fi-sep">·</span>
                <span class="fi-time">{timeAgo(item.timestamp)}</span>
                {#if item.status === 'pending'}
                  <span class="fi-sep">·</span>
                  <span class="fi-pending-badge">PENDING</span>
                {/if}
              </div>
            </div>
            <div class="fi-delta" class:positive={item.score_delta > 0}>
              {#if item.score_delta > 0}+{item.score_delta}{:else}—{/if}
            </div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .feed-view {
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    min-height: calc(100dvh - 56px);
  }

  .feed-hdr {
    display: flex;
    justify-content: space-between;
    align-items: baseline;
    margin-bottom: 8px;
    border-bottom: 1px solid #1e1e30;
    padding-bottom: 8px;
  }
  .feed-hdr h2 { font-size: 16px; font-weight: 700; color: #7c7cff; }
  .feed-right { display: flex; align-items: center; gap: 8px; }
  .feed-count { font-size: 12px; color: #555; }
  .hdr-help {
    width: 22px; height: 22px; border-radius: 50%;
    background: rgba(124,124,255,0.1); border: 1px solid rgba(124,124,255,0.25);
    color: #7c7cff; font-size: 12px; font-weight: 700; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent; flex-shrink: 0;
  }

  /* Tab bar */
  .tab-bar {
    display: flex;
    gap: 0;
    margin-bottom: 12px;
    border-radius: 8px;
    overflow: hidden;
    background: #12121e;
  }
  .tab {
    flex: 1;
    padding: 8px 4px;
    background: transparent;
    border: none;
    color: #555;
    font-size: 13px;
    font-weight: 600;
    cursor: pointer;
    text-align: center;
    transition: all 0.15s;
    position: relative;
    -webkit-tap-highlight-color: transparent;
  }
  .tab.active {
    color: #7c7cff;
    background: rgba(124,124,255,0.08);
    border-bottom: 2px solid #7c7cff;
  }
  .badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    border-radius: 9px;
    background: #ff4444;
    color: white;
    font-size: 11px;
    font-weight: 700;
    margin-left: 4px;
    vertical-align: middle;
  }

  /* Pending actions bar */
  .pending-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
    padding: 8px 10px;
    background: rgba(255,170,0,0.06);
    border: 1px solid rgba(255,170,0,0.15);
    border-radius: 8px;
  }
  .pa-count { font-size: 12px; color: #aa8800; font-weight: 600; }
  .btn-approve-all {
    padding: 4px 12px;
    border-radius: 6px;
    background: rgba(46,204,113,0.15);
    border: 1px solid rgba(46,204,113,0.3);
    color: #2ecc71;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }

  .empty {
    text-align: center;
    padding: 40px 0;
  }
  .empty-icon { font-size: 40px; margin-bottom: 8px; }
  .empty p { font-size: 14px; opacity: 0.5; }
  .hint { margin-top: 4px; }
  .hint code { font-family: monospace; color: #7c7cff; }
  .loading { text-align: center; padding: 30px; color: #555; font-size: 13px; }

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
  .feed-item.is-pending { opacity: 0.5; }
  .feed-item.pending-item { opacity: 1; }

  .fi-icon { font-size: 20px; flex-shrink: 0; }
  .fi-body { flex: 1; min-width: 0; }
  .fi-title {
    font-size: 13px; font-weight: 500;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .fi-meta {
    font-size: 11px; opacity: 0.4;
    display: flex; align-items: center; gap: 4px; margin-top: 2px;
    flex-wrap: wrap;
  }
  .fi-lane { font-weight: 600; opacity: 1; }
  .fi-sep { opacity: 0.3; }
  .fi-link {
    font-size: 11px; color: #4a9eff; text-decoration: none;
    display: inline-block; margin-top: 3px;
  }
  .fi-pending-badge {
    font-size: 9px; font-weight: 700;
    background: rgba(255,170,0,0.15);
    color: #ffaa00;
    padding: 1px 5px;
    border-radius: 3px;
    letter-spacing: 0.5px;
  }

  .fi-delta {
    font-size: 14px; font-weight: 700;
    font-variant-numeric: tabular-nums;
    color: #555; flex-shrink: 0;
  }
  .fi-delta.positive { color: #2ecc71; }

  /* Approve/Reject buttons */
  .fi-actions {
    display: flex; gap: 6px; flex-shrink: 0;
  }
  .btn-approve, .btn-reject {
    width: 32px; height: 32px; border-radius: 8px;
    border: none; cursor: pointer; font-size: 16px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent;
    transition: all 0.15s;
  }
  .btn-approve {
    background: rgba(46,204,113,0.12);
    color: #2ecc71;
    border: 1px solid rgba(46,204,113,0.25);
  }
  .btn-approve:active { background: rgba(46,204,113,0.3); }
  .btn-reject {
    background: rgba(255,68,68,0.12);
    color: #ff4444;
    border: 1px solid rgba(255,68,68,0.25);
  }
  .btn-reject:active { background: rgba(255,68,68,0.3); }
  .acting { color: #555; font-size: 12px; }
</style>
