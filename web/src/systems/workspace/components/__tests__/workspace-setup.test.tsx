import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { dialogShellClass, UIProvider } from "@agh/ui";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { WorkspaceOnboarding, WorkspaceSetupDialog } from "../workspace-setup";

const {
  mockResolveMutateAsync,
  mockCreateMutateAsync,
  mockToastSuccess,
  mockToastError,
  mockDaemonStatusState,
  mockBrowseState,
} = vi.hoisted(() => ({
  mockResolveMutateAsync: vi.fn(),
  mockCreateMutateAsync: vi.fn(),
  mockToastSuccess: vi.fn(),
  mockToastError: vi.fn(),
  mockDaemonStatusState: {
    data: { user_home_dir: "/Users/pedro" } as { user_home_dir: string } | undefined,
    isLoading: false,
  },
  mockBrowseState: {
    data: {
      path: "/Users/pedro/Dev",
      parent: "/Users/pedro",
      home: "/Users/pedro",
      entries: [
        { name: "checkout-platform", path: "/Users/pedro/Dev/checkout-platform", is_dir: true },
        { name: "billing", path: "/Users/pedro/Dev/billing", is_dir: true },
      ],
    } as unknown,
    isLoading: false,
    isFetching: false,
    error: null as Error | null,
  },
}));

vi.mock("sonner", () => ({
  toast: { success: mockToastSuccess, error: mockToastError },
}));

vi.mock("@/systems/status", () => ({
  useDaemonStatus: () => mockDaemonStatusState,
}));

vi.mock("@/systems/onboarding", async () => {
  const actual =
    await vi.importActual<typeof import("@/systems/onboarding")>("@/systems/onboarding");
  return { ...actual, useDirectoryBrowser: () => mockBrowseState };
});

vi.mock("@/systems/agent", async () => {
  const actual = await vi.importActual<typeof import("@/systems/agent")>("@/systems/agent");
  return { ...actual, useAgents: () => ({ data: [], isLoading: false, error: null }) };
});

vi.mock("@/systems/settings", async () => {
  const actual = await vi.importActual<typeof import("@/systems/settings")>("@/systems/settings");
  return {
    ...actual,
    useSettingsSandboxes: () => ({ data: { sandboxes: [] }, isLoading: false, error: null }),
  };
});

vi.mock("../../hooks/use-workspaces", () => ({
  useResolveWorkspace: () => ({ mutateAsync: mockResolveMutateAsync }),
  useCreateWorkspace: () => ({ mutateAsync: mockCreateMutateAsync }),
}));

const createdWorkspace = {
  id: "ws_project",
  root_dir: "/Users/pedro/Dev/checkout-platform",
  add_dirs: [],
  name: "checkout-platform",
  created_at: "2026-04-10T12:00:00Z",
  updated_at: "2026-04-10T12:00:00Z",
};

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

function resetMocks() {
  mockDaemonStatusState.data = { user_home_dir: "/Users/pedro" };
  mockDaemonStatusState.isLoading = false;
  mockBrowseState.isLoading = false;
  mockBrowseState.isFetching = false;
  mockBrowseState.error = null;
  mockBrowseState.data = {
    path: "/Users/pedro/Dev",
    parent: "/Users/pedro",
    home: "/Users/pedro",
    entries: [
      { name: "checkout-platform", path: "/Users/pedro/Dev/checkout-platform", is_dir: true },
      { name: "billing", path: "/Users/pedro/Dev/billing", is_dir: true },
    ],
  };
  mockResolveMutateAsync.mockReset();
  mockCreateMutateAsync.mockReset();
  mockToastSuccess.mockReset();
  mockToastError.mockReset();
}

