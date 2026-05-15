import { fireEvent, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { ProviderInspectorState } from "@/hooks/routes/use-settings-providers-page";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ProviderDraft, SettingsProviderEntry } from "@/systems/settings";
import { settingsProviderFixtures } from "@/systems/settings/mocks/fixtures";
import {
  DEFAULT_PROVIDER_FILTERS,
  type ProviderFilterState,
} from "@/systems/settings/lib/providers-list-filters";

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
  command_available: true,
  settings: {
    command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
    models: {
      default: "claude-sonnet-4-6",
      curated: [{ id: "claude-sonnet-4-6" }, { id: "claude-haiku-4-5" }],
    },
    auth_mode: "native_cli",
    env_policy: "filtered",
    home_policy: "operator",
    auth_status_command: "claude auth status",
    auth_login_command: "claude login",
  },
  auth_status: {
    mode: "native_cli",
    env_policy: "filtered",
    home_policy: "operator",
    state: "native_cli",
    message: "Provider owns authentication through its native CLI login state.",
    status_command: "claude auth status",
    login_command: "claude login",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "global-config", scope: "global" },
    shadowed_sources: [{ kind: "builtin-provider", scope: "global" }],
  },
  fallback: {
    settings: { command: "npx -y @agentclientprotocol/claude-agent-acp@latest" },
    source: { kind: "builtin-provider", scope: "global" },
  },
};

