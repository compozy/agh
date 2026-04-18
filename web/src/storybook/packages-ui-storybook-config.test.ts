import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const packagesUiMain = (await import("../../../packages/ui/.storybook/main")).default;
const packagesUiPreviewModule = await import("../../../packages/ui/.storybook/preview");
const packagesUiPreview = packagesUiPreviewModule.default;

const packagesUiPreviewPath = resolve(process.cwd(), "../packages/ui/.storybook/preview.ts");
const packagesUiPreviewSource = readFileSync(packagesUiPreviewPath, "utf8");

describe("packages/ui Storybook config", () => {
  it("scopes Storybook to packages/ui stories with the expected addons and framework", () => {
    expect(packagesUiMain.stories).toEqual(["../src/**/*.stories.@(ts|tsx)"]);
    expect(packagesUiMain.addons).toEqual([
      "@storybook/addon-docs",
      "@storybook/addon-a11y",
      "@storybook/addon-themes",
    ]);
    expect(packagesUiMain.framework).toEqual({
      name: "@storybook/react-vite",
      options: {},
    });
  });

  it("imports shared tokens without pulling in web styling or data-layer providers", () => {
    expect(packagesUiPreviewSource).toContain('import "@agh/ui/tokens.css";');
    expect(packagesUiPreviewSource).not.toContain("web/src/styles.css");
    expect(packagesUiPreviewSource).not.toContain("msw");
    expect(packagesUiPreviewSource).not.toContain("QueryClient");
    expect(packagesUiPreviewSource).not.toContain("createRouter");
    expect(packagesUiPreviewSource).not.toContain("RouterProvider");
  });

  it("stays render-only with no Storybook loaders", () => {
    expect(packagesUiPreview.loaders).toBeUndefined();
    expect(packagesUiPreview.decorators).toHaveLength(1);
    expect(packagesUiPreview.parameters).toEqual({
      backgrounds: {
        disable: true,
      },
      controls: {
        expanded: true,
      },
    });
  });
});
