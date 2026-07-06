import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import { StorySurface } from "@/storybook/story-layout";

import { LoopCatalog } from "../catalog/loop-catalog";
import type { LoopBindingKind } from "../../lib/loop-bindings";
import type { LoopCatalogFilter } from "../../lib/loop-catalog";
import { loopCatalogFixtures } from "../../mocks/fixtures";

const meta: Meta<typeof LoopCatalog> = {
  title: "systems/loops/LoopCatalog",
  component: LoopCatalog,
  parameters: { layout: "fullscreen" },
};

export default meta;
type Story = StoryObj<typeof meta>;

const BOUND_LOOPS = new Map<string, LoopBindingKind[]>([["software-delivery", ["schedule"]]]);

function CatalogHarness() {
  const [filter, setFilter] = useState<LoopCatalogFilter>({ kind: "all", category: null });
  return (
    <StorySurface className="p-8">
      <div className="mx-auto max-w-[1320px]">
        <LoopCatalog
          entries={loopCatalogFixtures}
          filter={filter}
          onFilterChange={setFilter}
          boundLoops={BOUND_LOOPS}
          onRun={() => {}}
        />
      </div>
    </StorySurface>
  );
}

export const Default: Story = {
  render: () => <CatalogHarness />,
};

export const ReadOnlyOnly: Story = {
  render: () => (
    <StorySurface className="p-8">
      <div className="mx-auto max-w-[1320px]">
        <LoopCatalog
          entries={loopCatalogFixtures}
          filter={{ kind: "read-only", category: null }}
          onFilterChange={() => {}}
          boundLoops={BOUND_LOOPS}
          onRun={() => {}}
        />
      </div>
    </StorySurface>
  ),
};
