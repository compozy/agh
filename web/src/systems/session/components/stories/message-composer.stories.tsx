import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import { MessageComposer } from "../message-composer";

const meta: Meta<typeof MessageComposer> = {
  title: "systems/session/MessageComposer",
  component: MessageComposer,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ComposerFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]">
        {children}
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={() => undefined} />
    </ComposerFrame>
  ),
};

export const Disabled: Story = {
  render: () => (
    <ComposerFrame>
      <MessageComposer disabled onSend={() => undefined} />
    </ComposerFrame>
  ),
};
