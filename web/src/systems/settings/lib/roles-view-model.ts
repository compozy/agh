import type {
  RoleFallbackEntry,
  RoleName,
  RoleResolutionMode,
  RoleStatus,
  SettingsRolesConfig,
} from "../types";
import {
  ROLE_DESCRIPTIONS,
  ROLE_FIELDS,
  ROLE_LABELS,
  ROLE_ORDER,
  type RoleFieldDescriptor,
} from "./roles-config";

/** Compact header pill states. `catalog` mode carries no dedicated pill — the routed agent name conveys it. */
export type RoleBadge = "builtin" | "inherit" | "off";

export interface RoleViewModel {
  role: RoleName;
  label: string;
  description: string;
  /** Editable scalar fields for this role (routing + policy). */
  fields: readonly RoleFieldDescriptor[];
  /** Draft values for the scalar fields, flattened at this read boundary. */
  values: Record<string, string | number | boolean>;
  /** Draft fallback routes for this role. */
  fallbackChain: RoleFallbackEntry[];
  /**
   * Effective (projected) values for routing fields — `null` means "resolves at
   * invocation" and is never replaced with a fabricated default.
   */
  effective: Record<string, string | null>;
  badges: RoleBadge[];
  resolutionMode: RoleResolutionMode;
  resolutionLine: string;
  /** The raw projection — provenance, diagnostics, and null truth all read from it. */
  status: RoleStatus;
}

function computeBadges(status: RoleStatus): RoleBadge[] {
  const badges: RoleBadge[] = [];
  if (!status.enabled) {
    badges.push("off");
  }
  if (status.resolution_mode === "builtin") {
    badges.push("builtin");
  } else if (status.resolution_mode === "inherit") {
    badges.push("inherit");
  }
  return badges;
}

/** Inherit-mode roles resolve only at invocation; the surface says so rather than guessing. */
export function computeResolutionLine(status: RoleStatus): string {
  switch (status.resolution_mode) {
    case "inherit":
      return "Resolves at invocation.";
    case "builtin":
      return status.agent ? `Built-in · ${status.agent}` : "Built-in identity.";
    case "catalog":
      return status.agent ? `Agent · ${status.agent}` : "Routed to a catalog agent.";
    default:
      return "Resolves at invocation.";
  }
}

function buildEffective(status: RoleStatus): Record<string, string | null> {
  const effective: Record<string, string | null> = {
    agent: status.agent,
    provider: status.provider,
    model: status.model,
    reasoning_effort: status.reasoning_effort,
  };
  if (status.timeout != null) {
    effective.timeout = status.timeout;
  }
  return effective;
}

function buildRoleViewModel(
  role: RoleName,
  status: RoleStatus,
  config: SettingsRolesConfig
): RoleViewModel {
  const fields = ROLE_FIELDS[role];
  // Flatten the union-typed role config into a descriptor-keyed value map. The
  // cast is contained to this read boundary; only scalar descriptor keys are
  // read (never `fallback_chain`), so widening through `unknown` is safe.
  const roleConfig = config[role] as unknown as Record<string, string | number | boolean>;
  const values: Record<string, string | number | boolean> = {};
  for (const field of fields) {
    values[field.key] = roleConfig[field.key];
  }
  return {
    role,
    label: ROLE_LABELS[role],
    description: ROLE_DESCRIPTIONS[role],
    fields,
    values,
    fallbackChain: config[role].fallback_chain,
    effective: buildEffective(status),
    badges: computeBadges(status),
    resolutionMode: status.resolution_mode,
    resolutionLine: computeResolutionLine(status),
    status,
  };
}

/**
 * Join the read-only projection with the editable draft into the six role view
 * models in fixed product order (`ROLE_ORDER`), independent of the API's lexical
 * ordering. A role absent from the projection is skipped rather than fabricated.
 */
export function buildRolesViewModel(
  statuses: readonly RoleStatus[],
  config: SettingsRolesConfig
): RoleViewModel[] {
  const byRole = new Map(statuses.map(status => [status.role, status]));
  const models: RoleViewModel[] = [];
  for (const role of ROLE_ORDER) {
    const status = byRole.get(role);
    if (status) {
      models.push(buildRoleViewModel(role, status, config));
    }
  }
  return models;
}
