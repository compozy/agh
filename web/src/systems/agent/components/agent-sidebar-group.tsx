import { ChevronRight, Plus } from "lucide-react";
import type { ReactNode } from "react";

import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import {
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
} from "@/components/ui/sidebar";
import type { AgentPayload } from "../types";
import { AgentIcon } from "./agent-icon";

interface AgentSidebarGroupProps {
  agent: AgentPayload;
  onNewSession?: (agentName: string) => void;
  newSessionDisabled?: boolean;
  children?: ReactNode;
}

function AgentSidebarGroup({
  agent,
  onNewSession,
  newSessionDisabled = false,
  children,
}: AgentSidebarGroupProps) {
  const hasChildren = children != null;

  return (
    <Collapsible defaultOpen className="group/collapsible">
      <SidebarGroup className="p-0 pb-1">
        <SidebarGroupLabel
          className="font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--color-text-label)]"
          render={<CollapsibleTrigger />}
        >
          <AgentIcon provider={agent.provider} className="mr-1 size-3.5" />
          <span className="truncate">{agent.name}</span>
          <ChevronRight className="ml-auto size-3 transition-transform group-data-[panel-open]/collapsible:rotate-90" />
        </SidebarGroupLabel>
        <SidebarGroupAction
          title="New Session"
          onClick={() => onNewSession?.(agent.name)}
          disabled={newSessionDisabled}
        >
          <Plus className="size-4" />
          <span className="sr-only">New Session</span>
        </SidebarGroupAction>
        <CollapsibleContent>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuSub>
                {hasChildren ? (
                  children
                ) : (
                  <SidebarMenuItem>
                    <SidebarMenuButton
                      size="sm"
                      className="text-[color:var(--color-text-tertiary)]"
                      tooltip="No sessions"
                    >
                      <span className="text-xs">No sessions</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                )}
              </SidebarMenuSub>
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}

export { AgentSidebarGroup };
