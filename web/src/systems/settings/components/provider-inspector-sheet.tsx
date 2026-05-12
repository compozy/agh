import { AlertCircle, Pencil, Plus, Save, Trash2 } from "lucide-react";

import { Alert, AlertDescription, Button, Pill, Sheet, SheetContent, Spinner } from "@agh/ui";

import type { ProviderDraft } from "@/hooks/routes/use-settings-providers-page";

import { getProviderStateView } from "../lib/provider-state";
import type { SettingsProviderEntry } from "../types";
import { ProviderEditForm } from "./provider-edit-form";
import { ProviderInspectView } from "./provider-inspect-view";
import { ProviderLogo } from "./provider-logo";

type InspectorMode = "inspect" | "edit" | "create";

export interface ProviderInspectorSheetProps {
  open: boolean;
  mode: InspectorMode;
  entry: SettingsProviderEntry | null;
  draft: ProviderDraft | null;
  existingNames: string[];
  error: string | null;
  warnings: string[] | undefined;
  canSave: boolean;
  isSaving: boolean;
  isDeleting: boolean;
  onOpenChange: (open: boolean) => void;
  onDraftChange: (updater: (draft: ProviderDraft) => ProviderDraft) => void;
  onSwitchToEdit: () => void;
  onCancelEdit: () => void;
  onSave: () => void;
  onRequestDelete: () => void;
  onRefreshCatalog: () => void;
}

export function ProviderInspectorSheet(props: ProviderInspectorSheetProps) {
  const {
    open,
    mode,
    entry,
    draft,
    error,
    warnings,
    canSave,
    isSaving,
    isDeleting,
    onOpenChange,
    onDraftChange,
    onSwitchToEdit,
    onCancelEdit,
    onSave,
    onRequestDelete,
    onRefreshCatalog,
  } = props;

  const provider = entry;
  const state = provider ? getProviderStateView(provider) : null;
  const isEditing = mode === "edit" || mode === "create";
  const isCreate = mode === "create";
  const deletable = Boolean(
    provider && provider.source_metadata.effective_source.kind !== "builtin-provider"
  );

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        showCloseButton={false}
        className="grid w-full grid-rows-[auto_1fr_auto] gap-0 p-0 sm:max-w-[36rem]"
        data-testid="provider-inspector-sheet"
        data-mode={mode}
      >
        <SheetTitleBlock
          mode={mode}
          provider={provider}
          draftName={draft?.name ?? ""}
          stateDisplay={state?.display ?? null}
          stateTone={state?.tone ?? "neutral"}
          isDefault={provider?.default ?? false}
        />

        <div className="min-h-0 overflow-y-auto px-6 py-5">
          {isEditing ? (
            draft ? (
              <ProviderEditForm
                mode={isCreate ? "create" : "edit"}
                draft={draft}
                onChange={onDraftChange}
              />
            ) : null
          ) : provider ? (
            <ProviderInspectView provider={provider} onRefreshCatalog={onRefreshCatalog} />
          ) : null}
        </div>

        <SheetFooterBlock
          mode={mode}
          isEditing={isEditing}
          canSave={canSave}
          isSaving={isSaving}
          isDeleting={isDeleting}
          deletable={deletable}
          error={error}
          warnings={warnings}
          onSwitchToEdit={onSwitchToEdit}
          onCancelEdit={onCancelEdit}
          onClose={() => onOpenChange(false)}
          onSave={onSave}
          onRequestDelete={onRequestDelete}
        />
      </SheetContent>
    </Sheet>
  );
}

interface SheetTitleBlockProps {
  mode: InspectorMode;
  provider: SettingsProviderEntry | null;
  draftName: string;
  stateDisplay: string | null;
  stateTone: "success" | "warning" | "danger" | "neutral" | "accent" | "info";
  isDefault: boolean;
}

