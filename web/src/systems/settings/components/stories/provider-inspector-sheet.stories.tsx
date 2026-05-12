import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { useMemo, useState } from "react";
import { fn } from "storybook/test";

import { StorySurface } from "@/storybook/story-layout";
import type { SettingsProviderEntry } from "@/systems/settings";
import { settingsProviderFixtures } from "@/systems/settings/mocks";

import { ProviderInspectorSheet } from "../provider-inspector-sheet";
import type {
  ProviderDraft,
  ProviderInspectorState,
} from "@/hooks/routes/use-settings-providers-page";

const claude = settingsProviderFixtures[0]!;
const openrouter = settingsProviderFixtures.find(entry => entry.name === "openrouter")!;

const freshHandlers = [
  http.get("/api/providers/:provider_id/models/status", () =>
    HttpResponse.json({
      sources: [
        {
          source_id: "models.dev",
          source_kind: "models_dev",
          priority: 0,
          provider_id: "claude",
          refresh_state: "succeeded",
          row_count: 42,
          stale: false,
          last_success: "2026-04-17T18:10:00Z",
        },
      ],
    })
  ),
  http.post("/api/providers/:provider_id/models/refresh", () =>
    HttpResponse.json({ operation_id: "model_refresh_story", status: "queued" })
  ),
];

interface HarnessProps {
  initialState: ProviderInspectorState;
}

function draftFor(entry: SettingsProviderEntry): ProviderDraft {
  return {
    name: entry.name,
    command: entry.settings.command ?? "",
    display_name: entry.settings.display_name ?? "",
    model_default: entry.settings.models?.default ?? "",
    curated_models: (entry.settings.models?.curated ?? [])
      .map(model => model.id)
      .filter(Boolean)
      .join("\n"),
    curated_snapshot: (entry.settings.models?.curated ?? []).map(model => ({ ...model })),
    target_env: entry.settings.credential_slots?.[0]?.target_env ?? "",
    harness: entry.settings.harness ?? "acp",
    runtime_provider: entry.settings.runtime_provider ?? "",
    transport: entry.settings.transport ?? "",
    base_url: entry.settings.base_url ?? "",
    auth_mode: entry.settings.auth_mode ?? "native_cli",
    env_policy: entry.settings.env_policy ?? "filtered",
    home_policy: entry.settings.home_policy ?? "operator",
    auth_status_command: entry.settings.auth_status_command ?? "",
    auth_login_command: entry.settings.auth_login_command ?? "",
    secret_ref: entry.settings.credential_slots?.[0]?.secret_ref ?? "",
    secret_value: "",
    credential_slots: (entry.settings.credential_slots ?? []).map(slot => ({ ...slot })),
    credential_secret_values: (entry.settings.credential_slots ?? []).map(() => ""),
  };
}

function Harness({ initialState }: HarnessProps) {
  const [state, setState] = useState<ProviderInspectorState>(initialState);
  const entry = state.mode === "inspect" || state.mode === "edit" ? state.entry : null;
  const draft = useMemo(() => {
    if (state.mode === "edit" || state.mode === "create") return state.draft;
    return null;
  }, [state]);

  return (
    <StorySurface className="min-h-160 bg-canvas p-6">
      <p className="text-xs text-subtle">
        Sheet harness — toggle Inspect/Edit via the footer buttons.
      </p>
      <ProviderInspectorSheet
        open={state.mode !== "closed"}
        mode={state.mode === "closed" ? "inspect" : state.mode}
        entry={entry}
        draft={draft}
        existingNames={settingsProviderFixtures.map(p => p.name)}
        error={null}
        warnings={undefined}
        canSave={true}
        isSaving={false}
        isDeleting={false}
        onOpenChange={next => {
          if (!next) setState({ mode: "closed" });
        }}
        onDraftChange={updater => {
          setState(current => {
            if (current.mode !== "edit" && current.mode !== "create") return current;
            return { ...current, draft: updater(current.draft) };
          });
        }}
        onSwitchToEdit={() => {
          setState(current => {
            if (current.mode !== "inspect") return current;
            return {
              mode: "edit",
              entry: current.entry,
              draft: draftFor(current.entry),
              cameFrom: "inspect",
            };
          });
        }}
        onCancelEdit={() => {
          setState(current => {
            if (current.mode === "edit" && current.cameFrom === "inspect") {
              return { mode: "inspect", entry: current.entry };
            }
            return { mode: "closed" };
          });
        }}
        onSave={fn()}
        onRequestDelete={fn()}
        onRefreshCatalog={fn()}
      />
    </StorySurface>
  );
}

const meta: Meta<typeof ProviderInspectorSheet> = {
  title: "systems/settings/ProviderInspectorSheet",
  component: ProviderInspectorSheet,
  parameters: {
    layout: "fullscreen",
    msw: { handlers: freshHandlers },
    docs: {
      description: {
        component:
          "Unified right-side Sheet that hosts both Inspect (read-only) and Edit (form) modes for a provider. The Sheet replaces the previous Details Sheet + Edit Dialog split — one shell, single visual rhythm, mode switch via the footer.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const InspectClaudeDefault: Story = {
  args: {},
  render: () => <Harness initialState={{ mode: "inspect", entry: claude }} />,
};

export const InspectUnconfigured: Story = {
  args: {},
  render: () => <Harness initialState={{ mode: "inspect", entry: openrouter }} />,
};

export const EditMode: Story = {
  args: {},
  render: () => (
    <Harness
      initialState={{
        mode: "edit",
        entry: claude,
        draft: draftFor(claude),
        cameFrom: "inspect",
      }}
    />
  ),
};

export const CreateMode: Story = {
  args: {},
  render: () => (
    <Harness
      initialState={{
        mode: "create",
        draft: {
          ...draftFor(claude),
          name: "",
          command: "",
          display_name: "",
          model_default: "",
          curated_models: "",
        },
      }}
    />
  ),
};
