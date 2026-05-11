import { Link } from "@tanstack/react-router";
import type { ItemInstance } from "@headless-tree/core";
import { Bot, TriangleAlert } from "lucide-react";

import { cn, Empty, Pill, Spinner, Tree, TreeItem, TreeItemLabel } from "@agh/ui";

import {
  ACTIVE_NAV_INDICATOR_CLASS,
  ACTIVE_NAV_ROW_CLASS,
  NAV_ROW_CLASS,
} from "@/components/sidebar-nav-classes";
import type { SessionPayload } from "@/systems/session";

import { joinAgentCategorySegments } from "../lib/agent-category";
import {
  useAgentCategoryTreeModel,
  type AgentCategoryTreeItemData,
} from "../hooks/use-agent-category-tree-model";
import { AgentIcon } from "./agent-icon";
import type { AgentPayload } from "../types";

const AGENT_TREE_INDENT = 12;

export interface AgentCategoryTreeProps {
  agents: AgentPayload[] | undefined;
  agentsLoading: boolean;
  agentsError: boolean;
  sessions: SessionPayload[] | undefined;
}

export function AgentCategoryTree({
  agents,
  agentsLoading,
  agentsError,
  sessions,
}: AgentCategoryTreeProps) {
  const hasAgents = Array.isArray(agents) && agents.length > 0;

  if (agentsLoading) {
    return (
      <Empty
        icon={<Spinner className="size-4" />}
        title="Loading agents..."
        titleAs="span"
        fill={false}
        data-testid="agents-loading"
        className="items-start gap-2 px-2 py-2 text-left"
      />
    );
  }

  if (hasAgents) {
    return <AgentCategoryTreeContent agents={agents} sessions={sessions} />;
  }

  if (agentsError) {
    return (
      <Empty
        icon={TriangleAlert}
        title="Could not load agents. Retry once the daemon is reachable."
        titleAs="span"
        fill={false}
        data-testid="agents-error"
        className="items-start gap-2 px-2 py-2 text-left"
      />
    );
  }

  if (!agents || agents.length === 0) {
    return (
      <Empty
        icon={Bot}
        title="Run `agh install` to bootstrap AGH"
        titleAs="span"
        fill={false}
        data-testid="agents-empty"
        className="items-start gap-2 px-2 py-2 text-left"
      />
    );
  }

  return null;
}

interface AgentCategoryTreeContentProps {
  agents: AgentPayload[];
  sessions: SessionPayload[] | undefined;
}

function AgentCategoryTreeContent({ agents, sessions }: AgentCategoryTreeContentProps) {
  const { tree, activeAgentNames, matchRoute } = useAgentCategoryTreeModel(agents, sessions);

  return (
    <Tree
      tree={tree}
      indent={AGENT_TREE_INDENT}
      data-testid="agent-category-tree"
      aria-label="Agents"
      className="gap-0.5"
    >
      {tree.getItems().map(item => {
        const data = item.getItemData();
        if (data.kind === "folder") {
          return (
            <FolderRow
              key={item.getId()}
              item={item}
              label={data.label}
              segments={data.segments ?? []}
            />
          );
        }
        if (data.kind === "leaf" && data.agent) {
          const agent = data.agent;
          const isActive = Boolean(
            matchRoute({ to: "/agents/$name", params: { name: agent.name }, fuzzy: true })
          );
          return (
            <LeafRow
              key={item.getId()}
              item={item}
              agent={agent}
              isActive={isActive}
              hasActiveSession={activeAgentNames.has(agent.name)}
            />
          );
        }
        return null;
      })}
    </Tree>
  );
}

interface FolderRowProps {
  item: ItemInstance<AgentCategoryTreeItemData>;
  label: string;
  segments: string[];
}

function FolderRow({ item, label, segments }: FolderRowProps) {
  const expanded = item.isExpanded();
  const joined = joinAgentCategorySegments(segments);
  return (
    <TreeItem
      item={item}
      data-testid={`agent-category-${joined}`}
      data-expanded={expanded}
      className="text-(--muted)"
    >
      <TreeItemLabel
        item={item}
        className={cn(
          "eyebrow flex items-center gap-1 rounded-mono-badge bg-transparent px-2 py-1 text-(--muted)",
          "hover:bg-(--hover) hover:text-(--fg)"
        )}
      >
        <span className="truncate">{label}</span>
      </TreeItemLabel>
    </TreeItem>
  );
}

interface LeafRowProps {
  item: ItemInstance<AgentCategoryTreeItemData>;
  agent: AgentPayload;
  isActive: boolean;
  hasActiveSession: boolean;
}

function LeafRow({ item, agent, isActive, hasActiveSession }: LeafRowProps) {
  const level = item.getItemMeta().level;
  return (
    <TreeItem
      item={item}
      render={<Link to="/agents/$name" params={{ name: agent.name }} />}
      data-testid={`agent-row-${agent.name}`}
      data-active={isActive}
      className={cn(NAV_ROW_CLASS, isActive && ACTIVE_NAV_ROW_CLASS)}
      style={{ paddingInlineStart: `${level * AGENT_TREE_INDENT + 8}px` }}
    >
      {isActive ? (
        <span
          aria-hidden="true"
          data-testid={`agent-active-${agent.name}`}
          className={ACTIVE_NAV_INDICATOR_CLASS}
        />
      ) : null}
      <AgentIcon provider={agent.provider} className="size-3.5 shrink-0 text-(--subtle)" />
      <span className="truncate">{agent.name}</span>
      {hasActiveSession ? (
        <Pill.Dot
          tone="success"
          size="sm"
          className="ml-auto"
          data-testid={`agent-status-dot-${agent.name}`}
        />
      ) : null}
    </TreeItem>
  );
}
