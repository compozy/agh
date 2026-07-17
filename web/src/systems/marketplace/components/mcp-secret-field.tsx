import { Plus } from "lucide-react";

import {
  Button,
  Eyebrow,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  Input,
  Pill,
  PillGroup,
  RadioCard,
  Spinner,
} from "@agh/ui";

import { bindingValuePresent, type MCPEnvField, type MCPFieldBinding } from "./mcp-install-model";

interface MCPSecretFieldProps {
  binding: MCPFieldBinding;
  canonicalRef: string;
  createPending: boolean;
  field: MCPEnvField;
  vaultError?: string | null;
  vaultLoading: boolean;
  vaultSecrets: ReadonlyArray<{ present: boolean; ref: string }>;
  onChange: (next: MCPFieldBinding) => void;
  onCreate: () => void;
}

const BINDING_MODES = [
  { value: "typed" as const, label: "Enter value" },
  { value: "vault" as const, label: "Use Vault" },
];

function MCPSecretField({
  binding,
  canonicalRef,
  createPending,
  field,
  vaultError,
  vaultLoading,
  vaultSecrets,
  onChange,
  onCreate,
}: MCPSecretFieldProps) {
  const invalid = field.required && !bindingValuePresent(binding);
  const descriptionId = `mcp-field-${field.name}-description`;
  const errorId = `mcp-field-${field.name}-error`;

  return (
    <Field data-invalid={binding.touched && invalid ? "true" : undefined}>
      <div className="flex flex-col gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <FieldLabel className="font-mono" htmlFor={`mcp-field-${field.name}`}>
            {field.name}
          </FieldLabel>
          {field.secret ? (
            <Pill mono size="xs">
              secret
            </Pill>
          ) : null}
          {field.required ? (
            <Pill mono size="xs" tone="warning">
              required
            </Pill>
          ) : null}
        </div>
        <FieldDescription id={descriptionId}>
          {field.prompt ||
            (field.required ? "Required by this server." : "Optional configuration.")}
        </FieldDescription>
        {field.secret ? (
          <PillGroup
            aria-label={`${field.name} binding method`}
            className="grid w-full grid-cols-2"
            items={BINDING_MODES}
            onChange={mode => onChange({ ...binding, mode })}
            size="sm"
            value={binding.mode}
          />
        ) : null}
      </div>

      {!field.secret || binding.mode === "typed" ? (
        <Input
          aria-describedby={`${descriptionId}${binding.touched && invalid ? ` ${errorId}` : ""}`}
          aria-invalid={binding.touched && invalid ? true : undefined}
          autoComplete="off"
          id={`mcp-field-${field.name}`}
          onBlur={() => onChange({ ...binding, touched: true })}
          onChange={event => onChange({ ...binding, typedValue: event.target.value })}
          placeholder={field.secret ? "Stored write-only during install" : field.default}
          required={field.required}
          type={field.secret ? "password" : "text"}
          value={binding.typedValue}
        />
      ) : (
        <div
          className="flex flex-col gap-2.5 rounded-md bg-canvas px-3 py-3"
          data-testid={`mcp-vault-selector-${field.name}`}
        >
          {vaultLoading ? (
            <div className="flex items-center gap-2 text-small-body text-muted" role="status">
              <Spinner aria-hidden="true" className="size-3" />
              Loading Vault metadata
            </div>
          ) : null}
          {vaultError ? <p className="text-small-body text-danger">{vaultError}</p> : null}
          {!binding.createOpen ? (
            <>
              <div className="flex items-center gap-2">
                <Eyebrow className="text-muted">Choose existing secret</Eyebrow>
                <Pill className="ml-auto" mono size="xs">
                  namespace=mcp
                </Pill>
              </div>
              {!vaultLoading && !vaultError && vaultSecrets.length === 0 ? (
                <p className="text-small-body text-muted">No MCP secrets are stored yet.</p>
              ) : null}
              {vaultSecrets.length > 0 ? (
                <div
                  aria-label={`${field.name} Vault references`}
                  className="grid gap-1.5"
                  role="radiogroup"
                >
                  {vaultSecrets.map(secret => (
                    <RadioCard
                      badge={
                        <Pill mono size="xs" tone={secret.present ? "success" : "danger"}>
                          {secret.present ? "present" : "missing"}
                        </Pill>
                      }
                      className="gap-0 border border-line bg-input-fill px-2.5 py-2 data-[selected]:border-line-strong data-[selected]:bg-row-selected"
                      disabled={!secret.present}
                      key={secret.ref}
                      onSelect={() => onChange({ ...binding, vaultRef: secret.ref, touched: true })}
                      selected={binding.vaultRef === secret.ref}
                      title={
                        <code className="block break-all font-mono text-mono-id font-normal leading-snug tracking-normal text-fg">
                          {secret.ref}
                        </code>
                      }
                      titleClassName="overflow-visible whitespace-normal font-normal tracking-normal"
                    />
                  ))}
                </div>
              ) : null}
              <div className="flex items-center gap-2">
                <p className="mr-auto text-form-hint text-muted">Values stay hidden.</p>
                <Button
                  onClick={() => onChange({ ...binding, createOpen: true })}
                  size="sm"
                  type="button"
                  variant="outline"
                >
                  <Plus aria-hidden="true" className="size-3" />
                  Create Vault secret
                </Button>
              </div>
            </>
          ) : (
            <div className="flex flex-col gap-2">
              <Eyebrow className="text-muted">Create inline</Eyebrow>
              <code className="break-all font-mono text-form-hint text-muted">{canonicalRef}</code>
              <Input
                aria-label={`New Vault value for ${field.name}`}
                autoComplete="new-password"
                onChange={event =>
                  onChange({ ...binding, createError: undefined, createValue: event.target.value })
                }
                placeholder="Secret value"
                type="password"
                value={binding.createValue}
              />
              {binding.createError ? (
                <p className="text-small-body text-danger" role="alert">
                  {binding.createError}
                </p>
              ) : null}
              <div className="flex items-center gap-2">
                <Button
                  data-testid={`mcp-create-secret-${field.name}`}
                  disabled={createPending || binding.createValue.trim() === ""}
                  onClick={onCreate}
                  size="sm"
                  type="button"
                  variant="neutral"
                >
                  {createPending ? <Spinner aria-hidden="true" className="size-3" /> : null}
                  {createPending ? "Creating…" : "Create secret"}
                </Button>
                <Button
                  disabled={createPending}
                  onClick={() => onChange({ ...binding, createOpen: false })}
                  size="sm"
                  type="button"
                  variant="ghost"
                >
                  Cancel
                </Button>
              </div>
            </div>
          )}
        </div>
      )}

      {binding.touched && invalid ? (
        <FieldError id={errorId}>Choose a present Vault ref or enter a value.</FieldError>
      ) : null}
    </Field>
  );
}

export { MCPSecretField };
export type { MCPSecretFieldProps };
