import { KeyRound, Loader2 } from "lucide-react";

import { Empty, Pill } from "@agh/ui";

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
    return "—";
  }
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

export function SessionVaultPanel({
  secrets,
  isLoading = false,
  error = null,
  sessionId,
}: SessionVaultPanelProps) {
  if (isLoading) {
    return (
      <div
        className="flex min-h-full items-center justify-center"
        data-testid="session-inspector-vault-loading"
      >
        <Loader2 className="size-4 animate-spin text-[color:var(--color-text-tertiary)]" />
      </div>
    );
  }

  if (error) {
    return (
      <Empty
        icon={KeyRound}
        title="Vault unavailable"
        description={error.message}
        data-testid="session-inspector-vault-error"
      />
    );
  }

  if (secrets.length === 0) {
    return (
      <Empty
        icon={KeyRound}
        title="No session vault secrets"
        description="Session-scoped vault metadata appears here when tools store write-only values."
        data-testid="session-inspector-vault-empty"
      />
    );
  }

  return (
    <ul
      className="flex flex-col divide-y divide-[color:var(--color-divider)]"
      data-testid="session-inspector-vault-list"
    >
      {secrets.map(secret => (
        <li
          key={secret.ref}
          className="flex items-center gap-2 py-2"
          data-testid="session-inspector-vault-row"
        >
          <Pill mono tone="success">
            {secret.kind?.trim() || "secret"}
          </Pill>
          <span
            className="min-w-0 flex-1 truncate font-mono text-[11.5px] text-[color:var(--color-text-primary)]"
            data-testid="session-inspector-vault-ref"
            title={secret.ref}
          >
            {displayVaultName(secret, sessionId)}
          </span>
          <span className="shrink-0 font-mono text-[10px] text-[color:var(--color-text-tertiary)]">
            {formatUpdated(secret.updated_at)}
          </span>
        </li>
      ))}
    </ul>
  );
}
