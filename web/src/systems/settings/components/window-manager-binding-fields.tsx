import { Input, Textarea } from "@agh/ui";

import { SettingsFieldRow } from "./settings-field-row";
import { SettingsGroup } from "./settings-group";
import type { ConfigFieldsProps } from "./window-manager-config-field-types";
import { WindowManagerSelectField } from "./window-manager-select-field";

interface WindowManagerBindingFieldsProps extends ConfigFieldsProps {
  shortcutsText: string;
  shortcutsValid: boolean;
  setShortcutsText: (value: string) => void;
}

export function WindowManagerBindingFields({
  draft,
  setDraft,
  shortcutsText,
  shortcutsValid,
  setShortcutsText,
}: WindowManagerBindingFieldsProps) {
  return (
    <SettingsGroup
      title="Bindings and history"
      description="Edge-center claims, revision history, and the canonical shortcut map."
    >
      <WindowManagerSelectField
        label="Top center"
        description="Special action at the center of the top edge"
        value={draft.bindings.topCenter}
        options={[
          { value: "none", label: "No special action" },
          { value: "reserved", label: "Reserved" },
          { value: "zoom", label: "Zoom" },
        ]}
        onChange={topCenter =>
          setDraft(current => ({
            ...current,
            bindings: { ...current.bindings, topCenter },
          }))
        }
      />
      <WindowManagerSelectField
        label="Bottom center"
        description="Special action at the center of the bottom edge"
        value={draft.bindings.bottomCenter}
        options={[
          { value: "none", label: "No special action" },
          { value: "reserved", label: "Reserved" },
          { value: "zoom", label: "Zoom" },
        ]}
        onChange={bottomCenter =>
          setDraft(current => ({
            ...current,
            bindings: { ...current.bindings, bottomCenter },
          }))
        }
      />
      <SettingsFieldRow
        label="History limit"
        description="Maximum undo and redo operations retained per workspace"
        control={
          <Input
            aria-invalid={
              !Number.isInteger(draft.historyLimit) ||
              draft.historyLimit < 1 ||
              draft.historyLimit > 500
            }
            className="h-11 w-32"
            min={1}
            max={500}
            step={1}
            type="number"
            value={Number.isFinite(draft.historyLimit) ? draft.historyLimit : ""}
            onChange={event => {
              const historyLimit = event.currentTarget.valueAsNumber;
              setDraft(current => ({ ...current, historyLimit }));
            }}
          />
        }
      />
      <div className="border-t border-line p-4">
        <label className="flex flex-col gap-1 text-form-label text-muted">
          Shortcut map
          <Textarea
            aria-invalid={!shortcutsValid}
            className="min-h-40 font-mono text-mono-id"
            value={shortcutsText}
            onChange={event => setShortcutsText(event.target.value)}
          />
        </label>
      </div>
    </SettingsGroup>
  );
}
