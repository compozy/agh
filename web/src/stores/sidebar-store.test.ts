import { beforeEach, describe, expect, it } from "vitest";

import { useSidebarStore } from "./sidebar-store";

describe("sidebar-store", () => {
  beforeEach(() => {
    useSidebarStore.setState({ collapsed: false });
  });

  describe("initial state", () => {
    it("starts with collapsed false", () => {
      expect(useSidebarStore.getState().collapsed).toBe(false);
    });
  });

  describe("toggle", () => {
    it("toggles collapsed from false to true", () => {
      useSidebarStore.getState().toggle();
      expect(useSidebarStore.getState().collapsed).toBe(true);
    });

    it("toggles collapsed from true to false", () => {
      useSidebarStore.setState({ collapsed: true });
      useSidebarStore.getState().toggle();
      expect(useSidebarStore.getState().collapsed).toBe(false);
    });

    it("round-trips correctly", () => {
      useSidebarStore.getState().toggle();
      useSidebarStore.getState().toggle();
      expect(useSidebarStore.getState().collapsed).toBe(false);
    });
  });

  describe("setCollapsed", () => {
    it("sets collapsed to true", () => {
      useSidebarStore.getState().setCollapsed(true);
      expect(useSidebarStore.getState().collapsed).toBe(true);
    });

    it("sets collapsed to false", () => {
      useSidebarStore.setState({ collapsed: true });
      useSidebarStore.getState().setCollapsed(false);
      expect(useSidebarStore.getState().collapsed).toBe(false);
    });

    it("is idempotent", () => {
      useSidebarStore.getState().setCollapsed(true);
      useSidebarStore.getState().setCollapsed(true);
      expect(useSidebarStore.getState().collapsed).toBe(true);
    });
  });
});
