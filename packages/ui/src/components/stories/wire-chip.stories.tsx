import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { KIND_DOT_COLORS, WireChip } from "@agh/ui";

const meta: Meta<typeof WireChip> = {
  title: "ui/WireChip",
  component: WireChip,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Free-floating filter chip — mirrors `.wire-chip` in `docs/design/web-inspiration/styles/app.css`. For a contained segmented toggle, use `Pills` instead.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const KINDS = ["say", "greet", "direct", "receipt", "recipe", "trace", "whois"] as const;

export const Default: Story = {
  args: { children: "say" },
};

export const KindFilterRow: Story = {
  render: () => {
    const [active, setActive] = useState<string>("all");
    return (
      <div className="flex flex-wrap items-center gap-1.5">
        <WireChip active={active === "all"} onClick={() => setActive("all")}>
          all
        </WireChip>
        {KINDS.map(kind => (
          <WireChip
            active={active === kind}
            dotColor={KIND_DOT_COLORS[kind]}
            key={kind}
            onClick={() => setActive(kind)}
          >
            {kind}
          </WireChip>
        ))}
      </div>
    );
  },
};
