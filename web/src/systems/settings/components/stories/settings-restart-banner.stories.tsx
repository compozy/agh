import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, userEvent, within } from "storybook/test";

import type { RestartBannerState } from "../settings-restart-banner";
import { SettingsRestartBanner } from "../settings-restart-banner";

type RestartOverrides = Partial<RestartBannerState>;

function makeRestart(overrides: RestartOverrides = {}): RestartBannerState {
  return {
    isVisible: true,
    isRestartRequired: false,
    isPolling: false,
    isSuccessful: false,
    isFailed: false,
    operationId: null,
    status: null,
    failureReason: undefined,
    activeSessionCount: 0,
    trigger: fn(),
    isTriggerPending: false,
    triggerError: null,
    dismiss: fn(),
    ...overrides,
  };
}

const meta: Meta<typeof SettingsRestartBanner> = {
  title: "systems/settings/SettingsRestartBanner",
  component: SettingsRestartBanner,
  parameters: {
    layout: "padded",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Warning: Story = {
  args: {
    slug: "general",
    restart: makeRestart({ isRestartRequired: true }),
  },
};

export const Polling: Story = {
  args: {
    slug: "general",
    restart: makeRestart({
      isRestartRequired: true,
      isPolling: true,
      status: "stopping",
      operationId: "op_abcdef",
    }),
  },
};

export const Success: Story = {
  args: {
    slug: "general",
    restart: makeRestart({ isRestartRequired: true, isSuccessful: true }),
  },
};

export const Failure: Story = {
  args: {
    slug: "general",
    restart: makeRestart({
      isRestartRequired: true,
      isFailed: true,
      failureReason: "helper exited non-zero",
    }),
  },
};

/**
 * Interaction test: cycles warning → polling → success tones and asserts the
 * Dismiss button only appears in success/failure states.
 */
export const ToneTransitionInteraction: Story = {
  tags: ["play-fn"],
  render: () => {
    function Harness() {
      type Phase = "warning" | "polling" | "success";
      const [phase, setPhase] = useState<Phase>("warning");

      const restart = makeRestart({
        isRestartRequired: phase === "warning",
        isPolling: phase === "polling",
        isSuccessful: phase === "success",
        status: phase === "polling" ? "stopping" : null,
        trigger: () => setPhase("polling"),
        dismiss: () => setPhase("warning"),
      });

      return (
        <div className="flex flex-col gap-3">
          <div className="flex gap-2 text-xs">
            <button
              type="button"
              data-testid="banner-phase-warning"
              onClick={() => setPhase("warning")}
            >
              warning
            </button>
            <button
              type="button"
              data-testid="banner-phase-polling"
              onClick={() => setPhase("polling")}
            >
              polling
            </button>
            <button
              type="button"
              data-testid="banner-phase-success"
              onClick={() => setPhase("success")}
            >
              success
            </button>
          </div>
          <SettingsRestartBanner slug="general" restart={restart} />
        </div>
      );
    }
    return <Harness />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const banner = canvas.getByTestId("settings-page-general-restart-banner");
    await expect(banner).toHaveAttribute("data-tone", "warning");
    await expect(
      canvas.queryByTestId("settings-page-general-restart-banner-dismiss")
    ).not.toBeInTheDocument();

    await userEvent.click(canvas.getByTestId("banner-phase-polling"));
    await expect(canvas.getByTestId("settings-page-general-restart-banner")).toHaveAttribute(
      "data-tone",
      "info"
    );
    await expect(
      canvas.queryByTestId("settings-page-general-restart-banner-dismiss")
    ).not.toBeInTheDocument();

    await userEvent.click(canvas.getByTestId("banner-phase-success"));
    await expect(canvas.getByTestId("settings-page-general-restart-banner")).toHaveAttribute(
      "data-tone",
      "success"
    );
    await expect(
      canvas.getByTestId("settings-page-general-restart-banner-dismiss")
    ).toBeInTheDocument();
  },
};
