import { Link } from "@tanstack/react-router";
import { Box, MoreHorizontal, PackageOpen, Puzzle, SearchX } from "lucide-react";
import { useState } from "react";

import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Empty,
  ListingPage,
  ListingRow,
  ListingToolbar,
  Pill,
  Skeleton,
  Spinner,
  Switch,
  Time,
} from "@agh/ui";

import {
  DeactivateBundleDialog,
  ExtensionProvenanceDialog,
  RemoveExtensionDialog,
} from "./extension-dialogs";
import { useExtensionInventoryState } from "../hooks/use-extension-inventory-state";
import { useBundleActivations } from "../hooks/use-extensions";
import {
  useDeactivateBundle,
  useToggleExtension,
  useUpdateBundleActivation,
  useUpdateExtension,
} from "../hooks/use-extension-actions";
import type { BundleActivation, InstalledExtensionView } from "../types";

export function ExtensionsInventory({ tab }: { tab: "extensions" | "bundles" }) {
  return tab === "bundles" ? <BundleInventory /> : <ExtensionInventory />;
}

function ExtensionInventory() {
  const state = useExtensionInventoryState();
  return (
    <ListingPage data-testid="extensions-page">
      <ListingPage.Head
        count={state.inventory.data.length}
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
      </ListingToolbar>
      {state.inventory.isLoading ? (
        <InventorySkeleton />
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
      ) : (
        <div className="overflow-hidden rounded-lg border border-line" data-testid="extension-list">
          {state.visible.map(item => (
            <ExtensionRow
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

function ExtensionRow({
  item,
  onProvenance,
  onRemove,
}: {
  item: InstalledExtensionView;
  onProvenance: () => void;
  onRemove: () => void;
}) {
  const toggle = useToggleExtension();
  const update = useUpdateExtension();
  const acting =
    (toggle.isPending && toggle.variables?.name === item.extension.name) ||
    (update.isPending && update.variables === item.extension.name);
  const description = item.listing?.description;
  return (
    <ListingRow data-testid={`extension-row-${item.extension.name}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${item.extension.name}`}
            params={{ name: item.extension.name }}
            to="/extensions/$name"
          />
        }
      >
        <ListingRow.Icon>
          <Puzzle className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name mono>
            <ListingRow.Title>{item.extension.name}</ListingRow.Title>
            <Pill mono size="xs">
              {item.extension.type}
            </Pill>
            <ListingRow.Slug>v{item.extension.version}</ListingRow.Slug>
          </ListingRow.Name>
          {description ? <ListingRow.Description>{description}</ListingRow.Description> : null}
          <ListingRow.Meta>
            <span>{item.extension.source}</span>
            {item.extension.provenance?.installed_at ? (
              <>
                <ListingRow.MetaDot />
                <span>
                  installed <Time iso={item.extension.provenance.installed_at} />
                </span>
              </>
            ) : null}
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        {acting ? (
          <Spinner className="size-3.5" />
        ) : (
          <Switch
            aria-label={`${item.extension.enabled ? "Disable" : "Enable"} ${item.extension.name}`}
            checked={item.extension.enabled}
            onCheckedChange={enabled => toggle.mutate({ name: item.extension.name, enabled })}
            size="sm"
          />
        )}
        {item.updateAvailable ? (
          <Button
            disabled={acting}
            onClick={() => update.mutate(item.extension.name)}
            size="sm"
            variant="outline"
          >
            Update
          </Button>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label={`Actions for ${item.extension.name}`}
                size="icon-sm"
                variant="ghost"
              />
            }
          >
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem onClick={onProvenance}>Provenance</DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="text-danger" onClick={onRemove}>
              Remove…
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </ListingRow.Trail>
    </ListingRow>
  );
}

function BundleInventory() {
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
      <ListingPage.Head
        count={bundles.data?.length ?? 0}
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
      </ListingToolbar>
      {bundles.isLoading ? (
        <InventorySkeleton />
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
      ) : (
        <div
          className="overflow-hidden rounded-lg border border-line"
          data-testid="bundle-activation-list"
        >
          {visible.map(activation => (
            <BundleRow
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

function BundleRow({
  activation,
  pending,
  onUpdate,
  onDeactivate,
}: {
  activation: BundleActivation;
  pending: boolean;
  onUpdate: () => void;
  onDeactivate: () => void;
}) {
  const contentCount = (activation.inventory ?? []).length;
  return (
    <ListingRow data-testid={`bundle-row-${activation.id}`}>
      <ListingRow.Link
        render={
          <Link
            aria-label={`Open ${activation.bundle_name}`}
            params={{ id: activation.id }}
            to="/extensions/bundles/$id"
          />
        }
      >
        <ListingRow.Icon>
          <Box className="size-4" />
        </ListingRow.Icon>
        <ListingRow.Main>
          <ListingRow.Name mono>
            <ListingRow.Title>{activation.bundle_name}</ListingRow.Title>
            <Pill mono size="xs">
              {activation.profile_name}
            </Pill>
          </ListingRow.Name>
          <ListingRow.Meta>
            <span>{contentCount} capabilities</span>
            <ListingRow.MetaDot />
            <span>
              {activation.scope}
              {activation.workspace_id ? ` · ${activation.workspace_id}` : ""}
            </span>
            <ListingRow.MetaDot />
            <span>
              activated <Time iso={activation.created_at} />
            </span>
          </ListingRow.Meta>
        </ListingRow.Main>
      </ListingRow.Link>
      <ListingRow.Trail>
        <Pill tone="success">active</Pill>
        {activation.spec_drift ? (
          <Button disabled={pending} onClick={onUpdate} size="sm" variant="outline">
            {pending ? <Spinner className="size-3" /> : null}Update
          </Button>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <Button
                aria-label={`Actions for ${activation.bundle_name}`}
                size="icon-sm"
                variant="ghost"
              />
            }
          >
            <MoreHorizontal className="size-4" />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem className="text-danger" onClick={onDeactivate}>
              Deactivate…
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </ListingRow.Trail>
    </ListingRow>
  );
}

function InventorySkeleton() {
  return (
    <div className="overflow-hidden rounded-lg border border-line" data-testid="extensions-loading">
      {[0, 1, 2].map(index => (
        <div
          className="grid grid-cols-[34px_minmax(0,1fr)_auto] gap-3.5 border-b border-line-soft px-4 py-3 last:border-0"
          key={index}
        >
          <Skeleton className="size-[34px]" />
          <div className="space-y-2">
            <Skeleton className="h-3 w-40" />
            <Skeleton className="h-3 w-72" />
          </div>
          <Skeleton className="h-5 w-20" />
        </div>
      ))}
    </div>
  );
}

function InventoryEmpty({ kind }: { kind: "extensions" | "bundles" }) {
  const bundles = kind === "bundles";
  return (
    <Empty
      action={
        <Button
          render={<Link search={{ kind: bundles ? "bundles" : "extensions" }} to="/marketplace" />}
          nativeButton={false}
          size="sm"
        >
          Browse marketplace
        </Button>
      }
      description={
        bundles
          ? "A bundle activates a curated set of capabilities in one step."
          : "Install one from the marketplace or run agh extension install."
      }
      icon={PackageOpen}
      title={bundles ? "No bundles activated" : "No extensions installed"}
    />
  );
}
