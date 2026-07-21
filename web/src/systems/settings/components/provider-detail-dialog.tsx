import { AlertCircle, Pencil, Plus, Save, Trash2 } from "lucide-react";

import {
  Alert,
  AlertDescription,
  Button,
  Dialog,
  DialogContent,
  DialogTitle,
  Eyebrow,
  LaneTabs,
  Pill,
  Spinner,
} from "@agh/ui";

import { getProviderStateView } from "../lib/provider-state";
import type { ProviderDraft, SettingsProviderEntry } from "../types";
import { ProviderEditForm } from "./provider-edit-form";
import { ProviderInspectView } from "./provider-inspect-view";
import { ProviderLogo } from "./provider-logo";

type DetailMode = "inspect" | "edit" | "create";

type DetailTab = "overview" | "configure";

export interface ProviderDetailDialogProps {
  open: boolean;
  mode: DetailMode;
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

/**
 * Provider detail as a centered modal (design decision D2 over the prototype's
 * side sheet): overlay click and Esc both dismiss; Overview / Configure lane
 * tabs map onto the page hook's inspect / edit modes.
 */
export function ProviderDetailDialog(props: ProviderDetailDialogProps) {
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
  const activeTab: DetailTab = isEditing ? "configure" : "overview";
  const deletable = Boolean(
    provider && provider.source_metadata.effective_source.kind !== "builtin-provider"
  );

  const handleTabChange = (next: DetailTab) => {
    if (next === "configure" && mode === "inspect") onSwitchToEdit();
    if (next === "overview" && mode === "edit") onCancelEdit();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange} disablePointerDismissal={false}>
      <DialogContent
        unframed
        className={[
          "w-(--width-modal-md) max-w-[calc(100%-2rem)] sm:max-w-(--width-modal-md)",
          "grid-rows-[auto_minmax(0,1fr)_auto]",
          "max-h-[min(var(--height-modal-md),calc(100%-2rem))]",
        ].join(" ")}
        data-testid="provider-detail-dialog"
        data-mode={mode}
      >
        <DetailHeaderBlock
          mode={mode}
          provider={provider}
          draftName={draft?.name ?? ""}
          stateDisplay={state?.display ?? null}
          stateTone={state?.tone ?? "neutral"}
          isDefault={provider?.default ?? false}
          activeTab={activeTab}
          onTabChange={handleTabChange}
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

        <DetailFooterBlock
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
      </DialogContent>
    </Dialog>
  );
}

interface DetailHeaderBlockProps {
  mode: DetailMode;
  provider: SettingsProviderEntry | null;
  draftName: string;
  stateDisplay: string | null;
  stateTone: "success" | "warning" | "danger" | "neutral" | "accent" | "info";
  isDefault: boolean;
  activeTab: DetailTab;
  onTabChange: (next: DetailTab) => void;
}

function DetailHeaderBlock({
  mode,
  provider,
  draftName,
  stateDisplay,
  stateTone,
  isDefault,
  activeTab,
  onTabChange,
}: DetailHeaderBlockProps) {
  const name = mode === "create" ? draftName || "New provider" : (provider?.name ?? "");
  const subtitle =
    mode === "create"
      ? "Create a new provider overlay"
      : provider?.settings.display_name ||
        (mode === "edit" ? "Edit provider overlay" : "Provider configuration");

  return (
    <header className="flex flex-col border-b border-line-soft px-6 pt-4">
      <div className="flex items-start gap-3">
        {mode === "create" ? (
          <span className="flex size-10 shrink-0 items-center justify-center rounded-icon-well bg-canvas text-subtle">
            <Plus aria-hidden="true" className="size-4" />
          </span>
        ) : (
          <span className="flex size-10 shrink-0 items-center justify-center rounded-icon-well bg-canvas text-fg">
            <ProviderLogo provider={provider?.name ?? "agh"} className="size-5" />
          </span>
        )}
        <div className="flex min-w-0 flex-1 flex-col gap-0.5">
          <Eyebrow className="text-faint">Provider</Eyebrow>
          <DialogTitle
            className="truncate font-mono text-sm font-medium text-fg-strong"
            data-testid="provider-detail-title"
          >
            {name}
          </DialogTitle>
          <p className="truncate text-xs text-muted">{subtitle}</p>
        </div>
        {mode !== "create" && (isDefault || stateDisplay) ? (
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5 pe-7">
            {isDefault ? <Pill tone="accent">Default</Pill> : null}
            {stateDisplay ? (
              <Pill tone={stateTone}>
                <Pill.Dot tone={stateTone} />
                {stateDisplay}
              </Pill>
            ) : null}
          </div>
        ) : null}
      </div>
      {mode === "create" ? (
        <div className="pb-4" />
      ) : (
        <LaneTabs<DetailTab>
          ariaLabel="Provider detail sections"
          className="mt-2"
          items={[
            { value: "overview", label: "Overview", testId: "provider-detail-tab-overview" },
            { value: "configure", label: "Configure", testId: "provider-detail-tab-configure" },
          ]}
          value={activeTab}
          onChange={onTabChange}
        />
      )}
    </header>
  );
}

interface DetailFooterBlockProps {
  mode: DetailMode;
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

function DetailFooterBlock(props: DetailFooterBlockProps) {
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
        <Alert variant="danger" data-testid="provider-detail-error">
          <AlertCircle className="mt-0.5 size-3 shrink-0" />
          <AlertDescription className="text-xs">{error}</AlertDescription>
        </Alert>
      ) : null}
      {!error && warnings && warnings.length > 0 ? (
        <Alert variant="warning" data-testid="provider-detail-warnings">
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
              data-testid="provider-detail-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={onSave}
              disabled={!canSave || isSaving}
              data-testid="provider-detail-save"
            >
              {isSaving ? (
                <Spinner className="size-3" />
              ) : (
                <Save aria-hidden="true" className="size-3" />
              )}
              {isSaving ? "Saving…" : mode === "create" ? "Create provider" : "Save provider"}
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
              data-testid="provider-detail-delete"
            >
              <Trash2 aria-hidden="true" className="size-3" />
              Delete overlay
            </Button>
            <Button
              type="button"
              variant="default"
              size="sm"
              onClick={onSwitchToEdit}
              disabled={isDeleting}
              data-testid="provider-detail-edit"
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
