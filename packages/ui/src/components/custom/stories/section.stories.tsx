import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button } from "../../button";
import { Pill } from "../pill";
import { Section } from "../section";

const meta: Meta<typeof Section> = {
  title: "components/custom/Section",
  component: Section,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component: "Section shell with mono eyebrow + optional right-aligned slot + body.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Basic: Story = {
  args: {},
  render: () => (
    <div className="w-[520px]">
      <Section label="Routes">
        <ul className="divide-y divide-[color:var(--color-divider)] text-sm text-[color:var(--color-text-secondary)]">
          <li className="py-2">/runtime/sessions</li>
          <li className="py-2">/runtime/memory</li>
          <li className="py-2">/runtime/skills</li>
        </ul>
      </Section>
    </div>
  ),
};

export const WithRightSlot: Story = {
  args: {},
  render: () => (
    <div className="w-[520px]">
      <Section
        label="Recent runs"
        right={
          <>
            <Pill tone="success">Live</Pill>
            <Button size="xs" variant="outline" type="button">
              View all
            </Button>
          </>
        }
      >
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          Dense operational rows go here: replay, inspect, or open detail.
        </p>
      </Section>
    </div>
  ),
};

export const WithNote: Story = {
  args: {},
  render: () => (
    <div className="w-[520px]">
      <Section label="Runtime" note="Read-only daemon state" divided>
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          Use note for compact section context without creating a local wrapper.
        </p>
      </Section>
    </div>
  ),
};

export const BodyOnly: Story = {
  args: {},
  render: () => (
    <div className="w-[520px]">
      <Section>
        <p className="text-sm text-[color:var(--color-text-secondary)]">
          Section with no eyebrow; the surrounding layout supplies the heading.
        </p>
      </Section>
    </div>
  ),
};
