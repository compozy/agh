import { Pill } from "@/components/design-system";
import { cn } from "@/lib/utils";
import {
  bridgeProviderHealthTone,
  bridgeProviderStateTone,
  buildBridgeProviderKey,
  isBridgeProviderSelectable,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeProvider } from "@/systems/bridges/types";

interface BridgeProviderCardProps {
  onSelect?: () => void;
  provider: BridgeProvider;
  selected?: boolean;
}

export function BridgeProviderCard({
  onSelect,
  provider,
  selected = false,
}: BridgeProviderCardProps) {
  const selectable = isBridgeProviderSelectable(provider);
  const content = (
    <>
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <h3 className="text-sm font-medium text-[color:var(--color-text-primary)]">
            {provider.display_name}
          </h3>
          <p className="font-mono text-[0.65rem] uppercase tracking-[0.14em] text-[color:var(--color-text-label)]">
            {provider.platform} / {provider.extension_name}
          </p>
        </div>
        <Pill emphasis="strong" kind="state" tone={bridgeProviderHealthTone(provider.health)}>
          {provider.health}
        </Pill>
      </div>

      {provider.description ? (
        <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
          {provider.description}
        </p>
      ) : (
        <p className="text-sm leading-relaxed text-[color:var(--color-text-secondary)]">
          Bridge adapter installed and ready for instance configuration.
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2">
        <Pill kind="tag" tone={bridgeProviderStateTone(provider.state)}>
          {provider.state}
        </Pill>
        {!selectable && (
          <Pill kind="tag" tone="danger">
            unavailable
          </Pill>
        )}
      </div>

      <p className="text-xs leading-relaxed text-[color:var(--color-text-tertiary)]">
        {provider.health_message ||
          (selectable
            ? "This provider can be used to create a bridge instance."
            : "This provider is installed but not available for bridge creation right now.")}
      </p>
    </>
  );

  const className = cn(
    "flex w-full flex-col gap-3 rounded-xl border bg-[color:var(--color-surface)] p-4 text-left transition-colors",
    selected
      ? "border-[color:var(--color-accent)] bg-[color:var(--color-surface-elevated)]"
      : "border-[color:var(--color-divider)]",
    onSelect &&
      selectable &&
      "cursor-pointer hover:border-[color:var(--color-accent)] hover:bg-[color:var(--color-hover)]",
    onSelect && !selectable && "cursor-not-allowed opacity-70"
  );

  if (!onSelect) {
    return (
      <div
        className={className}
        data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
      >
        {content}
      </div>
    );
  }

  return (
    <button
      className={className}
      data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
      disabled={!selectable}
      onClick={onSelect}
      type="button"
    >
      {content}
    </button>
  );
}
