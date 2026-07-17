import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";

import { TasksFanOutRunsCard } from "../tasks-fan-out-runs-card";

const meta: Meta<typeof TasksFanOutRunsCard> = {
  title: "systems/tasks/components/TasksFanOutRunsCard",
  component: TasksFanOutRunsCard,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Creates designated sibling runs with one explicit Network participation policy shared by every run.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[700px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/** The open fan-out dialog starts Local and exposes the shared participation controls. */
export const DialogOpen: Story = {
  args: {
    onFanOut: async () => ({ designation_group_id: "desig_storybook", runs: [] }),
  },
  play: async ({ canvasElement }) => {
    await userEvent.click(within(canvasElement).getByTestId("tasks-fan-out-runs-trigger"));
    const documentBody = within(canvasElement.ownerDocument.body);
    await expect(documentBody.findByTestId("tasks-fan-out-runs-dialog")).resolves.toBeVisible();
    await expect(documentBody.getByTestId("tasks-fan-out-network-mode")).toHaveValue("local");
  },
};
