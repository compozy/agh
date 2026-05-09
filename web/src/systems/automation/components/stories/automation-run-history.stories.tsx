import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { automationRunFixtures } from "@/systems/automation/mocks";

import { AutomationRunHistory } from "../automation-run-history";

const meta: Meta<typeof AutomationRunHistory> = {
  title: "systems/automation/AutomationRunHistory",
  component: AutomationRunHistory,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <AutomationRunHistory error={null} isLoading={false} runs={automationRunFixtures} />
      </div>
    </CenteredSurface>
  ),
};

export const Empty: Story = {
  args: {},
  render: () => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <AutomationRunHistory error={null} isLoading={false} runs={[]} />
      </div>
    </CenteredSurface>
  ),
};

export const Loading: Story = {
  args: {},
  render: () => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <AutomationRunHistory error={null} isLoading runs={[]} />
      </div>
    </CenteredSurface>
  ),
};

export const ErrorState: Story = {
  args: {},
  render: () => (
    <CenteredSurface>
      <div className="w-full max-w-3xl">
        <AutomationRunHistory
          error={new globalThis.Error("Failed to load automation runs")}
          isLoading={false}
          runs={[]}
        />
      </div>
    </CenteredSurface>
  ),
};
