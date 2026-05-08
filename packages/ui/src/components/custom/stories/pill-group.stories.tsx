import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { PillGroup } from "../pill-group";

const meta: Meta<typeof PillGroup> = {
  title: "components/custom/PillGroup",
  component: PillGroup,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Segmented toggle track. Controlled via `items` + `value` + `onChange`. Replaces the legacy segmented pills toggle.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function PillGroupHarness() {
  const [value, setValue] = useState<"list" | "kanban" | "dashboard" | "inbox">("list");
  return (
    <PillGroup
      value={value}
      onChange={setValue}
      items={[
        { value: "list", label: "List", testId: "mode-list" },
        { value: "kanban", label: "Kanban", testId: "mode-kanban" },
        { value: "dashboard", label: "Dashboard", testId: "mode-dashboard" },
        { value: "inbox", label: "Inbox", badge: 3, testId: "mode-inbox" },
      ]}
      aria-label="Task views"
    />
  );
}

export const Default: Story = {
  render: () => <PillGroupHarness />,
};

export const Selection: Story = {
  render: () => <PillGroupHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const list = await canvas.findByTestId("mode-list");
    const kanban = await canvas.findByTestId("mode-kanban");

    await expect(list).toHaveAttribute("aria-pressed", "true");
    await expect(kanban).toHaveAttribute("aria-pressed", "false");

    await userEvent.click(kanban);
    await waitFor(() => expect(kanban).toHaveAttribute("aria-pressed", "true"));
    await expect(list).toHaveAttribute("aria-pressed", "false");
  },
};

export const SizeSm: Story = {
  render: () => {
    const [value, setValue] = useState<"a" | "b" | "c">("a");
    return (
      <PillGroup
        value={value}
        onChange={setValue}
        size="sm"
        items={[
          { value: "a", label: "Alpha" },
          { value: "b", label: "Beta" },
          { value: "c", label: "Gamma" },
        ]}
      />
    );
  },
};

export const DisabledItem: Story = {
  render: () => (
    <PillGroup
      value="list"
      onChange={() => {}}
      items={[
        { value: "list", label: "List" },
        { value: "kanban", label: "Kanban", disabled: true },
      ]}
    />
  ),
};
