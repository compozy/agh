import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemSeparator,
  ItemSelectionIndicator,
  ItemTitle,
} from "../item";

describe("Item", () => {
  it("Should render a list container for ItemGroup", () => {
    const { container } = render(
      <ItemGroup>
        <Item>
          <ItemContent>
            <ItemTitle>One</ItemTitle>
          </ItemContent>
        </Item>
      </ItemGroup>
    );
    const group = container.querySelector('[data-slot="item-group"]');
    expect(group).not.toBeNull();
    expect(group?.getAttribute("role")).toBe("list");
  });

  it("Should compose media + content + actions slots in declared order", () => {
    const { container } = render(
      <Item>
        <ItemMedia variant="icon">icon</ItemMedia>
        <ItemContent>
          <ItemTitle>Title</ItemTitle>
          <ItemDescription>Description</ItemDescription>
        </ItemContent>
        <ItemActions>actions</ItemActions>
      </Item>
    );
    const item = container.querySelector('[data-slot="item"]');
    const slots = Array.from(item?.children ?? []).map(node => node.getAttribute("data-slot"));
    expect(slots).toEqual(["item-media", "item-content", "item-actions"]);
  });

  it("Should expose variant + size via state for useRender", () => {
    const { container } = render(
      <Item variant="outline" size="xs">
        <ItemContent>
          <ItemTitle>Slim</ItemTitle>
        </ItemContent>
      </Item>
    );
    const item = container.querySelector('[data-slot="item"]');
    expect(item?.getAttribute("data-variant")).toBe("outline");
    expect(item?.getAttribute("data-size")).toBe("xs");
  });

  it("Should expose selected state on the elevated surface step and render the 2px white rail indicator by default", () => {
    render(
      <Item selected indicator="rail" data-testid="selectable-item">
        <ItemContent>
          <ItemTitle>Selected row</ItemTitle>
        </ItemContent>
      </Item>
    );

    const item = screen.getByTestId("selectable-item");
    const indicator = item.querySelector('[data-slot="item-selection-indicator"]');
    expect(item.dataset.selected).toBe("true");
    expect(item.className).toContain("bg-(--elevated)");
    expect(item.className).toContain("text-(--fg-strong)");
    expect(indicator).not.toBeNull();
    expect(indicator?.getAttribute("data-indicator")).toBe("rail");
    expect(indicator?.getAttribute("data-tone")).toBe("white");
    expect(indicator?.className).toContain("w-[2px]");
    expect(indicator?.className).toContain("bg-(--fg-strong)");
  });

  it("Should render an accent rail when indicatorTone='accent'", () => {
    render(
      <Item indicator="rail" indicatorTone="accent" data-testid="unread-item">
        <ItemContent>
          <ItemTitle>Unread row</ItemTitle>
        </ItemContent>
      </Item>
    );

    const item = screen.getByTestId("unread-item");
    const indicator = item.querySelector('[data-slot="item-selection-indicator"]');
    expect(indicator?.getAttribute("data-tone")).toBe("accent");
    expect(indicator?.className).toContain("bg-(--accent)");
  });

  it("Should render as a pressed button when as=button and selected", () => {
    render(
      <Item as="button" selected data-testid="selectable-button">
        <ItemContent>
          <ItemTitle>Button row</ItemTitle>
        </ItemContent>
      </Item>
    );

    const button = screen.getByRole("button", { name: "Button row" });
    expect(button).toBe(screen.getByTestId("selectable-button"));
    expect(button).toHaveAttribute("aria-pressed", "true");
    expect(button).toHaveAttribute("type", "button");
  });

  it("Should preserve button-specific props when rendered as a button", () => {
    const onClick = vi.fn();

    render(
      <Item as="button" disabled onClick={onClick}>
        <ItemContent>
          <ItemTitle>Disabled button row</ItemTitle>
        </ItemContent>
      </Item>
    );

    const button = screen.getByRole("button", { name: "Disabled button row" });
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(onClick).not.toHaveBeenCalled();
  });

  it("Should render the dot indicator as a standalone subpart", () => {
    render(<ItemSelectionIndicator kind="dot" data-testid="item-dot-indicator" />);

    const indicator = screen.getByTestId("item-dot-indicator");
    expect(indicator.dataset.indicator).toBe("dot");
    expect(indicator.className).toContain("size-1.5");
  });

  it("Should render ItemSeparator as a horizontal separator between rows", () => {
    const { container } = render(
      <ItemGroup>
        <Item>
          <ItemContent>
            <ItemTitle>A</ItemTitle>
          </ItemContent>
        </Item>
        <ItemSeparator />
        <Item>
          <ItemContent>
            <ItemTitle>B</ItemTitle>
          </ItemContent>
        </Item>
      </ItemGroup>
    );
    const separator = container.querySelector('[data-slot="item-separator"]');
    expect(separator).not.toBeNull();
    expect(separator?.getAttribute("data-orientation")).toBe("horizontal");
  });
});
