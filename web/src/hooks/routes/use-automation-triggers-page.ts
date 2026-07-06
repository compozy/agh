import { startTransition, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";

import {
  automationTriggerToDraft,
  createAutomationDialogHandle,
  createAutomationTriggerDraft,
  createLoopTargetTriggerDraft,
  filterAutomationTriggers,
  normalizeAutomationRetry,
  sortAutomationTriggers,
  useAutomationTrigger,
  useAutomationTriggerRuns,
  useAutomationTriggers,
  useCreateAutomationTrigger,
  useDeleteAutomationTrigger,
  useUpdateAutomationTrigger,
} from "@/systems/automation";
import type { AutomationScopeFilter, CreateAutomationTriggerRequest } from "@/systems/automation";

import {
  automationUnavailableMessage,
  buildEmptyState,
  resolveSelectedId,
  useAutomationPageBase,
  type AutomationCreateSeed,
  type TriggerEditorState,
} from "./use-automation-page-base";

export function useAutomationTriggersPage(seed: AutomationCreateSeed = {}) {
  const page = useAutomationPageBase();
  const [editor, setEditor] = useState<TriggerEditorState | null>(null);
  const triggerSubmitInFlightRef = useRef(false);
  const seededRef = useRef(false);
  const editorHandle = useMemo(() => createAutomationDialogHandle(), []);

  const triggersQuery = useAutomationTriggers(page.listFilters);
  const triggers = triggersQuery.data ?? [];
  const runtimeUnavailableMessage = automationUnavailableMessage(
    "triggers",
    page.automationRuntime,
    triggersQuery.error
  );
  const visibleTriggers = useMemo(
    () => sortAutomationTriggers(filterAutomationTriggers(triggers, page.deferredSearchQuery)),
    [page.deferredSearchQuery, triggers]
  );
  const effectiveSelectedTriggerId = useMemo(
    () => resolveSelectedId(page.selectedId, visibleTriggers),
    [page.selectedId, visibleTriggers]
  );

  const triggerDetailQuery = useAutomationTrigger(effectiveSelectedTriggerId ?? "", {
    enabled: Boolean(effectiveSelectedTriggerId),
  });
  const triggerRunsQuery = useAutomationTriggerRuns(
    effectiveSelectedTriggerId ?? "",
    { limit: 10 },
    { enabled: Boolean(effectiveSelectedTriggerId) }
  );

  const createTriggerMutation = useCreateAutomationTrigger();
  const updateTriggerMutation = useUpdateAutomationTrigger();
  const deleteTriggerMutation = useDeleteAutomationTrigger();

  const selectedTrigger =
    triggerDetailQuery.data ??
    visibleTriggers.find(trigger => trigger.id === effectiveSelectedTriggerId) ??
    triggers.find(trigger => trigger.id === effectiveSelectedTriggerId);

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      page.setScopeFilter(nextScope);
      page.setSelectedId(null);
      setEditor(null);
    });
  };

  const handleCreate = () => {
    setEditor({ draft: createAutomationTriggerDraft(page.activeWorkspaceId), mode: "create" });
  };

  // Open the create sheet pre-targeted at a Loop when arriving from the detail CTA.
  useEffect(() => {
    if (seededRef.current || !seed.loop) return;
    seededRef.current = true;
    setEditor({
      draft: createLoopTargetTriggerDraft(page.activeWorkspaceId, seed.loop),
      mode: "create",
    });
  }, [seed.loop, page.activeWorkspaceId]);

  const handleEdit = () => {
    if (!selectedTrigger) {
      return;
    }

    setEditor({
      draft: automationTriggerToDraft(selectedTrigger),
      id: selectedTrigger.id,
      mode: "edit",
    });
  };

  const handleSubmit = async () => {
    if (!editor || triggerSubmitInFlightRef.current) {
      return;
    }

    triggerSubmitInFlightRef.current = true;
    try {
      const payload = {
        ...editor.draft,
        retry: normalizeAutomationRetry(editor.draft.retry ?? undefined),
      };
      const trigger =
        editor.mode === "create"
          ? await createTriggerMutation.mutateAsync(payload)
          : await updateTriggerMutation.mutateAsync({ data: payload, id: editor.id });

      page.setSelectedId(trigger.id);
      setEditor(null);
      toast.success(
        editor.mode === "create"
          ? `Created trigger ${trigger.name}.`
          : `Updated trigger ${trigger.name}.`
      );
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to save automation trigger");
    } finally {
      triggerSubmitInFlightRef.current = false;
    }
  };

  const handleDelete = async () => {
    if (!selectedTrigger) {
      return;
    }

    try {
      await deleteTriggerMutation.mutateAsync({ id: selectedTrigger.id });
      page.setSelectedId(null);
      toast.success(`Deleted ${selectedTrigger.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to delete automation trigger");
    }
  };

  const handleToggleEnabled = async (enabled: boolean) => {
    if (!selectedTrigger) {
      return;
    }

    try {
      await updateTriggerMutation.mutateAsync({ data: { enabled }, id: selectedTrigger.id });
      toast.success(`${enabled ? "Enabled" : "Disabled"} ${selectedTrigger.name}.`);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Failed to update automation state");
    }
  };

  const emptyState =
    visibleTriggers.length === 0
      ? buildEmptyState({
          hasQuery: page.deferredSearchQuery.trim() !== "",
          kind: "triggers",
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: page.activeWorkspace?.name,
    errorMessage: triggersQuery.error?.message ?? null,
    isLoading: triggersQuery.isLoading,
    jobs: [],
    kind: "triggers" as const,
    onSearchChange: page.setSearchQuery,
    onSelect: (id: string) =>
      startTransition(() => {
        page.setSelectedId(id);
      }),
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    selectedId: effectiveSelectedTriggerId,
    totalCount: triggers.length,
    triggers: visibleTriggers,
  };

  const detailPanelProps = {
    emptyState,
    error: triggerDetailQuery.error,
    state: {
      isDeleting: deleteTriggerMutation.isPending,
      isLoading: triggerDetailQuery.isLoading,
      isTogglePending: updateTriggerMutation.isPending,
      isTriggerPending: false,
    },
    item: selectedTrigger,
    kind: "triggers" as const,
    onDelete: () => {
      void handleDelete();
    },
    onEdit: handleEdit,
    onToggleEnabled: (enabled: boolean) => {
      void handleToggleEnabled(enabled);
    },
    runs: triggerRunsQuery.data ?? [],
    runsError: triggerRunsQuery.error,
    runsLoading: triggerRunsQuery.isLoading,
  };

  const editorDialogProps = {
    activeWorkspaceId: page.activeWorkspaceId,
    handle: editorHandle,
    workspaces: page.workspaces,
    editor: editor
      ? {
          ...editor,
          kind: "triggers" as const,
          isPending: createTriggerMutation.isPending || updateTriggerMutation.isPending,
          onCancel: () => setEditor(null),
          onChange: (draft: CreateAutomationTriggerRequest) =>
            setEditor(current => (current ? { ...current, draft } : current)),
          onSubmit: () => {
            void handleSubmit();
          },
        }
      : null,
  };

  return {
    currentTotalCount: triggers.length,
    detailPanelProps,
    editorDialogProps,
    handleCreate,
    handleScopeChange,
    initialError:
      runtimeUnavailableMessage && triggers.length === 0
        ? new Error(runtimeUnavailableMessage)
        : triggersQuery.error && triggers.length === 0
          ? triggersQuery.error
          : null,
    isInitialLoading: triggersQuery.isLoading && triggers.length === 0,
    listPanelProps,
    runtimeUnavailableMessage,
    scopeFilter: page.scopeFilter,
  };
}
