import "@testing-library/jest-dom";
import { beforeEach } from "vitest";

if (typeof window !== "undefined") {
  beforeEach(() => {
    window.sessionStorage.clear();
  });

  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {},
      removeListener: () => {},
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  });

  class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
  }

  window.ResizeObserver = ResizeObserverMock;
  window.scrollTo = () => {};
}