describe("WorkspaceOnboarding", () => {
  beforeEach(resetMocks);

  it("renders onboarding hero, the global card, and the directory browser", () => {
    renderOnboarding();

    expect(screen.getByTestId("workspace-onboarding")).toBeInTheDocument();
    expect(
      screen.getByRole("heading", {
        name: "Start AGH with a real workspace, not an empty shell.",
      })
    ).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-global-card")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-browser")).toBeInTheDocument();
  });

  it("resolves the global workspace from daemon user_home_dir", async () => {
    const user = userEvent.setup();
    const onWorkspaceResolved = vi.fn();
    mockResolveMutateAsync.mockResolvedValue({ ...createdWorkspace, id: "ws_home", name: "pedro" });

    renderOnboarding(onWorkspaceResolved);
    await user.click(screen.getByTestId("workspace-use-global"));

    await waitFor(() => {
      expect(mockResolveMutateAsync).toHaveBeenCalledWith({ path: "/Users/pedro" });
    });
    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_home");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: pedro");
  });

  it("disables the global workspace CTA when daemon status is unavailable", () => {
    mockDaemonStatusState.data = undefined;

    renderOnboarding();

    expect(screen.getByTestId("workspace-use-global")).toBeDisabled();
    expect(screen.getByTestId("workspace-global-meta").textContent).toContain(
      "Daemon status unavailable"
    );
  });

  it("keeps a single accent surface on the workspace setup global card (CTA only)", () => {
    renderOnboarding();
    const globalCard = screen.getByTestId("workspace-setup-global-card");
    expect(globalCard.querySelectorAll('[data-tone="accent"]').length).toBe(0);
    expect(screen.getByTestId("workspace-use-global")).toBeInTheDocument();
    expect(globalCard.querySelector('[data-tone="success"]')).not.toBeNull();
  });

  it("offers no plain path input for root selection", () => {
    renderOnboarding();

    expect(screen.queryByLabelText("Workspace path")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-manual-path-input")).not.toBeInTheDocument();
    expect(screen.queryByTestId("workspace-register-manual")).not.toBeInTheDocument();
  });

  it("registers a browsed root and returns the created workspace to the caller", async () => {
    const user = userEvent.setup();
    const onWorkspaceResolved = vi.fn();
    mockCreateMutateAsync.mockResolvedValue(createdWorkspace);

    renderOnboarding(onWorkspaceResolved);

    await user.click(
      screen.getByRole("button", { name: "Use checkout-platform as the workspace root" })
    );
    await user.click(screen.getByTestId("workspace-setup-onboarding-submit"));

    await waitFor(() => {
      expect(mockCreateMutateAsync).toHaveBeenCalledTimes(1);
    });
    expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      root_dir: "/Users/pedro/Dev/checkout-platform",
      name: "checkout-platform",
    });
    expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_project");
    expect(mockToastSuccess).toHaveBeenCalledWith("Workspace ready: checkout-platform");
  });

  it("reports a failed registration inline and keeps the draft", async () => {
    const user = userEvent.setup();
    mockCreateMutateAsync.mockRejectedValue(new Error("root is already registered"));

    renderOnboarding();

    await user.click(screen.getByTestId("workspace-setup-browser-use-current"));
    await user.click(screen.getByTestId("workspace-setup-onboarding-submit"));

    expect(await screen.findByTestId("workspace-setup-error")).toHaveTextContent(
      "root is already registered"
    );
    expect(screen.getByTestId("workspace-setup-name-input")).toHaveValue("Dev");
  });
});

