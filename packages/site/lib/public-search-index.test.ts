import { describe, expect, it, vi } from "vitest";

const searchApi = vi.hoisted(() => ({
  calls: [] as Array<{
    mode: string;
    options: {
      indexes: Array<{
        title: string;
        description: string;
        structuredData: unknown;
        id: string;
        url: string;
        tag: string;
      }>;
    };
  }>,
}));

const mockedDocs = vi.hoisted(() => ({
  protocolPages: [
    {
      url: "/protocol/implementation-status",
      data: {
        title: "Implementation Status",
        description: "Understand what agh-network/v0 implements in the alpha runtime.",
        structuredData: { headings: [{ content: "Current runtime" }] },
      },
    },
  ],
  runtimePages: [
    {
      url: "/runtime/use-cases/prepare-a-project-workspace",
      data: {
        title: "Prepare a Project Workspace",
        description: "Register a project workspace and run a first smoke-test session.",
        structuredData: { headings: [{ content: "Setup" }] },
      },
    },
    {
      url: "/runtime/how-to-use-these-docs",
      data: {
        title: "How to Use These Docs",
        description: "Choose the right AGH documentation path for your goal.",
        structuredData: { headings: [{ content: "Choose a path" }] },
      },
    },
  ],
}));

vi.mock("@/lib/source", () => ({
  protocolDocs: {
    getPages: () => mockedDocs.protocolPages,
  },
  runtimeDocs: {
    getPages: () => mockedDocs.runtimePages,
  },
}));

vi.mock("fumadocs-core/search/server", () => ({
  createSearchAPI: (
    mode: string,
    options: { indexes: (typeof searchApi.calls)[number]["options"]["indexes"] }
  ) => {
    searchApi.calls.push({ mode, options });
    return {
      staticGET: () => Response.json({ indexes: options.indexes }),
    };
  },
}));

describe("public search index", () => {
  it("indexes runtime and protocol docs with stable sorted route metadata", async () => {
    const route = await import("@/app/api/search/route");

    expect(route.revalidate).toBe(false);
    expect(searchApi.calls).toHaveLength(1);
    expect(searchApi.calls[0]?.mode).toBe("advanced");

    const indexes = searchApi.calls[0]?.options.indexes ?? [];
    expect(indexes).toEqual([
      {
        title: "How to Use These Docs",
        description: "Choose the right AGH documentation path for your goal.",
        structuredData: { headings: [{ content: "Choose a path" }] },
        id: "/runtime/how-to-use-these-docs",
        url: "/runtime/how-to-use-these-docs",
        tag: "Runtime",
      },
      {
        title: "Prepare a Project Workspace",
        description: "Register a project workspace and run a first smoke-test session.",
        structuredData: { headings: [{ content: "Setup" }] },
        id: "/runtime/use-cases/prepare-a-project-workspace",
        url: "/runtime/use-cases/prepare-a-project-workspace",
        tag: "Runtime",
      },
      {
        title: "Implementation Status",
        description: "Understand what agh-network/v0 implements in the alpha runtime.",
        structuredData: { headings: [{ content: "Current runtime" }] },
        id: "/protocol/implementation-status",
        url: "/protocol/implementation-status",
        tag: "AGH Network",
      },
    ]);
    expect(indexes.map(index => index.id)).toEqual(indexes.map(index => index.url));

    const response = await route.GET();
    await expect(response.json()).resolves.toEqual({ indexes });
  });
});
