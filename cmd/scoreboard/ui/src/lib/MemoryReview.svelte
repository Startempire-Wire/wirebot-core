<script>
  let items = $state([]);
  let counts = $state({ pending: 0, approved: 0, rejected: 0 });
  let loading = $state(true);
  let currentIndex = $state(0);
  let correctionMode = $state(false);
  let correctionText = $state('');
  
  // Swipe state
  let startX = 0;
  let currentX = 0;
  let dragging = $state(false);
  let swipeOffset = $state(0);
  let swipeDirection = $state(null); // 'left' | 'right' | null
  
  const SWIPE_THRESHOLD = 80;
  
  const API = '';
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || '' : '';
  
  function authHeaders() {
    return token ? { 'Authorization': `Bearer ${token}` } : {};
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
      currentIndex = 0;
    } catch (e) {
      console.error('Failed to load memory queue:', e);
    }
    loading = false;
  }
  
  async function takeAction(action, correction = null) {
    if (!currentItem) return;
    
    const body = correction ? { correction } : {};
    try {
      const res = await fetch(`${API}/v1/memory/queue/${currentItem.id}/${action}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify(body)
      });
      if (res.ok) {
        if (navigator.vibrate) navigator.vibrate(action === 'approve' ? [50] : [30, 30]);
        
        // Animate out then remove
        items = items.filter((_, i) => i !== currentIndex);
        counts.pending = Math.max(0, counts.pending - 1);
        if (action === 'approve' || action === 'correct') counts.approved++;
        if (action === 'reject') counts.rejected++;
        
        // Reset for next card
        if (currentIndex >= items.length) currentIndex = Math.max(0, items.length - 1);
        correctionMode = false;
        correctionText = '';
      }
    } catch (e) {
      console.error('Action failed:', e);
    }
  }
  
  function handleTouchStart(e) {
    if (correctionMode) return;
    startX = e.touches[0].clientX;
    currentX = startX;
    dragging = true;
  }
  
  function handleTouchMove(e) {
    if (!dragging || correctionMode) return;
    currentX = e.touches[0].clientX;
    swipeOffset = currentX - startX;
    
    if (swipeOffset > 30) swipeDirection = 'right';
    else if (swipeOffset < -30) swipeDirection = 'left';
    else swipeDirection = null;
  }
  
  function handleTouchEnd() {
    if (!dragging) return;
    dragging = false;
    
    if (swipeOffset > SWIPE_THRESHOLD) {
      // Swipe right = approve
      swipeOffset = 300;
      setTimeout(() => {
        takeAction('approve');
        swipeOffset = 0;
        swipeDirection = null;
      }, 200);
    } else if (swipeOffset < -SWIPE_THRESHOLD) {
      // Swipe left = reject
      swipeOffset = -300;
      setTimeout(() => {
        takeAction('reject');
        swipeOffset = 0;
        swipeDirection = null;
      }, 200);
    } else {
      // Snap back
      swipeOffset = 0;
      swipeDirection = null;
    }
  }
  
  function handleMouseDown(e) {
    if (correctionMode) return;
    startX = e.clientX;
    currentX = startX;
    dragging = true;
  }
  
  function handleMouseMove(e) {
    if (!dragging || correctionMode) return;
    currentX = e.clientX;
    swipeOffset = currentX - startX;
    
    if (swipeOffset > 30) swipeDirection = 'right';
    else if (swipeOffset < -30) swipeDirection = 'left';
    else swipeDirection = null;
  }
  
  function handleMouseUp() {
    handleTouchEnd();
  }
  
  function getConfidenceColor(conf) {
    if (conf >= 0.8) return '#10b981';
    if (conf >= 0.5) return '#f59e0b';
    return '#ef4444';
  }
  
  function getSourceIcon(type) {
    switch (type) {
      case 'obsidian': return '📓';
      case 'conversation': return '💬';
      case 'bootstrap': return '🚀';
      case 'inference': return '🤖';
      default: return '📄';
    }
  }
  
  $effect(() => {
    loadQueue();
  });
  
  let currentItem = $derived(items[currentIndex]);
</script>

<svelte:window on:mouseup={handleMouseUp} on:mousemove={handleMouseMove} />

<div class="memory-review">
  <header class="mr-header">
    <h3>🧠 Memory Review</h3>
    <div class="mr-stats">
      <span class="stat pending">{counts.pending} pending</span>
      <span class="stat approved">✓ {counts.approved}</span>
      <span class="stat rejected">✗ {counts.rejected}</span>
    </div>
  </header>
  
  {#if loading}
    <div class="mr-loading">
      <div class="spinner"></div>
      Loading memories...
    </div>
  {:else if items.length === 0}
    <div class="mr-empty">
      <div class="empty-icon">✅</div>
      <div class="empty-text">No pending memories</div>
      <div class="empty-sub">All caught up!</div>
    </div>
  {:else if currentItem}
    <div class="swipe-hints">
      <span class="hint-left" class:active={swipeDirection === 'left'}>← Reject</span>
      <span class="hint-right" class:active={swipeDirection === 'right'}>Approve →</span>
    </div>
    
    <div class="card-container">
      <div 
        class="memory-card"
        class:dragging
        class:swipe-left={swipeDirection === 'left'}
        class:swipe-right={swipeDirection === 'right'}
        style="transform: translateX({swipeOffset}px) rotate({swipeOffset * 0.03}deg)"
        ontouchstart={handleTouchStart}
        ontouchmove={handleTouchMove}
        ontouchend={handleTouchEnd}
        onmousedown={handleMouseDown}
      >
        {#if !correctionMode}
          <div class="card-memory">"{currentItem.memory_text}"</div>
          
          <div class="card-meta">
            <span class="meta-source">
              {getSourceIcon(currentItem.source_type)} {currentItem.source_type}
            </span>
            <span class="meta-conf" style="background: {getConfidenceColor(currentItem.confidence)}20; color: {getConfidenceColor(currentItem.confidence)}">
              {Math.round(currentItem.confidence * 100)}%
            </span>
          </div>
          
          {#if currentItem.source_file}
            <div class="card-file">📁 {currentItem.source_file}</div>
          {/if}
          
          {#if currentItem.source_context}
            <div class="card-context">
              <div class="context-label">Context:</div>
              <div class="context-text">{currentItem.source_context.slice(0, 300)}</div>
            </div>
          {/if}
          
          <div class="card-actions">
            <button class="btn-reject" onclick={() => takeAction('reject')}>
              ❌ Reject
            </button>
            <button class="btn-edit" onclick={() => { correctionMode = true; correctionText = currentItem.memory_text; }}>
              ✏️ Edit
            </button>
            <button class="btn-approve" onclick={() => takeAction('approve')}>
              ✅ Approve
            </button>
          </div>
        {:else}
          <div class="correction-mode">
            <div class="correction-label">✏️ Edit this memory:</div>
            <textarea 
              bind:value={correctionText}
              placeholder="Enter the corrected fact..."
              rows="4"
            ></textarea>
            <div class="correction-actions">
              <button class="btn-cancel" onclick={() => { correctionMode = false; correctionText = ''; }}>
                Cancel
              </button>
              <button class="btn-save" onclick={() => takeAction('correct', correctionText)}>
                Save & Approve
              </button>
            </div>
          </div>
        {/if}
      </div>
      
      <div class="card-progress">
        {currentIndex + 1} of {items.length}
      </div>
    </div>
    
    <div class="swipe-tutorial">
      Swipe right to approve • Swipe left to reject • Tap Edit to correct
    </div>
  {/if}
</div>

<style>
  .memory-review {
    padding: 1rem;
    min-height: 70vh;
    display: flex;
    flex-direction: column;
  }
  
  .mr-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }
  
  .mr-header h3 {
    margin: 0;
    font-size: 1.1rem;
  }
  
  .mr-stats {
    display: flex;
    gap: 0.5rem;
    font-size: 0.75rem;
  }
  
  .stat {
    padding: 0.25rem 0.5rem;
    border-radius: 12px;
    background: #222;
  }
  
  .stat.pending { color: #f59e0b; }
  .stat.approved { color: #10b981; }
  .stat.rejected { color: #888; }
  
  .mr-loading, .mr-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #666;
  }
  
  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid #333;
    border-top-color: #7c7cff;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 1rem;
  }
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .empty-icon {
    font-size: 3rem;
    margin-bottom: 0.5rem;
  }
  
  .empty-text {
    font-size: 1.1rem;
    color: #888;
  }
  
  .empty-sub {
    font-size: 0.85rem;
    color: #555;
  }
  
  .swipe-hints {
    display: flex;
    justify-content: space-between;
    padding: 0 1rem;
    margin-bottom: 0.5rem;
    font-size: 0.8rem;
    color: #444;
  }
  
  .hint-left.active { color: #ef4444; font-weight: 600; }
  .hint-right.active { color: #10b981; font-weight: 600; }
  
  .card-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  
  .memory-card {
    width: 100%;
    max-width: 380px;
    background: linear-gradient(145deg, #1a1a24, #12121a);
    border: 1px solid #2a2a3a;
    border-radius: 16px;
    padding: 1.25rem;
    touch-action: pan-y;
    cursor: grab;
    transition: transform 0.1s ease-out, box-shadow 0.2s;
    user-select: none;
  }
  
  .memory-card.dragging {
    cursor: grabbing;
    transition: none;
  }
  
  .memory-card.swipe-right {
    box-shadow: 0 0 30px #10b98140;
    border-color: #10b981;
  }
  
  .memory-card.swipe-left {
    box-shadow: 0 0 30px #ef444440;
    border-color: #ef4444;
  }
  
  .card-memory {
    font-size: 1.05rem;
    line-height: 1.5;
    color: #fff;
    margin-bottom: 1rem;
  }
  
  .card-meta {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
  }
  
  .meta-source {
    font-size: 0.75rem;
    color: #888;
    background: #252530;
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
  }
  
  .meta-conf {
    font-size: 0.75rem;
    padding: 0.25rem 0.5rem;
    border-radius: 6px;
    font-weight: 600;
  }
  
  .card-file {
    font-size: 0.75rem;
    color: #666;
    margin-bottom: 0.75rem;
  }
  
  .card-context {
    background: #0f0f15;
    border-radius: 8px;
    padding: 0.75rem;
    margin-bottom: 1rem;
    font-size: 0.8rem;
  }
  
  .context-label {
    color: #7c7cff;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
  }
  
  .context-text {
    color: #888;
    line-height: 1.4;
  }
  
  .card-actions {
    display: flex;
    gap: 0.5rem;
  }
  
  .card-actions button {
    flex: 1;
    padding: 0.75rem 0.5rem;
    border-radius: 10px;
    border: none;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .btn-reject {
    background: #7f1d1d;
    color: #fff;
  }
  
  .btn-reject:hover, .btn-reject:active {
    background: #991b1b;
  }
  
  .btn-edit {
    background: #854d0e;
    color: #fff;
  }
  
  .btn-edit:hover, .btn-edit:active {
    background: #a16207;
  }
  
  .btn-approve {
    background: #166534;
    color: #fff;
  }
  
  .btn-approve:hover, .btn-approve:active {
    background: #15803d;
  }
  
  .correction-mode {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  
  .correction-label {
    font-size: 0.9rem;
    color: #f59e0b;
  }
  
  .correction-mode textarea {
    width: 100%;
    padding: 0.75rem;
    border-radius: 10px;
    border: 1px solid #333;
    background: #0f0f15;
    color: #fff;
    font-family: inherit;
    font-size: 0.95rem;
    resize: vertical;
    line-height: 1.5;
  }
  
  .correction-mode textarea:focus {
    outline: none;
    border-color: #7c7cff;
  }
  
  .correction-actions {
    display: flex;
    gap: 0.5rem;
  }
  
  .correction-actions button {
    flex: 1;
    padding: 0.75rem;
    border-radius: 10px;
    border: none;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
  }
  
  .btn-cancel {
    background: #333;
    color: #888;
  }
  
  .btn-save {
    background: #7c7cff;
    color: #fff;
  }
  
  .card-progress {
    margin-top: 1rem;
    font-size: 0.8rem;
    color: #555;
  }
  
  .swipe-tutorial {
    text-align: center;
    font-size: 0.75rem;
    color: #444;
    margin-top: 1rem;
    padding: 0.75rem;
    background: #111;
    border-radius: 8px;
  }
</style>
