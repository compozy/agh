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
          "Protocol kind marker (`say`, `greet`, `direct`, `receipt`, `recipe`, `whois`, `trace`). Uppercase mono, transparent surface with neutral border + colored 7px wire-dot — mirrors `.intent-badge` + `.wire-dot` in `docs/design/web-inspiration/styles/app.css`.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const KINDS = ["say", "greet", "direct", "receipt", "recipe", "trace", "whois"] as const;

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
