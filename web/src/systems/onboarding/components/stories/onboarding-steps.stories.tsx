import type { Meta, StoryObj } from "@storybook/react-vite";

import type { ProviderSummary } from "@/systems/providers";

import type { OnboardingDefaultModelApi } from "../../hooks/use-onboarding-default-model";
import type { OnboardingWorkspacesApi } from "../../hooks/use-onboarding-workspaces";
import { StepDefaultModel } from "../step-default-model";
import { StepWorkspaces } from "../step-workspaces";

function provider(name: string, displayName: string): ProviderSummary {
  return {
    name,
    display_name: displayName,
    default: false,
    auth_status: {
      env_policy: "filtered",
      home_policy: "operator",
      mode: "native_cli",
      state: "ready",
    },
  };
}

const noop = () => {};

const baseModel: OnboardingDefaultModelApi = {
  providers: [
    provider("claude", "Claude Code"),
    provider("codex", "Codex"),
    provider("gemini", "Gemini CLI"),
    provider("openclaw", "OpenClaw"),
  ],
  providersLoading: false,
  providersError: null,
  provider: "claude",
  model: "claude-opus-4-7",
  reasoning: "high",
  authMode: "native_cli",
  envVar: "",
  apiKey: "",
  modelOptions: [
    { id: "claude-opus-4-7", label: "Claude Opus 4.7" },
    { id: "claude-sonnet-4-6", label: "Claude Sonnet 4.6" },
  ],
  reasoningOptions: [
    { value: "low", label: "Low", source: "catalog" },
    { value: "medium", label: "Medium", source: "catalog" },
    { value: "high", label: "High", source: "catalog" },
    { value: "xhigh", label: "Extra high · deepest", source: "catalog" },
  ],
  reasoningSupported: true,
  defaultReasoning: "medium",
  catalogLoading: false,
  catalogError: null,
  isValid: true,
  isCommitting: false,
  onProviderChange: noop,
  onModelChange: noop,
  onReasoningChange: noop,
  onAuthModeChange: noop,
  onEnvVarChange: noop,
  onApiKeyChange: noop,
  commit: async () => {},
};

const baseWorkspaces: OnboardingWorkspacesApi = {
  currentPath: "/Users/operator/Dev",
  parent: "/Users/operator",
  home: "/Users/operator",
  entries: [
    { name: "compozy", path: "/Users/operator/Dev/compozy", is_dir: true },
    { name: "infra", path: "/Users/operator/Dev/infra", is_dir: true },
    { name: "notes", path: "/Users/operator/Dev/notes", is_dir: true },
    { name: "README.md", path: "/Users/operator/Dev/README.md", is_dir: false },
  ],
  isBrowsing: false,
  browseError: null,
  workspaces: [{ path: "/Users/operator/Dev/compozy", name: "compozy" }],
  isResolving: false,
  resolveError: null,
  navigateTo: noop,
  goToParent: noop,
  goHome: noop,
  addWorkspace: async () => {},
  removeWorkspace: noop,
  isAdded: (path: string) => path === "/Users/operator/Dev/compozy",
};

const meta: Meta = {
  title: "systems/onboarding/Steps",
  parameters: { layout: "fullscreen" },
  decorators: [
    Story => (
      <div className="min-h-dvh bg-canvas px-8 py-7">
        <div className="mx-auto max-w-2xl">
          <Story />
        </div>
      </div>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

export const DefaultModelNativeCli: Story = {
  render: () => <StepDefaultModel model={baseModel} />,
};

export const DefaultModelApiKey: Story = {
  render: () => (
    <StepDefaultModel
      model={{ ...baseModel, authMode: "bound_secret", envVar: "ANTHROPIC_API_KEY" }}
    />
  ),
};

export const Workspaces: Story = {
  render: () => <StepWorkspaces workspaces={baseWorkspaces} />,
};

export const WorkspacesEmpty: Story = {
  render: () => <StepWorkspaces workspaces={{ ...baseWorkspaces, workspaces: [] }} />,
};
