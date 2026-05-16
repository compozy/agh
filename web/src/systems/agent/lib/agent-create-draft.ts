import type { ProviderSelectOption } from "@/systems/runtime";

import type { CreateAgentParams } from "../types";

export type AgentCreateScope = CreateAgentParams["scope"];
export type AgentCreatePermission = NonNullable<CreateAgentParams["agent"]["permissions"]>;
export type AgentCreatePermissionChoice = "" | AgentCreatePermission;
export type AgentCreateStep = "basics" | "runtime" | "instructions" | "access";

export interface AgentCreateDialogDraft {
  scope: AgentCreateScope;
  name: string;
  categoryPath: string;
  provider: string;
  model: string;
  command: string;
  prompt: string;
  permissions: AgentCreatePermissionChoice;
  tools: string[];
  toolsets: string[];
  denyTools: string[];
  disabledSkills: string[];
}

export interface AgentCreateValidationContext {
  hasActiveWorkspace: boolean;
  providerOptions: readonly ProviderSelectOption[];
  providersError: string | null;
  providersLoading: boolean;
}

export interface AgentCreateValidation {
  fields: Partial<Record<AgentCreateFieldKey, string>>;
  stepValidity: Record<AgentCreateStep, boolean>;
  canSubmit: boolean;
  categorySegments: string[];
}

export type AgentCreateFieldKey =
  | "name"
  | "scope"
  | "categoryPath"
  | "provider"
  | "prompt"
  | "tools"
  | "toolsets"
  | "denyTools";

export const AGENT_CREATE_PERMISSION_OPTIONS: readonly {
  value: AgentCreatePermissionChoice;
  label: string;
}[] = [
  { value: "", label: "Inherit default" },
  { value: "deny-all", label: "Deny all" },
  { value: "approve-reads", label: "Approve reads" },
  { value: "approve-all", label: "Approve all" },
] as const;

export function createDefaultAgentCreateDraft(hasActiveWorkspace: boolean): AgentCreateDialogDraft {
  return {
    scope: hasActiveWorkspace ? "workspace" : "global",
    name: "",
    categoryPath: "",
    provider: "",
    model: "",
    command: "",
    prompt: "",
    permissions: "",
    tools: [],
    toolsets: [],
    denyTools: [],
    disabledSkills: [],
  };
}

export function updateAgentCreateScope(
  draft: AgentCreateDialogDraft,
  scope: AgentCreateScope
): AgentCreateDialogDraft {
  if (draft.scope === scope) return draft;
  return {
    ...draft,
    scope,
    provider: "",
    model: "",
  };
}

export function appendAgentCreateTokens(current: readonly string[], rawInput: string): string[] {
  const next = [...current];
  const seen = new Set(next);
  for (const token of splitAgentCreateTokens(rawInput)) {
    if (seen.has(token)) continue;
    seen.add(token);
    next.push(token);
  }
  return next;
}

export function removeAgentCreateToken(current: readonly string[], target: string): string[] {
  return current.filter(value => value !== target);
}

export function splitAgentCreateTokens(rawInput: string): string[] {
  const values = rawInput
    .split(/[,\n]/)
    .map(value => value.trim())
    .filter(Boolean);
  return [...new Set(values)];
}

export function parseAgentCreateCategoryPath(rawInput: string): {
  segments: string[];
  error: string | null;
} {
  const trimmed = rawInput.trim();
  if (trimmed.length === 0) {
    return { segments: [], error: null };
  }
  if (trimmed.includes("\\")) {
    return { segments: [], error: "Category path cannot contain backslashes." };
  }

  const rawSegments = trimmed.split("/");
  const segments: string[] = [];
  for (const rawSegment of rawSegments) {
    const segment = rawSegment.trim();
    if (segment.length === 0) {
      return { segments: [], error: "Category path cannot contain blank segments." };
    }
    if (segment === "." || segment === "..") {
      return { segments: [], error: "Category path cannot contain . or .. segments." };
    }
    if (segment.includes("\\") || segment.includes("/")) {
      return { segments: [], error: "Category path segments cannot contain path separators." };
    }
    segments.push(segment);
  }

  return { segments, error: null };
}

