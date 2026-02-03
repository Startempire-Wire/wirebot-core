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
  let currentX = 0;
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
    // Also try navigation
    if (typeof window !== 'undefined') {
      window.history.back();
    }
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
      
      // Calculate initial power from approved count
      const totalReviewed = counts.approved + counts.rejected;
      memoryPower = (totalReviewed * 10) % POWER_PER_LEVEL;
      powerLevel = Math.floor((totalReviewed * 10) / POWER_PER_LEVEL) + 1;
      
      // Load conflicts
      await loadConflicts();
    } catch (e) {
      console.error('Failed to load memory queue:', e);
    }
    loading = false;
  }
  
  async function loadConflicts() {
    try {
      const res = await fetch(`${API}/v1/memory/conflicts`, {
        headers: authHeaders()
      });
      if (res.ok) {
        const data = await res.json();
        conflicts = data.conflicts || [];
      }
    } catch (e) {
      // Conflicts endpoint may not exist yet
      conflicts = [];
    }
  }
  
  function addPower(action) {
    const gain = POWER_PER_ACTION[action] || 10;
    memoryPower += gain;
    streak++;
    
    // Level up check
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
  
  async function resolveConflict(conflictId, keepId) {
    try {
      await fetch(`${API}/v1/memory/conflicts/${conflictId}/resolve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({ keep: keepId })
      });
      conflicts = conflicts.filter(c => c.id !== conflictId);
      addPower('correct');
    } catch (e) {
      console.error('Failed to resolve conflict:', e);
    }
  }
  
  function handleTouchStart(e) {
    if (correctionMode || animatingOut) return;
    startX = e.touches[0].clientX;
    startY = e.touches[0].clientY;
    currentX = startX;
    dragging = true;
  }
  
  function handleTouchMove(e) {
    if (!dragging || correctionMode || animatingOut) return;
    currentX = e.touches[0].clientX;
    const deltaY = Math.abs(e.touches[0].clientY - startY);
    const deltaX = currentX - startX;
    
    if (Math.abs(deltaX) > deltaY) {
      e.preventDefault();
      swipeOffset = deltaX;
      swipeOpacity = Math.min(Math.abs(deltaX) / SWIPE_THRESHOLD, 1);
      
      if (deltaX > 40) swipeDirection = 'right';
      else if (deltaX < -40) swipeDirection = 'left';
      else swipeDirection = null;
    }
  }
  
  function handleTouchEnd() {
    if (!dragging || animatingOut) return;
    dragging = false;
    
    if (swipeOffset > SWIPE_THRESHOLD) {
      takeAction('approve');
    } else if (swipeOffset < -SWIPE_THRESHOLD) {
      takeAction('reject');
    } else {
      swipeOffset = 0;
      swipeOpacity = 0;
      swipeDirection = null;
    }
  }
  
  function handleMouseDown(e) {
    if (correctionMode || animatingOut) return;
    // Only start drag on the card itself
    if (e.target.closest('.memory-card')) {
      startX = e.clientX;
      currentX = startX;
      dragging = true;
      e.preventDefault();
    }
  }
  
  function handleMouseMove(e) {
    if (!dragging || correctionMode || animatingOut) return;
    currentX = e.clientX;
    swipeOffset = currentX - startX;
    swipeOpacity = Math.min(Math.abs(swipeOffset) / SWIPE_THRESHOLD, 1);
    
    if (swipeOffset > 40) swipeDirection = 'right';
    else if (swipeOffset < -40) swipeDirection = 'left';
    else swipeDirection = null;
  }
  
  function handleMouseUp() {
    handleTouchEnd();
  }
  
  function getConfidenceBadge(conf) {
    if (conf >= 0.8) return { label: 'High', color: '#10b981' };
    if (conf >= 0.5) return { label: 'Medium', color: '#f59e0b' };
    return { label: 'Low', color: '#ef4444' };
  }
  
  function getSourceLabel(type) {
    switch (type) {
      case 'obsidian': return { icon: '📓', label: 'Obsidian Vault' };
      case 'conversation': return { icon: '💬', label: 'Conversation' };
      case 'bootstrap': return { icon: '🚀', label: 'Initial Setup' };
      case 'recovered': return { icon: '🔄', label: 'Recovered Memory' };
      case 'inference': return { icon: '🤖', label: 'AI Inference' };
      default: return { icon: '📄', label: type || 'Unknown' };
    }
  }
  
  function getPowerLabel(level) {
    const labels = ['Novice', 'Aware', 'Tuned', 'Sharp', 'Synced', 'Bonded', 'Master'];
    return labels[Math.min(level - 1, labels.length - 1)];
  }
  
  $effect(() => {
    loadQueue();
  });
  
  let currentItem = $derived(items[currentIndex]);
  let nextItems = $derived(items.slice(currentIndex + 1, currentIndex + 3));
  let powerPercent = $derived((memoryPower / POWER_PER_LEVEL) * 100);
</script>

<svelte:window on:mouseup={handleMouseUp} on:mousemove={handleMouseMove} />

<div class="memory-review">
  <!-- Tap to exit header -->
  <header class="mr-header" onclick={exit}>
    <button class="back-btn" onclick={exit}>
      <span>←</span>
    </button>
    <div class="mr-title">
      <span class="mr-icon">🧠</span>
      <span>Memory Review</span>
    </div>
    <div class="tap-hint">tap to exit</div>
  </header>
  
  <!-- Memory Power Bar -->
  <div class="power-section" class:flash={powerFlash}>
    <div class="power-header">
      <span class="power-label">Memory Sync</span>
      <span class="power-level">Lv.{powerLevel} {getPowerLabel(powerLevel)}</span>
    </div>
    <div class="power-bar">
      <div class="power-fill" style="width: {powerPercent}%">
        <div class="power-glow"></div>
      </div>
    </div>
    <div class="power-stats">
      <span>✓ {counts.approved}</span>
      <span>✗ {counts.rejected}</span>
      {#if streak > 2}
        <span class="streak">🔥 {streak} streak</span>
      {/if}
    </div>
  </div>
  
  <!-- Conflicts Alert -->
  {#if conflicts.length > 0}
    <button class="conflicts-alert" onclick={() => showConflicts = !showConflicts}>
      <span class="alert-icon">⚠️</span>
      <span class="alert-text">{conflicts.length} memory conflict{conflicts.length > 1 ? 's' : ''} detected</span>
      <span class="alert-arrow">{showConflicts ? '▼' : '▶'}</span>
    </button>
    
    {#if showConflicts}
      <div class="conflicts-list">
        {#each conflicts as conflict}
          <div class="conflict-card">
            <div class="conflict-header">⚡ Conflicting Memories</div>
            <div class="conflict-items">
              <button class="conflict-option" onclick={() => resolveConflict(conflict.id, conflict.a.id)}>
                <span class="option-text">"{conflict.a.text}"</span>
                <span class="option-meta">{conflict.a.source}</span>
              </button>
              <div class="conflict-vs">VS</div>
              <button class="conflict-option" onclick={() => resolveConflict(conflict.id, conflict.b.id)}>
                <span class="option-text">"{conflict.b.text}"</span>
                <span class="option-meta">{conflict.b.source}</span>
              </button>
            </div>
            <div class="conflict-hint">Tap the correct one to keep</div>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
  
  {#if loading}
    <div class="mr-loading">
      <div class="spinner"></div>
      <span>Loading memories...</span>
    </div>
  {:else if items.length === 0}
    <div class="mr-empty" onclick={exit}>
      <div class="empty-check">✓</div>
      <h3>All caught up!</h3>
      <p>No memories need review right now.</p>
      <div class="power-summary">
        <div class="summary-level">Level {powerLevel} • {getPowerLabel(powerLevel)}</div>
        <div class="summary-stats">{counts.approved + counts.rejected} memories reviewed</div>
      </div>
      <div class="tap-exit">Tap anywhere to exit</div>
    </div>
  {:else if currentItem}
    <div class="card-stack">
      {#each nextItems as _, i}
        <div 
          class="card-behind"
          style="transform: scale({0.95 - i * 0.05}) translateY({(i + 1) * 8}px); opacity: {0.5 - i * 0.2};"
        ></div>
      {/each}
      
      <div 
        class="memory-card"
        class:dragging
        class:animating={animatingOut}
        style="transform: translateX({swipeOffset}px) rotate({swipeOffset * 0.05}deg);"
        ontouchstart={handleTouchStart}
        ontouchmove={handleTouchMove}
        ontouchend={handleTouchEnd}
        onmousedown={handleMouseDown}
      >
        <div class="swipe-overlay approve" style="opacity: {swipeDirection === 'right' ? swipeOpacity : 0}">
          <span>✓ KEEP</span>
        </div>
        <div class="swipe-overlay reject" style="opacity: {swipeDirection === 'left' ? swipeOpacity : 0}">
          <span>✗ DROP</span>
        </div>
        
        {#if !correctionMode}
          <div class="card-content">
            <div class="card-source">
              <span class="source-icon">{getSourceLabel(currentItem.source_type).icon}</span>
              <span class="source-label">{getSourceLabel(currentItem.source_type).label}</span>
            </div>
            
            <div class="card-memory">
              {currentItem.memory_text}
            </div>
            
            {#if currentItem.source_context}
              <div class="card-context">
                <div class="context-header">💡 Context</div>
                <div class="context-body">{currentItem.source_context}</div>
              </div>
            {/if}
            
            <div class="card-confidence">
              <div class="conf-bar">
                <div 
                  class="conf-fill" 
                  style="width: {currentItem.confidence * 100}%; background: {getConfidenceBadge(currentItem.confidence).color}"
                ></div>
              </div>
              <span class="conf-label" style="color: {getConfidenceBadge(currentItem.confidence).color}">
                {getConfidenceBadge(currentItem.confidence).label} confidence
              </span>
            </div>
          </div>
          
          <div class="card-actions">
            <button class="btn-action reject" onclick={() => takeAction('reject')}>
              <span class="btn-icon">✗</span>
              <span class="btn-label">Drop</span>
              <span class="btn-power">+5</span>
            </button>
            <button class="btn-action edit" onclick={() => { correctionMode = true; correctionText = currentItem.memory_text; }}>
              <span class="btn-icon">✎</span>
              <span class="btn-label">Edit</span>
              <span class="btn-power">+25</span>
            </button>
            <button class="btn-action approve" onclick={() => takeAction('approve')}>
              <span class="btn-icon">✓</span>
              <span class="btn-label">Keep</span>
              <span class="btn-power">+15</span>
            </button>
          </div>
        {:else}
          <div class="correction-mode">
            <div class="corr-header">
              <span>✎</span> Edit Memory
            </div>
            <div class="corr-original">
              <span class="corr-label">Original:</span>
              <span class="corr-text">{currentItem.memory_text}</span>
            </div>
            <textarea 
              bind:value={correctionText}
              placeholder="Write the corrected memory..."
              rows="4"
            ></textarea>
            <div class="corr-actions">
              <button class="btn-cancel" onclick={() => { correctionMode = false; correctionText = ''; }}>
                Cancel
              </button>
              <button class="btn-save" onclick={() => takeAction('correct', correctionText)}>
                Save +25 ⚡
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
    
    <div class="progress-section">
      <div class="progress-text">{currentIndex + 1} of {items.length} remaining</div>
    </div>
  {/if}
</div>

<style>
  .memory-review {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    padding: 1rem;
    background: linear-gradient(180deg, #0a0a12 0%, #12121a 100%);
  }
  
  .mr-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
    cursor: pointer;
    padding: 0.5rem;
    margin: -0.5rem -0.5rem 1rem -0.5rem;
    border-radius: 12px;
    transition: background 0.2s;
  }
  
  .mr-header:hover {
    background: #1a1a24;
  }
  
  .back-btn {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    border: none;
    background: #1a1a24;
    color: #888;
    font-size: 1.2rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .mr-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.1rem;
    font-weight: 600;
    flex: 1;
  }
  
  .tap-hint {
    font-size: 0.7rem;
    color: #444;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  /* Power Bar */
  .power-section {
    background: linear-gradient(135deg, #1a1a2e 0%, #16162a 100%);
    border: 1px solid #2a2a4a;
    border-radius: 16px;
    padding: 1rem;
    margin-bottom: 1rem;
    transition: all 0.3s;
  }
  
  .power-section.flash {
    border-color: #a855f7;
    box-shadow: 0 0 30px #a855f720;
    animation: levelUp 0.5s ease-out;
  }
  
  @keyframes levelUp {
    0%, 100% { transform: scale(1); }
    50% { transform: scale(1.02); }
  }
  
  .power-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
  }
  
  .power-label {
    font-size: 0.75rem;
    color: #888;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  
  .power-level {
    font-size: 0.85rem;
    font-weight: 700;
    color: #a855f7;
  }
  
  .power-bar {
    height: 8px;
    background: #0f0f1a;
    border-radius: 4px;
    overflow: hidden;
    margin-bottom: 0.5rem;
  }
  
  .power-fill {
    height: 100%;
    background: linear-gradient(90deg, #7c3aed, #a855f7, #c084fc);
    border-radius: 4px;
    transition: width 0.3s ease-out;
    position: relative;
  }
  
  .power-glow {
    position: absolute;
    right: 0;
    top: 0;
    bottom: 0;
    width: 20px;
    background: linear-gradient(90deg, transparent, #fff4);
    animation: glow 1.5s ease-in-out infinite;
  }
  
  @keyframes glow {
    0%, 100% { opacity: 0.3; }
    50% { opacity: 0.8; }
  }
  
  .power-stats {
    display: flex;
    gap: 1rem;
    font-size: 0.75rem;
    color: #666;
  }
  
  .streak {
    color: #f59e0b;
    font-weight: 600;
  }
  
  /* Conflicts */
  .conflicts-alert {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.75rem 1rem;
    background: linear-gradient(135deg, #7f1d1d20, #991b1b10);
    border: 1px solid #7f1d1d;
    border-radius: 12px;
    margin-bottom: 1rem;
    cursor: pointer;
    color: #fca5a5;
  }
  
  .alert-icon { font-size: 1.1rem; }
  .alert-text { flex: 1; text-align: left; font-size: 0.85rem; }
  .alert-arrow { color: #666; }
  
  .conflicts-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  
  .conflict-card {
    background: #1a1a24;
    border: 1px solid #2a2a3a;
    border-radius: 12px;
    padding: 1rem;
  }
  
  .conflict-header {
    font-size: 0.8rem;
    color: #f59e0b;
    margin-bottom: 0.75rem;
    font-weight: 600;
  }
  
  .conflict-items {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .conflict-option {
    background: #0f0f15;
    border: 1px solid #2a2a3a;
    border-radius: 10px;
    padding: 0.75rem;
    cursor: pointer;
    text-align: left;
    transition: all 0.2s;
  }
  
  .conflict-option:hover {
    border-color: #10b981;
    background: #10b98110;
  }
  
  .option-text {
    display: block;
    color: #fff;
    font-size: 0.9rem;
    margin-bottom: 0.25rem;
  }
  
  .option-meta {
    font-size: 0.7rem;
    color: #555;
  }
  
  .conflict-vs {
    text-align: center;
    font-size: 0.7rem;
    color: #555;
    font-weight: 700;
  }
  
  .conflict-hint {
    font-size: 0.7rem;
    color: #444;
    text-align: center;
    margin-top: 0.5rem;
  }
  
  /* Loading & Empty */
  .mr-loading {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1rem;
    color: #666;
  }
  
  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #222;
    border-top-color: #a855f7;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }
  
  @keyframes spin { to { transform: rotate(360deg); } }
  
  .mr-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
    cursor: pointer;
  }
  
  .empty-check {
    width: 80px;
    height: 80px;
    background: linear-gradient(135deg, #10b981, #059669);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 2.5rem;
    color: white;
    margin-bottom: 1rem;
  }
  
  .mr-empty h3 { margin: 0 0 0.5rem 0; }
  .mr-empty p { color: #666; margin: 0 0 1.5rem 0; }
  
  .power-summary {
    background: #1a1a2e;
    border-radius: 12px;
    padding: 1rem 2rem;
    margin-bottom: 1rem;
  }
  
  .summary-level {
    font-size: 1.1rem;
    font-weight: 700;
    color: #a855f7;
  }
  
  .summary-stats {
    font-size: 0.8rem;
    color: #666;
  }
  
  .tap-exit {
    font-size: 0.75rem;
    color: #444;
  }
  
  /* Cards */
  .card-stack {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    padding: 0.5rem 0;
  }
  
  .card-behind {
    position: absolute;
    width: 100%;
    max-width: 340px;
    height: 380px;
    background: #1a1a24;
    border-radius: 20px;
    border: 1px solid #2a2a3a;
  }
  
  .memory-card {
    position: relative;
    width: 100%;
    max-width: 340px;
    min-height: 380px;
    background: linear-gradient(160deg, #1e1e2a 0%, #16161e 100%);
    border-radius: 20px;
    border: 2px solid #2a2a3a;
    overflow: hidden;
    touch-action: pan-y;
    cursor: grab;
    transition: transform 0.15s ease-out;
    user-select: none;
    display: flex;
    flex-direction: column;
  }
  
  .memory-card.dragging { cursor: grabbing; transition: none; }
  .memory-card.animating { transition: transform 0.25s ease-out; }
  
  .swipe-overlay {
    position: absolute;
    top: 20px;
    padding: 10px 24px;
    border-radius: 8px;
    font-size: 1.1rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    z-index: 10;
    pointer-events: none;
  }
  
  .swipe-overlay.approve {
    right: 20px;
    background: #10b981;
    color: white;
    transform: rotate(12deg);
    border: 3px solid #fff;
  }
  
  .swipe-overlay.reject {
    left: 20px;
    background: #ef4444;
    color: white;
    transform: rotate(-12deg);
    border: 3px solid #fff;
  }
  
  .card-content {
    flex: 1;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  
  .card-source {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
  }
  
  .source-icon { font-size: 1rem; }
  .source-label { color: #a855f7; font-weight: 500; }
  
  .card-memory {
    font-size: 1.1rem;
    line-height: 1.6;
    color: #fff;
    flex: 1;
  }
  
  .card-context {
    background: #0f0f15;
    border-radius: 10px;
    padding: 0.75rem;
  }
  
  .context-header {
    font-size: 0.7rem;
    color: #a855f7;
    margin-bottom: 0.4rem;
  }
  
  .context-body {
    font-size: 0.8rem;
    color: #888;
    line-height: 1.4;
  }
  
  .card-confidence {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  
  .conf-bar {
    height: 4px;
    background: #222;
    border-radius: 2px;
    overflow: hidden;
  }
  
  .conf-fill {
    height: 100%;
    border-radius: 2px;
  }
  
  .conf-label {
    font-size: 0.7rem;
    font-weight: 500;
  }
  
  .card-actions {
    display: flex;
    gap: 0.5rem;
    padding: 0.75rem 1.25rem 1.25rem;
  }
  
  .btn-action {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.15rem;
    padding: 0.6rem 0.4rem;
    border-radius: 12px;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .btn-action.reject { background: linear-gradient(135deg, #dc2626, #b91c1c); color: white; }
  .btn-action.edit { background: linear-gradient(135deg, #d97706, #b45309); color: white; }
  .btn-action.approve { background: linear-gradient(135deg, #16a34a, #15803d); color: white; }
  .btn-action:active { transform: scale(0.95); }
  
  .btn-icon { font-size: 1.2rem; font-weight: 700; }
  .btn-label { font-size: 0.7rem; font-weight: 600; }
  .btn-power { font-size: 0.6rem; opacity: 0.7; }
  
  /* Correction mode */
  .correction-mode {
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    height: 100%;
  }
  
  .corr-header {
    font-size: 1rem;
    font-weight: 600;
    color: #f59e0b;
  }
  
  .corr-original {
    background: #0f0f15;
    padding: 0.75rem;
    border-radius: 10px;
    font-size: 0.85rem;
  }
  
  .corr-label { color: #555; display: block; font-size: 0.65rem; text-transform: uppercase; margin-bottom: 0.2rem; }
  .corr-text { color: #888; }
  
  .correction-mode textarea {
    flex: 1;
    padding: 0.75rem;
    border-radius: 10px;
    border: 2px solid #2a2a3a;
    background: #0f0f15;
    color: #fff;
    font-family: inherit;
    font-size: 0.95rem;
    resize: none;
  }
  
  .correction-mode textarea:focus { outline: none; border-color: #a855f7; }
  
  .corr-actions {
    display: flex;
    gap: 0.5rem;
  }
  
  .corr-actions button {
    flex: 1;
    padding: 0.75rem;
    border-radius: 10px;
    border: none;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
  }
  
  .btn-cancel { background: #2a2a3a; color: #888; }
  .btn-save { background: linear-gradient(135deg, #a855f7, #7c3aed); color: white; }
  
  .progress-section {
    text-align: center;
    padding: 0.75rem 0;
  }
  
  .progress-text {
    font-size: 0.8rem;
    color: #555;
  }
</style>
