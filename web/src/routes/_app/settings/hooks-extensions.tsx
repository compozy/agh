import { AlertCircle, AlertTriangle, Check, Loader2, Puzzle, Webhook, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import {
  Alert,
  AlertAction,
  AlertDescription,
  Button,
  Empty,
  Input,
  Pill,
  NativeSelect,
  NativeSelectOption,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  cn,
  pillGroupSegmentVariants,
} from "@agh/ui";
import { useSettingsHooksExtensionsPage } from "@/hooks/routes/use-settings-hooks-extensions-page";
import type {
  SettingsExtensionEntry,
  SettingsHookEntry,
  SettingsHooksExtensionsSection,
  SettingsHooksExtensionsTransportParity,
} from "@/systems/settings";
import {
  SettingsFieldRow,
  SettingsNumberInput,
  SettingsPageActions,
  SettingsPageShell,
  SettingsRestartBanner,
  SettingsSectionCard,
  SettingsStatusLine,
} from "@/systems/settings/components";

export const Route = createFileRoute("/_app/settings/hooks-extensions")({
  component: HooksExtensionsSettingsPage,
});

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

function HooksExtensionsSettingsPage() {
  const page = useSettingsHooksExtensionsPage();

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-extensions-loading"
      >
        <Loader2 className="size-5 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (page.error || !page.envelope || !page.draft) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-extensions-error"
      >
        <div className="flex flex-col items-center gap-2 text-center">
          <AlertCircle className="size-6 text-[color:var(--color-danger)]" />
          <p className="text-sm text-[color:var(--color-text-tertiary)]">
            {page.error?.message ?? "Failed to load hooks & extensions settings"}
          </p>
          <Button onClick={page.handleRetry} size="sm" type="button" variant="outline">
            Retry
          </Button>
        </div>
      </div>
    );
  }

  const { draft, hooks, extensions, transportParity } = page;

  return (
    <SettingsPageShell
      slug="hooks-extensions"
      title="Hooks & Extensions"
      statusLine={
        <SettingsStatusLine
          data-testid="settings-page-hooks-extensions-status-line"
          daemonAvailable
          items={[
            <span key="hooks" data-testid="settings-page-hooks-extensions-hooks-total">
              {page.hooksCounts.enabled}/{page.hooksCounts.total} hooks enabled
            </span>,
            <span key="extensions" data-testid="settings-page-hooks-extensions-extensions-total">
              {page.extensionsCounts.enabled}/{page.extensionsCounts.total} extensions enabled
            </span>,
          ]}
        />
      }
      actions={<SettingsPageActions slug="hooks-extensions" restart={page.restart} />}
      banner={<SettingsRestartBanner slug="hooks-extensions" restart={page.restart} />}
    >
      {page.lastAction ? (
        <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <TransportParityBanner parity={transportParity} />

      <HooksSection
        hooks={hooks}
        pendingHookName={page.pendingHookName}
        hookError={page.hookError}
        canMutate={page.canMutateHooks}
        onToggle={page.toggleHookEnabled}
      />

      <ExtensionsSection
        extensions={extensions}
        pendingExtensionName={page.pendingExtensionName}
        error={page.extensionActionError ?? page.extensionsError}
        isLoading={page.extensionsLoading}
        canMutate={page.canMutateExtensions}
        onToggle={page.toggleExtensionEnabled}
      />

      <PolicySection
        draft={draft}
        setDraft={value =>
          page.updatePolicyDraft(current => (typeof value === "function" ? value(current) : value))
        }
        onToggleAllowedKind={page.toggleAllowedKind}
        isDirty={page.isPolicyDirty}
        isSaving={page.isSavingPolicy}
        error={page.savePolicyError}
        warnings={page.policyWarnings}
        canMutate={page.canMutatePolicy}
        onSave={page.handleSavePolicy}
        onReset={page.handleResetPolicy}
      />
    </SettingsPageShell>
  );
}

function TransportParityBanner({
  parity,
}: {
  parity: SettingsHooksExtensionsTransportParity | null;
}) {
  if (!parity || !parity.known) return null;
  if (parity.extensions_http && parity.settings_http) return null;
  const unavailable = describeUnavailableHttpOperations(parity);

  return (
    <Alert
      variant="warning"
      role="status"
      data-testid="settings-page-hooks-extensions-transport-parity"
    >
      <AlertTriangle className="size-3.5" />
      <AlertDescription className="text-xs">
        <span className="font-medium text-[color:var(--color-warning)]">
          Some operations are unavailable over HTTP.
        </span>{" "}
        HTTP is bound outside the loopback host. {unavailable} stay available over UDS but return
        403 on HTTP. Use the CLI or rebind to loopback to edit from the web app.
      </AlertDescription>
    </Alert>
  );
}

function describeUnavailableHttpOperations(parity: SettingsHooksExtensionsTransportParity): string {
  const operations: string[] = [];
  if (parity.settings_http === false) {
    operations.push("Hook toggles and policy edits");
  }
  if (parity.extensions_http === false) {
    operations.push("Extension enable/disable");
  }

  if (operations.length === 0) return "These operations";
  if (operations.length === 1) return operations[0];
  return `${operations.slice(0, -1).join(", ")} and ${operations.at(-1)}`;
}

interface HooksSectionProps {
  hooks: SettingsHookEntry[];
  pendingHookName: string | null;
  hookError: string | null;
  canMutate: boolean;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}

function HooksSection({
  hooks,
  pendingHookName,
  hookError,
  canMutate,
  onToggle,
}: HooksSectionProps) {
  return (
    <SettingsSectionCard
      data-testid="settings-page-hooks-extensions-hooks-section"
      eyebrow="Lifecycle hooks"
      note="restart required to re-read declarations · toggles persist now"
    >
      {hookError ? (
        <span
          className="text-xs text-[color:var(--color-danger)]"
          data-testid="settings-page-hooks-extensions-hooks-error"
        >
          {hookError}
        </span>
      ) : null}
      {hooks.length === 0 ? (
        <Empty
          icon={Webhook}
          title="No hooks registered"
          description="Add a hook declaration to ~/.agh/config.toml or a workspace overlay to register one."
          data-testid="settings-page-hooks-extensions-hooks-empty"
        />
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
          data-testid="settings-page-hooks-extensions-hooks-list"
        >
          <Table>
            <TableHeader>
              <TableRow className="bg-[color:var(--color-surface-elevated)]">
                <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                  Name
                </TableHead>
                <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                  Event
                </TableHead>
                <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                  Mode
                </TableHead>
                <TableHead className="text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                  Matcher
                </TableHead>
                <TableHead className="w-[1%] text-right text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
                  Enabled
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {hooks.map(entry => (
                <HookRow
                  key={entry.name}
                  entry={entry}
                  pending={pendingHookName === entry.name}
                  canMutate={canMutate}
                  onToggle={onToggle}
                />
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </SettingsSectionCard>
  );
}

function HookRow({
  entry,
  pending,
  canMutate,
  onToggle,
}: {
  entry: SettingsHookEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}) {
  const declaration = entry.declaration;
  const enabled = declaration.required !== false;
  const matcherSummary = summarizeMatcher(declaration.matcher);
  const mode = declaration.mode === "sync" ? "blocking" : (declaration.mode ?? "async");

  return (
    <TableRow data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}`}>
      <TableCell>
        <div className="flex flex-col gap-0.5">
          <span className="font-mono text-sm text-[color:var(--color-text-primary)]">
            {entry.name}
          </span>
          {declaration.command ? (
            <span className="font-mono text-[0.62rem] text-[color:var(--color-text-tertiary)]">
              {[declaration.command, ...(declaration.args ?? [])].join(" ")}
            </span>
          ) : null}
        </div>
      </TableCell>
      <TableCell>
        <Pill mono tone="info">
          {declaration.event}
        </Pill>
      </TableCell>
      <TableCell className="font-mono text-xs text-[color:var(--color-text-secondary)]">
        {mode}
      </TableCell>
      <TableCell
        className="font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-matcher`}
      >
        {matcherSummary || "—"}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          {pending ? (
            <Loader2 className="size-3.5 animate-spin text-[color:var(--color-text-tertiary)]" />
          ) : null}
          <Switch
            data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-toggle`}
            checked={enabled}
            disabled={pending || !canMutate}
            onCheckedChange={checked => onToggle(entry, checked)}
            aria-label={`Toggle hook ${entry.name}`}
          />
        </div>
      </TableCell>
    </TableRow>
  );
}

function summarizeMatcher(matcher: SettingsHookEntry["declaration"]["matcher"]): string {
  const entries = Object.entries(matcher).filter(
    ([, value]) => value !== undefined && value !== null && value !== ""
  );
  if (entries.length === 0) return "";
  return entries.map(([key, value]) => `${key}=${String(value)}`).join(" · ");
}

interface ExtensionsSectionProps {
  extensions: SettingsExtensionEntry[];
  pendingExtensionName: string | null;
  error: string | null;
  isLoading: boolean;
  canMutate: boolean;
  onToggle: (entry: SettingsExtensionEntry, nextEnabled: boolean) => void;
}

function ExtensionsSection({
  extensions,
  pendingExtensionName,
  error,
  isLoading,
  canMutate,
  onToggle,
}: ExtensionsSectionProps) {
  return (
    <SettingsSectionCard
      data-testid="settings-page-hooks-extensions-extensions-section"
      eyebrow="Installed extensions"
      note="toggles apply immediately · no restart"
    >
      {error ? (
        <span
          className="text-xs text-[color:var(--color-danger)]"
          data-testid="settings-page-hooks-extensions-extensions-error"
        >
          {error}
        </span>
      ) : null}
      {isLoading && extensions.length === 0 ? (
        <div
          className="flex items-center gap-2 text-xs text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-hooks-extensions-extensions-loading"
        >
          <Loader2 className="size-3.5 animate-spin" />
          Loading extensions…
        </div>
      ) : extensions.length === 0 ? (
        <Empty
          icon={Puzzle}
          title="No extensions installed"
          description="Install an extension with `agh extensions install` to see it here."
          data-testid="settings-page-hooks-extensions-extensions-empty"
        />
      ) : (
        <ul
          className="flex flex-col gap-2"
          data-testid="settings-page-hooks-extensions-extensions-list"
        >
          {extensions.map(entry => (
            <ExtensionRow
              key={entry.name}
              entry={entry}
              pending={pendingExtensionName === entry.name}
              canMutate={canMutate}
              onToggle={onToggle}
            />
          ))}
        </ul>
      )}
    </SettingsSectionCard>
  );
}

function ExtensionRow({
  entry,
  pending,
  canMutate,
  onToggle,
}: {
  entry: SettingsExtensionEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (entry: SettingsExtensionEntry, nextEnabled: boolean) => void;
}) {
  const healthTone: "success" | "warning" | "danger" | "neutral" =
    entry.health === "healthy"
      ? "success"
      : entry.health === "degraded"
        ? "warning"
        : entry.health === "unhealthy"
          ? "danger"
          : "neutral";
  const missingEnv = entry.missing_env ?? [];

  return (
    <li
      className="flex items-center justify-between gap-3 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-2"
      data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}`}
    >
      <div className="flex min-w-0 items-center gap-3">
        <Pill.Dot tone={healthTone} size="md" pulse={entry.health === "degraded"} />
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate font-mono text-sm text-[color:var(--color-text-primary)]">
            {entry.name}
          </span>
          <span className="flex flex-wrap items-center gap-1.5 font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
            <span>{entry.state || (entry.enabled ? "running" : "stopped")}</span>
            {entry.version ? (
              <Pill mono tone="neutral">
                v{entry.version}
              </Pill>
            ) : null}
            {entry.health ? (
              <Pill mono tone={healthTone}>
                {entry.health}
              </Pill>
            ) : null}
            {missingEnv.length > 0 ? (
              <Pill mono tone="warning">
                env missing
              </Pill>
            ) : null}
          </span>
          {entry.last_error ? (
            <span
              className="text-[0.62rem] text-[color:var(--color-danger)]"
              data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-error`}
            >
              {entry.last_error}
            </span>
          ) : null}
          {missingEnv.length > 0 ? (
            <span
              className="max-w-full break-all font-mono text-[0.62rem] text-[color:var(--color-warning)]"
              data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-missing-env`}
            >
              Missing env: {missingEnv.join(", ")}
            </span>
          ) : null}
        </div>
      </div>
      <div className="flex items-center gap-2">
        {pending ? (
          <Loader2 className="size-3.5 animate-spin text-[color:var(--color-text-tertiary)]" />
        ) : null}
        <Switch
          data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-toggle`}
          checked={entry.enabled}
          disabled={pending || !canMutate}
          onCheckedChange={checked => onToggle(entry, checked)}
          aria-label={`Toggle extension ${entry.name}`}
        />
      </div>
    </li>
  );
}

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

function PolicySection({
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
  const setValidationError = useCallback(
    (key: string) => (message: string | null) => {
      setValidationErrors(current =>
        current[key] === message ? current : { ...current, [key]: message }
      );
    },
    []
  );
  const isInvalid = useMemo(
    () => Object.values(validationErrors).some(message => message !== null),
    [validationErrors]
  );

  return (
    <SettingsSectionCard
      data-testid="settings-page-hooks-extensions-policy-section"
      eyebrow="Extensions policy"
      note="restart required to apply"
      headerAction={
        <SaveControls
          isDirty={isDirty}
          isSaving={isSaving}
          isInvalid={isInvalid}
          canMutate={canMutate}
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
              setDraft({
                ...draft,
                marketplace: { ...draft.marketplace, registry: event.target.value },
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
              setDraft({
                ...draft,
                marketplace: { ...draft.marketplace, base_url: event.target.value },
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
              setDraft({
                ...draft,
                resources: {
                  ...draft.resources,
                  max_scope: event.target.value as PolicyConfig["resources"]["max_scope"],
                },
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
          setDraft({
            ...draft,
            resources: { ...draft.resources, snapshot_rate_limit: next },
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
          setDraft({
            ...draft,
            resources: { ...draft.resources, operator_write_rate_limit: next },
          })
        }
      />
    </SettingsSectionCard>
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
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
            per
          </span>
          <Input
            className="w-20 font-mono"
            data-testid={`${testId}-window`}
            value={value.window}
            placeholder="5m"
            disabled={!canMutate}
            onChange={event => onChange({ ...value, window: event.target.value })}
          />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
            queue
          </span>
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
  isDirty: boolean;
  isSaving: boolean;
  isInvalid: boolean;
  canMutate: boolean;
  error: string | null;
  warnings?: string[];
  onSave: () => void;
  onReset: () => void;
}

function SaveControls({
  isDirty,
  isSaving,
  isInvalid,
  canMutate,
  error,
  warnings,
  onSave,
  onReset,
}: SaveControlsProps) {
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
            className="text-xs text-[color:var(--color-danger)]"
            data-testid="settings-page-hooks-extensions-policy-error"
          >
            {error}
          </span>
        ) : warnings && warnings.length > 0 ? (
          <span
            className="text-xs text-[color:var(--color-warning)]"
            data-testid="settings-page-hooks-extensions-policy-warning"
          >
            {warnings.join(" · ")}
          </span>
        ) : !canMutate ? (
          <span
            className="text-xs text-[color:var(--color-warning)]"
            data-testid="settings-page-hooks-extensions-policy-unavailable"
          >
            Policy edits are unavailable over HTTP
          </span>
        ) : isInvalid ? (
          <span
            className="text-xs text-[color:var(--color-warning)]"
            data-testid="settings-page-hooks-extensions-policy-invalid"
          >
            Resolve validation errors before saving
          </span>
        ) : isDirty ? (
          <span
            className="text-xs text-[color:var(--color-text-tertiary)]"
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
        {isSaving ? <Loader2 className="size-3.5 animate-spin" /> : null}
        {isSaving ? "Saving…" : "Save policy"}
      </Button>
    </div>
  );
}

function ActionResultBanner({
  action,
  onDismiss,
}: {
  action: NonNullable<ReturnType<typeof useSettingsHooksExtensionsPage>["lastAction"]>;
  onDismiss: () => void;
}) {
  const { message, tone } = describeAction(action);
  return (
    <Alert
      variant={tone === "success" ? "success" : "info"}
      role="status"
      data-testid="settings-page-hooks-extensions-action-result"
      data-kind={action.kind}
    >
      <Check className="size-3.5" />
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="settings-page-hooks-extensions-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}

function describeAction(
  action: NonNullable<ReturnType<typeof useSettingsHooksExtensionsPage>["lastAction"]>
): { message: string; tone: "success" | "info" } {
  if (action.kind === "saved") {
    const restartBadge = action.result.restart_required
      ? "restart required to apply"
      : "applied immediately";
    return { message: `Policy saved · ${restartBadge}.`, tone: "success" };
  }
  if (action.kind === "hook-toggled") {
    const state = action.enabled ? "enabled" : "disabled";
    const restartBadge = action.result.restart_required
      ? "restart required to reload"
      : "applied immediately";
    return {
      message: `Hook "${action.name}" ${state} · ${restartBadge}.`,
      tone: "success",
    };
  }
  const state = action.enabled ? "enabled" : "disabled";
  return {
    message: `Extension "${action.name}" ${state} · applied immediately.`,
    tone: "info",
  };
}
