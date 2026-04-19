import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";
import {
  assistantMessageFixture,
  diffMessageFixture,
  streamingAssistantMessageFixture,
  systemMessageFixture,
  userMessageFixture,
} from "@/systems/session/mocks";
import type { UIMessage } from "@/systems/session/types";

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
      <div className="mx-auto w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] py-6">
        {children}
      </div>
    </StorySurface>
  );
}

const userTurnFixture: UIMessage = {
  ...userMessageFixture,
  content:
    "Find the event mapper that groups tool calls by turn and extract the grouping logic into a pure helper.",
};

const agentTurnFixture: UIMessage = {
  ...assistantMessageFixture,
  thinking: undefined,
  content:
    "I can see two candidates — `packages/runtime/src/events/map.ts` and `packages/runtime/src/session/stream.ts`. The grouping lives in `stream.ts`.",
};

export const User: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="claude-code" message={userTurnFixture} />
    </BubbleFrame>
  ),
};

export const Agent: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="claude-code" message={agentTurnFixture} />
    </BubbleFrame>
  ),
};

export const AgentStreaming: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="claude-code" message={streamingAssistantMessageFixture} />
    </BubbleFrame>
  ),
};

export const AgentWithThinking: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="claude-code" message={assistantMessageFixture} />
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

export const Diff: Story = {
  render: () => (
    <BubbleFrame>
      <MessageBubble agentName="claude-code" message={diffMessageFixture} />
    </BubbleFrame>
  ),
};
