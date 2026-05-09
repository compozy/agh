import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, within } from "storybook/test";

import { Pill, type PillTone } from "../pill";

const meta: Meta<typeof Pill> = {
  title: "components/custom/Pill",
  component: Pill,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Unified semantic pill , replaces legacy `MonoBadge`, `StatusDot`, `KindChip`, `WireChip`, and connection-state label compositions. Compose with `Pill.Dot` for leading status dots.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const TONES: PillTone[] = ["neutral", "accent", "success", "warning", "danger", "info"];

const KIND_DOT_COLORS: Record<string, string> = {
  say: "var(--color-kind-say)",
  greet: "var(--color-kind-greet)",
  direct: "var(--color-kind-direct)",
  receipt: "var(--color-kind-receipt)",
  capability: "var(--color-kind-capability)",
  trace: "var(--color-kind-trace)",
  whois: "var(--color-kind-whois)",
};

export const Default: Story = {
  args: { children: "label" },
};

export const Tones: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {TONES.map(tone => (
        <Pill key={tone} tone={tone} mono>
          {tone}
        </Pill>
      ))}
    </div>
  ),
};

export const TonesSans: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {TONES.map(tone => (
        <Pill key={tone} tone={tone}>
          {tone}
        </Pill>
      ))}
    </div>
  ),
};

export const SolidEmphasis: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {TONES.map(tone => (
        <Pill key={tone} tone={tone} mono solid>
          {tone}
        </Pill>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "`solid` swaps the 15% tinted bg for a fully filled accent + ink-text formula.",
      },
    },
  },
};

export const Sizes: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-3">
      <Pill mono size="xs" tone="neutral">
        capability-id
      </Pill>
      <Pill mono size="sm" tone="accent">
        v0.2.1
      </Pill>
      <Pill mono size="md" tone="success">
        FILTER
      </Pill>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "`xs` = chip (5px radius). `sm` = badge (22px tall, 6px radius). `md` = filter (32px, 20px radius).",
      },
    },
  },
};

export const MonoLowercaseIdentifier: Story = {
  args: {},
  render: () => (
    <Pill mono uppercase={false}>
      agh-network/v0
    </Pill>
  ),
  parameters: {
    docs: {
      description: {
        story: "Override the auto-uppercase default for protocol strings.",
      },
    },
  },
};

export const WithDot: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      <Pill mono tone="success">
        <Pill.Dot />
        Connected
      </Pill>
      <Pill mono tone="warning">
        <Pill.Dot pulse />
        Reconnecting
      </Pill>
      <Pill mono tone="danger">
        <Pill.Dot />
        Disconnected
      </Pill>
    </div>
  ),
};

export const KindChipReplacement: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      {Object.keys(KIND_DOT_COLORS).map(kind => (
        <Pill key={kind} mono size="sm" tone="neutral">
          <Pill.Dot color={KIND_DOT_COLORS[kind]} size="sm" />
          {kind}
        </Pill>
      ))}
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story: "Protocol kind markers, leading dot keyed off the kind, label preserved.",
      },
    },
  },
};

export const ToggleInteractive: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-2">
      <Pill mono active render={<button type="button" />}>
        ALL
      </Pill>
      <Pill mono active={false} render={<button type="button" />}>
        SAY
      </Pill>
      <Pill mono active={false} render={<button type="button" />}>
        DIRECT
      </Pill>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Pass `render={<button />}` and `active` to render a stand-alone toggle chip (replaces `WireChip`).",
      },
    },
  },
};

export const LinkChip: Story = {
  args: {},
  render: () => <Pill.Link href="/tasks/task-102">Open task</Pill.Link>,
  parameters: {
    docs: {
      description: {
        story: "`Pill.Link` renders the same semantic pill chrome as an accessible anchor.",
      },
    },
  },
};

export const ConnectionIndicator: Story = {
  args: {},
  render: () => (
    <div className="flex flex-col gap-2">
      <div role="status" aria-live="polite" className="inline-flex items-center gap-2">
        <Pill.Dot tone="success" />
        <span className="font-mono text-[11px] font-medium uppercase tracking-[0.08em] text-(--color-text-label)">
          Connected
        </span>
      </div>
      <div role="status" aria-live="polite" className="inline-flex items-center gap-2">
        <Pill.Dot tone="warning" pulse />
        <span className="font-mono text-[11px] font-medium uppercase tracking-[0.08em] text-(--color-text-label)">
          Reconnecting
        </span>
      </div>
      <div role="status" aria-live="polite" className="inline-flex items-center gap-2">
        <Pill.Dot tone="danger" />
        <span className="font-mono text-[11px] font-medium uppercase tracking-[0.08em] text-(--color-text-label)">
          Disconnected
        </span>
      </div>
    </div>
  ),
  parameters: {
    docs: {
      description: {
        story:
          "Replacement composition for the legacy `ConnectionIndicator`, `Pill.Dot` + monospace label inside an `aria-live=polite` region.",
      },
    },
  },
};

export const StandaloneDots: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-6" data-testid="pill-dots">
      {TONES.map(tone => (
        <div key={tone} className="flex items-center gap-2" data-testid={`pill-dot-${tone}`}>
          <Pill.Dot tone={tone} />
          <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-(--color-text-label)">
            {tone}
          </span>
        </div>
      ))}
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    for (const tone of TONES) {
      const wrapper = await canvas.findByTestId(`pill-dot-${tone}`);
      const dot = wrapper.querySelector('[data-slot="pill-dot"]') as HTMLElement;
      await expect(dot).toBeInTheDocument();
      await expect(dot.getAttribute("data-tone")).toBe(tone);
    }
  },
};

export const DotSizes: Story = {
  args: {},
  render: () => (
    <div className="flex items-center gap-6">
      <div className="flex items-center gap-2">
        <Pill.Dot size="sm" tone="success" />
        <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-(--color-text-label)">
          sm · 6px
        </span>
      </div>
      <div className="flex items-center gap-2">
        <Pill.Dot size="md" tone="success" />
        <span className="font-mono text-[11px] uppercase tracking-[0.08em] text-(--color-text-label)">
          md · 8px
        </span>
      </div>
    </div>
  ),
};

export const PulseAnimation: Story = {
  args: { tone: "accent", mono: true, children: "RUNNING" },
  render: args => (
    <Pill {...args}>
      <Pill.Dot pulse />
      {args.children}
    </Pill>
  ),
};
