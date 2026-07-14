import {
  Field,
  FieldDescription,
  FieldLabel,
  Input,
  NativeSelect,
  NativeSelectOption,
} from "@agh/ui";

import type {
  NetworkParticipationDraft,
  NetworkParticipationMode,
} from "../lib/network-participation";

export interface NetworkParticipationFieldsProps {
  value: NetworkParticipationDraft;
  onChange: (next: NetworkParticipationDraft) => void;
  disabled?: boolean;
  testIdPrefix?: string;
}

/** Shared create/edit participation controls (UT-060). Local is the default. */
export function NetworkParticipationFields({
  value,
  onChange,
  disabled = false,
  testIdPrefix = "network-participation",
}: NetworkParticipationFieldsProps) {
  return (
    <div className="flex flex-col gap-3" data-testid={testIdPrefix}>
      <Field>
        <FieldLabel htmlFor={`${testIdPrefix}-mode`}>Network participation</FieldLabel>
        <FieldDescription>
          Local keeps the execution off the network. Live joins a channel explicitly — availability
          settings never enroll executions.
        </FieldDescription>
        <NativeSelect
          disabled={disabled}
          id={`${testIdPrefix}-mode`}
          data-testid={`${testIdPrefix}-mode`}
          value={value.mode}
          onChange={event =>
            onChange({
              ...value,
              mode: event.target.value as NetworkParticipationMode,
            })
          }
        >
          <NativeSelectOption value="local">Local</NativeSelectOption>
          <NativeSelectOption value="live">Live</NativeSelectOption>
        </NativeSelect>
      </Field>
      {value.mode === "live" ? (
        <>
          <Field>
            <FieldLabel htmlFor={`${testIdPrefix}-channel`}>Channel</FieldLabel>
            <Input
              disabled={disabled}
              id={`${testIdPrefix}-channel`}
              data-testid={`${testIdPrefix}-channel`}
              placeholder="channel id"
              value={value.channelId}
              onChange={event => onChange({ ...value, channelId: event.target.value })}
            />
          </Field>
          <Field>
            <FieldLabel htmlFor={`${testIdPrefix}-strategy`}>Channel strategy</FieldLabel>
            <Input
              disabled={disabled}
              id={`${testIdPrefix}-strategy`}
              data-testid={`${testIdPrefix}-strategy`}
              placeholder="optional strategy"
              value={value.channelStrategy}
              onChange={event => onChange({ ...value, channelStrategy: event.target.value })}
            />
          </Field>
        </>
      ) : null}
    </div>
  );
}
