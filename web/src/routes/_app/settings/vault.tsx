import { AlertCircle, Check, KeyRound, Plus, RefreshCw, Trash2, X } from "lucide-react";
import { createFileRoute } from "@tanstack/react-router";

import {
  Alert,
  AlertAction,
  AlertDescription,
  BlockLoading,
  Button,
  ConfirmDialog,
  Empty,
  Input,
  PageShell,
  Section,
} from "@agh/ui";

import {
  useSettingsVaultPage,
  type VaultDraft,
  type VaultEditorState,
  type VaultLastAction,
  type VaultNamespaceFilter,
} from "@/hooks/routes/use-settings-vault-page";
import {
  SettingsEditorDialog,
  SettingsFieldRow,
  SettingsStatusLine,
} from "@/systems/settings/components";
import { VAULT_NAMESPACES, VaultSecretsTable, type VaultSecret } from "@/systems/vault";

export const Route = createFileRoute("/_app/settings/vault")({
  component: VaultSettingsPage,
});

function VaultSettingsPage() {
  const page = useSettingsVaultPage();

  if (page.isLoading) {
    return <BlockLoading className="flex-1" data-testid="settings-page-vault-loading" />;
  }

  return (
    <PageShell
      slug="vault"
      title="Vault"
      statusLine={
        <SettingsStatusLine
          daemonLabel={page.queryError ? "vault unavailable" : "vault available"}
          status={page.queryError ? "error" : "connected"}
          data-testid="settings-page-vault-status-line"
          items={[
            <span key="total" data-testid="settings-page-vault-total">
              {page.counts.total} secrets
            </span>,
            <span key="sessions" data-testid="settings-page-vault-sessions">
              {page.counts.sessions} session-scoped
            </span>,
            <span key="providers" data-testid="settings-page-vault-providers">
              {page.counts.providers} provider-scoped
            </span>,
          ]}
        />
      }
      actions={
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() => void page.refetch()}
          disabled={page.isRefetching}
          data-testid="settings-page-vault-refresh"
        >
          <RefreshCw className={page.isRefetching ? "size-3.5 animate-spin" : "size-3.5"} />
          Refresh
        </Button>
      }
    >
      {page.lastAction ? (
        <ActionResultBanner action={page.lastAction} onDismiss={page.dismissLastAction} />
      ) : null}

      <Section
        label="Secrets"
        data-testid="settings-page-vault-header-row"
        note={<>{page.counts.total} redacted metadata records exposed by the daemon vault</>}
        right={
          <Button
            type="button"
            variant="default"
            size="sm"
            onClick={page.openCreate}
            data-testid="settings-page-vault-create"
          >
            <Plus className="size-3.5" />
            New secret
          </Button>
        }
      />

      <VaultFilterBar
        namespace={page.namespace}
        prefix={page.prefix}
        onNamespaceChange={page.setNamespace}
        onPrefixChange={page.setPrefix}
      />

      {page.queryError && page.secrets.length === 0 ? (
        <Empty
          icon={AlertCircle}
          title="Unable to load vault metadata"
          description={page.queryError}
          data-testid="settings-page-vault-error"
        />
      ) : (
        <VaultSecretsTable
          secrets={page.secrets}
          isLoading={page.isRefetching && page.secrets.length === 0}
          error={page.queryError ? new Error(page.queryError) : null}
          onDelete={page.openDelete}
          emptyTitle="No vault secrets"
          emptyDescription="Vault metadata appears here after a write-only secret is stored."
          data-testid="settings-page-vault-table"
        />
      )}

      <VaultEditor
        editor={page.editor}
        isSaving={page.editorIsSaving}
        canSave={page.editorIsValid}
        error={page.editorError}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <VaultDeleteDialog
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.secret : null}
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
      />
    </PageShell>
  );
}

interface VaultFilterBarProps {
  namespace: VaultNamespaceFilter;
  prefix: string;
  onNamespaceChange: (namespace: VaultNamespaceFilter) => void;
  onPrefixChange: (prefix: string) => void;
}

function VaultFilterBar({
  namespace,
  prefix,
  onNamespaceChange,
  onPrefixChange,
}: VaultFilterBarProps) {
  return (
    <div
      className="grid gap-4 rounded-lg border border-(--color-divider) bg-(--color-surface-panel) p-4 md:grid-cols-[12rem_minmax(0,1fr)]"
      data-testid="settings-page-vault-filters"
    >
      <label className="flex min-w-0 flex-col gap-2">
        <span className="font-mono text-eyebrow font-semibold uppercase tracking-mono text-(--color-text-label)">
          Namespace
        </span>
        <select
          value={namespace}
          onChange={event => onNamespaceChange(event.target.value as VaultNamespaceFilter)}
          className="h-9 rounded-md border border-(--color-divider) bg-(--color-surface-elevated) px-3 text-sm text-(--color-text-primary) outline-none"
          data-testid="settings-page-vault-namespace"
        >
          <option value="all">All namespaces</option>
          {VAULT_NAMESPACES.map(item => (
            <option key={item} value={item}>
              {item}
            </option>
          ))}
        </select>
      </label>
      <label className="flex min-w-0 flex-col gap-2">
        <span className="font-mono text-eyebrow font-semibold uppercase tracking-mono text-(--color-text-label)">
          Prefix
        </span>
        <Input
          value={prefix}
          onChange={event => onPrefixChange(event.target.value)}
          placeholder="vault:sessions/sess_123/"
          className="font-mono"
          data-testid="settings-page-vault-prefix"
        />
      </label>
    </div>
  );
}

