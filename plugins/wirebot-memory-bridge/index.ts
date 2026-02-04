/**
 * Wirebot Memory Bridge
 *
 * OpenClaw plugin that coordinates Wirebot's memory + state systems.
 *
 * Architecture:
 *   - memory-core (OpenClaw built-in): workspace file recall (instant)
 *   - Mem0 (:8200): conversation fact storage + semantic search (~200ms)
 *   - Letta (:8283): structured self-editing business state (~100ms reads)
 *   - Scoreboard (:8100): execution scoring, events, memory queue, alerts
 *   - State Feeder (scoreboard goroutine): approved memories + events → Letta
 *
 * Data flow:
 *   Conversation → Bridge agent_end → Mem0 (facts) + Scoreboard queue
 *   Scoreboard queue → Human review → Approved → State Feeder → Letta
 *   (Single controlled path to Letta, rate-limited, watermarked)
 *
 * Tools (6):
 *   - wirebot_recall: cascading search (Mem0 + Letta blocks + archival)
 *   - wirebot_remember: store durable fact in Mem0
 *   - wirebot_business_state: read/update/message Letta memory blocks
 *   - wirebot_memory_queue: review/approve/reject pending memories
 *   - wirebot_checklist: multi-business setup checklist engine
 *   - wirebot_score: scoreboard query & event submission
 *
 * Hooks (2):
 *   - agent_start: parallel fetch (scoreboard + Letta blocks + alerts)
 *     → writes SCOREBOARD_STATE.md for workspace context injection
 *   - agent_end: extract facts → Mem0 + scoreboard queue (NOT direct to Letta)
 */

import { Type } from "@sinclair/typebox";
import type { OpenClawPluginApi } from "openclaw/plugin-sdk";

// ============================================================================
// Config
// ============================================================================

interface BridgeConfig {
  mem0Url: string;
  lettaUrl: string;
  lettaAgentId: string;
  mem0Namespace: string;
  autoExtract: boolean;
}

function getConfig(api: OpenClawPluginApi): BridgeConfig {
  const raw = api.pluginConfig as Record<string, unknown>;
  return {
    mem0Url: (raw.mem0Url as string) ?? "http://127.0.0.1:8200",
    lettaUrl: (raw.lettaUrl as string) ?? "http://127.0.0.1:8283",
    lettaAgentId: raw.lettaAgentId as string,
    mem0Namespace: (raw.mem0Namespace as string) ?? "wirebot_verious",
    autoExtract: (raw.autoExtract as boolean) ?? true,
  };
}

// ============================================================================
// HTTP helpers (no dependencies, just fetch)
// ============================================================================

async function mem0Search(
  baseUrl: string,
  namespace: string,
  query: string,
  limit = 5,
): Promise<Array<{ memory: string; score: number; id: string }>> {
  try {
    const resp = await fetch(`${baseUrl}/v1/search`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        query,
        namespace,
        limit,
      }),
      signal: AbortSignal.timeout(5000),
    });
    if (!resp.ok) return [];
    const data = (await resp.json()) as {
      results?: Array<{ memory: string; score: number; id: string }>;
    };
    return data.results ?? [];
  } catch {
    return [];
  }
}

async function mem0Store(
  baseUrl: string,
  namespace: string,
  text: string,
  messages?: Array<{ role: string; content: string }>,
): Promise<{ ok: boolean }> {
  try {
    const body: Record<string, unknown> = {
      namespace,
      category: "conversation",
    };
    // Prefer structured messages (Mem0's designed API) over raw text
    if (messages && messages.length > 0) {
      body.messages = messages;
    } else {
      body.text = text;
    }
    const resp = await fetch(`${baseUrl}/v1/store`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
      signal: AbortSignal.timeout(15000),
    });
    return { ok: resp.ok };
  } catch {
    return { ok: false };
  }
}

interface LettaBlock {
  id: string;
  label: string;
  value: string;
  description?: string;
  limit?: number;
}

async function lettaGetBlocks(
  baseUrl: string,
  agentId: string,
): Promise<LettaBlock[]> {
  try {
    const resp = await fetch(
      `${baseUrl}/v1/agents/${agentId}/core-memory/blocks`,
      {
        headers: { "Content-Type": "application/json" },
        signal: AbortSignal.timeout(5000),
      },
    );
    if (!resp.ok) return [];
    const data = (await resp.json()) as LettaBlock[];
    return Array.isArray(data) ? data : [];
  } catch {
    return [];
  }
}

async function lettaUpdateBlock(
  baseUrl: string,
  agentId: string,
  blockLabel: string,
  value: string,
): Promise<{ ok: boolean }> {
  try {
    const resp = await fetch(
      `${baseUrl}/v1/agents/${agentId}/core-memory/blocks/${blockLabel}`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value }),
        signal: AbortSignal.timeout(5000),
      },
    );
    return { ok: resp.ok };
  } catch {
    return { ok: false };
  }
}

async function lettaSendMessage(
  baseUrl: string,
  agentId: string,
  message: string,
): Promise<{ ok: boolean }> {
  try {
    const resp = await fetch(
      `${baseUrl}/v1/agents/${agentId}/messages`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          messages: [{ role: "user", content: message }],
        }),
        signal: AbortSignal.timeout(30000),
      },
    );
    return { ok: resp.ok };
  } catch {
    return { ok: false };
  }
}

async function lettaArchivalSearch(
  baseUrl: string,
  agentId: string,
  query: string,
  limit = 3,
): Promise<Array<{ text: string; id: string }>> {
  try {
    const resp = await fetch(
      `${baseUrl}/v1/agents/${agentId}/archival-memory/search?query=${encodeURIComponent(query)}&limit=${limit}`,
      { signal: AbortSignal.timeout(10000) },
    );
    if (!resp.ok) return [];
    const data = await resp.json();
    const results = data.results || data;
    if (!Array.isArray(results)) return [];
    return results.map((p: Record<string, unknown>) => ({
      text: String(p.content || p.text || ""),
      id: String(p.id || ""),
    }));
  } catch {
    return [];
  }
}

// ============================================================================
// Plugin
// ============================================================================

