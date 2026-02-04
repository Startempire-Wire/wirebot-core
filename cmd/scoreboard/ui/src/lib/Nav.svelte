<script>
  import { createEventDispatcher, onMount } from 'svelte';
  let { active, pendingCount = 0, memoryPendingCount = 0 } = $props();
  const dispatch = createEventDispatcher();

  const defaultTabs = [
    { id: 'dashboard', icon: '🏠', label: 'Home' },
    { id: 'score', icon: '⚡', label: 'Score' },
    { id: 'feed', icon: '📋', label: 'Feed' },
    { id: 'memory', icon: '🧠', label: 'Memory' },
    { id: 'settings', icon: '⚙️', label: 'Settings' },
  ];

  let tabs = $state([...defaultTabs]);
  let dragging = $state(null);
  let dragOver = $state(null);
  let editMode = $state(false);

  onMount(() => {
    // Load saved order from localStorage
    const saved = localStorage.getItem('nav_order');
    if (saved) {
      try {
        const order = JSON.parse(saved);
        tabs = order.map(id => defaultTabs.find(t => t.id === id)).filter(Boolean);
        // Add any missing tabs
        defaultTabs.forEach(t => {
          if (!tabs.find(x => x.id === t.id)) tabs.push(t);
        });
      } catch {}
    }
  });

  function saveOrder() {
    localStorage.setItem('nav_order', JSON.stringify(tabs.map(t => t.id)));
  }

  // Drag and drop handlers
  function handleDragStart(e, idx) {
    if (!editMode) return;
    dragging = idx;
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/plain', idx);
    // Haptic
    if (navigator.vibrate) navigator.vibrate(20);
  }

  function handleDragOver(e, idx) {
    if (!editMode || dragging === null) return;
    e.preventDefault();
    dragOver = idx;
  }

  function handleDrop(e, idx) {
    if (!editMode || dragging === null) return;
    e.preventDefault();
    if (dragging !== idx) {
      const newTabs = [...tabs];
      const [moved] = newTabs.splice(dragging, 1);
      newTabs.splice(idx, 0, moved);
      tabs = newTabs;
      saveOrder();
      if (navigator.vibrate) navigator.vibrate([20, 10, 20]);
    }
    dragging = null;
    dragOver = null;
  }

  function handleDragEnd() {
    dragging = null;
    dragOver = null;
  }

  // Touch-based drag for mobile
  let touchStartX = 0;
  let touchStartIdx = null;

  function handleTouchStart(e, idx) {
    if (!editMode) return;
    touchStartX = e.touches[0].clientX;
    touchStartIdx = idx;
    dragging = idx;
    if (navigator.vibrate) navigator.vibrate(20);
  }

  function handleTouchMove(e) {
    if (!editMode || touchStartIdx === null) return;
    const touch = e.touches[0];
    const el = document.elementFromPoint(touch.clientX, touch.clientY);
    const tabEl = el?.closest('.tab');
    if (tabEl) {
      const idx = parseInt(tabEl.dataset.idx);
      if (!isNaN(idx)) dragOver = idx;
    }
  }

  function handleTouchEnd() {
    if (!editMode || touchStartIdx === null || dragOver === null) {
      dragging = null;
      dragOver = null;
      touchStartIdx = null;
      return;
    }
    if (touchStartIdx !== dragOver) {
      const newTabs = [...tabs];
      const [moved] = newTabs.splice(touchStartIdx, 1);
      newTabs.splice(dragOver, 0, moved);
      tabs = newTabs;
      saveOrder();
      if (navigator.vibrate) navigator.vibrate([20, 10, 20]);
    }
    dragging = null;
    dragOver = null;
    touchStartIdx = null;
  }

  function toggleEditMode() {
    editMode = !editMode;
    if (navigator.vibrate) navigator.vibrate(editMode ? [30, 20, 30] : 30);
  }

  function resetOrder() {
    tabs = [...defaultTabs];
    localStorage.removeItem('nav_order');
    if (navigator.vibrate) navigator.vibrate(50);
  }
</script>

