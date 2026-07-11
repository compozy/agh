import type { Meta, StoryObj } from "@storybook/react-vite";

import type { OnboardingDefaultModelApi } from "../../hooks/use-onboarding-default-model";
import type { OnboardingWorkspacesApi } from "../../hooks/use-onboarding-workspaces";
import { StepDefaultModel } from "../step-default-model";
import { StepWorkspaces } from "../step-workspaces";

const noop = () => {};

const baseModel: OnboardingDefaultModelApi = {
  providersLoading: false,
  providersError: null,
  runtimeValue: { provider: "claude", model: "claude-opus-4-8", reasoning_effort: "high" },
  runtimeProviders: [
    { id: "claude", name: "Claude Code", harness: "acp", runtime_provider: "claude" },
    { id: "codex", name: "Codex", harness: "acp", runtime_provider: "codex" },
    { id: "gemini", name: "Gemini CLI", runtime_provider: "gemini" },
    { id: "openclaw", name: "OpenClaw", runtime_provider: "openclaw" },
  ],
  runtimeModels: [
    {
      id: "claude-opus-4-8",
      provider: "claude",
      name: "Claude Opus 4.8",
      efforts: ["low", "medium", "high", "xhigh", "max"],
      availability: "live",
      curated: true,
      context_window: 1_000_000,
      cost_input: 5,
      cost_output: 25,
      supports_tools: true,
      reasoning_source: "catalog",
    },
    {
      id: "claude-sonnet-5",
      provider: "claude",
      name: "Claude Sonnet 5",
      efforts: ["low", "medium", "high", "xhigh", "max"],
      availability: "live",
      curated: true,
      featured: true,
      context_window: 1_000_000,
      cost_input: 3,
      cost_output: 15,
      supports_tools: true,
      reasoning_source: "catalog",
    },
  ],
  authMode: "native_cli",
  envVar: "",
  apiKey: "",
  catalogLoading: false,
  catalogLoaded: true,
  catalogRefreshing: false,
  catalogError: null,
  configurationError: null,
  isValid: true,
  isCommitting: false,
  onRuntimeChange: noop,
  onRefreshCatalog: noop,
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
  isRemoving: false,
  resolveError: null,
  navigateTo: noop,
  goToParent: noop,
  goHome: noop,
  addWorkspace: async () => {},
  removeWorkspace: async () => {},
  isAdded: (path: string) => path === "/Users/operator/Dev/compozy",
};

const meta: Meta = {
  title: "systems/onboarding/components/Steps",
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

export const DefaultModelApiKeyMissingEnv: Story = {
  render: () => (
    <StepDefaultModel
      model={{
        ...baseModel,
        authMode: "bound_secret",
        envVar: "",
        configurationError: "Enter the environment variable the provider expects.",
        isValid: false,
      }}
    />
  ),
};

export const Workspaces: Story = {
  render: () => <StepWorkspaces workspaces={baseWorkspaces} />,
};

export const WorkspacesEmpty: Story = {
  render: () => <StepWorkspaces workspaces={{ ...baseWorkspaces, workspaces: [] }} />,
};
