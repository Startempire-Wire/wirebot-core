<script>
  /**
   * Dashboard — Figma "Home Overview" brought to life
   * 
   * Mobile-first business operating dashboard:
   * - Welcome greeting
   * - Score summary (compact)
   * - Business setup progress + next task
   * - Stage selector (Idea / Launch / Growth)
   * - Daily standup tasks
   * - Network growth partners
   * - AI suggestions
   * - Ask Wirebot input
   */
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  let { data = null, user = null, token = '' } = $props();

  let checklist = $state(null);
  let dailyTasks = $state([]);
  let setupTasks = $state([]);
  let nextTask = $state(null);
  let stage = $state('launch');
  let askInput = $state('');
  let chatResponse = $state('');
  let chatLoading = $state(false);
  let chatSessionId = $state(null);
  let drift = $state(null);
  let suggestions = $state([]);
  let loading = $state(true);

  const API = '';

  async function authFetch(path) {
    const res = await fetch(`${API}${path}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    });
    if (!res.ok) return null;
    return res.json();
  }

  $effect(() => {
    if (token) loadAll();
  });

  async function loadAll() {
    loading = true;
    try {
      // Fetch checklist
      const cl = await authFetch('/v1/checklist?action=summary');
      if (cl) {
        checklist = cl;
        nextTask = cl.next_task || null;
        stage = cl.stage || 'launch';
      }

      // Fetch daily tasks
      const dt = await authFetch('/v1/checklist?action=daily');
      if (dt?.tasks) dailyTasks = dt.tasks.slice(0, 5);

      // Fetch setup tasks for current stage
      const st = await authFetch(`/v1/checklist?action=list&stage=${stage}`);
      if (st?.tasks) setupTasks = st.tasks.filter(t => !t.completed).slice(0, 4);

      // Fetch drift
      const dr = await authFetch('/v1/pairing/neural-drift');
      if (dr?.drift) drift = dr.drift;

      // Generate suggestions based on current context
      buildSuggestions();
    } catch (e) {
      console.error('Dashboard load:', e);
    }
    loading = false;
  }

  function buildSuggestions() {
    const s = [];
    if (data?.score?.execution_score < 30) {
      s.push({ icon: '🚀', text: 'Ship something today — even small wins count', action: 'ship' });
    }
    if (data?.score?.revenue_score < 10) {
      s.push({ icon: '💰', text: 'Connect Stripe or FreshBooks to track revenue', action: 'settings' });
    }
    if (!drift || drift.score < 50) {
      s.push({ icon: '🤝', text: 'Start your daily Neural Handshake', action: 'handshake' });
    }
    if (checklist && checklist.percent < 50) {
      s.push({ icon: '📋', text: `Complete business setup (${checklist.percent}% done)`, action: 'checklist' });
    }
    if (data?.score?.intent === '') {
      s.push({ icon: '🎯', text: 'Set your shipping intent for today', action: 'intent' });
    }
    // Always show at least 3
    if (s.length < 3) {
      s.push({ icon: '⚡', text: 'Keep building — you\'re on a streak!', action: 'score' });
    }
    suggestions = s.slice(0, 3);
  }

  async function askWirebot() {
    if (!askInput.trim() || chatLoading) return;
    chatLoading = true;
    chatResponse = '';
    try {
      const res = await fetch(`${API}/v1/chat`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          message: askInput,
          session_id: chatSessionId || undefined
        })
      });
      const d = await res.json();
      chatResponse = d.response || d.content || d.error || 'No response';
      if (d.session_id) chatSessionId = d.session_id;
    } catch (e) {
      chatResponse = 'Error connecting to Wirebot';
    }
    askInput = '';
    chatLoading = false;
  }

  async function completeTask(id) {
    await fetch(`${API}/v1/checklist?action=complete&id=${id}`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${token}` }
    });
    loadAll();
  }

  function changeStage(s) {
    stage = s;
    loadAll();
  }

  function handleSuggestion(action) {
    if (action === 'score') dispatch('nav', 'score');
    else if (action === 'settings') dispatch('nav', 'settings');
    else if (action === 'ship') dispatch('openFab');
    else if (action === 'handshake') doHandshake();
    else if (action === 'intent') dispatch('nav', 'score');
    else if (action === 'checklist') { /* scroll to checklist */ }
  }

  async function doHandshake() {
    try {
      const d = await fetch(`${API}/v1/pairing/handshake`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` }
      }).then(r => r.json());
      if (d.drift_score) drift = { ...drift, score: d.drift_score, signal: d.drift_signal, handshake_streak: d.handshake_streak };
    } catch(e) {}
  }

  function keydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); askWirebot(); }
  }

  function driftColor(signal) {
    return { deep_sync: '#00ff64', in_drift: '#4a9eff', drifting: '#ffc800', weak: '#ff9500', disconnected: '#ff3232' }[signal] || '#666';
  }
  function driftLabel(signal) {
    return { deep_sync: 'DEEP SYNC', in_drift: 'IN DRIFT', drifting: 'DRIFTING', weak: 'WEAK', disconnected: 'OFFLINE' }[signal] || '?';
  }
  function signalColor(score) {
    return score >= 60 ? '#00ff64' : score >= 30 ? '#ffc800' : '#ff3232';
  }
</script>

<div class="dashboard">
  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else}
    <!-- Welcome -->
    <div class="welcome">
      <div>
        <h1>Welcome, {user?.display_name || 'Operator'}!</h1>
        {#if data?.score}
          <div class="welcome-score" style="color: {signalColor(data.score.execution_score)}">
            ⚡ {data.score.execution_score}/100
            {#if data.streak?.current > 0}
              <span class="streak-badge">🔥 {data.streak.current}</span>
            {/if}
          </div>
        {/if}
      </div>
      {#if drift}
        <div class="drift-pill" style="background: {driftColor(drift.signal)}20; border-color: {driftColor(drift.signal)}40">
          <span style="color: {driftColor(drift.signal)}">🧠 {drift.score}%</span>
        </div>
      {/if}
    </div>

    <!-- First-time welcome for new users -->
    {#if !data?.score && !checklist}
      <div class="card welcome-card">
        <div class="wc-icon">🚀</div>
        <h2 class="wc-title">Your Business Dashboard</h2>
        <p class="wc-desc">Wirebot is your AI operating partner. Track execution, ship work, and build your business — all from right here.</p>
        <div class="wc-steps">
          <div class="wc-step"><span class="wc-num">1</span> Complete your pairing assessment</div>
          <div class="wc-step"><span class="wc-num">2</span> Set your first shipping intent</div>
          <div class="wc-step"><span class="wc-num">3</span> Ship something and log it</div>
        </div>
        <button class="wc-btn" onclick={() => dispatch('nav', 'score')}>Get Started →</button>
      </div>
    {/if}

    <!-- Business Setup Progress -->
    {#if checklist}
      <div class="card setup-card">
        <div class="card-header">
          <span class="card-label">BUSINESS SETUP TASKS — {checklist.percent || 0}%</span>
          <span class="card-badge">{checklist.completed || 0}/{checklist.total || 0}</span>
        </div>
        <div class="progress-track">
          <div class="progress-fill" style="width: {checklist.percent || 0}%"></div>
        </div>
        {#if nextTask}
          <div class="next-task">
            <span class="next-label">NEXT TASK:</span>
            <span class="next-title">{nextTask.title || nextTask}</span>
            <button class="next-go" onclick={() => nextTask?.id && completeTask(nextTask.id)}>✓</button>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Stage Selector -->
    <div class="stage-row">
      {#each ['idea', 'launch', 'growth'] as s}
        <button
          class="stage-pill {stage === s ? 'active' : ''}"
          onclick={() => changeStage(s)}
        >
          {s === 'idea' ? '💡' : s === 'launch' ? '🚀' : '📈'} {s.charAt(0).toUpperCase() + s.slice(1)}
        </button>
      {/each}
    </div>

    <!-- Daily Stand Up Tasks -->
    {#if dailyTasks.length > 0}
      <div class="card">
        <div class="card-label">DAILY STAND UP TASKS</div>
        <div class="task-list">
          {#each dailyTasks as task}
            <div class="task-item">
              <button class="task-check" onclick={() => task.id && completeTask(task.id)}>
                {task.completed ? '☑' : '☐'}
              </button>
              <span class="task-title {task.completed ? 'done' : ''}">{task.title}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Business Setup Tasks -->
    {#if setupTasks.length > 0}
      <div class="card">
        <div class="card-label">BUSINESS SET UP TASKS</div>
        <div class="task-list">
          {#each setupTasks as task}
            <div class="task-item">
              <button class="task-check" onclick={() => task.id && completeTask(task.id)}>☐</button>
              <span class="task-title">{task.title}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <!-- AI Suggestions -->
    {#if suggestions.length > 0}
      <div class="suggestions">
        <div class="suggestions-scroll">
          {#each suggestions as sug}
            <button class="suggestion-card" onclick={() => handleSuggestion(sug.action)}>
              <span class="sug-icon">{sug.icon}</span>
              <span class="sug-text">{sug.text}</span>
            </button>
          {/each}
        </div>
      </div>
    {/if}

    <!-- Ask Wirebot -->
    <div class="ask-bar">
      {#if chatResponse}
        <div class="chat-response">
          <div class="chat-label">⚡ Wirebot</div>
          <div class="chat-text">{chatResponse}</div>
        </div>
      {/if}
      <div class="ask-input-wrap">
        <input
          type="text"
          bind:value={askInput}
          onkeydown={keydown}
          placeholder="Ask Wire Bot A Question..."
          disabled={chatLoading}
        />
        <button onclick={askWirebot} disabled={chatLoading || !askInput.trim()}>
          {chatLoading ? '⏳' : '⚡'}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .dashboard {
    padding: 16px 16px 120px;
    max-width: 480px;
    margin: 0 auto;
  }
  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 60vh;
  }
  .spinner {
    width: 32px; height: 32px;
    border: 3px solid #333;
    border-top-color: #7c7cff;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* Welcome */
  .welcome {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 16px;
  }
  .welcome h1 {
    font-size: 22px;
    font-weight: 700;
    color: #f0f0f0;
    margin: 0;
  }
  .welcome-score {
    font-size: 14px;
    font-weight: 600;
    margin-top: 4px;
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .streak-badge {
    font-size: 12px;
    background: #ff440020;
    border: 1px solid #ff440040;
    padding: 2px 6px;
    border-radius: 8px;
  }
  .drift-pill {
    padding: 6px 10px;
    border-radius: 12px;
    border: 1px solid;
    font-size: 12px;
    font-weight: 600;
    white-space: nowrap;
  }

  /* Cards */
  .card {
    background: #16161e;
    border: 1px solid #1e1e30;
    border-radius: 12px;
    padding: 14px;
    margin-bottom: 12px;
  }
  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }
  .card-label {
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: #888;
    margin-bottom: 8px;
  }
  .card-badge {
    font-size: 11px;
    background: #7c7cff20;
    color: #7c7cff;
    padding: 2px 8px;
    border-radius: 8px;
    font-weight: 600;
  }
  .setup-card .card-label { margin-bottom: 0; }

  /* Progress */
  .progress-track {
    width: 100%;
    height: 6px;
    background: #1e1e30;
    border-radius: 3px;
    overflow: hidden;
    margin: 8px 0;
  }
  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #7c7cff, #a78bfa);
    border-radius: 3px;
    transition: width 0.6s ease;
  }

  /* Next Task */
  .next-task {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-top: 8px;
  }
  .next-label {
    font-size: 10px;
    font-weight: 700;
    color: #7c7cff;
    letter-spacing: 0.05em;
    white-space: nowrap;
  }
  .next-title {
    font-size: 13px;
    color: #d0d0d0;
    flex: 1;
  }
  .next-go {
    width: 28px; height: 28px;
    border-radius: 50%;
    border: 1px solid #7c7cff40;
    background: #7c7cff10;
    color: #7c7cff;
    font-size: 14px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .next-go:hover { background: #7c7cff30; }

  /* Stage Selector */
  .stage-row {
    display: flex;
    gap: 8px;
    margin-bottom: 14px;
  }
  .stage-pill {
    flex: 1;
    padding: 8px 4px;
    border-radius: 20px;
    border: 1px solid #1e1e30;
    background: #16161e;
    color: #888;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.2s;
    text-align: center;
  }
  .stage-pill.active {
    background: #7c7cff15;
    border-color: #7c7cff50;
    color: #c8c8ff;
  }

  /* Task List */
  .task-list { display: flex; flex-direction: column; gap: 6px; }
  .task-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
  }
  .task-check {
    background: none;
    border: none;
    color: #555;
    font-size: 16px;
    cursor: pointer;
    padding: 0;
    width: 24px;
    text-align: center;
  }
  .task-title {
    font-size: 13px;
    color: #c0c0c0;
    flex: 1;
  }
  .task-title.done {
    text-decoration: line-through;
    opacity: 0.5;
  }

  /* Suggestions */
  .suggestions {
    margin-bottom: 14px;
    overflow: hidden;
  }
  .suggestions-scroll {
    display: flex;
    gap: 10px;
    overflow-x: auto;
    padding-bottom: 4px;
    scrollbar-width: none;
  }
  .suggestions-scroll::-webkit-scrollbar { display: none; }
  .suggestion-card {
    flex-shrink: 0;
    width: 160px;
    padding: 12px;
    background: #16161e;
    border: 1px solid #1e1e30;
    border-radius: 10px;
    text-align: left;
    cursor: pointer;
    transition: border-color 0.2s;
    color: inherit;
  }
  .suggestion-card:hover { border-color: #7c7cff50; }
  .sug-icon { font-size: 18px; display: block; margin-bottom: 6px; }
  .sug-text { font-size: 12px; color: #aaa; line-height: 1.4; }

  /* Ask Bar */
  .ask-bar {
    margin-top: 8px;
  }
  .chat-response {
    background: #16161e;
    border: 1px solid #1e1e30;
    border-radius: 10px;
    padding: 12px;
    margin-bottom: 10px;
  }
  .chat-label {
    font-size: 11px;
    color: #7c7cff;
    font-weight: 600;
    margin-bottom: 4px;
  }
  .chat-text {
    font-size: 13px;
    color: #d0d0d0;
    white-space: pre-wrap;
    line-height: 1.5;
  }
  .ask-input-wrap {
    display: flex;
    gap: 8px;
  }
  .ask-input-wrap input {
    flex: 1;
    padding: 12px 14px;
    background: #16161e;
    border: 1px solid #1e1e30;
    border-radius: 10px;
    color: #e0e0e0;
    font-size: 14px;
    outline: none;
  }
  .ask-input-wrap input:focus { border-color: #7c7cff50; }
  .ask-input-wrap input::placeholder { color: #555; }
  .ask-input-wrap button {
    width: 44px;
    border-radius: 10px;
    border: 1px solid #7c7cff40;
    background: #7c7cff15;
    color: #7c7cff;
    font-size: 18px;
    cursor: pointer;
  }
  .ask-input-wrap button:hover { background: #7c7cff30; }
  .ask-input-wrap button:disabled { opacity: 0.3; cursor: default; }

  /* Welcome Card (new users) */
  .welcome-card {
    text-align: center;
    padding: 24px 20px;
    background: linear-gradient(135deg, #16161e, #1a1a30);
    border-color: #7c7cff30;
  }
  .wc-icon { font-size: 36px; margin-bottom: 8px; }
  .wc-title { font-size: 18px; font-weight: 700; color: #e8e8ff; margin: 0 0 8px; }
  .wc-desc { font-size: 13px; color: #888; line-height: 1.5; margin: 0 0 16px; }
  .wc-steps { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; text-align: left; }
  .wc-step {
    display: flex; align-items: center; gap: 10px;
    font-size: 13px; color: #bbb;
  }
  .wc-num {
    width: 24px; height: 24px; border-radius: 50%;
    background: #7c7cff20; color: #7c7cff;
    display: flex; align-items: center; justify-content: center;
    font-size: 11px; font-weight: 700; flex-shrink: 0;
  }
  .wc-btn {
    width: 100%;
    padding: 12px;
    background: #7c7cff;
    border: none;
    border-radius: 10px;
    color: white;
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    transition: opacity 0.2s;
  }
  .wc-btn:hover { opacity: 0.9; }
</style>
