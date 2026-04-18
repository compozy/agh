import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const envelope = {
  section: "memory" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  actions: {
    consolidate: {
      available: true,
      behavior: "action_trigger" as const,
      name: "consolidate",
    },
  },
  config: {
    dream: {
      agent: "general",
      check_interval: "30m",
      enabled: true,
      min_hours: 24,
      min_sessions: 3,
    },
    enabled: true,
    global_dir: "~/.agh/memory",
  },
  health: {
    available: true,
    dream_enabled: true,
    file_count: 42,
    last_consolidated_at: "2026-04-17T14:00:00Z",
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
  handleConsolidate: ReturnType<typeof vi.fn>;
  isConsolidating: boolean;
  actionMessage: string | null;
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
}));

vi.mock("@/hooks/routes/use-settings-memory-page", () => ({
  useSettingsMemoryPage: () => pageState,
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
    handleConsolidate: vi.fn(),
    isConsolidating: false,
    actionMessage: null,
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
  };
});

import { Route } from "./memory";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const MemorySettingsPage = (Route as any).component as () => ReactNode;

describe("MemorySettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-loading")).toBeInTheDocument();
  });

  it("renders the error state when the memory query fails", () => {
    pageState.error = new Error("memory boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-error")).toHaveTextContent("memory boom");
  });

  it("renders config, dream fields, and health metadata from the envelope", () => {
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-global-dir-input")).toHaveValue(
      "~/.agh/memory"
    );
    expect(screen.getByTestId("settings-page-memory-dream-agent")).toHaveValue("general");
    expect(screen.getByTestId("settings-page-memory-status-line")).toHaveTextContent(
      "42 memory files"
    );
    expect(screen.getByTestId("settings-page-memory-last-consolidated")).toHaveTextContent(
      "last run"
    );
  });

  it("triggers the consolidate action when the user clicks Trigger now", () => {
    render(<MemorySettingsPage />);
    const button = screen.getByTestId("settings-page-memory-consolidate");
    fireEvent.click(button);
    expect(pageState.handleConsolidate).toHaveBeenCalledTimes(1);
  });

  it("disables the consolidate button while consolidation is pending", () => {
    pageState.isConsolidating = true;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-consolidate")).toBeDisabled();
  });

  it("shows the returned action message from the consolidate action", () => {
    pageState.actionMessage = "Consolidation triggered";
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-action-message")).toHaveTextContent(
      "Consolidation triggered"
    );
  });

  it("disables the consolidate button when the action is unavailable via envelope flags", () => {
    pageState.envelope = {
      ...envelope,
      actions: {
        consolidate: { ...envelope.actions.consolidate, available: false },
      },
    };
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-consolidate")).toBeDisabled();
  });
});
