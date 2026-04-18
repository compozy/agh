import type { Meta, StoryObj } from "@storybook/react-vite";
import { InboxIcon, PlusIcon, ServerCrashIcon } from "lucide-react";
import { Button } from "@agh/ui";

import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "../empty";

const meta: Meta<typeof Empty> = {
  title: "components/ui/Empty",
  component: Empty,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Empty-state block for lists, dashboards, and dialogs. Compose Media + Header + Content to guide the next action.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const WithIconTitleAction: Story = {
  args: {},
  render: () => (
    <div className="w-[32rem] rounded-xl border border-dashed">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <InboxIcon />
          </EmptyMedia>
          <EmptyTitle>No sessions yet</EmptyTitle>
          <EmptyDescription>
            Start a run to stream agent events. Sessions appear here once they are created.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button>
            <PlusIcon />
            New session
          </Button>
        </EmptyContent>
      </Empty>
    </div>
  ),
};

export const ErrorState: Story = {
  args: {},
  render: () => (
    <div className="w-[32rem] rounded-xl border border-dashed">
      <Empty>
        <EmptyHeader>
          <EmptyMedia variant="icon">
            <ServerCrashIcon />
          </EmptyMedia>
          <EmptyTitle>Daemon unreachable</EmptyTitle>
          <EmptyDescription>
            Could not open the UDS socket. Start AGH with <code>agh daemon</code> and retry.
          </EmptyDescription>
        </EmptyHeader>
        <EmptyContent>
          <Button variant="outline">Retry</Button>
        </EmptyContent>
      </Empty>
    </div>
  ),
};

export const Minimal: Story = {
  args: {},
  render: () => (
    <div className="w-[24rem] rounded-xl border border-dashed">
      <Empty>
        <EmptyHeader>
          <EmptyTitle>No results</EmptyTitle>
          <EmptyDescription>Try widening the time range filter.</EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  ),
};
