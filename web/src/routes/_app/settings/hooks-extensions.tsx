import { createFileRoute } from "@tanstack/react-router";
import {
  AlertCircle,
  AlertTriangle,
  Check,
  Download,
  Info,
  Puzzle,
  RefreshCw,
  Search,
  ShieldCheck,
  Trash2,
  Webhook,
  X,
} from "lucide-react";
import { useCallback, useMemo, useState, type Dispatch, type SetStateAction } from "react";

import { useSettingsHooksExtensionsPage } from "@/hooks/routes/use-settings-hooks-extensions-page";
import type {
  SettingsExtensionEntry,
  SettingsExtensionMarketplaceEntry,
  SettingsExtensionProvenance,
  SettingsHookEntry,
  SettingsHooksExtensionsSection,
  SettingsHooksExtensionsTransportParity,
} from "@/systems/settings";
import { NotificationPresetsPanel } from "@/systems/notifications";
import { SettingsFieldRow, SettingsNumberInput } from "@/systems/settings/components";
import { restartBannerPropsFor } from "@/systems/settings/lib/restart-banner-mapper";
import type { TopbarRouteContext } from "@/types/topbar";
import {
  Alert,
  AlertAction,
  AlertDescription,
  Button,
  Empty,
  Eyebrow,
  Input,
  NativeSelect,
  NativeSelectOption,
  PageShell,
  Pill,
  RestartBanner,
  Section,
  Spinner,
  StatusLineTopbarSlot,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  cn,
  pillGroupSegmentVariants,
  useTopbarSlot,
} from "@agh/ui";

