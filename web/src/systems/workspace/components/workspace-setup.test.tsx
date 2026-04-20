import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { UIProvider } from "@agh/ui";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceOnboarding, WorkspaceSetupDialog } from "./workspace-setup";

const { mockMutateAsync, mockToastSuccess, mockToastError, mockDaemonStatusState } = vi.hoisted(
  () => ({
    mockMutateAsync: vi.fn(),
    mockToastSuccess: vi.fn(),
    mockToastError: vi.fn(),
    mockDaemonStatusState: {
      data: { user_home_dir: "/Users/pedro" } as { user_home_dir: string } | undefined,
      isLoading: false,
    },
  })
);

vi.mock("sonner", () => ({
  toast: {
    success: mockToastSuccess,
    error: mockToastError,
  },
}));

vi.mock("@/systems/daemon", () => ({
  useDaemonStatus: () => mockDaemonStatusState,
}));

vi.mock("../hooks/use-workspaces", () => ({
  useResolveWorkspace: () => ({
    mutateAsync: mockMutateAsync,
  }),
}));

function renderOnboarding(onWorkspaceResolved = vi.fn()) {
  return render(
    <UIProvider reducedMotion="always">
      <WorkspaceOnboarding onWorkspaceResolved={onWorkspaceResolved} />
    </UIProvider>
  );
}

function renderDialog(props: Partial<React.ComponentProps<typeof WorkspaceSetupDialog>> = {}) {
  const onOpenChange = props.onOpenChange ?? vi.fn();
  const onWorkspaceResolved = props.onWorkspaceResolved ?? vi.fn();
  const open = props.open ?? true;

  const utils = render(
    <UIProvider reducedMotion="always">
      <WorkspaceSetupDialog
        open={open}
        onOpenChange={onOpenChange}
        onWorkspaceResolved={onWorkspaceResolved}
      />
    </UIProvider>
  );

  return { ...utils, onOpenChange, onWorkspaceResolved };
}

describe("WorkspaceOnboarding", () => {
  beforeEach(() => {
    mockDaemonStatusState.data = { user_home_dir: "/Users/pedro" };
    mockDaemonStatusState.isLoading = false;
    mockMutateAsync.mockReset();
    mockToastSuccess.mockReset();
    mockToastError.mockReset();
  });

  it("renders onboarding hero + global + manual cards", () => {
    renderOnboarding();

    const onboarding = screen.getByTestId("workspace-onboarding");
    expect(onboarding).toBeInTheDocument();
    expect(onboarding.className).toContain("flex-1");
    expect(onboarding.className).toContain("overflow-y-auto");
    expect(onboarding.className).not.toContain("min-h-screen");
    expect(
      screen.getByRole("heading", {
        name: "Start AGH with a real workspace, not an empty shell.",
      })
    ).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-global-card")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-manual-card")).toBeInTheDocument();
  });

  it("resolves the global workspace from daemon user_home_dir", async () => {
    const user = userEvent.setup();
    const onWorkspaceResolved = vi.fn();

    mockMutateAsync.mockResolvedValue({
      id: "ws_home",
      root_dir: "/Users/pedro",
      add_dirs: [],
      name: "pedro",
      created_at: "2026-04-10T12:00:00Z",
      updated_at: "2026-04-10T12:00:00Z",
    });

    renderOnboarding(onWorkspaceResolved);
    await user.click(screen.getByTestId("workspace-use-global"));

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({ path: "/Users/pedro" });
    });

    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_home");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: pedro");
  });

  it("disables the global workspace CTA when daemon status is unavailable", () => {
    mockDaemonStatusState.data = undefined;
    mockDaemonStatusState.isLoading = false;

    renderOnboarding();

    expect(screen.getByTestId("workspace-use-global")).toBeDisabled();
    expect(screen.getByTestId("workspace-global-meta").textContent).toContain(
      "Daemon status unavailable"
    );
  });

  it("stacks onboarding setup cards into a single constrained options rail", () => {
    renderOnboarding();

    const optionsRail = screen.getByTestId("workspace-setup-options");
    expect(optionsRail.className).toContain("flex-col");
    expect(optionsRail.className).toContain("lg:max-w-[24rem]");
    expect(optionsRail.className).not.toContain("grid-cols-2");
  });

  it("rejects relative manual paths before calling resolve", async () => {
    const user = userEvent.setup();

    renderOnboarding();

    await user.type(screen.getByLabelText("Workspace path"), "projects/agh");
    await user.click(screen.getByTestId("workspace-register-manual"));

    expect(screen.getByTestId("workspace-path-error")).toHaveTextContent(
      "Workspace path must be absolute."
    );
    expect(mockMutateAsync).not.toHaveBeenCalled();
  });

  it("registers a manual absolute path and returns the selected workspace", async () => {
    const user = userEvent.setup();
    const onWorkspaceResolved = vi.fn();

    mockMutateAsync.mockResolvedValue({
      id: "ws_project",
      root_dir: "/Users/pedro/Dev/projects/agh",
      add_dirs: [],
      name: "agh",
      created_at: "2026-04-10T12:00:00Z",
      updated_at: "2026-04-10T12:00:00Z",
    });

    renderOnboarding(onWorkspaceResolved);

    await user.type(screen.getByLabelText("Workspace path"), "/Users/pedro/Dev/projects/agh");
    await user.click(screen.getByTestId("workspace-register-manual"));

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        path: "/Users/pedro/Dev/projects/agh",
      });
    });

    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_project");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: agh");
  });
});

describe("WorkspaceSetupDialog", () => {
  beforeEach(() => {
    mockDaemonStatusState.data = { user_home_dir: "/Users/pedro" };
    mockDaemonStatusState.isLoading = false;
    mockMutateAsync.mockReset();
    mockToastSuccess.mockReset();
    mockToastError.mockReset();
  });

  it("renders a portaled dialog when `open` is true", () => {
    renderDialog({ open: true });
    expect(screen.getByTestId("workspace-setup-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-dialog-body")).toBeInTheDocument();
    expect(screen.getByText("Add workspace")).toBeInTheDocument();
  });

  it("does not mount the dialog body when `open` is false", () => {
    renderDialog({ open: false });
    expect(screen.queryByTestId("workspace-setup-dialog-body")).not.toBeInTheDocument();
  });

  it("closes via onOpenChange after a successful workspace registration", async () => {
    const user = userEvent.setup();

    mockMutateAsync.mockResolvedValue({
      id: "ws_home",
      root_dir: "/Users/pedro",
      add_dirs: [],
      name: "pedro",
      created_at: "2026-04-10T12:00:00Z",
      updated_at: "2026-04-10T12:00:00Z",
    });

    const { onOpenChange, onWorkspaceResolved } = renderDialog({ open: true });

    await user.click(screen.getByTestId("workspace-use-global"));

    await waitFor(() => {
      expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_home");
    });

    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });
});
