import { beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";

import { useSessionStore } from "../hooks/use-session-store";
import { MessageComposer, type MessageComposerPayload } from "./message-composer";

function resetStore() {
  useSessionStore.setState({
    activeSessionId: null,
    historyMessages: [],
    liveMessages: [],
    isStreaming: false,
    awaitingTranscriptSync: false,
    pendingPermission: null,
    drafts: {},
  });
}

function renderComposer(props: Partial<React.ComponentProps<typeof MessageComposer>> = {}) {
  const onSend = vi.fn<(payload: MessageComposerPayload) => void>();
  const utils = render(<MessageComposer onSend={onSend} {...props} />);
  return { onSend, ...utils };
}

describe("MessageComposer", () => {
  beforeEach(() => {
    resetStore();
    cleanup();
  });

  it("renders container with divider border, surface fill, and accent focus state", () => {
    renderComposer();

    const container = screen.getByTestId("composer-container");
    expect(container.className).toContain("rounded-xl");
    expect(container.className).toMatch(/border-\[color:var\(--color-divider\)\]/);
    expect(container.className).toMatch(/bg-\[color:var\(--color-surface\)\]/);
    expect(container.className).toMatch(/focus-within:border-\[color:var\(--color-accent\)\]/);
  });

  it("renders a 36px circular accent send button with SendHorizontal icon", () => {
    renderComposer();

    const sendButton = screen.getByTestId("composer-send-button");
    expect(sendButton.className).toContain("rounded-full");
    expect(sendButton.className).toContain("size-9");
    expect(sendButton.className).toMatch(/bg-\[color:var\(--color-accent\)\]/);
    expect(sendButton.querySelector("svg")).not.toBeNull();
  });

  it("Enter sends trimmed text, clears the textarea, and calls onSend with payload", () => {
    const { onSend } = renderComposer();
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "  Hello world  " } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(onSend).toHaveBeenCalledTimes(1);
    expect(onSend).toHaveBeenCalledWith({ text: "Hello world" });
    expect(textarea.value).toBe("");
  });

  it("Shift+Enter inserts a newline without calling onSend", () => {
    const { onSend } = renderComposer();
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "Line 1" } });
    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true });

    expect(onSend).not.toHaveBeenCalled();
    expect(textarea.value).toBe("Line 1");
  });

  it("does not send whitespace-only messages", () => {
    const { onSend } = renderComposer();
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "   " } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(onSend).not.toHaveBeenCalled();
  });

  it("disabled state makes textarea and send button unavailable", () => {
    const { onSend } = renderComposer({ disabled: true });
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    const sendButton = screen.getByTestId("composer-send-button") as HTMLButtonElement;

    expect(textarea).toBeDisabled();
    expect(sendButton).toBeDisabled();
    expect(sendButton.className).toMatch(/disabled:opacity-50/);
    expect(sendButton.className).toMatch(/disabled:cursor-not-allowed/);

    fireEvent.change(textarea, { target: { value: "Nope" } });
    fireEvent.keyDown(textarea, { key: "Enter" });
    fireEvent.click(sendButton);
    expect(onSend).not.toHaveBeenCalled();
  });

  it("inert state prevents interaction without relying only on disabled styling", () => {
    renderComposer({ inert: true, channels: [{ id: "release", name: "release" }] });

    const container = screen.getByTestId("composer-container");
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    const sendButton = screen.getByTestId("composer-send-button") as HTMLButtonElement;

    expect(container).toHaveAttribute("inert");
    expect(container.className).toMatch(/pointer-events-none/);
    expect(textarea).toBeDisabled();
    expect(sendButton).toBeDisabled();
  });

  it("clicks the send button to submit the payload", () => {
    const { onSend } = renderComposer();
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;

    fireEvent.change(textarea, { target: { value: "Via button" } });
    fireEvent.click(screen.getByTestId("composer-send-button"));

    expect(onSend).toHaveBeenCalledWith({ text: "Via button" });
  });

  it("auto-grows up to 200px and caps scroll height at that ceiling", () => {
    renderComposer();
    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;

    Object.defineProperty(textarea, "scrollHeight", { configurable: true, value: 60 });
    fireEvent.change(textarea, { target: { value: "one" } });
    expect(textarea.style.height).toBe("60px");

    Object.defineProperty(textarea, "scrollHeight", { configurable: true, value: 900 });
    fireEvent.change(textarea, { target: { value: "one\ntwo\nthree\nfour\nfive" } });
    expect(textarea.style.height).toBe("200px");
  });

  it("hides channel pill when no channels are provided, shows it when they are", () => {
    const { rerender, onSend } = renderComposer();
    expect(screen.queryByTestId("composer-channel-pill")).toBeNull();

    rerender(<MessageComposer onSend={onSend} channels={[{ id: "release", name: "release" }]} />);
    expect(screen.getByTestId("composer-channel-pill")).toBeInTheDocument();
  });

  it("persists text draft through the session store and survives unmount/remount", () => {
    const sessionId = "session-alpha";
    const { unmount } = render(<MessageComposer sessionId={sessionId} onSend={() => {}} />);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "draft survives" } });

    expect(useSessionStore.getState().drafts[sessionId]?.text).toBe("draft survives");

    unmount();
    cleanup();

    render(<MessageComposer sessionId={sessionId} onSend={() => {}} />);
    const remounted = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    expect(remounted.value).toBe("draft survives");
  });

  it("clears the persisted draft after send", () => {
    const sessionId = "session-beta";
    const onSend = vi.fn();
    render(<MessageComposer sessionId={sessionId} onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "ship it" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith({ text: "ship it" });
    expect(useSessionStore.getState().drafts[sessionId]).toBeUndefined();
  });

  it("attaches channel to the onSend payload after selecting a channel", async () => {
    const onSend = vi.fn();
    render(
      <MessageComposer
        onSend={onSend}
        channels={[
          { id: "release", name: "release" },
          { id: "storybook", name: "storybook" },
        ]}
      />
    );

    fireEvent.click(screen.getByTestId("composer-channel-pill"));
    const option = await waitFor(() => screen.getByTestId("composer-channel-item-release"));
    fireEvent.click(option);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "ping team" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith({ text: "ping team", channel: "release" });
  });

  it("attach popover adds a chip that reflects the file name", async () => {
    const onSend = vi.fn();
    render(
      <MessageComposer
        onSend={onSend}
        attachOptions={[
          { id: "spec-md", name: "spec.md" },
          { id: "plan-md", name: "plan.md" },
        ]}
      />
    );

    fireEvent.click(screen.getByTestId("composer-attach-pill"));
    const option = await waitFor(() => screen.getByTestId("composer-attach-option-spec-md"));
    fireEvent.click(option);

    const chip = await waitFor(() => screen.getByTestId("composer-attachment-name-spec-md"));
    expect(chip.textContent).toBe("spec.md");

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "review" } });
    fireEvent.keyDown(textarea, { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith({
      text: "review",
      attachments: [{ id: "spec-md", name: "spec.md" }],
    });
  });
});
