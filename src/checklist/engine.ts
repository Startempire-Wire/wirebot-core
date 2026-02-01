/**
 * Business Setup Checklist Engine — v2 (Multi-Business)
 *
 * Operator-centric: manages multiple businesses with individual
 * checklists, health scoring, and cross-cutting tasks.
 *
 * Storage: /home/wirebot/clawd/checklist.json
 * Sync: Changes update Letta business_stage block via bridge
 */

import { randomUUID } from "node:crypto";
import { readFileSync, writeFileSync, existsSync } from "node:fs";
import type {
  Business,
  BusinessHealth,
  BusinessPriority,
  BusinessStage,
  ChecklistState,
  ChecklistStateV1,
  DailyStandUp,
  DailyTask,
  StageProgress,
  Task,
  TaskCategory,
  TaskStatus,
} from "./schema.js";
import { DEFAULT_CATEGORIES, SEED_TASKS } from "./schema.js";

// Re-export for convenience
export type { Business, BusinessStage, BusinessHealth, Task, TaskStatus, StageProgress, DailyStandUp };

// ============================================================================
// Migration: v1 → v2
// ============================================================================

function migrateV1toV2(v1: ChecklistStateV1): ChecklistState {
  const defaultBusiness: Business = {
    id: randomUUID(),
    name: v1.businessName || "My Business",
    shortName: (v1.businessName || "BIZ").slice(0, 3).toUpperCase(),
    description: "",
    stage: v1.currentStage as BusinessStage,
    role: "founder",
    revenueStatus: "pre-revenue",
    priority: "primary",
    createdAt: v1.createdAt,
    updatedAt: v1.updatedAt,
  };

  const tasks: Task[] = v1.tasks.map((t) => ({
    ...t,
    businessId: t.businessId || defaultBusiness.id,
    stage: t.stage as BusinessStage,
  }));

  return {
    version: 2,
    operatorId: v1.userId,
    businesses: [defaultBusiness],
    activeBusiness: defaultBusiness.id,
    tasks,
    categories: v1.categories,
    createdAt: v1.createdAt,
    updatedAt: new Date().toISOString(),
  };
}

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
      const parsed = JSON.parse(raw);

      // Auto-migrate v1 → v2
      if (!parsed.version || parsed.version === 1) {
        const migrated = migrateV1toV2(parsed as ChecklistStateV1);
        this.dirty = true;
        return migrated;
      }

      return parsed as ChecklistState;
    }
    // Return empty v2 state — caller should call initFromTemplate()
    return {
      version: 2,
      operatorId: "",
      businesses: [],
      activeBusiness: "",
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
    operatorId: string,
    businessName: string,
    businessShortName?: string
  ): Business {
    const business: Business = {
      id: randomUUID(),
      name: businessName,
      shortName: businessShortName || businessName.slice(0, 3).toUpperCase(),
      description: "",
      stage: "idea",
      role: "founder",
      revenueStatus: "pre-revenue",
      priority: "primary",
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    };

    const now = new Date().toISOString();
    const tasks: Task[] = SEED_TASKS.map((t) => ({
      ...t,
      id: randomUUID(),
      businessId: business.id,
      createdAt: now,
      updatedAt: now,
    }));

    this.state = {
      version: 2,
      operatorId,
      businesses: [business],
      activeBusiness: business.id,
      tasks,
      categories: [...DEFAULT_CATEGORIES],
      createdAt: now,
      updatedAt: now,
    };
    this.dirty = true;
    return business;
  }

  // ── Businesses ──────────────────────────────────────────────────────────

  getBusinesses(): Business[] {
    return this.state.businesses;
  }

  getBusiness(id: string): Business | undefined {
    return this.state.businesses.find((b) => b.id === id);
  }

  getBusinessByName(name: string): Business | undefined {
    const lower = name.toLowerCase();
    return this.state.businesses.find(
      (b) => b.name.toLowerCase() === lower || b.shortName.toLowerCase() === lower
    );
  }

  getActiveBusiness(): Business | undefined {
    return this.state.businesses.find((b) => b.id === this.state.activeBusiness);
  }

  setActiveBusiness(id: string): boolean {
    if (!this.state.businesses.find((b) => b.id === id)) return false;
    this.state.activeBusiness = id;
    this.dirty = true;
    return true;
  }

  addBusiness(opts: {
    name: string;
    shortName?: string;
    description?: string;
    stage?: BusinessStage;
    role?: Business["role"];
    revenueStatus?: Business["revenueStatus"];
    priority?: BusinessPriority;
    domain?: string;
    relatedTo?: string[];
    tags?: string[];
    seedTasks?: boolean;
  }): Business {
    const now = new Date().toISOString();
    const biz: Business = {
      id: randomUUID(),
      name: opts.name,
      shortName: opts.shortName || opts.name.slice(0, 3).toUpperCase(),
      description: opts.description || "",
      stage: opts.stage || "idea",
      role: opts.role || "founder",
      revenueStatus: opts.revenueStatus || "pre-revenue",
      priority: opts.priority || "secondary",
      domain: opts.domain,
      relatedTo: opts.relatedTo,
      tags: opts.tags,
      createdAt: now,
      updatedAt: now,
    };

    this.state.businesses.push(biz);

    // Seed template tasks for this business
    if (opts.seedTasks !== false) {
      const tasks: Task[] = SEED_TASKS.map((t) => ({
        ...t,
        id: randomUUID(),
        businessId: biz.id,
        createdAt: now,
        updatedAt: now,
      }));
      this.state.tasks.push(...tasks);
    }

    this.dirty = true;
    return biz;
  }

  updateBusiness(id: string, updates: Partial<Omit<Business, "id" | "createdAt">>): Business | undefined {
    const biz = this.state.businesses.find((b) => b.id === id);
    if (!biz) return undefined;
    Object.assign(biz, updates);
    biz.updatedAt = new Date().toISOString();
    this.dirty = true;
    return biz;
  }

  // ── Stage (per business) ────────────────────────────────────────────────

  getCurrentStage(businessId?: string): BusinessStage {
    const biz = businessId
      ? this.getBusiness(businessId)
      : this.getActiveBusiness();
    return biz?.stage || "idea";
  }

  setStage(stage: BusinessStage, businessId?: string): void {
    const id = businessId || this.state.activeBusiness;
    const biz = this.state.businesses.find((b) => b.id === id);
    if (biz) {
      biz.stage = stage;
      biz.updatedAt = new Date().toISOString();
      this.dirty = true;
    }
  }

  // ── Tasks: CRUD ─────────────────────────────────────────────────────────

  getTasks(opts?: { stage?: BusinessStage; category?: string; businessId?: string }): Task[] {
    let tasks = this.state.tasks;
    if (opts?.businessId) tasks = tasks.filter((t) => t.businessId === opts.businessId);
    if (opts?.stage) tasks = tasks.filter((t) => t.stage === opts.stage);
    if (opts?.category) tasks = tasks.filter((t) => t.category === opts.category);
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

  updateTask(
    id: string,
    updates: Partial<Pick<Task, "title" | "description" | "status" | "priority" | "notes" | "dueDate" | "aiSuggestion" | "order">>
  ): Task | undefined {
    const task = this.state.tasks.find((t) => t.id === id);
    if (!task) return undefined;

    Object.assign(task, updates);
    task.updatedAt = new Date().toISOString();

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

  // ── Progress (per business) ─────────────────────────────────────────────

  getProgress(businessId?: string, stage?: BusinessStage): StageProgress[] {
    const bizId = businessId || this.state.activeBusiness;
    const stages: BusinessStage[] = stage ? [stage] : ["idea", "launch", "growth"];
    return stages.map((s) => {
      const tasks = this.state.tasks.filter((t) => t.businessId === bizId && t.stage === s);
      const completed = tasks.filter((t) => t.status === "completed").length;
      const skipped = tasks.filter((t) => t.status === "skipped").length;
      const total = tasks.length;
      const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
      return { stage: s, total, completed, skipped, percent };
    });
  }

  getOverallProgress(businessId?: string): { total: number; completed: number; percent: number } {
    const bizId = businessId || this.state.activeBusiness;
    const tasks = this.state.tasks.filter((t) => t.businessId === bizId);
    const total = tasks.length;
    const completed = tasks.filter((t) => t.status === "completed").length;
    const percent = total > 0 ? Math.round((completed / total) * 100) : 0;
    return { total, completed, percent };
  }

  // ── Business Health Scoring ─────────────────────────────────────────────

  getBusinessHealth(businessId: string): BusinessHealth {
    const biz = this.getBusiness(businessId);
    if (!biz) {
      return {
        businessId,
        businessName: "Unknown",
        shortName: "???",
        stage: "idea",
        priority: "passive",
        health: 0,
        checklistPercent: 0,
        daysSinceActivity: 999,
        criticalBlocked: 0,
        revenueStatus: "pre-revenue",
        signal: "critical",
      };
    }

    const bizTasks = this.state.tasks.filter((t) => t.businessId === businessId);
    const completed = bizTasks.filter((t) => t.status === "completed").length;
    const total = bizTasks.length;
    const checklistPercent = total > 0 ? Math.round((completed / total) * 100) : 0;

    // Days since last activity (any task update)
    const lastActivity = bizTasks.reduce((latest, t) => {
      const d = new Date(t.updatedAt).getTime();
      return d > latest ? d : latest;
    }, new Date(biz.updatedAt).getTime());
    const daysSinceActivity = Math.floor((Date.now() - lastActivity) / (1000 * 60 * 60 * 24));

    // Critical blocked tasks
    const criticalBlocked = bizTasks.filter(
      (t) => t.priority === "critical" && (t.status === "pending" || t.status === "in_progress")
    ).length;

    // Health score calculation
    let health = 0;
    // Checklist progress (20%)
    health += checklistPercent * 0.2;
    // Revenue (25%): active=25, pre-rev=10, declining=5, paused=0
    const revScores = { active: 25, "pre-revenue": 10, declining: 5, paused: 0 };
    health += revScores[biz.revenueStatus] || 0;
    // Operator attention (15%): 0 days=15, 7 days=10, 14 days=5, 30+=0
    health += Math.max(0, 15 - daysSinceActivity);
    // Blockers (20%): fewer is better
    health += Math.max(0, 20 - criticalBlocked * 4);
    // Stage alignment (10%): having tasks = aligned
    health += total > 0 ? 10 : 0;
    // Dependency health (10%): placeholder — uses relatedTo in future
    health += 10;

    health = Math.min(100, Math.max(0, Math.round(health)));

    // Signal
    let signal: BusinessHealth["signal"] = "healthy";
    if (health < 30) signal = "critical";
    else if (health < 50 || daysSinceActivity > 14) signal = "stale";
    else if (health < 70) signal = "attention";

    return {
      businessId,
      businessName: biz.name,
      shortName: biz.shortName,
      stage: biz.stage,
      priority: biz.priority,
      health,
      checklistPercent,
      daysSinceActivity,
      criticalBlocked,
      revenueStatus: biz.revenueStatus,
      signal,
    };
  }

  getAllBusinessHealth(): BusinessHealth[] {
    return this.state.businesses
      .map((b) => this.getBusinessHealth(b.id))
      .sort((a, b) => a.health - b.health); // Worst health first
  }

  // ── Next Task (multi-business aware) ────────────────────────────────────

  getNextTask(businessId?: string): Task | undefined {
    const bizId = businessId || this.state.activeBusiness;
    const biz = this.getBusiness(bizId);
    const stage = biz?.stage || "idea";

    const pending = this.state.tasks
      .filter(
        (t) =>
          t.businessId === bizId &&
          t.stage === stage &&
          (t.status === "pending" || t.status === "in_progress")
      )
      .sort((a, b) => {
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

    return pending[0];
  }

  /** Get the most important next task across ALL businesses */
  getGlobalNextTask(): { task: Task; business: Business } | undefined {
    // Priority order: primary businesses first, then by health (worst first)
    const healthAll = this.getAllBusinessHealth();
    const prioOrder: Record<BusinessPriority, number> = {
      primary: 0,
      secondary: 1,
      supporting: 2,
      passive: 3,
    };

    const sorted = healthAll.sort((a, b) => {
      const pa = prioOrder[a.priority] ?? 3;
      const pb = prioOrder[b.priority] ?? 3;
      if (pa !== pb) return pa - pb;
      return a.health - b.health; // Worse health = needs attention first
    });

    for (const h of sorted) {
      const task = this.getNextTask(h.businessId);
      const biz = this.getBusiness(h.businessId);
      if (task && biz) return { task, business: biz };
    }
    return undefined;
  }

  // ── Daily Stand-Up (multi-business) ─────────────────────────────────────

  generateDailyStandUp(date?: string): DailyStandUp {
    const d = date || new Date().toISOString().split("T")[0];
    const dailyTasks: DailyTask[] = [];
    const crossCutting: DailyTask[] = [];
    let focusRecommendation: string | undefined;

    // Gather top tasks per business
    for (const biz of this.state.businesses) {
      const stage = biz.stage;
      const inProgress = this.state.tasks.filter(
        (t) => t.businessId === biz.id && t.status === "in_progress"
      );
      const pending = this.state.tasks
        .filter(
          (t) =>
            t.businessId === biz.id &&
            t.stage === stage &&
            t.status === "pending" &&
            (t.priority === "critical" || t.priority === "high")
        )
        .sort((a, b) => a.order - b.order)
        .slice(0, Math.max(0, 2 - inProgress.length));

      for (const t of [...inProgress, ...pending]) {
        const entry: DailyTask = {
          taskId: t.id,
          businessId: biz.id,
          businessName: biz.shortName,
          title: t.title,
          completed: false,
        };
        if (t.crossCutting) {
          crossCutting.push(entry);
        } else {
          dailyTasks.push(entry);
        }
      }
    }

    // Focus recommendation: worst-health primary business
    const health = this.getAllBusinessHealth();
    const primaryWorst = health.find((h) => h.priority === "primary");
    if (primaryWorst && primaryWorst.signal !== "healthy") {
      focusRecommendation = `${primaryWorst.businessName} needs attention (health: ${primaryWorst.health}/100)`;
    }

    return { date: d, tasks: dailyTasks, crossCutting, focusRecommendation };
  }

  // ── Export for Letta sync ───────────────────────────────────────────────

  toLettaSummary(): string {
    const businesses = this.state.businesses;
    if (businesses.length === 0) return "No businesses configured.";

    let summary = `Operator: ${this.state.operatorId}\n`;
    summary += `Businesses: ${businesses.length}\n\n`;

    for (const biz of businesses) {
      const progress = this.getOverallProgress(biz.id);
      const health = this.getBusinessHealth(biz.id);
      const next = this.getNextTask(biz.id);
      const signal = { healthy: "🟢", attention: "🟡", stale: "🟠", critical: "🔴" }[health.signal];

      summary += `${signal} ${biz.name} [${biz.stage}] — ${biz.priority}\n`;
      summary += `   Health: ${health.health}/100 | Progress: ${progress.completed}/${progress.total} (${progress.percent}%)\n`;
      if (next) summary += `   Next: ${next.title} [${next.priority}]\n`;
      summary += `\n`;
    }

    const global = this.getGlobalNextTask();
    if (global) {
      summary += `▶ Top priority: ${global.task.title} (${global.business.shortName})`;
    }

    return summary;
  }

  // ── Operator overview ───────────────────────────────────────────────────

  getOperatorOverview(): string {
    const businesses = this.state.businesses;
    const health = this.getAllBusinessHealth();
    const incomeBusinesses = businesses.filter((b) => b.revenueStatus === "active");
    const primaryBiz = businesses.filter((b) => b.priority === "primary");
    const staleBiz = health.filter((h) => h.signal === "stale" || h.signal === "critical");

    let out = "";
    out += `BUSINESSES  ${businesses.length} total | ${primaryBiz.length} primary | ${incomeBusinesses.length} generating income\n\n`;

    for (const h of health) {
      const sig = { healthy: "🟢", attention: "🟡", stale: "🟠", critical: "🔴" }[h.signal];
      const bar = this.progressBar(h.health, 10);
      out += `${sig} ${h.shortName.padEnd(16)} [${h.stage.padEnd(6)}] ${bar} ${h.health}%`;
      if (h.signal === "stale") out += `  ⚠️ ${h.daysSinceActivity}d stale`;
      if (h.criticalBlocked > 0) out += `  ${h.criticalBlocked} blocked`;
      out += `\n`;
    }

    if (staleBiz.length > 0) {
      out += `\nNEEDS ATTENTION\n`;
      for (const s of staleBiz) {
        out += `  ⚠️ ${s.businessName} — ${s.daysSinceActivity} days since activity\n`;
      }
    }

    const global = this.getGlobalNextTask();
    if (global) {
      out += `\n▶ Focus: ${global.task.title} (${global.business.shortName}) [${global.task.priority}]`;
    }

    return out;
  }

  private progressBar(percent: number, width: number): string {
    const filled = Math.round((percent / 100) * width);
    return "█".repeat(filled) + "░".repeat(width - filled);
  }

  // ── State access ────────────────────────────────────────────────────────

  getState(): Readonly<ChecklistState> {
    return this.state;
  }

  getCategories(stage?: BusinessStage): TaskCategory[] {
    if (stage) return this.state.categories.filter((c) => c.stage === stage);
    return this.state.categories;
  }
}
