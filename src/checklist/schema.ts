/**
 * Business Setup Checklist Engine — Data Model
 *
 * Three stages: Idea → Launch → Growth
 * Each stage has categories, each category has tasks.
 * Tasks can be AI-suggested, user-created, or template-seeded.
 * Progress is tracked per-stage and overall.
 *
 * Storage: JSON file in workspace + Letta business_stage block sync
 */

// ============================================================================
// Core Types
// ============================================================================

export type BusinessStage = "idea" | "launch" | "growth";

export type TaskStatus = "pending" | "in_progress" | "completed" | "skipped";

export type TaskSource = "template" | "ai" | "user";

export type TaskPriority = "critical" | "high" | "medium" | "low";

export interface Task {
  id: string;                    // UUID
  title: string;                 // "Create Mission Statement"
  description?: string;          // Detailed explanation
  stage: BusinessStage;          // Which stage this belongs to
  category: string;              // "Business Identity", "Legal", etc.
  status: TaskStatus;
  priority: TaskPriority;
  source: TaskSource;            // Who created it
  aiSuggestion?: string;         // AI tip for completing this task
  dueDate?: string;              // ISO date
  completedAt?: string;          // ISO datetime
  createdAt: string;             // ISO datetime
  updatedAt: string;             // ISO datetime
  dependencies?: string[];       // Task IDs that must complete first
  notes?: string;                // User notes
  order: number;                 // Sort order within category
}

export interface TaskCategory {
  id: string;
  name: string;                  // "Business Identity"
  stage: BusinessStage;
  description?: string;
  order: number;
  icon?: string;                 // Emoji or icon name
}

export interface StageProgress {
  stage: BusinessStage;
  total: number;
  completed: number;
  skipped: number;
  percent: number;               // 0-100
}

export interface ChecklistState {
  version: 1;
  userId: string;                // "verious"
  businessName?: string;
  currentStage: BusinessStage;
  tasks: Task[];
  categories: TaskCategory[];
  createdAt: string;
  updatedAt: string;
}

// ============================================================================
// Daily Stand-Up
// ============================================================================

export interface DailyStandUp {
  date: string;                  // ISO date
  tasks: DailyTask[];
  reflection?: string;           // EOD reflection
  aiInsight?: string;            // AI-generated insight
}

export interface DailyTask {
  taskId: string;                // Reference to checklist task
  title: string;                 // Denormalized for quick display
  completed: boolean;
  notes?: string;
}

// ============================================================================
// Template: Default categories and seed tasks per stage
// ============================================================================

export const DEFAULT_CATEGORIES: TaskCategory[] = [
  // ── Idea Stage ──
  { id: "idea-identity",   name: "Business Identity",     stage: "idea",   order: 1, icon: "💡" },
  { id: "idea-research",   name: "Market Research",       stage: "idea",   order: 2, icon: "🔍" },
  { id: "idea-planning",   name: "Business Planning",     stage: "idea",   order: 3, icon: "📋" },
  { id: "idea-finance",    name: "Financial Planning",    stage: "idea",   order: 4, icon: "💰" },
  { id: "idea-legal",      name: "Legal Foundation",      stage: "idea",   order: 5, icon: "⚖️" },

  // ── Launch Stage ──
  { id: "launch-brand",    name: "Brand & Marketing",     stage: "launch", order: 1, icon: "🎨" },
  { id: "launch-digital",  name: "Digital Presence",      stage: "launch", order: 2, icon: "🌐" },
  { id: "launch-ops",      name: "Operations Setup",      stage: "launch", order: 3, icon: "⚙️" },
  { id: "launch-product",  name: "Product/Service Ready", stage: "launch", order: 4, icon: "📦" },
  { id: "launch-sales",    name: "Sales Pipeline",        stage: "launch", order: 5, icon: "🤝" },

  // ── Growth Stage ──
  { id: "growth-scale",    name: "Scaling Operations",    stage: "growth", order: 1, icon: "📈" },
  { id: "growth-team",     name: "Team Building",         stage: "growth", order: 2, icon: "👥" },
  { id: "growth-revenue",  name: "Revenue Optimization",  stage: "growth", order: 3, icon: "💵" },
  { id: "growth-systems",  name: "Systems & Automation",  stage: "growth", order: 4, icon: "🤖" },
  { id: "growth-network",  name: "Network & Partnerships",stage: "growth", order: 5, icon: "🌍" },
];

