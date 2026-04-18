import { AlertCircle, ExternalLink, Loader2 } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import { Switch } from "@agh/ui";
import { useSettingsNetworkPage } from "@/hooks/routes/use-settings-network-page";
import type { SettingsNetworkSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsSectionCard,
  SettingsStatGrid,
  SettingsStatItem,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/network")({
  component: NetworkSettingsPage,
});

type NetworkConfig = SettingsNetworkSection["config"];
type NetworkRuntime = SettingsNetworkSection["runtime"];

function NetworkSettingsPage() {
  const page = useSettingsNetworkPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-network-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-network-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load network settings"}
          </p>
        </div>
      </div>
    );
  }

  const { envelope, draft, setDraft, restart } = page;
  const runtime = envelope.runtime;

  return (
    <SettingsPageShell
      slug="network"
      title="Network"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-network-status-line"
          daemonAvailable={runtime.available}
          items={[
            <span key="status">
              {runtime.status ?? (runtime.enabled ? "enabled" : "disabled")}
            </span>,
            <span key="peers">
              {runtime.local_peers} local · {runtime.remote_peers} remote peers
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="network" restart={restart} />}
      banner={<SettingsRestartBanner slug="network" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="network"
          isDirty={page.isDirty}
          isSaving={page.isSaving}
          error={page.saveError}
          warnings={page.warnings}
          lastAppliedLabel={page.lastAppliedLabel}
          onSave={page.handleSave}
          onReset={page.handleReset}
        />
      }
    >
      <OperationalLinksRow />
      <RuntimeStatusSection runtime={runtime} />
      <ListenerSection draft={draft} setDraft={setDraft} />
      <DeliverySection draft={draft} setDraft={setDraft} />
    </SettingsPageShell>
  );
}

function OperationalLinksRow() {
  return (
    <SettingsSectionCard
      eyebrow="Operational"
      note="inspect channels, peers, and live message flow"
    >
      <div className="flex flex-wrap gap-2" data-testid="settings-page-network-operational-links">
        <Link
          to="/network"
          className="inline-flex items-center gap-1.5 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-1.5 text-xs font-medium text-[color:var(--color-text-primary)] hover:bg-[color:var(--color-hover)]"
          data-testid="settings-page-network-link-network"
        >
          <ExternalLink className="size-3.5 text-[color:var(--color-text-tertiary)]" />
          Open Network
        </Link>
      </div>
    </SettingsSectionCard>
  );
}

function RuntimeStatusSection({ runtime }: { runtime: NetworkRuntime }) {
  const listener =
    runtime.listener_host && runtime.listener_port
      ? `${runtime.listener_host}:${runtime.listener_port}`
      : "—";

  return (
    <SettingsSectionCard eyebrow="Runtime" note="read-only">
      <SettingsStatGrid>
        <SettingsStatItem
          label="Status"
          value={runtime.status ?? (runtime.enabled ? "enabled" : "disabled")}
          testId="settings-page-network-runtime-status"
        />
        <SettingsStatItem
          label="Listener"
          value={listener}
          testId="settings-page-network-runtime-listener"
        />
        <SettingsStatItem
          label="Local peers"
          value={String(runtime.local_peers)}
          testId="settings-page-network-runtime-local-peers"
        />
        <SettingsStatItem
          label="Remote peers"
          value={String(runtime.remote_peers)}
          testId="settings-page-network-runtime-remote-peers"
        />
        <SettingsStatItem
          label="Channels"
          value={String(runtime.channels)}
          testId="settings-page-network-runtime-channels"
        />
        <SettingsStatItem
          label="Queued messages"
          value={String(runtime.queued_messages)}
          testId="settings-page-network-runtime-queued-messages"
        />
        <SettingsStatItem
          label="Queued sessions"
          value={String(runtime.queued_sessions)}
          testId="settings-page-network-runtime-queued-sessions"
        />
        <SettingsStatItem
          label="Delivery workers"
          value={String(runtime.delivery_workers)}
          testId="settings-page-network-runtime-delivery-workers"
        />
      </SettingsStatGrid>
    </SettingsSectionCard>
  );
}

interface DraftSectionProps {
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
}

function ListenerSection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Listener" note="topology requires restart">
      <SettingsFieldRow
        data-testid="settings-page-network-enabled"
        label="Embedded network"
        description="Enable agent-to-agent coordination inside this daemon"
        control={
          <Switch
            data-testid="settings-page-network-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked => setDraft({ ...draft, enabled: checked })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-network-port"
        label="Listener port"
        description="TCP port for the embedded network"
        hint="CONFIG.TOML"
        control={
          <input
            type="number"
            min={0}
            className="h-8 w-28 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-network-port-input"
            value={draft.port}
            onChange={event => setDraft({ ...draft, port: Number(event.target.value || 0) })}
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-network-default-channel"
        label="Default channel"
        description="Channel new sessions join when none is specified"
        hint="DEFAULT"
        control={
          <input
            className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-network-default-channel-input"
            value={draft.default_channel ?? ""}
            placeholder="agh"
            onChange={event => setDraft({ ...draft, default_channel: event.target.value })}
          />
        }
      />
    </SettingsSectionCard>
  );
}

function DeliverySection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsSectionCard eyebrow="Delivery" note="queue limits and retention">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <NumberField
          label="Greet interval"
          suffix="sec"
          testId="settings-page-network-greet-interval"
          value={draft.greet_interval}
          onChange={value => setDraft({ ...draft, greet_interval: value })}
        />
        <NumberField
          label="Max payload"
          suffix="bytes"
          testId="settings-page-network-max-payload"
          value={draft.max_payload}
          onChange={value => setDraft({ ...draft, max_payload: value })}
        />
        <NumberField
          label="Max queue depth"
          testId="settings-page-network-max-queue-depth"
          value={draft.max_queue_depth}
          onChange={value => setDraft({ ...draft, max_queue_depth: value })}
        />
        <NumberField
          label="Max replay age"
          suffix="sec"
          testId="settings-page-network-max-replay-age"
          value={draft.max_replay_age}
          onChange={value => setDraft({ ...draft, max_replay_age: value })}
        />
      </div>
    </SettingsSectionCard>
  );
}

interface NumberFieldProps {
  label: string;
  testId: string;
  value: number;
  suffix?: string;
  onChange: (value: number) => void;
}

function NumberField({ label, testId, value, suffix, onChange }: NumberFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
        {label}
      </span>
      <div className="flex items-center gap-2">
        <input
          type="number"
          min={0}
          className="h-8 w-full rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 text-sm text-[color:var(--color-text-primary)]"
          data-testid={testId}
          value={value}
          onChange={event => onChange(Number(event.target.value || 0))}
        />
        {suffix ? (
          <span className="font-mono text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
            {suffix}
          </span>
        ) : null}
      </div>
    </div>
  );
}
