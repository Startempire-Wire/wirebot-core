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
  let tokenStatus = $state(null);  // null | 'ok' | 'fail' | 'saving'
  let tokenMsg = $state('');
  let loginUser = $state('');
  let loginPass = $state('');
  let loginLoading = $state(false);
  let loginError = $state('');
  let loggedInUser = $state(null); // { display_name, tier, tier_level, user_id }

  // ── Integrations (Connected Accounts) ──
  let integrations = $state([]);
  let showConnectForm = $state(null); // provider ID being configured, or null
  let connectCred = $state('');
  let connectExtra = $state(''); // channel_id, feed URL, etc.
  let connectStatus = $state(null); // null | 'saving' | 'ok' | 'fail'
  let connectMsg = $state('');

  // Integration registry — extensible catalog of connectable services
  // Every service connects from the UI. No server-side auto-magic.
  // OAuth services: click Connect → redirect to provider → callback with token
  // Credential services: paste API key or URL directly
  const PROVIDERS = [
    // ── Revenue ──
    { id: 'stripe', name: 'Stripe', icon: '💳', lane: 'revenue',
      auth: 'oauth', desc: 'Payment & subscription tracking',
      oauthUrl: '/v1/oauth/stripe/authorize',
      hint: 'Connect your Stripe account to track payments, subscriptions, and payouts',
      scopes: 'read_write' },
    { id: 'paypal', name: 'PayPal', icon: '💰', lane: 'revenue',
      auth: 'oauth', desc: 'Payment & invoice tracking',
      hint: 'Connect PayPal to track incoming payments',
      comingSoon: true },
    // ── Shipping ──
    { id: 'github', name: 'GitHub', icon: '🐙', lane: 'shipping',
      auth: 'oauth', desc: 'Commits, PRs, releases, and deploy tracking',
      oauthUrl: '/v1/oauth/github/authorize',
      hint: 'Connect GitHub to auto-track code shipping across your repos',
      scopes: 'repo,admin:repo_hook' },
    { id: 'gitlab', name: 'GitLab', icon: '🦊', lane: 'shipping',
      auth: 'oauth', desc: 'Commits, MRs, and pipeline tracking',
      hint: 'Connect GitLab to track shipping activity',
      comingSoon: true },
    { id: 'vercel', name: 'Vercel', icon: '▲', lane: 'shipping',
      auth: 'api_key', desc: 'Deploy tracking',
      hint: 'Vercel API token from dashboard → Settings → Tokens',
      credLabel: 'API Token', credPlaceholder: 'vercel_...' },
    // ── Distribution ──
    { id: 'youtube', name: 'YouTube', icon: '📺', lane: 'distribution',
      auth: 'oauth', desc: 'Video publishing detection',
      oauthUrl: '/v1/oauth/google/authorize?scope=youtube',
      hint: 'Connect YouTube to detect when you publish new videos',
      comingSoon: true },
    { id: 'youtube_key', name: 'YouTube (API Key)', icon: '📺', lane: 'distribution',
      auth: 'api_key', desc: 'Video publishing via API key',
      hint: 'YouTube Data API v3 key from Google Cloud Console',
      credLabel: 'API Key', credPlaceholder: 'AIza...',
      fields: [{ key: 'channel_id', label: 'Channel ID', placeholder: 'UC...' }] },
    { id: 'blog_rss', name: 'Blog / RSS', icon: '📝', lane: 'distribution',
      auth: 'rss_url', desc: 'Blog post detection via RSS or Atom feed',
      hint: 'Your blog\'s RSS feed URL — we\'ll check for new posts every 15 minutes',
      credLabel: 'Feed URL', credPlaceholder: 'https://yourblog.com/feed' },
    { id: 'podcast_rss', name: 'Podcast', icon: '🎙️', lane: 'distribution',
      auth: 'rss_url', desc: 'Episode detection via podcast RSS',
      hint: 'RSS feed URL from your podcast host (Anchor, Spotify, Apple)',
      credLabel: 'Feed URL', credPlaceholder: 'https://anchor.fm/s/.../podcast/rss' },
    { id: 'twitter', name: 'X (Twitter)', icon: '𝕏', lane: 'distribution',
      auth: 'oauth', desc: 'Post and engagement tracking',
      hint: 'Connect X to track business posts',
      comingSoon: true },
    { id: 'linkedin', name: 'LinkedIn', icon: '💼', lane: 'distribution',
      auth: 'oauth', desc: 'Post and article tracking',
      hint: 'Connect LinkedIn to track professional content',
      comingSoon: true },
    { id: 'newsletter', name: 'Newsletter', icon: '📧', lane: 'distribution',
      auth: 'api_key', desc: 'Email campaign tracking',
      hint: 'ConvertKit, Mailchimp, or Beehiiv API key',
      credLabel: 'API Key', credPlaceholder: 'API key from your email platform',
      comingSoon: true },
    // ── Systems ──
    { id: 'analytics', name: 'Google Analytics', icon: '📊', lane: 'systems',
      auth: 'oauth', desc: 'Traffic and conversion tracking',
      hint: 'Connect GA4 to monitor site performance',
      comingSoon: true },
    { id: 'uptime', name: 'Uptime Monitor', icon: '🟢', lane: 'systems',
      auth: 'webhook_url', desc: 'Uptime & downtime alerts',
      hint: 'Point your uptime service webhook to this URL:',
      webhookUrl: true,
      credLabel: 'Service Name', credPlaceholder: 'e.g. UptimeRobot, BetterStack' },
    // ── Wellness (Operator Sustainability — Pillar 10) ──
    { id: 'fitness', name: 'Fitness', icon: '💪', lane: 'systems',
      auth: 'oauth', desc: 'Workout & activity tracking',
      hint: 'Apple Health, Garmin, Fitbit, Strava',
      comingSoon: true },
    { id: 'sleep', name: 'Sleep', icon: '😴', lane: 'systems',
      auth: 'oauth', desc: 'Sleep quality tracking',
      hint: 'Oura, Whoop, Apple Health, Fitbit',
      comingSoon: true },
    { id: 'nutrition', name: 'Nutrition', icon: '🥗', lane: 'systems',
      auth: 'oauth', desc: 'Meal & nutrition tracking',
      hint: 'MyFitnessPal, Cronometer',
      comingSoon: true },
  ];

  const API = window.location.origin;

  async function loadIntegrations() {
    try {
      const res = await fetch(`${API}/v1/integrations`, { headers: authHeaders() });
      if (res.ok) {
        const data = await res.json();
        integrations = data.integrations || [];
      }
    } catch {}
  }

  function getConnectedProviders(providerId) {
    return integrations.filter(i => i.provider === providerId && (i.status === 'active' || i.status === 'error'));
  }

  function startOAuth(provider) {
    // Store return state so callback knows what we're connecting
    localStorage.setItem('wb_oauth_provider', provider.id);
    // Redirect to server OAuth initiation endpoint
    window.location.href = `${API}${provider.oauthUrl}`;
  }

  async function connectProvider(provider) {
    if (provider.comingSoon) return;

    // OAuth providers → redirect flow
    if (provider.auth === 'oauth' && provider.oauthUrl) {
      startOAuth(provider);
      return;
    }

    connectStatus = 'saving';
    connectMsg = 'Connecting...';

    // Build display name from credential for distinguishability
    let displayName = provider.name;
    if (provider.auth === 'rss_url' && connectCred) {
      try { displayName = new URL(connectCred).hostname; } catch { displayName = connectCred.substring(0, 40); }
    } else if (provider.auth === 'api_key' && connectExtra) {
      displayName = `${provider.name} (${connectExtra.substring(0, 20)})`;
    } else if (provider.auth === 'webhook_url' && connectCred) {
      displayName = `${provider.name} — ${connectCred}`;
    }

    const body = {
      provider: provider.id,
      auth_type: provider.auth,
      display_name: displayName,
      credential: connectCred,
      config: '{}',
    };

    // Add extra fields to config
    if (connectExtra && provider.fields?.length) {
      const cfg = {};
      cfg[provider.fields[0].key] = connectExtra;
      body.config = JSON.stringify(cfg);
    }

    // For webhook_url type, credential is the service name, we generate the URL
    if (provider.auth === 'webhook_url') {
      body.auth_type = 'webhook';
      body.credential = connectCred || provider.id;
    }

    try {
      const res = await fetch(`${API}/v1/integrations`, {
        method: 'POST',
        headers: { ...authHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        connectStatus = 'ok';
        connectMsg = `✓ ${provider.name} connected`;
        connectCred = '';
        connectExtra = '';
        showConnectForm = null;
        await loadIntegrations();
      } else {
        const err = await res.json();
        connectStatus = 'fail';
        connectMsg = err.error || 'Connection failed';
      }
    } catch (e) {
      connectStatus = 'fail';
      connectMsg = 'Network error';
    }
    setTimeout(() => { connectStatus = null; }, 4000);
  }

  // Handle OAuth callback (page load with ?oauth=provider&status=ok)
  function handleOAuthCallback() {
    const params = new URLSearchParams(window.location.search);
    const oauthProvider = params.get('oauth');
    const oauthStatus = params.get('oauth_status');
    if (oauthProvider && oauthStatus) {
      // Clean URL
      window.history.replaceState({}, '', '/');
      if (oauthStatus === 'ok') {
        connectStatus = 'ok';
        connectMsg = `✓ ${oauthProvider} connected`;
        view = 'settings';
        loadIntegrations();
      } else {
        connectStatus = 'fail';
        connectMsg = `✗ ${oauthProvider}: ${params.get('error') || 'connection failed'}`;
        view = 'settings';
      }
      localStorage.removeItem('wb_oauth_provider');
      setTimeout(() => { connectStatus = null; }, 5000);
    }
  }

  async function disconnectProvider(integrationId) {
    try {
      await fetch(`${API}/v1/integrations/${integrationId}`, {
        method: 'DELETE',
        headers: authHeaders(),
      });
      await loadIntegrations();
    } catch {}
  }

  // Try to get token from localStorage for authenticated calls
  function getToken() {
    return localStorage.getItem('wb_token') || '';
  }

  function authHeaders() {
    const token = getToken();
    return token ? { 'Authorization': `Bearer ${token}` } : {};
  }

  // ── Login via Ring Leader (per bigpicture.mdx auth flow) ──
  const RL_API = 'https://startempirewire.network/wp-json/sewn/v1';

  async function loginViaRingLeader() {
    if (!loginUser || !loginPass) { loginError = 'Enter username and password'; return; }
    loginLoading = true;
    loginError = '';
    try {
      const creds = btoa(`${loginUser}:${loginPass}`);
      const res = await fetch(`${RL_API}/auth/token`, {
        method: 'POST',
        headers: { 'Authorization': `Basic ${creds}` }
      });
      const data = await res.json();
      if (data.token) {
        localStorage.setItem('wb_token', data.token);
        localStorage.setItem('wb_user', JSON.stringify(data.user));
        localStorage.setItem('wb_token_exp', String(Date.now() + (data.expires_in || 86400) * 1000));
        loggedInUser = data.user;
        loginPass = '';
        tokenStatus = 'ok';
        tokenMsg = `✓ Connected as ${data.user.display_name} (${data.user.tier})`;
        setTimeout(() => { tokenStatus = null; }, 4000);
        view = 'score';
        fetchAll();
        loadIntegrations();
      } else {
        loginError = data.error || 'Login failed';
      }
    } catch (e) {
      loginError = 'Connection error — check network';
    }
    loginLoading = false;
  }

  function logout() {
    localStorage.removeItem('wb_token');
    localStorage.removeItem('wb_user');
    localStorage.removeItem('wb_token_exp');
    loggedInUser = null;
    tokenStatus = null;
    tokenMsg = '';
    data = null;
    feed = [];
    history = [];
    wrapped = null;
    view = 'score';
  }

  function restoreSession() {
    const token = getToken();
    if (!token) return;

    const exp = parseInt(localStorage.getItem('wb_token_exp') || '0');

    // JWT/SSO user — check expiry
    const userJson = localStorage.getItem('wb_user');
    if (userJson) {
      if (exp > 0 && exp < Date.now()) {
        logout(); return; // Expired
      }
      try { loggedInUser = JSON.parse(userJson); } catch { logout(); }
      return;
    }

    // Operator token — no wb_user stored, create minimal profile
    if (token && !userJson) {
      loggedInUser = { display_name: 'Operator', tier: 'operator', tier_level: 99, is_admin: true };
      // Set expiry if missing (24h)
      if (!exp) localStorage.setItem('wb_token_exp', String(Date.now() + 86400000));
    }
  }

  let tokenTimer = null;

  function debounceToken() {
    clearTimeout(tokenTimer);
    const input = document.getElementById('token-input');
    const val = (input?.value || '').trim();

    if (!val) {
      localStorage.removeItem('wb_token');
      tokenStatus = 'ok';
      tokenMsg = 'Token cleared';
      setTimeout(() => { tokenStatus = null; }, 2500);
      return;
    }

    // Instant feedback while typing
    tokenStatus = 'saving';
    tokenMsg = 'Saving...';

    // Debounce: verify after 600ms of no input (or immediate on paste)
    tokenTimer = setTimeout(() => verifyToken(val), 600);
  }

  async function verifyToken(val) {
    localStorage.setItem('wb_token', val);
    tokenStatus = 'saving';
    tokenMsg = 'Verifying...';

    try {
      const res = await fetch(`${API}/v1/events?limit=1`, {
        headers: { 'Authorization': `Bearer ${val}` }
      });
      if (res.ok) {
        tokenStatus = 'ok';
        tokenMsg = '✓ Connected — write features enabled';
        localStorage.setItem('wb_token_exp', String(Date.now() + 86400000));
        if (!loggedInUser) {
          loggedInUser = { display_name: 'Operator', tier: 'operator', tier_level: 99, is_admin: true };
        }
        fetchAll();
      } else {
        tokenStatus = 'fail';
        tokenMsg = '✗ Invalid token';
        localStorage.removeItem('wb_token');
      }
    } catch (e) {
      tokenStatus = 'fail';
      tokenMsg = '✗ Connection error';
      localStorage.removeItem('wb_token');
    }
    setTimeout(() => { tokenStatus = null; }, 4000);
  }

  async function fetchAll() {
    if (!getToken()) return; // Don't fetch without auth
    const hdrs = authHeaders();
    try {
      const [sbRes, feedRes, histRes] = await Promise.all([
        fetch(`${API}/v1/scoreboard?mode=dashboard`, { headers: hdrs }),
        fetch(`${API}/v1/feed?limit=50`, { headers: hdrs }),
        fetch(`${API}/v1/history?range=season`, { headers: hdrs }),
      ]);

      if (sbRes.status === 401 || sbRes.status === 403) {
        logout(); return;
      }
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
      const res = await fetch(`${API}/v1/season/wrapped`, { headers: authHeaders() });
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
    restoreSession();
    handleOAuthCallback();
    if (loggedInUser) {
      fetchAll();
      loadIntegrations();
    }
    const interval = setInterval(() => { if (loggedInUser) fetchAll(); }, 30000);
    if ('serviceWorker' in navigator) {
      navigator.serviceWorker.register('/sw.js').catch(() => {});
    }
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

{#if !loggedInUser}
  <!-- ── Not logged in: full-screen login ── -->
  <div class="login-screen">
    <div class="login-card">
      <div class="login-logo">⚡</div>
      <h1 class="login-title">Wirebot Scoreboard</h1>
      <p class="login-sub">Track execution. Ship work. Score progress.</p>
      <a class="btn-sso login-sso" href="https://startempirewire.com/?sewn_sso=1&redirect_uri=https://wins.wirebot.chat">
        🚀 Sign in with Startempire Wire
      </a>
      <details class="login-manual">
        <summary>Sign in with app password</summary>
        <input type="text" bind:value={loginUser} placeholder="Username"
          onkeydown={(e) => e.key === 'Enter' && document.getElementById('login-pass-main')?.focus()} />
        <input type="password" id="login-pass-main" bind:value={loginPass} placeholder="App password"
          onkeydown={(e) => e.key === 'Enter' && loginViaRingLeader()} />
        {#if loginError}
          <div class="token-status fail">{loginError}</div>
        {/if}
        <button class="btn-login" onclick={loginViaRingLeader} disabled={loginLoading}>
          {loginLoading ? 'Connecting...' : '→ Sign in'}
        </button>
      </details>
      <p class="login-privacy">Your data is private. No public access without authentication.</p>
    </div>
  </div>
{:else if error && !data}
  <div class="loading">
    <div class="ld-icon">⚡</div>
    <p>Connecting...</p>
    <p class="err">{error}</p>
  </div>
{:else if data}
  <div class="app">
    <div class="content">
      {#if view === 'score'}
        <Score {data} {lastUpdate} onHelp={() => showHints = true} user={loggedInUser} />
      {:else if view === 'feed'}
        <Feed items={feed} pendingCount={data?.pending_count || 0} onHelp={() => showHints = true} />
      {:else if view === 'season'}
        <Season season={data.season} {history} streak={data.streak} onHelp={() => showHints = true} />
      {:else if view === 'wrapped'}
        <Wrapped {wrapped} />
      {:else if view === 'settings'}
        <div class="settings-view">
          <div class="s-hdr"><h2>⚙️ Settings</h2></div>

          <!-- Auth: Login or Session -->
          {#if loggedInUser}
            <div class="s-group">
              <label>Account</label>
              <div class="session-card">
                <div class="sc-header">
                  {#if loggedInUser.avatar_url}
                    <img class="sc-avatar" src={loggedInUser.avatar_url} alt="" />
                  {/if}
                  <div class="sc-identity">
                    <div class="sc-name">{loggedInUser.display_name}</div>
                    {#if loggedInUser.username}<div class="sc-username">@{loggedInUser.username}</div>{/if}
                  </div>
                </div>

                <div class="sc-badges">
                  <span class="tier-badge tier-{loggedInUser.tier}">{loggedInUser.tier}</span>
                  {#if loggedInUser.is_admin}<span class="admin-badge">Admin</span>{/if}
                  {#if loggedInUser.roles?.includes('bbp_keymaster')}<span class="role-badge">Keymaster</span>{/if}
                </div>

                {#if loggedInUser.membership_ids?.length > 0}
                  <div class="sc-row">
                    <span class="sc-label">Membership</span>
                    <span class="sc-val">ID {loggedInUser.membership_ids.join(', ')}</span>
                  </div>
                {/if}

                {#if loggedInUser.email}
                  <div class="sc-row">
                    <span class="sc-label">Email</span>
                    <span class="sc-val">{loggedInUser.email}</span>
                  </div>
                {/if}

                {#if loggedInUser.url}
                  <div class="sc-row">
                    <span class="sc-label">Website</span>
                    <a class="sc-link" href={loggedInUser.url} target="_blank" rel="noopener">{loggedInUser.url.replace('https://', '')}</a>
                  </div>
                {/if}

                {#if loggedInUser.registered}
                  <div class="sc-row">
                    <span class="sc-label">Member since</span>
                    <span class="sc-val">{new Date(loggedInUser.registered).toLocaleDateString('en-US', { year: 'numeric', month: 'short' })}</span>
                  </div>
                {/if}

                {#if loggedInUser.description}
                  <div class="sc-bio">{loggedInUser.description.substring(0, 200)}{loggedInUser.description.length > 200 ? '...' : ''}</div>
                {/if}

                <button class="btn-logout" onclick={logout}>Sign out</button>
              </div>
              {#if tokenStatus}
                <div class="token-status" class:ok={tokenStatus === 'ok'}>{tokenMsg}</div>
              {/if}
            </div>
          {:else}
            <div class="s-group">
              <label>Sign in</label>
              <a class="btn-sso" href="https://startempirewire.com/?sewn_sso=1&redirect_uri=https://wins.wirebot.chat">
                → Sign in with Startempire Wire
              </a>
              <p class="s-hint">Uses your startempirewire.com login — no extra password needed</p>
            </div>

            <!-- Manual login fallback -->
            <details class="s-group">
              <summary class="s-detail-label">Manual login (app password)</summary>
              <input type="text" bind:value={loginUser} placeholder="Username"
                onkeydown={(e) => e.key === 'Enter' && document.getElementById('login-pass')?.focus()} />
              <input type="password" id="login-pass" bind:value={loginPass} placeholder="App password"
                onkeydown={(e) => e.key === 'Enter' && loginViaRingLeader()} />
              {#if loginError}
                <div class="token-status fail">{loginError}</div>
              {/if}
              <button class="btn-login" onclick={loginViaRingLeader} disabled={loginLoading}>
                {loginLoading ? 'Connecting...' : '→ Sign in'}
              </button>
            </details>

            <!-- Operator fallback -->
            <details class="s-group">
              <summary class="s-detail-label">Operator token (advanced)</summary>
              <input type="password" id="token-input" value={getToken()}
                oninput={debounceToken}
                onpaste={debounceToken}
                placeholder="Paste operator token" />
              {#if tokenStatus}
                <div class="token-status" class:ok={tokenStatus === 'ok'} class:fail={tokenStatus === 'fail'} class:saving={tokenStatus === 'saving'}>
                  {tokenMsg}
                </div>
              {/if}
            </details>
          {/if}
          <!-- ── Connected Accounts ── -->
          <div class="s-group">
            <label>Connected Accounts</label>
            <p class="s-hint-text">Connect services to flow real data into your scoreboard</p>

            {#if connectStatus}
              <div class="token-status" class:ok={connectStatus === 'ok'} class:fail={connectStatus === 'fail'} class:saving={connectStatus === 'saving'}>
                {connectMsg}
              </div>
            {/if}

            <!-- Group by lane -->
            {#each ['revenue', 'shipping', 'distribution', 'systems'] as lane}
              {@const laneProviders = PROVIDERS.filter(p => p.lane === lane)}
              <div class="int-lane-group">
                <div class="int-lane-header">
                  <span class="int-lane lane-{lane}">{lane}</span>
                  {@const laneConnected = laneProviders.reduce((n, p) => n + getConnectedProviders(p.id).length, 0)}
                  {#if laneConnected > 0}
                    <span class="int-lane-count">{laneConnected} connected</span>
                  {/if}
                </div>

                {#each laneProviders as provider}
                  {@const accounts = getConnectedProviders(provider.id)}
                  {@const hasAccounts = accounts.length > 0}
                  <div class="int-card" class:int-connected={hasAccounts} class:int-coming={provider.comingSoon}>
                    <div class="int-row">
                      <span class="int-icon">{provider.icon}</span>
                      <div class="int-info">
                        <div class="int-name">
                          {provider.name}
                          {#if hasAccounts}
                            <span class="int-count">{accounts.length}</span>
                          {/if}
                        </div>
                        <div class="int-desc">{provider.desc}</div>
                      </div>
                      <div class="int-action">
                        {#if provider.comingSoon}
                          <span class="int-soon">Soon</span>
                        {:else if showConnectForm === provider.id}
                          <button class="int-btn int-cancel" onclick={() => { showConnectForm = null; connectCred = ''; connectExtra = ''; }}>Cancel</button>
                        {:else if provider.auth === 'oauth' && provider.oauthUrl}
                          <button class="int-btn int-connect" onclick={() => connectProvider(provider)}>
                            {hasAccounts ? '+ Add' : 'Connect'}
                          </button>
                        {:else}
                          <button class="int-btn int-connect" onclick={() => { showConnectForm = provider.id; connectCred = ''; connectExtra = ''; }}>
                            {hasAccounts ? '+ Add' : 'Add'}
                          </button>
                        {/if}
                      </div>
                    </div>

                    <!-- Connected accounts list -->
                    {#if hasAccounts}
                      <div class="int-accounts">
                        {#each accounts as acct}
                          <div class="int-acct-row">
                            <span class="int-status-dot" class:active={acct.status === 'active'} class:error={acct.status === 'error'}></span>
                            <span class="int-acct-name">{acct.display_name || provider.name}</span>
                            {#if acct.last_used_at}
                              <span class="int-last-poll">Synced {new Date(acct.last_used_at).toLocaleDateString()}</span>
                            {/if}
                            <button class="int-acct-remove" onclick={() => disconnectProvider(acct.id)} title="Disconnect">✕</button>
                          </div>
                          {#if acct.last_error}
                            <div class="int-error">⚠ {acct.last_error}</div>
                          {/if}
                        {/each}
                      </div>
                    {/if}

                    <!-- Credential input form (non-OAuth) -->
                    {#if showConnectForm === provider.id && provider.auth !== 'oauth'}
                      <div class="int-form">
                        <p class="int-hint">{provider.hint}</p>

                        {#if provider.webhookUrl}
                          <div class="int-webhook-url">
                            <span class="int-wh-label">Your webhook URL:</span>
                            <code class="int-wh-code">{API}/v1/webhooks/{provider.id}</code>
                          </div>
                        {/if}

                        <input type={provider.auth === 'rss_url' ? 'url' : 'password'}
                          bind:value={connectCred}
                          placeholder={provider.credPlaceholder || provider.credLabel || 'API key or credential'}
                          onkeydown={(e) => {
                            if (e.key === 'Enter' && (!provider.fields?.length || connectExtra)) connectProvider(provider);
                            else if (e.key === 'Enter') document.getElementById('int-extra')?.focus();
                          }} />

                        {#if provider.fields?.length}
                          {#each provider.fields as field}
                            <input type="text" id="int-extra"
                              bind:value={connectExtra}
                              placeholder={field.placeholder || field.label}
                              onkeydown={(e) => e.key === 'Enter' && connectProvider(provider)} />
                          {/each}
                        {/if}

                        <button class="int-save" onclick={() => connectProvider(provider)}
                          disabled={!connectCred || connectStatus === 'saving'}>
                          {connectStatus === 'saving' ? 'Connecting...' : `Add ${provider.name}`}
                        </button>
                      </div>
                    {/if}

                    <!-- OAuth hint (not configured yet) -->
                    {#if showConnectForm === provider.id && provider.auth === 'oauth' && !provider.oauthUrl}
                      <div class="int-form">
                        <p class="int-hint">{provider.hint}</p>
                        <p class="int-hint" style="color: #7c7cff;">OAuth app setup required — coming soon</p>
                      </div>
                    {/if}
                  </div>
                {/each}
              </div>
            {/each}
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

  /* Login / Session */
  .btn-sso {
    display: block; text-align: center; text-decoration: none;
    background: #7c7cff; color: #fff; border-radius: 8px;
    padding: 14px; font-size: 15px; font-weight: 700;
    transition: background 0.15s;
  }
  .btn-sso:active { background: #5c5cdd; }

  .btn-login {
    background: #7c7cff; color: #fff; border: none; border-radius: 8px;
    padding: 12px; font-size: 14px; font-weight: 700; cursor: pointer;
    margin-top: 4px; transition: background 0.15s;
  }
  .btn-login:active { background: #5c5cdd; }
  .btn-login:disabled { opacity: 0.5; cursor: default; }

  .session-card {
    background: #111118; border: 1px solid #2a2a40; border-radius: 10px;
    padding: 14px; display: flex; flex-direction: column; gap: 10px;
  }
  .sc-header { display: flex; align-items: center; gap: 12px; }
  .sc-avatar {
    width: 48px; height: 48px; border-radius: 50%;
    border: 2px solid #2a2a40; object-fit: cover; flex-shrink: 0;
  }
  .sc-identity { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .sc-name { font-size: 16px; font-weight: 700; color: #eee; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  .sc-username { font-size: 12px; color: #555; }

  .sc-badges { display: flex; gap: 6px; flex-wrap: wrap; }
  .admin-badge {
    font-size: 10px; font-weight: 700; padding: 2px 8px; border-radius: 4px;
    background: #2e1a0a; color: #ff9500; text-transform: uppercase; letter-spacing: 0.05em;
  }
  .role-badge {
    font-size: 10px; font-weight: 600; padding: 2px 8px; border-radius: 4px;
    background: #1a1a1a; color: #666; text-transform: uppercase; letter-spacing: 0.05em;
  }

  .sc-row { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
  .sc-label { font-size: 11px; color: #444; flex-shrink: 0; }
  .sc-val { font-size: 12px; color: #888; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sc-link { font-size: 12px; color: #7c7cff; text-decoration: none; text-align: right; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sc-link:hover { text-decoration: underline; }
  .sc-bio { font-size: 11px; color: #555; line-height: 1.5; border-top: 1px solid #1e1e2e; padding-top: 8px; }

  .tier-badge {
    font-size: 11px; font-weight: 700; padding: 2px 8px; border-radius: 4px;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .tier-free { background: #222; color: #666; }
  .tier-freewire { background: #1a2a1a; color: #4caf50; }
  .tier-wire { background: #1a1a2e; color: #7c7cff; }
  .tier-extrawire { background: #2e1a2e; color: #ff7cff; }

  .btn-logout {
    background: transparent; border: 1px solid #333; color: #666; border-radius: 6px;
    padding: 6px 14px; font-size: 12px; cursor: pointer; align-self: flex-start; margin-top: 4px;
  }
  .btn-logout:hover { color: #ff4444; border-color: #ff4444; }

  .s-detail-label {
    font-size: 12px; color: #444; cursor: pointer; padding: 4px 0;
  }
  .s-detail-label:hover { color: #666; }
  details[open] .s-detail-label { color: #7c7cff; }

  .token-status {
    font-size: 12px; padding: 6px 10px; border-radius: 6px; margin-top: 2px;
    animation: fadeIn 0.2s ease;
  }
  .token-status.ok { color: #2ecc71; background: rgba(46,204,113,0.08); }
  .token-status.fail { color: #ff4444; background: rgba(255,68,68,0.08); }
  .token-status.saving { color: #ffaa00; background: rgba(255,170,0,0.08); }
  @keyframes fadeIn { from { opacity: 0; transform: translateY(-4px); } to { opacity: 1; transform: translateY(0); } }

  .s-hint-text { font-size: 11px; color: #555; margin-bottom: 4px; }

  /* Integration lane groups */
  .int-lane-group { display: flex; flex-direction: column; gap: 6px; }
  .int-lane-header { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
  .int-lane-count { font-size: 10px; color: #2ecc71; font-weight: 600; }

  /* Integration cards */
  .int-card {
    background: #111118; border: 1px solid #1e1e30; border-radius: 10px;
    padding: 12px; transition: border-color 0.2s;
  }
  .int-card.int-connected { border-color: #1a3a1a; }
  .int-card.int-coming { opacity: 0.4; }
  .int-row { display: flex; align-items: center; gap: 10px; }
  .int-icon { font-size: 22px; flex-shrink: 0; width: 28px; text-align: center; }
  .int-info { flex: 1; min-width: 0; }
  .int-name { font-size: 13px; font-weight: 700; color: #ddd; }
  .int-desc { font-size: 11px; color: #555; margin-top: 1px; }
  .int-lane {
    font-size: 9px; font-weight: 700; padding: 2px 8px; border-radius: 3px;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .lane-revenue { background: #1a2a1a; color: #2ecc71; }
  .lane-distribution { background: #1a1a2e; color: #7c7cff; }
  .lane-shipping { background: #2e2a1a; color: #ffaa00; }
  .lane-systems { background: #1a1a1a; color: #888; }

  .int-count {
    font-size: 10px; font-weight: 700; background: #1a3a1a; color: #2ecc71;
    width: 16px; height: 16px; border-radius: 50%; display: inline-flex;
    align-items: center; justify-content: center; margin-left: 2px;
  }

  .int-accounts {
    display: flex; flex-direction: column; gap: 4px;
    margin-top: 8px; padding-top: 8px; border-top: 1px solid #1a1a2a;
  }
  .int-acct-row {
    display: flex; align-items: center; gap: 6px;
    padding: 4px 8px; background: #0d0d16; border-radius: 6px;
  }
  .int-acct-name {
    font-size: 12px; color: #aaa; flex: 1; min-width: 0;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  }
  .int-acct-remove {
    background: none; border: none; color: #444; font-size: 12px;
    cursor: pointer; padding: 2px 4px; flex-shrink: 0;
  }
  .int-acct-remove:hover { color: #ff4444; }

  .int-action { flex-shrink: 0; }
  .int-btn {
    padding: 5px 12px; border-radius: 6px; font-size: 11px; font-weight: 600;
    cursor: pointer; border: none; transition: all 0.15s;
  }
  .int-connect { background: #7c7cff; color: white; }
  .int-connect:active { background: #5c5cdd; }
  .int-disconnect {
    background: transparent; color: #555; border: 1px solid #2a2a3a;
    font-size: 10px; padding: 4px 8px;
  }
  .int-disconnect:hover { color: #ff4444; border-color: #ff4444; }
  .int-cancel { background: #2a2a3a; color: #888; padding: 4px 10px; }
  .int-soon { font-size: 10px; color: #444; font-style: italic; }

  .int-status-row {
    display: flex; align-items: center; gap: 6px; margin-top: 8px;
    padding-top: 8px; border-top: 1px solid #1a1a2a; flex-wrap: wrap;
  }
  .int-status-dot {
    width: 6px; height: 6px; border-radius: 50%; flex-shrink: 0;
  }
  .int-status-dot.active { background: #2ecc71; }
  .int-status-dot.error { background: #ff4444; }
  .int-status-text { font-size: 11px; color: #2ecc71; font-weight: 600; }
  .int-status-text.error-text { color: #ff4444; }
  .int-last-poll { font-size: 10px; color: #444; margin-left: auto; }
  .int-error { font-size: 10px; color: #ff9500; width: 100%; margin-top: 4px; }

  .int-form {
    display: flex; flex-direction: column; gap: 8px; margin-top: 10px;
    padding-top: 10px; border-top: 1px solid #1a1a2a;
  }
  .int-hint { font-size: 11px; color: #666; line-height: 1.5; margin: 0; }
  .int-form input {
    background: #0a0a15; border: 1px solid #2a2a40; border-radius: 6px;
    padding: 8px 10px; color: #ddd; font-size: 12px; outline: none;
  }
  .int-form input:focus { border-color: #7c7cff; }
  .int-save {
    background: #7c7cff; color: white; border: none; border-radius: 6px;
    padding: 8px; font-size: 12px; font-weight: 600; cursor: pointer;
  }
  .int-save:disabled { opacity: 0.4; cursor: default; }
  .int-save:active:not(:disabled) { background: #5c5cdd; }

  .int-webhook-url {
    background: #0a0a15; border: 1px solid #2a2a40; border-radius: 6px;
    padding: 8px 10px; display: flex; flex-direction: column; gap: 4px;
  }
  .int-wh-label { font-size: 10px; color: #555; }
  .int-wh-code {
    font-size: 11px; color: #7c7cff; font-family: monospace;
    word-break: break-all; user-select: all; cursor: text;
  }

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

  /* ── Login Screen ── */
  .login-screen {
    display: flex; justify-content: center; align-items: center;
    min-height: 100dvh; width: 100%; padding: 24px;
    background: #0a0a12;
  }
  .login-card {
    width: min(400px, 100%);
    text-align: center;
  }
  .login-logo { font-size: 48px; margin-bottom: 12px; }
  .login-title { font-size: 22px; font-weight: 800; color: #eee; margin-bottom: 6px; }
  .login-sub { font-size: 14px; color: #666; margin-bottom: 28px; }
  .login-sso {
    display: block; padding: 16px; font-size: 16px; font-weight: 700;
    margin-bottom: 20px;
  }
  .login-manual {
    text-align: left; background: #12121e; border-radius: 12px;
    border: 1px solid #1e1e30; padding: 16px; margin-bottom: 20px;
  }
  .login-manual summary {
    cursor: pointer; color: #888; font-size: 13px; margin-bottom: 12px;
  }
  .login-manual input {
    width: 100%; padding: 12px; background: #1a1a2e; border: 1px solid #333;
    border-radius: 8px; color: #fff; font-size: 15px; margin-bottom: 10px;
  }
  .login-manual input:focus { outline: none; border-color: #7c7cff; }
  .login-privacy {
    font-size: 12px; color: #444; margin-top: 8px;
  }
</style>
