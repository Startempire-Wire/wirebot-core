<script>
  let { items, pendingCount = 0, onHelp } = $props();

  let tab = $state('all');
  let projects = $state([]);
  let expandedProject = $state(null);
  let projectEvents = $state([]);
  let loadingProjects = $state(false);
  let loadingEvents = $state(false);
  let actionInFlight = $state('');

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

  // ─── Load projects ─────────────────────────────────────────────────
  async function loadProjects() {
    loadingProjects = true;
    try {
      const res = await fetch(`/v1/projects`);
      const data = await res.json();
      projects = data.projects || [];
    } catch(e) { console.error(e); }
    loadingProjects = false;
  }

  // ─── Load events for a specific project ─────────────────────────────
  async function loadProjectEvents(projectName) {
    loadingEvents = true;
    try {
      const res = await fetch(`/v1/feed?status=pending&limit=200`);
      const data = await res.json();
      projectEvents = (data.items || []).filter(i => {
        const title = i.title || '';
        return title.startsWith(`[${projectName}]`);
      });
    } catch(e) { console.error(e); }
    loadingEvents = false;
  }

  // ─── Project actions ────────────────────────────────────────────────
  async function approveProject(name) {
    actionInFlight = `proj-${name}`;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/projects/${name}/approve${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        // Update local state
        projects = projects.map(p => p.name === name ? {...p, status: 'approved', auto_approve: true, pending: 0, approved: p.approved + data.events_affected} : p);
        pendingCount = Math.max(0, pendingCount - (data.events_affected || 0));
        if (expandedProject === name) {
          projectEvents = [];
          expandedProject = null;
        }
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function rejectProject(name) {
    actionInFlight = `proj-${name}`;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/projects/${name}/reject${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        projects = projects.map(p => p.name === name ? {...p, status: 'rejected', pending: 0, rejected: p.rejected + data.events_affected} : p);
        pendingCount = Math.max(0, pendingCount - (data.events_affected || 0));
        if (expandedProject === name) {
          projectEvents = [];
          expandedProject = null;
        }
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  // ─── Individual event actions ───────────────────────────────────────
  async function approveEvent(id) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/events/${id}/approve${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        projectEvents = projectEvents.filter(i => i.id !== id);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function rejectEvent(id) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/events/${id}/reject${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        projectEvents = projectEvents.filter(i => i.id !== id);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  // ─── Toggle expanded project ────────────────────────────────────────
  function toggleProject(name) {
    if (expandedProject === name) {
      expandedProject = null;
      projectEvents = [];
    } else {
      expandedProject = name;
      loadProjectEvents(name);
    }
  }

  function switchTab(t) {
    tab = t;
    if (t === 'pending' && projects.length === 0) loadProjects();
  }

  // ─── Helpers ────────────────────────────────────────────────────────
  function timeAgo(ts) {
    const d = new Date(ts);
    const now = new Date();
    const mins = Math.floor((now - d) / 60000);
    if (mins < 1) return 'now';
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    return `${Math.floor(hrs / 24)}d ago`;
  }

  function laneColor(lane) {
    const colors = { shipping: '#4a9eff', distribution: '#9b59b6', revenue: '#2ecc71', systems: '#e67e22' };
    return colors[lane] || '#888';
  }

  function sourceLabel(source) {
    if (source === 'github-webhook') return '✓ VERIFIED';
    if (source === 'rss-poller') return '⚡ AUTO';
    if (source === 'git-discovery') return '🔍 DISCOVERED';
    return source;
  }

  function statusIcon(s) {
    if (s === 'approved') return '✅';
    if (s === 'rejected') return '❌';
    return '⏳';
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
    <button class="tab" class:active={tab === 'all'} onclick={() => switchTab('all')}>All</button>
    <button class="tab" class:active={tab === 'pending'} onclick={() => switchTab('pending')}>
      Pending
      {#if pendingCount > 0}<span class="badge">{pendingCount}</span>{/if}
    </button>
    <button class="tab" class:active={tab === 'approved'} onclick={() => switchTab('approved')}>Scored</button>
  </div>

  <!-- ═══ PENDING TAB: Project-grouped ═══ -->
  {#if tab === 'pending'}
    {#if loadingProjects}
      <div class="loading">Loading projects...</div>
    {:else if projects.length === 0}
      <div class="empty">
        <div class="empty-icon">✅</div>
        <p>No discovered projects</p>
        <p class="hint">Discovery runs every 5 min via cron</p>
      </div>
    {:else}
      <div class="project-list">
        {#each projects as proj}
          <div class="project-card" class:expanded={expandedProject === proj.name}>
            <!-- Project header -->
            <div class="proj-header" onclick={() => proj.pending > 0 ? toggleProject(proj.name) : null}>
              <div class="proj-info">
                <span class="proj-status">{statusIcon(proj.status)}</span>
                <div class="proj-name-wrap">
                  <span class="proj-name">{proj.name}</span>
                  {#if proj.business}
                    <span class="proj-biz">{proj.business}</span>
                  {/if}
                </div>
              </div>
              <div class="proj-stats">
                {#if proj.pending > 0}
                  <span class="proj-pending">{proj.pending} pending</span>
                {:else if proj.status === 'approved'}
                  <span class="proj-ok">{proj.approved} scored</span>
                {:else if proj.status === 'rejected'}
                  <span class="proj-rej">rejected</span>
                {:else}
                  <span class="proj-none">{proj.total_events} total</span>
                {/if}
              </div>
            </div>

            <!-- Project actions (only if has pending) -->
            {#if proj.pending > 0}
              <div class="proj-actions">
                {#if actionInFlight === `proj-${proj.name}`}
                  <span class="acting">Processing...</span>
                {:else}
                  <button class="btn-proj-approve" onclick={(e) => { e.stopPropagation(); approveProject(proj.name); }}>
                    ✓ Approve Project ({proj.pending})
                  </button>
                  <button class="btn-proj-reject" onclick={(e) => { e.stopPropagation(); rejectProject(proj.name); }}>
                    ✗ Reject
                  </button>
                  <button class="btn-proj-expand" onclick={(e) => { e.stopPropagation(); toggleProject(proj.name); }}>
                    {expandedProject === proj.name ? '▲' : '▼'} Events
                  </button>
                {/if}
              </div>
            {/if}

            <!-- Expanded: individual events -->
            {#if expandedProject === proj.name}
              <div class="proj-events">
                {#if loadingEvents}
                  <div class="loading-sm">Loading events...</div>
                {:else if projectEvents.length === 0}
                  <div class="loading-sm">No pending events</div>
                {:else}
                  {#each projectEvents as evt}
                    <div class="evt-row">
                      <div class="evt-body">
                        <div class="evt-title">{evt.title?.replace(`[${proj.name}] `, '') || evt.type}</div>
                        <div class="evt-meta">
                          <span style="color:{laneColor(evt.lane)}">{evt.lane}</span>
                          <span class="sep">·</span>
                          <span>{timeAgo(evt.timestamp)}</span>
                          {#if evt.url}
                            <span class="sep">·</span>
                            <a class="evt-link" href={evt.url} target="_blank">view →</a>
                          {/if}
                        </div>
                      </div>
                      <div class="evt-actions">
                        {#if actionInFlight === evt.id}
                          <span class="acting-sm">...</span>
                        {:else}
                          <button class="btn-sm-approve" onclick={() => approveEvent(evt.id)}>✓</button>
                          <button class="btn-sm-reject" onclick={() => rejectEvent(evt.id)}>✗</button>
                        {/if}
                      </div>
                    </div>
                  {/each}
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

  <!-- ═══ ALL / SCORED TAB ═══ -->
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
                <span>{sourceLabel(item.source)}</span>
                <span class="fi-sep">·</span>
                <span>{timeAgo(item.timestamp)}</span>
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
    display: flex; justify-content: space-between; align-items: baseline;
    margin-bottom: 8px; border-bottom: 1px solid #1e1e30; padding-bottom: 8px;
  }
  .feed-hdr h2 { font-size: 16px; font-weight: 700; color: #7c7cff; }
  .feed-right { display: flex; align-items: center; gap: 8px; }
  .feed-count { font-size: 12px; color: #555; }
  .hdr-help {
    width: 22px; height: 22px; border-radius: 50%;
    background: rgba(124,124,255,0.1); border: 1px solid rgba(124,124,255,0.25);
    color: #7c7cff; font-size: 12px; font-weight: 700; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
  }

  /* Tab bar */
  .tab-bar { display: flex; margin-bottom: 12px; border-radius: 8px; overflow: hidden; background: #12121e; }
  .tab {
    flex: 1; padding: 8px 4px; background: transparent; border: none;
    color: #555; font-size: 13px; font-weight: 600; cursor: pointer;
    text-align: center; transition: all 0.15s; -webkit-tap-highlight-color: transparent;
  }
  .tab.active { color: #7c7cff; background: rgba(124,124,255,0.08); border-bottom: 2px solid #7c7cff; }
  .badge {
    display: inline-flex; align-items: center; justify-content: center;
    min-width: 18px; height: 18px; padding: 0 5px; border-radius: 9px;
    background: #ff4444; color: white; font-size: 11px; font-weight: 700; margin-left: 4px;
  }

  /* ═══ Project cards ═══ */
  .project-list { display: flex; flex-direction: column; gap: 8px; }
  .project-card {
    background: #12121e; border-radius: 10px; border: 1px solid #1e1e30;
    overflow: hidden; transition: border-color 0.15s;
  }
  .project-card.expanded { border-color: #333; }
  .proj-header {
    display: flex; justify-content: space-between; align-items: center;
    padding: 12px 14px; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .proj-info { display: flex; align-items: center; gap: 8px; }
  .proj-status { font-size: 16px; }
  .proj-name-wrap { display: flex; flex-direction: column; }
  .proj-name { font-size: 14px; font-weight: 600; color: #ddd; }
  .proj-biz { font-size: 10px; color: #555; text-transform: uppercase; letter-spacing: 0.5px; }
  .proj-stats { font-size: 12px; }
  .proj-pending { color: #ffaa00; font-weight: 600; }
  .proj-ok { color: #2ecc71; }
  .proj-rej { color: #ff4444; }
  .proj-none { color: #555; }

  .proj-actions {
    display: flex; gap: 8px; padding: 0 14px 10px;
    flex-wrap: wrap;
  }
  .btn-proj-approve {
    padding: 6px 14px; border-radius: 8px; font-size: 12px; font-weight: 600;
    background: rgba(46,204,113,0.12); border: 1px solid rgba(46,204,113,0.3);
    color: #2ecc71; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .btn-proj-approve:active { background: rgba(46,204,113,0.25); }
  .btn-proj-reject {
    padding: 6px 12px; border-radius: 8px; font-size: 12px; font-weight: 600;
    background: rgba(255,68,68,0.08); border: 1px solid rgba(255,68,68,0.2);
    color: #ff4444; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .btn-proj-expand {
    padding: 6px 10px; border-radius: 8px; font-size: 12px;
    background: rgba(124,124,255,0.08); border: 1px solid rgba(124,124,255,0.2);
    color: #7c7cff; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .acting { color: #555; font-size: 12px; }

  /* ═══ Expanded project events ═══ */
  .proj-events {
    border-top: 1px solid #1a1a28; padding: 8px 14px 10px;
    background: rgba(0,0,0,0.15); max-height: 300px; overflow-y: auto;
  }
  .loading-sm { text-align: center; padding: 12px; color: #555; font-size: 12px; }
  .evt-row {
    display: flex; align-items: center; gap: 8px;
    padding: 6px 0; border-bottom: 1px solid #151520;
  }
  .evt-row:last-child { border-bottom: none; }
  .evt-body { flex: 1; min-width: 0; }
  .evt-title { font-size: 12px; color: #aaa; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .evt-meta { font-size: 10px; color: #444; display: flex; gap: 4px; align-items: center; margin-top: 1px; }
  .sep { opacity: 0.3; }
  .evt-link { color: #4a9eff; text-decoration: none; }
  .evt-actions { display: flex; gap: 4px; flex-shrink: 0; }
  .btn-sm-approve, .btn-sm-reject {
    width: 26px; height: 26px; border-radius: 6px; border: none;
    cursor: pointer; font-size: 13px; display: flex; align-items: center;
    justify-content: center; -webkit-tap-highlight-color: transparent;
  }
  .btn-sm-approve { background: rgba(46,204,113,0.12); color: #2ecc71; }
  .btn-sm-reject { background: rgba(255,68,68,0.08); color: #ff4444; }
  .acting-sm { color: #555; font-size: 11px; }

  /* ═══ Feed items (All/Scored tabs) ═══ */
  .empty { text-align: center; padding: 40px 0; }
  .empty-icon { font-size: 40px; margin-bottom: 8px; }
  .empty p { font-size: 14px; opacity: 0.5; }
  .hint { margin-top: 4px; }
  .hint code { font-family: monospace; color: #7c7cff; }
  .loading { text-align: center; padding: 30px; color: #555; font-size: 13px; }

  .feed-list { display: flex; flex-direction: column; gap: 2px; }
  .feed-item {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 0; border-bottom: 1px solid #1a1a25;
  }
  .feed-item.is-pending { opacity: 0.45; }
  .fi-icon { font-size: 20px; flex-shrink: 0; }
  .fi-body { flex: 1; min-width: 0; }
  .fi-title { font-size: 13px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .fi-meta { font-size: 11px; opacity: 0.4; display: flex; align-items: center; gap: 4px; margin-top: 2px; flex-wrap: wrap; }
  .fi-lane { font-weight: 600; opacity: 1; }
  .fi-sep { opacity: 0.3; }
  .fi-pending-badge {
    font-size: 9px; font-weight: 700; background: rgba(255,170,0,0.15);
    color: #ffaa00; padding: 1px 5px; border-radius: 3px; letter-spacing: 0.5px;
  }
  .fi-delta { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; color: #555; flex-shrink: 0; }
  .fi-delta.positive { color: #2ecc71; }
</style>
