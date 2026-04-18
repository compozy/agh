import type { Meta, StoryObj } from "@storybook/react-vite";
import { useState } from "react";
import { PlusIcon } from "lucide-react";

import { Button } from "../button";
import { Pills } from "../pills";
import { SearchInput } from "../search-input";
import { Toolbar } from "../toolbar";

const meta: Meta<typeof Toolbar> = {
  title: "ui/Toolbar",
  component: Toolbar,
  parameters: {
    layout: "padded",
    docs: {
      description: {
        component:
          "Composition-first toolbar shell — pass `SearchInput`, `Pills`, `Button` children directly. Wraps on narrow viewports.",
      },
    },
  },
  tags: ["autodocs"],
};

export default meta;
type Story = StoryObj<typeof meta>;

function Harness() {
  const [mode, setMode] = useState<"list" | "kanban">("list");
  const [search, setSearch] = useState("");
  return (
    <Toolbar aria-label="Tasks toolbar">
      <Pills
        value={mode}
        onChange={setMode}
        items={[
          { value: "list", label: "List" },
          { value: "kanban", label: "Kanban" },
        ]}
      />
      <SearchInput
        value={search}
        onChange={setSearch}
        placeholder="Search tasks…"
        aria-label="Search tasks"
        containerClassName="ml-auto w-64"
      />
      <Button size="sm" type="button">
        <PlusIcon className="size-3.5" />
        New
      </Button>
    </Toolbar>
  );
}

export const Basic: Story = {
  render: () => (
    <div className="w-[720px]">
      <Harness />
    </div>
  ),
};

export const NarrowWrap: Story = {
  render: () => (
    <div className="w-[380px]">
      <Harness />
    </div>
  ),
};

export const Sticky: Story = {
  render: () => (
    <div className="w-[560px] max-h-48 overflow-auto rounded-md border border-[color:var(--color-divider)]">
      <Toolbar sticky>
        <Pills
          value="list"
          onChange={() => {}}
          items={[
            { value: "list", label: "List" },
            { value: "kanban", label: "Kanban" },
          ]}
        />
        <Button size="sm" type="button">
          <PlusIcon className="size-3.5" />
          New
        </Button>
      </Toolbar>
      <div className="h-96 p-4 text-sm text-[color:var(--color-text-secondary)]">
        Scroll — the toolbar stays pinned to the top edge of the scroll container.
      </div>
    </div>
  ),
};
