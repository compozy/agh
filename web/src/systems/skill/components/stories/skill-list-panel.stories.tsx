import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, HttpResponse } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";
import { expect, userEvent, within } from "storybook/test";

import { ListingToolbar } from "@agh/ui";
import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { storybookMswParameters } from "@/storybook/msw";
import { PanelSurface } from "@/storybook/story-layout";

import { SkillListFilters } from "../skill-list-filters";
import { SkillListPanel } from "../skill-list-panel";

const meta: Meta<typeof SkillListPanel> = {
  title: "systems/skill/components/SkillListPanel",
  component: SkillListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function SkillListPanelFromPage(props: { errorMessage?: string | null; isLoading?: boolean }) {
  const page = useSkillsPage();
  return (
    <PanelSurface className="max-w-3xl">
      <div className="flex flex-col gap-4 p-4">
        <ListingToolbar>
          <ListingToolbar.Leading>
            <ListingToolbar.Search
              data-testid="skill-search-input"
              onChange={page.setSearchQuery}
              placeholder="Search skills"
              value={page.searchQuery}
            />
            <ListingToolbar.Filters>
              <SkillListFilters
                enabledFilter={page.enabledFilter}
                onEnabledFilterChange={page.setEnabledFilter}
                onSourceFilterChange={page.setSourceFilter}
                sourceFilter={page.sourceFilter}
              />
            </ListingToolbar.Filters>
          </ListingToolbar.Leading>
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
          </ListingToolbar.Trailing>
        </ListingToolbar>
        <SkillListPanel
          enabledFilter={page.enabledFilter}
          errorMessage={props.errorMessage ?? (page.error ? page.error.message : null)}
          isActionPending={page.isActionPending}
          isLoading={props.isLoading ?? page.isLoading}
          onClearFilters={page.clearFilters}
          onDisable={page.handleDisable}
          onEnable={page.handleEnable}
          searchQuery={page.searchQuery}
          sourceFilter={page.sourceFilter}
          skills={page.skills}
          view={page.view}
        />
      </div>
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
        aghApiMock.get("/api/skills", async () => {
          await delay("infinite");
          return HttpResponse.json({ skills: [] });
        }),
      ],
    }),
  },
  render: () => <SkillListPanelFromPage />,
};

export const ErrorState: Story = {
  parameters: {
    ...storybookMswParameters({
      skill: [
        aghApiMock.get("/api/skills", () =>
          HttpResponse.json({ error: "skills registry offline" }, { status: 500 })
        ),
      ],
    }),
  },
  render: () => <SkillListPanelFromPage errorMessage="Skills registry offline" />,
};

export const Empty: Story = {
  parameters: {
    ...storybookMswParameters({
      skill: [aghApiMock.get("/api/skills", () => HttpResponse.json({ skills: [] }))],
    }),
  },
  render: () => <SkillListPanelFromPage />,
};

/**
 * Typing in the filter narrows the list. Tagged as play-fn so the story is
 * clearly marked as interaction-focused.
 */
export const SearchFilter: Story = {
  tags: ["play-fn"],
  render: () => <SkillListPanelFromPage />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByTestId("skill-search-input");
    await userEvent.type(input, "merchant-dispute");
    await expect(canvas.findByTestId("skill-item-merchant-dispute-triage")).resolves.toBeDefined();
  },
};
