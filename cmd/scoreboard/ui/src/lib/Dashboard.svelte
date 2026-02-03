<script>
  /**
   * Dashboard — Figma "Home Overview" faithful implementation
   * 
   * Sections (top to bottom, matching Figma exactly):
   * 1. Welcome header + user avatar + score pill
   * 2. Business Setup Progress bar (START % / COMPLETED) + NEXT TASK
   * 3. Finish Onboarding cards (horizontal scroll, only if incomplete)
   * 4. Network Growth Partners (avatar row + CONNECT)
   * 5. Stage selector pills (Idea / Launch / Growth)
   * 6. Daily Stand Up Tasks (checkboxes + action icons)
   * 7. Business Set Up Tasks (checkboxes + action icons)
   * 8. Wire Bot Intelligent Suggestions (horizontal scroll cards)
   * 9. Ask Wire Bot input bar + bot avatar
   *
   * Business-aware: optional filter lens, aggregate by default
   */
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  let { data = null, user = null, token = '', activeBusiness = '' } = $props();

  const BUSINESSES = [
    { id: '', label: 'All', icon: '🌐' },
    { id: 'STA', label: 'Startempire', icon: '⚡' },
    { id: 'WIR', label: 'Wirebot', icon: '🤖' },
    { id: 'PHI', label: 'Philoveracity', icon: '📘' },
    { id: 'SEW', label: 'SEW Network', icon: '🕸' },
  ];

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
  let partners = $state([]);
  let expandedTask = $state(null);
  let onboardingComplete = $state(false);

  const API = '';

  function headers() {
    return { 'Authorization': `Bearer ${token}` };
  }
  async function authFetch(path) {
    try {
      const res = await fetch(`${API}${path}`, { headers: headers() });
      if (!res.ok) return null;
      return res.json();
    } catch { return null; }
  }

  $effect(() => { if (token) loadAll(); });

  async function loadAll() {
    loading = true;
    const [cl, dt, st, dr] = await Promise.all([
      authFetch('/v1/checklist?action=summary'),
      authFetch('/v1/checklist?action=daily'),
      authFetch(`/v1/checklist?action=list&stage=${stage}`),
      authFetch('/v1/pairing/neural-drift'),
    ]);

    if (cl) {
      checklist = cl;
      nextTask = cl.next_task || null;
      stage = cl.stage || 'launch';
      onboardingComplete = (cl.percent || 0) >= 100;
    }
    if (dt?.tasks) dailyTasks = dt.tasks.slice(0, 5);
    if (st?.tasks) setupTasks = st.tasks.filter(t => !t.completed).slice(0, 5);
    if (dr?.drift) drift = dr.drift;

    // Mock partners until real network data exists
    partners = [
      { name: 'Network', avatar: '👤' },
      { name: 'Growth', avatar: '👤' },
      { name: 'Partners', avatar: '👤' },
      { name: 'Coming', avatar: '👤' },
      { name: 'Soon', avatar: '👤' },
    ];

    buildSuggestions();
    loading = false;
  }

  function buildSuggestions() {
    const s = [];
    const score = data?.score;
    if (score && score.execution_score < 30)
      s.push({ icon: '🚀', title: 'Ship Something', text: 'Even small wins count toward your daily score', action: 'ship' });
    if (score && score.revenue_score < 10)
      s.push({ icon: '💰', title: 'Track Revenue', text: 'Connect Stripe or FreshBooks to see real money flow', action: 'settings' });
    if (!drift || drift.score < 50)
      s.push({ icon: '🤝', title: 'Neural Handshake', text: 'Start your daily sync with Wirebot', action: 'handshake' });
    if (checklist && checklist.percent < 50)
      s.push({ icon: '📋', title: 'Setup Tasks', text: `Complete business setup (${checklist.percent}% done)`, action: 'checklist' });
    if (score && (!score.intent || score.intent === ''))
      s.push({ icon: '🎯', title: 'Set Intent', text: 'Declare what you\'ll ship today', action: 'intent' });
    if (s.length < 3)
      s.push({ icon: '⚡', title: 'Keep Building', text: 'You\'re making progress — keep the streak alive', action: 'score' });
    suggestions = s.slice(0, 4);
  }

  async function askWirebot() {
    if (!askInput.trim() || chatLoading) return;
    chatLoading = true; chatResponse = '';
    try {
      const res = await fetch(`${API}/v1/chat`, {
        method: 'POST',
        headers: { ...headers(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: askInput, session_id: chatSessionId || undefined })
      });
      const d = await res.json();
      chatResponse = d.response || d.content || d.error || 'No response';
      if (d.session_id) chatSessionId = d.session_id;
    } catch { chatResponse = 'Error connecting to Wirebot'; }
    askInput = ''; chatLoading = false;
  }

  async function completeTask(id) {
    await fetch(`${API}/v1/checklist?action=complete&id=${id}`, { method: 'POST', headers: headers() });
    loadAll();
  }

  function changeStage(s) { stage = s; loadAll(); }

  function handleSuggestion(action) {
    if (action === 'score') dispatch('nav', 'score');
    else if (action === 'settings') dispatch('nav', 'settings');
    else if (action === 'ship') dispatch('openFab');
    else if (action === 'handshake') doHandshake();
    else if (action === 'intent') dispatch('nav', 'score');
  }

  async function doHandshake() {
    try {
      const d = await fetch(`${API}/v1/pairing/handshake`, { method: 'POST', headers: headers() }).then(r => r.json());
      if (d.drift_score) drift = { ...drift, score: d.drift_score, signal: d.drift_signal, handshake_streak: d.handshake_streak };
    } catch {}
  }

  function keydown(e) { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); askWirebot(); } }

  function signalColor(score) { return score >= 60 ? '#00ff64' : score >= 30 ? '#ffc800' : '#ff3232'; }
  function driftColor(sig) { return { deep_sync:'#00ff64', in_drift:'#4a9eff', drifting:'#ffc800', weak:'#ff9500', disconnected:'#ff3232' }[sig] || '#666'; }
