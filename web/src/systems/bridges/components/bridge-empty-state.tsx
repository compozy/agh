import { Plus, Waypoints } from "lucide-react";

import { Button, CatalogCard, Empty, Eyebrow, KindChip, Pill, Section } from "@agh/ui";

import {
  buildBridgeProviderKey,
  isBridgeProviderSelectable,
} from "@/systems/bridges/lib/bridge-formatters";
import { providerHealthTone, providerStateTone } from "@/systems/model-catalog";
import type { BridgeProvider } from "@/systems/bridges/types";

interface BridgeEmptyStateProps {
  onCreate: () => void;
  providers: BridgeProvider[];
}

export function BridgeEmptyState({ onCreate, providers }: BridgeEmptyStateProps) {
  const hasInstalledProviders = providers.length > 0;
  const canCreate = providers.some(isBridgeProviderSelectable);

  const title = hasInstalledProviders ? "No bridges configured" : "No bridge providers installed";
  const description = hasInstalledProviders
    ? "Start by creating a bridge from an installed provider. Bridge instances keep routing, delivery defaults, and health separated per workspace or globally."
    : "Install a bridge-capable extension first. The create flow only becomes available when AGH can discover a provider through the runtime catalog.";

  const action = (
    <>
      <Button
        className="min-w-36"
        data-testid="bridge-empty-create-btn"
        disabled={!canCreate}
        onClick={onCreate}
        size="lg"
      >
        <Plus className="size-4" />
        Create Bridge
      </Button>
      {!canCreate && hasInstalledProviders ? (
        <p className="text-xs text-subtle">
          Installed providers are currently unavailable. Resolve extension health first.
        </p>
      ) : null}
    </>
  );

  return (
    <div className="flex flex-1 overflow-y-auto p-6" data-testid="bridges-empty-state">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <Empty action={action} description={description} icon={Waypoints} title={title} />

        {hasInstalledProviders ? (
          <Section label="Installed providers">
            <p className="text-small-body text-muted">
              Providers come from installed bridge-capable extensions. Unavailable providers stay
              visible so the operator can diagnose runtime state.
            </p>
            <div className="mt-3 grid gap-4 lg:grid-cols-2">
              {providers.map(provider => (
                <BridgeProviderCatalogCard
                  key={buildBridgeProviderKey(provider)}
                  provider={provider}
                />
              ))}
            </div>
          </Section>
        ) : null}
      </div>
    </div>
  );
}

interface BridgeProviderCatalogCardProps {
  provider: BridgeProvider;
}

function BridgeProviderCatalogCard({ provider }: BridgeProviderCatalogCardProps) {
  const selectable = isBridgeProviderSelectable(provider);

  return (
    <CatalogCard
      aria-disabled={selectable ? undefined : true}
      data-testid={`bridge-provider-card-${buildBridgeProviderKey(provider)}`}
    >
      <div className="flex items-start gap-3">
        <CatalogCard.Logo size="lg" />
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
      <p className="text-eyebrow leading-relaxed text-subtle">
        {provider.health_message ||
          (selectable
            ? "This provider can be used to create a bridge instance."
            : "This provider is installed but not available for bridge creation right now.")}
      </p>
    </CatalogCard>
  );
}
