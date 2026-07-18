import { Box, Puzzle, SearchX } from "lucide-react";
import { useState } from "react";

import {
  Button,
  Empty,
  ListingPage,
  ListingToolbar,
  PageHead,
  type ListingViewMode,
} from "@agh/ui";

import { BundleInventoryCard } from "./bundle-inventory-card";
import { BundleInventoryRow } from "./bundle-inventory-row";
import {
  DeactivateBundleDialog,
  ExtensionProvenanceDialog,
  RemoveExtensionDialog,
} from "./extension-dialogs";
import { ExtensionInventoryCard } from "./extension-inventory-card";
import { ExtensionInventoryRow } from "./extension-inventory-row";
import { InventoryEmpty } from "./inventory-empty";
import { InventorySkeleton } from "./inventory-skeleton";
import { useExtensionInventoryState } from "../hooks/use-extension-inventory-state";
import { useBundleActivations } from "../hooks/use-extensions";
import { useDeactivateBundle, useUpdateBundleActivation } from "../hooks/use-extension-actions";
import type { BundleActivation } from "../types";

export interface ExtensionsInventoryProps {
  tab: "extensions" | "bundles";
  view?: ListingViewMode;
  onViewChange?: (view: ListingViewMode) => void;
}

export function ExtensionsInventory({
  tab,
  view = "rows",
  onViewChange,
}: ExtensionsInventoryProps) {
  return tab === "bundles" ? (
    <BundleInventory onViewChange={onViewChange} view={view} />
  ) : (
    <ExtensionInventory onViewChange={onViewChange} view={view} />
  );
}

function ExtensionInventory({
  view,
  onViewChange,
}: {
  view: ListingViewMode;
  onViewChange?: (view: ListingViewMode) => void;
}) {
  const state = useExtensionInventoryState();
  return (
    <ListingPage data-testid="extensions-page">
      <PageHead
        count={state.inventory.data.length}
        icon={Puzzle}
        meta="Installed extensions and their runtime state"
        title="Extensions"
      />
      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search installed extensions"
            className="w-full"
            containerClassName="w-full max-w-105"
            onChange={state.setQuery}
            placeholder="Search installed"
            value={state.query}
          />
        </ListingToolbar.Leading>
        {onViewChange ? (
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={onViewChange} value={view} />
          </ListingToolbar.Trailing>
        ) : null}
      </ListingToolbar>
      {state.inventory.isLoading ? (
        <InventorySkeleton view={view} />
      ) : state.inventory.error ? (
        <Empty
          description={state.inventory.error.message}
          icon={Puzzle}
          title="Unable to load extensions"
        />
      ) : state.inventory.data.length === 0 ? (
        <InventoryEmpty kind="extensions" />
      ) : state.visible.length === 0 ? (
        <Empty
          action={
            <Button onClick={() => state.setQuery("")} size="sm" variant="outline">
              Clear search
            </Button>
          }
          description={`No installed extension matches “${state.query}”.`}
          icon={SearchX}
          title="No matching extensions"
        />
      ) : view === "cards" ? (
        <div
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="extension-list"
          data-view="cards"
        >
          {state.visible.map(item => (
            <ExtensionInventoryCard
              item={item}
              key={item.extension.name}
              onProvenance={() => state.setProvenance(item.extension)}
              onRemove={() => state.setRemoving(item.extension)}
            />
          ))}
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
          data-testid="extension-list"
          data-view="rows"
        >
          {state.visible.map(item => (
            <ExtensionInventoryRow
              item={item}
              key={item.extension.name}
              onProvenance={() => state.setProvenance(item.extension)}
              onRemove={() => state.setRemoving(item.extension)}
            />
          ))}
        </div>
      )}
      <ExtensionProvenanceDialog
        extension={state.provenance}
        onOpenChange={open => !open && state.setProvenance(null)}
        open={Boolean(state.provenance)}
      />
      <RemoveExtensionDialog
        activeBundles={state.activations.data}
        dependencyError={state.activations.error}
        dependencyLoading={state.activations.isLoading}
        extension={state.removing}
        onOpenChange={open => !open && state.setRemoving(null)}
        onRetryDependencies={() => void state.activations.refetch()}
        open={Boolean(state.removing)}
      />
    </ListingPage>
  );
}

function BundleInventory({
  view,
  onViewChange,
}: {
  view: ListingViewMode;
  onViewChange?: (view: ListingViewMode) => void;
}) {
  const bundles = useBundleActivations();
  const update = useUpdateBundleActivation();
  const deactivate = useDeactivateBundle();
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<BundleActivation | null>(null);
  const visible = (bundles.data ?? []).filter(item =>
    item.bundle_name.toLowerCase().includes(query.toLowerCase())
  );
  return (
    <ListingPage data-testid="bundle-activations-page">
      <PageHead
        count={bundles.data?.length ?? 0}
        icon={Box}
        meta="Active bundle activations on this daemon"
        title="Bundles"
      />
      <ListingToolbar>
        <ListingToolbar.Leading>
          <ListingToolbar.Search
            aria-label="Search bundle activations"
            className="w-full"
            containerClassName="w-full max-w-105"
            onChange={setQuery}
            placeholder="Search activated bundles"
            value={query}
          />
        </ListingToolbar.Leading>
        {onViewChange ? (
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={onViewChange} value={view} />
          </ListingToolbar.Trailing>
        ) : null}
      </ListingToolbar>
      {bundles.isLoading ? (
        <InventorySkeleton view={view} />
      ) : bundles.error ? (
        <Empty
          description={bundles.error.message}
          icon={Box}
          title="Unable to load bundle activations"
        />
      ) : (bundles.data?.length ?? 0) === 0 ? (
        <InventoryEmpty kind="bundles" />
      ) : visible.length === 0 ? (
        <Empty
          action={
            <Button onClick={() => setQuery("")} size="sm" variant="outline">
              Clear search
            </Button>
          }
          icon={SearchX}
          title="No matching bundles"
        />
      ) : view === "cards" ? (
        <div
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="bundle-activation-list"
          data-view="cards"
        >
          {visible.map(activation => (
            <BundleInventoryCard
              activation={activation}
              key={activation.id}
              onDeactivate={() => setSelected(activation)}
              onUpdate={() =>
                update.mutate({
                  id: activation.id,
                  body: { expected_version: activation.version },
                })
              }
              pending={update.isPending && update.variables?.id === activation.id}
            />
          ))}
        </div>
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
          data-testid="bundle-activation-list"
          data-view="rows"
        >
          {visible.map(activation => (
            <BundleInventoryRow
              activation={activation}
              key={activation.id}
              onDeactivate={() => setSelected(activation)}
              onUpdate={() =>
                update.mutate({
                  id: activation.id,
                  body: { expected_version: activation.version },
                })
              }
              pending={update.isPending && update.variables?.id === activation.id}
            />
          ))}
        </div>
      )}
      <DeactivateBundleDialog
        activation={selected}
        error={deactivate.error?.message}
        onConfirm={async () => {
          if (!selected) return;
          await deactivate.mutateAsync(selected.id);
          setSelected(null);
        }}
        onOpenChange={open => !open && setSelected(null)}
        open={Boolean(selected)}
        pending={deactivate.isPending}
      />
    </ListingPage>
  );
}
