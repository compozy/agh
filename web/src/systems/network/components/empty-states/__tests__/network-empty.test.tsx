import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { NetworkEmpty } from "../network-empty";

describe("NetworkEmpty", () => {
  it("Should answer orientation questions with one settings action", () => {
    render(<NetworkEmpty onOpenSettings={() => undefined} />);
    expect(screen.getByTestId("network-empty")).toBeInTheDocument();
    expect(screen.getByText(/Network area/i)).toBeInTheDocument();
    expect(screen.getByTestId("network-empty-open-settings")).toBeInTheDocument();
  });

  it("Should name who can change availability when disabled", () => {
    render(<NetworkEmpty disabledByAdmin onOpenSettings={() => undefined} />);
    expect(screen.getByText(/operator with admin access/i)).toBeInTheDocument();
  });
});
