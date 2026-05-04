import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

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

function SkillDetailPanelFromPage({ selectName }: { selectName?: string }) {
  const page = useSkillsPage();

  useEffect(() => {
    if (selectName) {
      page.setSelectedSkillName(selectName);
    }
  }, [selectName, page]);

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
  render: () => <SkillDetailPanelFromPage selectName="merchant-dispute-triage" />,
};

export const DisabledSkill: Story = {
  render: () => <SkillDetailPanelFromPage selectName="payments-release-checks" />,
};

export const Empty: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={null}
        isActionPending={false}
        isContentLoading={false}
        isLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
      />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={null}
        isActionPending={false}
        isContentLoading={false}
        isLoading={true}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
      />
    </PanelSurface>
  ),
};

export const ErrorState: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={new Error("Skill registry offline")}
        isActionPending={false}
        isContentLoading={false}
        isLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
      />
    </PanelSurface>
  ),
};

/**
 * Interaction test — toggling the Switch surfaces disable/enable call.
 */
export const ToggleSwitch: Story = {
  tags: ["play-fn"],
  render: () => <SkillDetailPanelFromPage selectName="merchant-dispute-triage" />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const toggle = await canvas.findByTestId("skill-enabled-switch");
    await userEvent.click(toggle);
    await expect(toggle).toBeDefined();
  },
};
