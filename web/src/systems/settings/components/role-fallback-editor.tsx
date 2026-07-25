import type { ComponentPropsWithoutRef } from "react";

import { Plus, Trash2 } from "lucide-react";

import { Button, cn, Input, NativeSelect, NativeSelectOption } from "@agh/ui";

import { useLocalRowKeys } from "@/hooks/use-local-row-keys";

import { REASONING_OPTIONS } from "../lib/roles-config";
import { fallbackFieldId } from "../lib/roles-validation";
import type { RoleFallbackEntry, RoleName } from "../types";

interface RoleFallbackEditorProps extends Omit<ComponentPropsWithoutRef<"div">, "role"> {
  role: RoleName;
  entries: readonly RoleFallbackEntry[];
  errors: Record<string, string>;
  disabled?: boolean;
  testId: string;
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, field: keyof RoleFallbackEntry, value: string) => void;
  registerFieldRef: (id: string) => (element: HTMLElement | null) => void;
}

interface FallbackTextFieldProps extends Omit<ComponentPropsWithoutRef<"div">, "id" | "onChange"> {
  id: string;
  label: string;
  value: string;
  placeholder: string;
  error?: string;
  onChange: (value: string) => void;
  registerRef: (element: HTMLElement | null) => void;
}

function FallbackTextField({
  id,
  label,
  value,
  placeholder,
  error,
  onChange,
  registerRef,
  className,
  ...rootProps
}: FallbackTextFieldProps) {
  const errorId = error ? `${id}-error` : undefined;
  return (
    <div {...rootProps} className={cn("flex min-w-40 flex-1 flex-col gap-1", className)}>
      <label className="text-form-label font-medium text-muted" htmlFor={id}>
        {label}
      </label>
      <Input
        id={id}
        ref={registerRef}
        className="font-mono"
        data-testid={`${id}-input`}
        value={value}
        placeholder={placeholder}
        aria-invalid={error ? true : undefined}
        aria-describedby={errorId}
        onChange={event => onChange(event.target.value)}
      />
      {error ? (
        <span className="text-form-hint text-danger" id={errorId} role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}

/**
 * Accessible fallback-chain array editor for one role. Add/remove routes and
 * edit provider, model, and reasoning effort per entry. Edits flow into the
 * draft `roles.<role>.fallback_chain` and submit with the full section.
 */
export function RoleFallbackEditor({
  role,
  entries,
  errors,
  disabled,
  testId,
  onAdd,
  onRemove,
  onUpdate,
  registerFieldRef,
  className,
  ...rootProps
}: RoleFallbackEditorProps) {
  // Stable client-side row keys so React keys editable rows without the array
  // index. Append/remove keep keys aligned; length mismatches (mount / external
  // load) are normalized by useLocalRowKeys without mutating refs during render.
  const rowKeys = useLocalRowKeys(entries, "fallback");

  const handleAddRoute = () => {
    rowKeys.append();
    onAdd();
  };
  const handleRemoveRoute = (index: number) => {
    rowKeys.remove(index);
    onRemove(index);
  };

  return (
    <div {...rootProps} className={cn("flex flex-col gap-3", className)} data-testid={testId}>
      <div className="flex items-start justify-between gap-3">
        <div className="flex flex-col gap-0.5">
          <span className="text-ws-name font-medium text-fg-strong">Fallback chain</span>
          <span className="text-form-label text-muted">
            Ordered routes tried before session acceptance when the primary fails.
          </span>
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={disabled}
          onClick={handleAddRoute}
          data-testid={`${testId}-add`}
        >
          <Plus className="size-3" aria-hidden="true" />
          Add route
        </Button>
      </div>
      {entries.length === 0 ? (
        <p className="text-form-label text-subtle" data-testid={`${testId}-empty`}>
          No fallback routes configured.
        </p>
      ) : (
        <ul className="flex flex-col gap-3" data-testid={`${testId}-list`}>
          {entries.map((entry, index) => {
            const providerId = fallbackFieldId(role, index, "provider");
            const modelId = fallbackFieldId(role, index, "model");
            const reasoningId = fallbackFieldId(role, index, "reasoning_effort");
            const reasoningError = errors[reasoningId];
            const reasoningErrorId = reasoningError ? `${reasoningId}-error` : undefined;
            return (
              <li
                key={rowKeys.keys[index]}
                className="rounded-md border border-line bg-canvas-soft p-3"
                data-testid={`${testId}-entry-${index}`}
              >
                <fieldset
                  className="m-0 flex flex-wrap items-start gap-2 border-0 p-0"
                  disabled={disabled}
                >
                  <legend className="sr-only">Fallback route {index + 1}</legend>
                  <FallbackTextField
                    id={providerId}
                    label="Provider"
                    value={entry.provider}
                    placeholder="anthropic"
                    error={errors[providerId]}
                    onChange={value => onUpdate(index, "provider", value)}
                    registerRef={registerFieldRef(providerId)}
                  />
                  <FallbackTextField
                    id={modelId}
                    label="Model"
                    value={entry.model}
                    placeholder="claude-haiku-4-5"
                    error={errors[modelId]}
                    onChange={value => onUpdate(index, "model", value)}
                    registerRef={registerFieldRef(modelId)}
                  />
                  <div className="flex min-w-32 flex-col gap-1">
                    <label className="text-form-label font-medium text-muted" htmlFor={reasoningId}>
                      Reasoning
                    </label>
                    <NativeSelect
                      id={reasoningId}
                      ref={registerFieldRef(reasoningId)}
                      data-testid={`${reasoningId}-input`}
                      value={entry.reasoning_effort}
                      aria-invalid={reasoningError ? true : undefined}
                      aria-describedby={reasoningErrorId}
                      onChange={event => onUpdate(index, "reasoning_effort", event.target.value)}
                    >
                      {REASONING_OPTIONS.map(option => (
                        <NativeSelectOption key={option.value} value={option.value}>
                          {option.label}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                    {reasoningError ? (
                      <span
                        className="text-form-hint text-danger"
                        id={reasoningErrorId}
                        role="alert"
                      >
                        {reasoningError}
                      </span>
                    ) : null}
                  </div>
                  <div className="flex shrink-0 flex-col gap-1">
                    <span aria-hidden="true" className="text-form-label font-medium">
                      &nbsp;
                    </span>
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      aria-label={`Remove fallback route ${index + 1}`}
                      data-testid={`${testId}-remove-${index}`}
                      onClick={() => handleRemoveRoute(index)}
                    >
                      <Trash2 className="size-3" aria-hidden="true" />
                    </Button>
                  </div>
                </fieldset>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
