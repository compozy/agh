import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { markdownFixture } from "@/systems/session/mocks";

import { MessageMarkdown } from "../message-markdown";

const meta: Meta<typeof MessageMarkdown> = {
  title: "systems/session/MessageMarkdown",
  component: MessageMarkdown,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <CenteredSurface>
      <div className="prose prose-invert max-w-3xl rounded-2xl border border-(--line) bg-(--canvas-soft) p-6 text-sm">
        <MessageMarkdown content={markdownFixture} />
      </div>
    </CenteredSurface>
  ),
};
