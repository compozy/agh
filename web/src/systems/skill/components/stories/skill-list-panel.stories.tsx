import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { expect, userEvent, within } from "storybook/test";

import { useSkillsPage } from "@/hooks/routes/use-skills-page";
import { storybookMswParameters } from "@/storybook/msw";
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

function SkillListPanelFromPage(props: { errorMessage?: string | null; isLoading?: boolean }) {
  const page = useSkillsPage();
  return (
    <PanelSurface className="max-w-[340px]">
      <SkillListPanel
        errorMessage={props.errorMessage ?? (page.error ? page.error.message : null)}
        isLoading={props.isLoading ?? page.isLoading}
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

export const ErrorState: Story = {
  parameters: {
    ...storybookMswParameters({
      skill: [
        http.get("/api/skills", () =>
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
      skill: [http.get("/api/skills", () => HttpResponse.json({ skills: [] }))],
    }),
  },
  render: () => <SkillListPanelFromPage />,
};

/**
 * Typing in the filter narrows the list. Tagged as play-fn so it is excluded
 * from the visual snapshot suite and used only as an interaction test.
 */
export const SearchFilter: Story = {
  tags: ["play-fn"],
  render: () => <SkillListPanelFromPage />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByTestId("skill-search-input");
    await userEvent.type(input, "no-workarounds");
    await expect(canvas.findByTestId("skill-item-no-workarounds")).resolves.toBeDefined();
  },
};
