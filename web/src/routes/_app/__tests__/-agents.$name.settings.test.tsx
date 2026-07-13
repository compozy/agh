import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { validateAgentSettingsSearch } from "@/systems/agent";
import { primaryAgentFixture } from "@/systems/agent/testing";
import {
  buildSettingsDraftFromAgent,
  validateAgentSettingsDraft,
} from "@/systems/agent/lib/agent-settings-draft";

const mockUseAgentSettingsPage = vi.fn();

vi.mock("@tanstack/react-router", () => ({
  createFileRoute:
    () =>
    (opts: {
      component: () => ReactNode;
      validateSearch?: (search: Record<string, unknown>) => unknown;
    }) => ({
      component: opts.component,
      validateSearch: opts.validateSearch,
      useParams: () => ({ name: "codex-agent" }),
      useSearch: () => ({ section: "basics" }),
    }),
}));

vi.mock("@/systems/agent/hooks/use-agent-settings-page", () => ({
  useAgentSettingsPage: (args: unknown) => mockUseAgentSettingsPage(args),
}));

vi.mock("@/systems/agent/components/agent-settings-panels", () => ({
  AgentSettingsPanels: ({ section }: { section: string }) => (
    <div data-testid="agent-settings-panels" data-section={section} />
  ),
}));

import { Route } from "../agents.$name.settings";

const AgentSettingsRoute = (Route as unknown as { component: () => ReactNode }).component;

function makePage(overrides: Record<string, unknown> = {}) {
  const draft = buildSettingsDraftFromAgent(primaryAgentFixture);
  return {
    agent: primaryAgentFixture,
    agentLoading: false,
    agentError: null,
    draft,
    setDraft: vi.fn(),
    patchDraft: vi.fn(),
    dirty: false,
    validation: validateAgentSettingsDraft(draft),
    canSave: true,
    saveBlocked: false,
    saveBlockedCaption: undefined,
    section: "basics",
    setSection: vi.fn(),
    onSave: vi.fn(),
    onDiscard: vi.fn(),
    onReloadAndRetry: vi.fn(),
    onBackToDetail: vi.fn(),
    onOpenProviderSettings: vi.fn(),
    isSaving: false,
    saveError: null,
    conflictBanner: null,
    mutationDenied: false,
    fieldsDisabled: false,
    fieldsReadOnly: false,
    providerOptions: [],
    providersLoading: false,
    runtimeModels: [],
    modelCatalogLoading: false,
    modelCatalogLoaded: true,
    modelCatalogRefreshing: false,
    modelCatalogError: null,
    onRefreshCatalog: vi.fn(),
    workspaceName: "ws",
    deleteFlow: {
      open: false,
      openDialog: vi.fn(),
      closeDialog: vi.fn(),
      confirmDialog: null,
      isDeleting: false,
    },
    unsavedGuardDialog: null,
    ...overrides,
  };
}

describe("Agent settings route", () => {
  beforeEach(() => {
    mockUseAgentSettingsPage.mockReset();
    mockUseAgentSettingsPage.mockReturnValue(makePage());
  });

  it("Should validate section search defaults", () => {
    expect(validateAgentSettingsSearch({})).toEqual({ section: "basics" });
    expect(validateAgentSettingsSearch({ section: "danger" })).toEqual({ section: "danger" });
    expect(validateAgentSettingsSearch({ section: "nope" })).toEqual({ section: "basics" });
  });

  it("Should render a settings dialog with footer actions when pristine", () => {
    render(<AgentSettingsRoute />);
    expect(screen.getByTestId("agent-settings-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("agent-settings-page")).toBeInTheDocument();
    expect(screen.getByTestId("agent-settings-panels")).toHaveAttribute("data-section", "basics");
    expect(screen.getByTestId("agent-settings-footer-note")).toBeInTheDocument();
    expect(screen.getByTestId("agent-settings-save")).toBeDisabled();
    expect(screen.queryByTestId("agent-settings-unsaved")).not.toBeInTheDocument();
  });

  it("Should show Unsaved pill and enable Save when dirty", () => {
    mockUseAgentSettingsPage.mockReturnValue(makePage({ dirty: true }));
    render(<AgentSettingsRoute />);
    expect(screen.getByTestId("agent-settings-unsaved")).toBeInTheDocument();
    expect(screen.getByTestId("agent-settings-save")).not.toBeDisabled();
  });

  it("Should block save for unrecognized legacy permissions until an explicit choice", () => {
    const draft = {
      ...buildSettingsDraftFromAgent(primaryAgentFixture),
      permissions: null,
      legacyPermissions: "legacy-mode",
    };
    const validation = validateAgentSettingsDraft(draft);
    expect(validation.canSave).toBe(false);
    expect(validation.fields.permissions).toContain("Unrecognized permission mode: legacy-mode");
  });

  it("Should keep Save disabled with a caption when mutation is denied", () => {
    mockUseAgentSettingsPage.mockReturnValue(
      makePage({
        dirty: true,
        saveBlocked: true,
        saveBlockedCaption: "Editing is not permitted for this agent.",
        mutationDenied: true,
        fieldsReadOnly: true,
      })
    );
    render(<AgentSettingsRoute />);
    const save = screen.getByTestId("agent-settings-save");
    expect(save).toBeDisabled();
    expect(save).toHaveAttribute("title", "Editing is not permitted for this agent.");
  });

  it("Should navigate back when Cancel is pressed", async () => {
    const user = userEvent.setup();
    const onBackToDetail = vi.fn();
    mockUseAgentSettingsPage.mockReturnValue(makePage({ onBackToDetail }));
    render(<AgentSettingsRoute />);
    await user.click(screen.getByTestId("agent-settings-cancel"));
    expect(onBackToDetail).toHaveBeenCalled();
  });

  it("Should render nothing when the agent is missing so detail owns not-found", () => {
    mockUseAgentSettingsPage.mockReturnValue(
      makePage({ agent: null, draft: null, agentError: new Error("missing") })
    );
    const { container } = render(<AgentSettingsRoute />);
    expect(container).toBeEmptyDOMElement();
  });
});
