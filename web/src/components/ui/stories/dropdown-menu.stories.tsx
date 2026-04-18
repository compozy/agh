import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button } from "@agh/ui";

import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuShortcut,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const meta: Meta<typeof DropdownMenu> = {
  title: "components/ui/DropdownMenu",
  component: DropdownMenu,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Action menu anchored to a trigger. Mix plain items, checkbox groups, radios, and shortcuts from a single composition.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const agentItems = [
  { value: "rename", label: "Rename session" },
  { value: "duplicate", label: "Duplicate session" },
  { value: "export", label: "Export transcript" },
] as const;

export const Default: Story = {
  args: {},
  render: () => (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline">Session actions</Button>} />
      <DropdownMenuContent>
        <DropdownMenuLabel>Session</DropdownMenuLabel>
        {agentItems.map(item => (
          <DropdownMenuItem key={item.value}>{item.label}</DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive">
          Delete
          <DropdownMenuShortcut>⌘⌫</DropdownMenuShortcut>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  ),
};

function StatefulMenu() {
  const [lane, setLane] = useState("critical");
  const [autoScroll, setAutoScroll] = useState(true);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="outline">View options</Button>} />
      <DropdownMenuContent>
        <DropdownMenuLabel>Severity lane</DropdownMenuLabel>
        <DropdownMenuRadioGroup value={lane} onValueChange={setLane}>
          <DropdownMenuRadioItem value="critical">Critical</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="warning">Warning</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="info">Info</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuCheckboxItem checked={autoScroll} onCheckedChange={setAutoScroll}>
          Auto-scroll transcript
        </DropdownMenuCheckboxItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export const SelectionStates: Story = {
  args: {},
  render: () => <StatefulMenu />,
};