const wirebotMemoryBridge = {
  id: "wirebot-memory-bridge",
  name: "Wirebot Memory Bridge",
  description:
    "Coordinates memory across memory-core, Mem0, and Letta for unified recall and business state",
  kind: "extension" as const,

  register(api: OpenClawPluginApi) {
    const cfg = getConfig(api);

    api.logger.info(
      `wirebot-memory-bridge: registered (mem0: ${cfg.mem0Url}, letta: ${cfg.lettaUrl}, agent: ${cfg.lettaAgentId})`,
    );

    // ========================================================================
    // Tool: wirebot_recall
    // Cascading search across all three memory layers
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_recall",
        label: "Wirebot Recall",
        description:
          "Search Wirebot's complete memory across all layers. " +
          "Layer 1: workspace files (already in context via memory-core). " +
          "Layer 2: conversation facts from Mem0. " +
          "Layer 3: business state from Letta blocks. " +
          "Use when you need to remember something from past conversations, " +
          "business context, user preferences, or decisions.",
        parameters: Type.Object({
          query: Type.String({ description: "What to recall" }),
          layers: Type.Optional(
            Type.Array(
              Type.Union([
                Type.Literal("mem0"),
                Type.Literal("letta"),
                Type.Literal("all"),
              ]),
            ),
          ),
        }),
        async execute(_toolCallId, params) {
          const { query, layers } = params as {
            query: string;
            layers?: string[];
          };
          const searchAll =
            !layers || layers.length === 0 || layers.includes("all");
          const results: string[] = [];

          // Layer 1: memory-core — skip, already injected by memory-core plugin

          // Always fetch Letta blocks (2KB total, structured identity — always relevant)
          // and Mem0 vector search (semantic, handles NL queries) in parallel.
          // The Go cache substring search is only useful for exact keyword matches,
          // so we skip it and go direct to get better results.

          const promises: Promise<void>[] = [];

          // Mem0: vector/semantic search (handles natural language queries)
          if (searchAll || layers?.includes("mem0")) {
            promises.push(
              mem0Search(cfg.mem0Url, cfg.mem0Namespace, query, 5).then((facts) => {
                for (const f of facts) {
                  const pct = typeof f.score === "number"
                    ? ` (${(f.score * 100).toFixed(0)}%)`
                    : "";
                  results.push(`[fact] ${f.memory}${pct}`);
                }
              }),
            );
          }

          // Letta: ALWAYS include all blocks — they're the operator's structured
          // identity, goals, KPIs, and business stage. 2KB total, always relevant.
          // Also search archival memory for deep doc context (PAIRING, SCOREBOARD, etc.)
          if (searchAll || layers?.includes("letta")) {
            promises.push(
              lettaGetBlocks(cfg.lettaUrl, cfg.lettaAgentId).then((blocks) => {
                for (const block of blocks) {
                  if (block.value) {
                    results.push(`[state:${block.label}] ${block.value}`);
                  }
                }
              }),
            );
            // Archival search for complex "why" / doc-level queries
            promises.push(
              lettaArchivalSearch(cfg.lettaUrl, cfg.lettaAgentId, query, 2).then(
                (passages) => {
                  for (const p of passages) {
                    if (p.text) {
                      // Truncate long passages to ~500 chars for tool response
                      const snippet =
                        p.text.length > 500
                          ? p.text.slice(0, 500) + "..."
                          : p.text;
                      results.push(`[archival] ${snippet}`);
                    }
                  }
                },
              ),
            );
          }

          await Promise.all(promises);

          if (results.length === 0) {
            return {
              content: [
                {
                  type: "text" as const,
                  text: `No results found for "${query}" in Mem0 or Letta. Check workspace files (memory-core) which are already in your context.`,
                },
              ],
            };
          }

          return {
            content: [
              {
                type: "text" as const,
                text: `Found ${results.length} result(s) for "${query}":\n\n${results.join("\n\n")}`,
              },
            ],
          };
        },
      },
      { name: "wirebot_recall" },
    );

    // ========================================================================
    // Tool: wirebot_remember
    // Store a durable fact in Mem0
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_remember",
        label: "Wirebot Remember",
        description:
          "Store a fact, preference, or decision in long-term memory (Mem0). " +
          "Use for: user preferences, business decisions, relationships, context " +
          "that should survive across sessions. Mem0 handles dedup and contradiction " +
          "resolution automatically.",
        parameters: Type.Object({
          fact: Type.String({
            description:
              "The fact to remember. Be specific and concise. Example: " +
              "'Verious prefers ExtraWire tier for API access partners'",
          }),
        }),
        async execute(_toolCallId, params) {
          const { fact } = params as { fact: string };

          const result = await mem0Store(
            cfg.mem0Url,
            cfg.mem0Namespace,
            fact,
          );

          if (!result.ok) {
            return {
              content: [
                {
                  type: "text" as const,
                  text: `Failed to store fact in Mem0. Service may be down. Fact: "${fact}"`,
                },
              ],
            };
          }

          api.logger.info(
            `wirebot-memory-bridge: stored fact: "${fact.slice(0, 80)}..."`,
          );

          return {
            content: [
              {
                type: "text" as const,
                text: `Remembered: "${fact}"`,
              },
            ],
          };
        },
      },
      { name: "wirebot_remember" },
    );

    // ========================================================================
    // Tool: wirebot_business_state
    // Read or update Letta memory blocks
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_business_state",
        label: "Wirebot Business State",
        description:
          "Read or update structured business state stored in Letta memory blocks. " +
          "Blocks: business_stage (Idea/Launch/Growth), goals (active goals with " +
          "deadlines), kpis (key metrics), human (user context). " +
          "Use 'read' to see current state. " +
          "Use 'message' (PREFERRED) to let the Letta agent decide how to update — " +
          "send it context and it self-edits the right blocks intelligently. " +
          "Use 'update' ONLY for bulk resets or initialization — it overwrites the entire block.",
        parameters: Type.Object({
          action: Type.Union([
            Type.Literal("read"),
            Type.Literal("update"),
            Type.Literal("message"),
          ]),
          block: Type.Optional(
            Type.String({
              description:
                "Block label to read/update: business_stage, goals, kpis, human",
            }),
          ),
          value: Type.Optional(
            Type.String({
              description:
                "New value for block (action=update) or message to Letta agent (action=message)",
            }),
          ),
        }),
        async execute(_toolCallId, params) {
          const { action, block, value } = params as {
            action: "read" | "update" | "message";
            block?: string;
            value?: string;
          };

          // READ
          if (action === "read") {
            const blocks = await lettaGetBlocks(
              cfg.lettaUrl,
              cfg.lettaAgentId,
            );

            if (blocks.length === 0) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: "No business state blocks found. Letta may be down.",
                  },
                ],
              };
            }

            // Filter to specific block if requested
            const filtered = block
              ? blocks.filter((b) => b.label === block)
              : blocks;

            if (filtered.length === 0) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: `No block named "${block}". Available: ${blocks.map((b) => b.label).join(", ")}`,
                  },
                ],
              };
            }

            const text = filtered
              .map((b) => `## ${b.label}\n${b.value}`)
              .join("\n\n");

            return {
              content: [{ type: "text" as const, text }],
            };
          }

          // UPDATE (direct block write)
          if (action === "update") {
            if (!block || !value) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: "update requires both 'block' and 'value' parameters.",
                  },
                ],
              };
            }

            const result = await lettaUpdateBlock(
              cfg.lettaUrl,
              cfg.lettaAgentId,
              block,
              value,
            );

            if (!result.ok) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: `Failed to update block "${block}". Letta may be down or block doesn't exist.`,
                  },
                ],
              };
            }

            api.logger.info(
              `wirebot-memory-bridge: updated Letta block "${block}" (${value.length} chars)`,
            );

            return {
              content: [
                {
                  type: "text" as const,
                  text: `Updated business state block "${block}".`,
                },
              ],
            };
          }

          // MESSAGE (let Letta agent self-manage)
          if (action === "message") {
            if (!value) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: "message action requires 'value' parameter with the message to send.",
                  },
                ],
              };
            }

            const result = await lettaSendMessage(
              cfg.lettaUrl,
              cfg.lettaAgentId,
              value,
            );

            if (!result.ok) {
              return {
                content: [
                  {
                    type: "text" as const,
                    text: "Failed to send message to Letta agent. Service may be down.",
                  },
                ],
              };
            }

            return {
              content: [
                {
                  type: "text" as const,
                  text: `Sent to Letta agent: "${value.slice(0, 100)}..."`,
                },
              ],
            };
          }

          return {
            content: [
              {
                type: "text" as const,
                text: 'Invalid action. Use "read", "update", or "message".',
              },
            ],
          };
        },
      },
      { name: "wirebot_business_state" },
    );

    // ========================================================================
    // Tool: wirebot_memory_queue
    // Review and manage the memory approval queue from any surface
    // ========================================================================

    const scoreboardUrl = "http://127.0.0.1:8100";
    const scoreboardToken = "65b918ba-baf5-4996-8b53-6fb0f662a0c3";

    api.registerTool(
      {
        name: "wirebot_memory_queue",
        label: "Memory Queue",
        description:
          "View and manage the memory approval queue. Shows pending extracted memories " +
          "waiting for human review. Actions: status (counts by status), list (show pending), " +
          "approve (approve by ID), reject (reject by ID), correct (fix text and approve), " +
          "extract-vault (trigger Obsidian vault extraction). " +
          "Use this when asked about pending memories, memory review, or extraction progress.",
        parameters: Type.Object({
          action: Type.Union([
            Type.Literal("status"),
            Type.Literal("list"),
            Type.Literal("approve"),
            Type.Literal("reject"),
            Type.Literal("correct"),
            Type.Literal("extract-vault"),
          ], { description: "status=counts, list=show pending, approve/reject/correct=act on ID, extract-vault=start extraction" }),
          id: Type.Optional(Type.String({ description: "Memory queue item ID (for approve/reject/correct)" })),
          correction: Type.Optional(Type.String({ description: "Corrected memory text (for correct action)" })),
          limit: Type.Optional(Type.Number({ description: "Max items to list (default 10)" })),
        }),
        async execute(_toolCallId, params) {
          const p = params as {
            action: string;
            id?: string;
            correction?: string;
            limit?: number;
          };

          const headers = {
            "Authorization": `Bearer ${scoreboardToken}`,
            "Content-Type": "application/json",
          };

          try {
            switch (p.action) {
              case "status": {
                const resp = await fetch(`${scoreboardUrl}/v1/memory/queue?status=pending&limit=1`, { headers });
                const data = await resp.json() as { counts?: Record<string, number> };
                const counts = data.counts || {};
                let text = "📋 Memory Queue Status\n\n";
                for (const [status, count] of Object.entries(counts)) {
                  const icon = status === "pending" ? "⏳" : status === "approved" ? "✅" : status === "rejected" ? "❌" : "✏️";
                  text += `${icon} ${status}: ${count}\n`;
                }
                return { content: [{ type: "text" as const, text }] };
              }

              case "list": {
                const limit = p.limit || 10;
                const resp = await fetch(`${scoreboardUrl}/v1/memory/queue?status=pending&limit=${limit}`, { headers });
                const data = await resp.json() as { items?: Array<{ id: string; memory_text: string; source_type: string; source_file: string; confidence: number; created_at: string }> };
                const items = data.items || [];
                if (items.length === 0) {
                  return { content: [{ type: "text" as const, text: "✅ No pending memories to review." }] };
                }
                let text = `📋 Pending Memories (${items.length})\n\n`;
                for (const item of items) {
                  text += `**${item.id}** (${item.source_type}, conf: ${item.confidence})\n`;
                  text += `  "${item.memory_text}"\n`;
                  text += `  Source: ${item.source_file}\n\n`;
                }
                text += `\nTo approve: wirebot_memory_queue(action: "approve", id: "<id>")\n`;
                text += `To reject: wirebot_memory_queue(action: "reject", id: "<id>")`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "approve": {
                if (!p.id) return { content: [{ type: "text" as const, text: "❌ id required for approve" }] };
                const resp = await fetch(`${scoreboardUrl}/v1/memory/queue/${p.id}/approve`, {
                  method: "POST", headers,
                });
                const data = await resp.json() as { ok?: boolean; memory?: string };
                if (data.ok) {
                  return { content: [{ type: "text" as const, text: `✅ Approved: "${data.memory?.slice(0, 80)}..."` }] };
                }
                return { content: [{ type: "text" as const, text: `❌ Failed to approve ${p.id}` }] };
              }

              case "reject": {
                if (!p.id) return { content: [{ type: "text" as const, text: "❌ id required for reject" }] };
                const resp = await fetch(`${scoreboardUrl}/v1/memory/queue/${p.id}/reject`, {
                  method: "POST", headers,
                });
                const data = await resp.json() as { ok?: boolean };
                if (data.ok) {
                  return { content: [{ type: "text" as const, text: `❌ Rejected: ${p.id}` }] };
                }
                return { content: [{ type: "text" as const, text: `Failed to reject ${p.id}` }] };
              }

              case "correct": {
                if (!p.id || !p.correction) {
                  return { content: [{ type: "text" as const, text: "❌ id and correction required" }] };
                }
                const resp = await fetch(`${scoreboardUrl}/v1/memory/queue/${p.id}/correct`, {
                  method: "POST", headers,
                  body: JSON.stringify({ correction: p.correction }),
                });
                const data = await resp.json() as { ok?: boolean };
                if (data.ok) {
                  return { content: [{ type: "text" as const, text: `✏️ Corrected: "${p.correction.slice(0, 80)}..."` }] };
                }
                return { content: [{ type: "text" as const, text: `Failed to correct ${p.id}` }] };
              }

              case "extract-vault": {
                const resp = await fetch(`${scoreboardUrl}/v1/memory/extract-vault`, {
                  method: "POST", headers,
                  body: JSON.stringify({ path: "/data/wirebot/obsidian", limit: 853 }),
                });
                const data = await resp.json() as { message?: string; limit?: number };
                return { content: [{ type: "text" as const, text: `🔄 ${data.message || "Extraction started"} (limit: ${data.limit || 853})` }] };
              }

              default:
                return { content: [{ type: "text" as const, text: `❌ Unknown action: ${p.action}` }] };
            }
          } catch (err) {
            return { content: [{ type: "text" as const, text: `Error: ${String(err)}` }] };
          }
        },
      },
      { name: "wirebot_memory_queue" },
    );

    // ========================================================================
    // Tool: wirebot_checklist
    // Business Setup Checklist Engine — Idea → Launch → Growth
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_checklist",
        label: "Business Checklist",
        description:
          "Multi-business Checklist Engine. Track tasks across businesses and stages. " +
          "Actions: overview (all businesses), businesses (list with health), focus (switch business), " +
          "add-business, status (active business progress), next (get next task — global or per-business), " +
          "complete, skip, add, daily (multi-biz standup), list, detail, set-stage.",
        parameters: Type.Object({
          action: Type.Union([
            Type.Literal("status"),
            Type.Literal("overview"),
            Type.Literal("businesses"),
            Type.Literal("focus"),
            Type.Literal("add-business"),
            Type.Literal("next"),
            Type.Literal("complete"),
            Type.Literal("skip"),
            Type.Literal("add"),
            Type.Literal("daily"),
            Type.Literal("list"),
            Type.Literal("detail"),
            Type.Literal("set-stage"),
          ], { description: "Action to perform" }),
          stage: Type.Optional(
            Type.Union([
              Type.Literal("idea"),
              Type.Literal("launch"),
              Type.Literal("growth"),
              Type.Literal("mature"),
              Type.Literal("sunset"),
            ], { description: "Business stage filter" }),
          ),
          category: Type.Optional(Type.String({ description: "Category filter" })),
          taskId: Type.Optional(Type.String({ description: "Task ID for complete/skip/detail" })),
          businessName: Type.Optional(Type.String({ description: "Business name for focus/add-business/status" })),
          businessId: Type.Optional(Type.String({ description: "Business UUID" })),
          title: Type.Optional(Type.String({ description: "Task title (for add) or business name (for add-business)" })),
          description: Type.Optional(Type.String({ description: "Task or business description" })),
          shortName: Type.Optional(Type.String({ description: "Business short name (for add-business)" })),
          priority: Type.Optional(
            Type.Union([
              Type.Literal("critical"),
              Type.Literal("high"),
              Type.Literal("medium"),
              Type.Literal("low"),
            ], { description: "Task priority (for add)" }),
          ),
          businessPriority: Type.Optional(
            Type.Union([
              Type.Literal("primary"),
              Type.Literal("secondary"),
              Type.Literal("supporting"),
              Type.Literal("passive"),
            ], { description: "Business priority (for add-business)" }),
          ),
          domain: Type.Optional(Type.String({ description: "Business domain (for add-business)" })),
          status: Type.Optional(
            Type.Union([
              Type.Literal("pending"),
              Type.Literal("in_progress"),
              Type.Literal("completed"),
              Type.Literal("skipped"),
            ], { description: "Status filter (for list)" }),
          ),
        }),
        async execute(_toolCallId, params) {
          try {
            const { readFileSync, writeFileSync, existsSync } = await import("node:fs");
            const { randomUUID } = await import("node:crypto");

            const CHECKLIST_PATH = "/home/wirebot/clawd/checklist.json";

            // ── Types ──
            interface BizData {
              id: string; name: string; shortName: string; description: string;
              stage: string; role: string; revenueStatus: string; monthlyRevenue?: number;
              domain?: string; priority: string; relatedTo?: string[]; tags?: string[];
              createdAt: string; updatedAt: string;
            }
            interface TaskData {
              id: string; title: string; description?: string; businessId: string;
              stage: string; category: string; status: string; priority: string;
              source: string; aiSuggestion?: string; dueDate?: string;
              completedAt?: string; createdAt: string; updatedAt: string;
              dependencies?: string[]; notes?: string; order: number; crossCutting?: boolean;
            }
            interface ChecklistV2 {
              version: 2; operatorId: string; businesses: BizData[];
              activeBusiness: string; tasks: TaskData[];
              categories: Array<{ id: string; name: string; stage: string; order: number }>;
              createdAt: string; updatedAt: string;
            }

            let data: ChecklistV2;
            if (existsSync(CHECKLIST_PATH)) {
              const raw = JSON.parse(readFileSync(CHECKLIST_PATH, "utf-8"));
              // Auto-migrate v1 → v2
              if (!raw.version || raw.version === 1) {
                const defaultBiz: BizData = {
                  id: randomUUID(), name: raw.businessName || "My Business",
                  shortName: (raw.businessName || "BIZ").slice(0, 3).toUpperCase(),
                  description: "", stage: raw.currentStage || "idea", role: "founder",
                  revenueStatus: "pre-revenue", priority: "primary",
                  createdAt: raw.createdAt, updatedAt: raw.updatedAt,
                };
                data = {
                  version: 2, operatorId: raw.userId || "verious",
                  businesses: [defaultBiz], activeBusiness: defaultBiz.id,
                  tasks: raw.tasks.map((t: any) => ({ ...t, businessId: t.businessId || defaultBiz.id })),
                  categories: raw.categories, createdAt: raw.createdAt,
                  updatedAt: new Date().toISOString(),
                };
                writeFileSync(CHECKLIST_PATH, JSON.stringify(data, null, 2));
              } else {
                data = raw as ChecklistV2;
              }
            } else {
              return { content: [{ type: "text" as const, text: "Checklist not initialized. Run: wb add-business \"Name\"" }] };
            }

            const save = () => {
              data.updatedAt = new Date().toISOString();
              writeFileSync(CHECKLIST_PATH, JSON.stringify(data, null, 2));
            };

            const p = params as any;
            const progressBar = (pct: number) => "█".repeat(Math.round(pct / 5)) + "░".repeat(20 - Math.round(pct / 5));

            const findBiz = (nameOrId?: string): BizData | undefined => {
              if (!nameOrId) return data.businesses.find(b => b.id === data.activeBusiness);
              return data.businesses.find(b => b.id === nameOrId)
                || data.businesses.find(b => b.name.toLowerCase() === nameOrId.toLowerCase())
                || data.businesses.find(b => b.shortName.toLowerCase() === nameOrId.toLowerCase());
            };

            const bizTasks = (bizId: string, stage?: string) => {
              let t = data.tasks.filter(x => x.businessId === bizId);
              if (stage) t = t.filter(x => x.stage === stage);
              return t;
            };

            const getProgress = (bizId: string, s: string) => {
              const tasks = bizTasks(bizId, s);
              const completed = tasks.filter(t => t.status === "completed").length;
              return { stage: s, completed, total: tasks.length, percent: tasks.length > 0 ? Math.round(completed / tasks.length * 100) : 0 };
            };

            const bizHealth = (biz: BizData) => {
              const tasks = bizTasks(biz.id);
              const completed = tasks.filter(t => t.status === "completed").length;
              const checkPct = tasks.length > 0 ? Math.round(completed / tasks.length * 100) : 0;
              const lastAct = tasks.reduce((m, t) => Math.max(m, new Date(t.updatedAt).getTime()), new Date(biz.updatedAt).getTime());
              const daysSince = Math.floor((Date.now() - lastAct) / 86400000);
              const blocked = tasks.filter(t => t.priority === "critical" && (t.status === "pending" || t.status === "in_progress")).length;
              const rev: Record<string, number> = { active: 25, "pre-revenue": 10, declining: 5, paused: 0 };
              let h = checkPct * 0.2 + (rev[biz.revenueStatus] || 0) + Math.max(0, 15 - daysSince) + Math.max(0, 20 - blocked * 4) + (tasks.length > 0 ? 10 : 0) + 10;
              h = Math.min(100, Math.max(0, Math.round(h)));
              const sig = h < 30 ? "critical" : (h < 50 || daysSince > 14) ? "stale" : h < 70 ? "attention" : "healthy";
              return { health: h, signal: sig, checkPct, daysSince, blocked };
            };

            const sigIcon: Record<string, string> = { healthy: "🟢", attention: "🟡", stale: "🟠", critical: "🔴" };

            switch (p.action) {
              case "overview": {
                let text = `BUSINESSES  ${data.businesses.length} total\n\n`;
                for (const biz of data.businesses) {
                  const h = bizHealth(biz);
                  text += `${sigIcon[h.signal]} ${biz.shortName.padEnd(16)} [${biz.stage.padEnd(6)}] ${progressBar(h.health)} ${h.health}%`;
                  if (h.signal === "stale") text += `  ⚠️ ${h.daysSince}d stale`;
                  if (h.blocked > 0) text += `  ${h.blocked} blocked`;
                  text += "\n";
                }
                return { content: [{ type: "text" as const, text }] };
              }

              case "businesses": {
                if (data.businesses.length === 0) return { content: [{ type: "text" as const, text: "No businesses. Use add-business." }] };
                const active = data.activeBusiness;
                let text = "⚡ Your Businesses\n\n";
                for (const biz of data.businesses) {
                  const h = bizHealth(biz);
                  const isActive = biz.id === active ? " ◀ active" : "";
                  text += `${sigIcon[h.signal]} ${biz.name.padEnd(24)} [${biz.stage}]  Health: ${h.health}/100${isActive}\n`;
                  text += `   ${h.checkPct}% setup | ${biz.revenueStatus} | ${biz.priority}`;
                  if (h.daysSince > 7) text += ` | ${h.daysSince}d stale`;
                  if (h.blocked > 0) text += ` | ${h.blocked} blocked`;
                  text += "\n";
                }
                return { content: [{ type: "text" as const, text }] };
              }

              case "focus": {
                const name = p.businessName || p.businessId;
                if (!name) return { content: [{ type: "text" as const, text: "❌ businessName required" }] };
                const biz = findBiz(name);
                if (!biz) return { content: [{ type: "text" as const, text: `❌ Business not found: ${name}` }] };
                data.activeBusiness = biz.id;
                save();
                const h = bizHealth(biz);
                const prog = getProgress(biz.id, biz.stage);
                return { content: [{ type: "text" as const, text: `⚡ Active: ${biz.name} [${biz.stage}]\nHealth: ${h.health}/100 | ${prog.completed}/${prog.total} (${prog.percent}%)` }] };
              }

              case "add-business": {
                const name = p.businessName || p.title;
                if (!name) return { content: [{ type: "text" as const, text: "❌ businessName required" }] };
                const newBiz: BizData = {
                  id: randomUUID(), name, shortName: p.shortName || name.slice(0, 3).toUpperCase(),
                  description: p.description || "", stage: p.stage || "idea", role: "founder",
                  revenueStatus: "pre-revenue", priority: p.businessPriority || "secondary",
                  domain: p.domain, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
                };
                data.businesses.push(newBiz);
                // Note: seed tasks would need the template list — for now, no auto-seed from plugin inline
                save();
                return { content: [{ type: "text" as const, text: `➕ Added: ${newBiz.name} (${newBiz.shortName}) [${newBiz.stage}]\nID: ${newBiz.id}` }] };
              }

              case "status": {
                const biz = findBiz(p.businessName || p.businessId);
                if (!biz) return { content: [{ type: "text" as const, text: "❌ No active business" }] };
                const h = bizHealth(biz);
                const stages = p.stage ? [p.stage] : ["idea", "launch", "growth"];
                const allTasks = bizTasks(biz.id);
                const totalDone = allTasks.filter(t => t.status === "completed").length;
                let text = `📊 ${biz.name} — ${biz.stage.toUpperCase()}\n`;
                text += `${sigIcon[h.signal]} Health: ${h.health}/100 | Overall: ${totalDone}/${allTasks.length} (${allTasks.length > 0 ? Math.round(totalDone/allTasks.length*100) : 0}%)\n\n`;
                for (const s of stages) {
                  const prog = getProgress(biz.id, s);
                  text += `${s.toUpperCase()}: ${progressBar(prog.percent)} ${prog.percent}% (${prog.completed}/${prog.total})\n`;
                }
                const next = bizTasks(biz.id, biz.stage)
                  .filter(t => t.status === "pending" || t.status === "in_progress")
                  .sort((a, b) => { const pr: Record<string, number> = {critical:0,high:1,medium:2,low:3}; return (pr[a.priority]??2) - (pr[b.priority]??2) || a.order - b.order; })[0];
                if (next) text += `\n▶ Next: ${next.title} [${next.priority}]`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "next": {
                // Per-business or global
                if (p.businessName || p.businessId) {
                  const biz = findBiz(p.businessName || p.businessId);
                  if (!biz) return { content: [{ type: "text" as const, text: "❌ Business not found" }] };
                  const task = bizTasks(biz.id, biz.stage)
                    .filter(t => t.status === "pending" || t.status === "in_progress")
                    .sort((a, b) => { const pr: Record<string, number> = {critical:0,high:1,medium:2,low:3}; return (pr[a.priority]??2) - (pr[b.priority]??2) || a.order - b.order; })[0];
                  if (!task) return { content: [{ type: "text" as const, text: `✅ All tasks in ${biz.name} are complete!` }] };
                  let text = `▶ ${task.title} (${biz.shortName})\nPriority: ${task.priority} | ${task.stage}/${task.category}`;
                  if (task.aiSuggestion) text += `\n💡 ${task.aiSuggestion}`;
                  text += `\nID: ${task.id}`;
                  return { content: [{ type: "text" as const, text }] };
                }
                // Global: active business first
                const activeBiz = findBiz();
                if (activeBiz) {
                  const task = bizTasks(activeBiz.id, activeBiz.stage)
                    .filter(t => t.status === "pending" || t.status === "in_progress")
                    .sort((a, b) => { const pr: Record<string, number> = {critical:0,high:1,medium:2,low:3}; return (pr[a.priority]??2) - (pr[b.priority]??2) || a.order - b.order; })[0];
                  if (task) {
                    let text = `▶ ${task.title} (${activeBiz.shortName})\nPriority: ${task.priority} | ${task.stage}/${task.category}`;
                    if (task.aiSuggestion) text += `\n💡 ${task.aiSuggestion}`;
                    text += `\nID: ${task.id}`;
                    return { content: [{ type: "text" as const, text }] };
                  }
                }
                return { content: [{ type: "text" as const, text: "✅ All tasks complete!" }] };
              }

              case "complete": {
                if (!p.taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === p.taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                task.status = "completed"; task.completedAt = new Date().toISOString(); task.updatedAt = new Date().toISOString();
                save();
                const biz = findBiz(task.businessId);
                const prog = getProgress(task.businessId, task.stage);
                return { content: [{ type: "text" as const, text: `✅ ${task.title}${biz ? ` (${biz.shortName})` : ""}\nProgress: ${prog.completed}/${prog.total} (${prog.percent}%)` }] };
              }

              case "skip": {
                if (!p.taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === p.taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                task.status = "skipped"; task.updatedAt = new Date().toISOString();
                save();
                return { content: [{ type: "text" as const, text: `⏭ Skipped: ${task.title}` }] };
              }

              case "add": {
                if (!p.title) return { content: [{ type: "text" as const, text: "❌ title required" }] };
                const biz = findBiz(p.businessName || p.businessId);
                const bizId = biz?.id || data.activeBusiness;
                const stg = p.stage || (biz?.stage || "idea");
                const newTask: TaskData = {
                  id: randomUUID(), title: p.title, description: p.description, businessId: bizId,
                  stage: stg, category: p.category || `${stg}-custom`,
                  status: "pending", priority: p.priority || "medium", source: "user",
                  order: 999, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
                };
                data.tasks.push(newTask);
                save();
                return { content: [{ type: "text" as const, text: `➕ ${newTask.title}${biz ? ` (${biz.shortName})` : ""}\nID: ${newTask.id}` }] };
              }

              case "daily": {
                let text = `📋 Daily Stand-Up\n`;
                for (const biz of data.businesses) {
                  const tasks = bizTasks(biz.id, biz.stage)
                    .filter(t => (t.status === "pending" || t.status === "in_progress") && (t.priority === "critical" || t.priority === "high"))
                    .sort((a, b) => a.order - b.order)
                    .slice(0, 2);
                  if (tasks.length > 0) {
                    text += `\n${biz.shortName}:\n`;
                    for (const t of tasks) text += `  ☐ ${t.title} [${t.priority}]\n`;
                  }
                }
                return { content: [{ type: "text" as const, text }] };
              }

              case "list": {
                const biz = findBiz(p.businessName || p.businessId);
                let tasks = biz ? bizTasks(biz.id) : data.tasks;
                if (p.stage) tasks = tasks.filter(t => t.stage === p.stage);
                if (p.category) tasks = tasks.filter(t => t.category === p.category);
                if (p.status) tasks = tasks.filter(t => t.status === p.status);
                const icons: Record<string, string> = { pending: "☐", in_progress: "⏳", completed: "✅", skipped: "⏭" };
                let text = `Tasks (${tasks.length}):\n\n`;
                for (const t of tasks.slice(0, 25)) {
                  const b = findBiz(t.businessId);
                  text += `${icons[t.status] || "?"} ${b ? `[${b.shortName}] ` : ""}${t.title} [${t.priority}] ${t.id.slice(0,8)}\n`;
                }
                if (tasks.length > 25) text += `\n... +${tasks.length - 25} more`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "detail": {
                if (!p.taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === p.taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                const biz = findBiz(task.businessId);
                let text = `📌 ${task.title}\n`;
                if (biz) text += `Business: ${biz.name} (${biz.shortName})\n`;
                text += `Status: ${task.status} | Priority: ${task.priority}\n`;
                text += `Stage: ${task.stage} | Category: ${task.category}\n`;
                if (task.aiSuggestion) text += `💡 ${task.aiSuggestion}\n`;
                if (task.notes) text += `📝 ${task.notes}\n`;
                text += `ID: ${task.id}`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "set-stage": {
                if (!p.stage) return { content: [{ type: "text" as const, text: "❌ stage required" }] };
                const biz = findBiz(p.businessName || p.businessId);
                if (!biz) return { content: [{ type: "text" as const, text: "❌ Business not found" }] };
                biz.stage = p.stage; biz.updatedAt = new Date().toISOString();
                save();
                const prog = getProgress(biz.id, p.stage);
                return { content: [{ type: "text" as const, text: `🔄 ${biz.name} → ${p.stage.toUpperCase()} — ${prog.completed}/${prog.total} (${prog.percent}%)` }] };
              }
            }

            return { content: [{ type: "text" as const, text: `❌ Unknown action: ${p.action}` }] };
          } catch (err) {
            return { content: [{ type: "text" as const, text: `Error: ${String(err)}` }] };
          }
        },
      },
      { name: "wirebot_checklist" },
    );

    // ========================================================================
    // Tool: wirebot_score — scoreboard query & event submission
    // (scoreboardUrl + scoreboardToken already declared above in wirebot_memory_queue)
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_score",
        description:
          "Query the Business Performance Scoreboard — get today's score, season record, " +
          "activity feed, streak, or submit a new event. The scoreboard tracks execution " +
          "across 4 lanes: shipping, distribution, revenue, systems.",
        parameters: Type.Object({
          action: Type.Union([
            Type.Literal("dashboard"),
            Type.Literal("feed"),
            Type.Literal("season"),
            Type.Literal("intent"),
            Type.Literal("submit"),
            Type.Literal("financial"),
            Type.Literal("integrations"),
            Type.Literal("projects"),
            Type.Literal("stalls"),
          ], { description: "dashboard=score+lanes+feed, feed=recent events, season=record, intent=set/get, submit=new event, financial=revenue snapshot, integrations=connected accounts, projects=approved repos, stalls=detect inactivity" }),
          // For "submit" action
          event_type: Type.Optional(Type.String({ description: "e.g. TASK_COMPLETED, CODE_SHIPPED, REVENUE_EVENT" })),
          lane: Type.Optional(Type.String({ description: "shipping|distribution|revenue|systems" })),
          artifact_title: Type.Optional(Type.String({ description: "What was shipped/done" })),
          artifact_url: Type.Optional(Type.String({ description: "URL to artifact" })),
          source: Type.Optional(Type.String({ description: "Event source (default: wirebot)" })),
          business_id: Type.Optional(Type.String({ description: "Business ID (STA, WIR, PHI, SEW)" })),
          // For "intent" action
          intent_text: Type.Optional(Type.String({ description: "Today's intent statement" })),
        }),
        async execute(_toolCallId, params) {
          const p = params as {
            action: string;
            event_type?: string;
            lane?: string;
            artifact_title?: string;
            artifact_url?: string;
            source?: string;
            business_id?: string;
            intent_text?: string;
          };

          const headers = {
            "Authorization": `Bearer ${scoreboardToken}`,
            "Content-Type": "application/json",
          };

          try {
            switch (p.action) {
              case "dashboard": {
                const resp = await fetch(`${scoreboardUrl}/v1/scoreboard?mode=dashboard`, { headers });
                const data = await resp.json();
                return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
              }
              case "feed": {
                const resp = await fetch(`${scoreboardUrl}/v1/feed?limit=10`, { headers });
                const data = await resp.json();
                return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
              }
              case "season": {
                const resp = await fetch(`${scoreboardUrl}/v1/season`, { headers });
                const data = await resp.json();
                return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
              }
              case "intent": {
                if (p.intent_text) {
                  const resp = await fetch(`${scoreboardUrl}/v1/intent`, {
                    method: "POST",
                    headers,
                    body: JSON.stringify({ intent: p.intent_text }),
                  });
                  const data = await resp.json();
                  return { content: [{ type: "text" as const, text: `Intent set: ${p.intent_text}\n${JSON.stringify(data)}` }] };
                }
                const resp = await fetch(`${scoreboardUrl}/v1/scoreboard`, { headers });
                const data = await resp.json() as { intent?: string };
                return { content: [{ type: "text" as const, text: `Current intent: ${data.intent || "(none set)"}` }] };
              }
              case "financial": {
                const resp = await fetch(`${scoreboardUrl}/v1/financial/snapshot`, { headers });
                const data = await resp.json();
                return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
              }
              case "integrations": {
                const resp = await fetch(`${scoreboardUrl}/v1/integrations`, { headers });
                const data = await resp.json();
                // Summarize for AI context
                const ints = (data as { integrations?: Array<{ provider: string; display_name: string; status: string; last_used_at: string }> }).integrations || [];
                const summary = ints.map((i: { provider: string; display_name: string; status: string; last_used_at: string }) =>
                  `${i.provider}: ${i.display_name} (${i.status}, last: ${i.last_used_at || "never"})`
                ).join("\n");
                return { content: [{ type: "text" as const, text: `Connected integrations (${ints.length}):\n${summary}` }] };
              }
              case "projects": {
                const resp = await fetch(`${scoreboardUrl}/v1/projects`, { headers });
                const data = await resp.json();
                return { content: [{ type: "text" as const, text: JSON.stringify(data, null, 2) }] };
              }
              case "stalls": {
                // Check for stall signals: hours since last ship, lane gaps, streak risk
                const resp = await fetch(`${scoreboardUrl}/v1/scoreboard?mode=dashboard`, { headers });
                const data = await resp.json() as {
                  scoreboard?: { score: number; stall_hours: number; streak: { current: number }; lanes: { shipping: number; distribution: number; revenue: number; systems: number } };
                };
                const sb = data.scoreboard || data as any;
                const stalls: string[] = [];
                if (sb.stall_hours > 12) stalls.push(`⚠️ ${sb.stall_hours}h since last ship`);
                if (sb.lanes?.shipping === 0) stalls.push("🔴 Shipping lane empty today");
                if (sb.lanes?.revenue === 0) stalls.push("🟡 Revenue lane empty today");
                if (sb.lanes?.distribution === 0) stalls.push("🟡 Distribution lane empty today");
                if (sb.lanes?.systems === 0) stalls.push("🟡 Systems lane empty today");
                if (sb.streak?.current === 0) stalls.push("🔴 Streak broken!");
                if (stalls.length === 0) stalls.push("✅ No stalls detected — all lanes active");
                return { content: [{ type: "text" as const, text: `Stall check:\n${stalls.join("\n")}\nScore: ${sb.score || 0}/100, Streak: ${sb.streak?.current || 0}d` }] };
              }
              case "submit": {
                if (!p.event_type || !p.lane) {
                  return { content: [{ type: "text" as const, text: "❌ submit requires event_type and lane" }] };
                }
                const resp = await fetch(`${scoreboardUrl}/v1/events`, {
                  method: "POST",
                  headers,
                  body: JSON.stringify({
                    event_type: p.event_type,
                    lane: p.lane,
                    source: p.source || "wirebot",
                    artifact_title: p.artifact_title || "",
                    artifact_url: p.artifact_url || "",
                    business_id: p.business_id || "",
                    status: "pending", // Wirebot-submitted events are gated
                  }),
                });
                const data = await resp.json() as { ok?: boolean; event_id?: string; status?: string; score_delta?: number };
                if (data.ok) {
                  return { content: [{ type: "text" as const, text: `✅ Event submitted (${data.status}): ${p.artifact_title || p.event_type} → ${p.lane} | delta: ${data.score_delta} | id: ${data.event_id}` }] };
                }
                return { content: [{ type: "text" as const, text: `❌ ${JSON.stringify(data)}` }] };
              }
              default:
                return { content: [{ type: "text" as const, text: `❌ Unknown action: ${p.action}` }] };
            }
          } catch (err) {
            return { content: [{ type: "text" as const, text: `Error querying scoreboard: ${String(err)}` }] };
          }
        },
      },
      { name: "wirebot_score" },
    );

    // ========================================================================
    // Hook: agent_start — write live scoreboard state to workspace file
    // The workspace bootstrap system will inject this into the system prompt.
    // ========================================================================

    api.on("agent_start", async () => {
      try {
        const headers = {
          "Authorization": `Bearer ${scoreboardToken}`,
          "Content-Type": "application/json",
        };

        // Parallel fetch: scoreboard + financial + integrations + Letta blocks + alerts
        // (was sequential — saved ~400ms by parallelizing)
        const [sbResp, fResp, iResp, lettaBlocks, alertsResp] = await Promise.all([
          fetch(`${scoreboardUrl}/v1/scoreboard?mode=dashboard`, { headers }).catch(() => null),
          fetch(`${scoreboardUrl}/v1/financial/snapshot`, { headers }).catch(() => null),
          fetch(`${scoreboardUrl}/v1/integrations`, { headers }).catch(() => null),
          lettaGetBlocks(cfg.lettaUrl, cfg.lettaAgentId),
          fetch(`${scoreboardUrl}/v1/alerts`, { headers }).catch(() => null),
        ]);

        if (!sbResp || !sbResp.ok) return;
        const data = await sbResp.json() as Record<string, unknown>;

        const sb = (data.scoreboard || data) as Record<string, unknown>;
        const lanes = (sb.lanes || {}) as Record<string, number>;
        const streak = (sb.streak || {}) as Record<string, number>;
        const feed = ((data.feed || []) as Array<{ artifact_title: string; timestamp: string }>);

        let financial = "";
        if (fResp?.ok) {
          const fData = await fResp.json() as Record<string, unknown>;
          financial = `Revenue 30d: $${fData.revenue_30d || 0} | MRR est: $${fData.mrr_estimate || 0}`;
        }

        let integrations = "";
        if (iResp?.ok) {
          const iData = await iResp.json() as { integrations?: Array<{ provider: string; display_name: string; status: string }> };
          const active = (iData.integrations || []).filter(i => i.status === "active");
          if (active.length > 0) {
            integrations = `Connected: ${active.map(i => i.display_name).join(", ")} (${active.length} total)`;
          }
        }

        // Letta blocks → structured business context
        let lettaSection = "";
        if (lettaBlocks.length > 0) {
          const blockLines: string[] = [];
          for (const b of lettaBlocks) {
            if (b.label === "goals" && b.value) {
              // Extract just active goals (first 10 lines)
              const lines = b.value.split("\n").filter((l: string) => l.trim()).slice(0, 10);
              blockLines.push(`### Goals\n${lines.join("\n")}`);
            } else if (b.label === "kpis" && b.value) {
              blockLines.push(`### KPIs\n${b.value}`);
            } else if (b.label === "business_stage" && b.value) {
              // First 3 lines of business stage (score + level + profile summary)
              const lines = b.value.split("\n").filter((l: string) => l.trim()).slice(0, 3);
              blockLines.push(`### Business Stage\n${lines.join("\n")}`);
            }
          }
          if (blockLines.length > 0) {
            lettaSection = blockLines.join("\n\n");
          }
        }

        // Active alerts
        let alertsSection = "";
        if (alertsResp?.ok) {
          const alerts = await alertsResp.json() as Array<{ title: string; detail: string; severity: string }>;
          if (alerts.length > 0) {
            alertsSection = alerts.map(a => {
              const icon = a.severity === "critical" ? "🔴" : a.severity === "warning" ? "🟡" : "ℹ️";
              return `${icon} **${a.title}** — ${a.detail}`;
            }).join("\n");
          }
        }

        const briefing = [
          `# Wirebot Context (live)`,
          ``,
          `Updated: ${new Date().toISOString()}`,
          ``,
          `## Execution Score`,
          `- Score: ${sb.score || 0}/100 | Signal: ${sb.signal || "unknown"}`,
          `- Season: ${sb.season_day || "?"}`,
          `- Streak: ${streak.current || 0} days (best: ${streak.best || 0})`,
          sb.stall_hours ? `- ⚠️ STALL: ${sb.stall_hours}h since last ship` : `- No stall detected`,
          sb.intent ? `- Intent: ${sb.intent}` : `- Intent: (not set — nudge operator)`,
          ``,
          `## Lane Breakdown`,
          `- Shipping:     ${lanes.shipping || 0}/40`,
          `- Revenue:      ${lanes.revenue || 0}/20`,
          `- Distribution: ${lanes.distribution || 0}/25`,
          `- Systems:      ${lanes.systems || 0}/15`,
          ``,
          financial ? `## Financial\n${financial}\n` : "",
          integrations ? `## Integrations\n${integrations}\n` : "",
          alertsSection ? `## ⚠️ Active Alerts\n${alertsSection}\n` : "",
          lettaSection ? `## Business State (Letta)\n${lettaSection}\n` : "",
          feed.length > 0 ? `## Recent Activity\n${feed.slice(0, 5).map(e => `- ${e.artifact_title}`).join("\n")}` : "",
          ``,
          `## Behavioral Rules`,
          `- If score < 30 and signal is RED: focus on shipping (highest-weight lane)`,
          `- If stall > 8h: proactively ask what's blocking, suggest smallest shippable unit`,
          `- If intent not set: remind operator to set daily intent`,
          `- If streak > 7: celebrate consistency, protect the streak`,
          `- If revenue lane is 0: check if Stripe/Plaid/WooCommerce events are flowing`,
          `- Reference specific lane scores, alerts, and KPIs in your responses`,
          `- When operator asks "how am I doing?" → read this data, don't guess`,
        ].filter(Boolean).join("\n");

        const fs = await import("fs");
        const path = "/home/wirebot/clawd/SCOREBOARD_STATE.md";
        fs.writeFileSync(path, briefing, "utf-8");
        api.logger.info(`Wirebot context written (score=${sb.score}, streak=${streak.current}, blocks=${lettaBlocks.length}, alerts=${alertsSection ? "yes" : "none"})`);
      } catch (err) {
        api.logger.warn(`Wirebot context write failed: ${String(err)}`);
      }
    });

    // ========================================================================
    // Hook: agent_end — async fact extraction to Mem0
    // ========================================================================

    if (cfg.autoExtract) {
      let extracting = false; // re-entrancy guard
      api.on("agent_end", async (event) => {
        if (!event.success || !event.messages || event.messages.length === 0) {
          return;
        }

        // Guard: skip if we're already inside an extraction (prevents infinite loop).
        // The extraction sends an LLM call which creates a new session, which fires
        // agent_end again. Without this guard, sessions multiply at ~500/day.
        if (extracting) return;

        // Guard: skip if the conversation IS an extraction (detect by content).
        // Catches the case where the flag was reset between restarts.
        const firstMsg = event.messages.find(
          (m: Record<string, unknown>) => m && (m as Record<string, unknown>).role === "user",
        );
        if (firstMsg) {
          const text = typeof (firstMsg as Record<string, unknown>).content === "string"
            ? ((firstMsg as Record<string, unknown>).content as string)
            : JSON.stringify((firstMsg as Record<string, unknown>).content);
          if (
            text.includes("Extract PERSONAL FACTS") ||
            text.includes("You are extracting PERSONAL FACTS") ||
            text.startsWith("Input:\nuser: Input:")
          ) {
            return;
          }
        }

        extracting = true;
        try {
          // Extract user and assistant messages from the conversation
          const conversation: Array<{ role: string; content: string }> = [];

          for (const msg of event.messages) {
            if (!msg || typeof msg !== "object") continue;
            const m = msg as Record<string, unknown>;
            const role = m.role as string;
            if (role !== "user" && role !== "assistant") continue;

            let content = "";
            if (typeof m.content === "string") {
              content = m.content;
            } else if (Array.isArray(m.content)) {
              for (const block of m.content) {
                if (
                  block &&
                  typeof block === "object" &&
                  (block as Record<string, unknown>).type === "text" &&
                  typeof (block as Record<string, unknown>).text === "string"
                ) {
                  content += (block as Record<string, unknown>).text as string;
                }
              }
            }

            if (content.length > 0) {
              // Skip memory-injected context
              if (content.includes("<relevant-memories>")) continue;
              // Skip very short messages
              if (content.length < 10) continue;

              conversation.push({ role, content });
            }
          }

          if (conversation.length < 2) return; // Need at least 1 exchange

          // ── Flow 1a: Send to Mem0 for its native conversation-aware extraction ──
          // Mem0 handles dedup and contradiction resolution internally.
          const result = await mem0Store(
            cfg.mem0Url,
            cfg.mem0Namespace,
            "", // text fallback unused when messages provided
            conversation,
          );

          if (result.ok) {
            api.logger.info(
              `wirebot-memory-bridge: sent ${conversation.length} messages to Mem0 for fact extraction`,
            );
          } else {
            api.logger.warn(
              "wirebot-memory-bridge: Mem0 fact extraction failed",
            );
          }

          // Compute last user/assistant once for Flows 1b and 2
          const lastUser = conversation.filter((m) => m.role === "user").pop();
          const lastAssistant = conversation.filter((m) => m.role === "assistant").pop();

          // ── Flow 1b: Also route through scoreboard's extraction queue ──
          // This feeds the approval pipeline → MEMORY.md (the vector-indexed store).
          // Without this, Mem0 gets facts but MEMORY.md doesn't, causing divergence.
          // Uses POST /v1/memory/extract-conversation endpoint on the scoreboard.
          if (lastUser && lastAssistant && lastUser.content.length >= 20) {
            try {
              const extractResp = await fetch(`${scoreboardUrl}/v1/memory/extract-conversation`, {
                method: "POST",
                headers: {
                  "Authorization": `Bearer ${scoreboardToken}`,
                  "Content-Type": "application/json",
                },
                body: JSON.stringify({
                  user_message: lastUser.content,
                  assistant_message: lastAssistant.content,
                }),
                signal: AbortSignal.timeout(5000),
              });
              if (extractResp.ok) {
                api.logger.info("wirebot-memory-bridge: routed conversation to scoreboard extraction queue");
              }
            } catch {
              // Scoreboard routing is best-effort
            }
          }

          // ── Flow 2: Letta updates handled by the State Feeder ──
          // Previously this hook sent business-relevant conversations directly
          // to Letta. That created a second uncontrolled path to Letta alongside
          // the feeder, with no rate limiting or watermarks.
          //
          // Now: conversations go to the scoreboard extraction queue (Flow 1b above).
          // The operator reviews and approves. The state feeder picks up approved
          // memories and delivers them to Letta — single path, rate-limited,
          // watermarked, no loops.
          //
          // See: cmd/scoreboard/letta_feeder.go
        } catch (err) {
          api.logger.warn(
            `wirebot-memory-bridge: agent_end hook error: ${String(err)}`,
          );
        } finally {
          extracting = false;
        }
      });
    }

    // ========================================================================
    // Service registration
    // ========================================================================

    api.registerService({
      id: "wirebot-memory-bridge",
      async start() {
        // Health check Mem0
        try {
          const resp = await fetch(`${cfg.mem0Url}/health`, {
            signal: AbortSignal.timeout(5000),
          });
          if (resp.ok) {
            const data = (await resp.json()) as { memories?: number };
            api.logger.info(
              `wirebot-memory-bridge: Mem0 ✓ reachable (${data.memories ?? "?"} memories)`,
            );
          } else {
            api.logger.warn(
              `wirebot-memory-bridge: Mem0 ✗ error ${resp.status}`,
            );
          }
        } catch {
          api.logger.warn(
            "wirebot-memory-bridge: Mem0 unreachable at startup (will retry on use)",
          );
        }

        // Health check Letta
        try {
          const blocks = await lettaGetBlocks(cfg.lettaUrl, cfg.lettaAgentId);
          api.logger.info(
            `wirebot-memory-bridge: Letta ✓ ${blocks.length} blocks (${blocks.map((b) => b.label).join(", ")})`,
          );
        } catch {
          api.logger.warn(
            "wirebot-memory-bridge: Letta unreachable at startup (will retry on use)",
          );
        }
      },
      stop() {
        api.logger.info("wirebot-memory-bridge: stopped");
      },
    });
  },
};

export default wirebotMemoryBridge;
