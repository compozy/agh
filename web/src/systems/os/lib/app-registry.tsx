import type { QueryClient } from "@tanstack/react-query";
import {
  BookOpen,
  Boxes,
  Bot,
  Clock3,
  Globe,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Repeat2,
  Settings,
  SquareTerminal,
  Store,
  Waypoints,
  Zap,
  type LucideIcon,
} from "lucide-react";
import { lazy, type ComponentType } from "react";

import type { OsAppId, OsRect } from "./os-types";

export interface OsAppDefinition {
  id: OsAppId;
  title: string;
  icon: LucideIcon;
  /** Route prefixes owned by this app's window subtree. */
  paths: string[];
  /** App-specific opening geometry; enlarged work surfaces override the prototype cascade. */
  defaultRect: OsRect;
  /** Dock strip group, rail toggle, or null for menubar-only settings. */
  dock: { group: 1 | 2 | 3 | 4 } | "rail-toggle" | null;
  badge?: "sessions" | "tasks";
  /** Extracts the multi-instance key from a pathname (session windows). */
  matchInstance?: (pathname: string) => string | null;
  /** Warms the app's index caches; loaders and unfocused mounts share it. */
  preload?: (qc: QueryClient, ctx: { workspaceId: string }) => Promise<void>;
  Controller: ComponentType<{ windowId: string }>;
}

const DashboardWindow = lazy(() =>
  import("../apps/dashboard/dashboard-window").then(m => ({ default: m.DashboardWindow }))
);
const SettingsWindow = lazy(() =>
  import("../apps/settings/settings-window").then(m => ({ default: m.SettingsWindow }))
);
const SessionWindow = lazy(() =>
  import("../apps/session/session-window").then(m => ({ default: m.SessionWindow }))
);
const TasksWindow = lazy(() =>
  import("../apps/tasks/tasks-window").then(m => ({ default: m.TasksWindow }))
);
const AgentsWindow = lazy(() =>
  import("../apps/agents/agents-window").then(m => ({ default: m.AgentsWindow }))
);
const NetworkWindow = lazy(() =>
  import("../apps/network/network-window").then(m => ({ default: m.NetworkWindow }))
);
const SandboxWindow = lazy(() =>
  import("../apps/sandbox/sandbox-window").then(m => ({ default: m.SandboxWindow }))
);
const VaultWindow = lazy(() =>
  import("../apps/vault/vault-window").then(m => ({ default: m.VaultWindow }))
);
const KnowledgeWindow = lazy(() =>
  import("../apps/knowledge/knowledge-window").then(m => ({ default: m.KnowledgeWindow }))
);
const BridgesWindow = lazy(() =>
  import("../apps/bridges/bridges-window").then(m => ({ default: m.BridgesWindow }))
);
const LoopsWindow = lazy(() =>
  import("../apps/loops/loops-window").then(m => ({ default: m.LoopsWindow }))
);
const JobsWindow = lazy(() =>
  import("../apps/jobs/jobs-window").then(m => ({ default: m.JobsWindow }))
);
const TriggersWindow = lazy(() =>
  import("../apps/triggers/triggers-window").then(m => ({ default: m.TriggersWindow }))
);
const MarketplaceWindow = lazy(() =>
  import("../apps/marketplace/marketplace-window").then(m => ({ default: m.MarketplaceWindow }))
);
const SESSION_PATH_PATTERN = /^\/agents\/[^/]+\/sessions\/([^/]+)/;

export function matchSessionInstance(pathname: string): string | null {
  const match = SESSION_PATH_PATTERN.exec(pathname);
  return match ? decodeURIComponent(match[1]) : null;
}

async function preloadDashboard(qc: QueryClient, ctx: { workspaceId: string }): Promise<void> {
  const { preloadHomeWorkspace } = await import("@/routes/_app/-app-preload");
  await preloadHomeWorkspace(qc, ctx.workspaceId);
}

async function preloadSettings(qc: QueryClient): Promise<void> {
  const { preloadSettingsGeneralRoute } = await import("@/routes/_app/-settings-preload");
  await preloadSettingsGeneralRoute(qc);
}

