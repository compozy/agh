import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "@agh/ui";
import { CircleDotIcon, CpuIcon, DatabaseIcon, LayersIcon, SettingsIcon } from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuBadge,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarTrigger,
} from "../sidebar";

const meta: Meta<typeof Sidebar> = {
  title: "components/ui/Sidebar",
  component: Sidebar,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "App shell chrome. Always wrap in a SidebarProvider; pair Sidebar with SidebarInset + SidebarTrigger for a realistic layout.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

type NavItem = {
  label: string;
  icon: typeof CircleDotIcon;
  badge?: string;
};

const navItems: NavItem[] = [
  { label: "Sessions", icon: CircleDotIcon, badge: "4" },
  { label: "Agents", icon: CpuIcon },
  { label: "Skills", icon: LayersIcon },
  { label: "Memory", icon: DatabaseIcon },
];

export const Default: Story = {
  args: {},
  render: () => (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center gap-2 px-2 text-sm font-medium">
            <SettingsIcon className="size-4" />
            AGH
          </div>
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Workspace</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems.map((item, index) => (
                  <SidebarMenuItem key={item.label}>
                    <SidebarMenuButton isActive={index === 0}>
                      <item.icon />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                    {item.badge ? <SidebarMenuBadge>{item.badge}</SidebarMenuBadge> : null}
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
        <SidebarFooter>
          <Button variant="outline" size="sm" className="justify-start">
            Invite teammates
          </Button>
        </SidebarFooter>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-12 items-center gap-2 border-b bg-background px-4">
          <SidebarTrigger />
          <span className="text-sm font-medium">Sessions</span>
        </header>
        <div className="p-6 text-sm text-muted-foreground">
          Collapse the rail with the trigger above to exercise open/closed state.
        </div>
      </SidebarInset>
    </SidebarProvider>
  ),
};

export const StartsCollapsed: Story = {
  args: {},
  render: () => (
    <SidebarProvider defaultOpen={false}>
      <Sidebar collapsible="icon">
        <SidebarHeader>
          <SettingsIcon className="size-4" />
        </SidebarHeader>
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupContent>
              <SidebarMenu>
                {navItems.map(item => (
                  <SidebarMenuItem key={item.label}>
                    <SidebarMenuButton tooltip={item.label}>
                      <item.icon />
                      <span>{item.label}</span>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
      <SidebarInset>
        <header className="flex h-12 items-center gap-2 border-b bg-background px-4">
          <SidebarTrigger />
          <span className="text-sm font-medium">Collapsed rail</span>
        </header>
        <div className="p-6 text-sm text-muted-foreground">
          Hover a rail icon to see the tooltip fallback while collapsed.
        </div>
      </SidebarInset>
    </SidebarProvider>
  ),
};
