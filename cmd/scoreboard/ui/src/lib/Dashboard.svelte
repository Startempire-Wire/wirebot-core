<script>
  /**
   * Dashboard — Figma "Home Overview" faithful implementation
   * 
   * Sections (top to bottom, matching Figma exactly):
   * 1. Welcome header + user avatar + score pill
   * 2. Business Setup Progress bar (START % / COMPLETED) + NEXT TASK
   * 3. Finish Onboarding cards (horizontal scroll, only if incomplete)
   * 4. Network Growth Partners (collapsible, real members only)
   * 5. Stage selector pills (Idea / Launch / Growth) — filters categories in-place
   * 6. Daily Stand Up Tasks (checkboxes + action icons)
   * 7. Business Set Up Tasks (grouped by category, collapsible)
   * 8. Wire Bot Intelligent Suggestions (horizontal scroll cards)
   * 9. Ask Wire Bot input bar + bot avatar
   *
   * Business-aware: optional filter lens, aggregate by default
   */
  import { createEventDispatcher } from 'svelte';
  const dispatch = createEventDispatcher();

  let { data = null, user = null, token = '', activeBusiness: parentBiz = '', pairingComplete = false, onOpenPairing = null, onNav = null } = $props();
  let localBiz = $state(parentBiz || '');  // local business filter state

  // Keep local business selector in sync with parent (prevents stale UI if parentBiz changes externally)
  $effect(() => {
    if ((parentBiz || '') !== localBiz) localBiz = parentBiz || '';
  });

  // Business = legal entity, Product = offering within a business
  // Startempire Wire (LLC) is the business. Network + Wirebot are products.
  const ENTITIES = [
    { id: '', label: 'All', icon: '🌐', type: 'all' },
    { id: 'SEW', label: 'SEW', fullName: 'Startempire Wire', icon: '🚀', type: 'business', legal: 'LLC',
      products: [
        { id: 'SEWN', label: 'SEWN', fullName: 'Startempire Wire Network', icon: '🕸', type: 'product' },
        { id: 'WIR', label: 'WB', fullName: 'Wirebot', icon: '🤖', type: 'product' },
      ]},
    { id: 'PVD', label: 'PVD', fullName: 'Philoveracity Design', icon: '📘', type: 'business', legal: 'Sole Prop', products: [] },
  ];

  // Flat list for iteration (with nesting info)
  const BUSINESSES = [];
  for (const e of ENTITIES) {
    BUSINESSES.push(e);
    if (e.products) {
      for (const p of e.products) {
        BUSINESSES.push({ ...p, parent: e.id });
      }
    }
  }

  // All checklist data (loaded once per stage)
  let allCategories = $state([]);   // full categories from API
  let categories = $state([]);      // filtered for display
  let checklist = $state(null);
  let dailyTasks = $state([]);
  let nextTask = $state(null);
  let stage = $state('launch');
  let askInput = $state('');
  let chatResponse = $state('');
  let chatLoading = $state(false);
  let chatSessionId = $state(null);
  let drift = $state(null);
  let suggestions = $state([]);
  let proposals = $state([]);
  let loading = $state(true);
  let partners = $state([]);
  let expandedTask = $state(null);
  let expandedCat = $state(null);
  let deferTarget = $state(null);  // task ID being deferred
  let deferMode = $state('time');
  let deferValue = $state('1w');
  let expandedEvidence = $state(null);  // "taskId:evIdx" for expanded snippet
  let expandedContext = $state('');     // full context from API
  let loadingContext = $state(false);
  let partnersOpen = $state(false); // collapsed by default (empty)
  let onboardingComplete = $state(false);
  let stageLoading = $state(false);
  let hasLoaded = $state(false);
  let bizKey = $state(0);  // bump to trigger content transition

  const API = '';
  function headers() { return { 'Authorization': `Bearer ${token}` }; }

  async function authFetch(path) {
    try {
      const res = await fetch(`${API}${path}`, { headers: headers() });
      if (!res.ok) return null;
      return res.json();
    } catch { return null; }
  }

  // Only fire once on mount, not on every token reactivity tick
  $effect(() => { if (token && !hasLoaded) { hasLoaded = true; loadAll(); } });

  async function loadAll() {
    loading = true;

    // Parallel: checklist + daily tasks + drift (skip slow external member call)
    const [grouped, dt, dr] = await Promise.all([
      authFetch(`/v1/checklist?action=grouped&stage=${stage}`),
      authFetch('/v1/checklist?action=daily'),
      authFetch('/v1/pairing/neural-drift'),
    ]);

    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      nextTask = grouped.next_task || null;
      stage = grouped.stage || 'launch';
      allCategories = grouped.categories || [];
      categories = allCategories;
      onboardingComplete = (grouped.percent || 0) >= 100;
    }
    if (dt?.tasks) dailyTasks = dt.tasks.slice(0, 5);
    if (dr?.drift) drift = dr.drift;

    buildSuggestions();
    loading = false;

    // Load partners + proposals in background (non-blocking)
    authFetch('/v1/network/members?limit=8').then(mem => {
      if (mem?.members) partners = mem.members;
      if (partners.length > 0) partnersOpen = true;
    });
    // Load proposals: both proposed (evidence-based) and action_ready (Wirebot drafted content)
    Promise.all([
      authFetch('/v1/proposals?action=list&status=proposed'),
      authFetch('/v1/proposals?action=list&status=action_ready'),
    ]).then(([proposed, ready]) => {
      const all = [...(ready?.proposals || []), ...(proposed?.proposals || [])];
      proposals = all;
    });
  }

  // Stage change — only reload checklist, not everything
  async function changeStage(s) {
    if (s === stage) return;
    stage = s;
    stageLoading = true;
    const grouped = await authFetch(`/v1/checklist?action=grouped&stage=${stage}`);
    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      nextTask = grouped.next_task || null;
      allCategories = grouped.categories || [];
      categories = allCategories;
      onboardingComplete = (grouped.percent || 0) >= 100;
    }
    expandedCat = null;
    stageLoading = false;
  }

  // Business filter — immediate local update + dispatch to parent
  function switchBusiness(bizId) {
    if (bizId === localBiz) return;
    localBiz = bizId;
    bizKey++;
    dispatch('businessChange', bizId);
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

  // Expand snippet to show more surrounding context
  async function expandSnippet(taskId, evIdx, ev) {
    const key = `${taskId}:${evIdx}`;
    if (expandedEvidence === key) {
      expandedEvidence = null;
      expandedContext = '';
      return;
    }
    expandedEvidence = key;
    expandedContext = '';
    loadingContext = true;

    try {
      // Fetch more context from the source file
      const res = await fetch(`${API}/v1/proposals?action=context&file=${encodeURIComponent(ev.file)}&section=${encodeURIComponent(ev.section || '')}`, {
        headers: headers()
      });
      const data = await res.json();
      expandedContext = data.context || ev.snippet;
    } catch {
      expandedContext = ev.snippet || 'Could not load context';
    }
    loadingContext = false;
  }

  // Mark evidence as "not related" — Wirebot learns to ignore similar matches
  async function markNotRelated(taskId, ev, scope = 'snippet') {
    try {
      await fetch(`${API}/v1/proposals?action=feedback`, {
        method: 'POST',
        headers: { ...headers(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          task_id: taskId,
          file: ev.file,
          section: ev.section,
          snippet: ev.snippet,
          keywords: ev.keywords,
          feedback: 'not_related',
          scope: scope  // 'snippet' = this specific match, 'file' = all matches from this file, 'task' = entire task proposal
        })
      });
      // Remove this evidence from the proposal locally
      proposals = proposals.map(p => {
        if (p.task_id === taskId) {
          const newEvidence = p.evidence.filter(e => e !== ev);
          // If no evidence left, remove proposal entirely
          if (newEvidence.length === 0) return null;
          return { ...p, evidence: newEvidence };
        }
        return p;
      }).filter(Boolean);
    } catch (e) {
      console.error('Feedback failed:', e);
    }
  }

  // Mark entire proposal as not related — Wirebot learns about this task
  async function markProposalNotRelated(taskId) {
    try {
      await fetch(`${API}/v1/proposals?action=feedback`, {
        method: 'POST',
        headers: { ...headers(), 'Content-Type': 'application/json' },
        body: JSON.stringify({
          task_id: taskId,
          feedback: 'not_related',
          scope: 'task'
        })
      });
      // Remove proposal locally
      proposals = proposals.filter(p => p.task_id !== taskId);
    } catch (e) {
      console.error('Feedback failed:', e);
    }
  }

  async function deferTask(taskId, source = 'checklist') {
    const endpoint = source === 'proposal'
      ? `/v1/proposals?action=defer&id=${taskId}&mode=${deferMode}&value=${deferValue}`
      : `/v1/checklist?action=defer&id=${taskId}&mode=${deferMode}&value=${deferValue}`;
    await fetch(`${API}${endpoint}`, { method: 'POST', headers: headers() });
    deferTarget = null;
    deferMode = 'time';
    deferValue = '1w';
    // Remove from proposals if it was a proposal
    proposals = proposals.filter(p => p.task_id !== taskId);
    // Reload checklist
    const grouped = await authFetch(`/v1/checklist?action=grouped&stage=${stage}`);
    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      allCategories = grouped.categories || [];
      categories = allCategories;
    }
  }

  async function acceptProposal(taskId) {
    // Haptic feedback on proposal accept
    if (navigator.vibrate) navigator.vibrate([30, 20, 50]);
    await fetch(`${API}/v1/proposals?action=accept&id=${taskId}`, { method: 'POST', headers: headers() });
    proposals = proposals.filter(p => p.task_id !== taskId);
    // Reload checklist to show updated progress
    const grouped = await authFetch(`/v1/checklist?action=grouped&stage=${stage}`);
    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      allCategories = grouped.categories || [];
      categories = allCategories;
    }
  }

  async function rejectProposal(taskId) {
    await fetch(`${API}/v1/proposals?action=reject&id=${taskId}`, { method: 'POST', headers: headers() });
    proposals = proposals.filter(p => p.task_id !== taskId);
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
    // Haptic feedback on task complete
    if (navigator.vibrate) navigator.vibrate([30, 20, 50]);
    await fetch(`${API}/v1/checklist?action=complete&id=${id}`, { method: 'POST', headers: headers() });
    // Reload just checklist
    const grouped = await authFetch(`/v1/checklist?action=grouped&stage=${stage}`);
    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      nextTask = grouped.next_task || null;
      allCategories = grouped.categories || [];
      categories = allCategories;
      onboardingComplete = (grouped.percent || 0) >= 100;
    }
    const dt = await authFetch('/v1/checklist?action=daily');
    if (dt?.tasks) dailyTasks = dt.tasks.slice(0, 5);
    buildSuggestions();
  }

  async function skipTask(id) {
    await fetch(`${API}/v1/checklist?action=complete&id=${id}`, { method: 'POST', headers: headers() });
    const grouped = await authFetch(`/v1/checklist?action=grouped&stage=${stage}`);
    if (grouped) {
      checklist = { total: grouped.total, completed: grouped.completed, percent: grouped.percent, stage: grouped.stage };
      nextTask = grouped.next_task || null;
      allCategories = grouped.categories || [];
      categories = allCategories;
      onboardingComplete = (grouped.percent || 0) >= 100;
    }
  }

  function handleSuggestion(action) {
    const go = (v) => { if (onNav) onNav(v); else dispatch('nav', v); };
    if (action === 'score' || action === 'intent') go('score');
    else if (action === 'settings') go('settings');
    else if (action === 'ship') go('feed');
    else if (action === 'handshake') doHandshake();
    else if (action === 'checklist') {
      document.querySelector('.cat-list')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    }
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
        <h1>Welcome, {user?.display_name?.split(' ')[0] || 'Operator'}!</h1>
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
        {#if user?.avatar_url}
          <img class="avatar-img" src={user.avatar_url} alt="" />
        {:else}
          <div class="avatar">{user?.display_name?.[0] || '👤'}</div>
        {/if}
      </div>
    </div>

    <!-- ═══ PAIRING ASSESSMENT CTA (prominent, only when incomplete) ═══ -->
    {#if !pairingComplete}
      <button class="pairing-cta-card" onclick={() => { if (onOpenPairing) onOpenPairing(); else dispatch('openPairing'); }}>
        <div class="pcc-left">
          <div class="pcc-icon">🧬</div>
          <div class="pcc-text">
            <div class="pcc-title">Take Your Founder Assessment</div>
            <div class="pcc-desc">5 min — helps Wirebot understand how you think, decide, and work</div>
          </div>
        </div>
        <div class="pcc-arrow">→</div>
      </button>
    {/if}

    <!-- ═══ BUSINESS FILTER (hierarchical: businesses → products) ═══ -->
    {#if data?.score}
      <div class="biz-row">
        {#each BUSINESSES as biz}
          <button
            class="biz-chip {biz.type || ''}"
            class:active={localBiz === biz.id}
            class:child={biz.parent}
            onclick={() => switchBusiness(biz.id)}>
            {#if biz.parent}<span class="biz-indent">└</span>{/if}
            <span class="biz-icon">{biz.icon}</span>
            <span class="biz-name">{biz.label}</span>
            {#if biz.legal}<span class="biz-legal">{biz.legal}</span>{/if}
          </button>
        {/each}
      </div>
      {#if localBiz}
        {@const active = BUSINESSES.find(b => b.id === localBiz)}
        <div class="biz-context">
          {active?.icon} {active?.fullName || active?.label}
          {#if active?.type === 'product'}
            <span class="biz-ctx-type">Product under {ENTITIES.find(e => e.id === active?.parent)?.fullName || 'Startempire Wire'}</span>
          {:else if active?.legal}
            <span class="biz-ctx-type">{active.legal}</span>
          {/if}
        </div>
      {/if}
    {/if}

    {#key bizKey}
    <div class="biz-content">

    <!-- ═══ 2. BUSINESS SETUP PROGRESS ═══ -->
    {#if checklist}
      <div class="card setup-card">
        <div class="setup-header">
          <span class="setup-label">{localBiz ? ((BUSINESSES.find(b=>b.id===localBiz)?.fullName || BUSINESSES.find(b=>b.id===localBiz)?.label || '').toUpperCase()+' — ') : ''}SETUP — {stage.toUpperCase()}</span>
          <span class="setup-count">{checklist.completed || 0}/{checklist.total || 0}</span>
        </div>
        <div class="progress-wrap">
          <div class="progress-track">
            <div class="progress-fill" style="width:{checklist.percent || 0}%"></div>
          </div>
          <div class="progress-labels">
            <span class="progress-start">{checklist.percent || 0}%</span>
            <span class="progress-end">{onboardingComplete ? '✅ COMPLETED' : 'IN PROGRESS'}</span>
          </div>
        </div>
        <div class="big-stat">{checklist.completed || 0} <span class="big-stat-label">TASKS COMPLETED</span></div>
        {#if nextTask}
          <button class="next-task" onclick={() => {
            // Scroll to the task's category and expand it
            if (nextTask.category) {
              expandedCat = nextTask.category;
              setTimeout(() => document.querySelector('.cat-tasks')?.scrollIntoView({ behavior: 'smooth', block: 'center' }), 50);
            }
          }}>
            <span class="next-tag">NEXT →</span>
            <span class="next-title">{nextTask.title || nextTask}</span>
            <span class="next-cat">{nextTask._catIcon || '📋'}</span>
          </button>
        {/if}
      </div>
    {/if}

    <!-- ═══ 3. FINISH ONBOARDING (if not complete) ═══ -->
    {#if checklist && !onboardingComplete}
      <div class="section-header"><span>FINISH ONBOARDING</span></div>
      <div class="onboard-scroll">
        <button class="onboard-card" onclick={() => { if (onOpenPairing) onOpenPairing(); }}>
          <div class="ob-icon">🎯</div>
          <div class="ob-title">Pairing Assessment</div>
          <div class="ob-desc">Help Wirebot understand you</div>
          <span class="ob-btn">Start →</span>
        </button>
        <button class="onboard-card" onclick={() => { if (onNav) onNav('settings'); }}>
          <div class="ob-icon">💳</div>
          <div class="ob-title">Connect Revenue</div>
          <div class="ob-desc">Stripe, FreshBooks, or Bank</div>
          <span class="ob-btn">Connect →</span>
        </button>
        <button class="onboard-card" onclick={() => { if (onNav) onNav('feed'); }}>
          <div class="ob-icon">🚀</div>
          <div class="ob-title">Check Your Feed</div>
          <div class="ob-desc">See events flowing in automatically</div>
          <span class="ob-btn">View →</span>
        </button>
      </div>
    {/if}

    <!-- ═══ 4. NETWORK GROWTH PARTNERS ═══ -->
    <button class="section-header section-toggle" onclick={() => partnersOpen = !partnersOpen}>
      <span>{partnersOpen ? '▾' : '▸'} NETWORK GROWTH PARTNERS {partners.length > 0 ? `(${partners.length})` : ''}</span>
      <span class="connect-link" onclick={(e) => { e.stopPropagation(); window.open('https://startempirewire.com/members/', '_blank'); }}>CONNECT ➜</span>
    </button>
    {#if partnersOpen}
      {#if partners.length > 0}
        <div class="partners-row">
          {#each partners as p}
            <a class="partner-avatar" href={p.link || '#'} target="_blank" title={p.name}>
              {#if p.avatar}
                <img class="pa-img" src={p.avatar} alt={p.name} />
              {:else}
                <div class="pa-circle">{p.name?.[0] || '?'}</div>
              {/if}
              <span class="pa-name">{p.name?.split(' ')[0] || ''}</span>
            </a>
          {/each}
          <a class="partner-avatar" href="https://startempirewire.com/members/" target="_blank">
            <div class="pa-circle pa-add">+</div>
            <span class="pa-name">Add</span>
          </a>
        </div>
      {:else}
        <div class="partners-empty">
          <div class="pe-avatars">
            <div class="pa-circle pe-ghost">👤</div>
            <div class="pa-circle pe-ghost">👤</div>
            <div class="pa-circle pe-ghost">👤</div>
            <div class="pa-circle pa-add pe-pulse">+</div>
          </div>
          <p class="pe-text">No growth partners yet</p>
          <p class="pe-hint">Connect with members on Startempire Wire, then designate them as growth partners.</p>
          <a class="pe-btn" href="https://startempirewire.com/members/" target="_blank">Find Partners →</a>
        </div>
      {/if}
    {/if}

    <!-- ═══ 5. STAGE SELECTOR + BUSINESS SETUP TASKS (together) ═══ -->
    <div class="section-header"><span>BUSINESS SET UP TASKS</span></div>
    <div class="stage-row">
      {#each ['idea', 'launch', 'growth'] as s}
        <button class="stage-pill" class:active={stage === s} disabled={stageLoading} onclick={() => changeStage(s)}>
          <span class="stage-dot" class:active={stage === s}></span>
          {s.charAt(0).toUpperCase() + s.slice(1)}
          {#if stageLoading && stage === s}<span class="stage-spin">⟳</span>{/if}
        </button>
      {/each}
    </div>

    {#if stageLoading}
      <div class="task-empty"><span class="spinner small"></span></div>
    {:else if categories.length > 0}
      <div class="cat-list">
        {#each categories as cat}
          <div class="cat-group">
            <button class="cat-header" onclick={() => { expandedCat = expandedCat === cat.id ? null : cat.id; }}>
              <span class="cat-icon">{cat.icon}</span>
              <span class="cat-label">{cat.label}</span>
              <div class="cat-right">
                <span class="cat-progress-text">{cat.completed}/{cat.total}</span>
                <div class="cat-bar"><div class="cat-fill" style="width:{cat.percent}%"></div></div>
                <span class="cat-chevron">{expandedCat === cat.id ? '▾' : '▸'}</span>
              </div>
            </button>
            {#if expandedCat === cat.id}
              <div class="cat-tasks">
                {#each cat.tasks || [] as task}
                  <div class="task-item" class:done={task.status === 'completed' || task.status === 'done'}>
                    <button class="task-check" onclick={() => task.id && completeTask(task.id)}>
                      <span class="check-box" class:checked={task.status === 'completed' || task.status === 'done'}>
                        {task.status === 'completed' || task.status === 'done' ? '✓' : ''}
                      </span>
                    </button>
                    <div class="task-body">
                      <div class="task-title-row">
                        <span class="task-title">{task.title}</span>
                        {#if task.business_id}
                          {@const tb = ENTITIES.flatMap(e => [e, ...(e.products||[])]).find(b => b.id === task.business_id)}
                          {#if tb}
                            <span class="task-biz">{tb.icon}{tb.label}</span>
                          {/if}
                        {/if}
                      </div>
                      {#if expandedTask === task.id}
                        {#if task.description}<p class="tdi-desc">{task.description}</p>{/if}
                        {#if task.aiSuggestion}<div class="task-ai">💡 {task.aiSuggestion}</div>{/if}
                      {/if}
                    </div>
                    <div class="task-actions">
                      <button class="ta-btn" title="AI Hint" onclick={() => { expandedTask = expandedTask === task.id ? null : task.id; }}>
                        {expandedTask === task.id ? '▾' : '💡'}
                      </button>
                      {#if task.status !== 'completed' && task.status !== 'done' && task.status !== 'deferred'}
                        <button class="ta-btn" title="Defer" onclick={() => { deferTarget = deferTarget === task.id ? null : task.id; deferMode = 'time'; deferValue = '1w'; }}>⏳</button>
                      {/if}
                      {#if task.status === 'deferred' && task.defer}
                        <span class="task-deferred-badge" title="Deferred: {task.defer.mode} → {task.defer.value}">⏳</span>
                      {/if}
                    </div>
                  </div>
                  {#if deferTarget === task.id}
                    <div class="defer-picker task-defer-picker">
                      <div class="defer-modes">
                        <button class="dm {deferMode === 'time' ? 'active' : ''}" onclick={() => { deferMode = 'time'; deferValue = '1w'; }}>⏱️ After</button>
                        <button class="dm {deferMode === 'stage' ? 'active' : ''}" onclick={() => { deferMode = 'stage'; deferValue = 'launch'; }}>🎯 Stage</button>
                        <button class="dm {deferMode === 'date' ? 'active' : ''}" onclick={() => { deferMode = 'date'; deferValue = ''; }}>📅 Date</button>
                        <button class="dm {deferMode === 'task' ? 'active' : ''}" onclick={() => { deferMode = 'task'; deferValue = ''; }}>🔗 Task</button>
                      </div>
                      <div class="defer-value">
                        {#if deferMode === 'time'}
                          <div class="defer-chips">
                            {#each [['2h','2 hrs'],['1d','1 day'],['3d','3 days'],['1w','1 wk'],['2w','2 wks'],['1m','1 mo']] as [v, label]}
                              <button class="dc {deferValue === v ? 'active' : ''}" onclick={() => deferValue = v}>{label}</button>
                            {/each}
                          </div>
                        {:else if deferMode === 'stage'}
                          <div class="defer-chips">
                            {#each [['idea','💡 Idea'],['launch','🚀 Launch'],['growth','📈 Growth']] as [v, label]}
                              <button class="dc {deferValue === v ? 'active' : ''}" onclick={() => deferValue = v}>{label}</button>
                            {/each}
                          </div>
                        {:else if deferMode === 'date'}
                          <input type="date" class="defer-date" bind:value={deferValue} min={new Date().toISOString().split('T')[0]} />
                        {:else if deferMode === 'task'}
                          <select class="defer-select" bind:value={deferValue}>
                            <option value="">Select blocking task…</option>
                            {#each allCategories.flatMap(c => c.tasks || []).filter(t => t.id !== task.id && t.status !== 'completed') as t}
                              <option value={t.id}>{t.title}</option>
                            {/each}
                          </select>
                        {/if}
                      </div>
                      <div class="defer-confirm">
                        <button class="defer-go" disabled={!deferValue} onclick={() => deferTask(task.id)}>⏳ Defer</button>
                        <button class="defer-cancel" onclick={() => deferTarget = null}>Cancel</button>
                      </div>
                    </div>
                  {/if}
                {/each}
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {:else}
      <div class="task-empty">No tasks for this stage</div>
    {/if}

    <!-- ═══ 7. DAILY STAND UP TASKS ═══ -->
    <div class="section-header"><span>DAILY STAND UP TASKS</span></div>
    {#if dailyTasks.length > 0}
      <div class="task-list">
        {#each dailyTasks as task}
          <div class="task-item" class:done={task.completed}>
            <button class="task-check" onclick={() => task.id && completeTask(task.id)}>
              <span class="check-box" class:checked={task.completed}>{task.completed ? '✓' : ''}</span>
            </button>
            <div class="task-body">
              <span class="task-title">{task.title || 'Untitled task'}</span>
              {#if expandedTask === `daily-${task.id}`}
                <div class="task-detail-inline">
                  {#if task.description}<p class="tdi-desc">{task.description}</p>{/if}
                  {#if task.aiSuggestion}<div class="task-ai">💡 {task.aiSuggestion}</div>{/if}
                  <div class="tdi-meta">
                    <span>{task.category || 'General'}</span>
                    <span>•</span>
                    <span>{task.stage || stage}</span>
                  </div>
                </div>
              {/if}
            </div>
            <div class="task-actions">
              <button class="ta-btn" title="Complete" onclick={() => task.id && completeTask(task.id)}>✅</button>
              <button class="ta-btn" title="Details" onclick={() => { expandedTask = expandedTask === `daily-${task.id}` ? null : `daily-${task.id}`; }}>
                {expandedTask === `daily-${task.id}` ? '▾' : '💡'}
              </button>
            </div>
          </div>
        {/each}
      </div>
    {:else}
      <div class="task-empty">All caught up! 🎉</div>
    {/if}

    <!-- ═══ 7b. WIREBOT PROPOSALS (auto-inferred completions) ═══ -->
    {#if proposals.length > 0}
      <div class="section-header"><span>📝 WIREBOT ACTIONS</span></div>
      <div class="proposals-list">
        {#each proposals as prop}
          {@const bizInfo = prop.business_id ? ENTITIES.flatMap(e => [e, ...(e.products||[])]).find(b => b.id === prop.business_id) : null}
          <div class="proposal-card" class:action-ready={prop.status === 'action_ready'}>
            <div class="prop-header">
              <span class="prop-title">
                {#if prop.status === 'action_ready'}📄{:else}🔍{/if}
                {prop.title}
              </span>
              <div class="prop-meta">
                {#if bizInfo}
                  <span class="prop-biz">{bizInfo.icon} {bizInfo.label}</span>
                {/if}
                {#if prop.status === 'action_ready'}
                  <span class="prop-conf prop-drafted">DRAFTED</span>
                {:else}
                  <span class="prop-conf" title="Confidence">{Math.round(prop.confidence * 100)}%</span>
                {/if}
              </div>
            </div>
            {#if prop.status === 'action_ready' && prop.draft}
              <div class="prop-draft-preview">
                {prop.draft.substring(0, 300)}{prop.draft.length > 300 ? '…' : ''}
              </div>
            {/if}
            <div class="prop-evidence-list">
              {#each prop.evidence as ev, evIdx}
                {@const evKey = `${prop.task_id}:${evIdx}`}
                <div class="prop-ev-item" class:expanded={expandedEvidence === evKey}>
                  <div class="prop-ev-header">
                    <span class="prop-ev-icon">{ev.source === 'vault' ? '📓' : ev.source === 'gdrive' ? '📁' : ev.source === 'dropbox' ? '📦' : ev.source === 'chat' ? '💬' : '📊'}</span>
                    <span class="prop-ev-file">{ev.file?.split('/').pop() || ev.file}</span>
                    {#if ev.section}
                      <span class="prop-ev-section">§ {ev.section}</span>
                    {/if}
                    <button class="ev-expand" title="Show more context" onclick={() => expandSnippet(prop.task_id, evIdx, ev)}>
                      {expandedEvidence === evKey ? '▾' : '▸'}
                    </button>
                  </div>
                  {#if ev.snippet}
                    <div class="prop-ev-snippet" onclick={() => expandSnippet(prop.task_id, evIdx, ev)}>
                      {#if expandedEvidence === evKey && expandedContext}
                        {#if loadingContext}
                          <span class="ev-loading">Loading...</span>
                        {:else}
                          {expandedContext}
                        {/if}
                      {:else}
                        "{ev.snippet}"
                      {/if}
                    </div>
                  {/if}
                  <div class="ev-feedback">
                    <button class="ev-not-related" title="This snippet is not related — Wirebot will learn" onclick={(e) => { e.stopPropagation(); markNotRelated(prop.task_id, ev, 'snippet'); }}>
                      ❌ Not related
                    </button>
                  </div>
                </div>
              {/each}
            </div>
            {#if prop.evidence?.length > 0}
              <div class="prop-all-unrelated">
                <button class="all-unrelated-btn" onclick={() => markProposalNotRelated(prop.task_id)}>
                  🚫 None of these are related
                </button>
              </div>
            {/if}
            <div class="prop-actions">
              <button class="prop-accept" onclick={() => acceptProposal(prop.task_id)}>✅ Done</button>
              <button class="prop-defer" onclick={() => { deferTarget = prop.task_id; deferMode = 'time'; deferValue = '1w'; }}>⏳ Defer</button>
              <button class="prop-reject" onclick={() => rejectProposal(prop.task_id)}>❌ No</button>
            </div>
            {#if deferTarget === prop.task_id}
              <div class="defer-picker">
                <div class="defer-modes">
                  <button class="dm {deferMode === 'time' ? 'active' : ''}" onclick={() => { deferMode = 'time'; deferValue = '1w'; }}>⏱️ After</button>
                  <button class="dm {deferMode === 'stage' ? 'active' : ''}" onclick={() => { deferMode = 'stage'; deferValue = 'launch'; }}>🎯 At Stage</button>
                  <button class="dm {deferMode === 'date' ? 'active' : ''}" onclick={() => { deferMode = 'date'; deferValue = ''; }}>📅 Date</button>
                  <button class="dm {deferMode === 'task' ? 'active' : ''}" onclick={() => { deferMode = 'task'; deferValue = ''; }}>🔗 After Task</button>
                </div>
                <div class="defer-value">
                  {#if deferMode === 'time'}
                    <div class="defer-chips">
                      {#each [['2h','2 hrs'],['1d','1 day'],['3d','3 days'],['1w','1 week'],['2w','2 weeks'],['1m','1 month']] as [v, label]}
                        <button class="dc {deferValue === v ? 'active' : ''}" onclick={() => deferValue = v}>{label}</button>
                      {/each}
                    </div>
                  {:else if deferMode === 'stage'}
                    <div class="defer-chips">
                      {#each [['idea','💡 Idea'],['launch','🚀 Launch'],['growth','📈 Growth']] as [v, label]}
                        <button class="dc {deferValue === v ? 'active' : ''}" onclick={() => deferValue = v}>{label}</button>
                      {/each}
                    </div>
                  {:else if deferMode === 'date'}
                    <input type="date" class="defer-date" bind:value={deferValue} min={new Date().toISOString().split('T')[0]} />
                  {:else if deferMode === 'task'}
                    <select class="defer-select" bind:value={deferValue}>
                      <option value="">Select blocking task…</option>
                      {#each allCategories.flatMap(c => c.tasks || []).filter(t => t.id !== prop.task_id && t.status !== 'completed' && t.status !== 'done') as t}
                        <option value={t.id}>{t.title}</option>
                      {/each}
                    </select>
                  {/if}
                </div>
                <div class="defer-confirm">
                  <button class="defer-go" disabled={!deferValue} onclick={() => deferTask(prop.task_id, 'proposal')}>⏳ Defer</button>
                  <button class="defer-cancel" onclick={() => deferTarget = null}>Cancel</button>
                </div>
              </div>
            {/if}
          </div>
        {/each}
      </div>
    {/if}

    <!-- ═══ 8. WIREBOT SUGGESTIONS ═══ -->
    <div class="section-header"><span>WIRE BOT SUGGESTIONS</span></div>
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
        {#if chatLoading}<span class="spinner small"></span>{:else}<span class="bot-icon">🤖</span>{/if}
      </button>
    </div>

    </div><!-- .biz-content -->
    {/key}

  {/if}
</div>

<style>
  .dashboard { padding: 16px 16px 120px; max-width: 480px; margin: 0 auto; }
  .loading { display: flex; align-items: center; justify-content: center; height: 60vh; }
  .spinner { width: 32px; height: 32px; border: 3px solid #333; border-top-color: #7c7cff; border-radius: 50%; animation: spin .8s linear infinite; }
  .spinner.small { width: 16px; height: 16px; border-width: 2px; }
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
  .avatar-img { width: 40px; height: 40px; border-radius: 50%; object-fit: cover; border: 2px solid #2a2a3a; }

  /* ─── Business Filter (hierarchical) ─── */
  .biz-row { display: flex; gap: 5px; overflow-x: auto; margin-bottom: 6px; scrollbar-width: none; -webkit-overflow-scrolling: touch; flex-wrap: nowrap; }
  .biz-row::-webkit-scrollbar { display: none; }
  .biz-chip { display: flex; align-items: center; gap: 3px; padding: 5px 10px; border-radius: 20px; font-size: 11px; font-weight: 600; background: #16161e; border: 1px solid #1e1e30; color: #666; cursor: pointer; white-space: nowrap; transition: all .2s; flex-shrink: 0; }
  .biz-chip:hover { border-color: #7c7cff40; color: #aaa; }
  .biz-chip.active { background: #7c7cff15; border-color: #7c7cff; color: #7c7cff; }
  .biz-chip.child { padding-left: 6px; font-size: 10px; border-style: dashed; }
  .biz-chip.child.active { border-style: solid; }
  .biz-chip.business { font-weight: 700; }
  .biz-indent { font-size: 9px; color: #444; margin-right: 1px; }
  .biz-icon { font-size: 12px; }
  .biz-name { }
  .biz-legal { font-size: 8px; font-weight: 400; color: #555; background: #1e1e30; padding: 1px 4px; border-radius: 4px; margin-left: 2px; }
  .biz-chip.active .biz-legal { color: #7c7cff80; background: #7c7cff10; }
  .biz-context { font-size: 10px; font-weight: 700; letter-spacing: .06em; color: #7c7cff; text-align: center; margin: 4px 0 10px; animation: biz-label-in 350ms ease-out; display: flex; align-items: center; justify-content: center; gap: 6px; }
  .biz-ctx-type { font-weight: 400; color: #555; font-size: 9px; }
  @keyframes biz-label-in { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }
  .biz-content { animation: biz-fade 450ms cubic-bezier(0.25, 0.46, 0.45, 0.94); }
  @keyframes biz-fade {
    0% { opacity: 0; transform: translateY(12px); }
    40% { opacity: 0.6; }
    100% { opacity: 1; transform: translateY(0); }
  }

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

  .next-task { display: flex; align-items: center; gap: 8px; padding: 8px 10px; background: #1a1a2a; border-radius: 8px; margin-top: 6px; width: 100%; border: 1px solid #1e1e30; cursor: pointer; transition: border-color .15s; color: inherit; }
  .next-task:hover { border-color: #7c7cff40; }
  .next-tag { font-size: 10px; font-weight: 800; color: #7c7cff; letter-spacing: .05em; white-space: nowrap; }
  .next-title { font-size: 13px; color: #c8c8d0; flex: 1; text-align: left; }
  .next-cat { font-size: 14px; }

  /* ─── Section Headers ─── */
  .section-header { display: flex; justify-content: space-between; align-items: center; margin: 16px 0 8px; }
  .section-header span { font-size: 11px; font-weight: 800; letter-spacing: .08em; color: #888; }
  .connect-link { background: none; border: none; color: #7c7cff; font-size: 11px; font-weight: 700; cursor: pointer; letter-spacing: .05em; }
  .section-toggle { cursor: pointer; border: none; background: none; width: 100%; padding: 0; color: inherit; }

  /* ─── Onboarding Cards ─── */
  .onboard-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; margin-bottom: 8px; scrollbar-width: none; }
  .onboard-scroll::-webkit-scrollbar { display: none; }
  .onboard-card { flex-shrink: 0; width: 140px; padding: 14px; background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; text-align: left; cursor: pointer; transition: border-color .15s; color: inherit; }
  .onboard-card:hover { border-color: #7c7cff40; }
  .ob-icon { font-size: 24px; margin-bottom: 6px; }
  .ob-title { font-size: 12px; font-weight: 700; color: #d0d0d8; margin-bottom: 2px; }
  .ob-desc { font-size: 11px; color: #666; margin-bottom: 8px; }
  .ob-btn { background: #7c7cff20; color: #7c7cff; font-size: 11px; font-weight: 600; padding: 5px 12px; border-radius: 6px; display: inline-block; }

  /* ─── Partners (real BuddyBoss members) ─── */
  .partners-row { display: flex; gap: 12px; overflow-x: auto; padding: 4px 0 12px; scrollbar-width: none; animation: section-slide 350ms ease-out; }
  .partners-row::-webkit-scrollbar { display: none; }
  .partner-avatar { display: flex; flex-direction: column; align-items: center; gap: 4px; text-decoration: none; flex-shrink: 0; }
  .pa-img { width: 44px; height: 44px; border-radius: 50%; object-fit: cover; border: 2px solid #2a2a3a; }
  .pa-circle { width: 44px; height: 44px; border-radius: 50%; background: #2a2a3a; display: flex; align-items: center; justify-content: center; font-size: 18px; color: #888; font-weight: 700; }
  .pa-add { border: 2px dashed #333; background: transparent; color: #555; }
  .pa-name { font-size: 10px; color: #666; max-width: 50px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: center; }

  /* Partners empty state */
  .partners-empty { background: #16161e; border: 1px solid #1e1e30; border-radius: 12px; padding: 20px; text-align: center; margin-bottom: 12px; animation: section-slide 350ms ease-out; }
  .pe-avatars { display: flex; justify-content: center; gap: 10px; margin-bottom: 12px; }
  .pe-ghost { opacity: 0.2; }
  .pe-pulse { animation: pulse 2s ease-in-out infinite; opacity: 0.6; }
  @keyframes pulse { 0%,100% { opacity: 0.4; } 50% { opacity: 0.8; } }
  .pe-text { font-size: 14px; font-weight: 600; color: #888; margin: 0 0 4px; }
  .pe-hint { font-size: 12px; color: #555; margin: 0 0 12px; line-height: 1.5; }
  .pe-btn { display: inline-block; padding: 8px 20px; background: #7c7cff20; border: 1px solid #7c7cff40; color: #7c7cff; font-size: 12px; font-weight: 600; border-radius: 8px; text-decoration: none; transition: all .2s; }
  .pe-btn:hover { background: #7c7cff30; }

  /* ─── Stage Pills ─── */
  .stage-row { display: flex; gap: 8px; margin-bottom: 14px; }
  .stage-pill { flex: 1; padding: 9px 4px; border-radius: 22px; border: 1px solid #1e1e30; background: #16161e; color: #888; font-size: 12px; font-weight: 600; cursor: pointer; text-align: center; display: flex; align-items: center; justify-content: center; gap: 6px; transition: all .2s; }
  .stage-pill:disabled { opacity: 0.6; }
  .stage-pill.active { background: #7c7cff; border-color: #7c7cff; color: white; }
  .stage-dot { width: 6px; height: 6px; border-radius: 50%; background: #444; transition: background .2s; }
  .stage-dot.active { background: white; }
  .stage-spin { animation: spin .6s linear infinite; font-size: 10px; }

  /* ─── Section slide animation ─── */
  @keyframes section-slide {
    0% { opacity: 0; transform: translateY(10px); }
    50% { opacity: 0.7; }
    100% { opacity: 1; transform: translateY(0); }
  }

  /* ─── Tasks ─── */
  .task-list { display: flex; flex-direction: column; gap: 2px; margin-bottom: 8px; }
  .task-item { display: flex; align-items: flex-start; gap: 8px; padding: 8px 4px; border-bottom: 1px solid #1a1a28; }
  .task-item.done { opacity: 0.45; }
  .task-check { background: none; border: none; cursor: pointer; padding: 0; margin-top: 1px; }
  .check-box { width: 20px; height: 20px; border: 2px solid #333; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 12px; color: #7c7cff; transition: all .15s; }
  .check-box.checked { background: #7c7cff20; border-color: #7c7cff; }
  .task-body { flex: 1; min-width: 0; }
  .task-title-row { display: flex; align-items: center; gap: 6px; }
  .task-title { font-size: 13px; color: #c0c0c0; }
  .task-biz { font-size: 9px; padding: 1px 4px; border-radius: 4px; background: rgba(255,255,255,0.05); color: #777; white-space: nowrap; flex-shrink: 0; }
  .task-ai { font-size: 11px; color: #888; margin-top: 6px; line-height: 1.5; padding: 6px 8px; background: #12121a; border-radius: 6px; border-left: 2px solid #7c7cff40; animation: section-slide 250ms ease-out; }
  .task-detail-inline { margin-top: 6px; animation: section-slide 250ms ease-out; }
  .tdi-desc { font-size: 12px; color: #777; margin: 0 0 4px; line-height: 1.5; }
  .tdi-meta { font-size: 10px; color: #555; display: flex; gap: 6px; margin-top: 4px; }
  .task-actions { display: flex; gap: 2px; flex-shrink: 0; }
  .ta-btn { background: none; border: none; font-size: 12px; cursor: pointer; padding: 4px 5px; opacity: 0.5; transition: opacity .15s; border-radius: 4px; }
  .ta-btn:hover { opacity: 1; background: #1e1e30; }
  .task-empty { padding: 20px; text-align: center; color: #555; font-size: 13px; }

  /* ─── Category Groups (like 100tasks) ─── */
  .cat-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 12px; }
  .cat-group { background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; overflow: hidden; }
  .cat-header { display: flex; align-items: center; gap: 8px; width: 100%; padding: 12px; background: none; border: none; color: inherit; cursor: pointer; text-align: left; }
  .cat-header:hover { background: #1a1a28; }
  .cat-icon { font-size: 16px; flex-shrink: 0; }
  .cat-label { font-size: 13px; font-weight: 600; color: #d0d0d8; flex: 1; }
  .cat-right { display: flex; align-items: center; gap: 8px; }
  .cat-progress-text { font-size: 11px; color: #666; font-weight: 600; white-space: nowrap; }
  .cat-bar { width: 40px; height: 4px; background: #1e1e30; border-radius: 2px; overflow: hidden; }
  .cat-fill { height: 100%; background: #7c7cff; border-radius: 2px; transition: width .4s; }
  .cat-chevron { font-size: 10px; color: #555; }
  .cat-tasks { padding: 4px 12px 8px; border-top: 1px solid #1a1a28; animation: section-slide 300ms ease-out; }

  /* ─── Suggestions ─── */
  .suggestions-scroll { display: flex; gap: 10px; overflow-x: auto; padding-bottom: 4px; margin-bottom: 14px; scrollbar-width: none; }
  .suggestions-scroll::-webkit-scrollbar { display: none; }

  /* ─── Pairing Assessment CTA ─── */
  .pairing-cta-card {
    display: flex; justify-content: space-between; align-items: center;
    background: linear-gradient(135deg, rgba(124,124,255,0.15), rgba(124,124,255,0.05));
    border: 1px solid rgba(124,124,255,0.4);
    border-radius: 12px; padding: 14px 16px; margin-bottom: 12px;
    cursor: pointer; width: 100%; text-align: left;
    -webkit-tap-highlight-color: transparent;
    animation: pairingPulse 3s infinite;
  }
  @keyframes pairingPulse {
    0%, 100% { border-color: rgba(124,124,255,0.4); }
    50% { border-color: rgba(124,124,255,0.8); box-shadow: 0 0 12px rgba(124,124,255,0.2); }
  }
  .pcc-left { display: flex; align-items: center; gap: 12px; }
  .pcc-icon { font-size: 28px; }
  .pcc-title { font-size: 14px; font-weight: 700; color: #e8e8ff; }
  .pcc-desc { font-size: 11px; color: #8888bb; margin-top: 2px; }
  .pcc-arrow { font-size: 20px; color: #7c7cff; }

  /* ─── Proposals ─── */
  .proposals-list { display: flex; flex-direction: column; gap: 10px; margin-bottom: 16px; }
  .proposal-card {
    background: linear-gradient(135deg, #1a1a2e, #16213e);
    border: 1px solid rgba(124,124,255,0.3);
    border-radius: 10px; padding: 12px;
  }
  .prop-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; gap: 8px; }
  .prop-title { font-size: 13px; font-weight: 600; color: #e8e8ff; flex: 1; }
  .prop-meta { display: flex; gap: 6px; align-items: center; flex-shrink: 0; }
  .prop-biz { font-size: 10px; padding: 2px 6px; border-radius: 8px; background: rgba(255,255,255,0.06); color: #aaa; white-space: nowrap; }
  .prop-biz-all { color: #666; }
  .prop-conf { font-size: 11px; color: #7c7cff; background: rgba(124,124,255,0.1); padding: 2px 6px; border-radius: 8px; }
  .proposal-card.action-ready { border-color: rgba(0,220,130,0.3); background: linear-gradient(135deg, #0d1117 0%, rgba(0,220,130,0.05) 100%); }
  .prop-drafted { background: rgba(0,220,130,0.2) !important; color: #00dc82 !important; }
  .prop-draft-preview {
    margin: 8px 0; padding: 10px; border-radius: 6px; background: rgba(0,0,0,0.4);
    font-size: 11px; color: #999; line-height: 1.5; white-space: pre-wrap;
    max-height: 120px; overflow-y: auto; border-left: 2px solid #00dc82;
  }
  .prop-evidence-list { display: flex; flex-direction: column; gap: 8px; margin-bottom: 10px; }
  .prop-ev-item {
    background: rgba(255,255,255,0.03); border-left: 2px solid #333;
    padding: 6px 8px; border-radius: 0 6px 6px 0;
  }
  .prop-ev-header { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
  .prop-ev-icon { font-size: 14px; flex-shrink: 0; }
  .prop-ev-file { font-size: 12px; font-weight: 500; color: #aaa; }
  .prop-ev-section { font-size: 11px; color: #7c7cff; font-style: italic; }
  .prop-ev-snippet {
    font-size: 11px; color: #999; margin-top: 4px; line-height: 1.4;
    font-style: italic; padding-left: 20px; cursor: pointer;
    overflow: hidden; display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical;
    transition: all 0.2s ease;
  }
  .prop-ev-item.expanded .prop-ev-snippet {
    -webkit-line-clamp: unset; max-height: none;
    background: rgba(0,0,0,0.3); padding: 10px; border-radius: 6px;
    white-space: pre-wrap; font-style: normal;
  }
  .ev-expand {
    background: none; border: none; color: #666; font-size: 12px;
    cursor: pointer; padding: 2px 6px; margin-left: auto;
  }
  .ev-expand:hover { color: #7c7cff; }
  .ev-feedback { display: flex; justify-content: flex-end; margin-top: 4px; }
  .ev-not-related {
    background: none; border: 1px solid rgba(255,80,80,0.2); color: #ff5050;
    font-size: 9px; padding: 2px 6px; border-radius: 4px; cursor: pointer;
    opacity: 0.6; transition: opacity 0.2s;
  }
  .ev-not-related:hover { opacity: 1; background: rgba(255,80,80,0.1); }
  .ev-loading { color: #666; }
  .prop-all-unrelated { margin-bottom: 10px; }
  .all-unrelated-btn {
    width: 100%; padding: 6px; border-radius: 6px;
    background: rgba(255,80,80,0.05); border: 1px dashed rgba(255,80,80,0.2);
    color: #aa5555; font-size: 11px; cursor: pointer;
  }
  .all-unrelated-btn:hover { background: rgba(255,80,80,0.1); color: #ff5050; }
  .prop-actions { display: flex; gap: 6px; }
  .prop-accept, .prop-defer, .prop-reject {
    flex: 1; padding: 8px; border-radius: 8px; border: none;
    font-size: 11px; font-weight: 600; cursor: pointer;
  }
  .prop-accept { background: rgba(0,220,130,0.15); color: #00dc82; }
  .prop-defer { background: rgba(255,200,50,0.1); color: #e8b830; }
  .prop-reject { background: rgba(255,80,80,0.1); color: #ff5050; }

  /* ─── Defer Picker ─── */
  .defer-picker {
    margin-top: 8px; padding: 10px; border-radius: 8px;
    background: rgba(0,0,0,0.3); border: 1px solid #2a2a40;
  }
  .task-defer-picker { margin: 4px 0 8px 28px; }
  .defer-modes { display: flex; gap: 4px; margin-bottom: 8px; }
  .dm {
    flex: 1; padding: 5px 2px; border-radius: 6px; border: 1px solid #2a2a40;
    background: transparent; color: #888; font-size: 10px; cursor: pointer; text-align: center;
  }
  .dm.active { border-color: #e8b830; color: #e8b830; background: rgba(232,184,48,0.1); }
  .defer-value { margin-bottom: 8px; }
  .defer-chips { display: flex; gap: 4px; flex-wrap: wrap; }
  .dc {
    padding: 4px 8px; border-radius: 6px; border: 1px solid #2a2a40;
    background: transparent; color: #999; font-size: 11px; cursor: pointer;
  }
  .dc.active { border-color: #e8b830; color: #e8b830; background: rgba(232,184,48,0.1); }
  .defer-date, .defer-select {
    width: 100%; padding: 6px 8px; border-radius: 6px; border: 1px solid #2a2a40;
    background: #0a0a15; color: #ddd; font-size: 12px;
  }
  .defer-confirm { display: flex; gap: 6px; }
  .defer-go {
    flex: 1; padding: 7px; border-radius: 6px; border: none;
    background: rgba(232,184,48,0.2); color: #e8b830; font-size: 12px; font-weight: 600; cursor: pointer;
  }
  .defer-go:disabled { opacity: 0.3; cursor: default; }
  .defer-cancel {
    padding: 7px 12px; border-radius: 6px; border: none;
    background: rgba(255,255,255,0.05); color: #666; font-size: 12px; cursor: pointer;
  }
  .task-deferred-badge { font-size: 12px; opacity: 0.5; }
  .sug-card { flex-shrink: 0; width: 150px; padding: 12px; background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; text-align: left; cursor: pointer; transition: border-color .15s; color: inherit; }
  .sug-card:hover { border-color: #7c7cff40; }
  .sug-icon { font-size: 20px; margin-bottom: 4px; }
  .sug-title { font-size: 12px; font-weight: 700; color: #d0d0d8; margin-bottom: 2px; }
  .sug-text { font-size: 11px; color: #666; line-height: 1.4; }

  /* ─── Ask Bar ─── */
  .chat-bubble { background: #16161e; border: 1px solid #1e1e30; border-radius: 10px; padding: 12px; margin-bottom: 10px; animation: section-slide 350ms ease-out; }
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
