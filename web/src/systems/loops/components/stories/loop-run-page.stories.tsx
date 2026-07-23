import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { LoopRunPageBody } from "../run-page/loop-run-page-body";
import {
  buildScenarioProps,
  exhaustedScenario,
  failedScenario,
  needsApprovalScenario,
  noOpScenario,
  pausedScenario,
  runningScenario,
  watchingScenario,
  type LoopRunStoryScenario,
} from "./loop-run-page-fixtures";

/**
 * The redesigned run detail page (LOOP-RUN-REDESIGN-SPEC.md) across its §7
 * states, derived from synthetic frames through the production reducer + libs.
 * These stories are the visual-contract capture targets for the canonical
 * prototypes (`loop-run-detail.html` / `loop-run-detail-states.html`).
 */

function ScenarioPage({
  scenario,
  inspectInitiallyOpen = false,
}: {
  scenario: LoopRunStoryScenario;
  inspectInitiallyOpen?: boolean;
}) {
  const [inspectOpen, setInspectOpen] = useState(inspectInitiallyOpen);
  const props = buildScenarioProps(scenario);
  return (
    <div className="flex h-dvh flex-col bg-canvas">
      <LoopRunPageBody {...props} inspect={{ open: inspectOpen, onOpenChange: setInspectOpen }} />
    </div>
  );
}

const meta: Meta<typeof ScenarioPage> = {
  title: "systems/loops/components/LoopRunPage",
  component: ScenarioPage,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Running: Story = {
  render: () => <ScenarioPage scenario={runningScenario()} />,
};

export const NeedsApproval: Story = {
  render: () => <ScenarioPage scenario={needsApprovalScenario()} />,
};

export const Watching: Story = {
  render: () => <ScenarioPage scenario={watchingScenario()} />,
};

export const Paused: Story = {
  render: () => <ScenarioPage scenario={pausedScenario()} />,
};

export const Failed: Story = {
  render: () => <ScenarioPage scenario={failedScenario()} />,
};

export const Exhausted: Story = {
  render: () => <ScenarioPage scenario={exhaustedScenario()} />,
};

export const NoOp: Story = {
  render: () => <ScenarioPage scenario={noOpScenario()} />,
};

export const InspectOpen: Story = {
  render: () => <ScenarioPage scenario={runningScenario()} inspectInitiallyOpen />,
};
