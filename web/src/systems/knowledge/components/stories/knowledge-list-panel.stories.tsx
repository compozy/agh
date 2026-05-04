import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { storyAgentNames } from "@/storybook/fintech-scenario";
import { PanelSurface } from "@/storybook/story-layout";
import { knowledgeMemoryKey } from "@/systems/knowledge";
import type { KnowledgeMemoryItem } from "@/systems/knowledge/types";

import { KnowledgeListPanel } from "@/systems/knowledge/components/knowledge-list-panel";

const meta: Meta<typeof KnowledgeListPanel> = {
  title: "systems/knowledge/KnowledgeListPanel",
  component: KnowledgeListPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const defaultMemories: KnowledgeMemoryItem[] = [
  {
    filename: "operator-style.md",
    key: "global:operator-style.md",
    mod_time: "2026-04-17T17:30:00Z",
    name: "Operator Style",
    scope: "global",
    type: "user",
    description: "Northstar guidance for concise, accountable operator communication.",
  },
  {
    filename: "launch-week-brief.md",
    key: "global:launch-week-brief.md",
    mod_time: "2026-04-17T09:00:00Z",
    name: "Launch Week Brief",
    scope: "global",
    type: "project",
    description: "Shared context for launch KPIs, cutover timing, and cross-functional owners.",
  },
  {
    filename: "executive-risk-memo.md",
    key: "workspace:executive-risk-memo.md",
    mod_time: "2026-04-17T16:10:00Z",
    name: "Executive Risk Memo",
    scope: "workspace",
    type: "reference",
    description:
      "Workspace-local memo with launch blockers, fallback paths, and decision thresholds.",
    agent_name: storyAgentNames.cto,
  },
  {
    filename: "support-macro-pack.md",
    key: "workspace:support-macro-pack.md",
    mod_time: "2026-04-17T14:45:00Z",
    name: "Support Macro Pack",
    scope: "workspace",
    type: "reference",
    description:
      "Approved language for pricing questions, launch delays, and high-touch merchant callbacks.",
  },
];

export const Default: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={defaultMemories[0] ? knowledgeMemoryKey(defaultMemories[0]) : null}
      />
    </PanelSurface>
  ),
};

export const Empty: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={[]}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={null}
      />
    </PanelSurface>
  ),
};

export const FilteredEmpty: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={[]}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery="zzzzzz"
        selectedMemoryKey={null}
      />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        isLoading
        memories={[]}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={null}
      />
    </PanelSurface>
  ),
};

export const Error: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        errorMessage="Network failure while loading memories"
        memories={[]}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={null}
      />
    </PanelSurface>
  ),
};

export const ScopeGlobalOnly: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories.filter(memory => memory.scope === "global")}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={defaultMemories[0] ? knowledgeMemoryKey(defaultMemories[0]) : null}
      />
    </PanelSurface>
  ),
};

export const ScopeWorkspaceOnly: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories.filter(memory => memory.scope === "workspace")}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={null}
      />
    </PanelSurface>
  ),
};

export const SearchFilter: Story = {
  tags: ["play-fn"],
  render: () => {
    let searchQuery = "";
    return (
      <PanelSurface className="max-w-[340px]">
        <KnowledgeListPanel
          memories={defaultMemories}
          onSearchChange={next => {
            searchQuery = next;
          }}
          onSelectMemory={() => undefined}
          searchQuery={searchQuery}
          selectedMemoryKey={null}
        />
      </PanelSurface>
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByTestId("knowledge-search-input");
    await userEvent.type(input, "launch");
    await expect(input).toHaveValue("launch");
  },
};

export const RowSelect: Story = {
  tags: ["play-fn"],
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedMemoryKey={defaultMemories[0] ? knowledgeMemoryKey(defaultMemories[0]) : null}
      />
    </PanelSurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const row = await canvas.findByTestId(`memory-item-${knowledgeMemoryKey(defaultMemories[2])}`);
    await userEvent.click(row);
    await expect(row).toBeVisible();
  },
};
