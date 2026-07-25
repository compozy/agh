import type { ReasoningEffort } from "@/lib/api-contract";

import type { RoleFallbackEntry, RoleName, SettingsRolesConfig } from "../types";

/**
 * Fixed product display order, independent of the API's lexical sort
 * (`GET /api/roles` returns roles sorted by name).
 */
export const ROLE_ORDER: readonly RoleName[] = [
  "coordinator",
  "dream",
  "checkpoint_summary",
  "memory_extractor",
  "auto_title",
  "memory_controller",
];

/** Sentence-case role names for panel headers. */
export const ROLE_LABELS: Record<RoleName, string> = {
  coordinator: "Coordinator",
  dream: "Dream",
  checkpoint_summary: "Checkpoint summary",
  memory_extractor: "Memory extractor",
  auto_title: "Auto title",
  memory_controller: "Memory controller",
};

/** One-line description of what each background role does. */
export const ROLE_DESCRIPTIONS: Record<RoleName, string> = {
  coordinator: "Orchestrates managed sessions.",
  dream: "Consolidates memory while sessions are idle.",
  checkpoint_summary: "Summarizes session checkpoints.",
  memory_extractor: "Extracts durable memories from turns.",
  auto_title: "Names new sessions from their first exchange.",
  memory_controller: "In-process tie-breaker for memory writes.",
};

export type RoleFieldKind = "switch" | "text" | "select" | "number";

export interface RoleFieldDescriptor {
  /** Config field key under `roles.<role>`. */
  key: string;
  label: string;
  description?: string;
  kind: RoleFieldKind;
  placeholder?: string;
  /** Minimum for `number` controls. */
  min?: number;
  /** Render the text control in the mono family (ids, durations, model refs). */
  mono?: boolean;
}

const ENABLED_FIELD: RoleFieldDescriptor = {
  key: "enabled",
  label: "Enabled",
  description: "Turn this background role on or off.",
  kind: "switch",
};
const AGENT_FIELD: RoleFieldDescriptor = {
  key: "agent",
  label: "Agent",
  description: "Route to a catalog agent, or leave empty for the role default.",
  kind: "text",
  placeholder: "role default",
  mono: true,
};
const PROVIDER_FIELD: RoleFieldDescriptor = {
  key: "provider",
  label: "Provider",
  description: "Provider override; empty inherits the agent or default provider.",
  kind: "text",
  placeholder: "inherit",
  mono: true,
};
const MODEL_FIELD: RoleFieldDescriptor = {
  key: "model",
  label: "Model",
  description: "Model override; empty inherits the provider or agent default.",
  kind: "text",
  placeholder: "inherit",
  mono: true,
};
const REASONING_FIELD: RoleFieldDescriptor = {
  key: "reasoning_effort",
  label: "Reasoning effort",
  description: "Effort override; empty inherits the provider or agent default.",
  kind: "select",
};

const ROUTING_FIELDS: readonly RoleFieldDescriptor[] = [
  ENABLED_FIELD,
  AGENT_FIELD,
  PROVIDER_FIELD,
  MODEL_FIELD,
  REASONING_FIELD,
];

const COORDINATOR_FIELDS: readonly RoleFieldDescriptor[] = [
  ...ROUTING_FIELDS,
  {
    key: "ttl",
    label: "Session TTL",
    description: "Coordinator session lifetime.",
    kind: "text",
    placeholder: "2h",
    mono: true,
  },
  {
    key: "max_children",
    label: "Max children",
    description: "Safe-spawn child cap.",
    kind: "number",
    min: 1,
  },
  {
    key: "max_active_sessions_per_workspace",
    label: "Max active sessions",
    description: "Managed-session cap per workspace.",
    kind: "number",
    min: 1,
  },
];

const MEMORY_CONTROLLER_FIELDS: readonly RoleFieldDescriptor[] = [
  ENABLED_FIELD,
  PROVIDER_FIELD,
  MODEL_FIELD,
  REASONING_FIELD,
  {
    key: "timeout",
    label: "Timeout",
    description: "Wall-clock ceiling for the in-process call.",
    kind: "text",
    placeholder: "250ms",
    mono: true,
  },
  {
    key: "top_k",
    label: "Top K",
    description: "Tie-breaker candidate count.",
    kind: "number",
    min: 1,
  },
  { key: "prompt_version", label: "Prompt version", kind: "text", placeholder: "v1", mono: true },
  { key: "max_tokens_out", label: "Max output tokens", kind: "number", min: 1 },
];

/** Editable scalar fields per role (routing + role-specific policy). */
export const ROLE_FIELDS: Record<RoleName, readonly RoleFieldDescriptor[]> = {
  coordinator: COORDINATOR_FIELDS,
  dream: ROUTING_FIELDS,
  checkpoint_summary: ROUTING_FIELDS,
  memory_extractor: ROUTING_FIELDS,
  auto_title: ROUTING_FIELDS,
  memory_controller: MEMORY_CONTROLLER_FIELDS,
};

const REASONING_VALUES = [
  "none",
  "minimal",
  "low",
  "medium",
  "high",
  "xhigh",
  "max",
] as const satisfies readonly ReasoningEffort[];

/** Reasoning-effort options ("" = inherit) shared with the AGENT.md enum. */
export const REASONING_OPTIONS: readonly { value: string; label: string }[] = [
  { value: "", label: "Default (inherit)" },
  ...REASONING_VALUES.map(value => ({ value, label: value })),
];

export function isReasoningEffortOptionValue(value: string): boolean {
  return value === "" || (REASONING_VALUES as readonly string[]).includes(value);
}

/**
 * Immutably set one scalar role field, returning a new full section config.
 * The cast is contained here: `ROLE_FIELDS` guarantees `field` is a valid key
 * for `role` and `value`'s runtime kind matches the descriptor.
 */
export function applyRoleFieldEdit(
  config: SettingsRolesConfig,
  role: RoleName,
  field: string,
  value: string | number | boolean
): SettingsRolesConfig {
  return {
    ...config,
    [role]: { ...config[role], [field]: value },
  } as SettingsRolesConfig;
}

export function emptyFallbackEntry(): RoleFallbackEntry {
  return { provider: "", model: "", reasoning_effort: "" };
}

function applyRoleFallbackChain(
  config: SettingsRolesConfig,
  role: RoleName,
  chain: RoleFallbackEntry[]
): SettingsRolesConfig {
  return {
    ...config,
    [role]: { ...config[role], fallback_chain: chain },
  } as SettingsRolesConfig;
}

export function addFallbackEntry(config: SettingsRolesConfig, role: RoleName): SettingsRolesConfig {
  return applyRoleFallbackChain(config, role, [
    ...config[role].fallback_chain,
    emptyFallbackEntry(),
  ]);
}

export function removeFallbackEntry(
  config: SettingsRolesConfig,
  role: RoleName,
  index: number
): SettingsRolesConfig {
  return applyRoleFallbackChain(
    config,
    role,
    config[role].fallback_chain.filter((_entry, entryIndex) => entryIndex !== index)
  );
}

export function updateFallbackEntry(
  config: SettingsRolesConfig,
  role: RoleName,
  index: number,
  field: keyof RoleFallbackEntry,
  value: string
): SettingsRolesConfig {
  const chain = config[role].fallback_chain.map((entry, entryIndex) =>
    entryIndex === index ? { ...entry, [field]: value } : entry
  );
  return applyRoleFallbackChain(config, role, chain);
}
