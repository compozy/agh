import type { AghApiOkJsonResponseFor } from "@/storybook/openapi-msw";
import { storyDefaultWorkspaceId } from "@/storybook/fintech-scenario";
import { windowManagerSnapshotFixture, windowManagerStoryWindowId } from "@/systems/os/mocks";
import type { SettingsWindowManagerSection } from "@/systems/settings";

type WindowManagerResourceRecord = AghApiOkJsonResponseFor<
  "get",
  "/api/workspaces/{workspace_id}/window-manager/layout-profiles"
>["records"][number];

type WindowManagerLayoutProfileWire = {
  version: 1;
  id: string;
  display_name: string;
  aspect_variant: "landscape" | "portrait" | "square" | "wide";
  participant_slots: string[];
  overflow_policy: "floating" | "ignore" | "stack";
  document: typeof windowManagerLayoutDocumentFixture;
};

type WindowManagerLayoutResourceFixture = Omit<WindowManagerResourceRecord, "spec"> & {
  spec: WindowManagerLayoutProfileWire;
};

export const settingsWindowManagerSectionFixture: SettingsWindowManagerSection = {
  section: "window-manager",
  scope: "global",
  available_scopes: ["global"],
  config: {
    new_window_policy: "floating",
    small_viewport_policy: "stack",
    focus_policy: "click_directional",
    focus_wrap: true,
    focus_follows_pointer: false,
    raise_on_focus: true,
    drag_away_policy: "window",
    group_move_modifier: "alt",
    swap_modifier: "shift",
    history_limit: 100,
    desktop_transition: "slide",
    gaps: {
      inner: 8,
      top: 8,
      right: 8,
      bottom: 8,
      left: 8,
    },
    snap: {
      edge_band: 24,
      corner_reach: 96,
      exit_slack: 16,
      repeat_ratios: [0.5, 0.33, 0.67],
    },
    bindings: {
      top_center: "zoom",
      bottom_center: "none",
    },
    shortcuts: {},
  },
};

export const windowManagerLayoutDocumentFixture: AghApiOkJsonResponseFor<
  "get",
  "/api/workspaces/{workspace_id}/window-manager/layout"
> = {
  version: 1,
  workspace_id: storyDefaultWorkspaceId,
  desktops: windowManagerSnapshotFixture.desktops,
  windows: windowManagerSnapshotFixture.windows,
  overrides: {},
};

export const windowManagerLayoutResourceFixture: WindowManagerLayoutResourceFixture = {
  kind: "window_layout",
  id: "launch-console",
  version: 3,
  scope: { kind: "workspace", id: storyDefaultWorkspaceId },
  owner: { kind: "daemon", id: "storybook" },
  source: { kind: "operator", id: "storybook" },
  spec: {
    version: 1,
    id: "launch-console",
    display_name: "Launch console",
    aspect_variant: "landscape",
    participant_slots: [windowManagerStoryWindowId],
    overflow_policy: "stack",
    document: windowManagerLayoutDocumentFixture,
  },
  created_at: "2026-07-22T22:00:00Z",
  updated_at: "2026-07-23T01:00:00Z",
};
