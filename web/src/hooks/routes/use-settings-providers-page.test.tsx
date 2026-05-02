import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@tanstack/react-router", () => ({
  useMatchRoute: () => () => false,
}));

vi.mock("@/systems/settings/adapters/settings-api", () => ({
  getSettingsRestartStatus: vi.fn(),
  listSettingsProviders: vi.fn(),
  putSettingsProvider: vi.fn(),
  deleteSettingsProvider: vi.fn(),
  triggerSettingsRestart: vi.fn(),
  SettingsApiError: class SettingsApiError extends Error {
    status = 500;
    constructor(message: string, status: number) {
      super(message);
      this.status = status;
    }
  },
}));

import {
  deleteSettingsProvider,
  listSettingsProviders,
  putSettingsProvider,
  SettingsApiError,
} from "@/systems/settings/adapters/settings-api";
import { initialSettingsRestartState } from "@/systems/settings/stores/settings-restart-store";
import { useSettingsRestartStore } from "@/systems/settings/stores/use-settings-restart-store";
import type { SettingsProviderCollection } from "@/systems/settings";
import { useSettingsProvidersPage } from "./use-settings-providers-page";

const claudeEntry: SettingsProviderCollection["providers"][number] = {
  name: "claude",
  default: true,
  command_available: true,
  settings: {
    command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
    default_model: "claude-sonnet-4-6",
    credential_slots: [
      {
        name: "api_key",
        target_env: "ANTHROPIC_API_KEY",
        secret_ref: "env:ANTHROPIC_API_KEY",
        kind: "api_key",
        required: false,
      },
    ],
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "global-config", scope: "global" },
    shadowed_sources: [{ kind: "builtin-provider", scope: "global" }],
  },
  fallback: {
    settings: {
      command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
      default_model: "claude-sonnet-4-6",
    },
    source: { kind: "builtin-provider", scope: "global" },
  },
};

const codexEntry: SettingsProviderCollection["providers"][number] = {
  name: "codex",
  default: false,
  command_available: true,
  settings: {
    command: "npx -y @zed-industries/codex-acp@latest",
    default_model: "gpt-5.4",
    credential_slots: [
      {
        name: "api_key",
        target_env: "OPENAI_API_KEY",
        secret_ref: "env:OPENAI_API_KEY",
        kind: "api_key",
        required: true,
      },
    ],
  },
  credentials: [
    {
      name: "api_key",
      target_env: "OPENAI_API_KEY",
      secret_ref: "env:OPENAI_API_KEY",
      kind: "api_key",
      required: true,
      present: false,
      source: "env",
    },
  ],
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "builtin-provider", scope: "global" },
  },
};

const collection: SettingsProviderCollection = {
  collection: "providers",
  scope: "global",
  available_scopes: ["global"],
  providers: [claudeEntry, codexEntry],
};

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });

  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);

  return { queryClient, wrapper };
}

