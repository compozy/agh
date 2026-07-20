import { AlertCircle, Check, KeyRound, Lock, Plus, RefreshCw, Trash2, X } from "lucide-react";

import {
  Alert,
  AlertAction,
  AlertDescription,
  BlockLoading,
  Button,
  ConfirmDialog,
  Empty,
  Input,
  ListingPage,
  ListingToolbar,
  useTopbarSlot,
} from "@agh/ui";

import {
  useVaultPage,
  type VaultDraft,
  type VaultEditorState,
  type VaultLastAction,
  type VaultRouteSearch,
} from "../hooks/use-vault-page";
import { SettingsEditorDialog, SettingsFieldRow } from "@/systems/settings";
import { VaultListFilters } from "../components/vault-list-filters";
import { VaultSecretSheet } from "../components/vault-secret-sheet";
import { VaultSecretsList } from "../components/vault-secrets-list";
import type { VaultSecret } from "../types";

export function VaultPage({ search = {} }: { search?: VaultRouteSearch }) {
  const page = useVaultPage(search);

  useTopbarSlot({
    glyph: <KeyRound />,
    count: page.isLoading ? undefined : page.counts.total,
    crumb: "Vault",
    actions: (
      <div className="flex items-center gap-2" data-testid="vault-topbar-actions">
        <Button
          data-testid="vault-page-refresh"
          disabled={page.isRefetching}
          onClick={() => void page.refetch()}
          size="sm"
          type="button"
          variant="ghost"
        >
          <RefreshCw className={page.isRefetching ? "size-3 animate-spin" : "size-3"} />
          Refresh
        </Button>
        <Button data-testid="vault-page-create" onClick={page.openCreate} size="sm" type="button">
          <Plus className="size-3" />
          New secret
        </Button>
      </div>
    ),
    toolbar: page.isLoading ? undefined : (
      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Filter by vault ref prefix"
            data-testid="vault-page-prefix"
            onChange={page.setPrefix}
            placeholder="Filter by ref prefix"
            value={page.prefix}
          />
          <ListingToolbar.Filters>
            <VaultListFilters namespace={page.namespace} onNamespaceChange={page.setNamespace} />
          </ListingToolbar.Filters>
        </ListingToolbar.Leading>
        <ListingToolbar.Trailing>
          <ListingToolbar.ViewToggle onChange={page.setView} value={page.view} />
        </ListingToolbar.Trailing>
      </ListingToolbar>
    ),
  });

  if (page.isLoading) {
    return <BlockLoading className="flex-1" data-testid="vault-page-loading" />;
  }

  return (
    <ListingPage
      banner={
        page.lastAction ? (
          <div className="px-9 pt-4">
            <LastActionAlert action={page.lastAction} onDismiss={page.dismissLastAction} />
          </div>
        ) : null
      }
      data-testid="vault-shell"
    >
      <p
        className="mb-3 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-subtle"
        data-testid="vault-page-sec-note"
      >
        <Lock aria-hidden="true" className="size-3.5 shrink-0 text-faint" />
        <span>
          {page.counts.total} redacted metadata {page.counts.total === 1 ? "entry" : "entries"} —
          values are write-only and never leave the daemon.
        </span>
        <span aria-hidden="true" className="text-faint">
          ·
        </span>
        <span data-testid="vault-page-count">{page.counts.total}</span>
        <span aria-hidden="true" className="text-faint">
          ·
        </span>
        <span data-testid="vault-page-sessions">{page.counts.sessions} session-scoped</span>
        <span aria-hidden="true" className="text-faint">
          ·
        </span>
        <span data-testid="vault-page-providers">{page.counts.providers} provider-scoped</span>
      </p>

      {page.queryError && page.secrets.length === 0 ? (
        <Empty
          action={
            <Button
              data-testid="vault-page-error-retry"
              disabled={page.isRefetching}
              onClick={() => void page.refetch()}
              size="sm"
              type="button"
              variant="ghost"
            >
              <RefreshCw className={page.isRefetching ? "size-3 animate-spin" : "size-3"} />
              Retry
            </Button>
          }
          data-testid="vault-page-error"
          description={page.queryError}
          icon={AlertCircle}
          title="Unable to load vault metadata"
        />
      ) : (
        <VaultSecretsList
          data-testid="vault-page-list"
          emptyDescription="Vault metadata appears here after a write-only secret is stored."
          emptyTitle="No vault secrets"
          error={page.queryError ? new Error(page.queryError) : null}
          isLoading={page.isRefetching && page.secrets.length === 0}
          onDelete={page.openDelete}
          onSelect={page.openInspect}
          secrets={page.secrets}
          selectedRef={page.selectedSecret?.ref ?? null}
          view={page.view}
        />
      )}

      <VaultSecretSheet
        deleteIsDisabled={page.replaceIsPending}
        onOpenChange={open => {
          if (!open) page.closeInspect();
        }}
        onReplace={page.replaceSecret}
        onReplaceValueChange={page.setReplaceValue}
        onRequestDelete={page.openDelete}
        open={page.selectedSecret !== null}
        replaceError={page.replaceError}
        replaceIsPending={page.replaceIsPending}
        replaceIsValid={page.replaceIsValid}
        replaceValue={page.replaceValue}
        secret={page.selectedSecret}
      />

      <VaultEditor
        canSave={page.editorIsValid}
        editor={page.editor}
        error={page.editorError}
        isSaving={page.editorIsSaving}
        onChange={page.updateDraft}
        onClose={page.closeEditor}
        onSave={page.saveEditor}
      />

      <VaultDeleteDialog
        error={page.deleteError}
        isDeleting={page.deleteIsPending}
        onClose={page.closeDelete}
        onConfirm={page.confirmDelete}
        target={page.deleteTarget.mode === "open" ? page.deleteTarget.secret : null}
      />
    </ListingPage>
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

function isSessionScopedVaultRef(ref: string): boolean {
  return ref.startsWith("vault:sessions/");
}

function VaultDeleteDialog({
  target,
  error,
  isDeleting,
  onClose,
  onConfirm,
}: VaultDeleteDialogProps) {
  const sessionScope = target ? isSessionScopedVaultRef(target.ref) : false;
  const confirmTypingValue = target && !sessionScope ? target.ref : undefined;
  return (
    <ConfirmDialog
      open={target !== null}
      title={sessionScope ? "Delete session vault secret?" : "Delete vault secret?"}
      description={
        target ? (
          <span>
            Delete metadata and encrypted value for{" "}
            <code className="font-mono text-fg">{target.ref}</code>.
            {sessionScope
              ? " This is a session-scoped secret; it is removed immediately."
              : " Cross-scope vault entries require typed confirmation."}
          </span>
        ) : null
      }
      error={error}
      isPending={isDeleting}
      cancelLabel="Cancel"
      confirmLabel={sessionScope ? "Confirm" : "Delete secret"}
      confirmIcon={Trash2}
      confirmTyping={confirmTypingValue}
      contentProps={{
        "data-testid": "settings-vault-delete",
        "data-scope": sessionScope ? "session" : "cross",
      }}
      descriptionProps={{ "data-testid": "settings-vault-delete-description" }}
      errorProps={{ "data-testid": "settings-vault-delete-error" }}
      cancelButtonProps={{
        "data-testid": "settings-vault-delete-cancel",
        disabled: isDeleting,
      }}
      confirmButtonProps={{
        "data-testid": "settings-vault-delete-confirm",
      }}
      confirmInputProps={{ "data-testid": "settings-vault-delete-confirm-typing" }}
      onConfirm={onConfirm}
      onOpenChange={next => {
        if (!next) onClose();
      }}
    />
  );
}

function LastActionAlert({
  action,
  onDismiss,
}: {
  action: VaultLastAction;
  onDismiss: () => void;
}) {
  const saved = action.kind === "saved";
  return (
    <Alert variant={saved ? "success" : "warning"} data-testid="vault-page-action-result">
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
          data-testid="vault-page-action-result-dismiss"
        >
          <X className="size-3" />
        </Button>
      </AlertAction>
    </Alert>
  );
}
