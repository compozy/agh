import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import type {
  VaultDeleteState,
  VaultDraft,
  VaultEditorState,
  VaultLastAction,
  VaultNamespaceFilter,
} from "@/hooks/routes/use-settings-vault-page";
import type { VaultListFilter, VaultSecret } from "@/systems/vault";

type PageState = {
  counts: { total: number; sessions: number; providers: number };
  deleteError: string | null;
  deleteIsPending: boolean;
  deleteTarget: VaultDeleteState;
  dismissLastAction: ReturnType<typeof vi.fn>;
  editor: VaultEditorState;
  editorError: string | null;
  editorIsSaving: boolean;
  editorIsValid: boolean;
  filter: VaultListFilter;
  isLoading: boolean;
  isRefetching: boolean;
  lastAction: VaultLastAction | null;
  namespace: VaultNamespaceFilter;
  prefix: string;
  queryError: string | null;
  refetch: ReturnType<typeof vi.fn>;
  secrets: VaultSecret[];
  setNamespace: ReturnType<typeof vi.fn>;
  setPrefix: ReturnType<typeof vi.fn>;
  closeDelete: ReturnType<typeof vi.fn>;
  closeEditor: ReturnType<typeof vi.fn>;
  confirmDelete: ReturnType<typeof vi.fn>;
  openCreate: ReturnType<typeof vi.fn>;
  openDelete: ReturnType<typeof vi.fn>;
  saveEditor: ReturnType<typeof vi.fn>;
  updateDraft: ReturnType<typeof vi.fn>;
};

const { mockUseSettingsVaultPage } = vi.hoisted(() => ({
  mockUseSettingsVaultPage: vi.fn(),
}));

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { component: () => ReactNode }) => ({
    component: opts.component,
  }),
}));

vi.mock("@/hooks/routes/use-settings-vault-page", () => ({
  useSettingsVaultPage: () => mockUseSettingsVaultPage(),
}));

const sessionSecret: VaultSecret = {
  ref: "vault:sessions/sess_123/github-token",
  namespace: "sessions",
  kind: "token",
  present: true,
  created_at: "2026-05-02T10:00:00Z",
  updated_at: "2026-05-02T10:00:00Z",
};

function makeState(overrides: Partial<PageState> = {}): PageState {
  return {
    counts: { total: 1, sessions: 1, providers: 0 },
    deleteError: null,
    deleteIsPending: false,
    deleteTarget: { mode: "closed" },
    dismissLastAction: vi.fn(),
    editor: { mode: "closed" },
    editorError: null,
    editorIsSaving: false,
    editorIsValid: false,
    filter: {},
    isLoading: false,
    isRefetching: false,
    lastAction: null,
    namespace: "all",
    prefix: "",
    queryError: null,
    refetch: vi.fn(),
    secrets: [sessionSecret],
    setNamespace: vi.fn(),
    setPrefix: vi.fn(),
    closeDelete: vi.fn(),
    closeEditor: vi.fn(),
    confirmDelete: vi.fn(),
    openCreate: vi.fn(),
    openDelete: vi.fn(),
    saveEditor: vi.fn(),
    updateDraft: vi.fn(),
    ...overrides,
  };
}

beforeEach(() => {
  mockUseSettingsVaultPage.mockReturnValue(makeState());
});

import { Route } from "./vault";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const VaultSettingsPage = (Route as any).component as () => ReactNode;

describe("VaultSettingsPage", () => {
  it("renders vault counts, filters, and redacted metadata rows", () => {
    render(<VaultSettingsPage />);

    expect(screen.getByTestId("settings-page-vault-total")).toHaveTextContent("1 secrets");
    expect(screen.getByTestId("settings-page-vault-sessions")).toHaveTextContent(
      "1 session-scoped"
    );
    expect(screen.getByTestId("settings-page-vault-filters")).toBeInTheDocument();
    expect(screen.getByTestId("settings-page-vault-table")).toHaveTextContent(sessionSecret.ref);
    expect(screen.getByTestId("settings-page-vault-table")).not.toHaveTextContent(
      "super-secret-token"
    );
  });

  it("forwards filter changes to the page state hook", () => {
    const setNamespace = vi.fn();
    const setPrefix = vi.fn();
    mockUseSettingsVaultPage.mockReturnValue(makeState({ setNamespace, setPrefix }));

    render(<VaultSettingsPage />);

    fireEvent.change(screen.getByTestId("settings-page-vault-namespace"), {
      target: { value: "sessions" },
    });
    fireEvent.change(screen.getByTestId("settings-page-vault-prefix"), {
      target: { value: "vault:sessions/sess_123/" },
    });

    expect(setNamespace).toHaveBeenCalledWith("sessions");
    expect(setPrefix).toHaveBeenCalledWith("vault:sessions/sess_123/");
  });

  it("opens the create action and renders the write-only editor state", () => {
    const draft: VaultDraft = {
      ref: "vault:sessions/sess_123/github-token",
      kind: "token",
      secretValue: "super-secret-token",
    };
    const openCreate = vi.fn();
    mockUseSettingsVaultPage.mockReturnValue(
      makeState({
        editor: { mode: "create", draft },
        editorIsValid: true,
        openCreate,
      })
    );

    const { container } = render(<VaultSettingsPage />);

    fireEvent.click(screen.getByTestId("settings-page-vault-create"));
    expect(openCreate).toHaveBeenCalled();
    expect(screen.getByTestId("settings-vault-editor-secret-value-input")).toHaveAttribute(
      "type",
      "password"
    );
    expect(container.textContent).not.toContain("super-secret-token");
  });

  it("confirms delete against the selected vault ref", () => {
    const confirmDelete = vi.fn();
    mockUseSettingsVaultPage.mockReturnValue(
      makeState({
        deleteTarget: { mode: "open", secret: sessionSecret },
        confirmDelete,
      })
    );

    render(<VaultSettingsPage />);

    expect(screen.getByTestId("settings-vault-delete-description")).toHaveTextContent(
      sessionSecret.ref
    );
    fireEvent.click(screen.getByTestId("settings-vault-delete-confirm"));
    expect(confirmDelete).toHaveBeenCalled();
  });
});
