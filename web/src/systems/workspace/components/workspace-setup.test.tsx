import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceOnboarding } from "./workspace-setup";

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

describe("WorkspaceOnboarding", () => {
  beforeEach(() => {
    mockDaemonStatusState.data = { user_home_dir: "/Users/pedro" };
    mockDaemonStatusState.isLoading = false;
    mockMutateAsync.mockReset();
    mockToastSuccess.mockReset();
    mockToastError.mockReset();
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

    render(<WorkspaceOnboarding onWorkspaceResolved={onWorkspaceResolved} />);
    await user.click(screen.getByRole("button", { name: "Use global workspace" }));

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({ path: "/Users/pedro" });
    });

    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_home");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: pedro");
  });

  it("disables the global workspace CTA when daemon status is unavailable", () => {
    mockDaemonStatusState.data = undefined;
    mockDaemonStatusState.isLoading = false;

    render(<WorkspaceOnboarding onWorkspaceResolved={vi.fn()} />);

    expect(screen.getByRole("button", { name: "Use global workspace" })).toBeDisabled();
    expect(
      screen.getByText("Daemon status unavailable. Connect AGH to use your global workspace.")
    ).toBeInTheDocument();
  });

  it("stacks onboarding setup cards into a single constrained options rail", () => {
    render(<WorkspaceOnboarding onWorkspaceResolved={vi.fn()} />);

    const optionsRail = screen.getByTestId("workspace-setup-options");
    expect(optionsRail.className).toContain("flex-col");
    expect(optionsRail.className).toContain("lg:max-w-[24rem]");
    expect(optionsRail.className).not.toContain("grid-cols-2");
  });

  it("rejects relative manual paths before calling resolve", async () => {
    const user = userEvent.setup();

    render(<WorkspaceOnboarding onWorkspaceResolved={vi.fn()} />);

    await user.type(screen.getByLabelText("Workspace path"), "projects/agh");
    await user.click(screen.getByRole("button", { name: "Register workspace" }));

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

    render(<WorkspaceOnboarding onWorkspaceResolved={onWorkspaceResolved} />);

    await user.type(screen.getByLabelText("Workspace path"), "/Users/pedro/Dev/projects/agh");
    await user.click(screen.getByRole("button", { name: "Register workspace" }));

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith({
        path: "/Users/pedro/Dev/projects/agh",
      });
    });

    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_project");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: agh");
  });
});