export function validateAgentCreateDraft(
  draft: AgentCreateDialogDraft,
  context: AgentCreateValidationContext
): AgentCreateValidation {
  const fields: AgentCreateValidation["fields"] = {};
  const name = draft.name.trim();
  if (name.length === 0) {
    fields.name = "Enter an agent name.";
  } else if (name === "." || name === ".." || name.includes("/") || name.includes("\\")) {
    fields.name = "Agent names cannot be . or .. and cannot contain path separators.";
  }

  if (draft.scope === "workspace" && !context.hasActiveWorkspace) {
    fields.scope = "Select an active workspace or switch scope to global.";
  }

  const category = parseAgentCreateCategoryPath(draft.categoryPath);
  if (category.error) {
    fields.categoryPath = category.error;
  }

  const provider = draft.provider.trim();
  const providerKnown = context.providerOptions.some(option => option.name === provider);
  if (context.providersLoading) {
    fields.provider = "Provider options are still loading.";
  } else if (context.providersError) {
    fields.provider = context.providersError;
  } else if (context.providerOptions.length === 0) {
    fields.provider = "No providers are configured for this scope.";
  } else if (provider.length === 0) {
    fields.provider = "Choose a provider.";
  } else if (!providerKnown) {
    fields.provider = "Choose a provider from this scope.";
  }

  if (draft.prompt.trim().length === 0) {
    fields.prompt = "Enter the agent instructions.";
  }

  const toolsError = validateToolPatternList(draft.tools, "Tool");
  if (toolsError) fields.tools = toolsError;
  const denyToolsError = validateToolPatternList(draft.denyTools, "Denied tool");
  if (denyToolsError) fields.denyTools = denyToolsError;
  const toolsetsError = validateToolsetList(draft.toolsets);
  if (toolsetsError) fields.toolsets = toolsetsError;

  const stepValidity: Record<AgentCreateStep, boolean> = {
    basics: !fields.name && !fields.scope && !fields.categoryPath,
    runtime: !fields.provider,
    instructions: !fields.prompt,
    access: !fields.tools && !fields.toolsets && !fields.denyTools,
  };

  return {
    fields,
    stepValidity,
    canSubmit: Object.values(stepValidity).every(Boolean),
    categorySegments: category.segments,
  };
}

export function buildCreateAgentParams(
  draft: AgentCreateDialogDraft,
  workspaceId: string | null | undefined,
  context: AgentCreateValidationContext
): CreateAgentParams | null {
  const validation = validateAgentCreateDraft(draft, context);
  if (!validation.canSubmit) return null;
  const normalizedWorkspaceId = workspaceId?.trim() ?? "";
  if (draft.scope === "workspace" && normalizedWorkspaceId.length === 0) {
    return null;
  }

  const name = draft.name.trim();
  const provider = draft.provider.trim();
  const prompt = draft.prompt.trim();
  const model = draft.model.trim();
  const command = draft.command.trim();
  const tools = normalizeOrderedTokens(draft.tools);
  const toolsets = normalizeOrderedTokens(draft.toolsets);
  const denyTools = normalizeOrderedTokens(draft.denyTools);
  const disabledSkills = normalizeOrderedTokens(draft.disabledSkills);
  const permissions = draft.permissions === "" ? undefined : draft.permissions;

  return {
    scope: draft.scope,
    ...(draft.scope === "workspace" ? { workspace: normalizedWorkspaceId } : {}),
    agent: {
      name,
      provider,
      prompt,
      ...(command.length > 0 ? { command } : {}),
      ...(model.length > 0 ? { model } : {}),
      ...(tools.length > 0 ? { tools } : {}),
      ...(toolsets.length > 0 ? { toolsets } : {}),
      ...(denyTools.length > 0 ? { deny_tools: denyTools } : {}),
      ...(permissions ? { permissions } : {}),
      ...(validation.categorySegments.length > 0
        ? { category_path: validation.categorySegments }
        : {}),
      ...(disabledSkills.length > 0 ? { skills: { disabled: disabledSkills } } : {}),
    },
  };
}

function normalizeOrderedTokens(values: readonly string[]): string[] {
  const seen = new Set<string>();
  const normalized: string[] = [];
  for (const value of values) {
    const trimmed = value.trim();
    if (trimmed.length === 0 || seen.has(trimmed)) continue;
    seen.add(trimmed);
    normalized.push(trimmed);
  }
  return normalized;
}

function validateToolPatternList(values: readonly string[], label: string): string | null {
  for (const value of values) {
    const trimmed = value.trim();
    if (trimmed.length === 0) return label + " entries cannot be blank.";
    if (!isValidToolPattern(trimmed)) {
      return (
        label +
        " entries must use canonical IDs such as agh__skill_view or namespace wildcards such as agh__task_*."
      );
    }
  }
  return null;
}

function validateToolsetList(values: readonly string[]): string | null {
  for (const value of values) {
    const trimmed = value.trim();
    if (trimmed.length === 0) return "Toolset entries cannot be blank.";
    if (!isValidCanonicalRef(trimmed) || trimmed.includes("*")) {
      return "Toolset entries must use canonical IDs such as agh__catalog.";
    }
  }
  return null;
}

function isValidToolPattern(value: string): boolean {
  if (value.includes("*")) {
    if (value.split("*").length !== 2 || !value.endsWith("*")) return false;
    const prefix = value.slice(0, -1);
    if (prefix.length === 0 || (!prefix.endsWith("_") && !prefix.endsWith("__"))) {
      return false;
    }
    return isValidCanonicalRef(prefix + "x");
  }
  return isValidCanonicalRef(value);
}

function isValidCanonicalRef(value: string): boolean {
  if (!value.includes("__")) return false;
  if (/\s/.test(value)) return false;
  if (value.includes("/") || value.includes("\\")) return false;
  return /^[A-Za-z0-9][A-Za-z0-9_.-]*__[A-Za-z0-9][A-Za-z0-9_.-]*$/.test(value);
}
