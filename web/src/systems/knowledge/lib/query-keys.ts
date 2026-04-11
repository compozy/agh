export const knowledgeKeys = {
  all: ["knowledge"] as const,
  list: (scope?: string, workspace?: string) =>
    [...knowledgeKeys.all, "list", scope ?? "", workspace ?? ""] as const,
  detail: (scope?: string, filename?: string, workspace?: string) =>
    [...knowledgeKeys.all, "detail", scope ?? "", filename ?? "", workspace ?? ""] as const,
};
