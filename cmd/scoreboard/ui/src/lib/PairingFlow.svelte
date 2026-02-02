<script>
  let { instrument = '', apiBase = '', token = '', onComplete = () => {}, onBack = () => {} } = $props();

  let step = $state(0);
  let answers = $state([]);
  let submitting = $state(false);
  let submitted = $state(false);
  let transitioning = $state(false);

  // ─── Assessment question banks ─────────────────────────────────────

  const ASI_QUESTIONS = [
    { id: 'ASI-01', a: 'Jump in and figure it out', b: 'Gather all facts first', aCode: 'QS', bCode: 'FF' },
    { id: 'ASI-02', a: 'Start with a rough plan', b: 'Build detailed roadmap', aCode: 'QS', bCode: 'FT' },
    { id: 'ASI-03', a: 'Try multiple approaches quickly', b: 'Perfect one approach', aCode: 'QS', bCode: 'IM' },
    { id: 'ASI-04', a: 'Research before deciding', b: 'Build something tangible', aCode: 'FF', bCode: 'IM' },
    { id: 'ASI-05', a: 'Organize existing resources', b: 'Create new solutions', aCode: 'FT', bCode: 'QS' },
    { id: 'ASI-06', a: 'Prototype fast', b: 'Document thoroughly', aCode: 'QS', bCode: 'FF' },
    { id: 'ASI-07', a: 'Improvise on the fly', b: 'Follow the established process', aCode: 'QS', bCode: 'FT' },
    { id: 'ASI-08', a: 'Delegate to get it built', b: 'Hands-on build it yourself', aCode: 'QS', bCode: 'IM' },
    { id: 'ASI-09', a: 'Act on gut feeling', b: 'Wait for complete data', aCode: 'QS', bCode: 'FF' },
    { id: 'ASI-10', a: 'Systematic follow-up', b: 'Build the physical thing', aCode: 'FT', bCode: 'IM' },
    { id: 'ASI-11', a: 'Deep research phase', b: 'Step-by-step implementation', aCode: 'FF', bCode: 'FT' },
    { id: 'ASI-12', a: 'Create from raw materials', b: 'Optimize existing systems', aCode: 'IM', bCode: 'FT' },
  ];

  const CSI_QUESTIONS = [
    { id: 'CSI-01', prompt: 'A teammate misses a deadline. You…',
      options: [
        { label: 'Address it directly: "We agreed on Tuesday."', code: 'D' },
        { label: 'Encourage: "What can I do to help?"', code: 'I' },
        { label: 'Cover quietly and check in later', code: 'S' },
        { label: 'Review the process that caused it', code: 'C' },
      ]},
    { id: 'CSI-02', prompt: 'You receive critical feedback on your product. You…',
      options: [
        { label: 'Push back with data on why it works', code: 'D' },
        { label: 'Thank them and rally the team around it', code: 'I' },
        { label: 'Accept it calmly and adjust quietly', code: 'S' },
        { label: 'Analyze: is the feedback statistically valid?', code: 'C' },
      ]},
    { id: 'CSI-03', prompt: 'Making a big business decision. You prefer…',
      options: [
        { label: 'Decide quickly, adjust course later', code: 'D' },
        { label: 'Discuss with the team, build consensus', code: 'I' },
        { label: 'Take your time, consider everyone', code: 'S' },
        { label: 'Full analysis with pros/cons matrix', code: 'C' },
      ]},
    { id: 'CSI-04', prompt: 'Pitching to an investor. You lead with…',
      options: [
        { label: 'Bold vision and competitive edge', code: 'D' },
        { label: 'Story, energy, market excitement', code: 'I' },
        { label: 'Reliable team and steady progress', code: 'S' },
        { label: 'Detailed financials and projections', code: 'C' },
      ]},
    { id: 'CSI-05', prompt: 'Team conflict arises. Your instinct is to…',
      options: [
        { label: 'Make a call and move on', code: 'D' },
        { label: 'Mediate and keep morale high', code: 'I' },
        { label: 'Listen to both sides patiently', code: 'S' },
        { label: 'Identify the root process failure', code: 'C' },
      ]},
    { id: 'CSI-06', prompt: 'Celebrating a win. You…',
      options: [
        { label: 'Already planning the next goal', code: 'D' },
        { label: 'Public celebration, shout-outs', code: 'I' },
        { label: 'Quiet satisfaction, thank the team', code: 'S' },
        { label: 'Post-mortem: what made this work?', code: 'C' },
      ]},
    { id: 'CSI-07', prompt: 'Under extreme pressure, you tend to…',
      options: [
        { label: 'Take charge and cut scope', code: 'D' },
        { label: 'Rally the team and keep spirits up', code: 'I' },
        { label: 'Stay calm and keep grinding', code: 'S' },
        { label: 'Double-check everything twice', code: 'C' },
      ]},
    { id: 'CSI-08', prompt: 'Preferred meeting style:',
      options: [
        { label: '15 min, decisions only', code: 'D' },
        { label: 'Brainstorm with energy and ideas', code: 'I' },
        { label: 'Structured agenda, everyone speaks', code: 'S' },
        { label: 'Pre-read docs, discuss exceptions', code: 'C' },
      ]},
  ];

  const RDS_QUESTIONS = [
    { id: 'RDS-01', prompt: 'How comfortable are you betting on an unproven idea?', label: 'Risk Tolerance', min: 'Very cautious', max: 'All-in' },
    { id: 'RDS-02', prompt: 'How fast do you typically make important decisions?', label: 'Decision Speed', min: 'Very deliberate', max: 'Instant' },
    { id: 'RDS-03', prompt: 'When something\'s working, how likely are you to pivot anyway?', label: 'Sunk Cost Immunity', min: 'Stick with it', max: 'Pivot freely' },
    { id: 'RDS-04', prompt: 'How much does potential loss weigh on your decisions?', label: 'Loss Aversion', min: 'Loss looms large', max: 'Barely consider it' },
    { id: 'RDS-05', prompt: 'Do you act even without a clear path forward?', label: 'Bias to Action', min: 'Wait for clarity', max: 'Always move' },
    { id: 'RDS-06', prompt: 'How comfortable are you operating with incomplete information?', label: 'Ambiguity Comfort', min: 'Need full picture', max: 'Thrive in ambiguity' },
  ];

  const ETM_ENERGIES = [
    { code: 'W', name: 'Wonder', desc: 'Asking "why?" and "what if?" — curiosity, questioning, pondering' },
    { code: 'N', name: 'Invention', desc: 'Creating new solutions — building, designing, innovating' },
    { code: 'D_disc', name: 'Discernment', desc: 'Evaluating quality — testing, judging, filtering' },
    { code: 'G', name: 'Galvanizing', desc: 'Rallying people — inspiring, motivating, organizing' },
    { code: 'E', name: 'Enablement', desc: 'Supporting others — helping, training, clearing obstacles' },
    { code: 'T', name: 'Tenacity', desc: 'Finishing work — persistence, follow-through, closing' },
  ];

  const COG_QUESTIONS = [
    { id: 'COG-01', prompt: 'Learning something new, you prefer…',
      options: [
        { label: 'Big picture theory first, then details', code: 'holistic' },
        { label: 'Step-by-step building blocks', code: 'sequential' },
      ]},
    { id: 'COG-02', prompt: 'Explaining a concept, you tend to use…',
      options: [
        { label: 'Metaphors and analogies', code: 'abstract' },
        { label: 'Concrete examples and specifics', code: 'concrete' },
      ]},
    { id: 'COG-03', prompt: 'Planning a project, you start with…',
      options: [
        { label: 'The vision and desired outcome', code: 'holistic' },
        { label: 'The first actionable step', code: 'sequential' },
      ]},
    { id: 'COG-04', prompt: 'Debugging a problem, you…',
      options: [
        { label: 'Think about system-level causes', code: 'holistic' },
        { label: 'Isolate and test each component', code: 'sequential' },
      ]},
    { id: 'COG-05', prompt: 'Reading a business proposal, you focus on…',
      options: [
        { label: 'The narrative and strategic vision', code: 'abstract' },
        { label: 'The numbers and implementation plan', code: 'concrete' },
      ]},
    { id: 'COG-06', prompt: 'Solving a new problem, you tend to…',
      options: [
        { label: 'Draw parallels to unrelated domains', code: 'abstract' },
        { label: 'Find a proven template or case study', code: 'concrete' },
      ]},
    { id: 'COG-07', prompt: 'When priorities conflict, you resolve by…',
      options: [
        { label: 'Stepping back to see which serves the bigger mission', code: 'holistic' },
        { label: 'Ranking them by urgency and knocking out the top one', code: 'sequential' },
      ]},
    { id: 'COG-08', prompt: 'You remember best by…',
      options: [
        { label: 'Understanding the concept behind it', code: 'abstract' },
        { label: 'Doing it yourself hands-on', code: 'concrete' },
      ]},
  ];

  // ─── Business Reality (Φ₅) ─────────────────────────────────────────

  const BIZ_QUESTIONS = [
    { id: 'BIZ-01', prompt: 'How many active businesses are you running right now?',
      options: [
        { label: 'One — fully focused', code: 'focus_single' },
        { label: 'Two — one main, one side', code: 'focus_dual' },
        { label: 'Three or more — serial operator', code: 'focus_multi' },
      ]},
    { id: 'BIZ-02', prompt: 'What best describes your current revenue situation?',
      options: [
        { label: 'Pre-revenue — building the product', code: 'rev_pre' },
        { label: 'Early revenue — some customers, not profitable', code: 'rev_early' },
        { label: 'Sustaining — covering costs, maybe some profit', code: 'rev_sustain' },
        { label: 'Growing — profitable and reinvesting', code: 'rev_growing' },
      ]},
    { id: 'BIZ-03', prompt: 'Do you have a team or are you solo?',
      options: [
        { label: 'Solo — I do everything', code: 'team_solo' },
        { label: 'Solo + contractors when needed', code: 'team_contractors' },
        { label: 'Small team (2-5 people)', code: 'team_small' },
        { label: 'Growing team (6+)', code: 'team_growing' },
      ]},
    { id: 'BIZ-04', prompt: 'What\'s your biggest bottleneck right now?',
      options: [
        { label: 'Building the product / shipping features', code: 'bottle_ship' },
        { label: 'Getting customers / distribution', code: 'bottle_dist' },
        { label: 'Cash flow / revenue', code: 'bottle_rev' },
        { label: 'Operations / systems breaking', code: 'bottle_ops' },
      ]},
    { id: 'BIZ-05', prompt: 'How long have you been working on your current main venture?',
      options: [
        { label: 'Less than 6 months', code: 'age_new' },
        { label: '6 months to 2 years', code: 'age_early' },
        { label: '2-5 years', code: 'age_mid' },
        { label: '5+ years', code: 'age_mature' },
      ]},
    { id: 'BIZ-06', prompt: 'How much personal debt is tied to your business?',
      options: [
        { label: 'None — bootstrapped clean', code: 'debt_none' },
        { label: 'Some — manageable', code: 'debt_some' },
        { label: 'Significant — it weighs on me', code: 'debt_heavy' },
        { label: 'Critical — survival mode', code: 'debt_critical' },
      ]},
  ];

  // ─── Temporal Patterns (Φ₆) ────────────────────────────────────────

  const TIME_QUESTIONS = [
    { id: 'TIME-01', prompt: 'When do you do your best deep work?', label: 'Peak Hours',
      options: [
        { label: 'Early morning (5-9am)', code: 'peak_early' },
        { label: 'Mid-morning (9am-noon)', code: 'peak_mid_am' },
        { label: 'Afternoon (noon-5pm)', code: 'peak_afternoon' },
        { label: 'Evening / Night (5pm+)', code: 'peak_evening' },
      ]},
    { id: 'TIME-02', prompt: 'How do you typically plan your week?', label: 'Planning Style',
      options: [
        { label: 'Detailed calendar blocks for everything', code: 'plan_rigid' },
        { label: 'Rough priorities, flexible execution', code: 'plan_flex' },
        { label: 'I react to what feels most urgent each day', code: 'plan_reactive' },
        { label: 'I don\'t — I just work on whatever pulls me', code: 'plan_flow' },
      ]},
    { id: 'TIME-03', prompt: 'When you hit a wall on something, you…', label: 'Stall Behavior',
      options: [
        { label: 'Push through — force it', code: 'stall_push' },
        { label: 'Switch to something else, come back later', code: 'stall_switch' },
        { label: 'Take a break and reset', code: 'stall_break' },
        { label: 'Ask someone for help or a second opinion', code: 'stall_ask' },
      ]},
    { id: 'TIME-04', prompt: 'How many hours a week do you actually work on your business?', label: 'Work Hours',
      options: [
        { label: 'Under 20 — part-time alongside a job', code: 'hours_part' },
        { label: '20-40 — full-time but bounded', code: 'hours_standard' },
        { label: '40-60 — grinding hard', code: 'hours_heavy' },
        { label: '60+ — all-in, borderline obsessive', code: 'hours_max' },
      ]},
    { id: 'TIME-05', prompt: 'When someone interrupts your focus, you…', label: 'Context Switch Cost',
      options: [
        { label: 'Barely notice — I switch fast', code: 'switch_easy' },
        { label: 'Lose a few minutes but recover', code: 'switch_mild' },
        { label: 'It takes me 20+ minutes to get back in flow', code: 'switch_hard' },
        { label: 'I\'m wrecked for the rest of that session', code: 'switch_critical' },
      ]},
    { id: 'TIME-06', prompt: 'How far ahead do you typically plan?', label: 'Planning Horizon',
      options: [
        { label: 'Today / this week', code: 'horizon_short' },
        { label: 'This month / this quarter', code: 'horizon_mid' },
        { label: 'This year', code: 'horizon_long' },
        { label: '3-5 years out', code: 'horizon_visionary' },
      ]},
  ];

  // ─── Dynamic questions based on instrument ────────────────────────

  function getQuestions() {
    switch (instrument) {
      case 'ASI-12': return ASI_QUESTIONS;
      case 'CSI-8': return CSI_QUESTIONS;
      case 'RDS-6': return RDS_QUESTIONS;
      case 'ETM-6': return [{ id: 'ETM-6', type: 'drag' }];
      case 'COG-8': return COG_QUESTIONS;
      case 'BIZ-6': return BIZ_QUESTIONS;
      case 'TIME-6': return TIME_QUESTIONS;
      default: return [];
    }
  }

  let questions = $derived(getQuestions());
  let currentQ = $derived(questions[step]);
  let progress = $derived(questions.length > 0 ? ((step + (submitted ? 1 : 0)) / questions.length) * 100 : 0);

  // ─── ETM drag-to-sort state ───────────────────────────────────────

  let etmOrder = $state([...ETM_ENERGIES]);
  let dragIndex = $state(-1);

  function etmDragStart(idx) { dragIndex = idx; }
  function etmDragOver(e, idx) {
    e.preventDefault();
    if (dragIndex === idx || dragIndex === -1) return;
    const items = [...etmOrder];
    const [moved] = items.splice(dragIndex, 1);
    items.splice(idx, 0, moved);
    etmOrder = items;
    dragIndex = idx;
  }
  function etmDragEnd() { dragIndex = -1; }

  // ─── RDS slider state ─────────────────────────────────────────────

  let sliderValues = $state({});

  // ─── Answer handling ──────────────────────────────────────────────

  function selectChoice(value) {
    if (transitioning) return;
    answers = [...answers, { instrument_id: instrument, question_id: currentQ.id, value }];
    advance();
  }

  function selectScenario(code) {
    if (transitioning) return;
    answers = [...answers, { instrument_id: instrument, question_id: currentQ.id, value: code }];
    advance();
  }

  function submitSlider(qid, label) {
    const val = sliderValues[qid] ?? 50;
    answers = [...answers, { instrument_id: instrument, question_id: qid, value: val / 10 }];
    advance();
  }

  function submitETM() {
    const order = etmOrder.map(e => e.code);
    answers = [...answers, { instrument_id: instrument, question_id: 'ETM-6', value: order }];
    advance();
  }

  function advance() {
    if (step >= questions.length - 1) {
      submitAll();
    } else {
      transitioning = true;
      setTimeout(() => { step++; transitioning = false; }, 300);
    }
  }

  async function submitAll() {
    submitting = true;
    try {
      await fetch(apiBase + '/v1/pairing/answers', {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ answers }),
      });
      submitted = true;
    } catch (e) {
      console.error('Submit error:', e);
    }
    submitting = false;
  }
