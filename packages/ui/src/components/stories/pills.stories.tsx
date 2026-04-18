import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { Pill, Pills } from "../pills";

const meta: Meta<typeof Pill> = {
  title: "ui/Pills",
  component: Pill,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "`Pill` renders a static semantic tag. `Pills` renders the segmented toggle group from the mock — `items` + controlled `value`/`onChange`.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const AllVariants: Story = {
  render: () => (
    <div className="flex flex-wrap gap-2">
      <Pill>Neutral</Pill>
      <Pill variant="accent">Running</Pill>
      <Pill variant="success">Live</Pill>
      <Pill variant="warning">Pending</Pill>
      <Pill variant="danger">Error</Pill>
      <Pill variant="info">Info</Pill>
    </div>
  ),
};

export const Sizes: Story = {
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      <Pill size="sm">sm · tag</Pill>
      <Pill size="md">md · filter</Pill>
      <Pill variant="accent" size="sm">
        sm · accent
      </Pill>
      <Pill variant="success" size="md">
        md · success
      </Pill>
    </div>
  ),
};

function PillsHarness() {
  const [value, setValue] = useState<"list" | "kanban" | "dashboard" | "inbox">("list");
  return (
    <Pills
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

export const Segmented: Story = {
  render: () => <PillsHarness />,
};

export const SegmentedSelection: Story = {
  render: () => <PillsHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const list = await canvas.findByTestId("mode-list");
    const kanban = await canvas.findByTestId("mode-kanban");

    await expect(list).toHaveAttribute("aria-selected", "true");
    await expect(kanban).toHaveAttribute("aria-selected", "false");

    await userEvent.click(kanban);
    await waitFor(() => expect(kanban).toHaveAttribute("aria-selected", "true"));
    await expect(list).toHaveAttribute("aria-selected", "false");
  },
};

export const DisabledItem: Story = {
  render: () => {
    return (
      <Pills
        value="list"
        onChange={() => {}}
        items={[
          { value: "list", label: "List" },
          { value: "kanban", label: "Kanban", disabled: true },
        ]}
      />
    );
  },
};
