<script>
  /**
   * SystemStatus — tiny status strip showing service health at a glance.
   * Sits at the top of the dashboard. Tappable to expand details.
   * Auto-refreshes every 60s.
   */
  let { token = '' } = $props();
  let health = $state(null);
  let expanded = $state(false);
  let loading = $state(true);
  let error = $state(false);

  async function fetchHealth() {
    try {
      const resp = await fetch('/v1/system/health', {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (resp.ok) {
        health = await resp.json();
        error = false;
      } else {
        error = true;
      }
    } catch {
      error = true;
    }
    loading = false;
  }

  // Initial fetch + auto-refresh
  $effect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 60000);
    return () => clearInterval(interval);
  });

  function dot(status) {
    if (status === 'up') return '🟢';
    if (status === 'degraded') return '🟡';
    return '🔴';
  }

  function overallIcon(overall) {
    if (overall === 'healthy') return '🟢';
    return '🟡';
  }

  function svcLabel(name) {
    const labels = {
      scoreboard: 'Scores',
      gateway: 'AI',
      mem0: 'Memory',
      letta: 'State',
      'memory-sync': 'Sync'
    };
    return labels[name] || name;
  }
</script>

{#if loading}
  <!-- silent loading, no flash -->
{:else if error}
  <button class="status-strip status-error" onclick={() => fetchHealth()}>
    <span class="status-dots">⚠️ Status unavailable</span>
    <span class="status-tap">tap to retry</span>
  </button>
{:else if health}
  <button class="status-strip" class:degraded={health.overall !== 'healthy'} onclick={() => expanded = !expanded}>
    <span class="status-dots">
      {#each health.services as svc}
        <span class="svc-dot" title="{svc.name}: {svc.status}">
          {dot(svc.status)}
        </span>
      {/each}
      <span class="status-label">
        {#if health.overall === 'healthy'}
          All systems go
        {:else}
          {health.services.filter(s => s.status !== 'up').map(s => svcLabel(s.name)).join(', ')} down
        {/if}
      </span>
    </span>
    <span class="status-chevron">{expanded ? '▲' : '▼'}</span>
  </button>

  {#if expanded}
    <div class="status-detail">
      <div class="svc-grid">
        {#each health.services as svc}
          <div class="svc-row">
            <span class="svc-icon">{dot(svc.status)}</span>
            <span class="svc-name">{svcLabel(svc.name)}</span>
            <span class="svc-latency">
              {#if svc.latency_ms > 0}{svc.latency_ms}ms{/if}
            </span>
            <span class="svc-detail">{svc.detail || ''}</span>
          </div>
        {/each}
      </div>
      <div class="status-meta">
        <span>💾 Disk: {health.disk_percent}%</span>
        <span>📋 {health.memory_pending} pending memories</span>
        <span>✅ {health.memory_approved} approved</span>
      </div>
    </div>
  {/if}
{/if}

<style>
  .status-strip {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding: 6px 16px;
    background: var(--card-bg, rgba(255,255,255,0.04));
    border: 1px solid var(--border, rgba(255,255,255,0.08));
    border-radius: 10px;
    cursor: pointer;
    font-size: 12px;
    color: var(--text-secondary, #888);
    margin-bottom: 8px;
    transition: all 0.2s;
  }
  .status-strip:hover {
    background: var(--card-bg-hover, rgba(255,255,255,0.06));
  }
  .status-strip.degraded {
    border-color: rgba(255, 180, 0, 0.3);
    background: rgba(255, 180, 0, 0.05);
  }
  .status-error {
    border-color: rgba(255, 80, 80, 0.3);
    background: rgba(255, 80, 80, 0.05);
  }
  .status-dots {
    display: flex;
    align-items: center;
    gap: 4px;
  }
  .svc-dot {
    font-size: 8px;
    line-height: 1;
  }
  .status-label {
    margin-left: 6px;
    font-size: 11px;
  }
  .status-chevron {
    font-size: 9px;
    opacity: 0.5;
  }
  .status-tap {
    font-size: 10px;
    opacity: 0.5;
  }
  .status-detail {
    background: var(--card-bg, rgba(255,255,255,0.04));
    border: 1px solid var(--border, rgba(255,255,255,0.08));
    border-radius: 10px;
    padding: 12px 16px;
    margin-bottom: 8px;
    font-size: 12px;
  }
  .svc-grid {
    display: flex;
    flex-direction: column;
    gap: 6px;
    margin-bottom: 10px;
  }
  .svc-row {
    display: grid;
    grid-template-columns: 20px 70px 50px 1fr;
    align-items: center;
    gap: 4px;
  }
  .svc-icon { font-size: 10px; }
  .svc-name { font-weight: 500; color: var(--text-primary, #fff); }
  .svc-latency { color: var(--text-tertiary, #555); font-size: 10px; text-align: right; }
  .svc-detail { color: var(--text-secondary, #888); font-size: 10px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .status-meta {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
    padding-top: 8px;
    border-top: 1px solid var(--border, rgba(255,255,255,0.08));
    font-size: 11px;
    color: var(--text-secondary, #888);
  }
</style>
