import type { Meta, StoryObj } from "@storybook/react-vite";

import { CenteredSurface } from "@/storybook/story-layout";
import { emptySearchToolMessageFixture, searchToolMessageFixture } from "@/systems/session/mocks";

import { SearchContent } from "../search-content";

const meta: Meta<typeof SearchContent> = {
  title: "systems/session/tool-renderers/SearchContent",
  component: SearchContent,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

function SearchFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)] p-4">
        {children}
      </div>
    </CenteredSurface>
  );
}

export const Default: Story = {
  render: () => (
    <SearchFrame>
      <SearchContent message={searchToolMessageFixture} />
    </SearchFrame>
  ),
};

export const EmptyResultSet: Story = {
  render: () => (
    <SearchFrame>
      <SearchContent message={emptySearchToolMessageFixture} />
    </SearchFrame>
  ),
};