</script>

<div class="dashboard">
  {#if loading}
    <div class="loading"><div class="spinner"></div></div>
  {:else}

    <!-- ═══ 1. WELCOME HEADER ═══ -->
    <div class="header">
      <div class="header-left">
        <h1>Welcome, {user?.display_name?.split(' ')[0] || 'Verious'}!</h1>
        {#if data?.score}
          <div class="score-row">
            <span class="score-num" style="color:{signalColor(data.score.execution_score)}">⚡ {data.score.execution_score}</span>
            {#if data.streak?.current > 0}
              <span class="streak">🔥 {data.streak.current}</span>
            {/if}
            {#if drift}
              <span class="drift-chip" style="background:{driftColor(drift.signal)}20; color:{driftColor(drift.signal)}">🧠 {drift.score}%</span>
            {/if}
          </div>
        {/if}
      </div>
      <div class="header-right">
        <div class="avatar">{user?.display_name?.[0] || '👤'}</div>
      </div>
    </div>

    <!-- ═══ BUSINESS FILTER (optional lens) ═══ -->
    {#if data?.score}
      <div class="biz-row">
        {#each BUSINESSES as biz}
          <button class="biz-chip" class:active={activeBusiness === biz.id}
            onclick={() => { dispatch('businessChange', biz.id); }}>
            {biz.icon} {biz.label}
          </button>
        {/each}
      </div>
    {/if}

    <!-- ═══ 2. BUSINESS SETUP PROGRESS ═══ -->
    {#if checklist}
      <div class="card setup-card">
        <div class="setup-header">
          <span class="setup-label">BUSINESS SETUP TASKS — {checklist.percent || 0}%</span>
          <span class="setup-count">{checklist.completed || 0}/{checklist.total || 0}</span>
        </div>
        <div class="progress-wrap">
          <div class="progress-track">
            <div class="progress-fill" style="width:{checklist.percent || 0}%"></div>
          </div>
          <div class="progress-labels">
            <span class="progress-start">START {checklist.percent || 0}%</span>
            <span class="progress-end">{onboardingComplete ? 'COMPLETED' : 'IN PROGRESS'}</span>
          </div>
        </div>
        <div class="big-stat">{checklist.completed || 0} <span class="big-stat-label">TASKS COMPLETED</span></div>
        {#if nextTask}
          <div class="next-task">
            <span class="next-tag">NEXT TASK:</span>
            <span class="next-title">{nextTask.title || nextTask}</span>
            <button class="next-action" onclick={() => nextTask?.id && completeTask(nextTask.id)}>
              <span class="action-icons">✏️ ✓</span>
            </button>
          </div>
        {/if}
      </div>
    {/if}

    <!-- ═══ 3. FINISH ONBOARDING (if not complete) ═══ -->
    {#if checklist && !onboardingComplete}
      <div class="section-header">
        <span>FINISH ONBOARDING</span>
      </div>
      <div class="onboard-scroll">
        <div class="onboard-card">
          <div class="ob-icon">🎯</div>
          <div class="ob-title">Pairing Assessment</div>
          <div class="ob-desc">Help Wirebot understand you</div>
          <button class="ob-btn" onclick={() => dispatch('openPairing')}>Start →</button>
        </div>
        <div class="onboard-card">
          <div class="ob-icon">💳</div>
          <div class="ob-title">Connect Revenue</div>
          <div class="ob-desc">Stripe, FreshBooks, or Bank</div>
          <button class="ob-btn" onclick={() => dispatch('nav', 'settings')}>Connect →</button>
        </div>
        <div class="onboard-card">
          <div class="ob-icon">🚀</div>
          <div class="ob-title">Ship First Thing</div>
          <div class="ob-desc">Log your first ship event</div>
          <button class="ob-btn" onclick={() => dispatch('openFab')}>Ship →</button>
        </div>
      </div>
    {/if}

    <!-- ═══ 4. NETWORK GROWTH PARTNERS ═══ -->
    <div class="section-header">
      <span>NETWORK GROWTH PARTNERS</span>
      <button class="connect-link" onclick={() => dispatch('nav', 'settings')}>CONNECT ➜</button>
    </div>
    <div class="partners-row">
      {#each partners as p}
        <div class="partner-avatar">
          <div class="pa-circle">{p.avatar}</div>
        </div>
      {/each}
      <div class="partner-avatar">
        <div class="pa-circle pa-add">+</div>
      </div>
    </div>

    <!-- ═══ 5. STAGE SELECTOR ═══ -->
    <div class="stage-row">
      {#each ['idea', 'launch', 'growth'] as s}
        <button class="stage-pill" class:active={stage === s} onclick={() => changeStage(s)}>
          <span class="stage-dot" class:active={stage === s}></span>
          {s.charAt(0).toUpperCase() + s.slice(1)}
        </button>
      {/each}
    </div>

    <!-- ═══ 6. DAILY STAND UP TASKS ═══ -->
    <div class="section-header"><span>DAILY STAND UP TASKS</span></div>
    {#if dailyTasks.length > 0}
      <div class="task-list">
        {#each dailyTasks as task}
          <div class="task-item" class:done={task.completed}>
            <button class="task-check" onclick={() => task.id && completeTask(task.id)}>
              <span class="check-box" class:checked={task.completed}>{task.completed ? '✓' : ''}</span>
            </button>
            <span class="task-title">{task.title || 'Create Mission Statement'}</span>
            <div class="task-actions">
              <button class="ta-btn" title="Fire" onclick={() => task.id && completeTask(task.id)}>🔥</button>
              <button class="ta-btn" title="Configure" onclick={() => { expandedTask = expandedTask === task.id ? null : task.id; }}>⚙️</button>
              <button class="ta-btn" title="Details" onclick={() => dispatch('taskDetail', task)}>📋</button>
            </div>
          </div>
          {#if expandedTask === task.id}
            <div class="task-detail">
              <div class="td-row"><span class="td-label">Category:</span> <span>{task.category || 'General'}</span></div>
              <div class="td-row"><span class="td-label">Stage:</span> <span>{task.stage || stage}</span></div>
              {#if task.description}
                <div class="td-desc">{task.description}</div>
              {/if}
            </div>
          {/if}
        {/each}
      </div>
    {:else}
      <div class="task-empty">All caught up! 🎉</div>
    {/if}

    <!-- ═══ 7. BUSINESS SET UP TASKS ═══ -->
    <div class="section-header"><span>BUSINESS SET UP TASKS</span></div>
    {#if setupTasks.length > 0}
      <div class="task-list">
        {#each setupTasks as task}
          <div class="task-item">
            <button class="task-check" onclick={() => task.id && completeTask(task.id)}>
              <span class="check-box"></span>
            </button>
            <span class="task-title">{task.title || 'Create Mission Statement'}</span>
            <div class="task-actions">
              <button class="ta-btn">🔥</button>
              <button class="ta-btn" onclick={() => { expandedTask = expandedTask === task.id ? null : task.id; }}>⚙️</button>
              <button class="ta-btn">📋</button>
            </div>
          </div>
          {#if expandedTask === task.id}
            <div class="task-detail">
              <div class="td-row"><span class="td-label">Category:</span> <span>{task.category || 'General'}</span></div>
              <div class="td-row"><span class="td-label">Stage:</span> <span>{task.stage || stage}</span></div>
              {#if task.description}
                <div class="td-desc">{task.description}</div>
              {/if}
            </div>
          {/if}
        {/each}
      </div>
    {:else}
      <div class="task-empty">All setup tasks complete ✅</div>
    {/if}

    <!-- ═══ 8. WIREBOT SUGGESTIONS ═══ -->
    <div class="section-header"><span>WIRE BOT INTELLIGENT SUGGESTIONS</span></div>
    <div class="suggestions-scroll">
      {#each suggestions as sug}
        <button class="sug-card" onclick={() => handleSuggestion(sug.action)}>
          <div class="sug-icon">{sug.icon}</div>
          <div class="sug-title">{sug.title}</div>
          <div class="sug-text">{sug.text}</div>
        </button>
      {/each}
    </div>

    <!-- ═══ 9. ASK WIREBOT BAR ═══ -->
    {#if chatResponse}
      <div class="chat-bubble">
        <div class="cb-header">⚡ Wirebot</div>
        <div class="cb-text">{chatResponse}</div>
      </div>
    {/if}
    <div class="ask-bar">
      <input type="text" bind:value={askInput} onkeydown={keydown}
        placeholder="Ask Wire Bot A Question..." disabled={chatLoading} />
      <button class="ask-send" onclick={askWirebot} disabled={chatLoading || !askInput.trim()}>
        <span class="bot-icon">🤖</span>
      </button>
    </div>

  {/if}
</div>

<style>
  .dashboard { padding: 16px 16px 120px; max-width: 480px; margin: 0 auto; }
  .loading { display: flex; align-items: center; justify-content: center; height: 60vh; }
  .spinner { width: 32px; height: 32px; border: 3px solid #333; border-top-color: #7c7cff; border-radius: 50%; animation: spin .8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }

  /* ─── Header ─── */
  .header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 12px; }
  .header-left { flex: 1; }
  .header h1 { font-size: 24px; font-weight: 700; color: #f0f0f0; margin: 0; line-height: 1.2; }
  .score-row { display: flex; align-items: center; gap: 8px; margin-top: 6px; flex-wrap: wrap; }
  .score-num { font-size: 15px; font-weight: 700; }
  .streak { font-size: 12px; background: #ff440020; border: 1px solid #ff440040; padding: 2px 8px; border-radius: 10px; color: #ff8800; }
  .drift-chip { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: 10px; border: 1px solid transparent; }
  .avatar { width: 40px; height: 40px; border-radius: 50%; background: #2a2a3a; display: flex; align-items: center; justify-content: center; font-size: 18px; color: #888; font-weight: 700; }

  /* ─── Business Filter ─── */
  .biz-row { display: flex; gap: 6px; overflow-x: auto; margin-bottom: 14px; scrollbar-width: none; -webkit-overflow-scrolling: touch; }
  .biz-row::-webkit-scrollbar { display: none; }
  .biz-chip { padding: 5px 12px; border-radius: 20px; font-size: 11px; font-weight: 600; background: #16161e; border: 1px solid #1e1e30; color: #666; cursor: pointer; white-space: nowrap; transition: all .15s; }
  .biz-chip:hover { border-color: #7c7cff40; color: #aaa; }
  .biz-chip.active { background: #7c7cff15; border-color: #7c7cff; color: #7c7cff; }

  /* ─── Cards ─── */
  .card { background: #16161e; border: 1px solid #1e1e30; border-radius: 12px; padding: 14px; margin-bottom: 12px; }

  /* ─── Setup Card ─── */
  .setup-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 6px; }
  .setup-label { font-size: 11px; font-weight: 700; letter-spacing: .06em; color: #888; }
  .setup-count { font-size: 11px; background: #7c7cff20; color: #7c7cff; padding: 2px 8px; border-radius: 8px; font-weight: 600; }
  .progress-wrap { margin-bottom: 10px; }
  .progress-track { width: 100%; height: 8px; background: #1e1e30; border-radius: 4px; overflow: hidden; }
  .progress-fill { height: 100%; background: linear-gradient(90deg, #7c7cff, #a78bfa); border-radius: 4px; transition: width .6s ease; }
  .progress-labels { display: flex; justify-content: space-between; margin-top: 4px; }
  .progress-start { font-size: 10px; font-weight: 700; color: #7c7cff; }
  .progress-end { font-size: 10px; font-weight: 700; color: #555; }

  .big-stat { font-size: 32px; font-weight: 800; color: #e0e0e8; text-align: center; margin: 10px 0 6px; }
  .big-stat-label { font-size: 12px; font-weight: 700; color: #666; letter-spacing: .08em; display: block; }

  .next-task { display: flex; align-items: center; gap: 8px; padding: 8px 10px; background: #1a1a2a; border-radius: 8px; margin-top: 6px; }
  .next-tag { font-size: 10px; font-weight: 800; color: #7c7cff; letter-spacing: .05em; white-space: nowrap; }
  .next-title { font-size: 13px; color: #c8c8d0; flex: 1; }
  .next-action { background: none; border: none; color: #888; font-size: 13px; cursor: pointer; padding: 4px; }
  .action-icons { display: flex; gap: 4px; }

  /* ─── Section Headers ─── */
  .section-header { display: flex; justify-content: space-between; align-items: center; margin: 16px 0 8px; }
  .section-header span { font-size: 11px; font-weight: 800; letter-spacing: .08em; color: #888; }
  .connect-link { background: none; border: none; color: #7c7cff; font-size: 11px; font-weight: 700; cursor: pointer; letter-spacing: .05em; }

  /* ─── Onboarding Cards ─── */
  .onboard-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; margin-bottom: 8px; scrollbar-width: none; }
  .onboard-scroll::-webkit-scrollbar { display: none; }
  .onboard-card { flex-shrink: 0; width: 140px; padding: 14px; background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; }
  .ob-icon { font-size: 24px; margin-bottom: 6px; }
  .ob-title { font-size: 12px; font-weight: 700; color: #d0d0d8; margin-bottom: 2px; }
  .ob-desc { font-size: 11px; color: #666; margin-bottom: 8px; }
  .ob-btn { background: #7c7cff20; border: none; color: #7c7cff; font-size: 11px; font-weight: 600; padding: 5px 12px; border-radius: 6px; cursor: pointer; }
  .ob-btn:hover { background: #7c7cff30; }

  /* ─── Partners ─── */
  .partners-row { display: flex; gap: 10px; overflow-x: auto; padding: 4px 0 12px; scrollbar-width: none; }
  .partners-row::-webkit-scrollbar { display: none; }
  .pa-circle { width: 44px; height: 44px; border-radius: 50%; background: #2a2a3a; display: flex; align-items: center; justify-content: center; font-size: 20px; color: #888; }
  .pa-add { border: 2px dashed #333; background: transparent; color: #555; font-size: 18px; font-weight: 700; }

  /* ─── Stage Pills ─── */
  .stage-row { display: flex; gap: 8px; margin-bottom: 14px; }
  .stage-pill { flex: 1; padding: 9px 4px; border-radius: 22px; border: 1px solid #1e1e30; background: #16161e; color: #888; font-size: 12px; font-weight: 600; cursor: pointer; text-align: center; display: flex; align-items: center; justify-content: center; gap: 6px; transition: all .2s; }
  .stage-pill.active { background: #7c7cff; border-color: #7c7cff; color: white; }
  .stage-dot { width: 6px; height: 6px; border-radius: 50%; background: #444; }
  .stage-dot.active { background: white; }

  /* ─── Tasks ─── */
  .task-list { display: flex; flex-direction: column; gap: 2px; margin-bottom: 8px; }
  .task-item { display: flex; align-items: center; gap: 8px; padding: 8px 4px; border-bottom: 1px solid #1a1a28; }
  .task-item.done { opacity: 0.5; }
  .task-check { background: none; border: none; cursor: pointer; padding: 0; }
  .check-box { width: 20px; height: 20px; border: 2px solid #333; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #7c7cff; transition: all .15s; }
  .check-box.checked { background: #7c7cff20; border-color: #7c7cff; }
  .task-title { font-size: 13px; color: #c0c0c0; flex: 1; }
  .task-actions { display: flex; gap: 4px; }
  .ta-btn { background: none; border: none; font-size: 13px; cursor: pointer; padding: 2px 3px; opacity: 0.5; transition: opacity .15s; }
  .ta-btn:hover { opacity: 1; }
  .task-empty { padding: 20px; text-align: center; color: #555; font-size: 13px; }

  /* Task Detail (expandable) */
  .task-detail { padding: 8px 12px 8px 36px; background: #12121a; border-bottom: 1px solid #1a1a28; }
  .td-row { font-size: 12px; color: #888; margin-bottom: 4px; }
  .td-label { font-weight: 700; color: #666; }
  .td-desc { font-size: 12px; color: #777; margin-top: 4px; line-height: 1.5; }

  /* ─── Suggestions ─── */
  .suggestions-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; margin-bottom: 14px; scrollbar-width: none; }
  .suggestions-scroll::-webkit-scrollbar { display: none; }
  .sug-card { flex-shrink: 0; width: 150px; padding: 12px; background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; text-align: left; cursor: pointer; transition: border-color .2s; color: inherit; }
  .sug-card:hover { border-color: #7c7cff40; }
  .sug-icon { font-size: 20px; margin-bottom: 4px; }
  .sug-title { font-size: 12px; font-weight: 700; color: #d0d0d8; margin-bottom: 2px; }
  .sug-text { font-size: 11px; color: #666; line-height: 1.4; }

  /* ─── Ask Bar ─── */
  .chat-bubble { background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; padding: 12px; margin-bottom: 10px; }
  .cb-header { font-size: 11px; color: #7c7cff; font-weight: 700; margin-bottom: 4px; }
  .cb-text { font-size: 13px; color: #d0d0d0; white-space: pre-wrap; line-height: 1.5; }
  .ask-bar { display: flex; gap: 8px; }
  .ask-bar input { flex: 1; padding: 13px 14px; background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; color: #e0e0e0; font-size: 14px; outline: none; }
  .ask-bar input:focus { border-color: #7c7cff50; }
  .ask-bar input::placeholder { color: #555; }
  .ask-send { width: 48px; border-radius: 10px; border: 1px solid #7c7cff40; background: #7c7cff15; cursor: pointer; display: flex; align-items: center; justify-content: center; }
  .ask-send:hover { background: #7c7cff30; }
  .ask-send:disabled { opacity: .3; cursor: default; }
  .bot-icon { font-size: 22px; }
</style>
