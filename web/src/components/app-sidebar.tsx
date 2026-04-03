import { Bot, Search, Settings, Terminal } from "lucide-react";

import { ConnectionIndicator } from "@/components/design-system/connection-indicator";
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

function AppSidebar() {
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
        <SidebarGroup>
          <SidebarGroupLabel className="font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--ds-text-mono)]">
            Agents
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton tooltip="No agents loaded">
                  <Bot className="size-4 text-[color:var(--ds-text-muted)]" />
                  <span className="text-[color:var(--ds-text-muted)]">No agents loaded</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarSeparator />

        <SidebarGroup>
          <SidebarGroupLabel className="font-mono text-[0.64rem] uppercase tracking-[0.22em] text-[color:var(--ds-text-mono)]">
            Recent Sessions
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <SidebarMenuButton tooltip="No sessions">
                  <span className="text-[color:var(--ds-text-muted)]">No sessions yet</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>

      <SidebarFooter>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton tooltip="Connection status">
              <ConnectionIndicator status="disconnected" />
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
