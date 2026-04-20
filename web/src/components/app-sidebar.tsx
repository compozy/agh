import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  Book,
  Bot,
  ChevronRight,
  ListChecks,
  Loader2,
  Network,
  Plus,
  Settings,
  Terminal,
  Waypoints,
  Wrench,
  type LucideIcon,
} from "lucide-react";

import {
  cn,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  ConnectionIndicator,
  type ConnectionStatus,
  Sidebar,
  StatusDot,
  type StatusDotTone,
} from "@agh/ui";

import { useSessionsByAgent } from "@/hooks/use-sessions-by-agent";
import { AgentIcon, type AgentPayload } from "@/systems/agent";
import type { SessionPayload, SessionState } from "@/systems/session";
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
        className="mb-1 inline-flex size-7 items-center justify-center rounded-md bg-[color:var(--color-accent)] text-[color:var(--color-accent-ink)] transition-colors hover:bg-[color:var(--color-accent-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[color:var(--color-canvas-deep)]"
      >
        <Terminal aria-hidden="true" className="size-3.5" />
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

interface HeaderSlotProps {
  activeWorkspace: WorkspacePayload | undefined;
}

function HeaderSlot({ activeWorkspace }: HeaderSlotProps) {
  return (
    <span
      data-testid="sidebar-workspace-name"
      className="flex-1 truncate text-[13px] font-medium text-[color:var(--color-text-primary)]"
    >
      {activeWorkspace?.name ?? ""}
    </span>
  );
}

const SESSION_STATE_TONE: Record<SessionState, { tone: StatusDotTone; pulse: boolean }> = {
  active: { tone: "success", pulse: false },
  starting: { tone: "warning", pulse: true },
  stopping: { tone: "neutral", pulse: true },
  stopped: { tone: "neutral", pulse: false },
};

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
      className={cn(
        "relative flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
        isActive &&
          "bg-[color:var(--color-surface-elevated)] font-medium text-[color:var(--color-text-primary)]"
      )}
    >
      {isActive && (
        <span
          aria-hidden="true"
          data-testid={`nav-active-${testKey}`}
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
        />
      )}
      <Icon aria-hidden="true" className="size-3.5 shrink-0" />
      <span className="truncate">{label}</span>
    </Link>
  );
}

interface SidebarSessionItemProps {
  session: SessionPayload;
}

function SidebarSessionItem({ session }: SidebarSessionItemProps) {
  const matchRoute = useMatchRoute();
  const isActive = Boolean(matchRoute({ to: "/session/$id", params: { id: session.id } }));
  const displayTitle = session.name || session.id.slice(0, 8);
  const { tone, pulse } = SESSION_STATE_TONE[session.state];

  return (
    <Link
      to="/session/$id"
      params={{ id: session.id }}
      data-testid={`session-row-${session.id}`}
      data-active={isActive}
      className={cn(
        "relative flex items-center gap-2 rounded-md px-2 py-1 text-[12px] text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
        isActive &&
          "bg-[color:var(--color-surface-elevated)] font-medium text-[color:var(--color-text-primary)]"
      )}
    >
      {isActive && (
        <span
          aria-hidden="true"
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
        />
      )}
      <StatusDot tone={tone} pulse={pulse} size="sm" />
      <span className="truncate">{displayTitle}</span>
    </Link>
  );
}

interface AgentItemProps {
  agent: AgentPayload;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  newSessionDisabled: boolean;
}

