import { useState } from "react";

import { roleFieldId } from "../lib/roles-validation";
import type { RoleViewModel } from "../lib/roles-view-model";
import type { RoleFallbackEntry, RoleName } from "../types";
import { RoleAdvancedDetails } from "./role-advanced-details";
import { RoleDiagnosticsNotice } from "./role-diagnostics-notice";
import { RoleFieldControl } from "./role-field-control";
import { RoleStatusBadges } from "./role-status-badges";
import { SettingsGroup } from "./settings-group";

const TEST_PREFIX = "settings-page-roles";

export interface RoleSettingsGroupProps {
  vm: RoleViewModel;
  validationErrors: Record<string, string>;
  disabled?: boolean;
  draftRevision?: number;
  setRoleField: (role: RoleName, field: string, value: string | number | boolean) => void;
  setNumberFieldValidity: (id: string) => (message: string | null) => void;
  addFallback: (role: RoleName) => void;
  removeFallback: (role: RoleName, index: number) => void;
  updateFallback: (
    role: RoleName,
    index: number,
    field: keyof RoleFallbackEntry,
    value: string
  ) => void;
  registerFieldRef: (id: string) => (element: HTMLElement | null) => void;
}

/**
 * One stacked role panel: header (name, badges, resolution line), inline
 * diagnostics, editable routing/policy fields, and the advanced fold with the
 * fallback editor + provenance chips.
 */
export function RoleSettingsGroup({
  vm,
  validationErrors,
  disabled,
  draftRevision,
  setRoleField,
  setNumberFieldValidity,
  addFallback,
  removeFallback,
  updateFallback,
  registerFieldRef,
}: RoleSettingsGroupProps) {
  const {
    role,
    label,
    description,
    fields,
    values,
    fallbackChain,
    effective,
    badges,
    resolutionLine,
    status,
  } = vm;
  const [userOpen, setUserOpen] = useState(false);
  const hasFallbackError = Object.keys(validationErrors).some(id =>
    id.startsWith(`${role}.fallback.`)
  );

  return (
    <SettingsGroup
      bare
      data-testid={`${TEST_PREFIX}-group-${role}`}
      title={label}
      description={
        <span className="flex flex-col gap-0.5">
          <span>{description}</span>
          <span
            className="text-form-hint text-subtle"
            data-testid={`${TEST_PREFIX}-${role}-resolution`}
          >
            {resolutionLine}
          </span>
        </span>
      }
      action={<RoleStatusBadges badges={badges} data-testid={`${TEST_PREFIX}-${role}-badges`} />}
    >
      <RoleDiagnosticsNotice
        diagnostics={status.diagnostics}
        data-testid={`${TEST_PREFIX}-${role}-diagnostics`}
      />
      <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
        {fields.map(field => {
          const id = roleFieldId(role, field.key);
          const hasEffective = field.key in effective;
          return (
            <RoleFieldControl
              key={field.key}
              field={field}
              value={values[field.key]}
              hasEffective={hasEffective}
              effective={hasEffective ? effective[field.key] : null}
              error={validationErrors[id]}
              disabled={disabled}
              testId={`${TEST_PREFIX}-${role}-${field.key}`}
              resetRevision={draftRevision}
              fieldRef={field.kind === "number" ? registerFieldRef(id) : undefined}
              onValueChange={value => setRoleField(role, field.key, value)}
              onValidityChange={field.kind === "number" ? setNumberFieldValidity(id) : undefined}
            />
          );
        })}
      </div>
      <RoleAdvancedDetails
        role={role}
        entries={fallbackChain}
        provenance={status.provenance}
        errors={validationErrors}
        open={userOpen || hasFallbackError}
        onOpenChange={setUserOpen}
        disabled={disabled}
        testId={`${TEST_PREFIX}-${role}-advanced`}
        onAdd={() => addFallback(role)}
        onRemove={index => removeFallback(role, index)}
        onUpdate={(index, field, value) => updateFallback(role, index, field, value)}
        registerFieldRef={registerFieldRef}
      />
    </SettingsGroup>
  );
}
