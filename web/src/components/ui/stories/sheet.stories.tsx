import type { Meta, StoryObj } from "@storybook/react-vite";
import { Button, Input, Label } from "@agh/ui";

import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "../sheet";

const meta: Meta<typeof Sheet> = {
  title: "components/ui/Sheet",
  component: Sheet,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Side sheet for supplementary editing. Choose a side via the Content `side` prop and compose with @agh/ui form primitives.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <Sheet>
      <SheetTrigger render={<Button variant="outline">Open sheet</Button>} />
      <SheetContent>
        <SheetHeader>
          <SheetTitle>Edit agent preset</SheetTitle>
          <SheetDescription>Adjust the defaults applied to new sessions.</SheetDescription>
        </SheetHeader>
        <div className="grid gap-4 px-4">
          <div className="grid gap-2">
            <Label htmlFor="sheet-model">Model</Label>
            <Input id="sheet-model" defaultValue="claude-opus-4-7" />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="sheet-temp">Temperature</Label>
            <Input id="sheet-temp" defaultValue="0.3" />
          </div>
        </div>
        <SheetFooter>
          <SheetClose render={<Button variant="ghost">Cancel</Button>} />
          <SheetClose render={<Button>Save preset</Button>} />
        </SheetFooter>
      </SheetContent>
    </Sheet>
  ),
};

export const LeftSide: Story = {
  args: {},
  render: () => (
    <Sheet>
      <SheetTrigger render={<Button variant="outline">Open from left</Button>} />
      <SheetContent side="left">
        <SheetHeader>
          <SheetTitle>Filters</SheetTitle>
          <SheetDescription>Scope the dashboard to a specific workspace.</SheetDescription>
        </SheetHeader>
        <div className="px-4 text-sm text-muted-foreground">
          Filters stay in place until cleared so long runs stay visible.
        </div>
        <SheetFooter>
          <SheetClose render={<Button>Apply</Button>} />
        </SheetFooter>
      </SheetContent>
    </Sheet>
  ),
};
