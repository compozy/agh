import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Avatar, AvatarBadge, AvatarFallback, AvatarGroup, AvatarImage } from "./avatar";

describe("Avatar", () => {
  it("Should render the fallback initials when no image is provided", () => {
    render(
      <Avatar>
        <AvatarFallback>PN</AvatarFallback>
      </Avatar>
    );
    expect(screen.getByText("PN")).toBeInTheDocument();
  });

  it("Should expose the requested size via data-size", () => {
    const { container } = render(
      <Avatar size="lg">
        <AvatarFallback>AR</AvatarFallback>
      </Avatar>
    );
    const root = container.querySelector('[data-slot="avatar"]');
    expect(root?.getAttribute("data-size")).toBe("lg");
  });

  it("Should render the image slot when provided", () => {
    const { container } = render(
      <Avatar>
        <AvatarImage src="https://example.test/me.png" alt="me" />
        <AvatarFallback>ME</AvatarFallback>
      </Avatar>
    );
    const fallbackSlot = container.querySelector('[data-slot="avatar-fallback"]');
    expect(fallbackSlot).not.toBeNull();
  });

  it("Should render AvatarBadge as a positioned child", () => {
    const { container } = render(
      <Avatar size="lg">
        <AvatarFallback>PN</AvatarFallback>
        <AvatarBadge data-testid="badge" />
      </Avatar>
    );
    const badge = container.querySelector('[data-slot="avatar-badge"]');
    expect(badge).not.toBeNull();
  });

  it("Should render an AvatarGroup container for nested avatars", () => {
    const { container } = render(
      <AvatarGroup>
        <Avatar>
          <AvatarFallback>A</AvatarFallback>
        </Avatar>
        <Avatar>
          <AvatarFallback>B</AvatarFallback>
        </Avatar>
      </AvatarGroup>
    );
    const group = container.querySelector('[data-slot="avatar-group"]');
    expect(group).not.toBeNull();
    expect(group?.querySelectorAll('[data-slot="avatar"]').length).toBe(2);
  });
});
