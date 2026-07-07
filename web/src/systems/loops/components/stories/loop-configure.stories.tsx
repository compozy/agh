import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { LoopConfigureSheet } from "../configure/loop-configure-sheet";
import { loopConfigFixture, loopDetailByName } from "../../mocks/fixtures";

const meta: Meta<typeof LoopConfigureSheet> = {
  title: "systems/loops/LoopConfigureSheet",
  component: LoopConfigureSheet,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

const deliveryLoop = loopDetailByName.get("software-delivery")!;
const watchLoop = loopDetailByName.get("reviews-watch")!;

const noop = () => {};

/** The full configure sheet: command check + command field, locked agent-judge, human gate,
 *  re-attempt cards, and the 6 clamped limit overrides seeded from a stored config. */
export const Delivery: Story = {
  render: () => (
    <StorySurface className="h-[880px] p-0">
      <LoopConfigureSheet
        open
        workspaceId="ws_default"
        loop={deliveryLoop}
        config={loopConfigFixture}
        onOpenChange={noop}
        onFork={noop}
      />
    </StorySurface>
  ),
};

/** No stored config — the sheet opens on the inherited loop defaults (all checks on, human
 *  gate off, failed-only, limits placeholder-only). */
export const InheritedDefaults: Story = {
  render: () => (
    <StorySurface className="h-[880px] p-0">
      <LoopConfigureSheet
        open
        workspaceId="ws_default"
        loop={deliveryLoop}
        config={null}
        onOpenChange={noop}
        onFork={noop}
      />
    </StorySurface>
  ),
};

/** A watch loop that declares no verification checks — the Review gate group is empty. */
export const WatchNoChecks: Story = {
  render: () => (
    <StorySurface className="h-[880px] p-0">
      <LoopConfigureSheet
        open
        workspaceId="ws_default"
        loop={watchLoop}
        config={null}
        onOpenChange={noop}
        onFork={noop}
      />
    </StorySurface>
  ),
};
