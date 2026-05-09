import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { ListChecksIcon, PlusIcon, SparklesIcon } from "lucide-react";

import { Button } from "../../button";
import { PageHeader } from "../page-header";
import { PillGroup } from "../pill-group";

const meta: Meta<typeof PageHeader> = {
  title: "components/custom/PageHeader",
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
  args: {},
  render: () => <PageHeader title="Settings" icon={SparklesIcon} />,
};

export const WithCount: Story = {
  args: {},
  render: () => <PageHeader title="Tasks" icon={ListChecksIcon} count={128} />,
};

export const WithControlsAndMeta: Story = {
  args: {},
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

export const WithBreadcrumb: Story = {
  args: {},
  render: () => (
    <PageHeader
      breadcrumb="Settings / Providers"
      title="Providers"
      meta={
        <Button size="sm" type="button">
          <PlusIcon className="size-3.5" />
          Provider
        </Button>
      }
    />
  ),
};

export const WithSubtitleAndStatus: Story = {
  args: {},
  render: () => (
    <PageHeader
      title="Network"
      subtitle="Runtime transport and delivery limits."
      statusRow={
        <>
          <span>Daemon online</span>
          <span>NATS enabled</span>
          <span>3 channels</span>
        </>
      }
    />
  ),
};

export const AllSlots: Story = {
  args: {},
  render: () => (
    <PageHeader
      breadcrumb="Settings / Automation"
      title="Automation"
      icon={SparklesIcon}
      count={6}
      subtitle="Manage jobs, triggers, and execution limits."
      controls={
        <PillGroup
          value="runtime"
          onChange={() => undefined}
          items={[{ value: "runtime", label: "Runtime" }]}
        />
      }
      statusRow={
        <>
          <span>Scheduler enabled</span>
          <span>Next fire in 4m</span>
        </>
      }
      meta={
        <Button size="sm" type="button">
          <PlusIcon className="size-3.5" />
          Job
        </Button>
      }
    />
  ),
};