describe("WorkspaceSetupDialog", () => {
  beforeEach(resetMocks);

  it("renders a portaled dialog when `open` is true", () => {
    renderDialog({ open: true });
    const host = screen.getByTestId("workspace-setup-dialog");
    expect(host).toBeInTheDocument();
    expect(host.closest("body")).toBe(document.body);
    expect(screen.getByTestId("workspace-setup-dialog-body")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Add workspace" })).toBeInTheDocument();
  });

  it("hosts the split shell on the xl modal token", () => {
    renderDialog({ open: true });

    const host = screen.getByTestId("workspace-setup-dialog");
    for (const token of dialogShellClass("xl").split(" ")) {
      expect(host.className).toContain(token);
    }
    expect(host.className).not.toMatch(/max-w-xl/);
    expect(host.dataset.frame).toBe("unframed");

    // Two independent panes: location on the left, session defaults on the right.
    // The ≤980px collapse itself is owned by the `EntityDialogBody` suite.
    const body = screen.getByTestId("workspace-setup-dialog-body");
    expect(body).toHaveAttribute("data-variant", "split");
    const main = body.querySelector('[data-slot="entity-dialog-body-main"]');
    const side = body.querySelector('[data-slot="entity-dialog-body-side"]');
    expect(main).not.toBeNull();
    expect(side).not.toBeNull();
    expect(within(main as HTMLElement).getByTestId("workspace-setup-browser")).toBeInTheDocument();
    expect(
      within(side as HTMLElement).getByTestId("workspace-setup-add-dir-input")
    ).toBeInTheDocument();

    const header = host.querySelector('[data-slot="dialog-header"]') as HTMLElement | null;
    expect(header?.dataset.variant).toBe("ruled");
    expect(screen.getByText("Workspace")).toBeInTheDocument();
  });

  it("does not mount the dialog body when `open` is false", () => {
    renderDialog({ open: false });
    expect(screen.queryByTestId("workspace-setup-dialog-body")).not.toBeInTheDocument();
  });

  it("selects a root from the browser without registering anything", async () => {
    const user = userEvent.setup();
    renderDialog({ open: true });

    expect(screen.getByTestId("workspace-setup-root-empty")).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-submit")).toBeDisabled();

    await user.click(
      screen.getByRole("button", { name: "Use checkout-platform as the workspace root" })
    );

    // Picking a root must not write anything — registration waits for submit.
    expect(mockCreateMutateAsync).not.toHaveBeenCalled();
    expect(mockResolveMutateAsync).not.toHaveBeenCalled();
    expect(
      within(screen.getByTestId("workspace-setup-selected-root")).getByText(
        "/Users/pedro/Dev/checkout-platform"
      )
    ).toBeInTheDocument();
    expect(screen.getByTestId("workspace-setup-name-input")).toHaveValue("checkout-platform");
    expect(screen.getByTestId("workspace-setup-submit")).toBeEnabled();
  });

  it("issues exactly one create request carrying the full draft on submit", async () => {
    const user = userEvent.setup();
    mockCreateMutateAsync.mockResolvedValue(createdWorkspace);
    const { onOpenChange, onWorkspaceResolved } = renderDialog({ open: true });

    await user.click(screen.getByTestId("workspace-setup-browser-use-current"));
    await user.clear(screen.getByTestId("workspace-setup-name-input"));
    await user.type(screen.getByTestId("workspace-setup-name-input"), "Checkout platform");
    await user.type(screen.getByTestId("workspace-setup-add-dir-input"), "/Users/pedro/Dev/shared");
    await user.click(screen.getByTestId("workspace-setup-add-dir-submit"));
    await user.click(screen.getByTestId("workspace-setup-submit"));

    await waitFor(() => {
      expect(mockCreateMutateAsync).toHaveBeenCalledTimes(1);
    });
    expect(mockCreateMutateAsync).toHaveBeenCalledWith({
      root_dir: "/Users/pedro/Dev",
      name: "Checkout platform",
      add_dirs: ["/Users/pedro/Dev/shared"],
    });
    await waitFor(() => {
      expect(onWorkspaceResolved).toHaveBeenCalledWith("ws_project");
    });
    await waitFor(() => {
      expect(onOpenChange).toHaveBeenCalledWith(false);
    });
  });

  it("blocks duplicate submits and the close route while the create is in flight", async () => {
    const user = userEvent.setup();
    let releaseCreate: (() => void) | undefined;
    mockCreateMutateAsync.mockImplementation(
      () =>
        new Promise(resolve => {
          releaseCreate = () => resolve(createdWorkspace);
        })
    );
    renderDialog({ open: true });

    await user.click(screen.getByTestId("workspace-setup-browser-use-current"));
    await user.click(screen.getByTestId("workspace-setup-submit"));

    await waitFor(() => {
      expect(screen.getByTestId("workspace-setup-submit")).toBeDisabled();
    });
    expect(screen.getByTestId("workspace-setup-cancel")).toBeDisabled();
    expect(
      screen
        .getByTestId("workspace-setup-dialog")
        .querySelectorAll('[data-slot="entity-dialog-header-close"]')
    ).toHaveLength(0);

    releaseCreate?.();
    await waitFor(() => {
      expect(mockCreateMutateAsync).toHaveBeenCalledTimes(1);
    });
  });

  it("keeps the draft and reports the failure when the create is rejected", async () => {
    const user = userEvent.setup();
    mockCreateMutateAsync.mockRejectedValue(new Error("root is already registered"));
    const { onOpenChange } = renderDialog({ open: true });

    await user.click(screen.getByTestId("workspace-setup-browser-use-current"));
    await user.click(screen.getByTestId("workspace-setup-submit"));

    expect(await screen.findByTestId("workspace-setup-error")).toHaveTextContent(
      "root is already registered"
    );
    expect(screen.getByTestId("workspace-setup-name-input")).toHaveValue("Dev");
    expect(onOpenChange).not.toHaveBeenCalledWith(false);
  });

  it("surfaces the browser reading, empty, and error states", () => {
    mockBrowseState.isLoading = true;
    const { unmount } = renderDialog({ open: true });
    expect(screen.getByTestId("workspace-setup-browser-loading")).toBeInTheDocument();
    unmount();

    resetMocks();
    mockBrowseState.data = {
      path: "/Users/pedro/Dev",
      parent: "/Users/pedro",
      home: "/Users/pedro",
      entries: [],
    };
    const empty = renderDialog({ open: true });
    expect(screen.getByTestId("workspace-setup-browser-empty")).toBeInTheDocument();
    empty.unmount();

    resetMocks();
    mockBrowseState.error = new Error("permission denied");
    renderDialog({ open: true });
    expect(screen.getByTestId("workspace-setup-browser-error")).toHaveTextContent(
      "permission denied"
    );
  });

  it("closes via onOpenChange after registering the global workspace", async () => {
    const user = userEvent.setup();
    mockResolveMutateAsync.mockResolvedValue({ ...createdWorkspace, id: "ws_home", name: "pedro" });
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
