<script>
  import { onMount } from 'svelte';
  
  let interactions = $state([]);
  let loading = $state(true);
  let filter = $state({ mode: '', limit: 50 });
  let feedbackModal = $state(null);
  let feedbackText = $state('');
  let feedbackType = $state('good');
  
  const API = '';
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : '';
  
  onMount(() => loadInteractions());
  
  async function loadInteractions() {
    loading = true;
    try {
      const params = new URLSearchParams();
      if (filter.mode) params.set('mode', filter.mode);
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
  
  async function submitFeedback() {
    if (!feedbackModal) return;
    
    const payload = {
      interaction_id: feedbackModal.interaction_id,
      feedback_type: feedbackType,
      feedback_text: feedbackText
    };
    
    if (feedbackType === 'memory') {
      payload.memory_action = 'add';
    }
    
    try {
      await fetch(`${API}/v1/discord/feedback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        body: JSON.stringify(payload)
      });
      
      // Add feedback to local state
      const idx = interactions.findIndex(i => i.interaction_id === feedbackModal.interaction_id);
      if (idx >= 0) {
        interactions[idx].feedback = [...(interactions[idx].feedback || []), {
          feedback_type: feedbackType,
          feedback_text: feedbackText,
          created_at: new Date().toISOString()
        }];
      }
      
      feedbackModal = null;
      feedbackText = '';
      if (navigator.vibrate) navigator.vibrate(50);
    } catch (e) {
      console.error('Failed to submit feedback:', e);
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
  
  function formatTime(ts) {
    if (!ts) return '';
    const d = new Date(ts);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  
  function truncate(s, len = 100) {
    if (!s) return '';
    return s.length > len ? s.slice(0, len) + '...' : s;
  }
</script>

<div class="audit-container">
  <header class="audit-header">
    <h2>🎯 Discord Training</h2>
    <div class="filters">
      <select bind:value={filter.mode} onchange={loadInteractions}>
        <option value="">All Modes</option>
        <option value="sovereign">Sovereign</option>
        <option value="community">Community</option>
        <option value="guest">Guest</option>
      </select>
      <button onclick={loadInteractions} class="refresh-btn">🔄</button>
    </div>
  </header>
  
  {#if loading}
    <div class="loading">Loading interactions...</div>
  {:else if interactions.length === 0}
    <div class="empty">No Discord interactions yet. Start chatting with Wirebot!</div>
  {:else}
    <div class="interactions-list">
      {#each interactions as int}
        <div class="interaction-card">
          <div class="interaction-meta">
            <span class="mode-badge" style="background: {getModeColor(int.mode)}">{int.mode}</span>
            <span class="channel">#{int.channel_name || 'unknown'}</span>
            <span class="user">@{int.user_name}</span>
            <span class="time">{formatTime(int.created_at)}</span>
          </div>
          
          <div class="message user-msg">
            <strong>User:</strong> {int.user_message}
          </div>
          
          <div class="message bot-msg">
            <strong>Wirebot:</strong> {truncate(int.bot_response, 300)}
          </div>
          
          {#if int.tools_used?.length > 0}
            <div class="tools">
              Tools: {int.tools_used.join(', ')}
            </div>
          {/if}
          
          <div class="feedback-section">
            {#if int.feedback?.length > 0}
              <div class="existing-feedback">
                {#each int.feedback as fb}
                  <span class="fb-badge {fb.feedback_type}">
                    {fb.feedback_type === 'good' ? '👍' : fb.feedback_type === 'bad' ? '👎' : fb.feedback_type === 'memory' ? '🧠' : '📝'}
                    {fb.feedback_text ? `: ${truncate(fb.feedback_text, 30)}` : ''}
                  </span>
                {/each}
              </div>
            {/if}
            
            <div class="feedback-actions">
              <button onclick={() => { feedbackModal = int; feedbackType = 'good'; }} class="fb-btn good">👍</button>
              <button onclick={() => { feedbackModal = int; feedbackType = 'bad'; }} class="fb-btn bad">👎</button>
              <button onclick={() => { feedbackModal = int; feedbackType = 'note'; }} class="fb-btn note">📝</button>
              <button onclick={() => { feedbackModal = int; feedbackType = 'memory'; }} class="fb-btn memory">🧠</button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if feedbackModal}
  <div class="modal-overlay" onclick={() => feedbackModal = null}>
    <div class="modal" onclick={(e) => e.stopPropagation()}>
      <h3>
        {feedbackType === 'good' ? '👍 Mark as Good' : 
         feedbackType === 'bad' ? '👎 Needs Improvement' :
         feedbackType === 'memory' ? '🧠 Add to Memory' : '📝 Add Note'}
      </h3>
      
      <div class="modal-context">
        <strong>User asked:</strong> {truncate(feedbackModal.user_message, 100)}
      </div>
      
      <textarea 
        bind:value={feedbackText}
        placeholder={feedbackType === 'memory' ? 'What should Wirebot remember/learn from this?' :
                    feedbackType === 'bad' ? 'What was wrong? How should it respond?' :
                    'Add your note...'}
        rows="3"
      ></textarea>
      
      <div class="modal-actions">
        <button onclick={() => feedbackModal = null} class="cancel-btn">Cancel</button>
        <button onclick={submitFeedback} class="submit-btn">Submit</button>
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
  
  .audit-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  
  .audit-header h2 {
    font-size: 1.25rem;
    margin: 0;
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
  
  .loading, .empty {
    text-align: center;
    padding: 2rem;
    color: var(--text-secondary);
  }
  
  .interactions-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  
  .interaction-card {
    background: var(--bg-card);
    border-radius: 12px;
    padding: 1rem;
    border: 1px solid var(--border-light);
  }
  
  .interaction-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    font-size: 0.8rem;
  }
  
  .mode-badge {
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    font-weight: 600;
    text-transform: uppercase;
    font-size: 0.7rem;
  }
  
  .channel, .user, .time {
    color: var(--text-secondary);
  }
  
  .message {
    margin: 0.5rem 0;
    padding: 0.5rem;
    border-radius: 8px;
    font-size: 0.9rem;
    line-height: 1.4;
  }
  
  .user-msg {
    background: var(--bg-elevated);
  }
  
  .bot-msg {
    background: var(--success-bg);
  }
  
  .tools {
    font-size: 0.75rem;
    color: var(--text-secondary);
    margin-top: 0.5rem;
  }
  
  .feedback-section {
    margin-top: 0.75rem;
    padding-top: 0.75rem;
    border-top: 1px solid #333;
  }
  
  .existing-feedback {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  
  .fb-badge {
    font-size: 0.75rem;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    background: var(--bg-elevated);
  }
  
  .fb-badge.good { background: #166534; }
  .fb-badge.bad { background: #7f1d1d; }
  .fb-badge.memory { background: #4338ca; }
  
  .feedback-actions {
    display: flex;
    gap: 0.5rem;
  }
  
  .fb-btn {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-elevated);
    cursor: pointer;
    font-size: 1.1rem;
    transition: all 0.2s;
  }
  
  .fb-btn:hover { transform: scale(1.1); }
  .fb-btn.good:hover { background: #166534; }
  .fb-btn.bad:hover { background: #7f1d1d; }
  .fb-btn.memory:hover { background: #4338ca; }
  .fb-btn.note:hover { background: #854d0e; }
  
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: var(--bg-overlay);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 1rem;
  }
  
  .modal {
    background: var(--bg-card);
    border-radius: 16px;
    padding: 1.5rem;
    max-width: 400px;
    width: 100%;
    border: 1px solid var(--border-light);
  }
  
  .modal h3 {
    margin: 0 0 1rem 0;
  }
  
  .modal-context {
    background: var(--bg-elevated);
    padding: 0.75rem;
    border-radius: 8px;
    margin-bottom: 1rem;
    font-size: 0.85rem;
  }
  
  .modal textarea {
    width: 100%;
    padding: 0.75rem;
    border-radius: 8px;
    border: 1px solid var(--border-light);
    background: var(--bg-elevated);
    color: var(--text);
    resize: vertical;
    font-family: inherit;
  }
  
  .modal-actions {
    display: flex;
    gap: 0.5rem;
    margin-top: 1rem;
    justify-content: flex-end;
  }
  
  .cancel-btn, .submit-btn {
    padding: 0.5rem 1rem;
    border-radius: 8px;
    border: none;
    cursor: pointer;
  }
  
  .cancel-btn {
    background: var(--bg-elevated);
    color: var(--text);
  }
  
  .submit-btn {
    background: #3b82f6;
    color: var(--text);
  }
</style>
