import type { Meta, StoryObj } from "@storybook/react-vite";
import { Bot, ShieldCheck } from "lucide-react";

import { FormSection, Input, Label, Textarea } from "@agh/ui";

const meta: Meta<typeof FormSection> = {
  title: "components/custom/FormSection",
  component: FormSection,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Editable-surface container — 18/20 padding (comfortable) or 14/16 (compact), --radius-lg corners, --canvas-soft background, no border. Head is 13/510/-0.008em with optional leading icon and right-aligned 11 px eyebrow.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[720px] bg-background p-6">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/** Comfortable density — the default for full-page forms. */
export const Comfortable: Story = {
  args: {},
  render: () => (
    <FormSection title="Scope" icon={ShieldCheck} rightLabel="Required">
      <div className="flex flex-col gap-2">
        <Label htmlFor="scope">Visibility</Label>
        <Input id="scope" placeholder="workspace" />
      </div>
      <div className="flex flex-col gap-2">
        <Label htmlFor="notes">Notes</Label>
        <Textarea id="notes" placeholder="Optional context for the operator" />
      </div>
    </FormSection>
  ),
};

/** Compact density — for dialog hosts and OptionCard wrappers. */
export const Compact: Story = {
  args: {},
  render: () => (
    <FormSection
      title="Owner"
      size="compact"
      icon={Bot}
      description="Who runs this task when no override is set."
    >
      <Input placeholder="agent_session" />
    </FormSection>
  ),
};

/** No icon / no right label — verifies head spacing collapses cleanly. */
export const HeadOnly: Story = {
  args: {},
  render: () => (
    <FormSection title="Schedule">
      <Input placeholder="cron expression" />
    </FormSection>
  ),
};
