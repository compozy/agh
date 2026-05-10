import type { Meta, StoryObj } from "@storybook/react-vite";
import { Plus } from "lucide-react";
import { fn } from "storybook/test";

import { Button } from "@agh/ui";

import { CenteredSurface } from "@/storybook/story-layout";

import type { RestartBannerState } from "../settings-restart-banner";
import { SettingsPageActions } from "../settings-page-actions";

const meta: Meta<typeof SettingsPageActions> = {
  title: "systems/settings/SettingsPageActions",
  component: SettingsPageActions,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Topbar action cluster pushed by every settings sub-page. Renders an optional secondary slot followed by the standard 'Restart daemon' button. The button respects the restart state machine (`isTriggerPending` + `isPolling`).",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function makeRestart(overrides: Partial<RestartBannerState> = {}): RestartBannerState {
  return {
    isVisible: false,
    isRestartRequired: false,
    isPolling: false,
    isSuccessful: false,
    isFailed: false,
    operationId: null,
    status: null,
    activeSessionCount: 0,
    trigger: fn(),
    isTriggerPending: false,
    triggerError: null,
    dismiss: fn(),
    ...overrides,
  };
}

/**
 * Default — idle state. Restart button is enabled and ready.
 */
export const Default: Story = {
  args: {
    slug: "general",
    restart: makeRestart(),
  },
  render: args => (
    <CenteredSurface>
      <div className="flex w-full max-w-3xl items-center justify-end">
        <SettingsPageActions {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * Restarting — restart button is disabled and labeled with the in-flight state.
 */
export const Restarting: Story = {
  args: {
    slug: "providers",
    restart: makeRestart({ isPolling: true, status: "draining sessions" }),
  },
  render: args => (
    <CenteredSurface>
      <div className="flex w-full max-w-3xl items-center justify-end">
        <SettingsPageActions {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * WithSecondaryAction — verifies the optional secondary action slot renders
 * before the canonical restart button.
 */
export const WithSecondaryAction: Story = {
  args: {
    slug: "providers",
    restart: makeRestart(),
    secondaryAction: (
      <Button type="button" variant="outline" size="sm" data-testid="settings-page-add-provider">
        <Plus className="size-3.5" />
        Add provider
      </Button>
    ),
  },
  render: args => (
    <CenteredSurface>
      <div className="flex w-full max-w-3xl items-center justify-end">
        <SettingsPageActions {...args} />
      </div>
    </CenteredSurface>
  ),
};
