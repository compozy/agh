import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { OsMenuBar } from "../os-menubar";
import { DesktopShell } from "./_desktop";

const meta: Meta<typeof OsMenuBar> = {
  title: "systems/os/components/OsMenuBar",
  component: OsMenuBar,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "The desktop menubar: AGH mark, workspace trigger, app menus, the approvals bell, the ⌘K palette chip, and Settings. Glass shell chrome. Controls are buttons only when a callback is supplied.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Populated — workspace trigger, full menus, bell with a count, ⌘K chip, and
 * Settings, all wired with callbacks, over the carbon desktop.
 */
export const Populated: Story = {
  args: {
    workspace: { name: "agh", monogram: "AG" },
    menus: ["Session", "View", "Help"],
    notifications: 2,
    onLogoClick: fn(),
    onWorkspaceClick: fn(),
    onMenuClick: fn(),
    onNotificationsClick: fn(),
    onCommandClick: fn(),
    onSettingsClick: fn(),
  },
  render: args => (
    <DesktopShell menubar={false} wallpaper="carbon" deskHint>
      <OsMenuBar {...args} />
    </DesktopShell>
  ),
};

/**
 * Presentation-only — no callbacks, controls render as inert chrome.
 */
export const PresentationOnly: Story = {
  args: { workspace: { name: "agh", monogram: "AG" }, notifications: 0 },
  render: args => (
    <DesktopShell menubar={false} wallpaper="carbon" deskHint>
      <OsMenuBar {...args} />
    </DesktopShell>
  ),
};
