<script>
  /**
   * SystemStatus — Admin-only system health panel for Settings page.
   * - Lazy: only polls when visible (IntersectionObserver)
   * - Non-blocking: fetch in background, never freezes UI
   * - Live: 30s interval when visible, stops when scrolled away
   */
  let { token = '' } = $props();
  let expanded = $state(JSON.parse(localStorage.getItem('sys_health_open') || 'false'));
  let health = $state(null);
  let loading = $state(true);
  let error = $state(false);
  let polling = $state(false);
  let restarting = $state({});
  let restartMsg = $state({});
  let lastChecked = $state('');
  let pollCount = $state(0);
  let visible = $state(false);
  let panelEl = $state(null);
  let pollTimer = null;

  async function fetchHealth() {
    if (polling) return; // skip if previous still running
    polling = true;
    pollCount++;
    try {
      const controller = new AbortController();
      const timeout = setTimeout(() => controller.abort(), 8000);
      const resp = await fetch('/v1/system/health', {
        headers: { 'Authorization': `Bearer ${token}` },
        signal: controller.signal
      });
      clearTimeout(timeout);
      if (resp.ok) {
        health = await resp.json();
        lastChecked = new Date().toLocaleTimeString();
        error = false;
      } else {
        error = true;
      }
    } catch (e) {
      if (e.name !== 'AbortError') error = true;
    }
    loading = false;
    polling = false;
  }

  // IntersectionObserver — only poll when panel is on screen
  $effect(() => {
    if (!panelEl) return;
    const obs = new IntersectionObserver(
      ([entry]) => {
        visible = entry.isIntersecting;
      },
      { threshold: 0.1 }
    );
    obs.observe(panelEl);
    return () => obs.disconnect();
  });

  function toggleExpand() {
    expanded = !expanded;
    localStorage.setItem('sys_health_open', JSON.stringify(expanded));
  }

  // Start/stop polling based on visibility AND expanded state
  $effect(() => {
    if (visible && expanded) {
      // First fetch immediately (non-blocking via requestIdleCallback)
      if ('requestIdleCallback' in window) {
        requestIdleCallback(() => fetchHealth());
      } else {
        setTimeout(fetchHealth, 100);
      }
      pollTimer = setInterval(() => {
        if ('requestIdleCallback' in window) {
          requestIdleCallback(() => fetchHealth());
        } else {
          fetchHealth();
        }
      }, 30000);
    } else {
      if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
    }
    return () => { if (pollTimer) { clearInterval(pollTimer); pollTimer = null; } };
  });

  async function restartService(name) {
    restarting = { ...restarting, [name]: true };
    restartMsg = { ...restartMsg, [name]: '' };
    try {
      const resp = await fetch('/v1/system/restart', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ service: name })
      });
      const data = await resp.json();
      if (resp.ok) {
        restartMsg = { ...restartMsg, [name]: '✓ Restarted' };
        setTimeout(fetchHealth, 5000);
        setTimeout(fetchHealth, 12000);
      } else {
        restartMsg = { ...restartMsg, [name]: `✗ ${data.error || 'Failed'}` };
      }
    } catch (e) {
      restartMsg = { ...restartMsg, [name]: `✗ ${e.message}` };
    }
    restarting = { ...restarting, [name]: false };
    setTimeout(() => { restartMsg = { ...restartMsg, [name]: '' }; }, 8000);
  }

  function dot(status) {
    if (status === 'up') return '🟢';
    if (status === 'degraded') return '🟡';
    return '🔴';
  }

  function svcLabel(name) {
    return { scoreboard: 'Scoreboard', gateway: 'AI Gateway', mem0: 'Mem0 Memory', letta: 'Letta State', 'memory-sync': 'Memory Sync' }[name] || name;
  }

  function svcDesc(name) {
    return {
      scoreboard: 'WINS portal + API',
      gateway: 'Discord bot, LLM routing',
      mem0: 'Fact storage & search',
      letta: 'Business state blocks',
      'memory-sync': 'Hot cache daemon'
    }[name] || '';
  }

  function canRestart(name) {
    return name !== 'scoreboard';
  }

  function diskColor(pct) {
    if (pct < 60) return 'var(--green, #22c55e)';
    if (pct < 80) return 'var(--yellow, #eab308)';
    return 'var(--red, #ef4444)';
  }
</script>

