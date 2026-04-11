import { Link, useMatchRoute } from "@tanstack/react-router";
import {
  Book,
  Bot,
  ChevronRight,
  Loader2,
  PanelLeftClose,
  PanelLeftOpen,
  Plus,
  Search,
  Settings,
  Terminal,
  Wrench,
} from "lucide-react";
import { useMemo, type ReactNode } from "react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Kbd } from "@/components/ui/kbd";
import { cn } from "@/lib/utils";
import { AgentIcon, type AgentPayload } from "@/systems/agent";
import { ConnectionStatus } from "@/systems/daemon";
import type { SessionPayload, SessionState as SessionStateType } from "@/systems/session";
import type { WorkspacePayload } from "@/systems/workspace";

function useSessionsByAgent(sessions: SessionPayload[] | undefined) {
  return useMemo(() => {
    if (!sessions) return {};

    const grouped: Record<string, SessionPayload[]> = {};
    for (const session of sessions) {
      const key = session.agent_name;
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(session);
    }

    for (const key of Object.keys(grouped)) {
      grouped[key].sort(
        (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      );
    }

    return grouped;
  }, [sessions]);
}

interface IconRailProps {
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspaceId: string | null;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
}

function IconRail({
  workspaces,
  activeWorkspaceId,
  onSelectWorkspace,
  onAddWorkspace,
}: IconRailProps) {
  return (
    <div
      className="flex w-10 shrink-0 flex-col items-center border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface)] py-2"
      data-testid="icon-rail"
    >
      <button
        className="mb-3 flex size-8 items-center justify-center rounded-lg bg-[color:var(--color-accent)] text-white"
        aria-label="AGH Home"
        data-testid="app-logo"
        type="button"
      >
        <Terminal className="size-4" />
      </button>

      <div className="flex flex-1 flex-col items-center gap-2 overflow-y-auto">
        {workspaces?.map(workspace => {
          const isActive = workspace.id === activeWorkspaceId;
          const letter = workspace.name.charAt(0).toUpperCase();

          return (
            <button
              key={workspace.id}
              onClick={() => onSelectWorkspace(workspace.id)}
              className={cn(
                "flex size-8 items-center justify-center rounded-full border-2 border-transparent bg-[color:var(--color-surface-elevated)] text-xs font-medium text-[color:var(--color-text-primary)] transition-colors",
                isActive && "border-[color:var(--color-accent)]"
              )}
              aria-label={`Workspace: ${workspace.name}`}
              data-testid={`workspace-avatar-${workspace.id}`}
              title={workspace.name}
              type="button"
            >
              {letter}
            </button>
          );
        })}
      </div>

      <button
        onClick={onAddWorkspace}
        className="mt-2 flex size-8 items-center justify-center rounded-full text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
        aria-label="Add workspace"
        data-testid="add-workspace-btn"
        type="button"
      >
        <Plus className="size-4" />
      </button>
    </div>
  );
}

function SidebarSessionItem({ session }: { session: SessionPayload }) {
  const matchRoute = useMatchRoute();
  const isActive = !!matchRoute({ to: "/session/$id", params: { id: session.id } });
  const displayTitle = session.name || session.id.slice(0, 8);

  return (
    <Link
      to="/session/$id"
      params={{ id: session.id }}
      className={cn(
        "relative flex items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors",
        "text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]",
        isActive &&
          "bg-[color:var(--color-hover)] font-medium text-[color:var(--color-text-primary)]"
      )}
    >
      {isActive && (
        <span className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]" />
      )}
      <SessionStateDot state={session.state} />
      <span className="truncate">{displayTitle}</span>
    </Link>
  );
}

