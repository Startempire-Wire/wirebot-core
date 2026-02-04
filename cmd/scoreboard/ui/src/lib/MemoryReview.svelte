<script>
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();
  
  let items = $state([]);
  let conflicts = $state([]);
  let counts = $state({ pending: 0, approved: 0, rejected: 0, total: 0 });
  let loading = $state(true);
  let currentIndex = $state(0);
  let correctionMode = $state(false);
  let correctionText = $state('');
  let showConflicts = $state(false);
  
  // Powerup state
  let memoryPower = $state(0);
  let powerLevel = $state(1);
  let powerFlash = $state(false);
  let streak = $state(0);
  
  // Swipe state
  let startX = 0;
  let startY = 0;
  let dragging = $state(false);
  let swipeOffset = $state(0);
  let swipeOpacity = $state(0);
  let swipeDirection = $state(null);
  let animatingOut = $state(false);
  
  const SWIPE_THRESHOLD = 100;
  const POWER_PER_ACTION = { approve: 15, correct: 25, reject: 5 };
  const POWER_PER_LEVEL = 100;
  
  const API = '';
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || '' : '';
  
  function authHeaders() {
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }
  
  function exit() {
    dispatch('exit');
  }
  
  async function loadQueue() {
    loading = true;
    try {
      const res = await fetch(`${API}/v1/memory/queue?status=pending&limit=100`, {
        headers: authHeaders()
      });
      const data = await res.json();
      items = data.items || [];
      counts = data.counts || { pending: 0, approved: 0, rejected: 0 };
      counts.total = counts.pending + counts.approved + counts.rejected;
      currentIndex = 0;
      
      const totalReviewed = counts.approved + counts.rejected;
      memoryPower = (totalReviewed * 10) % POWER_PER_LEVEL;
      powerLevel = Math.floor((totalReviewed * 10) / POWER_PER_LEVEL) + 1;
      
      await loadConflicts();
    } catch (e) {
      console.error('Failed to load memory queue:', e);
    }
    loading = false;
  }
  
  async function loadConflicts() {
    try {
      const res = await fetch(`${API}/v1/memory/conflicts`, { headers: authHeaders() });
      if (res.ok) {
        const data = await res.json();
        conflicts = data.conflicts || [];
      }
    } catch (e) {
      conflicts = [];
    }
  }
  
  function addPower(action) {
    const gain = POWER_PER_ACTION[action] || 10;
    memoryPower += gain;
    streak++;
    
    if (memoryPower >= POWER_PER_LEVEL) {
      memoryPower = memoryPower - POWER_PER_LEVEL;
      powerLevel++;
      powerFlash = true;
      if (navigator.vibrate) navigator.vibrate([100, 50, 100]);
      setTimeout(() => powerFlash = false, 1000);
    } else {
      if (navigator.vibrate) navigator.vibrate(30);
    }
  }
  
  async function takeAction(action, correction = null) {
    if (!currentItem) return;
    
    animatingOut = true;
    swipeOffset = action === 'approve' || action === 'correct' ? 400 : -400;
    
    const body = correction ? { correction } : {};
    try {
      const res = await fetch(`${API}/v1/memory/queue/${currentItem.id}/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify(body)
      });
      
      if (res.ok) {
        addPower(action);
        
        setTimeout(() => {
          items = items.filter((_, i) => i !== currentIndex);
          counts.pending = Math.max(0, counts.pending - 1);
          if (action === 'approve' || action === 'correct') counts.approved++;
          if (action === 'reject') counts.rejected++;
          
          if (currentIndex >= items.length) currentIndex = Math.max(0, items.length - 1);
          correctionMode = false;
          correctionText = '';
          swipeOffset = 0;
          swipeOpacity = 0;
          swipeDirection = null;
          animatingOut = false;
        }, 250);
      }
    } catch (e) {
      console.error('Action failed:', e);
      animatingOut = false;
      swipeOffset = 0;
    }
  }
  
  function handleTouchStart(e) {
    if (correctionMode || animatingOut) return;
    startX = e.touches[0].clientX;
    startY = e.touches[0].clientY;
    dragging = true;
  }
  
  function handleTouchMove(e) {
    if (!dragging || correctionMode || animatingOut) return;
    const deltaY = Math.abs(e.touches[0].clientY - startY);
    const deltaX = e.touches[0].clientX - startX;
    
    if (Math.abs(deltaX) > deltaY) {
      e.preventDefault();
      swipeOffset = deltaX;
      swipeOpacity = Math.min(Math.abs(deltaX) / SWIPE_THRESHOLD, 1);
      swipeDirection = deltaX > 40 ? 'right' : deltaX < -40 ? 'left' : null;
    }
  }
  
  function handleTouchEnd() {
    if (!dragging || animatingOut) return;
    dragging = false;
    
    if (swipeOffset > SWIPE_THRESHOLD) takeAction('approve');
    else if (swipeOffset < -SWIPE_THRESHOLD) takeAction('reject');
    else { swipeOffset = 0; swipeOpacity = 0; swipeDirection = null; }
  }
  
  function handleMouseDown(e) {
    if (correctionMode || animatingOut) return;
    if (e.target.closest('.memory-card')) {
      startX = e.clientX;
      dragging = true;
      e.preventDefault();
    }
  }
  
  function handleMouseMove(e) {
    if (!dragging || correctionMode || animatingOut) return;
    swipeOffset = e.clientX - startX;
    swipeOpacity = Math.min(Math.abs(swipeOffset) / SWIPE_THRESHOLD, 1);
    swipeDirection = swipeOffset > 40 ? 'right' : swipeOffset < -40 ? 'left' : null;
  }
  
  function handleMouseUp() { handleTouchEnd(); }
  
  function getSourceInfo(item) {
    const type = item.source_type;
    const file = item.source_file;
    
    if (type === 'recovered') {
      return {
        icon: '⚠️',
        label: 'Migrated Memory',
        sublabel: 'Source unknown - imported from previous system',
        hasContext: false
      };
    }
    if (type === 'pairing') {
      return {
        icon: '🎯',
        label: 'Pairing Answer',
        sublabel: file || 'From onboarding questionnaire',
        hasContext: true
      };
    }
    if (type === 'obsidian') {
      return {
        icon: '📓',
        label: 'Obsidian Note',
        sublabel: file || 'From your vault',
        hasContext: true
      };
    }
    if (type === 'conversation') {
      return {
        icon: '💬',
        label: 'Conversation',
        sublabel: file || 'Extracted from chat',
        hasContext: true
      };
    }
    return {
      icon: '📄',
      label: type || 'Unknown',
      sublabel: file || 'No source information',
      hasContext: !!item.source_context
    };
  }
  
  function getPowerLabel(level) {
    const labels = ['Novice', 'Aware', 'Tuned', 'Sharp', 'Synced', 'Bonded', 'Master'];
    return labels[Math.min(level - 1, labels.length - 1)];
  }
  
  $effect(() => { loadQueue(); });
  
  let currentItem = $derived(items[currentIndex]);
  let nextItems = $derived(items.slice(currentIndex + 1, currentIndex + 3));
  let powerPercent = $derived((memoryPower / POWER_PER_LEVEL) * 100);
</script>

<svelte:window on:mouseup={handleMouseUp} on:mousemove={handleMouseMove} />

<div class="memory-review">
  <header class="mr-header" onclick={exit}>
    <button class="back-btn">←</button>
    <div class="mr-title">🧠 Memory Review</div>
    <div class="tap-hint">tap to exit</div>
  </header>
  
  <!-- Power Bar -->
  <div class="power-section" class:flash={powerFlash}>
    <div class="power-header">
      <span class="power-label">Memory Sync</span>
      <span class="power-level">Lv.{powerLevel} {getPowerLabel(powerLevel)}</span>
    </div>
    <div class="power-bar">
      <div class="power-fill" style="width: {powerPercent}%"></div>
    </div>
    <div class="power-stats">
      <span>✓ {counts.approved}</span>
      <span>✗ {counts.rejected}</span>
      {#if streak > 2}<span class="streak">🔥 {streak}</span>{/if}
    </div>
  </div>
  
  <!-- Conflicts -->
  {#if conflicts.length > 0}
    <button class="conflicts-alert" onclick={() => showConflicts = !showConflicts}>
      <span>⚠️</span>
      <span class="alert-text">{conflicts.length} conflict{conflicts.length > 1 ? 's' : ''}</span>
      <span>{showConflicts ? '▼' : '▶'}</span>
    </button>
  {/if}
  
  {#if loading}
    <div class="mr-loading"><div class="spinner"></div>Loading...</div>
  {:else if items.length === 0}
    <div class="mr-empty" onclick={exit}>
      <div class="empty-check">✓</div>
      <h3>All caught up!</h3>
      <p>Level {powerLevel} • {getPowerLabel(powerLevel)}</p>
      <p class="tap-exit">Tap to exit</p>
    </div>
  {:else if currentItem}
    <div class="card-stack">
      {#each nextItems as _, i}
        <div class="card-behind" style="transform: scale({0.95 - i * 0.03}) translateY({(i + 1) * 12}px); opacity: {0.4 - i * 0.15};"></div>
      {/each}
      
      <div 
        class="memory-card"
        class:dragging
        class:animating={animatingOut}
        style="transform: translateX({swipeOffset}px) rotate({swipeOffset * 0.04}deg);"
        ontouchstart={handleTouchStart}
        ontouchmove={handleTouchMove}
        ontouchend={handleTouchEnd}
        onmousedown={handleMouseDown}
      >
        <!-- Swipe stamps -->
        <div class="swipe-stamp approve" style="opacity: {swipeDirection === 'right' ? swipeOpacity : 0}">KEEP</div>
        <div class="swipe-stamp reject" style="opacity: {swipeDirection === 'left' ? swipeOpacity : 0}">DROP</div>
        
        {#if !correctionMode}
          {@const sourceInfo = getSourceInfo(currentItem)}
          
          <div class="card-inner">
            <!-- Source Header -->
            <div class="card-source">
              <span class="source-icon">{sourceInfo.icon}</span>
              <div class="source-text">
                <div class="source-label">{sourceInfo.label}</div>
                <div class="source-sub">{sourceInfo.sublabel}</div>
              </div>
            </div>
            
            <!-- The Memory -->
            <div class="card-memory">
              "{currentItem.memory_text}"
            </div>
            
            <!-- Context Section -->
            <div class="card-context-section">
              {#if currentItem.source_context}
                <div class="context-header">📎 Source Context</div>
                <blockquote class="context-quote">
                  {currentItem.source_context}
                </blockquote>
              {:else if !sourceInfo.hasContext}
                <div class="no-context">
                  <div class="no-context-icon">❓</div>
                  <div class="no-context-text">
                    <strong>No source context available</strong>
                    <p>This memory was imported without its original context. You'll need to verify it based on your own knowledge.</p>
                  </div>
                </div>
              {:else}
                <div class="no-context">
                  <div class="no-context-icon">📝</div>
                  <div class="no-context-text">
                    <strong>Context not captured</strong>
                    <p>The source exists but context wasn't recorded during extraction.</p>
                  </div>
                </div>
              {/if}
            </div>
            
            <!-- Confidence -->
            <div class="card-confidence">
              <div class="conf-row">
                <span class="conf-label">Confidence</span>
                <span class="conf-value" class:high={currentItem.confidence >= 0.8} class:med={currentItem.confidence >= 0.5 && currentItem.confidence < 0.8} class:low={currentItem.confidence < 0.5}>
                  {currentItem.confidence >= 0.8 ? 'High' : currentItem.confidence >= 0.5 ? 'Medium' : 'Low'}
                </span>
              </div>
              <div class="conf-bar">
                <div class="conf-fill" class:high={currentItem.confidence >= 0.8} class:med={currentItem.confidence >= 0.5 && currentItem.confidence < 0.8} class:low={currentItem.confidence < 0.5} style="width: {currentItem.confidence * 100}%"></div>
              </div>
            </div>
          </div>
          
          <!-- Actions -->
          <div class="card-actions">
            <button class="btn-action reject" onclick={() => takeAction('reject')}>
              <span class="btn-icon">✗</span>
              <span class="btn-text">Drop</span>
            </button>
            <button class="btn-action edit" onclick={() => { correctionMode = true; correctionText = currentItem.memory_text; }}>
              <span class="btn-icon">✎</span>
              <span class="btn-text">Edit</span>
            </button>
            <button class="btn-action approve" onclick={() => takeAction('approve')}>
              <span class="btn-icon">✓</span>
              <span class="btn-text">Keep</span>
            </button>
          </div>
        {:else}
          <!-- Edit Mode -->
          <div class="edit-mode">
            <div class="edit-header">✎ Edit Memory</div>
            <div class="edit-original">
              <span class="edit-label">Original:</span>
              "{currentItem.memory_text}"
            </div>
            <textarea bind:value={correctionText} placeholder="Write the correct version..." rows="4"></textarea>
            <div class="edit-actions">
              <button class="btn-cancel" onclick={() => { correctionMode = false; correctionText = ''; }}>Cancel</button>
              <button class="btn-save" onclick={() => takeAction('correct', correctionText)}>Save</button>
            </div>
          </div>
        {/if}
      </div>
    </div>
    
    <div class="progress-row">
      <span>{currentIndex + 1} of {items.length}</span>
    </div>
  {/if}
</div>

<style>
  .memory-review {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 1rem;
    background: var(--page-gradient);
  }
  
  .mr-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.5rem;
    margin: -0.5rem -0.5rem 1rem -0.5rem;
    border-radius: 12px;
    cursor: pointer;
  }
  .mr-header:active { background: var(--bg-card); }
  
  .back-btn {
    width: 36px; height: 36px;
    border-radius: 50%;
    border: none;
    background: var(--bg-card);
    color: var(--text-secondary);
    font-size: 1.2rem;
    cursor: pointer;
  }
  
  .mr-title { flex: 1; font-size: 1.1rem; font-weight: 600; }
  .tap-hint { font-size: 0.65rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
  
  /* Power Bar */
  .power-section {
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 0.75rem 1rem;
    margin-bottom: 1rem;
  }
  .power-section.flash { border-color: #a855f7; box-shadow: 0 0 20px #a855f720; }
  
  .power-header { display: flex; justify-content: space-between; margin-bottom: 0.4rem; }
  .power-label { font-size: 0.7rem; color: var(--text-secondary); text-transform: uppercase; letter-spacing: 0.05em; }
  .power-level { font-size: 0.8rem; font-weight: 700; color: #a855f7; }
  
  .power-bar { height: 6px; background: var(--bg); border-radius: 3px; overflow: hidden; margin-bottom: 0.4rem; }
  .power-fill { height: 100%; background: linear-gradient(90deg, #7c3aed, #a855f7); border-radius: 3px; transition: width 0.3s; }
  
  .power-stats { display: flex; gap: 1rem; font-size: 0.75rem; color: var(--text-muted); }
  .streak { color: #f59e0b; }
  
  /* Conflicts */
  .conflicts-alert {
    display: flex; align-items: center; gap: 0.5rem;
    width: 100%; padding: 0.6rem 1rem;
    background: #1a1010; border: 1px solid #7f1d1d; border-radius: 10px;
    color: #fca5a5; cursor: pointer; margin-bottom: 1rem;
  }
  .alert-text { flex: 1; text-align: left; font-size: 0.85rem; }
  
  /* Loading/Empty */
  .mr-loading { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 1rem; color: var(--text-secondary); }
  .spinner { width: 36px; height: 36px; border: 3px solid var(--border); border-top-color: #a855f7; border-radius: 50%; animation: spin 1s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  
  .mr-empty { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; cursor: pointer; }
  .empty-check { width: 70px; height: 70px; background: linear-gradient(135deg, #10b981, #059669); border-radius: 50%; display: flex; align-items: center; justify-content: center; font-size: 2rem; color: white; margin-bottom: 1rem; }
  .mr-empty h3 { margin: 0 0 0.25rem 0; }
  .mr-empty p { color: var(--text-secondary); margin: 0; }
  .tap-exit { margin-top: 1.5rem; font-size: 0.75rem; color: var(--text-muted); }
  
  /* Card Stack */
  .card-stack { flex: 1; display: flex; align-items: center; justify-content: center; position: relative; padding: 0.5rem 0; min-height: 520px; }
  
  .card-behind {
    position: absolute;
    width: 100%; max-width: 360px;
    height: 480px;
    background: var(--card-bg-behind);
    border-radius: 24px;
    border: 1px solid var(--border);
  }
  
  .memory-card {
    position: relative;
    width: 100%; max-width: 360px;
    min-height: 480px;
    background: var(--card-gradient);
    border-radius: 24px;
    border: 2px solid var(--card-border);
    overflow: hidden;
    touch-action: pan-y;
    cursor: grab;
    transition: transform 0.12s ease-out;
    user-select: none;
    display: flex;
    flex-direction: column;
  }
  .memory-card.dragging { cursor: grabbing; transition: none; }
  .memory-card.animating { transition: transform 0.25s ease-out; }
  
  /* Swipe Stamps */
  .swipe-stamp {
    position: absolute;
    top: 30px;
    padding: 12px 28px;
    border-radius: 8px;
    font-size: 1.3rem;
    font-weight: 900;
    letter-spacing: 0.15em;
    z-index: 10;
    pointer-events: none;
    border: 4px solid white;
  }
  .swipe-stamp.approve { right: 20px; background: #10b981; color: white; transform: rotate(15deg); }
  .swipe-stamp.reject { left: 20px; background: #ef4444; color: white; transform: rotate(-15deg); }
  
  /* Card Inner */
  .card-inner { flex: 1; padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; }
  
  .card-source { display: flex; align-items: flex-start; gap: 0.75rem; }
  .source-icon { font-size: 1.5rem; }
  .source-text { flex: 1; }
  .source-label { font-size: 0.9rem; font-weight: 600; color: #a855f7; }
  .source-sub { font-size: 0.75rem; color: var(--text-muted); margin-top: 0.1rem; }
  
  .card-memory {
    font-size: 1.2rem;
    line-height: 1.6;
    color: var(--text);
    font-style: italic;
    padding: 1rem 0;
    border-top: 1px solid var(--card-border);
    border-bottom: 1px solid var(--card-border);
  }
  
  /* Context Section */
  .card-context-section { flex: 1; }
  
  .context-header { font-size: 0.75rem; color: var(--text-secondary); margin-bottom: 0.5rem; text-transform: uppercase; letter-spacing: 0.05em; }
  
  .context-quote {
    margin: 0;
    padding: 1rem;
    background: var(--card-context-bg);
    border-left: 3px solid #7c3aed;
    border-radius: 0 8px 8px 0;
    font-size: 0.9rem;
    line-height: 1.6;
    color: var(--text-secondary);
    font-style: italic;
  }
  
  .no-context {
    display: flex;
    gap: 0.75rem;
    padding: 1rem;
    background: var(--card-context-bg);
    border-radius: 10px;
    border: 1px dashed var(--border-light);
  }
  .no-context-icon { font-size: 1.5rem; }
  .no-context-text strong { color: var(--text-secondary); font-size: 0.85rem; display: block; margin-bottom: 0.25rem; }
  .no-context-text p { color: var(--text-muted); font-size: 0.8rem; margin: 0; line-height: 1.4; }
  
  /* Confidence */
  .card-confidence { margin-top: auto; }
  .conf-row { display: flex; justify-content: space-between; margin-bottom: 0.3rem; }
  .conf-label { font-size: 0.7rem; color: var(--text-muted); text-transform: uppercase; }
  .conf-value { font-size: 0.75rem; font-weight: 600; }
  .conf-value.high { color: #10b981; }
  .conf-value.med { color: #f59e0b; }
  .conf-value.low { color: #ef4444; }
  
  .conf-bar { height: 4px; background: var(--bg-card); border-radius: 2px; overflow: hidden; }
  .conf-fill { height: 100%; border-radius: 2px; transition: width 0.3s; }
  .conf-fill.high { background: #10b981; }
  .conf-fill.med { background: #f59e0b; }
  .conf-fill.low { background: #ef4444; }
  
  /* Actions */
  .card-actions { display: flex; gap: 0.75rem; padding: 1rem 1.5rem 1.5rem; background: var(--card-actions-gradient); }
  
  .btn-action {
    flex: 1;
    display: flex; flex-direction: column; align-items: center; gap: 0.25rem;
    padding: 0.9rem 0.5rem;
    border-radius: 14px;
    border: none;
    cursor: pointer;
    transition: transform 0.15s;
  }
  .btn-action:active { transform: scale(0.95); }
  
  .btn-action.reject { background: linear-gradient(135deg, #dc2626, #b91c1c); color: white; }
  .btn-action.edit { background: linear-gradient(135deg, #d97706, #b45309); color: white; }
  .btn-action.approve { background: linear-gradient(135deg, #16a34a, #15803d); color: white; }
  
  .btn-icon { font-size: 1.4rem; font-weight: 700; }
  .btn-text { font-size: 0.8rem; font-weight: 600; text-transform: uppercase; }
  
  /* Edit Mode */
  .edit-mode { padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; height: 100%; }
  .edit-header { font-size: 1.1rem; font-weight: 600; color: #f59e0b; }
  .edit-original { background: var(--card-context-bg); padding: 1rem; border-radius: 10px; font-size: 0.9rem; color: var(--text-secondary); }
  .edit-label { display: block; font-size: 0.7rem; color: var(--text-muted); text-transform: uppercase; margin-bottom: 0.3rem; }
  
  .edit-mode textarea {
    flex: 1;
    padding: 1rem;
    border-radius: 12px;
    border: 2px solid var(--card-border);
    background: var(--card-context-bg);
    color: var(--text);
    font-family: inherit;
    font-size: 1rem;
    line-height: 1.5;
    resize: none;
  }
  .edit-mode textarea:focus { outline: none; border-color: #a855f7; }
  
  .edit-actions { display: flex; gap: 0.75rem; }
  .edit-actions button { flex: 1; padding: 0.9rem; border-radius: 12px; border: none; font-size: 0.9rem; font-weight: 600; cursor: pointer; }
  .btn-cancel { background: var(--bg-elevated); color: var(--text-secondary); }
  .btn-save { background: linear-gradient(135deg, #a855f7, #7c3aed); color: white; }
  
  .progress-row { text-align: center; padding: 0.75rem 0; font-size: 0.8rem; color: var(--text-muted); }
</style>
