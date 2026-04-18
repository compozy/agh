import { Plus, Waypoints } from "lucide-react";

import { Button, Empty } from "@agh/ui";
import { BridgeProviderCard } from "@/systems/bridges/components/bridge-provider-card";
import { isBridgeProviderSelectable } from "@/systems/bridges/lib/bridge-formatters";
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

  return (
    <div className="flex flex-1 overflow-y-auto p-6" data-testid="bridges-empty-state">
      <div className="mx-auto flex w-full max-w-5xl flex-col gap-6">
        <Empty
          icon={Waypoints}
          title={title}
          description={description}
          action={
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
                <p className="text-xs text-[color:var(--color-text-tertiary)]">
                  Installed providers are currently unavailable. Resolve extension health first.
                </p>
              ) : null}
            </>
          }
        />

        {hasInstalledProviders ? (
          <section className="space-y-4">
            <div className="space-y-1">
              <p className="font-mono text-[0.68rem] uppercase tracking-[0.16em] text-[color:var(--color-text-label)]">
                Installed providers
              </p>
              <p className="text-sm text-[color:var(--color-text-secondary)]">
                Providers come from installed bridge-capable extensions. Unavailable providers stay
                visible so the operator can diagnose runtime state.
              </p>
            </div>
            <div className="grid gap-4 lg:grid-cols-2">
              {providers.map(provider => (
                <BridgeProviderCard
                  key={`${provider.extension_name}:${provider.platform}`}
                  provider={provider}
                />
              ))}
            </div>
          </section>
        ) : null}
      </div>
    </div>
  );
}
