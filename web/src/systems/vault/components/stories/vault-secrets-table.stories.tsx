import type { Meta, StoryObj } from "@storybook/react-vite";
import { fn } from "storybook/test";

import { PanelSurface } from "@/storybook/story-layout";
import type { VaultSecret } from "@/systems/vault/types";

import { VaultSecretsTable } from "../vault-secrets-table";

const secrets: VaultSecret[] = [
  {
    ref: "vault:providers/codex/api_key",
    namespace: "providers",
    kind: "api_key",
    present: true,
    created_at: "2026-04-17T17:30:00Z",
    updated_at: "2026-04-17T17:42:00Z",
  },
  {
    ref: "vault:bridges/slack/signing_secret",
    namespace: "bridges",
    kind: "signing_secret",
    present: true,
    created_at: "2026-04-17T17:31:00Z",
    updated_at: "2026-04-17T17:55:00Z",
  },
  {
    ref: "vault:sessions/session_launch_coordination/partner_webhook_secret",
    namespace: "sessions",
    kind: "webhook",
    present: true,
    created_at: "2026-04-17T17:58:00Z",
    updated_at: "2026-04-17T18:18:00Z",
  },
];

const meta: Meta<typeof VaultSecretsTable> = {
  title: "systems/vault/VaultSecretsTable",
  component: VaultSecretsTable,
  parameters: {
    layout: "fullscreen",
    docs: {
      description: {
        component: "Vault metadata table with namespace chips and optional destructive actions.",
      },
    },
  },
  decorators: [
    Story => (
      <PanelSurface className="min-h-[420px] p-6">
        <Story />
      </PanelSurface>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof meta>;

/**
 * Read-only table shown on the settings vault page.
 */
export const Default: Story = {
  args: {
    secrets,
  },
};

/**
 * Delete actions appear only when the parent route wires the mutation.
 */
export const WithDeleteActions: Story = {
  args: {
    secrets,
    onDelete: fn(),
  },
};

/**
 * Loading state preserves table footprint while metadata resolves.
 */
export const Loading: Story = {
  args: {
    secrets: [],
    isLoading: true,
  },
};

/**
 * Empty state uses caller-provided copy for scoped vault views.
 */
export const Empty: Story = {
  args: {
    secrets: [],
    emptyTitle: "No provider secrets",
    emptyDescription: "Provider credential metadata appears here after a secret is stored.",
  },
};
