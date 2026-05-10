import type { Meta, StoryObj } from "@storybook/react-vite";
import { InboxIcon, OctagonXIcon } from "lucide-react";

import { Button } from "@agh/ui";
import { RouteState } from "../route-state";

const meta: Meta<typeof RouteState> = {
  title: "components/custom/RouteState",
  component: RouteState,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Unified loading / empty / error route surface (TechSpec composite). Stays flat on `--canvas-soft` with `--line` ring; the icon well sits on `--canvas` to read as a recessed step. Loading uses `aria-live=polite` automatically.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[640px] bg-background p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Loading mode — minimal, accessible status text. No spinner by default.
 */
export const Loading: Story = {
  args: {},
  render: () => <RouteState mode="loading" title="Loading sessions" />,
};

/**
 * Empty state with an icon, message, and a primary action.
 */
export const Empty: Story = {
  args: {},
  render: () => (
    <RouteState
      mode="empty"
      icon={InboxIcon}
      title="No sessions yet"
      message="Sessions appear here as soon as an agent starts working."
      action={<Button size="sm">Start a session</Button>}
    />
  ),
};

/**
 * Error mode with a `cause` block rendered in mono — never auto-renders stack traces.
 */
export const Error: Story = {
  args: {},
  render: () => (
    <RouteState
      mode="error"
      icon={OctagonXIcon}
      title="Session resume failed"
      message="The daemon could not reattach to the prior ACP transport."
      cause="ECONNREFUSED 127.0.0.1:2123"
      action={
        <Button size="sm" variant="outline">
          Retry
        </Button>
      }
    />
  ),
};
