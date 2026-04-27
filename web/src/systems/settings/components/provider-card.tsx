import {
  Button,
  Card,
  CardAction,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
  MonoBadge,
  Pill,
  StatusDot,
  type StatusDotTone,
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
            <Pill variant="accent">DEFAULT</Pill>
          </CardAction>
        ) : null}
      </CardHeader>

      <CardContent className="flex flex-col gap-2 border-t border-[color:var(--color-divider)] pt-4">
        <MetaRow label="Command" testId={`${testId}-command`}>
          {provider.settings.command ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="Default model" testId={`${testId}-model`}>
          {provider.settings.default_model ?? <EmptyValue />}
        </MetaRow>
        <MetaRow label="API key env" testId={`${testId}-api-key`}>
          {provider.settings.api_key_env ? (
            <span className="flex flex-wrap items-center gap-1.5">
              <span className="truncate">{provider.settings.api_key_env}</span>
              <MonoBadge
                tone={provider.api_key_env_present ? "success" : "warning"}
                data-testid={`${testId}-api-key-state`}
              >
                {provider.api_key_env_present ? "SET" : "MISSING"}
              </MonoBadge>
            </span>
          ) : (
            <EmptyValue />
          )}
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
          <StatusDot
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

export function providerStateTone(provider: SettingsProviderEntry): {
  tone: StatusDotTone;
  label: string;
} {
  if (!provider.command_available) {
    return { tone: "warning", label: "binary-missing" };
  }
  if (!provider.api_key_env_present && provider.settings.api_key_env) {
    return { tone: "warning", label: "unconfigured" };
  }
  return { tone: "success", label: "installed" };
}
