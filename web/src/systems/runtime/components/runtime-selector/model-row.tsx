import { Brain, Check, Star, Wrench } from "lucide-react";
import { Fragment, type ReactElement } from "react";

import { cn, KindIcon, providerKindIconRegistry } from "@agh/ui";

import type { RuntimeModelOption } from "./types";

function formatContext(tokens: number | null | undefined): string | null {
  if (tokens == null) return null;
  if (tokens >= 1_000_000) return `${tokens / 1_000_000}M`;
  if (tokens >= 1000) return `${Math.round(tokens / 1000)}k`;
  return String(tokens);
}

function formatCost(model: RuntimeModelOption): string | null {
  if (model.cost_input == null) return null;
  const out = model.cost_output ?? model.cost_input;
  return `$${model.cost_input}/${out}`;
}

function DotSep() {
  return <span aria-hidden="true" className="size-0.5 shrink-0 rounded-full bg-faint" />;
}

function buildChips(model: RuntimeModelOption): ReactElement[] {
  const chips: ReactElement[] = [];
  const ctx = formatContext(model.context_window);
  if (ctx)
    chips.push(
      <span key="ctx" className="font-mono text-badge text-subtle tabular-nums">
        {ctx}
      </span>
    );
  const cost = formatCost(model);
  if (cost) {
    chips.push(
      <span key="cost" className="font-mono text-badge text-subtle tabular-nums">
        {cost}
      </span>
    );
  }
  if (model.supports_tools) {
    chips.push(
      <span key="tools" className="inline-flex items-center gap-1 font-mono text-badge text-subtle">
        <Wrench aria-hidden="true" className="size-[11px] text-faint" />
        tools
      </span>
    );
  }
  if (model.efforts.length > 0) {
    chips.push(
      <span
        key="rz"
        className="inline-flex items-center gap-1 font-mono text-badge text-accent-strong"
      >
        <Brain aria-hidden="true" className="size-[11px] text-accent-strong" />
        {model.efforts.length} levels
      </span>
    );
  } else if (model.supports_reasoning) {
    chips.push(
      <span key="rz" className="inline-flex items-center gap-1 font-mono text-badge text-subtle">
        <Brain aria-hidden="true" className="size-[11px] text-faint" />
        reasoning
      </span>
    );
  }
  return chips;
}

export interface ModelRowProps {
  /** DOM id used for the combobox `aria-activedescendant` relationship. */
  id: string;
  model: RuntimeModelOption;
  providerName: string;
  /** Icon key from the owning provider option (`runtime_provider` or id). */
  iconKind: string;
  selected: boolean;
  favorite: boolean;
  highlighted: boolean;
  onSelect: (provider: string, id: string) => void;
  /** Pointer hover makes this the active row (so the external favorite action targets it). */
  onHover: () => void;
}

/**
 * One `role="option"` in the models listbox. A listbox option MUST NOT wrap a
 * focusable/interactive control, nor carry `aria-keyshortcuts` (that belongs on
 * the real external favorite button), so this row is pure: selection is its only
 * option-level action. The favorite star is a NON-interactive `aria-hidden`,
 * pointer-inert indicator — it shows favorite state visually but is never
 * clickable and never a fake `role="button"` span. Favoriting is a real external
 * control (the footer favorite button + its `Alt+F` shortcut) acting on the
 * active row; the option's accessible name still carries the current state via
 * the visually-hidden "Favorited" text. Pointer hover activates the row so that
 * external control targets whatever the cursor is over.
 */
export function ModelRow({
  id,
  model,
  providerName,
  iconKind,
  selected,
  favorite,
  highlighted,
  onSelect,
  onHover,
}: ModelRowProps) {
  const disabled = Boolean(model.disabled);
  const chips = buildChips(model);

  return (
    <div
      role="option"
      id={id}
      aria-selected={selected}
      aria-disabled={disabled || undefined}
      tabIndex={-1}
      data-provider={model.provider}
      data-model={model.id}
      data-selected={selected ? "true" : "false"}
      data-disabled={disabled ? "true" : "false"}
      data-highlighted={highlighted ? "true" : "false"}
      data-favorite={favorite ? "true" : "false"}
      className={cn(
        "group flex w-full items-center gap-[11px] rounded-md px-2.5 py-2 text-left transition-colors",
        disabled ? "cursor-not-allowed opacity-50" : "cursor-pointer hover:bg-row-hover",
        highlighted && !disabled && "bg-row-hover ring-1 ring-line-strong ring-inset",
        selected && "bg-accent-tint"
      )}
      onMouseEnter={disabled ? undefined : onHover}
      onClick={event => {
        if (disabled) return;
        event.preventDefault();
        onSelect(model.provider, model.id);
      }}
    >
      <span
        className={cn(
          "grid size-7 shrink-0 place-items-center overflow-hidden rounded-sm bg-elevated p-[5px] ring-1 ring-inset",
          selected ? "ring-accent-dim" : "ring-line-soft"
        )}
      >
        <KindIcon
          kind={iconKind}
          registry={providerKindIconRegistry}
          size="sm"
          tone="default"
          className="size-full"
        />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate text-small-body font-medium text-fg-strong">{model.name}</span>
          {favorite ? <span className="sr-only">, Favorited</span> : null}
        </span>
        <span className="mt-0.5 flex flex-wrap items-center gap-[7px] text-badge text-faint">
          <span className="text-subtle">{providerName}</span>
          {chips.length > 0 ? <DotSep /> : null}
          {chips.map((chip, index) => (
            <Fragment key={chip.key}>
              {index > 0 ? <DotSep /> : null}
              {chip}
            </Fragment>
          ))}
        </span>
      </span>
      <span className="flex shrink-0 items-center gap-2.5">
        {disabled ? (
          <span className="text-badge font-medium whitespace-nowrap text-warning">
            {model.disabled_reason ?? "Unavailable"}
          </span>
        ) : null}
        {/* Non-color structural cue for selection (in addition to aria-selected + tint). */}
        {selected ? (
          <Check
            aria-hidden="true"
            className="size-3.5 shrink-0 text-accent-strong"
            data-selected-check="true"
          />
        ) : null}
        {/* Decorative favorite-state indicator only — never interactive. The real
            favorite control is the footer button + Alt+F acting on the active row. */}
        <span
          aria-hidden="true"
          data-favorite-indicator={favorite ? "true" : "false"}
          className={cn(
            "pointer-events-none grid size-5 place-items-center rounded text-faint transition-opacity",
            favorite ? "text-warning opacity-100" : "opacity-0"
          )}
        >
          <Star aria-hidden="true" className={cn("size-3.5", favorite && "fill-current")} />
        </span>
      </span>
    </div>
  );
}
