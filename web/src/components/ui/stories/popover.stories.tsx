import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button, Input, Label } from "@agh/ui";

import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from "@/components/ui/popover";

const meta: Meta<typeof Popover> = {
  title: "components/ui/Popover",
  component: Popover,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Floating surface for contextual actions and quick forms. Base UI positioning handles alignment and collisions.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <Popover>
      <PopoverTrigger render={<Button variant="outline">Open popover</Button>} />
      <PopoverContent>
        <PopoverHeader>
          <PopoverTitle>Quick rename</PopoverTitle>
          <PopoverDescription>Changes apply only to this row.</PopoverDescription>
        </PopoverHeader>
        <div className="grid gap-2">
          <Label htmlFor="popover-name">Label</Label>
          <Input id="popover-name" defaultValue="daemon-us-east" />
        </div>
        <Button size="sm">Save</Button>
      </PopoverContent>
    </Popover>
  ),
};

export const RightAligned: Story = {
  args: {},
  render: () => (
    <Popover>
      <PopoverTrigger render={<Button variant="ghost">Show context</Button>} />
      <PopoverContent align="end" side="right">
        <PopoverHeader>
          <PopoverTitle>Run metadata</PopoverTitle>
        </PopoverHeader>
        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt className="text-muted-foreground">Session</dt>
          <dd className="font-mono">s-0194</dd>
          <dt className="text-muted-foreground">Agent</dt>
          <dd>claude-code</dd>
          <dt className="text-muted-foreground">Duration</dt>
          <dd>8m 21s</dd>
        </dl>
      </PopoverContent>
    </Popover>
  ),
};