const builtinEntry: SettingsProviderEntry = {
  name: "codex",
  default: false,
  command_available: true,
  settings: {
    command: "npx -y @zed-industries/codex-acp@latest",
    models: {
      default: "gpt-5.4",
      curated: [
        {
          id: "gpt-5.4",
          supports_reasoning: true,
          reasoning_efforts: ["low", "medium", "high"],
        },
        { id: "gpt-5.4-mini" },
      ],
    },
    auth_mode: "bound_secret",
    env_policy: "filtered",
    home_policy: "operator",
    credential_slots: [
      {
        name: "api_key",
        target_env: "OPENAI_API_KEY",
        secret_ref: "env:OPENAI_API_KEY",
        kind: "api_key",
        required: true,
      },
    ],
  },
  credentials: [
    {
      name: "api_key",
      target_env: "OPENAI_API_KEY",
      secret_ref: "env:OPENAI_API_KEY",
      kind: "api_key",
      required: true,
      present: false,
      source: "env",
    },
  ],
  auth_status: {
    mode: "bound_secret",
    env_policy: "filtered",
    home_policy: "operator",
    state: "missing_required",
    message: "Missing required AGH-managed provider credential.",
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
  filteredProviders: SettingsProviderEntry[];
  filters: ProviderFilterState;
  setStatusFilter: ReturnType<typeof vi.fn>;
  setSourceFilter: ReturnType<typeof vi.fn>;
  setHarnessFilter: ReturnType<typeof vi.fn>;
  setAuthModeFilter: ReturnType<typeof vi.fn>;
  setDefaultFilter: ReturnType<typeof vi.fn>;
  setNameQuery: ReturnType<typeof vi.fn>;
  counts: { total: number; installed: number; binaryMissing: number; unconfigured: number };
  restart: RestartBanner;
  inspector: ProviderInspectorState;
  inspectorIsValid: boolean;
  inspectorError: string | null;
  inspectorWarnings: string[] | undefined;
  inspectorIsSaving: boolean;
  openInspect: ReturnType<typeof vi.fn>;
  openCreate: ReturnType<typeof vi.fn>;
  switchToEdit: ReturnType<typeof vi.fn>;
  cancelEdit: ReturnType<typeof vi.fn>;
  closeInspector: ReturnType<typeof vi.fn>;
  updateDraft: ReturnType<typeof vi.fn>;
  saveInspector: ReturnType<typeof vi.fn>;
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

vi.mock("@/systems/model-catalog", async () => {
  const actual =
    await vi.importActual<typeof import("@/systems/model-catalog")>("@/systems/model-catalog");
  return {
    ...actual,
    useProviderModels: () => ({
      data: undefined,
      isLoading: false,
      isFetching: false,
      error: null,
    }),
    useProviderModelStatus: () => ({
      data: { sources: [] },
      isLoading: false,
      isFetching: false,
      error: null,
    }),
    useRefreshProviderModels: () => ({
      mutate: vi.fn(),
      mutateAsync: vi.fn(),
      isPending: false,
      error: null,
    }),
  };
});

function defaultInspector(): ProviderInspectorState {
  return { mode: "closed" };
}

function makeState(overrides: Partial<PageState> = {}): PageState {
  const providers = overrides.providers ?? [claudeEntry, builtinEntry];
  return {
    isLoading: false,
    error: null,
    envelope: { providers },
    providers,
    filteredProviders: overrides.filteredProviders ?? providers,
    filters: overrides.filters ?? DEFAULT_PROVIDER_FILTERS,
    setStatusFilter: vi.fn(),
    setSourceFilter: vi.fn(),
    setHarnessFilter: vi.fn(),
    setAuthModeFilter: vi.fn(),
    setDefaultFilter: vi.fn(),
    setNameQuery: vi.fn(),
    counts: { total: providers.length, installed: 1, binaryMissing: 0, unconfigured: 1 },
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    inspector: defaultInspector(),
    inspectorIsValid: false,
    inspectorError: null,
    inspectorWarnings: undefined,
    inspectorIsSaving: false,
    openInspect: vi.fn(),
    openCreate: vi.fn(),
    switchToEdit: vi.fn(),
    cancelEdit: vi.fn(),
    closeInspector: vi.fn(),
    updateDraft: vi.fn(),
    saveInspector: vi.fn(),
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

import { routeComponent } from "@/test/route-options";
import { Route } from "../providers";

const ProvidersSettingsPage = routeComponent(Route);

const draftFor = (entry: SettingsProviderEntry): ProviderDraft => ({
  name: entry.name,
  command: entry.settings.command ?? "",
  display_name: entry.settings.display_name ?? "",
  model_default: entry.settings.models?.default ?? "",
  curated_models: (entry.settings.models?.curated ?? [])
    .map(model => model.id)
    .filter(Boolean)
    .join("\n"),
  curated_snapshot: (entry.settings.models?.curated ?? []).map(model => ({ ...model })),
  target_env: entry.settings.credential_slots?.[0]?.target_env ?? "",
  harness: entry.settings.harness ?? "acp",
  runtime_provider: entry.settings.runtime_provider ?? "",
  transport: entry.settings.transport ?? "",
  base_url: entry.settings.base_url ?? "",
  auth_mode: entry.settings.auth_mode ?? "native_cli",
  env_policy: entry.settings.env_policy ?? "filtered",
  home_policy: entry.settings.home_policy ?? "operator",
  auth_status_command: entry.settings.auth_status_command ?? "",
  auth_login_command: entry.settings.auth_login_command ?? "",
  secret_ref: entry.settings.credential_slots?.[0]?.secret_ref ?? "",
  secret_value: "",
  credential_slots: (entry.settings.credential_slots ?? []).map(slot => ({ ...slot })),
  credential_secret_values: (entry.settings.credential_slots ?? []).map(() => ""),
});

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

  it("opens the create flow when clicking the new-provider action", () => {
    render(<ProvidersSettingsPage />);
    fireEvent.click(screen.getByTestId("settings-page-providers-create"));
    expect(pageState.openCreate).toHaveBeenCalled();
  });

  it("restores focus to the new-provider action after the create sheet closes", async () => {
    pageState = makeState({ inspector: { mode: "create", draft: draftFor(builtinEntry) } });
    const { rerender } = render(<ProvidersSettingsPage />);
    const trigger = screen.getByTestId("settings-page-providers-create");

    screen.getByTestId("provider-inspector-cancel").focus();
    expect(screen.getByTestId("provider-inspector-cancel")).toHaveFocus();

    pageState = makeState();
    rerender(<ProvidersSettingsPage />);

    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it("renders each provider card with identity, summary, and state tone", () => {
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-card-claude")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-providers-card-claude-name")).toHaveTextContent(
      "claude"
    );
    expect(screen.getByTestId("settings-page-providers-card-claude-model")).toHaveTextContent(
      "claude-sonnet-4-6"
    );
    expect(screen.getByTestId("settings-page-providers-card-claude-auth-state")).toHaveTextContent(
      "native_cli"
    );
    expect(
      screen.getByTestId("settings-page-providers-card-claude-source-effective")
    ).toHaveTextContent("CONFIG");
    expect(
      screen.getByTestId("settings-page-providers-card-codex-source-effective")
    ).toHaveTextContent("BUILTIN");
    expect(screen.getByTestId("settings-page-providers-card-claude-status")).toHaveAttribute(
      "data-state",
      "installed"
    );
    expect(screen.getByTestId("settings-page-providers-card-codex-status")).toHaveAttribute(
      "data-state",
      "unconfigured"
    );
  });

  it("surfaces the inline hint when credentials are missing", () => {
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("settings-page-providers-card-codex-hint")).toHaveTextContent(
      "Bind OPENAI_API_KEY to continue."
    );
  });

  it("renders the newly supported ACP provider cards from the catalog", () => {
    pageState = makeState({
      envelope: { providers: settingsProviderFixtures },
      providers: settingsProviderFixtures,
      counts: {
        total: settingsProviderFixtures.length,
        installed: 0,
        binaryMissing: 0,
        unconfigured: 0,
      },
    });

    render(<ProvidersSettingsPage />);

    const expectedProviders = [
      "blackbox",
      "cline",
      "goose",
      "hermes",
      "junie",
      "kimi-cli",
      "openclaw",
      "openhands",
      "qoder",
      "qwen-code",
    ];
    for (const provider of expectedProviders) {
      expect(screen.getByTestId(`settings-page-providers-card-${provider}`)).toBeInTheDocument();
    }
  });

  it("invokes openInspect when the card open action is clicked", () => {
    render(<ProvidersSettingsPage />);
    fireEvent.click(screen.getByTestId("settings-page-providers-card-claude-open"));
    expect(pageState.openInspect).toHaveBeenCalledWith(claudeEntry);
  });

  it("renders the inspector sheet with the entry config when in inspect mode", () => {
    pageState = makeState({ inspector: { mode: "inspect", entry: claudeEntry } });
    render(<ProvidersSettingsPage />);
    const sheet = screen.getByTestId("provider-inspector-sheet");
    expect(sheet).toHaveAttribute("data-mode", "inspect");
    expect(screen.getByTestId("provider-inspector-title")).toHaveTextContent("claude");
    expect(screen.getByTestId("inspect-command")).toHaveTextContent(
      "npx -y @agentclientprotocol/claude-agent-acp@latest"
    );
    expect(screen.getByTestId("inspect-auth-mode")).toHaveTextContent("native_cli");
  });

  it("uses the provider configuration subtitle when display name is blank", () => {
    pageState = makeState({
      inspector: {
        mode: "inspect",
        entry: { ...claudeEntry, settings: { ...claudeEntry.settings, display_name: "" } },
      },
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByText("Provider configuration")).toBeInTheDocument();
  });

  it("renders the edit form inside the sheet when inspector is in edit mode", () => {
    pageState = makeState({
      inspector: {
        mode: "edit",
        entry: claudeEntry,
        draft: draftFor(claudeEntry),
        cameFrom: "inspect",
      },
      inspectorIsValid: true,
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("provider-inspector-sheet")).toHaveAttribute("data-mode", "edit");
    expect(screen.getByTestId("settings-providers-editor-name-input")).toBeDisabled();
    expect(screen.getByTestId("settings-providers-editor-command-input")).toHaveValue(
      "npx -y @agentclientprotocol/claude-agent-acp@latest"
    );
  });

  it("clears draft credential secrets when auth mode leaves bound secret", async () => {
    const draft: ProviderDraft = {
      ...draftFor(builtinEntry),
      auth_mode: "bound_secret",
      target_env: "OPENAI_API_KEY",
      secret_ref: "vault:providers/codex/api-key",
      secret_value: "sk-primary",
      credential_slots: [
        {
          name: "api_key",
          target_env: "OPENAI_API_KEY",
          secret_ref: "vault:providers/codex/api-key",
          kind: "api_key",
          required: true,
        },
        {
          name: "organization",
          target_env: "OPENAI_ORG_ID",
          secret_ref: "vault:providers/codex/organization",
          kind: "organization",
          required: false,
        },
      ],
      credential_secret_values: ["sk-primary", "org-secret"],
    };
    const updateDraft = vi.fn();
    pageState = makeState({
      inspector: { mode: "edit", entry: builtinEntry, draft, cameFrom: "inspect" },
      updateDraft,
    });
    const user = userEvent.setup();
    render(<ProvidersSettingsPage />);

    await user.selectOptions(
      screen.getByTestId("settings-providers-editor-auth-mode-input"),
      "native_cli"
    );

    expect(updateDraft).toHaveBeenCalledTimes(1);
    const updater = updateDraft.mock.calls[0]?.[0] as
      | ((current: ProviderDraft) => ProviderDraft)
      | undefined;
    if (!updater) {
      throw new Error("expected provider draft updater");
    }
    const next = updater(draft);
    expect(next.auth_mode).toBe("native_cli");
    expect(next.target_env).toBe("");
    expect(next.secret_ref).toBe("");
    expect(next.secret_value).toBe("");
    expect(next.credential_slots).toEqual([]);
    expect(next.credential_secret_values).toEqual([]);
  });

  it("disables delete in inspect footer for builtin-only providers", () => {
    pageState = makeState({ inspector: { mode: "inspect", entry: builtinEntry } });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("provider-inspector-delete")).toBeDisabled();
  });

  it("invokes openDelete from inspect footer for deletable providers", async () => {
    pageState = makeState({ inspector: { mode: "inspect", entry: claudeEntry } });
    const user = userEvent.setup();
    render(<ProvidersSettingsPage />);
    await user.click(screen.getByTestId("provider-inspector-delete"));
    expect(pageState.openDelete).toHaveBeenCalledWith(claudeEntry);
  });

  it("switches the inspector into edit mode via the inspect footer", () => {
    pageState = makeState({ inspector: { mode: "inspect", entry: claudeEntry } });
    render(<ProvidersSettingsPage />);
    fireEvent.click(screen.getByTestId("provider-inspector-edit"));
    expect(pageState.switchToEdit).toHaveBeenCalled();
  });

  it("disables the edit action while delete is pending", async () => {
    pageState = makeState({
      inspector: { mode: "inspect", entry: claudeEntry },
      deleteIsPending: true,
    });
    const user = userEvent.setup();
    render(<ProvidersSettingsPage />);
    const edit = screen.getByTestId("provider-inspector-edit");
    expect(edit).toBeDisabled();

    await user.click(edit);

    expect(pageState.switchToEdit).not.toHaveBeenCalled();
  });

  it("surfaces inspector validation errors in the sheet footer", () => {
    pageState = makeState({
      inspector: {
        mode: "edit",
        entry: claudeEntry,
        draft: draftFor(claudeEntry),
        cameFrom: "inspect",
      },
      inspectorError: "command must not be empty",
    });
    render(<ProvidersSettingsPage />);
    expect(screen.getByTestId("provider-inspector-error")).toHaveTextContent(
      "command must not be empty"
    );
  });

  it("renders the @agh/ui Empty card when the catalog is empty", () => {
    pageState = makeState({
      providers: [],
      envelope: { providers: [] },
      counts: { total: 0, installed: 0, binaryMissing: 0, unconfigured: 0 },
    });
    render(<ProvidersSettingsPage />);
    const empty = screen.getByTestId("settings-page-providers-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No providers configured");
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
