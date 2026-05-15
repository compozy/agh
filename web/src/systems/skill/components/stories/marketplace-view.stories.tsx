import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { expect, userEvent, within } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import { skillMarketplaceListingFixtures } from "@/systems/skill/mocks";
import type { SkillMarketplaceListingPayload } from "@/systems/skill";

import { MarketplaceView } from "../marketplace-view";

const meta: Meta<typeof MarketplaceView> = {
  title: "systems/skill/MarketplaceView",
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
}: StoryHarnessProps) {
  const [query, setQuery] = useState(initialQuery);
  const enabled = isSearchEnabled ?? query.trim() !== "";
  return (
    <PanelSurface>
      <MarketplaceView
        installedSkillNames={installedSkillNames ?? new Set()}
        isInstalling={isInstalling}
        isRemoving={isRemoving}
        isSearchEnabled={enabled}
        isSearching={isSearching}
        isUpdating={isUpdating}
        listings={listings}
        onInstall={() => undefined}
        onRemove={() => undefined}
        onSearchChange={setQuery}
        onUpdate={() => undefined}
        searchError={searchError}
        searchQuery={query}
      />
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
