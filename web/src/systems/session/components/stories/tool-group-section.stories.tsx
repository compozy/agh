import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import {
  bashToolMessageFixture,
  editToolMessageFixture,
  errorToolMessageFixture,
  runningBashToolMessageFixture,
  searchToolMessageFixture,
} from "@/systems/session/mocks";

import { ToolGroupSection } from "../tool-group-section";

const meta: Meta<typeof ToolGroupSection> = {
  title: "systems/session/ToolGroupSection",
  component: ToolGroupSection,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ToolGroupFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl">{children}</div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => (
    <ToolGroupFrame>
      <ToolGroupSection
        tools={[
          runningBashToolMessageFixture,
          bashToolMessageFixture,
          editToolMessageFixture,
          searchToolMessageFixture,
        ]}
      />
    </ToolGroupFrame>
  ),
};

export const WithError: Story = {
  render: () => (
    <ToolGroupFrame>
      <ToolGroupSection tools={[bashToolMessageFixture, errorToolMessageFixture]} />
    </ToolGroupFrame>
  ),
};

export const SingleTool: Story = {
  render: () => (
    <ToolGroupFrame>
      <ToolGroupSection tools={[bashToolMessageFixture]} />
    </ToolGroupFrame>
  ),
};
