import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";

import { windowManagerKeys } from "@/systems/os";

import {
  applyWindowManagerLayout,
  previewWindowManagerLayout,
  validateWindowManagerLayout,
} from "../adapters/window-manager-layouts-api";
import {
  parseWindowManagerLayoutDocument,
  windowManagerLayoutDocumentToWire,
} from "../lib/window-manager-layout-schema";
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

function fingerprint(document: WindowManagerLayoutDocument): string {
  return JSON.stringify(windowManagerLayoutDocumentToWire(document));
}

export function useWindowManagerLayoutEditor(
  workspaceId: string,
  initial: WindowManagerLayoutState
) {
  const queryClient = useQueryClient();
  const importInput = useRef<HTMLInputElement | null>(null);
  const [revision, setRevision] = useState(initial.revision);
  const [baseline, setBaseline] = useState(initial.document);
  const [draft, setDraft] = useState(initial.document);
  const draftFingerprint = useRef(fingerprint(initial.document));
  const [reviewed, setReviewed] = useState<ReviewedLayout | null>(null);
  const [validation, setValidation] = useState<WindowManagerLayoutValidation | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const currentFingerprint = fingerprint(draft);
  const dirty = currentFingerprint !== fingerprint(baseline);
  const reviewCurrent = reviewed?.fingerprint === currentFingerprint;

  const updateDraft = (next: WindowManagerLayoutDocument) => {
    draftFingerprint.current = fingerprint(next);
    setDraft(next);
    setReviewed(null);
    setValidation(null);
  };

  const review = useMutation({
    mutationFn: async () => {
      const candidate = structuredClone(draft);
      const candidateFingerprint = fingerprint(candidate);
      const checked = await validateWindowManagerLayout(workspaceId, candidate);
      if (!checked.valid) {
        return { checked, fingerprint: candidateFingerprint, preview: null };
      }
      const preview = await previewWindowManagerLayout(workspaceId, revision, candidate);
      return { checked, fingerprint: candidateFingerprint, preview };
    },
    onSuccess: result => {
      if (result.fingerprint !== draftFingerprint.current) return;
      setValidation(result.checked);
      setReviewed(
        result.preview === null
          ? null
          : {
              fingerprint: result.fingerprint,
              preview: result.preview,
            }
      );
    },
  });

  const apply = useMutation({
    mutationFn: async () => {
      const candidate = structuredClone(draft);
      const result = await applyWindowManagerLayout(workspaceId, revision, candidate);
      return { candidate, result };
    },
    onSuccess: async ({ candidate, result }) => {
      setRevision(result.revision);
      setBaseline(candidate);
      setReviewed(null);
      setValidation(null);
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
    importInput,
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
