import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { LoopDetailView } from "../detail/loop-detail";
import type { LoopBindingRow } from "../../lib/loop-bindings";
import { readLoopGraph } from "../../lib/loop-graph";
import { loopCatalogFixtures, loopDetailByName, loopRunFixtures } from "../../mocks/fixtures";

const meta: Meta<typeof LoopDetailView> = {
  title: "systems/loops/components/LoopDetail",
  component: LoopDetailView,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

const loop = loopDetailByName.get("software-delivery")!;
const catalogEntry = loopCatalogFixtures.find(entry => entry.name === "software-delivery")!;
const recentRuns = loopRunFixtures.filter(run => run.loop_name === "software-delivery").slice(0, 5);

const BINDINGS: LoopBindingRow[] = [
  {
    id: "job_nightly",
    name: "nightly",
    kind: "schedule",
    enabled: false,
    meta: "Cron 0 3 * * * · next in 6h",
  },
];

const noop = () => {};

export const WithBinding: Story = {
  render: () => (
    <StorySurface>
      <LoopDetailView
        loop={loop}
        graph={readLoopGraph(loop.definition)}
        recentRuns={recentRuns}
        bindings={BINDINGS}
        bindingsLoading={false}
        successRate={catalogEntry.success_rate_30d}
        aggregate={catalogEntry.aggregate_30d}
        onBack={noop}
        onRun={noop}
        onConfigure={noop}
        onFork={noop}
        onAddTrigger={noop}
        onAddSchedule={noop}
      />
    </StorySurface>
  ),
};

export const NoBindings: Story = {
  render: () => (
    <StorySurface>
      <LoopDetailView
        loop={loop}
        graph={readLoopGraph(loop.definition)}
        recentRuns={recentRuns}
        bindings={[]}
        bindingsLoading={false}
        successRate={catalogEntry.success_rate_30d}
        aggregate={catalogEntry.aggregate_30d}
        onBack={noop}
        onRun={noop}
        onConfigure={noop}
        onFork={noop}
        onAddTrigger={noop}
        onAddSchedule={noop}
      />
    </StorySurface>
  ),
};
