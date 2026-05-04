import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { storyCompany } from "@/storybook/fintech-scenario";

const envelope = {
  section: "skills" as const,
  scope: "global" as const,
  available_scopes: ["global"] as const,
  runtime_available: true,
  discovered_count: 12,
  disabled_count: 2,
  config: {
    enabled: true,
    disabled_skills: ["alpha", "beta"],
    poll_interval: "5m",
    marketplace: {
      registry: "agh",
      base_url: storyCompany.registryBaseUrl,
    },
    allowed_marketplace_mcp: ["mcp-one"],
    allowed_marketplace_hooks: [],
  },
  links: [{ label: "skills", path: "/skills" }],
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
  toggleDisabled: ReturnType<typeof vi.fn>;
  isDisabledDirty: boolean;
  isPolicyDirty: boolean;
  isSavingDisabled: boolean;
  isSavingPolicy: boolean;
  saveDisabledError: string | null;
  savePolicyError: string | null;
  disabledWarnings: string[] | undefined;
  policyWarnings: string[] | undefined;
  lastDisabledLabel: string | null;
  lastPolicyLabel: string | null;
  handleResetDisabled: ReturnType<typeof vi.fn>;
  handleResetPolicy: ReturnType<typeof vi.fn>;
  handleSaveDisabled: ReturnType<typeof vi.fn>;
  handleSavePolicy: ReturnType<typeof vi.fn>;
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

vi.mock("@/hooks/routes/use-settings-skills-page", () => ({
  useSettingsSkillsPage: () => pageState,
}));

beforeEach(() => {
  pageState = {
    isLoading: false,
    error: null,
    envelope,
    draft: structuredClone(envelope.config),
    setDraft: vi.fn(),
    toggleDisabled: vi.fn(),
    isDisabledDirty: false,
    isPolicyDirty: false,
    isSavingDisabled: false,
    isSavingPolicy: false,
    saveDisabledError: null,
    savePolicyError: null,
    disabledWarnings: undefined,
    policyWarnings: undefined,
    lastDisabledLabel: null,
    lastPolicyLabel: null,
    handleResetDisabled: vi.fn(),
    handleResetPolicy: vi.fn(),
    handleSaveDisabled: vi.fn(),
    handleSavePolicy: vi.fn(),
    handleRetry: vi.fn(),
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
  };
});

import { Route } from "./skills";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const SkillsSettingsPage = (Route as any).component as () => ReactNode;

describe("SkillsSettingsPage", () => {
  it("renders a loading indicator during the initial fetch", () => {
    pageState.isLoading = true;
    pageState.envelope = null;
    pageState.draft = null;
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("skills boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-error")).toHaveTextContent("skills boom");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(pageState.handleRetry).toHaveBeenCalledTimes(1);
  });

  it("renders disabled-skill items and marketplace fields separately from the status line", () => {
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-status-line")).toHaveTextContent(
      "12 discovered"
    );
    expect(screen.getByTestId("settings-page-skills-disabled-list")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-skills-disabled-item-alpha")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-skills-marketplace-registry-input")).toHaveValue(
      "agh"
    );
    expect(screen.getByTestId("settings-page-skills-marketplace-base-url-input")).toHaveValue(
      storyCompany.registryBaseUrl
    );
    expect(screen.getByTestId("settings-page-skills-allowed-mcp-input")).toHaveValue("mcp-one");
  });

  it("shows the applied label for disabled skills when the draft is clean", () => {
    pageState.lastDisabledLabel = "Saved · applied immediately";
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-disabled-applied")).toHaveTextContent(
      "applied immediately"
    );
  });

  it("wires disabled-skill save controls and prioritizes dirty state over stale labels", () => {
    pageState.isDisabledDirty = true;
    pageState.lastDisabledLabel = "Saved · applied immediately";
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-disabled-dirty")).toHaveTextContent(
      "Unsaved changes"
    );

    fireEvent.click(screen.getByTestId("settings-page-skills-disabled-save"));
    expect(pageState.handleSaveDisabled).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByTestId("settings-page-skills-disabled-reset"));
    expect(pageState.handleResetDisabled).toHaveBeenCalledTimes(1);
  });

  it("shows the applied label for policy settings when the draft is clean", () => {
    pageState.lastPolicyLabel = "Saved · restart required to apply";
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-policy-applied")).toHaveTextContent(
      "restart required"
    );
  });

  it("wires policy save controls and prioritizes dirty state over stale labels", () => {
    pageState.isPolicyDirty = true;
    pageState.lastPolicyLabel = "Saved · restart required to apply";
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-policy-dirty")).toHaveTextContent(
      "Unsaved changes"
    );

    fireEvent.click(screen.getByTestId("settings-page-skills-policy-save"));
    expect(pageState.handleSavePolicy).toHaveBeenCalledTimes(1);
  });

  it("toggles a disabled-skill entry via the per-item switch", () => {
    render(<SkillsSettingsPage />);
    fireEvent.click(screen.getByTestId("settings-page-skills-disabled-toggle-alpha"));
    expect(pageState.toggleDisabled).toHaveBeenCalledWith("alpha");
  });

  it("deep-links to the operational Skills route for managing skills", () => {
    render(<SkillsSettingsPage />);
    const link = screen.getByTestId("settings-page-skills-link-skills");
    expect(link).toHaveAttribute("href", "/skills");
  });

  it("renders the restart banner when the restart banner state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-restart-banner")).toBeInTheDocument();
  });

  it("shows a save error when the disabled-skills mutation fails", () => {
    pageState.saveDisabledError = "server exploded";
    render(<SkillsSettingsPage />);
    expect(screen.getByTestId("settings-page-skills-disabled-error")).toHaveTextContent(
      "server exploded"
    );
  });

  it("renders the @agh/ui Empty card when no skills are disabled", () => {
    pageState.envelope = {
      ...envelope,
      disabled_count: 0,
      config: { ...envelope.config, disabled_skills: [] },
    };
    pageState.draft = { ...envelope.config, disabled_skills: [] };
    render(<SkillsSettingsPage />);
    const empty = screen.getByTestId("settings-page-skills-disabled-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No skills installed");
  });
});
