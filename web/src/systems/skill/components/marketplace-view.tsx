import { AlertCircle, Download, RotateCw, Search, Trash2, Wrench } from "lucide-react";
import type { ReactNode } from "react";

import {
  Alert,
  AlertDescription,
  AlertTitle,
  Button,
  CatalogCard,
  ConfirmDialog,
  DialogTrigger,
  Empty,
  ListingRow,
  Pill,
  Spinner,
  type ListingViewMode,
} from "@agh/ui";

import type { SkillMarketplaceListingPayload } from "../types";

export interface MarketplaceViewProps {
  listings: SkillMarketplaceListingPayload[];
  installedSkillNames: Set<string>;
  view: ListingViewMode;
  searchStatus: "prompt" | "searching" | "error" | "ready";
  searchError: Error | null;
  onClearSearch: () => void;
  onInstall: (slug: string) => void;
  onUpdate: (name: string) => void;
  onRemove: (name: string) => void;
  pendingActions?: ReadonlySet<"install" | "update" | "remove">;
}

type InstallActionState = "idle" | "installing";
type UpdateActionState = "idle" | "updating";
type RemoveActionState = "idle" | "removing";

interface MarketplaceItemActionsProps {
  listing: SkillMarketplaceListingPayload;
  installed: boolean;
  onInstall: () => void;
  onUpdate: () => void;
  onRemove: () => void;
  installState: InstallActionState;
  updateState: UpdateActionState;
  removeState: RemoveActionState;
}

function InstallActionIcon({ state }: { state: InstallActionState }) {
  if (state === "installing") {
    return <Spinner aria-hidden="true" className="size-3" />;
  }
  return <Download aria-hidden="true" className="size-3" />;
}

function UpdateActionIcon({ state }: { state: UpdateActionState }) {
  if (state === "updating") {
    return <Spinner aria-hidden="true" className="size-3" />;
  }
  return <RotateCw aria-hidden="true" className="size-3" />;
}

function RemoveActionIcon({ state }: { state: RemoveActionState }) {
  if (state === "removing") {
    return <Spinner aria-hidden="true" className="size-3" />;
  }
  return <Trash2 aria-hidden="true" className="size-3" />;
}

function MarketplaceItemActions({
  listing,
  installed,
  onInstall,
  onUpdate,
  onRemove,
  installState,
  updateState,
  removeState,
}: MarketplaceItemActionsProps) {
  if (!installed) {
    const installPending = installState === "installing";
    return (
      <Button
        data-testid={`install-btn-${listing.name}`}
        disabled={installPending}
        onClick={onInstall}
        size="sm"
        type="button"
        variant="neutral"
      >
        <InstallActionIcon state={installState} />
        {installPending ? "Installing" : "Install"}
      </Button>
    );
  }

  const updatePending = updateState === "updating";
  const removePending = removeState === "removing";
  return (
    <>
      <Pill mono data-testid={`installed-pill-${listing.name}`} tone="success">
        installed
      </Pill>
      <Button
        data-testid={`update-btn-${listing.name}`}
        disabled={updatePending || removePending}
        onClick={onUpdate}
        size="sm"
        type="button"
        variant="neutral"
      >
        <UpdateActionIcon state={updateState} />
        {updatePending ? "Updating" : "Update"}
      </Button>
      <ConfirmDialog
        cancelButtonProps={{
          "data-testid": `cancel-remove-${listing.name}`,
          disabled: removePending,
        }}
        cancelLabel="Cancel"
        confirmButtonProps={{ "data-testid": `confirm-remove-${listing.name}` }}
        confirmIcon={Trash2}
        confirmLabel={removePending ? "Removing" : "Remove skill"}
        contentProps={{ "data-testid": `remove-dialog-${listing.name}` }}
        description={
          <>
            This removes <strong>{listing.name}</strong> from the workspace. Marketplace metadata
            stays available so you can reinstall later.
          </>
        }
        isPending={removePending}
        onConfirm={onRemove}
        title="Remove marketplace skill?"
        tone="danger"
      >
        <DialogTrigger
          render={
            <Button
              data-testid={`remove-btn-${listing.name}`}
              disabled={removePending || updatePending}
              size="sm"
              type="button"
              variant="outline"
            />
          }
        >
          <RemoveActionIcon state={removeState} />
          Remove
        </DialogTrigger>
      </ConfirmDialog>
    </>
  );
}

function MarketplaceMeta({ listing }: { listing: SkillMarketplaceListingPayload }) {
  return (
    <>
      <span>{`@${listing.author}`}</span>
      {listing.version ? <span>{`v${listing.version}`}</span> : null}
      <span className="inline-flex items-center gap-1">
        <Download aria-hidden="true" className="size-3" />
        {String(listing.downloads)}
      </span>
    </>
  );
}

