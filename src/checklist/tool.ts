/**
 * OpenClaw Tool: wirebot_checklist (v2 — Multi-Business)
 *
 * Exposes the Business Setup Checklist Engine as an AI tool.
 * Registered via the wirebot-memory-bridge plugin (or standalone).
 *
 * Actions:
 *   status       — Show progress for active (or specified) business
 *   overview     — Operator-level view of ALL businesses
 *   businesses   — List all businesses with health scores
 *   focus        — Switch active business
 *   add-business — Add a new business
 *   next         — Get the next recommended task (business or global)
 *   complete     — Mark a task as completed
 *   skip         — Skip a task
 *   add          — Add a custom task
 *   daily        — Generate today's stand-up tasks (multi-business)
 *   list         — List tasks (filterable)
 *   detail       — Get full details of a specific task
 *   set-stage    — Change a business's stage
 */

import { ChecklistEngine, type BusinessStage, type TaskStatus } from "./engine.js";

const CHECKLIST_PATH = "/home/wirebot/clawd/checklist.json";

let engine: ChecklistEngine | null = null;

function getEngine(): ChecklistEngine {
  if (!engine) {
    engine = new ChecklistEngine(CHECKLIST_PATH);
    // Auto-init if empty and no businesses exist
    if (engine.getBusinesses().length === 0) {
      engine.initFromTemplate("verious", "Startempire Wire", "SEW");
      engine.save();
    }
    // Auto-migrate: save if dirty from v1→v2 migration
    if (engine.isDirty()) {
      engine.save();
    }
  }
  return engine;
}

export interface ChecklistToolParams {
  action: "status" | "overview" | "businesses" | "focus" | "add-business" |
          "next" | "complete" | "skip" | "add" | "daily" | "list" | "detail" | "set-stage";
  stage?: BusinessStage;
  category?: string;
  taskId?: string;
  businessId?: string;
  businessName?: string;
  title?: string;
  description?: string;
  priority?: "critical" | "high" | "medium" | "low";
  status?: TaskStatus;
  // add-business fields
  shortName?: string;
  role?: string;
  revenueStatus?: string;
  businessPriority?: string;
  domain?: string;
}

