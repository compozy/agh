import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { windowManagerKeys, type WindowManagerConfig } from "@/systems/os";

import { updateWindowManagerSettings } from "../adapters/window-manager-layouts-api";

function parseRatios(value: string): number[] | null {
  const parts = value.split(",").map(part => part.trim());
  const ratios = parts.map(Number);
  const canonicalRatios = ratios.map(ratio => Math.round(ratio * 1_000_000));
  return ratios.length >= 1 &&
    ratios.length <= 8 &&
    parts.every(part => part !== "") &&
    new Set(canonicalRatios).size === ratios.length &&
    ratios.every(ratio => ratio >= 0.1 && ratio <= 0.9)
    ? ratios
    : null;
}

function numericConfigIsValid(config: WindowManagerConfig): boolean {
  const gaps = Object.values(config.gaps);
  return (
    Number.isInteger(config.historyLimit) &&
    config.historyLimit >= 1 &&
    config.historyLimit <= 500 &&
    gaps.every(value => Number.isInteger(value) && value >= 0 && value <= 64) &&
    Number.isInteger(config.snap.edgeBand) &&
    config.snap.edgeBand >= 4 &&
    config.snap.edgeBand <= 128 &&
    Number.isInteger(config.snap.cornerReach) &&
    config.snap.cornerReach >= 16 &&
    config.snap.cornerReach <= 512 &&
    Number.isInteger(config.snap.exitSlack) &&
    config.snap.exitSlack >= 0 &&
    config.snap.exitSlack <= 64
  );
}

function parseShortcuts(value: string): Record<string, string> | null {
  try {
    const candidate: unknown = JSON.parse(value);
    if (candidate === null || typeof candidate !== "object" || Array.isArray(candidate)) {
      return null;
    }
    const entries = Object.entries(candidate);
    return entries.every(([, chord]) => typeof chord === "string")
      ? (Object.fromEntries(entries) as Record<string, string>)
      : null;
  } catch {
    return null;
  }
}

export function useWindowManagerConfigEditor(initialConfig: WindowManagerConfig) {
  const queryClient = useQueryClient();
  const [baseline, setBaseline] = useState(initialConfig);
  const [draft, setDraft] = useState(initialConfig);
  const [ratioText, setRatioText] = useState(() => initialConfig.snap.repeatRatios.join(", "));
  const [shortcutsText, setShortcutsText] = useState(() =>
    JSON.stringify(initialConfig.shortcuts, null, 2)
  );
  const ratios = parseRatios(ratioText);
  const shortcuts = parseShortcuts(shortcutsText);
  const numericConfigValid = numericConfigIsValid(draft);
  const dirty =
    JSON.stringify(draft) !== JSON.stringify(baseline) ||
    ratioText !== baseline.snap.repeatRatios.join(", ") ||
    shortcutsText !== JSON.stringify(baseline.shortcuts, null, 2);

  const save = useMutation({
    mutationFn: async () => {
      if (ratios === null || shortcuts === null) {
        throw new Error("Fix repeat ratios and shortcut JSON before saving.");
      }
      const next = {
        ...draft,
        snap: { ...draft.snap, repeatRatios: ratios },
        shortcuts,
      };
      await updateWindowManagerSettings(next);
      return next;
    },
    onSuccess: next => {
      setBaseline(next);
      setDraft(next);
      setRatioText(next.snap.repeatRatios.join(", "));
      setShortcutsText(JSON.stringify(next.shortcuts, null, 2));
      queryClient.setQueryData(windowManagerKeys.config(), next);
    },
  });

  const reset = () => {
    setDraft(baseline);
    setRatioText(baseline.snap.repeatRatios.join(", "));
    setShortcutsText(JSON.stringify(baseline.shortcuts, null, 2));
  };

  return {
    canSave:
      dirty && numericConfigValid && ratios !== null && shortcuts !== null && !save.isPending,
    dirty,
    draft,
    error: save.error,
    isSaving: save.isPending,
    ratioText,
    reset,
    save: () => save.mutate(),
    setDraft,
    setRatioText,
    setShortcutsText,
    shortcutsText,
    ratiosValid: ratios !== null,
    shortcutsValid: shortcuts !== null,
  };
}

export type WindowManagerConfigEditorModel = ReturnType<typeof useWindowManagerConfigEditor>;
