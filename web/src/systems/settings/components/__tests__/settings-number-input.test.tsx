// Suite: Settings number input
// Invariant: Controlled updates preserve the active editing element and accept consecutive digits.
// Boundary IN: Number input state synchronization and parent-controlled value updates.
// Boundary OUT: Settings persistence and daemon-side validation, owned by settings adapter suites.
import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it } from "vitest";

import { SettingsNumberInput } from "../settings-number-input";

describe("SettingsNumberInput", () => {
  it("Should preserve focus across controlled value updates", () => {
    function Harness() {
      const [value, setValue] = useState(1);
      return (
        <SettingsNumberInput aria-label="Retry limit" onValueChange={setValue} value={value} />
      );
    }

    render(<Harness />);
    const input = screen.getByRole("textbox", { name: "Retry limit" });
    input.focus();
    fireEvent.change(input, { target: { value: "12" } });

    expect(input).toHaveValue("12");
    expect(input).toHaveFocus();
  });
});
