import { Check, Package } from "lucide-react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { WireCard, WireCardBody, WireCardFoot, WireCardHead } from "@agh/ui";

const meta: Meta<typeof WireCard> = {
  title: "ui/WireCard",
  component: WireCard,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Bordered protocol card — mirrors `.wire-card` (head/body/foot) in `docs/design/web-inspiration/styles/app.css`. Used to embed wire-protocol payloads (recipes, receipts, capabilities) inside message threads.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Recipe: Story = {
  render: () => (
    <WireCard>
      <WireCardHead>
        <Package className="size-3" />
        <span>Recipe</span>
        <span className="text-[color:var(--color-text-primary)]">rag.embed.bulk · v2</span>
      </WireCardHead>
      <WireCardBody>
        <div className="grid grid-cols-2 gap-3 text-[11px]">
          <div>
            <span className="text-[color:var(--color-text-tertiary)]">accepts:</span>{" "}
            <span className="text-[color:var(--color-text-primary)]">urls</span>
          </div>
          <div>
            <span className="text-[color:var(--color-text-tertiary)]">emits:</span>{" "}
            <span className="text-[color:var(--color-text-primary)]">vector_ids</span>
          </div>
        </div>
      </WireCardBody>
      <WireCardFoot>
        <button
          className="rounded-[4px] border border-[color:var(--color-divider)] px-2 py-1 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
          type="button"
        >
          Call recipe
        </button>
        <button
          className="rounded-[4px] border border-[color:var(--color-divider)] px-2 py-1 font-mono text-[10px] uppercase tracking-[0.06em] text-[color:var(--color-text-secondary)] hover:text-[color:var(--color-text-primary)]"
          type="button"
        >
          View schema
        </button>
      </WireCardFoot>
    </WireCard>
  ),
};

export const InlineReceipt: Story = {
  render: () => (
    <WireCard inline>
      <Check className="size-3 text-[color:var(--color-success)]" />
      <span className="font-mono text-[11px] text-[color:var(--color-text-primary)]">
        receipt for direct#8471 · ok
      </span>
      <span className="ml-auto font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
        latency 212ms
      </span>
    </WireCard>
  ),
};