</script>

<div class="flow" class:transitioning>
  <!-- Header -->
  <div class="flow-header">
    <button class="back-btn" onclick={onBack}>
      ← Back
    </button>
    <div class="flow-title">
      {instrument === 'ASI-12' ? '⚡ Action Style' :
       instrument === 'CSI-8' ? '💬 Communication' :
       instrument === 'ETM-6' ? '🔋 Energy Topology' :
       instrument === 'RDS-6' ? '🎲 Risk Disposition' :
       instrument === 'COG-8' ? '🧠 Cognitive Style' :
       instrument === 'BIZ-6' ? '🏢 Business Reality' :
       instrument === 'TIME-6' ? '⏰ Temporal Patterns' : ''}
    </div>
    <div class="flow-step">{step + 1}/{questions.length}</div>
  </div>

  <!-- Progress bar -->
  <div class="progress-track">
    <div class="progress-fill" style="width:{progress}%"></div>
  </div>

  {#if submitted}
    <!-- Success -->
    <div class="flow-done">
      <div class="done-icon">✅</div>
      <h3>Assessment Complete</h3>
      <p>{answers.length} answers recorded. Profile updating...</p>
      <button class="done-btn" onclick={onComplete}>View Profile</button>
    </div>
  {:else if submitting}
    <div class="flow-submitting">
      <div class="pulse">🧬</div>
      <p>Processing answers...</p>
    </div>
  {:else if currentQ}
    <div class="question-area" class:slide-out={transitioning}>

      {#if instrument === 'ASI-12'}
        <!-- Forced Choice Pair -->
        <div class="q-prompt">Which describes you more?</div>
        <div class="choice-cards">
          <button class="choice-card" onclick={() => selectChoice('A')}>
            <span class="choice-letter">A</span>
            <span class="choice-text">{currentQ.a}</span>
            <span class="choice-dim">{currentQ.aCode}</span>
          </button>
          <div class="choice-vs">vs</div>
          <button class="choice-card" onclick={() => selectChoice('B')}>
            <span class="choice-letter">B</span>
            <span class="choice-text">{currentQ.b}</span>
            <span class="choice-dim">{currentQ.bCode}</span>
          </button>
        </div>

      {:else if instrument === 'CSI-8'}
        <!-- Scenario Pick -->
        <div class="q-prompt">{currentQ.prompt}</div>
        <div class="scenario-options">
          {#each currentQ.options as opt}
            <button class="scenario-btn" onclick={() => selectScenario(opt.code)}>
              <span class="s-label">{opt.label}</span>
            </button>
          {/each}
        </div>

      {:else if instrument === 'COG-8'}
        <!-- Binary Scenario -->
        <div class="q-prompt">{currentQ.prompt}</div>
        <div class="scenario-options">
          {#each currentQ.options as opt}
            <button class="scenario-btn" onclick={() => selectScenario(opt.code)}>
              <span class="s-label">{opt.label}</span>
            </button>
          {/each}
        </div>

      {:else if instrument === 'BIZ-6' || instrument === 'TIME-6'}
        <!-- Scenario Pick (same UI) -->
        <div class="q-prompt">{currentQ.prompt}</div>
        <div class="scenario-options">
          {#each currentQ.options as opt}
            <button class="scenario-btn" onclick={() => selectScenario(opt.code)}>
              <span class="s-label">{opt.label}</span>
            </button>
          {/each}
        </div>

      {:else if instrument === 'RDS-6'}
        <!-- Slider -->
        <div class="q-prompt">{currentQ.prompt}</div>
        <div class="slider-area">
          <div class="slider-labels">
            <span class="s-min">{currentQ.min}</span>
            <span class="s-max">{currentQ.max}</span>
          </div>
          <input type="range" min="0" max="100" step="5"
            value={sliderValues[currentQ.id] ?? 50}
            oninput={(e) => sliderValues = {...sliderValues, [currentQ.id]: parseInt(e.target.value)}} />
          <div class="slider-value">{sliderValues[currentQ.id] ?? 50}</div>
          <button class="slider-submit" onclick={() => submitSlider(currentQ.id, currentQ.label)}>
            Confirm →
          </button>
        </div>

      {:else if instrument === 'ETM-6'}
        <!-- Drag to Sort -->
        <div class="q-prompt">Drag to sort by what gives you ENERGY (top = most)</div>
        <div class="drag-list">
          {#each etmOrder as energy, idx}
            <div class="drag-item"
              class:dragging={dragIndex === idx}
              draggable="true"
              ondragstart={() => etmDragStart(idx)}
              ondragover={(e) => etmDragOver(e, idx)}
              ondragend={etmDragEnd}
              role="listitem"
            >
              <span class="drag-rank">#{idx + 1}</span>
              <div class="drag-info">
                <div class="drag-name">{energy.name}</div>
                <div class="drag-desc">{energy.desc}</div>
              </div>
              <span class="drag-handle">⠿</span>
            </div>
          {/each}
        </div>
        <button class="etm-submit" onclick={submitETM}>
          Lock Order →
        </button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .flow { padding: 0 16px 24px; max-width: 500px; margin: 0 auto; }

  .flow-header { display: flex; align-items: center; gap: 8px; padding: 16px 0 12px; }
  .back-btn {
    background: rgba(124,124,255,0.08); border: 1px solid rgba(124,124,255,0.15);
    color: #7c7cff; font-size: 14px; cursor: pointer;
    padding: 6px 12px; border-radius: 8px;
    -webkit-tap-highlight-color: transparent;
  }
  .flow-title { flex: 1; font-size: 16px; font-weight: 600; color: #fff; text-align: center; }
  .flow-step { font-size: 12px; color: #888; font-variant-numeric: tabular-nums; }

  .progress-track { height: 3px; background: rgba(255,255,255,0.08); border-radius: 2px; margin-bottom: 24px; overflow: hidden; }
  .progress-fill { height: 100%; background: linear-gradient(90deg, #7c7cff, #ff7cff); transition: width 0.4s ease; border-radius: 2px; }

  /* Question area transition */
  .question-area { transition: opacity 0.3s, transform 0.3s; }
  .question-area.slide-out { opacity: 0; transform: translateX(-20px); }

  .q-prompt { font-size: 18px; font-weight: 500; color: #fff; margin-bottom: 20px; line-height: 1.4; }

  /* ── Forced Choice (ASI) ── */
  .choice-cards { display: flex; flex-direction: column; gap: 12px; }
  .choice-card {
    display: flex; align-items: center; gap: 12px;
    padding: 16px; border-radius: 12px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.1);
    cursor: pointer; color: #fff; text-align: left;
    transition: all 0.2s;
  }
  .choice-card:hover, .choice-card:active {
    background: rgba(124,124,255,0.12);
    border-color: rgba(124,124,255,0.4);
    transform: scale(1.01);
  }
  .choice-letter {
    width: 32px; height: 32px; border-radius: 50%;
    background: rgba(124,124,255,0.15); color: #7c7cff;
    display: flex; align-items: center; justify-content: center;
    font-weight: 700; font-size: 14px; flex-shrink: 0;
  }
  .choice-text { flex: 1; font-size: 15px; line-height: 1.3; }
  .choice-dim { font-size: 10px; color: #555; }
  .choice-vs { text-align: center; color: #555; font-size: 12px; }

  /* ── Scenario Pick (CSI, COG) ── */
  .scenario-options { display: flex; flex-direction: column; gap: 8px; }
  .scenario-btn {
    padding: 14px 16px; border-radius: 12px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.1);
    cursor: pointer; color: #fff; text-align: left;
    transition: all 0.2s;
  }
  .scenario-btn:hover, .scenario-btn:active {
    background: rgba(124,124,255,0.12);
    border-color: rgba(124,124,255,0.4);
  }
  .s-label { font-size: 14px; line-height: 1.4; }

  /* ── Slider (RDS) ── */
  .slider-area { display: flex; flex-direction: column; gap: 12px; }
  .slider-labels { display: flex; justify-content: space-between; }
  .s-min, .s-max { font-size: 11px; color: #888; }
  input[type="range"] {
    width: 100%; height: 6px; -webkit-appearance: none; appearance: none;
    background: rgba(255,255,255,0.1); border-radius: 3px; outline: none;
  }
  input[type="range"]::-webkit-slider-thumb {
    -webkit-appearance: none; width: 28px; height: 28px; border-radius: 50%;
    background: #7c7cff; cursor: pointer; border: 3px solid #1a1a2e;
  }
  .slider-value { text-align: center; font-size: 32px; font-weight: 700; color: #7c7cff; }
  .slider-submit {
    padding: 12px; border-radius: 10px; background: #7c7cff; color: #fff;
    border: none; font-size: 15px; font-weight: 600; cursor: pointer;
    transition: transform 0.15s;
  }
  .slider-submit:active { transform: scale(0.97); }

  /* ── Drag to Sort (ETM) ── */
  .drag-list { display: flex; flex-direction: column; gap: 4px; margin-bottom: 16px; }
  .drag-item {
    display: flex; align-items: center; gap: 10px;
    padding: 12px; border-radius: 10px;
    background: rgba(255,255,255,0.04);
    border: 1px solid rgba(255,255,255,0.08);
    cursor: grab; user-select: none;
    transition: background 0.2s, transform 0.15s;
  }
  .drag-item:active { cursor: grabbing; }
  .drag-item.dragging { opacity: 0.5; background: rgba(124,124,255,0.1); }
  .drag-rank {
    font-size: 14px; font-weight: 700; color: #7c7cff;
    width: 24px; text-align: center;
  }
  .drag-info { flex: 1; }
  .drag-name { font-size: 14px; font-weight: 600; color: #fff; }
  .drag-desc { font-size: 11px; color: #888; margin-top: 2px; }
  .drag-handle { color: #555; font-size: 18px; }
  .etm-submit {
    width: 100%; padding: 14px; border-radius: 10px;
    background: #7c7cff; color: #fff; border: none;
    font-size: 15px; font-weight: 600; cursor: pointer;
  }

  /* ── Done ── */
  .flow-done, .flow-submitting { text-align: center; padding: 60px 20px; }
  .done-icon { font-size: 48px; margin-bottom: 16px; }
  .flow-done h3 { color: #fff; font-size: 20px; margin: 0 0 8px; }
  .flow-done p { color: #888; font-size: 14px; margin: 0 0 24px; }
  .done-btn {
    padding: 12px 32px; border-radius: 10px;
    background: #7c7cff; color: #fff; border: none;
    font-size: 15px; font-weight: 600; cursor: pointer;
  }
  .pulse { font-size: 40px; animation: pulse 1.5s ease-in-out infinite; }
  @keyframes pulse { 0%,100% { opacity: 0.5; } 50% { opacity: 1; } }
  .flow-submitting p { color: #888; font-size: 14px; margin-top: 12px; }

  /* Global transition */
  .flow.transitioning .question-area { opacity: 0; }
</style>
