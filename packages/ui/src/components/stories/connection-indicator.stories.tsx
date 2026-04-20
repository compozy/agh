import type { Meta, StoryObj } from "@storybook/react-vite";

import { ConnectionIndicator, type ConnectionStatus } from "../connection-indicator";

const meta: Meta<typeof ConnectionIndicator> = {
  title: "ui/ConnectionIndicator",
  component: ConnectionIndicator,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "StatusDot + mono label composite for daemon / socket connection state. `reconnecting` pulses the dot unless the user prefers reduced motion.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const STATES: ConnectionStatus[] = ["connected", "disconnected", "reconnecting"];

export const Connected: Story = {
  args: { status: "connected" },
};

export const Disconnected: Story = {
  args: { status: "disconnected" },
};

export const Reconnecting: Story = {
  args: { status: "reconnecting" },
};

export const AllStates: Story = {
  render: () => (
    <div className="flex flex-col items-start gap-3">
      {STATES.map(state => (
        <ConnectionIndicator key={state} status={state} />
      ))}
    </div>
  ),
};

export const CustomLabel: Story = {
  args: {
    status: "connected",
    label: "Live",
  },
};
