import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  storyAgentNames,
  storyWorkspaceIds,
  storyWorkspaceNames,
} from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";
import type { CreateAutomationJobRequest } from "@/systems/automation";

import { createAutomationJobDraft, setJobOutputMode } from "../../lib/automation-drafts";

import { AutomationJobForm } from "../automation-job-form";

const meta: Meta<typeof AutomationJobForm> = {
  title: "systems/automation/AutomationJobForm",
  component: AutomationJobForm,
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
  storyAgentNames.release,
  storyAgentNames.compliance,
  storyAgentNames.support,
  storyAgentNames.fraud,
];

const REVIEW_PROMPT =
  "Review the diffs merged in the last 24 hours. Flag risky changes, summarize what shipped, " +
  "and post a launch-room digest with one follow-up per concern.";

function baseDraft(
  overrides: Partial<CreateAutomationJobRequest> = {}
): CreateAutomationJobRequest {
  return {
    ...createAutomationJobDraft(storyWorkspaceIds.hq),
    name: "daily-code-review",
    agent_name: storyAgentNames.release,
    prompt: REVIEW_PROMPT,
    schedule: { mode: "cron", expr: "0 9 * * 1-5" },
    ...overrides,
  };
}

function JobFormHarness({
  initialDraft,
  mode = "create",
  isPending = false,
}: {
  initialDraft: CreateAutomationJobRequest;
  mode?: "create" | "edit";
  isPending?: boolean;
}) {
  const [draft, setDraft] = useState(initialDraft);

  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="grid h-(--height-modal-xl) max-h-[90vh] w-full max-w-(--width-modal-xl) grid-rows-[minmax(0,1fr)] overflow-hidden rounded-xl border border-line bg-canvas-soft">
        <AutomationJobForm
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
  render: () => <JobFormHarness initialDraft={baseDraft()} />,
};

export const TaskDelegation: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={setJobOutputMode(
        baseDraft({
          name: "nightly-reconcile",
          task: {
            title: "Reconcile settlement ledger",
            description: "Diff the settlement ledger against the gateway and open a task per gap.",
            owner: { kind: "pool", ref: "finance-ops" },
          },
        }),
        "task"
      )}
    />
  ),
};

export const EveryInterval: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={baseDraft({
        name: "fraud-sweep",
        agent_name: storyAgentNames.fraud,
        prompt: "Sweep open fraud signals and escalate anything above the launch-week threshold.",
        schedule: { mode: "every", interval: "30m" },
      })}
    />
  ),
};

export const RunOnce: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={baseDraft({
        name: "launch-kickoff",
        agent_name: storyAgentNames.compliance,
        prompt: "Post the launch-window go/no-go checklist to the war room.",
        schedule: { mode: "at", time: "2027-01-04T09:00" },
      })}
    />
  ),
};

export const ReliabilityExpanded: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={baseDraft({
        retry: { strategy: "backoff", max_retries: 4, base_delay: "5s" },
        fire_limit: { max: 6, window: "1h" },
        enabled: false,
      })}
    />
  ),
};

export const EditMode: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={baseDraft({
        name: "weekly-digest",
        agent_name: storyAgentNames.support,
        prompt: "Compile the weekly support digest and rank the top recurring escalations.",
        schedule: { mode: "cron", expr: "0 8 * * 1" },
      })}
      mode="edit"
    />
  ),
};

export const ValidationDisabled: Story = {
  args: {},
  render: () => (
    <JobFormHarness
      initialDraft={{
        ...createAutomationJobDraft(null),
        name: "",
        agent_name: "",
        prompt: "",
        schedule: { mode: "cron", expr: "0 9 * * *" },
      }}
    />
  ),
};
