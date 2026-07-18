import { fireEvent, screen } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import { settingsNetworkSectionFixture } from "@/systems/settings/mocks";
import type { SettingsNetworkSection } from "@/systems/settings";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const envelope = structuredClone(settingsNetworkSectionFixture);
type Envelope = SettingsNetworkSection;

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

import { routeComponent } from "@/test/route-options";
import { Route } from "../network";

const NetworkSettingsPage = routeComponent(Route);

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

  it("renders runtime truth and finite Live settings from the envelope", () => {
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-status-line")).toHaveTextContent("active");
    expect(screen.getByTestId("settings-page-network-runtime-live-participants")).toHaveTextContent(
      "2"
    );
    expect(screen.getByTestId("settings-page-network-runtime-channels")).toHaveTextContent("4");
    expect(screen.getByTestId("settings-page-network-enrollment-note")).toHaveTextContent(
      /do not opt/i
    );
    expect(screen.getByTestId("settings-page-network-max-replay-age")).toHaveValue("300");
    expect(screen.getByTestId("settings-page-network-live-default-max-wakes")).toHaveValue("8");
    expect(screen.getByTestId("settings-page-network-live-limit-max-wakes")).toHaveValue("64");
    expect(screen.getByTestId("settings-page-network-live-default-coalesce")).toHaveValue("500ms");
    expect(screen.getByText("Messages delivered")).toBeInTheDocument();
    expect(screen.getByLabelText("Network availability")).toBe(
      screen.getByTestId("settings-page-network-enabled-switch")
    );
    expect(screen.getByLabelText("Replay window")).toBe(
      screen.getByTestId("settings-page-network-max-replay-age")
    );
    expect(screen.queryByLabelText("Listener port")).toBeNull();
    expect(screen.queryByLabelText("Activation top K")).toBeNull();
    expect(screen.queryByLabelText("Digest flush interval")).toBeNull();
  });

  it("wires save bar buttons to the restart-required page handlers", () => {
    pageState.isDirty = true;
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-save-dirty")).toHaveTextContent(
      "Unsaved changes"
    );

    fireEvent.click(screen.getByTestId("settings-page-network-save"));
    expect(pageState.handleSave).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("settings-page-network-reset"));
    expect(pageState.handleReset).toHaveBeenCalledTimes(1);
  });

  it("routes a finite Live default edit through the settings draft", () => {
    pageState.isDirty = true;

    render(<NetworkSettingsPage />);

    fireEvent.change(screen.getByTestId("settings-page-network-live-default-max-wakes"), {
      target: { value: "12" },
    });

    expect(pageState.setDraft).toHaveBeenCalledTimes(1);
    const update = pageState.setDraft.mock.calls[0]?.[0] as (
      current: Envelope["config"]
    ) => Envelope["config"];
    const current = structuredClone(envelope.config);
    const next = update(current);
    expect(next).toEqual({
      ...current,
      live: {
        ...current.live,
        defaults: {
          ...current.live.defaults,
          max_wakes: 12,
        },
      },
    });
    expect(next.live.limits).toEqual(envelope.config.live.limits);
  });

  it("shows the committed Live number after the draft is discarded", () => {
    const view = render(<NetworkSettingsPage />);
    const input = screen.getByTestId("settings-page-network-live-default-max-wakes");

    fireEvent.change(input, { target: { value: "12" } });
    expect(input).toHaveValue("12");

    pageState.draft = {
      ...structuredClone(envelope.config),
      live: {
        ...structuredClone(envelope.config.live),
        defaults: { ...structuredClone(envelope.config.live.defaults), max_wakes: 12 },
      },
    };
    view.rerender(<NetworkSettingsPage />);

    pageState.draft = structuredClone(envelope.config);
    view.rerender(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-live-default-max-wakes")).toHaveValue("8");
  });

  it("normalizes duration whitespace before updating the settings draft", () => {
    render(<NetworkSettingsPage />);

    fireEvent.change(screen.getByTestId("settings-page-network-live-default-wake-time"), {
      target: { value: " 5m " },
    });

    const update = pageState.setDraft.mock.calls.at(-1)?.[0] as (
      current: Envelope["config"]
    ) => Envelope["config"];
    expect(update(structuredClone(envelope.config)).live.defaults.max_wake_wall_time).toBe("5m");
  });

  it("rejects token values outside the JavaScript safe integer range", () => {
    render(<NetworkSettingsPage />);
    const input = screen.getByTestId("settings-page-network-live-default-input-tokens");

    fireEvent.change(input, { target: { value: "9007199254740993" } });

    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(pageState.setDraft).not.toHaveBeenCalled();
  });

  it("surfaces the last-applied label when the save bar has a success message", () => {
    pageState.lastAppliedLabel = "Saved · restart required to apply";
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-save-applied")).toHaveTextContent(
      "restart required"
    );
  });

  it("deep-links to the operational Network route", () => {
    render(<NetworkSettingsPage />);
    const link = screen.getByTestId("settings-page-network-link-network");
    expect(link).toHaveAttribute("href", "/network");
    expect(screen.queryByTestId("settings-page-network-link-usage")).toBeNull();
  });

  it("renders the restart banner when the restart state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<NetworkSettingsPage />);
    expect(screen.getByTestId("settings-page-network-restart-banner")).toBeInTheDocument();
  });

  it("does not render unsupported network conversation controls", () => {
    render(<NetworkSettingsPage />);
    // Settings must surface only existing aggregate metrics and listener/delivery
    // primitives, never unsupported conversation lifecycle controls.
    expect(screen.queryByLabelText(/thread retention/i)).toBeNull();
    expect(screen.queryByLabelText(/retention policy/i)).toBeNull();
    expect(screen.queryByLabelText(/unread sync/i)).toBeNull();
    expect(screen.queryByLabelText(/notification preferences/i)).toBeNull();
    expect(screen.queryByLabelText(/mute channel/i)).toBeNull();
    expect(screen.queryByLabelText(/transcript export/i)).toBeNull();
    expect(screen.queryByLabelText(/direct room (retention|policy)/i)).toBeNull();
  });

  it("does not register settings testids for unsupported lifecycle features", () => {
    render(<NetworkSettingsPage />);
    expect(screen.queryByTestId("settings-page-network-thread-retention")).toBeNull();
    expect(screen.queryByTestId("settings-page-network-unread-sync")).toBeNull();
    expect(screen.queryByTestId("settings-page-network-notification-prefs")).toBeNull();
    expect(screen.queryByTestId("settings-page-network-mute-rules")).toBeNull();
    expect(screen.queryByTestId("settings-page-network-transcript-export")).toBeNull();
  });
});
