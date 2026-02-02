<script>
  let { items, pendingCount = 0, onHelp } = $props();

  let tab = $state('all');
  let projects = $state([]);         // project summaries
  let projectEvents = $state({});    // { projectName: [events] }
  let expanded = $state({});         // { projectName: true/false }
  let loadingProjects = $state(false);
  let actionInFlight = $state('');

  const TOKEN = new URLSearchParams(window.location.search).get('token') || 
                new URLSearchParams(window.location.search).get('key') || '';

  function authParam() { return TOKEN ? `token=${TOKEN}` : ''; }
  function headers() {
    const h = { 'Content-Type': 'application/json' };
    if (TOKEN) h['Authorization'] = `Bearer ${TOKEN}`;
    return h;
  }

  // ─── Load everything for pending tab ────────────────────────────────
  async function loadPending() {
    loadingProjects = true;
    try {
      // Fetch projects + all pending events in parallel
      const [projRes, evtRes] = await Promise.all([
        fetch('/v1/projects'),
        fetch('/v1/feed?status=pending&limit=500')
      ]);
      const projData = await projRes.json();
      const evtData = await evtRes.json();

      projects = (projData.projects || []).filter(p => p.pending > 0 || p.status !== 'pending');

      // Group events by project name (parsed from title prefix "[name] ...")
      const grouped = {};
      for (const evt of (evtData.items || [])) {
        const match = (evt.title || '').match(/^\[([^\]]+)\]/);
        const proj = match ? match[1] : 'other';
        if (!grouped[proj]) grouped[proj] = [];
        grouped[proj].push(evt);
      }
      projectEvents = grouped;

      // Add projects that appear in events but not in project list
      const projNames = new Set(projects.map(p => p.name));
      for (const name of Object.keys(grouped)) {
        if (!projNames.has(name)) {
          projects.push({
            name, path: '', business: '', github: '', status: 'pending',
            auto_approve: false, total_events: grouped[name].length,
            pending: grouped[name].length, approved: 0, rejected: 0, primary_lane: 'shipping'
          });
        }
      }

      // Sort: most pending first, then approved
      projects.sort((a, b) => b.pending - a.pending || b.total_events - a.total_events);
    } catch(e) { console.error(e); }
    loadingProjects = false;
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
        // Remove from pending, update project status
        projects = projects.map(p => p.name === name
          ? {...p, status: 'approved', auto_approve: true, pending: 0, approved: p.approved + (data.events_affected || 0)}
          : p);
        delete projectEvents[name];
        projectEvents = {...projectEvents};
        pendingCount = Math.max(0, pendingCount - (data.events_affected || 0));
        delete expanded[name];
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
        projects = projects.map(p => p.name === name
          ? {...p, status: 'rejected', pending: 0, rejected: p.rejected + (data.events_affected || 0)}
          : p);
        delete projectEvents[name];
        projectEvents = {...projectEvents};
        pendingCount = Math.max(0, pendingCount - (data.events_affected || 0));
        delete expanded[name];
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function approveEvent(id, projName) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/events/${id}/approve${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        if (projectEvents[projName]) {
          projectEvents[projName] = projectEvents[projName].filter(e => e.id !== id);
          projectEvents = {...projectEvents};
        }
        projects = projects.map(p => p.name === projName
          ? {...p, pending: Math.max(0, p.pending - 1), approved: p.approved + 1}
          : p);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  async function rejectEvent(id, projName) {
    actionInFlight = id;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/events/${id}/reject${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers()
      });
      const data = await res.json();
      if (data.ok) {
        if (projectEvents[projName]) {
          projectEvents[projName] = projectEvents[projName].filter(e => e.id !== id);
          projectEvents = {...projectEvents};
        }
        projects = projects.map(p => p.name === projName
          ? {...p, pending: Math.max(0, p.pending - 1), rejected: p.rejected + 1}
          : p);
        pendingCount = Math.max(0, pendingCount - 1);
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
  }

  // ─── Toggle ─────────────────────────────────────────────────────────
  function toggle(name) {
    expanded = {...expanded, [name]: !expanded[name]};
  }

  function switchTab(t) {
    tab = t;
    if (t === 'pending' && projects.length === 0) loadPending();
  }

  // ─── Helpers ────────────────────────────────────────────────────────
  function timeAgo(ts) {
    const d = new Date(ts);
    const now = new Date();
    const mins = Math.floor((now - d) / 60000);
    if (mins < 1) return 'now';
    if (mins < 60) return `${mins}m`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h`;
    return `${Math.floor(hrs / 24)}d`;
  }
  function laneColor(lane) {
    return { shipping: '#4a9eff', distribution: '#9b59b6', revenue: '#2ecc71', systems: '#e67e22' }[lane] || '#888';
  }
  function sourceLabel(src) {
    if (src === 'github-webhook') return '✓ verified';
    if (src === 'rss-poller') return '⚡ auto';
    if (src === 'git-discovery') return '🔍 discovered';
    return src;
  }
  function statusIcon(s) {
    return { approved: '✅', rejected: '❌' }[s] || '⏳';
  }
  function commitMsg(title, projName) {
    // Strip "[project-name] " prefix for cleaner display
    return (title || '').replace(`[${projName}] `, '');
  }
  function commitType(title) {
    const m = (title || '').match(/^(?:\[[^\]]+\]\s*)?(feat|fix|docs|refactor|perf|chore|test|style)[:(!]/i);
    return m ? m[1].toLowerCase() : '';
  }
  function typeColor(t) {
    return { feat: '#2ecc71', fix: '#e74c3c', docs: '#9b59b6', refactor: '#3498db',
             perf: '#f39c12', chore: '#7f8c8d', test: '#1abc9c', style: '#e67e22' }[t] || '#555';
  }
</script>

<div class="feed-view">
  <div class="feed-hdr">
    <h2>Activity Feed</h2>
    <span class="feed-right">
      <span class="feed-count">{items.length} events</span>
      <button class="hdr-help" onclick={onHelp}>?</button>
    </span>
  </div>

  <div class="tab-bar">
    <button class="tab" class:active={tab === 'all'} onclick={() => switchTab('all')}>All</button>
    <button class="tab" class:active={tab === 'pending'} onclick={() => switchTab('pending')}>
      Pending{#if pendingCount > 0}<span class="badge">{pendingCount}</span>{/if}
    </button>
    <button class="tab" class:active={tab === 'approved'} onclick={() => switchTab('approved')}>Scored</button>
  </div>

  <!-- ═══════════ PENDING TAB ═══════════ -->
  {#if tab === 'pending'}
    {#if loadingProjects}
      <div class="loading">Loading projects...</div>
    {:else if projects.filter(p => p.pending > 0).length === 0}
      <div class="empty">
        <div class="empty-icon">✅</div>
        <p>No pending events</p>
        <p class="hint">New commits discovered every 5 min</p>
      </div>
    {:else}
      <!-- Approved projects (collapsed summary) -->
      {#if projects.filter(p => p.status === 'approved').length > 0}
        <div class="section-label">Auto-approved</div>
        {#each projects.filter(p => p.status === 'approved') as proj}
          <div class="proj-row approved-row">
            <span class="proj-icon">✅</span>
            <span class="proj-label">{proj.name}</span>
            <span class="proj-count-ok">{proj.approved} scored</span>
          </div>
        {/each}
        <div class="section-divider"></div>
      {/if}

      <!-- Pending projects (expandable) -->
      <div class="section-label">Awaiting review</div>
      {#each projects.filter(p => p.pending > 0) as proj}
        <div class="proj-group">
          <!-- Project header row -->
          <div class="proj-row" onclick={() => toggle(proj.name)}>
            <span class="proj-chevron">{expanded[proj.name] ? '▼' : '▶'}</span>
            <span class="proj-icon">{statusIcon(proj.status)}</span>
            <div class="proj-info">
              <span class="proj-label">{proj.name}</span>
              {#if proj.business}<span class="proj-biz">{proj.business}</span>{/if}
            </div>
            <span class="proj-pending-count">{proj.pending}</span>
          </div>

          <!-- Project action bar -->
          <div class="proj-action-bar">
            {#if actionInFlight === `proj-${proj.name}`}
              <span class="acting">Processing...</span>
            {:else}
              <button class="btn-approve-proj" onclick={(e) => { e.stopPropagation(); approveProject(proj.name); }}>
                ✓ Approve all {proj.pending}
              </button>
              <button class="btn-reject-proj" onclick={(e) => { e.stopPropagation(); rejectProject(proj.name); }}>
                ✗ Reject
              </button>
            {/if}
          </div>

          <!-- Expanded: nested commit list -->
          {#if expanded[proj.name]}
            <div class="commit-list">
              {#each (projectEvents[proj.name] || []) as evt}
                {@const msg = commitMsg(evt.title, proj.name)}
                {@const ctype = commitType(evt.title)}
                <div class="commit-row">
                  <div class="commit-body">
                    <div class="commit-msg">
                      {#if ctype}
                        <span class="commit-type" style="color:{typeColor(ctype)}">{ctype}</span>
                      {/if}
                      <span class="commit-text">{msg.replace(/^(feat|fix|docs|refactor|perf|chore|test|style)[:(! ]+/i, '')}</span>
                    </div>
                    <div class="commit-meta">
                      <span class="cm-lane" style="color:{laneColor(evt.lane)}">{evt.lane}</span>
                      <span class="cm-sep">·</span>
                      <span class="cm-time">{timeAgo(evt.timestamp)}</span>
                      {#if evt.url}
                        <span class="cm-sep">·</span>
                        <a class="cm-link" href={evt.url} target="_blank" rel="noopener">view</a>
                      {/if}
                      {#if evt.score_delta > 0}
                        <span class="cm-sep">·</span>
                        <span class="cm-pts">+{evt.score_delta}</span>
                      {/if}
                    </div>
                  </div>
                  <div class="commit-actions">
                    {#if actionInFlight === evt.id}
                      <span class="acting-sm">…</span>
                    {:else}
                      <button class="btn-sm ok" onclick={() => approveEvent(evt.id, proj.name)} title="Approve">✓</button>
                      <button class="btn-sm no" onclick={() => rejectEvent(evt.id, proj.name)} title="Reject">✗</button>
                    {/if}
                  </div>
                </div>
              {/each}
              {#if !(projectEvents[proj.name]?.length)}
                <div class="commit-empty">Events loading or already processed</div>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    {/if}

  <!-- ═══════════ ALL / SCORED TAB ═══════════ -->
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

  /* ── Tabs ── */
  .tab-bar { display: flex; margin-bottom: 12px; border-radius: 8px; overflow: hidden; background: #12121e; }
  .tab {
    flex: 1; padding: 8px 4px; background: transparent; border: none;
    color: #555; font-size: 13px; font-weight: 600; cursor: pointer;
    text-align: center; -webkit-tap-highlight-color: transparent;
  }
  .tab.active { color: #7c7cff; background: rgba(124,124,255,0.08); border-bottom: 2px solid #7c7cff; }
  .badge {
    min-width: 18px; height: 18px; padding: 0 5px; border-radius: 9px;
    background: #ff4444; color: white; font-size: 10px; font-weight: 700;
    margin-left: 4px; display: inline-flex; align-items: center; justify-content: center;
  }

  /* ── Section labels ── */
  .section-label {
    font-size: 10px; text-transform: uppercase; letter-spacing: 1px;
    color: #444; font-weight: 700; margin: 12px 0 6px; padding-left: 2px;
  }
  .section-divider { border-bottom: 1px solid #1a1a28; margin: 8px 0; }

  /* ── Project group ── */
  .proj-group {
    background: #111119; border: 1px solid #1e1e2e; border-radius: 10px;
    margin-bottom: 8px; overflow: hidden;
  }
  .proj-row {
    display: flex; align-items: center; gap: 8px;
    padding: 11px 12px; cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  .proj-row.approved-row {
    padding: 8px 12px; cursor: default; opacity: 0.5;
  }
  .proj-chevron { font-size: 10px; color: #555; width: 12px; flex-shrink: 0; }
  .proj-icon { font-size: 14px; flex-shrink: 0; }
  .proj-info { flex: 1; min-width: 0; display: flex; align-items: baseline; gap: 6px; }
  .proj-label { font-size: 14px; font-weight: 600; color: #ccc; }
  .proj-biz { font-size: 10px; color: #555; text-transform: uppercase; letter-spacing: 0.5px; }
  .proj-pending-count {
    background: rgba(255,170,0,0.12); color: #ffaa00; font-size: 12px;
    font-weight: 700; padding: 2px 8px; border-radius: 10px; flex-shrink: 0;
  }
  .proj-count-ok { font-size: 11px; color: #2ecc71; margin-left: auto; }

  /* ── Project action bar ── */
  .proj-action-bar {
    display: flex; gap: 8px; padding: 0 12px 10px; padding-left: 40px;
  }
  .btn-approve-proj, .btn-reject-proj {
    padding: 5px 12px; border-radius: 7px; font-size: 12px; font-weight: 600;
    border: none; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .btn-approve-proj {
    background: rgba(46,204,113,0.1); color: #2ecc71;
    border: 1px solid rgba(46,204,113,0.25);
  }
  .btn-approve-proj:active { background: rgba(46,204,113,0.25); }
  .btn-reject-proj {
    background: rgba(255,68,68,0.06); color: #ff4444;
    border: 1px solid rgba(255,68,68,0.15);
  }
  .btn-reject-proj:active { background: rgba(255,68,68,0.2); }
  .acting { color: #555; font-size: 12px; }

  /* ── Nested commit list ── */
  .commit-list {
    border-top: 1px solid #1a1a28;
    max-height: 50vh; overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
  .commit-row {
    display: flex; align-items: flex-start; gap: 8px;
    padding: 8px 12px 8px 40px;
    border-bottom: 1px solid #141420;
  }
  .commit-row:last-child { border-bottom: none; }
  .commit-body { flex: 1; min-width: 0; }
  .commit-msg {
    font-size: 12px; color: #aaa; line-height: 1.35;
    display: flex; gap: 5px; align-items: baseline;
  }
  .commit-type {
    font-size: 10px; font-weight: 700; text-transform: uppercase;
    letter-spacing: 0.3px; flex-shrink: 0;
  }
  .commit-text {
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .commit-meta {
    font-size: 10px; color: #3a3a4a; display: flex; align-items: center;
    gap: 4px; margin-top: 2px;
  }
  .cm-lane { font-weight: 600; }
  .cm-sep { opacity: 0.3; }
  .cm-link { color: #4a9eff; text-decoration: none; }
  .cm-pts { color: #2ecc71; font-weight: 600; }
  .commit-empty { padding: 12px 40px; color: #333; font-size: 12px; text-align: center; }

  .commit-actions { display: flex; gap: 4px; flex-shrink: 0; padding-top: 1px; }
  .btn-sm {
    width: 26px; height: 26px; border-radius: 6px; border: none;
    cursor: pointer; font-size: 13px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }
  .btn-sm.ok { background: rgba(46,204,113,0.1); color: #2ecc71; }
  .btn-sm.ok:active { background: rgba(46,204,113,0.25); }
  .btn-sm.no { background: rgba(255,68,68,0.06); color: #ff4444; }
  .btn-sm.no:active { background: rgba(255,68,68,0.2); }
  .acting-sm { color: #444; font-size: 11px; }

  /* ── All/Scored feed ── */
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
    color: #ffaa00; padding: 1px 5px; border-radius: 3px;
  }
  .fi-delta { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; color: #555; flex-shrink: 0; }
  .fi-delta.positive { color: #2ecc71; }
</style>
