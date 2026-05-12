import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  SettingsExtensionEntry,
  SettingsHookEntry,
  SettingsHooksExtensionsSection,
  SettingsHooksExtensionsTransportParity,
} from "@/systems/settings";

type Envelope = SettingsHooksExtensionsSection;
type PolicyConfig = Envelope["config"];

const baseEnvelope: Envelope = {
  section: "hooks-extensions",
  scope: "global",
  available_scopes: ["global"],
  config: {
    marketplace: { registry: "github", base_url: "https://api.github.com" },
    resources: {
      allowed_kinds: ["snapshot", "artifact"],
      max_scope: "workspace",
      snapshot_rate_limit: { queue: 100, requests: 30, window: "5m" },
      operator_write_rate_limit: { queue: 20, requests: 10, window: "1m" },
    },
  },
  hooks: [
    {
      name: "pre-commit-lint",
      declaration: {
        name: "pre-commit-lint",
        event: "tool.pre_call",
        mode: "sync",
        command: "make",
        args: ["lint"],
        matcher: { tool_name: "Bash" },
        required: true,
      },
      source_metadata: {
        available_targets: ["global-config"],
        effective_source: { kind: "global-config", scope: "global" },
      },
    },
    {
      name: "slack-notify",
      declaration: {
        name: "slack-notify",
        event: "permission.denied",
        mode: "async",
        command: "node",
        args: ["./hooks/slack.js"],
        matcher: { agent_name: "coder" },
        required: false,
      },
      source_metadata: {
        available_targets: ["global-config"],
        effective_source: { kind: "global-config", scope: "global" },
      },
    },
  ],
  installed: [
    {
      name: "daytona",
      enabled: true,
      version: "1.2.3",
      state: "running",
      health: "healthy",
    },
  ],
  transport_parity: {
    known: true,
    settings_http: true,
    settings_uds: true,
    extensions_http: true,
    extensions_uds: true,
  },
};

const extensionEntry: SettingsExtensionEntry = {
  name: "daytona",
  enabled: true,
  version: "1.2.3",
  state: "running",
  source: "marketplace",
  type: "backend",
  daemon_running: true,
  health: "healthy",
  requires_env: ["DAYTONA_TOKEN"],
  missing_env: ["DAYTONA_TOKEN"],
};

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

type PageState = {
  isLoading: boolean;
  error: Error | null;
  envelope: Envelope | null;
  draft: PolicyConfig | null;
  hooks: SettingsHookEntry[];
  hooksCounts: { total: number; enabled: number };
  pendingHookName: string | null;
  toggleHookEnabled: ReturnType<typeof vi.fn>;
  hookError: string | null;
  canMutateHooks: boolean;
  extensions: SettingsExtensionEntry[];
  extensionsCounts: { total: number; enabled: number };
  extensionsLoading: boolean;
  extensionsError: string | null;
  pendingExtensionName: string | null;
  toggleExtensionEnabled: ReturnType<typeof vi.fn>;
  extensionActionError: string | null;
  canMutateExtensions: boolean;
  transportParity: SettingsHooksExtensionsTransportParity | null;
  isPolicyDirty: boolean;
  isSavingPolicy: boolean;
  savePolicyError: string | null;
  policyWarnings: string[] | undefined;
  canMutatePolicy: boolean;
  handleSavePolicy: ReturnType<typeof vi.fn>;
  handleResetPolicy: ReturnType<typeof vi.fn>;
  updatePolicyDraft: ReturnType<typeof vi.fn>;
  toggleAllowedKind: ReturnType<typeof vi.fn>;
  handleRetry: ReturnType<typeof vi.fn>;
  lastAction:
    | null
    | { kind: "saved"; result: { restart_required: boolean } }
    | {
        kind: "hook-toggled";
        name: string;
        enabled: boolean;
        result: { restart_required: boolean };
      }
    | { kind: "extension-toggled"; name: string; enabled: boolean };
  dismissLastAction: ReturnType<typeof vi.fn>;
  restart: RestartBanner;
};

