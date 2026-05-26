import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  Book,
  Boxes,
  ChevronsUpDown,
  Clock3,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Network,
  Plus,
  Settings,
  Waypoints,
  Wrench,
  Zap,
  type LucideIcon,
} from "lucide-react";

import { Button, Logo, Sidebar, SidebarSectionLabel, cn } from "@agh/ui";

import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import { AgentCategoryTree, type AgentPayload } from "@/systems/agent";
import { isSessionRunning, type SessionPayload } from "@/systems/session";
import type { WorkspacePayload } from "@/systems/workspace";

import { RuntimeConnectionIndicator } from "./connection-indicator";
import { RestartDaemonButton } from "./restart-daemon-button";

interface RailSlotProps {
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspaceId: string | null;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
}

function RailSlot({
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  onAddWorkspace,
}: RailSlotProps) {
  return (
    <div data-testid="icon-rail" className="flex flex-1 flex-col items-center gap-1.5">
      <Link
        to="/"
        aria-label="Go to dashboard"
        data-testid="app-logo"
        className="mb-1 inline-flex size-7 items-center justify-center rounded-md transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-canvas"
      >
        <Logo variant="symbol" decorative className="size-7" />
      </Link>
      {workspaces?.map(workspace => {
        const isActive = workspace.id === activeWorkspaceId;
        const letter = workspace.name.charAt(0).toUpperCase() || "·";
        return (
          <button
            key={workspace.id}
            type="button"
            onClick={() => onSelectWorkspace(workspace.id)}
            data-testid={`workspace-avatar-${workspace.id}`}
            data-active={isActive}
            title={workspace.name}
            aria-label={`Workspace: ${workspace.name}`}
            aria-pressed={isActive}
            className={cn(
              "inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-elevated font-mono text-eyebrow font-medium text-muted transition-colors hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
              isActive && "border-accent text-fg"
            )}
          >
            {letter}
          </button>
        );
      })}
      <button
        type="button"
        onClick={onAddWorkspace}
        data-testid="add-workspace-btn"
        aria-label="Add workspace"
        className="inline-flex size-7 items-center justify-center rounded-md border border-dashed border-line text-subtle transition-colors hover:border-accent hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
      >
        <Plus aria-hidden="true" className="size-3" />
      </button>
    </div>
  );
}

interface NavItemProps {
  to: string;
  icon: LucideIcon;
  label: string;
  fuzzy?: boolean;
}

function NavItem({ to, icon: Icon, label, fuzzy }: NavItemProps) {
  const matchRoute = useMatchRoute();
  const isActive = Boolean(matchRoute({ to, fuzzy }));
  const testKey = label.toLowerCase();

  return (
    <Link
      to={to}
      data-testid={`nav-${testKey}`}
      data-active={isActive}
      className={cn(NAV_ROW_CLASS, isActive && ACTIVE_NAV_ROW_CLASS)}
    >
      {isActive && (
        <span
          aria-hidden="true"
          data-testid={`nav-active-${testKey}`}
          className={ACTIVE_NAV_INDICATOR_CLASS}
        />
      )}
      <Icon aria-hidden="true" className="size-3 shrink-0" />
      <span className="truncate">{label}</span>
    </Link>
  );
}

const DASHBOARD_NAV_ITEM: NavItemProps = {
  to: "/",
  icon: LayoutDashboard,
  label: "Dashboard",
};

const OPERATE_NAV_ITEMS: NavItemProps[] = [
  { to: "/network", icon: Network, label: "Network" },
  { to: "/tasks", icon: ListChecks, label: "Tasks", fuzzy: true },
  { to: "/jobs", icon: Clock3, label: "Jobs" },
  { to: "/triggers", icon: Zap, label: "Triggers" },
];

const CATALOG_NAV_ITEMS: NavItemProps[] = [
  { to: "/knowledge", icon: Book, label: "Knowledge" },
  { to: "/skills", icon: Wrench, label: "Skills" },
  { to: "/bridges", icon: Waypoints, label: "Bridges" },
];

const SYSTEM_NAV_ITEMS: NavItemProps[] = [
  { to: "/sandbox", icon: Boxes, label: "Sandbox" },
  { to: "/vault", icon: KeyRound, label: "Vault" },
  { to: "/settings", icon: Settings, label: "Settings", fuzzy: true },
];

interface AgentsCount {
  live: number;
  total: number;
}

function computeAgentsCount(
  agents: AgentPayload[] | undefined,
  sessions: SessionPayload[] | undefined
): AgentsCount {
  const total = agents?.length ?? 0;
  if (total === 0) return { live: 0, total: 0 };
  const liveNames = new Set<string>();
  for (const session of sessions ?? []) {
    if (isSessionRunning(session)) {
      liveNames.add(session.agent_name);
    }
  }
  let live = 0;
  for (const agent of agents ?? []) {
    if (liveNames.has(agent.name)) live += 1;
  }
  return { live, total };
}

function countActiveSessions(sessions: SessionPayload[] | undefined): number {
  if (!sessions) return 0;
  let count = 0;
  for (const session of sessions) {
    if (isSessionRunning(session)) count += 1;
  }
  return count;
}

interface NavSlotProps {
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onAddAgent: () => void;
}

