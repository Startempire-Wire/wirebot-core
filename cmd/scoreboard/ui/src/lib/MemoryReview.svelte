<script>
  let items = $state([]);
  let counts = $state({ pending: 0, approved: 0, rejected: 0 });
  let loading = $state(true);
  let currentIndex = $state(0);
  let correctionMode = $state(false);
  let correctionText = $state('');
  
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
        if (navigator.vibrate) navigator.vibrate(action === 'approve' ? [50] : [30, 30]);
        
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
    currentX = startX;
    dragging = true;
  }
  
  function handleTouchMove(e) {
    if (!dragging || correctionMode || animatingOut) return;
    currentX = e.touches[0].clientX;
    const deltaY = Math.abs(e.touches[0].clientY - startY);
    const deltaX = currentX - startX;
    
    // Only swipe if horizontal movement is greater than vertical
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
    startX = e.clientX;
    currentX = startX;
    dragging = true;
    e.preventDefault();
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
  
  function expandMemory(text) {
    // Add more readable context to terse memories
    const expansions = {
      'Values systematizing everything': 'You value creating systems and processes to organize everything in life and work.',
      'Business involves debt carrying': 'Your business currently has debt obligations to manage.',
      'Focuses on revenue generation': 'Your primary focus is on generating revenue and closing deals.',
      'Has strong opinions': 'You tend to have strong, well-formed opinions on topics.',
    };
    
    for (const [key, val] of Object.entries(expansions)) {
      if (text.includes(key)) return val;
    }
    return text;
  }
  
  $effect(() => {
    loadQueue();
  });
  
  let currentItem = $derived(items[currentIndex]);
  let nextItems = $derived(items.slice(currentIndex + 1, currentIndex + 3));
</script>

<svelte:window on:mouseup={handleMouseUp} on:mousemove={handleMouseMove} />

<div class="memory-review">
  <header class="mr-header">
    <div class="mr-title">
      <span class="mr-icon">🧠</span>
      <span>Memory Review</span>
    </div>
    <div class="mr-counter">
      <span class="counter-num">{counts.pending}</span>
      <span class="counter-label">to review</span>
    </div>
  </header>
  
  {#if loading}
    <div class="mr-loading">
      <div class="spinner"></div>
      <span>Loading memories...</span>
    </div>
  {:else if items.length === 0}
    <div class="mr-empty">
      <div class="empty-check">✓</div>
      <h3>All caught up!</h3>
      <p>No memories need review right now.</p>
      <div class="empty-stats">
        <span>✓ {counts.approved} approved</span>
        <span>✗ {counts.rejected} rejected</span>
      </div>
    </div>
  {:else if currentItem}
    <div class="card-stack">
      <!-- Background cards for depth -->
      {#each nextItems as _, i}
        <div 
          class="card-behind"
          style="transform: scale({0.95 - i * 0.05}) translateY({(i + 1) * 8}px); opacity: {0.5 - i * 0.2};"
        ></div>
      {/each}
      
      <!-- Main swipeable card -->
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
        <!-- Swipe overlays -->
        <div class="swipe-overlay approve" style="opacity: {swipeDirection === 'right' ? swipeOpacity : 0}">
          <span>APPROVE</span>
        </div>
        <div class="swipe-overlay reject" style="opacity: {swipeDirection === 'left' ? swipeOpacity : 0}">
          <span>REJECT</span>
        </div>
        
        {#if !correctionMode}
          <div class="card-content">
            <!-- Source badge -->
            <div class="card-source">
              <span class="source-icon">{getSourceLabel(currentItem.source_type).icon}</span>
              <span class="source-label">{getSourceLabel(currentItem.source_type).label}</span>
              {#if currentItem.source_file && currentItem.source_file !== 'mem0_migration'}
                <span class="source-file">• {currentItem.source_file}</span>
              {/if}
            </div>
            
            <!-- Main memory text -->
            <div class="card-memory">
              {expandMemory(currentItem.memory_text)}
            </div>
            
            <!-- Original text if expanded -->
            {#if expandMemory(currentItem.memory_text) !== currentItem.memory_text}
              <div class="card-original">
                Original: "{currentItem.memory_text}"
              </div>
            {/if}
            
            <!-- Context if available -->
            {#if currentItem.source_context}
              <div class="card-context">
                <div class="context-header">📝 Source Context</div>
                <div class="context-body">{currentItem.source_context}</div>
              </div>
            {/if}
            
            <!-- Confidence indicator -->
            <div class="card-confidence">
              <div class="conf-bar">
                <div 
                  class="conf-fill" 
                  style="width: {currentItem.confidence * 100}%; background: {getConfidenceBadge(currentItem.confidence).color}"
                ></div>
              </div>
              <span class="conf-label" style="color: {getConfidenceBadge(currentItem.confidence).color}">
                {getConfidenceBadge(currentItem.confidence).label} confidence ({Math.round(currentItem.confidence * 100)}%)
              </span>
            </div>
          </div>
          
          <!-- Action buttons -->
          <div class="card-actions">
            <button class="btn-action reject" onclick={() => takeAction('reject')}>
              <span class="btn-icon">✗</span>
              <span class="btn-label">Reject</span>
            </button>
            <button class="btn-action edit" onclick={() => { correctionMode = true; correctionText = currentItem.memory_text; }}>
              <span class="btn-icon">✎</span>
              <span class="btn-label">Edit</span>
            </button>
            <button class="btn-action approve" onclick={() => takeAction('approve')}>
              <span class="btn-icon">✓</span>
              <span class="btn-label">Approve</span>
            </button>
          </div>
        {:else}
          <!-- Correction mode -->
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
                Save & Approve
              </button>
            </div>
          </div>
        {/if}
      </div>
    </div>
    
    <!-- Progress indicator -->
    <div class="progress-section">
      <div class="progress-dots">
        {#each items.slice(0, Math.min(10, items.length)) as _, i}
          <div class="dot" class:active={i === currentIndex}></div>
        {/each}
        {#if items.length > 10}
          <span class="more-dots">+{items.length - 10}</span>
        {/if}
      </div>
      <div class="progress-text">{currentIndex + 1} of {items.length}</div>
    </div>
    
    <!-- Swipe hints -->
    <div class="swipe-hints">
      <div class="hint">
        <span class="hint-arrow">←</span>
        <span>Swipe to reject</span>
      </div>
      <div class="hint">
        <span>Swipe to approve</span>
        <span class="hint-arrow">→</span>
      </div>
    </div>
  {/if}
</div>

<style>
  .memory-review {
    min-height: 80vh;
    display: flex;
    flex-direction: column;
    padding: 1rem;
    background: linear-gradient(180deg, #0a0a12 0%, #12121a 100%);
  }
  
  .mr-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }
  
  .mr-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1.2rem;
    font-weight: 600;
  }
  
  .mr-icon {
    font-size: 1.4rem;
  }
  
  .mr-counter {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
  }
  
  .counter-num {
    font-size: 1.5rem;
    font-weight: 700;
    color: #a855f7;
  }
  
  .counter-label {
    font-size: 0.7rem;
    color: #666;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
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
  
  @keyframes spin {
    to { transform: rotate(360deg); }
  }
  
  .mr-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
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
  
  .mr-empty h3 {
    margin: 0 0 0.5rem 0;
    font-size: 1.3rem;
  }
  
  .mr-empty p {
    color: #666;
    margin: 0 0 1.5rem 0;
  }
  
  .empty-stats {
    display: flex;
    gap: 1.5rem;
    font-size: 0.85rem;
    color: #888;
  }
  
  .card-stack {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
    padding: 1rem 0;
  }
  
  .card-behind {
    position: absolute;
    width: 100%;
    max-width: 340px;
    height: 400px;
    background: #1a1a24;
    border-radius: 20px;
    border: 1px solid #2a2a3a;
  }
  
  .memory-card {
    position: relative;
    width: 100%;
    max-width: 340px;
    min-height: 400px;
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
  
  .memory-card.dragging {
    cursor: grabbing;
    transition: none;
  }
  
  .memory-card.animating {
    transition: transform 0.25s ease-out;
  }
  
  .swipe-overlay {
    position: absolute;
    top: 20px;
    padding: 8px 20px;
    border-radius: 8px;
    font-size: 1.2rem;
    font-weight: 800;
    letter-spacing: 0.1em;
    z-index: 10;
    pointer-events: none;
    transition: opacity 0.1s;
  }
  
  .swipe-overlay.approve {
    right: 20px;
    background: #10b981;
    color: white;
    transform: rotate(15deg);
    border: 3px solid #fff;
  }
  
  .swipe-overlay.reject {
    left: 20px;
    background: #ef4444;
    color: white;
    transform: rotate(-15deg);
    border: 3px solid #fff;
  }
  
  .card-content {
    flex: 1;
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  
  .card-source {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.8rem;
    color: #888;
  }
  
  .source-icon {
    font-size: 1.1rem;
  }
  
  .source-label {
    color: #a855f7;
    font-weight: 500;
  }
  
  .source-file {
    color: #555;
  }
  
  .card-memory {
    font-size: 1.15rem;
    line-height: 1.6;
    color: #fff;
    flex: 1;
  }
  
  .card-original {
    font-size: 0.8rem;
    color: #555;
    font-style: italic;
    padding: 0.75rem;
    background: #0f0f15;
    border-radius: 8px;
    border-left: 3px solid #333;
  }
  
  .card-context {
    background: #0f0f15;
    border-radius: 10px;
    padding: 0.75rem;
    border: 1px solid #1a1a24;
  }
  
  .context-header {
    font-size: 0.7rem;
    color: #a855f7;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.5rem;
  }
  
  .context-body {
    font-size: 0.85rem;
    color: #888;
    line-height: 1.5;
  }
  
  .card-confidence {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
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
    transition: width 0.3s;
  }
  
  .conf-label {
    font-size: 0.75rem;
    font-weight: 500;
  }
  
  .card-actions {
    display: flex;
    gap: 0.75rem;
    padding: 1rem 1.5rem 1.5rem;
    background: linear-gradient(0deg, #12121a 0%, transparent 100%);
  }
  
  .btn-action {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    padding: 0.75rem;
    border-radius: 14px;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .btn-action.reject {
    background: linear-gradient(135deg, #dc2626, #b91c1c);
    color: white;
  }
  
  .btn-action.edit {
    background: linear-gradient(135deg, #d97706, #b45309);
    color: white;
  }
  
  .btn-action.approve {
    background: linear-gradient(135deg, #16a34a, #15803d);
    color: white;
  }
  
  .btn-action:active {
    transform: scale(0.95);
  }
  
  .btn-icon {
    font-size: 1.3rem;
    font-weight: 700;
  }
  
  .btn-label {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
  
  .correction-mode {
    padding: 1.5rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: 100%;
  }
  
  .corr-header {
    font-size: 1.1rem;
    font-weight: 600;
    color: #f59e0b;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .corr-original {
    background: #0f0f15;
    padding: 0.75rem;
    border-radius: 10px;
    font-size: 0.85rem;
  }
  
  .corr-label {
    color: #555;
    display: block;
    margin-bottom: 0.25rem;
    font-size: 0.7rem;
    text-transform: uppercase;
  }
  
  .corr-text {
    color: #888;
  }
  
  .correction-mode textarea {
    flex: 1;
    padding: 1rem;
    border-radius: 12px;
    border: 2px solid #2a2a3a;
    background: #0f0f15;
    color: #fff;
    font-family: inherit;
    font-size: 1rem;
    line-height: 1.5;
    resize: none;
  }
  
  .correction-mode textarea:focus {
    outline: none;
    border-color: #a855f7;
  }
  
  .corr-actions {
    display: flex;
    gap: 0.75rem;
  }
  
  .corr-actions button {
    flex: 1;
    padding: 0.9rem;
    border-radius: 12px;
    border: none;
    font-size: 0.9rem;
    font-weight: 600;
    cursor: pointer;
  }
  
  .btn-cancel {
    background: #2a2a3a;
    color: #888;
  }
  
  .btn-save {
    background: linear-gradient(135deg, #a855f7, #7c3aed);
    color: white;
  }
  
  .progress-section {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
    margin-top: 1rem;
  }
  
  .progress-dots {
    display: flex;
    align-items: center;
    gap: 6px;
  }
  
  .dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: #333;
    transition: all 0.2s;
  }
  
  .dot.active {
    background: #a855f7;
    transform: scale(1.3);
  }
  
  .more-dots {
    font-size: 0.7rem;
    color: #555;
    margin-left: 4px;
  }
  
  .progress-text {
    font-size: 0.8rem;
    color: #555;
  }
  
  .swipe-hints {
    display: flex;
    justify-content: space-between;
    padding: 1rem 0.5rem 0;
    font-size: 0.75rem;
    color: #444;
  }
  
  .hint {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  
  .hint-arrow {
    font-size: 1rem;
    color: #555;
  }
</style>