let pageState: PageState;

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    isLoading: false,
    error: null,
    envelope: baseEnvelope,
    draft: structuredClone(baseEnvelope.config),
    hooks: baseEnvelope.hooks ?? [],
    hooksCounts: { total: 2, enabled: 1 },
    pendingHookName: null,
    toggleHookEnabled: vi.fn(),
    hookError: null,
    canMutateHooks: true,
    extensions: [extensionEntry],
    extensionsCounts: { total: 1, enabled: 1 },
    extensionsLoading: false,
    extensionsError: null,
    pendingExtensionName: null,
    toggleExtensionEnabled: vi.fn(),
    extensionActionError: null,
    canMutateExtensions: true,
    transportParity: baseEnvelope.transport_parity,
    isPolicyDirty: false,
    isSavingPolicy: false,
    savePolicyError: null,
    policyWarnings: undefined,
    canMutatePolicy: true,
    handleSavePolicy: vi.fn(),
    handleResetPolicy: vi.fn(),
    updatePolicyDraft: vi.fn(),
    toggleAllowedKind: vi.fn(),
    handleRetry: vi.fn(),
    lastAction: null,
    dismissLastAction: vi.fn(),
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    ...overrides,
  };
}

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-settings-hooks-extensions-page", () => ({
  useSettingsHooksExtensionsPage: () => pageState,
}));

