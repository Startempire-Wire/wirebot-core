<script>
  import { createEventDispatcher } from 'svelte';
  let { active, pendingCount = 0 } = $props();
  const dispatch = createEventDispatcher();

  const tabs = [
    { id: 'dashboard', icon: '🏠', label: 'Home' },
    { id: 'score', icon: '⚡', label: 'Score' },
    { id: 'feed', icon: '📋', label: 'Feed' },
    { id: 'season', icon: '🏆', label: 'Season' },
    { id: 'settings', icon: '⚙️', label: 'Settings' },
  ];
</script>

<nav>
  {#each tabs as tab}
    <button
      class="tab {active === tab.id ? 'active' : ''}"
      onclick={() => dispatch('nav', tab.id)}
    >
      <span class="tab-icon">
        {tab.icon}
        {#if tab.id === 'feed' && pendingCount > 0}
          <span class="nav-badge">{pendingCount > 99 ? '99+' : pendingCount}</span>
        {/if}
      </span>
      <span class="tab-label">{tab.label}</span>
    </button>
  {/each}
</nav>

<style>
  nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 56px;
    display: flex;
    background: #111118;
    border-top: 1px solid #1e1e30;
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
    color: #555;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    transition: color 0.2s;
    position: relative;
  }

  .tab.active { color: #7c7cff; }
  .tab.active::before {
    content: '';
    position: absolute;
    top: 0;
    left: 50%;
    transform: translateX(-50%);
    width: 24px;
    height: 2px;
    background: #7c7cff;
    border-radius: 0 0 2px 2px;
    animation: indicator-in 200ms cubic-bezier(0.32, 0.72, 0, 1);
  }
  @keyframes indicator-in {
    from { width: 0; opacity: 0; }
    to { width: 24px; opacity: 1; }
  }
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
    background: #ff4444;
    color: white;
    font-size: 9px;
    font-weight: 700;
    display: flex;
    align-items: center;
    justify-content: center;
    line-height: 1;
  }
</style>
