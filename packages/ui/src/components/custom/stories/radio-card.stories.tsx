import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { CpuIcon, ServerIcon, ZapIcon } from "lucide-react";

import { Pill } from "@agh/ui";
import { RadioCard } from "../radio-card";

const meta: Meta<typeof RadioCard> = {
  title: "components/custom/RadioCard",
  component: RadioCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Single radio choice rendered as a card. Selected state shifts the border to `--accent` and the surface to `--accent-tint` so the chosen option reads at a glance without raising the surface step.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[480px] bg-background p-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

interface Provider {
  id: "anthropic" | "openai" | "local";
  title: string;
  description: string;
  icon: typeof CpuIcon;
  badge?: string;
}

const PROVIDERS: ReadonlyArray<Provider> = [
  {
    id: "anthropic",
    title: "Anthropic Claude",
    description: "Bound to ~/.claude credentials.",
    icon: CpuIcon,
    badge: "Default",
  },
  { id: "openai", title: "OpenAI", description: "Bound to OPENAI_API_KEY env.", icon: ZapIcon },
  {
    id: "local",
    title: "Local llama.cpp",
    description: "Connect to a local model server.",
    icon: ServerIcon,
  },
];

/**
 * Three options stacked in a column; selection state animates only the border + tint.
 */
export const ProviderPicker: Story = {
  args: {},
  render: () => {
    const [selected, setSelected] = useState<Provider["id"]>("anthropic");
    return (
      <div role="radiogroup" aria-label="Pick a provider" className="flex flex-col gap-2">
        {PROVIDERS.map(provider => (
          <RadioCard
            key={provider.id}
            title={provider.title}
            description={provider.description}
            icon={provider.icon}
            selected={selected === provider.id}
            onSelect={() => setSelected(provider.id)}
            badge={provider.badge ? <Pill tone="accent">{provider.badge}</Pill> : undefined}
          />
        ))}
      </div>
    );
  },
};

/**
 * Static selected vs unselected for visual diffing.
 */
export const SelectedVsRest: Story = {
  args: {},
  render: () => (
    <div role="radiogroup" aria-label="Sample" className="flex flex-col gap-2">
      <RadioCard
        title="Selected option"
        description="Border + tint applied."
        icon={CpuIcon}
        selected
        onSelect={() => undefined}
      />
      <RadioCard
        title="Resting option"
        description="Hover lifts the border to `--line-strong`."
        icon={ZapIcon}
        selected={false}
        onSelect={() => undefined}
      />
    </div>
  ),
};
