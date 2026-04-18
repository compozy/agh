import type { Meta, StoryObj } from "@storybook/react-vite";
import { Kbd, KbdGroup } from "@agh/ui";
import {
  CircleDotIcon,
  CpuIcon,
  DatabaseIcon,
  LayersIcon,
  SearchIcon,
  SettingsIcon,
} from "lucide-react";

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
  CommandShortcut,
} from "../command";

const meta: Meta<typeof Command> = {
  title: "components/ui/Command",
  component: Command,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "cmdk-powered palette. Use as a standalone panel or inside a CommandDialog. Items stay searchable via the input.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const navigateItems = [
  { value: "sessions", label: "Go to sessions", icon: CircleDotIcon },
  { value: "agents", label: "Go to agents", icon: CpuIcon },
  { value: "skills", label: "Go to skills", icon: LayersIcon },
  { value: "memory", label: "Go to memory", icon: DatabaseIcon },
];

const quickItems = [
  { value: "new-session", label: "Start new session", shortcut: "⌘N" },
  { value: "search", label: "Search events", shortcut: "⌘F" },
  { value: "settings", label: "Open settings", icon: SettingsIcon },
];

export const Default: Story = {
  args: {},
  render: () => (
    <Command className="w-[24rem] border">
      <CommandInput placeholder="Type a command or search…" />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>
        <CommandGroup heading="Navigate">
          {navigateItems.map(item => (
            <CommandItem key={item.value} value={item.value}>
              <item.icon />
              {item.label}
            </CommandItem>
          ))}
        </CommandGroup>
        <CommandSeparator />
        <CommandGroup heading="Actions">
          {quickItems.map(item => (
            <CommandItem key={item.value} value={item.value}>
              {item.icon ? <item.icon /> : <SearchIcon />}
              {item.label}
              {item.shortcut ? <CommandShortcut>{item.shortcut}</CommandShortcut> : null}
            </CommandItem>
          ))}
        </CommandGroup>
      </CommandList>
    </Command>
  ),
};

export const WithShortcuts: Story = {
  args: {},
  render: () => (
    <Command className="w-[24rem] border">
      <CommandInput placeholder="Jump to…" />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>
        <CommandGroup heading="Suggestions">
          <CommandItem value="palette">
            Open command palette
            <KbdGroup>
              <Kbd>⌘</Kbd>
              <Kbd>K</Kbd>
            </KbdGroup>
          </CommandItem>
          <CommandItem value="toggle-sidebar">
            Toggle sidebar
            <KbdGroup>
              <Kbd>⌘</Kbd>
              <Kbd>B</Kbd>
            </KbdGroup>
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </Command>
  ),
};
