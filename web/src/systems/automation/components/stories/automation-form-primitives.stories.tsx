import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import {
  AutomationCheckbox,
  AutomationField,
  AutomationFormSection,
  AutomationInput,
  AutomationTextarea,
} from "../automation-form-primitives";

const meta: Meta<typeof AutomationFormSection> = {
  title: "systems/automation/AutomationFormPrimitives",
  component: AutomationFormSection,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <CenteredSurface>
      <div className="w-[38rem] rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
        <AutomationFormSection description="Primitives used by the automation forms." title="Core">
          <div className="grid gap-4 md:grid-cols-2">
            <AutomationField label="Name">
              <AutomationInput defaultValue="nightly-docs" />
            </AutomationField>
            <AutomationField label="Agent">
              <AutomationInput defaultValue="reviewer" />
            </AutomationField>
          </div>
          <AutomationField hint="Supports multiline prompts." label="Prompt">
            <AutomationTextarea defaultValue="Summarize the Storybook rollout progress." />
          </AutomationField>
          <AutomationCheckbox
            checked
            description="Allow this automation item to dispatch without additional toggles."
            label="Enabled"
            onCheckedChange={() => undefined}
          />
        </AutomationFormSection>
      </div>
    </CenteredSurface>
  ),
};

export const ValidationState: Story = {
  render: () => (
    <CenteredSurface>
      <div className="w-[38rem] rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-4">
        <AutomationFormSection
          description="A dense error state using the same primitive building blocks."
          title="Validation"
        >
          <AutomationField hint="Name is required before saving." label="Name">
            <AutomationInput
              aria-invalid
              className="border-[color:var(--color-danger)]"
              placeholder="Required"
              value=""
              onChange={() => undefined}
            />
          </AutomationField>
          <p className="text-sm text-[color:var(--color-danger)]">Name is required.</p>
        </AutomationFormSection>
      </div>
    </CenteredSurface>
  ),
};
