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

    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }

    return () => clearInterval(interval);
  });
</script>

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

<style>
  :global(*) {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
  }

  :global(html, body) {
    background: #0a0a0f;
    color: #e0e0e0;
    width: 100%;
    height: 100%;
    overflow-x: hidden;
  }

  .loading {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    font-family: system-ui, -apple-system, sans-serif;
  }

  .loading h1 {
    font-size: 3rem;
  }

  .loading p {
    font-size: 1rem;
    opacity: 0.6;
  }

  .error {
    color: #ff4444;
  }
</style>
