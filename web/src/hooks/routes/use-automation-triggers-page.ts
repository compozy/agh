import { startTransition, useEffect, useRef, useState } from "react";
import { toast } from "sonner";

import {
  AutomationApiError,
  automationTriggerToDraft,
  buildAutomationTriggerRequest,
  createAutomationDialogHandle,
  createAutomationTriggerDraft,
  createLoopTargetTriggerDraft,
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
  type AutomationRouteSearch,
  type TriggerEditorState,
} from "./use-automation-page-base";

export function useAutomationTriggersPage(
  seed: AutomationCreateSeed = {},
  search: AutomationRouteSearch = {}
) {
  const page = useAutomationPageBase("triggers", search);
  const [editor, setEditor] = useState<TriggerEditorState | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const triggerSubmitInFlightRef = useRef(false);
  const seededRef = useRef(false);
  const [editorHandle] = useState(createAutomationDialogHandle);

  const triggersQuery = useAutomationTriggers({ ...page.listFilters, event: search.event });
  const triggers = triggersQuery.triggers;
  const runtimeUnavailableMessage = automationUnavailableMessage(
    "triggers",
    page.automationRuntime,
    triggersQuery.error
  );
  const effectiveSelectedTriggerId = resolveSelectedId(page.selectedId, triggers);

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
    triggerDetailQuery.data ?? triggers.find(trigger => trigger.id === effectiveSelectedTriggerId);

  const handleScopeChange = (nextScope: AutomationScopeFilter) => {
    startTransition(() => {
      page.setScopeFilter(nextScope);
      page.setSelectedId(null);
      setEditor(null);
    });
  };

  const handleCreate = () => {
    setSubmitError(null);
    setEditor({ draft: createAutomationTriggerDraft(page.activeWorkspaceId), mode: "create" });
  };

  // Open the create sheet pre-targeted at a Loop when arriving from the detail CTA.
  useEffect(() => {
    if (seededRef.current || !seed.loop) return;
    seededRef.current = true;
    setSubmitError(null);
    setEditor({
      draft: createLoopTargetTriggerDraft(page.activeWorkspaceId, seed.loop),
      mode: "create",
    });
  }, [seed.loop, page.activeWorkspaceId]);

  const handleEdit = () => {
    if (!selectedTrigger) {
      return;
    }

    setSubmitError(null);
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
    setSubmitError(null);
    try {
      const payload = buildAutomationTriggerRequest(editor.draft);
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
      const detail =
        error instanceof Error && error.message.trim() !== ""
          ? error.message.trim().replace(/\.$/, "")
          : "Failed to save automation trigger";
      const message =
        error instanceof AutomationApiError && error.status === 0
          ? "Failed to save automation trigger. Check your connection and try again."
          : `${detail}. Review the target and try again.`;
      setSubmitError(message);
      toast.error(detail);
    }
    triggerSubmitInFlightRef.current = false;
  };

  const handleDelete = async () => {
    if (!selectedTrigger) {
      throw new Error("The selected trigger is no longer available.");
    }

    await deleteTriggerMutation.mutateAsync({ id: selectedTrigger.id });
    page.setSelectedId(null);
    toast.success(`Deleted ${selectedTrigger.name}.`);
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
    triggers.length === 0
      ? buildEmptyState({
          hasQuery: page.searchQuery.trim() !== "",
          kind: "triggers",
          onCreate: handleCreate,
        })
      : null;

  const listPanelProps = {
    activeWorkspaceName: page.activeWorkspace?.name,
    errorMessage: triggersQuery.error?.message ?? null,
    isLoading: triggersQuery.isLoading,
    jobs: [],
    hasNextPage: triggersQuery.hasNextPage,
    isFetchingNextPage: triggersQuery.isFetchingNextPage,
    kind: "triggers" as const,
    onSearchChange: page.setSearchQuery,
    onLoadMore: () => void triggersQuery.fetchNextPage(),
    onSelect: (id: string) =>
      startTransition(() => {
        page.setSelectedId(id);
      }),
    scopeFilter: page.scopeFilter,
    searchQuery: page.searchQuery,
    selectedId: effectiveSelectedTriggerId,
    totalCount: triggersQuery.total,
    triggers,
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
    onDelete: handleDelete,
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
          onCancel: () => {
            setSubmitError(null);
            setEditor(null);
          },
          onChange: (draft: CreateAutomationTriggerRequest) => {
            setSubmitError(null);
            setEditor(current => (current ? { ...current, draft } : current));
          },
          onSubmit: () => {
            void handleSubmit();
          },
          submitError,
        }
      : null,
  };

  return {
    currentTotalCount: triggersQuery.total,
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
