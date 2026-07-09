import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";

import { ListingToolbar, type ListingViewMode } from "@agh/ui";
import { PanelSurface } from "@/storybook/story-layout";
import { skillMarketplaceListingFixtures } from "@/systems/skill/mocks";
import type { SkillMarketplaceListingPayload } from "@/systems/skill";

import { MarketplaceView } from "../marketplace-view";

const meta: Meta<typeof MarketplaceView> = {
  title: "systems/skill/components/MarketplaceView",
  component: MarketplaceView,
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

interface StoryHarnessProps {
  initialQuery?: string;
  listings?: SkillMarketplaceListingPayload[];
  installedSkillNames?: Set<string>;
  isSearchEnabled?: boolean;
  isSearching?: boolean;
  searchError?: Error | null;
  isInstalling?: boolean;
  isUpdating?: boolean;
  isRemoving?: boolean;
  initialView?: ListingViewMode;
}

function MarketplaceViewHarness({
  initialQuery = "",
  listings = [],
  installedSkillNames,
  isSearchEnabled,
  isSearching = false,
  searchError = null,
  isInstalling = false,
  isUpdating = false,
  isRemoving = false,
  initialView = "cards",
}: StoryHarnessProps) {
  const [query, setQuery] = useState(initialQuery);
  const [view, setView] = useState<ListingViewMode>(initialView);
  const enabled = isSearchEnabled ?? query.trim() !== "";
  return (
    <PanelSurface>
      <div className="flex flex-col gap-4 p-4">
        <ListingToolbar>
          <ListingToolbar.Leading>
            <ListingToolbar.Search
              data-testid="marketplace-search-input"
              onChange={setQuery}
              placeholder="Search skills on the marketplace…"
              value={query}
            />
          </ListingToolbar.Leading>
          <ListingToolbar.Trailing>
            <ListingToolbar.ViewToggle onChange={setView} value={view} />
          </ListingToolbar.Trailing>
        </ListingToolbar>
        <MarketplaceView
          installedSkillNames={installedSkillNames ?? new Set()}
          isInstalling={isInstalling}
          isRemoving={isRemoving}
          isSearchEnabled={enabled}
          isSearching={isSearching}
          isUpdating={isUpdating}
          listings={listings}
          onClearSearch={() => setQuery("")}
          onInstall={() => undefined}
          onRemove={() => undefined}
          onUpdate={() => undefined}
          searchError={searchError}
          view={view}
        />
      </div>
    </PanelSurface>
  );
}

export const SearchPrompt: Story = {
  render: () => <MarketplaceViewHarness />,
};

export const SearchResults: Story = {
  render: () => (
    <MarketplaceViewHarness initialQuery="demo" listings={skillMarketplaceListingFixtures} />
  ),
};

export const WithInstalled: Story = {
  render: () => (
    <MarketplaceViewHarness
      initialQuery="demo"
      installedSkillNames={new Set([skillMarketplaceListingFixtures[0].name])}
      listings={skillMarketplaceListingFixtures}
    />
  ),
};

export const Loading: Story = {
  render: () => <MarketplaceViewHarness initialQuery="demo" isSearching listings={[]} />,
};

export const ErrorState: Story = {
  render: () => (
    <MarketplaceViewHarness
      initialQuery="demo"
      listings={[]}
      searchError={new Error("Marketplace search failed with 503")}
    />
  ),
};

export const NoResults: Story = {
  render: () => <MarketplaceViewHarness initialQuery="demo" listings={[]} />,
};

export const InstallingDisablesAction: Story = {
  render: () => (
    <MarketplaceViewHarness
      initialQuery="demo"
      isInstalling
      listings={skillMarketplaceListingFixtures}
    />
  ),
};

export const RemoveConfirmation: Story = {
  tags: ["play-fn"],
  render: () => (
    <MarketplaceViewHarness
      initialQuery="demo"
      installedSkillNames={new Set([skillMarketplaceListingFixtures[0].name])}
      listings={skillMarketplaceListingFixtures}
    />
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const removeBtn = await canvas.findByTestId(
      `remove-btn-${skillMarketplaceListingFixtures[0].name}`
    );
    await userEvent.click(removeBtn);
    await expect(
      within(document.body).findByTestId(`remove-dialog-${skillMarketplaceListingFixtures[0].name}`)
    ).resolves.toBeDefined();
  },
};
