import type { Meta, StoryObj } from "@storybook/react-vite";

import { Eyebrow } from "../custom/eyebrow";
import type { PillTone } from "../custom/pill";

const meta: Meta<typeof Eyebrow> = {
  title: "components/custom/Eyebrow",
  component: Eyebrow,
  args: {},
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component: "Canonical mono uppercase metadata label for dense AGH runtime surfaces.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const TONES: PillTone[] = ["neutral", "accent", "success", "warning", "danger", "info"];

/** Default metadata label. */
export const Default: Story = {
  args: {
    children: "Queue health",
  },
};

/** Weight variants keep the same mono scale and tracking token. */
export const Weights: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-4">
      <Eyebrow weight="medium">medium</Eyebrow>
      <Eyebrow weight="semibold">semibold</Eyebrow>
    </div>
  ),
};

/** Signal tones are text-only and preserve the flat surface model. */
export const Tones: Story = {
  args: {},
  render: () => (
    <div className="grid gap-2">
      {TONES.map(tone => (
        <Eyebrow key={tone} tone={tone}>
          {tone}
        </Eyebrow>
      ))}
    </div>
  ),
};