interface VaultEditorProps {
  editor: VaultEditorState;
  isSaving: boolean;
  canSave: boolean;
  error: string | null;
  onChange: (updater: (draft: VaultDraft) => VaultDraft) => void;
  onClose: () => void;
  onSave: () => void;
}

function VaultEditor({
  editor,
  isSaving,
  canSave,
  error,
  onChange,
  onClose,
  onSave,
}: VaultEditorProps) {
  if (editor.mode === "closed") {
    return null;
  }

  const draft = editor.draft;
  const refError =
    draft.ref.trim() && !draft.ref.trim().startsWith("vault:")
      ? "Vault refs must start with vault:."
      : null;

  return (
    <SettingsEditorDialog
      open
      mode="create"
      title="New vault secret"
      slug="vault"
      description="Stores a write-only secret value and returns redacted metadata."
      error={error ?? refError}
      canSave={canSave && !refError}
      isSaving={isSaving}
      saveLabel="Store secret"
      onSave={onSave}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    >
      <SettingsFieldRow
        label="Ref"
        description="Daemon-owned vault reference."
        hint="REQUIRED"
        error={refError}
        data-testid="settings-vault-editor-ref"
        control={
          <Input
            className="w-[min(100%,28rem)] font-mono"
            value={draft.ref}
            onChange={event => onChange(current => ({ ...current, ref: event.target.value }))}
            placeholder="vault:sessions/sess_123/github-token"
            data-testid="settings-vault-editor-ref-input"
          />
        }
      />
      <SettingsFieldRow
        label="Kind"
        description="Metadata label returned on public Vault surfaces."
        hint="OPTIONAL"
        data-testid="settings-vault-editor-kind"
        control={
          <Input
            className="w-48 font-mono"
            value={draft.kind}
            onChange={event => onChange(current => ({ ...current, kind: event.target.value }))}
            placeholder="api_key"
            data-testid="settings-vault-editor-kind-input"
          />
        }
      />
      <SettingsFieldRow
        label="Secret value"
        description="Write-only payload. The daemon never returns this value."
        hint="REQUIRED"
        data-testid="settings-vault-editor-secret-value"
        control={
          <Input
            className="w-[min(100%,28rem)] font-mono"
            type="password"
            value={draft.secretValue}
            onChange={event =>
              onChange(current => ({ ...current, secretValue: event.target.value }))
            }
            placeholder="Stored without plaintext readback"
            data-testid="settings-vault-editor-secret-value-input"
          />
        }
      />
    </SettingsEditorDialog>
  );
}

interface VaultDeleteDialogProps {
  target: VaultSecret | null;
  error: string | null;
  isDeleting: boolean;
  onClose: () => void;
  onConfirm: () => void;
}

function VaultDeleteDialog({
  target,
  error,
  isDeleting,
  onClose,
  onConfirm,
}: VaultDeleteDialogProps) {
  return (
    <ConfirmDialog
      open={target !== null}
      title="Delete vault secret?"
      description={
        target ? (
          <span>
            Delete metadata and encrypted value for{" "}
            <code className="font-mono text-(--color-text-primary)">{target.ref}</code>.
          </span>
        ) : null
      }
      error={error}
      isPending={isDeleting}
      cancelLabel="Cancel"
      confirmLabel="Delete secret"
      confirmIcon={Trash2}
      contentProps={{ "data-testid": "settings-vault-delete" }}
      descriptionProps={{ "data-testid": "settings-vault-delete-description" }}
      errorProps={{ "data-testid": "settings-vault-delete-error" }}
      cancelButtonProps={{
        "data-testid": "settings-vault-delete-cancel",
        disabled: isDeleting,
      }}
      confirmButtonProps={{
        "data-testid": "settings-vault-delete-confirm",
      }}
      onConfirm={onConfirm}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    />
  );
}

function ActionResultBanner({
  action,
  onDismiss,
}: {
  action: VaultLastAction;
  onDismiss: () => void;
}) {
  const saved = action.kind === "saved";
  return (
    <Alert variant={saved ? "success" : "warning"} data-testid="settings-page-vault-action-result">
      {saved ? <Check className="size-4" /> : <KeyRound className="size-4" />}
      <AlertDescription>
        {saved ? "Stored vault metadata for " : "Deleted vault secret "}
        <code className="font-mono">{action.ref}</code>.
      </AlertDescription>
      <AlertAction>
        <Button
          type="button"
          variant="ghost"
          size="icon-sm"
          aria-label="Dismiss vault action result"
          onClick={onDismiss}
          data-testid="settings-page-vault-action-result-dismiss"
        >
          <X className="size-3.5" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
