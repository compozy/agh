import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { windowManagerKeys } from "@/systems/os";

import { applyWindowManagerLayout } from "../adapters/window-manager-layouts-api";
import { parseWindowManagerLayoutDocument } from "../lib/window-manager-layout-schema";
import {
  windowManagerLayoutFingerprint,
  windowManagerLayoutReviewOptions,
} from "../lib/window-manager-layout-query";
import type {
  WindowManagerLayoutDocument,
  WindowManagerLayoutPreview,
  WindowManagerLayoutState,
  WindowManagerLayoutValidation,
} from "../lib/window-manager-layout-types";

interface ReviewedLayout {
  fingerprint: string;
  preview: WindowManagerLayoutPreview;
}

export function useWindowManagerLayoutEditor(
  workspaceId: string,
  initial: WindowManagerLayoutState
) {
  const queryClient = useQueryClient();
  const [revision, setRevision] = useState(initial.revision);
  const [baseline, setBaseline] = useState(initial.document);
  const [draft, setDraft] = useState(initial.document);
  const [importError, setImportError] = useState<string | null>(null);
  const currentFingerprint = windowManagerLayoutFingerprint(draft);
  const dirty = currentFingerprint !== windowManagerLayoutFingerprint(baseline);

  const updateDraft = (next: WindowManagerLayoutDocument) => {
    setDraft(next);
  };

  const review = useQuery(windowManagerLayoutReviewOptions(workspaceId, revision, draft));
  const reviewResult = review.data?.fingerprint === currentFingerprint ? review.data : null;
  const validation: WindowManagerLayoutValidation | null = reviewResult?.validation ?? null;
  const reviewed: ReviewedLayout | null =
    reviewResult?.preview == null
      ? null
      : { fingerprint: reviewResult.fingerprint, preview: reviewResult.preview };
  const reviewCurrent = reviewed?.fingerprint === currentFingerprint;

  const apply = useMutation({
    mutationFn: async () => {
      const candidate = structuredClone(draft);
      const result = await applyWindowManagerLayout(workspaceId, revision, candidate);
      return { candidate, result };
    },
    onSuccess: async ({ candidate, result }) => {
      setRevision(result.revision);
      setBaseline(candidate);
      await queryClient.invalidateQueries({
        queryKey: windowManagerKeys.snapshot(workspaceId),
      });
    },
  });

  const importDocument = async (file: File) => {
    try {
      const value: unknown = JSON.parse(await file.text());
      const imported = parseWindowManagerLayoutDocument(value);
      updateDraft({ ...imported, workspaceId });
      setImportError(null);
    } catch (error) {
      setImportError(
        error instanceof Error ? error.message : "The selected file is not a layout document."
      );
    }
  };

  const setGroupRoot = (
    desktopIndex: number,
    groupIndex: number,
    root: WindowManagerLayoutDocument["desktops"][number]["groups"][number]["root"]
  ) => {
    const desktops = structuredClone(draft.desktops);
    const group = desktops[desktopIndex]?.groups[groupIndex];
    if (!group) return;
    group.root = root;
    updateDraft({ ...draft, desktops });
  };

  const setGroupFrame = (
    desktopIndex: number,
    groupIndex: number,
    key: "x" | "y" | "w" | "h",
    value: number
  ) => {
    const desktops = structuredClone(draft.desktops);
    const group = desktops[desktopIndex]?.groups[groupIndex];
    if (!group) return;
    group.frame[key] = value;
    updateDraft({ ...draft, desktops });
  };

  return {
    apply,
    dirty,
    draft,
    importDocument,
    importError,
    mutationError: review.error ?? apply.error,
    reset: () => updateDraft(structuredClone(baseline)),
    review,
    reviewed,
    reviewCurrent,
    revision,
    setGroupFrame,
    setGroupRoot,
    updateDraft,
    validation,
  };
}

export type WindowManagerLayoutEditorModel = ReturnType<typeof useWindowManagerLayoutEditor>;
