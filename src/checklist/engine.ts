/**
 * Business Setup Checklist Engine
 *
 * CRUD operations, progress tracking, AI integration hooks,
 * and workspace file sync for the checklist data model.
 *
 * Storage: /home/wirebot/clawd/checklist.json
 * Sync: Changes update Letta business_stage block via bridge
 */

import { randomUUID } from "node:crypto";
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import type {
  BusinessStage,
  ChecklistState,
  DailyStandUp,
  DailyTask,
  StageProgress,
  Task,
  TaskStatus,
  DEFAULT_CATEGORIES,
  SEED_TASKS,
} from "./schema.js";

// Re-export for convenience
export type { BusinessStage, Task, TaskStatus, StageProgress, DailyStandUp };

// ============================================================================
// Engine
// ============================================================================

export class ChecklistEngine {
  private state: ChecklistState;
  private filePath: string;
  private dirty = false;

  constructor(filePath: string) {
    this.filePath = filePath;
    this.state = this.load();
  }

  // ── Persistence ─────────────────────────────────────────────────────────

  private load(): ChecklistState {
    if (existsSync(this.filePath)) {
      const raw = readFileSync(this.filePath, "utf-8");
      return JSON.parse(raw) as ChecklistState;
    }
    // Return empty state — caller should call initFromTemplate()
    return {
      version: 1,
      userId: "",
      currentStage: "idea",
      tasks: [],
      categories: [],
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };
  }

  save(): void {
    this.state.updatedAt = new Date().toISOString();
    writeFileSync(this.filePath, JSON.stringify(this.state, null, 2));
    this.dirty = false;
  }

  isDirty(): boolean {
    return this.dirty;
  }

  // ── Initialize from template ────────────────────────────────────────────

  initFromTemplate(
    userId: string,
    categories: typeof DEFAULT_CATEGORIES,
    seedTasks: typeof SEED_TASKS,
    businessName?: string
  ): void {
    const now = new Date().toISOString();
    this.state = {
      version: 1,
      userId,
      businessName,
      currentStage: "idea",
      categories: [...categories],
      tasks: seedTasks.map((t) => ({
        ...t,
        id: randomUUID(),
        createdAt: now,
        updatedAt: now,
      })),
      createdAt: now,
      updatedAt: now,
    };
    this.dirty = true;
  }

  // ── Stage ───────────────────────────────────────────────────────────────

  getCurrentStage(): BusinessStage {
    return this.state.currentStage;
  }

  setStage(stage: BusinessStage): void {
    this.state.currentStage = stage;
    this.dirty = true;
  }

  // ── Tasks: CRUD ─────────────────────────────────────────────────────────

  getTasks(stage?: BusinessStage, category?: string): Task[] {
    let tasks = this.state.tasks;
    if (stage) tasks = tasks.filter((t) => t.stage === stage);
    if (category) tasks = tasks.filter((t) => t.category === category);
    return tasks.sort((a, b) => a.order - b.order);
  }

  getTask(id: string): Task | undefined {
    return this.state.tasks.find((t) => t.id === id);
  }

  addTask(task: Omit<Task, "id" | "createdAt" | "updatedAt">): Task {
    const now = new Date().toISOString();
    const newTask: Task = {
      ...task,
      id: randomUUID(),
      createdAt: now,
      updatedAt: now,
    };
    this.state.tasks.push(newTask);
    this.dirty = true;
    return newTask;
  }

  updateTask(id: string, updates: Partial<Pick<Task, "title" | "description" | "status" | "priority" | "notes" | "dueDate" | "aiSuggestion" | "order">>): Task | undefined {
    const task = this.state.tasks.find((t) => t.id === id);
    if (!task) return undefined;

    Object.assign(task, updates);
    task.updatedAt = new Date().toISOString();

    // Auto-set completedAt
    if (updates.status === "completed" && !task.completedAt) {
      task.completedAt = new Date().toISOString();
    }
    if (updates.status && updates.status !== "completed") {
      task.completedAt = undefined;
    }

    this.dirty = true;
    return task;
  }

  completeTask(id: string): Task | undefined {
    return this.updateTask(id, { status: "completed" });
  }

