import type { Meta, StoryObj } from "@storybook/react-vite";
import { Book, Plus } from "lucide-react";

import { Button } from "@agh/ui";
import { PanelSurface } from "@/storybook/story-layout";

import { WorkspacePageShell } from "../workspace-page-shell";

const meta: Meta<typeof WorkspacePageShell> = {
  title: "systems/workspace/WorkspacePageShell",
  component: WorkspacePageShell,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => (
    <PanelSurface className="min-h-[520px]">
      <WorkspacePageShell
        count={3}
        controls={
          <Button size="sm" variant="outline">
            Filter
          </Button>
        }
        icon={<Book className="size-4" />}
        meta={
          <Button size="sm">
            <Plus className="size-4" />
            Create
          </Button>
        }
        title="Knowledge"
      >
        <div className="grid min-h-0 flex-1 place-items-center text-sm text-[color:var(--color-text-secondary)]">
          Shell content area
        </div>
      </WorkspacePageShell>
    </PanelSurface>
  ),
};
