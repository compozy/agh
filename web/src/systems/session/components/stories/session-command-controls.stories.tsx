import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import type { AgentEventPayload, RuntimeActivityPayload } from "@/systems/session/types";
import type { ModelOption, ReasoningOption } from "@/systems/model-catalog";
import { PanelSurface } from "@/storybook/story-layout";

import { ModelCommandSelect } from "../model-command-select";
import { REASONING_EFFORTS, ReasoningCommandSelect } from "../reasoning-command-select";
import { RuntimeActivityNotice, SessionActivityInline } from "../runtime-activity-notice";

const modelOptions: ModelOption[] = [
  {
    id: "gpt-5.4",
    displayName: "GPT-5.4",
    availabilityState: "available_live",
    available: true,
    stale: false,
    refreshedAt: "2026-04-17T18:10:00Z",
    source: "catalog",
  },
  {
    id: "gpt-5.4-mini",
    displayName: "GPT-5.4 Mini",
    availabilityState: "available_stale",
    available: true,
    stale: true,
    refreshedAt: "2026-04-16T18:10:00Z",
    source: "catalog",
  },
];

const reasoningOptions: ReasoningOption[] = REASONING_EFFORTS.filter(effort =>
  ["low", "medium", "high", "xhigh"].includes(effort)
).map(effort => ({
  value: effort,
  label: effort === "xhigh" ? "Extra high" : effort[0]!.toUpperCase() + effort.slice(1),
  source: "catalog",
}));

const runtimeActivity: RuntimeActivityPayload = {
  current_tool: "read_file",
  elapsed_ms: 184_000,
  elapsed_seconds: 184,
  idle_seconds: 12,
  last_activity_detail: "Reading task orchestration plan",
  last_activity_kind: "tool_call",
};

const progressEvent: AgentEventPayload = {
  type: "runtime_progress",
  text: "Still working",
  runtime: runtimeActivity,
  timestamp: "2026-04-17T18:12:00Z",
};

const warningEvent: AgentEventPayload = {
  ...progressEvent,
  type: "runtime_warning",
  text: "Runtime is waiting on provider output",
  runtime: { ...runtimeActivity, current_tool: undefined, idle_seconds: 74 },
};

const meta: Meta<typeof ModelCommandSelect> = {
  title: "systems/session/SessionCommandControls",
  component: ModelCommandSelect,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Model and reasoning command selectors plus runtime activity notices used by session creation and active session headers.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[420px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * ModelCommandSelect presents catalog-backed model choices and availability
 * badges while preserving the provider default action.
 */
export const ModelSelect: Story = {
  args: {},
  render: () => (
    <div className="grid max-w-md gap-2">
      <ModelCommandSelect
        options={modelOptions}
        defaultModel="gpt-5.4"
        value="gpt-5.4-mini"
        onChange={fn()}
        triggerTestId="story-model-select"
      />
    </div>
  ),
};

/**
 * ReasoningCommandSelect renders provider-backed effort levels and disabled
 * provider-default text.
 */
export const ReasoningSelect: StoryObj<typeof ReasoningCommandSelect> = {
  args: {},
  render: () => (
    <div className="grid max-w-md gap-2">
      <ReasoningCommandSelect
        options={reasoningOptions}
        value="high"
        onChange={fn()}
        triggerTestId="story-reasoning-select"
      />
      <ReasoningCommandSelect
        options={[]}
        value=""
        onChange={fn()}
        disabled
        disabledHint="Selected model does not advertise reasoning effort"
      />
    </div>
  ),
};

/**
 * RuntimeActivityNotice and SessionActivityInline expose progress and warning
 * state without opening a live session.
 */
export const RuntimeActivity: StoryObj<typeof RuntimeActivityNotice> = {
  args: {},
  render: () => (
    <div className="grid gap-4">
      <RuntimeActivityNotice event={progressEvent} />
      <RuntimeActivityNotice event={warningEvent} />
      <SessionActivityInline activity={runtimeActivity} />
    </div>
  ),
};
