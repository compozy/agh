import { Eyebrow, Item, ItemContent, ItemFooter, ItemHeader, ItemTitle, Pill } from "@agh/ui";

import { cn } from "@/lib/utils";
import { KindChip } from "@/systems/network";
import { providerHealthTone, providerStateTone } from "@/systems/model-catalog";
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

export function BridgeProviderCard({
  onSelect,
  provider,
  selected = false,
}: BridgeProviderCardProps) {
  const selectable = isBridgeProviderSelectable(provider);
  const content = (
    <>
      <ItemHeader className="items-start">
        <ItemContent className="min-w-0">
          <ItemTitle>{provider.display_name}</ItemTitle>
          <div className="flex flex-wrap items-center gap-1.5">
            <KindChip kind={provider.platform} />
            <Eyebrow tone="neutral">{provider.extension_name}</Eyebrow>
          </div>
        </ItemContent>
        <Pill mono tone={providerHealthTone(provider.health)}>
          {provider.health}
        </Pill>
      </ItemHeader>

      <p className="basis-full text-xs leading-relaxed text-(--color-text-secondary)">
        {provider.description ?? "Bridge adapter installed and ready for instance configuration."}
      </p>

      <ItemFooter className="justify-start gap-1.5">
        <Pill mono tone={providerStateTone(provider.state)}>
          {provider.state}
        </Pill>
        {!selectable ? (
          <Pill mono tone="danger">
            UNAVAILABLE
          </Pill>
        ) : null}
      </ItemFooter>

      <p className="basis-full text-eyebrow leading-relaxed text-(--color-text-tertiary)">
        {provider.health_message ||
          (selectable
            ? "This provider can be used to create a bridge instance."
            : "This provider is installed but not available for bridge creation right now.")}
      </p>
    </>
  );

  const className = cn(
    "gap-3 rounded-md border bg-(--color-surface) p-4 text-left",
    selected ? "border-accent bg-(--color-surface-elevated)" : "border-(--color-divider)",
    onSelect && selectable && "cursor-pointer hover:border-accent hover:bg-(--color-hover)",
    onSelect && !selectable && "cursor-not-allowed opacity-70"
  );

  return (
    <Item
      aria-disabled={onSelect && !selectable ? true : undefined}
      as={onSelect ? "button" : "div"}
      className={className}
      data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
      disabled={Boolean(onSelect && !selectable)}
      onClick={selectable ? onSelect : undefined}
      selected={selected}
      selectable={Boolean(onSelect && selectable)}
      tabIndex={onSelect && !selectable ? -1 : undefined}
    >
      {content}
    </Item>
  );
}
