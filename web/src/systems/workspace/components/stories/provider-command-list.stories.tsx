import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { CenteredSurface } from "@/storybook/story-layout";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

import { ProviderCommandList } from "../provider-command-list";

const providerOptions = workspaceDetailFixture.providers ?? [];

const meta: Meta<typeof ProviderCommandList> = {
  title: "systems/workspace/ProviderCommandList",
  component: ProviderCommandList,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Command list grouped by harness for selecting a session provider.",
      },
    },
  },
  decorators: [
    Story => (
      <CenteredSurface>
        <div className="w-full max-w-md rounded-lg border border-(--line) bg-(--canvas-soft)">
          <Story />
        </div>
      </CenteredSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Providers are grouped by harness with the selected option marked.
 */
export const Default: Story = {
  args: {
    options: providerOptions,
    isSelected: option => option.name === "codex",
    onSelect: fn(),
  },
};

/**
 * Empty state appears inside the command menu when no providers are available.
 */
export const Empty: Story = {
  args: {
    options: [],
    isSelected: () => false,
    onSelect: fn(),
    emptyState: "No provider commands are configured.",
  },
};
