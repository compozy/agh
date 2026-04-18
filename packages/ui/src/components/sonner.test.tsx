import { act, render, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Toaster, toast } from "./sonner";

describe("Toaster", () => {
  it("Should mount the Sonner root region", () => {
    render(<Toaster position="top-right" />);
    const region = document.querySelector("section[aria-label^='Notifications']");
    expect(region).not.toBeNull();
  });

  it("Should display a success toast body when toast.success is invoked", async () => {
    render(<Toaster position="top-right" duration={500} />);
    act(() => {
      toast.success("All good.");
    });
    await waitFor(() => {
      expect(document.body.textContent).toContain("All good.");
    });
  });

  it("Should display an error toast body when toast.error is invoked", async () => {
    render(<Toaster position="top-right" duration={500} />);
    act(() => {
      toast.error("Something broke.");
    });
    await waitFor(() => {
      expect(document.body.textContent).toContain("Something broke.");
    });
  });
});
