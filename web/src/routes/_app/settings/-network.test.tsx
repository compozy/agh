import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const envelope = {
  section: "network" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  config: {
    enabled: true,
    port: 4222,
    default_channel: "agh",
    greet_interval: 30,
    max_payload: 131072,
    max_queue_depth: 1024,
    max_replay_age: 86400,
  },
  runtime: {
    available: true,
    enabled: true,
    status: "ready",
    listener_host: "127.0.0.1",
    listener_port: 4222,
    local_peers: 2,
    remote_peers: 1,
    channels: 4,
    queued_messages: 7,
    queued_sessions: 0,
    delivery_workers: 3,
  },
  links: [{ label: "network", path: "/network" }],
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

vi.mock("@/hooks/routes/use-settings-network-page", () => ({
  useSettingsNetworkPage: () => pageState,
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

import { Route } from "./network";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const NetworkSettingsPage = (Route as any).component as () => ReactNode;

describe("NetworkSettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("network boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-error")).toHaveTextContent("network boom");
  });

  it("renders runtime summary, listener, and delivery config from the envelope", () => {
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-status-line")).toHaveTextContent("ready");
    expect(screen.getByTestId("settings-page-network-runtime-listener")).toHaveTextContent(
      "127.0.0.1:4222"
    );
    expect(screen.getByTestId("settings-page-network-runtime-local-peers")).toHaveTextContent("2");
    expect(screen.getByTestId("settings-page-network-runtime-channels")).toHaveTextContent("4");
    expect(screen.getByTestId("settings-page-network-default-channel-input")).toHaveValue("agh");
    expect(screen.getByTestId("settings-page-network-port-input")).toHaveValue(4222);
    expect(screen.getByTestId("settings-page-network-max-queue-depth")).toHaveValue(1024);
  });

  it("wires save bar buttons to the restart-required page handlers", () => {
    pageState.isDirty = true;
    pageState.lastAppliedLabel = "Saved · restart required to apply";
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-save-applied")).toHaveTextContent(
      "restart required"
    );

    fireEvent.click(screen.getByTestId("settings-page-network-save"));
    expect(pageState.handleSave).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("settings-page-network-reset"));
    expect(pageState.handleReset).toHaveBeenCalledTimes(1);
  });

  it("deep-links to the operational Network route", () => {
    render(<NetworkSettingsPage />);
    const link = screen.getByTestId("settings-page-network-link-network");
    expect(link).toHaveAttribute("href", "/network");
  });

  it("renders the restart banner when the restart state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-restart-banner")).toBeInTheDocument();
  });
});
