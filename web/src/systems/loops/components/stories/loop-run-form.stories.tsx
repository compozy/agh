import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { LoopRunForm } from "../run-form/loop-run-form";
import { loopDetailByName } from "../../mocks/fixtures";

const meta: Meta<typeof LoopRunForm> = {
  title: "systems/loops/components/LoopRunForm",
  component: LoopRunForm,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

const deliveryLoop = loopDetailByName.get("software-delivery")!;
const watchLoop = loopDetailByName.get("reviews-watch")!;

/** The hero run form: auto-generated typed inputs, Advanced overrides, live preview. */
export const Delivery: Story = {
  render: () => (
    <StorySurface className="flex h-[840px] flex-col p-0">
      <LoopRunForm workspaceId="ws_default" loop={deliveryLoop} />
    </StorySurface>
  ),
};

/** A watch loop with no declared inputs — run it directly. */
export const NoInputs: Story = {
  render: () => (
    <StorySurface className="flex h-[840px] flex-col p-0">
      <LoopRunForm workspaceId="ws_default" loop={watchLoop} />
    </StorySurface>
  ),
};
