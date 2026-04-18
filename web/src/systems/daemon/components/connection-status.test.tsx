import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { ConnectionStatus } from "./connection-status";

describe("ConnectionStatus", () => {
  it("renders green indicator when daemon is healthy (connected)", () => {
    render(<ConnectionStatus status="connected" />);
    expect(screen.getByText("Connected")).toBeInTheDocument();
  });

  it("renders red indicator when daemon is unreachable (disconnected)", () => {
    render(<ConnectionStatus status="disconnected" />);
    expect(screen.getByText("Disconnected")).toBeInTheDocument();
  });

  it("renders warning indicator when reconnecting", () => {
    render(<ConnectionStatus status="reconnecting" />);
    expect(screen.getByText("Reconnecting")).toBeInTheDocument();
  });
});
