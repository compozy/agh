import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { OsTrafficLights } from "../os-traffic-lights";
import { OsZoomMenu } from "../os-zoom-menu";
import { StoryShellProvider } from "./_shell";

const meta: Meta<typeof OsZoomMenu> = {
  title: "systems/os/components/OsZoomMenu",
  component: OsZoomMenu,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "macOS-style zoom-button menu: hover the zoom traffic light (~250ms intent) to open Move & Resize (halves + quarters as zone glyphs, restore while snapped) and Fill & Arrange (fill, 2-up, grid — disabled without a second visible window). Click stays toggleZoom; the palette carries every action for keyboard parity.",
      },
    },
  },
  decorators: [
    Story => (
      <div className="flex h-72 w-96 items-start justify-center bg-canvas p-6">
        <Story />
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/** Hover the zoom (third) control to open the menu; arranges are live. */
export const Default: Story = {
  args: {},
  render: () => (
    <StoryShellProvider
      setup={store => {
        store.getState().openOrFocus({ app: "tasks" });
        store.getState().openOrFocus({ app: "vault" });
        store.getState().clampToViewport({ width: 1100, height: 680 });
      }}
    >
      <OsTrafficLights
        onSelect={fn()}
        wrapZoom={button => <OsZoomMenu windowId="app:tasks">{button}</OsZoomMenu>}
      />
    </StoryShellProvider>
  ),
};

/** Single window: Fill & Arrange presets disable (truthful UI — no fake targets). */
export const ArrangeDisabled: Story = {
  args: {},
  render: () => (
    <StoryShellProvider
      setup={store => {
        store.getState().openOrFocus({ app: "tasks" });
        store.getState().clampToViewport({ width: 1100, height: 680 });
      }}
    >
      <OsTrafficLights
        onSelect={fn()}
        wrapZoom={button => <OsZoomMenu windowId="app:tasks">{button}</OsZoomMenu>}
      />
    </StoryShellProvider>
  ),
};
