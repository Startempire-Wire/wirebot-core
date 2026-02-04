<script>
  import { onMount } from 'svelte';
  
  let interactions = $state([]);
  let stats = $state({ total: 0, good: 0, bad: 0, memory: 0, accuracy: null });
  let loading = $state(true);
  let filter = $state({ mode: '', channel: '', limit: 50 });
  let feedbackModal = $state(null);
  let feedbackText = $state('');
  let feedbackType = $state('good');
  let submitting = $state(false);
  let toast = $state('');
  
  const API = '';
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : '';
  
  onMount(() => {
    loadInteractions();
    loadStats();
  });
  
  async function loadInteractions() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (filter.mode) params.set('mode', filter.mode);
      if (filter.channel) params.set('channel', filter.channel);
      params.set('limit', filter.limit);
      
      const res = await fetch(`${API}/v1/discord/interactions?${params}`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      const data = await res.json();
      interactions = data.interactions || [];
    } catch (e) {
      console.error('Failed to load interactions:', e);
    }
    loading = false;
  }
  
  async function loadStats() {
    try {
      const res = await fetch(`${API}/v1/training/stats`, {
        headers: { 'Authorization': `Bearer ${token}` }
      });
      if (res.ok) {
        stats = await res.json();
      }
    } catch { /* stats are optional enhancement */ }
  }
  
  async function submitFeedback() {
    if (!feedbackModal || submitting) return;
    submitting = true;
    
    const payload = {
      interaction_id: feedbackModal.interaction_id,
      feedback_type: feedbackType,
      feedback_text: feedbackText
    };
    
    if (feedbackType === 'memory') {
      payload.memory_action = 'add';
    }
    
    try {
      const res = await fetch(`${API}/v1/discord/feedback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(payload)
      });
      const result = await res.json();
      
      if (result.ok) {
        // Update local state
        const idx = interactions.findIndex(i => i.interaction_id === feedbackModal.interaction_id);
        if (idx >= 0) {
          interactions[idx].feedback = [...(interactions[idx].feedback || []), {
            feedback_type: feedbackType,
            feedback_text: feedbackText,
            created_at: new Date().toISOString()
          }];
        }
        
        // Show pipeline confirmation
        const pipelineMsg = result.pipeline_actions || [];
        if (pipelineMsg.length > 0) {
          toast = `✓ ${pipelineMsg.join(' • ')}`;
        } else {
          toast = feedbackType === 'good' ? '✓ Positive pattern saved' :
                  feedbackType === 'bad' ? '✓ Correction stored' :
                  feedbackType === 'memory' ? '✓ Memory queued for review' :
                  '✓ Note saved';
        }
        setTimeout(() => toast = '', 4000);
        
        feedbackModal = null;
        feedbackText = '';
        loadStats();
        if (navigator.vibrate) navigator.vibrate(50);
      }
    } catch (e) {
      console.error('Failed to submit feedback:', e);
      toast = '✗ Failed to submit';
      setTimeout(() => toast = '', 3000);
    }
    submitting = false;
  }
  
  function quickFeedback(interaction, type) {
    if (type === 'good' || type === 'bad') {
      // Quick submit without modal for simple thumbs
      feedbackModal = interaction;
      feedbackType = type;
      feedbackText = '';
      submitFeedback();
    } else {
      feedbackModal = interaction;
      feedbackType = type;
      feedbackText = '';
    }
  }
  
  function getModeColor(mode) {
    switch (mode) {
      case 'sovereign': return '#10b981';
      case 'community': return '#3b82f6';
      case 'guest': return '#8b5cf6';
      default: return '#6b7280';
    }
  }
  
  function getChannelIcon(ch) {
    if (!ch) return '💬';
    if (ch.includes('discord')) return '🎮';
    if (ch.includes('web') || ch.includes('api')) return '🌐';
    if (ch.includes('sms') || ch.includes('whatsapp')) return '📱';
    return '💬';
  }
  
  function formatTime(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    const now = new Date();
    const diffH = (now - d) / 3600000;
    if (diffH < 1) return `${Math.floor(diffH * 60)}m ago`;
    if (diffH < 24) return `${Math.floor(diffH)}h ago`;
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  
  function hasFeedback(interaction, type) {
    return (interaction.feedback || []).some(f => f.feedback_type === type);
  }
</script>

<div class="audit-container">
  <!-- Stats bar -->
  {#if stats.total > 0}
    <div class="stats-bar">
      <div class="stat">
        <span class="stat-num">{stats.total}</span>
        <span class="stat-label">Reviewed</span>
      </div>
      <div class="stat good">
        <span class="stat-num">{stats.good}</span>
        <span class="stat-label">👍 Good</span>
      </div>
      <div class="stat bad">
        <span class="stat-num">{stats.bad}</span>
        <span class="stat-label">👎 Fix</span>
      </div>
      <div class="stat mem">
        <span class="stat-num">{stats.memory}</span>
        <span class="stat-label">🧠 Memory</span>
      </div>
      {#if stats.accuracy !== null && stats.accuracy !== undefined}
        <div class="stat accent">
          <span class="stat-num">{stats.accuracy}%</span>
          <span class="stat-label">Accuracy</span>
        </div>
      {/if}
    </div>
  {/if}

  <header class="audit-header">
    <h2>🧠 Training Lab</h2>
    <p class="subtitle">Review Wirebot's responses. Your feedback makes it smarter.</p>
    <div class="filters">
      <select bind:value={filter.mode} onchange={loadInteractions}>
        <option value="">All Modes</option>
        <option value="sovereign">Sovereign</option>
        <option value="community">Community</option>
      </select>
      <button onclick={loadInteractions} class="refresh-btn">🔄</button>
    </div>
  </header>

  <!-- Pipeline explainer -->
  <div class="pipeline-info">
    <div class="pipe-step">👍 Good → Pattern saved to TRAINING.md</div>
    <div class="pipe-step">👎 Bad → Correction → Mem0 + TRAINING.md</div>
    <div class="pipe-step">🧠 Memory → Mem0 + Letta (via approval queue)</div>
    <div class="pipe-step">📝 Note → Stored for context</div>
  </div>
  
  {#if loading}
    <div class="loading">Loading interactions...</div>
  {:else if interactions.length === 0}
    <div class="empty">
      <div class="empty-icon">🎯</div>
      <p>No interactions yet.</p>
      <p class="empty-sub">Chat with Wirebot on Discord, web, or API — interactions appear here automatically.</p>
    </div>
  {:else}
    <div class="interactions-list">
      {#each interactions as int}
        <div class="interaction-card" class:reviewed={int.feedback?.length > 0}>
          <div class="interaction-meta">
            <span class="mode-badge" style="background: {getModeColor(int.mode)}">{int.mode}</span>
            <span class="channel">{getChannelIcon(int.channel_name)} #{int.channel_name || 'api'}</span>
            <span class="user">@{int.user_name}</span>
            <span class="time">{formatTime(int.created_at)}</span>
            {#if int.response_time_ms > 0}
              <span class="latency">{(int.response_time_ms / 1000).toFixed(1)}s</span>
            {/if}
          </div>
          
          <div class="message user-msg">
            <div class="msg-label">👤 User</div>
            {int.user_message}
          </div>
          
          <div class="message bot-msg">
            <div class="msg-label">🤖 Wirebot</div>
            {int.bot_response}
          </div>
          
          {#if int.tools_used?.length > 0}
            <div class="tools">
              🔧 {int.tools_used.join(', ')}
            </div>
          {/if}
          
          <div class="feedback-section">
            {#if int.feedback?.length > 0}
              <div class="existing-feedback">
                {#each int.feedback as fb}
                  <span class="fb-badge {fb.feedback_type}">
                    {fb.feedback_type === 'good' ? '👍' : fb.feedback_type === 'bad' ? '👎' : fb.feedback_type === 'memory' ? '🧠' : '📝'}
                    {fb.feedback_text ? `: ${fb.feedback_text.slice(0, 40)}${fb.feedback_text.length > 40 ? '...' : ''}` : ''}
                  </span>
                {/each}
              </div>
            {/if}
            
            <div class="feedback-actions">
              <button 
                onclick={() => quickFeedback(int, 'good')} 
                class="fb-btn good" 
                class:active={hasFeedback(int, 'good')}
                title="Good response — save as positive pattern"
              >👍</button>
              <button 
                onclick={() => quickFeedback(int, 'bad')} 
                class="fb-btn bad"
                class:active={hasFeedback(int, 'bad')}
                title="Bad response — provide correction"
              >👎</button>
              <button 
                onclick={() => { feedbackModal = int; feedbackType = 'memory'; feedbackText = ''; }}
                class="fb-btn memory"
                class:active={hasFeedback(int, 'memory')}
                title="Teach Wirebot something from this interaction"
              >🧠</button>
              <button 
                onclick={() => { feedbackModal = int; feedbackType = 'note'; feedbackText = ''; }}
                class="fb-btn note"
                title="Add a note"
              >📝</button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if toast}
  <div class="toast">{toast}</div>
{/if}

{#if feedbackModal}
  <div class="modal-overlay" onclick={() => feedbackModal = null}>
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h3>
        {feedbackType === 'good' ? '👍 What was good?' : 
         feedbackType === 'bad' ? '👎 What should change?' :
         feedbackType === 'memory' ? '🧠 What should Wirebot learn?' : '📝 Add Note'}
      </h3>
      
      <div class="modal-context">
        <div class="modal-q"><strong>User:</strong> {feedbackModal.user_message?.slice(0, 150)}</div>
        <div class="modal-a"><strong>Wirebot:</strong> {feedbackModal.bot_response?.slice(0, 200)}</div>
      </div>

      <!-- Pipeline preview -->
      <div class="pipeline-preview">
        {#if feedbackType === 'good'}
          <span>→ Pattern saved to TRAINING.md (Wirebot reads this every conversation)</span>
        {:else if feedbackType === 'bad'}
          <span>→ Correction sent to Mem0 + TRAINING.md + review queue</span>
        {:else if feedbackType === 'memory'}
          <span>→ Memory sent to Mem0 + queued for Letta state update</span>
        {:else}
          <span>→ Note stored for future reference</span>
        {/if}
      </div>
      
      <textarea 
        bind:value={feedbackText}
        placeholder={feedbackType === 'good' ? 'What pattern should Wirebot repeat? (optional — just 👍 is fine)' :
                    feedbackType === 'bad' ? 'What was wrong? How should it respond instead?' :
                    feedbackType === 'memory' ? 'What fact or preference should Wirebot remember?' :
                    'Your note...'}
        rows="3"
      ></textarea>
      
      <div class="modal-actions">
        <button onclick={() => feedbackModal = null} class="cancel-btn">Cancel</button>
        <button onclick={submitFeedback} class="submit-btn" disabled={submitting}>
          {submitting ? 'Saving...' : 'Submit'}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .audit-container {
    padding: 1rem;
    max-width: 100%;
    overflow-x: hidden;
  }
  
  .stats-bar {
    display: flex;
    gap: 0.75rem;
    margin-bottom: 1rem;
    padding: 0.75rem;
    background: var(--bg-card);
    border-radius: 12px;
    border: 1px solid var(--border-light);
    overflow-x: auto;
  }
  .stat {
    text-align: center;
    min-width: 50px;
  }
  .stat-num {
    display: block;
    font-size: 1.4rem;
    font-weight: 700;
  }
  .stat-label {
    font-size: 0.7rem;
    color: var(--text-secondary);
  }
  .stat.good .stat-num { color: #10b981; }
  .stat.bad .stat-num { color: #ef4444; }
  .stat.mem .stat-num { color: #8b5cf6; }
  .stat.accent .stat-num { color: #f59e0b; }
  
  .audit-header {
    margin-bottom: 0.75rem;
  }
  .audit-header h2 {
    font-size: 1.25rem;
    margin: 0;
  }
  .subtitle {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin: 0.25rem 0 0.75rem 0;
  }
  .filters {
    display: flex;
    gap: 0.5rem;
  }
  .filters select {
    padding: 0.5rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-card);
    color: var(--text);
  }
  .refresh-btn {
    padding: 0.5rem 0.75rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-card);
    cursor: pointer;
  }
  
  .pipeline-info {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 1rem;
    padding: 0.6rem;
    background: var(--bg-elevated);
    border-radius: 8px;
    font-size: 0.7rem;
    color: var(--text-secondary);
  }
  .pipe-step {
    white-space: nowrap;
  }
  .pipe-step::after { content: '  |'; color: var(--border-light); }
  .pipe-step:last-child::after { content: ''; }
  
  .loading, .empty {
    text-align: center;
    padding: 2rem;
    color: var(--text-secondary);
  }
  .empty-icon { font-size: 2.5rem; margin-bottom: 0.5rem; }
  .empty-sub { font-size: 0.8rem; margin-top: 0.5rem; }
  
  .interactions-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  
  .interaction-card {
    background: var(--bg-card);
    border-radius: 12px;
    padding: 0.75rem;
    border: 1px solid var(--border-light);
    transition: border-color 0.2s;
  }
  .interaction-card.reviewed {
    border-left: 3px solid #10b981;
  }
  
  .interaction-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
    font-size: 0.75rem;
    align-items: center;
  }
  .mode-badge {
    padding: 0.15rem 0.4rem;
    border-radius: 4px;
    font-weight: 600;
    text-transform: uppercase;
    font-size: 0.65rem;
    color: white;
  }
  .channel, .user, .time, .latency {
    color: var(--text-secondary);
  }
  .latency {
    background: var(--bg-elevated);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-size: 0.65rem;
  }
  
  .message {
    margin: 0.4rem 0;
    padding: 0.5rem;
    border-radius: 8px;
    font-size: 0.85rem;
    line-height: 1.4;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .msg-label {
    font-size: 0.7rem;
    font-weight: 600;
    margin-bottom: 0.25rem;
    color: var(--text-secondary);
  }
  .user-msg { background: var(--bg-elevated); }
  .bot-msg { background: color-mix(in srgb, var(--bg-elevated) 80%, #10b981 20%); }
  
  .tools {
    font-size: 0.7rem;
    color: var(--text-secondary);
    margin: 0.25rem 0;
    padding: 0.25rem 0.5rem;
    background: var(--bg-elevated);
    border-radius: 4px;
    display: inline-block;
  }
  
  .feedback-section {
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid var(--border-light);
  }
  .existing-feedback {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    margin-bottom: 0.5rem;
  }
  .fb-badge {
    font-size: 0.7rem;
    padding: 0.2rem 0.4rem;
    border-radius: 4px;
    background: var(--bg-elevated);
  }
  .fb-badge.good { background: #166534; color: #bbf7d0; }
  .fb-badge.bad { background: #7f1d1d; color: #fecaca; }
  .fb-badge.memory { background: #4338ca; color: #c4b5fd; }
  
  .feedback-actions {
    display: flex;
    gap: 0.4rem;
  }
  .fb-btn {
    padding: 0.4rem 0.75rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-elevated);
    cursor: pointer;
    font-size: 1rem;
    transition: all 0.15s;
  }
  .fb-btn:hover { transform: scale(1.1); }
  .fb-btn.good:hover, .fb-btn.good.active { background: #166534; }
  .fb-btn.bad:hover, .fb-btn.bad.active { background: #7f1d1d; }
  .fb-btn.memory:hover, .fb-btn.memory.active { background: #4338ca; }
  .fb-btn.note:hover { background: #854d0e; }
  
  .toast {
    position: fixed;
    bottom: 1rem;
    left: 50%;
    transform: translateX(-50%);
    background: #166534;
    color: white;
    padding: 0.6rem 1.2rem;
    border-radius: 8px;
    font-size: 0.85rem;
    z-index: 2000;
    animation: slideUp 0.3s ease;
  }
  @keyframes slideUp {
    from { transform: translateX(-50%) translateY(1rem); opacity: 0; }
    to { transform: translateX(-50%) translateY(0); opacity: 1; }
  }
  
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1rem;
  }
  .modal {
    background: var(--bg-card);
    border-radius: 16px;
    padding: 1.25rem;
    max-width: 420px;
    width: 100%;
    border: 1px solid var(--border-light);
  }
  .modal h3 { margin: 0 0 0.75rem 0; font-size: 1.1rem; }
  .modal-context {
    background: var(--bg-elevated);
    padding: 0.6rem;
    border-radius: 8px;
    margin-bottom: 0.75rem;
    font-size: 0.8rem;
    line-height: 1.3;
  }
  .modal-q, .modal-a { margin: 0.25rem 0; }
  .pipeline-preview {
    font-size: 0.75rem;
    color: #10b981;
    margin-bottom: 0.5rem;
    padding: 0.3rem 0.5rem;
    background: color-mix(in srgb, var(--bg-elevated) 80%, #10b981 20%);
    border-radius: 4px;
  }
  .modal textarea {
    width: 100%;
    padding: 0.6rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-elevated);
    color: var(--text);
    resize: vertical;
    font-family: inherit;
    font-size: 0.85rem;
    box-sizing: border-box;
  }
  .modal-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 0.75rem;
    justify-content: flex-end;
  }
  .cancel-btn, .submit-btn {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    border: none;
    cursor: pointer;
    font-size: 0.85rem;
  }
  .cancel-btn { background: var(--bg-elevated); color: var(--text); }
  .submit-btn { background: #3b82f6; color: white; }
  .submit-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
