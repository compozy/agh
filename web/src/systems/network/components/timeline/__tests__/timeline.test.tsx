// @vitest-environment jsdom

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { Timeline } from "../timeline";
import type { NetworkConversationMessage } from "../../../types";

function makeMessage(overrides: Partial<NetworkConversationMessage>): NetworkConversationMessage {
  return {
    body: { text: overrides.text ?? "Hello" },
    channel: "ops",
    direction: "sent",
    display_name: overrides.display_name ?? "Codex",
    kind: overrides.kind ?? "say",
    local: true,
    message_id: overrides.message_id ?? "msg-1",
    peer_from: overrides.peer_from ?? "peer-codex",
    preview_text: overrides.text ?? "Hello",
    session_id: overrides.session_id ?? "sess-1",
    text: overrides.text ?? "Hello",
    timestamp: overrides.timestamp ?? "2026-04-17T14:32:00Z",
    ...overrides,
  } as NetworkConversationMessage;
}

describe("Timeline", () => {
  it("Should render a full message row with avatar, name, role chip, and body", () => {
    render(
      <Timeline
        messages={[
          makeMessage({
            message_id: "m1",
            text: "Sample body content",
          }),
        ]}
      />
    );

    expect(screen.getByTestId("network-message-row-full")).toBeInTheDocument();
    expect(screen.getByTestId("network-message-avatar")).toBeInTheDocument();
    expect(screen.getByTestId("network-message-role-chip")).toHaveTextContent("agent");
    expect(screen.getByText("Sample body content")).toBeInTheDocument();
  });

  it("Should render a collapsed continuation row that suppresses the role chip", () => {
    render(
      <Timeline
        messages={[
          makeMessage({ message_id: "m1", timestamp: "2026-04-17T14:32:00Z" }),
          makeMessage({
            message_id: "m2",
            text: "Continuation",
            timestamp: "2026-04-17T14:32:30Z",
          }),
        ]}
      />
    );

    const collapsed = screen.getByTestId("network-message-row-collapsed");
    expect(collapsed).toBeInTheDocument();
    // Role chip is rendered only on the FIRST row of a group.
    const roleChips = screen.queryAllByTestId("network-message-role-chip");
    expect(roleChips).toHaveLength(1);
  });

  it("Should never render a kind chip for kind say", () => {
    render(<Timeline messages={[makeMessage({ message_id: "m1", text: "default kind" })]} />);
    expect(screen.queryByTestId("network-message-kind-chip")).toBeNull();
  });

  it("Should render system kinds as a single-line system row", () => {
    render(
      <Timeline messages={[makeMessage({ kind: "trace", message_id: "m1", text: "tracing" })]} />
    );

    expect(screen.getByTestId("network-message-row-system")).toBeInTheDocument();
  });

  it("Should render a date pill across midnight boundaries", () => {
    render(
      <Timeline
        messages={[
          makeMessage({ message_id: "m1", timestamp: "2026-04-17T23:50:00Z" }),
          makeMessage({ message_id: "m2", timestamp: "2026-04-18T00:10:00Z" }),
        ]}
        now={new Date("2026-04-18T00:30:00Z")}
      />
    );

    const pills = screen.getAllByTestId("network-timeline-date-pill");
    expect(pills.length).toBeGreaterThanOrEqual(2);
  });

  it("Should render the New divider at the boundary of the last-read timestamp", () => {
    render(
      <Timeline
        lastReadAt="2026-04-17T14:32:30Z"
        messages={[
          makeMessage({ message_id: "m1", timestamp: "2026-04-17T14:32:00Z" }),
          makeMessage({ message_id: "m2", timestamp: "2026-04-17T14:33:00Z" }),
        ]}
      />
    );

    expect(screen.getByTestId("network-timeline-new-divider")).toBeInTheDocument();
  });

  it("Should reveal the timestamp on collapsed gutter hover", async () => {
    const user = userEvent.setup();
    render(
      <Timeline
        messages={[
          makeMessage({ message_id: "m1", timestamp: "2026-04-17T14:32:00Z" }),
          makeMessage({
            message_id: "m2",
            text: "next",
            timestamp: "2026-04-17T14:32:30Z",
          }),
        ]}
      />
    );

    const collapsedTimestamp = screen.getByTestId("network-message-collapsed-timestamp");
    expect(collapsedTimestamp).toBeInTheDocument();
    expect(collapsedTimestamp.className).toContain("opacity-0");
    await user.hover(screen.getByTestId("network-message-row-collapsed"));
    // jsdom does not apply group-hover variants, so we assert the title carries the ISO.
    expect(collapsedTimestamp.getAttribute("title")).toMatch(/\d{4}-\d{2}-\d{2}T/);
  });

  it("Should render the avatar with a 4px corner radius (no circles)", () => {
    render(<Timeline messages={[makeMessage({ message_id: "m1" })]} />);

    const avatar = screen.getByTestId("network-message-avatar");
    expect(avatar.className).toContain("rounded-chip");
    expect(avatar.className).not.toContain("rounded-full");
  });

  it("Should not declare any box-shadow on the timeline subtree", () => {
    render(
      <Timeline
        messages={[
          makeMessage({ message_id: "m1" }),
          makeMessage({ message_id: "m2", text: "next", timestamp: "2026-04-17T14:32:20Z" }),
          makeMessage({ kind: "trace", message_id: "m3", text: "trace" }),
        ]}
      />
    );

    const root = screen.getByTestId("network-timeline");
    const allElements = root.querySelectorAll("*");
    for (const element of [root, ...allElements]) {
      expect(element.getAttribute("style") ?? "").not.toContain("box-shadow");
    }
  });
});
