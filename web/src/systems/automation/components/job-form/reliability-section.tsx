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
  locked: boolean;
  mode: "create" | "edit";
  badge: string;
  defaultOpen: boolean;
  onRetryChange: (retry: AutomationRetry) => void;
  onFireLimitChange: (fireLimit: AutomationFireLimit) => void;
  onEnabledChange: (enabled: boolean) => void;
}

/**
 * Collapsible reliability & limits controls (retry, rate limit, enabled). When a
 * job delegates to a task, the task owns retries — `locked` disables the retry
 * controls and surfaces an "owned by the task" hint.
 */
export function ReliabilitySection({
  retry,
  fireLimit,
  enabled,
  locked,
  mode,
  badge,
  defaultOpen,
  onRetryChange,
  onFireLimitChange,
  onEnabledChange,
}: ReliabilitySectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  const isBackoff = retry.strategy === "backoff";
  const retryDisabled = locked || !isBackoff;

  return (
    <Collapsible className="mt-5 border-t border-line-soft pt-1" onOpenChange={setOpen} open={open}>
      <CollapsibleTrigger
        className="flex w-full items-center gap-2 py-2.5 text-left outline-none"
        data-testid="job-governance-toggle"
        type="button"
      >
        <ChevronRight
          aria-hidden="true"
          className={cn("size-4 text-muted transition-transform", open && "rotate-90")}
        />
        <span className="flex-1 text-small-body font-semibold text-fg-strong">
          Reliability &amp; limits
        </span>
        <span className="font-mono text-form-hint text-subtle">{badge}</span>
      </CollapsibleTrigger>

      <CollapsibleContent className="grid grid-cols-2 gap-x-4 gap-y-4 pt-2 pb-1">
        <Field className="col-span-2">
          <FieldTitle>
            Retry policy
            {locked ? <span className="font-normal text-faint">(owned by the task)</span> : null}
          </FieldTitle>
          <PillGroup
            aria-label="Retry policy"
            items={[
              {
                value: "none",
                label: "None",
                testId: "job-retry-strategy-none",
                disabled: locked,
              },
              {
                value: "backoff",
                label: "Exponential backoff",
                testId: "job-retry-strategy-backoff",
                disabled: locked,
              },
            ]}
            onChange={next => onRetryChange(retryDraftForStrategy(next, retry))}
            size="sm"
            value={retry.strategy}
          />
        </Field>
        <Field>
          <FieldLabel htmlFor="job-retry-max">Max retries</FieldLabel>
          <Input
            className="font-mono"
            data-testid="job-retry-max"
            disabled={retryDisabled}
            id="job-retry-max"
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
          <FieldLabel htmlFor="job-retry-delay">Base delay</FieldLabel>
          <Input
            className="font-mono"
            data-testid="job-retry-delay"
            disabled={retryDisabled}
            id="job-retry-delay"
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
          <FieldLabel htmlFor="job-fire-limit-max">Max fires</FieldLabel>
          <Input
            className="font-mono"
            data-testid="job-fire-limit-max"
            id="job-fire-limit-max"
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
          <FieldLabel htmlFor="job-fire-limit-window">Per window</FieldLabel>
          <Input
            className="font-mono"
            data-testid="job-fire-limit-window"
            id="job-fire-limit-window"
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
            data-testid="job-enabled-toggle"
            onCheckedChange={onEnabledChange}
          />
          <FieldContent>
            <FieldTitle>{mode === "create" ? "Enabled on create" : "Enabled"}</FieldTitle>
            <FieldDescription>
              Disabled jobs stay stored but never dispatch on their schedule.
            </FieldDescription>
          </FieldContent>
        </Field>
      </CollapsibleContent>
    </Collapsible>
  );
}
