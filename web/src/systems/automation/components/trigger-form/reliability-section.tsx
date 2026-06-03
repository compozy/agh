import { useState } from "react";
import { ChevronRight } from "lucide-react";

import {
  cn,
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
  Field,
  FieldContent,
  FieldDescription,
  FieldLabel,
  FieldTitle,
  Input,
  PillGroup,
  Switch,
} from "@agh/ui";

import { retryDraftForStrategy } from "../../lib/automation-drafts";
import type { AutomationFireLimit, AutomationRetry } from "../../types";

interface ReliabilitySectionProps {
  retry: AutomationRetry;
  fireLimit: AutomationFireLimit | undefined;
  enabled: boolean;
  badge: string;
  defaultOpen: boolean;
  onRetryChange: (retry: AutomationRetry) => void;
  onFireLimitChange: (fireLimit: AutomationFireLimit) => void;
  onEnabledChange: (enabled: boolean) => void;
}

/** Collapsible reliability & state controls (retry, rate limit, enabled). */
export function ReliabilitySection({
  retry,
  fireLimit,
  enabled,
  badge,
  defaultOpen,
  onRetryChange,
  onFireLimitChange,
  onEnabledChange,
}: ReliabilitySectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  const isBackoff = retry.strategy === "backoff";

  return (
    <Collapsible className="mt-5 border-t border-line-soft pt-1" onOpenChange={setOpen} open={open}>
      <CollapsibleTrigger
        className="flex w-full items-center gap-2 py-2.5 text-left outline-none"
        data-testid="trigger-governance-toggle"
        type="button"
      >
        <ChevronRight
          aria-hidden="true"
          className={cn("size-4 text-muted transition-transform", open && "rotate-90")}
        />
        <span className="flex-1 text-small-body font-semibold text-fg-strong">
          Reliability &amp; state
        </span>
        <span className="font-mono text-form-hint text-subtle">{badge}</span>
      </CollapsibleTrigger>

      <CollapsibleContent className="grid grid-cols-2 gap-x-4 gap-y-4 pt-2 pb-1">
        <Field className="col-span-2">
          <FieldTitle>Retry policy</FieldTitle>
          <PillGroup
            aria-label="Retry policy"
            items={[
              { value: "none", label: "None", testId: "trigger-retry-strategy-none" },
              {
                value: "backoff",
                label: "Exponential backoff",
                testId: "trigger-retry-strategy-backoff",
              },
            ]}
            onChange={next => onRetryChange(retryDraftForStrategy(next, retry))}
            size="sm"
            value={retry.strategy}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="trigger-retry-max">Max retries</FieldLabel>
          <Input
            className="font-mono"
            data-testid="trigger-retry-max"
            disabled={!isBackoff}
            id="trigger-retry-max"
            min={0}
            onChange={event =>
              onRetryChange({
                ...retryDraftForStrategy("backoff", retry),
                max_retries: Number(event.target.value || "0"),
              })
            }
            type="number"
            value={isBackoff ? retry.max_retries : 0}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="trigger-retry-delay">Base delay</FieldLabel>
          <Input
            className="font-mono"
            data-testid="trigger-retry-delay"
            disabled={!isBackoff}
            id="trigger-retry-delay"
            onChange={event =>
              onRetryChange({
                ...retryDraftForStrategy("backoff", retry),
                base_delay: event.target.value,
              })
            }
            placeholder="2s"
            value={isBackoff ? retry.base_delay : ""}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="trigger-fire-limit-max">Max fires</FieldLabel>
          <Input
            className="font-mono"
            data-testid="trigger-fire-limit-max"
            id="trigger-fire-limit-max"
            min={1}
            onChange={event =>
              onFireLimitChange({
                ...(fireLimit ?? { window: "1h", max: 12 }),
                max: Number(event.target.value || "1"),
              })
            }
            type="number"
            value={fireLimit?.max ?? 12}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="trigger-fire-limit-window">Per window</FieldLabel>
          <Input
            className="font-mono"
            data-testid="trigger-fire-limit-window"
            id="trigger-fire-limit-window"
            onChange={event =>
              onFireLimitChange({
                ...(fireLimit ?? { max: 12, window: "1h" }),
                window: event.target.value,
              })
            }
            placeholder="1h"
            value={fireLimit?.window ?? "1h"}
          />
        </Field>
        <Field className="col-span-2" orientation="horizontal">
          <Switch
            checked={enabled}
            data-testid="trigger-enabled-toggle"
            onCheckedChange={onEnabledChange}
          />
          <FieldContent>
            <FieldTitle>Trigger enabled</FieldTitle>
            <FieldDescription>Disabled triggers stay registered but never fire.</FieldDescription>
          </FieldContent>
        </Field>
      </CollapsibleContent>
    </Collapsible>
  );
}