  skipTask(id: string): Task | undefined {
    return this.updateTask(id, { status: "skipped" });
  }

  deleteTask(id: string): boolean {
    const idx = this.state.tasks.findIndex((t) => t.id === id);
    if (idx === -1) return false;
    this.state.tasks.splice(idx, 1);
    this.dirty = true;
    return true;
  }

  // ── Progress ────────────────────────────────────────────────────────────

  getProgress(stage?: BusinessStage): StageProgress[] {
    const stages: BusinessStage[] = stage ? [stage] : ["idea", "launch", "growth"];
    return stages.map((s) => {
      const tasks = this.state.tasks.filter((t) => t.stage === s);
      const completed = tasks.filter((t) => t.status === "completed").length;
      const skipped = tasks.filter((t) => t.status === "skipped").length;
      const total = tasks.length;
      const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
      return { stage: s, total, completed, skipped, percent };
    });
  }

  getOverallProgress(): { total: number; completed: number; percent: number } {
    const total = this.state.tasks.length;
    const completed = this.state.tasks.filter((t) => t.status === "completed").length;
    const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
    return { total, completed, percent };
  }

  // ── Next Task ───────────────────────────────────────────────────────────

  getNextTask(stage?: BusinessStage): Task | undefined {
    const s = stage || this.state.currentStage;
    const pending = this.state.tasks
      .filter((t) => t.stage === s && (t.status === "pending" || t.status === "in_progress"))
      .sort((a, b) => {
        // Priority order: critical > high > medium > low
        const prio = { critical: 0, high: 1, medium: 2, low: 3 };
        const diff = prio[a.priority] - prio[b.priority];
        if (diff !== 0) return diff;
        return a.order - b.order;
      });

    // Check dependencies
    for (const task of pending) {
      if (!task.dependencies?.length) return task;
      const allDepsMet = task.dependencies.every((depId) => {
        const dep = this.getTask(depId);
        return dep && (dep.status === "completed" || dep.status === "skipped");
      });
      if (allDepsMet) return task;
    }

    return pending[0]; // Fallback to highest priority even if deps unmet
  }

  // ── Daily Stand-Up ──────────────────────────────────────────────────────

  generateDailyStandUp(date?: string): DailyStandUp {
    const d = date || new Date().toISOString().split("T")[0];
    const stage = this.state.currentStage;

    // Pick top tasks for today: in_progress first, then next critical/high pending
    const inProgress = this.state.tasks.filter(
      (t) => t.stage === stage && t.status === "in_progress"
    );
    const pending = this.state.tasks
      .filter((t) => t.stage === stage && t.status === "pending" && (t.priority === "critical" || t.priority === "high"))
      .sort((a, b) => a.order - b.order)
      .slice(0, Math.max(0, 3 - inProgress.length));

    const dailyTasks: DailyTask[] = [...inProgress, ...pending].map((t) => ({
      taskId: t.id,
      title: t.title,
      completed: false,
    }));

    return { date: d, tasks: dailyTasks };
  }

  // ── Export for Letta sync ───────────────────────────────────────────────

  toLettaSummary(): string {
    const progress = this.getProgress();
    const next = this.getNextTask();
    const overall = this.getOverallProgress();

    let summary = `Stage: ${this.state.currentStage}\n`;
    summary += `Overall: ${overall.completed}/${overall.total} (${overall.percent}%)\n\n`;

    for (const p of progress) {
      summary += `${p.stage}: ${p.completed}/${p.total} (${p.percent}%)\n`;
    }

    if (next) {
      summary += `\nNext task: ${next.title} [${next.priority}]`;
      if (next.category) {
        const cat = this.state.categories.find((c) => c.id === next.category);
        summary += ` (${cat?.name || next.category})`;
      }
    }

    return summary;
  }

  // ── State access ────────────────────────────────────────────────────────

  getState(): Readonly<ChecklistState> {
    return this.state;
  }

  getCategories(stage?: BusinessStage): typeof this.state.categories {
    if (stage) return this.state.categories.filter((c) => c.stage === stage);
    return this.state.categories;
  }
}
