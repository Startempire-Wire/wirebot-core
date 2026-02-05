<script>
  import { onMount } from 'svelte';

  let grid = $state(null);       // { days, sources, totals }
  let loading = $state(true);
  let sourceFilter = $state(''); // '' = all sources
  let dayItems = $state([]);     // memories for selected day
  let dayLoading = $state(false);
  let selectedDay = $state(null);// 'YYYY-MM-DD'
  let detail = $state(null);     // full memory item for detail popup
  let editing = $state(false);
  let editText = $state('');
  let saving = $state(false);

  function token() {
    return localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || '';
  }
  function hdrs() { return token() ? { 'Authorization': `Bearer ${token()}` } : {}; }

  onMount(() => loadGrid());

  async function loadGrid() {
    loading = true;
    try {
      const res = await fetch('/v1/memory/grid', { headers: hdrs() });
      if (res.ok) grid = await res.json();
    } catch (e) { console.error(e); }
    loading = false;
  }

  // Aggregate a day's sources into one cell
  function dayCell(day) {
    const sources = grid?.days?.[day];
    if (!sources) return { approved: 0, pending: 0, rejected: 0, total: 0 };
    let a = 0, p = 0, r = 0;
    for (const [src, counts] of Object.entries(sources)) {
      if (sourceFilter && src !== sourceFilter) continue;
      a += counts.approved || 0;
      p += counts.pending || 0;
      r += counts.rejected || 0;
    }
    return { approved: a, pending: p, rejected: r, total: a + p + r };
  }

  function dayCellColor(cell) {
    if (cell.total === 0) return 'rgba(255,255,255,0.04)';
    const types = [cell.approved > 0, cell.pending > 0, cell.rejected > 0].filter(Boolean).length;
    if (types > 1) return '#6366f1';
    if (cell.approved > 0) return '#10b981';
    if (cell.pending > 0) return '#f59e0b';
    if (cell.rejected > 0) return '#ef4444';
    return 'rgba(255,255,255,0.04)';
  }

  function dayCellOpacity(cell) {
    if (cell.total === 0) return 0.15;
    return Math.max(0.35, Math.min(1, cell.total / 200));
  }

  function sortedDays() {
    return Object.keys(grid?.days || {}).sort().reverse();
  }

  // Tap a day cell → load that day's memories
  async function openDay(day) {
    if (selectedDay === day) { selectedDay = null; return; } // toggle
    selectedDay = day;
    detail = null;
    editing = false;
    dayLoading = true;
    try {
      const src = sourceFilter ? `&source=${sourceFilter}` : '';
      const res = await fetch(`/v1/memory/queue?status=&date=${day}&limit=500${src}`, { headers: hdrs() });
      if (res.ok) {
        const data = await res.json();
        dayItems = data.items || [];
      }
    } catch (e) { console.error(e); }
    dayLoading = false;
  }

  // Tap a memory in the day list → load full detail
  async function openDetail(id) {
    editing = false;
    detail = { _loading: true };
    try {
      const res = await fetch(`/v1/memory/item/${id}`, { headers: hdrs() });
      if (res.ok) detail = await res.json();
      else detail = null;
    } catch (e) { console.error(e); detail = null; }
  }

  function closeDetail() { detail = null; editing = false; }

  function startEdit() {
    editText = detail.correction || detail.memory_text;
    editing = true;
  }

  async function saveEdit() {
    if (!editText.trim() || !detail) return;
    saving = true;
    try {
      const res = await fetch(`/v1/memory/item/${detail.id}`, {
        method: 'PATCH',
        headers: { ...hdrs(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ memory_text: editText.trim() }),
      });
      if (res.ok) {
        const data = await res.json();
        detail = { ...detail, correction: editText.trim(), correction_hash: data.hash };
        editing = false;
        dayItems = dayItems.map(i => i.id === detail.id ? { ...i, correction: editText.trim() } : i);
      }
    } catch (e) { console.error(e); }
    saving = false;
  }

  async function doAction(id, action) {
    try {
      const res = await fetch(`/v1/memory/queue/${id}/${action}`, { method: 'POST', headers: hdrs() });
      if (!res.ok) return;
      const newStatus = action === 'reject' ? 'rejected' : 'approved';
      dayItems = dayItems.map(i => i.id === id ? { ...i, status: newStatus } : i);
      if (detail?.id === id) detail = { ...detail, status: newStatus };
      loadGrid(); // refresh day cell colors
    } catch (e) { console.error(e); }
  }

  function sourceIcon(s) {
    return { conversation: '💬', gdrive: '📁', ai_chat: '🤖', obsidian: '📝',
             recovered: '♻️', dropbox: '📦', unknown: '❓' }[s] || '📄';
  }
  function destBadge(d) {
    return { mem0: '🟢', letta: '🔵', memory_md: '📄', git: '🔒' }[d] || '⚪';
  }
  function statusClass(s) {
    return { approved: 'st-ok', pending: 'st-wait', rejected: 'st-no' }[s] || '';
  }
  function statusIcon(s) {
    return { approved: '✓', pending: '⏳', rejected: '✗' }[s] || s;
  }
  function portal(node) {
    document.body.appendChild(node);
    return { destroy() { node.remove(); } };
  }