beforeEach(() => {
  pageState = makeState();
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../hooks-extensions";

const HooksExtensionsSettingsPage = routeComponent(Route);

describe("HooksExtensionsSettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState = makeState({ isLoading: true, envelope: null, draft: null });
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState = makeState({
      envelope: null,
      draft: null,
      error: new Error("hooks boom"),
    });
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-error")).toHaveTextContent(
      "hooks boom"
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(pageState.handleRetry).toHaveBeenCalledTimes(1);
  });

  it("renders the status line with combined hook and extension counts", () => {
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-hooks-total")).toHaveTextContent(
      "1/2 hooks enabled"
    );
    expect(screen.getByTestId("settings-page-hooks-extensions-extensions-total")).toHaveTextContent(
      "1/1 extensions enabled"
    );
  });

  it("renders hook declarations with event, mode, and matcher summary", () => {
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-hooks-list")).toBeInTheDocument();
    expect(
      screen.getByTestId("settings-page-hooks-extensions-hooks-row-pre-commit-lint")
    ).toBeInTheDocument();
    expect(
      screen.getByTestId("settings-page-hooks-extensions-hooks-row-pre-commit-lint-matcher")
    ).toHaveTextContent("tool_name=Bash");
  });

  it("wires the hook enable switch through to toggleHookEnabled", () => {
    render(<HooksExtensionsSettingsPage />);
    fireEvent.click(
      screen.getByTestId("settings-page-hooks-extensions-hooks-row-slack-notify-toggle")
    );
    expect(pageState.toggleHookEnabled).toHaveBeenCalledTimes(1);
    const [entry, nextEnabled] = pageState.toggleHookEnabled.mock.calls[0];
    expect((entry as SettingsHookEntry).name).toBe("slack-notify");
    expect(nextEnabled).toBe(true);
  });

  it("renders installed extensions with state and health badges", () => {
    render(<HooksExtensionsSettingsPage />);
    expect(
      screen.getByTestId("settings-page-hooks-extensions-extensions-item-daytona")
    ).toHaveTextContent("running");
  });

  it("renders missing extension environment requirements by name only", () => {
    render(<HooksExtensionsSettingsPage />);
    const diagnostic = screen.getByTestId(
      "settings-page-hooks-extensions-extensions-item-daytona-missing-env"
    );
    expect(diagnostic).toHaveTextContent("Missing env: DAYTONA_TOKEN");
    expect(diagnostic).not.toHaveTextContent("secret");
  });

  it("disables the extension toggle when HTTP mutation parity is false", () => {
    pageState = makeState({
      canMutateExtensions: false,
      transportParity: {
        known: true,
        settings_http: true,
        settings_uds: true,
        extensions_http: false,
        extensions_uds: true,
      },
    });
    render(<HooksExtensionsSettingsPage />);
    const toggle = screen.getByTestId(
      "settings-page-hooks-extensions-extensions-item-daytona-toggle"
    );
    expect(toggle).toHaveAttribute("aria-disabled", "true");
    expect(
      screen.getByTestId("settings-page-hooks-extensions-transport-parity")
    ).toBeInTheDocument();
  });

  it("disables hook toggles when settings mutation parity is false", () => {
    pageState = makeState({
      canMutateHooks: false,
      transportParity: {
        known: true,
        settings_http: false,
        settings_uds: true,
        extensions_http: true,
        extensions_uds: true,
      },
    });
    render(<HooksExtensionsSettingsPage />);
    expect(
      screen.getByTestId("settings-page-hooks-extensions-hooks-row-slack-notify-toggle")
    ).toHaveAttribute("aria-disabled", "true");
  });

  it("invokes toggleExtensionEnabled when the extension switch is toggled off", () => {
    render(<HooksExtensionsSettingsPage />);
    fireEvent.click(
      screen.getByTestId("settings-page-hooks-extensions-extensions-item-daytona-toggle")
    );
    expect(pageState.toggleExtensionEnabled).toHaveBeenCalledTimes(1);
    const [entry, nextEnabled] = pageState.toggleExtensionEnabled.mock.calls[0];
    expect((entry as SettingsExtensionEntry).name).toBe("daytona");
    expect(nextEnabled).toBe(false);
  });

  it("wires policy save and reset controls", () => {
    pageState = makeState({ isPolicyDirty: true });
    render(<HooksExtensionsSettingsPage />);

    fireEvent.click(screen.getByTestId("settings-page-hooks-extensions-policy-save"));
    expect(pageState.handleSavePolicy).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("settings-page-hooks-extensions-policy-reset"));
    expect(pageState.handleResetPolicy).toHaveBeenCalledTimes(1);
  });

  it("surfaces policy save errors separately from extension action errors", () => {
    pageState = makeState({
      isPolicyDirty: true,
      savePolicyError: "forbidden",
      extensionActionError: "remote denied",
    });
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-policy-error")).toHaveTextContent(
      "forbidden"
    );
    expect(screen.getByTestId("settings-page-hooks-extensions-extensions-error")).toHaveTextContent(
      "remote denied"
    );
  });

  it("shows the saved action banner with restart-required wording", () => {
    pageState = makeState({
      lastAction: { kind: "saved", result: { restart_required: true } },
    });
    render(<HooksExtensionsSettingsPage />);
    expect(screen.getByTestId("settings-page-hooks-extensions-action-result")).toHaveTextContent(
      "restart required"
    );
  });

  it("shows the extension-toggled action banner with immediate wording", () => {
    pageState = makeState({
      lastAction: { kind: "extension-toggled", name: "daytona", enabled: false },
    });
    render(<HooksExtensionsSettingsPage />);
    const banner = screen.getByTestId("settings-page-hooks-extensions-action-result");
    expect(banner).toHaveAttribute("data-kind", "extension-toggled");
    expect(banner).toHaveTextContent('Extension "daytona" disabled');
    expect(banner).toHaveTextContent("applied immediately");
  });

  it("renders the @agh/ui Empty cards when no hooks and no extensions are registered", () => {
    pageState = makeState({
      hooks: [],
      hooksCounts: { total: 0, enabled: 0 },
      extensions: [],
      extensionsCounts: { total: 0, enabled: 0 },
    });
    render(<HooksExtensionsSettingsPage />);
    const hooksEmpty = screen.getByTestId("settings-page-hooks-extensions-hooks-empty");
    expect(hooksEmpty).toHaveAttribute("data-slot", "empty");
    expect(hooksEmpty).toHaveTextContent("No hooks registered");
    const extensionsEmpty = screen.getByTestId("settings-page-hooks-extensions-extensions-empty");
    expect(extensionsEmpty).toHaveAttribute("data-slot", "empty");
    expect(extensionsEmpty).toHaveTextContent("No extensions installed");
  });

  it("renders the action banner through @agh/ui Alert with role=status", () => {
    pageState = makeState({
      lastAction: { kind: "extension-toggled", name: "daytona", enabled: true },
    });
    render(<HooksExtensionsSettingsPage />);
    const banner = screen.getByTestId("settings-page-hooks-extensions-action-result");
    expect(banner).toHaveAttribute("data-slot", "alert");
    expect(banner).toHaveAttribute("role", "status");
  });

  it("renders allowed-kinds chips as active when selected in the draft", () => {
    render(<HooksExtensionsSettingsPage />);
    const snapshotChip = screen.getByTestId(
      "settings-page-hooks-extensions-policy-allowed-kinds-snapshot"
    );
    const sessionChip = screen.getByTestId(
      "settings-page-hooks-extensions-policy-allowed-kinds-session"
    );
    expect(snapshotChip).toHaveAttribute("data-active", "true");
    expect(sessionChip).toHaveAttribute("data-active", "false");

    fireEvent.click(sessionChip);
    expect(pageState.toggleAllowedKind).toHaveBeenCalledWith("session");
  });
});
