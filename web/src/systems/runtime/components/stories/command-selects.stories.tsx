import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";

import { ModelCommandSelect } from "../model-command-select";
import { ProviderCommandSelect } from "../provider-command-select";
import { ReasoningCommandSelect } from "../reasoning-command-select";
import type { ModelSelectOption, ProviderSelectOption, ReasoningSelectOption } from "../../types";

const providerOptions: ProviderSelectOption[] = [
  { name: "codex", display_name: "Codex", harness: "openai", runtime_provider: "openai" },
  { name: "claude", display_name: "Claude", harness: "anthropic", runtime_provider: "anthropic" },
  { name: "local-acp" },
];

const modelOptions: ModelSelectOption[] = [
  {
    id: "gpt-5.4",
    label: "GPT-5.4",
    availability: { label: "Live", tone: "success", state: "available_live" },
  },
  {
    id: "gpt-5.4-mini",
    label: "GPT-5.4 Mini",
    availability: { label: "Stale", tone: "warning", state: "available_stale" },
  },
];

const reasoningOptions: ReasoningSelectOption[] = [
  { value: "low", label: "Low", source: "catalog" },
  { value: "medium", label: "Medium", source: "catalog" },
  { value: "high", label: "High", source: "acp" },
  { value: "xhigh", label: "Extra high", source: "acp" },
];

const meta: Meta<typeof ProviderCommandSelect> = {
  title: "systems/runtime/CommandSelects",
  component: ProviderCommandSelect,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Canonical provider, model, and reasoning command selectors shared by the session and agent create flows.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[420px] p-6">
        <div className="grid max-w-md gap-3">
          <Story />
        </div>
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Provider selector groups options by harness and previews the selected
 * provider's display name plus harness in the trigger.
 */
export const Provider: Story = {
  args: {},
  render: () => {
    function Harness() {
      const [value, setValue] = useState<string | null>("codex");
      return (
        <ProviderCommandSelect
          options={providerOptions}
          value={value}
          onChange={setValue}
          triggerTestId="story-provider-select"
        />
      );
    }
    return <Harness />;
  },
};

/**
 * Model selector exposes the provider default action, catalog availability
 * badges, and a custom-typed-model escape hatch.
 */
export const Model: StoryObj<typeof ModelCommandSelect> = {
  args: {},
  render: () => (
    <ModelCommandSelect
      options={modelOptions}
      defaultModel="gpt-5.4"
      value="gpt-5.4-mini"
      onChange={fn()}
      triggerTestId="story-model-select"
    />
  ),
};

/**
 * Reasoning selector renders provider-backed effort levels and a disabled
 * provider-default hint when the model does not advertise reasoning effort.
 */
export const Reasoning: StoryObj<typeof ReasoningCommandSelect> = {
  args: {},
  render: () => (
    <>
      <ReasoningCommandSelect
        options={reasoningOptions}
        value="high"
        onChange={fn()}
        triggerTestId="story-reasoning-select"
      />
      <ReasoningCommandSelect
        options={[]}
        value=""
        onChange={fn()}
        disabled
        disabledHint="Selected model does not advertise reasoning effort"
      />
    </>
  ),
};
