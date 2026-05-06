import type { KnowledgeSelector } from "@/systems/knowledge/types";

function selectorTuple(selector?: KnowledgeSelector) {
  return [
    selector?.scope ?? "",
    selector?.workspaceId ?? "",
    selector?.agentName ?? "",
    selector?.agentTier ?? "",
  ] as const;
}

export const knowledgeKeys = {
  all: ["knowledge"] as const,
  lists: () => [...knowledgeKeys.all, "list"] as const,
  list: (selector?: KnowledgeSelector) =>
    [...knowledgeKeys.lists(), ...selectorTuple(selector)] as const,
  details: () => [...knowledgeKeys.all, "detail"] as const,
  detail: (filename: string, selector?: KnowledgeSelector) =>
    [...knowledgeKeys.details(), filename, ...selectorTuple(selector)] as const,
  searches: () => [...knowledgeKeys.all, "search"] as const,
  search: (queryText: string, selector?: KnowledgeSelector) =>
    [...knowledgeKeys.searches(), queryText, ...selectorTuple(selector)] as const,
  decisions: () => [...knowledgeKeys.all, "decisions"] as const,
  decisionsFor: (filename: string, selector?: KnowledgeSelector) =>
    [...knowledgeKeys.decisions(), filename, ...selectorTuple(selector)] as const,
};
