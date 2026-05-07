/* Realistic AGH-flavored mock data shared across the 3 variations.
 * Shape mirrors the OpenAPI types behind web/src/systems/tasks/ and
 * web/src/systems/automation/. Status coverage:
 *   tasks  → running, in_progress, ready, failed, blocked, pending,
 *            draft, completed, canceled
 *   jobs   → enabled/disabled × dynamic/manual × workspace/global
 *
 * STATUS_TONE keeps a single semantic per color: orange means LIVE
 * (running). In-progress moves to info (purple). Blocked stays warning.
 */

const TASKS = [
  {
    id: "tsk-7f3a2b",
    title: "Investigate auth middleware leak in session-bridge",
    status: "running",
    priority: "high",
    owner: { kind: "agent", label: "claude-opus-4-7" },
    activeRun: { attempt: 2, maxAttempts: 3, error: null },
    childCount: 1,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "2m ago",
    timestampIso: "2026-05-06T17:34:11Z",
  },
  {
    id: "tsk-9c4d18",
    title: "Backfill task_runs.queue_position for legacy rows",
    status: "in_progress",
    priority: "medium",
    owner: { kind: "human", label: "pedro" },
    activeRun: { attempt: 1, maxAttempts: 3, error: null },
    childCount: 4,
    dependencyCount: 2,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "11m ago",
    timestampIso: "2026-05-06T17:25:02Z",
  },
  {
    id: "tsk-4d7e23",
    title: "Resolve typescript errors in @agh/extension-sdk client.ts",
    status: "failed",
    priority: "high",
    owner: { kind: "agent", label: "codex-gpt-5.4" },
    activeRun: {
      attempt: 3,
      maxAttempts: 3,
      error: "tsc(2345): Argument of type 'Bridge | null' is not assignable to parameter of type 'Bridge'.",
    },
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: "tsk-9c4d18",
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "23m ago",
    timestampIso: "2026-05-06T17:13:48Z",
  },
  {
    id: "tsk-6b1c44",
    title: "Land RFC-002 protocol receipt schema",
    status: "blocked",
    priority: "high",
    owner: { kind: "human", label: "pedro" },
    activeRun: null,
    childCount: 3,
    dependencyCount: 1,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: true,
    blockedReason: "waiting on review of RFC-001",
    timestamp: "1h ago",
    timestampIso: "2026-05-06T16:32:18Z",
  },
  {
    id: "tsk-3e8b07",
    title: "Implement gallery view for capability marketplace",
    status: "ready",
    priority: "medium",
    owner: { kind: "agent", label: "hermes-sonnet" },
    activeRun: null,
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: "pending",
    isDraft: false,
    isBlocked: false,
    timestamp: "1h ago",
    timestampIso: "2026-05-06T16:14:55Z",
  },
  {
    id: "tsk-8e2a76",
    title: "Define agent execution profile defaults for sandbox tier",
    status: "draft",
    priority: "low",
    owner: { kind: "human", label: "pedro" },
    activeRun: null,
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: null,
    isDraft: true,
    isBlocked: false,
    timestamp: "3h ago",
    timestampIso: "2026-05-06T14:31:09Z",
  },
  {
    id: "tsk-2a5f91",
    title: "Migrate workspace bootstrap fixtures to new manifest format",
    status: "pending",
    priority: "low",
    owner: { kind: "agent", label: "openclaw" },
    activeRun: null,
    childCount: 0,
    dependencyCount: 1,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "4h ago",
    timestampIso: "2026-05-06T13:48:22Z",
  },
  {
    id: "tsk-0a3e58",
    title: "Wire daemon /healthz to status footer",
    status: "in_progress",
    priority: "medium",
    owner: { kind: "agent", label: "claude-opus-4-7" },
    activeRun: { attempt: 1, maxAttempts: 2, error: null },
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "5h ago",
    timestampIso: "2026-05-06T12:22:40Z",
  },
  {
    id: "tsk-5f9b32",
    title: "Patch SQLite migration registry race on first boot",
    status: "completed",
    priority: "high",
    owner: { kind: "agent", label: "claude-opus-4-7" },
    activeRun: null,
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "yesterday",
    timestampIso: "2026-05-05T18:11:02Z",
  },
  {
    id: "tsk-1c6d05",
    title: "Refresh peer card avatar resolver",
    status: "canceled",
    priority: "low",
    owner: { kind: "human", label: "pedro" },
    activeRun: null,
    childCount: 0,
    dependencyCount: 0,
    parentTaskId: null,
    approvalState: null,
    isDraft: false,
    isBlocked: false,
    timestamp: "2d ago",
    timestampIso: "2026-05-04T09:55:38Z",
  },
];

