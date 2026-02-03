<script>
  import { onMount } from 'svelte';
  import Dashboard from './lib/Dashboard.svelte';
  import Score from './lib/Score.svelte';
  import Feed from './lib/Feed.svelte';
  import Season from './lib/Season.svelte';
  import Wrapped from './lib/Wrapped.svelte';
  import Nav from './lib/Nav.svelte';
  import Hints from './lib/Hints.svelte';
  import Chat from './lib/Chat.svelte';
  import Profile from './lib/Profile.svelte';
  import PairingFlow from './lib/PairingFlow.svelte';

  let view = $state('dashboard');
  let data = $state(null);
  let feed = $state([]);
  let history = $state([]);
  let wrapped = $state(null);
  let error = $state(null);
  let lastUpdate = $state('');

  // ── Multi-Business Context ──
  const BUSINESSES = [
    { id: '', label: 'All Businesses', icon: '🌐', color: '#7c7cff' },
    { id: 'SEW', label: 'Startempire Wire', icon: '🚀', color: '#ffaa00' },
    { id: 'WB', label: 'Wirebot', icon: '🤖', color: '#7c7cff' },
    { id: 'PVD', label: 'Philoveracity Design', icon: '📘', color: '#2ecc71' },
    { id: 'SEW', label: 'SEW Network', icon: '🕸', color: '#ff7c7c' },
  ];
  let activeBusiness = $state(''); // '' = all businesses
  let showFab = $state(false); // kept for backward compat with Dashboard dispatch
  let showHints = $state(false);
  let showChat = $state(false);
  let showPairing = $state(false);
  let pairingInstrument = $state('');
  let showProfile = $state(false);
  let eqBars = $state([]);
  let eqScore = $state(0);
  let eqLevel = $state('');
  let eqAcc = $state(0);
  let selfReportCount = $state(0);
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
  let connectBusiness = $state(''); // business_id for multi-account tagging
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
    { id: 'plaid', name: 'Bank Account', icon: '🏦', lane: 'revenue',
      auth: 'plaid', desc: 'Real bank data — balances, deposits, expenses',
      hint: 'Connect via Plaid — supports Novo, Chase, BofA, Wells Fargo, and 12,000+ banks',
      plaidProducts: 'transactions' },
    { id: 'stripe', name: 'Stripe', icon: '💳', lane: 'revenue',
      auth: 'oauth', desc: 'Payment & subscription tracking',
      oauthUrl: '/v1/oauth/stripe/authorize',
      hint: 'Connect your Stripe account to track payments, subscriptions, and payouts',
      scopes: 'read_write' },
    { id: 'woocommerce', name: 'WooCommerce', icon: '🛒', lane: 'revenue',
      auth: 'api_key', desc: 'Order & subscription tracking',
      hint: 'WooCommerce → Settings → REST API → Add key (Read access)',
      credLabel: 'Consumer Key', credPlaceholder: 'ck_...',
      fields: [{ key: 'consumer_secret', label: 'Consumer Secret', placeholder: 'cs_...' },
               { key: 'store_url', label: 'Store URL', placeholder: 'https://startempirewire.com' }] },
    { id: 'freshbooks', name: 'FreshBooks', icon: '📗', lane: 'revenue',
      desc: 'Invoices, expenses, payments — real P&L', auth: 'oauth',
      hint: 'Connect your FreshBooks account to track invoices, expenses, and payments' },
    { id: 'hubspot', name: 'HubSpot', icon: '🔶', lane: 'revenue',
      auth: 'oauth', desc: 'CRM deals, contacts, pipeline',
      hint: 'Connect HubSpot to track deals, contacts, and pipeline' },
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
    { id: 'cloudflare', name: 'Cloudflare', icon: '🔥', lane: 'shipping',
      auth: 'api_key', desc: 'DNS changes, deploys, security events',
      hint: 'Cloudflare → My Profile → API Tokens → Create Token (Zone:Read)',
      credLabel: 'API Token', credPlaceholder: 'Bearer token or Global API Key',
      fields: [{ key: 'account_id', label: 'Account ID', placeholder: 'From dashboard overview' }] },
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
    { id: 'sendy', name: 'Sendy', icon: '📧', lane: 'distribution',
      auth: 'api_key', desc: 'Self-hosted email — campaigns, subscribers, opens',
      hint: 'Sendy → Settings → API key. Also provide your Sendy install URL.',
      credLabel: 'API Key', credPlaceholder: 'Your Sendy API key',
      fields: [{ key: 'sendy_url', label: 'Sendy URL', placeholder: 'https://sendy.yourdomain.com' }] },
    { id: 'newsletter', name: 'Newsletter (Other)', icon: '✉️', lane: 'distribution',
      auth: 'api_key', desc: 'ConvertKit, Mailchimp, Beehiiv campaign tracking',
      hint: 'API key from your email platform',
      credLabel: 'API Key', credPlaceholder: 'API key from your email platform',
      comingSoon: true },
    // ── Systems ──
    { id: 'posthog', name: 'PostHog', icon: '🦔', lane: 'systems',
      auth: 'api_key', desc: 'Product analytics — pageviews, events, users',
      hint: 'PostHog → Project Settings → Personal API Keys (or use your VPS instance)',
      credLabel: 'Personal API Key', credPlaceholder: 'phx_...',
      fields: [{ key: 'host', label: 'PostHog Host', placeholder: 'https://data.philoveracity.com' }] },
    { id: 'uptimerobot', name: 'UptimeRobot', icon: '🟢', lane: 'systems',
      auth: 'api_key', desc: 'Uptime monitoring for all your sites',
      hint: 'UptimeRobot → My Settings → API Settings → Main API Key',
      credLabel: 'API Key', credPlaceholder: 'ur...' },
    { id: 'uptime', name: 'Uptime Webhook', icon: '🔔', lane: 'systems',
      auth: 'webhook_url', desc: 'Generic uptime webhook (BetterStack, Pingdom)',
      hint: 'Point your uptime service webhook to this URL:',
      webhookUrl: true,
      credLabel: 'Service Name', credPlaceholder: 'e.g. BetterStack, Pingdom' },
    { id: 'discord_webhook', name: 'Discord', icon: '💬', lane: 'systems',
      auth: 'api_key', desc: 'Bot activity, alerts, team messages',
      hint: 'Discord → Server Settings → Integrations → Webhooks → Copy Webhook URL',
      credLabel: 'Webhook URL', credPlaceholder: 'https://discord.com/api/webhooks/...' },
    { id: 'rescuetime', name: 'RescueTime', icon: '⏱️', lane: 'systems',
      auth: 'api_key', desc: 'Focus hours, productivity score, screen time',
      hint: 'RescueTime → Settings → Integrations/API → API Key',
      credLabel: 'API Key', credPlaceholder: 'B63...' },
    { id: 'analytics', name: 'Google Analytics', icon: '📊', lane: 'systems',
      auth: 'oauth', desc: 'Traffic and conversion tracking',
      hint: 'Connect GA4 to monitor site performance',
      comingSoon: true },
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
    // ── Documents (Business Intelligence — powers task proposals) ──
    { id: 'gdrive', name: 'Google Drive', icon: '📁', lane: 'systems',
      auth: 'api_key', desc: 'GSuite docs — Wirebot scans file names to auto-detect completed tasks',
      hint: 'Authorize via gogcli: run `gog auth add you@gsuite.com` then enter your GSuite email here',
      credLabel: 'GSuite Email', credPlaceholder: 'you@yourdomain.com',
      fields: [{ key: 'account', label: 'Account (same email)', placeholder: 'you@yourdomain.com' }] },
    { id: 'dropbox', name: 'Dropbox', icon: '📦', lane: 'systems',
      auth: 'api_key', desc: 'Business docs — Wirebot scans files to auto-detect completed tasks',
      hint: 'Dropbox → Settings → Developer → My Apps → Generate Access Token',
      credLabel: 'Access Token', credPlaceholder: 'sl.B...' },
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

  // ── Plaid Link ──
  let plaidReady = $state(false);

  function loadPlaidScript() {
    if (document.getElementById('plaid-link-script')) { plaidReady = true; return; }
    const s = document.createElement('script');
    s.id = 'plaid-link-script';
    s.src = 'https://cdn.plaid.com/link/v2/stable/link-initialize.js';
    s.onload = () => { plaidReady = true; };
    document.head.appendChild(s);
  }

  async function startPlaidLink(provider) {
    connectStatus = 'saving';
    connectMsg = 'Preparing bank connection...';

    // 1. Get a link_token from our server
    try {
      const res = await fetch(`${API}/v1/plaid/link-token`, {
        method: 'POST',
        headers: { ...authHeaders(), 'Content-Type': 'application/json' },
        body: JSON.stringify({ products: provider.plaidProducts || 'transactions' }),
      });
      const data = await res.json();

      if (!data.link_token) {
        connectStatus = 'fail';
        connectMsg = data.error || 'Failed to create link token';
        setTimeout(() => { connectStatus = null; }, 5000);
        return;
      }

      // 2. Load Plaid Link script if not loaded
      loadPlaidScript();
      await new Promise(resolve => {
        const check = setInterval(() => {
          if (window.Plaid) { clearInterval(check); resolve(); }
        }, 100);
        setTimeout(() => { clearInterval(check); resolve(); }, 5000);
      });

      if (!window.Plaid) {
        connectStatus = 'fail';
        connectMsg = 'Failed to load Plaid — check your connection';
        setTimeout(() => { connectStatus = null; }, 5000);
        return;
      }

      connectStatus = null;

      // 3. Open Plaid Link
      const handler = window.Plaid.create({
        token: data.link_token,
        onSuccess: async (publicToken, metadata) => {
          connectStatus = 'saving';
          connectMsg = 'Connecting bank account...';

          // 4. Exchange public_token for access_token on server
          try {
            const exchRes = await fetch(`${API}/v1/plaid/exchange`, {
              method: 'POST',
              headers: { ...authHeaders(), 'Content-Type': 'application/json' },
              body: JSON.stringify({
                public_token: publicToken,
                institution: metadata.institution,
                accounts: metadata.accounts,
              }),
            });
            const exchData = await exchRes.json();
            if (exchData.ok) {
              connectStatus = 'ok';
              connectMsg = `✓ ${metadata.institution?.name || 'Bank'} connected`;
              await loadIntegrations();
            } else {
              connectStatus = 'fail';
              connectMsg = exchData.error || 'Exchange failed';
            }
          } catch (e) {
            connectStatus = 'fail';
            connectMsg = 'Network error during exchange';
          }
          setTimeout(() => { connectStatus = null; }, 5000);
        },
        onExit: (err) => {
          if (err) {
            connectStatus = 'fail';
            connectMsg = err.display_message || 'Bank connection canceled';
            setTimeout(() => { connectStatus = null; }, 4000);
          }
        },
      });
      handler.open();

    } catch (e) {
      connectStatus = 'fail';
      connectMsg = 'Failed to start bank connection';
      setTimeout(() => { connectStatus = null; }, 5000);
    }
  }

  async function connectProvider(provider) {
    if (provider.comingSoon) return;

    // Plaid → Plaid Link widget flow
    if (provider.auth === 'plaid') {
      startPlaidLink(provider);
      return;
    }

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
      business_id: connectBusiness || '',
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
    const biz = activeBusiness ? `&business=${activeBusiness}` : '';
    try {
      const [sbRes, feedRes, histRes] = await Promise.all([
        fetch(`${API}/v1/scoreboard?mode=dashboard`, { headers: hdrs }),
        fetch(`${API}/v1/feed?limit=50${biz}`, { headers: hdrs }),
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

  async function fetchProfile() {
    const token = getToken();
    if (!token) return;
    try {
      const res = await fetch(`${API}/v1/pairing/profile/effective`, { headers: authHeaders() });
      if (!res.ok) return;
      const eff = await res.json();
      eqScore = Math.round(eff.pairing_score || 0);
      eqLevel = eff.level || '';
      eqAcc = Math.round((eff.accuracy || 0) * 100);
      selfReportCount = eff.self_report_count || 0;

      // Build equalizer bars from all dimensions
      const colors = {
        action_style: '#7c7cff', disc: '#ff7c7c', energy: '#7cff7c',
        risk: '#ffd700', cognitive: '#ff7cff',
        business: '#ff9f43', temporal: '#54a0ff'
      };
      const codeLabels = {
        FF: 'Fact', FT: 'Follow', QS: 'Quick', IM: 'Impl',
        D: 'Dom', I: 'Infl', S: 'Steady', C: 'Consc',
        W: 'Wonder', N: 'Invent', D_disc: 'Discern', G: 'Galv', E: 'Enable', T: 'Tenac',
        tolerance: 'Tol', speed: 'Speed', loss_aversion: 'Loss', ambiguity: 'Ambig',
        bias_to_action: 'Action', sunk_cost: 'Sunk',
        holistic: 'Holist', abstract: 'Abstr', sequential: 'Seq', concrete: 'Concr',
        focus: 'Focus', revenue_maturity: 'Rev', team_size: 'Team', bottleneck: 'Block', venture_age: 'Age', debt_pressure: 'Debt',
        peak_hour: 'Peak', planning_style: 'Plan', stall_recovery: 'Stall', work_intensity: 'Hours', context_switch_cost: 'Switch', planning_horizon: 'Horiz'
      };
      const bars = [];
      for (const [construct, color] of Object.entries(colors)) {
        const dims = eff[construct];
        if (!dims || typeof dims !== 'object') continue;
        for (const [code, val] of Object.entries(dims)) {
          if (typeof val !== 'number') continue;
          bars.push({ code: code.substring(0, 2).toUpperCase(), label: codeLabels[code] || code, pct: Math.min(100, val * 10), color });
        }
      }
      eqBars = bars;
    } catch {}
  }

  // Tab index order for slide direction
  const TAB_ORDER = ['dashboard', 'score', 'feed', 'season', 'settings', 'wrapped'];
  let prevView = $state('dashboard');

  function handleNav(e) {
    const next = e.detail;
    if (next === view) return;

    const from = TAB_ORDER.indexOf(view);
    const to = TAB_ORDER.indexOf(next);
    const direction = to > from ? 'forward' : 'back';

    // Use View Transitions API if available (Chrome 111+, Safari 18+)
    if (document.startViewTransition) {
      document.documentElement.dataset.direction = direction;
      document.startViewTransition(() => {
        prevView = view;
        view = next;
        if (next === 'wrapped' && !wrapped) fetchWrapped();
      });
    } else {
      // Fallback: CSS class-based animation
      const el = document.querySelector('.content');
      if (el) {
        el.classList.add(`slide-${direction}`);
        el.addEventListener('animationend', () => {
          el.classList.remove(`slide-${direction}`);
        }, { once: true });
      }
      prevView = view;
      view = next;
      if (next === 'wrapped' && !wrapped) fetchWrapped();
    }
  }

  onMount(() => {
    restoreSession();
    handleOAuthCallback();
    if (loggedInUser) {
      fetchAll();
      fetchProfile();
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
      {#if view === 'dashboard'}
        <Dashboard {data} user={loggedInUser} token={getToken()} {activeBusiness}
          pairingComplete={selfReportCount > 0}
          onOpenPairing={() => showPairing = true}
          onnav={(e) => view = e.detail}
          onopenFab={() => showChat = true}
          onopenPairing={() => showPairing = true}
          onbusinessChange={(e) => { activeBusiness = e.detail; fetchAll(); }} />
      {:else if view === 'score'}
        <Score {data} {lastUpdate} onHelp={() => showHints = true} user={loggedInUser} onPairing={selfReportCount === 0 ? () => showPairing = true : null} />
      {:else if view === 'feed'}
        <Feed items={feed} pendingCount={data?.pending_count || 0} onHelp={() => showHints = true}
          {activeBusiness} onBusinessChange={(biz) => { activeBusiness = biz; fetchAll(); }} />
      {:else if view === 'season'}
        <Season season={data.season} {history} streak={data.streak} onHelp={() => showHints = true} onnav={(e) => view = e.detail} />
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

                <!-- Founder Profile Equalizer (inside profile card) -->
                <div class="eq-strip" onclick={() => showProfile = true}>
                  <div class="eq-label">
                    <span>🧬 Founder Profile</span>
                    <span class="eq-arrow">→</span>
                  </div>
                  <div class="eq-viz">
                    {#each eqBars as bar}
                      <div class="eq-col" title="{bar.label}">
                        <div class="eq-bar-track">
                          <div class="eq-bar-fill" style="height:{bar.pct}%; background:{bar.color}"></div>
                        </div>
                        <span class="eq-bar-code">{bar.code}</span>
                      </div>
                    {/each}
                  </div>
                  <div class="eq-foot">
                    <span class="eq-score-line">{eqScore}/100 · {eqLevel || 'Initializing'}</span>
                    <span class="eq-acc-line">{eqAcc}% accurate</span>
                  </div>
                </div>

                {#if selfReportCount === 0}
                  <button class="eq-cta" onclick={() => showPairing = true}>
                    🧬 Take Founder Assessment <span class="eq-cta-sub">5 min · improves accuracy</span>
                  </button>
                {:else}
                  <button class="eq-cta eq-cta-retake" onclick={() => showPairing = true}>
                    🔄 Retake Assessment <span class="eq-cta-sub">{selfReportCount} answers · {eqAcc}% accurate</span>
                  </button>
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

            {#if connectStatus}
              <div class="token-status" class:ok={connectStatus === 'ok'} class:fail={connectStatus === 'fail'} class:saving={connectStatus === 'saving'}>
                {connectMsg}
              </div>
            {/if}

            <!-- Active integrations (what's already working) -->
            {#if integrations.length > 0}
              <div class="int-active-section">
                {#each integrations as acct}
                  {@const prov = PROVIDERS.find(p => p.id === acct.provider) || { icon: '🔗', name: acct.provider }}
                  <div class="int-active-card">
                    <span class="int-active-dot" class:active={acct.status === 'active'} class:error={acct.status === 'error'}></span>
                    <span class="int-active-icon">{prov.icon}</span>
                    <div class="int-active-info">
                      <div class="int-active-name">
                        {acct.display_name || prov.name}
                        {#if acct.business_id}
                          <span class="int-biz-badge">{acct.business_id}</span>
                        {/if}
                      </div>
                      <div class="int-active-meta">
                        {#if acct.status === 'active'}
                          ✓ Connected
                        {:else if acct.status === 'error'}
                          ⚠ Error
                        {:else}
                          {acct.status}
                        {/if}
                        {#if acct.last_used_at}
                          · Last sync {new Date(acct.last_used_at).toLocaleDateString()}
                        {/if}
                      </div>
                    </div>
                    <button class="int-active-remove" onclick={() => disconnectProvider(acct.id)} title="Disconnect">✕</button>
                  </div>
                {/each}
                <button class="int-add-another" onclick={() => { const el = document.getElementById('int-all'); if (el) el.style.display = 'block'; }}>
                  + Add account
                </button>
              </div>
            {:else}
              <div class="int-empty">
                <span class="int-empty-icon">🔌</span>
                <p>No accounts connected yet</p>
                <p class="int-empty-hint">Connect your tools to flow real data into your scoreboard</p>
              </div>
            {/if}

            <!-- Recommended integrations (high-impact, not yet connected) -->
            {#if PROVIDERS.filter(p => !p.comingSoon && !getConnectedProviders(p.id).length).length > 0}
              <div class="int-rec-header">Recommended</div>
              <div class="int-rec-grid">
                {#each PROVIDERS.filter(p => !p.comingSoon && !getConnectedProviders(p.id).length).slice(0, 4) as provider}
                  <button class="int-rec-card" onclick={() => { showConnectForm = provider.id; connectCred = ''; connectExtra = ''; }}>
                    <span class="int-rec-icon">{provider.icon}</span>
                    <span class="int-rec-name">{provider.name}</span>
                  </button>
                {/each}
              </div>
            {/if}

            <!-- Setup form (shown when user taps a recommended card) -->
            {#if showConnectForm}
              {@const provider = PROVIDERS.find(p => p.id === showConnectForm)}
              {#if provider}
                <div class="int-setup-card">
                  <div class="int-setup-header">
                    <span>{provider.icon} Connect {provider.name}</span>
                    <button class="int-setup-close" onclick={() => { showConnectForm = null; connectCred = ''; connectExtra = ''; }}>✕</button>
                  </div>
                  <p class="int-setup-desc">{provider.desc}</p>

                  {#if provider.auth === 'oauth'}
                    {#await fetch(`${API}/v1/oauth/config`, { headers: authHeaders() }).then(r => r.json()) then oauthCfg}
                      {#if oauthCfg?.providers?.[provider.id === 'youtube' || provider.id === 'youtube_key' ? 'google' : provider.id]}
                        <!-- OAuth configured — real connect button -->
                        <button class="int-setup-oauth" onclick={() => { window.location.href = `/v1/oauth/${provider.id === 'youtube' || provider.id === 'youtube_key' ? 'google' : provider.id}/authorize`; }}>
                          Connect {provider.name} →
                        </button>
                      {:else}
                        <!-- Not configured — one-click setup -->
                        {#if provider.id === 'github'}
                          <button class="int-setup-oauth" onclick={() => { window.location.href = '/v1/oauth/setup/github'; }}>
                            Set Up GitHub →
                          </button>
                          <p class="int-setup-hint">Creates a GitHub app for your account automatically. One click.</p>
                        {:else if provider.id === 'stripe'}
                          <button class="int-setup-oauth" onclick={() => { window.location.href = '/v1/oauth/setup/stripe'; }}>
                            Set Up Stripe →
                          </button>
                          <p class="int-setup-hint">Opens Stripe to enable Connect for your account.</p>
                        {:else if provider.id === 'freshbooks'}
                          <button class="int-setup-oauth" onclick={() => { window.location.href = '/v1/oauth/setup/freshbooks'; }}>
                            Set Up FreshBooks →
                          </button>
                          <p class="int-setup-hint">Creates a FreshBooks app for your account. One click.</p>
                        {:else if provider.id === 'hubspot'}
                          <button class="int-setup-oauth" onclick={() => { window.location.href = '/v1/oauth/setup/hubspot'; }}>
                            Set Up HubSpot →
                          </button>
                          <p class="int-setup-hint">Creates a HubSpot app for your account. One click.</p>
                        {:else if provider.id === 'google' || provider.id === 'youtube' || provider.id === 'youtube_key'}
                          <p class="int-setup-hint">Google/YouTube connection coming soon</p>
                        {:else}
                          <p class="int-setup-hint">{provider.name} connection coming soon</p>
                        {/if}
                      {/if}
                    {/await}
                  {:else if provider.auth === 'plaid'}
                    <!-- Plaid Link — one button, zero typing -->
                    <button class="int-setup-oauth" onclick={async () => {
                      connectStatus = 'saving'; connectMsg = 'Opening bank connection...';
                      showConnectForm = null;
                      await startPlaidLink(provider);
                    }}>
                      🏦 Connect Bank Account →
                    </button>
                    <p class="int-setup-hint">Opens secure bank login. Supports 12,000+ banks — Novo, Chase, BofA, Wells Fargo, and more.</p>
                  {:else}
                    <div class="int-setup-steps">
                      <div class="int-setup-step">
                        <span class="int-step-num">1</span>
                        <span>{provider.hint}</span>
                      </div>
                      <div class="int-setup-step">
                        <span class="int-step-num">2</span>
                        <span>Paste it below</span>
                      </div>
                    </div>

                    {#if provider.webhookUrl}
                      <div class="int-webhook-url">
                        <span class="int-wh-label">Your webhook URL:</span>
                        <code class="int-wh-code">{API}/v1/webhooks/{provider.id}</code>
                      </div>
                    {/if}

                    <input type={provider.auth === 'rss_url' ? 'url' : 'password'}
                      bind:value={connectCred}
                      placeholder={provider.credPlaceholder || provider.credLabel || 'Paste here'}
                      class="int-setup-input"
                      onkeydown={(e) => {
                        if (e.key === 'Enter' && (!provider.fields?.length || connectExtra)) connectProvider(provider);
                        else if (e.key === 'Enter') document.getElementById('int-extra')?.focus();
                      }} />

                    {#if provider.fields?.length}
                      {#each provider.fields as field}
                        <input type="text" id="int-extra"
                          bind:value={connectExtra}
                          placeholder={field.placeholder || field.label}
                          class="int-setup-input"
                          onkeydown={(e) => e.key === 'Enter' && connectProvider(provider)} />
                      {/each}
                    {/if}

                    <select class="int-setup-biz" bind:value={connectBusiness}>
                      <option value="">All businesses</option>
                      <option value="SEW">Startempire Wire</option>
                      <option value="SEWN">Startempire Wire Network</option>
                      <option value="WIR">Wirebot</option>
                      <option value="PVD">Philoveracity Design</option>
                      <option value="SEW">SEW Network</option>
                    </select>

                    <button class="int-setup-save" onclick={() => connectProvider(provider)}
                      disabled={!connectCred || connectStatus === 'saving'}>
                      {connectStatus === 'saving' ? 'Connecting...' : `Connect ${provider.name}`}
                    </button>
                  {/if}
                </div>
              {/if}
            {/if}

            <!-- Browse all (expandable) -->
            <button class="int-browse-toggle" onclick={() => { const el = document.getElementById('int-all'); el.style.display = el.style.display === 'none' ? 'block' : 'none'; }}>
              Browse all integrations ▾
            </button>
            <div id="int-all" style="display: none;">
              {#each ['revenue', 'shipping', 'distribution', 'systems'] as lane}
                {@const laneProviders = PROVIDERS.filter(p => p.lane === lane)}
                <div class="int-lane-group">
                  <div class="int-lane-header">
                    <span class="int-lane lane-{lane}">{lane}</span>
                  </div>
                  {#each laneProviders as provider}
                    {@const accounts = getConnectedProviders(provider.id)}
                    {@const hasAccounts = accounts.length > 0}
                    <button class="int-browse-item" class:int-connected={hasAccounts} class:int-coming={provider.comingSoon}
                      onclick={() => { if (!provider.comingSoon) { showConnectForm = provider.id; connectCred = ''; connectExtra = ''; } }}>
                      <span class="int-icon">{provider.icon}</span>
                      <span class="int-browse-name">{provider.name}</span>
                      {#if hasAccounts}
                        <span class="int-check">✓ {accounts.length > 1 ? accounts.length : ''}</span>
                      {:else if provider.comingSoon}
                        <span class="int-soon-sm">Soon</span>
                      {/if}
                    </button>
                  {/each}
              </div>
            {/each}
            </div>
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

    <!-- Chat FAB -->
    {#if view === 'dashboard' || view === 'score' || view === 'feed'}
      <button class="fab" onclick={() => showChat = true} title="Ask Wirebot">⚡</button>
    {/if}

    <!-- Wirebot Chat -->
    <Chat bind:visible={showChat} onPairing={selfReportCount === 0 ? () => { showChat = false; showPairing = true; pairingInstrument = ''; } : null} />

    <!-- Pairing Modal (slide-up sheet, same pattern as Chat) -->
    {#if showPairing}
      <div class="pairing-overlay" role="dialog" aria-label="Founder Profile Assessment">
        <div class="pairing-backdrop" onclick={() => showPairing = false} role="presentation"></div>
        <div class="pairing-sheet">
          {#if pairingInstrument}
            <PairingFlow
              instrument={pairingInstrument}
              apiBase={window.location.origin}
              token={getToken()}
              onComplete={() => { pairingInstrument = ''; fetchProfile(); }}
              onBack={() => { pairingInstrument = ''; }}
            />
          {:else}
            <!-- Instrument picker -->
            <div class="pi-header">
              <h2 class="pi-title">🧬 Calibrate Your Profile</h2>
              <button class="pi-close" onclick={() => showPairing = false}>✕</button>
            </div>
            <div class="pi-desc">Each assessment helps Wirebot understand how you operate. Pick any to start.</div>
            <div class="pi-cards">
              <button class="pi-card" onclick={() => pairingInstrument = 'ASI-12'}>
                <span class="pi-icon">⚡</span>
                <div class="pi-info">
                  <div class="pi-name">Action Style</div>
                  <div class="pi-sub">12 forced-choice pairs · 2 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'CSI-8'}>
                <span class="pi-icon">💬</span>
                <div class="pi-info">
                  <div class="pi-name">Communication Style</div>
                  <div class="pi-sub">8 scenario picks · 2 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'ETM-6'}>
                <span class="pi-icon">🔋</span>
                <div class="pi-info">
                  <div class="pi-name">Energy Topology</div>
                  <div class="pi-sub">Drag to sort · 1 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'RDS-6'}>
                <span class="pi-icon">🎲</span>
                <div class="pi-info">
                  <div class="pi-name">Risk Disposition</div>
                  <div class="pi-sub">6 sliders · 1 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'COG-8'}>
                <span class="pi-icon">🧠</span>
                <div class="pi-info">
                  <div class="pi-name">Cognitive Style</div>
                  <div class="pi-sub">8 scenario picks · 2 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'BIZ-6'}>
                <span class="pi-icon">🏢</span>
                <div class="pi-info">
                  <div class="pi-name">Business Reality</div>
                  <div class="pi-sub">6 context questions · 1 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
              <button class="pi-card" onclick={() => pairingInstrument = 'TIME-6'}>
                <span class="pi-icon">⏰</span>
                <div class="pi-info">
                  <div class="pi-name">Temporal Patterns</div>
                  <div class="pi-sub">6 schedule/rhythm questions · 1 min</div>
                </div>
                <span class="pi-go">→</span>
              </button>
            </div>
            <div class="pi-footer">Takes ~10 minutes total. You can do them one at a time.</div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Profile Modal (full equalizer view) -->
    {#if showProfile}
      <div class="pairing-overlay" role="dialog" aria-label="Founder Profile">
        <div class="pairing-backdrop" onclick={() => showProfile = false} role="presentation"></div>
        <div class="pairing-content">
          <div class="pi-header">
            <button class="pi-close" onclick={() => showProfile = false}>✕</button>
            <h2 class="pi-title">🧬 Founder Profile</h2>
          </div>
          <div class="profile-scroll">
            <Profile
              apiBase=""
              token={localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || ''}
              onAssess={(id) => { showProfile = false; pairingInstrument = id; showPairing = true; }}
            />
          </div>
        </div>
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
  .content { flex: 1; overflow-y: auto; padding-bottom: 56px; view-transition-name: content; }

  /* ── View Transitions (native API) ── */
  @view-transition { navigation: auto; }

  /* View Transitions must be :global — they're document-level pseudo-elements */

  /* vt-slide keyframes moved to global <svelte:head> for View Transitions API */

  /* Fallback for browsers without View Transitions API */
  .content.slide-forward {
    animation: fallback-slide-left 350ms cubic-bezier(0.25, 0.46, 0.45, 0.94);
  }
  .content.slide-back {
    animation: fallback-slide-right 350ms cubic-bezier(0.25, 0.46, 0.45, 0.94);
  }
  @keyframes fallback-slide-left {
    0% { opacity: 0.3; transform: translateX(40px); }
    100% { opacity: 1; transform: translateX(0); }
  }
  @keyframes fallback-slide-right {
    0% { opacity: 0.3; transform: translateX(-40px); }
    100% { opacity: 1; transform: translateX(0); }
  }

  .loading { min-height: 100dvh; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; }
  .ld-icon { font-size: 48px; }
  .loading p { font-size: 14px; opacity: 0.5; }
  .err { color: #f44; }

  /* FAB Cluster */
  .fab {
    position: fixed;
    bottom: 72px;
    right: 16px;
    z-index: 50;
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: #7c7cff;
    color: white;
    font-size: 22px;
    border: none;
    cursor: pointer;
    box-shadow: 0 4px 16px rgba(124,124,255,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    -webkit-tap-highlight-color: transparent;
    animation: fabPulse 3s infinite;
    view-transition-name: fab;
  }
  @keyframes fabPulse {
    0%, 100% { box-shadow: 0 4px 16px rgba(124,124,255,0.5); }
    50% { box-shadow: 0 4px 24px rgba(124,124,255,0.8); }
  }

  /* Quick Ship panel removed — events come in automatically */

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

  /* Equalizer strip inside profile card */
  .eq-strip {
    margin: 12px 0 8px;
    padding: 10px 12px;
    background: rgba(124,124,255,0.04);
    border: 1px solid rgba(124,124,255,0.12);
    border-radius: 10px;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
    transition: background 0.2s;
  }
  .eq-strip:active { background: rgba(124,124,255,0.1); }
  .eq-label {
    display: flex; justify-content: space-between; align-items: center;
    font-size: 12px; font-weight: 600; color: #7c7cff; margin-bottom: 8px;
  }
  .eq-arrow { opacity: 0.5; }
  .eq-viz {
    display: flex; gap: 2px; align-items: flex-end; height: 40px;
  }
  .eq-col {
    flex: 1; display: flex; flex-direction: column; align-items: center; gap: 2px;
    min-width: 0;
  }
  .eq-bar-track {
    width: 100%; height: 32px; background: rgba(255,255,255,0.04);
    border-radius: 2px; display: flex; flex-direction: column; justify-content: flex-end;
    overflow: hidden;
  }
  .eq-bar-fill {
    width: 100%; border-radius: 2px 2px 0 0;
    transition: height 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
  .eq-bar-code {
    font-size: 6px; color: #555; letter-spacing: -0.02em;
    overflow: hidden; text-overflow: clip; white-space: nowrap;
    max-width: 100%;
  }
  .eq-foot {
    display: flex; justify-content: space-between;
    font-size: 10px; color: #666; margin-top: 6px;
  }
  .eq-score-line { color: #7c7cff; font-weight: 500; }

  .eq-cta {
    width: 100%; padding: 10px; margin-top: 8px; border-radius: 8px;
    background: rgba(124,124,255,0.15); border: 1px solid rgba(124,124,255,0.4);
    color: #e8e8ff; font-size: 13px; font-weight: 600; cursor: pointer;
    display: flex; align-items: center; gap: 6px; justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }
  .eq-cta-retake { background: rgba(255,255,255,0.05); border-color: #333; color: #999; }
  .eq-cta-sub { font-size: 10px; font-weight: 400; color: #888; }

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

  /* ── Connected Accounts (redesigned) ── */

  /* Active integrations list */
  .int-active-section { display: flex; flex-direction: column; gap: 6px; }
  .int-active-card {
    display: flex; align-items: center; gap: 10px;
    padding: 10px 12px; background: #111118; border: 1px solid #1a3a1a;
    border-radius: 10px;
  }
  .int-active-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
  .int-active-dot.active { background: #2ecc71; box-shadow: 0 0 6px #2ecc7140; }
  .int-active-dot.error { background: #ff4444; }
  .int-active-icon { font-size: 20px; flex-shrink: 0; }
  .int-active-info { flex: 1; min-width: 0; }
  .int-active-name { font-size: 13px; font-weight: 600; color: #ddd; display: flex; align-items: center; gap: 6px; }
  .int-biz-badge {
    font-size: 9px; font-weight: 700; background: rgba(124,124,255,0.1);
    color: #7c7cff; padding: 1px 6px; border-radius: 3px; letter-spacing: 0.05em;
  }
  .int-active-meta { font-size: 11px; color: #555; margin-top: 1px; }
  .int-active-remove {
    background: none; border: none; color: #333; font-size: 14px;
    cursor: pointer; padding: 4px; flex-shrink: 0;
  }
  .int-active-remove:hover { color: #ff4444; }

  /* Add another account */
  .int-add-another {
    display: block; width: 100%; padding: 10px;
    background: none; border: 1px dashed #1e1e30; border-radius: 10px;
    color: #555; font-size: 12px; cursor: pointer;
    text-align: center; transition: all 0.2s;
  }
  .int-add-another:hover { border-color: #7c7cff40; color: #7c7cff; }

  /* Empty state */
  .int-empty { text-align: center; padding: 24px 16px; color: #555; }
  .int-empty-icon { font-size: 32px; display: block; margin-bottom: 8px; }
  .int-empty p { margin: 0; font-size: 14px; }
  .int-empty-hint { font-size: 12px !important; color: #444 !important; margin-top: 4px !important; }

  /* Recommended grid */
  .int-rec-header {
    font-size: 11px; font-weight: 700; color: #666;
    letter-spacing: 0.08em; text-transform: uppercase;
    margin-top: 12px; margin-bottom: 6px;
  }
  .int-rec-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 8px; }
  .int-rec-card {
    display: flex; align-items: center; gap: 8px;
    padding: 12px; background: #111118; border: 1px solid #1e1e30;
    border-radius: 10px; cursor: pointer; transition: border-color 0.2s;
    color: inherit;
  }
  .int-rec-card:hover { border-color: #7c7cff40; }
  .int-rec-icon { font-size: 20px; }
  .int-rec-name { font-size: 12px; font-weight: 600; color: #aaa; }

  /* Setup card (expanded connect form) */
  .int-setup-card {
    background: #111118; border: 1px solid #7c7cff30;
    border-radius: 12px; padding: 16px; margin-top: 8px;
  }
  .int-setup-header {
    display: flex; justify-content: space-between; align-items: center;
    font-size: 14px; font-weight: 700; color: #ddd; margin-bottom: 8px;
  }
  .int-setup-close {
    background: none; border: none; color: #555; font-size: 16px;
    cursor: pointer; padding: 4px;
  }
  .int-setup-desc { font-size: 12px; color: #666; margin: 0 0 12px; }
  .int-setup-steps { display: flex; flex-direction: column; gap: 8px; margin-bottom: 12px; }
  .int-setup-step { display: flex; align-items: flex-start; gap: 8px; font-size: 12px; color: #999; line-height: 1.5; }
  .int-step-num {
    width: 20px; height: 20px; border-radius: 50%;
    background: #7c7cff15; color: #7c7cff; font-size: 10px; font-weight: 700;
    display: flex; align-items: center; justify-content: center; flex-shrink: 0;
  }
  .int-setup-input {
    width: 100%; padding: 10px 12px; background: #0a0a15;
    border: 1px solid #2a2a40; border-radius: 8px;
    color: #ddd; font-size: 13px; outline: none;
    box-sizing: border-box; margin-bottom: 4px;
  }
  .int-setup-input:focus { border-color: #7c7cff; }
  .int-setup-biz {
    width: 100%; padding: 8px 10px; border-radius: 8px;
    background: #0d0d16; border: 1px solid #222;
    color: #aaa; font-size: 12px; margin-bottom: 8px;
    appearance: none; -webkit-appearance: none;
  }
  .int-setup-save {
    width: 100%; padding: 10px; background: #7c7cff; color: white;
    border: none; border-radius: 8px; font-size: 13px; font-weight: 600;
    cursor: pointer;
  }
  .int-setup-save:disabled { opacity: 0.4; cursor: default; }
  .int-setup-save:active:not(:disabled) { background: #5c5cdd; }
  .int-setup-oauth {
    width: 100%; padding: 12px; background: #7c7cff; color: white;
    border: none; border-radius: 8px; font-size: 13px; font-weight: 600;
    cursor: pointer; text-align: center;
  }
  .int-setup-coming { font-size: 12px; color: #555; text-align: center; font-style: italic; }
  .int-setup-hint { font-size: 11px; color: #555; margin: 6px 0 0; line-height: 1.4; }

  /* OAuth setup (cleaned up — no copy-paste) */

  /* Webhook URL display */
  .int-webhook-url {
    background: #0a0a15; border: 1px solid #2a2a40; border-radius: 6px;
    padding: 8px 10px; display: flex; flex-direction: column; gap: 4px; margin-bottom: 4px;
  }
  .int-wh-label { font-size: 10px; color: #555; }
  .int-wh-code {
    font-size: 11px; color: #7c7cff; font-family: monospace;
    word-break: break-all; user-select: all; cursor: text;
  }

  /* Browse all toggle */
  .int-browse-toggle {
    display: block; width: 100%; padding: 10px;
    background: none; border: 1px solid #1e1e30; border-radius: 8px;
    color: #555; font-size: 12px; cursor: pointer; margin-top: 10px;
    text-align: center; transition: border-color 0.2s;
  }
  .int-browse-toggle:hover { border-color: #7c7cff40; color: #888; }

  /* Browse all list */
  .int-lane-group { display: flex; flex-direction: column; gap: 4px; margin-top: 8px; }
  .int-lane-header { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
  .int-lane {
    font-size: 9px; font-weight: 700; padding: 2px 8px; border-radius: 3px;
    text-transform: uppercase; letter-spacing: 0.05em;
  }
  .lane-revenue { background: #1a2a1a; color: #2ecc71; }
  .lane-distribution { background: #1a1a2e; color: #7c7cff; }
  .lane-shipping { background: #2e2a1a; color: #ffaa00; }
  .lane-systems { background: #1a1a1a; color: #888; }

  .int-browse-item {
    display: flex; align-items: center; gap: 10px;
    padding: 8px 10px; background: #111118; border: 1px solid #1e1e30;
    border-radius: 8px; cursor: pointer; transition: border-color 0.15s;
    color: inherit; width: 100%;
  }
  .int-browse-item:hover { border-color: #7c7cff30; }
  .int-browse-item.int-connected { border-color: #1a3a1a; }
  .int-browse-item.int-coming { opacity: 0.35; cursor: default; }
  .int-icon { font-size: 18px; flex-shrink: 0; width: 24px; text-align: center; }
  .int-browse-name { font-size: 12px; color: #aaa; flex: 1; }
  .int-check { color: #2ecc71; font-size: 12px; }
  .int-soon-sm { font-size: 9px; color: #444; font-style: italic; }

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

  /* ── Pairing Modal (full-screen takeover on mobile) ── */
  .pairing-overlay {
    position: fixed; inset: 0; z-index: 1100;
  }
  .pairing-backdrop {
    position: absolute; inset: 0;
    background: rgba(0,0,0,0.6); backdrop-filter: blur(4px);
    animation: backdrop-fade 200ms ease-out;
  }
  @keyframes backdrop-fade {
    from { opacity: 0; backdrop-filter: blur(0); }
    to { opacity: 1; backdrop-filter: blur(4px); }
  }
  .pairing-sheet {
    position: absolute; inset: 0;
    background: #0d0d1a;
    display: flex; flex-direction: column;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    padding-bottom: calc(env(safe-area-inset-bottom, 0px) + 16px);
    animation: pairingFadeIn 0.25s ease-out;
  }
  @keyframes pairingFadeIn {
    from { opacity: 0; transform: translateY(24px); }
    to { opacity: 1; transform: translateY(0); }
  }
  .pairing-content {
    position: absolute; inset: 0;
    background: #0d0d1a;
    display: flex; flex-direction: column;
    overflow-y: auto;
    -webkit-overflow-scrolling: touch;
    animation: pairingFadeIn 0.25s ease-out;
  }
  .profile-scroll {
    flex: 1; overflow-y: auto; padding: 0 16px 80px;
    -webkit-overflow-scrolling: touch;
  }
  .pi-header {
    display: flex; align-items: center; justify-content: space-between;
    padding: 20px 20px 8px; flex-shrink: 0;
    position: sticky; top: 0; background: #0d0d1a; z-index: 2;
  }
  .pi-title { font-size: 20px; font-weight: 700; color: #fff; }
  .pi-close {
    width: 36px; height: 36px; border-radius: 50%;
    background: rgba(255,50,50,0.08); border: 1px solid rgba(255,50,50,0.2);
    color: #ff5050; font-size: 16px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
  }
  .pi-desc {
    padding: 0 20px 16px; font-size: 13px; color: #888; line-height: 1.4;
  }
  .pi-cards {
    display: flex; flex-direction: column; gap: 8px;
    padding: 0 16px;
  }
  .pi-card {
    display: flex; align-items: center; gap: 12px;
    padding: 14px 16px; border-radius: 14px;
    background: rgba(255,255,255,0.03);
    border: 1px solid rgba(255,255,255,0.08);
    cursor: pointer; text-align: left;
    transition: all 0.2s;
    -webkit-tap-highlight-color: transparent;
  }
  .pi-card:hover, .pi-card:active {
    background: rgba(124,124,255,0.08);
    border-color: rgba(124,124,255,0.25);
    transform: scale(1.01);
  }
  .pi-icon { font-size: 24px; flex-shrink: 0; }
  .pi-info { flex: 1; }
  .pi-name { font-size: 15px; font-weight: 600; color: #fff; }
  .pi-sub { font-size: 11px; color: #666; margin-top: 2px; }
  .pi-go { font-size: 16px; color: #7c7cff; opacity: 0.6; }
  .pi-footer {
    text-align: center; padding: 16px; font-size: 12px; color: #555;
  }
</style>

<svelte:head>
  {@html `<style>
    /* View Transition styles — must be global (document-level pseudo-elements) */
    ::view-transition-old(nav),
    ::view-transition-new(nav) {
      animation: none !important;
    }
    ::view-transition-old(content) {
      animation: 300ms cubic-bezier(0.25, 0.46, 0.45, 0.94) both vt-slide-out;
    }
    ::view-transition-new(content) {
      animation: 300ms cubic-bezier(0.25, 0.46, 0.45, 0.94) both vt-slide-in;
    }
    [data-direction="forward"]::view-transition-old(content) {
      animation-name: vt-slide-out-left;
    }
    [data-direction="forward"]::view-transition-new(content) {
      animation-name: vt-slide-in-right;
    }
    [data-direction="back"]::view-transition-old(content) {
      animation-name: vt-slide-out-right;
    }
    [data-direction="back"]::view-transition-new(content) {
      animation-name: vt-slide-in-left;
    }
    @keyframes vt-slide-out-left {
      from { opacity: 1; transform: translateX(0); }
      to { opacity: 0; transform: translateX(-60px); }
    }
    @keyframes vt-slide-in-right {
      from { opacity: 0; transform: translateX(60px); }
      to { opacity: 1; transform: translateX(0); }
    }
    @keyframes vt-slide-out-right {
      from { opacity: 1; transform: translateX(0); }
      to { opacity: 0; transform: translateX(60px); }
    }
    @keyframes vt-slide-in-left {
      from { opacity: 0; transform: translateX(-60px); }
      to { opacity: 1; transform: translateX(0); }
    }
  `}
</svelte:head>
