import { useEffect, useRef, useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { StorySurface } from "@/storybook/story-layout";

import { WorkspaceOnboarding, WorkspaceSetupDialog } from "../workspace-setup";

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

export const OnboardingDefault: Story = {
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
};

export const OnboardingPathError: Story = {
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceSetupValidationHarness />
    </StorySurface>
  ),
};

export const OnboardingGlobalUnavailable: Story = {
  parameters: {
    ...storybookMswParameters({
      daemon: [
        http.get("/api/daemon/status", () =>
          HttpResponse.json({ status: "running", user_home_dir: "" })
        ),
      ],
    }),
  },
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
};

export const SetupDialogOpen: StoryObj<typeof WorkspaceSetupDialog> = {
  render: () => (
    <StorySurface className="p-10">
      <WorkspaceSetupDialog
        open
        onOpenChange={() => undefined}
        onWorkspaceResolved={() => undefined}
      />
    </StorySurface>
  ),
};

export const SubmitManualPath: Story = {
  tags: ["play-fn"],
  render: () => {
    function Harness() {
      const [status, setStatus] = useState("");
      return (
        <StorySurface className="p-0">
          <WorkspaceOnboarding onWorkspaceResolved={id => setStatus(`resolved:${id}`)} />
          <div data-testid="resolve-status" className="sr-only">
            {status}
          </div>
        </StorySurface>
      );
    }

    return <Harness />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByLabelText("Workspace path");
    await userEvent.type(input, "/Users/pedro/Dev/agh");
    const submit = canvas.getByTestId("workspace-register-manual");
    await userEvent.click(submit);
    await expect(canvas.queryByTestId("workspace-path-error")).not.toBeInTheDocument();
  },
};

export const UseGlobalWorkspace: Story = {
  tags: ["play-fn"],
  render: () => {
    function Harness() {
      const [status, setStatus] = useState("");
      return (
        <StorySurface className="p-0">
          <WorkspaceOnboarding onWorkspaceResolved={id => setStatus(`resolved:${id}`)} />
          <div data-testid="resolve-status" className="sr-only">
            {status}
          </div>
        </StorySurface>
      );
    }

    return <Harness />;
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = await canvas.findByTestId("workspace-use-global");
    await userEvent.click(button);
  },
};
