import type { SettingsProviderEntry } from "@/systems/settings";

import { ProviderCard } from "./provider-card";

interface ProvidersGridProps {
  providers: SettingsProviderEntry[];
  onEdit: (entry: SettingsProviderEntry) => void;
  onDelete: (entry: SettingsProviderEntry) => void;
}

export function ProvidersGrid({ providers, onEdit, onDelete }: ProvidersGridProps) {
  return (
    <section
      className="grid gap-4 md:grid-cols-2 xl:grid-cols-3"
      data-testid="settings-page-providers-list"
    >
      {providers.map(provider => (
        <ProviderCard key={provider.name} provider={provider} onEdit={onEdit} onDelete={onDelete} />
      ))}
    </section>
  );
}
