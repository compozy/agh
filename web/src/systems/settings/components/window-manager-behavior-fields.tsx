import type { WindowManagerConfig } from "@/systems/os";

import { SettingsGroup } from "./settings-group";
import type { ConfigFieldsProps, SelectOption } from "./window-manager-config-field-types";
import { WindowManagerSelectField } from "./window-manager-select-field";
import { WindowManagerToggleField } from "./window-manager-toggle-field";

type DragModifier = WindowManagerConfig["swapModifier"];

const DRAG_MODIFIER_OPTIONS: readonly SelectOption<DragModifier>[] = [
  { value: "alt", label: "Alt" },
  { value: "control", label: "Control" },
  { value: "meta", label: "Meta" },
  { value: "shift", label: "Shift" },
  { value: "none", label: "None" },
];

export function WindowManagerBehaviorFields({ draft, setDraft }: ConfigFieldsProps) {
  return (
    <SettingsGroup
      title="Behavior"
      description="Global defaults hot-apply to every workspace unless that workspace overrides them."
    >
      <WindowManagerSelectField
        label="New windows"
        description="Open floating or beside the focused window"
        value={draft.newWindowPolicy}
        options={[
          { value: "floating", label: "Floating" },
          { value: "beside_focus", label: "Beside focus" },
        ]}
        onChange={newWindowPolicy => setDraft(current => ({ ...current, newWindowPolicy }))}
      />
      <WindowManagerSelectField
        label="Small viewport"
        description="Adapt the layout to a stack or reject the viewport"
        value={draft.smallViewportPolicy}
        options={[
          { value: "stack", label: "Adaptive stack" },
          { value: "reject", label: "Reject" },
        ]}
        onChange={smallViewportPolicy => setDraft(current => ({ ...current, smallViewportPolicy }))}
      />
      <WindowManagerSelectField
        label="Focus policy"
        description="Pointer clicks may focus, or focus can stay directional-only"
        value={draft.focusPolicy}
        options={[
          { value: "click_directional", label: "Click and directional" },
          { value: "directional", label: "Directional only" },
        ]}
        onChange={focusPolicy => setDraft(current => ({ ...current, focusPolicy }))}
      />
      <WindowManagerToggleField
        label="Wrap directional focus"
        description="Continue at the opposite edge when no window is in that direction"
        checked={draft.focusWrap}
        onChange={focusWrap => setDraft(current => ({ ...current, focusWrap }))}
      />
      <WindowManagerToggleField
        label="Focus follows pointer"
        description="Focus a window when the pointer enters it"
        checked={draft.focusFollowsPointer}
        onChange={focusFollowsPointer => setDraft(current => ({ ...current, focusFollowsPointer }))}
      />
      <WindowManagerToggleField
        label="Raise on focus"
        description="Bring a focused floating window above its peers"
        checked={draft.raiseOnFocus}
        onChange={raiseOnFocus => setDraft(current => ({ ...current, raiseOnFocus }))}
      />
      <WindowManagerSelectField
        label="Drag tiled windows"
        description="Move one window or its complete tiled group"
        value={draft.dragAwayPolicy}
        options={[
          { value: "window", label: "Window" },
          { value: "group", label: "Group" },
        ]}
        onChange={dragAwayPolicy => setDraft(current => ({ ...current, dragAwayPolicy }))}
      />
      <WindowManagerSelectField
        label="Group move modifier"
        description="Temporarily move the group while dragging"
        value={draft.groupMoveModifier}
        options={DRAG_MODIFIER_OPTIONS}
        onChange={groupMoveModifier => setDraft(current => ({ ...current, groupMoveModifier }))}
      />
      <WindowManagerSelectField
        label="Swap modifier"
        description="Hold while dropping onto a window to swap places"
        value={draft.swapModifier}
        options={DRAG_MODIFIER_OPTIONS}
        onChange={swapModifier => setDraft(current => ({ ...current, swapModifier }))}
      />
      <WindowManagerSelectField
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
