import { ChevronRight, Plus } from "lucide-react";

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
}

function AgentSidebarGroup({ agent, onNewSession }: AgentSidebarGroupProps) {
  return (
    <Collapsible defaultOpen className="group/collapsible">
      <SidebarGroup className="p-0 pb-1">
        <SidebarGroupLabel
          className="font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--ds-text-mono)]"
          render={<CollapsibleTrigger />}
        >
          <AgentIcon provider={agent.provider} className="mr-1 size-3.5" />
          <span className="truncate">{agent.name}</span>
          <ChevronRight className="ml-auto size-3 transition-transform group-data-[panel-open]/collapsible:rotate-90" />
        </SidebarGroupLabel>
        <SidebarGroupAction title="New Session" onClick={() => onNewSession?.(agent.name)}>
          <Plus className="size-4" />
          <span className="sr-only">New Session</span>
        </SidebarGroupAction>
        <CollapsibleContent>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuSub>
                {/* Session items will be rendered here by task_03 */}
                <SidebarMenuItem>
                  <SidebarMenuButton
                    size="sm"
                    className="text-[color:var(--ds-text-muted)]"
                    tooltip="No sessions"
                  >
                    <span className="text-xs">No sessions</span>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenuSub>
            </SidebarMenu>
          </SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  );
}

export { AgentSidebarGroup };
