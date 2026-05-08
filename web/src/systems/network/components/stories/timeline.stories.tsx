import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import { networkThreadMessagesFixture } from "@/systems/network/mocks";
import { Timeline } from "@/systems/network/components/timeline";

const meta: Meta<typeof Timeline> = {
  title: "systems/network/Timeline",
  component: Timeline,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Channel-pane timeline composing full message rows, collapsed continuations, system events, date pills, and the New divider per `_design.md` §5.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px] p-0">
      <Timeline messages={networkThreadMessagesFixture} />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px] p-0">
      <Timeline isLoading messages={[]} />
    </PanelSurface>
  ),
};

export const Empty: Story = {
  render: () => (
    <PanelSurface className="min-h-[640px] p-0">
      <Timeline
        emptyState={<p className="text-xs text-(--color-text-tertiary)">Thread has no replies.</p>}
        messages={[]}
      />
    </PanelSurface>
  ),
};

export const NewDividerStory: Story = {
  name: "With New Divider",
  render: () => (
    <PanelSurface className="min-h-[640px] p-0">
      <Timeline lastReadAt="2026-04-17T18:01:00Z" messages={networkThreadMessagesFixture} />
    </PanelSurface>
  ),
};
