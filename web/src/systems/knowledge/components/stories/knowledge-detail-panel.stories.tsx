import type { Meta, StoryObj } from "@storybook/react-vite";

import { PanelSurface } from "@/storybook/story-layout";
import type { MemoryHeader } from "@/systems/knowledge/types";

import { KnowledgeDetailPanel } from "@/systems/knowledge/components/knowledge-detail-panel";

const meta: Meta<typeof KnowledgeDetailPanel> = {
  title: "systems/knowledge/KnowledgeDetailPanel",
  component: KnowledgeDetailPanel,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const defaultMemory: MemoryHeader = {
  filename: "global/user-role.md",
  mod_time: "2026-04-17T17:30:00Z",
  name: "User Role",
  type: "user",
  description: "Guidance that shapes the assistant's tone and ownership.",
};

const workspaceMemory: MemoryHeader = {
  filename: "workspace/project-context.md",
  mod_time: "2026-04-17T16:10:00Z",
  name: "Project Context",
  type: "project",
  description: "Workspace-local notes about Storybook rollout decisions.",
  agent_name: "codex-agent",
};

const defaultContent = [
  "# User Role",
  "",
  "You own the outcome end to end.",
  "",
  "- Prefer direct fixes.",
  "- Verify before handoff.",
  "- Call out ambiguity explicitly.",
].join("\n");

export const Default: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={defaultContent}
        error={null}
        isDeletePending={false}
        isLoading={false}
        memory={defaultMemory}
        onDelete={() => undefined}
        scope="global"
      />
    </PanelSurface>
  ),
};

export const WorkspaceScope: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={defaultContent}
        error={null}
        isDeletePending={false}
        isLoading={false}
        memory={workspaceMemory}
        onDelete={() => undefined}
        scope="workspace"
      />
    </PanelSurface>
  ),
};

export const NoContent: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={undefined}
        error={null}
        isDeletePending={false}
        isLoading={false}
        memory={defaultMemory}
        onDelete={() => undefined}
        scope="global"
      />
    </PanelSurface>
  ),
};

export const Loading: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={undefined}
        error={null}
        isDeletePending={false}
        isLoading
        memory={defaultMemory}
        onDelete={() => undefined}
        scope="global"
      />
    </PanelSurface>
  ),
};

export const ErrorState: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={undefined}
        error={new globalThis.Error("Content fetch failed")}
        isDeletePending={false}
        isLoading={false}
        memory={defaultMemory}
        onDelete={() => undefined}
        scope="global"
      />
    </PanelSurface>
  ),
};

export const EmptySelection: Story = {
  render: () => (
    <PanelSurface>
      <KnowledgeDetailPanel
        content={undefined}
        error={null}
        isDeletePending={false}
        isLoading={false}
        memory={undefined}
        onDelete={() => undefined}
      />
    </PanelSurface>
  ),
};
