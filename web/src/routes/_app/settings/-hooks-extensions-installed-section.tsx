import { Info, Puzzle, RefreshCw, Trash2, X } from "lucide-react";

import type {
  SettingsExtensionEntry,
  SettingsExtensionMarketplaceEntry,
  SettingsExtensionProvenance,
} from "@/systems/settings";
import { Button, Empty, Eyebrow, Pill, Section, Spinner, Switch } from "@agh/ui";

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

export function ExtensionsSection({
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

export function TrustBadge({ trust }: { trust: ExtensionTrustReport }) {
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
