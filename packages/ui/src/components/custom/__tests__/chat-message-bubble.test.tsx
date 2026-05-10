import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ChatMessageBubble } from "../chat-message-bubble";

describe("ChatMessageBubble", () => {
  it("Should render a right-aligned surface-elevated bubble for role='user'", () => {
    const { container } = render(
      <ChatMessageBubble role="user" meta="YOU · 12:02">
        Find the event mapper.
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    const body = container.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
    const meta = container.querySelector<HTMLElement>('[data-slot="chat-message-meta"]');
    expect(root?.getAttribute("data-role")).toBe("user");
    expect(root?.getAttribute("data-align")).toBe("right");
    expect(root?.className).toContain("justify-end");
    expect(body?.className).toContain("bg-[color:var(--elevated)]");
    expect(body?.className).toContain("rounded-[var(--radius-lg)]");
    expect(body?.className).toContain("text-[color:var(--fg)]");
    expect(body?.textContent).toContain("Find the event mapper.");
    expect(meta?.textContent).toBe("YOU · 12:02");
    expect(meta?.className).toContain("text-right");
    expect(meta?.className).toContain("font-mono");
    expect(meta?.className).toContain("uppercase");
  });

  it("Should render role='agent' left-aligned with no bubble wrapper", () => {
    const { container } = render(
      <ChatMessageBubble
        role="agent"
        meta={
          <>
            <span data-testid="dot" />
            <span data-testid="name">claude</span>
          </>
        }
      >
        I can see two candidates.
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    const body = container.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
    const meta = container.querySelector<HTMLElement>('[data-slot="chat-message-meta"]');
    expect(root?.getAttribute("data-role")).toBe("agent");
    expect(root?.getAttribute("data-align")).toBe("left");
    expect(root?.className).toContain("flex-col");
    expect(body?.className).not.toContain("bg-[color:var(--elevated)]");
    expect(body?.className).not.toContain("rounded-[var(--radius-lg)]");
    expect(body?.className).toContain("text-[color:var(--muted)]");
    expect(meta?.className).toContain("items-center");
    expect(meta?.querySelector('[data-testid="dot"]')).not.toBeNull();
    expect(meta?.querySelector('[data-testid="name"]')?.textContent).toBe("claude");
  });

  it("Should render role='system' as a divider row with hairlines flanking the body", () => {
    const { container } = render(
      <ChatMessageBubble role="system">Session resumed</ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    const body = container.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
    expect(root?.getAttribute("data-role")).toBe("system");
    expect(root?.className).toContain("items-center");
    const dividers = root?.querySelectorAll('span[aria-hidden="true"]') ?? [];
    expect(dividers.length).toBe(2);
    for (const divider of Array.from(dividers)) {
      expect(divider.className).toContain("bg-[color:var(--line)]");
      expect(divider.className).toContain("h-px");
      expect(divider.className).toContain("flex-1");
    }
    expect(body?.className).toContain("font-mono");
    expect(body?.textContent).toBe("Session resumed");
  });

  it("Should place the meta slot above the body for role='user'", () => {
    const { container } = render(
      <ChatMessageBubble role="user" meta="YOU · 12:02">
        hello
      </ChatMessageBubble>
    );
    const inner = container.querySelector<HTMLElement>('[data-slot="chat-message-inner"]');
    const slots = inner ? Array.from(inner.children).map(el => el.getAttribute("data-slot")) : [];
    expect(slots[0]).toBe("chat-message-meta");
    expect(slots[1]).toBe("chat-message-body");
  });

  it("Should place the meta slot beside the agent name (inline row) for role='agent'", () => {
    const { container } = render(
      <ChatMessageBubble
        role="agent"
        meta={
          <>
            <span data-testid="dot" />
            <span data-testid="name">claude</span>
          </>
        }
      >
        body
      </ChatMessageBubble>
    );
    const meta = container.querySelector<HTMLElement>('[data-slot="chat-message-meta"]');
    expect(meta?.className).toContain("flex");
    expect(meta?.className).toContain("items-center");
    expect(meta?.className).toContain("gap-2");
    const children = Array.from(meta?.children ?? []);
    expect(children.length).toBeGreaterThanOrEqual(2);
    expect(children[0]?.getAttribute("data-testid")).toBe("dot");
    expect(children[1]?.getAttribute("data-testid")).toBe("name");
  });

  it.each(["tool", "diff"] as const)(
    "Should render role='%s' as a left-aligned pass-through container",
    role => {
      const { container } = render(
        <ChatMessageBubble role={role}>
          <div data-testid="inner-card">payload</div>
        </ChatMessageBubble>
      );
      const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
      const body = container.querySelector<HTMLElement>('[data-slot="chat-message-body"]');
      expect(root?.getAttribute("data-role")).toBe(role);
      expect(root?.getAttribute("data-align")).toBe("left");
      expect(root?.className).toContain("flex-col");
      expect(body?.className ?? "").not.toContain("bg-[color:var(--elevated)]");
      expect(body?.querySelector('[data-testid="inner-card"]')?.textContent).toBe("payload");
    }
  );

  it("Should honour an explicit align override", () => {
    const { container } = render(
      <ChatMessageBubble role="user" align="left">
        override
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    expect(root?.getAttribute("data-align")).toBe("left");
    expect(root?.className).toContain("justify-start");
  });

  it.each(["agent", "tool", "diff"] as const)("Should honour align='right' for role='%s'", role => {
    const { container } = render(
      <ChatMessageBubble role={role} align="right" meta="META">
        payload
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    const meta = container.querySelector<HTMLElement>('[data-slot="chat-message-meta"]');
    expect(root?.getAttribute("data-align")).toBe("right");
    expect(root?.className).toContain("items-end");
    expect(root?.className).toContain("text-right");
    expect(meta?.className).toContain("justify-end");
  });

  it("Should keep role='system' centered even when align is overridden", () => {
    const { container } = render(
      <ChatMessageBubble role="system" align="right">
        Session resumed
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    expect(root?.getAttribute("data-align")).toBe("right");
    expect(root?.className).toContain("items-center");
    expect(root?.className).not.toContain("items-end");
  });

  it("Should forward extra HTML props to the root", () => {
    const { container } = render(
      <ChatMessageBubble role="agent" className="ring-1" data-testid="m1">
        body
      </ChatMessageBubble>
    );
    const root = container.querySelector<HTMLElement>('[data-slot="chat-message"]');
    expect(root?.getAttribute("data-testid")).toBe("m1");
    expect(root?.className).toContain("ring-1");
  });
});
