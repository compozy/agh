import { Eyebrow, Input, NativeSelect, NativeSelectOption, RadioCard } from "@agh/ui";

import type { MCPSecretBinding } from "../lib/mcp-editor-model";

export interface MCPSecretBindingControlProps {
  binding: MCPSecretBinding;
  /** Existing `vault:mcp/**` refs offered by "Use Vault". */
  vaultRefs: string[];
  idPrefix: string;
  valueLabel?: string;
  onChange: (binding: MCPSecretBinding) => void;
}

/**
 * Hybrid secret binding consistent with the task-06 modal vocabulary: write a
 * new value (write-only, the daemon returns the computed canonical ref) OR select
 * an existing Vault ref. Plaintext is never reflected.
 */
export function MCPSecretBindingControl({
  binding,
  vaultRefs,
  idPrefix,
  valueLabel = "New secret value",
  onChange,
}: MCPSecretBindingControlProps) {
  const refs =
    binding.existingRef && !vaultRefs.includes(binding.existingRef)
      ? [binding.existingRef, ...vaultRefs]
      : vaultRefs;

  return (
    <div>
      <div className="grid grid-cols-2 gap-1.5">
        <RadioCard
          selected={binding.mode === "typed"}
          title="Write a new value"
          description="Creates its canonical Vault ref"
          onSelect={() => onChange({ ...binding, mode: "typed" })}
          data-testid={`${idPrefix}-mode-typed`}
        />
        <RadioCard
          selected={binding.mode === "ref"}
          title="Use Vault"
          description="Select an existing credential"
          onSelect={() =>
            onChange({
              ...binding,
              mode: "ref",
              vaultRef: binding.vaultRef || binding.existingRef || refs[0] || "",
            })
          }
          data-testid={`${idPrefix}-mode-ref`}
        />
      </div>

      {binding.mode === "typed" ? (
        <div className="mt-2">
          <label
            className="mb-1.5 flex items-center gap-2 text-form-label font-medium text-fg"
            htmlFor={`${idPrefix}-typed`}
          >
            {valueLabel}
            <Eyebrow className="ml-auto text-muted">write-only</Eyebrow>
          </label>
          <Input
            id={`${idPrefix}-typed`}
            type="password"
            autoComplete="new-password"
            className="font-mono"
            value={binding.typedValue}
            placeholder="Enter replacement value"
            onChange={event => onChange({ ...binding, typedValue: event.target.value })}
            data-testid={`${idPrefix}-typed-input`}
          />
          <p className="mt-1.5 text-caption text-muted">
            AGH stores this value on save and returns the computed canonical Vault ref. The value is
            never reflected.
          </p>
        </div>
      ) : (
        <div className="mt-2">
          <label
            className="mb-1.5 flex items-center gap-2 text-form-label font-medium text-fg"
            htmlFor={`${idPrefix}-ref`}
          >
            Existing Vault ref
          </label>
          {refs.length > 0 ? (
            <NativeSelect
              id={`${idPrefix}-ref`}
              className="font-mono"
              value={binding.vaultRef || binding.existingRef || refs[0]}
              onChange={event => onChange({ ...binding, vaultRef: event.target.value })}
              data-testid={`${idPrefix}-ref-select`}
            >
              {refs.map(ref => (
                <NativeSelectOption key={ref} value={ref}>
                  {ref}
                </NativeSelectOption>
              ))}
            </NativeSelect>
          ) : (
            <p className="text-caption text-muted" data-testid={`${idPrefix}-ref-empty`}>
              No existing Vault refs in the mcp namespace for this scope.
            </p>
          )}
          <p className="mt-1.5 text-caption text-muted">
            This path selects an existing credential only. Choose Write a new value to create a new
            canonical binding.
          </p>
        </div>
      )}
    </div>
  );
}
