import type { Meta, StoryObj } from "@storybook/react-vite";

import { RuntimeConnectionIndicator } from "../connection-indicator";

const meta: Meta<typeof RuntimeConnectionIndicator> = {
  title: "app/runtime/RuntimeConnectionIndicator",
  component: RuntimeConnectionIndicator,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Single owner of the daemon connection LED Three states: `success` solid (reachable + recent activity), `success` pulse (reachable + degraded heartbeat), `danger` solid (unreachable). The sidebar footer is the only mount point; the rail no longer carries a duplicate LED.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const SuccessSolid: Story = {
  args: { status: "connected", degraded: false },
  parameters: {
    docs: {
      description: { story: "Daemon reachable and within the heartbeat window." },
    },
  },
};

export const SuccessPulse: Story = {
  args: { status: "connected", degraded: true },
  parameters: {
    docs: {
      description: {
        story: "Daemon reachable but the heartbeat is degraded (SSE drop, stale counts).",
      },
    },
  },
};

export const Connecting: Story = {
  args: { status: "connecting", degraded: false },
  parameters: {
    docs: {
      description: { story: "Initial connection in flight; pulses until resolved." },
    },
  },
};

export const DangerSolid: Story = {
  args: { status: "disconnected", degraded: false },
  parameters: {
    docs: {
      description: { story: "Daemon unreachable; user action required." },
    },
  },
};

export const Error: Story = {
  args: { status: "error", degraded: false },
  parameters: {
    docs: {
      description: { story: "Daemon errored; surfaces a danger tone with the error label." },
    },
  },
};

export const RailDotOnly: Story = {
  args: { status: "connected", degraded: false, dotOnly: true },
  parameters: {
    docs: {
      description: {
        story:
          "Collapsed rail mode renders only the dot. Reserved for future iterations — the runtime shell still mounts the footer variant by default.",
      },
    },
  },
};
