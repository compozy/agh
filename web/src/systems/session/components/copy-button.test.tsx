import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

import { CopyButton } from "./copy-button";

describe("CopyButton", () => {
  let clipboardDescriptorBeforeTest: PropertyDescriptor | undefined;

  beforeEach(() => {
    vi.useFakeTimers();
    clipboardDescriptorBeforeTest = Object.getOwnPropertyDescriptor(navigator, "clipboard");
  });

  afterEach(() => {
    if (clipboardDescriptorBeforeTest) {
      Object.defineProperty(navigator, "clipboard", clipboardDescriptorBeforeTest);
    } else {
      Reflect.deleteProperty(navigator, "clipboard");
    }
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("marks the button copied only after clipboard writes successfully", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<CopyButton text="hello world" ariaLabel="Copy message" className="rounded-md" />);

    const button = screen.getByRole("button", { name: "Copy message" });
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });

    expect(writeText).toHaveBeenCalledWith("hello world");
    expect(button).toHaveAttribute("data-state", "copied");
    expect(vi.getTimerCount()).toBe(1);

    act(() => {
      vi.advanceTimersByTime(1200);
    });

    expect(button).toHaveAttribute("data-state", "idle");
  });

  it("logs clipboard failures and keeps the button idle", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("permission denied"));
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => undefined);

    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<CopyButton text="hello world" ariaLabel="Copy message" className="rounded-md" />);

    const button = screen.getByRole("button", { name: "Copy message" });
    await act(async () => {
      fireEvent.click(button);
      await Promise.resolve();
    });

    expect(writeText).toHaveBeenCalledWith("hello world");
    expect(consoleError).toHaveBeenCalled();
    expect(button).toHaveAttribute("data-state", "idle");
    expect(vi.getTimerCount()).toBe(0);
  });
});
