import type { Meta, StoryObj } from "@storybook/react-vite";
import { HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import { StorySurface } from "@/storybook/story-layout";

import { LoopEditor } from "../editor/loop-editor";
import { handlers as loopHandlers } from "../../mocks";
import { loopDetailByName } from "../../mocks/fixtures";
import type { LoopDetail } from "../../types";

const WS = "ws_default";
const delivery = loopDetailByName.get("software-delivery")!;

type RawGraph = { nodes: Record<string, unknown>[]; edges: unknown[] };

/** A software-delivery detail whose fan-out node breaches the ceiling, so the editor's
 *  auto-validate surfaces the fan_out_ceiling_exceeded issue + node badge + gate. */
function overCeilingDetail(): LoopDetail {
  const graph = delivery.definition.graph as unknown as RawGraph;
  const nodes = graph.nodes.map(node =>
    node.id === "implement" ? { ...node, max_fan_out: 80 } : node
  );
  return {
    ...delivery,
    definition: {
      ...delivery.definition,
      graph: { ...graph, nodes } as unknown as LoopDetail["definition"]["graph"],
    },
  };
}

function editorHandlers(detail: LoopDetail) {
  // The override getLoop must win, but keep the loop handlers (validate lints the posted
  // definition) so the auto-validate still surfaces the fan_out_ceiling_exceeded issue.
  return [
    aghApiMock.get("/api/workspaces/{workspace_id}/loops/{name}", () =>
      HttpResponse.json({ loop: detail })
    ),
    ...loopHandlers,
  ];
}

const meta: Meta<typeof LoopEditor> = {
  title: "systems/loops/components/LoopEditor",
  component: LoopEditor,
  parameters: { layout: "fullscreen" },
  render: args => (
    <StorySurface className="flex h-[880px] p-0">
      <LoopEditor {...args} />
    </StorySurface>
  ),
};

export default meta;
type Story = StoryObj<typeof meta>;

/** The clean editor over a workspace software-delivery Loop: canvas, palette,
 *  inspector, linter dock (all invariants pass), and the read-only Start summary. */
export const Editor: Story = {
  args: { workspaceId: WS, name: "software-delivery" },
};

/** A fan-out node over the daemon ceiling: the shared linter returns a per-node 422 —
 *  the fan-out chip fails, the node gets a danger ring + badge, and Publish is disabled. */
export const FanOutError: Story = {
  args: { workspaceId: WS, name: "software-delivery" },
  parameters: { msw: { handlers: editorHandlers(overCeilingDetail()) } },
};

/** Editing a read-only watch Loop before a workspace fork exists. */
export const WatchFork: Story = {
  args: { workspaceId: WS, name: "reviews-watch" },
};
