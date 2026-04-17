import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const envelope = {
  section: "automation" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  config: {
    enabled: true,
    timezone: "UTC",
    max_concurrent_jobs: 8,
    default_fire_limit: { max: 5, window: "1m" },
  },
  runtime: {
    available: true,
    running: true,
    scheduler_running: true,
    job_enabled: 3,
    job_total: 5,
    trigger_enabled: 2,
    trigger_total: 4,
    last_synced_at: "2026-04-17T10:00:00Z",
    next_fire: "2026-04-17T12:00:00Z",
  },
  links: [{ label: "automation", path: "/automation" }],
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

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
  Link: ({
    children,
    to,
    ...rest
  }: {
    children: ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...(rest as Record<string, unknown>)}>
      {children}
    </a>
  ),
}));

vi.mock("@/hooks/routes/use-settings-automation-page", () => ({
  useSettingsAutomationPage: () => pageState,
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
  };
});

import { Route } from "./automation";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const AutomationSettingsPage = (Route as any).component as () => ReactNode;

describe("AutomationSettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<AutomationSettingsPage />);
    expect(screen.getByTestId("settings-page-automation-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("automation boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<AutomationSettingsPage />);
    expect(screen.getByTestId("settings-page-automation-error")).toHaveTextContent(
      "automation boom"
    );
  });

  it("renders the manager summary and engine/limits config from the envelope", () => {
    render(<AutomationSettingsPage />);
    expect(screen.getByTestId("settings-page-automation-status-line")).toHaveTextContent(
      "3/5 jobs active"
    );
    expect(screen.getByTestId("settings-page-automation-runtime-engine")).toHaveTextContent(
      "running"
    );
    expect(screen.getByTestId("settings-page-automation-runtime-jobs")).toHaveTextContent("3 / 5");
    expect(screen.getByTestId("settings-page-automation-timezone-input")).toHaveValue("UTC");
    expect(screen.getByTestId("settings-page-automation-max-concurrent-input")).toHaveValue(8);
    expect(screen.getByTestId("settings-page-automation-fire-limit-window-input")).toHaveValue(
      "1m"
    );
  });

  it("wires save bar buttons to the restart-required page handlers", () => {
    pageState.isDirty = true;
    pageState.lastAppliedLabel = "Saved · restart required to apply";
    render(<AutomationSettingsPage />);
    expect(screen.getByTestId("settings-page-automation-save-applied")).toHaveTextContent(
      "restart required"
    );

    fireEvent.click(screen.getByTestId("settings-page-automation-save"));
    expect(pageState.handleSave).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("settings-page-automation-reset"));
    expect(pageState.handleReset).toHaveBeenCalledTimes(1);
  });

  it("deep-links to the operational Automation route", () => {
    render(<AutomationSettingsPage />);
    const link = screen.getByTestId("settings-page-automation-link-automation");
    expect(link).toHaveAttribute("href", "/automation");
  });

  it("renders the restart banner when the restart state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<AutomationSettingsPage />);
    expect(screen.getByTestId("settings-page-automation-restart-banner")).toBeInTheDocument();
  });
});
