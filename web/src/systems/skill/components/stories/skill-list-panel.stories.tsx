import type { Meta, StoryObj } from "@storybook/react-vite";
import { Skeleton } from "@agh/ui";
import { delay, http, HttpResponse } from "msw";

import { storybookMswParameters } from "@/storybook/msw";
import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { PanelSurface } from "@/storybook/story-layout";

import { SkillListPanel } from "../skill-list-panel";

const meta: Meta<typeof SkillListPanel> = {
  title: "systems/skill/SkillListPanel",
  component: SkillListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function SkillListLoadingState() {
  return (
    <PanelSurface className="max-w-[280px]">
      <aside className="flex w-[280px] flex-col border-r border-[color:var(--color-divider)] bg-[color:var(--color-surface-panel)] p-3">
        <div className="space-y-3">
          <Skeleton className="h-9 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-xl" />
          <Skeleton className="h-12 w-full rounded-xl" />
          <Skeleton className="h-12 w-full rounded-xl" />
        </div>
      </aside>
    </PanelSurface>
  );
}

function SkillListPanelFromPage() {
  const page = useSkillsPage();

  if (page.isLoading) {
    return <SkillListLoadingState />;
  }

  return (
    <PanelSurface className="max-w-[280px]">
      <SkillListPanel
        onSearchChange={page.setSearchQuery}
        onSelectSkill={page.setSelectedSkillName}
        searchQuery={page.searchQuery}
        selectedSkillName={page.effectiveSelectedName}
        skills={page.skills}
      />
    </PanelSurface>
  );
}

export const Default: Story = {
  render: () => <SkillListPanelFromPage />,
};

export const Loading: Story = {
  parameters: {
    ...storybookMswParameters({
      skill: [
        http.get("/api/skills", async () => {
          await delay("infinite");
          return HttpResponse.json({ skills: [] });
        }),
      ],
    }),
  },
  render: () => <SkillListPanelFromPage />,
};
