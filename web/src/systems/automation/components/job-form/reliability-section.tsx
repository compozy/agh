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
  type PillGroupItem,
  Switch,
} from "@agh/ui";

import { retryDraftForStrategy } from "../../lib/automation-drafts";
import type { AutomationCatchUpPolicy, AutomationFireLimit, AutomationRetry } from "../../types";

/** Sentinel for the target-aware default: the daemon picks the policy, so the request omits it. */
const CATCH_UP_DEFAULT = "default";
type CatchUpChoice = AutomationCatchUpPolicy | typeof CATCH_UP_DEFAULT;

const CATCH_UP_ITEMS: PillGroupItem<CatchUpChoice>[] = [
  { value: CATCH_UP_DEFAULT, label: "Default", testId: "job-catch-up-default" },
  { value: "skip_missed", label: "Skip missed", testId: "job-catch-up-skip-missed" },
  { value: "coalesce", label: "Coalesce", testId: "job-catch-up-coalesce" },
  { value: "replay", label: "Replay", testId: "job-catch-up-replay" },
  { value: "run_once_on_catchup", label: "Run once", testId: "job-catch-up-run-once" },
];

/** How each catch-up choice reacts to fires missed while the runtime was down. */
const CATCH_UP_DESCRIPTIONS: Record<CatchUpChoice, string> = {
  [CATCH_UP_DEFAULT]: "Runtime picks the catch-up behavior for this target.",
  skip_missed: "Run the latest missed fire within grace; skip it beyond grace.",
  coalesce: "Collapse all missed fires into one catch-up run.",
  replay: "Run every missed fire in order.",
  run_once_on_catchup: "Run once to catch up, then resume the schedule.",
};

interface ReliabilitySectionProps {
  retry: AutomationRetry;
  fireLimit: AutomationFireLimit | undefined;
  enabled: boolean;
  locked: boolean;
  mode: "create" | "edit";
  badge: string;
  defaultOpen: boolean;
  /** True for cron/every schedules; catch-up + grace are recurring-only, hidden for one-shot `at`. */
  recurring: boolean;
  catchUpPolicy: AutomationCatchUpPolicy | undefined;
  misfireGraceSeconds: number | undefined;
  onRetryChange: (retry: AutomationRetry) => void;
  onFireLimitChange: (fireLimit: AutomationFireLimit) => void;
  onEnabledChange: (enabled: boolean) => void;
  /** `undefined` selects the target-aware default (omitted from the request). */
  onCatchUpPolicyChange: (policy: AutomationCatchUpPolicy | undefined) => void;
  /** `undefined` (zero/empty) applies the scheduler's default jitter grace. */
  onMisfireGraceChange: (seconds: number | undefined) => void;
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
  recurring,
  catchUpPolicy,
  misfireGraceSeconds,
  onRetryChange,
  onFireLimitChange,
  onEnabledChange,
  onCatchUpPolicyChange,
  onMisfireGraceChange,
}: ReliabilitySectionProps) {
  const [open, setOpen] = useState(defaultOpen);
  const isBackoff = retry.strategy === "backoff";
  const retryDisabled = locked || !isBackoff;
  const catchUpValue: CatchUpChoice = catchUpPolicy ?? CATCH_UP_DEFAULT;

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
        {recurring ? (
          <>
            <Field className="col-span-2" data-testid="job-catch-up-field">
              <FieldTitle>Catch-up policy</FieldTitle>
              <PillGroup
                aria-label="Catch-up policy"
                items={CATCH_UP_ITEMS}
                onChange={next =>
                  onCatchUpPolicyChange(next === CATCH_UP_DEFAULT ? undefined : next)
                }
                size="sm"
                value={catchUpValue}
              />
              <FieldDescription>{CATCH_UP_DESCRIPTIONS[catchUpValue]}</FieldDescription>
            </Field>
            <Field>
              <FieldLabel htmlFor="job-misfire-grace">Grace window</FieldLabel>
              <Input
                className="font-mono"
                data-testid="job-misfire-grace"
                id="job-misfire-grace"
                inputMode="numeric"
                min={0}
                step={1}
                onChange={event => {
                  // Store the entered value as-is (no flooring); serialization keeps
                  // it only when it is a positive whole number of seconds.
                  const raw = event.target.value;
                  const seconds = Number(raw);
                  onMisfireGraceChange(raw === "" || Number.isNaN(seconds) ? undefined : seconds);
                }}
                placeholder="0"
                type="number"
                value={misfireGraceSeconds ?? ""}
              />
              <FieldDescription>
                Applies only when the effective policy is Skip missed. Whole seconds the latest
                missed fire may still run; 0 or empty uses the scheduler&apos;s default grace.
              </FieldDescription>
            </Field>
          </>
        ) : null}
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
