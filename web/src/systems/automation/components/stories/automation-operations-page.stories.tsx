import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { useAutomationJobsPage } from "@/hooks/routes/use-automation-page";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";

import { AutomationOperationsPage } from "../automation-operations-page";

function JobsPageStory() {
  const page = useAutomationJobsPage();
  return (
    <PanelSurface className="min-h-[760px] p-0">
      <AutomationOperationsPage
        createButtonTestId="automation-create-job"
        createLabel="New job"
        page={page}
        title="Jobs"
        titlePrefix="jobs"
      />
    </PanelSurface>
  );
}

const meta: Meta<typeof AutomationOperationsPage> = {
  title: "systems/automation/AutomationOperationsPage",
  component: AutomationOperationsPage,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Shared jobs/triggers page shell with list, detail, editor, and scope filters.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Jobs page rendered from the same route hook used by the app, backed by MSW.
 */
export const Default: Story = {
  args: {},
  render: () => <JobsPageStory />,
};

/**
 * Initial error state keeps the page truthful when the automation endpoint fails.
 */
export const InitialError: Story = {
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
  render: () => <JobsPageStory />,
};
