import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { Button, Input } from "@agh/ui";

import { PanelSurface } from "@/storybook/story-layout";
import { settingsProvidersCollectionFixture } from "@/systems/settings/mocks";

import { SettingsCollectionHeader } from "../settings-collection-header";
import { SettingsFieldRow } from "../settings-field-row";
import { SettingsPageActions } from "../settings-page-actions";
import { SettingsPageShell } from "../settings-page-shell";
import { SettingsSectionCard } from "../settings-section-card";
import { SettingsSourceBadge } from "../settings-source-badge";
import { SettingsStatGrid, SettingsStatItem } from "../settings-stat-grid";
import { SettingsStatusLine } from "../settings-status-line";
import type { RestartBannerState } from "../settings-restart-banner";

const restart: RestartBannerState = {
  isVisible: true,
  isRestartRequired: true,
  isPolling: false,
  isSuccessful: false,
  isFailed: false,
  operationId: null,
  status: null,
  activeSessionCount: 2,
  trigger: fn(),
  isTriggerPending: false,
  triggerError: null,
  dismiss: fn(),
};

const meta: Meta<typeof SettingsPageShell> = {
  title: "systems/settings/SettingsPageShell",
  component: SettingsPageShell,
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
    <SettingsPageShell
      slug="providers"
      title="Providers"
      statusLine={
        <SettingsStatusLine
          daemonAvailable
          items={["3 configured", "1 workspace override"]}
          data-testid="settings-story-status"
        />
      }
      actions={<SettingsPageActions slug="providers" restart={restart} />}
      footer={
        <div className="px-6 py-3 text-small-body text-(--color-text-secondary)">
          Restart required after provider command changes.
        </div>
      }
    >
      <SettingsCollectionHeader
        eyebrow="Provider sources"
        summary="Effective config merged from built-in and workspace sources."
        action={
          <Button type="button" variant="outline" size="sm">
            Add provider
          </Button>
        }
      />
      <SettingsStatGrid>
        <SettingsStatItem label="Providers" value="12" detail="3 workspace scoped" />
        <SettingsStatItem label="Native auth" value="4" detail="2 need login" />
        <SettingsStatItem label="Catalogs" value="9" detail="1 stale source" />
        <SettingsStatItem label="Defaults" value="1" detail="codex" />
      </SettingsStatGrid>
      <SettingsSectionCard
        eyebrow="Codex"
        note="Provider settings are rendered as field rows so descriptions and validation stay aligned."
        headerAction={
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
      </SettingsSectionCard>
    </SettingsPageShell>
  ),
};

/**
 * Unavailable daemon status keeps the shell copy truthful.
 */
export const DaemonUnavailable: Story = {
  args: {},
  render: () => (
    <SettingsPageShell
      slug="observability"
      title="Observability"
      statusLine={
        <SettingsStatusLine
          daemonAvailable={false}
          items={["last check failed", "retry from the daemon"]}
        />
      }
    >
      <SettingsSectionCard
        eyebrow="Runtime state"
        note="Controls are read-only until daemon health recovers."
      >
        <SettingsFieldRow label="Trace level" control={<Input defaultValue="info" disabled />} />
      </SettingsSectionCard>
    </SettingsPageShell>
  ),
};
