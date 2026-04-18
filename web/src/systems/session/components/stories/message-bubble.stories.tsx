import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";
import {
  assistantMessageFixture,
  systemMessageFixture,
  userMessageFixture,
} from "@/systems/session/mocks";

import { MessageBubble } from "../message-bubble";

const meta: Meta<typeof MessageBubble> = {
  title: "systems/session/MessageBubble",
  component: MessageBubble,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function BubbleFrame({ children }: { children: React.ReactNode }) {
  return (
    <StorySurface>
      <div className="mx-auto max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] py-6">
        {children}
      </div>
    </StorySurface>
  );
}

export const User: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="codex-agent" message={userMessageFixture} />
    </BubbleFrame>
  ),
};

export const Assistant: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="codex-agent" message={assistantMessageFixture} />
    </BubbleFrame>
  ),
};

export const System: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="system" message={systemMessageFixture} />
    </BubbleFrame>
  ),
};
