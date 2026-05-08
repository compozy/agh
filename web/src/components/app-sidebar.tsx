import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  Book,
  Boxes,
  Clock3,
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

import { cn, Logo, Sidebar, SidebarSectionLabel } from "@agh/ui";

import { ConnectionIndicator, type ConnectionStatus } from "@/components/connection-indicator";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import { AgentCategoryTree, type AgentPayload } from "@/systems/agent";
import type { SessionPayload } from "@/systems/session";
import type { WorkspacePayload } from "@/systems/workspace";

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
        className="mb-1 inline-flex size-7 items-center justify-center rounded-md transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-offset-2 focus-visible:ring-offset-(--color-canvas-deep)"
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
              "inline-flex size-7 items-center justify-center rounded-full border border-transparent bg-(--color-surface-elevated) font-mono text-eyebrow font-medium text-(--color-text-secondary) transition-colors hover:bg-(--color-hover) hover:text-(--color-text-primary) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent",
              isActive && "border-accent text-(--color-text-primary)"
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
        className="inline-flex size-7 items-center justify-center rounded-full border border-dashed border-(--color-divider) text-(--color-text-tertiary) transition-colors hover:border-accent hover:text-(--color-text-primary) focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent"
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
      <Icon aria-hidden="true" className="size-3.5 shrink-0" />
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
  { to: "/settings", icon: Settings, label: "Settings", fuzzy: true },
];

interface NavSlotProps {
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
}

function NavSlot({ agents, agentsLoading, agentsError, sessions }: NavSlotProps) {
  return (
    <div data-testid="sidebar-nav" className="flex flex-col gap-1 px-2 py-3">
      <NavItem
        to={DASHBOARD_NAV_ITEM.to}
        icon={DASHBOARD_NAV_ITEM.icon}
        label={DASHBOARD_NAV_ITEM.label}
      />

      <SectionLabel className="mt-4">Agents</SectionLabel>
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
    <SidebarSectionLabel
      data-testid="sidebar-section-label"
      className={cn("px-2 pt-2 pb-1 text-(--color-text-label) tracking-mono text-micro", className)}
    >
      {children}
    </SidebarSectionLabel>
  );
}

interface FooterSlotProps {
  connectionStatus: ConnectionStatus;
  health: { version: string } | undefined;
}

function FooterSlot({ connectionStatus, health }: FooterSlotProps) {
  return (
    <div data-testid="sidebar-footer" className="flex items-center gap-2">
      <ConnectionIndicator status={connectionStatus} />
      {health && (
        <span
          data-testid="sidebar-version"
          className="ml-auto font-mono text-badge text-(--color-text-tertiary)"
        >
          v{health.version}
        </span>
      )}
    </div>
  );
}

export interface AppSidebarProps {
  collapsed: boolean;
  onCollapseChange: (next: boolean) => void;
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspaceId: string | null;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
  health: { version: string } | undefined;
  connectionStatus: ConnectionStatus;
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
}

function AppSidebar({
  collapsed,
  onCollapseChange,
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  onAddWorkspace,
  health,
  connectionStatus,
  agents,
  agentsLoading,
  agentsError,
  sessions,
}: AppSidebarProps) {
  return (
    <Sidebar
      data-testid="app-sidebar"
      collapsed={collapsed}
      onCollapse={onCollapseChange}
      panelWidth={240}
      rail={
        <RailSlot
          workspaces={workspaces}
          activeWorkspaceId={activeWorkspaceId}
          onSelectWorkspace={onSelectWorkspace}
          onAddWorkspace={onAddWorkspace}
        />
      }
      nav={
        <NavSlot
          agents={agents}
          agentsLoading={agentsLoading}
          agentsError={agentsError}
          sessions={sessions}
        />
      }
      footer={<FooterSlot connectionStatus={connectionStatus} health={health} />}
    />
  );
}

export { AppSidebar };
