import type { Meta, StoryObj } from "@storybook/react-vite";
import { Label } from "@agh/ui";

import { Field, FieldDescription, FieldLabel } from "../field";
import { Switch } from "../switch";

const meta: Meta<typeof Switch> = {
  title: "components/ui/Switch",
  component: Switch,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Binary on/off control. Use inside a horizontal Field for settings rows or pair with a standalone Label for toolbars.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-3">
      <Switch id="switch-default" defaultChecked />
      <Label htmlFor="switch-default">Stream responses</Label>
    </div>
  ),
};

export const InField: Story = {
  args: {},
  render: () => (
    <div className="w-[24rem]">
      <Field orientation="horizontal">
        <Switch id="switch-field" defaultChecked />
        <FieldLabel htmlFor="switch-field">Auto-compact conversation</FieldLabel>
        <FieldDescription>
          Summarize long runs once they exceed 80% of the context window.
        </FieldDescription>
      </Field>
    </div>
  ),
};

export const Small: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-3">
      <Switch id="switch-sm" size="sm" />
      <Label htmlFor="switch-sm" className="text-sm">
        Compact mode
      </Label>
    </div>
  ),
};

export const Disabled: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-3">
      <Switch id="switch-disabled" disabled defaultChecked />
      <Label htmlFor="switch-disabled">Workspace lock (set by admin)</Label>
    </div>
  ),
};