</script>

<div class="ma">
  <!-- Totals -->
  {#if grid?.totals}
    <div class="ma-bar">
      <span class="mb-n">{grid.totals.total}</span>
      <span class="mb-ok">{grid.totals.approved} ✓</span>
      <span class="mb-wait">{grid.totals.pending} ⏳</span>
      <span class="mb-no">{grid.totals.rejected} ✗</span>
      <div class="ma-legend">
        <span class="ml" style="background:#10b981"></span>
        <span class="ml" style="background:#f59e0b"></span>
        <span class="ml" style="background:#ef4444"></span>
        <span class="ml" style="background:#6366f1"></span>
      </div>
    </div>
  {/if}

  <!-- Source filter pills -->
  {#if grid?.sources?.length > 1}
    <div class="ma-src-pills">
      <button class="sp" class:active={sourceFilter === ''} onclick={() => { sourceFilter = ''; selectedDay = null; }}>All</button>
      {#each grid.sources as src}
        <button class="sp" class:active={sourceFilter === src} onclick={() => { sourceFilter = src; selectedDay = null; }}>
          {sourceIcon(src)}
        </button>
      {/each}
    </div>
  {/if}

  <!-- Day grid -->
  {#if loading}
    <div class="ma-msg">Loading...</div>
  {:else if !grid?.days || Object.keys(grid.days).length === 0}
    <div class="ma-msg">No memories yet</div>
  {:else}
    <div class="ma-grid">
      {#each sortedDays() as day}
        {@const cell = dayCell(day)}
        {#if cell.total > 0 || !sourceFilter}
          <button class="mg-day" class:mg-active={selectedDay === day}
            style="background:{dayCellColor(cell)}; opacity:{dayCellOpacity(cell)}"
            title="{day}: {cell.approved}✓ {cell.pending}⏳ {cell.rejected}✗"
            onclick={() => openDay(day)}>
            <span class="mg-n">{cell.total}</span>
          </button>
        {/if}
      {/each}
    </div>

    <!-- Selected day expanded -->
    {#if selectedDay}
      <div class="ma-day-detail">
        <div class="dd-hdr">
          <span class="dd-date">{selectedDay}</span>
          <span class="dd-counts">
            <span class="mb-ok">{dayCell(selectedDay).approved}✓</span>
            <span class="mb-wait">{dayCell(selectedDay).pending}⏳</span>
            <span class="mb-no">{dayCell(selectedDay).rejected}✗</span>
          </span>
          <button class="dd-close" onclick={() => selectedDay = null}>✕</button>
        </div>
        {#if dayLoading}
          <div class="ma-msg">Loading...</div>
        {:else if dayItems.length === 0}
          <div class="ma-msg">No items</div>
        {:else}
          <div class="dd-list">
            {#each dayItems as item}
              <button class="dd-item {statusClass(item.status)}" onclick={() => openDetail(item.id)}>
                <span class="di-icon">{sourceIcon(item.source_type)}</span>
                <span class="di-text">{item.correction || item.memory_text}</span>
                <span class="di-st">{statusIcon(item.status)}</span>
                {#if item.correction}<span class="di-edit">✎</span>{/if}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<!-- Memory detail popup -->
{#if detail}
  <div class="ma-overlay" use:portal>
    <div class="ma-bk" onclick={closeDetail} role="presentation"></div>
    <div class="ma-popup">
      {#if detail._loading}
        <div class="ma-msg" style="padding:40px">Loading...</div>
      {:else}
        <div class="mp-hdr">
          <span class="mp-src">{sourceIcon(detail.source_type)} {detail.source_type}</span>
          <span class="mp-st {statusClass(detail.status)}">{detail.status}</span>
          <button class="mp-x" onclick={closeDetail}>✕</button>
        </div>

        <div class="mp-sec">
          <div class="mp-lbl">Memory</div>
          {#if editing}
            <textarea class="mp-ta" bind:value={editText} rows="3"></textarea>
            <div class="mp-row">
              <button class="mp-btn mp-save" onclick={saveEdit} disabled={saving}>{saving ? '...' : 'Save'}</button>
              <button class="mp-btn mp-cancel" onclick={() => editing = false}>Cancel</button>
            </div>
          {:else}
            <div class="mp-txt">{detail.correction || detail.memory_text}</div>
            {#if detail.correction}
              <div class="mp-orig"><span class="mp-sm">Original:</span> {detail.memory_text}</div>
            {/if}
            <button class="mp-btn mp-edit" onclick={startEdit}>✎ Edit</button>
          {/if}
        </div>

        {#if detail.correction_hash}
          <div class="mp-hash"><span class="mp-sm">Hash:</span> <code>{detail.correction_hash}</code></div>
        {/if}

        {#if detail.source_context}
          <div class="mp-sec">
            <div class="mp-lbl">Inferred from</div>
            <div class="mp-ctx">{detail.source_context}</div>
            {#if detail.source_file}<div class="mp-file">📎 {detail.source_file}</div>{/if}
          </div>
        {/if}

        <div class="mp-sec">
          <span class="mp-sm">Confidence</span>
          <div class="mp-bar"><div class="mp-fill" style="width:{detail.confidence * 100}%"></div></div>
          <span class="mp-sm">{Math.round(detail.confidence * 100)}%</span>
        </div>

        <div class="mp-sec">
          <div class="mp-lbl">Stored in</div>
          <div class="mp-dests">
            {#if detail.destinations?.length > 0}
              {#each detail.destinations as d}<span class="mp-dest">{destBadge(d)} {d}</span>{/each}
            {:else}
              <span class="mp-dest mp-none">Not yet stored</span>
            {/if}
          </div>
        </div>

        <div class="mp-ts">
          <span>Created: {detail.created_at?.replace('T', ' ').slice(0, 19)}</span>
          {#if detail.reviewed_at}<span>Reviewed: {detail.reviewed_at?.replace('T', ' ').slice(0, 19)}</span>{/if}
        </div>

        {#if detail.status === 'pending'}
          <div class="mp-acts">
            <button class="mp-btn mp-approve" onclick={() => doAction(detail.id, 'approve')}>✓ Approve</button>
            <button class="mp-btn mp-reject" onclick={() => doAction(detail.id, 'reject')}>✗ Reject</button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
{/if}

<style>
  .ma { padding: 8px 0; }

  /* Totals bar */
  .ma-bar { display: flex; gap: 10px; font-size: 12px; margin-bottom: 8px; align-items: center; }
  .mb-n { opacity: 0.5; }
  .mb-ok { color: #10b981; }
  .mb-wait { color: #f59e0b; }
  .mb-no { color: #ef4444; }
  .ma-legend { display: flex; gap: 3px; margin-left: auto; }
  .ml { width: 8px; height: 8px; border-radius: 2px; }

  /* Source pills */
  .ma-src-pills { display: flex; gap: 4px; margin-bottom: 10px; flex-wrap: wrap; }
  .sp { height: 28px; min-width: 28px; padding: 0 8px; border-radius: 6px; border: 1px solid rgba(255,255,255,0.08); background: transparent; color: inherit; font-size: 13px; cursor: pointer; display: flex; align-items: center; justify-content: center; }
  .sp.active { background: var(--bg-hover); border-color: rgba(124,124,255,0.3); }

  /* Day grid */
  .ma-grid { display: flex; flex-wrap: wrap; gap: 4px; }
  .mg-day { width: 36px; height: 36px; border-radius: 5px; border: 2px solid transparent; cursor: pointer; display: flex; align-items: center; justify-content: center; transition: transform 0.1s, border-color 0.15s; flex-shrink: 0; padding: 0; }
  .mg-day:hover { transform: scale(1.15); z-index: 1; }
  .mg-day.mg-active { border-color: #fff; transform: scale(1.15); }
  .mg-n { font-size: 10px; font-weight: 700; color: #fff; text-shadow: 0 1px 3px rgba(0,0,0,0.6); }

  .ma-msg { text-align: center; padding: 30px; opacity: 0.4; font-size: 13px; }

  /* Day detail panel */
  .ma-day-detail { margin-top: 10px; background: var(--bg-hover, rgba(255,255,255,0.03)); border-radius: 10px; padding: 10px; }
  .dd-hdr { display: flex; align-items: center; gap: 10px; margin-bottom: 8px; }
  .dd-date { font-weight: 600; font-size: 14px; }
  .dd-counts { font-size: 12px; display: flex; gap: 6px; }
  .dd-close { margin-left: auto; background: none; border: none; color: inherit; opacity: 0.4; cursor: pointer; font-size: 16px; }
  .dd-list { display: flex; flex-direction: column; gap: 2px; max-height: 50vh; overflow-y: auto; }
  .dd-item { display: flex; gap: 8px; align-items: center; padding: 8px 10px; border-radius: 8px; background: rgba(255,255,255,0.02); border: 1px solid transparent; cursor: pointer; text-align: left; font-size: 13px; color: inherit; transition: border-color 0.1s; }
  .dd-item:hover { border-color: rgba(124,124,255,0.2); }
  .dd-item.st-ok { border-left: 3px solid #10b981; }
  .dd-item.st-wait { border-left: 3px solid #f59e0b; }
  .dd-item.st-no { border-left: 3px solid #ef4444; opacity: 0.5; }
  .di-icon { flex-shrink: 0; }
  .di-text { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .di-st { flex-shrink: 0; font-size: 11px; opacity: 0.5; }
  .di-edit { color: #6366f1; font-size: 11px; flex-shrink: 0; }

  /* Popup overlay */
  .ma-overlay { position: fixed; inset: 0; z-index: 9999; display: flex; align-items: flex-end; justify-content: center; }
  .ma-bk { position: absolute; inset: 0; background: rgba(0,0,0,0.6); }
  .ma-popup { position: relative; background: var(--bg-card, #1a1a2e); border-radius: 16px 16px 0 0; width: 100%; max-width: 500px; max-height: 85vh; overflow-y: auto; padding: 20px; }

  .mp-hdr { display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px; }
  .mp-src { font-size: 14px; font-weight: 600; }
  .mp-st { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
  .st-ok { color: #10b981; background: rgba(16,185,129,0.15); }
  .st-wait { color: #f59e0b; background: rgba(245,158,11,0.15); }
  .st-no { color: #ef4444; background: rgba(239,68,68,0.15); }
  .mp-x { background: none; border: none; font-size: 18px; cursor: pointer; color: inherit; opacity: 0.5; }

  .mp-sec { margin-bottom: 14px; }
  .mp-lbl { font-size: 10px; text-transform: uppercase; letter-spacing: 0.5px; opacity: 0.4; margin-bottom: 4px; }
  .mp-sm { font-size: 10px; opacity: 0.4; }
  .mp-txt { font-size: 14px; line-height: 1.5; padding: 10px; background: rgba(255,255,255,0.03); border-radius: 8px; }
  .mp-orig { font-size: 12px; opacity: 0.4; margin-top: 6px; padding: 6px 10px; border-left: 2px solid rgba(255,255,255,0.1); font-style: italic; }
  .mp-ctx { font-size: 12px; line-height: 1.5; padding: 10px; background: rgba(255,255,255,0.03); border-radius: 8px; white-space: pre-wrap; max-height: 150px; overflow-y: auto; }
  .mp-file { font-size: 11px; opacity: 0.4; margin-top: 4px; }
  .mp-hash { font-size: 11px; margin-bottom: 12px; }
  .mp-hash code { background: rgba(99,102,241,0.15); padding: 2px 6px; border-radius: 4px; font-family: monospace; color: #6366f1; }
  .mp-bar { height: 4px; background: rgba(255,255,255,0.1); border-radius: 2px; margin: 4px 0; }
  .mp-fill { height: 100%; background: #6366f1; border-radius: 2px; }
  .mp-dests { display: flex; gap: 6px; flex-wrap: wrap; }
  .mp-dest { font-size: 12px; padding: 3px 8px; background: rgba(255,255,255,0.05); border-radius: 8px; }
  .mp-none { opacity: 0.3; }
  .mp-ts { font-size: 11px; opacity: 0.3; display: flex; gap: 16px; margin-bottom: 14px; }
  .mp-acts { display: flex; gap: 8px; }
  .mp-row { display: flex; gap: 8px; margin-top: 8px; }
  .mp-btn { padding: 8px 16px; border-radius: 8px; border: none; cursor: pointer; font-size: 13px; font-weight: 500; }
  .mp-approve { background: rgba(16,185,129,0.2); color: #10b981; }
  .mp-reject { background: rgba(239,68,68,0.15); color: #ef4444; }
  .mp-edit { background: rgba(99,102,241,0.15); color: #6366f1; font-size: 12px; padding: 4px 12px; margin-top: 8px; }
  .mp-save { background: rgba(16,185,129,0.2); color: #10b981; }
  .mp-cancel { background: rgba(255,255,255,0.05); color: inherit; }
  .mp-ta { width: 100%; padding: 10px; border-radius: 8px; border: 1px solid rgba(124,124,255,0.3); background: rgba(255,255,255,0.03); color: inherit; font-size: 14px; resize: vertical; font-family: inherit; box-sizing: border-box; }

  :global([data-theme="light"]) .ma-popup { background: #fff; }
  :global([data-theme="light"]) .mp-txt, :global([data-theme="light"]) .mp-ctx { background: rgba(0,0,0,0.03); }
  :global([data-theme="light"]) .dd-item { background: rgba(0,0,0,0.02); }
  :global([data-theme="light"]) .ma-day-detail { background: rgba(0,0,0,0.03); }
</style>
