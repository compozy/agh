import { describe, expect, it, vi } from "vitest";
import { siteConfig } from "./site-config";

const nextOg = vi.hoisted(() => ({
  ImageResponse: class MockImageResponse {
    readonly element: unknown;
    readonly init: unknown;

    constructor(element: unknown, init: unknown) {
      this.element = element;
      this.init = init;
    }
  },
}));

vi.mock("next/og", () => ({
  ImageResponse: nextOg.ImageResponse,
}));

type ElementLike = {
  props?: {
    children?: unknown;
    style?: Record<string, unknown>;
  };
};

function isElementLike(value: unknown): value is ElementLike {
  return typeof value === "object" && value !== null && "props" in value;
}

function textContent(value: unknown): string {
  if (typeof value === "string" || typeof value === "number") {
    return String(value);
  }
  if (Array.isArray(value)) {
    return value.map(textContent).join("");
  }
  if (isElementLike(value)) {
    return textContent(value.props?.children);
  }
  return "";
}

function styles(value: unknown): Array<Record<string, unknown>> {
  if (Array.isArray(value)) {
    return value.flatMap(styles);
  }
  if (!isElementLike(value)) {
    return [];
  }
  return [value.props?.style, ...styles(value.props?.children)].filter(
    (style): style is Record<string, unknown> => style !== undefined
  );
}

type MockedImageResponse = InstanceType<typeof nextOg.ImageResponse>;

function asMockImageResponse(value: unknown): MockedImageResponse {
  expect(value).toBeInstanceOf(nextOg.ImageResponse);
  return value as MockedImageResponse;
}

describe("OpenGraph image route", () => {
  it("publishes a static 1200x630 PNG response", async () => {
    const { contentType, default: Image, dynamic, size } = await import("@/app/opengraph-image");
    const response = asMockImageResponse(Image());

    expect(contentType).toBe("image/png");
    expect(dynamic).toBe("force-static");
    expect(size).toEqual({ width: 1200, height: 630 });
    expect(response.init).toEqual(size);
  });

  it("keeps public copy and design tokens aligned with AGH", async () => {
    const { default: Image } = await import("@/app/opengraph-image");
    const response = asMockImageResponse(Image());
    const copy = textContent(response.element);
    const styleValues = styles(response.element).flatMap(style =>
      Object.values(style).filter((value): value is string => typeof value === "string")
    );

    expect(copy).toContain("AGH");
    expect(copy).toContain("Artificial General Hivemind");
    expect(copy).toContain("An open workplace for AI agents.");
    expect(copy).toContain(siteConfig.description);
    expect(styleValues).toContain("#141312");
    expect(styleValues).toContain("#1E1C1B");
    expect(styleValues).toContain("#3C3A39");
    expect(styleValues).toContain("#E5E5E7");
    expect(styleValues).toContain("#8E8E93");
    expect(styleValues).toContain("#E8572A");
    expect(styleValues).not.toContain("#E8572B");
  });
});
