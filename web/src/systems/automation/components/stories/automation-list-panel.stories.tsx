import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton } from "@agh/ui";
import { http, HttpResponse } from "msw";

import { useAutomationPage } from "@/hooks/routes/use-automation-page";
import { PanelSurface } from "@/storybook/story-layout";

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

function AutomationListLoadingState() {
  return (
    <PanelSurface className="max-w-[320px]">
      <aside className="flex w-[320px] flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-4">
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

function AutomationListPanelFromPage() {
  const page = useAutomationPage();

  if (page.isInitialLoading) {
    return <AutomationListLoadingState />;
  }

  if (page.initialError) {
    return (
      <PanelSurface className="max-w-[320px] items-center justify-center px-6 text-center text-sm text-[color:var(--color-danger)]">
        {page.initialError.message}
      </PanelSurface>
    );
  }

  return (
    <PanelSurface className="max-w-[320px]">
      <AutomationListPanel {...page.listPanelProps} />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <AutomationListPanelFromPage />,
};

export const Empty: Story = {
  parameters: {
    msw: {
      handlers: [http.get("/api/automation/jobs", () => HttpResponse.json({ jobs: [] }))],
    },
  },
  render: () => <AutomationListPanelFromPage />,
};

export const Error: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/api/automation/jobs", () =>
          HttpResponse.json({ error: "automation unavailable" }, { status: 500 })
        ),
      ],
    },
  },
  render: () => <AutomationListPanelFromPage />,
};
