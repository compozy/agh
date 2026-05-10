import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SettingsSandboxEntry } from "@/systems/settings";

type RestartBanner = {
  isVisible: boolean;
  isRestartRequired: boolean;
  isPolling: boolean;
  isSuccessful: boolean;
  isFailed: boolean;
  operationId: string | null;
  status: string | null;
  failureReason?: string;
  activeSessionCount: number;
  trigger: ReturnType<typeof vi.fn>;
  isTriggerPending: boolean;
  triggerError: unknown;
  dismiss: ReturnType<typeof vi.fn>;
};

const localEnv: SettingsSandboxEntry = {
  name: "local",
  workspace_usage_count: 3,
  profile: {
    backend: "local",
    sync_mode: "none",
    persistence: "transient",
    runtime_root: "~",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "global-config", scope: "global" },
  },
};

const builtinEnv: SettingsSandboxEntry = {
  name: "builtin-local",
  workspace_usage_count: 0,
  profile: {
    backend: "local",
    sync_mode: "none",
    persistence: "transient",
    runtime_root: "~",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "builtin-provider", scope: "global" },
  },
};

type PageState = {
  isLoading: boolean;
  error: Error | null;
  envelope: { sandboxes: SettingsSandboxEntry[] } | null;
  sandboxes: SettingsSandboxEntry[];
  counts: { total: number; totalWorkspaces: number };
  restart: RestartBanner;
  editor: { mode: "closed" | "create" | "edit"; [key: string]: unknown };
  editorIsValid: boolean;
  editorError: string | null;
  editorWarnings: string[] | undefined;
  editorIsSaving: boolean;
  openCreate: ReturnType<typeof vi.fn>;
  openEdit: ReturnType<typeof vi.fn>;
  closeEditor: ReturnType<typeof vi.fn>;
  updateDraft: ReturnType<typeof vi.fn>;
  saveEditor: ReturnType<typeof vi.fn>;
  deleteTarget: { mode: "closed" | "open"; entry?: SettingsSandboxEntry };
  deleteError: string | null;
  deleteIsPending: boolean;
  openDelete: ReturnType<typeof vi.fn>;
  closeDelete: ReturnType<typeof vi.fn>;
  confirmDelete: ReturnType<typeof vi.fn>;
  lastAction: null | {
    kind: "saved" | "deleted";
    name: string;
    result: { restart_required: boolean };
    usageCount?: number;
  };
  dismissLastAction: ReturnType<typeof vi.fn>;
};

const restartBanner: RestartBanner = {
  isVisible: false,
  isRestartRequired: false,
  isPolling: false,
  isSuccessful: false,
  isFailed: false,
  operationId: null,
  status: null,
  failureReason: undefined,
  activeSessionCount: 0,
  trigger: vi.fn(),
  isTriggerPending: false,
  triggerError: null,
  dismiss: vi.fn(),
};

let pageState: PageState;

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-sandbox-page", () => ({
  useSandboxPage: () => pageState,
}));

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: { sandboxes: [localEnv, builtinEnv] },
    sandboxes: [localEnv, builtinEnv],
    counts: { total: 2, totalWorkspaces: 3 },
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    editor: { mode: "closed" },
    editorIsValid: false,
    editorError: null,
    editorWarnings: undefined,
    editorIsSaving: false,
    openCreate: vi.fn(),
    openEdit: vi.fn(),
    closeEditor: vi.fn(),
    updateDraft: vi.fn(),
    saveEditor: vi.fn(),
    deleteTarget: { mode: "closed" },
    deleteError: null,
    deleteIsPending: false,
    openDelete: vi.fn(),
    closeDelete: vi.fn(),
    confirmDelete: vi.fn(),
    lastAction: null,
    dismissLastAction: vi.fn(),
    ...overrides,
  };
}

beforeEach(() => {
  pageState = makeState();
});

import { Route } from "../sandbox";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const SandboxPage = (Route as any).component as () => ReactNode;

describe("SandboxPage", () => {
  it("renders loading state", () => {
    pageState = makeState({ isLoading: true, envelope: null, sandboxes: [] });
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-page-loading")).toBeInTheDocument();
  });

  it("renders error state with the error message", () => {
    pageState = makeState({
      envelope: null,
      sandboxes: [],
      error: new Error("nope"),
    });
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-page-error")).toHaveTextContent("nope");
  });

  it("renders the profile grid with workspace usage counts", () => {
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-page-total")).toHaveTextContent("2 profiles");
    expect(screen.getByTestId("sandbox-page-workspaces")).toHaveTextContent(
      "3 workspace references"
    );
    expect(screen.getByTestId("sandbox-page-card-local-usage")).toHaveTextContent("3 workspaces");
    expect(screen.getByTestId("sandbox-page-card-local-source")).toHaveTextContent("CONFIG");
  });

  it("shows the @agh/ui Empty card when no sandboxes exist", () => {
    pageState = makeState({
      sandboxes: [],
      envelope: { sandboxes: [] },
      counts: { total: 0, totalWorkspaces: 0 },
    });
    render(<SandboxPage />);
    const empty = screen.getByTestId("sandbox-page-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No sandbox profiles defined");
  });

  it("wires create, edit, and delete controls", () => {
    render(<SandboxPage />);
    fireEvent.click(screen.getByTestId("sandbox-page-create"));
    expect(pageState.openCreate).toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("sandbox-page-card-local-edit"));
    expect(pageState.openEdit).toHaveBeenCalledWith(localEnv);

    fireEvent.click(screen.getByTestId("sandbox-page-card-local-delete"));
    expect(pageState.openDelete).toHaveBeenCalledWith(localEnv);
  });

  it("disables delete for builtin sandboxes", () => {
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-page-card-builtin-local-delete")).toBeDisabled();
  });

  it("renders the create dialog with the required hint", () => {
    pageState = makeState({
      editor: {
        mode: "create",
        draft: {
          name: "",
          backend: "local",
          sync_mode: "",
          persistence: "",
          runtime_root: "",
          preserved: {},
        },
      },
    });
    render(<SandboxPage />);
    expect(screen.getByTestId("settings-sandbox-editor-title")).toHaveTextContent(
      "New sandbox profile"
    );
    expect(screen.getByTestId("sandbox-editor-name-input")).not.toBeDisabled();
    expect(screen.getByTestId("sandbox-editor-backend-input")).toHaveValue("local");
  });

  it("renders preserved-fields notice when nested profile keys exist", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "local",
        draft: {
          name: "local",
          backend: "local",
          sync_mode: "",
          persistence: "",
          runtime_root: "",
          preserved: { network: { required: true } },
        },
        entry: localEnv,
      },
    });
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-editor-preserved")).toHaveTextContent("network");
  });

  it("surfaces usage warnings in the delete dialog when a profile is referenced", () => {
    pageState = makeState({ deleteTarget: { mode: "open", entry: localEnv } });
    render(<SandboxPage />);
    expect(screen.getByTestId("sandbox-delete-usage")).toHaveTextContent(
      "3 workspaces currently reference this profile"
    );
  });

  it("shows the last action banner after save and delete via @agh/ui Alert", () => {
    pageState = makeState({
      lastAction: {
        kind: "deleted",
        name: "local",
        result: { restart_required: true },
        usageCount: 3,
      },
    });
    render(<SandboxPage />);
    const banner = screen.getByTestId("sandbox-page-action-result");
    expect(banner).toHaveAttribute("data-slot", "alert");
    expect(banner).toHaveTextContent('Deleted "local"');
    expect(banner).toHaveTextContent("3 workspaces affected");
  });
});
