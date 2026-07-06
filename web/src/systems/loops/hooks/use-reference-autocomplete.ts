import { useEffect, useLayoutEffect, useRef, useState } from "react";

import {
  activeReferenceQuery,
  filterReferences,
  type LoopReferenceSuggestion,
} from "../lib/loop-references";

const MAX_SUGGESTIONS = 8;

type FieldElement = HTMLInputElement | HTMLTextAreaElement;

function insertTemplateReference(
  value: string,
  caret: number,
  path: string
): { value: string; caret: number } {
  const open = value.slice(0, caret).lastIndexOf("{{");
  if (open === -1) return { value, caret };
  const head = value.slice(0, open);
  const tail = value.slice(caret);
  const inserted = `{{ .${path} }}`;
  return { value: `${head}${inserted}${tail}`, caret: head.length + inserted.length };
}

function insertCelReference(
  value: string,
  caret: number,
  path: string
): { value: string; caret: number } {
  const match = value.slice(0, caret).match(/[A-Za-z_][A-Za-z0-9_.]*$/);
  const start = match ? caret - match[0].length : caret;
  const head = value.slice(0, start);
  const tail = value.slice(caret);
  return { value: `${head}${path}${tail}`, caret: head.length + path.length };
}

export interface ReferenceAutocomplete {
  ref: React.RefObject<FieldElement | null>;
  matches: LoopReferenceSuggestion[];
  onChange: (event: React.ChangeEvent<FieldElement>) => void;
  onCaretMove: (event: { currentTarget: FieldElement }) => void;
  onBlur: () => void;
  select: (path: string) => void;
}

/**
 * The ADR-020 `{{ }}` reference picker state machine, extracted so the input component
 * stays presentational: it tracks the active `{{ .partial` fragment under the caret,
 * filters the loop namespace, inserts a chosen path, and restores the caret. Authoring
 * UX only — it never blocks a keystroke; the linter owns reference resolution.
 */
export function useReferenceAutocomplete(
  onValueChange: (value: string) => void,
  suggestions: readonly LoopReferenceSuggestion[],
  cel = false
): ReferenceAutocomplete {
  const ref = useRef<FieldElement>(null);
  const blurTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const [query, setQuery] = useState<string | null>(null);
  const [pendingCaret, setPendingCaret] = useState<number | null>(null);

  const matches =
    query === null ? [] : filterReferences(suggestions, query).slice(0, MAX_SUGGESTIONS);

  // Clear the pending blur-dismiss timer on unmount (avoids a no-op setState after unmount).
  useEffect(
    () => () => {
      if (blurTimerRef.current) clearTimeout(blurTimerRef.current);
    },
    []
  );

  useLayoutEffect(() => {
    if (pendingCaret !== null && ref.current) {
      ref.current.focus();
      ref.current.setSelectionRange(pendingCaret, pendingCaret);
      setPendingCaret(null);
    }
  }, [pendingCaret]);

  useEffect(() => {
    if (query === null) return;
    const onEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setQuery(null);
    };
    window.addEventListener("keydown", onEscape);
    return () => window.removeEventListener("keydown", onEscape);
  }, [query]);

  const refresh = (element: FieldElement) =>
    setQuery(
      activeReferenceQuery(
        element.value,
        element.selectionStart ?? element.value.length,
        cel ? "cel" : "template"
      )
    );

  return {
    ref,
    matches,
    onChange: event => {
      onValueChange(event.target.value);
      refresh(event.target);
    },
    onCaretMove: event => refresh(event.currentTarget),
    onBlur: () => {
      if (blurTimerRef.current) clearTimeout(blurTimerRef.current);
      blurTimerRef.current = setTimeout(() => setQuery(null), 120);
    },
    select: path => {
      const element = ref.current;
      if (!element) return;
      const caret = element.selectionStart ?? element.value.length;
      const next = cel
        ? insertCelReference(element.value, caret, path)
        : insertTemplateReference(element.value, caret, path);
      onValueChange(next.value);
      setPendingCaret(next.caret);
      setQuery(null);
    },
  };
}
