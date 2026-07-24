import type { Dispatch, SetStateAction } from "react";

import { Input, NativeSelect, NativeSelectOption, Switch, Textarea } from "@agh/ui";
import type { WindowManagerConfig } from "@/systems/os";

import { SettingsFieldRow } from "./settings-field-row";
import { SettingsGroup } from "./settings-group";

interface ConfigFieldsProps {
  draft: WindowManagerConfig;
  setDraft: Dispatch<SetStateAction<WindowManagerConfig>>;
}

interface SelectOption<V extends string> {
  value: V;
  label: string;
}

function SelectField<V extends string>({
  label,
  description,
  value,
  options,
  onChange,
}: {
  label: string;
  description: string;
  value: V;
  options: readonly SelectOption<V>[];
  onChange: (value: V) => void;
}) {
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

function ToggleField({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <SettingsFieldRow
      label={label}
      description={description}
      control={<Switch checked={checked} onCheckedChange={onChange} />}
    />
  );
}

function NumberField({
  label,
  value,
  minimum = 0,
  maximum,
  integer = false,
  onChange,
}: {
  label: string;
  value: number;
  minimum?: number;
  maximum?: number;
  integer?: boolean;
  onChange: (value: number) => void;
}) {
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

export function WindowManagerBehaviorFields({ draft, setDraft }: ConfigFieldsProps) {
  return (
    <SettingsGroup
      title="Behavior"
      description="Global defaults hot-apply to every workspace unless that workspace overrides them."
    >
      <SelectField
        label="New windows"
        description="Open floating or beside the focused window"
        value={draft.newWindowPolicy}
        options={[
          { value: "floating", label: "Floating" },
          { value: "beside_focus", label: "Beside focus" },
        ]}
        onChange={newWindowPolicy => setDraft(current => ({ ...current, newWindowPolicy }))}
      />
      <SelectField
        label="Small viewport"
        description="Adapt the layout to a stack or reject the viewport"
        value={draft.smallViewportPolicy}
        options={[
          { value: "stack", label: "Adaptive stack" },
          { value: "reject", label: "Reject" },
        ]}
        onChange={smallViewportPolicy => setDraft(current => ({ ...current, smallViewportPolicy }))}
      />
      <SelectField
        label="Focus policy"
        description="Pointer clicks may focus, or focus can stay directional-only"
        value={draft.focusPolicy}
        options={[
          { value: "click_directional", label: "Click and directional" },
          { value: "directional", label: "Directional only" },
        ]}
        onChange={focusPolicy => setDraft(current => ({ ...current, focusPolicy }))}
      />
      <ToggleField
        label="Wrap directional focus"
        description="Continue at the opposite edge when no window is in that direction"
        checked={draft.focusWrap}
        onChange={focusWrap => setDraft(current => ({ ...current, focusWrap }))}
      />
      <ToggleField
        label="Focus follows pointer"
        description="Focus a window when the pointer enters it"
        checked={draft.focusFollowsPointer}
        onChange={focusFollowsPointer => setDraft(current => ({ ...current, focusFollowsPointer }))}
      />
      <ToggleField
        label="Raise on focus"
        description="Bring a focused floating window above its peers"
        checked={draft.raiseOnFocus}
        onChange={raiseOnFocus => setDraft(current => ({ ...current, raiseOnFocus }))}
      />
      <SelectField
        label="Drag tiled windows"
        description="Move one window or its complete tiled group"
        value={draft.dragAwayPolicy}
        options={[
          { value: "window", label: "Window" },
          { value: "group", label: "Group" },
        ]}
        onChange={dragAwayPolicy => setDraft(current => ({ ...current, dragAwayPolicy }))}
      />
      <SelectField
        label="Group move modifier"
        description="Temporarily move the group while dragging"
        value={draft.groupMoveModifier}
        options={[
          { value: "alt", label: "Alt" },
          { value: "control", label: "Control" },
          { value: "meta", label: "Meta" },
          { value: "shift", label: "Shift" },
          { value: "none", label: "None" },
        ]}
        onChange={groupMoveModifier => setDraft(current => ({ ...current, groupMoveModifier }))}
      />
      <SelectField
        label="Desktop transition"
        description="Animation used when this browser switches desktops"
        value={draft.desktopTransition}
        options={[
          { value: "slide", label: "Slide" },
          { value: "crossfade", label: "Crossfade" },
          { value: "instant", label: "Instant" },
        ]}
        onChange={desktopTransition => setDraft(current => ({ ...current, desktopTransition }))}
      />
    </SettingsGroup>
  );
}

export function WindowManagerGeometryFields({
  draft,
  setDraft,
  ratioText,
  ratiosValid,
  setRatioText,
}: ConfigFieldsProps & {
  ratioText: string;
  ratiosValid: boolean;
  setRatioText: (value: string) => void;
}) {
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
          <NumberField
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
        <NumberField
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
        <NumberField
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
        <NumberField
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

export function WindowManagerBindingFields({
  draft,
  setDraft,
  shortcutsText,
  shortcutsValid,
  setShortcutsText,
}: ConfigFieldsProps & {
  shortcutsText: string;
  shortcutsValid: boolean;
  setShortcutsText: (value: string) => void;
}) {
  return (
    <SettingsGroup
      title="Bindings and history"
      description="Edge-center claims, revision history, and the canonical shortcut map."
    >
      <SelectField
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
      <SelectField
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
              setDraft(current => ({
                ...current,
                historyLimit,
              }));
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