<div class="sys-panel" bind:this={panelEl}>
  <button class="sys-header" onclick={toggleExpand}>
    <div class="sys-title">
      <span class="sys-chevron" class:open={expanded}>▸</span>
      <span class="sys-icon">🖥️</span>
      <span>System Health</span>
      {#if polling}
        <span class="sys-polling">⟳</span>
      {/if}
    </div>
    <div class="sys-meta">
      {#if health && !expanded}
        <span class="sys-dots">
          {#each health.services as svc}{dot(svc.status)}{/each}
        </span>
      {/if}
      {#if lastChecked && expanded}
        <span class="sys-time">{lastChecked}</span>
      {/if}
      {#if expanded}
        <button class="sys-refresh" onclick={(e) => { e.stopPropagation(); fetchHealth(); }} disabled={polling} title="Refresh now">↻</button>
      {/if}
    </div>
  </button>

  {#if expanded}
  {#if loading}
    <div class="sys-loading">
      <div class="sys-skeleton">
        <div class="skel-bar"></div>
        <div class="skel-bar short"></div>
        <div class="skel-bar"></div>
        <div class="skel-bar short"></div>
        <div class="skel-bar"></div>
      </div>
    </div>
  {:else if error}
    <div class="sys-error">
      <span>⚠️ Health check failed</span>
      <button class="sys-retry" onclick={fetchHealth}>Retry</button>
    </div>
  {:else if health}
    <!-- Overall status bar -->
    <div class="sys-overall" class:healthy={health.overall === 'healthy'} class:degraded={health.overall !== 'healthy'}>
      <span class="sys-overall-dot">{health.overall === 'healthy' ? '🟢' : '🟡'}</span>
      <span class="sys-overall-text">
        {#if health.overall === 'healthy'}
          All systems operational
        {:else}
          {health.services.filter(s => s.status !== 'up').length} service{health.services.filter(s => s.status !== 'up').length > 1 ? 's' : ''} degraded
        {/if}
      </span>
    </div>

    <!-- Service cards -->
    <div class="sys-services">
      {#each health.services as svc}
        <div class="sys-svc" class:down={svc.status === 'down'} class:degraded={svc.status === 'degraded'}>
          <div class="sys-svc-main">
            <span class="sys-svc-dot">{dot(svc.status)}</span>
            <div class="sys-svc-info">
              <div class="sys-svc-name">{svcLabel(svc.name)}</div>
              <div class="sys-svc-desc">{svcDesc(svc.name)}</div>
            </div>
            <div class="sys-svc-right">
              {#if svc.latency_ms > 0}
                <span class="sys-svc-latency">{svc.latency_ms}ms</span>
              {/if}
              {#if canRestart(svc.name)}
                <button
                  class="sys-restart-btn"
                  class:restarting={restarting[svc.name]}
                  disabled={restarting[svc.name]}
                  onclick={() => restartService(svc.name)}
                  title="Restart {svcLabel(svc.name)}"
                >
                  {#if restarting[svc.name]}
                    ⟳
                  {:else}
                    ↻
                  {/if}
                </button>
              {/if}
            </div>
          </div>
          {#if restartMsg[svc.name]}
            <div class="sys-svc-msg" class:ok={restartMsg[svc.name].startsWith('✓')} class:fail={restartMsg[svc.name].startsWith('✗')}>
              {restartMsg[svc.name]}
            </div>
          {/if}
          {#if svc.detail && svc.detail !== 'this service'}
            <div class="sys-svc-detail">{svc.detail}</div>
          {/if}
        </div>
      {/each}
    </div>

    <!-- Gauges row -->
    <div class="sys-gauges">
      <div class="sys-gauge">
        <div class="sys-gauge-label">💾 Disk</div>
        <div class="sys-gauge-bar">
          <div class="sys-gauge-fill" style="width:{health.disk_percent}%; background:{diskColor(health.disk_percent)}"></div>
        </div>
        <div class="sys-gauge-val">{health.disk_percent}%</div>
      </div>
      <div class="sys-gauge">
        <div class="sys-gauge-label">📋 Queue</div>
        <div class="sys-gauge-counts">
          <span class="sys-cnt pending">{health.memory_pending}</span>
          <span class="sys-cnt-label">pending</span>
          <span class="sys-cnt approved">{health.memory_approved}</span>
          <span class="sys-cnt-label">approved</span>
        </div>
      </div>
    </div>
  {/if}
  {/if}
</div>

<style>
  .sys-panel {
    background: var(--card-bg, rgba(255,255,255,0.04));
    border: 1px solid var(--border, rgba(255,255,255,0.08));
    border-radius: 12px;
    padding: 16px;
    margin-top: 4px;
  }
  .sys-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    background: none;
    border: none;
    padding: 0;
    margin-bottom: 0;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  .sys-header:has(~ .sys-loading, ~ .sys-error, ~ .sys-overall) {
    margin-bottom: 12px;
  }
  .sys-title {
    display: flex;
    align-items: center;
    gap: 6px;
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary, #fff);
  }
  .sys-icon { font-size: 16px; }
  .sys-chevron {
    font-size: 11px;
    transition: transform 0.2s;
    color: var(--text-tertiary, #555);
  }
  .sys-chevron.open { transform: rotate(90deg); }
  .sys-dots {
    font-size: 8px;
    letter-spacing: 2px;
  }
  .sys-polling {
    animation: spin 1s linear infinite;
    font-size: 12px;
    opacity: 0.5;
  }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  .sys-meta {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 10px;
    color: var(--text-tertiary, #555);
  }
  .sys-refresh {
    width: 22px;
    height: 22px;
    border-radius: 5px;
    border: 1px solid var(--border, rgba(255,255,255,0.12));
    background: transparent;
    color: var(--text-secondary, #888);
    font-size: 12px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .sys-refresh:hover { background: var(--accent, #6366f1); color: #fff; border-color: var(--accent); }
  .sys-refresh:disabled { opacity: 0.3; cursor: default; }

  /* Skeleton loading */
  .sys-loading { padding: 8px 0; }
  .sys-skeleton { display: flex; flex-direction: column; gap: 8px; }
  .skel-bar {
    height: 32px;
    background: linear-gradient(90deg, var(--card-bg-hover, rgba(255,255,255,0.04)) 25%, rgba(255,255,255,0.08) 50%, var(--card-bg-hover, rgba(255,255,255,0.04)) 75%);
    background-size: 200% 100%;
    animation: shimmer 1.5s infinite;
    border-radius: 6px;
  }
  .skel-bar.short { width: 60%; }
  @keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

  .sys-error {
    text-align: center;
    padding: 20px;
    font-size: 12px;
    color: var(--text-secondary, #888);
    display: flex;
    align-items: center;
    gap: 8px;
    justify-content: center;
  }
  .sys-retry {
    background: var(--accent, #6366f1);
    color: #fff;
    border: none;
    border-radius: 6px;
    padding: 4px 10px;
    font-size: 11px;
    cursor: pointer;
  }

  .sys-overall {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    border-radius: 8px;
    font-size: 12px;
    font-weight: 500;
    margin-bottom: 10px;
  }
  .sys-overall.healthy { background: rgba(34, 197, 94, 0.08); color: var(--green, #22c55e); }
  .sys-overall.degraded { background: rgba(234, 179, 8, 0.08); color: var(--yellow, #eab308); }

  .sys-services {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 12px;
  }
  .sys-svc {
    background: var(--card-bg-hover, rgba(255,255,255,0.02));
    border: 1px solid transparent;
    border-radius: 8px;
    padding: 8px 10px;
    transition: border-color 0.3s, background 0.3s;
  }
  .sys-svc.down { border-color: rgba(239, 68, 68, 0.3); background: rgba(239, 68, 68, 0.05); }
  .sys-svc.degraded { border-color: rgba(234, 179, 8, 0.3); background: rgba(234, 179, 8, 0.05); }
  .sys-svc-main {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .sys-svc-dot { font-size: 10px; flex-shrink: 0; }
  .sys-svc-info { flex: 1; min-width: 0; }
  .sys-svc-name { font-size: 12px; font-weight: 500; color: var(--text-primary, #fff); }
  .sys-svc-desc { font-size: 10px; color: var(--text-tertiary, #555); }
  .sys-svc-right { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
  .sys-svc-latency { font-size: 10px; color: var(--text-tertiary, #555); min-width: 35px; text-align: right; }

  .sys-restart-btn {
    width: 26px;
    height: 26px;
    border-radius: 6px;
    border: 1px solid var(--border, rgba(255,255,255,0.12));
    background: transparent;
    color: var(--text-secondary, #888);
    font-size: 14px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.15s;
  }
  .sys-restart-btn:hover { background: var(--accent, #6366f1); color: #fff; border-color: var(--accent, #6366f1); }
  .sys-restart-btn.restarting { animation: spin 0.8s linear infinite; opacity: 0.5; pointer-events: none; }

  .sys-svc-msg {
    font-size: 10px;
    padding: 2px 0 0 26px;
    animation: fadeIn 0.2s;
  }
  .sys-svc-msg.ok { color: var(--green, #22c55e); }
  .sys-svc-msg.fail { color: var(--red, #ef4444); }
  @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }

  .sys-svc-detail {
    font-size: 10px;
    color: var(--text-tertiary, #555);
    padding: 2px 0 0 26px;
  }

  .sys-gauges {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding-top: 10px;
    border-top: 1px solid var(--border, rgba(255,255,255,0.08));
  }
  .sys-gauge {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .sys-gauge-label {
    font-size: 11px;
    color: var(--text-secondary, #888);
    min-width: 55px;
    white-space: nowrap;
  }
  .sys-gauge-bar {
    flex: 1;
    height: 6px;
    background: var(--card-bg-hover, rgba(255,255,255,0.06));
    border-radius: 3px;
    overflow: hidden;
  }
  .sys-gauge-fill {
    height: 100%;
    border-radius: 3px;
    transition: width 0.5s ease;
  }
  .sys-gauge-val {
    font-size: 11px;
    font-weight: 600;
    color: var(--text-primary, #fff);
    min-width: 30px;
    text-align: right;
  }
  .sys-gauge-counts {
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
  }
  .sys-cnt {
    font-weight: 700;
    font-size: 13px;
    font-variant-numeric: tabular-nums;
  }
  .sys-cnt.pending { color: var(--yellow, #eab308); }
  .sys-cnt.approved { color: var(--green, #22c55e); }
  .sys-cnt-label { font-size: 10px; color: var(--text-tertiary, #555); margin-right: 6px; }
</style>
