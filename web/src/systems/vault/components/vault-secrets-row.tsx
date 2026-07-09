import { KeyRound, Trash2 } from "lucide-react";

import { Button, ListingRow, Pill, Time } from "@agh/ui";

import { vaultNamespaceTone } from "../lib/vault-tones";
import type { VaultSecret } from "../types";

export interface VaultSecretsRowProps {
  secret: VaultSecret;
  onDelete?: (secret: VaultSecret) => void;
}

/**
 * One vault inventory row: neutral KeyRound well, mono `ref` title with the
 * namespace Pill (`sessions` → info tone), an `updated` meta line, and the
 * always-visible delete action in the trail. Vault secrets have no detail
 * route, so the row is not a link.
 */
export function VaultSecretsRow({ secret, onDelete }: VaultSecretsRowProps) {
  const trimmedKind = secret.kind?.trim();
  return (
    <ListingRow data-testid="vault-secrets-row" data-ref={secret.ref} interactive={false}>
      <ListingRow.Icon>
        <KeyRound aria-hidden="true" className="size-4" />
      </ListingRow.Icon>
      <ListingRow.Main>
        <ListingRow.Name mono>
          <ListingRow.Title>{secret.ref}</ListingRow.Title>
          <Pill mono size="sm" tone={vaultNamespaceTone(secret.namespace)}>
            {secret.namespace}
          </Pill>
        </ListingRow.Name>
        <ListingRow.Meta>
          <span>updated</span>
          <Time
            className="font-mono text-[11px] text-faint"
            data-testid={`vault-secrets-updated-${secret.ref}`}
            iso={secret.updated_at}
          />
        </ListingRow.Meta>
      </ListingRow.Main>
      <ListingRow.Trail>
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
        {onDelete ? (
          <Button
            aria-label={`Delete ${secret.ref}`}
            data-testid={`vault-secrets-delete-${secret.ref}`}
            onClick={() => onDelete(secret)}
            size="icon-sm"
            type="button"
            variant="ghost"
          >
            <Trash2 aria-hidden="true" className="size-3" />
          </Button>
        ) : null}
      </ListingRow.Trail>
    </ListingRow>
  );
}
