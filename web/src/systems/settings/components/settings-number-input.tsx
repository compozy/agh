import { useEffect, useMemo, useState, type ComponentProps } from "react";

import { Input } from "@agh/ui";

interface SettingsNumberInputProps extends Omit<
  ComponentProps<typeof Input>,
  "onChange" | "type" | "value"
> {
  value: number;
  min?: number;
  onValueChange: (value: number) => void;
  onValidityChange?: (message: string | null) => void;
}

function validateIntegerInput(rawValue: string, min: number): string | null {
  const trimmed = rawValue.trim();
  if (trimmed === "") {
    return "Enter a value.";
  }

  if (!/^\d+$/.test(trimmed)) {
    return "Enter a whole number.";
  }

  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isFinite(parsed) || parsed < min) {
    return `Value must be ${min} or greater.`;
  }

  return null;
}

function SettingsNumberInput({
  value,
  min = 0,
  onValueChange,
  onValidityChange,
  ...props
}: SettingsNumberInputProps) {
  const [rawValue, setRawValue] = useState(() => String(value));

  useEffect(() => {
    setRawValue(String(value));
  }, [value]);

  const validationMessage = useMemo(() => validateIntegerInput(rawValue, min), [min, rawValue]);

  useEffect(() => {
    onValidityChange?.(validationMessage);
  }, [onValidityChange, validationMessage]);

  return (
    <Input
      {...props}
      type="text"
      inputMode="numeric"
      pattern="[0-9]*"
      value={rawValue}
      aria-invalid={validationMessage ? true : undefined}
      onChange={event => {
        const nextRawValue = event.target.value;
        setRawValue(nextRawValue);

        const nextValidationMessage = validateIntegerInput(nextRawValue, min);
        if (nextValidationMessage) {
          return;
        }

        onValueChange(Number.parseInt(nextRawValue, 10));
      }}
    />
  );
}

export { SettingsNumberInput };
