import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

import { CenteredSurface } from "@/storybook/story-layout";
import { useDaemonHealth } from "@/systems/daemon";

import { ConnectionStatus } from "../connection-status";

const meta: Meta<typeof ConnectionStatus> = {
  title: "systems/daemon/ConnectionStatus",
  component: ConnectionStatus,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ConnectionStatusFromQuery() {
  const { connectionStatus } = useDaemonHealth();

  return (
    <CenteredSurface>
      <ConnectionStatus status={connectionStatus} />
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => <ConnectionStatusFromQuery />,
};

export const Disconnected: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/api/observe/health", async () => {
          await delay(150);
          return HttpResponse.json({ error: "daemon offline" }, { status: 503 });
        }),
      ],
    },
  },
  render: () => <ConnectionStatusFromQuery />,
};

export const Reconnecting: Story = {
  render: () => (
    <CenteredSurface>
      <ConnectionStatus status="reconnecting" />
    </CenteredSurface>
  ),
};
