import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import type { MemoryHeader } from "@/systems/knowledge/types";

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

const defaultMemories: MemoryHeader[] = [
  {
    filename: "global/user-role.md",
    mod_time: "2026-04-17T17:30:00Z",
    name: "User Role",
    type: "user",
    description: "Guidance that shapes the assistant's tone and ownership.",
  },
  {
    filename: "global/feedback-testing.md",
    mod_time: "2026-04-17T09:00:00Z",
    name: "Testing Feedback",
    type: "feedback",
    description: "Always keep the real database in integration tests.",
  },
  {
    filename: "workspace/project-context.md",
    mod_time: "2026-04-17T16:10:00Z",
    name: "Project Context",
    type: "project",
    description: "Workspace-local notes about Storybook rollout decisions.",
    agent_name: "codex-agent",
  },
  {
    filename: "workspace/release-checklist.md",
    mod_time: "2026-04-17T14:45:00Z",
    name: "Release Checklist",
    type: "reference",
    description: "Operational checklist for release verification.",
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
        selectedFilename={defaultMemories[0]?.filename ?? null}
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
        selectedFilename={null}
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
        selectedFilename={null}
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
        selectedFilename={null}
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
        selectedFilename={null}
      />
    </PanelSurface>
  ),
};

export const ScopeGlobalOnly: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories.filter(memory => memory.filename.startsWith("global/"))}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedFilename={defaultMemories[0]?.filename ?? null}
      />
    </PanelSurface>
  ),
};

export const ScopeWorkspaceOnly: Story = {
  render: () => (
    <PanelSurface className="max-w-[340px]">
      <KnowledgeListPanel
        memories={defaultMemories.filter(memory => memory.filename.startsWith("workspace/"))}
        onSearchChange={() => undefined}
        onSelectMemory={() => undefined}
        searchQuery=""
        selectedFilename={null}
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
          selectedFilename={null}
        />
      </PanelSurface>
    );
  },
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const input = await canvas.findByTestId("knowledge-search-input");
    await userEvent.type(input, "project");
    await expect(input).toHaveValue("project");
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
        selectedFilename={defaultMemories[0]?.filename ?? null}
      />
    </PanelSurface>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const row = await canvas.findByTestId(`memory-item-${defaultMemories[2].filename}`);
    await userEvent.click(row);
    await expect(row).toBeVisible();
  },
};
