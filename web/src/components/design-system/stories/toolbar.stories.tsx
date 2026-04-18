import type { Meta, StoryObj } from "@storybook/react-vite";

import { Pill } from "../pill";
import { Toolbar, ToolbarAction, ToolbarGroup, ToolbarSearch } from "../toolbar";

import { StoryFrame } from "./story-frame";

const meta: Meta<typeof Toolbar> = {
  title: "components/design-system/Toolbar",
  component: Toolbar,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "A compact control band pattern for filters, search, and primary actions across AGH command surfaces.",
      },
    },
  },
  decorators: [
    Story => (
      <StoryFrame className="max-w-5xl">
        <Story />
      </StoryFrame>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default toolbar composition with filter pills, search, and a primary action.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <Toolbar>
      <ToolbarGroup>
        <Pill emphasis="strong" kind="filter" tone="amber">
          Foundations
        </Pill>
        <Pill kind="filter">Panels</Pill>
        <Pill kind="filter">Signals</Pill>
      </ToolbarGroup>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <ToolbarSearch placeholder="Search surfaces..." />
        <ToolbarAction>Build surface</ToolbarAction>
      </div>
    </Toolbar>
  ),
};

/**
 * Dense filter state for trays that need tighter control clusters.
 */
export const DenseFilters: Story = {
  args: {},
  render: () => (
    <Toolbar>
      <ToolbarGroup>
        <Pill emphasis="strong" kind="filter" tone="amber">
          Live
        </Pill>
        <Pill kind="tag">Runtime</Pill>
        <Pill kind="tag">Queues</Pill>
        <Pill emphasis="strong" kind="state" tone="green">
          Healthy
        </Pill>
      </ToolbarGroup>
      <ToolbarAction>Inspect queue</ToolbarAction>
    </Toolbar>
  ),
};
