import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames } from "@/storybook/fintech-scenario";
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
  filename: "global/operator-style.md",
  mod_time: "2026-04-17T17:30:00Z",
  name: "Operator Style",
  type: "user",
  description: "Northstar guidance for concise, accountable operator communication.",
};

const workspaceMemory: MemoryHeader = {
  filename: "workspace/executive-risk-memo.md",
  mod_time: "2026-04-17T16:10:00Z",
  name: "Executive Risk Memo",
  type: "reference",
  description: "Workspace-local memo for launch blockers, fallback paths, and decision thresholds.",
  agent_name: storyAgentNames.cto,
};

const defaultContent = [
  "# Operator Style",
  "",
  "Own the launch outcome end to end.",
  "",
  "- Lead with the current fact pattern and the next owner.",
  "- Escalate ambiguity with the next concrete evidence request.",
  "- Keep customer-facing language calm, specific, and launch-safe.",
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
        onDelete={async () => {}}
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
        onDelete={async () => {}}
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
        onDelete={async () => {}}
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
        onDelete={async () => {}}
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
        onDelete={async () => {}}
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
        onDelete={async () => {}}
      />
    </PanelSurface>
  ),
};
