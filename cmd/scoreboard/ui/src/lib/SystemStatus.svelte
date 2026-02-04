<script>
  /**
   * SystemStatus — Admin-only system health panel for Settings page.
   * Shows service status with restart buttons and live polling counters.
   * Only visible to superadmin (is_admin + operator token).
   */
  let { token = '' } = $props();
  let health = $state(null);
  let loading = $state(true);
  let error = $state(false);
  let polling = $state(false);
  let pollTimer = $state(null);
  let restarting = $state({});  // { serviceName: true } while restart in progress
  let restartMsg = $state({});  // { serviceName: 'Restarted ✓' }
  let lastChecked = $state('');
  let pollCount = $state(0);

  async function fetchHealth() {
    polling = true;
    pollCount++;
    try {
      const resp = await fetch('/v1/system/health', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (resp.ok) {
        health = await resp.json();
        lastChecked = new Date().toLocaleTimeString();
        error = false;
      } else {
        error = true;
      }
    } catch {
      error = true;
    }
    loading = false;
    polling = false;
  }

  function startPolling() {
    fetchHealth();
    pollTimer = setInterval(fetchHealth, 15000); // 15s for admin
  }

  function stopPolling() {
    if (pollTimer) { clearInterval(pollTimer); pollTimer = null; }
  }

  $effect(() => {
    startPolling();
    return () => stopPolling();
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
        // Re-poll after restart settles
        setTimeout(fetchHealth, 5000);
        setTimeout(fetchHealth, 12000);
      } else {
        restartMsg = { ...restartMsg, [name]: `✗ ${data.error || 'Failed'}` };
      }
    } catch (e) {
      restartMsg = { ...restartMsg, [name]: `✗ ${e.message}` };
    }
    restarting = { ...restarting, [name]: false };
    // Clear message after 8s
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
    // Scoreboard can't restart itself
    return name !== 'scoreboard';
  }

  function diskColor(pct) {
    if (pct < 60) return 'var(--green, #22c55e)';
    if (pct < 80) return 'var(--yellow, #eab308)';
    return 'var(--red, #ef4444)';
  }
</script>

<div class="sys-panel">
  <div class="sys-header">
    <div class="sys-title">
      <span class="sys-icon">🖥️</span>
      <span>System Health</span>
      {#if polling}
        <span class="sys-polling">⟳</span>
      {/if}
    </div>
    <div class="sys-meta">
      {#if lastChecked}
        <span class="sys-time">{lastChecked}</span>
      {/if}
      <span class="sys-polls">#{pollCount}</span>
    </div>
  </div>

  {#if loading}
    <div class="sys-loading">Checking systems…</div>
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
        <div class="sys-gauge-label">📋 Memory Queue</div>
        <div class="sys-gauge-counts">
          <span class="sys-cnt pending">{health.memory_pending}</span>
          <span class="sys-cnt-label">pending</span>
          <span class="sys-cnt approved">{health.memory_approved}</span>
          <span class="sys-cnt-label">approved</span>
        </div>
      </div>
    </div>
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
  .sys-polling {
    animation: spin 1s linear infinite;
    font-size: 12px;
    opacity: 0.5;
  }
  @keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
  .sys-meta {
    display: flex;
    gap: 8px;
    font-size: 10px;
    color: var(--text-tertiary, #555);
  }
  .sys-polls { opacity: 0.4; }

  .sys-loading, .sys-error {
    text-align: center;
    padding: 20px;
    font-size: 12px;
    color: var(--text-secondary, #888);
  }
  .sys-error { display: flex; align-items: center; gap: 8px; justify-content: center; }
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
    transition: all 0.2s;
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
