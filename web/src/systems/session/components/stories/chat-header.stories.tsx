import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { primarySessionFixture } from "@/systems/session/mocks";

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

export const Default: Story = {
  render: () => (
    <CenteredSurface>
      <div className="w-full max-w-4xl overflow-hidden rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)]">
        <ChatHeader
          onResume={() => undefined}
          onStop={() => undefined}
          session={primarySessionFixture}
          workspaceName="agh2"
        />
      </div>
    </CenteredSurface>
  ),
};
