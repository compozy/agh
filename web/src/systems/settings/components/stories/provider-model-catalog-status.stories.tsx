import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { CenteredSurface } from "@/storybook/story-layout";

import { ProviderModelCatalogStatus } from "../provider-model-catalog-status";

const meta: Meta<typeof ProviderModelCatalogStatus> = {
  title: "systems/settings/ProviderModelCatalogStatus",
  component: ProviderModelCatalogStatus,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Inline status block for a provider's model catalog refresh state. Shows per-source row counts + tone pills + a refresh action wired to the model-catalog mutation.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Default — two healthy sources reporting recent refresh state.
 */
export const Default: Story = {
  args: {
    providerId: "claude",
    testId: "settings-page-providers-card-claude-catalog",
  },
  parameters: {
    msw: {
      handlers: [
        http.get("/api/v1/providers/:provider/models/status", () =>
          HttpResponse.json({
            sources: [
              {
                source_id: "anthropic-cloud",
                refresh_state: "fresh",
                stale: false,
                row_count: 14,
                last_refreshed_at: "2026-04-17T18:00:00Z",
              },
              {
                source_id: "manifest",
                refresh_state: "fresh",
                stale: false,
                row_count: 6,
                last_refreshed_at: "2026-04-17T18:00:00Z",
              },
            ],
          })
        ),
      ],
    },
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-md rounded-md border border-line bg-canvas-soft p-4">
        <ProviderModelCatalogStatus {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * StaleSources — one source flagged as stale; warning-tone pill renders.
 */
export const StaleSources: Story = {
  args: {
    providerId: "codex",
    testId: "settings-page-providers-card-codex-catalog",
  },
  parameters: {
    msw: {
      handlers: [
        http.get("/api/v1/providers/:provider/models/status", () =>
          HttpResponse.json({
            sources: [
              {
                source_id: "openai-cloud",
                refresh_state: "fresh",
                stale: false,
                row_count: 22,
                last_refreshed_at: "2026-04-17T18:00:00Z",
              },
              {
                source_id: "preview-channel",
                refresh_state: "stale",
                stale: true,
                row_count: 4,
                last_refreshed_at: "2026-04-12T08:00:00Z",
              },
            ],
          })
        ),
      ],
    },
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-md rounded-md border border-line bg-canvas-soft p-4">
        <ProviderModelCatalogStatus {...args} />
      </div>
    </CenteredSurface>
  ),
};

/**
 * EmptySources — provider has no sources reporting yet.
 */
export const EmptySources: Story = {
  args: {
    providerId: "qwen-code",
    testId: "settings-page-providers-card-qwen-code-catalog",
  },
  parameters: {
    msw: {
      handlers: [
        http.get("/api/v1/providers/:provider/models/status", () =>
          HttpResponse.json({ sources: [] })
        ),
      ],
    },
  },
  render: args => (
    <CenteredSurface>
      <div className="w-full max-w-md rounded-md border border-line bg-canvas-soft p-4">
        <ProviderModelCatalogStatus {...args} />
      </div>
    </CenteredSurface>
  ),
};
