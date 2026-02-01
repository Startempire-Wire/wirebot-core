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
    // Tool: wirebot_checklist
    // Business Setup Checklist Engine — Idea → Launch → Growth
    // ========================================================================

    api.registerTool(
      {
        name: "wirebot_checklist",
        label: "Business Checklist",
        description:
          "Business Setup Checklist Engine. Track tasks across Idea → Launch → Growth stages. " +
          "Actions: status (show progress), next (get next task), complete (mark done), " +
          "skip, add (new task), daily (stand-up), list (filter tasks), detail, set-stage.",
        parameters: Type.Object({
          action: Type.Union([
            Type.Literal("status"),
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
            ], { description: "Business stage filter" }),
          ),
          category: Type.Optional(Type.String({ description: "Category filter" })),
          taskId: Type.Optional(Type.String({ description: "Task ID for complete/skip/detail" })),
          title: Type.Optional(Type.String({ description: "Task title (for add)" })),
          description: Type.Optional(Type.String({ description: "Task description (for add)" })),
          priority: Type.Optional(
            Type.Union([
              Type.Literal("critical"),
              Type.Literal("high"),
              Type.Literal("medium"),
              Type.Literal("low"),
            ], { description: "Task priority (for add)" }),
          ),
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
            // Dynamic import — checklist engine lives outside the plugin dir
            const { readFileSync, writeFileSync, existsSync } = await import("node:fs");
            const { randomUUID } = await import("node:crypto");

            const CHECKLIST_PATH = "/home/wirebot/clawd/checklist.json";

            // Inline minimal engine for the plugin context
            // (avoiding cross-module import complexity in openclaw plugin loader)
            interface TaskData {
              id: string; title: string; description?: string; stage: string;
              category: string; status: string; priority: string; source: string;
              aiSuggestion?: string; dueDate?: string; completedAt?: string;
              createdAt: string; updatedAt: string; dependencies?: string[];
              notes?: string; order: number;
            }
            interface ChecklistData {
              version: number; userId: string; businessName?: string;
              currentStage: string; tasks: TaskData[];
              categories: Array<{ id: string; name: string; stage: string; order: number }>;
              createdAt: string; updatedAt: string;
            }

            let data: ChecklistData;
            if (existsSync(CHECKLIST_PATH)) {
              data = JSON.parse(readFileSync(CHECKLIST_PATH, "utf-8"));
            } else {
              return { content: [{ type: "text" as const, text: "Checklist not initialized. Run the onboarding flow first." }] };
            }

            const save = () => {
              data.updatedAt = new Date().toISOString();
              writeFileSync(CHECKLIST_PATH, JSON.stringify(data, null, 2));
            };

            const { action, stage, category, taskId, title, description: desc, priority, status: statusFilter } = params as any;
            const currentStage = stage || data.currentStage;

            const progressBar = (pct: number) => "█".repeat(Math.round(pct / 5)) + "░".repeat(20 - Math.round(pct / 5));

            const getProgress = (s: string) => {
              const tasks = data.tasks.filter(t => t.stage === s);
              const completed = tasks.filter(t => t.status === "completed").length;
              const total = tasks.length;
              return { stage: s, completed, total, percent: total > 0 ? Math.round(completed / total * 100) : 0 };
            };

            switch (action) {
              case "status": {
                const stages = stage ? [stage] : ["idea", "launch", "growth"];
                const all = data.tasks;
                const totalDone = all.filter(t => t.status === "completed").length;
                let text = `📊 Business Setup — ${data.currentStage.toUpperCase()}\n`;
                text += `Overall: ${totalDone}/${all.length} (${all.length > 0 ? Math.round(totalDone/all.length*100) : 0}%)\n\n`;
                for (const s of stages) {
                  const p = getProgress(s);
                  text += `${s.toUpperCase()}: ${progressBar(p.percent)} ${p.percent}% (${p.completed}/${p.total})\n`;
                }
                const next = data.tasks
                  .filter(t => t.stage === currentStage && t.status === "pending")
                  .sort((a, b) => { const pr: Record<string, number> = {critical:0,high:1,medium:2,low:3}; return (pr[a.priority]??2) - (pr[b.priority]??2) || a.order - b.order; })[0];
                if (next) text += `\n▶ Next: ${next.title} [${next.priority}]`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "next": {
                const task = data.tasks
                  .filter(t => t.stage === currentStage && (t.status === "pending" || t.status === "in_progress"))
                  .sort((a, b) => { const pr: Record<string, number> = {critical:0,high:1,medium:2,low:3}; return (pr[a.priority]??2) - (pr[b.priority]??2) || a.order - b.order; })[0];
                if (!task) return { content: [{ type: "text" as const, text: "✅ All tasks in this stage are complete!" }] };
                let text = `▶ ${task.title}\nPriority: ${task.priority} | ${task.stage}/${task.category}\n`;
                if (task.aiSuggestion) text += `💡 ${task.aiSuggestion}\n`;
                text += `ID: ${task.id}`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "complete": {
                if (!taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                task.status = "completed";
                task.completedAt = new Date().toISOString();
                task.updatedAt = new Date().toISOString();
                save();
                const p = getProgress(task.stage);
                return { content: [{ type: "text" as const, text: `✅ ${task.title}\nProgress: ${p.completed}/${p.total} (${p.percent}%)` }] };
              }

              case "skip": {
                if (!taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                task.status = "skipped";
                task.updatedAt = new Date().toISOString();
                save();
                return { content: [{ type: "text" as const, text: `⏭ Skipped: ${task.title}` }] };
              }

              case "add": {
                if (!title) return { content: [{ type: "text" as const, text: "❌ title required" }] };
                const newTask: TaskData = {
                  id: randomUUID(), title, description: desc, stage: stage || data.currentStage,
                  category: category || `${stage || data.currentStage}-custom`,
                  status: "pending", priority: priority || "medium", source: "user",
                  order: 999, createdAt: new Date().toISOString(), updatedAt: new Date().toISOString(),
                };
                data.tasks.push(newTask);
                save();
                return { content: [{ type: "text" as const, text: `➕ ${newTask.title}\nID: ${newTask.id}` }] };
              }

              case "daily": {
                const tasks = data.tasks
                  .filter(t => t.stage === data.currentStage && (t.status === "pending" || t.status === "in_progress") && (t.priority === "critical" || t.priority === "high"))
                  .sort((a, b) => a.order - b.order)
                  .slice(0, 3);
                if (tasks.length === 0) return { content: [{ type: "text" as const, text: "📋 No tasks for today" }] };
                let text = `📋 Daily Stand-Up\n\n`;
                for (const t of tasks) text += `☐ ${t.title} [${t.priority}]\n`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "list": {
                let tasks = data.tasks;
                if (stage) tasks = tasks.filter(t => t.stage === stage);
                if (category) tasks = tasks.filter(t => t.category === category);
                if (statusFilter) tasks = tasks.filter(t => t.status === statusFilter);
                const icons: Record<string, string> = { pending: "☐", in_progress: "⏳", completed: "✅", skipped: "⏭" };
                let text = `Tasks (${tasks.length}):\n\n`;
                for (const t of tasks.slice(0, 25)) {
                  text += `${icons[t.status] || "?"} ${t.title} [${t.priority}] ${t.id.slice(0,8)}\n`;
                }
                if (tasks.length > 25) text += `\n... +${tasks.length - 25} more`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "detail": {
                if (!taskId) return { content: [{ type: "text" as const, text: "❌ taskId required" }] };
                const task = data.tasks.find(t => t.id === taskId);
                if (!task) return { content: [{ type: "text" as const, text: "❌ Task not found" }] };
                let text = `📌 ${task.title}\nStatus: ${task.status} | Priority: ${task.priority}\n`;
                text += `Stage: ${task.stage} | Category: ${task.category}\n`;
                if (task.aiSuggestion) text += `💡 ${task.aiSuggestion}\n`;
                if (task.notes) text += `📝 ${task.notes}\n`;
                text += `ID: ${task.id}`;
                return { content: [{ type: "text" as const, text }] };
              }

              case "set-stage": {
                if (!stage) return { content: [{ type: "text" as const, text: "❌ stage required (idea/launch/growth)" }] };
                data.currentStage = stage;
                save();
                const p = getProgress(stage);
                return { content: [{ type: "text" as const, text: `🔄 Stage: ${stage.toUpperCase()} — ${p.completed}/${p.total} (${p.percent}%)` }] };
              }
            }

            return { content: [{ type: "text" as const, text: "❌ Unknown action" }] };
          } catch (err) {
            return { content: [{ type: "text" as const, text: `Error: ${String(err)}` }] };
          }
        },
      },
      { name: "wirebot_checklist" },
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
