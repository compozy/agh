import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import {
  primarySkillFixture,
  skillContentFixtures,
  skillShadowsFixtures,
} from "@/systems/skill/mocks/fixtures";
import type { SkillPayload } from "@/systems/skill";

import { SkillDetailPanel } from "../skill-detail-panel";

const meta: Meta<typeof SkillDetailPanel> = {
  title: "systems/skill/components/SkillDetailPanel",
  component: SkillDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const disabledSkillFixture: SkillPayload = {
  ...primarySkillFixture,
  name: "payments-release-checks",
  enabled: false,
};

function DetailHarness(props: Partial<React.ComponentProps<typeof SkillDetailPanel>> = {}) {
  return (
    <PanelSurface>
      <SkillDetailPanel
        content={props.content}
        contentError={props.contentError ?? null}
        contentStatus={props.contentStatus ?? "ready"}
        detailStatus={props.detailStatus ?? "ready"}
        error={props.error ?? null}
        actionStatus={props.actionStatus ?? "idle"}
        shadowsStatus={props.shadowsStatus ?? "ready"}
        onBack={props.onBack ?? (() => undefined)}
        onDisable={props.onDisable ?? (() => undefined)}
        onEnable={props.onEnable ?? (() => undefined)}
        onRetryContent={props.onRetryContent ?? (() => undefined)}
        onViewContent={props.onViewContent ?? (() => undefined)}
        shadows={props.shadows ?? skillShadowsFixtures[primarySkillFixture.name]}
        shadowsError={props.shadowsError ?? null}
        skill={
          props.skill === undefined && (props.detailStatus === "loading" || props.error)
            ? undefined
            : (props.skill ?? primarySkillFixture)
        }
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <DetailHarness />,
};

export const DisabledSkill: Story = {
  render: () => <DetailHarness skill={disabledSkillFixture} />,
};

export const Empty: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        contentStatus="ready"
        detailStatus="ready"
        error={null}
        actionStatus="idle"
        shadowsStatus="ready"
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

export const Loading: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        contentStatus="ready"
        detailStatus="loading"
        error={null}
        actionStatus="idle"
        shadowsStatus="ready"
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

export const ErrorState: Story = {
  render: () => (
    <PanelSurface>
      <SkillDetailPanel
        content={undefined}
        contentError={null}
        contentStatus="ready"
        detailStatus="error"
        error={new Error("Skill registry offline")}
        actionStatus="idle"
        shadowsStatus="ready"
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

export const WithLoadedContent: Story = {
  render: () => (
    <DetailHarness
      content={skillContentFixtures[primarySkillFixture.name]}
      shadows={skillShadowsFixtures[primarySkillFixture.name]}
    />
  ),
};

export const ToggleSwitch: Story = {
  tags: ["play-fn"],
  render: () => <DetailHarness />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const toggle = await canvas.findByTestId("skill-enabled-switch");
    await userEvent.click(toggle);
    await expect(toggle).toBeDefined();
  },
};
