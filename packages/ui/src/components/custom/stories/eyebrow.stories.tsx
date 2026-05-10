import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  Eyebrow,
  type EyebrowCase,
  type EyebrowSize,
  type EyebrowTone,
  type EyebrowWeight,
} from "../eyebrow";

const meta: Meta<typeof Eyebrow> = {
  title: "components/custom/Eyebrow",
  component: Eyebrow,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Canonical eyebrow primitive. `case='upper'` renders JetBrains Mono with `--text-eyebrow` (11 px) and `--tracking-mono` (0.06em). `case='sentence'` falls back to Inter at 12 px. `size` exposes the three token-aligned scales (`eyebrow`, `badge`, `micro`) and `tone` covers the full signal palette plus subtle/strong neutrals. Every other surface MUST go through this component — see DESIGN.md §3.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const SIZES: EyebrowSize[] = ["eyebrow", "badge", "micro"];
const TONES: EyebrowTone[] = [
  "neutral",
  "muted",
  "subtle",
  "strong",
  "accent",
  "success",
  "warning",
  "danger",
  "info",
];
const WEIGHTS: EyebrowWeight[] = ["medium", "semibold"];
const CASES: EyebrowCase[] = ["upper", "sentence"];

export const Default: Story = {
  args: {
    children: "Active sessions",
    case: "upper",
  },
};

export const Matrix: Story = {
  parameters: {
    docs: {
      description: {
        story:
          "Full case × size × tone × weight matrix used as a visual contract. Everything below is reachable via the public component API — never inline these classes.",
      },
    },
  },
  render: () => (
    <div className="flex flex-col gap-8">
      {CASES.map(caseVariant => (
        <section key={caseVariant} className="flex flex-col gap-4">
          <h3 className="text-[12px] font-medium tracking-[-0.005em] text-(--fg-strong)">
            case=&quot;{caseVariant}&quot;
          </h3>
          <div className="grid grid-cols-[120px_repeat(3,1fr)] items-baseline gap-x-6 gap-y-3">
            <span className="text-[11px] text-(--subtle)">tone / size</span>
            {SIZES.map(size => (
              <span key={size} className="text-[11px] text-(--subtle)">
                {size}
              </span>
            ))}
            {TONES.map(tone => (
              <Row key={tone} caseVariant={caseVariant} tone={tone} />
            ))}
          </div>
        </section>
      ))}
      <section className="flex flex-col gap-3">
        <h3 className="text-[12px] font-medium tracking-[-0.005em] text-(--fg-strong)">Weights</h3>
        <div className="flex items-baseline gap-6">
          {WEIGHTS.map(weight => (
            <Eyebrow key={weight} case="upper" weight={weight} tone="neutral">
              weight={weight}
            </Eyebrow>
          ))}
        </div>
      </section>
    </div>
  ),
};

function Row({ caseVariant, tone }: { caseVariant: EyebrowCase; tone: EyebrowTone }) {
  return (
    <>
      <span className="text-[11px] text-(--subtle)">{tone}</span>
      {SIZES.map(size => (
        <Eyebrow key={size} case={caseVariant} tone={tone} size={size}>
          {tone}
        </Eyebrow>
      ))}
    </>
  );
}
