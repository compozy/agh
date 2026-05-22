import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithTopbar } from "@/test/render-with-topbar";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const envelope = {
  section: "observability" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  config: {
    enabled: true,
    max_global_bytes: 1024 * 1024 * 1024,
    retention_days: 7,
    transcripts: {
      enabled: true,
      max_bytes_per_session: 256 * 1024 * 1024,
      segment_bytes: 1024 * 1024,
    },
  },
  log_tail: {
    available: true,
    stream_url: "/api/settings/observability/log-tail" as string | undefined,
    transport: "sse" as "sse" | undefined,
  },
  runtime: {
    active_agents: 2,
    active_sessions: 4,
    available: true,
    global_db_size_bytes: 180 * 1024 * 1024,
    session_db_size_bytes: 132 * 1024 * 1024,
    uptime_seconds: 3600,
  },
};

type Envelope = typeof envelope;
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

const supportBundleMock = vi.hoisted(() => ({
  create: vi.fn(),
  reset: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/systems/support", () => ({
  useSupportBundleDownload: () => ({
    create: supportBundleMock.create,
    operation: null,
    isPending: false,
    error: null,
    reset: supportBundleMock.reset,
  }),
}));

vi.mock("@/hooks/routes/use-settings-observability-page", () => ({
  useSettingsObservabilityPage: () => pageState,
}));

beforeEach(() => {
  supportBundleMock.create.mockReset();
  supportBundleMock.create.mockResolvedValue(undefined);
  supportBundleMock.reset.mockReset();
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
  };
});

import { routeComponent } from "@/test/route-options";
import { Route } from "../observability";

const ObservabilitySettingsPage = routeComponent(Route);

function render(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return renderWithTopbar(<QueryClientProvider client={client}>{ui}</QueryClientProvider>);
}

describe("ObservabilitySettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<ObservabilitySettingsPage />);
    expect(screen.getByTestId("settings-page-observability-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("observe boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<ObservabilitySettingsPage />);
    expect(screen.getByTestId("settings-page-observability-error")).toHaveTextContent(
      "observe boom"
    );
  });

  it("renders config, DB metrics, and log-tail metadata from the envelope", () => {
    render(<ObservabilitySettingsPage />);

    expect(screen.getByTestId("settings-page-observability-retention-days")).toHaveValue("7");
    expect(screen.getByTestId("settings-page-observability-storage-summary")).toHaveTextContent(
      "storage"
    );
    expect(screen.getByTestId("settings-page-observability-usage-breakdown")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-observability-log-tail-transport")).toHaveTextContent(
      "transport: sse"
    );
  });

  it("links to the log-tail stream URL when the capability is available", () => {
    render(<ObservabilitySettingsPage />);

    const link = screen.getByTestId("settings-page-observability-log-tail-link");
    expect(link).toHaveAttribute("href", "/api/settings/observability/log-tail");
  });

  it("renders support bundle consent before daemon download", () => {
    render(<ObservabilitySettingsPage />);

    expect(screen.getByTestId("settings-page-observability-support-bundle")).toBeInTheDocument();
    expect(
      screen.getByTestId("settings-page-observability-support-bundle-status")
    ).toHaveTextContent("status: idle");
    expect(
      screen.getByTestId("settings-page-observability-support-bundle-consent")
    ).not.toBeChecked();
  });

  it("sends support bundle consent after checkbox approval", async () => {
    const user = userEvent.setup();
    render(<ObservabilitySettingsPage />);

    await user.click(screen.getByTestId("settings-page-observability-support-bundle-consent"));
    await user.click(screen.getByTestId("settings-page-observability-support-bundle-button"));

    expect(supportBundleMock.create).toHaveBeenCalledWith({ includeStatus: true, yes: true });
  });

  it("renders the overview metric grid via @agh/ui Metric", () => {
    render(<ObservabilitySettingsPage />);

    const sessions = screen.getByTestId("settings-page-observability-metric-sessions");
    expect(sessions).toHaveAttribute("data-slot", "metric");
    expect(sessions).toHaveTextContent("Active sessions");
    expect(sessions).toHaveTextContent("4");

    const storage = screen.getByTestId("settings-page-observability-metric-storage");
    expect(storage).toHaveAttribute("data-slot", "metric");
    expect(storage).toHaveTextContent("Storage used");
  });

  it("marks log-tail unavailable when the envelope reports no capability", () => {
    pageState.envelope = {
      ...envelope,
      log_tail: { available: false, stream_url: undefined, transport: undefined },
    };
    render(<ObservabilitySettingsPage />);

    const block = screen.getByTestId("settings-page-observability-log-tail");
    expect(block).toHaveAttribute("data-available", "false");
    expect(
      screen.queryByTestId("settings-page-observability-log-tail-link")
    ).not.toBeInTheDocument();
  });
});
