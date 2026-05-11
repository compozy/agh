import { KeyRound, Trash2 } from "lucide-react";

import {
  Button,
  DataSurface,
  Pill,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Time,
} from "@agh/ui";

import { vaultNamespaceTone } from "../lib/vault-tones";
import type { VaultSecret } from "../types";

interface VaultSecretsTableProps {
  secrets: VaultSecret[];
  isLoading?: boolean;
  error?: Error | null;
  onDelete?: (secret: VaultSecret) => void;
  emptyTitle?: string;
  emptyDescription?: string;
  "data-testid"?: string;
}

export function VaultSecretsTable({
  secrets,
  isLoading = false,
  error = null,
  onDelete,
  emptyTitle = "No vault secrets",
  emptyDescription = "Vault metadata appears here after a secret is stored.",
  "data-testid": testId = "vault-secrets-table",
}: VaultSecretsTableProps) {
  return (
    <DataSurface
      state={isLoading ? "loading" : error ? "error" : secrets.length === 0 ? "empty" : "ready"}
    >
      <DataSurface.Loading data-testid={`${testId}-loading`} label="Loading vault metadata" />
      <DataSurface.Error
        icon={KeyRound}
        title="Unable to load vault metadata"
        description={error?.message}
        data-testid={`${testId}-error`}
      />
      <DataSurface.Empty
        icon={KeyRound}
        title={emptyTitle}
        description={emptyDescription}
        data-testid={`${testId}-empty`}
      />
      <DataSurface.Content
        className="overflow-hidden rounded-lg border border-line"
        data-testid={testId}
      >
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Ref</TableHead>
              <TableHead>Namespace</TableHead>
              <TableHead>Kind</TableHead>
              <TableHead>Updated</TableHead>
              {onDelete ? <TableHead className="w-12 text-right">Action</TableHead> : null}
            </TableRow>
          </TableHeader>
          <TableBody>
            {secrets.map(secret => {
              const trimmedKind = secret.kind?.trim();
              return (
                <TableRow key={secret.ref} data-testid="vault-secrets-row">
                  <TableCell className="min-w-0">
                    <span className="block max-w-2xl truncate font-mono text-xs text-fg">
                      {secret.ref}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Pill mono tone={vaultNamespaceTone(secret.namespace)}>
                      {secret.namespace}
                    </Pill>
                  </TableCell>
                  <TableCell>
                    {trimmedKind ? (
                      <Pill mono data-testid={`vault-secrets-kind-${secret.ref}`} tone="neutral">
                        {trimmedKind}
                      </Pill>
                    ) : (
                      <span
                        className="font-mono text-xs text-muted"
                        data-testid={`vault-secrets-kind-empty-${secret.ref}`}
                      >
                        --
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Time
                      className="font-mono text-xs text-subtle"
                      data-testid={`vault-secrets-updated-${secret.ref}`}
                      iso={secret.updated_at}
                    />
                  </TableCell>
                  {onDelete ? (
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        aria-label={`Delete ${secret.ref}`}
                        onClick={() => onDelete(secret)}
                        data-testid={`vault-secrets-delete-${secret.ref}`}
                      >
                        <Trash2 className="size-3.5" />
                      </Button>
                    </TableCell>
                  ) : null}
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </DataSurface.Content>
    </DataSurface>
  );
}
