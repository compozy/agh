import { useEffect, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import { CopyButton } from "../copy-button";

const meta: Meta<typeof CopyButton> = {
  title: "systems/session/CopyButton",
  component: CopyButton,
  parameters: {
    layout: "centered",
  },
  args: {
    ariaLabel: "Copy transcript line",
    text: "Storybook rollout is green.",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function CopyButtonFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="rounded-full border border-[color:var(--color-divider)] bg-[color:var(--color-surface)] p-2">
        {children}
      </div>
    </CenteredSurface>
  );
}

function CopiedButtonHarness() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const originalClipboard = navigator.clipboard;

    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async () => undefined,
      },
    });

    const frame = window.requestAnimationFrame(() => {
      containerRef.current?.querySelector("button")?.click();
    });

    return () => {
      window.cancelAnimationFrame(frame);
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: originalClipboard,
      });
    };
  }, []);

  return (
    <div ref={containerRef}>
      <CopyButton ariaLabel="Copy transcript line" text="Storybook rollout is green." />
    </div>
  );
}

export const Default: Story = {
  render: args => (
    <CopyButtonFrame>
      <CopyButton {...args} />
    </CopyButtonFrame>
  ),
};

export const Copied: Story = {
  render: () => (
    <CopyButtonFrame>
      <CopiedButtonHarness />
    </CopyButtonFrame>
  ),
};