async function preloadTasks(qc: QueryClient): Promise<void> {
  const { preloadTasksRoute } = await import("@/routes/_app/-tasks-preload");
  await preloadTasksRoute(qc, undefined);
}

async function preloadAgents(qc: QueryClient): Promise<void> {
  const { preloadAgentsRoute } = await import("@/routes/_app/-agents-preload");
  await preloadAgentsRoute(qc, { limit: 50 });
}

async function preloadNetwork(qc: QueryClient, ctx: { workspaceId: string }): Promise<void> {
  const { preloadNetworkWindowRoute } = await import("@/routes/_app/-network-preload");
  await preloadNetworkWindowRoute(qc, ctx.workspaceId);
}

async function preloadSandbox(qc: QueryClient): Promise<void> {
  const { preloadSandboxRoute } = await import("@/routes/_app/-settings-preload");
  await preloadSandboxRoute(qc);
}

async function preloadVault(qc: QueryClient): Promise<void> {
  const { preloadVaultRoute } = await import("@/routes/_app/-vault-preload");
  await preloadVaultRoute(qc);
}

async function preloadKnowledge(qc: QueryClient): Promise<void> {
  const { preloadKnowledgeRoute } = await import("@/routes/_app/-knowledge-preload");
  await preloadKnowledgeRoute(qc);
}

async function preloadBridges(qc: QueryClient): Promise<void> {
  const { preloadBridgesRoute } = await import("@/routes/_app/-bridges-preload");
  await preloadBridgesRoute(qc, { scope: "all" });
}

async function preloadLoops(qc: QueryClient): Promise<void> {
  const { preloadLoopsRoute } = await import("@/routes/_app/-loops-preload");
  await preloadLoopsRoute(qc, { limit: 50, sort: "name" });
}

async function preloadJobs(qc: QueryClient): Promise<void> {
  const { preloadAutomationJobsRoute } = await import("@/routes/_app/-automation-preload");
  await preloadAutomationJobsRoute(qc, {});
}

async function preloadTriggers(qc: QueryClient): Promise<void> {
  const { preloadAutomationTriggersRoute } = await import("@/routes/_app/-automation-preload");
  await preloadAutomationTriggersRoute(qc, {});
}

