import { Clock, Webhook, Zap } from "lucide-react";

import type { LoopBindingKind } from "../../lib/loop-bindings";
import { bindingKindLabel } from "../../lib/loop-bindings";

const BINDING_ICON: Record<LoopBindingKind, typeof Clock> = {
  schedule: Clock,
  webhook: Webhook,
  trigger: Zap,
};

interface LoopBindingBadgeProps {
  /** Distinct attached-automation kinds for this loop; empty renders nothing. */
  kinds: readonly LoopBindingKind[];
}

/**
 * Neutral catalog badge shown on rows whose Loop has at least one attached
 * loop-target automation (design §4.1). Color never encodes the kind: the glyph
 * distinguishes schedule / webhook / trigger, the tint stays neutral.
 */
export function LoopBindingBadge({ kinds }: LoopBindingBadgeProps) {
  if (kinds.length === 0) return null;
  return (
    <span className="inline-flex items-center gap-1.5" data-testid="loop-binding-badge">
      {kinds.map(kind => {
        const Icon = BINDING_ICON[kind];
        return (
          <span
            key={kind}
            className="inline-flex items-center gap-1 rounded-xs bg-badge-fill px-1.5 py-px font-mono text-[10px] text-subtle"
            data-binding-kind={kind}
          >
            <Icon aria-hidden="true" className="size-2.5 text-faint" />
            {bindingKindLabel(kind)}
          </span>
        );
      })}
    </span>
  );
}