export const SEED_TASKS: Omit<Task, "id" | "createdAt" | "updatedAt">[] = [
  // ── Idea Stage: Business Identity ──
  { title: "Create Mission Statement",           stage: "idea", category: "idea-identity",  status: "pending", priority: "critical", source: "template", order: 1,
    aiSuggestion: "A strong mission statement answers: What do you do? Who do you serve? Why does it matter?" },
  { title: "Define Vision Statement",            stage: "idea", category: "idea-identity",  status: "pending", priority: "critical", source: "template", order: 2,
    aiSuggestion: "Where do you see this business in 5 years? Paint the future you're building toward." },
  { title: "Identify Core Values",               stage: "idea", category: "idea-identity",  status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Choose Business Name",               stage: "idea", category: "idea-identity",  status: "pending", priority: "critical", source: "template", order: 4 },
  { title: "Write Elevator Pitch",               stage: "idea", category: "idea-identity",  status: "pending", priority: "high",     source: "template", order: 5 },

  // ── Idea Stage: Market Research ──
  { title: "Identify Target Customer",           stage: "idea", category: "idea-research",  status: "pending", priority: "critical", source: "template", order: 1,
    aiSuggestion: "Be specific: age, income, location, pain points, where they hang out online." },
  { title: "Analyze Competitors",                stage: "idea", category: "idea-research",  status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Validate Problem-Solution Fit",      stage: "idea", category: "idea-research",  status: "pending", priority: "critical", source: "template", order: 3 },
  { title: "Estimate Market Size (TAM/SAM/SOM)", stage: "idea", category: "idea-research",  status: "pending", priority: "medium",   source: "template", order: 4 },
  { title: "Talk to 10 Potential Customers",      stage: "idea", category: "idea-research",  status: "pending", priority: "critical", source: "template", order: 5,
    aiSuggestion: "Nothing replaces real conversations. Ask open-ended questions, listen more than talk." },

  // ── Idea Stage: Business Planning ──
  { title: "Write One-Page Business Plan",       stage: "idea", category: "idea-planning",  status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Define Revenue Model",               stage: "idea", category: "idea-planning",  status: "pending", priority: "critical", source: "template", order: 2 },
  { title: "Set 90-Day Goals",                   stage: "idea", category: "idea-planning",  status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Identify Key Milestones",            stage: "idea", category: "idea-planning",  status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Idea Stage: Financial Planning ──
  { title: "Calculate Startup Costs",            stage: "idea", category: "idea-finance",   status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Set Pricing Strategy",               stage: "idea", category: "idea-finance",   status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Open Business Bank Account",         stage: "idea", category: "idea-finance",   status: "pending", priority: "medium",   source: "template", order: 3 },
  { title: "Create Budget Forecast (6 months)",  stage: "idea", category: "idea-finance",   status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Idea Stage: Legal Foundation ──
  { title: "Choose Business Structure (LLC/Corp/Sole Prop)", stage: "idea", category: "idea-legal", status: "pending", priority: "high", source: "template", order: 1 },
  { title: "Register Business Name",             stage: "idea", category: "idea-legal",     status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Get EIN (Tax ID)",                   stage: "idea", category: "idea-legal",     status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Research Required Licenses/Permits", stage: "idea", category: "idea-legal",     status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Launch Stage: Brand & Marketing ──
  { title: "Design Logo",                        stage: "launch", category: "launch-brand",   status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Create Brand Style Guide",           stage: "launch", category: "launch-brand",   status: "pending", priority: "medium",   source: "template", order: 2 },
  { title: "Write Brand Story",                  stage: "launch", category: "launch-brand",   status: "pending", priority: "medium",   source: "template", order: 3 },
  { title: "Plan Launch Marketing Campaign",     stage: "launch", category: "launch-brand",   status: "pending", priority: "high",     source: "template", order: 4 },
  { title: "Set Up Social Media Accounts",       stage: "launch", category: "launch-brand",   status: "pending", priority: "high",     source: "template", order: 5 },

  // ── Launch Stage: Digital Presence ──
  { title: "Register Domain Name",               stage: "launch", category: "launch-digital", status: "pending", priority: "critical", source: "template", order: 1 },
  { title: "Build Website (MVP)",                stage: "launch", category: "launch-digital", status: "pending", priority: "critical", source: "template", order: 2 },
  { title: "Set Up Business Email",              stage: "launch", category: "launch-digital", status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Set Up Google Business Profile",     stage: "launch", category: "launch-digital", status: "pending", priority: "high",     source: "template", order: 4 },
  { title: "Implement Basic SEO",               stage: "launch", category: "launch-digital", status: "pending", priority: "medium",   source: "template", order: 5 },

  // ── Launch Stage: Operations Setup ──
  { title: "Set Up Accounting System",           stage: "launch", category: "launch-ops",     status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Create Standard Operating Procedures", stage: "launch", category: "launch-ops",   status: "pending", priority: "medium",   source: "template", order: 2 },
  { title: "Set Up Customer Communication Tools", stage: "launch", category: "launch-ops",    status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Choose Payment Processing",          stage: "launch", category: "launch-ops",     status: "pending", priority: "critical", source: "template", order: 4 },

  // ── Launch Stage: Product/Service Ready ──
  { title: "Finalize Product/Service Offering",  stage: "launch", category: "launch-product", status: "pending", priority: "critical", source: "template", order: 1 },
  { title: "Create Sales Materials",             stage: "launch", category: "launch-product", status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Set Up Fulfillment Process",         stage: "launch", category: "launch-product", status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Get Beta Customers (3-5)",           stage: "launch", category: "launch-product", status: "pending", priority: "critical", source: "template", order: 4,
    aiSuggestion: "Beta customers validate your offering AND become your first testimonials." },

  // ── Launch Stage: Sales Pipeline ──
  { title: "Define Sales Process",               stage: "launch", category: "launch-sales",   status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Create Lead Generation Strategy",    stage: "launch", category: "launch-sales",   status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Set Up CRM",                         stage: "launch", category: "launch-sales",   status: "pending", priority: "medium",   source: "template", order: 3 },
  { title: "Make First 10 Sales",                stage: "launch", category: "launch-sales",   status: "pending", priority: "critical", source: "template", order: 4 },

  // ── Growth Stage: Scaling Operations ──
  { title: "Document All Key Processes",         stage: "growth", category: "growth-scale",   status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Identify Bottlenecks",               stage: "growth", category: "growth-scale",   status: "pending", priority: "critical", source: "template", order: 2 },
  { title: "Set Up Quality Metrics",             stage: "growth", category: "growth-scale",   status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Plan for 10x Volume",                stage: "growth", category: "growth-scale",   status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Growth Stage: Team Building ──
  { title: "Define First Hire Role",             stage: "growth", category: "growth-team",    status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Create Hiring Process",              stage: "growth", category: "growth-team",    status: "pending", priority: "medium",   source: "template", order: 2 },
  { title: "Build Company Culture Doc",          stage: "growth", category: "growth-team",    status: "pending", priority: "medium",   source: "template", order: 3 },
  { title: "Set Up Onboarding Playbook",         stage: "growth", category: "growth-team",    status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Growth Stage: Revenue Optimization ──
  { title: "Analyze Unit Economics",             stage: "growth", category: "growth-revenue", status: "pending", priority: "critical", source: "template", order: 1 },
  { title: "Identify Upsell/Cross-sell Paths",   stage: "growth", category: "growth-revenue", status: "pending", priority: "high",     source: "template", order: 2 },
  { title: "Build Retention Strategy",           stage: "growth", category: "growth-revenue", status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Set Revenue Targets (Monthly)",      stage: "growth", category: "growth-revenue", status: "pending", priority: "high",     source: "template", order: 4 },

  // ── Growth Stage: Systems & Automation ──
  { title: "Automate Repetitive Tasks",          stage: "growth", category: "growth-systems", status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Set Up Analytics Dashboard",         stage: "growth", category: "growth-systems", status: "pending", priority: "medium",   source: "template", order: 2 },
  { title: "Implement Customer Feedback Loop",   stage: "growth", category: "growth-systems", status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Plan Tech Stack for Scale",          stage: "growth", category: "growth-systems", status: "pending", priority: "medium",   source: "template", order: 4 },

  // ── Growth Stage: Network & Partnerships ──
  { title: "Identify Strategic Partners",        stage: "growth", category: "growth-network", status: "pending", priority: "high",     source: "template", order: 1 },
  { title: "Join Industry Communities",          stage: "growth", category: "growth-network", status: "pending", priority: "medium",   source: "template", order: 2 },
  { title: "Build Referral Program",             stage: "growth", category: "growth-network", status: "pending", priority: "high",     source: "template", order: 3 },
  { title: "Attend/Host 1 Industry Event",       stage: "growth", category: "growth-network", status: "pending", priority: "medium",   source: "template", order: 4 },
];
