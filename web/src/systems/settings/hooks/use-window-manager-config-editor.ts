import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { windowManagerKeys, type WindowManagerConfig } from "@/systems/os";

import { updateWindowManagerSettings } from "../adapters/window-manager-layouts-api";

function parseRatios(value: string): number[] | null {
  const ratios = value
    .split(",")
    .map(part => Number(part.trim()))
    .filter(value => Number.isFinite(value));
  return ratios.length > 0 && ratios.every(ratio => ratio >= 0.1 && ratio <= 0.9) ? ratios : null;
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
    canSave: dirty && ratios !== null && shortcuts !== null && !save.isPending,
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
