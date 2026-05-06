import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { settingsMemoryConfigFixture } from "@/systems/settings/mocks/fixtures";
import { MemorySettingsPage } from "./memory";

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
    ...settingsMemoryConfigFixture,
    dream: { ...settingsMemoryConfigFixture.dream, agent: "dreaming-curator" },
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
  handleTriggerDream: ReturnType<typeof vi.fn>;
  isTriggeringDream: boolean;
  actionMessage: string | null;
  handleRetry: ReturnType<typeof vi.fn>;
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
  createFileRoute: () => (opts: unknown) => opts,
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
    handleTriggerDream: vi.fn(),
    isTriggeringDream: false,
    actionMessage: null,
    handleRetry: vi.fn(),
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
  };
});

describe("MemorySettingsPage", () => {
  it("Should render a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-loading")).toBeInTheDocument();
  });

  it("Should render the error state when the memory query fails", () => {
    pageState.error = new Error("memory boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-error")).toHaveTextContent("memory boom");
  });

  it("Should render system, dream, and health metadata from the envelope", () => {
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-global-dir-input")).toHaveValue(
      "~/.agh/memory"
    );
    expect(screen.getByTestId("settings-page-memory-dream-agent-input")).toHaveValue(
      "dreaming-curator"
    );
    expect(screen.getByTestId("settings-page-memory-status-line")).toHaveTextContent(
      "42 memory files"
    );
    expect(screen.getByTestId("settings-page-memory-last-consolidated")).toHaveTextContent(
      "last dream"
    );
  });

  it("Should render the controller, controller LLM, and recall configuration sections", () => {
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-controller-mode-input")).toHaveValue("hybrid");
    expect(
      screen.getByTestId("settings-page-memory-controller-policy-allow-origins-input")
    ).toHaveValue("cli, http, uds, tool, extractor, dreaming, file, provider");
    expect(screen.getByTestId("settings-page-memory-controller-llm-model-input")).toHaveValue(
      "anthropic/claude-haiku-4"
    );
    expect(screen.getByTestId("settings-page-memory-recall-top-k-input")).toHaveValue("5");
    expect(screen.getByTestId("settings-page-memory-recall-weight-bm25-unicode-input")).toHaveValue(
      "0.55"
    );
  });

  it("Should render extractor, decisions, session ledger, daily, file caps, and workspace identity", () => {
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-extractor-mode-input")).toHaveValue(
      "post_message"
    );
    expect(screen.getByTestId("settings-page-memory-extractor-inbox-path-input")).toHaveAttribute(
      "readonly"
    );
    expect(screen.getByTestId("settings-page-memory-decisions-prune-after-input")).toHaveValue(
      "90"
    );
    expect(screen.getByTestId("settings-page-memory-session-ledger-format-input")).toHaveValue(
      "jsonl"
    );
    expect(screen.getByTestId("settings-page-memory-session-ledger-root-input")).toHaveAttribute(
      "readonly"
    );
    expect(screen.getByTestId("settings-page-memory-daily-max-bytes-input")).toHaveValue(
      String(settingsMemoryConfigFixture.daily.max_bytes)
    );
    expect(screen.getByTestId("settings-page-memory-file-max-lines-input")).toHaveValue("200");
    expect(screen.getByTestId("settings-page-memory-workspace-toml-path-input")).toHaveAttribute(
      "readonly"
    );
  });

  it("Should render provider resilience controls from the envelope", () => {
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-provider-name-input")).toHaveValue("");
    expect(screen.getByTestId("settings-page-memory-provider-timeout-input")).toHaveValue("2s");
    expect(screen.getByTestId("settings-page-memory-provider-failure-threshold-input")).toHaveValue(
      "5"
    );
    expect(screen.getByTestId("settings-page-memory-provider-cooldown-input")).toHaveValue("30s");
  });

  it("Should trigger the dream action when the operator clicks Trigger dream", () => {
    render(<MemorySettingsPage />);
    const button = screen.getByTestId("settings-page-memory-dream-trigger");
    fireEvent.click(button);
    expect(pageState.handleTriggerDream).toHaveBeenCalledTimes(1);
  });

  it("Should disable the Trigger dream button while a dream is pending", () => {
    pageState.isTriggeringDream = true;
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-dream-trigger")).toBeDisabled();
  });

  it("Should render the action message returned by the dream action", () => {
    pageState.actionMessage = "Dream triggered";
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-action-message")).toHaveTextContent(
      "Dream triggered"
    );
  });

  it("Should disable the Trigger dream button when the action is unavailable via envelope flags", () => {
    pageState.envelope = {
      ...envelope,
      actions: {
        consolidate: { ...envelope.actions.consolidate, available: false },
      },
    };
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-dream-trigger")).toBeDisabled();
  });

  it("Should disable the Trigger dream button when dreaming is disabled in the draft", () => {
    pageState.draft = {
      ...envelope.config,
      dream: { ...envelope.config.dream, enabled: false },
    };
    render(<MemorySettingsPage />);
    expect(screen.getByTestId("settings-page-memory-dream-trigger")).toBeDisabled();
  });
});
