import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { agentFixtures } from "@/systems/agent/mocks";
import type { ModelOption, ReasoningOption } from "@/systems/model-catalog";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

import { SessionCreateDialog } from "../session-create-dialog";

const workspace = workspaceDetailFixture.workspace;
const providers = workspaceDetailFixture.providers ?? [];
const selectedProvider = providers.find(provider => provider.name === "codex") ?? providers[0];

const modelOptions: ModelOption[] = [
  {
    id: "gpt-5.4",
    displayName: "GPT-5.4",
    availabilityState: "available",
    available: true,
    stale: false,
    refreshedAt: "2026-04-17T18:10:00Z",
    source: "catalog",
  },
  {
    id: "gpt-5.4-mini",
    displayName: "GPT-5.4 Mini",
    availabilityState: "available",
    available: true,
    stale: false,
    refreshedAt: "2026-04-17T18:10:00Z",
    source: "catalog",
  },
];

const reasoningOptions: ReasoningOption[] = [
  { value: "medium", label: "Medium", source: "catalog" },
  { value: "high", label: "High", source: "catalog" },
  { value: "xhigh", label: "Extra high", source: "catalog" },
];

const baseArgs = {
  open: true,
  onOpenChange: fn(),
  agents: agentFixtures,
  workspace,
  selectedAgentName: agentFixtures[0]?.name ?? "",
  selectedProvider: selectedProvider?.name ?? "",
  selectedProviderOption: selectedProvider,
  selectedModel: "gpt-5.4",
  selectedReasoning: "high",
  modelOptions,
  reasoningOptions,
  reasoningSupported: true,
  catalogStale: false,
  catalogLoading: false,
  catalogError: null,
  catalogRefreshing: false,
  catalogRefreshError: null,
  defaultReasoning: "medium",
  providerOptions: providers,
  providersLoading: false,
  providersError: null,
  onAgentChange: fn(),
  onProviderChange: fn(),
  onModelChange: fn(),
  onReasoningChange: fn(),
  onCatalogRefresh: fn(),
  onSubmit: fn(),
  isSubmitting: false,
  submitError: null,
};

const meta: Meta<typeof SessionCreateDialog> = {
  title: "systems/session/SessionCreateDialog",
  component: SessionCreateDialog,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Session creation dialog with agent, provider, model, and reasoning selectors.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Fully configured dialog ready to start a session.
 */
export const Default: Story = {
  args: baseArgs,
};

/**
 * Catalog stale state keeps the refresh affordance visible.
 */
export const CatalogStale: Story = {
  args: {
    ...baseArgs,
    catalogStale: true,
    catalogError: "Model catalog is older than the current provider config.",
  },
};

/**
 * Submit error stays inline without closing the dialog.
 */
export const SubmitError: Story = {
  args: {
    ...baseArgs,
    submitError: "Provider codex rejected the selected reasoning effort.",
  },
};
