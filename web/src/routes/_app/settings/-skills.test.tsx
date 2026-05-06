import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { UIProvider } from "@agh/ui";
import { storyCompany } from "@/storybook/fintech-scenario";
import type { AgentPayload } from "@/systems/agent";
import type { SettingsSkillsSection } from "@/systems/settings";
import type { WorkspacePayload } from "@/systems/workspace";

type Envelope = SettingsSkillsSection;

const envelope: Envelope = {
  section: "skills" as const,
  scope: "global" as const,
  available_scopes: ["global"],
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

const agentFixture: AgentPayload = {
  name: "coder",
  provider: "codex",
  prompt: "Review code.",
};

const workspaceFixture: WorkspacePayload = {
  id: "ws-polybot",
  name: "polybot",
  root_dir: "/workspace/polybot",
  add_dirs: [],
  created_at: "2026-05-04T21:00:00Z",
  updated_at: "2026-05-04T21:00:00Z",
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
  availableScopes: readonly ("global" | "workspace" | "agent")[];
  selection: { scope: "global" } | { scope: "agent"; agentName: string; workspaceId?: string };
  agents: AgentPayload[];
  workspaces: WorkspacePayload[];
  selectedAgent: AgentPayload | null;
  selectedWorkspaceContext: WorkspacePayload | null;
  selectGlobal: ReturnType<typeof vi.fn>;
  selectAgentScope: ReturnType<typeof vi.fn>;
  selectAgent: ReturnType<typeof vi.fn>;
  selectWorkspaceContext: ReturnType<typeof vi.fn>;
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
    availableScopes: ["global"],
    selection: { scope: "global" },
    agents: [agentFixture],
    workspaces: [workspaceFixture],
    selectedAgent: null,
    selectedWorkspaceContext: null,
    selectGlobal: vi.fn(),
    selectAgentScope: vi.fn(),
    selectAgent: vi.fn(),
    selectWorkspaceContext: vi.fn(),
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
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-loading")).toBeInTheDocument();
  });

  it("renders the error state when the query fails", () => {
    pageState.error = new Error("skills boom");
    pageState.envelope = null;
    pageState.draft = null;
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-error")).toHaveTextContent("skills boom");
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(pageState.handleRetry).toHaveBeenCalledTimes(1);
  });

  it("renders disabled-skill items and marketplace fields separately from the status line", () => {
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-status-line")).toHaveTextContent(
      "12 discovered"
    );
    expect(screen.getByTestId("settings-page-skills-scope-label")).toHaveTextContent(
      "scope: global"
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
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-disabled-applied")).toHaveTextContent(
      "applied immediately"
    );
  });

  it("wires disabled-skill save controls and prioritizes dirty state over stale labels", () => {
    pageState.isDisabledDirty = true;
    pageState.lastDisabledLabel = "Saved · applied immediately";
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
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
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-policy-applied")).toHaveTextContent(
      "restart required"
    );
  });

  it("wires policy save controls and prioritizes dirty state over stale labels", () => {
    pageState.isPolicyDirty = true;
    pageState.lastPolicyLabel = "Saved · restart required to apply";
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-policy-dirty")).toHaveTextContent(
      "Unsaved changes"
    );

    fireEvent.click(screen.getByTestId("settings-page-skills-policy-save"));
    expect(pageState.handleSavePolicy).toHaveBeenCalledTimes(1);
  });

  it("toggles a disabled-skill entry via the per-item switch", () => {
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    fireEvent.click(screen.getByTestId("settings-page-skills-disabled-toggle-alpha"));
    expect(pageState.toggleDisabled).toHaveBeenCalledWith("alpha");
  });

  it("deep-links to the operational Skills route for managing skills", () => {
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    const link = screen.getByTestId("settings-page-skills-link-skills");
    expect(link).toHaveAttribute("href", "/skills");
  });

  it("renders agent scope controls and hides policy writes when scoped to one agent", () => {
    pageState.envelope = {
      ...envelope,
      scope: "agent",
      agent_name: "coder",
      workspace_id: "ws-polybot",
      available_scopes: ["global", "agent"],
      config: { ...envelope.config, disabled_skills: ["review"] },
    };
    pageState.draft = { ...envelope.config, disabled_skills: ["review"] };
    pageState.availableScopes = ["global", "agent"];
    pageState.selection = { scope: "agent", agentName: "coder", workspaceId: "ws-polybot" };
    pageState.selectedAgent = agentFixture;
    pageState.selectedWorkspaceContext = workspaceFixture;

    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );

    expect(screen.getByTestId("settings-page-skills-scope-label")).toHaveTextContent(
      "scope: agent coder"
    );
    expect(screen.getByTestId("settings-page-skills-workspace-context-summary")).toHaveTextContent(
      "context: polybot"
    );
    expect(screen.getByTestId("settings-agent-select")).toHaveTextContent("coder");
    expect(screen.getByTestId("settings-page-skills-workspace-context-input")).toHaveValue(
      "ws-polybot"
    );
    expect(screen.getByTestId("settings-page-skills-agent-policy-note")).toBeInTheDocument();
    expect(
      screen.queryByTestId("settings-page-skills-marketplace-registry-input")
    ).not.toBeInTheDocument();
  });

  it("wires the scope controls to the page hook callbacks", () => {
    pageState.availableScopes = ["global", "agent"];

    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );

    fireEvent.click(screen.getByTestId("settings-page-skills-scope-agent"));
    expect(pageState.selectAgentScope).toHaveBeenCalledTimes(1);
  });

  it("wires the agent selectors to the page hook callbacks", async () => {
    const user = userEvent.setup();
    pageState.availableScopes = ["global", "agent"];
    pageState.selection = { scope: "agent", agentName: "coder", workspaceId: "ws-polybot" };
    pageState.selectedAgent = agentFixture;
    pageState.selectedWorkspaceContext = workspaceFixture;
    pageState.agents = [agentFixture, { name: "writer", provider: "claude", prompt: "" }];

    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );

    await user.click(screen.getByTestId("settings-agent-select"));
    await user.click(screen.getByTestId("agent-command-item-writer"));
    expect(pageState.selectAgent).toHaveBeenCalledWith("writer");

    fireEvent.change(screen.getByTestId("settings-page-skills-workspace-context-input"), {
      target: { value: "" },
    });
    expect(pageState.selectWorkspaceContext).toHaveBeenCalledWith("");
  });

  it("renders the restart banner when the restart banner state reports visible", () => {
    pageState.restart.isVisible = true;
    pageState.restart.isRestartRequired = true;
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    expect(screen.getByTestId("settings-page-skills-restart-banner")).toBeInTheDocument();
  });

  it("shows a save error when the disabled-skills mutation fails", () => {
    pageState.saveDisabledError = "server exploded";
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
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
    render(
      <UIProvider reducedMotion="always">
        <SkillsSettingsPage />
      </UIProvider>
    );
    const empty = screen.getByTestId("settings-page-skills-disabled-empty");
    expect(empty).toBeInTheDocument();
    expect(empty).toHaveAttribute("data-slot", "empty");
    expect(empty).toHaveTextContent("No skills installed");
  });
});
