import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";

import { useAutomationPage } from "@/hooks/routes/use-automation-page";
import { PanelSurface } from "@/storybook/story-layout";

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
  const page = useAutomationPage();

  return (
    <PanelSurface>
      <AutomationDetailPanel {...page.detailPanelProps} />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <AutomationDetailPanelFromPage />,
};

export const Error: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/api/automation/jobs/:id", ({ params }) =>
          HttpResponse.json(
            { error: `Failed to load automation job ${String(params.id)}` },
            { status: 500 }
          )
        ),
      ],
    },
  },
  render: () => <AutomationDetailPanelFromPage />,
};
