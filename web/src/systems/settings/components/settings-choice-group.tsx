import { Check } from "lucide-react";

import { cn } from "@agh/ui";

export interface SettingsChoiceOption<V extends string> {
  value: V;
  /** Humanized name ("Ask first"). */
  name: string;
  /** One-line consequence. */
  description: string;
}

export interface SettingsChoiceGroupProps<V extends string> {
  options: ReadonlyArray<SettingsChoiceOption<V>>;
  value: V;
  onChange: (next: V) => void;
  ariaLabel: string;
  "data-testid"?: string;
}

/**
 * Choice cards for decisions with real consequences (design system §06).
 * Selection state is neutral (glaze + inset ring + check) — accent is not a
 * selection color. The machine value whispers at the card foot.
 */
export function SettingsChoiceGroup<V extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  "data-testid": testId,
}: SettingsChoiceGroupProps<V>) {
  return (
    <div
      aria-label={ariaLabel}
      className="grid gap-2 p-3 sm:grid-cols-3"
      data-testid={testId}
      role="radiogroup"
    >
      {options.map(option => {
        const checked = option.value === value;
        return (
          <button
            aria-checked={checked}
            className={cn(
              "relative flex flex-col gap-1.5 rounded-md border border-line-soft bg-canvas-tint px-3 py-2.5 text-left",
              "transition-colors duration-base hover:border-line hover:bg-elevated",
              "focus-visible:outline-none focus-visible:shadow-focus-ring",
              checked &&
                "border-transparent bg-surface-glaze shadow-[inset_0_0_0_1px_var(--color-line-strong)]"
            )}
            data-testid={testId ? `${testId}-${option.value}` : undefined}
            key={option.value}
            onClick={() => onChange(option.value)}
            role="radio"
            type="button"
          >
            <span className="flex items-center gap-2">
              <span className="flex-1 text-small-body font-medium text-fg-strong">
                {option.name}
              </span>
              <Check
                aria-hidden="true"
                className={cn(
                  "size-3.5 shrink-0 text-fg-strong transition-opacity duration-base",
                  checked ? "opacity-100" : "opacity-0"
                )}
              />
            </span>
            <span className="text-form-label leading-snug text-muted">{option.description}</span>
            <span className="font-mono text-micro text-faint">{option.value}</span>
          </button>
        );
      })}
    </div>
  );
}
