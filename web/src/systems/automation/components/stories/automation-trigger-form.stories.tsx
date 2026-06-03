import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  storyAgentNames,
  storyWorkspaceIds,
  storyWorkspaceNames,
} from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";
import { createAutomationTriggerDraft } from "@/systems/automation";
import type { CreateAutomationTriggerRequest } from "@/systems/automation";

import { AutomationTriggerForm } from "../automation-trigger-form";

const meta: Meta<typeof AutomationTriggerForm> = {
  title: "systems/automation/AutomationTriggerForm",
  component: AutomationTriggerForm,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const storyWorkspaces = [
  { id: storyWorkspaceIds.hq, name: storyWorkspaceNames.hq },
  { id: storyWorkspaceIds.risk, name: storyWorkspaceNames.risk },
  { id: storyWorkspaceIds.growth, name: storyWorkspaceNames.growth },
];

const storyAgents = [
  storyAgentNames.compliance,
  storyAgentNames.support,
  storyAgentNames.release,
  storyAgentNames.fraud,
];

const STOP_PROMPT =
  "Session {{ .Data.session_id }} (agent {{ .Data.agent_name }}) stopped with reason " +
  '"{{ .Data.stop_reason }}".\n\nRead the transcript, summarize what went wrong, the likely ' +
  "root cause, and one suggested next step.";

function baseDraft(
  overrides: Partial<CreateAutomationTriggerRequest> = {}
): CreateAutomationTriggerRequest {
  return {
    ...createAutomationTriggerDraft(storyWorkspaceIds.hq),
    event: "session.stopped",
    scope: "workspace",
    workspace_id: storyWorkspaceIds.hq,
    name: "summarize-failures",
    agent_name: storyAgentNames.compliance,
    filter: { "data.stop_reason": "error" },
    prompt: STOP_PROMPT,
    ...overrides,
  };
}

function TriggerFormHarness({
  initialDraft,
  mode = "create",
  isPending = false,
}: {
  initialDraft: CreateAutomationTriggerRequest;
  mode?: "create" | "edit";
  isPending?: boolean;
}) {
  const [draft, setDraft] = useState(initialDraft);

  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="grid h-(--height-modal-xl) max-h-[90vh] w-full max-w-(--width-modal-xl) grid-rows-[minmax(0,1fr)] overflow-hidden rounded-xl border border-line bg-canvas-soft">
        <AutomationTriggerForm
          activeWorkspaceId={storyWorkspaceIds.hq}
          agents={storyAgents}
          draft={draft}
          isPending={isPending}
          mode={mode}
          onCancel={() => undefined}
          onChange={setDraft}
          onSubmit={() => undefined}
          workspaces={storyWorkspaces}
        />
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  args: {},
  render: () => <TriggerFormHarness initialDraft={baseDraft()} />,
};

export const WebhookSelected: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={baseDraft({
        name: "deploy-watch",
        event: "webhook",
        scope: "global",
        workspace_id: undefined,
        filter: {},
        endpoint_slug: "ci-deploys",
        webhook_id: "wbh_a1b2c3",
        webhook_secret_value: "whsec_live_8f3a9c2e",
        agent_name: storyAgentNames.release,
        prompt: "A deploy webhook arrived: {{ .Data.payload }}. Post a launch-room status update.",
      })}
    />
  ),
};

export const ExtensionEvent: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={baseDraft({
        name: "release-deploys",
        event: "ext.release.deploy-started",
        filter: { "data.status": "started" },
        agent_name: storyAgentNames.release,
        prompt: "Release {{ .Data.version }} started deploying to {{ .Data.repo }}.",
      })}
    />
  ),
};

export const NoFilters: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={baseDraft({
        name: "session-watch",
        event: "session.created",
        filter: {},
        prompt: "Session {{ .Data.session_id }} started for agent {{ .Data.agent_name }}.",
      })}
    />
  ),
};

export const ReliabilityExpanded: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={baseDraft({
        retry: { strategy: "backoff", max_retries: 2, base_delay: "5s" },
        fire_limit: { max: 4, window: "1h" },
        enabled: false,
      })}
    />
  ),
};

export const EditMode: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={baseDraft({
        name: "hook-failures",
        event: "hook.transform.completed",
        filter: { "data.hook_outcome": "failed" },
        agent_name: storyAgentNames.support,
        prompt:
          "Hook {{ .Data.hook_name }} failed: {{ .Data.error }}. Investigate and report back.",
      })}
      mode="edit"
    />
  ),
};

export const ValidationDisabled: Story = {
  args: {},
  render: () => (
    <TriggerFormHarness
      initialDraft={{
        ...createAutomationTriggerDraft(null),
        event: "session.stopped",
        scope: "workspace",
        workspace_id: undefined,
        name: "",
        agent_name: "",
        prompt: "",
        filter: {},
      }}
    />
  ),
};