function SessionStateDot({ state }: { state: SessionStateType }) {
  const colorMap: Record<SessionStateType, string> = {
    active: "bg-[color:var(--color-success)]",
    starting: "bg-[color:var(--color-warning)] animate-pulse",
    stopping: "bg-[color:var(--color-text-tertiary)] animate-pulse",
    stopped: "bg-[color:var(--color-text-tertiary)]",
  };

  return <span className={cn("size-1.5 shrink-0 rounded-full", colorMap[state])} />;
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
      <div className="flex items-center gap-1 px-2 py-1">
        <CollapsibleTrigger className="flex flex-1 items-center gap-2 rounded-md py-0.5 text-left text-xs font-medium text-[color:var(--color-text-primary)] hover:text-[color:var(--color-text-primary)]">
          <AgentIcon
            provider={agent.provider}
            className="size-3.5 text-[color:var(--color-text-tertiary)]"
          />
          <span className="truncate">{agent.name}</span>
          <span className="ml-auto font-mono text-[0.6rem] text-[color:var(--color-text-tertiary)]">
            {count}
          </span>
          <ChevronRight className="size-3 text-[color:var(--color-text-tertiary)] transition-transform group-data-[state=open]/agent:rotate-90" />
        </CollapsibleTrigger>
        <button
          onClick={() => onNewSession(agent.name)}
          disabled={newSessionDisabled}
          className="flex size-5 items-center justify-center rounded text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)] disabled:opacity-40"
          aria-label={`New session for ${agent.name}`}
          data-testid={`new-session-${agent.name}`}
          type="button"
        >
          <Plus className="size-3" />
        </button>
      </div>
      <CollapsibleContent>
        <div className="ml-4 flex flex-col gap-0.5 pb-1">
          {sessions && sessions.length > 0 ? (
            sessions.map(session => <SidebarSessionItem key={session.id} session={session} />)
          ) : (
            <span className="px-2 py-1 text-[0.68rem] text-[color:var(--color-text-tertiary)]">
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
      <div className="flex items-center gap-2 px-3 py-2 text-xs text-[color:var(--color-text-tertiary)]">
        <Loader2 className="size-3 animate-spin" />
        <span>Loading agents...</span>
      </div>
    );
  }

  if (agentsError || !agents || agents.length === 0) {
    return (
      <div className="flex items-center gap-2 px-3 py-2 text-xs text-[color:var(--color-text-tertiary)]">
        <Bot className="size-3" />
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

interface NavItemProps {
  to: string;
  icon: ReactNode;
  label: string;
}

function NavItem({ to, icon, label }: NavItemProps) {
  const matchRoute = useMatchRoute();
  const isActive = !!matchRoute({ to });

  return (
    <Link
      to={to}
      className={cn(
        "relative flex items-center gap-2 rounded-md px-2 py-1.5 text-xs transition-colors",
        "text-[color:var(--color-text-secondary)] hover:bg-[color:var(--color-hover)]",
        isActive &&
          "bg-[color:var(--color-hover)] font-medium text-[color:var(--color-text-primary)]"
      )}
      data-testid={`nav-${label.toLowerCase()}`}
    >
      {isActive && (
        <span
          className="absolute left-0 top-1 bottom-1 w-[3px] rounded-r bg-[color:var(--color-accent)]"
          data-testid={`nav-active-${label.toLowerCase()}`}
        />
      )}
      {icon}
      <span>{label}</span>
    </Link>
  );
}

interface SidebarPanelProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
  activeWorkspace: WorkspacePayload | undefined;
  activeWorkspaceId: string | null;
  health: { version: string } | undefined;
  connectionStatus: "connected" | "disconnected" | "reconnecting";
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  isCreatingSession: boolean;
}

function SidebarPanel({
  collapsed,
  onToggleCollapsed,
  activeWorkspace,
  activeWorkspaceId,
  health,
  connectionStatus,
  agents,
  agentsLoading,
  agentsError,
  sessions,
  onNewSession,
  isCreatingSession,
}: SidebarPanelProps) {
  return (
    <div
      className={cn(
        "flex flex-col overflow-hidden bg-[color:var(--color-surface)] transition-[width] duration-200",
        collapsed ? "w-0" : "w-[220px]"
      )}
      data-testid="sidebar-panel"
    >
      <div className="flex min-w-[220px] flex-1 flex-col">
        <div className="flex items-center gap-2 border-b border-[color:var(--color-divider)] px-3 py-2.5">
          <span className="flex-1 truncate text-sm font-semibold text-[color:var(--color-text-primary)]">
            {activeWorkspace?.name ?? "AGH"}
          </span>
          <button
            className="flex size-6 items-center justify-center rounded text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
            aria-label="Search"
            type="button"
          >
            <Search className="size-3.5" />
          </button>
          <button
            onClick={onToggleCollapsed}
            className="flex size-6 items-center justify-center rounded text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
            aria-label="Collapse sidebar"
            data-testid="collapse-toggle"
            type="button"
          >
            <PanelLeftClose className="size-3.5" />
          </button>
        </div>

        <div className="px-3 py-2">
          <div className="flex items-center gap-2 rounded-md bg-[color:var(--color-canvas)] px-2.5 py-1.5 text-xs text-[color:var(--color-text-tertiary)]">
            <Search className="size-3" />
            <span className="flex-1">Search...</span>
            <Kbd className="text-[0.55rem]">⌘K</Kbd>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto px-1">
          <div className="px-2 pb-1 pt-2">
            <span className="font-mono text-[0.6rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]">
              Agents
            </span>
          </div>
          <AgentList
            activeWorkspaceId={activeWorkspaceId}
            agents={agents}
            agentsLoading={agentsLoading}
            agentsError={agentsError}
            sessions={sessions}
            onNewSession={onNewSession}
            isCreatingSession={isCreatingSession}
          />

          <div className="mt-3 px-2 pb-1">
            <span className="font-mono text-[0.6rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]">
              Workspace
            </span>
          </div>
          <div className="flex flex-col gap-0.5 px-1">
            <NavItem to="/knowledge" icon={<Book className="size-3.5" />} label="Knowledge" />
            <NavItem to="/skills" icon={<Wrench className="size-3.5" />} label="Skills" />
          </div>
        </div>

        <div className="border-t border-[color:var(--color-divider)] px-3 py-2">
          <div className="flex items-center gap-2" data-testid="sidebar-footer">
            <ConnectionStatus status={connectionStatus} />
            {health && (
              <span className="font-mono text-[0.55rem] text-[color:var(--color-text-tertiary)]">
                v{health.version}
              </span>
            )}
          </div>
          <button
            className="mt-1.5 flex w-full items-center gap-2 rounded-md px-0 py-1 text-xs text-[color:var(--color-text-secondary)] transition-colors hover:text-[color:var(--color-text-primary)]"
            type="button"
          >
            <Settings className="size-3.5" />
            <span>Settings</span>
          </button>
        </div>
      </div>
    </div>
  );
}

function ExpandButton({
  collapsed,
  onToggleCollapsed,
}: {
  collapsed: boolean;
  onToggleCollapsed: () => void;
}) {
  if (!collapsed) return null;

  return (
    <button
      onClick={onToggleCollapsed}
      className="absolute left-10 top-2 z-10 flex size-6 items-center justify-center rounded bg-[color:var(--color-surface)] text-[color:var(--color-text-tertiary)] transition-colors hover:bg-[color:var(--color-hover)] hover:text-[color:var(--color-text-secondary)]"
      aria-label="Expand sidebar"
      data-testid="expand-toggle"
      type="button"
    >
      <PanelLeftOpen className="size-3.5" />
    </button>
  );
}

export interface AppSidebarProps {
  collapsed: boolean;
  onToggleCollapsed: () => void;
  workspaces: WorkspacePayload[] | undefined;
  activeWorkspace: WorkspacePayload | undefined;
  activeWorkspaceId: string | null;
  onSelectWorkspace: (id: string) => void;
  onAddWorkspace: () => void;
  health: { version: string } | undefined;
  connectionStatus: "connected" | "disconnected" | "reconnecting";
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
  onNewSession: (agentName: string) => void;
  isCreatingSession: boolean;
}

function AppSidebar({
  collapsed,
  onToggleCollapsed,
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
    <aside
      className="relative flex h-screen shrink-0 border-r border-[color:var(--color-divider)]"
      data-testid="app-sidebar"
    >
      <IconRail
        workspaces={workspaces}
        activeWorkspaceId={activeWorkspaceId}
        onSelectWorkspace={onSelectWorkspace}
        onAddWorkspace={onAddWorkspace}
      />
      <SidebarPanel
        collapsed={collapsed}
        onToggleCollapsed={onToggleCollapsed}
        activeWorkspace={activeWorkspace}
        activeWorkspaceId={activeWorkspaceId}
        health={health}
        connectionStatus={connectionStatus}
        agents={agents}
        agentsLoading={agentsLoading}
        agentsError={agentsError}
        sessions={sessions}
        onNewSession={onNewSession}
        isCreatingSession={isCreatingSession}
      />
      <ExpandButton collapsed={collapsed} onToggleCollapsed={onToggleCollapsed} />
    </aside>
  );
}

export { AppSidebar };
