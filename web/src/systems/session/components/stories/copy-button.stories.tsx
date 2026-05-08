import { useEffect, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { CopyButton } from "@/systems/session/components/copy-button";

const meta: Meta<typeof CopyButton> = {
  title: "systems/session/CopyButton",
  component: CopyButton,
  parameters: {
    layout: "centered",
  },
  args: {
    ariaLabel: "Copy transcript line",
    text: "Merchant NSP-1024 is cleared for payout release.",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function CopyButtonFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="rounded-full border border-(--color-divider) bg-(--color-surface) p-2">
        {children}
      </div>
    </CenteredSurface>
  );
}

function CopiedButtonHarness() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const hadClipboard = Boolean(navigator.clipboard);

    if (!navigator.clipboard) {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: {
          writeText: async () => undefined,
        },
      });
    }

    const originalWriteText = navigator.clipboard.writeText;
    navigator.clipboard.writeText = async () => undefined;

    const frame = window.requestAnimationFrame(() => {
      containerRef.current?.querySelector("button")?.click();
    });

    return () => {
      window.cancelAnimationFrame(frame);
      navigator.clipboard.writeText = originalWriteText;

      if (!hadClipboard) {
        Reflect.deleteProperty(navigator, "clipboard");
      }
    };
  }, []);

  return (
    <div ref={containerRef}>
      <CopyButton
        ariaLabel="Copy transcript line"
        text="Merchant NSP-1024 is cleared for payout release."
      />
    </div>
  );
}

export const Default: Story = {
  args: {},
  render: args => (
    <CopyButtonFrame>
      <CopyButton {...args} />
    </CopyButtonFrame>
  ),
};

export const Copied: Story = {
  args: {},
  render: () => (
    <CopyButtonFrame>
      <CopiedButtonHarness />
    </CopyButtonFrame>
  ),
};
