import type { Dispatch, SetStateAction } from "react";

import { Switch } from "@agh/ui";

import type { SettingsNetworkSection } from "../types";
import { NetworkLiveSettingsSections } from "./network-live-settings-sections";
import { SettingLinkRow } from "./setting-row";
import { SettingsAdvancedFold } from "./settings-advanced-fold";
import { SettingsFieldRow } from "./settings-field-row";
import { SettingsGroup } from "./settings-group";
import { SettingsNumberInput } from "./settings-number-input";
import { SettingsRuntimeUnavailable } from "./settings-runtime-unavailable";
import { SettingsTile, SettingsTiles } from "./settings-tiles";

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
      <SettingsAdvancedFold data-testid="settings-page-network-advanced" padded>
        <NetworkLiveSettingsSections {...props} />
      </SettingsAdvancedFold>
    </>
  );
}

function OperationalLinksSection() {
  return (
    <SettingsGroup
      description="These settings control Network availability and finite Live defaults and ceilings. They never opt sessions, tasks, loops, or automations into participation."
      title="Operational"
    >
      <SettingLinkRow
        data-testid="settings-page-network-link-network"
        description="Channels, peers, and live coordination for this workspace."
        label="Open Network"
        to="/network"
      />
    </SettingsGroup>
  );
}

function RuntimeStatusSection({ runtime }: { runtime: NetworkRuntime }) {
  return (
    <SettingsGroup
      bare
      description="What the listener is doing right now. Read-only."
      title="Runtime"
    >
      {runtime.available ? (
        <SettingsTiles>
          <SettingsTile
            data-testid="settings-page-network-runtime-status"
            dotTone={runtime.enabled ? "success" : "neutral"}
            label="Status"
            value={runtime.status ?? (runtime.enabled ? "ready" : "disabled")}
          />
          <SettingsTile
            data-testid="settings-page-network-runtime-live-participants"
            label="Live participants"
            value={String(runtime.local_peers)}
          />
          <SettingsTile
            data-testid="settings-page-network-runtime-channels"
            label="Channels"
            value={String(runtime.channels)}
          />
          <SettingsTile
            data-testid="settings-page-network-runtime-messages-received"
            detail={`${runtime.messages_delivered} delivered · ${runtime.messages_rejected} rejected`}
            label="Messages received"
            value={String(runtime.messages_received)}
          />
        </SettingsTiles>
      ) : (
        <SettingsRuntimeUnavailable
          slug="network"
          description="The Network listener did not return live status or traffic counters."
        />
      )}
    </SettingsGroup>
  );
}

interface DraftSectionProps {
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
}

function AvailabilitySection({ draft, setDraft }: DraftSectionProps) {
  return (
    <SettingsGroup title="Availability" description="applies live without enrollment">
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
    </SettingsGroup>
  );
}

function ProtocolSafetySection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: NetworkSettingsSectionsProps) {
  return (
    <SettingsGroup title="Protocol safety" description="replay protection">
      <SettingsFieldRow
        label="Replay window"
        description="seconds"
        error={validationErrors.maxReplayAge ?? undefined}
        control={
          <SettingsNumberInput
            aria-label="Replay window"
            className="w-32"
            data-testid="settings-page-network-max-replay-age"
            min={1}
            value={draft.max_replay_age}
            onValidityChange={setValidationError("maxReplayAge")}
            onValueChange={maxReplayAge =>
              setDraft(current => ({ ...(current ?? draft), max_replay_age: maxReplayAge }))
            }
          />
        }
      />
    </SettingsGroup>
  );
}