function SheetTitleBlock({
  mode,
  provider,
  draftName,
  stateDisplay,
  stateTone,
  isDefault,
}: SheetTitleBlockProps) {
  const name = mode === "create" ? draftName || "New provider" : (provider?.name ?? "");
  const subtitle =
    mode === "create"
      ? "Create a new provider overlay"
      : (provider?.settings.display_name ??
        (mode === "edit" ? "Edit provider overlay" : "Provider configuration"));

  return (
    <header className="flex flex-col gap-3 border-b border-line-soft px-6 py-4">
      <div className="flex items-start gap-3">
        {mode === "create" ? (
          <span className="flex size-10 shrink-0 items-center justify-center rounded-icon-well bg-canvas-soft text-subtle">
            <Plus aria-hidden="true" className="size-4" />
          </span>
        ) : (
          <span className="flex size-10 shrink-0 items-center justify-center rounded-icon-well bg-canvas-soft text-fg">
            <ProviderLogo provider={provider?.name ?? "agh"} className="size-5" />
          </span>
        )}
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <h2
            className="truncate font-mono text-sm font-medium text-fg-strong"
            data-testid="provider-inspector-title"
          >
            {name}
          </h2>
          <p className="truncate text-xs text-muted">{subtitle}</p>
        </div>
      </div>
      {mode !== "create" && (isDefault || stateDisplay) ? (
        <div className="flex flex-wrap items-center gap-1.5">
          {isDefault ? <Pill tone="accent">DEFAULT</Pill> : null}
          {stateDisplay ? (
            <Pill tone={stateTone}>
              <Pill.Dot tone={stateTone} />
              {stateDisplay}
            </Pill>
          ) : null}
        </div>
      ) : null}
    </header>
  );
}

interface SheetFooterBlockProps {
  mode: InspectorMode;
  isEditing: boolean;
  canSave: boolean;
  isSaving: boolean;
  isDeleting: boolean;
  deletable: boolean;
  error: string | null;
  warnings: string[] | undefined;
  onSwitchToEdit: () => void;
  onCancelEdit: () => void;
  onClose: () => void;
  onSave: () => void;
  onRequestDelete: () => void;
}

function SheetFooterBlock(props: SheetFooterBlockProps) {
  const {
    mode,
    isEditing,
    canSave,
    isSaving,
    isDeleting,
    deletable,
    error,
    warnings,
    onSwitchToEdit,
    onCancelEdit,
    onClose,
    onSave,
    onRequestDelete,
  } = props;

  return (
    <footer className="flex flex-col gap-3 border-t border-line-soft px-6 py-4">
      {error ? (
        <Alert variant="danger" data-testid="provider-inspector-error">
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <AlertDescription className="text-xs">{error}</AlertDescription>
        </Alert>
      ) : null}
      {!error && warnings && warnings.length > 0 ? (
        <Alert variant="warning" data-testid="provider-inspector-warnings">
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <AlertDescription>
            <ul className="flex flex-col gap-1 text-xs">
              {warnings.map(warning => (
                <li key={warning}>{warning}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      ) : null}
      <div className="flex items-center justify-between gap-2">
        {isEditing ? (
          <>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={mode === "create" ? onClose : onCancelEdit}
              disabled={isSaving}
              data-testid="provider-inspector-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={onSave}
              disabled={!canSave || isSaving}
              data-testid="provider-inspector-save"
            >
              {isSaving ? (
                <Spinner className="size-3" />
              ) : (
                <Save aria-hidden="true" className="size-3" />
              )}
              {isSaving ? "Saving…" : mode === "create" ? "Create provider" : "Save changes"}
            </Button>
          </>
        ) : (
          <>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={onRequestDelete}
              disabled={!deletable || isDeleting}
              title={
                deletable
                  ? undefined
                  : "Builtin providers cannot be deleted -- edit the overlay to override them."
              }
              data-testid="provider-inspector-delete"
            >
              <Trash2 aria-hidden="true" className="size-3" />
              Delete overlay
            </Button>
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={onSwitchToEdit}
              data-testid="provider-inspector-edit"
            >
              <Pencil aria-hidden="true" className="size-3" />
              Edit settings
            </Button>
          </>
        )}
      </div>
    </footer>
  );
}
