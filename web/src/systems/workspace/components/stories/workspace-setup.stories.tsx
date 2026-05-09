import { useEffect, useRef, useState } from "react";
import { expect, userEvent, within } from "storybook/test";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";

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
  args: {},
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
};

export const OnboardingPathError: Story = {
  args: {},
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceSetupValidationHarness />
    </StorySurface>
  ),
};

export const OnboardingGlobalUnavailable: Story = {
  args: {},
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
  args: {},
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
  args: {},
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

export const OnboardingMobile: Story = {
  args: {},
  parameters: {
    viewport: { defaultViewport: "iphone14" },
    docs: {
      description: {
        story:
          "Below `lg` breakpoint the two-column hero collapses to a single rail so onboarding remains tappable on mobile.",
      },
    },
  },
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
};

export const OnboardingLoadingGlobal: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: {
    ...storybookMswParameters({
      workspace: [
        http.post("/api/workspaces", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
    docs: {
      description: {
        story:
          "Drives the global submission CTA into its loading state by stalling `POST /api/workspaces` indefinitely.",
      },
    },
  },
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const button = await canvas.findByTestId("workspace-use-global");
    await userEvent.click(button);
    await expect(canvas.getByTestId("workspace-use-global")).toBeDisabled();
  },
};

export const OnboardingLoadingManual: Story = {
  args: {},
  tags: ["play-fn"],
  parameters: {
    ...storybookMswParameters({
      workspace: [
        http.post("/api/workspaces", async () => {
          await delay("infinite");
          return HttpResponse.json({});
        }),
      ],
    }),
    docs: {
      description: {
        story:
          "Manual submission spinner: stall `POST /api/workspaces` and submit a valid absolute path.",
      },
    },
  },
  render: () => (
    <StorySurface className="p-0">
      <WorkspaceOnboarding onWorkspaceResolved={() => undefined} />
    </StorySurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByLabelText("Workspace path");
    await userEvent.type(input, "/Users/pedro/Dev/agh");
    await userEvent.click(canvas.getByTestId("workspace-register-manual"));
    await expect(canvas.getByTestId("workspace-register-manual")).toBeDisabled();
  },
};

export const UseGlobalWorkspace: Story = {
  args: {},
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
