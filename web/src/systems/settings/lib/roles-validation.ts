import type { RoleName, SettingsRolesConfig } from "../types";
import { isReasoningEffortOptionValue, ROLE_ORDER } from "./roles-config";

export interface RoleFieldError {
  /** Deterministic field id shared with the rendered control's `data-field`. */
  id: string;
  message: string;
}

/** Stable id for a scalar role field control (`dream.model`). */
export function roleFieldId(role: RoleName, field: string): string {
  return `${role}.${field}`;
}

/** Stable id for a fallback-entry field control (`dream.fallback.0.provider`). */
export function fallbackFieldId(role: RoleName, index: number, field: string): string {
  return `${role}.fallback.${index}.${field}`;
}

/**
 * Validate every fallback entry across all roles in visual order (product role
 * order, entry order, provider → model → reasoning). Order is load-bearing:
 * the first element is the deterministic "first invalid field" the save flow
 * focuses. Provider and model are required; reasoning must be empty or a valid
 * enum value.
 */
export function collectRoleValidationErrors(config: SettingsRolesConfig): RoleFieldError[] {
  const errors: RoleFieldError[] = [];
  for (const role of ROLE_ORDER) {
    config[role].fallback_chain.forEach((entry, index) => {
      if (entry.provider.trim() === "") {
        errors.push({
          id: fallbackFieldId(role, index, "provider"),
          message: "Provider is required.",
        });
      }
      if (entry.model.trim() === "") {
        errors.push({ id: fallbackFieldId(role, index, "model"), message: "Model is required." });
      }
      if (!isReasoningEffortOptionValue(entry.reasoning_effort)) {
        errors.push({
          id: fallbackFieldId(role, index, "reasoning_effort"),
          message: "Choose a valid reasoning effort.",
        });
      }
    });
  }
  return errors;
}

/** Field-id → message map (first message wins per id). */
export function toRoleErrorMap(errors: RoleFieldError[]): Record<string, string> {
  const map: Record<string, string> = {};
  for (const error of errors) {
    if (!(error.id in map)) {
      map[error.id] = error.message;
    }
  }
  return map;
}

/** The first invalid field in visual order, or null when the draft is valid. */
export function firstRoleFieldError(errors: RoleFieldError[]): RoleFieldError | null {
  return errors[0] ?? null;
}
