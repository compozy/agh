import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import {
  bashToolMessageFixture,
  errorToolMessageFixture,
  runningBashToolMessageFixture,
} from "@/systems/session/mocks";

import { ToolCallCard } from "../tool-call-card";

const meta: Meta<typeof ToolCallCard> = {
  title: "systems/session/ToolCallCard",
  component: ToolCallCard,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ToolCallCardFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-2xl">{children}</div>
    </CenteredSurface>
  );
}

export const Running: Story = {
  render: () => (
    <ToolCallCardFrame>
      <ToolCallCard message={runningBashToolMessageFixture} />
    </ToolCallCardFrame>
  ),
};

export const Done: Story = {
  render: () => (
    <ToolCallCardFrame>
      <ToolCallCard message={bashToolMessageFixture} />
    </ToolCallCardFrame>
  ),
};

export const Error: Story = {
  render: () => (
    <ToolCallCardFrame>
      <ToolCallCard message={errorToolMessageFixture} />
    </ToolCallCardFrame>
  ),
};
