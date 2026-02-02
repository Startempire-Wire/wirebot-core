<script>
  import { onMount } from 'svelte';
  import Score from './lib/Score.svelte';
  import Feed from './lib/Feed.svelte';
  import Season from './lib/Season.svelte';
  import Wrapped from './lib/Wrapped.svelte';
  import Nav from './lib/Nav.svelte';
  import Hints from './lib/Hints.svelte';

  let view = $state('score');
  let data = $state(null);
  let feed = $state([]);
  let history = $state([]);
  let wrapped = $state(null);
  let error = $state(null);
  let lastUpdate = $state('');
  let showFab = $state(false);
  let fabTitle = $state('');
  let fabLane = $state('shipping');
  let showHints = $state(false);
  let showFirstVisit = $state(false);
  let tokenStatus = $state(null);  // null | 'ok' | 'fail'
  let tokenMsg = $state('');

  const API = window.location.origin;

  // Try to get token from localStorage for authenticated calls
  function getToken() {
    return localStorage.getItem('wb_token') || '';
  }

  function authHeaders() {
    const token = getToken();
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  async function saveToken() {
    const input = document.getElementById('token-input');
    const val = (input?.value || '').trim();
    if (!val) {
      localStorage.removeItem('wb_token');
      tokenStatus = 'ok';
      tokenMsg = 'Token cleared';
      setTimeout(() => { tokenStatus = null; }, 3000);
      return;
    }

    // Save immediately
    localStorage.setItem('wb_token', val);

    // Verify by calling a gated endpoint
    tokenStatus = null;
    tokenMsg = 'Verifying...';
    try {
      const res = await fetch(`${API}/v1/events?limit=1`, {
        headers: { 'Authorization': `Bearer ${val}` }
      });
      if (res.ok) {
        tokenStatus = 'ok';
        tokenMsg = '✓ Connected — write features enabled';
      } else {
        tokenStatus = 'fail';
        tokenMsg = `✗ Invalid token (${res.status})`;
        localStorage.removeItem('wb_token');
      }
    } catch (e) {
      tokenStatus = 'fail';
      tokenMsg = '✗ Connection error';
      localStorage.removeItem('wb_token');
    }
    setTimeout(() => { tokenStatus = null; }, 5000);
  }

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

  async function fetchWrapped() {
    const token = getToken();
    if (!token) return;
    try {
      const res = await fetch(`${API}/v1/season/wrapped?token=${token}`);
      if (res.ok) wrapped = await res.json();
    } catch {}
  }

  async function submitFabEvent() {
    if (!fabTitle.trim()) return;
    const token = getToken();
    if (!token) { alert('Set your token in Settings first'); return; }
    try {
      const res = await fetch(`${API}/v1/events`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({
          event_type: 'FEATURE_SHIPPED',
          lane: fabLane,
          source: 'pwa',
          artifact_title: fabTitle,
          confidence: 0.85,
        }),
      });
      if (res.ok) {
        fabTitle = '';
        showFab = false;
        fetchAll();
      }
    } catch {}
  }

  function handleNav(e) {
    view = e.detail;
    if (e.detail === 'wrapped' && !wrapped) fetchWrapped();
  }

  onMount(() => {
    fetchAll();
    const interval = setInterval(fetchAll, 30000);
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
    // First-visit detection
    if (!localStorage.getItem('wb_visited')) {
      showFirstVisit = true;
    }
    return () => clearInterval(interval);
  });

  function dismissFirstVisit() {
    showFirstVisit = false;
    localStorage.setItem('wb_visited', '1');
  }

  function openHintsFromFirstVisit() {
    showFirstVisit = false;
    localStorage.setItem('wb_visited', '1');
    showHints = true;
  }
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
        <Score {data} {lastUpdate} onHelp={() => showHints = true} />
      {:else if view === 'feed'}
        <Feed items={feed} pendingCount={data?.pending_count || 0} onHelp={() => showHints = true} />
      {:else if view === 'season'}
        <Season season={data.season} {history} streak={data.streak} onHelp={() => showHints = true} />
      {:else if view === 'wrapped'}
        <Wrapped {wrapped} />
      {:else if view === 'settings'}
        <div class="settings-view">
          <div class="s-hdr"><h2>⚙️ Settings</h2></div>
          <div class="s-group">
            <label>API Token</label>
            <div class="token-row">
              <input type="password" id="token-input" value={getToken()}
                placeholder="Paste token to enable write features" />
              <button class="btn-save-token" onclick={saveToken}>Save</button>
            </div>
            {#if tokenStatus}
              <div class="token-status" class:ok={tokenStatus === 'ok'} class:fail={tokenStatus === 'fail'}>
                {tokenMsg}
              </div>
            {/if}
            <p class="s-hint">Required for: approve/reject projects, rename, quick-add, intent</p>
          </div>
          <div class="s-group">
            <label>Season</label>
            <div class="s-info">
              <strong>{data.season?.name}</strong> — Season {data.season?.number}<br/>
              {data.season?.start_date} → {data.season?.end_date}<br/>
              "{data.season?.theme}"
            </div>
          </div>
          <div class="s-group">
            <label>Share Cards</label>
            <div class="s-links">
              <a href="/v1/card/daily" target="_blank">📤 Daily Card</a>
              <a href="/v1/card/weekly" target="_blank">📤 Weekly Card</a>
              <a href="/v1/card/season" target="_blank">📤 Season Card</a>
            </div>
          </div>
          <div class="s-group">
            <label>Info</label>
            <div class="s-info">
              Wirebot Scoreboard v1<br/>
              API: {API}/v1/<br/>
              Updated: {lastUpdate}
            </div>
          </div>
        </div>
      {/if}
    </div>

    <!-- Quick-add FAB -->
    {#if view === 'score' || view === 'feed'}
      <button class="fab" onclick={() => showFab = !showFab}>
        {showFab ? '✕' : '＋'}
      </button>
    {/if}

    <!-- FAB panel -->
    {#if showFab}
      <div class="fab-panel">
        <div class="fab-title">Quick Ship</div>
        <input bind:value={fabTitle} placeholder="What did you ship?" class="fab-input"
          onkeydown={(e) => e.key === 'Enter' && submitFabEvent()} />
        <div class="fab-lanes">
          {#each ['shipping', 'distribution', 'revenue', 'systems'] as lane}
            <button class="fab-lane {fabLane === lane ? 'active' : ''}"
              onclick={() => fabLane = lane}>{lane.slice(0,4).toUpperCase()}</button>
          {/each}
        </div>
        <button class="fab-submit" onclick={submitFabEvent}>🚀 Ship It</button>
      </div>
    {/if}

    <!-- Pending badge -->
    {#if data.pending_count > 0}
      <button class="pending-badge" onclick={() => view = 'feed'}>
        ⏳ {data.pending_count} pending
      </button>
    {/if}

    <Nav active={view} pendingCount={data?.pending_count || 0} on:nav={handleNav} />

    <!-- Hints panel -->
    <Hints bind:visible={showHints} />

    <!-- First visit welcome -->
    {#if showFirstVisit}
      <div class="first-visit-overlay">
        <div class="fv-card">
          <div class="fv-icon">⚡</div>
          <h2>Welcome to Scoreboard</h2>
          <p>This is your <strong>execution accountability surface</strong>. It answers one question every day:</p>
          <div class="fv-question">"Am I winning today?"</div>
          <p>Not "am I busy." Not "did I work." But: <strong>did reality change because I worked?</strong></p>

          <div class="fv-quick">
            <div class="fv-q-item">
              <span class="fv-q-icon">🚀</span>
              <span><strong>Ship things</strong> → score goes up</span>
            </div>
            <div class="fv-q-item">
              <span class="fv-q-icon">🎯</span>
              <span><strong>Declare intent</strong> → focus sharpens</span>
            </div>
            <div class="fv-q-item">
              <span class="fv-q-icon">🔥</span>
              <span><strong>Keep shipping</strong> → streak bonus grows</span>
            </div>
            <div class="fv-q-item">
              <span class="fv-q-icon">🏆</span>
              <span><strong>Score ≥ 50</strong> → you win the day</span>
            </div>
          </div>

          <div class="fv-buttons">
            <button class="fv-btn secondary" onclick={openHintsFromFirstVisit}>📘 Learn More</button>
            <button class="fv-btn primary" onclick={dismissFirstVisit}>Let's Go ⚡</button>
          </div>
        </div>
      </div>
    {/if}
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

  .app { display: flex; flex-direction: column; min-height: 100dvh; }
  .content { flex: 1; overflow-y: auto; padding-bottom: 56px; }

  .loading { min-height: 100dvh; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; }
  .ld-icon { font-size: 48px; }
  .loading p { font-size: 14px; opacity: 0.5; }
  .err { color: #f44; }

  /* FAB */
  .fab {
    position: fixed;
    bottom: 72px;
    right: 16px;
    width: 48px;
    height: 48px;
    border-radius: 50%;
    background: #7c7cff;
    color: white;
    font-size: 24px;
    border: none;
    cursor: pointer;
    z-index: 50;
    box-shadow: 0 4px 12px rgba(124,124,255,0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }

  .fab-panel {
    position: fixed;
    bottom: 130px;
    right: 16px;
    left: 16px;
    background: #151520;
    border: 1px solid #2a2a40;
    border-radius: 12px;
    padding: 14px;
    z-index: 50;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .fab-title { font-size: 14px; font-weight: 700; color: #7c7cff; }
  .fab-input {
    background: #0a0a15;
    border: 1px solid #2a2a40;
    border-radius: 8px;
    padding: 10px;
    color: #ddd;
    font-size: 14px;
    outline: none;
  }
  .fab-input:focus { border-color: #7c7cff; }
  .fab-lanes { display: flex; gap: 6px; }
  .fab-lane {
    flex: 1;
    padding: 6px;
    border-radius: 6px;
    background: #0a0a15;
    border: 1px solid #2a2a40;
    color: #888;
    font-size: 11px;
    cursor: pointer;
    text-align: center;
  }
  .fab-lane.active { border-color: #7c7cff; color: #7c7cff; background: rgba(124,124,255,0.1); }
  .fab-submit {
    background: #7c7cff;
    color: white;
    border: none;
    border-radius: 8px;
    padding: 10px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
  }

  /* Pending badge */
  .pending-badge {
    position: fixed;
    top: max(8px, env(safe-area-inset-top));
    right: 12px;
    background: rgba(255,200,0,0.15);
    color: #ffc800;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    z-index: 50;
    cursor: pointer;
  }

  /* First visit */
  .first-visit-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.9);
    z-index: 300;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }
  .fv-card {
    background: #0d0d18;
    border: 1px solid #2a2a4a;
    border-radius: 16px;
    padding: 24px 20px;
    max-width: 360px;
    width: 100%;
    text-align: center;
  }
  .fv-icon { font-size: 48px; margin-bottom: 8px; }
  .fv-card h2 { font-size: 20px; font-weight: 800; margin-bottom: 10px; }
  .fv-card p { font-size: 13px; color: #aaa; line-height: 1.6; margin-bottom: 8px; }
  .fv-card strong { color: #ddd; }
  .fv-question {
    font-size: 18px;
    font-weight: 800;
    color: #7c7cff;
    padding: 12px 0;
  }
  .fv-quick {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin: 16px 0;
    text-align: left;
  }
  .fv-q-item {
    display: flex;
    align-items: center;
    gap: 10px;
    font-size: 13px;
    color: #aaa;
    padding: 6px 10px;
    background: rgba(255,255,255,0.02);
    border-radius: 8px;
  }
  .fv-q-icon { font-size: 18px; flex-shrink: 0; }
  .fv-q-item strong { color: #ddd; }
  .fv-buttons { display: flex; gap: 8px; margin-top: 16px; }
  .fv-btn {
    flex: 1;
    padding: 10px;
    border-radius: 8px;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    border: none;
  }
  .fv-btn.primary { background: #7c7cff; color: white; }
  .fv-btn.secondary { background: transparent; border: 1px solid #333; color: #888; }

  /* Settings */
  .settings-view {
    padding: 12px 16px;
    padding-top: max(12px, env(safe-area-inset-top));
    min-height: calc(100dvh - 56px);
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .s-hdr h2 { font-size: 16px; font-weight: 700; border-bottom: 1px solid #1e1e30; padding-bottom: 6px; }
  .s-group { display: flex; flex-direction: column; gap: 6px; }
  .s-group label { font-size: 12px; font-weight: 600; color: #7c7cff; letter-spacing: 0.05em; }
  .s-group input {
    background: #111118;
    border: 1px solid #2a2a40;
    border-radius: 8px;
    padding: 10px;
    color: #ddd;
    font-size: 13px;
    outline: none;
  }
  .s-group input:focus { border-color: #7c7cff; }
  .s-hint { font-size: 11px; opacity: 0.35; }

  /* Token row */
  .token-row { display: flex; gap: 8px; align-items: stretch; }
  .token-row input { flex: 1; }
  .btn-save-token {
    background: #7c7cff; color: #fff; border: none; border-radius: 8px;
    padding: 8px 18px; font-size: 13px; font-weight: 700; cursor: pointer;
    white-space: nowrap; transition: background 0.15s;
  }
  .btn-save-token:active { background: #5c5cdd; }
  .token-status {
    font-size: 12px; padding: 6px 10px; border-radius: 6px; margin-top: 2px;
    animation: fadeIn 0.2s ease;
  }
  .token-status.ok { color: #2ecc71; background: rgba(46,204,113,0.08); }
  .token-status.fail { color: #ff4444; background: rgba(255,68,68,0.08); }
  @keyframes fadeIn { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }

  .s-info { font-size: 13px; opacity: 0.6; line-height: 1.6; }
  .s-links { display: flex; gap: 8px; flex-wrap: wrap; }
  .s-links a {
    color: #7c7cff;
    text-decoration: none;
    font-size: 13px;
    padding: 6px 12px;
    border: 1px solid #2a2a40;
    border-radius: 6px;
  }
</style>
