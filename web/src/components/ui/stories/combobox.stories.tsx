import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  Combobox,
  ComboboxCollection,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "../combobox";

const meta: Meta<typeof Combobox> = {
  title: "components/ui/Combobox",
  component: Combobox,
  parameters: {
    layout: "centered",
    docs: {
      description: {
        component:
          "Base UI Combobox with autofilter. Pass `items` for built-in filtering; type in the input to exercise keyboard navigation.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

type CityOption = { value: string; label: string };

const cities: CityOption[] = [
  { value: "albuquerque", label: "Albuquerque" },
  { value: "alexandria", label: "Alexandria" },
  { value: "amsterdam", label: "Amsterdam" },
  { value: "berlin", label: "Berlin" },
  { value: "boston", label: "Boston" },
  { value: "chicago", label: "Chicago" },
  { value: "seattle", label: "Seattle" },
];

export const Default: Story = {
  args: {},
  render: () => (
    <div className="w-80">
      <Combobox items={cities}>
        <ComboboxInput placeholder="Search cities" />
        <ComboboxContent>
          <ComboboxList>
            <ComboboxEmpty>No matches</ComboboxEmpty>
            <ComboboxCollection>
              {(item: CityOption) => (
                <ComboboxItem key={item.value} value={item}>
                  {item.label}
                </ComboboxItem>
              )}
            </ComboboxCollection>
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </div>
  ),
};

export const Filtering: Story = {
  args: {},
  parameters: {
    docs: {
      description: {
        story: "Type 'al' to narrow the list to items starting with that prefix.",
      },
    },
  },
  render: () => (
    <div className="w-80">
      <Combobox items={cities} defaultOpen>
        <ComboboxInput placeholder="Try typing 'al'" />
        <ComboboxContent>
          <ComboboxList>
            <ComboboxEmpty>No matches</ComboboxEmpty>
            <ComboboxCollection>
              {(item: CityOption) => (
                <ComboboxItem key={item.value} value={item}>
                  {item.label}
                </ComboboxItem>
              )}
            </ComboboxCollection>
          </ComboboxList>
        </ComboboxContent>
      </Combobox>
    </div>
  ),
};
