# SingleEye × Focusa — Overlap Analysis

> Two systems designed by the same mind, at different times, for different contexts — but converging on the same deep problems.

---

## TL;DR

**SingleEye** is a synthetic mind framework — brain-inspired, embodied, companion-bonded.
**Focusa** is a cognitive governance runtime — deterministic, observable, proxy-based.

They overlap **heavily** on core problems but approach them from opposite directions:

| Dimension | SingleEye | Focusa |
|-----------|-----------|--------|
| **Origin** | Visionary / philosophical | Engineering / spec-driven |
| **Architecture** | Multi-LLM inner dialogue | Single-agent cognitive proxy |
| **Approach** | Emergent (agents discover behavior) | Deterministic (behavior is specified) |
| **Memory** | Brain-inspired (episodic, semantic, working) | Structured state (Focus State, ASCC, ECS) |
| **Autonomy** | Trust + hormonal modulation | Earned capability (ARI, Autonomy Levels) |
| **Identity** | Biometric pairing, companion lock | Agent Constitution, behavioral principles |
| **Embodiment** | Sensors, BCI, robotics, HUD | Harness-agnostic proxy (stdin/stdout, HTTP) |

**The overlap is the WIREBOT opportunity** — take the best of both and avoid building the same thing twice.

---

## Concept-by-Concept Overlap Map

### 1. Inner Dialogue / Multi-Agent Reasoning

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| Multiple LLM agents (Perception, Reasoning, Reflection, Coordinator) debate internally before responding | **Reliability Focus Mode (RFM)** — Microcells: isolated sub-agents invoked for verification, not creativity. Return structured evidence. | **High overlap.** Both create multiple reasoning passes before output. SingleEye is creative/divergent; Focusa is verification/convergent. Wirebot needs BOTH — creative exploration for business strategy, rigorous verification for financial advice. |

**Wirebot synthesis:** Inner dialogue for complex decisions (SingleEye style) + RFM microcell verification for high-stakes outputs (Focusa style).

---

### 2. Background Processing / Subconscious

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Subconscious Daemon** — associative linking, memory consolidation, dream simulation, idle self-talk, "intent shadows" | **Intuition Engine** — async-only background observer. Detects temporal signals (prolonged inactivity), repetition signals (repeated errors), consistency signals (contradictory decisions). Emits signals, never acts. | **Very high overlap.** Both are always-on background processes that surface weak signals. SingleEye's is more creative/generative; Focusa's is more analytical/signal-based. |

**Wirebot synthesis:** Intuition Engine for pattern detection (Focusa) + Subconscious Daemon for proactive insight generation (SingleEye). The Go daemon (`wirebot-memory-syncd`) is the natural home for both.

---

### 3. Confidence / Truth Calibration

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **TruthSeekerAgent** — resolves contradictions, traces claims, fact-checks via retrieval. **Confidence Agent** — scores certainty, calibrates tone. | **Focus State** — explicit structured representation with `confidence_level` field. **Validator results** are structured evidence. ASCC checkpoints provide traceable reasoning. | **High overlap.** Both systems explicitly track confidence and ground truth. SingleEye uses dedicated agents; Focusa uses structured state. |