const JOBS = [
  {
    id: "job-7d2b",
    name: "Sync session memory hourly",
    enabled: true,
    source: "dynamic",
    scope: "workspace",
    schedule: "every hour, on the hour",
    nextRun: "in 23m",
    lastRun: "37m ago",
    avgDuration: "2.1s",
    lastFailure: null,
  },
  {
    id: "job-3a8e",
    name: "Refresh capability registry",
    enabled: true,
    source: "dynamic",
    scope: "global",
    schedule: "every 4 hours",
    nextRun: "in 1h 47m",
    lastRun: "2h 13m ago",
    avgDuration: "4.6s",
    lastFailure: null,
  },
  {
    id: "job-9c1f",
    name: "Daily knowledge consolidation",
    enabled: false,
    source: "manual",
    scope: "workspace",
    schedule: "daily at 03:00 local",
    nextRun: "paused",
    lastRun: "yesterday",
    avgDuration: "12.4s",
    lastFailure: "rate limit on llm provider",
  },
  {
    id: "job-5e4d",
    name: "Reconcile peer cards from registry",
    enabled: true,
    source: "dynamic",
    scope: "workspace",
    schedule: "every 30 minutes",
    nextRun: "in 8m",
    lastRun: "22m ago",
    avgDuration: "0.9s",
    lastFailure: null,
  },
  {
    id: "job-2b6a",
    name: "Compact bridge notification cursors",
    enabled: true,
    source: "dynamic",
    scope: "global",
    schedule: "daily at 04:30 UTC",
    nextRun: "in 6h 12m",
    lastRun: "17h 48m ago",
    avgDuration: "3.3s",
    lastFailure: null,
  },
  {
    id: "job-0f8c",
    name: "Vacuum events.db (scheduled)",
    enabled: false,
    source: "manual",
    scope: "global",
    schedule: "weekly on Sunday 02:00 UTC",
    nextRun: "paused",
    lastRun: "5d ago",
    avgDuration: "44s",
    lastFailure: null,
  },
  {
    id: "job-1d3b",
    name: "Garbage-collect orphaned task runs",
    enabled: true,
    source: "dynamic",
    scope: "workspace",
    schedule: "every 6 hours",
    nextRun: "in 3h 04m",
    lastRun: "2h 56m ago",
    avgDuration: "1.7s",
    lastFailure: null,
  },
];

const STATUS_TONE = {
  running: "accent",
  in_progress: "info",
  ready: "neutral",
  pending: "neutral",
  draft: "neutral",
  failed: "danger",
  blocked: "warning",
  completed: "success",
  canceled: "neutral",
};

const STATUS_LABEL = {
  running: "Running",
  in_progress: "In progress",
  ready: "Ready",
  pending: "Pending",
  draft: "Draft",
  failed: "Failed",
  blocked: "Blocked",
  completed: "Completed",
  canceled: "Canceled",
};

const PRIORITY_LABEL = {
  high: "High",
  medium: "Medium",
  low: "Low",
  none: "No priority",
};

const TONE_COLOR = {
  accent: "var(--color-accent)",
  success: "var(--color-success)",
  danger: "var(--color-danger)",
  warning: "var(--color-warning)",
  info: "var(--color-info)",
  neutral: "var(--color-text-tertiary)",
};

/* ── Per-task activity feed (most recent first) ──────────────────────── */

const SYS = { kind: "system", label: "agh-runtime" };

