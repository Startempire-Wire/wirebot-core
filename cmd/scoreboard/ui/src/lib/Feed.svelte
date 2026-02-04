<script>
  let { items, pendingCount = 0, onHelp, activeBusiness = '', onBusinessChange } = $props();

  const BIZ_LABELS = { SEW: '🚀', STA: '🚀', SEWN: '🕸', WB: '🤖', WIR: '🤖', PVD: '📘', PHI: '📘' };

  let tab = $state('all');
  let projects = $state([]);         // project summaries
  let projectEvents = $state({});    // { projectName: [events] }
  let expanded = $state({});         // { projectName: true/false }
  let editing = $state(null);        // project name being renamed
  let editValue = $state('');        // current rename input value
  let loadingProjects = $state(false);
  let actionInFlight = $state('');

  function getToken() {
    return new URLSearchParams(window.location.search).get('token') || 
           new URLSearchParams(window.location.search).get('key') ||
           localStorage.getItem('wb_token') || '';
  }

  function authParam() { const t = getToken(); return t ? `token=${t}` : ''; }
  function headers() {
    const h = { 'Content-Type': 'application/json' };
    const t = getToken();
    if (t) h['Authorization'] = `Bearer ${t}`;
    return h;
  }

  // ─── Load everything for pending tab ────────────────────────────────
  async function loadPending() {
    loadingProjects = true;
    try {
      // Fetch projects + all pending events in parallel (auth required)
      const [projRes, evtRes] = await Promise.all([
        fetch('/v1/projects', { headers: headers() }),
        fetch('/v1/feed?status=pending&limit=500', { headers: headers() })
      ]);
      const projData = await projRes.json();
      const evtData = await evtRes.json();

      projects = (projData.projects || []).filter(p => p.pending > 0 || p.status !== 'pending');

      // Group events by inferred project name
      const grouped = {};
      for (const evt of (evtData.items || [])) {
        const proj = inferProjectClient(evt);
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

  // ─── Rename ──────────────────────────────────────────────────────────
  function startRename(name) {
    editing = name;
    editValue = name;
  }

  function cancelRename() {
    editing = null;
    editValue = '';
  }

  async function submitRename(oldName) {
    const newName = editValue.trim();
    if (!newName || newName === oldName) { cancelRename(); return; }

    actionInFlight = `rename-${oldName}`;
    try {
      const ap = authParam();
      const res = await fetch(`/v1/projects/${encodeURIComponent(oldName)}/rename${ap ? '?' + ap : ''}`, {
        method: 'POST', headers: headers(),
        body: JSON.stringify({ new_name: newName })
      });
      const data = await res.json();
      if (data.ok) {
        // Update local state
        projects = projects.map(p => p.name === oldName ? {...p, name: newName} : p);
        if (projectEvents[oldName]) {
          projectEvents[newName] = projectEvents[oldName];
          delete projectEvents[oldName];
          projectEvents = {...projectEvents};
        }
        if (expanded[oldName]) {
          expanded[newName] = true;
          delete expanded[oldName];
          expanded = {...expanded};
        }
      }
    } catch(e) { console.error(e); }
    actionInFlight = '';
    cancelRename();
  }

  function handleRenameKey(e, oldName) {
    if (e.key === 'Enter') submitRename(oldName);
    if (e.key === 'Escape') cancelRename();
  }

  // ─── Toggle ─────────────────────────────────────────────────────────
  function toggle(name) {
    expanded = {...expanded, [name]: !expanded[name]};
  }

  function switchTab(t) {
    tab = t;
    if (t === 'projects') loadPending();
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
    if (!src) return '';
    if (src === 'github-webhook') return 'GitHub';
    if (src === 'rss-poller' || src.startsWith('rss')) return 'RSS';
    if (src === 'git-discovery') return 'Git';
    if (src === 'automated') return 'Auto';
    if (src === 'wirebot') return 'Wirebot';
    if (src.startsWith('integration:')) return 'Integration';
    if (src.includes('poller')) return src.replace('-poller', '');
    return src;
  }

  // Smarter title: strip [repo] prefix, clean up generic messages
  function smartTitle(item) {
    let t = item.title || item.type || '';

    // Strip [repo-name] prefix — the project column already shows this
    t = t.replace(/^\[[^\]]+\]\s*/, '');

    // Capitalize conventional commit type
    t = t.replace(/^(feat|fix|docs|refactor|perf|chore|test|style)[:(! ]+\s*/i, (_, type) => {
      const labels = { feat: '✨ ', fix: '🔧 ', docs: '📝 ', refactor: '♻️ ', perf: '⚡ ', chore: '🔩 ', test: '🧪 ' };
      return labels[type.toLowerCase()] || '';
    });

    // Generic system messages — make friendlier
    if (item.type === 'GDRIVE_SCAN') return `📁 Drive indexed ${t.match(/(\d+)/)?.[1] || ''} files`;
    if (item.type === 'SYSTEMS_SCORE') return t.replace('Systems score:', '⚙️ Systems');
    if (item.type === 'REVENUE_EVENT') return t.replace('Revenue score:', '💰 Revenue');
    if (item.type === 'SSL_CHECK') return t;
    if (item.type === 'DISK_CHECK') return t;
    if (item.type === 'INTEGRATION_CHECK') return t;

    return t || item.type;
  }

  // Deduplicate: hide duplicate commits, collapse repeated scans into one entry
  function dedupeItems(raw) {
    const out = [];
    // Collect repos that have PRODUCT_RELEASE events
    const releaseRepos = new Set();
    // Collect repeated event types to collapse (keep first, count rest)
    const collapsed = {};  // type → { item, count }

    for (const item of raw) {
      if (item.type === 'PRODUCT_RELEASE') {
        const m = (item.title || '').match(/^\[([^\]]+)\]/);
        if (m) releaseRepos.add(m[1]);
      }
    }

    for (const item of raw) {
      // Skip commit events when PRODUCT_RELEASE covers that repo
      if (item.type === 'commit') {
        const repo = (item.url || '').match(/\/([^/]+?)(?:\.git)?$/)?.[1] || '';
        if (repo && releaseRepos.has(repo)) continue;
      }

      // Collapse repeated scan/check types into one
      if (['GDRIVE_SCAN', 'DROPBOX_SCAN'].includes(item.type)) {
        if (!collapsed[item.type]) {
          collapsed[item.type] = { item: { ...item }, count: 1 };
        } else {
          collapsed[item.type].count++;
        }
        continue;
      }

      out.push(item);
    }

    // Insert collapsed items at their natural position (after first few items)
    for (const { item, count } of Object.values(collapsed)) {
      if (count > 1) item._count = count;
      out.push(item);
    }

    return out;
  }
  function statusIcon(s) {
    return { approved: '✅', rejected: '❌' }[s] || '⏳';
  }
  // Infer project from event signals — mirrors server-side inferProject
  const ghRepoMap = {
    'wirebot-core': 'wirebot-core', 'focusa': 'focusa',
    'Startempire-Wire-Network': 'chrome-extension',
    'Startempire-Wire-Network-Ring-Leader': 'ring-leader',
    'Startempire-Wire-Network-Connect': 'connect-plugin',
    'Startempire-Wire-Network-Parent-Core': 'parent-core',
    'Startempire-Wire-Network-Websockets': 'websockets',
    'Startempire-Wire-Network-Screenshots': 'screenshots',
  };

  function inferProjectClient(evt) {
    const title = evt.title || '';
    const url = evt.url || '';
    const source = evt.source || '';

    // 1. Title prefix [name]
    const m = title.match(/^\[([^\]]+)\]/);
    if (m) return m[1];

    // 2. GitHub URL
    const gh = url.match(/github\.com\/([^/]+)\/([^/]+)/);
    if (gh) {
      const repo = gh[2].replace('.git', '');
      return ghRepoMap[repo] || repo.toLowerCase();
    }

    // 3. URL domain
    if (url.includes('startempirewire.com')) return 'startempirewire.com';
    if (url.includes('startempirewire.network')) return 'startempirewire.network';
    if (url.includes('wirebot.chat')) return 'wirebot';

    // 4. Source fallback
    if (source === 'github-webhook') return 'github';
    if (source === 'rss-poller') return 'rss-content';
    if (source === 'stripe-webhook') return 'stripe';

    return 'other';
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
    <button class="tab" class:active={tab === 'projects'} onclick={() => switchTab('projects')}>
      Projects{#if pendingCount > 0}<span class="badge">{pendingCount}</span>{/if}
    </button>
    <button class="tab" class:active={tab === 'approved'} onclick={() => switchTab('approved')}>Scored</button>
  </div>

  <!-- ═══════════ PROJECTS TAB ═══════════ -->
  {#if tab === 'projects'}
    {#if loadingProjects}
      <div class="loading">Loading projects...</div>
    {:else if projects.length === 0}
      <div class="empty">
        <div class="empty-icon">📂</div>
        <p>No projects discovered</p>
        <p class="hint">Discovery runs every 5 min via cron</p>
      </div>
    {:else}
      <!-- Pending projects first -->
      {#if projects.filter(p => p.pending > 0).length > 0}
        <div class="section-label">Awaiting review</div>
        {#each projects.filter(p => p.pending > 0) as proj}
          {@const evts = projectEvents[proj.name] || []}
          <div class="proj-group">
            <div class="proj-row" onclick={() => toggle(proj.name)}>
              <span class="proj-chevron">{expanded[proj.name] ? '▼' : '▶'}</span>
              <span class="proj-icon">{statusIcon(proj.status)}</span>
              <div class="proj-info">
                {#if editing === proj.name}
                  <input class="rename-input" bind:value={editValue}
                    onkeydown={(e) => handleRenameKey(e, proj.name)}
                    onblur={() => submitRename(proj.name)}
                    onclick={(e) => e.stopPropagation()} autofocus />
                {:else}
                  <span class="proj-label">{proj.name}</span>
                  <button class="btn-rename" onclick={(e) => { e.stopPropagation(); startRename(proj.name); }}>✎</button>
                {/if}
                {#if proj.business}<span class="proj-biz">{proj.business}</span>{/if}
              </div>
              <span class="proj-pending-count">{proj.pending}</span>
            </div>
            <div class="proj-action-bar">
              {#if actionInFlight === `proj-${proj.name}`}
                <span class="acting">Processing...</span>
              {:else}
                <button class="btn-approve-proj" onclick={(e) => { e.stopPropagation(); approveProject(proj.name); }}>
                  ✓ Approve all {proj.pending}
                </button>
                <button class="btn-reject-proj" onclick={(e) => { e.stopPropagation(); rejectProject(proj.name); }}>✗ Reject</button>
              {/if}
            </div>
            {#if expanded[proj.name]}
              <div class="commit-list">
                {#each evts as evt}
                  {@const msg = commitMsg(evt.title, proj.name)}
                  {@const ctype = commitType(evt.title)}
                  <div class="commit-row">
                    <div class="commit-body">
                      <div class="commit-msg">
                        {#if ctype}<span class="commit-type" style="color:{typeColor(ctype)}">{ctype}</span>{/if}
                        <span class="commit-text">{msg.replace(/^(feat|fix|docs|refactor|perf|chore|test|style)[:(! ]+/i, '')}</span>
                      </div>
                      <div class="commit-meta">
                        <span class="cm-lane" style="color:{laneColor(evt.lane)}">{evt.lane}</span>
                        <span class="cm-sep">·</span>
                        <span class="cm-time">{timeAgo(evt.timestamp)}</span>
                        {#if evt.url}<span class="cm-sep">·</span><a class="cm-link" href={evt.url} target="_blank" rel="noopener">view</a>{/if}
                        {#if evt.score_delta > 0}<span class="cm-sep">·</span><span class="cm-pts">+{evt.score_delta}</span>{/if}
                      </div>
                    </div>
                    <div class="commit-actions">
                      {#if actionInFlight === evt.id}
                        <span class="acting-sm">…</span>
                      {:else}
                        <button class="btn-sm ok" onclick={() => approveEvent(evt.id, proj.name)}>✓</button>
                        <button class="btn-sm no" onclick={() => rejectEvent(evt.id, proj.name)}>✗</button>
                      {/if}
                    </div>
                  </div>
                {/each}
                {#if evts.length === 0}
                  <div class="commit-empty">No pending events loaded</div>
                {/if}
              </div>
            {/if}
          </div>
        {/each}
      {/if}

      <!-- All tracked projects -->
      <div class="section-label">All projects</div>
      {#each projects as proj}
        <div class="proj-group proj-summary">
          <div class="proj-row" onclick={() => toggle(proj.name)}>
            <span class="proj-chevron">{expanded[proj.name] ? '▼' : '▶'}</span>
            <span class="proj-icon">{statusIcon(proj.status)}</span>
            <div class="proj-info">
              {#if editing === proj.name}
                <input class="rename-input" bind:value={editValue}
                  onkeydown={(e) => handleRenameKey(e, proj.name)}
                  onblur={() => submitRename(proj.name)}
                  onclick={(e) => e.stopPropagation()} autofocus />
              {:else}
                <span class="proj-label">{proj.name}</span>
                <button class="btn-rename" onclick={(e) => { e.stopPropagation(); startRename(proj.name); }}>✎</button>
              {/if}
              {#if proj.business}<span class="proj-biz">{proj.business}</span>{/if}
            </div>
            <div class="proj-score-bar">
              <span class="ps-approved">{proj.approved}</span>
              {#if proj.pending > 0}<span class="ps-pending">+{proj.pending}</span>{/if}
              {#if proj.rejected > 0}<span class="ps-rejected">-{proj.rejected}</span>{/if}
            </div>
          </div>

          <!-- Expanded: source breakdown + quick stats -->
          {#if expanded[proj.name] && proj.pending === 0}
            <div class="proj-detail">
              <div class="pd-row"><span class="pd-label">Events</span><span class="pd-val">{proj.total_events}</span></div>
              <div class="pd-row"><span class="pd-label">Approved</span><span class="pd-val pd-ok">{proj.approved}</span></div>
              {#if proj.rejected > 0}
                <div class="pd-row"><span class="pd-label">Rejected</span><span class="pd-val pd-no">{proj.rejected}</span></div>
              {/if}
              <div class="pd-row"><span class="pd-label">Lane</span><span class="pd-val" style="color:{laneColor(proj.primary_lane)}">{proj.primary_lane}</span></div>
              {#if proj.sources?.length > 0}
                <div class="pd-row"><span class="pd-label">Sources</span><span class="pd-val">{proj.sources.join(', ')}</span></div>
              {/if}
              {#if proj.auto_approve}
                <div class="pd-row"><span class="pd-label">Mode</span><span class="pd-val pd-ok">Auto-approve ✓</span></div>
              {/if}
            </div>
          {/if}
        </div>
      {/each}
    {/if}

  <!-- ═══════════ ALL / SCORED TAB ═══════════ -->
  {:else}
    <!-- Business filter pills -->
    {#if items.some(i => i.business_id)}
      <div class="biz-pills">
        <button class="biz-pill" class:active={!activeBusiness} onclick={() => onBusinessChange?.('')}>All</button>
        {#each [...new Set(items.map(i => i.business_id).filter(Boolean))] as biz}
          <button class="biz-pill" class:active={activeBusiness === biz} onclick={() => onBusinessChange?.(biz)}>
            {BIZ_LABELS[biz] || '🏢'} {biz}
          </button>
        {/each}
      </div>
    {/if}

    {#if items.length === 0}
      <div class="empty">
        <div class="empty-icon">📋</div>
        <p>No events yet</p>
        <p class="hint">Push events via <code>wb ship</code> or the API</p>
      </div>
    {:else}
      <div class="feed-list">
        {#each dedupeItems(items.filter(i => tab === 'all' || i.status === 'approved')) as item}
          <div class="feed-item" class:is-pending={item.status === 'pending'}>
            <div class="fi-icon">{item.icon || '📌'}</div>
            <div class="fi-body">
              <div class="fi-title">{smartTitle(item)}{#if item._count} <span class="fi-repeat">×{item._count}</span>{/if}</div>
              <div class="fi-meta">
                <span class="fi-lane" style="color:{laneColor(item.lane)}">{item.lane}</span>
                <span class="fi-sep">·</span>
                <span>{sourceLabel(item.source)}</span>
                <span class="fi-sep">·</span>
                <span>{timeAgo(item.timestamp)}</span>
                {#if item.business_id}
                  <span class="fi-sep">·</span>
                  <span class="fi-biz-tag">{BIZ_LABELS[item.business_id] || ''} {item.business_id}</span>
                {/if}
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
    margin-bottom: 8px; border-bottom: 1px solid var(--border); padding-bottom: 8px;
  }
  .feed-hdr h2 { font-size: 16px; font-weight: 700; color: var(--accent); }
  .feed-right { display: flex; align-items: center; gap: 8px; }
  .feed-count { font-size: 12px; color: var(--text-muted); }
  .hdr-help {
    width: 22px; height: 22px; border-radius: 50%;
    background: rgba(124,124,255,0.1); border: 1px solid rgba(124,124,255,0.25);
    color: var(--accent); font-size: 12px; font-weight: 700; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
  }

  /* ── Tabs ── */
  .tab-bar { display: flex; margin-bottom: 12px; border-radius: 8px; overflow: hidden; background: var(--bg-card); }
  .tab {
    flex: 1; padding: 8px 4px; background: transparent; border: none;
    color: var(--text-muted); font-size: 13px; font-weight: 600; cursor: pointer;
    text-align: center; -webkit-tap-highlight-color: transparent;
  }
  .tab.active { color: var(--accent); background: rgba(124,124,255,0.08); border-bottom: 2px solid #7c7cff; }
  .badge {
    min-width: 18px; height: 18px; padding: 0 5px; border-radius: 9px;
    background: #ff4444; color: white; font-size: 10px; font-weight: 700;
    margin-left: 4px; display: inline-flex; align-items: center; justify-content: center;
  }

  /* ── Section labels ── */
  .section-label {
    font-size: 10px; text-transform: uppercase; letter-spacing: 1px;
    color: var(--text-muted); font-weight: 700; margin: 12px 0 6px; padding-left: 2px;
  }
  .section-divider { border-bottom: 1px solid var(--border); margin: 8px 0; }

  /* ── Project group ── */
  .proj-group {
    background: var(--bg-card); border: 1px solid var(--border); border-radius: 10px;
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
  .proj-chevron { font-size: 10px; color: var(--text-muted); width: 12px; flex-shrink: 0; }
  .proj-icon { font-size: 14px; flex-shrink: 0; }
  .proj-info { flex: 1; min-width: 0; display: flex; align-items: baseline; gap: 6px; }
  .proj-label { font-size: 14px; font-weight: 600; color: var(--text); }
  .proj-biz { font-size: 10px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px; }
  .rename-input {
    background: var(--bg); border: 1px solid var(--accent); border-radius: 5px;
    color: var(--text); font-size: 13px; font-weight: 600; padding: 3px 8px;
    width: 140px; outline: none;
  }
  .rename-input:focus { border-color: var(--accent); box-shadow: 0 0 0 2px rgba(124,124,255,0.15); }
  .btn-rename {
    background: none; border: none; color: var(--text-muted); font-size: 12px;
    cursor: pointer; padding: 2px 4px; opacity: 0;
    transition: opacity 0.15s; -webkit-tap-highlight-color: transparent;
  }
  .proj-info:hover .btn-rename, .proj-row:hover .btn-rename,
  .approved-row:hover .btn-rename { opacity: 1; }
  /* Always show on touch (no hover) */
  @media (hover: none) { .btn-rename { opacity: 0.5; } }

  .proj-pending-count {
    background: rgba(255,170,0,0.12); color: var(--warning); font-size: 12px;
    font-weight: 700; padding: 2px 8px; border-radius: 10px; flex-shrink: 0;
  }
  .proj-count-ok { font-size: 11px; color: var(--success); margin-left: auto; }

  /* ── Project action bar ── */
  .proj-action-bar {
    display: flex; gap: 8px; padding: 0 12px 10px; padding-left: 40px;
  }
  .btn-approve-proj, .btn-reject-proj {
    padding: 5px 12px; border-radius: 7px; font-size: 12px; font-weight: 600;
    border: none; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .btn-approve-proj {
    background: rgba(46,204,113,0.1); color: var(--success);
    border: 1px solid rgba(46,204,113,0.25);
  }
  .btn-approve-proj:active { background: rgba(46,204,113,0.25); }
  .btn-reject-proj {
    background: rgba(255,68,68,0.06); color: var(--error);
    border: 1px solid rgba(255,68,68,0.15);
  }
  .btn-reject-proj:active { background: rgba(255,68,68,0.2); }
  .acting { color: var(--text-muted); font-size: 12px; }

  /* ── Nested commit list ── */
  .commit-list {
    border-top: 1px solid var(--border);
    max-height: 50vh; overflow-y: auto;
    -webkit-overflow-scrolling: touch;
  }
  .commit-row {
    display: flex; align-items: flex-start; gap: 8px;
    padding: 8px 12px 8px 40px;
    border-bottom: 1px solid var(--border);
  }
  .commit-row:last-child { border-bottom: none; }
  .commit-body { flex: 1; min-width: 0; }
  .commit-msg {
    font-size: 12px; color: var(--text-secondary); line-height: 1.35;
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
    font-size: 10px; color: var(--text-muted); display: flex; align-items: center;
    gap: 4px; margin-top: 2px;
  }
  .cm-lane { font-weight: 600; }
  .cm-sep { opacity: 0.3; }
  .cm-link { color: var(--accent); text-decoration: none; }
  .cm-pts { color: var(--success); font-weight: 600; }
  .commit-empty { padding: 12px 40px; color: var(--text-muted); font-size: 12px; text-align: center; }

  /* ── Project score bar (All projects section) ── */
  .proj-summary { border-color: var(--border); }
  .proj-score-bar { display: flex; gap: 4px; align-items: center; flex-shrink: 0; }
  .ps-approved { font-size: 13px; font-weight: 700; color: var(--success); }
  .ps-pending { font-size: 11px; color: var(--warning); }
  .ps-rejected { font-size: 11px; color: var(--error); }

  /* ── Project detail panel ── */
  .proj-detail {
    border-top: 1px solid var(--border); padding: 8px 14px 10px 40px;
    background: rgba(0,0,0,0.1);
  }
  .pd-row { display: flex; justify-content: space-between; padding: 3px 0; font-size: 12px; }
  .pd-label { color: var(--text-muted); }
  .pd-val { color: var(--text-secondary); }
  .pd-ok { color: var(--success); }
  .pd-no { color: var(--error); }

  .commit-actions { display: flex; gap: 4px; flex-shrink: 0; padding-top: 1px; }
  .btn-sm {
    width: 26px; height: 26px; border-radius: 6px; border: none;
    cursor: pointer; font-size: 13px; font-weight: 700;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }
  .btn-sm.ok { background: rgba(46,204,113,0.1); color: var(--success); }
  .btn-sm.ok:active { background: rgba(46,204,113,0.25); }
  .btn-sm.no { background: rgba(255,68,68,0.06); color: var(--error); }
  .btn-sm.no:active { background: rgba(255,68,68,0.2); }
  .acting-sm { color: var(--text-muted); font-size: 11px; }

  /* ── All/Scored feed ── */
  .empty { text-align: center; padding: 40px 0; }
  .empty-icon { font-size: 40px; margin-bottom: 8px; }
  .empty p { font-size: 14px; opacity: 0.5; }
  .hint { margin-top: 4px; }
  .hint code { font-family: monospace; color: var(--accent); }
  .loading { text-align: center; padding: 30px; color: var(--text-muted); font-size: 13px; }

  .feed-list { display: flex; flex-direction: column; gap: 2px; }
  .feed-item {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 0; border-bottom: 1px solid var(--border);
  }
  .feed-item.is-pending { opacity: 0.45; }
  .fi-icon { font-size: 20px; flex-shrink: 0; }
  .fi-body { flex: 1; min-width: 0; }
  .fi-title { font-size: 13px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .fi-repeat { font-size: 11px; opacity: 0.5; font-weight: 400; }
  .fi-meta { font-size: 11px; opacity: 0.4; display: flex; align-items: center; gap: 4px; margin-top: 2px; flex-wrap: wrap; }
  .fi-lane { font-weight: 600; opacity: 1; }
  .fi-sep { opacity: 0.3; }
  .fi-pending-badge {
    font-size: 9px; font-weight: 700; background: rgba(255,170,0,0.15);
    color: var(--warning); padding: 1px 5px; border-radius: 3px;
  }
  .fi-biz-tag {
    font-size: 9px; font-weight: 600; background: rgba(124,124,255,0.1);
    color: var(--accent); padding: 1px 5px; border-radius: 3px;
  }
  .fi-delta { font-size: 14px; font-weight: 700; font-variant-numeric: tabular-nums; color: var(--text-muted); flex-shrink: 0; }
  .fi-delta.positive { color: var(--success); }

  /* Business filter pills */
  .biz-pills {
    display: flex; gap: 6px; overflow-x: auto; padding: 4px 0 10px;
    -webkit-overflow-scrolling: touch; scrollbar-width: none;
  }
  .biz-pills::-webkit-scrollbar { display: none; }
  .biz-pill {
    padding: 5px 12px; border-radius: 20px; font-size: 11px; font-weight: 600;
    background: var(--bg-card); border: 1px solid var(--border); color: var(--text-secondary);
    cursor: pointer; white-space: nowrap; transition: all 0.15s;
  }
  .biz-pill:hover { border-color: #7c7cff40; color: var(--text-secondary); }
  .biz-pill.active { background: #7c7cff15; border-color: var(--accent); color: var(--accent); }
</style>
