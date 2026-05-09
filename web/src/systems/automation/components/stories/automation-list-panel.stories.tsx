import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { expect, fn, userEvent, within } from "storybook/test";

import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";
import { storyDefaultWorkspaceName } from "@/storybook/fintech-scenario";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";
import { automationJobFixtures, automationTriggerFixtures } from "@/systems/automation/mocks";

import { AutomationListPanel } from "../automation-list-panel";

const meta: Meta<typeof AutomationListPanel> = {
  title: "systems/automation/AutomationListPanel",
  component: AutomationListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AutomationListPanelFromPage() {
  const page = useAutomationJobsPage();

  return (
    <PanelSurface className="max-w-[340px]">
      <AutomationListPanel {...page.listPanelProps} />
    </PanelSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => <AutomationListPanelFromPage />,
};

export const Empty: Story = {
  args: {},
  parameters: {
    ...storybookMswParameters({
      automation: [http.get("/api/automation/jobs", () => HttpResponse.json({ jobs: [] }))],
    }),
  },
  render: () => <AutomationListPanelFromPage />,
};

export const Error: Story = {
  args: {},
  parameters: {
    ...storybookMswParameters({
      automation: [
        http.get("/api/automation/jobs", () =>
          HttpResponse.json({ error: "automation unavailable" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <AutomationListPanelFromPage />,
};

export const TriggersDefault: Story = {
  args: {},
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <AutomationListPanel
        activeWorkspaceName={storyDefaultWorkspaceName}
        jobs={automationJobFixtures}
        kind="triggers"
        onSearchChange={fn()}
        onSelect={fn()}
        scopeFilter="all"
        searchQuery=""
        selectedId={automationTriggerFixtures[0]?.id ?? null}
        totalCount={automationTriggerFixtures.length}
        triggers={automationTriggerFixtures}
      />
    </PanelSurface>
  ),
};

export const TriggersEmpty: Story = {
  args: {},
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <AutomationListPanel
        activeWorkspaceName={storyDefaultWorkspaceName}
        jobs={[]}
        kind="triggers"
        onSearchChange={fn()}
        onSelect={fn()}
        scopeFilter="all"
        searchQuery=""
        selectedId={null}
        totalCount={0}
        triggers={[]}
      />
    </PanelSurface>
  ),
};

export const SearchFilter: Story = {
  args: {},
  tags: ["play-fn"],
  render: () => <AutomationListPanelFromPage />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const search = await canvas.findByTestId("automation-search-input");
    await userEvent.clear(search);
    await userEvent.type(search, "payout");
    await expect(
      canvas.findByTestId("automation-item-job_payout_watchlist")
    ).resolves.toBeDefined();
  },
};

export const Loading: Story = {
  args: {},
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <AutomationListPanel
        activeWorkspaceName={storyDefaultWorkspaceName}
        isLoading
        jobs={[]}
        kind="jobs"
        onSearchChange={fn()}
        onSelect={fn()}
        scopeFilter="all"
        searchQuery=""
        selectedId={null}
        totalCount={0}
        triggers={[]}
      />
    </PanelSurface>
  ),
};
