import { useEffect, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { assistantMessageFixture } from "@/systems/session/mocks";

import { ThinkingBlock } from "../thinking-block";

const meta: Meta<typeof ThinkingBlock> = {
  title: "systems/session/ThinkingBlock",
  component: ThinkingBlock,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function ThinkingFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-2xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] py-3">
        {children}
      </div>
    </CenteredSurface>
  );
}

function ExpandedThinkingHarness() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      containerRef.current
        ?.querySelector<HTMLButtonElement>("[data-testid='thinking-trigger']")
        ?.click();
    });

    return () => window.cancelAnimationFrame(frame);
  }, []);

  return (
    <div ref={containerRef}>
      <ThinkingBlock
        thinking={
          assistantMessageFixture.thinking ?? "Need typed fixtures first so stories stay truthful."
        }
        thinkingComplete
      />
    </div>
  );
}

export const Collapsed: Story = {
  render: () => (
    <ThinkingFrame>
      <ThinkingBlock
        thinking={
          assistantMessageFixture.thinking ?? "Need typed fixtures first so stories stay truthful."
        }
        thinkingComplete
      />
    </ThinkingFrame>
  ),
};

export const Expanded: Story = {
  render: () => (
    <ThinkingFrame>
      <ExpandedThinkingHarness />
    </ThinkingFrame>
  ),
};
