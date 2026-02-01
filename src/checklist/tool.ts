/**
 * OpenClaw Tool: wirebot_checklist
 *
 * Exposes the Business Setup Checklist Engine as an AI tool.
 * Registered via the wirebot-memory-bridge plugin (or standalone).
 *
 * Actions:
 *   status     — Show progress for current or specified stage
 *   next       — Get the next recommended task
 *   complete   — Mark a task as completed
 *   skip       — Skip a task
 *   add        — Add a custom task
 *   daily      — Generate today's stand-up tasks
 *   list       — List tasks (filterable by stage/category/status)
 *   detail     — Get full details of a specific task
 *   set-stage  — Change the current business stage
 */

import { ChecklistEngine, type BusinessStage, type TaskStatus } from "./engine.js";
import { DEFAULT_CATEGORIES, SEED_TASKS } from "./schema.js";

const CHECKLIST_PATH = "/home/wirebot/clawd/checklist.json";

let engine: ChecklistEngine | null = null;

function getEngine(): ChecklistEngine {
  if (!engine) {
    engine = new ChecklistEngine(CHECKLIST_PATH);
    // Auto-init if empty
    if (engine.getState().tasks.length === 0) {
      engine.initFromTemplate("verious", DEFAULT_CATEGORIES, SEED_TASKS, "Startempire Wire");
      engine.save();
    }
  }
  return engine;
}

export interface ChecklistToolParams {
  action: "status" | "next" | "complete" | "skip" | "add" | "daily" | "list" | "detail" | "set-stage";
  stage?: BusinessStage;
  category?: string;
  taskId?: string;
  title?: string;
  description?: string;
  priority?: "critical" | "high" | "medium" | "low";
  status?: TaskStatus;
}

export function executeChecklist(params: ChecklistToolParams): string {
  const eng = getEngine();

  switch (params.action) {
    case "status": {
      const progress = eng.getProgress(params.stage);
      const overall = eng.getOverallProgress();
      const next = eng.getNextTask(params.stage);

      let result = `📊 Business Setup Progress\n`;
      result += `Stage: ${eng.getCurrentStage().toUpperCase()}\n`;
      result += `Overall: ${overall.completed}/${overall.total} tasks (${overall.percent}%)\n\n`;

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
      const task = eng.getNextTask(params.stage);
      if (!task) return "✅ All tasks in this stage are complete!";

      let result = `▶ Next Task: ${task.title}\n`;
      result += `Priority: ${task.priority} | Stage: ${task.stage} | Category: ${task.category}\n`;
      if (task.description) result += `\n${task.description}\n`;
      if (task.aiSuggestion) result += `\n💡 Tip: ${task.aiSuggestion}\n`;
      result += `\nID: ${task.id}`;
      return result;
    }

    case "complete": {
      if (!params.taskId) return "❌ taskId required";
      const task = eng.completeTask(params.taskId);
      if (!task) return "❌ Task not found";
      eng.save();
      const progress = eng.getOverallProgress();
      return `✅ Completed: ${task.title}\nProgress: ${progress.completed}/${progress.total} (${progress.percent}%)`;
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
      const stage = params.stage || eng.getCurrentStage();
      const category = params.category || `${stage}-custom`;
      const task = eng.addTask({
        title: params.title,
        description: params.description,
        stage,
        category,
        status: "pending",
        priority: params.priority || "medium",
        source: "user",
        order: 999,
      });
      eng.save();
      return `➕ Added: ${task.title} (${stage}/${category})\nID: ${task.id}`;
    }

    case "daily": {
      const standup = eng.generateDailyStandUp();
      if (standup.tasks.length === 0) return "📋 No tasks for today's stand-up";

      let result = `📋 Daily Stand-Up — ${standup.date}\n\n`;
      for (const t of standup.tasks) {
        result += `☐ ${t.title}\n`;
      }
      return result;
    }

    case "list": {
      const tasks = eng.getTasks(params.stage, params.category);
      const filtered = params.status
        ? tasks.filter((t) => t.status === params.status)
        : tasks;

      if (filtered.length === 0) return "No tasks match the filter.";

      const statusIcon = (s: TaskStatus) =>
        ({ pending: "☐", in_progress: "⏳", completed: "✅", skipped: "⏭" })[s];

      let result = `Tasks (${filtered.length}):\n\n`;
      for (const t of filtered.slice(0, 20)) {
        result += `${statusIcon(t.status)} ${t.title} [${t.priority}] ${t.id.slice(0, 8)}\n`;
      }
      if (filtered.length > 20) result += `\n... and ${filtered.length - 20} more`;
      return result;
    }

    case "detail": {
      if (!params.taskId) return "❌ taskId required";
      const task = eng.getTask(params.taskId);
      if (!task) return "❌ Task not found";

      let result = `📌 ${task.title}\n`;
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
      if (!params.stage) return "❌ stage required (idea/launch/growth)";
      eng.setStage(params.stage);
      eng.save();
      const progress = eng.getProgress(params.stage);
      return `🔄 Stage set to: ${params.stage.toUpperCase()}\nProgress: ${progress[0].completed}/${progress[0].total} (${progress[0].percent}%)`;
    }

    default:
      return `❌ Unknown action: ${params.action}. Valid: status, next, complete, skip, add, daily, list, detail, set-stage`;
  }
}

function progressBar(percent: number): string {
  const filled = Math.round(percent / 5);
  const empty = 20 - filled;
  return "█".repeat(filled) + "░".repeat(empty);
}
