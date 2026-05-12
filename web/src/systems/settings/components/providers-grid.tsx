import type { SettingsProviderEntry } from "@/systems/settings";

import { ProviderCard } from "./provider-card";

interface ProvidersGridProps {
  providers: SettingsProviderEntry[];
  onOpen: (entry: SettingsProviderEntry) => void;
}

export function ProvidersGrid({ providers, onOpen }: ProvidersGridProps) {
  return (
    <section
      className="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      data-testid="settings-page-providers-list"
    >
      {providers.map(provider => (
        <ProviderCard key={provider.name} provider={provider} onOpen={onOpen} />
      ))}
    </section>
  );
}
