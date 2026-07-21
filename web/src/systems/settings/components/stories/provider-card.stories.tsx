import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import type { SettingsProviderEntry } from "@/systems/settings";
import { settingsProviderFixtures } from "@/systems/settings/mocks";

import { ProviderCard } from "../provider-card";
import { ProviderRow } from "../provider-row";

const claudeFixture = settingsProviderFixtures[0]!;
const codexFixture = settingsProviderFixtures[1]!;

const binaryMissingFixture: SettingsProviderEntry = {
  ...codexFixture,
  name: "qoder",
  default: false,
  command_available: false,
  settings: {
    ...codexFixture.settings,
    command: "qoder --acp",
    display_name: "Qoder CLI",
  },
  source_metadata: {
    available_targets: ["global-config"],
    effective_source: { kind: "builtin-provider", scope: "global" },
  },
  credentials: undefined,
};

const needsSetupFixture: SettingsProviderEntry = {
  ...codexFixture,
  name: "hermes",
  default: false,
  settings: {
    ...codexFixture.settings,
    command: "hermes --acp",
    display_name: "Hermes",
    auth_mode: "bound_secret",
  },
  credentials: [
    { name: "api_key", target_env: "HERMES_API_KEY", required: true, present: false },
  ] as SettingsProviderEntry["credentials"],
};

const meta: Meta<typeof ProviderCard> = {
  title: "systems/settings/components/ProviderCard",
  component: ProviderCard,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Provider card per the settings redesign: logo well, name + Default pill, mono command, one humanized status line, plain-language sign-in summary. The whole card opens the inspector sheet.",
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * The three listing states side by side: ready, needs setup, not installed.
 */
export const CardStates: Story = {
  args: {},
  render: () => (
    <PanelSurface className="grid max-w-[860px] gap-3 p-6 sm:grid-cols-3">
      <ProviderCard onOpen={fn()} provider={claudeFixture} />
      <ProviderCard onOpen={fn()} provider={needsSetupFixture} />
      <ProviderCard onOpen={fn()} provider={binaryMissingFixture} />
    </PanelSurface>
  ),
};

/**
 * Rows view of the same listing data.
 */
export const RowStates: Story = {
  args: {},
  render: () => (
    <PanelSurface className="max-w-[860px] p-6">
      <div className="overflow-hidden rounded-lg border border-line bg-canvas-soft">
        <ProviderRow onOpen={fn()} provider={claudeFixture} />
        <ProviderRow onOpen={fn()} provider={needsSetupFixture} />
        <ProviderRow onOpen={fn()} provider={binaryMissingFixture} />
      </div>
    </PanelSurface>
  ),
};
