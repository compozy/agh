import type { Meta, StoryObj } from "@storybook/react-vite";

import { Input, NativeSelect, NativeSelectOption } from "@agh/ui";
import { FieldRow } from "../field-row";

const meta: Meta<typeof FieldRow> = {
  title: "components/custom/FieldRow",
  component: FieldRow,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Settings field row with label, optional description, and a control slot. Two layouts: stacked (label-on-top) and two-column (label gutter on the left). Use inside settings panels and editor sheets.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="w-[480px] bg-background p-4 flex flex-col gap-4">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Stacked layout — default for settings panels.
 */
export const Stacked: Story = {
  args: {},
  render: () => (
    <>
      <FieldRow
        label="Workspace name"
        description="Visible to operators in the rail."
        control={<Input defaultValue="personal" />}
        htmlFor="workspace-name"
      />
      <FieldRow
        label="Default provider"
        control={
          <NativeSelect>
            <NativeSelectOption value="anthropic">Anthropic Claude</NativeSelectOption>
            <NativeSelectOption value="openai">OpenAI</NativeSelectOption>
            <NativeSelectOption value="local">Local llama.cpp</NativeSelectOption>
          </NativeSelect>
        }
        htmlFor="default-provider"
      />
    </>
  ),
};

/**
 * Two-column layout — label gutter aligns multiple field rows.
 */
export const TwoColumn: Story = {
  args: {},
  render: () => (
    <>
      <FieldRow
        layout="two-column"
        label="Display name"
        description="Shown on receipts."
        control={<Input defaultValue="agh-runtime" />}
      />
      <FieldRow
        layout="two-column"
        label="Region"
        control={
          <NativeSelect defaultValue="us-east-1">
            <NativeSelectOption value="us-east-1">us-east-1</NativeSelectOption>
            <NativeSelectOption value="eu-west-1">eu-west-1</NativeSelectOption>
          </NativeSelect>
        }
      />
    </>
  ),
};
