import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import type { SettingsProviderEntry } from "@/systems/settings";
import { settingsProviderFixtures } from "@/systems/settings/mocks";

import { ProviderCard } from "../provider-card";
import { ProviderLogo } from "../provider-logo";
import { ProviderModelCatalogStatus } from "../provider-model-catalog-status";
import { ProvidersGrid } from "../providers-grid";

const claudeFixture = settingsProviderFixtures[0]!;
const codexFixture = settingsProviderFixtures[1]!;

const openRouterFixture = settingsProviderFixtures.find(entry => entry.name === "openrouter")!;

const binaryMissingFixture: SettingsProviderEntry = {
  ...codexFixture,
  name: "qoder",
  default: false,
  command_available: false,
  settings: {
    ...codexFixture.settings,
    command: "qoder --acp",
    display_name: "Qoder CLI",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "builtin-provider", scope: "global" },
  },
  credentials: undefined,
};

const freshCatalogHandler = http.get(
  "/api/model-catalog/providers/:provider_id/models/status",
  () =>
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
);

const refreshHandler = http.post("/api/model-catalog/providers/:provider_id/models/refresh", () =>
  HttpResponse.json({ operation_id: "model_refresh_story", status: "queued" })
);

const staleCatalogHandler = http.get(
  "/api/model-catalog/providers/:provider_id/models/status",
  () =>
    HttpResponse.json({
      sources: [
        {
          source_id: "models.dev",
          source_kind: "models_dev",
          priority: 0,
          provider_id: "claude",
          refresh_state: "succeeded",
          row_count: 42,
          stale: true,
          last_success: "2026-04-10T08:00:00Z",
        },
        {
          source_id: "provider_live:claude",
          source_kind: "provider_live",
          priority: 1,
          provider_id: "claude",
          refresh_state: "succeeded",
          row_count: 3,
          stale: false,
          last_success: "2026-04-17T18:10:00Z",
        },
      ],
    })
);

const failedCatalogHandler = http.get(
  "/api/model-catalog/providers/:provider_id/models/status",
  () =>
    HttpResponse.json({
      sources: [
        {
          source_id: "models.dev",
          source_kind: "models_dev",
          priority: 0,
          provider_id: "claude",
          refresh_state: "failed",
          row_count: 0,
          stale: false,
          last_error: "registry timeout",
        },
      ],
    })
);

const emptyCatalogHandler = http.get(
  "/api/model-catalog/providers/:provider_id/models/status",
  () => HttpResponse.json({ sources: [] })
);

const meta: Meta<typeof ProviderCard> = {
  title: "systems/settings/ProviderCard",
  component: ProviderCard,
  parameters: {
    layout: "fullscreen",
    msw: { handlers: [freshCatalogHandler, refreshHandler] },
    docs: {
      description: {
        component:
          "Provider card with header (logo + name + DEFAULT + status), inline hint for warnings, summary block, and state-aware footer + overflow menu. Detailed config and per-source catalog list live in the right-side Details sheet.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="block min-h-[640px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

function CardFrame({ children }: { children: React.ReactNode }) {
  return <div className="w-full max-w-[420px]">{children}</div>;
}

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default provider card -- builtin claude, DEFAULT chip, catalog fresh.
 */
export const Default: Story = {
  args: {
    provider: claudeFixture,
    onOpen: fn(),
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Non-default provider with all credentials configured and a fresh catalog.
 */
export const Installed: Story = {
  args: {
    provider: codexFixture,
    onOpen: fn(),
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Binary missing -- the provider command is not on PATH, catalog row is suppressed
 * and the inline hint points at the missing command. Primary CTA stays "Edit settings".
 */
export const BinaryMissing: Story = {
  args: {
    provider: binaryMissingFixture,
    onOpen: fn(),
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Bound-secret provider missing the required credential. CTA becomes
 * "Configure credentials" and the hint cites the env slot to bind.
 */
export const Unconfigured: Story = {
  args: {
    provider: openRouterFixture,
    onOpen: fn(),
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Aggregated catalog chip flips to warning when any source is stale.
 */
export const CatalogStale: Story = {
  args: {
    provider: claudeFixture,
    onOpen: fn(),
  },
  parameters: {
    msw: { handlers: [staleCatalogHandler, refreshHandler] },
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Aggregated catalog chip flips to danger when any source failed to refresh.
 */
export const CatalogFailed: Story = {
  args: {
    provider: claudeFixture,
    onOpen: fn(),
  },
  parameters: {
    msw: { handlers: [failedCatalogHandler, refreshHandler] },
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Neutral catalog chip when no sources are reporting yet.
 */
export const CatalogEmpty: Story = {
  args: {
    provider: claudeFixture,
    onOpen: fn(),
  },
  parameters: {
    msw: { handlers: [emptyCatalogHandler, refreshHandler] },
  },
  render: args => (
    <CardFrame>
      <ProviderCard {...args} />
    </CardFrame>
  ),
};

/**
 * Grid layout -- exercises the responsive 2/3-column behaviour at sample widths.
 */
export const Grid: Story = {
  args: {},
  decorators: [
    Story => (
      <PanelSurface className="block min-h-[640px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
  render: () => <ProvidersGrid providers={settingsProviderFixtures.slice(0, 6)} onOpen={fn()} />,
};

/**
 * ProviderLogo set used across cards and the details sheet header.
 */
export const Logos: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-4">
      {["claude", "codex", "gemini", "opencode", "hermes"].map(provider => (
        <span
          key={provider}
          className="flex size-12 items-center justify-center rounded-icon-well border border-line bg-elevated"
          title={provider}
        >
          <ProviderLogo provider={provider} />
        </span>
      ))}
    </div>
  ),
};

/**
 * ProviderModelCatalogStatus rendered standalone -- used inside the Details sheet
 * to list every source with refresh state, stale chip, row count, and timestamp.
 */
export const CatalogStatus: Story = {
  args: {},
  parameters: {
    msw: { handlers: [staleCatalogHandler, refreshHandler] },
  },
  render: () => <ProviderModelCatalogStatus providerId="codex" testId="provider-catalog-story" />,
};
