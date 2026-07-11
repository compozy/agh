import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, fn, within } from "storybook/test";

import {
  AgentCreateDialog,
  createDefaultAgentCreateDraft,
  type AgentCreateDialogDraft,
  type AgentCreateStep,
} from "@/systems/agent";
import type { RuntimeModelOption, RuntimeProviderOption } from "@/systems/runtime";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

const providerOptions: RuntimeProviderOption[] = (workspaceDetailFixture.providers ?? []).map(
  provider => ({
    id: provider.name,
    name: provider.display_name?.trim() || provider.name,
    ...(provider.harness?.trim() ? { harness: provider.harness.trim() } : {}),
    runtime_provider: provider.runtime_provider?.trim() || provider.name,
  })
);

const modelProvider = providerOptions[0]?.id ?? "codex";

const runtimeModels: RuntimeModelOption[] = [
  {
    id: "gpt-5.6-sol",
    provider: modelProvider,
    name: "GPT-5.6 Sol",
    efforts: ["low", "medium", "high", "xhigh", "max"],
    availability: "live",
    curated: true,
    featured: true,
    context_window: 1_050_000,
    cost_input: 5,
    cost_output: 30,
    supports_tools: true,
    reasoning_source: "acp",
  },
  {
    id: "gpt-5.6-luna",
    provider: modelProvider,
    name: "GPT-5.6 Luna",
    efforts: ["low", "high"],
    availability: "live",
    curated: true,
    context_window: 1_050_000,
    cost_input: 1,
    cost_output: 6,
    supports_tools: true,
  },
];

const validDraft: AgentCreateDialogDraft = {
  ...createDefaultAgentCreateDraft(true),
  name: "release-captain",
  provider: modelProvider,
  model: "gpt-5.6-sol",
  reasoningEffort: "high",
  prompt: "Own release readiness, canary evidence, and rollback guardrails.",
  permissions: "approve-reads",
  tools: ["agh__skill_view"],
  toolsets: ["agh__catalog"],
  denyTools: ["agh__task_*"],
  disabledSkills: ["draft-blog-post"],
};

const meta: Meta<typeof AgentCreateDialog> = {
  title: "systems/agent/components/AgentCreateDialog",
  component: AgentCreateDialog,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function AgentCreateDialogHarness({
  initialDraft,
  initialStep,
  isSubmitting = false,
  submitError = null,
}: {
  initialDraft?: AgentCreateDialogDraft;
  initialStep?: AgentCreateStep;
  isSubmitting?: boolean;
  submitError?: string | null;
}) {
  const [draft, setDraft] = useState(initialDraft ?? createDefaultAgentCreateDraft(true));

  return (
    <AgentCreateDialog
      draft={draft}
      hasActiveWorkspace
      initialStep={initialStep}
      isSubmitting={isSubmitting}
      modelCatalogError={null}
      modelCatalogLoading={false}
      modelCatalogLoaded={true}
      modelCatalogRefreshing={false}
      onDraftChange={setDraft}
      onOpenChange={() => undefined}
      onOpenProviderSettings={fn()}
      onRefreshCatalog={fn()}
      onSubmit={fn()}
      open
      providerOptions={providerOptions}
      providersError={null}
      providersLoading={false}
      runtimeModels={runtimeModels}
      submitError={submitError}
      workspaceName={workspaceDetailFixture.workspace.name}
    />
  );
}

export const Default: Story = {
  render: () => <AgentCreateDialogHarness />,
};

export const ValidationError: Story = {
  render: () => (
    <AgentCreateDialogHarness
      initialDraft={{
        ...createDefaultAgentCreateDraft(true),
        name: "../release",
        categoryPath: "Engineering//Release",
      }}
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("agent-create-name-error")).toHaveTextContent(
      "Agent names cannot be . or .."
    );
    await expect(canvas.getByTestId("agent-create-category-path-error")).toHaveTextContent(
      "Category path cannot contain blank segments."
    );
  },
};

export const Submitting: Story = {
  render: () => (
    <AgentCreateDialogHarness initialDraft={validDraft} initialStep="access" isSubmitting />
  ),
};

export const DuplicateError: Story = {
  render: () => (
    <AgentCreateDialogHarness
      initialDraft={validDraft}
      initialStep="access"
      submitError="agent definition already exists"
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await expect(canvas.getByTestId("agent-create-submit-error")).toHaveTextContent(
      "agent definition already exists"
    );
  },
};