export const OS_APPS: Record<OsAppId, OsAppDefinition> = {
  dashboard: {
    id: "dashboard",
    title: "Dashboard",
    icon: LayoutDashboard,
    paths: ["/"],
    defaultRect: { x: 110, y: 56, w: 680, h: 540 },
    dock: { group: 1 },
    preload: preloadDashboard,
    Controller: DashboardWindow,
  },
  session: {
    id: "session",
    title: "Session",
    icon: SquareTerminal,
    paths: [],
    defaultRect: { x: 470, y: 26, w: 630, h: 570 },
    dock: "rail-toggle",
    badge: "sessions",
    matchInstance: matchSessionInstance,
    Controller: SessionWindow,
  },
  agents: {
    id: "agents",
    title: "Agents",
    icon: Bot,
    paths: ["/agents"],
    defaultRect: { x: 260, y: 118, w: 540, h: 390 },
    dock: { group: 2 },
    preload: preloadAgents,
    Controller: AgentsWindow,
  },
  network: {
    id: "network",
    title: "Network",
    icon: Globe,
    paths: ["/network"],
    defaultRect: { x: 130, y: 64, w: 1200, h: 720 },
    dock: { group: 2 },
    preload: preloadNetwork,
    Controller: NetworkWindow,
  },
  tasks: {
    id: "tasks",
    title: "Tasks",
    icon: ListChecks,
    paths: ["/tasks"],
    defaultRect: { x: 150, y: 60, w: 1160, h: 720 },
    dock: { group: 2 },
    badge: "tasks",
    preload: preloadTasks,
    Controller: TasksWindow,
  },
  loops: {
    id: "loops",
    title: "Loops",
    icon: Repeat2,
    paths: ["/loops", "/loop-runs"],
    defaultRect: { x: 240, y: 100, w: 560, h: 400 },
    dock: { group: 2 },
    preload: preloadLoops,
    Controller: LoopsWindow,
  },
  jobs: {
    id: "jobs",
    title: "Jobs",
    icon: Clock3,
    paths: ["/jobs"],
    defaultRect: { x: 280, y: 118, w: 600, h: 400 },
    dock: { group: 2 },
    preload: preloadJobs,
    Controller: JobsWindow,
  },
  triggers: {
    id: "triggers",
    title: "Triggers",
    icon: Zap,
    paths: ["/triggers"],
    defaultRect: { x: 310, y: 136, w: 620, h: 400 },
    dock: { group: 2 },
    preload: preloadTriggers,
    Controller: TriggersWindow,
  },
  marketplace: {
    id: "marketplace",
    title: "Marketplace",
    icon: Store,
    paths: ["/marketplace"],
    defaultRect: { x: 168, y: 52, w: 720, h: 550 },
    dock: { group: 3 },
    Controller: MarketplaceWindow,
  },
  bridges: {
    id: "bridges",
    title: "Bridges",
    icon: Waypoints,
    paths: ["/bridges"],
    defaultRect: { x: 340, y: 150, w: 560, h: 400 },
    dock: { group: 3 },
    preload: preloadBridges,
    Controller: BridgesWindow,
  },
  knowledge: {
    id: "knowledge",
    title: "Knowledge",
    icon: BookOpen,
    paths: ["/knowledge"],
    defaultRect: { x: 360, y: 164, w: 580, h: 390 },
    dock: { group: 3 },
    preload: preloadKnowledge,
    Controller: KnowledgeWindow,
  },
  sandbox: {
    id: "sandbox",
    title: "Sandbox",
    icon: Boxes,
    paths: ["/sandbox"],
    defaultRect: { x: 300, y: 126, w: 640, h: 450 },
    dock: { group: 4 },
    preload: preloadSandbox,
    Controller: SandboxWindow,
  },
  vault: {
    id: "vault",
    title: "Vault",
    icon: KeyRound,
    paths: ["/vault"],
    defaultRect: { x: 330, y: 128, w: 580, h: 390 },
    dock: { group: 4 },
    preload: preloadVault,
    Controller: VaultWindow,
  },
  settings: {
    id: "settings",
    title: "Settings",
    icon: Settings,
    paths: ["/settings"],
    defaultRect: { x: 180, y: 72, w: 1080, h: 680 },
    dock: null,
    preload: preloadSettings,
    Controller: SettingsWindow,
  },
};

export function getOsApp(id: OsAppId): OsAppDefinition {
  return OS_APPS[id];
}

/** Dock strip order: group 1..4 in registry order (prototype DOCK_ORDER). */
export function dockApps(): OsAppDefinition[][] {
  const groups: OsAppDefinition[][] = [[], [], [], []];
  for (const app of Object.values(OS_APPS)) {
    if (app.dock && app.dock !== "rail-toggle") groups[app.dock.group - 1].push(app);
  }
  return groups.filter(group => group.length > 0);
}

function ownsPath(prefix: string, pathname: string): boolean {
  if (prefix === "/") return pathname === "/";
  return pathname === prefix || pathname.startsWith(`${prefix}/`);
}

/**
 * Maps a pathname to its owning app. Instance-matched apps (session) win over
 * prefix owners so `/agents/<name>/sessions/<id>` resolves to a session window.
 */
export function resolveAppForPath(
  pathname: string
): { app: OsAppDefinition; instanceKey: string | null } | null {
  for (const app of Object.values(OS_APPS)) {
    const instanceKey = app.matchInstance?.(pathname) ?? null;
    if (instanceKey !== null) return { app, instanceKey };
  }
  for (const app of Object.values(OS_APPS)) {
    if (app.paths.some(prefix => ownsPath(prefix, pathname))) {
      return { app, instanceKey: null };
    }
  }
  return null;
}
