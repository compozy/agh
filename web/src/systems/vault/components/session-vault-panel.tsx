import { KeyRound } from "lucide-react";

import {
  DataSurface,
  Item,
  ItemActions,
  ItemContent,
  ItemGroup,
  ItemMedia,
  ItemTitle,
  Pill,
} from "@agh/ui";

import type { VaultSecret } from "../types";

interface SessionVaultPanelProps {
  secrets: readonly VaultSecret[];
  isLoading?: boolean;
  error?: Error | null;
  sessionId?: string;
}

function displayVaultName(secret: VaultSecret, sessionId?: string): string {
  const prefix = sessionId ? `vault:sessions/${sessionId}/` : "";
  if (prefix && secret.ref.startsWith(prefix)) {
    return secret.ref.slice(prefix.length);
  }
  return secret.ref;
}

function formatUpdated(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "--";
  }
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function SessionVaultPanel({
  secrets,
  isLoading = false,
  error = null,
  sessionId,
}: SessionVaultPanelProps) {
  return (
    <DataSurface
      state={isLoading ? "loading" : error ? "error" : secrets.length === 0 ? "empty" : "ready"}
    >
      <DataSurface.Loading
        data-testid="session-inspector-vault-loading"
        label="Loading session vault metadata"
        size="sm"
        surface="bare"
      />
      <DataSurface.Error
        icon={KeyRound}
        title="Vault unavailable"
        description={error?.message}
        data-testid="session-inspector-vault-error"
      />
      <DataSurface.Empty
        icon={KeyRound}
        title="No session vault secrets"
        description="Session-scoped vault metadata appears here when tools store write-only values."
        data-testid="session-inspector-vault-empty"
      />
      <DataSurface.Content>
        <ItemGroup data-testid="session-inspector-vault-list">
          {secrets.map(secret => (
            <Item key={secret.ref} data-testid="session-inspector-vault-row">
              <ItemMedia>
                <Pill mono tone="success">
                  {secret.kind?.trim() || "secret"}
                </Pill>
              </ItemMedia>
              <ItemContent>
                <ItemTitle
                  className="truncate font-mono text-eyebrow text-(--color-text-primary)"
                  data-testid="session-inspector-vault-ref"
                  title={secret.ref}
                >
                  {displayVaultName(secret, sessionId)}
                </ItemTitle>
              </ItemContent>
              <ItemActions>
                <span className="shrink-0 font-mono text-badge text-(--color-text-tertiary)">
                  {formatUpdated(secret.updated_at)}
                </span>
              </ItemActions>
            </Item>
          ))}
        </ItemGroup>
      </DataSurface.Content>
    </DataSurface>
  );
}
