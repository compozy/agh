import { useCallback } from "react";
import type { ChangeEvent, FormEvent } from "react";

import type { CreateTaskDraftInput } from "@/hooks/routes/use-tasks-page";
import type { TaskOwnerKind, TaskPriority, TaskScope } from "../types";

interface UseTasksCreateModalFormParams {
  onDraftChange: (
    next: CreateTaskDraftInput | ((current: CreateTaskDraftInput) => CreateTaskDraftInput)
  ) => void;
  onSubmit: (draft: CreateTaskDraftInput, asDraft: boolean) => Promise<unknown> | void;
  draft: CreateTaskDraftInput;
}

export function useTasksCreateModalForm({
  draft,
  onDraftChange,
  onSubmit,
}: UseTasksCreateModalFormParams) {
  const updateText = useCallback(
    (field: keyof CreateTaskDraftInput) =>
      (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
        const value = event.target.value;
        onDraftChange(current => ({ ...current, [field]: value }));
      },
    [onDraftChange]
  );

  const updateScope = useCallback(
    (scope: TaskScope) => onDraftChange(current => ({ ...current, scope })),
    [onDraftChange]
  );

  const updatePriority = useCallback(
    (priority: TaskPriority) => onDraftChange(current => ({ ...current, priority })),
    [onDraftChange]
  );

  const updateOwnerKind = useCallback(
    (ownerKind: TaskOwnerKind | "") => onDraftChange(current => ({ ...current, ownerKind })),
    [onDraftChange]
  );

  const updateMaxAttempts = useCallback(
    (maxAttempts: number | null) => onDraftChange(current => ({ ...current, maxAttempts })),
    [onDraftChange]
  );

  const updateApprovalPolicy = useCallback(
    (approvalPolicy: "none" | "manual") =>
      onDraftChange(current => ({ ...current, approvalPolicy })),
    [onDraftChange]
  );

  const submitForm = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      void onSubmit(draft, false);
    },
    [draft, onSubmit]
  );

  const submitDraft = useCallback(() => {
    void onSubmit(draft, true);
  }, [draft, onSubmit]);

  return {
    submitDraft,
    submitForm,
    updateApprovalPolicy,
    updateMaxAttempts,
    updateOwnerKind,
    updatePriority,
    updateScope,
    updateText,
  };
}
