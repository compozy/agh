import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { settingsProviderFixtures } from "@/systems/settings/mocks";

import { ProviderCard } from "../provider-card";
import { ProviderLogo } from "../provider-logo";
import { ProviderModelCatalogStatus } from "../provider-model-catalog-status";
import { ProvidersGrid } from "../providers-grid";

const catalogHandlers = [
  http.get("/api/providers/:provider_id/models/status", () =>
    HttpResponse.json({
      sources: [
        {
          source_id: "models.dev",
          source_kind: "models_dev",
          refresh_state: "succeeded",
          refreshed_at: "2026-04-17T18:10:00Z",
          row_count: 42,
          stale: false,
        },
      ],
    })
  ),
  http.post("/api/providers/:provider_id/models/refresh", () =>
    HttpResponse.json({
      operation_id: "model_refresh_story",
      status: "queued",
    })
  ),
];

const meta: Meta<typeof ProviderCard> = {
  title: "systems/settings/ProviderCard",
  component: ProviderCard,
  parameters: {
    layout: "fullscreen",
    msw: {
      handlers: catalogHandlers,
    },
    docs: {
      description: {
        component:
          "Provider card covering provider logos, source badges, auth state, credential slots, and model catalog status.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[640px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Built-in provider card with default marker and model catalog status.
 */
export const Default: Story = {
  args: {
    provider: settingsProviderFixtures[0]!,
    onEdit: fn(),
    onDelete: fn(),
  },
  render: args => <ProviderCard {...args} />,
};

/**
 * ProvidersGrid renders multiple ProviderCard instances with consistent spacing.
 */
export const Grid: Story = {
  args: {},
  render: () => (
    <ProvidersGrid providers={settingsProviderFixtures.slice(0, 3)} onEdit={fn()} onDelete={fn()} />
  ),
};

/**
 * ProviderLogo includes branded and fallback glyphs used across cards.
 */
export const Logos: Story = {
  args: {},
  render: () => (
    <div className="flex flex-wrap items-center gap-4">
      {["claude", "codex", "gemini", "opencode", "hermes"].map(provider => (
        <span
          key={provider}
          className="flex size-12 items-center justify-center rounded-icon-well border border-(--line) bg-(--elevated)"
          title={provider}
        >
          <ProviderLogo provider={provider} />
        </span>
      ))}
    </div>
  ),
};

/**
 * ProviderModelCatalogStatus can be inspected without the full ProviderCard chrome.
 */
export const CatalogStatus: Story = {
  args: {},
  render: () => <ProviderModelCatalogStatus providerId="codex" testId="provider-catalog-story" />,
};
