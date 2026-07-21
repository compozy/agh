import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";

import {
  StorybookRouteCanvas,
  StorybookWorkspaceSetup,
  appRouteParameters,
} from "@/storybook/route-story-meta";
import { desktopStore } from "@/systems/os/stores/desktop-store";
import type { OsWallpaper } from "@/systems/os/lib/os-types";

const meta: Meta<typeof StorybookRouteCanvas> = {
  title: "systems/settings/routes/SettingsAppearance",
  component: StorybookRouteCanvas,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Appearance pane route stories rendered through the real app shell: per-space wallpaper selection, dock magnification, and the in-product reduce-motion preference. Canonical VC-02 capture surface.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function StorybookWallpaperSetup({ wallpaper }: { wallpaper: OsWallpaper }) {
  useEffect(() => {
    desktopStore.getState().setWallpaper(wallpaper);
  }, [wallpaper]);

  return null;
}

/** Default appearance pane over the ember space. */
export const Default: Story = {
  args: {},
  parameters: appRouteParameters("/settings/appearance"),
  render: () => <StorybookWorkspaceSetup />,
};

/** Mesh wallpaper applied to the active space (VC-02 capture). */
export const Mesh: Story = {
  args: {},
  parameters: appRouteParameters("/settings/appearance"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookWallpaperSetup wallpaper="mesh" />
    </>
  ),
};

/** Carbon wallpaper applied to the active space (VC-02 capture). */
export const Carbon: Story = {
  args: {},
  parameters: appRouteParameters("/settings/appearance"),
  render: () => (
    <>
      <StorybookWorkspaceSetup />
      <StorybookWallpaperSetup wallpaper="carbon" />
    </>
  ),
};
