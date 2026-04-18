import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

import { MessageComposer } from "./message-composer";

describe("MessageComposer", () => {
  it("renders input container with rounded border and surface background", () => {
    render(<MessageComposer onSend={vi.fn()} />);
    const container = screen.getByTestId("composer-container");
    expect(container.className).toContain("rounded-xl");
    expect(container.className).toMatch(/border-\[color:var\(--color-divider\)\]/);
    expect(container.className).toMatch(/bg-\[color:var\(--color-surface\)\]/);
  });

  it("gains accent border color on focus-within", () => {
    render(<MessageComposer onSend={vi.fn()} />);
    const container = screen.getByTestId("composer-container");
    expect(container.className).toMatch(/focus-within:border-\[color:var\(--color-accent\)\]/);
  });

  it("renders circular send button with accent background", () => {
    render(<MessageComposer onSend={vi.fn()} />);
    const sendButton = screen.getByRole("button", { name: "Send message" });
    expect(sendButton.className).toContain("rounded-full");
    expect(sendButton.className).toMatch(/bg-\[color:var\(--color-accent\)\]/);
    expect(sendButton.className).toContain("text-white");
    expect(sendButton.className).toContain("size-9");
  });

  it("calls onSend on Enter key press", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea");
    fireEvent.input(textarea, { target: { value: "Hello world" } });
    (textarea as HTMLTextAreaElement).value = "Hello world";

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

    expect(onSend).toHaveBeenCalledWith("Hello world");
  });

  it("inserts newline on Shift+Enter (does not send)", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea");
    (textarea as HTMLTextAreaElement).value = "Line one";

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: true });

    expect(onSend).not.toHaveBeenCalled();
  });

  it("is disabled when disabled prop is true", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} disabled />);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    expect(textarea).toBeDisabled();

    const sendButton = screen.getByTestId("composer-send-button");
    expect(sendButton).toBeDisabled();
  });

  it("does not send on Enter when disabled", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} disabled />);

    const textarea = screen.getByTestId("composer-textarea");
    (textarea as HTMLTextAreaElement).value = "Hello";

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

    expect(onSend).not.toHaveBeenCalled();
  });

  it("does not send empty messages", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea");
    (textarea as HTMLTextAreaElement).value = "   ";

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

    expect(onSend).not.toHaveBeenCalled();
  });

  it("clears textarea after sending", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    textarea.value = "Hello";

    fireEvent.keyDown(textarea, { key: "Enter", shiftKey: false });

    expect(textarea.value).toBe("");
  });

  it("sends via send button click", () => {
    const onSend = vi.fn();
    render(<MessageComposer onSend={onSend} />);

    const textarea = screen.getByTestId("composer-textarea") as HTMLTextAreaElement;
    textarea.value = "Click send";

    fireEvent.click(screen.getByTestId("composer-send-button"));

    expect(onSend).toHaveBeenCalledWith("Click send");
  });
});
