import { describe, expect, it } from "vitest";

import {
  assertStorybookIndex,
  collectVisualTargets,
  StorybookIndexError,
  type StorybookIndex,
} from "../../src/testing/visual-story-index";

const fixture: StorybookIndex = {
  v: 5,
  entries: {
    "ui-button--default": {
      id: "ui-button--default",
      title: "ui/Button",
      name: "Default",
      type: "story",
      tags: ["dev", "test", "manifest", "autodocs"],
    },
    "ui-button--docs": {
      id: "ui-button--docs",
      title: "ui/Button",
      name: "Docs",
      type: "docs",
      tags: ["dev", "test", "manifest", "autodocs"],
    },
    "ui-dialog--opens-and-closes": {
      id: "ui-dialog--opens-and-closes",
      title: "ui/Dialog",
      name: "OpensAndCloses",
      type: "story",
      tags: ["dev", "test", "manifest", "autodocs", "play-fn"],
    },
    "ui-accordion--multiple-expansion": {
      id: "ui-accordion--multiple-expansion",
      title: "ui/Accordion",
      name: "MultipleExpansion",
      type: "story",
      tags: ["dev", "test", "manifest", "autodocs"],
    },
  },
};

describe("collectVisualTargets", () => {
  it("Should return one target per story entry and skip docs entries", () => {
    const targets = collectVisualTargets(fixture, "http://127.0.0.1:6007");
    const ids = targets.map(t => t.id);
    expect(ids).toEqual([
      "ui-accordion--multiple-expansion",
      "ui-button--default",
      "ui-dialog--opens-and-closes",
    ]);
    expect(ids).not.toContain("ui-button--docs");
  });

  it("Should build iframe.html story URLs with a pinned Storybook base", () => {
    const [accordion] = collectVisualTargets(fixture, "http://127.0.0.1:6007/");
    expect(accordion.storyUrl).toBe(
      "http://127.0.0.1:6007/iframe.html?id=ui-accordion--multiple-expansion&viewMode=story&globals=backgrounds%3A%21undefined"
    );
  });

  it("Should name snapshots by story id with a .png suffix (required by Playwright)", () => {
    const targets = collectVisualTargets(fixture, "http://127.0.0.1:6007");
    const button = targets.find(t => t.id === "ui-button--default");
    expect(button?.snapshotName).toBe("ui-button--default.png");
  });

  it("Should exclude stories whose tags intersect excludeTags", () => {
    const targets = collectVisualTargets(fixture, "http://127.0.0.1:6007", {
      excludeTags: ["play-fn"],
    });
    expect(targets.map(t => t.id)).toEqual([
      "ui-accordion--multiple-expansion",
      "ui-button--default",
    ]);
  });
});

describe("assertStorybookIndex", () => {
  it("Should throw StorybookIndexError when payload has no entries", () => {
    expect(() => assertStorybookIndex({})).toThrowError(StorybookIndexError);
  });

  it("Should accept a well-formed index object", () => {
    expect(() => assertStorybookIndex(fixture)).not.toThrow();
  });
});
