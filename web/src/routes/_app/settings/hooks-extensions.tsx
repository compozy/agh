import { AlertCircle, AlertTriangle, Check, Loader2, Puzzle, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";
import type { Dispatch, SetStateAction } from "react";

import { Button } from "@agh/ui";
import { Switch } from "@/components/ui/switch";
import { useSettingsHooksExtensionsPage } from "@/hooks/routes/use-settings-hooks-extensions-page";
import type {
  SettingsExtensionEntry,
  SettingsHookEntry,
  SettingsHooksExtensionsSection,
  SettingsHooksExtensionsTransportParity,
} from "@/systems/settings";
import {
  SettingsFieldRow,
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

  return (
    <div
      className="flex items-start gap-3 rounded-md border border-[color:var(--color-warning)] bg-[color:var(--color-warning-tint)] px-3 py-2 text-xs text-[color:var(--color-warning)]"
      data-testid="settings-page-hooks-extensions-transport-parity"
      role="status"
    >
      <AlertTriangle className="mt-0.5 size-3.5 shrink-0" />
      <div className="flex flex-col gap-0.5">
        <span className="font-medium">Some operations are unavailable over HTTP</span>
        <span>
          HTTP is bound outside the loopback host. Extension enable/disable and policy edits stay
          available over UDS but return 403 on HTTP. Use the CLI or rebind to loopback to edit from
          the web app.
        </span>
      </div>
    </div>
  );
}

interface HooksSectionProps {
  hooks: SettingsHookEntry[];
  pendingHookName: string | null;
  hookError: string | null;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}

function HooksSection({ hooks, pendingHookName, hookError, onToggle }: HooksSectionProps) {
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
        <div
          className="rounded-md border border-dashed border-[color:var(--color-divider)] px-4 py-8 text-center text-sm text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-hooks-extensions-hooks-empty"
        >
          No hook declarations found in config. Add one to <code>~/.agh/config.toml</code> or a
          workspace overlay to register a hook.
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-[color:var(--color-divider)]"
          data-testid="settings-page-hooks-extensions-hooks-list"
        >
          <table className="w-full border-collapse text-sm">
            <thead className="bg-[color:var(--color-surface-elevated)] text-[0.6rem] uppercase tracking-[0.18em] text-[color:var(--color-text-label)]">
              <tr>
                <th className="px-4 py-2.5 text-left">Name</th>
                <th className="px-4 py-2.5 text-left">Event</th>
                <th className="px-4 py-2.5 text-left">Mode</th>
                <th className="px-4 py-2.5 text-left">Matcher</th>
                <th className="px-4 py-2.5 text-right">Enabled</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[color:var(--color-divider)]">
              {hooks.map(entry => (
                <HookRow
                  key={entry.name}
                  entry={entry}
                  pending={pendingHookName === entry.name}
                  onToggle={onToggle}
                />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </SettingsSectionCard>
  );
}

function HookRow({
  entry,
  pending,
  onToggle,
}: {
  entry: SettingsHookEntry;
  pending: boolean;
  onToggle: (entry: SettingsHookEntry, nextEnabled: boolean) => void;
}) {
  const declaration = entry.declaration;
  const enabled = declaration.required !== false;
  const matcherSummary = summarizeMatcher(declaration.matcher);
  const mode = declaration.mode === "sync" ? "blocking" : (declaration.mode ?? "async");

  return (
    <tr data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}`}>
      <td className="px-4 py-3">
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
      </td>
      <td className="px-4 py-3">
        <span className="font-mono text-xs text-[color:var(--color-text-primary)]">
          {declaration.event}
        </span>
      </td>
      <td className="px-4 py-3 font-mono text-xs text-[color:var(--color-text-secondary)]">
        {mode}
      </td>
      <td
        className="px-4 py-3 font-mono text-xs text-[color:var(--color-text-secondary)]"
        data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-matcher`}
      >
        {matcherSummary || "—"}
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center justify-end gap-2">
          {pending ? (
            <Loader2 className="size-3.5 animate-spin text-[color:var(--color-text-tertiary)]" />
          ) : null}
          <Switch
            data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-toggle`}
            checked={enabled}
            disabled={pending}
            onCheckedChange={checked => onToggle(entry, checked)}
            aria-label={`Toggle hook ${entry.name}`}
          />
        </div>
      </td>
    </tr>
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
        <div
          className="rounded-md border border-dashed border-[color:var(--color-divider)] px-4 py-8 text-center text-sm text-[color:var(--color-text-tertiary)]"
          data-testid="settings-page-hooks-extensions-extensions-empty"
        >
          No extensions installed. Install an extension with <code>agh extensions install</code> to
          see it here.
        </div>
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
  const healthColor =
    entry.health === "healthy"
      ? "text-[color:var(--color-success)]"
      : entry.health === "degraded"
        ? "text-[color:var(--color-warning)]"
        : entry.health === "unhealthy"
          ? "text-[color:var(--color-danger)]"
          : "text-[color:var(--color-text-tertiary)]";

  return (
    <li
      className="flex items-center justify-between gap-3 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-3 py-2"
      data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}`}
    >
      <div className="flex min-w-0 items-center gap-3">
        <Puzzle className="size-3.5 shrink-0 text-[color:var(--color-text-tertiary)]" />
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate font-mono text-sm text-[color:var(--color-text-primary)]">
            {entry.name}
          </span>
          <span className="font-mono text-[0.6rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)]">
            {entry.state || (entry.enabled ? "running" : "stopped")}
            {entry.version ? ` · v${entry.version}` : ""}
            {entry.health ? (
              <>
                {" · "}
                <span className={healthColor}>{entry.health}</span>
              </>
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
  onSave,
  onReset,
}: PolicySectionProps) {
  return (
    <SettingsSectionCard
      data-testid="settings-page-hooks-extensions-policy-section"
      eyebrow="Extensions policy"
      note="restart required to apply"
      headerAction={
        <SaveControls
          isDirty={isDirty}
          isSaving={isSaving}
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
          <input
            className="h-8 w-56 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-hooks-extensions-policy-registry-input"
            value={draft.marketplace.registry ?? ""}
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
          <input
            className="h-8 w-72 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-hooks-extensions-policy-base-url-input"
            value={draft.marketplace.base_url ?? ""}
            placeholder="https://"
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
        onToggle={onToggleAllowedKind}
      />
      <SettingsFieldRow
        data-testid="settings-page-hooks-extensions-policy-max-scope"
        label="Max scope"
        description="Broadest scope an extension may claim"
        hint="SCOPE"
        control={
          <select
            className="h-8 w-40 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-sm text-[color:var(--color-text-primary)]"
            data-testid="settings-page-hooks-extensions-policy-max-scope-input"
            value={draft.resources.max_scope ?? "workspace"}
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
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        }
      />
      <RateLimitRow
        testId="settings-page-hooks-extensions-policy-snapshot-rate"
        label="Snapshot rate limit"
        description="Published snapshots per window (queue = burst)"
        value={draft.resources.snapshot_rate_limit}
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
  onToggle,
}: {
  selected: string[];
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
                onClick={() => onToggle(kind)}
                data-testid={`settings-page-hooks-extensions-policy-allowed-kinds-${kind}`}
                data-active={active ? "true" : "false"}
                className={
                  active
                    ? "rounded-full border border-[color:var(--color-success)] bg-[color:var(--color-success-tint)] px-2.5 py-0.5 font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-success)]"
                    : "rounded-full border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2.5 py-0.5 font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-tertiary)] hover:border-[color:var(--color-divider-strong)]"
                }
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

function RateLimitRow({
  testId,
  label,
  description,
  value,
  onChange,
}: {
  testId: string;
  label: string;
  description: string;
  value: RateLimit;
  onChange: (next: RateLimit) => void;
}) {
  return (
    <SettingsFieldRow
      data-testid={testId}
      label={label}
      description={description}
      hint="LIMIT"
      control={
        <div className="flex items-center gap-1.5">
          <NumberInput
            testId={`${testId}-requests`}
            value={value.requests}
            placeholder="reqs"
            onChange={next => onChange({ ...value, requests: next })}
          />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
            per
          </span>
          <input
            className="h-8 w-20 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
            data-testid={`${testId}-window`}
            value={value.window}
            placeholder="5m"
            onChange={event => onChange({ ...value, window: event.target.value })}
          />
          <span className="font-mono text-[0.62rem] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
            queue
          </span>
          <NumberInput
            testId={`${testId}-queue`}
            value={value.queue}
            placeholder="queue"
            onChange={next => onChange({ ...value, queue: next })}
          />
        </div>
      }
    />
  );
}

function NumberInput({
  testId,
  value,
  placeholder,
  onChange,
}: {
  testId: string;
  value: number;
  placeholder: string;
  onChange: (next: number) => void;
}) {
  return (
    <input
      className="h-8 w-16 rounded-md border border-[color:var(--color-divider)] bg-[color:var(--color-surface-elevated)] px-2 font-mono text-xs text-[color:var(--color-text-primary)]"
      data-testid={testId}
      type="number"
      min={0}
      value={value}
      placeholder={placeholder}
      onChange={event => {
        const parsed = Number.parseInt(event.target.value, 10);
        onChange(Number.isFinite(parsed) ? parsed : 0);
      }}
    />
  );
}

interface SaveControlsProps {
  isDirty: boolean;
  isSaving: boolean;
  error: string | null;
  warnings?: string[];
  onSave: () => void;
  onReset: () => void;
}

function SaveControls({ isDirty, isSaving, error, warnings, onSave, onReset }: SaveControlsProps) {
  const disabled = !isDirty || isSaving;
  return (
    <div
      className="flex items-center gap-2"
      data-testid="settings-page-hooks-extensions-policy-controls"
      data-dirty={isDirty ? "true" : "false"}
    >
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
      ) : null}
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
  const toneClasses =
    tone === "success"
      ? "border-[color:var(--color-success)] bg-[color:var(--color-success-tint)] text-[color:var(--color-success)]"
      : "border-[color:var(--color-info)] bg-[color:var(--color-info-tint)] text-[color:var(--color-info)]";

  return (
    <div
      className={`flex items-center justify-between gap-3 rounded-md border px-3 py-2 text-xs ${toneClasses}`}
      data-testid="settings-page-hooks-extensions-action-result"
      data-kind={action.kind}
      role="status"
    >
      <span className="flex items-center gap-2">
        <Check className="size-3.5" />
        <span>{message}</span>
      </span>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        onClick={onDismiss}
        data-testid="settings-page-hooks-extensions-action-result-dismiss"
      >
        <X className="size-3.5" />
      </Button>
    </div>
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
