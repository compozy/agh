import { describe, expect, it, vi } from "vitest";

const redirectMock = vi.fn((payload: { to: string }) => payload);

vi.mock("@tanstack/react-router", () => ({
  createFileRoute: () => (opts: { beforeLoad?: () => void; component: () => unknown }) => ({
    beforeLoad: opts.beforeLoad,
    component: opts.component,
  }),
  redirect: (payload: { to: string }) => redirectMock(payload),
}));

import { Route } from "../index";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const beforeLoad = (Route as any).beforeLoad as () => void;

describe("SettingsIndexRedirect", () => {
  it("redirects the default settings route to the general section", () => {
    expect(() => beforeLoad()).toThrow();
    expect(redirectMock).toHaveBeenCalledWith({ to: "/settings/general" });
  });
});
