import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

vi.mock("react-syntax-highlighter", () => ({
  PrismAsyncLight: Object.assign(
    ({ children }: { children: string }) => <pre data-testid="syntax-highlighter">{children}</pre>,
    {
      registerLanguage: vi.fn(),
    }
  ),
}));

vi.mock("react-syntax-highlighter/dist/esm/styles/prism", () => ({
  oneDark: {},
}));

vi.mock("@/lib/utils", () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(" "),
}));

import { MessageMarkdown } from "../message-markdown";

describe("MessageMarkdown", () => {
  it("keeps the code copy button visible for keyboard focus styles", async () => {
    render(<MessageMarkdown content={"```typescript\nconst value = 1;\n```"} />);

    expect(await screen.findByTestId("syntax-highlighter")).toBeInTheDocument();

    const copyButton = await screen.findByRole("button", { name: "Copy code" });
    expect(copyButton.className).toContain("group-hover/codeblock:opacity-100");
    expect(copyButton.className).toContain("group-focus-within/codeblock:opacity-100");
    expect(copyButton.className).toContain("focus-visible:opacity-100");
  });
});
