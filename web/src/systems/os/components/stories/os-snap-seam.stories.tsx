import type { Meta, StoryObj } from "@storybook/react-vite";
import { useStore } from "zustand";

import { useOsShell } from "../../hooks/use-os-shell";
import { deriveSnapRect } from "../../lib/os-snap-geometry";
import { OS_SNAP_ZONES } from "../../lib/os-snap-zones";
import { OsSnapSeamLayer } from "../os-snap-seam";
import { OsWindowFrame } from "../os-window-frame";
import { createStoryShell, StoryShellProvider } from "./_shell";

const meta: Meta<typeof OsSnapSeamLayer> = {
  title: "systems/os/components/OsSnapSeam",
  component: OsSnapSeamLayer,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Linked seam between adjacent snapped windows: the shared 8px gutter grows a drag handle (ResizableHandle visual grammar — pill on hover/focus) that resizes BOTH neighbors by rewriting their fractions. Seams are emergent from fraction adjacency; drag the seam in the canvas to feel the linked resize.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const BOUNDS = { width: 1100, height: 680 };

/** Window silhouettes that track the store's derived rects (incl. live seam preview). */
function SnappedSilhouettes() {
  const { store } = useOsShell();
  const windows = useStore(store, state => state.windows);
  const seamPreview = useStore(store, state => state.seamPreview);
  return (
    <>
      {Object.values(windows).map(win => {
        const zone = seamPreview?.[win.id] ?? win.snap;
        if (zone === null) return null;
        const rect = deriveSnapRect(zone, BOUNDS);
        return (
          <div
            key={win.id}
            className="absolute"
            style={{ left: rect.x, top: rect.y, width: rect.w, height: rect.h }}
          >
            <OsWindowFrame title={win.app} focused={win.app === "tasks"} className="h-full w-full">
              <div className="flex flex-1 items-center justify-center text-small-body text-subtle">
                {win.app}
              </div>
            </OsWindowFrame>
          </div>
        );
      })}
    </>
  );
}

function SeamScene({ quarters = false }: { quarters?: boolean }) {
  return (
    <StoryShellProvider
      createShell={() =>
        createStoryShell(store => {
          const left = store.getState().openOrFocus({ app: "tasks" });
          const right = store.getState().openOrFocus({ app: "vault" });
          store.getState().clampToViewport(BOUNDS);
          store.getState().snapWindow(left, OS_SNAP_ZONES.left);
          if (quarters) {
            const third = store.getState().openOrFocus({ app: "sandbox" });
            store.getState().clampToViewport(BOUNDS);
            store.getState().snapWindow(right, OS_SNAP_ZONES["top-right"]);
            store.getState().snapWindow(third, OS_SNAP_ZONES["bottom-right"]);
          } else {
            store.getState().snapWindow(right, OS_SNAP_ZONES.right);
          }
        })
      }
    >
      <div
        data-slot="os-win-layer"
        className="relative overflow-hidden bg-rail"
        style={{ width: BOUNDS.width, height: BOUNDS.height }}
      >
        <SnappedSilhouettes />
        <OsSnapSeamLayer />
      </div>
    </StoryShellProvider>
  );
}

/** Two halves sharing one vertical seam — drag it to resize the pair. */
export const HalvesPair: Story = {
  args: {},
  render: () => <SeamScene />,
};

/** A half beside stacked quarters: two vertical seams + one horizontal. */
export const HalfWithQuarters: Story = {
  args: {},
  render: () => <SeamScene quarters />,
};
