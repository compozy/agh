import type { Meta, StoryObj } from "@storybook/react-vite";

import { Field, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectSeparator,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const meta: Meta<typeof Select> = {
  title: "components/ui/Select",
  component: Select,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Base UI powered select with keyboard navigation. Compose Trigger + Value + Content + Item tuples.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const agents = [
  { value: "claude", label: "Claude Code" },
  { value: "codex", label: "Codex CLI" },
  { value: "gemini", label: "Gemini CLI" },
];

export const WithLabelAndHelper: Story = {
  args: {},
  render: () => (
    <div className="w-[18rem]">
      <Field>
        <FieldLabel htmlFor="select-agent">Agent driver</FieldLabel>
        <Select defaultValue={agents[0].value}>
          <SelectTrigger id="select-agent" className="w-full">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {agents.map(agent => (
              <SelectItem key={agent.value} value={agent.value}>
                {agent.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <FieldDescription>Default driver used when a session does not pin one.</FieldDescription>
      </Field>
    </div>
  ),
};

export const ErrorState: Story = {
  args: {},
  render: () => (
    <div className="w-[18rem]">
      <Field data-invalid>
        <FieldLabel htmlFor="select-error">Agent driver</FieldLabel>
        <Select>
          <SelectTrigger
            id="select-error"
            aria-invalid
            aria-describedby="select-error-message"
            className="w-full"
          >
            <SelectValue placeholder="Pick an agent" />
          </SelectTrigger>
          <SelectContent>
            {agents.map(agent => (
              <SelectItem key={agent.value} value={agent.value}>
                {agent.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <FieldError id="select-error-message">Agent driver is required.</FieldError>
      </Field>
    </div>
  ),
};

export const Grouped: Story = {
  args: {},
  render: () => (
    <div className="w-[18rem]">
      <Select defaultValue="claude">
        <SelectTrigger className="w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectLabel>Local</SelectLabel>
            <SelectItem value="claude">Claude Code</SelectItem>
            <SelectItem value="codex">Codex CLI</SelectItem>
          </SelectGroup>
          <SelectSeparator />
          <SelectGroup>
            <SelectLabel>Remote</SelectLabel>
            <SelectItem value="gemini">Gemini CLI</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  ),
};
