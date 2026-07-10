import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { LoopEditorWatchEvents } from "../editor/loop-editor-watch-events";
import { LoopWatchEventsPanel } from "../run-page/loop-watch-events-panel";
import type { LoopWatchEventsState } from "../../types";

/**
 * The two `watch-events` surfaces (WS1): the authoring subscription list editor (kind
 * select fed by the supported-kind matrix + CEL filter per entry) and the run-detail
 * read-model panel that renders the parked subscriptions, cursors, and last wake.
 */
const meta: Meta<typeof LoopEditorWatchEvents> = {
  title: "systems/loops/components/LoopWatchEvents",
  component: LoopEditorWatchEvents,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component: "Surfaces for configuring event subscriptions and inspecting a parked loop run.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const parked: LoopWatchEventsState = {
  subscriptions: [
    { kind: "task.status_changed", filter: "event.payload.to_status == 'completed'" },
    { kind: "loop.terminal" },
  ],
  cursors: { task_events: 4218, loop_run_events: 76 },
  last_wake_at: "2026-07-08T12:00:00Z",
};

function EditorHarness() {
  const [value, setValue] = useState<unknown>([
    { kind: "task.status_changed", filter: "event.payload.to_status == 'completed'" },
    { kind: "loop.terminal" },
  ]);
  return (
    <div className="w-[320px] rounded-lg border border-line bg-canvas p-4">
      <span className="mb-1.5 block text-[11.5px] font-medium text-fg-strong">Subscriptions</span>
      <LoopEditorWatchEvents value={value} onChange={setValue} />
    </div>
  );
}

/** The inspector subscription editor for a `watch-events` source node. */
export const SubscriptionEditor: Story = {
  args: {},
  render: () => (
    <StorySurface className="p-8">
      <EditorHarness />
    </StorySurface>
  ),
};

/** The run-detail "Watching events" rail panel (the parked read-model). */
export const RunDetailPanel: Story = {
  args: {},
  render: () => (
    <StorySurface className="p-8">
      <div className="w-[332px] border-l border-line bg-canvas-soft">
        <LoopWatchEventsPanel state={parked} />
      </div>
    </StorySurface>
  ),
};
