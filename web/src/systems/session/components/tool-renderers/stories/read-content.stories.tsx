import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { readToolMessageFixture, truncatedReadToolMessageFixture } from "@/systems/session/mocks";

import { ReadContent } from "../read-content";

const meta: Meta<typeof ReadContent> = {
  title: "systems/session/tool-renderers/ReadContent",
  component: ReadContent,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ReadFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-4">
        {children}
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => (
    <ReadFrame>
      <ReadContent message={readToolMessageFixture} />
    </ReadFrame>
  ),
};

export const Truncated: Story = {
  render: () => (
    <ReadFrame>
      <ReadContent message={truncatedReadToolMessageFixture} />
    </ReadFrame>
  ),
};
