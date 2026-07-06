import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { createMswFetch } from "@/test/msw-fetch";
import { LoopRunForm } from "../run-form/loop-run-form";
import { handlers } from "../../mocks";
import { loopDetailByName } from "../../mocks/fixtures";

const WS = "ws_default";
const loop = loopDetailByName.get("software-delivery")!;

function renderForm(onRunStarted = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  render(<LoopRunForm workspaceId={WS} loop={loop} onRunStarted={onRunStarted} />, { wrapper });
  return { onRunStarted };
}

describe("LoopRunForm", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      createMswFetch(() => handlers)
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("Should auto-generate a typed field per declared input with a type badge", () => {
    renderForm();
    const goal = screen.getByTestId("loop-run-field-goal");
    expect(goal).toHaveAttribute("data-input-type", "string");
    expect(goal).toHaveTextContent("string");
    const maxFiles = screen.getByTestId("loop-run-field-max_files");
    expect(maxFiles).toHaveAttribute("data-input-type", "number");
  });

  it("Should keep Run and Dry run disabled until the required input is filled", () => {
    renderForm();
    expect(screen.getByTestId("loop-run-submit-button")).toBeDisabled();
    fireEvent.change(screen.getByTestId("loop-run-field-input-goal"), {
      target: { value: "ship billing webhooks" },
    });
    expect(screen.getByTestId("loop-run-submit-button")).not.toBeDisabled();
    expect(screen.getByTestId("loop-run-dry-button")).not.toBeDisabled();
  });

  it("Should render NO cost-cap input anywhere in the form", () => {
    renderForm();
    expect(screen.queryByTestId("loop-run-override-cost")).not.toBeInTheDocument();
    expect(screen.getByTestId("loop-run-form")).not.toHaveTextContent(/cost cap/i);
  });

  it("Should render the gen-1 plan on Dry run without navigating", async () => {
    const { onRunStarted } = renderForm();
    fireEvent.change(screen.getByTestId("loop-run-field-input-goal"), {
      target: { value: "ship it" },
    });
    fireEvent.click(screen.getByTestId("loop-run-dry-button"));
    await waitFor(() => expect(screen.getByTestId("loop-run-plan")).toBeInTheDocument());
    expect(screen.getAllByTestId("loop-run-plan-node").length).toBeGreaterThan(0);
    expect(onRunStarted).not.toHaveBeenCalled();
  });

  it("Should start a run and hand the run id back for navigation", async () => {
    const { onRunStarted } = renderForm();
    fireEvent.change(screen.getByTestId("loop-run-field-input-goal"), {
      target: { value: "ship it" },
    });
    fireEvent.click(screen.getByTestId("loop-run-submit-button"));
    await waitFor(() => expect(onRunStarted).toHaveBeenCalledWith("looprun_running"));
  });
});
