import { Switch } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";

interface WindowManagerToggleFieldProps {
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}

export function WindowManagerToggleField({
  label,
  description,
  checked,
  onChange,
}: WindowManagerToggleFieldProps) {
  return (
    <SettingsFieldRow
      label={label}
      description={description}
      control={<Switch checked={checked} onCheckedChange={onChange} />}
    />
  );
}
