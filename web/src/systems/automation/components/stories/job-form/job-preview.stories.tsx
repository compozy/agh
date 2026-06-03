import type { Meta, StoryObj } from "@storybook/react-vite";

import { storyAgentNames, storyWorkspaceIds } from "@/storybook/fintech-scenario";
import { CenteredSurface } from "@/storybook/story-layout";
import { createAutomationJobDraft } from "@/systems/automation";
import type { CreateAutomationJobRequest } from "@/systems/automation";
import { buildJobPreview } from "@/systems/automation/lib/job-preview";

import { JobPreview } from "../../job-form/preview/job-preview";

const meta: Meta<typeof JobPreview> = {
  title: "systems/automation/job-form/JobPreview",
  component: JobPreview,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/** Fixed clock so the live "next runs" list renders deterministically. */
const STORY_NOW = Date.UTC(2026, 5, 3, 8, 0, 0);

function previewFor(overrides: Partial<CreateAutomationJobRequest>) {
  const draft: CreateAutomationJobRequest = {
    ...createAutomationJobDraft(storyWorkspaceIds.hq),
    name: "payout-watchlist",
    agent_name: storyAgentNames.fraud,
    prompt: "Review payout holds above the reserve threshold and draft the operator summary.",
    schedule: { mode: "cron", expr: "0 9 * * 1-5" },
    ...overrides,
  };
  return buildJobPreview(draft, STORY_NOW);
}

function JobPreviewHarness({ preview }: { preview: ReturnType<typeof buildJobPreview> }) {
  return (
    <CenteredSurface className="items-start justify-center p-6">
      <div className="grid h-[760px] w-(--width-right-rail-default) grid-rows-[minmax(0,1fr)] overflow-hidden rounded-xl border border-line">
        <JobPreview preview={preview} />
      </div>
    </CenteredSurface>
  );
}

export const AgentMode: Story = {
  args: {},
  render: () => <JobPreviewHarness preview={previewFor({})} />,
};

export const TaskMode: Story = {
  args: {},
  render: () => (
    <JobPreviewHarness
      preview={previewFor({
        name: "weekly-burn-report",
        task: {
          title: "Prepare weekly burn report",
          description: "Pull spend deltas across launch workspaces and draft the operator summary.",
          owner: { kind: "agent_session", ref: storyAgentNames.cfo },
        },
      })}
    />
  ),
};

export const InvalidSchedule: Story = {
  args: {},
  render: () => (
    <JobPreviewHarness
      preview={previewFor({
        schedule: { mode: "cron", expr: "0 99 * *" },
      })}
    />
  ),
};
