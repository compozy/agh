import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { primarySessionFixture } from "@/systems/session/mocks";
import type { SessionPayload } from "@/systems/session/types";

import { ChatHeader } from "../chat-header";

const meta: Meta<typeof ChatHeader> = {
  title: "systems/session/ChatHeader",
  component: ChatHeader,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function Frame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-(--color-divider) bg-(--color-canvas)">
        {children}
      </div>
    </CenteredSurface>
  );
}

function withState(state: SessionPayload["state"]): SessionPayload {
  return { ...primarySessionFixture, state };
}

export const Default: Story = {
  render: () => (
    <Frame>
      <ChatHeader
        onDelete={() => undefined}
        onResume={() => undefined}
        onStop={() => undefined}
        session={primarySessionFixture}
        workspaceName="agh-core"
      />
    </Frame>
  ),
};

export const Starting: Story = {
  render: () => (
    <Frame>
      <ChatHeader
        onDelete={() => undefined}
        onResume={() => undefined}
        onStop={() => undefined}
        session={withState("starting")}
        workspaceName="agh-core"
      />
    </Frame>
  ),
};

export const Stopped: Story = {
  render: () => (
    <Frame>
      <ChatHeader
        onDelete={() => undefined}
        onResume={() => undefined}
        onStop={() => undefined}
        session={withState("stopped")}
        workspaceName="agh-core"
      />
    </Frame>
  ),
};
