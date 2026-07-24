import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/settings/routes/SettingsLayouts",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Window-manager defaults, scoped layout profiles, and the active workspace document rendered through the daemon-authoritative Settings shell.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/** Global behavior plus the active workspace's revisioned declarative layout. */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/layouts"),
  render: () => <StorybookWorkspaceSetup />,
};
