import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { useKnowledgePage } from "@/hooks/routes/use-knowledge-page";
import { PanelSurface } from "@/storybook/story-layout";

import { KnowledgeDetailPanel } from "../knowledge-detail-panel";

const meta: Meta<typeof KnowledgeDetailPanel> = {
  title: "systems/knowledge/KnowledgeDetailPanel",
  component: KnowledgeDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function KnowledgeDetailPanelFromPage() {
  const page = useKnowledgePage();

  return (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={page.selectedContent}
        error={page.contentError}
        isDeletePending={page.isDeletePending}
        isLoading={page.isContentLoading}
        memory={page.selectedMemory}
        onDelete={page.handleDelete}
        scope={page.selectedScope}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <KnowledgeDetailPanelFromPage />,
};

export const Empty: Story = {
  parameters: {
    msw: {
      handlers: [http.get("/api/memory", () => HttpResponse.json([]))],
    },
  },
  render: () => <KnowledgeDetailPanelFromPage />,
};
