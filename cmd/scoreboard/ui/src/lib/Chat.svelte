<script>
  let { visible = $bindable(false) } = $props();
  let messages = $state([]);
  let input = $state('');
  let sending = $state(false);
  let chatBody = $state(null);

  function getToken() {
    return localStorage.getItem('rl_jwt') || localStorage.getItem('operator_token') || '';
  }

  function scrollBottom() {
    setTimeout(() => {
      if (chatBody) chatBody.scrollTop = chatBody.scrollHeight;
    }, 50);
  }

  async function send() {
    const text = input.trim();
    if (!text || sending) return;
    input = '';

    messages.push({ role: 'user', content: text });
    messages = messages; // trigger reactivity
    scrollBottom();
    sending = true;

    try {
      const token = getToken();
      // Build message history for context
      const apiMessages = messages.filter(m => m.role !== 'error').map(m => ({
        role: m.role,
        content: m.content,
      }));

      const resp = await fetch('/v1/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': token ? (token.startsWith('Bearer ') ? token : `Bearer ${token}`) : '',
        },
        body: JSON.stringify({ messages: apiMessages }),
      });

      if (!resp.ok) {
        const err = await resp.json().catch(() => ({ error: resp.statusText }));
        messages.push({ role: 'error', content: err.error || 'Request failed' });
      } else {
        const data = await resp.json();
        const reply = data.choices?.[0]?.message?.content || '(no response)';
        messages.push({ role: 'assistant', content: reply });
      }
    } catch (e) {
      messages.push({ role: 'error', content: 'Connection failed — is Wirebot running?' });
    }

    messages = messages;
    sending = false;
    scrollBottom();
  }

  function onKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      send();
    }
  }

  function close() {
    visible = false;
  }
</script>

{#if visible}
  <div class="chat-overlay" role="dialog" aria-label="Wirebot Chat">
    <!-- Backdrop -->
    <div class="chat-backdrop" onclick={close} role="presentation"></div>

    <!-- Bottom Sheet -->
    <div class="chat-sheet">
      <div class="chat-header">
        <span class="chat-title">⚡ Wirebot</span>
        <span class="chat-subtitle">Full conversation • Memory retained</span>
        <button class="chat-close" onclick={close}>✕</button>
      </div>

      <div class="chat-body" bind:this={chatBody}>
        {#if messages.length === 0}
          <div class="chat-empty">
            <div class="ce-icon">⚡</div>
            <div class="ce-text">Ask anything. Score context, business strategy, code help — full Wirebot.</div>
            <div class="ce-hints">
              <button class="ce-hint" onclick={() => { input = "How am I doing today?"; send(); }}>How am I doing?</button>
              <button class="ce-hint" onclick={() => { input = "What should I ship next?"; send(); }}>What should I ship?</button>
              <button class="ce-hint" onclick={() => { input = "Show me my revenue breakdown"; send(); }}>Revenue breakdown</button>
            </div>
          </div>
        {/if}

        {#each messages as msg, i}
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
    </div>
  </div>
{/if}

<style>
  .chat-overlay {
    position: fixed;
    inset: 0;
    z-index: 1000;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
  }
  .chat-backdrop {
    position: absolute;
    inset: 0;
    background: rgba(0,0,0,0.5);
    backdrop-filter: blur(2px);
  }
  .chat-sheet {
    position: relative;
    background: #0d0d1a;
    border-top: 1px solid #2a2a4a;
    border-radius: 16px 16px 0 0;
    display: flex;
    flex-direction: column;
    max-height: 75dvh;
    min-height: 40dvh;
    animation: slideUp 0.25s ease-out;
  }
  @keyframes slideUp {
    from { transform: translateY(100%); }
    to { transform: translateY(0); }
  }
  .chat-header {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px 16px;
    border-bottom: 1px solid #1e1e30;
    flex-shrink: 0;
  }
  .chat-title {
    font-size: 15px;
    font-weight: 700;
    color: #7c7cff;
  }
  .chat-subtitle {
    font-size: 11px;
    color: #555;
    flex: 1;
  }
  .chat-close {
    width: 28px; height: 28px;
    border-radius: 50%;
    background: rgba(255,50,50,0.1);
    border: none;
    color: #ff5050;
    font-size: 14px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .chat-body {
    flex: 1;
    overflow-y: auto;
    padding: 12px 16px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }
  .chat-empty {
    text-align: center;
    padding: 24px 0;
    color: #555;
  }
  .ce-icon { font-size: 40px; margin-bottom: 8px; }
  .ce-text { font-size: 13px; max-width: 280px; margin: 0 auto 16px; line-height: 1.5; }
  .ce-hints { display: flex; flex-wrap: wrap; gap: 6px; justify-content: center; }
  .ce-hint {
    background: rgba(124,124,255,0.08);
    border: 1px solid rgba(124,124,255,0.2);
    border-radius: 16px;
    padding: 6px 12px;
    font-size: 12px;
    color: #7c7cff;
    cursor: pointer;
    -webkit-tap-highlight-color: transparent;
  }
  .ce-hint:active { background: rgba(124,124,255,0.2); }

  .chat-msg {
    display: flex;
    gap: 8px;
    align-items: flex-start;
  }
  .chat-msg.user { justify-content: flex-end; }
  .chat-msg.error { justify-content: center; }
  .msg-avatar {
    width: 24px; height: 24px;
    border-radius: 50%;
    background: rgba(124,124,255,0.15);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 12px;
    flex-shrink: 0;
    margin-top: 2px;
  }
  .msg-bubble {
    max-width: 85%;
    padding: 8px 12px;
    border-radius: 12px;
    font-size: 13px;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
  }
  .user .msg-bubble {
    background: #7c7cff;
    color: #fff;
    border-bottom-right-radius: 4px;
  }
  .assistant .msg-bubble {
    background: #1a1a2e;
    color: #ccc;
    border-bottom-left-radius: 4px;
  }
  .error .msg-bubble {
    background: rgba(255,50,50,0.1);
    color: #ff5050;
    font-size: 12px;
  }
  .typing {
    display: flex;
    gap: 4px;
    padding: 10px 16px;
  }
  .dot {
    width: 6px; height: 6px;
    border-radius: 50%;
    background: #555;
    animation: bounce 1.2s infinite;
  }
  .dot:nth-child(2) { animation-delay: 0.2s; }
  .dot:nth-child(3) { animation-delay: 0.4s; }
  @keyframes bounce {
    0%, 60%, 100% { transform: translateY(0); }
    30% { transform: translateY(-6px); }
  }

  .chat-input-row {
    display: flex;
    gap: 8px;
    padding: 8px 12px;
    padding-bottom: max(8px, env(safe-area-inset-bottom));
    border-top: 1px solid #1e1e30;
    flex-shrink: 0;
  }
  .chat-input {
    flex: 1;
    background: #1a1a2e;
    border: 1px solid #2a2a4a;
    border-radius: 12px;
    padding: 10px 14px;
    color: #ddd;
    font-size: 14px;
    resize: none;
    outline: none;
    font-family: system-ui, -apple-system, sans-serif;
  }
  .chat-input::placeholder { color: #444; }
  .chat-input:focus { border-color: #7c7cff; }
  .chat-send {
    width: 42px; height: 42px;
    border-radius: 50%;
    background: #7c7cff;
    border: none;
    color: #fff;
    font-size: 18px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    -webkit-tap-highlight-color: transparent;
  }
  .chat-send:disabled { opacity: 0.3; cursor: default; }
  .chat-send:not(:disabled):active { background: #5c5cee; }
</style>
