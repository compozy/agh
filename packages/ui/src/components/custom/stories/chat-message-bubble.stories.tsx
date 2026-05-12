import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, within } from "storybook/test";

import { ChatMessageBubble, type ChatMessageRole } from "../chat-message-bubble";
import { Pill } from "../pill";
import { ToolCallCard, type ToolCallStatus } from "../tool-call-card";

const meta: Meta<typeof ChatMessageBubble> = {
  title: "components/custom/ChatMessageBubble",
  component: ChatMessageBubble,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Presentational chat message shell per DESIGN.md §4. Role drives layout: `user` right-aligns with a surface-elevated bubble, `agent` left-aligns without a bubble, `system` renders a full-width hairline row, and `tool`/`diff` are pass-through blocks for composed inline cards.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

const ROLES: ChatMessageRole[] = ["user", "agent", "system", "tool", "diff"];

const ROLE_ALIGN: Record<ChatMessageRole, "left" | "right"> = {
  user: "right",
  agent: "left",
  system: "left",
  tool: "left",
  diff: "left",
};

function AgentMeta() {
  return (
    <>
      <Pill.Dot tone="accent" size="sm" />
      <span>claude</span>
      <span className="text-subtle">· 12:03</span>
    </>
  );
}

export const UserRole: Story = {
  args: {
    role: "user",
    meta: "YOU · 12:02",
    children:
      "Find the event mapper that groups tool calls by turn and extract the grouping logic into a pure helper.",
  },
};

export const AgentRole: Story = {
  args: {
    role: "agent",
    meta: <AgentMeta />,
    children:
      "I can see two candidates, `stream.ts` and `map.ts`. I'll extract the grouping into `groupToolCallsByTurn` and point the call site at it.",
  },
};

export const SystemRole: Story = {
  args: {
    role: "system",
    children: "Session resumed from checkpoint 8471 · 3 prior tool calls replayed",
  },
};

export const ToolRole: Story = {
  args: {
    role: "tool",
    children: (
      <ToolCallCard toolName="shell.safe-run" filePath="packages/runtime" status="completed">
        <pre className="font-mono text-[12px] leading-[1.6] text-muted">
          $ rg &quot;onToolCall&quot; packages/runtime -l
        </pre>
      </ToolCallCard>
    ),
  },
};

export const DiffRole: Story = {
  args: {
    role: "diff",
    children: (
      <div className="rounded-md border border-line bg-rail p-4 font-mono text-[12px] leading-[1.65]">
        <div className="text-success">+ const groups = groupToolCallsByTurn(tool.events);</div>
        <div className="text-danger">- for (const ev of tool.events) {"{"}</div>
      </div>
    ),
  },
};

export const AllRoles: Story = {
  render: () => (
    <div
      className="flex flex-col gap-4"
      data-testid="all-roles"
      style={{ maxWidth: 820, margin: "0 auto" }}
    >
      <ChatMessageBubble role="system" data-role-key="system">
        Session resumed · 3 prior tool calls replayed
      </ChatMessageBubble>
      <ChatMessageBubble role="user" meta="YOU · 12:02" data-role-key="user">
        Find the event mapper that groups tool calls by turn.
      </ChatMessageBubble>
      <ChatMessageBubble role="agent" meta={<AgentMeta />} data-role-key="agent">
        Two candidates, I&apos;ll extract the grouping into `groupToolCallsByTurn`.
      </ChatMessageBubble>
      <ChatMessageBubble role="tool" data-role-key="tool">
        <ToolCallCard toolName="shell.safe-run" filePath="packages/runtime" status="completed" />
      </ChatMessageBubble>
      <ChatMessageBubble role="diff" data-role-key="diff">
        <div className="rounded-md border border-line bg-rail p-3 font-mono text-[12px]">
          + apply diff to stream.ts
        </div>
      </ChatMessageBubble>
    </div>
  ),
};

export const RoleAlignmentInteraction: Story = {
  render: () => (
    <div
      className="flex flex-col gap-3"
      data-testid="role-alignment"
      style={{ maxWidth: 820, margin: "0 auto" }}
    >
      {ROLES.map(role => (
        <ChatMessageBubble
          key={role}
          role={role}
          meta={role === "user" ? "YOU · 12:02" : role === "agent" ? <AgentMeta /> : undefined}
          data-role-key={role}
        >
          {role === "tool" ? (
            <ToolCallCard toolName="shell.run" status="in_progress" />
          ) : (
            `message for role ${role}`
          )}
        </ChatMessageBubble>
      ))}
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const wrapper = await canvas.findByTestId("role-alignment");
    for (const role of ROLES) {
      const node = wrapper.querySelector<HTMLElement>(`[data-role-key="${role}"]`);
      await expect(node).not.toBeNull();
      await expect(node?.getAttribute("data-role")).toBe(role);
      await expect(node?.getAttribute("data-align")).toBe(ROLE_ALIGN[role]);
      if (role === "user") {
        const body = node?.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
        await expect(body?.className).toContain("bg-elevated");
        await expect(node?.className).toContain("justify-end");
      }
      if (role === "agent") {
        const body = node?.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
        await expect(body?.className).not.toContain("bg-elevated");
        await expect(body?.className).toContain("text-muted");
      }
      if (role === "system") {
        const dividers = node?.querySelectorAll('span[aria-hidden="true"]') ?? [];
        await expect(dividers.length).toBe(2);
      }
    }
  },
};

export const StatusBadgeCycleInteraction: Story = {
  render: () => (
    <div
      className="flex flex-col gap-3"
      data-testid="tool-statuses"
      style={{ maxWidth: 820, margin: "0 auto" }}
    >
      {(["pending", "in_progress", "completed", "failed"] as ToolCallStatus[]).map(status => (
        <ChatMessageBubble key={status} role="tool">
          <ToolCallCard
            toolName="file.read"
            filePath="packages/runtime/src/session/stream.ts"
            status={status}
            data-status-key={status}
          />
        </ChatMessageBubble>
      ))}
    </div>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const wrapper = await canvas.findByTestId("tool-statuses");
    const expected: Record<ToolCallStatus, { tone: string; label: string }> = {
      pending: { tone: "neutral", label: "Pending" },
      in_progress: { tone: "info", label: "Running" },
      completed: { tone: "success", label: "Done" },
      failed: { tone: "danger", label: "Error" },
    };
    for (const status of ["pending", "in_progress", "completed", "failed"] as ToolCallStatus[]) {
      const card = wrapper.querySelector<HTMLElement>(`[data-status-key="${status}"]`);
      await expect(card).not.toBeNull();
      const badge = card?.querySelector<HTMLElement>('[data-slot="tool-call-card-status"]');
      await expect(badge?.textContent).toBe(expected[status].label);
      await expect(badge?.getAttribute("data-tone")).toBe(expected[status].tone);
    }
  },
};
