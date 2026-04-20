import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { useEffect, useState } from "react";
import { expect, waitFor, within } from "storybook/test";

import type { ConnectionStatus as ConnectionStatusType } from "@agh/ui";

import { storybookMswParameters } from "@/storybook/msw";
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
    ...storybookMswParameters({
      daemon: [
        http.get("/api/observe/health", async () => {
          await delay(150);
          return HttpResponse.json({ error: "daemon offline" }, { status: 503 });
        }),
      ],
    }),
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

interface StatusTransitionProps {
  sequence: ConnectionStatusType[];
  intervalMs: number;
}

function StatusTransitionDriver({ sequence, intervalMs }: StatusTransitionProps) {
  const [index, setIndex] = useState(0);

  useEffect(() => {
    if (index >= sequence.length - 1) {
      return;
    }
    const handle = window.setTimeout(() => setIndex(current => current + 1), intervalMs);
    return () => window.clearTimeout(handle);
  }, [index, intervalMs, sequence.length]);

  return (
    <CenteredSurface>
      <div data-testid="connection-status-driver" data-current-status={sequence[index]}>
        <ConnectionStatus status={sequence[index]} />
      </div>
    </CenteredSurface>
  );
}

/**
 * Storybook interaction test — drive ConnectionStatus through
 * connected → reconnecting → disconnected and assert that the
 * underlying ConnectionIndicator tone + label updates each step.
 */
export const StatusTransitions: Story = {
  tags: ["play-fn"],
  render: () => (
    <StatusTransitionDriver
      intervalMs={50}
      sequence={["connected", "reconnecting", "disconnected"]}
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const driver = await canvas.findByTestId("connection-status-driver");

    await waitFor(() => {
      expect(driver).toHaveAttribute("data-current-status", "connected");
    });
    expect(within(driver).getByText("Connected")).toBeInTheDocument();

    await waitFor(() => {
      expect(driver).toHaveAttribute("data-current-status", "reconnecting");
    });
    expect(within(driver).getByText("Reconnecting")).toBeInTheDocument();

    await waitFor(() => {
      expect(driver).toHaveAttribute("data-current-status", "disconnected");
    });
    expect(within(driver).getByText("Disconnected")).toBeInTheDocument();
  },
};
