import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { ThinkingBlock } from "../thinking-block";

describe("ThinkingBlock", () => {
  it("shows active reasoning inline while the turn is still running", () => {
    render(
      <ThinkingBlock thinking="Checking tool output before answering." thinkingComplete={false} />
    );

    expect(screen.getByTestId("thinking-trigger")).toHaveTextContent("Thinking");
    expect(screen.getByTestId("thinking-content")).toHaveTextContent(
      "Checking tool output before answering."
    );
  });

  it("keeps completed reasoning collapsed but inspectable", async () => {
    const user = userEvent.setup();
    render(<ThinkingBlock thinking="Checked the output." thinkingComplete />);

    expect(screen.getByTestId("thinking-trigger")).toHaveTextContent("Thought process");
    expect(screen.queryByTestId("thinking-content")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("thinking-trigger"));

    expect(screen.getByTestId("thinking-content")).toHaveTextContent("Checked the output.");
  });
});
