import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story-meta";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/index",
    "Settings root route redirecting to the default General section inside the real app shell."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Settings root route resolving to the default General section.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/general"),
  render: () => <StorybookWorkspaceSetup />,
};
