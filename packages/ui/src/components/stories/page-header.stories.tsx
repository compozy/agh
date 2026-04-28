import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ListChecksIcon, PlusIcon, SparklesIcon } from "lucide-react";

import { Button } from "../button";
import { PageHeader } from "../page-header";
import { PillGroup } from "../pill-group";

const meta: Meta<typeof PageHeader> = {
  title: "ui/PageHeader",
  component: PageHeader,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Top-of-page header — icon + title + count badge on the left, segmented `PillGroup` controls in the middle, meta/actions on the right.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Basic: Story = {
  render: () => <PageHeader title="Settings" icon={SparklesIcon} />,
};

export const WithCount: Story = {
  render: () => <PageHeader title="Tasks" icon={ListChecksIcon} count={128} />,
};

export const WithControlsAndMeta: Story = {
  render: () => {
    function Harness() {
      const [mode, setMode] = useState<"list" | "kanban" | "dashboard" | "inbox">("list");
      return (
        <PageHeader
          title="Tasks"
          icon={ListChecksIcon}
          count={42}
          controls={
            <PillGroup
              value={mode}
              onChange={setMode}
              items={[
                { value: "list", label: "List" },
                { value: "kanban", label: "Kanban" },
                { value: "dashboard", label: "Dashboard" },
                { value: "inbox", label: "Inbox", badge: 2 },
              ]}
            />
          }
          meta={
            <Button size="sm" type="button">
              <PlusIcon className="size-3.5" />
              Task
            </Button>
          }
        />
      );
    }
    return <Harness />;
  },
};