function AgentItem({ agent, sessions, onNewSession, newSessionDisabled }: AgentItemProps) {
  const count = sessions?.length ?? 0;

  return (
    <Collapsible defaultOpen={count > 0} className="group/agent">
      <div className="relative flex items-center">
        <CollapsibleTrigger
          data-testid={`agent-trigger-${agent.name}`}
          className="flex min-h-7 flex-1 items-center gap-1.5 rounded-md px-2 text-left text-[13px] text-[color:var(--color-text-primary)] transition-colors hover:bg-[color:var(--color-hover)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)]"
        >
          <ChevronRight
            aria-hidden="true"
            className="size-3 shrink-0 text-[color:var(--color-text-tertiary)] transition-transform group-data-[panel-open]/agent:rotate-90"
          />
          <AgentIcon
            provider={agent.provider}
            className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]"
          />
          <span className="truncate">{agent.name}</span>
          <span className="ml-auto font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
            {count}
          </span>
        </CollapsibleTrigger>
        <button
          type="button"
          onClick={() => onNewSession(agent.name)}
          disabled={newSessionDisabled}
          aria-label={`New session for ${agent.name}`}
          data-testid={`new-session-${agent.name}`}
          className="ml-1 inline-flex size-5 items-center justify-center rounded text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[color:var(--color-accent)] disabled:pointer-events-none disabled:opacity-40"
        >
          <Plus aria-hidden="true" className="size-3" />
        </button>
      </div>
      <CollapsibleContent>
        <div className="ml-[18px] flex flex-col gap-0.5 border-l border-[color:var(--color-divider)] pl-2 pt-0.5">
          {sessions && sessions.length > 0 ? (
            sessions.map(session => <SidebarSessionItem key={session.id} session={session} />)
          ) : (
            <span className="px-2 py-1 text-[11px] text-[color:var(--color-text-tertiary)]">
              No sessions
            </span>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}

interface AgentListProps {
  activeWorkspaceId: string | null;
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  isCreatingSession: boolean;
}

function AgentList({
  activeWorkspaceId,
  agents,
  agentsLoading,
  agentsError,
  sessions,
  onNewSession,
  isCreatingSession,
}: AgentListProps) {
  const sessionsByAgent = useSessionsByAgent(sessions);

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

  return (
    <div className="flex flex-col gap-0.5">
      {agents.map(agent => (
        <AgentItem
          key={agent.name}
          agent={agent}
          sessions={sessionsByAgent[agent.name]}
          onNewSession={onNewSession}
          newSessionDisabled={!activeWorkspaceId || isCreatingSession}
        />
      ))}
    </div>
  );
}

const WORKSPACE_NAV_ITEMS: NavItemProps[] = [
  { to: "/tasks", icon: ListChecks, label: "Tasks", fuzzy: true },
  { to: "/automation", icon: Bot, label: "Automation" },
  { to: "/bridges", icon: Waypoints, label: "Bridges" },
  { to: "/network", icon: Network, label: "Network" },
  { to: "/knowledge", icon: Book, label: "Knowledge" },
  { to: "/skills", icon: Wrench, label: "Skills" },
];

interface NavSlotProps {
  activeWorkspaceId: string | null;
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  isCreatingSession: boolean;
}

function NavSlot({
  activeWorkspaceId,
  agents,
  agentsLoading,
  agentsError,
  sessions,
  onNewSession,
  isCreatingSession,
}: NavSlotProps) {
  return (
    <div data-testid="sidebar-nav" className="flex flex-col gap-1 px-2 py-3">
      <SectionLabel>Agents</SectionLabel>
      <AgentList
        activeWorkspaceId={activeWorkspaceId}
        agents={agents}
        agentsLoading={agentsLoading}
        agentsError={agentsError}
        sessions={sessions}
        onNewSession={onNewSession}
        isCreatingSession={isCreatingSession}
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
    <span
      data-testid="sidebar-section-label"
      className={cn(
        "px-2 pt-2 pb-1 font-mono text-[11px] font-medium uppercase tracking-[0.18em] text-[color:var(--color-text-label)]",
        className
      )}
    >
      {children}
    </span>
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
        className={cn(
          "relative flex items-center gap-2 rounded-md px-2 py-1.5 text-[13px] text-[color:var(--color-text-secondary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-primary)]",
          settingsActive &&
            "bg-[color:var(--color-surface-elevated)] font-medium text-[color:var(--color-text-primary)]"
        )}
      >
        {settingsActive && (
          <span
            aria-hidden="true"
            data-testid="nav-active-settings"
            className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
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
  activeWorkspace: WorkspacePayload | undefined;
  activeWorkspaceId: string | null;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
  health: { version: string } | undefined;
  connectionStatus: ConnectionStatus;
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  isCreatingSession: boolean;
}

function AppSidebar({
  collapsed,
  onCollapseChange,
  workspaces,
  activeWorkspace,
  activeWorkspaceId,
  onSelectWorkspace,
  onAddWorkspace,
  health,
  connectionStatus,
  agents,
  agentsLoading,
  agentsError,
  sessions,
  onNewSession,
  isCreatingSession,
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
      header={<HeaderSlot activeWorkspace={activeWorkspace} />}
      nav={
        <NavSlot
          activeWorkspaceId={activeWorkspaceId}
          agents={agents}
          agentsLoading={agentsLoading}
          agentsError={agentsError}
          sessions={sessions}
          onNewSession={onNewSession}
          isCreatingSession={isCreatingSession}
        />
      }
      footer={<FooterSlot connectionStatus={connectionStatus} health={health} />}
    />
  );
}

export { AppSidebar };
