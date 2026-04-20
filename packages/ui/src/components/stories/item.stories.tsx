import { Fragment } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { CpuIcon, DatabaseIcon, EyeIcon, MoreHorizontalIcon, ZapIcon } from "lucide-react";

import { Badge } from "../badge";
import { Button } from "../button";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemSeparator,
  ItemTitle,
} from "../item";

const meta: Meta<typeof Item> = {
  title: "ui/Item",
  component: Item,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Compact list row with media, title, description, and actions slots. Wrap rows in ItemGroup for consistent spacing.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

const agents = [
  {
    id: "claude",
    icon: CpuIcon,
    title: "Claude Code",
    description: "Primary orchestrator for long-running sessions.",
    badge: "active",
  },
  {
    id: "codex",
    icon: ZapIcon,
    title: "Codex CLI",
    description: "Fast fallback for single-shot refactors.",
    badge: "ready",
  },
  {
    id: "gemini",
    icon: DatabaseIcon,
    title: "Gemini CLI",
    description: "Bound to the knowledge retrieval workspace.",
    badge: "idle",
  },
];

export const InGroup: Story = {
  args: {},
  render: () => (
    <div className="w-[32rem]">
      <ItemGroup>
        {agents.map((agent, index) => (
          <Fragment key={agent.id}>
            {index > 0 && <ItemSeparator />}
            <Item variant="outline">
              <ItemMedia variant="icon">
                <agent.icon />
              </ItemMedia>
              <ItemContent>
                <ItemTitle>
                  {agent.title}
                  <Badge variant="secondary">{agent.badge}</Badge>
                </ItemTitle>
                <ItemDescription>{agent.description}</ItemDescription>
              </ItemContent>
              <ItemActions>
                <Button variant="ghost" size="icon-sm" aria-label="Preview">
                  <EyeIcon />
                </Button>
                <Button variant="ghost" size="icon-sm" aria-label="More">
                  <MoreHorizontalIcon />
                </Button>
              </ItemActions>
            </Item>
          </Fragment>
        ))}
      </ItemGroup>
    </div>
  ),
};

export const Muted: Story = {
  args: {},
  render: () => (
    <div className="w-[32rem]">
      <Item variant="muted">
        <ItemMedia variant="icon">
          <CpuIcon />
        </ItemMedia>
        <ItemContent>
          <ItemTitle>Claude Code</ItemTitle>
          <ItemDescription>Currently running the incident triage playbook.</ItemDescription>
        </ItemContent>
      </Item>
    </div>
  ),
};

export const Compact: Story = {
  args: {},
  render: () => (
    <div className="w-[28rem]">
      <ItemGroup>
        {agents.slice(0, 2).map(agent => (
          <Item key={agent.id} size="xs" variant="outline">
            <ItemMedia variant="icon">
              <agent.icon />
            </ItemMedia>
            <ItemContent>
              <ItemTitle>{agent.title}</ItemTitle>
            </ItemContent>
          </Item>
        ))}
      </ItemGroup>
    </div>
  ),
};
