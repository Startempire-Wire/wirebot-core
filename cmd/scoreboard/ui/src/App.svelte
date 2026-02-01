<script>
  import { onMount } from 'svelte';
  import Score from './lib/Score.svelte';
  import Feed from './lib/Feed.svelte';
  import Season from './lib/Season.svelte';
  import Nav from './lib/Nav.svelte';

  let view = $state('score');
  let data = $state(null);
  let feed = $state([]);
  let history = $state([]);
  let error = $state(null);
  let lastUpdate = $state('');

  const API = window.location.origin;

  async function fetchAll() {
    try {
      const [sbRes, feedRes, histRes] = await Promise.all([
        fetch(`${API}/v1/scoreboard?mode=dashboard`),
        fetch(`${API}/v1/feed?limit=50`),
        fetch(`${API}/v1/history?range=season`),
      ]);

      if (sbRes.ok) {
        const sb = await sbRes.json();
        data = sb.scoreboard || sb;
        if (sb.feed) feed = sb.feed;
      }
      if (feedRes.ok) {
        const f = await feedRes.json();
        feed = f.items || [];
      }
      if (histRes.ok) {
        const h = await histRes.json();
        history = h.days || [];
      }
      error = null;
      lastUpdate = new Date().toLocaleTimeString();
    } catch (e) {
      error = e.message;
    }
  }

  function handleNav(e) {
    view = e.detail;
  }

  onMount(() => {
    fetchAll();
    const interval = setInterval(fetchAll, 30000);
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
    return () => clearInterval(interval);
  });
</script>

{#if error && !data}
  <div class="loading">
    <div class="ld-icon">⚡</div>
    <p>Connecting...</p>
    <p class="err">{error}</p>
  </div>
{:else if data}
  <div class="app">
    <div class="content">
      {#if view === 'score'}
        <Score {data} {lastUpdate} />
      {:else if view === 'feed'}
        <Feed items={feed} />
      {:else if view === 'season'}
        <Season season={data.season} {history} streak={data.streak} />
      {/if}
    </div>
    <Nav active={view} on:nav={handleNav} />
  </div>
{:else}
  <div class="loading">
    <div class="ld-icon">⚡</div>
    <p>Loading...</p>
  </div>
{/if}

<style>
  :global(*) { margin: 0; padding: 0; box-sizing: border-box; }
  :global(html, body) {
    background: #0a0a0f;
    color: #ddd;
    width: 100%;
    height: 100%;
    overflow-x: hidden;
    font-family: system-ui, -apple-system, sans-serif;
    -webkit-font-smoothing: antialiased;
  }

  .app {
    display: flex;
    flex-direction: column;
    min-height: 100dvh;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding-bottom: 56px; /* nav height */
  }

  .loading {
    min-height: 100dvh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
  }
  .ld-icon { font-size: 48px; }
  .loading p { font-size: 14px; opacity: 0.5; }
  .err { color: #f44; }
</style>
