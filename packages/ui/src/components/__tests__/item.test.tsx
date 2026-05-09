import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

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

  it("Should expose selected state and render the rail indicator", () => {
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
    expect(item.className).toContain("bg-[color:var(--color-surface)]");
    expect(indicator).not.toBeNull();
    expect(indicator?.getAttribute("data-indicator")).toBe("rail");
    expect(indicator?.className).toContain("w-[3px]");
    expect(indicator?.className).toContain("bg-[color:var(--color-accent)]");
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
