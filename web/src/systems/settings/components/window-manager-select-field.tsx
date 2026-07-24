import { NativeSelect, NativeSelectOption } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";
import type { SelectOption } from "./window-manager-config-field-types";

interface WindowManagerSelectFieldProps<V extends string> {
  label: string;
  description: string;
  value: V;
  options: readonly SelectOption<V>[];
  onChange: (value: V) => void;
}

export function WindowManagerSelectField<V extends string>({
  label,
  description,
  value,
  options,
  onChange,
}: WindowManagerSelectFieldProps<V>) {
  return (
    <SettingsFieldRow
      label={label}
      description={description}
      control={
        <NativeSelect
          className="w-48"
          value={value}
          onChange={event => onChange(event.target.value as V)}
        >
          {options.map(option => (
            <NativeSelectOption key={option.value} value={option.value}>
              {option.label}
            </NativeSelectOption>
          ))}
        </NativeSelect>
      }
    />
  );
}
