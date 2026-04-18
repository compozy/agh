import { useEffect, useRef, useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";

import { ProcessingIndicator } from "../processing-indicator";

const meta: Meta<typeof ProcessingIndicator> = {
  title: "systems/session/ProcessingIndicator",
  component: ProcessingIndicator,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function IndicatorFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] px-4 py-2">
        {children}
      </div>
    </CenteredSurface>
  );
}

function LongRunningProcessingIndicator() {
  const [ready, setReady] = useState(false);
  const originalNowRef = useRef(Date.now);

  useEffect(() => {
    const actualNow = originalNowRef.current();
    Date.now = () => actualNow - 95_000;
    setReady(true);

    const frame = window.requestAnimationFrame(() => {
      Date.now = originalNowRef.current;
    });

    return () => {
      window.cancelAnimationFrame(frame);
      Date.now = originalNowRef.current;
    };
  }, []);

  return ready ? <ProcessingIndicator /> : null;
}

export const Default: Story = {
  render: () => (
    <IndicatorFrame>
      <ProcessingIndicator />
    </IndicatorFrame>
  ),
};

export const LongRunning: Story = {
  render: () => (
    <IndicatorFrame>
      <LongRunningProcessingIndicator />
    </IndicatorFrame>
  ),
};
