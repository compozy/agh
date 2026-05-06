import { useCallback, useEffect, useState } from "react";

import type { TaskExecutionProfile, TaskExecutionProfileSetRequest } from "../types";

interface UseProfileEditorState {
  taskId: string;
  profile: TaskExecutionProfile | null;
  onSetProfile: (data: TaskExecutionProfileSetRequest) => Promise<void>;
}

function buildEmptyProfile(taskId: string, now: string): TaskExecutionProfileSetRequest {
  return {
    task_id: taskId,
    coordinator: { mode: "inherit" },
    worker: { mode: "inherit" },
    review: {},
    sandbox: { mode: "inherit" },
    participants: {},
    created_at: now,
    updated_at: now,
  } as TaskExecutionProfileSetRequest;
}

export function useProfileEditor({ taskId, profile, onSetProfile }: UseProfileEditorState) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      return;
    }
    const seed = profile ?? buildEmptyProfile(taskId, new Date().toISOString());
    setValue(JSON.stringify(seed, null, 2));
    setError(null);
  }, [open, profile, taskId]);

  const submit = useCallback(async () => {
    setError(null);
    let parsed: TaskExecutionProfileSetRequest;
    try {
      const candidate = JSON.parse(value) as Partial<TaskExecutionProfileSetRequest>;
      if (typeof candidate !== "object" || candidate === null) {
        setError("Profile must be a JSON object.");
        return;
      }
      parsed = {
        ...candidate,
        task_id: candidate.task_id ?? taskId,
      } as TaskExecutionProfileSetRequest;
    } catch (parseError) {
      setError(
        parseError instanceof Error ? parseError.message : "Profile JSON could not be parsed."
      );
      return;
    }
    if (parsed.task_id !== taskId) {
      setError(`task_id must equal ${taskId}.`);
      return;
    }
    try {
      await onSetProfile(parsed);
      setOpen(false);
    } catch {
      // Caller surfaces the toast; keep dialog open so the operator can retry.
    }
  }, [onSetProfile, taskId, value]);

  return { open, setOpen, value, setValue, error, submit };
}
