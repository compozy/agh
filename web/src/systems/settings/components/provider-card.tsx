import {
  Button,
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  Pill,
  type PillTone,
} from "@agh/ui";
import type { ReactNode } from "react";

import type { SettingsProviderEntry } from "@/systems/settings";

import { ProviderLogo } from "./provider-logo";
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
      className="transition-colors duration-150 ease-out hover:ring-[color-mix(in_oklab,var(--color-accent)_40%,transparent)]"
    >
      <CardHeader>
        <div className="flex items-center gap-3">
          <span
            aria-hidden="true"
            className="flex size-10 items-center justify-center rounded-[10px] bg-[color:var(--color-surface-elevated)] ring-1 ring-[color:var(--color-divider)]"
          >
            <ProviderLogo provider={provider.name} className="size-5" />
          </span>
          <CardTitle className="font-mono text-sm text-[color:var(--color-text-primary)]">
            {provider.name}
          </CardTitle>
        </div>
        {provider.default ? (
          <CardAction>
            <Pill tone="accent">DEFAULT</Pill>
          </CardAction>
        ) : null}
      </CardHeader>

      <CardContent className="flex flex-col flex-1 gap-2 border-t border-[color:var(--color-divider)] pt-4">
        <MetaRow label="Command" testId={`${testId}-command`}>
          {provider.settings.command ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Default model" testId={`${testId}-model`}>
          {provider.settings.default_model ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Harness" testId={`${testId}-harness`}>
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
        </MetaRow>
        <MetaRow label="Auth mode" testId={`${testId}-auth-mode`}>
          {provider.settings.auth_mode ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Env policy" testId={`${testId}-env-policy`}>
          {provider.settings.env_policy ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Home policy" testId={`${testId}-home-policy`}>
          {provider.settings.home_policy ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Auth status" testId={`${testId}-auth-status`}>
          <ProviderAuthStatus provider={provider} />
        </MetaRow>
        <MetaRow label="Credential slots" testId={`${testId}-api-key`}>
          <CredentialSlots provider={provider} />
        </MetaRow>
        <MetaRow label="Credential" testId={`${testId}-credential`}>
          <CredentialState provider={provider} testId={testId} />
        </MetaRow>
        <MetaRow label="Source">
          <SettingsSourceBadge
            data-testid={`${testId}-source`}
            source={source}
            shadowed={shadowed}
          />
        </MetaRow>
      </CardContent>

      <CardFooter className="justify-between">
        <span className="flex items-center gap-2">
          <Pill.Dot
            tone={state.tone}
            size="md"
            data-testid={`${testId}-status`}
            data-tone={state.label}
          />
          <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
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
                : "Builtin providers cannot be deleted — edit the overlay to override them."
            }
          >
            Delete
          </Button>
        </span>
      </CardFooter>
    </Card>
  );
}

interface MetaRowProps {
  label: string;
  children: ReactNode;
  testId?: string;
}

function MetaRow({ label, children, testId }: MetaRowProps) {
  return (
    <div className="grid grid-cols-[7.5rem_1fr] items-center gap-3">
      <span className="font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-[color:var(--color-text-tertiary)]">
        {label}
      </span>
      <span
        data-testid={testId}
        className="min-w-0 truncate font-mono text-xs text-[color:var(--color-text-secondary)]"
      >
        {children}
      </span>
    </div>
  );
}

function EmptyValue() {
  return <span className="text-[color:var(--color-text-label)]">—</span>;
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
