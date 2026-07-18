import { Link } from "@tanstack/react-router";
import {
  Book,
  Boxes,
  CircleAlert,
  Clock3,
  Home,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  Network,
  Plus,
  Repeat2,
  Settings,
  Store,
  Users2,
  Waypoints,
  Zap,
  type LucideIcon,
} from "lucide-react";
import type { ReactNode } from "react";

import { Logo, Sidebar, SidebarSectionLabel, cn } from "@agh/ui";

import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import {
  splitHomeWorkspace,
  useUserHomeDir,
  WorkspaceCommandSelect,
  type WorkspacePayload,
} from "@/systems/workspace";
import {
  createSessionReturnHistoryState,
  type WorkspaceSessionActivity,
  type WorkspaceSessionActivityMap,
} from "@/systems/session";

import { RuntimeConnectionIndicator } from "./connection-indicator";
import { RestartDaemonButton } from "./restart-daemon-button";

export interface AgentsCount {
  live: number;
  total: number;
}

interface RailSlotProps {
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspaceId: string | null;
  workspaceSessionActivity: WorkspaceSessionActivityMap;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
}

interface WorkspaceRailItemProps {
  workspace: WorkspacePayload;
  isActive: boolean;
  isHome: boolean;
  activity: WorkspaceSessionActivity | undefined;
  onSelect: (id: string) => void;
}

const workspaceRailItemClassName =
  "relative inline-flex size-7 items-center justify-center rounded-md border border-transparent bg-elevated font-mono text-eyebrow font-medium text-muted transition-colors hover:bg-hover hover:text-fg focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent";

function WorkspaceSessionCount({ workspaceId, count }: { workspaceId: string; count: number }) {
  return (
    <span
      aria-hidden="true"
      data-testid={`workspace-active-session-count-${workspaceId}`}
      className="absolute -top-1.5 -right-2 inline-flex min-h-4 min-w-4 items-center justify-center rounded-pill border border-rail bg-success-tint px-1 text-badge leading-none font-semibold text-success tabular-nums"
    >
      {count}
    </span>
  );
}

function WorkspaceRailItem({
  workspace,
  isActive,
  isHome,
  activity,
  onSelect,
}: WorkspaceRailItemProps) {
  const workspaceLabel = isHome
    ? `Home workspace: ${workspace.name}`
    : `Workspace: ${workspace.name}`;
  const activityCount =
    !isActive && activity?.state === "ready" && activity.count > 0 ? activity.count : 0;
  const activityError = activity?.state === "error" ? activity.message : null;
  const glyph = isHome ? (
    <Home aria-hidden="true" className="size-3.5" />
  ) : (
    workspace.name.charAt(0).toUpperCase() || "·"
  );

  if (activityCount > 0 && activity?.state === "ready" && activity.returnTarget) {
    const target = activity.returnTarget;
    return (
      <Link
        to="/agents/$name/sessions/$id"
        params={{ name: target.agentName, id: target.sessionId }}
        state={createSessionReturnHistoryState(target.sessionId, workspace.id)}
        data-testid={`workspace-avatar-${workspace.id}`}
        data-active="false"
        data-home={isHome ? "true" : undefined}
        title={`Return to ${workspace.name}: ${target.title}`}
        aria-label={`Return to ${workspace.name}: ${activityCount} active ${activityCount === 1 ? "session" : "sessions"}. Latest: ${target.title}`}
        className={workspaceRailItemClassName}
      >
        {glyph}
        <WorkspaceSessionCount workspaceId={workspace.id} count={activityCount} />
      </Link>
    );
  }

  const activityDescription =
    activityCount > 0
      ? `, ${activityCount} active ${activityCount === 1 ? "session" : "sessions"}`
      : "";
  const availabilityDescription = activityError ? ", session activity unavailable" : "";
  return (
    <button
      type="button"
      onClick={() => onSelect(workspace.id)}
      data-testid={`workspace-avatar-${workspace.id}`}
      data-active={isActive}
      data-home={isHome ? "true" : undefined}
      title={isHome ? "Home workspace" : workspace.name}
      aria-label={`${workspaceLabel}${activityDescription}${availabilityDescription}`}
      aria-pressed={isActive}
      aria-busy={activity?.state === "loading" || undefined}
      className={cn(workspaceRailItemClassName, isActive && "border-accent text-fg")}
    >
      {glyph}
      {activityCount > 0 ? (
        <WorkspaceSessionCount workspaceId={workspace.id} count={activityCount} />
      ) : null}
      {activityError ? (
        <span
          className="absolute -right-1.5 -bottom-1.5 grid size-3.5 place-items-center rounded-pill border border-rail bg-warning-tint text-warning"
          data-testid={`workspace-session-activity-error-${workspace.id}`}
          title={activityError}
        >
          <CircleAlert aria-hidden="true" className="size-2.5" />
        </span>
      ) : null}
    </button>
  );
}