const ACTIVITY_BY_TASK = {
  "tsk-7f3a2b": [
    {
      time: "2m ago",
      author: { kind: "agent", label: "claude-opus-4-7" },
      msg: "Started attempt 2",
      detail: "claim acquired · capability resolve.session-bridge.leak",
    },
    {
      time: "9m ago",
      author: SYS,
      msg: "Attempt 1 timed out",
      detail: "no progress for 600s · automatic retry queued",
    },
    {
      time: "14m ago",
      author: { kind: "human", label: "pedro" },
      msg: "Increased priority to High",
      detail: "Operator reclassified from Medium",
    },
    {
      time: "26m ago",
      author: { kind: "human", label: "pedro" },
      msg: "Created task",
      detail: "linked from session sess-aa7c · trigger origin manual",
    },
  ],
  "tsk-9c4d18": [
    {
      time: "11m ago",
      author: { kind: "human", label: "pedro" },
      msg: "Spawned 4 children",
      detail: "fan-out across workspaces agh-runtime · compozy · scratch · _root",
    },
    {
      time: "32m ago",
      author: SYS,
      msg: "Plan accepted",
      detail: "review gate satisfied · proceed-with-caveats",
    },
    {
      time: "44m ago",
      author: { kind: "agent", label: "claude-opus-4-7" },
      msg: "Submitted plan for review",
      detail: "estimated 4 child runs · 2 dependencies",
    },
  ],
  "tsk-4d7e23": [
    {
      time: "23m ago",
      author: SYS,
      msg: "Attempt 3 failed",
      detail: "tsc exit 1 · max attempts reached · awaiting operator",
    },
    {
      time: "27m ago",
      author: { kind: "agent", label: "codex-gpt-5.4" },
      msg: "Started attempt 3",
      detail: "applied last-good fixture, retry under reduced scope",
    },
    {
      time: "38m ago",
      author: SYS,
      msg: "Attempt 2 failed",
      detail: "same diagnostic · no patch applied",
    },
  ],
  "tsk-6b1c44": [
    {
      time: "1h ago",
      author: { kind: "human", label: "pedro" },
      msg: "Marked blocked",
      detail: "reason: waiting on review of RFC-001",
    },
    {
      time: "2h ago",
      author: { kind: "agent", label: "claude-opus-4-7" },
      msg: "Spawned 3 children",
      detail: "decomposed into receipt schema · validator · serializer",
    },
  ],
  "tsk-3e8b07": [
    {
      time: "1h ago",
      author: { kind: "agent", label: "hermes-sonnet" },
      msg: "Requested approval",
      detail: "policy require-human · scope marketplace.gallery",
    },
    {
      time: "1h 12m ago",
      author: SYS,
      msg: "Lease acquired",
      detail: "queue head, no prior pending claimants",
    },
  ],
};

const DEFAULT_ACTIVITY = (task) => [
  {
    time: task.timestamp,
    author: SYS,
    msg: "Task " + STATUS_LABEL[task.status].toLowerCase(),
    detail: task.id + " · workspace agh-runtime",
  },
  {
    time: "earlier",
    author: task.owner,
    msg: "Created task",
    detail: "trigger origin = manual",
  },
];

function getActivityFor(task) {
  return ACTIVITY_BY_TASK[task.id] ?? DEFAULT_ACTIVITY(task);
}

/* ── Sub-issues (children) for tasks with childCount > 0 ─────────────── */

const SUBISSUES_BY_TASK = {
  "tsk-9c4d18": [
    { id: "tsk-4d7e23", title: "Resolve typescript errors in @agh/extension-sdk client.ts", status: "failed", assignee: { kind: "agent", label: "codex-gpt-5.4" } },
    { id: "tsk-2c11ee", title: "Backfill queue_position rows for compozy workspace", status: "completed", assignee: { kind: "agent", label: "claude-opus-4-7" } },
    { id: "tsk-71b9a5", title: "Backfill queue_position rows for scratch workspace", status: "running", assignee: { kind: "agent", label: "claude-opus-4-7" } },
    { id: "tsk-bb40d2", title: "Backfill queue_position rows for _root workspace", status: "ready", assignee: { kind: "agent", label: "openclaw" } },
  ],
  "tsk-7f3a2b": [
    { id: "tsk-cf02a1", title: "Capture pprof goroutine dump from session-bridge", status: "completed", assignee: { kind: "agent", label: "claude-opus-4-7" } },
  ],
  "tsk-6b1c44": [
    { id: "tsk-aa31cd", title: "Draft receipt schema fields", status: "in_progress", assignee: { kind: "human", label: "pedro" } },
    { id: "tsk-7d99fe", title: "Write validator with negative cases", status: "ready", assignee: { kind: "agent", label: "openclaw" } },
    { id: "tsk-bf2206", title: "Wire serializer into bridge SDK", status: "draft", assignee: { kind: "agent", label: "hermes-sonnet" } },
  ],
};

