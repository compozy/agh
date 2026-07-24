import { Input } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";
import { SettingsGroup } from "./settings-group";
import type { ConfigFieldsProps } from "./window-manager-config-field-types";
import { WindowManagerNumberField } from "./window-manager-number-field";

interface WindowManagerGeometryFieldsProps extends ConfigFieldsProps {
  ratioText: string;
  ratiosValid: boolean;
  setRatioText: (value: string) => void;
}

export function WindowManagerGeometryFields({
  draft,
  setDraft,
  ratioText,
  ratiosValid,
  setRatioText,
}: WindowManagerGeometryFieldsProps) {
  return (
    <SettingsGroup
      title="Geometry"
      description="Pixel thresholds and gaps used by the runtime projector and gesture resolver."
    >
      <div className="flex flex-wrap gap-3 p-4">
        {(
          [
            ["Inner gap", "inner"],
            ["Top gap", "top"],
            ["Right gap", "right"],
            ["Bottom gap", "bottom"],
            ["Left gap", "left"],
          ] as const
        ).map(([label, key]) => (
          <WindowManagerNumberField
            key={key}
            label={label}
            value={draft.gaps[key]}
            maximum={64}
            integer
            onChange={value =>
              setDraft(current => ({
                ...current,
                gaps: { ...current.gaps, [key]: value },
              }))
            }
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-3 border-t border-line p-4">
        <WindowManagerNumberField
          label="Edge band"
          minimum={4}
          maximum={128}
          integer
          value={draft.snap.edgeBand}
          onChange={edgeBand =>
            setDraft(current => ({
              ...current,
              snap: { ...current.snap, edgeBand },
            }))
          }
        />
        <WindowManagerNumberField
          label="Corner reach"
          minimum={16}
          maximum={512}
          integer
          value={draft.snap.cornerReach}
          onChange={cornerReach =>
            setDraft(current => ({
              ...current,
              snap: { ...current.snap, cornerReach },
            }))
          }
        />
        <WindowManagerNumberField
          label="Exit slack"
          maximum={64}
          integer
          value={draft.snap.exitSlack}
          onChange={exitSlack =>
            setDraft(current => ({
              ...current,
              snap: { ...current.snap, exitSlack },
            }))
          }
        />
      </div>
      <SettingsFieldRow
        label="Repeat ratios"
        description="Comma-separated values from 0.1 through 0.9"
        control={
          <Input
            aria-invalid={!ratiosValid}
            className="w-56 font-mono"
            value={ratioText}
            onChange={event => setRatioText(event.target.value)}
          />
        }
      />
    </SettingsGroup>
  );
}
