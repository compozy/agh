import { ChevronDown, TriangleAlert } from "lucide-react";
import { forwardRef, type ComponentProps, type ReactNode } from "react";

import { cn, KindIcon, providerKindIconRegistry } from "@agh/ui";

import { IntensityMeter } from "./intensity-meter";
import {
  reasoningEffortLabel,
  reasoningEffortPosition,
  resolveReasoningState,
  type RuntimeModelOption,
  type RuntimeProviderOption,
  type RuntimeSelectorValue,
  type RuntimeSelectorVariant,
} from "./types";

export type TriggerFocus = "provider" | "model" | "reasoning";

export interface RuntimeSelectorTriggerProps extends Omit<ComponentProps<"div">, "onSelect"> {
  value: RuntimeSelectorValue;
  provider: RuntimeProviderOption | undefined;
  model: RuntimeModelOption | undefined;
  variant?: RuntimeSelectorVariant;
  open?: boolean;
  needsAuth?: boolean;
  disabled?: boolean;
  readOnly?: boolean;
  modelPlaceholder?: string;
  /** id of the popup dialog the segments control (announced via aria-controls). */
  popupId?: string;
  /**
   * id of the surface's visible field caption. When set the group is named by it
   * (truthful `aria-labelledby`); otherwise the group falls back to its own
   * summarised `aria-label`. Surfaces must NOT point a `<label htmlFor>` at this
   * group — a `<div role="group">` is not a labelable element.
   */
  ariaLabelledby?: string;
  onSegment?: (focus: TriggerFocus) => void;
}

function providerKind(
  provider: RuntimeProviderOption | undefined,
  value: RuntimeSelectorValue
): string {
  return provider?.runtime_provider ?? provider?.id ?? value.provider;
}

const SEGMENT_CLASS =
  "relative inline-flex h-full items-center gap-2 px-3 outline-none transition-colors hover:bg-btn-default-hover focus-visible:z-10 focus-visible:bg-btn-default-hover focus-visible:ring-2 focus-visible:ring-accent focus-visible:ring-inset disabled:cursor-not-allowed";

export const RuntimeSelectorTrigger = forwardRef<HTMLDivElement, RuntimeSelectorTriggerProps>(
  function RuntimeSelectorTrigger(
    {
      value,
      provider,
      model,
      variant = "default",
      open = false,
      needsAuth = false,
      disabled = false,
      readOnly = false,
      modelPlaceholder = "Select model",
      popupId,
      ariaLabelledby,
      onSegment,
      className,
      ...props
    },
    ref
  ) {
    const compact = variant === "compact";
    const reasoning = resolveReasoningState(model);
    const showReasoning = reasoning.mode === "levels" && !compact;
    const reasoningUnset = value.reasoning_effort === "";
    const currentEffort = value.reasoning_effort || reasoning.defaultEffort;
    const showWarning = needsAuth || model?.availability === "unavailable";
    const warningLabel = needsAuth ? "Provider needs sign in" : "Model unavailable";
    const providerName = provider?.name || value.provider;
    // `||` not `??`: an unset model is "" (not nullish), so the placeholder must
    // still win — otherwise the model segment renders blank in the no-model state.
    const modelName = model?.name || value.model || modelPlaceholder;

    const segment = (focus: TriggerFocus, key: string, content: ReactNode, label: string) => (
      <button
        key={key}
        type="button"
        disabled={disabled}
        aria-disabled={readOnly || undefined}
        aria-label={label}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={open ? popupId : undefined}
        data-focus={focus}
        className={cn(
          SEGMENT_CLASS,
          // The small variant tightens segment density to match the design (not just height).
          variant === "small" && "gap-[7px] px-2.5",
          key === "provider" && "rounded-l-md"
        )}
        onClick={event => {
          event.stopPropagation();
          if (!disabled && !readOnly) onSegment?.(focus);
        }}
      >
        {content}
      </button>
    );

    const segments: ReactNode[] = [];
    segments.push(
      segment(
        "provider",
        "provider",
        <>
          <span className="grid size-5 shrink-0 place-items-center overflow-hidden rounded-xs bg-canvas p-[3px] ring-1 ring-line ring-inset">
            <KindIcon
              kind={providerKind(provider, value)}
              registry={providerKindIconRegistry}
              size="sm"
              tone="default"
              className="size-full"
            />
          </span>
          {compact ? null : (
            <span className="text-small-body font-medium text-fg">{providerName}</span>
          )}
        </>,
        `Provider: ${providerName}`
      )
    );

    if (!compact) {
      segments.push(
        segment(
          "model",
          "model",
          <span className="max-w-[150px] truncate text-small-body font-medium text-fg-strong">
            {modelName}
          </span>,
          `Model: ${modelName}`
        )
      );
    }

    if (showReasoning) {
      segments.push(
        segment(
          "reasoning",
          "reasoning",
          <span className="inline-flex items-center gap-[7px]">
            <IntensityMeter
              position={reasoningUnset ? 0 : reasoningEffortPosition(currentEffort)}
              hollow={reasoningUnset}
            />
            <span className="text-small-body font-medium text-muted">
              {reasoningUnset ? "Default" : reasoningEffortLabel(currentEffort)}
            </span>
          </span>,
          `Reasoning: ${reasoningUnset ? "provider default" : reasoningEffortLabel(currentEffort)}`
        )
      );
    }

    const withDividers: ReactNode[] = [];
    segments.forEach((node, index) => {
      if (index > 0) {
        withDividers.push(
          <span
            key={`divider-${index}`}
            aria-hidden="true"
            className="my-[7px] w-px shrink-0 self-stretch bg-line-strong"
          />
        );
      }
      withDividers.push(node);
    });

    const summary = compact
      ? `Runtime: ${providerName}`
      : `Runtime: ${providerName} / ${modelName}`;

    return (
      // A labelled button group — each segment is a real button that opens the
      // dialog with a deep-link focus intent. `tabIndex={-1}` makes the group a
      // programmatic focus target for Escape restoration (finalFocus), never a
      // Tab stop and never a `role="button"` wrapper around the segment buttons.
      <div
        ref={ref}
        role="group"
        aria-labelledby={ariaLabelledby}
        aria-label={ariaLabelledby ? undefined : summary}
        tabIndex={-1}
        aria-disabled={disabled || readOnly || undefined}
        data-open={open ? "true" : "false"}
        data-variant={variant}
        className={cn(
          "inline-flex select-none items-stretch rounded-md border bg-elevated shadow-highlight transition-colors",
          variant === "small" ? "h-[30px]" : "h-[34px]",
          open ? "border-accent-dim" : "border-line-strong",
          (disabled || readOnly) && "cursor-not-allowed opacity-60",
          className
        )}
        {...props}
      >
        {withDividers}
        {showWarning ? (
          <span
            role="img"
            aria-label={warningLabel}
            title={warningLabel}
            className="grid place-items-center pl-1 text-warning"
          >
            <TriangleAlert aria-hidden="true" className="size-3.5" />
          </span>
        ) : null}
        <button
          type="button"
          disabled={disabled}
          aria-disabled={readOnly || undefined}
          aria-label="Open runtime selector"
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-controls={open ? popupId : undefined}
          data-focus="model"
          className={cn(SEGMENT_CLASS, "rounded-r-md px-[9px] text-faint", open && "text-fg")}
          onClick={event => {
            event.stopPropagation();
            if (!disabled && !readOnly) onSegment?.(compact ? "provider" : "model");
          }}
        >
          <ChevronDown
            aria-hidden="true"
            className={cn("size-3.5 transition-transform", open && "rotate-180")}
          />
        </button>
      </div>
    );
  }
);
