import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemGroup,
  ItemMedia,
  ItemSeparator,
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
