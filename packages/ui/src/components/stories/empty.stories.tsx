import type { Meta, StoryObj } from "@storybook/react-vite";
import { InboxIcon, PlusIcon, SearchIcon } from "lucide-react";

import { Button } from "../button";
import { Empty } from "../empty";

const meta: Meta<typeof Empty> = {
  title: "components/ui/Empty",
  component: Empty,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Empty state — icon well + muted title + optional description + optional action(s).",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const TitleOnly: Story = {
  render: () => (
    <div className="w-[420px]">
      <Empty title="Nothing here yet" />
    </div>
  ),
};

export const WithDescription: Story = {
  render: () => (
    <div className="w-[420px]">
      <Empty
        icon={SearchIcon}
        title="Nothing matches"
        description="Adjust the search or clear filters to see more results."
      />
    </div>
  ),
};

export const WithAction: Story = {
  render: () => (
    <div className="w-[420px]">
      <Empty
        icon={InboxIcon}
        title="Your inbox is empty"
        description="Approval requests, failed runs, and blockers appear here."
        action={
          <Button size="sm" type="button">
            <PlusIcon className="size-3.5" />
            New task
          </Button>
        }
      />
    </div>
  ),
};
