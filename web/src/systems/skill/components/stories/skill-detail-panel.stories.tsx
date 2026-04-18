import type { Meta, StoryObj } from "@storybook/react-vite";

import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { PanelSurface } from "@/storybook/story-layout";

import { SkillDetailPanel } from "../skill-detail-panel";

const meta: Meta<typeof SkillDetailPanel> = {
  title: "systems/skill/SkillDetailPanel",
  component: SkillDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function SkillDetailPanelFromPage() {
  const page = useSkillsPage();

  return (
    <PanelSurface>
      <SkillDetailPanel
        content={page.selectedSkillContent}
        contentError={page.contentError}
        error={page.detailError}
        isActionPending={page.isActionPending}
        isContentLoading={page.isContentLoading}
        isLoading={page.isLoadingDetail}
        onDisable={page.handleDisable}
        onEnable={page.handleEnable}
        onRetryContent={page.handleRetryContent}
        onViewContent={page.handleViewContent}
        skill={page.selectedSkill}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <SkillDetailPanelFromPage />,
};
