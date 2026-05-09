import {
  Button,
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  MetadataList,
  Pill,
  PillGroup,
  type PillTone,
} from "@agh/ui";

import type { SettingsProviderEntry } from "@/systems/settings";

import { ProviderLogo } from "./provider-logo";
import { ProviderModelCatalogStatus } from "./provider-model-catalog-status";
import { SettingsSourceBadge } from "./settings-source-badge";

interface ProviderCardProps {
  provider: SettingsProviderEntry;
  onEdit: (entry: SettingsProviderEntry) => void;
  onDelete: (entry: SettingsProviderEntry) => void;
}

const STATE_LABELS: Record<string, string> = {
  installed: "Installed",
  "binary-missing": "Binary missing",
  unconfigured: "Unconfigured",
};

export function ProviderCard({ provider, onEdit, onDelete }: ProviderCardProps) {
  const source = provider.source_metadata.effective_source;
  const shadowed = provider.source_metadata.shadowed_sources ?? [];
  const deletable = source.kind !== "builtin-provider";
  const state = providerStateTone(provider);
  const testId = `settings-page-providers-card-${provider.name}`;

  return (
    <Card
      data-testid={testId}
      className="transition-colors duration-150 ease-out hover:ring-accent/40"
    >
      <CardHeader>
        <div className="flex items-center gap-3">
          <span
            aria-hidden="true"
            className="flex size-10 items-center justify-center rounded-icon-well bg-(--color-surface-elevated) ring-1 ring-(--color-divider)"
          >
            <ProviderLogo provider={provider.name} className="size-5" />
          </span>
          <CardTitle className="font-mono text-sm text-(--color-text-primary)">
            {provider.name}
          </CardTitle>
        </div>
        {provider.default ? (
          <CardAction>
            <Pill tone="accent">DEFAULT</Pill>
          </CardAction>
        ) : null}
      </CardHeader>

      <CardContent className="flex flex-col flex-1 gap-2 border-t border-(--color-divider) pt-4">
        <MetadataList className="gap-y-2">
          <MetadataList.Row
            label="Command"
            valueProps={{
              "data-testid": `${testId}-command`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.command ?? <EmptyValue />}
          </MetadataList.Row>
          <MetadataList.Row
            label="Default model"
            valueProps={{
              "data-testid": `${testId}-model`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.models?.default ?? <EmptyValue />}
          </MetadataList.Row>
          <MetadataList.Row
            label="Curated models"
            valueProps={{
              "data-testid": `${testId}-curated-models`,
              className: "truncate font-mono text-xs",
            }}
          >
            <CuratedModels provider={provider} />
          </MetadataList.Row>
          <MetadataList.Row
            label="Reasoning"
            valueProps={{
              "data-testid": `${testId}-reasoning`,
              className: "truncate font-mono text-xs",
            }}
          >
            <ReasoningSupport provider={provider} />
          </MetadataList.Row>
          <MetadataList.Row
            label="Harness"
            valueProps={{
              "data-testid": `${testId}-harness`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.harness ? (
              <span className="flex flex-wrap items-center gap-1.5">
                <span>{provider.settings.harness}</span>
                {provider.settings.runtime_provider ? (
                  <Pill mono tone="neutral">
                    {provider.settings.runtime_provider}
                  </Pill>
                ) : null}
              </span>
            ) : (
              <EmptyValue />
            )}
          </MetadataList.Row>
          <MetadataList.Row
            label="Auth mode"
            valueProps={{
              "data-testid": `${testId}-auth-mode`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.auth_mode ?? <EmptyValue />}
          </MetadataList.Row>
          <MetadataList.Row
            label="Env policy"
            valueProps={{
              "data-testid": `${testId}-env-policy`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.env_policy ?? <EmptyValue />}
          </MetadataList.Row>
          <MetadataList.Row
            label="Home policy"
            valueProps={{
              "data-testid": `${testId}-home-policy`,
              className: "truncate font-mono text-xs",
            }}
          >
            {provider.settings.home_policy ?? <EmptyValue />}
          </MetadataList.Row>
          <MetadataList.Row
            label="Auth status"
            valueProps={{
              "data-testid": `${testId}-auth-status`,
              className: "truncate font-mono text-xs",
            }}
          >
            <ProviderAuthStatus provider={provider} />
          </MetadataList.Row>
          <MetadataList.Row
            label="Credential slots"
            valueProps={{
              "data-testid": `${testId}-api-key`,
              className: "truncate font-mono text-xs",
            }}
          >
            <CredentialSlots provider={provider} />
          </MetadataList.Row>
          <MetadataList.Row
            label="Credential"
            valueProps={{
              "data-testid": `${testId}-credential`,
              className: "truncate font-mono text-xs",
            }}
          >
            <CredentialState provider={provider} testId={testId} />
          </MetadataList.Row>
          <MetadataList.Row label="Source">
            <SettingsSourceBadge
              data-testid={`${testId}-source`}
              source={source}
              shadowed={shadowed}
            />
          </MetadataList.Row>
          <MetadataList.Row label="Catalog" className="items-start pt-1">
            <ProviderModelCatalogStatus providerId={provider.name} testId={`${testId}-catalog`} />
          </MetadataList.Row>
        </MetadataList>
      </CardContent>

      <CardFooter className="justify-between">
        <span className="flex items-center gap-2">
          <Pill.Dot
            tone={state.tone}
            size="md"
            data-testid={`${testId}-status`}
            data-tone={state.label}
          />
          <span className="font-mono text-badge font-semibold uppercase tracking-badge text-(--color-text-tertiary)">
            {STATE_LABELS[state.label] ?? state.label}
          </span>
        </span>
        <span className="flex items-center gap-2">
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onEdit(provider)}
            data-testid={`${testId}-edit`}
          >
            Edit
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onDelete(provider)}
            disabled={!deletable}
            data-testid={`${testId}-delete`}
            title={
              deletable
                ? undefined
                : "Builtin providers cannot be deleted -- edit the overlay to override them."
            }
          >
            Delete
          </Button>
        </span>
      </CardFooter>
    </Card>
  );
}

function EmptyValue() {
  return <span className="text-(--color-text-label)">--</span>;
}

function ProviderAuthStatus({ provider }: { provider: SettingsProviderEntry }) {
  const status = provider.auth_status;
  if (!status) {
    return <EmptyValue />;
  }
  const state = status.state || "unknown";
  const tone = authStatusTone(state);
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      <span>{state}</span>
      <Pill mono tone={tone}>
        {status.mode}
      </Pill>
    </span>
  );
}

function CuratedModels({ provider }: { provider: SettingsProviderEntry }) {
  const models = provider.settings.models?.curated ?? [];
  if (models.length === 0) {
    return <EmptyValue />;
  }
  const ids = models.map(model => model.id).filter(Boolean);
  if (ids.length === 0) {
    return <EmptyValue />;
  }
  const visibleIds = ids.slice(0, 2);
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      <PillGroup
        aria-label="Curated models"
        size="sm"
        value={visibleIds[0] ?? ""}
        onChange={() => undefined}
        items={visibleIds.map(id => ({
          value: id,
          label: id,
          disabled: true,
        }))}
      />
      {ids.length > 2 ? (
        <Pill mono tone="neutral">
          +{ids.length - 2}
        </Pill>
      ) : null}
    </span>
  );
}

function ReasoningSupport({ provider }: { provider: SettingsProviderEntry }) {
  const models = provider.settings.models?.curated ?? [];
  const supported = models.some(
    model => model.supports_reasoning || (model.reasoning_efforts?.length ?? 0) > 0
  );
  return supported ? "Per model" : "Not declared";
}

function authStatusTone(state: string): PillTone {
  switch (state) {
    case "authenticated":
    case "native_cli":
    case "present":
    case "none":
      return "success";
    case "missing_required":
    case "needs_login":
      return "warning";
    default:
      return "neutral";
  }
}

function CredentialState({
  provider,
  testId,
}: {
  provider: SettingsProviderEntry;
  testId: string;
}) {
  const credentials = provider.credentials ?? [];
  const credential = credentials[0];
  if (!credential) {
    return <EmptyValue />;
  }
  const missingRequired = credentials.some(item => item.required && !item.present);
  const presentCount = credentials.filter(item => item.present).length;
  const stateLabel = missingRequired
    ? "MISSING"
    : presentCount === credentials.length
      ? "BOUND"
      : "OPTIONAL";
  const stateTone = missingRequired ? "warning" : presentCount > 0 ? "success" : "neutral";
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      <span className="truncate">
        {credential.secret_ref}
        {credentials.length > 1 ? ` +${credentials.length - 1}` : ""}
      </span>
      <Pill mono tone={stateTone} data-testid={`${testId}-credential-state`}>
        {stateLabel}
      </Pill>
    </span>
  );
}

function CredentialSlots({ provider }: { provider: SettingsProviderEntry }) {
  const slots = provider.settings.credential_slots ?? [];
  const slot = slots[0];
  if (!slot) {
    return <EmptyValue />;
  }
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      <span className="truncate">
        {slot.target_env}
        {slots.length > 1 ? ` +${slots.length - 1}` : ""}
      </span>
      <Pill mono tone="neutral">
        {slot.name}
      </Pill>
    </span>
  );
}

function providerCredentialsConfigured(provider: SettingsProviderEntry): boolean {
  const credentials = provider.credentials ?? [];
  if (credentials.length === 0) {
    return true;
  }
  return credentials.every(credential => !credential.required || credential.present);
}

export function providerStateTone(provider: SettingsProviderEntry): {
  tone: PillTone;
  label: string;
} {
  if (!provider.command_available) {
    return { tone: "warning", label: "binary-missing" };
  }
  if (!providerCredentialsConfigured(provider)) {
    return { tone: "warning", label: "unconfigured" };
  }
  return { tone: "success", label: "installed" };
}