function getSubissuesFor(task) {
  return SUBISSUES_BY_TASK[task.id] ?? [];
}

/* ── Avatar primitive helpers ────────────────────────────────────────── */

/* Deterministic warm-palette tints. Agents draw from the accent/info family
 * to read as "machines". Humans draw from a calmer warm palette. The seed
 * is the owner.label so identity is stable across renders. */
const AVATAR_PALETTE_AGENT = [
  { bg: "rgba(232,87,42,0.16)",  fg: "#F6874F" },
  { bg: "rgba(191,90,242,0.14)", fg: "#D7A3FB" },
  { bg: "rgba(91,166,255,0.14)", fg: "#9EC8FF" },
  { bg: "rgba(79,209,197,0.14)", fg: "#8FE2D6" },
];

const AVATAR_PALETTE_HUMAN = [
  { bg: "rgba(255,214,10,0.14)", fg: "#FFE372" },
  { bg: "rgba(48,209,88,0.14)",  fg: "#7CE0A0" },
  { bg: "rgba(232,135,79,0.18)", fg: "#FFB58E" },
  { bg: "rgba(152,152,157,0.18)", fg: "#D6D6D9" },
];

function avatarSeed(label) {
  let h = 0;
  for (let i = 0; i < label.length; i++) h = (h * 31 + label.charCodeAt(i)) | 0;
  return Math.abs(h);
}

function avatarColors(owner) {
  const palette = owner.kind === "agent" ? AVATAR_PALETTE_AGENT : AVATAR_PALETTE_HUMAN;
  return palette[avatarSeed(owner.label) % palette.length];
}

function avatarInitials(label) {
  const parts = label.split(/[-_\s.]+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[1][0]).toUpperCase();
}

/* ── Sidebar nav ─────────────────────────────────────────────────────── */

const NAV_PRIMARY = [
  { id: "tasks", label: "Tasks", icon: "ListChecks", count: 12, kbd: "T" },
  { id: "jobs", label: "Jobs", icon: "Clock", count: 7, kbd: "J" },
  { id: "sessions", label: "Sessions", icon: "Activity", count: 3, kbd: "S" },
  { id: "network", label: "Network", icon: "Network", count: null, kbd: "N" },
];

const NAV_SECONDARY = [
  { id: "automation", label: "Automation", icon: "Sparkles", count: null },
  { id: "bridges", label: "Bridges", icon: "Plug", count: 4 },
  { id: "knowledge", label: "Knowledge", icon: "Book", count: null },
  { id: "skills", label: "Skills", icon: "Boxes", count: 18 },
];

const NAV_FOOTER = [{ id: "settings", label: "Settings", icon: "Settings", count: null }];

const WORKSPACES = [
  { id: "agh-runtime", label: "AGH Runtime", color: "var(--color-accent)", initial: "A", active: true },
  { id: "compozy", label: "Compozy", color: "var(--color-surface-elevated)", initial: "C" },
  { id: "scratch", label: "Scratch", color: "var(--color-surface-elevated)", initial: "S" },
];

/* ── Aggregate counts surfaced to mastheads ──────────────────────────── */

const TASK_SUMMARY = {
  total: TASKS.length,
  open: TASKS.filter((t) => !["completed", "canceled"].includes(t.status)).length,
  blocked: TASKS.filter((t) => t.isBlocked).length,
  failed: TASKS.filter((t) => t.status === "failed").length,
  running: TASKS.filter((t) => t.status === "running").length,
  workspaces: 3,
};

const JOB_SUMMARY = {
  total: JOBS.length,
  active: JOBS.filter((j) => j.enabled).length,
  paused: JOBS.filter((j) => !j.enabled).length,
};

window.AGH_DATA = {
  TASKS,
  JOBS,
  STATUS_TONE,
  STATUS_LABEL,
  PRIORITY_LABEL,
  TONE_COLOR,
  NAV_PRIMARY,
  NAV_SECONDARY,
  NAV_FOOTER,
  WORKSPACES,
  TASK_SUMMARY,
  JOB_SUMMARY,
  getActivityFor,
  getSubissuesFor,
  avatarColors,
  avatarInitials,
};