<nav class:edit-mode={editMode}>
  {#each tabs as tab, idx}
    {@const isMiddle = idx === Math.floor(tabs.length / 2)}
    <button
      class="tab {active === tab.id ? 'active' : ''} {isMiddle ? 'middle' : ''} {dragging === idx ? 'dragging' : ''} {dragOver === idx && dragging !== idx ? 'drag-over' : ''}"
      data-idx={idx}
      onclick={() => !editMode && dispatch('nav', tab.id)}
      draggable={editMode}
      ondragstart={(e) => handleDragStart(e, idx)}
      ondragover={(e) => handleDragOver(e, idx)}
      ondrop={(e) => handleDrop(e, idx)}
      ondragend={handleDragEnd}
      ontouchstart={(e) => handleTouchStart(e, idx)}
      ontouchmove={handleTouchMove}
      ontouchend={handleTouchEnd}
    >
      {#if editMode}
        <span class="drag-handle">⋮⋮</span>
      {/if}
      <span class="tab-icon">
        {tab.icon}
        {#if tab.id === 'feed' && pendingCount > 0 && !editMode}
          <span class="nav-badge">{pendingCount > 99 ? '99+' : pendingCount}</span>
        {/if}
        {#if tab.id === 'settings' && memoryPendingCount > 0 && !editMode}
          <span class="nav-badge memory">{memoryPendingCount > 99 ? '99+' : memoryPendingCount}</span>
        {/if}
      </span>
      <span class="tab-label">{tab.label}</span>
    </button>
  {/each}

  <!-- Edit mode toggle (long-press hint area) -->
  <button class="edit-toggle" onclick={toggleEditMode} title={editMode ? 'Done editing' : 'Rearrange tabs'}>
    {editMode ? '✓' : '✎'}
  </button>
</nav>

{#if editMode}
  <div class="edit-hint">
    <span>Drag tabs to rearrange • Middle tab is featured</span>
    <button class="reset-btn" onclick={resetOrder}>Reset</button>
  </div>
{/if}

<style>
  nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 60px;
    display: flex;
    align-items: flex-end;
    background: var(--bg-card);
    border-top: 1px solid var(--border);
    padding-bottom: env(safe-area-inset-bottom, 0);
    z-index: 100;
    view-transition-name: nav;
  }

  .tab {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 2px;
    background: none;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    transition: all 0.2s ease;
    position: relative;
    height: 56px;
    padding-bottom: 4px;
  }

  /* Middle tab is 2x larger with floating effect */
  .tab.middle {
    flex: 1.8;
    height: 72px;
    margin-top: -16px;
    background: var(--card-gradient);
    border-radius: 16px 16px 0 0;
    border: 1px solid var(--border-light);
    border-bottom: none;
    box-shadow: 0 -4px 20px rgba(124, 124, 255, 0.15);
  }
  .tab.middle .tab-icon { font-size: 26px; }
  .tab.middle .tab-label { font-size: 10px; font-weight: 600; }
  .tab.middle.active {
    background: var(--card-gradient);
    box-shadow: 0 -4px 24px rgba(124, 124, 255, 0.25);
  }

  .tab.active { color: var(--accent); }
  .tab.active::before {
    content: '';
    position: absolute;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 24px;
    height: 2px;
    background: var(--accent);
    border-radius: 0 0 2px 2px;
  }
  .tab.middle.active::before { display: none; }

  .tab-icon { font-size: 18px; position: relative; transition: transform 0.15s; }
  .tab.active .tab-icon { transform: scale(1.1); }
  .tab-label { font-size: 9px; letter-spacing: 0.05em; }

  .nav-badge {
    position: absolute;
    top: -6px;
    right: -10px;
    min-width: 16px;
    height: 16px;
    padding: 0 4px;
    border-radius: 8px;
    background: var(--error);
    color: white;
    font-size: 9px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .nav-badge.memory { background: #a855f7; }

  /* Edit mode */
  nav.edit-mode { background: var(--bg); }
  nav.edit-mode .tab {
    cursor: grab;
    animation: wiggle 0.3s ease-in-out infinite;
  }
  nav.edit-mode .tab:nth-child(odd) { animation-delay: 0.1s; }
  @keyframes wiggle {
    0%, 100% { transform: rotate(-1deg); }
    50% { transform: rotate(1deg); }
  }

  .tab.dragging {
    opacity: 0.5;
    transform: scale(0.95);
    animation: none;
  }
  .tab.drag-over {
    background: rgba(124, 124, 255, 0.1);
    border: 1px dashed #7c7cff;
  }

  .drag-handle {
    position: absolute;
    top: 4px;
    font-size: 10px;
    color: var(--text-muted);
    letter-spacing: -2px;
  }

  .edit-toggle {
    position: absolute;
    top: -20px;
    right: 8px;
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--bg-elevated);
    border: 1px solid var(--border-light);
    color: var(--accent);
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0.6;
    transition: opacity 0.2s;
  }
  .edit-toggle:hover { opacity: 1; }
  nav.edit-mode .edit-toggle {
    background: var(--accent);
    color: var(--bg);
    opacity: 1;
  }

  .edit-hint {
    position: fixed;
    bottom: calc(60px + env(safe-area-inset-bottom, 0) + 8px);
    left: 50%;
    transform: translateX(-50%);
    background: var(--bg-card);
    border: 1px solid var(--border-light);
    border-radius: 8px;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    gap: 12px;
    font-size: 11px;
    color: var(--text-secondary);
    z-index: 99;
    backdrop-filter: blur(8px);
  }
  .reset-btn {
    background: rgba(255, 80, 80, 0.1);
    border: 1px solid rgba(255, 80, 80, 0.3);
    color: var(--error);
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 10px;
    cursor: pointer;
  }
</style>
