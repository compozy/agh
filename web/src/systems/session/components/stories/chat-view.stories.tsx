import { useEffect, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";
import {
  assistantMessageFixture,
  markdownFixture,
  streamingAssistantMessageFixture,
  uiMessageFixtures,
  userMessageFixture,
} from "@/systems/session/mocks";
import type { UIMessage } from "@/systems/session/types";

import { ChatView } from "../chat-view";

const meta: Meta<typeof ChatView> = {
  title: "systems/session/ChatView",
  component: ChatView,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ChatViewFrame({
  children,
  errorMessage,
}: {
  children: React.ReactNode;
  errorMessage?: string;
}) {
  return (
    <StorySurface className="min-h-[680px] space-y-4">
      {errorMessage ? (
        <div className="mx-auto max-w-4xl rounded-xl border border-[color:var(--color-danger)] bg-[color:var(--color-danger-tint)] px-4 py-3 text-sm text-[color:var(--color-danger)]">
          {errorMessage}
        </div>
      ) : null}
      <div className="mx-auto h-[560px] w-full max-w-4xl overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]">
        {children}
      </div>
    </StorySurface>
  );
}

function StreamingChatViewStory() {
  const [messages, setMessages] = useState<UIMessage[]>([
    userMessageFixture,
    streamingAssistantMessageFixture,
  ]);
  const [isStreaming, setIsStreaming] = useState(true);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      setMessages([
        userMessageFixture,
        {
          ...assistantMessageFixture,
          content: markdownFixture,
        },
      ]);
      setIsStreaming(false);
    }, 1200);

    return () => window.clearTimeout(timer);
  }, []);

  return <ChatView agentName="codex-agent" isStreaming={isStreaming} messages={messages} />;
}

export const Default: Story = {
  render: () => (
    <ChatViewFrame>
      <ChatView agentName="codex-agent" isStreaming={false} messages={uiMessageFixtures} />
    </ChatViewFrame>
  ),
};

export const Streaming: Story = {
  render: () => (
    <ChatViewFrame>
      <StreamingChatViewStory />
    </ChatViewFrame>
  ),
};

export const Error: Story = {
  render: () => (
    <ChatViewFrame errorMessage="Transcript stream disconnected. Showing the last persisted messages.">
      <ChatView agentName="codex-agent" isStreaming={false} messages={uiMessageFixtures} />
    </ChatViewFrame>
  ),
};