function MarketplaceCatalogItem({
  listing,
  children,
}: {
  listing: SkillMarketplaceListingPayload;
  children: ReactNode;
}) {
  return (
    <CatalogCard data-testid={`marketplace-row-${listing.name}`}>
      <div className="flex items-start gap-3">
        <CatalogCard.Logo>
          <Wrench className="size-4" />
        </CatalogCard.Logo>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <CatalogCard.Title>{listing.name}</CatalogCard.Title>
          <CatalogCard.Meta>
            <MarketplaceMeta listing={listing} />
          </CatalogCard.Meta>
        </div>
      </div>
      {listing.description ? (
        <CatalogCard.Description>{listing.description}</CatalogCard.Description>
      ) : null}
      <CatalogCard.Actions>
        <Pill mono size="sm" tone="neutral">
          marketplace
        </Pill>
        {children}
      </CatalogCard.Actions>
    </CatalogCard>
  );
}

function MarketplaceListingRow({
  listing,
  children,
}: {
  listing: SkillMarketplaceListingPayload;
  children: ReactNode;
}) {
  return (
    <ListingRow data-testid={`marketplace-row-${listing.name}`} interactive={false}>
      <ListingRow.Icon>
        <Wrench aria-hidden="true" className="size-4" />
      </ListingRow.Icon>
      <ListingRow.Main>
        <ListingRow.Name>
          <ListingRow.Title>{listing.name}</ListingRow.Title>
        </ListingRow.Name>
        {listing.description ? (
          <ListingRow.Description>{listing.description}</ListingRow.Description>
        ) : null}
        <ListingRow.Meta>
          <MarketplaceMeta listing={listing} />
        </ListingRow.Meta>
      </ListingRow.Main>
      <ListingRow.Trail className="gap-3">
        <Pill mono size="sm" tone="neutral">
          marketplace
        </Pill>
        {children}
      </ListingRow.Trail>
    </ListingRow>
  );
}

function MarketplaceView({
  listings,
  installedSkillNames,
  view,
  searchStatus,
  searchError,
  onClearSearch,
  onInstall,
  onUpdate,
  onRemove,
  pendingActions,
}: MarketplaceViewProps) {
  let body: ReactNode;

  if (searchStatus === "prompt") {
    body = (
      <div
        className="flex min-h-60 items-center justify-center"
        data-testid="marketplace-search-prompt"
      >
        <Empty
          className="max-w-sm"
          description="Type a skill name, author, or keyword to browse the marketplace."
          icon={Search}
          title="Search the marketplace"
        />
      </div>
    );
  } else if (searchStatus === "searching" && listings.length === 0) {
    body = (
      <div className="flex min-h-60 items-center justify-center" data-testid="marketplace-loading">
        <Spinner aria-hidden="true" className="size-5 text-subtle" />
      </div>
    );
  } else if (searchStatus === "error" && searchError) {
    body = (
      <div className="px-2 py-2" data-testid="marketplace-error">
        <Alert variant="danger">
          <AlertCircle aria-hidden="true" className="size-4" />
          <AlertTitle>Marketplace search failed</AlertTitle>
          <AlertDescription>
            {searchError.message ?? "The marketplace search request did not succeed."}
          </AlertDescription>
        </Alert>
      </div>
    );
  } else if (listings.length === 0) {
    body = (
      <div className="flex min-h-60 items-center justify-center" data-testid="marketplace-empty">
        <Empty
          action={
            <Button
              data-testid="marketplace-clear-search"
              onClick={onClearSearch}
              size="sm"
              type="button"
              variant="ghost"
            >
              Clear search
            </Button>
          }
          className="max-w-sm"
          description="No marketplace skills match this query. Try a different keyword or author."
          icon={Wrench}
          title="No skills found"
        />
      </div>
    );
  } else {
    const installState: InstallActionState = pendingActions?.has("install") ? "installing" : "idle";
    const updateState: UpdateActionState = pendingActions?.has("update") ? "updating" : "idle";
    const removeState: RemoveActionState = pendingActions?.has("remove") ? "removing" : "idle";

    const items = listings.map(listing => {
      const installed = installedSkillNames.has(listing.name);
      const actions = (
        <MarketplaceItemActions
          installState={installState}
          installed={installed}
          listing={listing}
          onInstall={() => onInstall(listing.slug)}
          onRemove={() => onRemove(listing.name)}
          onUpdate={() => onUpdate(listing.name)}
          removeState={removeState}
          updateState={updateState}
        />
      );

      if (view === "rows") {
        return (
          <MarketplaceListingRow key={listing.slug} listing={listing}>
            {actions}
          </MarketplaceListingRow>
        );
      }

      return (
        <MarketplaceCatalogItem key={listing.slug} listing={listing}>
          {actions}
        </MarketplaceCatalogItem>
      );
    });

    body =
      view === "rows" ? (
        <div
          className="overflow-hidden rounded-lg border border-line bg-canvas-soft"
          data-testid="marketplace-rows"
        >
          {items}
        </div>
      ) : (
        <div
          className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
          data-testid="marketplace-grid"
        >
          {items}
        </div>
      );
  }

  return <div data-testid="marketplace-view">{body}</div>;
}

export { MarketplaceView };
