import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

import { ProviderCommandSelect } from "../provider-command-select";

const providerOptions = workspaceDetailFixture.providers ?? [];

const meta: Meta<typeof ProviderCommandSelect> = {
  title: "systems/workspace/ProviderCommandSelect",
  component: ProviderCommandSelect,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Popover trigger wrapping the provider command list for setup and session forms.",
      },
    },
  },
  decorators: [
    Story => (
      <CenteredSurface>
        <div className="w-full max-w-md">
          <Story />
        </div>
      </CenteredSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Selected provider shows display name plus harness.
 */
export const Selected: Story = {
  args: {},
  render: () => {
    function Harness() {
      const [value, setValue] = useState<string | null>("codex");
      return <ProviderCommandSelect options={providerOptions} value={value} onChange={setValue} />;
    }
    return <Harness />;
  },
};

/**
 * Empty value keeps the placeholder visible until a provider is chosen.
 */
export const Empty: Story = {
  args: {},
  render: () => {
    function Harness() {
      const [value, setValue] = useState<string | null>(null);
      return (
        <ProviderCommandSelect
          options={providerOptions}
          value={value}
          onChange={setValue}
          placeholder="Choose runtime provider"
        />
      );
    }
    return <Harness />;
  },
};

/**
 * Disabled state is used while setup is resolving workspace context.
 */
export const Disabled: Story = {
  args: {
    options: providerOptions,
    value: "claude",
    disabled: true,
    onChange: () => undefined,
  },
};
