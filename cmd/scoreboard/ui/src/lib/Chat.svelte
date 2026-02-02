<script>
  import { onMount } from 'svelte';
  let { visible = $bindable(false) } = $props();

  let messages = $state([]);
  let input = $state('');
  let sending = $state(false);
  let chatBody = $state(null);
  let sessionID = $state('');
  let sessions = $state([]);
  let showHistory = $state(false);
  let pairing = $state(null); // { completed, score, answered, total }

  function getToken() {
    return localStorage.getItem('wb_token') || localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || '';
  }

  function authHeaders() {
    const t = getToken();
    return t ? { 'Authorization': t.startsWith('Bearer ') ? t : `Bearer ${t}` } : {};
  }

  function scrollBottom() {
    setTimeout(() => {
      if (chatBody) chatBody.scrollTop = chatBody.scrollHeight;
    }, 50);
  }

  async function loadSessions() {
    try {
      const resp = await fetch('/v1/chat/sessions', { headers: authHeaders() });
      if (resp.ok) sessions = await resp.json();
    } catch {}
  }

  async function loadSession(id) {
    try {
      const resp = await fetch(`/v1/chat/sessions/${id}`, { headers: authHeaders() });
      if (resp.ok) {
        const data = await resp.json();
        sessionID = data.id;
        messages = (data.messages || []).map(m => ({
          role: m.role,
          content: m.content,
        }));
        showHistory = false;
        scrollBottom();
      }
    } catch {}
  }

  async function newChat() {
    sessionID = '';
    messages = [];
    showHistory = false;
  }

  async function deleteSession(id) {
    try {
      await fetch(`/v1/chat/sessions/${id}`, { method: 'DELETE', headers: authHeaders() });
      sessions = sessions.filter(s => s.id !== id);
      if (sessionID === id) newChat();
    } catch {}
  }

  async function checkPairing() {
    try {
      const resp = await fetch('/v1/pairing/status', { headers: authHeaders() });
      if (resp.ok) pairing = await resp.json();
    } catch {}
  }

  onMount(() => {
    checkPairing();
    // Auto-load most recent session
    loadSessions().then(() => {
      if (sessions.length > 0) loadSession(sessions[0].id);
    });
  });

  async function send() {
    const text = input.trim();
    if (!text || sending) return;
    input = '';

    messages.push({ role: 'user', content: text });
    messages = messages;
    scrollBottom();
    sending = true;

    try {
      const apiMessages = messages.filter(m => m.role !== 'error').map(m => ({
        role: m.role, content: m.content,
      }));

      const resp = await fetch('/v1/chat', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', ...authHeaders() },
        body: JSON.stringify({ messages: apiMessages, session_id: sessionID, stream: true }),
      });

      if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: resp.statusText }));
        messages.push({ role: 'error', content: err.error || 'Request failed' });
      } else {
        messages.push({ role: 'assistant', content: '' });
        messages = messages;
        const aidx = messages.length - 1;

        const reader = resp.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });

          const lines = buffer.split('\n');
          buffer = lines.pop() || '';

          for (const line of lines) {
            // Capture session ID from custom event
            if (line.startsWith('event: session')) continue;
            if (line.startsWith('data: ') && !sessionID) {
              const d = line.slice(6).trim();
              // Check if it's a plain session ID (hex string, not JSON)
              if (/^[a-f0-9]{24}$/.test(d)) {
                sessionID = d;
                continue;
              }
            }
            if (!line.startsWith('data: ')) continue;
            const payload = line.slice(6).trim();
            if (payload === '[DONE]') continue;
            try {
              const chunk = JSON.parse(payload);
              const delta = chunk.choices?.[0]?.delta?.content;
              if (delta) {
                messages[aidx].content += delta;
                messages = messages;
                scrollBottom();
              }
            } catch {}
          }
        }

        if (!messages[aidx].content) {
          messages[aidx].content = '(no response)';
        }
      }
    } catch (e) {
      messages.push({ role: 'error', content: 'Connection failed — is Wirebot running?' });
    }

    messages = messages;
    sending = false;
    scrollBottom();
    loadSessions(); // Refresh sidebar
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); send(); }
  }

  function close() { visible = false; }

  function timeAgo(ts) {
    if (!ts) return '';
    const diff = (Date.now() - new Date(ts).getTime()) / 1000;
    if (diff < 60) return 'just now';
    if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
    if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
    return Math.floor(diff / 86400) + 'd ago';
  }
</script>

