import type { RoleFallbackEntry, RoleName } from "../types";
import { RoleFallbackEditor } from "./role-fallback-editor";
import { SettingsAdvancedFold, SettingsProvChip } from "./settings-advanced-fold";

interface RoleAdvancedDetailsProps {
  role: RoleName;
  entries: readonly RoleFallbackEntry[];
  provenance: Record<string, string>;
  errors: Record<string, string>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  disabled?: boolean;
  testId: string;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, field: keyof RoleFallbackEntry, value: string) => void;
  registerFieldRef: (id: string) => (element: HTMLElement | null) => void;
}

/**
 * Advanced fold hosting the editable fallback chain and the returned config
 * provenance chips. The fold is controlled so it opens automatically when its
 * role has a fallback validation error.
 */
export function RoleAdvancedDetails({
  role,
  entries,
  provenance,
  errors,
  open,
  onOpenChange,
  disabled,
  testId,
  onAdd,
  onRemove,
  onUpdate,
  registerFieldRef,
}: RoleAdvancedDetailsProps) {
  const provenanceKeys = Object.keys(provenance);
  return (
    <SettingsAdvancedFold open={open} onOpenChange={onOpenChange} padded data-testid={testId}>
      <RoleFallbackEditor
        role={role}
        entries={entries}
        errors={errors}
        disabled={disabled}
        testId={`${testId}-fallback`}
        onAdd={onAdd}
        onRemove={onRemove}
        onUpdate={onUpdate}
        registerFieldRef={registerFieldRef}
      />
      {provenanceKeys.length > 0 ? (
        <div className="flex flex-col gap-1.5" data-testid={`${testId}-provenance`}>
          <span className="text-form-label font-medium text-muted">Config provenance</span>
          <div className="flex flex-wrap items-center gap-1.5">
            {provenanceKeys.map(key => (
              <SettingsProvChip key={key}>
                {key} · {provenance[key]}
              </SettingsProvChip>
            ))}
          </div>
        </div>
      ) : null}
    </SettingsAdvancedFold>
  );
}
