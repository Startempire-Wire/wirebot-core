<script>
  import { createEventDispatcher } from 'svelte';
  let { active } = $props();
  const dispatch = createEventDispatcher();

  const tabs = [
    { id: 'score', icon: '⚡', label: 'Score' },
    { id: 'feed', icon: '📋', label: 'Feed' },
    { id: 'season', icon: '🏆', label: 'Season' },
    { id: 'wrapped', icon: '🎬', label: 'Wrapped' },
    { id: 'settings', icon: '⚙️', label: 'Settings' },
  ];
</script>

<nav>
  {#each tabs as tab}
    <button
      class="tab {active === tab.id ? 'active' : ''}"
      onclick={() => dispatch('nav', tab.id)}
    >
      <span class="tab-icon">{tab.icon}</span>
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
  }

  .tab.active { color: #7c7cff; }
  .tab-icon { font-size: 18px; }
  .tab-label { font-size: 9px; letter-spacing: 0.05em; }
</style>