{#if visible}
  <div class="chat-overlay" role="dialog" aria-label="Wirebot Chat">
    <div class="chat-backdrop" onclick={close} role="presentation"></div>

    <div class="chat-sheet">
      <!-- Header -->
      <div class="chat-header">
        <button class="ch-btn" onclick={() => { showHistory = !showHistory; if (showHistory) loadSessions(); }} title="Chat history">
          {showHistory ? '←' : '☰'}
        </button>
        <div class="ch-title-area">
          <span class="chat-title">⚡ Wirebot</span>
          {#if sessionID}
            <span class="chat-subtitle">Session active</span>
          {:else}
            <span class="chat-subtitle">New conversation</span>
          {/if}
        </div>
        <button class="ch-btn" onclick={newChat} title="New chat">＋</button>
        <button class="chat-close" onclick={close}>✕</button>
      </div>

      <!-- Pairing nudge -->
      {#if pairing && !pairing.completed}
        <div class="pairing-nudge" onclick={() => { input = "Let's do the pairing questionnaire"; send(); }}>
          <span class="pn-icon">🤝</span>
          <span class="pn-text">
            {#if pairing.answered === 0}
              Pair with Wirebot to unlock full context
            {:else}
              Pairing {pairing.answered}/{pairing.total} — continue setup
            {/if}
          </span>
          <span class="pn-arrow">→</span>
        </div>
      {/if}

      <!-- History drawer -->
      {#if showHistory}
        <div class="history-panel">
          <div class="hp-title">Conversations</div>
          {#if sessions.length === 0}
            <div class="hp-empty">No conversations yet</div>
          {/if}
          {#each sessions as s}
            <div class="hp-item {s.id === sessionID ? 'active' : ''}" onclick={() => loadSession(s.id)}>
              <div class="hp-item-title">{s.title}</div>
              <div class="hp-item-meta">
                {s.message_count} msgs · {timeAgo(s.updated_at)}
              </div>
              <button class="hp-del" onclick={(e) => { e.stopPropagation(); deleteSession(s.id); }} title="Delete">🗑</button>
            </div>
          {/each}
        </div>
      {:else}
        <!-- Chat body -->
        <div class="chat-body" bind:this={chatBody}>
          {#if messages.length === 0}
            <div class="chat-empty">
              <div class="ce-icon">⚡</div>
              <div class="ce-text">Full Wirebot — score context, strategy, memory, tools.</div>
              <div class="ce-hints">
                <button class="ce-hint" onclick={() => { input = "How am I doing today?"; send(); }}>How am I doing?</button>
                <button class="ce-hint" onclick={() => { input = "What should I ship next?"; send(); }}>What should I ship?</button>
                <button class="ce-hint" onclick={() => { input = "Show my revenue breakdown"; send(); }}>Revenue breakdown</button>
              </div>
            </div>
          {/if}

          {#each messages as msg}
            <div class="chat-msg {msg.role}">
              {#if msg.role === 'assistant'}
                <span class="msg-avatar">⚡</span>
              {/if}
              <div class="msg-bubble">
                {#if msg.role === 'error'}
                  ⚠️ {msg.content}
                {:else}
                  {msg.content}
                {/if}
              </div>
            </div>
          {/each}

          {#if sending}
            <div class="chat-msg assistant">
              <span class="msg-avatar">⚡</span>
              <div class="msg-bubble typing">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
              </div>
            </div>
          {/if}
        </div>

        <!-- Input -->
        <div class="chat-input-row">
          <textarea
            class="chat-input"
            placeholder="Message Wirebot..."
            bind:value={input}
            onkeydown={onKeydown}
            disabled={sending}
            rows="1"
          ></textarea>
          <button class="chat-send" onclick={send} disabled={sending || !input.trim()}>
            {sending ? '...' : '→'}
          </button>
        </div>
      {/if}
    </div>
  </div>
{/if}

<style>
  .chat-overlay {
    position: fixed; inset: 0; z-index: 1000;
    display: flex; flex-direction: column; justify-content: flex-end;
  }
  .chat-backdrop {
    position: absolute; inset: 0;
    background: rgba(0,0,0,0.5); backdrop-filter: blur(2px);
  }
  .chat-sheet {
    position: relative;
    background: #0d0d1a;
    border-top: 1px solid #2a2a4a;
    border-radius: 16px 16px 0 0;
    display: flex; flex-direction: column;
    max-height: 80dvh; min-height: 40dvh;
    animation: slideUp 0.25s ease-out;
  }
  @keyframes slideUp {
    from { transform: translateY(100%); } to { transform: translateY(0); }
  }

  /* Header */
  .chat-header {
    display: flex; align-items: center; gap: 8px;
    padding: 10px 12px; border-bottom: 1px solid #1e1e30; flex-shrink: 0;
  }
  .ch-btn {
    width: 32px; height: 32px; border-radius: 8px;
    background: rgba(124,124,255,0.08); border: 1px solid rgba(124,124,255,0.15);
    color: #7c7cff; font-size: 16px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    -webkit-tap-highlight-color: transparent;
  }
  .ch-btn:active { background: rgba(124,124,255,0.2); }
  .ch-title-area { flex: 1; }
  .chat-title { font-size: 15px; font-weight: 700; color: #7c7cff; }
  .chat-subtitle { display: block; font-size: 10px; color: #444; margin-top: 1px; }
  .chat-close {
    width: 32px; height: 32px; border-radius: 8px;
    background: rgba(255,50,50,0.08); border: 1px solid rgba(255,50,50,0.15);
    color: #ff5050; font-size: 14px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
  }

  /* Pairing nudge */
  .pairing-nudge {
    display: flex; align-items: center; gap: 8px;
    padding: 8px 14px;
    background: rgba(255,180,50,0.06);
    border-bottom: 1px solid rgba(255,180,50,0.12);
    cursor: pointer; flex-shrink: 0;
    -webkit-tap-highlight-color: transparent;
  }
  .pairing-nudge:active { background: rgba(255,180,50,0.12); }
  .pn-icon { font-size: 14px; }
  .pn-text { flex: 1; font-size: 12px; color: #c9a020; }
  .pn-arrow { font-size: 12px; color: #c9a020; opacity: 0.6; }

  /* History panel */
  .history-panel {
    flex: 1; overflow-y: auto; padding: 8px;
  }
  .hp-title { font-size: 13px; font-weight: 700; color: #666; padding: 4px 8px 8px; }
  .hp-empty { text-align: center; color: #444; font-size: 13px; padding: 24px 0; }
  .hp-item {
    display: flex; flex-wrap: wrap; align-items: center;
    padding: 10px 12px; border-radius: 8px; cursor: pointer;
    margin-bottom: 2px; position: relative;
    -webkit-tap-highlight-color: transparent;
  }
  .hp-item:hover, .hp-item:active { background: rgba(124,124,255,0.06); }
  .hp-item.active { background: rgba(124,124,255,0.1); border: 1px solid rgba(124,124,255,0.2); }
  .hp-item-title {
    flex: 1; font-size: 13px; color: #ccc;
    white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
    min-width: 0;
  }
  .hp-item-meta { width: 100%; font-size: 10px; color: #555; margin-top: 2px; }
  .hp-del {
    position: absolute; right: 8px; top: 8px;
    width: 24px; height: 24px; border: none; background: none;
    font-size: 12px; cursor: pointer; opacity: 0.3;
    display: flex; align-items: center; justify-content: center;
  }
  .hp-del:hover { opacity: 1; }

  /* Chat body */
  .chat-body {
    flex: 1; overflow-y: auto; padding: 12px 14px;
    display: flex; flex-direction: column; gap: 10px;
  }
  .chat-empty { text-align: center; padding: 24px 0; color: #555; }
  .ce-icon { font-size: 36px; margin-bottom: 8px; }
  .ce-text { font-size: 13px; max-width: 280px; margin: 0 auto 14px; line-height: 1.5; }
  .ce-hints { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; }
  .ce-hint {
    background: rgba(124,124,255,0.08); border: 1px solid rgba(124,124,255,0.2);
    border-radius: 16px; padding: 6px 12px; font-size: 12px;
    color: #7c7cff; cursor: pointer; -webkit-tap-highlight-color: transparent;
  }
  .ce-hint:active { background: rgba(124,124,255,0.2); }

  /* Messages */
  .chat-msg { display: flex; gap: 8px; align-items: flex-start; }
  .chat-msg.user { justify-content: flex-end; }
  .chat-msg.error { justify-content: center; }
  .msg-avatar {
    width: 24px; height: 24px; border-radius: 50%;
    background: rgba(124,124,255,0.15);
    display: flex; align-items: center; justify-content: center;
    font-size: 12px; flex-shrink: 0; margin-top: 2px;
  }
  .msg-bubble {
    max-width: 85%; padding: 8px 12px; border-radius: 12px;
    font-size: 13px; line-height: 1.5;
    white-space: pre-wrap; word-break: break-word;
  }
  .user .msg-bubble {
    background: #7c7cff; color: #fff; border-bottom-right-radius: 4px;
  }
  .assistant .msg-bubble {
    background: #1a1a2e; color: #ccc; border-bottom-left-radius: 4px;
  }
  .error .msg-bubble {
    background: rgba(255,50,50,0.1); color: #ff5050; font-size: 12px;
  }
  .typing { display: flex; gap: 4px; padding: 10px 16px; }
  .dot {
    width: 6px; height: 6px; border-radius: 50%;
    background: #555; animation: bounce 1.2s infinite;
  }
  .dot:nth-child(2) { animation-delay: 0.2s; }
  .dot:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce {
    0%, 60%, 100% { transform: translateY(0); } 30% { transform: translateY(-6px); }
  }

  /* Input */
  .chat-input-row {
    display: flex; gap: 8px; padding: 8px 12px;
    padding-bottom: max(8px, env(safe-area-inset-bottom));
    border-top: 1px solid #1e1e30; flex-shrink: 0;
  }
  .chat-input {
    flex: 1; background: #1a1a2e; border: 1px solid #2a2a4a;
    border-radius: 12px; padding: 10px 14px; color: #ddd;
    font-size: 14px; resize: none; outline: none;
    font-family: system-ui, -apple-system, sans-serif;
  }
  .chat-input::placeholder { color: #444; }
  .chat-input:focus { border-color: #7c7cff; }
  .chat-send {
    width: 42px; height: 42px; border-radius: 50%;
    background: #7c7cff; border: none; color: #fff;
    font-size: 18px; cursor: pointer;
    display: flex; align-items: center; justify-content: center;
    flex-shrink: 0; -webkit-tap-highlight-color: transparent;
  }
  .chat-send:disabled { opacity: 0.3; cursor: default; }
  .chat-send:not(:disabled):active { background: #5c5cee; }
</style>