export function executeChecklist(params: ChecklistToolParams): string {
  const eng = getEngine();

  switch (params.action) {
    case "overview": {
      return eng.getOperatorOverview();
    }

    case "businesses": {
      const health = eng.getAllBusinessHealth();
      if (health.length === 0) return "No businesses configured. Use add-business.";

      const active = eng.getActiveBusiness();
      let result = "⚡ Your Businesses\n\n";
      const sigIcon = { healthy: "🟢", attention: "🟡", stale: "🟠", critical: "🔴" };

      for (const h of health) {
        const icon = sigIcon[h.signal];
        const isActive = h.businessId === active?.id ? " ◀ active" : "";
        result += `${icon} ${h.businessName.padEnd(24)} [${h.stage}]  Health: ${h.health}/100${isActive}\n`;
        result += `   ${h.checklistPercent}% setup | ${h.revenueStatus} | ${h.priority}`;
        if (h.daysSinceActivity > 7) result += ` | ${h.daysSinceActivity}d stale`;
        if (h.criticalBlocked > 0) result += ` | ${h.criticalBlocked} blocked`;
        result += "\n";
      }

      return result;
    }

    case "focus": {
      const name = params.businessName || params.businessId;
      if (!name) return "❌ businessName or businessId required";

      let biz = eng.getBusiness(name);
      if (!biz) biz = eng.getBusinessByName(name);
      if (!biz) return `❌ Business not found: ${name}`;

      eng.setActiveBusiness(biz.id);
      eng.save();
      const progress = eng.getOverallProgress(biz.id);
      const health = eng.getBusinessHealth(biz.id);
      return `⚡ Active business: ${biz.name} [${biz.stage}]\n` +
             `Health: ${health.health}/100 | Progress: ${progress.completed}/${progress.total} (${progress.percent}%)`;
    }

    case "add-business": {
      if (!params.title && !params.businessName) return "❌ businessName or title required";
      const name = params.businessName || params.title!;
      const biz = eng.addBusiness({
        name,
        shortName: params.shortName,
        description: params.description,
        stage: params.stage || "idea",
        role: (params.role as any) || "founder",
        revenueStatus: (params.revenueStatus as any) || "pre-revenue",
        priority: (params.businessPriority as any) || "secondary",
        domain: params.domain,
        seedTasks: true,
      });
      eng.save();
      return `➕ Added business: ${biz.name} (${biz.shortName}) [${biz.stage}]\n` +
             `ID: ${biz.id}\n` +
             `64 template tasks seeded. Use 'focus' to switch to it.`;
    }

    case "status": {
      const bizId = params.businessId ||
        (params.businessName ? eng.getBusinessByName(params.businessName)?.id : undefined) ||
        eng.getActiveBusiness()?.id;

      if (!bizId) return "❌ No active business. Use add-business first.";

      const biz = eng.getBusiness(bizId);
      if (!biz) return "❌ Business not found";

      const progress = eng.getProgress(bizId, params.stage);
      const overall = eng.getOverallProgress(bizId);
      const next = eng.getNextTask(bizId);
      const health = eng.getBusinessHealth(bizId);
      const sigIcon = { healthy: "🟢", attention: "🟡", stale: "🟠", critical: "🔴" };

      let result = `📊 ${biz.name} — ${biz.stage.toUpperCase()}\n`;
      result += `${sigIcon[health.signal]} Health: ${health.health}/100 | Overall: ${overall.completed}/${overall.total} (${overall.percent}%)\n\n`;

      for (const p of progress) {
        const bar = progressBar(p.percent);
        result += `${p.stage.toUpperCase()}: ${bar} ${p.percent}% (${p.completed}/${p.total})\n`;
      }

      if (next) {
        result += `\n▶ Next: ${next.title} [${next.priority}]`;
      }

      return result;
    }

    case "next": {
      // If businessId specified, get next for that business
      // Otherwise get global next (most important across all businesses)
      if (params.businessId || params.businessName) {
        const bizId = params.businessId ||
          (params.businessName ? eng.getBusinessByName(params.businessName)?.id : undefined);
        const task = eng.getNextTask(bizId);
        if (!task) return "✅ All tasks in this business are complete!";

        const biz = eng.getBusiness(task.businessId);
        let result = `▶ ${task.title}`;
        if (biz) result += ` (${biz.shortName})`;
        result += `\nPriority: ${task.priority} | ${task.stage}/${task.category}`;
        if (task.aiSuggestion) result += `\n💡 ${task.aiSuggestion}`;
        result += `\nID: ${task.id}`;
        return result;
      }

      // Global next
      const global = eng.getGlobalNextTask();
      if (!global) return "✅ All tasks across all businesses are complete!";

      let result = `▶ ${global.task.title}`;
      result += ` (${global.business.shortName})`;
      result += `\nPriority: ${global.task.priority} | ${global.task.stage}/${global.task.category}`;
      if (global.task.aiSuggestion) result += `\n💡 ${global.task.aiSuggestion}`;
      result += `\nID: ${global.task.id}`;
      return result;
    }

    case "complete": {
      if (!params.taskId) return "❌ taskId required";
      const task = eng.completeTask(params.taskId);
      if (!task) return "❌ Task not found";
      eng.save();
      const biz = eng.getBusiness(task.businessId);
      const progress = eng.getOverallProgress(task.businessId);
      return `✅ Completed: ${task.title}` +
             (biz ? ` (${biz.shortName})` : "") +
             `\nProgress: ${progress.completed}/${progress.total} (${progress.percent}%)`;
    }

    case "skip": {
      if (!params.taskId) return "❌ taskId required";
      const task = eng.skipTask(params.taskId);
      if (!task) return "❌ Task not found";
      eng.save();
      return `⏭ Skipped: ${task.title}`;
    }

    case "add": {
      if (!params.title) return "❌ title required";
      const bizId = params.businessId ||
        (params.businessName ? eng.getBusinessByName(params.businessName)?.id : undefined) ||
        eng.getActiveBusiness()?.id;
      if (!bizId) return "❌ No active business";

      const biz = eng.getBusiness(bizId);
      const stage = params.stage || eng.getCurrentStage(bizId);
      const category = params.category || `${stage}-custom`;
      const task = eng.addTask({
        title: params.title,
        description: params.description,
        businessId: bizId,
        stage,
        category,
        status: "pending",
        priority: params.priority || "medium",
        source: "user",
        order: 999,
      });
      eng.save();
      return `➕ Added: ${task.title}` +
             (biz ? ` (${biz.shortName})` : "") +
             ` [${stage}/${category}]\nID: ${task.id}`;
    }

    case "daily": {
      const standup = eng.generateDailyStandUp();
      if (standup.tasks.length === 0 && standup.crossCutting.length === 0) {
        return "📋 No tasks for today's stand-up";
      }

      let result = `📋 Daily Stand-Up — ${standup.date}\n`;
      if (standup.focusRecommendation) {
        result += `💡 ${standup.focusRecommendation}\n`;
      }
      result += `\n`;

      // Group by business
      const byBiz = new Map<string, typeof standup.tasks>();
      for (const t of standup.tasks) {
        if (!byBiz.has(t.businessName)) byBiz.set(t.businessName, []);
        byBiz.get(t.businessName)!.push(t);
      }

      for (const [bizName, tasks] of byBiz) {
        result += `${bizName}:\n`;
        for (const t of tasks) {
          result += `  ☐ ${t.title}\n`;
        }
      }

      if (standup.crossCutting.length > 0) {
        result += `\nCROSS-CUTTING:\n`;
        for (const t of standup.crossCutting) {
          result += `  ☐ ${t.title}\n`;
        }
      }

      return result;
    }

    case "list": {
      const bizId = params.businessId ||
        (params.businessName ? eng.getBusinessByName(params.businessName)?.id : undefined) ||
        eng.getActiveBusiness()?.id;

      const tasks = eng.getTasks({
        businessId: bizId || undefined,
        stage: params.stage,
        category: params.category,
      });
      const filtered = params.status
        ? tasks.filter((t) => t.status === params.status)
        : tasks;

      if (filtered.length === 0) return "No tasks match the filter.";

      const statusIcon = (s: TaskStatus) =>
        ({ pending: "☐", in_progress: "⏳", completed: "✅", skipped: "⏭" })[s];

      let result = `Tasks (${filtered.length}):\n\n`;
      for (const t of filtered.slice(0, 20)) {
        const biz = eng.getBusiness(t.businessId);
        const prefix = biz ? `[${biz.shortName}] ` : "";
        result += `${statusIcon(t.status)} ${prefix}${t.title} [${t.priority}] ${t.id.slice(0, 8)}\n`;
      }
      if (filtered.length > 20) result += `\n... and ${filtered.length - 20} more`;
      return result;
    }

    case "detail": {
      if (!params.taskId) return "❌ taskId required";
      const task = eng.getTask(params.taskId);
      if (!task) return "❌ Task not found";
      const biz = eng.getBusiness(task.businessId);

      let result = `📌 ${task.title}\n`;
      if (biz) result += `Business: ${biz.name} (${biz.shortName})\n`;
      result += `Status: ${task.status} | Priority: ${task.priority}\n`;
      result += `Stage: ${task.stage} | Category: ${task.category}\n`;
      result += `Source: ${task.source} | Created: ${task.createdAt.split("T")[0]}\n`;
      if (task.completedAt) result += `Completed: ${task.completedAt.split("T")[0]}\n`;
      if (task.dueDate) result += `Due: ${task.dueDate}\n`;
      if (task.description) result += `\n${task.description}\n`;
      if (task.aiSuggestion) result += `\n💡 ${task.aiSuggestion}\n`;
      if (task.notes) result += `\n📝 ${task.notes}\n`;
      result += `\nID: ${task.id}`;
      return result;
    }

    case "set-stage": {
      if (!params.stage) return "❌ stage required (idea/launch/growth/mature/sunset)";
      const bizId = params.businessId ||
        (params.businessName ? eng.getBusinessByName(params.businessName)?.id : undefined);
      eng.setStage(params.stage, bizId);
      eng.save();
      const biz = eng.getBusiness(bizId || eng.getActiveBusiness()?.id || "");
      const progress = eng.getProgress(bizId, params.stage);
      return `🔄 ${biz?.name || "Business"} stage set to: ${params.stage.toUpperCase()}\n` +
             `Progress: ${progress[0]?.completed || 0}/${progress[0]?.total || 0} (${progress[0]?.percent || 0}%)`;
    }

    default:
      return `❌ Unknown action: ${params.action}. Valid: status, overview, businesses, focus, add-business, next, complete, skip, add, daily, list, detail, set-stage`;
  }
}

function progressBar(percent: number): string {
  const filled = Math.round(percent / 5);
  const empty = 20 - filled;
  return "█".repeat(filled) + "░".repeat(empty);
}
