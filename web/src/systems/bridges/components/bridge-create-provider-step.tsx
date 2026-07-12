import { FileJson2, Plug } from "lucide-react";

import {
  ActionResultBanner,
  bridgeKindIconRegistry,
  CatalogCard,
  Eyebrow,
  FormSection,
  KindChip,
  KindIcon,
  Pill,
} from "@agh/ui";

import { providerHealthTone, providerStateTone } from "@/systems/model-catalog";
import { buildBridgeProviderKey, isBridgeProviderSelectable } from "../lib/bridge-formatters";
import type { BridgeProvider } from "../types";

interface BridgeCreateProviderStepProps {
  onSelect: (providerKey: string) => void;
  providers: BridgeProvider[];
  selectedProviderKey: string;
  supportsManifest: boolean;
}

interface BridgeProviderCatalogCardProps {
  onSelect: () => void;
  provider: BridgeProvider;
  selected: boolean;
}

function BridgeProviderCatalogCard({
  onSelect,
  provider,
  selected,
}: BridgeProviderCatalogCardProps) {
  const providerKey = buildBridgeProviderKey(provider);
  const selectable = isBridgeProviderSelectable(provider);

  return (
    <CatalogCard
      actionable={selectable}
      aria-disabled={selectable ? undefined : true}
      aria-pressed={selected}
      data-testid={`bridge-provider-card-${providerKey}`}
      onClick={selectable ? onSelect : undefined}
      onKeyDown={
        selectable
          ? event => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                onSelect();
              }
            }
          : undefined
      }
      role="button"
      selected={selected}
      tabIndex={selectable ? 0 : -1}
    >
      <div className="flex items-start gap-3">
        <CatalogCard.Logo size="lg">
          <KindIcon
            kind={provider.platform}
            registry={bridgeKindIconRegistry}
            size="md"
            tone="default"
          />
        </CatalogCard.Logo>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <CatalogCard.Title className="min-w-0">{provider.display_name}</CatalogCard.Title>
            <Pill mono tone={providerHealthTone(provider.health)}>
              {provider.health}
            </Pill>
          </div>
          <div className="flex flex-wrap items-center gap-1.5">
            <KindChip kind={provider.platform} />
            <Eyebrow className="text-muted">{provider.extension_name}</Eyebrow>
          </div>
        </div>
      </div>
      <CatalogCard.Description>
        {provider.description ?? "Bridge adapter installed and ready for instance configuration."}
      </CatalogCard.Description>
      <CatalogCard.Actions className="border-t-0 pt-0">
        <Pill mono tone={providerStateTone(provider.state)}>
          {provider.state}
        </Pill>
        {selectable ? null : (
          <Pill mono tone="danger">
            UNAVAILABLE
          </Pill>
        )}
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

export function BridgeCreateProviderStep({
  onSelect,
  providers,
  selectedProviderKey,
  supportsManifest,
}: BridgeCreateProviderStepProps) {
  return (
    <FormSection
      data-testid="bridge-wizard-section-provider"
      description="Only providers with healthy runtime state can be selected for bridge creation."
      icon={Plug}
      title="Provider"
    >
      {providers.length === 0 ? (
        <div
          className="rounded bg-canvas-tint px-5 py-8 text-center text-small-body leading-6 text-muted"
          data-testid="bridge-provider-empty"
        >
          No bridge providers are currently available. Install or enable a bridge adapter extension
          before creating a new bridge.
        </div>
      ) : (
        <div className="grid gap-3 lg:grid-cols-2" data-testid="bridge-wizard-provider-grid">
          {providers.map(provider => {
            const providerKey = buildBridgeProviderKey(provider);
            return (
              <BridgeProviderCatalogCard
                key={providerKey}
                onSelect={() => onSelect(providerKey)}
                provider={provider}
                selected={providerKey === selectedProviderKey}
              />
            );
          })}
        </div>
      )}

      {supportsManifest && selectedProviderKey ? (
        <ActionResultBanner
          data-testid="bridge-manifest-precreate-hint"
          description="AGH generates the JSON from the persisted bridge ID and saved webhook URL. Create the bridge first, then copy the real manifest into Slack."
          icon={FileJson2}
          title="Slack manifest available after creation"
          tone="info"
        />
      ) : null}
    </FormSection>
  );
}
