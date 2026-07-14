import { Link } from "@tanstack/react-router";
import { ExternalLink } from "lucide-react";
import type { Dispatch, SetStateAction } from "react";

import { Metric, MetricGrid, Section, Switch } from "@agh/ui";

import type { SettingsNetworkSection } from "../types";
import { NetworkLiveSettingsSections } from "./network-live-settings-sections";
import { SettingsFieldRow } from "./settings-field-row";
import { SettingsNumberInput } from "./settings-number-input";

type NetworkConfig = SettingsNetworkSection["config"];
type NetworkRuntime = SettingsNetworkSection["runtime"];

interface NetworkSettingsSectionsProps {
  runtime: NetworkRuntime;
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}

export function NetworkSettingsSections(props: NetworkSettingsSectionsProps) {
  return (
    <>
      <OperationalLinksSection />
      <RuntimeStatusSection runtime={props.runtime} />
      <AvailabilitySection draft={props.draft} setDraft={props.setDraft} />
      <ProtocolSafetySection {...props} />
      <NetworkLiveSettingsSections {...props} />
    </>
  );
}

function OperationalLinksSection() {
  return (
    <Section divided label="Operational" note="availability never enrolls executions">
      <p className="text-xs text-subtle" data-testid="settings-page-network-enrollment-note">
        These settings control Network availability and finite Live defaults and ceilings. They do
        not opt sessions, tasks, loops, or automations into participation.
      </p>
      <div className="flex flex-wrap gap-2" data-testid="settings-page-network-operational-links">
        <Link
          to="/network"
          className="inline-flex items-center gap-1.5 rounded-md border border-line bg-elevated px-3 py-1.5 text-xs font-medium text-fg hover:bg-hover"
          data-testid="settings-page-network-link-network"
        >
          <ExternalLink className="size-3 text-subtle" />
          Open Network
        </Link>
        <Link
          to="/network"
          className="inline-flex items-center gap-1.5 rounded-md border border-line bg-elevated px-3 py-1.5 text-xs font-medium text-fg hover:bg-hover"
          data-testid="settings-page-network-link-usage"
        >
          <ExternalLink className="size-3 text-subtle" />
          View usage
        </Link>
      </div>
    </Section>
  );
}

function RuntimeStatusSection({ runtime }: { runtime: NetworkRuntime }) {
  return (
    <Section divided label="Runtime" note="read-only">
      <MetricGrid>
        <Metric
          label="Status"
          value={runtime.status ?? (runtime.enabled ? "ready" : "disabled")}
          data-testid="settings-page-network-runtime-status"
        />
        <Metric
          label="Live participants"
          value={String(runtime.local_peers)}
          data-testid="settings-page-network-runtime-live-participants"
        />
        <Metric
          label="Channels"
          value={String(runtime.channels)}
          data-testid="settings-page-network-runtime-channels"
        />
        <Metric
          label="Messages received"
          value={String(runtime.messages_received)}
          data-testid="settings-page-network-runtime-messages-received"
        />
        <Metric
          label="Wakes delivered"
          value={String(runtime.messages_delivered)}
          data-testid="settings-page-network-runtime-messages-delivered"
        />
        <Metric
          label="Messages rejected"
          value={String(runtime.messages_rejected)}
          data-testid="settings-page-network-runtime-messages-rejected"
        />
      </MetricGrid>
    </Section>
  );
}

interface DraftSectionProps {
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
}

function AvailabilitySection({ draft, setDraft }: DraftSectionProps) {
  return (
    <Section divided label="Availability" note="applies live without enrollment">
      <SettingsFieldRow
        data-testid="settings-page-network-enabled"
        label="Network availability"
        description="Allow explicitly Live executions to join coordination conversations"
        control={
          <Switch
            aria-label="Network availability"
            data-testid="settings-page-network-enabled-switch"
            checked={draft.enabled}
            onCheckedChange={enabled => setDraft(current => ({ ...(current ?? draft), enabled }))}
          />
        }
      />
    </Section>
  );
}

function ProtocolSafetySection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: NetworkSettingsSectionsProps) {
  const fields = [
    {
      key: "greetInterval",
      label: "Presence cadence",
      suffix: "sec",
      testId: "settings-page-network-greet-interval",
      value: draft.greet_interval,
      update: (value: number) => ({ greet_interval: value }),
    },
    {
      key: "maxReplayAge",
      label: "Replay window",
      suffix: "sec",
      testId: "settings-page-network-max-replay-age",
      value: draft.max_replay_age,
      update: (value: number) => ({ max_replay_age: value }),
    },
    {
      key: "maxQueueDepth",
      label: "Inbox depth",
      suffix: "messages",
      testId: "settings-page-network-max-queue-depth",
      value: draft.max_queue_depth,
      update: (value: number) => ({ max_queue_depth: value }),
    },
  ];

  return (
    <Section divided label="Protocol safety" note="presence, replay, and durable inbox bounds">
      <div className="grid gap-4 md:grid-cols-3">
        {fields.map(field => (
          <SettingsFieldRow
            key={field.key}
            label={field.label}
            description={field.suffix}
            error={validationErrors[field.key] ?? undefined}
            control={
              <SettingsNumberInput
                aria-label={field.label}
                className="w-32"
                data-testid={field.testId}
                min={1}
                value={field.value}
                onValidityChange={setValidationError(field.key)}
                onValueChange={value =>
                  setDraft(current => ({ ...(current ?? draft), ...field.update(value) }))
                }
              />
            }
          />
        ))}
      </div>
    </Section>
  );
}
