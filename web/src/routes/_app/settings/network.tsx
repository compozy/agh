import { AlertCircle, ExternalLink, Loader2, Network as NetworkIcon } from "lucide-react";
import { createFileRoute, Link } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import {
  Button,
  Eyebrow,
  Input,
  Metric,
  MetricGrid,
  PageShell,
  Section,
  Switch,
  useTopbarSlot,
} from "@agh/ui";
import type { TopbarRouteContext } from "@/types/topbar";
import { useSettingsNetworkPage } from "@/hooks/routes/use-settings-network-page";
import type { SettingsNetworkSection } from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsPageActions,
  SettingsRestartBanner,
  SettingsSaveBar,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/network")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Network settings", icon: NetworkIcon },
  }),
  component: NetworkSettingsPage,
});

type NetworkConfig = SettingsNetworkSection["config"];
type NetworkRuntime = SettingsNetworkSection["runtime"];

function NetworkSettingsPage() {
  const page = useSettingsNetworkPage();
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = useCallback(
    (key: string) => (message: string | null) => {
      setValidationErrors(current =>
        current[key] === message ? current : { ...current, [key]: message }
      );
    },
    []
  );
  const isInvalid = useMemo(
    () => Object.values(validationErrors).some(message => message !== null),
    [validationErrors]
  );
  const runtime = page.envelope?.runtime;
  useTopbarSlot({
    tabs: runtime ? (
      <SettingsStatusLine
        data-testid="settings-page-network-status-line"
        status={runtime.available ? "connected" : "error"}
        items={[
          <span key="status">{runtime.status ?? (runtime.enabled ? "enabled" : "disabled")}</span>,
          <span key="peers">
            {runtime.local_peers} local · {runtime.remote_peers} remote peers
          </span>,
        ]}
      />
    ) : undefined,
    actions: page.envelope ? (
      <SettingsPageActions slug="network" restart={page.restart} />
    ) : undefined,
  });

  if (page.isLoading) {
    return (
      <div
        aria-label="Loading network settings"
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-network-loading"
        role="status"
      >
        <Loader2 aria-hidden="true" className="size-5 animate-spin text-(--subtle)" />
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
          <AlertCircle className="size-6 text-(--danger)" />
          <p className="text-sm text-(--subtle)">
            {page.error?.message ?? "Failed to load network settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  if (!runtime) {
    return null;
  }
  const { draft, setDraft, restart } = page;

  return (
    <PageShell
      slug="network"
      banner={<SettingsRestartBanner slug="network" restart={restart} />}
      footer={
        <SettingsSaveBar
          slug="network"
          isDirty={page.isDirty}
          isInvalid={isInvalid}
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
      <ListenerSection
        draft={draft}
        setDraft={setDraft}
        portError={validationErrors.port ?? undefined}
        onPortValidityChange={setValidationError("port")}
      />
      <DeliverySection
        draft={draft}
        setDraft={setDraft}
        validationErrors={validationErrors}
        setValidationError={setValidationError}
      />
    </PageShell>
  );
}

function OperationalLinksRow() {
  return (
    <Section divided label="Operational" note="inspect channels, peers, and live message flow">
      <div className="flex flex-wrap gap-2" data-testid="settings-page-network-operational-links">
        <Link
          to="/network"
          className="inline-flex items-center gap-1.5 rounded-md border border-(--line) bg-(--elevated) px-3 py-1.5 text-xs font-medium text-(--fg) hover:bg-(--hover)"
          data-testid="settings-page-network-link-network"
        >
          <ExternalLink className="size-3.5 text-(--subtle)" />
          Open Network
        </Link>
      </div>
    </Section>
  );
}

function RuntimeStatusSection({ runtime }: { runtime: NetworkRuntime }) {
  const listener =
    runtime.listener_host && runtime.listener_port
      ? `${runtime.listener_host}:${runtime.listener_port}`
      : "--";

  return (
    <Section divided label="Runtime" note="read-only">
      <MetricGrid>
        <Metric
          label="Status"
          value={runtime.status ?? (runtime.enabled ? "enabled" : "disabled")}
          data-testid="settings-page-network-runtime-status"
        />
        <Metric
          label="Listener"
          value={listener}
          data-testid="settings-page-network-runtime-listener"
        />
        <Metric
          label="Local peers"
          value={String(runtime.local_peers)}
          data-testid="settings-page-network-runtime-local-peers"
        />
        <Metric
          label="Remote peers"
          value={String(runtime.remote_peers)}
          data-testid="settings-page-network-runtime-remote-peers"
        />
        <Metric
          label="Channels"
          value={String(runtime.channels)}
          data-testid="settings-page-network-runtime-channels"
        />
        <Metric
          label="Queued messages"
          value={String(runtime.queued_messages)}
          data-testid="settings-page-network-runtime-queued-messages"
        />
        <Metric
          label="Queued sessions"
          value={String(runtime.queued_sessions)}
          data-testid="settings-page-network-runtime-queued-sessions"
        />
        <Metric
          label="Delivery workers"
          value={String(runtime.delivery_workers)}
          data-testid="settings-page-network-runtime-delivery-workers"
        />
      </MetricGrid>
    </Section>
  );
}

interface DraftSectionProps {
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
}

function ListenerSection({
  draft,
  setDraft,
  portError,
  onPortValidityChange,
}: DraftSectionProps & {
  portError?: string;
  onPortValidityChange: (message: string | null) => void;
}) {
  return (
    <Section divided label="Listener" note="topology requires restart">
      <SettingsFieldRow
        data-testid="settings-page-network-enabled"
        label="Embedded network"
        description="Enable the open agent network protocol inside this daemon"
        control={
          <Switch
            aria-label="Embedded network"
            data-testid="settings-page-network-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={checked =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, enabled: checked };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-network-port"
        label="Listener port"
        description="TCP port for the embedded network"
        error={portError}
        hint="CONFIG.TOML"
        control={
          <SettingsNumberInput
            aria-label="Listener port"
            className="w-28"
            min={-1}
            data-testid="settings-page-network-port-input"
            value={draft.port}
            onValidityChange={onPortValidityChange}
            onValueChange={value =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, port: value };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-network-default-channel"
        label="Default channel"
        description="Channel new sessions join when none is specified"
        hint="DEFAULT"
        control={
          <Input
            className="w-56 font-mono"
            data-testid="settings-page-network-default-channel-input"
            value={draft.default_channel ?? ""}
            placeholder="agh"
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return { ...current, default_channel: event.target.value };
              })
            }
          />
        }
      />
    </Section>
  );
}

function DeliverySection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: DraftSectionProps & {
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}) {
  return (
    <Section divided label="Delivery" note="queue limits and retention">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <NumberField
          label="Greet interval"
          errorMessage={validationErrors.greetInterval ?? undefined}
          suffix="sec"
          testId="settings-page-network-greet-interval"
          value={draft.greet_interval}
          onValidityChange={setValidationError("greetInterval")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, greet_interval: value };
            })
          }
        />
        <NumberField
          label="Max payload"
          errorMessage={validationErrors.maxPayload ?? undefined}
          suffix="bytes"
          testId="settings-page-network-max-payload"
          value={draft.max_payload}
          onValidityChange={setValidationError("maxPayload")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, max_payload: value };
            })
          }
        />
        <NumberField
          label="Max queue depth"
          errorMessage={validationErrors.maxQueueDepth ?? undefined}
          testId="settings-page-network-max-queue-depth"
          value={draft.max_queue_depth}
          onValidityChange={setValidationError("maxQueueDepth")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, max_queue_depth: value };
            })
          }
        />
        <NumberField
          label="Max replay age"
          errorMessage={validationErrors.maxReplayAge ?? undefined}
          suffix="sec"
          testId="settings-page-network-max-replay-age"
          value={draft.max_replay_age}
          onValidityChange={setValidationError("maxReplayAge")}
          onChange={value =>
            setDraft(prev => {
              const current = prev ?? draft;
              return { ...current, max_replay_age: value };
            })
          }
        />
      </div>
    </Section>
  );
}

interface NumberFieldProps {
  label: string;
  testId: string;
  value: number;
  suffix?: string;
  errorMessage?: string;
  onValidityChange: (message: string | null) => void;
  onChange: (value: number) => void;
}

function NumberField({
  label,
  testId,
  value,
  suffix,
  errorMessage,
  onValidityChange,
  onChange,
}: NumberFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <Eyebrow case="upper" tone="muted" size="badge">
        {label}
      </Eyebrow>
      <div className="flex items-center gap-2">
        <SettingsNumberInput
          aria-label={label}
          className="w-full"
          min={0}
          data-testid={testId}
          value={value}
          onValidityChange={onValidityChange}
          onValueChange={onChange}
        />
        {suffix ? (
          <Eyebrow case="upper" tone="muted" size="badge">
            {suffix}
          </Eyebrow>
        ) : null}
      </div>
      {errorMessage ? <span className="text-xs text-(--danger)">{errorMessage}</span> : null}
    </div>
  );
}
