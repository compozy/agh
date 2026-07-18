import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { agentFixtures } from "@/systems/agent/mocks";
import type { NetworkParticipationDraft } from "@/systems/network";
import type {
  RuntimeModelOption,
  RuntimeProviderOption,
  RuntimeSelectorValue,
} from "@/systems/runtime";
import { workspaceDetailFixture } from "@/systems/workspace/mocks";

import { SessionCreateDialog } from "../session-create-dialog";

const workspace = workspaceDetailFixture.workspace;

// Truthful post-migration args: the dialog now hosts one unified RuntimeSelector
// (provider · model · reasoning) fed by aggregate catalog rows, not the deleted
// provider/model/reasoning leaf selects.
const runtimeProviders: RuntimeProviderOption[] = [
  { id: "codex", name: "Codex", runtime_provider: "codex", harness: "acp" },
  { id: "claude", name: "Claude", runtime_provider: "claude", harness: "acp" },
];

const runtimeModels: RuntimeModelOption[] = [
  {
    id: "gpt-5.6-sol",
    provider: "codex",
    name: "GPT-5.6 Sol",
    context_window: 1_050_000,
    cost_input: 5,
    cost_output: 30,
    supports_tools: true,
    supports_reasoning: true,
    efforts: ["none", "low", "medium", "high", "xhigh", "max"],
    default_effort: "",
    reasoning_source: "acp",
    availability: "live",
    curated: true,
    featured: true,
  },
  {
    id: "claude-fable-5",
    provider: "claude",
    name: "Claude Fable 5",
    context_window: 1_000_000,
    cost_input: 10,
    cost_output: 50,
    supports_tools: true,
    supports_reasoning: true,
    efforts: ["low", "medium", "high", "xhigh", "max"],
    default_effort: "",
    reasoning_source: "acp",
    availability: "live",
    curated: true,
    featured: true,
  },
];

const runtimeValue = {
  provider: "codex",
  model: "gpt-5.6-sol",
  reasoning_effort: "high",
} satisfies RuntimeSelectorValue;

const localParticipation = {
  mode: "local",
  channelStrategy: "",
  channelId: "",
} satisfies NetworkParticipationDraft;
const liveParticipation = {
  mode: "live",
  channelStrategy: "named",
  channelId: "release-room",
} satisfies NetworkParticipationDraft;

const baseArgs = {
  open: true,
  onOpenChange: fn(),
  agents: agentFixtures,
  workspace,
  selectedAgentName: agentFixtures[0]?.name ?? "",
  runtimeValue,
  runtimeProviders,
  runtimeModels,
  catalogStale: false,
  catalogLoading: false,
  catalogLoaded: true,
  catalogError: null,
  catalogRefreshing: false,
  catalogRefreshError: null,
  providersLoading: false,
  providersError: null,
  hasProviderOptions: true,
  networkParticipation: localParticipation,
  onAgentChange: fn(),
  onRuntimeChange: fn(),
  onNetworkParticipationChange: fn(),
  onCatalogRefresh: fn(),
  onOpenProviderSettings: fn(),
  onSubmit: fn(),
  isSubmitting: false,
  submitError: null,
};

const meta: Meta<typeof SessionCreateDialog> = {
  title: "systems/session/components/SessionCreateDialog",
  component: SessionCreateDialog,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Session-create dialog hosting the unified RuntimeSelector (provider · model · reasoning) plus the agent picker.",
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
 * Catalog stale state keeps the refresh affordance + status line visible.
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

/**
 * Explicit Live participation exposes only the named strategy accepted by Session creation.
 */
export const LiveParticipation: Story = {
  args: {
    ...baseArgs,
    networkParticipation: liveParticipation,
  },
};

/**
 * Pending startup keeps the dialog open until ACP confirms the session.
 */
export const PendingStartup: Story = {
  args: {
    ...baseArgs,
    isSubmitting: true,
  },
};
