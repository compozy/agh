import { Input } from "@agh/ui";

interface WindowManagerNumberFieldProps {
  label: string;
  value: number;
  minimum?: number;
  maximum?: number;
  integer?: boolean;
  onChange: (value: number) => void;
}

export function WindowManagerNumberField({
  label,
  value,
  minimum = 0,
  maximum,
  integer = false,
  onChange,
}: WindowManagerNumberFieldProps) {
  const valid =
    Number.isFinite(value) &&
    value >= minimum &&
    (maximum === undefined || value <= maximum) &&
    (!integer || Number.isInteger(value));
  return (
    <label className="flex min-w-28 flex-1 flex-col gap-1 text-form-label text-muted">
      {label}
      <Input
        aria-invalid={!valid}
        className="h-11"
        min={minimum}
        max={maximum}
        step={integer ? 1 : undefined}
        type="number"
        value={Number.isFinite(value) ? value : ""}
        onChange={event => onChange(event.currentTarget.valueAsNumber)}
      />
    </label>
  );
}
