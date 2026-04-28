import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  Book,
  Bot,
  Boxes,
  Clock3,
  ListChecks,
  Loader2,
  Network,
  Plus,
  Settings,
  Waypoints,
  Wrench,
  Zap,
  type LucideIcon,
} from "lucide-react";

import { cn, Logo, Sidebar, SidebarSectionLabel, Pill } from "@agh/ui";

import { ConnectionIndicator, type ConnectionStatus } from "@/components/connection-indicator";
import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import { AgentIcon, type AgentPayload } from "@/systems/agent";
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
        className="mb-1 inline-flex size-7 items-center justify-center rounded-md transition-opacity hover:opacity-90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[color:var(--color-canvas-deep)]"
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
              "inline-flex size-7 items-center justify-center rounded-full border border-transparent bg-[color:var(--color-surface-elevated)] font-mono text-[11px] font-medium text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]",
              isActive &&
                "border-[color:var(--color-accent)] text-[color:var(--color-text-primary)]"
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
        className="inline-flex size-7 items-center justify-center rounded-full border border-dashed border-[color:var(--color-divider)] text-[color:var(--color-text-tertiary)] transition-colors hover:border-[color:var(--color-accent)] hover:text-[color:var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]"
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

interface AgentItemProps {
  agent: AgentPayload;
  hasActiveSession: boolean;
}

function AgentItem({ agent, hasActiveSession }: AgentItemProps) {
  const matchRoute = useMatchRoute();
  const isActive = Boolean(
    matchRoute({ to: "/agents/$name", params: { name: agent.name }, fuzzy: true })
  );

  return (
    <Link
      to="/agents/$name"
      params={{ name: agent.name }}
      data-testid={`agent-row-${agent.name}`}
      data-active={isActive}
      className={cn(NAV_ROW_CLASS, isActive && ACTIVE_NAV_ROW_CLASS)}
    >
      {isActive && (
        <span
          aria-hidden="true"
          data-testid={`agent-active-${agent.name}`}
          className={ACTIVE_NAV_INDICATOR_CLASS}
        />
      )}
      <AgentIcon
        provider={agent.provider}
        className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
      />
      <span className="truncate">{agent.name}</span>
      {hasActiveSession ? (
        <Pill.Dot
          tone="success"
          size="sm"
          className="ml-auto"
          data-testid={`agent-status-dot-${agent.name}`}
        />
      ) : null}
    </Link>
  );
}

interface AgentListProps {
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
}

function AgentList({ agents, agentsLoading, agentsError, sessions }: AgentListProps) {
  if (agentsLoading) {
    return (
      <div
        data-testid="agents-loading"
        className="flex items-center gap-2 px-3 py-2 text-[12px] text-[color:var(--color-text-tertiary)]"
      >
        <Loader2 aria-hidden="true" className="size-3 animate-spin" />
        <span>Loading agents...</span>
      </div>
    );
  }

  if (agentsError || !agents || agents.length === 0) {
    return (
      <div
        data-testid="agents-empty"
        className="flex items-center gap-2 px-3 py-2 text-[12px] text-[color:var(--color-text-tertiary)]"
      >
        <Bot aria-hidden="true" className="size-3" />
        <span>Run `agh install` to bootstrap AGH</span>
      </div>
    );
  }

  const activeAgentNames = new Set<string>();
  if (sessions) {
    for (const session of sessions) {
      if (session.state === "active") activeAgentNames.add(session.agent_name);
    }
  }

  return (
    <div className="flex flex-col gap-0.5">
      {agents.map(agent => (
        <AgentItem
          key={agent.name}
          agent={agent}
          hasActiveSession={activeAgentNames.has(agent.name)}
        />
      ))}
    </div>
  );
}

const WORKSPACE_NAV_ITEMS: NavItemProps[] = [
  { to: "/network", icon: Network, label: "Network" },
  { to: "/tasks", icon: ListChecks, label: "Tasks", fuzzy: true },
  { to: "/bridges", icon: Waypoints, label: "Bridges" },
  { to: "/jobs", icon: Clock3, label: "Jobs" },
  { to: "/triggers", icon: Zap, label: "Triggers" },
  { to: "/knowledge", icon: Book, label: "Knowledge" },
  { to: "/skills", icon: Wrench, label: "Skills" },
  { to: "/sandbox", icon: Boxes, label: "Sandbox" },
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
      <SectionLabel>Agents</SectionLabel>
      <AgentList
        agents={agents}
        agentsLoading={agentsLoading}
        agentsError={agentsError}
        sessions={sessions}
      />

      <SectionLabel className="mt-3">Workspace</SectionLabel>
      <div className="flex flex-col gap-0.5">
        {WORKSPACE_NAV_ITEMS.map(item => (
          <NavItem
            key={item.to}
            to={item.to}
            icon={item.icon}
            label={item.label}
            fuzzy={item.fuzzy}
          />
        ))}
      </div>
    </div>
  );
}

function SectionLabel({ children, className }: { children: React.ReactNode; className?: string }) {
  return (
    <SidebarSectionLabel
      data-testid="sidebar-section-label"
      className={cn("px-2 pt-2 pb-1", className)}
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
  const matchRoute = useMatchRoute();
  const settingsActive = Boolean(matchRoute({ to: "/settings", fuzzy: true }));

  return (
    <div data-testid="sidebar-footer" className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        <ConnectionIndicator status={connectionStatus} />
        {health && (
          <span
            data-testid="sidebar-version"
            className="ml-auto font-mono text-[10px] text-[color:var(--color-text-tertiary)]"
          >
            v{health.version}
          </span>
        )}
      </div>
      <Link
        to="/settings"
        data-testid="nav-settings"
        data-active={settingsActive}
        className={cn(NAV_ROW_CLASS, settingsActive && ACTIVE_NAV_ROW_CLASS)}
      >
        {settingsActive && (
          <span
            aria-hidden="true"
            data-testid="nav-active-settings"
            className={ACTIVE_NAV_INDICATOR_CLASS}
          />
        )}
        <Settings aria-hidden="true" className="size-3.5 shrink-0" />
        <span>Settings</span>
      </Link>
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
