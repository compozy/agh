import { fireEvent, screen, within } from "@testing-library/react";
import { renderWithTopbar as render } from "@/test/render-with-topbar";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { buildRolesViewModel, type RoleViewModel } from "@/systems/settings";
import {
  rolesStatusFixture,
  rolesStatusWithDiagnosticFixture,
  settingsRolesConfigFixture,
} from "@/systems/settings/mocks";

const defaultRoles = buildRolesViewModel(rolesStatusFixture.roles, settingsRolesConfigFixture);
const diagnosticRoles = buildRolesViewModel(
  rolesStatusWithDiagnosticFixture.roles,
  settingsRolesConfigFixture
);

const restartBanner = {
  isVisible: false,
  isRestartRequired: false,
  isPolling: false,
  isSuccessful: false,
  isFailed: false,
  operationId: null,
  status: null,
  activeSessionCount: 0,
  lastMutation: null,
  trigger: vi.fn(),
  isTriggerPending: false,
  triggerError: null,
  dismiss: vi.fn(),
};

let pageState: {
  isLoading: boolean;
  isEmpty: boolean;
  error: Error | null;
  roles: RoleViewModel[];
  isDirty: boolean;
  isInvalid: boolean;
  draftRevision: number;
  validationErrors: Record<string, string>;
  isSaving: boolean;
  saveError: string | null;
  warnings: string[] | undefined;
  lastAppliedLabel: string | null;
  restart: typeof restartBanner;
  setRoleField: ReturnType<typeof vi.fn>;
  setNumberFieldValidity: ReturnType<typeof vi.fn>;
  addFallback: ReturnType<typeof vi.fn>;
  removeFallback: ReturnType<typeof vi.fn>;
  updateFallback: ReturnType<typeof vi.fn>;
  registerFieldRef: ReturnType<typeof vi.fn>;
  handleSave: ReturnType<typeof vi.fn>;
  handleReset: ReturnType<typeof vi.fn>;
  handleRetry: ReturnType<typeof vi.fn>;
};

vi.mock("@tanstack/react-router", async importOriginal => {
  const actual = await importOriginal<typeof import("@tanstack/react-router")>();
  return { ...actual, useNavigate: () => vi.fn() };
});

vi.mock("@/systems/settings/hooks/use-settings-roles-page", () => ({
  useSettingsRolesPage: () => pageState,
}));

beforeEach(() => {
  pageState = {
    isLoading: false,
    isEmpty: false,
    error: null,
    roles: defaultRoles,
    isDirty: false,
    isInvalid: false,
    draftRevision: 0,
    validationErrors: {},
    isSaving: false,
    saveError: null,
    warnings: undefined,
    lastAppliedLabel: null,
    restart: { ...restartBanner, trigger: vi.fn(), dismiss: vi.fn() },
    setRoleField: vi.fn(),
    setNumberFieldValidity: vi.fn(() => vi.fn()),
    addFallback: vi.fn(),
    removeFallback: vi.fn(),
    updateFallback: vi.fn(),
    registerFieldRef: vi.fn(() => vi.fn()),
    handleSave: vi.fn(),
    handleReset: vi.fn(),
    handleRetry: vi.fn(),
  };
});

import { RolesSettingsPage } from "../-roles-settings-page";

function group(role: string): HTMLElement {
  return screen.getByTestId(`settings-page-roles-group-${role}`);
}

describe("RolesSettingsPage", () => {
  it("renders a loading indicator while either read is pending", () => {
    pageState.isLoading = true;
    render(<RolesSettingsPage />);
    expect(screen.getByTestId("settings-page-roles-loading")).toBeInTheDocument();
  });

  it("renders the empty-projection anomaly with retry only", () => {
    pageState.isEmpty = true;
    render(<RolesSettingsPage />);
    expect(screen.getByTestId("settings-page-roles-empty")).toHaveTextContent(
      "Roles unavailable — no role projection was returned."
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(pageState.handleRetry).toHaveBeenCalledTimes(1);
  });

  it("renders the server error with a retry action", () => {
    pageState.error = new Error("roles service unavailable");
    render(<RolesSettingsPage />);
    expect(screen.getByTestId("settings-page-roles-error")).toHaveTextContent(
      "roles service unavailable"
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    expect(pageState.handleRetry).toHaveBeenCalledTimes(1);
  });

  it("renders six role panels in product order with builtin badges (UT-074)", () => {
    render(<RolesSettingsPage />);

    const groups = screen.getAllByTestId(/^settings-page-roles-group-/);
    expect(groups.map(node => node.dataset.testid)).toEqual([
      "settings-page-roles-group-coordinator",
      "settings-page-roles-group-dream",
      "settings-page-roles-group-checkpoint_summary",
      "settings-page-roles-group-memory_extractor",
      "settings-page-roles-group-auto_title",
      "settings-page-roles-group-memory_controller",
    ]);
    for (const role of ["coordinator", "dream", "checkpoint_summary"]) {
      expect(within(group(role)).getByText("BUILTIN")).toBeInTheDocument();
    }
    expect(within(group("coordinator")).getByText("OFF")).toBeInTheDocument();
  });

  it("shows inherit badge and resolves-at-invocation for inherit roles (UT-075)", () => {
    render(<RolesSettingsPage />);

    for (const role of ["auto_title", "memory_extractor"]) {
      expect(within(group(role)).getByText("INHERIT")).toBeInTheDocument();
      expect(screen.getByTestId(`settings-page-roles-${role}-resolution`)).toHaveTextContent(
        "Resolves at invocation."
      );
      // Null routing fields render as unresolved, never as a fabricated default.
      expect(screen.getByTestId(`settings-page-roles-${role}-model-input`)).toHaveValue("");
    }
  });

  it("renders the timeout input only for memory_controller (UT-076)", () => {
    render(<RolesSettingsPage />);

    expect(screen.queryByTestId("settings-page-roles-dream-timeout-input")).not.toBeInTheDocument();
    expect(
      screen.getByTestId("settings-page-roles-memory_controller-timeout-input")
    ).toBeInTheDocument();
    // memory_controller has no agent field.
    expect(
      screen.queryByTestId("settings-page-roles-memory_controller-agent-input")
    ).not.toBeInTheDocument();
  });

  it("surfaces a role diagnostic as a visible warning on the affected row (UT-077)", () => {
    pageState.roles = diagnosticRoles;
    render(<RolesSettingsPage />);

    const notice = screen.getByTestId("settings-page-roles-dream-diagnostics-role_agent_not_found");
    expect(notice).toHaveTextContent("Warning");
    expect(notice).toHaveTextContent("ghost");
  });

  it("renders the editable fallback chain and wires add to the page handler", () => {
    render(<RolesSettingsPage />);

    fireEvent.click(screen.getByTestId("settings-page-roles-dream-advanced-fallback-add"));
    expect(pageState.addFallback).toHaveBeenCalledWith("dream");
  });

  it("wires the shared save bar to the page handlers", () => {
    pageState.isDirty = true;
    render(<RolesSettingsPage />);

    fireEvent.click(screen.getByTestId("settings-page-roles-save"));
    expect(pageState.handleSave).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId("settings-page-roles-reset"));
    expect(pageState.handleReset).toHaveBeenCalledTimes(1);
  });
});
