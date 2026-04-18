import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
  createRouteStoryMeta,
} from "@/storybook/route-story";

const meta: Meta<typeof StorybookRouteCanvas> = {
  ...createRouteStoryMeta(
    "routes/app/settings/index",
    "Default settings landing route rendered through the real app shell and settings navigation."
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Settings landing page with the left navigation and placeholder content.
 */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings"),
  render: () => <StorybookWorkspaceSetup />,
};
