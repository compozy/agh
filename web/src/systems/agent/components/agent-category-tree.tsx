import { Link } from "@tanstack/react-router";
import type { ItemInstance } from "@headless-tree/core";
import { Bot, Loader2, TriangleAlert } from "lucide-react";

import { cn, Pill, Tree, TreeItem, TreeItemLabel } from "@agh/ui";

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
      <div
        data-testid="agents-loading"
        className="flex items-center gap-2 px-3 py-2 text-xs text-(--color-text-tertiary)"
      >
        <Loader2 aria-hidden="true" className="size-3 animate-spin" />
        <span>Loading agents...</span>
      </div>
    );
  }

  if (hasAgents) {
    return <AgentCategoryTreeContent agents={agents} sessions={sessions} />;
  }

  if (agentsError) {
    return (
      <div
        data-testid="agents-error"
        className="flex items-center gap-2 px-3 py-2 text-xs text-(--color-text-tertiary)"
      >
        <TriangleAlert aria-hidden="true" className="size-3" />
        <span>Could not load agents. Retry once the daemon is reachable.</span>
      </div>
    );
  }

  if (!agents || agents.length === 0) {
    return (
      <div
        data-testid="agents-empty"
        className="flex items-center gap-2 px-3 py-2 text-xs text-(--color-text-tertiary)"
      >
        <Bot aria-hidden="true" className="size-3" />
        <span>Run `agh install` to bootstrap AGH</span>
      </div>
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
      indent={12}
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
      className="text-(--color-text-secondary)"
    >
      <TreeItemLabel
        item={item}
        className={cn(
          "flex items-center gap-1 rounded-mono-badge bg-transparent px-1.5 py-1 font-mono text-badge font-medium uppercase tracking-mono text-(--color-text-label)",
          "hover:bg-(--color-hover) hover:text-(--color-text-primary)"
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
  return (
    <TreeItem
      item={item}
      render={<Link to="/agents/$name" params={{ name: agent.name }} />}
      data-testid={`agent-row-${agent.name}`}
      data-active={isActive}
      className={cn(NAV_ROW_CLASS, isActive && ACTIVE_NAV_ROW_CLASS)}
    >
      {isActive ? (
        <span
          aria-hidden="true"
          data-testid={`agent-active-${agent.name}`}
          className={ACTIVE_NAV_INDICATOR_CLASS}
        />
      ) : null}
      <AgentIcon
        provider={agent.provider}
        className="size-3.5 shrink-0 text-(--color-text-tertiary)"
      />
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
