import { useState } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { Button } from "@agh/ui";

import { PanelSurface } from "@/storybook/story-layout";
import type { VaultSecret } from "@/systems/vault/types";

import { SessionInspector, type SessionInspectorProps } from "../session-inspector";

const vaultSecrets: VaultSecret[] = [
  {
    ref: "vault:sessions/session_launch_coordination/stripe_api_key",
    namespace: "sessions",
    kind: "api_key",
    present: true,
    created_at: "2026-04-17T18:00:00Z",
    updated_at: "2026-04-17T18:14:00Z",
  },
];

const baseArgs: SessionInspectorProps = {
  messages: [],
  sessionId: "session_launch_coordination",
  usage: {
    tokensIn: 128_400,
    tokensOut: 24_900,
    costUsd: 18.42,
    ratePerSecond: 42.1,
    tokensInDelta: 4.8,
    tokensOutDelta: -2.1,
    costDelta: 1.2,
  },
  vaultSecrets,
  files: [
    { path: "web/src/systems/session/components/session-inspector.tsx", readCount: 3 },
    { path: "internal/session/runtime.go", readCount: 1 },
  ],
  onViewAllTrace: fn(),
};

const meta: Meta<typeof SessionInspector> = {
  title: "systems/session/SessionInspector",
  component: SessionInspector,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component:
          "Right-side session inspector consuming <DetailInspector>: inline at >= 1440 px viewport, drawer below.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[640px] justify-end">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Inline inspector at >= 1440 px viewport renders 5 flat tabs
 * (Trace, Usage, Memory, Files, Vault) inside the DetailInspector chrome.
 */
export const Inline: Story = {
  args: baseArgs,
  parameters: {
    viewport: { defaultViewport: "responsive" },
  },
};

/**
 * Drawer variant exposes the same body inside the sheet drawer when the
 * viewport is < 1440 px. Caller owns the open state.
 */
export const Drawer: Story = {
  args: baseArgs,
  render: function DrawerStory(args) {
    const [open, setOpen] = useState(false);
    return (
      <div className="flex h-full w-full items-start justify-end gap-2 p-4">
        <Button type="button" variant="neutral" size="sm" onClick={() => setOpen(true)}>
          Open inspector
        </Button>
        <SessionInspector {...args} drawerOpen={open} onDrawerOpenChange={setOpen} />
      </div>
    );
  },
};

/**
 * Empty inspector keeps each tab truthful before the first turn completes.
 */
export const Empty: Story = {
  args: {
    messages: [],
    sessionId: "session_empty",
    usage: null,
    vaultSecrets: [],
    files: [],
    onViewAllTrace: fn(),
  },
};
