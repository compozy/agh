import { useMemo } from "react";
import { Bot, Loader2, Search, Settings, Terminal } from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
  SidebarSeparator,
} from "@/components/ui/sidebar";
import { Kbd } from "@/components/ui/kbd";
import { AgentSidebarGroup } from "@/systems/agent/components/agent-sidebar-group";
import { useAgents } from "@/systems/agent/hooks/use-agents";
import { ConnectionStatus } from "@/systems/daemon/components/connection-status";
import { useDaemonHealth } from "@/systems/daemon/hooks/use-daemon-health";
import { SessionSidebarItem } from "@/systems/session/components/session-sidebar-item";
import { useCreateSession } from "@/systems/session/hooks/use-session-actions";
import { useSessions } from "@/systems/session/hooks/use-sessions";
import type { SessionPayload } from "@/systems/session/types";

function useSessionsByAgent(sessions: SessionPayload[] | undefined) {
  return useMemo(() => {
    if (!sessions) return {};
    const grouped: Record<string, SessionPayload[]> = {};
    for (const session of sessions) {
      const key = session.agent_name;
      if (!grouped[key]) grouped[key] = [];
      grouped[key].push(session);
    }
    // Sort each group by updated_at descending
    for (const key of Object.keys(grouped)) {
      grouped[key].sort(
        (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()
      );
    }
    return grouped;
  }, [sessions]);
}

function AgentsList() {
  const { data: agents, isLoading, isError } = useAgents();
  const { data: sessions } = useSessions();
  const sessionsByAgent = useSessionsByAgent(sessions);
  const createSession = useCreateSession();

  const handleNewSession = (agentName: string) => {
    createSession.mutate({ agent_name: agentName });
  };

  if (isLoading) {
    return (
      <SidebarGroup>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton tooltip="Loading agents...">
                <Loader2 className="size-4 animate-spin text-[color:var(--ds-text-muted)]" />
                <span className="text-[color:var(--ds-text-muted)]">Loading agents...</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    );
  }

  if (isError || !agents || agents.length === 0) {
    return (
      <SidebarGroup>
        <SidebarGroupLabel className="font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--ds-text-mono)]">
          Agents
        </SidebarGroupLabel>
        <SidebarGroupContent>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton tooltip="No agents loaded">
                <Bot className="size-4 text-[color:var(--ds-text-muted)]" />
                <span className="text-[color:var(--ds-text-muted)]">
                  Run `agh install` to bootstrap AGH
                </span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarGroupContent>
      </SidebarGroup>
    );
  }

  return (
    <>
      {agents.map(agent => {
        const agentSessions = sessionsByAgent[agent.name];
        return (
          <AgentSidebarGroup key={agent.name} agent={agent} onNewSession={handleNewSession}>
            {agentSessions?.map(session => (
              <SessionSidebarItem key={session.id} session={session} />
            ))}
          </AgentSidebarGroup>
        );
      })}
    </>
  );
}

function AppSidebar() {
  const { connectionStatus } = useDaemonHealth();

  return (
    <Sidebar side="left" collapsible="icon">
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton size="lg" tooltip="AGH">
              <div className="flex size-8 items-center justify-center rounded-lg bg-[color:var(--ds-panel-accent)] text-[color:var(--ds-accent-amber)]">
                <Terminal className="size-4" />
              </div>
              <div className="grid flex-1 text-left leading-tight">
                <span className="font-display text-sm font-semibold tracking-tight">AGH</span>
                <span className="font-mono text-[0.58rem] uppercase tracking-[0.18em] text-[color:var(--ds-text-mono)]">
                  Agent OS
                </span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Search (⌘K)">
              <Search className="size-4" />
              <span className="flex-1 text-[color:var(--ds-text-muted)]">Search...</span>
              <Kbd className="ml-auto">⌘K</Kbd>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <AgentsList />

        <SidebarSeparator />
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Connection status">
              <ConnectionStatus status={connectionStatus} />
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Settings">
              <Settings className="size-4" />
              <span>Settings</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>

      <SidebarRail />
    </Sidebar>
  );
}

export { AppSidebar };
