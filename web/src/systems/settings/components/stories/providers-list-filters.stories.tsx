import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import {
  DEFAULT_PROVIDER_FILTERS,
  type ProviderFilterState,
} from "@/systems/settings/lib/providers-list-filters";

import { ProvidersListFilters } from "../providers-list-filters";

interface HarnessProps {
  initial?: Partial<ProviderFilterState>;
  totalCount?: number;
  visibleCount?: number;
}

function FiltersHarness({ initial, totalCount = 26, visibleCount }: HarnessProps) {
  const [filters, setFilters] = useState<ProviderFilterState>({
    ...DEFAULT_PROVIDER_FILTERS,
    ...initial,
  });
  const effectiveVisible =
    visibleCount ?? (filters.statusFilter || filters.nameQuery.trim().length > 0 ? 8 : totalCount);

  return (
    <div className="flex flex-col gap-4 p-6">
      <ProvidersListFilters
        statusFilter={filters.statusFilter}
        sourceFilter={filters.sourceFilter}
        harnessFilter={filters.harnessFilter}
        authModeFilter={filters.authModeFilter}
        defaultFilter={filters.defaultFilter}
        nameQuery={filters.nameQuery}
        visibleCount={effectiveVisible}
        totalCount={totalCount}
        onStatusChange={value => setFilters(prev => ({ ...prev, statusFilter: value }))}
        onSourceChange={value => setFilters(prev => ({ ...prev, sourceFilter: value }))}
        onHarnessChange={value => setFilters(prev => ({ ...prev, harnessFilter: value }))}
        onAuthModeChange={value => setFilters(prev => ({ ...prev, authModeFilter: value }))}
        onDefaultChange={value => setFilters(prev => ({ ...prev, defaultFilter: value }))}
        onNameQueryChange={value => setFilters(prev => ({ ...prev, nameQuery: value }))}
      />
      <pre className="rounded-md bg-canvas-soft p-3 font-mono text-xs text-muted">
        {JSON.stringify(filters, null, 2)}
      </pre>
    </div>
  );
}

const meta: Meta<typeof ProvidersListFilters> = {
  title: "systems/settings/ProvidersListFilters",
  component: ProvidersListFilters,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Filter bar that sits above the providers grid. Mirrors the tasks-list-filters pattern: a search input for nameQuery, a `<Filters>` chip primitive for status/source/harness/auth_mode/default, and a right-aligned visible/total match count.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[320px] p-0">
        <Story />
      </PanelSurface>
    ),
  ],
  args: {
    onStatusChange: fn(),
    onSourceChange: fn(),
    onHarnessChange: fn(),
    onAuthModeChange: fn(),
    onDefaultChange: fn(),
    onNameQueryChange: fn(),
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {},
  render: () => <FiltersHarness />,
};

export const WithStatusFilter: Story = {
  args: {},
  render: () => <FiltersHarness initial={{ statusFilter: "unconfigured" }} visibleCount={8} />,
};

export const WithSearchQuery: Story = {
  args: {},
  render: () => <FiltersHarness initial={{ nameQuery: "claude" }} visibleCount={2} />,
};
