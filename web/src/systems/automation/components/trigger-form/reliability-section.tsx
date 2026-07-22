import type { AutomationFireLimit, AutomationRetry } from "../../types";
import {
  ReliabilityEnabledField,
  ReliabilityFireLimitFields,
  ReliabilityRetryFields,
  ReliabilitySectionShell,
} from "../reliability-controls";

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
  return (
    <ReliabilitySectionShell
      badge={badge}
      defaultOpen={defaultOpen}
      testId="trigger-governance-toggle"
      title="Reliability & state"
    >
      <ReliabilityRetryFields idPrefix="trigger" onChange={onRetryChange} retry={retry} />
      <ReliabilityFireLimitFields
        fireLimit={fireLimit}
        idPrefix="trigger"
        onChange={onFireLimitChange}
      />
      <ReliabilityEnabledField
        description="Disabled triggers stay registered but never fire."
        enabled={enabled}
        idPrefix="trigger"
        label="Trigger enabled"
        onChange={onEnabledChange}
      />
    </ReliabilitySectionShell>
  );
}
