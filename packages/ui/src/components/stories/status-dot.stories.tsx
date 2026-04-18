import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, within } from "storybook/test";

import { StatusDot, type StatusDotTone } from "../status-dot";

const meta: Meta<typeof StatusDot> = {
  title: "ui/StatusDot",
  component: StatusDot,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Tinted signal dot — semantic tone + optional `pulse` loop. Mirrors DESIGN.md §4 status indicators.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const TONES: StatusDotTone[] = ["success", "warning", "danger", "info", "accent", "neutral"];

const TONE_TO_COLOR: Record<StatusDotTone, string> = {
  success: "var(--color-success)",
  warning: "var(--color-warning)",
  danger: "var(--color-danger)",
  info: "var(--color-info)",
  accent: "var(--color-accent)",
  neutral: "var(--color-text-tertiary)",
};

export const Default: Story = {
  args: {
    tone: "success",
  },
};

export const Tones: Story = {
  render: () => (
    <div className="flex items-center gap-6" data-testid="status-dot-tones">
      {TONES.map(tone => (
        <div key={tone} className="flex items-center gap-2" data-testid={`tone-${tone}`}>
          <StatusDot tone={tone} />
          <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]">
            {tone}
          </span>
        </div>
      ))}
    </div>
  ),
};

export const ToneCycleInteraction: Story = {
  render: () => (
    <div className="flex items-center gap-6" data-testid="status-dot-tones">
      {TONES.map(tone => (
        <div key={tone} className="flex items-center gap-2" data-testid={`tone-${tone}`}>
          <StatusDot tone={tone} />
        </div>
      ))}
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    for (const tone of TONES) {
      const wrapper = await canvas.findByTestId(`tone-${tone}`);
      const dot = wrapper.querySelector('[data-slot="status-dot"]') as HTMLElement;
      await expect(dot).toBeInTheDocument();
      await expect(dot.getAttribute("data-tone")).toBe(tone);
      await expect(dot.style.backgroundColor).toBe(TONE_TO_COLOR[tone]);
    }
  },
};

export const PulseSuccess: Story = {
  args: {
    tone: "success",
    pulse: true,
  },
};

export const SizeVariants: Story = {
  render: () => (
    <div className="flex items-center gap-6">
      <div className="flex items-center gap-2">
        <StatusDot size="sm" tone="success" />
        <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]">
          sm · 6px
        </span>
      </div>
      <div className="flex items-center gap-2">
        <StatusDot size="md" tone="success" />
        <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-[color:var(--color-text-label)]">
          md · 8px
        </span>
      </div>
    </div>
  ),
};
