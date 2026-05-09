import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { expect, fn, userEvent, within } from "storybook/test";

import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";
import {
  automationRunFixtures,
  primaryAutomationJobFixture,
  primaryAutomationTriggerFixture,
} from "@/systems/automation/mocks";

import { AutomationDetailPanel } from "../automation-detail-panel";

const meta: Meta<typeof AutomationDetailPanel> = {
  title: "systems/automation/AutomationDetailPanel",
  component: AutomationDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AutomationDetailPanelFromPage() {
  const page = useAutomationJobsPage();

  return (
    <PanelSurface>
      <AutomationDetailPanel {...page.detailPanelProps} />
    </PanelSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => <AutomationDetailPanelFromPage />,
};

export const Error: Story = {
  args: {},
  parameters: {
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/jobs/:id", ({ params }) =>
          HttpResponse.json(
            { error: `Failed to load automation job ${String(params.id)}` },
            { status: 500 }
          )
        ),
      ],
    }),
  },
  render: () => <AutomationDetailPanelFromPage />,
};

export const TriggerDefault: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <AutomationDetailPanel
        emptyState={null}
        error={null}
        state={{
          isDeleting: false,
          isLoading: false,
          isTogglePending: false,
          isTriggerPending: false,
        }}
        item={primaryAutomationTriggerFixture}
        kind="triggers"
        onDelete={fn()}
        onEdit={fn()}
        onToggleEnabled={fn()}
        onTriggerNow={fn()}
        runs={automationRunFixtures.filter(
          run => run.trigger_id === primaryAutomationTriggerFixture.id
        )}
        runsError={null}
        runsLoading={false}
      />
    </PanelSurface>
  ),
};

export const TriggerHook: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <AutomationDetailPanelFromPage />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.tab();
    await expect(canvas.findByTestId("automation-detail-panel")).resolves.toBeDefined();
  },
};

export const Loading: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <AutomationDetailPanel
        emptyState={null}
        error={null}
        state={{
          isDeleting: false,
          isLoading: true,
          isTogglePending: false,
          isTriggerPending: false,
        }}
        item={undefined}
        kind="jobs"
        onDelete={fn()}
        onEdit={fn()}
        onToggleEnabled={fn()}
        onTriggerNow={fn()}
        runs={[]}
        runsError={null}
        runsLoading={false}
      />
    </PanelSurface>
  ),
};

export const Deleting: Story = {
  args: {},
  render: () => (
    <PanelSurface>
      <AutomationDetailPanel
        emptyState={null}
        error={null}
        state={{
          isDeleting: true,
          isLoading: false,
          isTogglePending: false,
          isTriggerPending: false,
        }}
        item={primaryAutomationJobFixture}
        kind="jobs"
        onDelete={fn()}
        onEdit={fn()}
        onToggleEnabled={fn()}
        onTriggerNow={fn()}
        runs={automationRunFixtures.filter(run => run.job_id === primaryAutomationJobFixture.id)}
        runsError={null}
        runsLoading={false}
      />
    </PanelSurface>
  ),
};