function NavSlot({ agents, agentsLoading, agentsError, sessions, onAddAgent }: NavSlotProps) {
  const agentsCount = computeAgentsCount(agents, sessions);
  return (
    <div data-testid="sidebar-nav" className="flex flex-col gap-1 px-2 py-3">
      <NavItem
        to={DASHBOARD_NAV_ITEM.to}
        icon={DASHBOARD_NAV_ITEM.icon}
        label={DASHBOARD_NAV_ITEM.label}
      />

      <SectionLabel className="mt-4">
        <span>Agents</span>
        <span className="ml-auto flex items-center gap-1.5">
          {agentsCount.total > 0 ? (
            <span className="tabular-nums text-subtle" data-testid="agents-live-count">
              {agentsCount.live}/{agentsCount.total} live
            </span>
          ) : null}
          <Button
            aria-label="Create agent"
            className="-mr-1 text-muted hover:text-fg"
            data-testid="sidebar-create-agent"
            onClick={onAddAgent}
            size="icon-xs"
            type="button"
            variant="ghost"
          >
            <Plus aria-hidden="true" className="size-3" />
          </Button>
        </span>
      </SectionLabel>
      <AgentCategoryTree
        agents={agents}
        agentsLoading={agentsLoading}
        agentsError={agentsError}
        sessions={sessions}
      />

      <SectionLabel className="mt-4">Operate</SectionLabel>
      <NavGroup items={OPERATE_NAV_ITEMS} />

      <SectionLabel className="mt-4">Catalog</SectionLabel>
      <NavGroup items={CATALOG_NAV_ITEMS} />

      <SectionLabel className="mt-4">System</SectionLabel>
      <NavGroup items={SYSTEM_NAV_ITEMS} />
    </div>
  );
}

function NavGroup({ items }: { items: NavItemProps[] }) {
  return (
    <div className="flex flex-col gap-0.5">
      {items.map(item => (
        <NavItem
          key={item.to}
          to={item.to}
          icon={item.icon}
          label={item.label}
          fuzzy={item.fuzzy}
        />
      ))}
    </div>
  );
}

function SectionLabel({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <SidebarSectionLabel data-testid="sidebar-section-label" className={cn("pt-2 pb-1", className)}>
      {children}
    </SidebarSectionLabel>
  );
}

interface WorkspaceSwitcherProps {
  workspace: WorkspacePayload | undefined;
}

function WorkspaceSwitcher({ workspace }: WorkspaceSwitcherProps) {
  const label = workspace?.name ?? "No workspace";
  const initial = label.charAt(0).toUpperCase() || "·";

  return (
    <div data-testid="workspace-switcher" className="flex h-12 w-full items-center gap-2.5 px-2">
      <span
        aria-hidden="true"
        data-testid="workspace-switcher-avatar"
        className="inline-flex size-button-icon-xs shrink-0 items-center justify-center rounded-sm bg-elevated font-mono text-eyebrow font-medium tracking-mono text-fg"
      >
        {initial}
      </span>
      <span
        data-testid="workspace-switcher-name"
        className="min-w-0 flex-1 truncate text-small-body font-medium tracking-tight text-fg"
      >
        {label}
      </span>
      <ChevronsUpDown
        aria-hidden="true"
        data-testid="workspace-switcher-chevron"
        className="size-3 shrink-0 text-subtle"
      />
    </div>
  );
}

interface FooterSlotProps {
  activeSessionCount: number;
}

function FooterSlot({ activeSessionCount }: FooterSlotProps) {
  return (
    <div data-testid="sidebar-footer" className="flex items-center gap-2 px-2">
      <RuntimeConnectionIndicator />
      <RestartDaemonButton activeSessionCount={activeSessionCount} />
    </div>
  );
}

export interface AppSidebarProps {
  collapsed: boolean;
  onCollapseChange: (next: boolean) => void;
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspaceId: string | null;
  activeWorkspace: WorkspacePayload | undefined;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
  onAddAgent: () => void;
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  className?: string;
}

function AppSidebar({
  collapsed,
  onCollapseChange,
  workspaces,
  activeWorkspaceId,
  activeWorkspace,
  onSelectWorkspace,
  onAddWorkspace,
  onAddAgent,
  agents,
  agentsLoading,
  agentsError,
  sessions,
  className,
}: AppSidebarProps) {
  const activeSessionCount = countActiveSessions(sessions);
  return (
    <Sidebar
      data-testid="app-sidebar"
      className={className}
      collapsed={collapsed}
      onCollapse={onCollapseChange}
      rail={
        <RailSlot
          workspaces={workspaces}
          activeWorkspaceId={activeWorkspaceId}
          onSelectWorkspace={onSelectWorkspace}
          onAddWorkspace={onAddWorkspace}
        />
      }
      header={<WorkspaceSwitcher workspace={activeWorkspace} />}
      nav={
        <NavSlot
          agents={agents}
          agentsLoading={agentsLoading}
          agentsError={agentsError}
          sessions={sessions}
          onAddAgent={onAddAgent}
        />
      }
      footer={<FooterSlot activeSessionCount={activeSessionCount} />}
    />
  );
}

export { AppSidebar, computeAgentsCount };
