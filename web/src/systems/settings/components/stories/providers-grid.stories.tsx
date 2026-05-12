import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { settingsProviderFixtures } from "@/systems/settings/mocks";

import { ProvidersGrid } from "../providers-grid";

const meta: Meta<typeof ProvidersGrid> = {
  title: "systems/settings/ProvidersGrid",
  component: ProvidersGrid,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Responsive grid of `ProviderCard`s for the Providers settings tab. Single column on small viewports, two on `md`, three on `xl`. Cards inherit the warm canvas-soft surface + warm-line ring.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default — full provider catalog as exposed by the daemon settings adapter.
 */
export const Default: Story = {
  args: {
    providers: settingsProviderFixtures,
    onOpen: fn(),
  },
  render: args => (
    <PanelSurface className="px-6 py-6">
      <ProvidersGrid {...args} />
    </PanelSurface>
  ),
};

/**
 * Empty — no providers configured. Verifies the grid collapses cleanly to zero
 * children without leaving phantom gap rows.
 */
export const Empty: Story = {
  args: {
    providers: [],
    onOpen: fn(),
  },
  render: args => (
    <PanelSurface className="px-6 py-6">
      <ProvidersGrid {...args} />
    </PanelSurface>
  ),
};
