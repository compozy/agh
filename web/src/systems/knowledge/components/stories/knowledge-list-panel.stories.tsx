import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton } from "@agh/ui";
import { delay, http, HttpResponse } from "msw";

import { useKnowledgePage } from "@/hooks/routes/use-knowledge-page";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";
import { KnowledgeListPanel } from "@/systems/knowledge/components/knowledge-list-panel";

const meta: Meta<typeof KnowledgeListPanel> = {
  title: "systems/knowledge/KnowledgeListPanel",
  component: KnowledgeListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function KnowledgeListLoadingState() {
  return (
    <PanelSurface className="max-w-[280px]">
      <aside className="flex w-[280px] flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-3">
        <div className="space-y-3">
          <Skeleton className="h-9 w-full rounded-lg" />
          <Skeleton className="h-16 w-full rounded-xl" />
          <Skeleton className="h-16 w-full rounded-xl" />
          <Skeleton className="h-16 w-full rounded-xl" />
        </div>
      </aside>
    </PanelSurface>
  );
}

function KnowledgeListPanelFromPage() {
  const page = useKnowledgePage();

  if (page.isLoading) {
    return <KnowledgeListLoadingState />;
  }

  return (
    <PanelSurface className="max-w-[280px]">
      <KnowledgeListPanel
        memories={page.memories}
        selectedFilename={page.effectiveSelectedFilename}
        onSearchChange={page.setSearchQuery}
        onSelectMemory={page.setSelectedFilename}
        searchQuery={page.searchQuery}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => <KnowledgeListPanelFromPage />,
};

export const Loading: Story = {
  args: {},
  parameters: {
    ...storybookMswParameters({
      knowledge: [
        http.get("/api/memory", async () => {
          await delay("infinite");
          return HttpResponse.json([]);
        }),
      ],
    }),
  },
  render: () => <KnowledgeListPanelFromPage />,
};
