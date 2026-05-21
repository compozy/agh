import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect } from "react";
import { expect, userEvent, within } from "storybook/test";

import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { PanelSurface } from "@/storybook/story-layout";
import {
  primarySkillFixture,
  skillContentFixtures,
  skillShadowsFixtures,
} from "@/systems/skill/mocks/fixtures";

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
        isShadowsLoading={page.isLoadingShadows}
        onDisable={page.handleDisable}
        onEnable={page.handleEnable}
        onRetryContent={page.handleRetryContent}
        onViewContent={page.handleViewContent}
        skill={page.selectedSkill}
        shadows={page.selectedSkillShadows}
        shadowsError={page.shadowsError}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => <SkillDetailPanelFromPage selectName="merchant-dispute-triage" />,
};

/**
 * Disabled skill with the same provenance and resolution sections.
 */
export const DisabledSkill: Story = {
  args: {},
  render: () => <SkillDetailPanelFromPage selectName="payments-release-checks" />,
};

/**
 * Empty state shown before a skill is selected.
 */
export const Empty: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={null}
        isActionPending={false}
        isContentLoading={false}
        isLoading={false}
        isShadowsLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
        shadows={undefined}
        shadowsError={null}
      />
    </PanelSurface>
  ),
};

/**
 * Loading state while skill details are resolving.
 */
export const Loading: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={null}
        isActionPending={false}
        isContentLoading={false}
        isLoading={true}
        isShadowsLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
        shadows={undefined}
        shadowsError={null}
      />
    </PanelSurface>
  ),
};

/**
 * Error state when the skill registry cannot serve details.
 */
export const ErrorState: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        error={new Error("Skill registry offline")}
        isActionPending={false}
        isContentLoading={false}
        isLoading={false}
        isShadowsLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={undefined}
        shadows={undefined}
        shadowsError={null}
      />
    </PanelSurface>
  ),
};

/**
 * Detail panel with full SKILL.md content already loaded.
 */
export const WithLoadedContent: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={skillContentFixtures[primarySkillFixture.name]}
        contentError={null}
        error={null}
        isActionPending={false}
        isContentLoading={false}
        isLoading={false}
        isShadowsLoading={false}
        onDisable={() => undefined}
        onEnable={() => undefined}
        onRetryContent={() => undefined}
        onViewContent={() => undefined}
        skill={primarySkillFixture}
        shadows={skillShadowsFixtures[primarySkillFixture.name]}
        shadowsError={null}
      />
    </PanelSurface>
  ),
};

/**
 * Interaction test: toggling the Switch surfaces disable/enable call.
 */
export const ToggleSwitch: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <SkillDetailPanelFromPage selectName="merchant-dispute-triage" />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const toggle = await canvas.findByTestId("skill-enabled-switch");
    await userEvent.click(toggle);
    await expect(toggle).toBeDefined();
  },
};
