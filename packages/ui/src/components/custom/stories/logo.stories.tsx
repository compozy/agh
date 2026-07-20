import type { Meta, StoryObj } from "@storybook/react-vite";

import { Logo, type LogoVariant } from "../logo";

const meta: Meta<typeof Logo> = {
  title: "components/custom/Logo",
  component: Logo,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "AGH brand mark. Use `logo` for full lockups, `symbol` for square app surfaces, `lettering` only where the symbol is already present nearby, `glyph` for compact marketing tiles, and `menubar` for the OS-shell chrome mark (OpenDesign `mb-logo`).",
      },
    },
  },
  argTypes: {
    variant: {
      control: "select",
      options: ["logo", "symbol", "lettering", "glyph", "menubar"],
    },
    decorative: {
      control: "boolean",
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const VARIANTS: LogoVariant[] = ["logo", "symbol", "lettering", "glyph", "menubar"];

export const Default: Story = {
  args: {
    variant: "logo",
    label: "AGH",
    className: "h-12 w-auto",
  },
};

export const Symbol: Story = {
  args: {
    variant: "symbol",
    label: "AGH symbol",
    className: "size-16",
  },
};

export const Lettering: Story = {
  args: {
    variant: "lettering",
    label: "AGH lettering",
    className: "h-12 w-auto",
  },
};

export const Glyph: Story = {
  args: {
    variant: "glyph",
    label: "AGH glyph",
    className: "size-8",
  },
};

/**
 * OS-shell menubar mark — OpenDesign `mb-logo` artwork owned by `@agh/ui` Logo.
 * Host controls supply `text-accent` so the tile inherits the accent fill.
 */
export const Menubar: Story = {
  args: {
    variant: "menubar",
    label: "AGH menubar",
    decorative: true,
    className: "size-menubar-logo text-accent",
  },
  decorators: [
    Story => (
      <div className="grid size-7 place-items-center rounded-menubar-control bg-shell-glass text-accent">
        <Story />
      </div>
    ),
  ],
};

export const Variants: Story = {
  render: () => (
    <div className="grid min-w-[520px] gap-6 rounded-lg border border-line bg-canvas-soft p-6">
      {VARIANTS.map(variant => (
        <div key={variant} className="grid grid-cols-[7rem_1fr] items-center gap-6">
          <span className="font-mono text-eyebrow font-medium uppercase tracking-badge text-subtle">
            {variant}
          </span>
          {variant === "menubar" ? (
            <span className="grid size-7 place-items-center rounded-menubar-control text-accent">
              <Logo variant="menubar" label="AGH menubar" className="size-menubar-logo" />
            </span>
          ) : (
            <Logo
              variant={variant}
              label={`AGH ${variant}`}
              className={variant === "symbol" || variant === "glyph" ? "size-14" : "h-12 w-auto"}
            />
          )}
        </div>
      ))}
    </div>
  ),
};
