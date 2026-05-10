import type { Meta, StoryObj } from "@storybook/react-vite";

import { Pill } from "@agh/ui";

import { CenteredSurface } from "@/storybook/story-layout";

import { SettingsStatusLine } from "../settings-status-line";

const meta: Meta<typeof SettingsStatusLine> = {
  title: "systems/settings/SettingsStatusLine",
  component: SettingsStatusLine,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Compact status row that pairs the daemon `ConnectionIndicator` with optional contextual chips (counts, source labels, etc.). Each chip is separated by a token `·` glyph in `--subtle`.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Connected — daemon is reachable; status line shows the canonical chip cluster.
 */
export const Connected: Story = {
  args: {
    status: "connected",
    daemonLabel: "Daemon",
    items: [
      <Pill key="rev" mono tone="neutral">
        rev e2c91
      </Pill>,
      <Pill key="ws" mono tone="info">
        workspace · launch
      </Pill>,
      <span key="loaded" className="text-xs text-(--muted)">
        4 providers loaded
      </span>,
    ],
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <SettingsStatusLine {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * Connecting — pulsing connecting state on the indicator.
 */
export const Connecting: Story = {
  args: {
    status: "connecting",
    daemonLabel: "Daemon",
    items: [
      <span key="msg" className="text-xs text-(--subtle)">
        Resyncing settings…
      </span>,
    ],
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <SettingsStatusLine {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * Disconnected — danger-tone indicator with a single message item.
 */
export const Disconnected: Story = {
  args: {
    status: "disconnected",
    daemonLabel: "Daemon",
    items: [
      <span key="msg" className="text-xs text-(--danger)">
        Reconnect from the topbar before saving changes.
      </span>,
    ],
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <SettingsStatusLine {...args} />
      </div>
    </CenteredSurface>
  ),
};
