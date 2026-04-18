import { useEffect, useRef } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { WorkspaceOnboarding } from "../workspace-setup";

const meta: Meta<typeof WorkspaceOnboarding> = {
  title: "systems/workspace/WorkspaceSetup",
  component: WorkspaceOnboarding,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function WorkspaceSetupValidationHarness() {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const frame = window.requestAnimationFrame(() => {
      const form = containerRef.current?.querySelector("form");
      form?.requestSubmit();
    });

    return () => window.cancelAnimationFrame(frame);
  }, []);

  return (
    <div ref={containerRef}>
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </div>
  );
}

export const Default: Story = {
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
};

export const ValidationError: Story = {
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceSetupValidationHarness />
    </StorySurface>
  ),
};
