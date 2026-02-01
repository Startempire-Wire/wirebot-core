/**
 * Wirebot Memory Bridge
 *
 * Clawdbot extension that coordinates three memory systems:
 *   - memory-core (embedded): workspace file recall (instant)
 *   - Mem0 (:8200): LLM-extracted conversation facts (~200ms)
 *   - Letta (:8283): structured self-editing business state (~100ms reads)
 *
 * Pattern: Write-Through, Read-Cascade
 *   Writes go to the appropriate system based on data type.
 *   Reads cascade: memory-core → Mem0 → Letta blocks.
 *
 * Provides 3 tools + 1 lifecycle hook:
 *   - wirebot_recall: cascading search across all layers
 *   - wirebot_remember: store durable fact in Mem0
 *   - wirebot_business_state: read/update Letta memory blocks
 *   - agent_end hook: async fact extraction to Mem0
 */

import { Type } from "@sinclair/typebox";
import type { ClawdbotPluginApi } from "clawdbot/plugin-sdk";

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

function getConfig(api: ClawdbotPluginApi): BridgeConfig {
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
): Promise<{ ok: boolean }> {
  try {
    const resp = await fetch(`${baseUrl}/v1/store`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        text,
        namespace,
        category: "conversation",
      }),
      signal: AbortSignal.timeout(15000),
    });
    return { ok: resp.ok };
  } catch {
    return { ok: false };
  }
}

async function mem0List(
  baseUrl: string,
  namespace: string,
): Promise<Array<{ memory: string; id: string }>> {
  try {
    const resp = await fetch(`${baseUrl}/v1/list`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ namespace }),
      signal: AbortSignal.timeout(5000),
    });
    if (!resp.ok) return [];
    const data = (await resp.json()) as {
      results?: Array<{ memory: string; id: string }>;
    };
    return data.results ?? [];
  } catch {
    return [];
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

// ============================================================================
// Plugin
// ============================================================================

const wirebotMemoryBridge = {
  id: "wirebot-memory-bridge",
  name: "Wirebot Memory Bridge",
  description:
    "Coordinates memory across memory-core, Mem0, and Letta for unified recall and business state",
  kind: "extension" as const,

  register(api: ClawdbotPluginApi) {
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

          // Layer 2: Mem0 facts
          if (searchAll || layers?.includes("mem0")) {
            const facts = await mem0Search(
              cfg.mem0Url,
              cfg.mem0Namespace,
              query,
              5,
            );
            for (const f of facts) {
              const pct = typeof f.score === "number"
                ? ` (${(f.score * 100).toFixed(0)}%)`
                : "";
              results.push(`[fact] ${f.memory}${pct}`);
            }
          }

          // Layer 3: Letta blocks
          if (searchAll || layers?.includes("letta")) {
            const blocks = await lettaGetBlocks(cfg.lettaUrl, cfg.lettaAgentId);
            const queryLower = query.toLowerCase();
            for (const block of blocks) {
              if (
                block.value &&
                block.value.toLowerCase().includes(queryLower)
              ) {
                results.push(`[state:${block.label}] ${block.value}`);
              }
            }
          }

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
          "Use 'read' to see current state. Use 'update' to change a specific block. " +
          "Use 'message' to let the Letta agent decide how to update (slower, uses LLM).",
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
    // Hook: agent_end — async fact extraction to Mem0
    // ========================================================================

    if (cfg.autoExtract) {
      api.on("agent_end", async (event) => {
        if (!event.success || !event.messages || event.messages.length === 0) {
          return;
        }

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

          // Concatenate conversation into a single text for Mem0's /v1/store
          // Mem0's LLM handles dedup, contradiction resolution, extraction
          const text = conversation
            .map((m) => `${m.role}: ${m.content}`)
            .join("\n\n");

          const result = await mem0Store(
            cfg.mem0Url,
            cfg.mem0Namespace,
            text,
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
        } catch (err) {
          api.logger.warn(
            `wirebot-memory-bridge: agent_end hook error: ${String(err)}`,
          );
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
