import type { Meta, StoryObj } from "@storybook/react-vite";
import { Hash } from "lucide-react";

import { PanelSurface } from "@/storybook/story-layout";

import { NetworkEmptyState } from "../network-empty-state";

const meta: Meta<typeof NetworkEmptyState> = {
  title: "systems/network/NetworkEmptyState",
  component: NetworkEmptyState,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <NetworkEmptyState
        actionLabel="Create Channel"
        description="Create your first channel to coordinate multiple agents inside the active workspace."
        icon={Hash}
        onAction={() => undefined}
        testId="network-empty-story"
        title="No channels yet"
      />
    </PanelSurface>
  ),
};
