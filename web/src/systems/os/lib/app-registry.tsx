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
  /** Hand-tuned cascade from the prototype (os-v2.js APPS). */
  defaultRect: OsRect;
  /** Dock strip group; null = settings (menubar cog) & session (rail-opened). */
  dock: { group: 1 | 2 | 3 | 4 } | null;
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
const PendingAppWindow = lazy(() =>
  import("../apps/pending-app-window").then(m => ({ default: m.PendingAppWindow }))
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
    dock: null,
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
    Controller: PendingAppWindow,
  },
  network: {
    id: "network",
    title: "Network",
    icon: Globe,
    paths: ["/network"],
    defaultRect: { x: 300, y: 98, w: 540, h: 480 },
    dock: { group: 2 },
    Controller: PendingAppWindow,
  },
  tasks: {
    id: "tasks",
    title: "Tasks",
    icon: ListChecks,
    paths: ["/tasks"],
    defaultRect: { x: 200, y: 88, w: 660, h: 480 },
    dock: { group: 2 },
    badge: "tasks",
    Controller: PendingAppWindow,
  },
  loops: {
    id: "loops",
    title: "Loops",
    icon: Repeat2,
    paths: ["/loops", "/loop-runs"],
    defaultRect: { x: 240, y: 100, w: 560, h: 400 },
    dock: { group: 2 },
    Controller: PendingAppWindow,
  },
  jobs: {
    id: "jobs",
    title: "Jobs",
    icon: Clock3,
    paths: ["/jobs"],
    defaultRect: { x: 280, y: 118, w: 600, h: 400 },
    dock: { group: 2 },
    Controller: PendingAppWindow,
  },
  triggers: {
    id: "triggers",
    title: "Triggers",
    icon: Zap,
    paths: ["/triggers"],
    defaultRect: { x: 310, y: 136, w: 620, h: 400 },
    dock: { group: 2 },
    Controller: PendingAppWindow,
  },
  marketplace: {
    id: "marketplace",
    title: "Marketplace",
    icon: Store,
    paths: ["/marketplace"],
    defaultRect: { x: 168, y: 52, w: 720, h: 550 },
    dock: { group: 3 },
    Controller: PendingAppWindow,
  },
  bridges: {
    id: "bridges",
    title: "Bridges",
    icon: Waypoints,
    paths: ["/bridges"],
    defaultRect: { x: 340, y: 150, w: 560, h: 400 },
    dock: { group: 3 },
    Controller: PendingAppWindow,
  },
  knowledge: {
    id: "knowledge",
    title: "Knowledge",
    icon: BookOpen,
    paths: ["/knowledge"],
    defaultRect: { x: 360, y: 164, w: 580, h: 390 },
    dock: { group: 3 },
    Controller: PendingAppWindow,
  },
  sandbox: {
    id: "sandbox",
    title: "Sandbox",
    icon: Boxes,
    paths: ["/sandbox"],
    defaultRect: { x: 300, y: 126, w: 640, h: 450 },
    dock: { group: 4 },
    Controller: PendingAppWindow,
  },
  vault: {
    id: "vault",
    title: "Vault",
    icon: KeyRound,
    paths: ["/vault"],
    defaultRect: { x: 330, y: 128, w: 580, h: 390 },
    dock: { group: 4 },
    Controller: PendingAppWindow,
  },
  settings: {
    id: "settings",
    title: "Settings",
    icon: Settings,
    paths: ["/settings"],
    defaultRect: { x: 280, y: 108, w: 640, h: 450 },
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
    if (app.dock) groups[app.dock.group - 1].push(app);
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
