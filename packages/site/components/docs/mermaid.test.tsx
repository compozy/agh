import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

let initializeAttempts = 0;

const mermaidMock = {
  initialize: vi.fn(),
  render: vi.fn(),
};

vi.mock("mermaid", () => ({
  default: mermaidMock,
}));

import { Mermaid } from "./mermaid";

describe("Mermaid", () => {
  beforeEach(() => {
    initializeAttempts = 0;
    mermaidMock.initialize.mockReset();
    mermaidMock.initialize.mockImplementation(() => {
      initializeAttempts += 1;
      if (initializeAttempts === 1) {
        throw new Error("transient initialization failure");
      }
    });
    mermaidMock.render.mockReset();
    mermaidMock.render.mockImplementation(async (id: string) => ({
      svg: `<svg id="${id}"><title>diagram</title></svg>`,
    }));
  });

  it("retries loading after a transient initialization failure", async () => {
    const { rerender } = render(<Mermaid chart="graph TD; A-->B" />);

    await screen.findByText(
      "Mermaid could not render this diagram in the current browser session."
    );
    expect(mermaidMock.initialize).toHaveBeenCalledTimes(1);
    expect(mermaidMock.render).not.toHaveBeenCalled();

    rerender(<Mermaid chart="graph TD; B-->C" />);

    await waitFor(() => {
      expect(screen.getByLabelText("Mermaid diagram")).toBeTruthy();
    });
    expect(mermaidMock.initialize).toHaveBeenCalledTimes(2);
    expect(mermaidMock.initialize).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        securityLevel: "strict",
        theme: "base",
        themeVariables: expect.objectContaining({
          background: "#0E0E0F",
          primaryBorderColor: "#E8572A",
          primaryTextColor: "#E5E5E7",
          lineColor: "#8E8E93",
          actorBorder: "#E8572A",
          fontFamily: "Inter, ui-sans-serif, system-ui, sans-serif",
        }),
      })
    );
    expect(mermaidMock.render).toHaveBeenCalledWith(
      expect.stringContaining("agh-mermaid-"),
      "graph TD; B-->C"
    );
  });
});
