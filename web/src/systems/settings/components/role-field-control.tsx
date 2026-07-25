import type { ReactNode } from "react";

import { cn, Input, NativeSelect, NativeSelectOption, Switch } from "@agh/ui";

import { REASONING_OPTIONS, type RoleFieldDescriptor } from "../lib/roles-config";
import { SettingsFieldRow } from "./settings-field-row";
import { SettingsNumberInput } from "./settings-number-input";

interface RoleFieldControlProps {
  field: RoleFieldDescriptor;
  value: string | number | boolean;
  /** Whether this field carries an effective (projected) value hint. */
  hasEffective: boolean;
  /** Projected effective value; `null` means "resolves at invocation" (never fabricated). */
  effective: string | null;
  error?: string;
  disabled?: boolean;
  testId: string;
  resetRevision?: number;
  fieldRef?: (element: HTMLInputElement | null) => void;
  onValueChange: (value: string | number | boolean) => void;
  onValidityChange?: (message: string | null) => void;
}

/** Truthful effective-value hint for a routing field — null renders as unresolved. */
function EffectiveHint({ effective }: { effective: string | null }) {
  if (effective == null) {
    return <span className="mt-0.5 block text-form-hint text-subtle">Resolves at invocation.</span>;
  }
  return (
    <span className="mt-0.5 block text-form-hint text-subtle">
      Effective <span className="font-mono text-muted">{effective}</span>
    </span>
  );
}

function renderControl({
  field,
  value,
  error,
  disabled,
  testId,
  onValueChange,
  onValidityChange,
  resetRevision,
  fieldRef,
}: RoleFieldControlProps): ReactNode {
  switch (field.kind) {
    case "switch":
      return (
        <Switch
          data-testid={`${testId}-switch`}
          checked={Boolean(value)}
          disabled={disabled}
          onCheckedChange={checked => onValueChange(checked)}
        />
      );
    case "number":
      return (
        <SettingsNumberInput
          className="w-24"
          data-testid={`${testId}-input`}
          min={field.min}
          value={Number(value)}
          resetRevision={resetRevision}
          ref={fieldRef}
          disabled={disabled}
          onValueChange={next => onValueChange(next)}
          onValidityChange={onValidityChange}
        />
      );
    case "select":
      return (
        <NativeSelect
          data-testid={`${testId}-input`}
          value={String(value)}
          disabled={disabled}
          aria-invalid={error ? true : undefined}
          onChange={event => onValueChange(event.target.value)}
        >
          {REASONING_OPTIONS.map(option => (
            <NativeSelectOption key={option.value} value={option.value}>
              {option.label}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      );
    default:
      return (
        <Input
          className={cn("w-56", field.mono && "font-mono")}
          data-testid={`${testId}-input`}
          value={String(value)}
          placeholder={field.placeholder}
          disabled={disabled}
          aria-invalid={error ? true : undefined}
          onChange={event => onValueChange(event.target.value)}
        />
      );
  }
}

/** One editable role field rendered as a settings row with an effective-value hint. */
export function RoleFieldControl(props: RoleFieldControlProps) {
  const { field, hasEffective, effective, error, testId } = props;
  const description = (
    <>
      {field.description ? <span>{field.description}</span> : null}
      {hasEffective ? <EffectiveHint effective={effective} /> : null}
    </>
  );
  return (
    <SettingsFieldRow
      data-testid={testId}
      label={field.label}
      description={description}
      error={error}
      control={renderControl(props)}
    />
  );
}
