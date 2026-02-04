<script>
  import { onMount } from 'svelte';

  let grid = $state(null);     // { days: {date: {source: {approved,pending,rejected}}}, sources: [], totals: {} }
  let loading = $state(true);
  let selected = $state(null);  // full memory item for popup
  let loadingItem = $state(false);
  let editing = $state(false);
  let editText = $state('');
  let saving = $state(false);
  let filter = $state('all');   // all | approved | pending | rejected
  let listItems = $state([]);
  let listLoading = $state(false);
  let listOffset = $state(0);
  let hasMore = $state(true);

  const API = '';
  function token() {
    return localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || '';
  }
  function hdrs() { return token() ? { 'Authorization': `Bearer ${token()}` } : {}; }

  onMount(() => {
    loadGrid();
    loadList();
  });

  async function loadGrid() {
    try {
      const res = await fetch(`${API}/v1/memory/grid`, { headers: hdrs() });
      if (res.ok) grid = await res.json();
    } catch (e) { console.error('Grid load failed:', e); }
    loading = false;
  }

  async function loadList(append = false) {
    listLoading = true;
    try {
      const status = filter === 'all' ? '' : filter;
      const params = `status=${status}&limit=50&offset=${append ? listOffset : 0}`;
      const res = await fetch(`${API}/v1/memory/queue?${params}`, { headers: hdrs() });
      if (res.ok) {
        const data = await res.json();
        const items = data.items || [];
        if (append) {
          listItems = [...listItems, ...items];
        } else {
          listItems = items;
          listOffset = 0;
        }
        listOffset = listItems.length;
        hasMore = items.length === 50;
      }
    } catch (e) { console.error(e); }
    listLoading = false;
  }

  function changeFilter(f) {
    filter = f;
    loadList();
  }

  async function openItem(id) {
    loadingItem = true;
    editing = false;
    try {
      const res = await fetch(`${API}/v1/memory/item/${id}`, { headers: hdrs() });
      if (res.ok) selected = await res.json();
    } catch (e) { console.error(e); }
    loadingItem = false;
  }

  function startEdit() {
    editText = selected.correction || selected.memory_text;
    editing = true;
  }

  async function saveEdit() {
    if (!editText.trim() || !selected) return;
    saving = true;
    try {
      const res = await fetch(`${API}/v1/memory/item/${selected.id}`, {
        method: 'PATCH',
        headers: { ...hdrs(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ memory_text: editText.trim() }),
      });
      if (res.ok) {
        const data = await res.json();
        selected = { ...selected, correction: editText.trim(), correction_hash: data.hash };
        editing = false;
      }
    } catch (e) { console.error(e); }
    saving = false;
  }

  async function doAction(id, action) {
    try {
      await fetch(`${API}/v1/memory/queue/${id}/${action}`, { method: 'POST', headers: hdrs() });
      // Update local state
      listItems = listItems.map(i => i.id === id ? { ...i, status: action === 'reject' ? 'rejected' : 'approved' } : i);
      if (selected?.id === id) selected = { ...selected, status: action === 'reject' ? 'rejected' : 'approved' };
      loadGrid(); // refresh counts
    } catch (e) { console.error(e); }
  }

  function closePopup() { selected = null; editing = false; }

  // Grid helpers
  function sortedDays(days) {
    return Object.keys(days || {}).sort().reverse();
  }

  function cellColor(cell) {
    if (!cell) return 'rgba(255,255,255,0.05)';
    if (cell.approved > 0 && cell.pending === 0) return '#10b981';
    if (cell.pending > 0 && cell.approved === 0) return '#f59e0b';
    if (cell.rejected > 0 && cell.approved === 0 && cell.pending === 0) return '#ef4444';
    return '#6366f1';
  }

  function cellCount(cell) {
    if (!cell) return 0;
    return (cell.approved || 0) + (cell.pending || 0) + (cell.rejected || 0);
  }

  function cellTooltip(day, source, cell) {
    if (!cell) return `${day} · ${source}: empty`;
    const parts = [];
    if (cell.approved) parts.push(`${cell.approved} ✓`);
    if (cell.pending) parts.push(`${cell.pending} ⏳`);
    if (cell.rejected) parts.push(`${cell.rejected} ✗`);
    return `${day} · ${source}: ${parts.join(', ')}`;
  }

  function sourceIcon(s) {
    return { conversation: '💬', gdrive: '📁', ai_chat: '🤖', obsidian: '📝',
             recovered: '♻️', dropbox: '📦', unknown: '❓' }[s] || '📄';
  }

  function destBadge(d) {
    return { mem0: '🟢', letta: '🔵', memory_md: '📄', git: '🔒' }[d] || '⚪';
  }

  function timeAgo(ts) {
    if (!ts) return '';
    const mins = Math.floor((Date.now() - new Date(ts).getTime()) / 60000);
    if (mins < 60) return `${mins}m`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h`;
    return `${Math.floor(hrs / 24)}d`;
  }

  function statusBadge(s) {
    return { approved: '✓', pending: '⏳', rejected: '✗' }[s] || s;
  }

  function statusClass(s) {
    return { approved: 'st-approved', pending: 'st-pending', rejected: 'st-rejected' }[s] || '';
  }

  // Portal for popup — escape transform stacking context
  function portal(node) {
    document.body.appendChild(node);
    return { destroy() { node.remove(); } };
  }
</script>

<div class="memory-audit">
  <!-- Header -->
  <div class="ma-hdr">
    <h2>🧠 Memory</h2>
    {#if grid?.totals}
      <div class="ma-totals">
        <span class="mt-item mt-all">{grid.totals.total}</span>
        <span class="mt-item mt-ok">{grid.totals.approved} ✓</span>
        <span class="mt-item mt-wait">{grid.totals.pending} ⏳</span>
        <span class="mt-item mt-no">{grid.totals.rejected} ✗</span>
      </div>
    {/if}
  </div>

  <!-- Heatmap Grid -->
  {#if loading}
    <div class="ma-loading">Loading memory grid...</div>
  {:else if grid}
    <div class="ma-grid-wrap">
      <!-- Column headers = source types -->
      <div class="ma-grid-header">
        <div class="ma-day-label"></div>
        {#each grid.sources || [] as src}
          <div class="ma-src-label" title={src}>{sourceIcon(src)}</div>
        {/each}
      </div>
      <!-- Rows = days -->
      <div class="ma-grid-body">
        {#each sortedDays(grid.days) as day}
          <div class="ma-grid-row">
            <div class="ma-day-label">{day.slice(5)}</div>
            {#each grid.sources || [] as src}
              {@const cell = grid.days[day]?.[src]}
              <div class="ma-cell"
                style="background:{cellColor(cell)}; opacity:{cell ? Math.max(0.3, Math.min(1, cellCount(cell) / 100)) : 0.1}"
                title={cellTooltip(day, src, cell)}>
                {#if cellCount(cell) > 0}<span class="ma-cell-n">{cellCount(cell)}</span>{/if}
              </div>
            {/each}
          </div>
        {/each}
      </div>
    </div>

    <!-- Legend -->
    <div class="ma-legend">
      <span class="ml-swatch" style="background:#10b981"></span> Approved
      <span class="ml-swatch" style="background:#f59e0b"></span> Pending
      <span class="ml-swatch" style="background:#ef4444"></span> Rejected
      <span class="ml-swatch" style="background:#6366f1"></span> Mixed
    </div>
  {/if}

  <!-- Filter tabs -->
  <div class="ma-filters">
    {#each ['all', 'approved', 'pending', 'rejected'] as f}
      <button class="ma-filter" class:active={filter === f} onclick={() => changeFilter(f)}>
        {f === 'all' ? 'All' : f === 'approved' ? '✓ Approved' : f === 'pending' ? '⏳ Pending' : '✗ Rejected'}
      </button>
    {/each}
  </div>

  <!-- Memory list -->
  <div class="ma-list">
    {#each listItems as item}
      <button class="ma-item" class:ma-approved={item.status === 'approved'} class:ma-pending={item.status === 'pending'} class:ma-rejected={item.status === 'rejected'}
        onclick={() => openItem(item.id)}>
        <div class="mi-icon">{sourceIcon(item.source_type)}</div>
        <div class="mi-body">
          <div class="mi-text">{item.correction || item.memory_text}</div>
          <div class="mi-meta">
            <span class="mi-status {statusClass(item.status)}">{statusBadge(item.status)}</span>
            <span class="mi-conf">{Math.round(item.confidence * 100)}%</span>
            <span class="mi-time">{timeAgo(item.created_at)}</span>
            {#if item.correction}<span class="mi-edited">✎</span>{/if}
          </div>
        </div>
      </button>
    {/each}
    {#if hasMore}
      <button class="ma-loadmore" onclick={() => loadList(true)} disabled={listLoading}>
        {listLoading ? 'Loading...' : 'Load more'}
      </button>
    {/if}
    {#if listItems.length === 0 && !listLoading}
      <div class="ma-empty">No memories in this filter</div>
    {/if}
  </div>
</div>

<!-- Detail popup (portaled to body) -->
{#if selected}
  <div class="ma-overlay" use:portal>
    <div class="ma-backdrop" onclick={closePopup} role="presentation"></div>
    <div class="ma-popup">
      <div class="mp-header">
        <span class="mp-source">{sourceIcon(selected.source_type)} {selected.source_type}</span>
        <span class="mp-status {statusClass(selected.status)}">{selected.status}</span>
        <button class="mp-close" onclick={closePopup}>✕</button>
      </div>

      <!-- Memory text -->
      <div class="mp-section">
        <div class="mp-label">Memory</div>
        {#if editing}
          <textarea class="mp-edit" bind:value={editText} rows="3"></textarea>
          <div class="mp-edit-actions">
            <button class="mp-btn mp-save" onclick={saveEdit} disabled={saving}>{saving ? 'Saving...' : 'Save'}</button>
            <button class="mp-btn mp-cancel" onclick={() => editing = false}>Cancel</button>
          </div>
        {:else}
          <div class="mp-text">{selected.correction || selected.memory_text}</div>
          {#if selected.correction}
            <div class="mp-original">
              <span class="mp-label-sm">Original:</span> {selected.memory_text}
            </div>
          {/if}
          <button class="mp-btn mp-edit-btn" onclick={startEdit}>✎ Edit</button>
        {/if}
      </div>

      <!-- Correction hash -->
      {#if selected.correction_hash}
        <div class="mp-hash">
          <span class="mp-label-sm">Edit hash:</span>
          <code>{selected.correction_hash}</code>
        </div>
      {/if}

      <!-- Source context -->
      {#if selected.source_context}
        <div class="mp-section">
          <div class="mp-label">Inferred from</div>
          <div class="mp-context">{selected.source_context}</div>
          {#if selected.source_file}
            <div class="mp-file">📎 {selected.source_file}</div>
          {/if}
        </div>
      {/if}

      <!-- Confidence + destinations -->
      <div class="mp-meta-row">
        <div class="mp-meta-item">
          <span class="mp-label-sm">Confidence</span>
          <div class="mp-conf-bar">
            <div class="mp-conf-fill" style="width:{selected.confidence * 100}%"></div>
          </div>
          <span>{Math.round(selected.confidence * 100)}%</span>
        </div>
      </div>

      <!-- Storage destinations -->
      <div class="mp-section">
        <div class="mp-label">Stored in</div>
        <div class="mp-dests">
          {#if selected.destinations?.length > 0}
            {#each selected.destinations as d}
              <span class="mp-dest">{destBadge(d)} {d}</span>
            {/each}
          {:else}
            <span class="mp-dest mp-dest-none">Not yet written to any store</span>
          {/if}
        </div>
      </div>

      <!-- Timestamps -->
      <div class="mp-timestamps">
        <span>Created: {selected.created_at?.replace('T', ' ').slice(0, 19)}</span>
        {#if selected.reviewed_at}
          <span>Reviewed: {selected.reviewed_at?.replace('T', ' ').slice(0, 19)}</span>
        {/if}
      </div>

      <!-- Actions for pending -->
      {#if selected.status === 'pending'}
        <div class="mp-actions">
          <button class="mp-btn mp-approve" onclick={() => doAction(selected.id, 'approve')}>✓ Approve</button>
          <button class="mp-btn mp-reject" onclick={() => doAction(selected.id, 'reject')}>✗ Reject</button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .memory-audit { padding: 16px; max-width: 600px; margin: 0 auto; }
  .ma-hdr { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
  .ma-hdr h2 { margin: 0; font-size: 20px; }
  .ma-totals { display: flex; gap: 8px; font-size: 13px; }
  .mt-item { padding: 2px 8px; border-radius: 10px; }
  .mt-all { background: var(--bg-hover); }
  .mt-ok { color: #10b981; }
  .mt-wait { color: #f59e0b; }
  .mt-no { color: #ef4444; }

  /* Grid */
  .ma-grid-wrap { overflow-x: auto; margin-bottom: 8px; }
  .ma-grid-header { display: flex; gap: 2px; padding-left: 48px; margin-bottom: 2px; }
  .ma-src-label { width: 28px; text-align: center; font-size: 14px; }
  .ma-grid-body { max-height: 200px; overflow-y: auto; }
  .ma-grid-row { display: flex; gap: 2px; margin-bottom: 2px; }
  .ma-day-label { width: 44px; font-size: 11px; opacity: 0.6; text-align: right; padding-right: 4px; line-height: 28px; flex-shrink: 0; }
  .ma-cell { width: 28px; height: 28px; border-radius: 4px; display: flex; align-items: center; justify-content: center; cursor: pointer; transition: transform 0.1s; }
  .ma-cell:hover { transform: scale(1.2); z-index: 1; }
  .ma-cell-n { font-size: 9px; font-weight: 600; color: #fff; text-shadow: 0 1px 2px rgba(0,0,0,0.5); }

  .ma-legend { display: flex; gap: 12px; font-size: 11px; opacity: 0.7; padding: 4px 0 12px; align-items: center; }
  .ml-swatch { width: 12px; height: 12px; border-radius: 3px; display: inline-block; }

  /* Filters */
  .ma-filters { display: flex; gap: 4px; margin-bottom: 12px; }
  .ma-filter { padding: 6px 12px; border-radius: 16px; border: 1px solid rgba(255,255,255,0.1); background: transparent; color: inherit; font-size: 12px; cursor: pointer; }
  .ma-filter.active { background: var(--bg-hover); border-color: rgba(124,124,255,0.3); }

  /* List */
  .ma-list { display: flex; flex-direction: column; gap: 2px; }
  .ma-item { display: flex; gap: 10px; padding: 10px 12px; border-radius: 10px; background: var(--bg-hover); border: 1px solid transparent; cursor: pointer; text-align: left; transition: border-color 0.15s; }
  .ma-item:hover { border-color: rgba(124,124,255,0.2); }
  .ma-item.ma-approved { border-left: 3px solid #10b981; }
  .ma-item.ma-pending { border-left: 3px solid #f59e0b; }
  .ma-item.ma-rejected { border-left: 3px solid #ef4444; opacity: 0.6; }
  .mi-icon { font-size: 18px; flex-shrink: 0; padding-top: 2px; }
  .mi-body { flex: 1; min-width: 0; }
  .mi-text { font-size: 13px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .mi-meta { display: flex; gap: 8px; font-size: 11px; opacity: 0.6; margin-top: 3px; }
  .mi-status { font-weight: 600; }
  .st-approved { color: #10b981; }
  .st-pending { color: #f59e0b; }
  .st-rejected { color: #ef4444; }
  .mi-edited { color: #6366f1; }

  .ma-loadmore { padding: 10px; text-align: center; opacity: 0.6; cursor: pointer; background: transparent; border: 1px dashed rgba(255,255,255,0.1); border-radius: 8px; color: inherit; }
  .ma-empty { text-align: center; padding: 40px; opacity: 0.4; }
  .ma-loading { text-align: center; padding: 40px; opacity: 0.5; }

  /* Popup overlay */
  .ma-overlay { position: fixed; inset: 0; z-index: 9999; display: flex; align-items: flex-end; justify-content: center; }
  .ma-backdrop { position: absolute; inset: 0; background: rgba(0,0,0,0.6); }
  .ma-popup { position: relative; background: var(--bg-card, #1a1a2e); border-radius: 16px 16px 0 0; width: 100%; max-width: 500px; max-height: 85vh; overflow-y: auto; padding: 20px; }

  .mp-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
  .mp-source { font-size: 14px; font-weight: 600; }
  .mp-status { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
  .mp-status.st-approved { background: rgba(16,185,129,0.15); }
  .mp-status.st-pending { background: rgba(245,158,11,0.15); }
  .mp-status.st-rejected { background: rgba(239,68,68,0.15); }
  .mp-close { background: none; border: none; font-size: 18px; cursor: pointer; color: inherit; opacity: 0.5; }

  .mp-section { margin-bottom: 16px; }
  .mp-label { font-size: 11px; text-transform: uppercase; letter-spacing: 0.5px; opacity: 0.5; margin-bottom: 6px; }
  .mp-label-sm { font-size: 10px; opacity: 0.5; }
  .mp-text { font-size: 14px; line-height: 1.5; padding: 10px; background: rgba(255,255,255,0.03); border-radius: 8px; }
  .mp-original { font-size: 12px; opacity: 0.5; margin-top: 6px; padding: 6px 10px; border-left: 2px solid rgba(255,255,255,0.1); font-style: italic; }
  .mp-context { font-size: 12px; line-height: 1.5; padding: 10px; background: rgba(255,255,255,0.03); border-radius: 8px; white-space: pre-wrap; max-height: 150px; overflow-y: auto; }
  .mp-file { font-size: 11px; opacity: 0.5; margin-top: 4px; }

  .mp-hash { font-size: 11px; margin-bottom: 12px; }
  .mp-hash code { background: rgba(99,102,241,0.15); padding: 2px 6px; border-radius: 4px; font-family: monospace; color: #6366f1; }

  .mp-meta-row { display: flex; gap: 16px; margin-bottom: 16px; }
  .mp-meta-item { flex: 1; }
  .mp-conf-bar { height: 4px; background: rgba(255,255,255,0.1); border-radius: 2px; margin: 4px 0; }
  .mp-conf-fill { height: 100%; background: #6366f1; border-radius: 2px; }

  .mp-dests { display: flex; gap: 6px; flex-wrap: wrap; }
  .mp-dest { font-size: 12px; padding: 3px 8px; background: rgba(255,255,255,0.05); border-radius: 8px; }
  .mp-dest-none { opacity: 0.4; }

  .mp-timestamps { font-size: 11px; opacity: 0.4; display: flex; gap: 16px; margin-bottom: 16px; }

  .mp-actions { display: flex; gap: 8px; }
  .mp-btn { padding: 8px 16px; border-radius: 8px; border: none; cursor: pointer; font-size: 13px; font-weight: 500; }
  .mp-approve { background: rgba(16,185,129,0.2); color: #10b981; }
  .mp-reject { background: rgba(239,68,68,0.15); color: #ef4444; }
  .mp-edit-btn { background: rgba(99,102,241,0.15); color: #6366f1; font-size: 12px; padding: 4px 12px; margin-top: 8px; }
  .mp-save { background: rgba(16,185,129,0.2); color: #10b981; }
  .mp-cancel { background: rgba(255,255,255,0.05); color: inherit; }

  .mp-edit { width: 100%; padding: 10px; border-radius: 8px; border: 1px solid rgba(124,124,255,0.3); background: rgba(255,255,255,0.03); color: inherit; font-size: 14px; resize: vertical; font-family: inherit; }
  .mp-edit-actions { display: flex; gap: 8px; margin-top: 8px; }

  /* Theme vars */
  :global([data-theme="light"]) .ma-item { background: rgba(0,0,0,0.03); }
  :global([data-theme="light"]) .ma-cell { }
  :global([data-theme="light"]) .mp-popup { background: #fff; }
  :global([data-theme="light"]) .mp-text, :global([data-theme="light"]) .mp-context { background: rgba(0,0,0,0.03); }
</style>
