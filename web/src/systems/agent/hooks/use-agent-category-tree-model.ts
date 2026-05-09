import { useMemo } from "react";
import { useMatchRoute } from "@tanstack/react-router";
import { hotkeysCoreFeature, selectionFeature, syncDataLoaderFeature } from "@headless-tree/core";
import { useTree } from "@headless-tree/react";

import type { SessionPayload } from "@/systems/session";

import {
  AGENT_CATEGORY_FOLDER_ID_PREFIX,
  buildAgentCategoryTree,
  type AgentCategoryNode,
} from "../lib/agent-category";
import type { AgentPayload } from "../types";

export const AGENT_CATEGORY_TREE_ROOT_ID = "__agents_root__";

export interface AgentCategoryTreeItemData {
  kind: "root" | "folder" | "leaf";
  label: string;
  agent?: AgentPayload;
  segments?: string[];
}

interface FlatItems {
  data: Record<string, AgentCategoryTreeItemData>;
  children: Record<string, string[]>;
}

function flattenNodes(nodes: AgentCategoryNode[]): FlatItems {
  const data: Record<string, AgentCategoryTreeItemData> = {
    [AGENT_CATEGORY_TREE_ROOT_ID]: { kind: "root", label: "" },
  };
  const children: Record<string, string[]> = {
    [AGENT_CATEGORY_TREE_ROOT_ID]: nodes.map(node => node.id),
  };

  const visit = (node: AgentCategoryNode) => {
    if (node.kind === "folder") {
      data[node.id] = { kind: "folder", label: node.label, segments: node.segments };
      children[node.id] = node.children.map(child => child.id);
      for (const child of node.children) visit(child);
      return;
    }
    data[node.id] = { kind: "leaf", label: node.label, agent: node.agent };
    children[node.id] = [];
  };
  for (const node of nodes) visit(node);
  return { data, children };
}

function ancestorFolderIds(segments: string[]): string[] {
  const ids: string[] = [];
  for (let i = 1; i <= segments.length; i += 1) {
    ids.push(`${AGENT_CATEGORY_FOLDER_ID_PREFIX}${segments.slice(0, i).join("/")}`);
  }
  return ids;
}

function topLevelFolderIds(nodes: AgentCategoryNode[]): string[] {
  return nodes.filter(node => node.kind === "folder").map(node => node.id);
}

function findActiveAgent(
  agents: AgentPayload[],
  matchRoute: ReturnType<typeof useMatchRoute>
): AgentPayload | null {
  for (const agent of agents) {
    const matched = matchRoute({
      to: "/agents/$name",
      params: { name: agent.name },
      fuzzy: true,
    });
    if (matched) return agent;
  }
  return null;
}

function collectActiveAgentNames(sessions: SessionPayload[] | undefined): Set<string> {
  const names = new Set<string>();
  if (!sessions) return names;
  for (const session of sessions) {
    if (session.state === "active") names.add(session.agent_name);
  }
  return names;
}

// Computed once at mount and handed to useTree's initialState.expandedItems.
// Headless-tree treats initialState as a seed value, so route changes after the
// tree has mounted MUST NOT silently auto-expand a different category -- operator
// expansion intent wins from that point on. The TechSpec accepts initial-only
// expansion; URL/config persistence is intentionally out of scope.
function deriveInitialExpanded(
  agents: AgentPayload[] | undefined,
  matchRoute: ReturnType<typeof useMatchRoute>,
  nodes: AgentCategoryNode[]
): string[] {
  const activeAgent = agents ? findActiveAgent(agents, matchRoute) : null;
  if (activeAgent && activeAgent.category_path && activeAgent.category_path.length > 0) {
    return ancestorFolderIds(activeAgent.category_path);
  }
  return topLevelFolderIds(nodes);
}

export interface AgentCategoryTreeModel {
  nodes: AgentCategoryNode[];
  tree: ReturnType<typeof useTree<AgentCategoryTreeItemData>>;
  activeAgentNames: Set<string>;
  matchRoute: ReturnType<typeof useMatchRoute>;
}

export function useAgentCategoryTreeModel(
  agents: AgentPayload[] | undefined,
  sessions: SessionPayload[] | undefined
): AgentCategoryTreeModel {
  const matchRoute = useMatchRoute();
  const activeAgentNames = useMemo(() => collectActiveAgentNames(sessions), [sessions]);
  const nodes = useMemo(() => buildAgentCategoryTree(agents ?? []), [agents]);
  const flat = useMemo(() => flattenNodes(nodes), [nodes]);
  const initialExpanded = useMemo(
    () => deriveInitialExpanded(agents, matchRoute, nodes),
    [agents, matchRoute, nodes]
  );
  const tree = useTree<AgentCategoryTreeItemData>({
    rootItemId: AGENT_CATEGORY_TREE_ROOT_ID,
    getItemName: item => item.getItemData().label,
    isItemFolder: item => item.getItemData().kind === "folder",
    initialState: { expandedItems: initialExpanded },
    dataLoader: {
      getItem: id => flat.data[id] ?? { kind: "leaf", label: "" },
      getChildren: id => flat.children[id] ?? [],
    },
    features: [syncDataLoaderFeature, selectionFeature, hotkeysCoreFeature],
  });
  return { nodes, tree, activeAgentNames, matchRoute };
}
