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
  Pill,
  SearchInput,
  Spinner,
} from "@agh/ui";

import type { SkillMarketplaceListingPayload } from "../types";

interface MarketplaceViewProps {
  searchQuery: string;
  onSearchChange: (query: string) => void;
  listings: SkillMarketplaceListingPayload[];
  installedSkillNames: Set<string>;
  isSearchEnabled: boolean;
  isSearching: boolean;
  searchError: Error | null;
  onInstall: (slug: string) => void;
  onUpdate: (name: string) => void;
  onRemove: (name: string) => void;
  isInstalling: boolean;
  isUpdating: boolean;
  isRemoving: boolean;
}

interface MarketplaceCatalogItemProps {
  listing: SkillMarketplaceListingPayload;
  children: ReactNode;
}

type InstallActionState = "idle" | "installing";
type UpdateActionState = "idle" | "updating";
type RemoveActionState = "idle" | "removing";

interface InstallableMarketplaceCatalogItemProps {
  listing: SkillMarketplaceListingPayload;
  onInstall: () => void;
  installState: InstallActionState;
}

interface InstalledMarketplaceCatalogItemProps {
  listing: SkillMarketplaceListingPayload;
  onUpdate: () => void;
  onRemove: () => void;
  updateState: UpdateActionState;
  removeState: RemoveActionState;
}

function MarketplaceCatalogItem({ listing, children }: MarketplaceCatalogItemProps) {
  return (
    <CatalogCard data-testid={`marketplace-row-${listing.name}`}>
      <div className="flex items-start gap-3">
        <CatalogCard.Logo>
          <Wrench className="size-4" />
        </CatalogCard.Logo>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <CatalogCard.Title>{listing.name}</CatalogCard.Title>
          <CatalogCard.Meta>
            <span>{`@${listing.author}`}</span>
            {listing.version ? <span>{`v${listing.version}`}</span> : null}
            <span className="inline-flex items-center gap-1">
              <Download aria-hidden="true" className="size-3" />
              {String(listing.downloads)}
            </span>
          </CatalogCard.Meta>
        </div>
      </div>
      <CatalogCard.Description>{listing.description}</CatalogCard.Description>
      <CatalogCard.Actions>{children}</CatalogCard.Actions>
    </CatalogCard>
  );
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

function InstallableMarketplaceCatalogItem({
  listing,
  onInstall,
  installState,
}: InstallableMarketplaceCatalogItemProps) {
  const installPending = installState === "installing";
  return (
    <MarketplaceCatalogItem listing={listing}>
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
    </MarketplaceCatalogItem>
  );
}

function InstalledMarketplaceCatalogItem({
  listing,
  onUpdate,
  onRemove,
  updateState,
  removeState,
}: InstalledMarketplaceCatalogItemProps) {
  const updatePending = updateState === "updating";
  const removePending = removeState === "removing";
  return (
    <MarketplaceCatalogItem listing={listing}>
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
    </MarketplaceCatalogItem>
  );
}

function MarketplaceView({
  searchQuery,
  onSearchChange,
  listings,
  installedSkillNames,
  isSearchEnabled,
  isSearching,
  searchError,
  onInstall,
  onUpdate,
  onRemove,
  isInstalling,
  isUpdating,
  isRemoving,
}: MarketplaceViewProps) {
  return (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden" data-testid="marketplace-view">
      <div className="flex flex-col gap-3 border-b border-line px-4 py-3">
        <SearchInput
          aria-label="Search marketplace skills"
          data-testid="marketplace-search-input"
          onChange={onSearchChange}
          placeholder="Search skills on the marketplace..."
          value={searchQuery}
        />
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto p-4">
        {!isSearchEnabled ? (
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
        ) : isSearching && listings.length === 0 ? (
          <div
            className="flex min-h-60 items-center justify-center"
            data-testid="marketplace-loading"
          >
            <Spinner aria-hidden="true" className="size-5 text-subtle" />
          </div>
        ) : searchError ? (
          <div className="px-2 py-2" data-testid="marketplace-error">
            <Alert variant="danger">
              <AlertCircle aria-hidden="true" className="size-4" />
              <AlertTitle>Marketplace search failed</AlertTitle>
              <AlertDescription>
                {searchError.message ?? "The marketplace search request did not succeed."}
              </AlertDescription>
            </Alert>
          </div>
        ) : listings.length === 0 ? (
          <div
            className="flex min-h-60 items-center justify-center"
            data-testid="marketplace-empty"
          >
            <Empty
              className="max-w-sm"
              description="No marketplace skills match this query. Try a different keyword or author."
              icon={Wrench}
              title="No skills found"
            />
          </div>
        ) : (
          <div
            className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3"
            data-testid="marketplace-grid"
          >
            {listings.map(listing =>
              installedSkillNames.has(listing.name) ? (
                <InstalledMarketplaceCatalogItem
                  key={listing.slug}
                  listing={listing}
                  onRemove={() => onRemove(listing.name)}
                  onUpdate={() => onUpdate(listing.name)}
                  removeState={isRemoving ? "removing" : "idle"}
                  updateState={isUpdating ? "updating" : "idle"}
                />
              ) : (
                <InstallableMarketplaceCatalogItem
                  installState={isInstalling ? "installing" : "idle"}
                  key={listing.slug}
                  listing={listing}
                  onInstall={() => onInstall(listing.slug)}
                />
              )
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export { MarketplaceView };
export type { MarketplaceViewProps };
