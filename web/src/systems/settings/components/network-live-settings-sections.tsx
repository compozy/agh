import type { Dispatch, SetStateAction } from "react";

import { Eyebrow, Input, Section } from "@agh/ui";

import type { SettingsNetworkSection } from "../types";
import { SettingsNumberInput } from "./settings-number-input";

type NetworkConfig = SettingsNetworkSection["config"];
type LiveDefaults = NetworkConfig["live"]["defaults"];
type LiveLimits = NetworkConfig["live"]["limits"];

interface NetworkLiveSettingsProps {
  draft: NetworkConfig;
  setDraft: Dispatch<SetStateAction<NetworkConfig | null>>;
  validationErrors: Record<string, string | null>;
  setValidationError: (key: string) => (message: string | null) => void;
}

export function NetworkLiveSettingsSections(props: NetworkLiveSettingsProps) {
  return (
    <>
      <LiveDefaultsSection {...props} />
      <LiveLimitsSection {...props} />
    </>
  );
}

function LiveDefaultsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: NetworkLiveSettingsProps) {
  const update = (patch: Partial<LiveDefaults>) => {
    setDraft(current => ({
      ...(current ?? draft),
      live: {
        ...(current ?? draft).live,
        defaults: { ...(current ?? draft).live.defaults, ...patch },
      },
    }));
  };

  return (
    <Section divided label="Live defaults" note="applied only after explicit Live opt-in">
      <p className="text-xs text-subtle">
        Omitted per-execution bounds resolve to these finite defaults. Local executions ignore every
        value in this section.
      </p>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <NumberField
          label="Max wakes"
          testId="settings-page-network-live-default-max-wakes"
          value={draft.live.defaults.max_wakes}
          errorMessage={validationErrors.defaultMaxWakes ?? undefined}
          onValidityChange={setValidationError("defaultMaxWakes")}
          onChange={value => update({ max_wakes: value })}
        />
        <NumberField
          label="Max wake depth"
          testId="settings-page-network-live-default-max-depth"
          value={draft.live.defaults.max_wake_depth}
          errorMessage={validationErrors.defaultMaxDepth ?? undefined}
          onValidityChange={setValidationError("defaultMaxDepth")}
          onChange={value => update({ max_wake_depth: value })}
        />
        <NumberField
          label="Input token budget"
          testId="settings-page-network-live-default-input-tokens"
          value={draft.live.defaults.max_input_tokens}
          errorMessage={validationErrors.defaultInputTokens ?? undefined}
          onValidityChange={setValidationError("defaultInputTokens")}
          onChange={value => update({ max_input_tokens: value })}
        />
        <NumberField
          label="Output token budget"
          testId="settings-page-network-live-default-output-tokens"
          value={draft.live.defaults.max_output_tokens}
          errorMessage={validationErrors.defaultOutputTokens ?? undefined}
          onValidityChange={setValidationError("defaultOutputTokens")}
          onChange={value => update({ max_output_tokens: value })}
        />
        <TextField
          label="Wake timeout"
          testId="settings-page-network-live-default-wake-time"
          value={draft.live.defaults.max_wake_wall_time}
          onChange={value => update({ max_wake_wall_time: value })}
        />
        <TextField
          label="Total timeout"
          testId="settings-page-network-live-default-total-time"
          value={draft.live.defaults.max_total_wall_time}
          onChange={value => update({ max_total_wall_time: value })}
        />
        <TextField
          label="Coalesce window"
          testId="settings-page-network-live-default-coalesce"
          value={draft.live.defaults.coalesce_window}
          onChange={value => update({ coalesce_window: value })}
        />
      </div>
    </Section>
  );
}

