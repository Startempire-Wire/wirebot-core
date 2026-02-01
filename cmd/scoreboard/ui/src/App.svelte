<script>
  import { onMount } from 'svelte';
  import Stadium from './lib/Stadium.svelte';

  let data = $state(null);
  let error = $state(null);
  let lastUpdate = $state('');

  const API = window.location.origin;

  async function fetchScoreboard() {
    try {
      const res = await fetch(`${API}/v1/scoreboard`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      data = await res.json();
      error = null;
      lastUpdate = new Date().toLocaleTimeString();
    } catch (e) {
      error = e.message;
    }
  }

  onMount(() => {
    fetchScoreboard();
    const interval = setInterval(fetchScoreboard, 30000);

    // Register PWA service worker
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }

    return () => clearInterval(interval);
  });
</script>

<main>
  {#if error && !data}
    <div class="loading">
      <h1>⚡</h1>
      <p>Connecting to scoreboard...</p>
      <p class="error">{error}</p>
    </div>
  {:else if data}
    <Stadium {data} {lastUpdate} />
  {:else}
    <div class="loading">
      <h1>⚡</h1>
      <p>Loading...</p>
    </div>
  {/if}
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    background: #0a0a0f;
    color: #e0e0e0;
    font-family: 'SF Mono', 'Fira Code', 'JetBrains Mono', monospace;
    overflow: hidden;
    height: 100vh;
    width: 100vw;
  }

  main {
    height: 100vh;
    width: 100vw;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .loading {
    text-align: center;
  }

  .loading h1 {
    font-size: 4rem;
    margin: 0;
  }

  .loading p {
    font-size: 1.2rem;
    opacity: 0.6;
  }

  .error {
    color: #ff4444;
  }
</style>
