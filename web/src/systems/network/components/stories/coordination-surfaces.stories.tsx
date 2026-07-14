import type { Meta, StoryObj } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";

import { PanelSurface } from "@/storybook/story-layout";
import { NetworkCoordinationInvitation } from "../coordination-invitation";
import { TaskRunConversationPanel } from "../task-run-conversation-panel";

function wrap(node: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return <QueryClientProvider client={client}>{node}</QueryClientProvider>;
}

const meta: Meta = {
  title: "systems/network/components/CoordinationSurfaces",
  parameters: {
    layout: "fullscreen",
  },
};

export default meta;
type Story = StoryObj;

export const InvitationVisible: Story = {
  render: () => (
    <PanelSurface className="min-h-[240px] p-6">
      <NetworkCoordinationInvitation
        onAccept={() => undefined}
        onDismiss={() => undefined}
        visible
      />
    </PanelSurface>
  ),
};

export const ConversationEmpty: Story = {
  render: () =>
    wrap(
      <PanelSurface className="min-h-[320px] p-6">
        <TaskRunConversationPanel
          boundsLabel="Participation live · builders"
          conversationEmpty
          messageCount={0}
        />
      </PanelSurface>
    ),
};
