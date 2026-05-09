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

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "--";
  }
  return date.toLocaleString([], {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
  });
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
        className="overflow-hidden rounded-lg border border-(--color-divider)"
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
            {secrets.map(secret => (
              <TableRow key={secret.ref} data-testid="vault-secrets-row">
                <TableCell className="min-w-0">
                  <span className="block max-w-2xl truncate font-mono text-xs text-(--color-text-primary)">
                    {secret.ref}
                  </span>
                </TableCell>
                <TableCell>
                  <Pill mono tone={vaultNamespaceTone(secret.namespace)}>
                    {secret.namespace}
                  </Pill>
                </TableCell>
                <TableCell>
                  <span className="font-mono text-xs text-(--color-text-secondary)">
                    {secret.kind?.trim() || "--"}
                  </span>
                </TableCell>
                <TableCell>
                  <span className="font-mono text-xs text-(--color-text-tertiary)">
                    {formatDateTime(secret.updated_at)}
                  </span>
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
            ))}
          </TableBody>
        </Table>
      </DataSurface.Content>
    </DataSurface>
  );
}