beforeEach(() => {
  vi.clearAllMocks();
  useSettingsRestartStore.setState({
    ...initialSettingsRestartState,
    startRestart: useSettingsRestartStore.getState().startRestart,
    updateRestart: useSettingsRestartStore.getState().updateRestart,
    clearRestart: useSettingsRestartStore.getState().clearRestart,
    recordMutation: useSettingsRestartStore.getState().recordMutation,
  });
  vi.mocked(listSettingsProviders).mockResolvedValue(collection);
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("useSettingsProvidersPage", () => {
  it("exposes the provider collection and installation counts", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => {
      expect(result.current.providers).toHaveLength(2);
    });

    expect(result.current.counts).toEqual({
      total: 2,
      installed: 1,
      binaryMissing: 0,
      unconfigured: 1,
    });
  });

  it("opens create editor with an empty draft", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
    });

    expect(result.current.editor).toMatchObject({
      mode: "create",
      draft: { name: "", command: "", default_model: "", target_env: "" },
    });
  });

  it("blocks create save when the name collides with an existing provider", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
      result.current.updateDraft(draft => ({ ...draft, name: "Claude" }));
    });

    expect(result.current.editorIsValid).toBe(false);
  });

  it("opens edit editor seeded from the entry and keeps the selected item", async () => {
    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openEdit(claudeEntry);
    });

    expect(result.current.editor).toMatchObject({
      mode: "edit",
      name: "claude",
      draft: expect.objectContaining({
        command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
        default_model: "claude-sonnet-4-6",
        target_env: "ANTHROPIC_API_KEY",
      }),
    });
  });

  it("submits a full replacement on save and records the last action", async () => {
    vi.mocked(putSettingsProvider).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openEdit(claudeEntry);
    });
    act(() => {
      result.current.updateDraft(draft => ({ ...draft, default_model: "claude-haiku" }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsProvider).toHaveBeenCalledWith("claude", {
      settings: {
        command: "npx -y @agentclientprotocol/claude-agent-acp@latest",
        default_model: "claude-haiku",
        harness: "acp",
        credential_slots: [
          {
            name: "api_key",
            target_env: "ANTHROPIC_API_KEY",
            secret_ref: "env:ANTHROPIC_API_KEY",
            kind: "api_key",
            required: false,
          },
        ],
      },
    });
    expect(result.current.editor.mode).toBe("closed");
  });

  it("preserves additional credential slots when editing provider metadata", async () => {
    vi.mocked(putSettingsProvider).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });
    const providerWithMultipleCredentials: SettingsProviderCollection["providers"][number] = {
      ...claudeEntry,
      name: "openrouter",
      settings: {
        command: "npx -y pi-acp@latest",
        default_model: "openai/gpt-5.4",
        harness: "pi_acp",
        runtime_provider: "openrouter",
        credential_slots: [
          {
            name: "api_key",
            target_env: "OPENROUTER_API_KEY",
            secret_ref: "vault:providers/openrouter/api-key",
            kind: "api_key",
            required: false,
          },
          {
            name: "organization",
            target_env: "OPENROUTER_ORG_ID",
            secret_ref: "env:OPENROUTER_ORG_ID",
            kind: "organization",
            required: true,
          },
        ],
      },
      credentials: [
        {
          name: "api_key",
          target_env: "OPENROUTER_API_KEY",
          secret_ref: "vault:providers/openrouter/api-key",
          present: true,
          required: false,
        },
        {
          name: "organization",
          target_env: "OPENROUTER_ORG_ID",
          secret_ref: "env:OPENROUTER_ORG_ID",
          present: true,
          required: true,
        },
      ],
    };
    vi.mocked(listSettingsProviders).mockResolvedValue({
      ...collection,
      providers: [providerWithMultipleCredentials],
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(1));

    act(() => {
      result.current.openEdit(providerWithMultipleCredentials);
    });
    act(() => {
      result.current.updateDraft(draft => ({
        ...draft,
        default_model: "anthropic/claude-sonnet",
        credential_slots: draft.credential_slots.map((slot, index) =>
          index === 1 ? { ...slot, secret_ref: "vault:providers/openrouter/organization" } : slot
        ),
        credential_secret_values: ["", "org-secret"],
      }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsProvider).toHaveBeenCalledWith("openrouter", {
      settings: {
        command: "npx -y pi-acp@latest",
        default_model: "anthropic/claude-sonnet",
        harness: "pi_acp",
        runtime_provider: "openrouter",
        credential_slots: [
          {
            name: "api_key",
            target_env: "OPENROUTER_API_KEY",
            secret_ref: "vault:providers/openrouter/api-key",
            kind: "api_key",
            required: false,
          },
          {
            name: "organization",
            target_env: "OPENROUTER_ORG_ID",
            secret_ref: "vault:providers/openrouter/organization",
            kind: "organization",
            required: true,
          },
        ],
      },
      secrets: [
        {
          name: "organization",
          secret_ref: "vault:providers/openrouter/organization",
          kind: "organization",
          value: "org-secret",
        },
      ],
    });
  });

  it("submits vault-backed provider secrets without reading them back", async () => {
    vi.mocked(putSettingsProvider).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openCreate();
      result.current.updateDraft(draft => ({
        ...draft,
        name: "openrouter",
        command: "npx -y pi-acp@latest",
        default_model: "openai/gpt-5.4",
        target_env: "OPENROUTER_API_KEY",
        harness: "pi_acp",
        runtime_provider: "openrouter",
        secret_ref: "vault:providers/openrouter/api-key",
        secret_value: "sk-live",
      }));
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("saved");
    });

    expect(putSettingsProvider).toHaveBeenCalledWith("openrouter", {
      settings: {
        command: "npx -y pi-acp@latest",
        default_model: "openai/gpt-5.4",
        harness: "pi_acp",
        runtime_provider: "openrouter",
        credential_slots: [
          {
            name: "api_key",
            target_env: "OPENROUTER_API_KEY",
            secret_ref: "vault:providers/openrouter/api-key",
            kind: "api_key",
            required: true,
          },
        ],
      },
      secrets: [
        {
          name: "api_key",
          secret_ref: "vault:providers/openrouter/api-key",
          kind: "api_key",
          value: "sk-live",
        },
      ],
    });
  });

  it("surfaces validation errors from the adapter without closing the editor", async () => {
    vi.mocked(putSettingsProvider).mockRejectedValue(
      new SettingsApiError("invalid credential_slots[0].secret_ref", 400)
    );

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openEdit(claudeEntry);
    });
    act(() => {
      result.current.saveEditor();
    });

    await waitFor(() => {
      expect(result.current.editorError).toBe("invalid credential_slots[0].secret_ref");
    });
    expect(result.current.editor.mode).toBe("edit");
    expect(result.current.lastAction).toBeNull();
  });

  it("marks a delete action with fallback metadata for overlaid providers", async () => {
    vi.mocked(deleteSettingsProvider).mockResolvedValue({
      section: "general",
      scope: "global",
      behavior: "restart_required",
      applied: true,
      restart_required: true,
      write_target: "global-config",
    });

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openDelete(claudeEntry);
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => {
      expect(result.current.lastAction?.kind).toBe("deleted");
    });

    expect(deleteSettingsProvider).toHaveBeenCalledWith("claude");
    expect(result.current.lastAction).toMatchObject({
      name: "claude",
      hadFallback: true,
    });
  });

  it("surfaces conflict errors from delete without mutating selection", async () => {
    vi.mocked(deleteSettingsProvider).mockRejectedValue(
      new SettingsApiError("provider is in use", 409)
    );

    const { wrapper } = createWrapper();
    const { result } = renderHook(() => useSettingsProvidersPage(), { wrapper });

    await waitFor(() => expect(result.current.providers).toHaveLength(2));

    act(() => {
      result.current.openDelete(claudeEntry);
    });
    act(() => {
      result.current.confirmDelete();
    });

    await waitFor(() => {
      expect(result.current.deleteError).toBe("provider is in use");
    });
    expect(result.current.deleteTarget.mode).toBe("open");
    expect(result.current.lastAction).toBeNull();
  });
});
