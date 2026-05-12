import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

type Envelope = typeof envelope;
type Mutation = {
  mutate: ReturnType<typeof vi.fn>;
  isPending: boolean;
  error: Error | null;
  data: { warnings: string[] } | undefined;
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

type UpdateStatus = {
  supported: boolean;
  managed: boolean;
  install_method: string;
  current_version: string;
  latest_version?: string;
  available: boolean;
  status: string;
  recommendation?: string;
  release_url?: string;
  checked_at?: string | null;
  last_error?: string;
};

const envelope = {
  section: "general" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  actions: {
    restart: { available: true, behavior: "action_trigger" as const, name: "restart" },
  },
  config: {
    daemon: { socket: "/tmp/agh.sock" },
    defaults: { agent: "general", provider: "claude", sandbox: "local" },
    http: { host: "127.0.0.1", port: 2123 },
    limits: { max_sessions: 10, max_concurrent_agents: 20 },
    permissions: { mode: "approve-all" as const },
    session_timeout: "0s",
  },
  config_paths: {
    daemon_info: "/tmp/daemon.json",
    global_config: "~/.agh/config.toml",
    global_mcp_sidecar: "~/.agh/mcp.json",
    home_dir: "~/.agh",
    log_file: "~/.agh/agh.log",
  },
  runtime: {
    active_agents: 7,
    active_sessions: 4,
    available: true,
    http_host: "127.0.0.1",
    http_port: 2123,
    socket: "~/.agh/daemon.sock",
    total_sessions: 12,
    uptime_seconds: 3600,
  },
};

let pageState: {
  isLoading: boolean;
  error: Error | null;
  envelope: Envelope | null;
  draft: Envelope["config"] | null;
  setDraft: ReturnType<typeof vi.fn>;
  isDirty: boolean;
  isSaving: boolean;
  saveError: string | null;
  warnings: string[] | undefined;
  lastAppliedLabel: string | null;
  handleReset: ReturnType<typeof vi.fn>;
  handleSave: ReturnType<typeof vi.fn>;
  restart: RestartBanner;
  update: {
    data: UpdateStatus | null;
    isLoading: boolean;
    isFetching: boolean;
    error: Error | null;
    refetch: ReturnType<typeof vi.fn>;
  };
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

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-settings-general-page", () => ({
  useSettingsGeneralPage: () => pageState,
}));

const mockMutation: Mutation = {
  mutate: vi.fn(),
  isPending: false,
  error: null,
  data: undefined,
};

vi.mock("@/systems/settings", () => ({
  useSettingsGeneral: () => ({ data: envelope, isLoading: false, error: null }),
  useUpdateSettingsGeneral: () => mockMutation,
  SettingsApiError: class SettingsApiError extends Error {},
}));

beforeEach(() => {
  pageState = {
    isLoading: false,
    error: null,
    envelope,
    draft: structuredClone(envelope.config),
    setDraft: vi.fn(),
    isDirty: false,
    isSaving: false,
    saveError: null,
    warnings: undefined,
    lastAppliedLabel: null,
    handleReset: vi.fn(),
    handleSave: vi.fn(),
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    update: {
      data: {
        supported: true,
        managed: false,
        install_method: "direct-binary",
        current_version: "v1.0.0",
        latest_version: "v1.1.0",
        available: true,
        status: "available",
        recommendation: "Run `agh update`.",
        release_url: "https://github.com/compozy/agh/releases/tag/v1.1.0",
        checked_at: "2026-05-03T19:00:00Z",
      },
      isLoading: false,
      isFetching: false,
      error: null,
      refetch: vi.fn(),
    },
  };
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../general";

const GeneralSettingsPage = routeComponent(Route);

describe("GeneralSettingsPage", () => {
  it("renders a loading indicator while fetching", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<GeneralSettingsPage />);
    expect(screen.getByTestId("settings-page-general-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<GeneralSettingsPage />);
    expect(screen.getByTestId("settings-page-general-error")).toHaveTextContent("boom");
  });

  it("renders runtime, defaults, permissions, and config path from the section envelope", () => {
    render(<GeneralSettingsPage />);

    expect(screen.getByTestId("settings-page-general-status-line")).toHaveTextContent(
      "config: ~/.agh/config.toml"
    );
    expect(screen.getByTestId("settings-page-general-default-agent-input")).toHaveValue("general");
    expect(screen.getByTestId("settings-page-general-default-provider-input")).toHaveValue(
      "claude"
    );
    expect(screen.getByTestId("settings-page-general-permissions-group")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-general-permission-approve-all")).toHaveAttribute(
      "aria-pressed",
      "true"
    );
    expect(screen.getByTestId("settings-page-general-update-status")).toHaveTextContent(
      "available"
    );
    expect(screen.getByTestId("settings-page-general-update-recommendation")).toHaveTextContent(
      "Run `agh update`."
    );
  });

  it("renders manual guidance and the last refresh error for unsupported update snapshots", () => {
    pageState.update.data = {
      supported: false,
      managed: false,
      install_method: "direct-binary",
      current_version: "v1.0.0",
      latest_version: "v1.1.0",
      available: true,
      status: "unsupported",
      recommendation:
        "Download the latest AGH Windows release archive and replace `agh.exe` manually.",
      release_url: "https://github.com/compozy/agh/releases/tag/v1.1.0",
      checked_at: "2026-05-03T19:00:00Z",
      last_error: "cached refresh failed",
    };

    render(<GeneralSettingsPage />);

    expect(screen.getByTestId("settings-page-general-update-status")).toHaveTextContent(
      "unsupported"
    );
    expect(screen.getByTestId("settings-page-general-update-recommendation")).toHaveTextContent(
      "replace `agh.exe` manually"
    );
    expect(screen.getByTestId("settings-page-general-update-last-error")).toHaveTextContent(
      "cached refresh failed"
    );
  });

  it("surfaces transport errors and retries the update query when refresh fails", () => {
    pageState.update = {
      data: null,
      isLoading: false,
      isFetching: false,
      error: new Error("update refresh timed out"),
      refetch: vi.fn(),
    };

    render(<GeneralSettingsPage />);

    expect(screen.getByTestId("settings-page-general-update-last-error")).toHaveTextContent(
      "update refresh timed out"
    );
    fireEvent.click(screen.getByTestId("settings-page-general-update-retry"));
    expect(pageState.update.refetch).toHaveBeenCalledTimes(1);
  });

  it("exposes the restart action button and triggers the restart mutation on click", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<GeneralSettingsPage />);
    const button = screen.getByRole("button", { name: "Restart daemon" });
    fireEvent.click(button);
    expect(pageState.restart.trigger).toHaveBeenCalledTimes(1);
  });

  it("renders the restart banner once the restart banner state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<GeneralSettingsPage />);
    expect(screen.getByTestId("settings-page-general-restart-banner")).toBeInTheDocument();
  });

  it("wires the save bar buttons to the page-level handlers", () => {
    pageState.isDirty = true;
    render(<GeneralSettingsPage />);

    fireEvent.click(screen.getByTestId("settings-page-general-save"));
    expect(pageState.handleSave).toHaveBeenCalled();

    fireEvent.click(screen.getByTestId("settings-page-general-reset"));
    expect(pageState.handleReset).toHaveBeenCalled();
  });

  it("surfaces the last-applied label when the save bar has a success message", () => {
    pageState.lastAppliedLabel = "Saved · restart required to apply";
    render(<GeneralSettingsPage />);
    expect(screen.getByTestId("settings-page-general-save-applied")).toHaveTextContent(
      "restart required"
    );
  });
});