function RailSlot({
  workspaces,
  activeWorkspaceId,
  workspaceSessionActivity,
  onSelectWorkspace,
  onAddWorkspace,
}: RailSlotProps) {
  const userHomeDir = useUserHomeDir();
  const { homeWorkspace, projectWorkspaces } = splitHomeWorkspace(workspaces, userHomeDir);

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
      {homeWorkspace && (
        <WorkspaceRailItem
          workspace={homeWorkspace}
          isActive={homeWorkspace.id === activeWorkspaceId}
          isHome
          activity={workspaceSessionActivity[homeWorkspace.id]}
          onSelect={onSelectWorkspace}
        />
      )}
      {homeWorkspace && projectWorkspaces.length > 0 && (
        <div
          aria-hidden="true"
          data-testid="rail-home-divider"
          className="my-0.5 h-px w-5 rounded-full bg-line"
        />
      )}
      {projectWorkspaces.map(workspace => (
        <WorkspaceRailItem
          key={workspace.id}
          workspace={workspace}
          isActive={workspace.id === activeWorkspaceId}
          isHome={false}
          activity={workspaceSessionActivity[workspace.id]}
          onSelect={onSelectWorkspace}
        />
      ))}
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
  badge?: ReactNode;
}

function NavItem({ to, icon: Icon, label, fuzzy, badge }: NavItemProps) {
  const testKey = label.toLowerCase();

  return (
    <Link
      to={to}
      activeOptions={{ exact: !fuzzy, includeSearch: false }}
      activeProps={{ className: ACTIVE_NAV_ROW_CLASS, "data-active": "true" }}
      data-testid={`nav-${testKey}`}
      inactiveProps={{ "data-active": "false" }}
      className={NAV_ROW_CLASS}
    >
      {({ isActive }) => (
        <>
          {isActive && (
            <span
              aria-hidden="true"
              data-testid={`nav-active-${testKey}`}
              className={ACTIVE_NAV_INDICATOR_CLASS}
            />
          )}
          <Icon aria-hidden="true" className="size-3 shrink-0" />
          <span className="truncate">{label}</span>
          {badge ? <span className="ml-auto shrink-0">{badge}</span> : null}
        </>
      )}
    </Link>
  );
}

const DASHBOARD_NAV_ITEM: NavItemProps = {
  to: "/",
  icon: LayoutDashboard,
  label: "Dashboard",
};

const OPERATE_NAV_ITEMS: NavItemProps[] = [
  { to: "/agents", icon: Users2, label: "Agents", fuzzy: true },
  { to: "/network", icon: Network, label: "Network" },
  { to: "/tasks", icon: ListChecks, label: "Tasks", fuzzy: true },
  { to: "/loops", icon: Repeat2, label: "Loops", fuzzy: true },
  { to: "/jobs", icon: Clock3, label: "Jobs", fuzzy: true },
  { to: "/triggers", icon: Zap, label: "Triggers", fuzzy: true },
];

const CATALOG_NAV_ITEMS: NavItemProps[] = [
  { to: "/marketplace", icon: Store, label: "Marketplace", fuzzy: true },
  { to: "/bridges", icon: Waypoints, label: "Bridges" },
  { to: "/knowledge", icon: Book, label: "Knowledge" },
];

const SYSTEM_NAV_ITEMS: NavItemProps[] = [
  { to: "/sandbox", icon: Boxes, label: "Sandbox" },
  { to: "/vault", icon: KeyRound, label: "Vault" },
  { to: "/settings", icon: Settings, label: "Settings", fuzzy: true },
];

interface NavSlotProps {
  agentsCount: AgentsCount | undefined;
}

function NavSlot({ agentsCount }: NavSlotProps) {
  const agentsBadge =
    agentsCount && agentsCount.total > 0 ? (
      <span className="tabular-nums text-subtle" data-testid="agents-live-count">
        {agentsCount.live}/{agentsCount.total}
      </span>
    ) : null;

  const operateItems = OPERATE_NAV_ITEMS.map(item =>
    item.to === "/agents" ? { ...item, badge: agentsBadge } : item
  );

  return (
    <div data-testid="sidebar-nav" className="flex flex-col gap-1 px-2 py-3">
      <NavItem
        to={DASHBOARD_NAV_ITEM.to}
        icon={DASHBOARD_NAV_ITEM.icon}
        label={DASHBOARD_NAV_ITEM.label}
      />

      <SectionLabel className="mt-4">Operate</SectionLabel>
      <NavGroup items={operateItems} />

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
          badge={item.badge}
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
  agentsCount: AgentsCount | undefined;
  activeSessionCount: number;
  workspaceSessionActivity: WorkspaceSessionActivityMap;
  className?: string;
}

function AppSidebar({
  collapsed,
  onCollapseChange,
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  onAddWorkspace,
  agentsCount,
  activeSessionCount,
  workspaceSessionActivity,
  className,
}: AppSidebarProps) {
  const rail = (
    <RailSlot
      workspaces={workspaces}
      activeWorkspaceId={activeWorkspaceId}
      workspaceSessionActivity={workspaceSessionActivity}
      onSelectWorkspace={onSelectWorkspace}
      onAddWorkspace={onAddWorkspace}
    />
  );
  const header = (
    <WorkspaceCommandSelect
      workspaces={workspaces}
      value={activeWorkspaceId}
      onChange={onSelectWorkspace}
      onAddWorkspace={onAddWorkspace}
    />
  );
  const nav = <NavSlot agentsCount={agentsCount} />;
  const footer = <FooterSlot activeSessionCount={activeSessionCount} />;

  return (
    <Sidebar
      data-testid="app-sidebar"
      className={className}
      collapsed={collapsed}
      onCollapse={onCollapseChange}
      rail={rail}
      header={header}
      nav={nav}
      footer={footer}
    />
  );
}

export { AppSidebar };
