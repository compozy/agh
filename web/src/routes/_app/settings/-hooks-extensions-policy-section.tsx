import { useState, type Dispatch, type SetStateAction } from "react";

import {
  SettingsFieldRow,
  SettingsNumberInput,
  type SettingsHooksExtensionsSection,
} from "@/systems/settings";
import {
  Button,
  Eyebrow,
  Input,
  NativeSelect,
  NativeSelectOption,
  Section,
  Spinner,
  cn,
  pillGroupSegmentVariants,
} from "@agh/ui";

type PolicyConfig = SettingsHooksExtensionsSection["config"];

const ALLOWED_KIND_OPTIONS = [
  "snapshot",
  "artifact",
  "memory",
  "transcript",
  "session",
  "workspace",
  "global",
];

const MAX_SCOPE_OPTIONS = ["session", "workspace", "global"] as const;

interface PolicySectionProps {
  draft: PolicyConfig;
  setDraft: Dispatch<SetStateAction<PolicyConfig>>;
  onToggleAllowedKind: (kind: string) => void;
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  canMutate: boolean;
  onSave: () => void;
  onReset: () => void;
}

export function PolicySection({
  draft,
  setDraft,
  onToggleAllowedKind,
  isDirty,
  isSaving,
  error,
  warnings,
  canMutate,
  onSave,
  onReset,
}: PolicySectionProps) {
  const [validationErrors, setValidationErrors] = useState<Record<string, string | null>>({});
  const setValidationError = (key: string) => (message: string | null) => {
    setValidationErrors(current =>
      current[key] === message ? current : { ...current, [key]: message }
    );
  };
  const isInvalid = Object.values(validationErrors).some(message => message !== null);

  return (
    <Section
      data-testid="settings-page-hooks-extensions-policy-section"
      label="Extensions policy"
      note="restart required to apply"
      right={
        <SaveControls
          state={{ isDirty, isSaving, isInvalid, canMutate }}
          error={error}
          warnings={warnings}
          onSave={onSave}
          onReset={onReset}
        />
      }
    >
      <SettingsFieldRow
        data-testid="settings-page-hooks-extensions-policy-registry"
        label="Marketplace registry"
        description="Identifier of the marketplace publisher"
        hint="CONFIG.TOML"
        control={
          <Input
            className="w-56"
            data-testid="settings-page-hooks-extensions-policy-registry-input"
            value={draft.marketplace.registry ?? ""}
            disabled={!canMutate}
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  marketplace: { ...current.marketplace, registry: event.target.value },
                };
              })
            }
          />
        }
      />
      <SettingsFieldRow
        data-testid="settings-page-hooks-extensions-policy-base-url"
        label="Base URL"
        description="Override the registry's default endpoint"
        hint="OPTIONAL"
        control={
          <Input
            className="w-72 font-mono"
            data-testid="settings-page-hooks-extensions-policy-base-url-input"
            value={draft.marketplace.base_url ?? ""}
            placeholder="https://"
            disabled={!canMutate}
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  marketplace: { ...current.marketplace, base_url: event.target.value },
                };
              })
            }
          />
        }
      />
      <AllowedKindsField
        selected={draft.resources.allowed_kinds ?? []}
        disabled={!canMutate}
        onToggle={onToggleAllowedKind}
      />
      <SettingsFieldRow
        data-testid="settings-page-hooks-extensions-policy-max-scope"
        label="Max scope"
        description="Broadest scope an extension may claim"
        hint="SCOPE"
        control={
          <NativeSelect
            className="w-40 font-mono"
            data-testid="settings-page-hooks-extensions-policy-max-scope-input"
            value={draft.resources.max_scope ?? "workspace"}
            disabled={!canMutate}
            onChange={event =>
              setDraft(prev => {
                const current = prev ?? draft;
                return {
                  ...current,
                  resources: {
                    ...current.resources,
                    max_scope: event.target.value as PolicyConfig["resources"]["max_scope"],
                  },
                };
              })
            }
          >
            {MAX_SCOPE_OPTIONS.map(option => (
              <NativeSelectOption key={option} value={option}>
                {option}
              </NativeSelectOption>
            ))}
          </NativeSelect>
        }
      />
      <RateLimitRow
        testId="settings-page-hooks-extensions-policy-snapshot-rate"
        label="Snapshot rate limit"
        description="Published snapshots per window (queue = burst)"
        value={draft.resources.snapshot_rate_limit}
        errorMessage={combineErrorMessages(
          validationErrors.snapshotRateRequests,
          validationErrors.snapshotRateQueue
        )}
        canMutate={canMutate}
        onRequestsValidityChange={setValidationError("snapshotRateRequests")}
        onQueueValidityChange={setValidationError("snapshotRateQueue")}
        onChange={next =>
          setDraft(prev => {
            const current = prev ?? draft;
            return {
              ...current,
              resources: { ...current.resources, snapshot_rate_limit: next },
            };
          })
        }
      />
      <RateLimitRow
        testId="settings-page-hooks-extensions-policy-operator-rate"
        label="Operator write rate limit"
        description="Operator writes per window (queue = burst)"
        value={draft.resources.operator_write_rate_limit}
        errorMessage={combineErrorMessages(
          validationErrors.operatorRateRequests,
          validationErrors.operatorRateQueue
        )}
        canMutate={canMutate}
        onRequestsValidityChange={setValidationError("operatorRateRequests")}
        onQueueValidityChange={setValidationError("operatorRateQueue")}
        onChange={next =>
          setDraft(prev => {
            const current = prev ?? draft;
            return {
              ...current,
              resources: { ...current.resources, operator_write_rate_limit: next },
            };
          })
        }
      />
    </Section>
  );
}

