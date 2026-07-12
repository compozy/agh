import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { http, HttpResponse } from "msw";
import { createElement, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { createMswFetch } from "@/test/msw-fetch";
import { LoopEditor } from "../editor/loop-editor";
import type { LoopDetail } from "../../types";
import { handlers } from "../../mocks";
import { loopDetailByName } from "../../mocks/fixtures";

const deliveryDetail = loopDetailByName.get("software-delivery")!;

const WS = "ws_default";

function renderEditor(
  name = "software-delivery",
  extraHandlers: ReturnType<typeof http.post>[] = []
) {
  vi.stubGlobal(
    "fetch",
    createMswFetch(() => [...extraHandlers, ...handlers])
  );
  const onPublished = vi.fn<(loop: LoopDetail) => void>();
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children);
  render(<LoopEditor workspaceId={WS} name={name} onPublished={onPublished} />, { wrapper });
  return { onPublished };
}

function nodeCard(id: string): HTMLElement {
  const cards = screen.getAllByTestId("loop-editor-node");
  const card = cards.find(el => el.getAttribute("data-node-id") === id);
  if (!card) throw new Error(`node card ${id} not found`);
  return card;
}

describe("LoopEditor", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("E2E-web-12: edits a workspace Loop draft — palette add, inspector swap, invariant chips", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    // The canonical body renders on the canvas.
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));
    // The four canonical invariant chips are present.
    for (const key of ["acyclicity", "reachability", "termination", "fan_out"]) {
      expect(screen.getByTestId(`loop-invariant-${key}`)).toBeInTheDocument();
    }
    // No unsaved-changes chip until the definition is edited.
    expect(screen.queryByTestId("loop-editor-dirty-chip")).not.toBeInTheDocument();
    // Adding a node from the palette extends the body, selects it, and marks the draft dirty.
    fireEvent.click(screen.getByTestId("loop-palette-item-run-agent"));
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(9));
    expect(screen.getByTestId("loop-editor-dirty-chip")).toBeInTheDocument();
    expect(
      within(screen.getByTestId("loop-editor-inspector")).getByTestId("loop-inspector-name")
    ).toHaveTextContent("run_agent");
  });

  it("Should preserve both nodes when palette additions are dispatched in one event turn", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));

    fireEvent.click(screen.getByTestId("loop-palette-item-run-agent"));
    fireEvent.click(screen.getByTestId("loop-palette-item-run-agent"));

    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(10));
    expect(nodeCard("run_agent")).toBeInTheDocument();
    expect(nodeCard("run_agent_2")).toBeInTheDocument();
  });

  it("E2E-web-12: swaps the inspector field set when a different node is selected", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));
    fireEvent.click(nodeCard("implement"));
    await waitFor(() =>
      expect(
        within(screen.getByTestId("loop-editor-inspector")).getByTestId("loop-inspector-name")
      ).toHaveTextContent("implement")
    );
    // The fan-out node exposes the max_fan_out knob; a gate node does not.
    expect(screen.getByTestId("loop-field-max_fan_out")).toBeInTheDocument();
    fireEvent.click(nodeCard("review"));
    await waitFor(() =>
      expect(screen.queryByTestId("loop-field-max_fan_out")).not.toBeInTheDocument()
    );
    expect(screen.getByTestId("loop-editor-criteria")).toBeInTheDocument();
  });

  it("E2E-web-13: raising fan-out past the ceiling surfaces a 422 per-node error and disables Publish", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));
    // Publish is enabled on a clean workspace-sourced Loop.
    await waitFor(() => expect(screen.getByTestId("loop-editor-publish")).not.toBeDisabled());
    fireEvent.click(nodeCard("implement"));
    const fanOut = await screen.findByTestId("loop-field-max_fan_out");
    fireEvent.change(fanOut, { target: { value: "80" } });
    // The daemon linter (mock) returns fan_out_ceiling_exceeded → issue + node badge + gate.
    await waitFor(() =>
      expect(screen.getByTestId("loop-linter-issue")).toHaveTextContent(/ceiling of 64/i)
    );
    expect(nodeCard("implement")).toHaveAttribute("data-node-error", "true");
    expect(screen.getByTestId("loop-editor-publish")).toBeDisabled();
    expect(screen.getByTestId("loop-invariant-fan_out")).toHaveAttribute("data-status", "fail");
    // Lowering it back under the ceiling clears the issue and re-enables Publish.
    fireEvent.change(fanOut, { target: { value: "32" } });
    await waitFor(() => expect(screen.queryByTestId("loop-linter-issue")).not.toBeInTheDocument());
    expect(screen.getByTestId("loop-editor-publish")).not.toBeDisabled();
    expect(nodeCard("implement")).toHaveAttribute("data-node-error", "false");
  });

  it("E2E-web-13: maps a cycle 422 onto the offending node and reveals it from the dock", async () => {
    const cycleHandler = http.post("/api/workspaces/:workspaceId/loops/:name/validate", () =>
      HttpResponse.json(
        {
          valid: false,
          errors: [
            {
              node_id: "review",
              code: "cycle_detected",
              message: "review is part of a cycle.",
              severity: "error",
            },
          ],
        },
        { status: 422 }
      )
    );
    renderEditor("software-delivery", [cycleHandler]);
    await screen.findByTestId("loop-editor");
    fireEvent.click(screen.getByTestId("loop-editor-validate"));
    await waitFor(() => expect(nodeCard("review")).toHaveAttribute("data-node-error", "true"));
    expect(screen.getByTestId("loop-invariant-acyclicity")).toHaveAttribute("data-status", "fail");
    expect(screen.getByTestId("loop-editor-publish")).toBeDisabled();
    // Reveal node selects it in the inspector.
    fireEvent.click(screen.getByTestId("loop-linter-reveal"));
    await waitFor(() =>
      expect(
        within(screen.getByTestId("loop-editor-inspector")).getByTestId("loop-inspector-name")
      ).toHaveTextContent("review")
    );
  });

  it("E2E-web-13: maps a publish (PATCH) 422 per-node error back onto nodes, not just a banner", async () => {
    // A publish 422 must badge its own offending nodes (task-22 MUST), like validate does.
    const reject = http.patch("/api/workspaces/:workspaceId/loops/:name", () =>
      HttpResponse.json(
        {
          valid: false,
          errors: [
            {
              node_id: "verify",
              code: "verdict_policy_requires_judge",
              message: "verify: fixed_passes gate needs a judge or human criterion.",
              severity: "error",
            },
          ],
        },
        { status: 422 }
      )
    );
    renderEditor("software-delivery", [reject]);
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getByTestId("loop-editor-publish")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("loop-editor-publish"));
    // The rejected node is badged from the publish response itself.
    await waitFor(() => expect(nodeCard("verify")).toHaveAttribute("data-node-error", "true"));
    expect(screen.getByTestId("loop-editor-publish-error")).toHaveTextContent(
      /1 issue to resolve/i
    );
    expect(screen.getByTestId("loop-linter-issue")).toHaveTextContent(/needs a judge/i);
    // The publish verdict now gates further publishes (an unattributed code fails safe).
    expect(screen.getByTestId("loop-editor-publish")).toBeDisabled();
  });

  it("Should keep the id input focused while renaming a node character-by-character", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));
    fireEvent.click(nodeCard("collect"));
    const idInput = (await screen.findByTestId("loop-field-id")) as HTMLInputElement;
    idInput.focus();
    fireEvent.change(idInput, { target: { value: "collect2" } });
    // The node is renamed everywhere...
    await waitFor(() =>
      expect(
        screen
          .getAllByTestId("loop-editor-node")
          .some(el => el.getAttribute("data-node-id") === "collect2")
      ).toBe(true)
    );
    // ...and the id input did not remount (focus survives), so multi-char renames are usable.
    expect(screen.getByTestId("loop-field-id")).toHaveFocus();
  });

  it("Should show an 'unavailable' dock state (not a stale pass) when validate can't reach the daemon", async () => {
    const downValidate = http.post("/api/workspaces/:workspaceId/loops/:name/validate", () =>
      HttpResponse.json({ error: "linter offline" }, { status: 500 })
    );
    renderEditor("software-delivery", [downValidate]);
    await screen.findByTestId("loop-editor");
    // The mount auto-validate fails → the dock reports unavailable instead of "all pass".
    await waitFor(() => expect(screen.getByTestId("loop-linter-unavailable")).toBeInTheDocument());
    expect(screen.getByTestId("loop-linter-count")).toHaveTextContent("unavailable");
    // Chips are neutral/pending, never a claimed pass.
    expect(screen.getByTestId("loop-invariant-acyclicity")).toHaveAttribute(
      "data-status",
      "pending"
    );
  });

  it("E2E-web-14: Graph/DSL toggle renders agh.loop/v1 and highlights the offending field", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getAllByTestId("loop-editor-node")).toHaveLength(8));
    // Raise the fan-out to produce an offending field, then flip to the DSL view.
    fireEvent.click(nodeCard("implement"));
    fireEvent.change(await screen.findByTestId("loop-field-max_fan_out"), {
      target: { value: "80" },
    });
    await waitFor(() => expect(screen.getByTestId("loop-linter-issue")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("tab", { name: /DSL/i }));
    const dsl = await screen.findByTestId("loop-editor-dsl");
    expect(dsl).toHaveTextContent("agh.loop/v1");
    const offending = dsl.querySelector('[data-offending="true"]');
    expect(offending?.textContent).toMatch(/max_fan_out/);
  });

  it("E2E-web-14: renders the read-only Start summary chips from the definition start[]", async () => {
    renderEditor();
    await screen.findByTestId("loop-editor");
    const summary = await screen.findByTestId("loop-editor-start-summary");
    expect(summary).toHaveTextContent("manual");
    expect(summary).toHaveTextContent("schedule");
  });

  it("E2E-web-15/16: publishes the fork (expected_version), bumps the version, and continues to run", async () => {
    let patchBody: { expected_version?: number | null } | null = null;
    const capture = http.patch("/api/workspaces/:workspaceId/loops/:name", async ({ request }) => {
      patchBody = (await request.json()) as { expected_version?: number | null };
      return HttpResponse.json({
        loop: {
          ...deliveryDetail,
          version: 5,
          definition: {
            ...deliveryDetail.definition,
            meta: { name: "software-delivery", version: 5, catalog: {} },
          },
        },
      });
    });
    const { onPublished } = renderEditor("software-delivery", [capture]);
    await screen.findByTestId("loop-editor");
    await waitFor(() => expect(screen.getByTestId("loop-editor-version")).toHaveTextContent("v4"));
    fireEvent.click(screen.getByTestId("loop-palette-item-collect"));
    fireEvent.click(screen.getByTestId("loop-editor-publish"));
    await waitFor(() => expect(onPublished).toHaveBeenCalledTimes(1));
    expect(patchBody).toEqual(expect.objectContaining({ expected_version: 4 }));
    expect(onPublished.mock.calls[0][0].version).toBe(5);
    // The published version replaces the draft version and clears the dirty chip.
    await waitFor(() => expect(screen.getByTestId("loop-editor-version")).toHaveTextContent("v5"));
    expect(screen.queryByTestId("loop-editor-dirty-chip")).not.toBeInTheDocument();
  });
});
