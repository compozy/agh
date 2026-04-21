import { useEffect } from "react";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { CenteredSurface } from "@/storybook/story-layout";
import { useSessionStore } from "@/systems/session/hooks/use-session-store";

import {
  MessageComposer,
  type MessageComposerAttachment,
  type MessageComposerChannel,
  type MessageComposerPayload,
} from "../message-composer";

const channels: MessageComposerChannel[] = [
  { id: "storybook", name: "storybook" },
  { id: "release", name: "release" },
];

const attachOptions: MessageComposerAttachment[] = [
  { id: "spec-md", name: "spec.md" },
  { id: "plan-md", name: "plan.md" },
  { id: "diagram-png", name: "diagram.png" },
];

const noopSend = (_payload: MessageComposerPayload) => undefined;

function ComposerFrame({ children }: { children: React.ReactNode }) {
  return (
    <CenteredSurface>
      <div className="w-full max-w-3xl rounded-2xl border border-[color:var(--color-divider)] bg-[color:var(--color-canvas)]">
        {children}
      </div>
    </CenteredSurface>
  );
}

function SeedDraftComposer({ sessionId, draftText }: { sessionId: string; draftText: string }) {
  useEffect(() => {
    useSessionStore.getState().setDraft(sessionId, { text: draftText });
    return () => {
      useSessionStore.getState().clearDraft(sessionId);
    };
  }, [sessionId, draftText]);
  return (
    <MessageComposer
      sessionId={sessionId}
      onSend={noopSend}
      channels={channels}
      attachOptions={attachOptions}
    />
  );
}

const meta: Meta<typeof MessageComposer> = {
  title: "systems/session/MessageComposer",
  component: MessageComposer,
  parameters: {
    layout: "centered",
  },
};

export default meta;
type Story = StoryObj<typeof meta>;

export const Empty: Story = {
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={noopSend} channels={channels} attachOptions={attachOptions} />
    </ComposerFrame>
  ),
};

export const Typing: Story = {
  render: () => (
    <ComposerFrame>
      <SeedDraftComposer
        sessionId="story-typing"
        draftText="Draft a release note for the composer rebuild."
      />
    </ComposerFrame>
  ),
};

export const Disabled: Story = {
  render: () => (
    <ComposerFrame>
      <MessageComposer
        disabled
        onSend={noopSend}
        channels={channels}
        attachOptions={attachOptions}
      />
    </ComposerFrame>
  ),
};

export const WithAttachOpen: Story = {
  tags: ["play-fn"],
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={noopSend} channels={channels} attachOptions={attachOptions} />
    </ComposerFrame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByTestId("composer-attach-pill"));
    await waitFor(() =>
      expect(
        within(document.body).getByTestId("composer-attach-option-spec-md")
      ).toBeInTheDocument()
    );
    await userEvent.click(within(document.body).getByTestId("composer-attach-option-spec-md"));
    await waitFor(() =>
      expect(
        within(canvasElement).getByTestId("composer-attachment-name-spec-md")
      ).toHaveTextContent("spec.md")
    );
  },
};

export const WithChannelPickerOpen: Story = {
  tags: ["play-fn"],
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={noopSend} channels={channels} attachOptions={attachOptions} />
    </ComposerFrame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.click(canvas.getByTestId("composer-channel-pill"));
    await waitFor(() =>
      expect(within(document.body).getByTestId("composer-channel-item-release")).toBeInTheDocument()
    );
  },
};

export const FocusBorder: Story = {
  tags: ["play-fn"],
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={noopSend} channels={channels} attachOptions={attachOptions} />
    </ComposerFrame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const container = canvas.getByTestId("composer-container");
    const textarea = canvas.getByTestId("composer-textarea");
    await userEvent.click(textarea);
    await expect(container.className).toMatch(
      /focus-within:border-\[color:var\(--color-accent\)\]/
    );
    textarea.blur();
    await expect(container.className).toMatch(/border-\[color:var\(--color-divider\)\]/);
  },
};

export const SendKeyboardShortcut: Story = {
  tags: ["play-fn"],
  render: () => (
    <ComposerFrame>
      <MessageComposer onSend={noopSend} channels={channels} attachOptions={attachOptions} />
    </ComposerFrame>
  ),
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    const textarea = canvas.getByTestId("composer-textarea") as HTMLTextAreaElement;
    await userEvent.click(textarea);
    await userEvent.type(textarea, "ship it");
    await userEvent.keyboard("{Enter}");
    await waitFor(() => expect(textarea).toHaveValue(""));
  },
};
