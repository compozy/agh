import { useRef } from "react";

import { cn } from "@agh/ui";

import { useReferenceAutocomplete } from "../../hooks/use-reference-autocomplete";
import type { LoopReferenceSuggestion } from "../../lib/loop-references";

interface LoopReferenceInputProps {
  value: string;
  onChange: (value: string) => void;
  suggestions: readonly LoopReferenceSuggestion[];
  multiline?: boolean;
  mono?: boolean;
  placeholder?: string;
  invalid?: boolean;
  /** CEL condition field: autocomplete triggers on a bare identifier, not `{{`. */
  cel?: boolean;
  ariaLabel?: string;
  testId?: string;
}

type ReferenceFieldElement = HTMLInputElement | HTMLTextAreaElement;

/**
 * A template-string input with the ADR-020 `{{ }}` reference picker: typing `{{ .`
 * surfaces the loop namespace (inputs, upstream node outputs, item/index, trigger,
 * generation) filtered by the partial path; selecting one inserts `{{ .path }}`. The
 * autocomplete state lives in `useReferenceAutocomplete`, so this stays presentational.
 */
export function LoopReferenceInput({
  value,
  onChange,
  suggestions,
  multiline = false,
  mono = false,
  placeholder,
  invalid = false,
  cel = false,
  ariaLabel,
  testId,
}: LoopReferenceInputProps) {
  const fieldRef = useRef<ReferenceFieldElement>(null);
  const auto = useReferenceAutocomplete(fieldRef, onChange, suggestions, cel);
  const setFieldRef = (element: ReferenceFieldElement | null) => {
    fieldRef.current = element;
  };

  const inputClass = cn(
    "w-full rounded-md border bg-elevated px-2.5 text-[12.5px] text-fg outline-none placeholder:text-faint focus:border-line-strong focus:bg-canvas",
    mono && "font-mono text-[12px]",
    invalid ? "border-danger" : "border-line",
    multiline ? "min-h-[74px] resize-y py-2 leading-relaxed" : "h-8"
  );

  const shared = {
    value,
    placeholder,
    onChange: auto.onChange,
    onKeyDown: auto.onKeyDown,
    onKeyUp: auto.onCaretMove,
    onClick: auto.onCaretMove,
    onBlur: auto.onBlur,
    className: inputClass,
    "aria-label": ariaLabel,
    "data-testid": testId,
  };

  return (
    <div className="relative">
      {multiline ? (
        <textarea ref={setFieldRef} {...shared} />
      ) : (
        <input type="text" ref={setFieldRef} {...shared} />
      )}
      {auto.matches.length > 0 ? (
        <ul
          className="absolute z-20 mt-1 max-h-52 w-full overflow-y-auto rounded-md border border-line-strong bg-elevated py-1 shadow-lg"
          data-testid="loop-reference-suggestions"
        >
          {auto.matches.map((match, index) => (
            <li key={match.path}>
              <button
                type="button"
                // Keep focus on the field so the caret survives the insert.
                onMouseDown={event => {
                  event.preventDefault();
                  auto.select(match.path);
                }}
                onMouseEnter={() => auto.setActiveIndex(index)}
                className={cn(
                  "flex w-full items-center justify-between gap-3 px-2.5 py-1 text-left hover:bg-row-hover",
                  index === auto.activeIndex && "bg-row-hover"
                )}
                data-active={index === auto.activeIndex ? "true" : "false"}
              >
                <span className="font-mono text-[11px] text-fg-strong">{match.path}</span>
                <span className="truncate text-[10px] text-subtle">{match.detail}</span>
              </button>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

export type { LoopReferenceSuggestion };
