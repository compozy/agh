import type { Meta, StoryObj } from "@storybook/react-vite";

import { Button, Input, Metric, MetricGrid, PageShell, Section } from "@agh/ui";

import { PanelSurface } from "@/storybook/story-layout";
import { settingsProvidersCollectionFixture } from "@/systems/settings/mocks";

import { SettingsFieldRow } from "../settings-field-row";
import { SettingsSourceBadge } from "../settings-source-badge";

const meta: Meta<typeof PageShell> = {
  title: "systems/settings/PageShell",
  component: PageShell,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Shared settings page shell plus section, field, status, source, stats, and action primitives.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[720px]">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Page shell with status line, restart action, stats, source badge, and field rows.
 */
export const Default: Story = {
  args: {},
  render: () => (
    <PageShell
      slug="providers"
      footer={
        <div className="px-6 py-3 text-small-body text-(--muted)">
          Restart required after provider command changes.
        </div>
      }
    >
      <Section
        label="Provider sources"
        note="Effective config merged from built-in and workspace sources."
        right={
          <Button type="button" variant="outline" size="sm">
            Add provider
          </Button>
        }
      />
      <MetricGrid>
        <Metric label="Providers" value="12" subtext="3 workspace scoped" />
        <Metric label="Native auth" value="4" subtext="2 need login" />
        <Metric label="Catalogs" value="9" subtext="1 stale source" />
        <Metric label="Defaults" value="1" subtext="codex" />
      </MetricGrid>
      <Section
        divided
        label="Codex"
        note="Provider settings are rendered as field rows so descriptions and validation stay aligned."
        right={
          <SettingsSourceBadge
            source={
              settingsProvidersCollectionFixture.providers[0]!.source_metadata.effective_source
            }
            shadowed={
              settingsProvidersCollectionFixture.providers[0]!.source_metadata.shadowed_sources
            }
          />
        }
      >
        <SettingsFieldRow
          label="Command"
          description="Executable used by the daemon when starting sessions."
          hint="required"
          control={<Input defaultValue="codex" />}
        />
        <SettingsFieldRow
          label="Default model"
          description="Model id passed to the provider unless a session override is selected."
          error="Model id is required when model catalog fallback is disabled."
          control={<Input aria-invalid defaultValue="" />}
        />
      </Section>
    </PageShell>
  ),
};

/**
 * Unavailable daemon status keeps the shell copy truthful.
 */
export const DaemonUnavailable: Story = {
  args: {},
  render: () => (
    <PageShell slug="observability">
      <Section
        divided
        label="Runtime state"
        note="Controls are read-only until daemon health recovers."
      >
        <SettingsFieldRow label="Trace level" control={<Input defaultValue="info" disabled />} />
      </Section>
    </PageShell>
  ),
};
