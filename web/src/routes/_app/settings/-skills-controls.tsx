import { Input } from "@agh/ui";

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
