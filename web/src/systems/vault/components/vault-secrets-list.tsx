import { KeyRound } from "lucide-react";

import { DataSurface } from "@agh/ui";

import type { VaultSecret } from "../types";
import { VaultSecretsRow } from "./vault-secrets-row";

interface VaultSecretsListProps {
  secrets: VaultSecret[];
  isLoading?: boolean;
  error?: Error | null;
  onDelete?: (secret: VaultSecret) => void;
  emptyTitle?: string;
  emptyDescription?: string;
  "data-testid"?: string;
}

/**
 * Vault inventory list: `ListingRow` rows inside the shared bordered list card,
 * with loading / error / empty routed through `DataSurface`. Replaces the former
 * secrets table so vault matches the catalog listing standard.
 */
export function VaultSecretsList({
  secrets,
  isLoading = false,
  error = null,
  onDelete,
  emptyTitle = "No vault secrets",
  emptyDescription = "Vault metadata appears here after a secret is stored.",
  "data-testid": testId = "vault-secrets-list",
}: VaultSecretsListProps) {
  return (
    <DataSurface
      state={isLoading ? "loading" : error ? "error" : secrets.length === 0 ? "empty" : "ready"}
    >
      <DataSurface.Loading data-testid={`${testId}-loading`} label="Loading vault metadata" />
      <DataSurface.Error
        description={error?.message}
        icon={KeyRound}
        title="Unable to load vault metadata"
        data-testid={`${testId}-error`}
      />
      <DataSurface.Empty
        description={emptyDescription}
        icon={KeyRound}
        title={emptyTitle}
        data-testid={`${testId}-empty`}
      />
      <DataSurface.Content
        className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
        data-testid={testId}
      >
        {secrets.map(secret => (
          <VaultSecretsRow key={secret.ref} onDelete={onDelete} secret={secret} />
        ))}
      </DataSurface.Content>
    </DataSurface>
  );
}
