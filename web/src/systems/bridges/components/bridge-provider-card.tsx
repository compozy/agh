import { KindChip, MonoBadge, type MonoBadgeTone } from "@agh/ui";
import { cn } from "@/lib/utils";
import {
  buildBridgeProviderKey,
  isBridgeProviderSelectable,
} from "@/systems/bridges/lib/bridge-formatters";
import type { BridgeProvider } from "@/systems/bridges/types";

interface BridgeProviderCardProps {
  onSelect?: () => void;
  provider: BridgeProvider;
  selected?: boolean;
}

function healthBadgeTone(health?: string): MonoBadgeTone {
  switch (health) {
    case "healthy":
      return "success";
    case "unhealthy":
      return "danger";
    default:
      return "neutral";
  }
}

function stateBadgeTone(state?: string): MonoBadgeTone {
  switch (state) {
    case "active":
      return "success";
    case "error":
      return "danger";
    case "registered":
      return "info";
    case "enabled":
      return "warning";
    default:
      return "neutral";
  }
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
        <div className="min-w-0 space-y-1">
          <h3 className="truncate text-[13px] font-medium text-[color:var(--color-text-primary)]">
            {provider.display_name}
          </h3>
          <div className="flex flex-wrap items-center gap-1.5">
            <KindChip kind={provider.platform} />
            <span className="font-mono text-[10px] uppercase tracking-[0.12em] text-[color:var(--color-text-label)]">
              {provider.extension_name}
            </span>
          </div>
        </div>
        <MonoBadge tone={healthBadgeTone(provider.health)}>{provider.health}</MonoBadge>
      </div>

      <p className="text-[12px] leading-relaxed text-[color:var(--color-text-secondary)]">
        {provider.description ?? "Bridge adapter installed and ready for instance configuration."}
      </p>

      <div className="flex flex-wrap items-center gap-1.5">
        <MonoBadge tone={stateBadgeTone(provider.state)}>{provider.state}</MonoBadge>
        {!selectable ? <MonoBadge tone="danger">UNAVAILABLE</MonoBadge> : null}
      </div>

      <p className="text-[11px] leading-relaxed text-[color:var(--color-text-tertiary)]">
        {provider.health_message ||
          (selectable
            ? "This provider can be used to create a bridge instance."
            : "This provider is installed but not available for bridge creation right now.")}
      </p>
    </>
  );

  const className = cn(
    "flex w-full flex-col gap-3 rounded-[var(--radius-md)] border bg-[color:var(--color-surface)] p-4 text-left transition-colors",
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
