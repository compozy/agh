import type { Meta, StoryObj } from "@storybook/react-vite";

import { KindChip } from "../kind-chip";

const meta: Meta<typeof KindChip> = {
  title: "ui/KindChip",
  component: KindChip,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Protocol kind marker (`greet`, `whois`, `say`, `direct`, `capability`, `receipt`, `trace`). Lowercase mono, accent-tint background, 5px radius — DESIGN.md §4.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const KINDS = ["greet", "whois", "say", "direct", "capability", "receipt", "trace"] as const;

export const Default: Story = {
  args: {
    kind: "greet",
  },
};

export const AllProtocolKinds: Story = {
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {KINDS.map(kind => (
        <KindChip key={kind} kind={kind} />
      ))}
    </div>
  ),
};

export const InlineWithCopy: Story = {
  render: () => (
    <p className="text-sm text-[color:var(--color-text-secondary)]">
      Messages of kind <KindChip kind="say" /> are forwarded by the router to any peer subscribed to
      the channel.
    </p>
  ),
};