function LiveLimitsSection({
  draft,
  setDraft,
  validationErrors,
  setValidationError,
}: NetworkLiveSettingsProps) {
  const update = (patch: Partial<LiveLimits>) => {
    setDraft(current => ({
      ...(current ?? draft),
      live: {
        ...(current ?? draft).live,
        limits: { ...(current ?? draft).live.limits, ...patch },
      },
    }));
  };

  return (
    <Section divided label="Live limits" note="inclusive administrative ceilings">
      <p className="text-xs text-subtle">
        Execution requests above these ceilings are rejected before durable run state is created.
      </p>
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <NumberField
          label="Max wakes ceiling"
          testId="settings-page-network-live-limit-max-wakes"
          value={draft.live.limits.max_wakes}
          errorMessage={validationErrors.limitMaxWakes ?? undefined}
          onValidityChange={setValidationError("limitMaxWakes")}
          onChange={value => update({ max_wakes: value })}
        />
        <NumberField
          label="Max wake depth ceiling"
          testId="settings-page-network-live-limit-max-depth"
          value={draft.live.limits.max_wake_depth}
          errorMessage={validationErrors.limitMaxDepth ?? undefined}
          onValidityChange={setValidationError("limitMaxDepth")}
          onChange={value => update({ max_wake_depth: value })}
        />
        <NumberField
          label="Input token ceiling"
          testId="settings-page-network-live-limit-input-tokens"
          value={draft.live.limits.max_input_tokens}
          errorMessage={validationErrors.limitInputTokens ?? undefined}
          onValidityChange={setValidationError("limitInputTokens")}
          onChange={value => update({ max_input_tokens: value })}
        />
        <NumberField
          label="Output token ceiling"
          testId="settings-page-network-live-limit-output-tokens"
          value={draft.live.limits.max_output_tokens}
          errorMessage={validationErrors.limitOutputTokens ?? undefined}
          onValidityChange={setValidationError("limitOutputTokens")}
          onChange={value => update({ max_output_tokens: value })}
        />
        <TextField
          label="Wake timeout ceiling"
          testId="settings-page-network-live-limit-wake-time"
          value={draft.live.limits.max_wake_wall_time}
          onChange={value => update({ max_wake_wall_time: value })}
        />
        <TextField
          label="Total timeout ceiling"
          testId="settings-page-network-live-limit-total-time"
          value={draft.live.limits.max_total_wall_time}
          onChange={value => update({ max_total_wall_time: value })}
        />
        <TextField
          label="Minimum coalesce window"
          testId="settings-page-network-live-limit-min-coalesce"
          value={draft.live.limits.min_coalesce_window}
          onChange={value => update({ min_coalesce_window: value })}
        />
        <TextField
          label="Maximum coalesce window"
          testId="settings-page-network-live-limit-max-coalesce"
          value={draft.live.limits.max_coalesce_window}
          onChange={value => update({ max_coalesce_window: value })}
        />
      </div>
    </Section>
  );
}

interface NumberFieldProps {
  label: string;
  testId: string;
  value: number;
  errorMessage?: string;
  onValidityChange: (message: string | null) => void;
  onChange: (value: number) => void;
}

function NumberField(props: NumberFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <Eyebrow className="text-muted">{props.label}</Eyebrow>
      <SettingsNumberInput
        aria-label={props.label}
        className="w-full"
        data-testid={props.testId}
        min={1}
        value={props.value}
        onValidityChange={props.onValidityChange}
        onValueChange={props.onChange}
      />
      {props.errorMessage ? (
        <span className="text-xs text-danger">{props.errorMessage}</span>
      ) : null}
    </div>
  );
}

interface TextFieldProps {
  label: string;
  testId: string;
  value: string;
  onChange: (value: string) => void;
}

function TextField(props: TextFieldProps) {
  return (
    <div className="flex flex-col gap-1">
      <Eyebrow className="text-muted">{props.label}</Eyebrow>
      <Input
        aria-label={props.label}
        className="font-mono"
        data-testid={props.testId}
        value={props.value}
        onChange={event => props.onChange(event.target.value)}
      />
    </div>
  );
}
