import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SettingsProviderEntry } from "@/systems/settings";

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

const claudeEntry: SettingsProviderEntry = {
  name: "claude",
  default: true,
  api_key_env_present: true,
  command_available: true,
  settings: {
    command: "npx claude",
    default_model: "claude-opus",
    api_key_env: "ANTHROPIC_API_KEY",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "global-config", scope: "global" },
    shadowed_sources: [{ kind: "builtin-provider", scope: "global" }],
  },
  fallback: {
    settings: { command: "npx claude" },
    source: { kind: "builtin-provider", scope: "global" },
  },
};

const builtinEntry: SettingsProviderEntry = {
  name: "codex",
  default: false,
  api_key_env_present: false,
  command_available: true,
  settings: {
    command: "npx codex",
    api_key_env: "OPENAI_API_KEY",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "builtin-provider", scope: "global" },
  },
};

type PageState = {
  isLoading: boolean;
  error: Error | null;
  envelope: { providers: SettingsProviderEntry[] } | null;
  providers: SettingsProviderEntry[];
  counts: { total: number; installed: number; binaryMissing: number; unconfigured: number };
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
  deleteTarget: { mode: "closed" | "open"; entry?: SettingsProviderEntry };
  deleteError: string | null;
  deleteIsPending: boolean;
  openDelete: ReturnType<typeof vi.fn>;
  closeDelete: ReturnType<typeof vi.fn>;
  confirmDelete: ReturnType<typeof vi.fn>;
  lastAction: null | {
    kind: "saved" | "deleted";
    name: string;
    result: { restart_required: boolean };
    hadFallback?: boolean;
  };
  dismissLastAction: ReturnType<typeof vi.fn>;
};

let pageState: PageState;

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

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-settings-providers-page", () => ({
  useSettingsProvidersPage: () => pageState,
}));

function defaultEditor() {
  return { mode: "closed" as const };
}

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: { providers: [claudeEntry, builtinEntry] },
    providers: [claudeEntry, builtinEntry],
    counts: { total: 2, installed: 1, binaryMissing: 0, unconfigured: 1 },
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    editor: defaultEditor(),
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

import { Route } from "./providers";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const ProvidersSettingsPage = (Route as any).component as () => ReactNode;

describe("ProvidersSettingsPage", () => {
  it("renders loading state while fetching", () => {
    pageState = makeState({ isLoading: true, envelope: null, providers: [] });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-loading")).toBeInTheDocument();
  });

  it("renders error state when the query fails", () => {
    pageState = makeState({
      envelope: null,
      providers: [],
      error: new Error("boom"),
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-error")).toHaveTextContent("boom");
  });

  it("renders counts in the status line and the catalog header", () => {
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-total")).toHaveTextContent("2 providers");
    expect(screen.getByTestId("settings-page-providers-installed")).toHaveTextContent(
      "1 installed"
    );
    expect(screen.getByTestId("settings-page-providers-unconfigured")).toHaveTextContent(
      "1 unconfigured"
    );
  });

  it("opens the create editor when clicking the new-provider action", () => {
    render(<ProvidersSettingsPage />);
    fireEvent.click(screen.getByTestId("settings-page-providers-create"));
    expect(pageState.openCreate).toHaveBeenCalled();
  });

  it("renders each provider row with settings, source metadata, and state tone", () => {
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-row-claude")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-providers-row-claude-command")).toHaveTextContent(
      "npx claude"
    );
    expect(
      screen.getByTestId("settings-page-providers-row-claude-api-key-state")
    ).toHaveTextContent("SET");
    expect(
      screen.getByTestId("settings-page-providers-row-claude-source-effective")
    ).toHaveTextContent("CONFIG");
    expect(
      screen.getByTestId("settings-page-providers-row-codex-source-effective")
    ).toHaveTextContent("BUILTIN");
    expect(screen.getByTestId("settings-page-providers-row-codex-api-key-state")).toHaveTextContent(
      "MISSING"
    );
  });

  it("disables delete for builtin-only providers", () => {
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-row-codex-delete")).toBeDisabled();
    expect(screen.getByTestId("settings-page-providers-row-claude-delete")).not.toBeDisabled();
  });

  it("invokes the edit and delete handlers from row controls", () => {
    render(<ProvidersSettingsPage />);
    fireEvent.click(screen.getByTestId("settings-page-providers-row-claude-edit"));
    expect(pageState.openEdit).toHaveBeenCalledWith(claudeEntry);

    fireEvent.click(screen.getByTestId("settings-page-providers-row-claude-delete"));
    expect(pageState.openDelete).toHaveBeenCalledWith(claudeEntry);
  });

  it("renders the empty state when the catalog is empty", () => {
    pageState = makeState({
      providers: [],
      envelope: { providers: [] },
      counts: { total: 0, installed: 0, binaryMissing: 0, unconfigured: 0 },
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-empty")).toBeInTheDocument();
  });

  it("renders the edit dialog seeded from the editor state", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "claude",
        draft: {
          name: "claude",
          command: "npx claude",
          default_model: "claude-opus",
          api_key_env: "ANTHROPIC_API_KEY",
        },
        entry: claudeEntry,
      },
      editorIsValid: true,
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-providers-editor-title")).toHaveTextContent(
      "Edit provider"
    );
    expect(screen.getByTestId("settings-providers-editor-name-input")).toBeDisabled();
    expect(screen.getByTestId("settings-providers-editor-command-input")).toHaveValue("npx claude");
    expect(screen.getByTestId("settings-providers-editor-source-effective")).toHaveTextContent(
      "CONFIG"
    );
  });

  it("surfaces validation errors returned by the mutation", () => {
    pageState = makeState({
      editor: {
        mode: "edit",
        name: "claude",
        draft: {
          name: "claude",
          command: "npx claude",
          default_model: "",
          api_key_env: "ANTHROPIC_API_KEY",
        },
        entry: claudeEntry,
      },
      editorError: "command must not be empty",
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-providers-editor-error")).toHaveTextContent(
      "command must not be empty"
    );
  });

  it("shows the builtin-fallback note in the delete dialog when fallback is present", () => {
    pageState = makeState({ deleteTarget: { mode: "open", entry: claudeEntry } });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-providers-delete-title")).toHaveTextContent(
      'Delete provider "claude"?'
    );
    expect(screen.getByTestId("settings-providers-delete-builtin")).toHaveTextContent(
      "Builtin provider will be revealed"
    );
  });

  it("reports the last save action with the restart badge", () => {
    pageState = makeState({
      lastAction: {
        kind: "saved",
        name: "claude",
        result: { restart_required: true },
      },
    });
    render(<ProvidersSettingsPage />);
    const banner = screen.getByTestId("settings-page-providers-action-result");
    expect(banner).toHaveTextContent('Saved provider "claude"');
    expect(banner).toHaveTextContent("restart required");
  });

  it("reports the fallback hint when a delete revealed a builtin", () => {
    pageState = makeState({
      lastAction: {
        kind: "deleted",
        name: "claude",
        result: { restart_required: true },
        hadFallback: true,
      },
    });
    render(<ProvidersSettingsPage />);
    const banner = screen.getByTestId("settings-page-providers-action-result");
    expect(banner).toHaveTextContent("builtin fallback now effective");
  });
});
