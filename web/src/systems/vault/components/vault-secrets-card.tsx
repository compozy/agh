import { KeyRound } from "lucide-react";

import { CatalogCard, Pill, Time } from "@agh/ui";

import { vaultNamespaceTone } from "../lib/vault-tones";
import type { VaultSecret } from "../types";

export interface VaultSecretsCardProps {
  secret: VaultSecret;
  selected?: boolean;
  onSelect?: (secret: VaultSecret) => void;
}

export function VaultSecretsCard({ secret, selected = false, onSelect }: VaultSecretsCardProps) {
  const trimmedKind = secret.kind?.trim();
  const selectable = onSelect !== undefined;

  return (
    <CatalogCard
      actionable={selectable}
      data-ref={secret.ref}
      data-testid="vault-secrets-card"
      selected={selected}
    >
      <button
        aria-label={`Inspect ${secret.ref}`}
        className="flex min-w-0 flex-col gap-3 text-left"
        data-testid={`vault-secrets-select-${secret.ref}`}
        disabled={!selectable}
        onClick={() => onSelect?.(secret)}
        type="button"
      >
        <div className="flex items-start gap-2.5">
          <CatalogCard.Logo>
            <KeyRound aria-hidden="true" className="size-3.5" />
          </CatalogCard.Logo>
          <div className="flex min-w-0 flex-1 flex-col gap-1">
            <CatalogCard.Title className="font-mono text-xs font-medium">
              {secret.ref}
            </CatalogCard.Title>
            <Pill mono size="sm" tone={vaultNamespaceTone(secret.namespace)}>
              {secret.namespace}
            </Pill>
          </div>
        </div>
      </button>
      <CatalogCard.Actions className="justify-between">
        {trimmedKind ? (
          <Pill mono data-testid={`vault-secrets-kind-${secret.ref}`} size="sm" tone="neutral">
            {trimmedKind}
          </Pill>
        ) : (
          <span
            className="font-mono text-[11px] text-faint"
            data-testid={`vault-secrets-kind-empty-${secret.ref}`}
          >
            --
          </span>
        )}
        <Time
          className="font-mono text-[11px] text-faint"
          data-testid={`vault-secrets-updated-${secret.ref}`}
          iso={secret.updated_at}
        />
      </CatalogCard.Actions>
    </CatalogCard>
  );
}
