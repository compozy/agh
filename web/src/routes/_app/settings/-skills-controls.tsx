import { Button, Input, Spinner } from "@agh/ui";

import { SettingsFieldRow } from "@/systems/settings";

interface AllowListFieldProps {
  label: string;
  description: string;
  testId: string;
  value: string[];
  onChange: (value: string[]) => void;
}

export function AllowListField({
  label,
  description,
  testId,
  value,
  onChange,
}: AllowListFieldProps) {
  return (
    <SettingsFieldRow
      data-testid={testId}
      label={label}
      description={description}
      hint="LIST"
      control={
        <Input
          key={value.join("\u0000")}
          className="w-72 font-mono"
          data-testid={`${testId}-input`}
          defaultValue={value.join(", ")}
          placeholder="none"
          onBlur={event => {
            const nextValue = event.currentTarget.value
              .split(",")
              .reduce<string[]>((entries, entry) => {
                const trimmed = entry.trim();
                if (trimmed.length > 0) entries.push(trimmed);
                return entries;
              }, []);
            const isUnchanged =
              nextValue.length === value.length &&
              nextValue.every((entry, index) => entry === value[index]);
            if (!isUnchanged) onChange(nextValue);
          }}
        />
      }
    />
  );
}

interface SaveControlsProps {
  slug: string;
  saveLabel: string;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  lastAppliedLabel: string | null;
  onSave: () => void;
  onReset: () => void;
}

export function SaveControls({
  slug,
  saveLabel,
  isDirty,
  isSaving,
  error,
  warnings,
  lastAppliedLabel,
  onSave,
  onReset,
}: SaveControlsProps) {
  const disabled = !isDirty || isSaving;
  return (
    <div
      className="flex items-center gap-2"
      data-testid={`settings-page-skills-${slug}-controls`}
      data-dirty={isDirty ? "true" : "false"}
    >
      <div className="min-w-0" role="status" aria-live={error ? "assertive" : "polite"}>
        {error ? (
          <span className="text-xs text-danger" data-testid={`settings-page-skills-${slug}-error`}>
            {error}
          </span>
        ) : warnings && warnings.length > 0 ? (
          <span
            className="text-xs text-warning"
            data-testid={`settings-page-skills-${slug}-warning`}
          >
            {warnings.join(" · ")}
          </span>
        ) : isDirty ? (
          <span className="text-xs text-subtle" data-testid={`settings-page-skills-${slug}-dirty`}>
            Unsaved changes
          </span>
        ) : lastAppliedLabel ? (
          <span
            className="text-xs text-subtle"
            data-testid={`settings-page-skills-${slug}-applied`}
          >
            {lastAppliedLabel}
          </span>
        ) : null}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onReset}
        disabled={!isDirty || isSaving}
        data-testid={`settings-page-skills-${slug}-reset`}
      >
        Discard
      </Button>
      <Button
        type="button"
        variant="default"
        size="sm"
        onClick={onSave}
        disabled={disabled}
        data-testid={`settings-page-skills-${slug}-save`}
      >
        {isSaving ? <Spinner className="size-3" /> : null}
        {isSaving ? "Saving..." : saveLabel}
      </Button>
    </div>
  );
}