export const Route = createFileRoute("/_app/settings/hooks-extensions")({
  beforeLoad: (): { topbar: TopbarRouteContext } => ({
    topbar: { title: "Hooks & extensions", icon: Puzzle },
  }),
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
  const notificationPresets = page.notificationPresets ?? [];
  const envelopeForSlot = page.envelope;
  useTopbarSlot({
    tabs: envelopeForSlot ? (
      <StatusLineTopbarSlot
        data-testid="settings-page-hooks-extensions-status-line"
        status="connected"
        items={[
          {
            key: "hooks",
            value: (
              <span data-testid="settings-page-hooks-extensions-hooks-total">
                {page.hooksCounts.enabled}/{page.hooksCounts.total} hooks enabled
              </span>
            ),
            tone: "neutral",
          },
          {
            key: "extensions",
            value: (
              <span data-testid="settings-page-hooks-extensions-extensions-total">
                {page.extensionsCounts.enabled}/{page.extensionsCounts.total} extensions enabled
              </span>
            ),
            tone: "neutral",
          },
        ]}
      />
    ) : undefined,
  });

  if (page.isLoading) {
    return (
      <div
        className="flex flex-1 items-center justify-center"
        data-testid="settings-page-hooks-extensions-loading"
      >
        <Spinner className="size-5 text-subtle" />
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
          <AlertCircle className="size-6 text-danger" />
          <p className="text-sm text-subtle">
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

  const bannerProps = restartBannerPropsFor("hooks-extensions", page.restart);

  return (
    <PageShell
      slug="hooks-extensions"
      banner={bannerProps ? <RestartBanner {...bannerProps} /> : null}
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
        onUpdate={page.updateExtension}
        onRemove={page.removeExtension}
        onOpenProvenance={page.openExtensionProvenance}
        selectedProvenanceName={page.selectedProvenanceName}
        selectedProvenance={page.selectedProvenance}
        provenanceLoading={page.provenanceLoading}
        provenanceError={page.provenanceError}
        onCloseProvenance={page.closeExtensionProvenance}
      />

      <MarketplaceSection
        entries={page.marketplaceEntries}
        query={page.marketplaceSearch}
        setQuery={page.setMarketplaceSearch}
        allowUnverified={page.marketplaceAllowUnverified}
        setAllowUnverified={page.setMarketplaceAllowUnverified}
        pendingSlug={page.pendingMarketplaceSlug}
        error={page.marketplaceError}
        isLoading={page.marketplaceLoading}
        canMutate={page.canMutateExtensions}
        onSearch={page.searchMarketplace}
        onInstall={page.installMarketplaceExtension}
      />

      <NotificationPresetsPanel
        presets={notificationPresets}
        isLoading={page.notificationPresetsLoading}
        error={page.notificationPresetsError ?? page.notificationPresetActionError}
        pendingName={page.pendingNotificationPresetName}
        canMutate={page.canMutateNotificationPresets}
        onCreate={page.createNotificationPreset}
        onToggle={page.toggleNotificationPreset}
        onDelete={page.deleteNotificationPreset}
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
    </PageShell>
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
      <AlertTriangle className="size-3" />
      <AlertDescription className="text-xs">
        <span className="font-medium text-warning">Some operations are unavailable over HTTP.</span>{" "}
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
    <Section
      data-testid="settings-page-hooks-extensions-hooks-section"
      label="Lifecycle hooks"
      note="restart required to re-read declarations · toggles persist now"
    >
      {hookError ? (
        <span
          className="text-xs text-danger"
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
          className="overflow-hidden rounded-lg border border-line"
          data-testid="settings-page-hooks-extensions-hooks-list"
        >
          <Table>
            <TableHeader>
              <TableRow className="bg-elevated">
                <TableHead className="eyebrow text-muted">Name</TableHead>
                <TableHead className="eyebrow text-muted">Event</TableHead>
                <TableHead className="eyebrow text-muted">Mode</TableHead>
                <TableHead className="eyebrow text-muted">Matcher</TableHead>
                <TableHead className="eyebrow w-[1%] text-right text-muted">Enabled</TableHead>
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
    </Section>
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
          <span className="font-mono text-sm text-fg">{entry.name}</span>
          {declaration.command ? (
            <span className="font-mono text-badge text-subtle">
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
      <TableCell className="font-mono text-xs text-muted">{mode}</TableCell>
      <TableCell
        className="font-mono text-xs text-muted"
        data-testid={`settings-page-hooks-extensions-hooks-row-${entry.name}-matcher`}
      >
        {matcherSummary || "--"}
      </TableCell>
      <TableCell>
        <div className="flex items-center justify-end gap-2">
          {pending ? <Spinner className="size-3 text-subtle" /> : null}
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
  onUpdate: (entry: SettingsExtensionEntry) => void;
  onRemove: (entry: SettingsExtensionEntry) => void;
  onOpenProvenance: (entry: SettingsExtensionEntry) => void;
  selectedProvenanceName: string | null;
  selectedProvenance: SettingsExtensionProvenance | null;
  provenanceLoading: boolean;
  provenanceError: string | null;
  onCloseProvenance: () => void;
}

function ExtensionsSection({
  extensions,
  pendingExtensionName,
  error,
  isLoading,
  canMutate,
  onToggle,
  onUpdate,
  onRemove,
  onOpenProvenance,
  selectedProvenanceName,
  selectedProvenance,
  provenanceLoading,
  provenanceError,
  onCloseProvenance,
}: ExtensionsSectionProps) {
  return (
    <Section
      data-testid="settings-page-hooks-extensions-extensions-section"
      label="Installed extensions"
      note="toggles apply immediately · no restart"
    >
      {error ? (
        <span
          className="text-xs text-danger"
          data-testid="settings-page-hooks-extensions-extensions-error"
        >
          {error}
        </span>
      ) : null}
      {isLoading && extensions.length === 0 ? (
        <div
          className="flex items-center gap-2 text-xs text-subtle"
          data-testid="settings-page-hooks-extensions-extensions-loading"
        >
          <Spinner className="size-3" />
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
              onUpdate={onUpdate}
              onRemove={onRemove}
              onOpenProvenance={onOpenProvenance}
              provenanceOpen={selectedProvenanceName === entry.name}
              selectedProvenance={selectedProvenance}
              provenanceLoading={provenanceLoading}
              provenanceError={provenanceError}
              onCloseProvenance={onCloseProvenance}
            />
          ))}
        </ul>
      )}
    </Section>
  );
}

function ExtensionRow({
  entry,
  pending,
  canMutate,
  onToggle,
  onUpdate,
  onRemove,
  onOpenProvenance,
  provenanceOpen,
  selectedProvenance,
  provenanceLoading,
  provenanceError,
  onCloseProvenance,
}: {
  entry: SettingsExtensionEntry;
  pending: boolean;
  canMutate: boolean;
  onToggle: (entry: SettingsExtensionEntry, nextEnabled: boolean) => void;
  onUpdate: (entry: SettingsExtensionEntry) => void;
  onRemove: (entry: SettingsExtensionEntry) => void;
  onOpenProvenance: (entry: SettingsExtensionEntry) => void;
  provenanceOpen: boolean;
  selectedProvenance: SettingsExtensionProvenance | null;
  provenanceLoading: boolean;
  provenanceError: string | null;
  onCloseProvenance: () => void;
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

  const provenance = entry.provenance;

  return (
    <li
      className="flex flex-col gap-3 rounded-md border border-line bg-elevated px-3 py-2"
      data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}`}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <Pill.Dot tone={healthTone} size="md" pulse={entry.health === "degraded"} />
          <div className="flex min-w-0 flex-col gap-0.5">
            <span className="truncate font-mono text-sm text-fg">{entry.name}</span>
            <Eyebrow className="text-subtle flex flex-wrap items-center gap-1.5">
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
              {entry.trust ? <TrustBadge trust={entry.trust} /> : null}
              {missingEnv.length > 0 ? (
                <Pill mono tone="warning">
                  env missing
                </Pill>
              ) : null}
            </Eyebrow>
            {provenance ? (
              <span
                className="max-w-full break-words font-mono text-badge text-muted"
                data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-provenance-summary`}
              >
                {provenance.installed_from} · {provenance.registry_tier}
                {provenance.allow_unverified ? " · allow_unverified=true" : ""}
              </span>
            ) : null}
            {entry.last_error ? (
              <span
                className="text-badge text-danger"
                data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-error`}
              >
                {entry.last_error}
              </span>
            ) : null}
            {missingEnv.length > 0 ? (
              <span
                className="max-w-full break-words font-mono text-badge text-warning"
                data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-missing-env`}
              >
                Missing env: {missingEnv.join(", ")}
              </span>
            ) : null}
          </div>
        </div>
        <div className="flex shrink-0 flex-wrap items-center justify-start gap-2 sm:justify-end">
          {pending ? <Spinner className="size-3 text-subtle" /> : null}
          <Button
            data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-provenance`}
            disabled={pending}
            onClick={() => onOpenProvenance(entry)}
            size="sm"
            type="button"
            variant="ghost"
          >
            <Info className="size-3.5" />
            Provenance
          </Button>
          <Button
            data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-update`}
            disabled={pending || !canMutate}
            onClick={() => onUpdate(entry)}
            size="sm"
            type="button"
            variant="outline"
          >
            <RefreshCw className="size-3.5" />
            Update
          </Button>
          <Button
            data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-remove`}
            disabled={pending || !canMutate}
            onClick={() => onRemove(entry)}
            size="sm"
            type="button"
            variant="destructive"
          >
            <Trash2 className="size-3.5" />
            Remove
          </Button>
          <Switch
            data-testid={`settings-page-hooks-extensions-extensions-item-${entry.name}-toggle`}
            checked={entry.enabled}
            disabled={pending || !canMutate}
            onCheckedChange={checked => onToggle(entry, checked)}
            aria-label={`Toggle extension ${entry.name}`}
          />
        </div>
      </div>
      {provenanceOpen ? (
        <ProvenancePanel
          name={entry.name}
          provenance={selectedProvenance}
          isLoading={provenanceLoading}
          error={provenanceError}
          onClose={onCloseProvenance}
        />
      ) : null}
    </li>
  );
}

type ExtensionTrustReport = NonNullable<
  SettingsExtensionEntry["trust"] | SettingsExtensionMarketplaceEntry["trust"]
>;

function trustTone(trust: ExtensionTrustReport): "success" | "warning" | "danger" | "neutral" {
  if (trust.decision === "verified" && trust.checksum_verified) return "success";
  if (trust.decision === "allowed_unverified" || trust.allow_unverified) return "warning";
  if (trust.decision === "blocked") return "danger";
  return "neutral";
}

function TrustBadge({ trust }: { trust: ExtensionTrustReport }) {
  return (
    <Pill
      mono
      tone={trustTone(trust)}
      data-testid={`settings-page-hooks-extensions-trust-${trust.decision}`}
    >
      {trust.decision}
      {trust.allow_unverified ? " · allow_unverified=true" : ""}
    </Pill>
  );
}

function ProvenancePanel({
  name,
  provenance,
  isLoading,
  error,
  onClose,
}: {
  name: string;
  provenance: SettingsExtensionProvenance | null;
  isLoading: boolean;
  error: string | null;
  onClose: () => void;
}) {
  return (
    <div
      className="rounded-md border border-line bg-canvas px-3 py-2"
      data-testid={`settings-page-hooks-extensions-extensions-item-${name}-provenance-panel`}
    >
      <div className="flex items-center justify-between gap-3">
        <Eyebrow className="text-muted">Provenance</Eyebrow>
        <Button
          aria-label={`Close provenance for ${name}`}
          onClick={onClose}
          size="icon-sm"
          type="button"
          variant="ghost"
        >
          <X className="size-3.5" />
        </Button>
      </div>
      {isLoading ? (
        <div className="mt-2 flex items-center gap-2 text-xs text-subtle">
          <Spinner className="size-3" />
          Loading provenance…
        </div>
      ) : error ? (
        <p className="mt-2 text-xs text-danger">{error}</p>
      ) : provenance ? (
        <div className="mt-2 grid gap-2 text-xs sm:grid-cols-2">
          <ProvenanceField label="installed_from" value={provenance.installed_from} />
          <ProvenanceField label="registry_tier" value={provenance.registry_tier} />
          <ProvenanceField label="checksum_sha256" value={provenance.checksum_sha256 || "--"} />
          <ProvenanceField
            label="checksum_verified"
            value={provenance.checksum_verified ? "true" : "false"}
          />
          <ProvenanceField
            label="allow_unverified"
            value={provenance.allow_unverified ? "true" : "false"}
          />
          <ProvenanceField label="installed_by" value={provenance.installed_by || "--"} />
        </div>
      ) : (
        <p className="mt-2 text-xs text-subtle">No provenance returned.</p>
      )}
      {provenance?.trust ? (
        <div className="mt-2 flex flex-wrap items-center gap-1.5">
          <TrustBadge trust={provenance.trust} />
          {provenance.trust.warnings?.map(item => (
            <Pill key={item.id} mono tone="warning">
              {item.code}
            </Pill>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function ProvenanceField({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="eyebrow text-muted">{label}</dt>
      <dd className="truncate font-mono text-badge text-fg" title={value}>
        {value}
      </dd>
    </div>
  );
}

interface MarketplaceSectionProps {
  entries: SettingsExtensionMarketplaceEntry[];
  query: string;
  setQuery: (value: string) => void;
  allowUnverified: boolean;
  setAllowUnverified: (value: boolean) => void;
  pendingSlug: string | null;
  error: string | null;
  isLoading: boolean;
  canMutate: boolean;
  onSearch: () => void;
  onInstall: (entry: SettingsExtensionMarketplaceEntry) => void;
}

function MarketplaceSection({
  entries,
  query,
  setQuery,
  allowUnverified,
  setAllowUnverified,
  pendingSlug,
  error,
  isLoading,
  canMutate,
  onSearch,
  onInstall,
}: MarketplaceSectionProps) {
  return (
    <Section
      data-testid="settings-page-hooks-extensions-marketplace-section"
      label="Extension marketplace"
      note="daemon-owned search and install"
      right={
        <label className="flex items-center gap-2 text-xs text-muted">
          <Switch
            aria-label="Allow unverified extension install"
            checked={allowUnverified}
            disabled={!canMutate}
            onCheckedChange={setAllowUnverified}
            data-testid="settings-page-hooks-extensions-marketplace-allow-unverified"
          />
          allow_unverified
        </label>
      }
    >
      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            aria-label="Search extension marketplace"
            className="font-mono"
            data-testid="settings-page-hooks-extensions-marketplace-search-input"
            placeholder="owner/repo or bridge"
            value={query}
            onChange={event => setQuery(event.target.value)}
            onKeyDown={event => {
              if (event.key === "Enter") {
                onSearch();
              }
            }}
          />
          <Button
            data-testid="settings-page-hooks-extensions-marketplace-search"
            disabled={isLoading}
            onClick={onSearch}
            type="button"
            variant="outline"
          >
            {isLoading ? <Spinner className="size-3.5" /> : <Search className="size-3.5" />}
            Search
          </Button>
        </div>
        {error ? (
          <span
            className="text-xs text-danger"
            data-testid="settings-page-hooks-extensions-marketplace-error"
          >
            {error}
          </span>
        ) : null}
        {isLoading && entries.length === 0 ? (
          <div
            className="flex items-center gap-2 text-xs text-subtle"
            data-testid="settings-page-hooks-extensions-marketplace-loading"
          >
            <Spinner className="size-3" />
            Loading marketplace…
          </div>
        ) : entries.length === 0 ? (
          <Empty
            icon={ShieldCheck}
            title="No marketplace entries"
            description="Search the configured registry for an installable extension slug."
            data-testid="settings-page-hooks-extensions-marketplace-empty"
          />
        ) : (
          <div
            className="overflow-hidden rounded-lg border border-line"
            data-testid="settings-page-hooks-extensions-marketplace-list"
          >
            <Table>
              <TableHeader>
                <TableRow className="bg-elevated">
                  <TableHead className="eyebrow text-muted">Extension</TableHead>
                  <TableHead className="eyebrow text-muted">Source</TableHead>
                  <TableHead className="eyebrow text-muted">Trust</TableHead>
                  <TableHead className="eyebrow w-[1%] text-right text-muted">Install</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {entries.map(entry => (
                  <MarketplaceRow
                    key={`${entry.source}:${entry.slug}`}
                    entry={entry}
                    pending={pendingSlug === entry.slug}
                    canMutate={canMutate}
                    onInstall={onInstall}
                  />
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </div>
    </Section>
  );
}

function MarketplaceRow({
  entry,
  pending,
  canMutate,
  onInstall,
}: {
  entry: SettingsExtensionMarketplaceEntry;
  pending: boolean;
  canMutate: boolean;
  onInstall: (entry: SettingsExtensionMarketplaceEntry) => void;
}) {
  return (
    <TableRow data-testid={`settings-page-hooks-extensions-marketplace-row-${entry.slug}`}>
      <TableCell>
        <div className="flex min-w-0 flex-col gap-0.5">
          <span className="truncate font-mono text-sm text-fg">{entry.name}</span>
          <span className="max-w-md truncate text-xs text-muted">
            {entry.description ?? entry.slug}
          </span>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex flex-col gap-0.5">
          <span className="font-mono text-xs text-fg">{entry.source}</span>
          <Eyebrow className="text-subtle">{entry.version ?? "latest"}</Eyebrow>
        </div>
      </TableCell>
      <TableCell>
        {entry.trust ? <TrustBadge trust={entry.trust} /> : <Pill mono>unknown</Pill>}
      </TableCell>
      <TableCell>
        <div className="flex justify-end">
          <Button
            data-testid={`settings-page-hooks-extensions-marketplace-row-${entry.slug}-install`}
            disabled={pending || !canMutate}
            onClick={() => onInstall(entry)}
            size="sm"
            type="button"
          >
            {pending ? <Spinner className="size-3.5" /> : <Download className="size-3.5" />}
            Install
          </Button>
        </div>
      </TableCell>
    </TableRow>
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
      <Check className="size-3" />
      <AlertDescription className="text-xs">{message}</AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onDismiss}
          data-testid="settings-page-hooks-extensions-action-result-dismiss"
        >
          <X className="size-3" />
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
  if (action.kind === "extension-installed") {
    return {
      message: `Extension "${action.name}" installed · trust decision recorded.`,
      tone: "info",
    };
  }
  if (action.kind === "extension-updated") {
    return {
      message: `Extension "${action.name}" update ${action.status}.`,
      tone: "info",
    };
  }
  if (action.kind === "extension-removed") {
    return {
      message: `Extension "${action.name}" removed.`,
      tone: "info",
    };
  }
  if (action.kind === "notification-preset-created") {
    return {
      message: "Notification preset " + action.name + " created.",
      tone: "success",
    };
  }
  if (action.kind === "notification-preset-toggled") {
    const state = action.enabled ? "enabled" : "disabled";
    return {
      message: "Notification preset " + action.name + " " + state + ".",
      tone: "success",
    };
  }
  if (action.kind === "notification-preset-deleted") {
    return {
      message: "Notification preset " + action.name + " deleted.",
      tone: "info",
    };
  }
  const state = action.enabled ? "enabled" : "disabled";
  return {
    message: `Extension "${action.name}" ${state} · applied immediately.`,
    tone: "info",
  };
}
