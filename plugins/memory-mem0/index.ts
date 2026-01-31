import { Type } from "@sinclair/typebox";
import type { ClawdbotPluginApi } from "clawdbot/plugin-sdk";
import { stringEnum } from "clawdbot/plugin-sdk";

const MEMORY_CATEGORIES = ["preference", "fact", "decision", "entity", "other"] as const;

type Mem0Config = {
  baseUrl: string;
  apiKey?: string;
  namespacePrefix?: string;
  endpoints: {
    search: string;
    store: string;
    delete: string;
  };
};

function resolveEnvVars(value: string): string {
  return value.replace(/\$\{([^}]+)\}/g, (_, envVar) => {
    const envValue = process.env[envVar];
    if (!envValue) throw new Error(`Environment variable ${envVar} is not set`);
    return envValue;
  });
}

function parseConfig(raw: unknown): Mem0Config {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) {
    throw new Error("mem0 config required");
  }
  const cfg = raw as Record<string, unknown>;
  if (typeof cfg.baseUrl !== "string") throw new Error("baseUrl required");
  if (!cfg.endpoints || typeof cfg.endpoints !== "object") {
    throw new Error("endpoints required");
  }
  const endpoints = cfg.endpoints as Record<string, unknown>;
  const search = endpoints.search;
  const store = endpoints.store;
  const del = endpoints.delete;
  if (typeof search !== "string" || typeof store !== "string" || typeof del !== "string") {
    throw new Error("endpoints.search/store/delete required");
  }
  return {
    baseUrl: resolveEnvVars(cfg.baseUrl),
    apiKey: typeof cfg.apiKey === "string" ? resolveEnvVars(cfg.apiKey) : undefined,
    namespacePrefix: typeof cfg.namespacePrefix === "string" ? cfg.namespacePrefix : "wirebot_",
    endpoints: { search, store, delete: del },
  };
}

async function mem0Request(
  cfg: Mem0Config,
  path: string,
  body: Record<string, unknown>,
) {
  const url = cfg.baseUrl.replace(/\/$/, "") + path;
  const headers: Record<string, string> = { "content-type": "application/json" };
  if (cfg.apiKey) headers.authorization = `Bearer ${cfg.apiKey}`;
  const res = await fetch(url, {
    method: "POST",
    headers,
    body: JSON.stringify(body),
  });
  const text = await res.text();
  if (!res.ok) {
    throw new Error(`Mem0 error ${res.status}: ${text}`);
  }
  return text ? JSON.parse(text) : {};
}

const memoryPlugin = {
  id: "memory-mem0",
  name: "Memory (Mem0)",
  description: "Mem0-backed long-term memory (skeleton)",
  kind: "memory" as const,

  register(api: ClawdbotPluginApi) {
    const cfg = parseConfig(api.pluginConfig);
    api.logger.info(`memory-mem0: registered (baseUrl: ${cfg.baseUrl})`);

    api.registerTool(
      {
        name: "memory_recall",
        label: "Memory Recall",
        description: "Search Mem0 for relevant memories.",
        parameters: Type.Object({
          query: Type.String({ description: "Search query" }),
          limit: Type.Optional(Type.Number({ description: "Max results" })),
        }),
        async execute(_toolCallId, params) {
          const { query, limit = 5 } = params as { query: string; limit?: number };
          const payload = {
            query,
            limit,
            namespace: `${cfg.namespacePrefix}${api.config?.identity?.id ?? "default"}`,
          };
          const result = await mem0Request(cfg, cfg.endpoints.search, payload);
          return {
            content: [{ type: "text", text: JSON.stringify(result, null, 2) }],
            details: { result },
          };
        },
      },
      { name: "memory_recall" },
    );

    api.registerTool(
      {
        name: "memory_store",
        label: "Memory Store",
        description: "Store memory in Mem0.",
        parameters: Type.Object({
          text: Type.String({ description: "Information to remember" }),
          category: Type.Optional(stringEnum(MEMORY_CATEGORIES)),
        }),
        async execute(_toolCallId, params) {
          const { text, category = "other" } = params as { text: string; category?: string };
          const payload = {
            text,
            category,
            namespace: `${cfg.namespacePrefix}${api.config?.identity?.id ?? "default"}`,
          };
          const result = await mem0Request(cfg, cfg.endpoints.store, payload);
          return {
            content: [{ type: "text", text: "Stored memory." }],
            details: { result },
          };
        },
      },
      { name: "memory_store" },
    );

    api.registerTool(
      {
        name: "memory_forget",
        label: "Memory Forget",
        description: "Delete memory in Mem0 by ID.",
        parameters: Type.Object({
          memoryId: Type.String({ description: "Memory ID to delete" }),
        }),
        async execute(_toolCallId, params) {
          const { memoryId } = params as { memoryId: string };
          const payload = {
            id: memoryId,
            namespace: `${cfg.namespacePrefix}${api.config?.identity?.id ?? "default"}`,
          };
          const result = await mem0Request(cfg, cfg.endpoints.delete, payload);
          return {
            content: [{ type: "text", text: `Deleted memory ${memoryId}.` }],
            details: { result },
          };
        },
      },
      { name: "memory_forget" },
    );
  },
};

export default memoryPlugin;