function AllowedKindsField({
  selected,
  disabled,
  onToggle,
}: {
  selected: string[];
  disabled: boolean;
  onToggle: (kind: string) => void;
}) {
  return (
    <SettingsFieldRow
      data-testid="settings-page-hooks-extensions-policy-allowed-kinds"
      label="Allowed kinds"
      description="Marketplace resource kinds extensions may publish"
      hint={`${selected.length}/${ALLOWED_KIND_OPTIONS.length} selected`}
      control={
        <div
          className="flex max-w-md flex-wrap items-center gap-1.5"
          data-testid="settings-page-hooks-extensions-policy-allowed-kinds-list"
        >
          {ALLOWED_KIND_OPTIONS.map(kind => {
            const active = selected.includes(kind);
            return (
              <button
                key={kind}
                type="button"
                disabled={disabled}
                onClick={() => onToggle(kind)}
                data-testid={`settings-page-hooks-extensions-policy-allowed-kinds-${kind}`}
                data-active={active ? "true" : "false"}
                aria-pressed={active}
                className={cn(pillGroupSegmentVariants({ active, size: "sm" }))}
              >
                {kind}
              </button>
            );
          })}
        </div>
      }
    />
  );
}

type RateLimit = PolicyConfig["resources"]["snapshot_rate_limit"];

function combineErrorMessages(...messages: Array<string | null | undefined>): string | undefined {
  const visible = messages.filter(Boolean);
  return visible.length > 0 ? visible.join(" ") : undefined;
}

function RateLimitRow({
  testId,
  label,
  description,
  value,
  errorMessage,
  canMutate,
  onRequestsValidityChange,
  onQueueValidityChange,
  onChange,
}: {
  testId: string;
  label: string;
  description: string;
  value: RateLimit;
  errorMessage?: string;
  canMutate: boolean;
  onRequestsValidityChange: (message: string | null) => void;
  onQueueValidityChange: (message: string | null) => void;
  onChange: (next: RateLimit) => void;
}) {
  return (
    <SettingsFieldRow
      data-testid={testId}
      label={label}
      description={description}
      error={errorMessage}
      hint="LIMIT"
      control={
        <div className="flex max-w-full flex-wrap items-center gap-1.5">
          <SettingsNumberInput
            min={0}
            className="w-16 font-mono"
            data-testid={`${testId}-requests`}
            value={value.requests}
            placeholder="reqs"
            disabled={!canMutate}
            onValidityChange={onRequestsValidityChange}
            onValueChange={next => onChange({ ...value, requests: next })}
          />
          <Eyebrow className="text-muted">per</Eyebrow>
          <Input
            className="w-20 font-mono"
            data-testid={`${testId}-window`}
            value={value.window}
            placeholder="5m"
            disabled={!canMutate}
            onChange={event => onChange({ ...value, window: event.target.value })}
          />
          <Eyebrow className="text-muted">queue</Eyebrow>
          <SettingsNumberInput
            min={0}
            className="w-16 font-mono"
            data-testid={`${testId}-queue`}
            value={value.queue}
            placeholder="queue"
            disabled={!canMutate}
            onValidityChange={onQueueValidityChange}
            onValueChange={next => onChange({ ...value, queue: next })}
          />
        </div>
      }
    />
  );
}

interface SaveControlsProps {
  state: {
    isDirty: boolean;
    isSaving: boolean;
    isInvalid: boolean;
    canMutate: boolean;
  };
  error: string | null;
  warnings?: string[];
  onSave: () => void;
  onReset: () => void;
}

function SaveControls({ state, error, warnings, onSave, onReset }: SaveControlsProps) {
  const { isDirty, isSaving, isInvalid, canMutate } = state;
  const disabled = !isDirty || isSaving || isInvalid || !canMutate;
  return (
    <div
      className="flex max-w-full flex-wrap items-center gap-2"
      data-testid="settings-page-hooks-extensions-policy-controls"
      data-dirty={isDirty ? "true" : "false"}
    >
      <div className="min-w-0" role="status" aria-live={error ? "assertive" : "polite"}>
        {error ? (
          <span
            className="text-xs text-danger"
            data-testid="settings-page-hooks-extensions-policy-error"
          >
            {error}
          </span>
        ) : warnings && warnings.length > 0 ? (
          <span
            className="text-xs text-warning"
            data-testid="settings-page-hooks-extensions-policy-warning"
          >
            {warnings.join(" · ")}
          </span>
        ) : !canMutate ? (
          <span
            className="text-xs text-warning"
            data-testid="settings-page-hooks-extensions-policy-unavailable"
          >
            Policy edits are unavailable over HTTP
          </span>
        ) : isInvalid ? (
          <span
            className="text-xs text-warning"
            data-testid="settings-page-hooks-extensions-policy-invalid"
          >
            Resolve validation errors before saving
          </span>
        ) : isDirty ? (
          <span
            className="text-xs text-subtle"
            data-testid="settings-page-hooks-extensions-policy-dirty"
          >
            Unsaved changes
          </span>
        ) : null}
      </div>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onReset}
        disabled={!isDirty || isSaving}
        data-testid="settings-page-hooks-extensions-policy-reset"
      >
        Discard
      </Button>
      <Button
        type="button"
        variant="default"
        size="sm"
        onClick={onSave}
        disabled={disabled}
        data-testid="settings-page-hooks-extensions-policy-save"
      >
        {isSaving ? <Spinner className="size-3" /> : null}
        {isSaving ? "Saving…" : "Save policy"}
      </Button>
    </div>
  );
}