**Wirebot synthesis:** Confidence score on every recommendation (already in SINGLEEYE_CONCEPTS.md Tier A #3). Focus State-style structured tracking of what's known vs. believed vs. guessed.

---

### 4. Autonomy / Trust Progression

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Trust Score Engine** — dynamic score, builds as recommendations prove correct. Low trust = ask permission. High trust = act first. | **Autonomy Calibration** — AL0→AL5 levels, earned through observed performance. **Autonomy Reliability Index (ARI)** — quantitative 0-100 score. Multi-dimensional: correctness, stability, efficiency. | **Near-identical concept.** Focusa's is more rigorous and formalized (evidence-based, multi-dimensional, revocable, scoped). |

**Wirebot synthesis:** Use Focusa's Autonomy Calibration framework directly. It's already spec'd in detail. Map SingleEye's trust-over-time concept onto ARI as the scoring mechanism.

---

### 5. Context Tracking / Thread Coherence

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Context Discovery System** — dynamic topic graphs, semantic threading, temporal anchoring. Memory Agent provides episodic recall. | **Thread Thesis** — living semantic anchor: user intent, goals, constraints, open questions, confidence level. **Context Lineage Tree (CLT)** — append-only tree of interaction history and compaction lineage. | **High overlap on purpose, different mechanism.** SingleEye discovers context emergently. Focusa externalizes it as an explicit, structured, revisable object. |

**Wirebot synthesis:** Thread Thesis is powerful — Wirebot should maintain a running "what is this operator trying to accomplish right now" object, updated deliberately. CLT gives audit trail. Memory Bridge provides the recall.

---

### 6. Agent Constitution / Identity

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Companion Mode** — biometric pairing, loyalty lock, personality core with hormonal modulation. IDENTITY.md + SOUL.md define who Wirebot IS. | **Agent Constitution** — declarative behavioral principles, self-evaluation heuristics, immutable rules. Governs what the agent may/may not do. | **Complementary, not overlapping.** SingleEye defines personality and relationship. Focusa defines governance and constraints. Both are needed. |

**Wirebot synthesis:** The 12 Pillars ARE the Agent Constitution. IDENTITY.md/SOUL.md define personality. Constitution defines boundaries. They coexist.

---

### 7. Hormonal / Mode States

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Simulo-Hormonal Engine** — dopamine (curiosity), cortisol (urgency), serotonin (stability), melatonin (rest cycles). Overridable, configurable. | **Focus Gate** — organic surfacing of priority candidates based on salience. **RFM** triggers based on risk/criticality. No explicit "mood" — but behavior changes based on measured task properties. | **Moderate overlap.** Both modulate system behavior based on context. SingleEye uses emotional metaphors; Focusa uses measured task properties. |

**Wirebot synthesis:** `operator_mode` state (crisis/focused/exploring/resting) from SINGLEEYE_CONCEPTS.md, implemented via Focus Gate-style salience scoring from Focusa. No need for full hormonal simulation — the mode state achieves the same result with less complexity.

---

### 8. Memory Architecture

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Multi-type**: Episodic, semantic, procedural, working memory. Brain-region mapping (hippocampus, cortex). Memory Daemon for consolidation. | **Structured state**: Focus State (current mind), ASCC (incremental summaries), ECS (artifact offloading), Reference Store (external knowledge handles). Semantic + procedural memory in Memory Plane. | **Same problem, different vocabulary.** Both need short-term, long-term, and working memory. Both need consolidation and decay. |

**Wirebot synthesis:** Already built — Mem0 = semantic/episodic, Letta = structured state (≈ Focus State), memory-core = reference/workspace, Go daemon = consolidation. The bridge IS the memory architecture.

---

### 9. Transparency / Observability

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **`/thoughts` command** — peek at inner dialogue. Transparent AI. | **Telemetry spec** (29-32) — comprehensive event logging, structured schemas, TUI visualization. **CLT** — full interaction lineage navigable and replayable. | **Same intent, different depth.** SingleEye exposes reasoning. Focusa exposes everything — events, metrics, lineage, state mutations. |

**Wirebot synthesis:** `/thoughts` for the operator (simple, useful). Focusa-style telemetry for debugging and system improvement. Both.

---

### 10. Proxy / Gateway Architecture

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| Not explicitly specified. Multi-LLM orchestration implied. | **ACP Proxy** — transparent JSON-RPC intermediary between client and agent. Passive observation mode or active cognitive proxy mode. Focus Gate + Prompt Assembly applied in the proxy layer. | **Focusa is more advanced here.** Its proxy architecture is exactly what OpenClaw gateway already is — a layer between user and LLM that adds cognition. |

**Wirebot synthesis:** OpenClaw gateway IS the proxy layer. Focusa's cognitive proxy concepts (Focus Gate, Prompt Assembly, CLT tracking) should be applied inside the gateway's plugin system.

---

### 11. Sensor Fusion / Embodiment

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| Full sensorium: EEG, computer vision, audio, GPS, IMU, environmental. BCI integration. Smart glasses. Robotics. | Not addressed. Focusa is a software cognitive layer — no hardware/sensor concept. | **No overlap.** SingleEye owns this domain entirely. |

**Wirebot synthesis:** Future capability. When Wirebot gets mobile app / wearable integration, SingleEye's sensor fusion architecture applies.

---

### 12. Permission / Consent Model

| SingleEye | Focusa | Overlap |
|-----------|--------|---------|
| **Linux-style permission hierarchy**: owner, mutual, group, public. Consent-based hormone sync between instances. | **Cache Permission Matrix** (doc 18), **Capability Permissions** (doc 25), **Agent Capability Scope** (doc 26). Scoped, revocable, explicitly granted. | **Moderate overlap.** Both define scoped, revocable permissions. SingleEye for data sharing between humans/instances. Focusa for what the agent is allowed to do. |

**Wirebot synthesis:** Both needed. SingleEye's consent model for Mentor/Collaborator/Mentee data sharing. Focusa's capability permissions for what Wirebot can do autonomously.

---

## Summary: What to Use Where

| Capability | Use SingleEye | Use Focusa | Use Both |
|-----------|--------------|------------|----------|
| Inner dialogue / reasoning | Creative divergence | Verification (RFM) | ✅ |
| Background processing | Proactive insights | Signal detection | ✅ |
| Confidence scoring | Tone calibration | Structured evidence | ✅ |
| Autonomy / trust | Trust-over-time concept | ARI + AL framework | ✅ Focusa framework, SingleEye concept |
| Context tracking | — | Thread Thesis + CLT | Focusa leads |
| Identity / personality | 12 Pillars, SOUL, IDENTITY | Agent Constitution | ✅ Complementary |
| Mode states | Operator mode concept | Focus Gate salience | ✅ Simplified |
| Memory | — | — | Already built (bridge) |
| Transparency | `/thoughts` command | Full telemetry | ✅ |
| Proxy / gateway | — | ACP cognitive proxy | Focusa leads (via OpenClaw) |
| Sensors / embodiment | Full architecture | — | SingleEye leads |
| Permissions | Data sharing consent | Capability scope | ✅ Different domains |

---

## Key Insight

**SingleEye and Focusa are not competing systems. They are two lenses on the same problem:**

- SingleEye asks: *"How should a mind FEEL and RELATE?"*
- Focusa asks: *"How should a mind GOVERN and VERIFY?"*

Wirebot needs both. The 12 Pillars already bridge them — Calm, Rigor, and Radical Truth are Focusa's governance. Resourceful, Proactive, and Communication are SingleEye's relationship layer.

The implementation path: **Focusa's specs are buildable today** (deterministic, well-defined). **SingleEye's vision guides what to build next** (creative, aspirational). Use Focusa for the skeleton. Use SingleEye for the soul.

---

## See Also

- [SINGLEEYE_CONCEPTS.md](./SINGLEEYE_CONCEPTS.md) — 15 concepts with priority matrix
- [FOCUSA_WIREBOT_INTEGRATION.md](./FOCUSA_WIREBOT_INTEGRATION.md) — Full Focusa mapping (152KB)
- [PAIRING.md](./PAIRING.md) — Pairing protocol (from SingleEye PUPP)
- [VISION.md](./VISION.md) — Sovereign mode philosophy
