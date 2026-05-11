import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { VaultSecret } from "../../types";
import { VaultSecretsTable } from "../vault-secrets-table";

const secrets: VaultSecret[] = [
  {
    ref: "vault:sessions/session_01/api_key",
    namespace: "sessions",
    kind: "api_key",
    present: true,
    created_at: "2026-04-17T18:00:00Z",
    updated_at: "2026-04-17T18:14:00Z",
  },
  {
    ref: "vault:providers/codex/api_key",
    namespace: "providers",
    kind: "",
    present: true,
    created_at: "2026-04-17T18:00:00Z",
    updated_at: "not-a-date",
  },
];

describe("VaultSecretsTable", () => {
  it("Should render ready rows with namespace tones", () => {
    render(<VaultSecretsTable secrets={secrets} />);

    expect(screen.getByTestId("vault-secrets-table")).toHaveAttribute(
      "data-slot",
      "data-surface-content"
    );
    expect(screen.getByText("sessions")).toHaveAttribute("data-tone", "info");
    expect(screen.getByText("providers")).toHaveAttribute("data-tone", "neutral");
  });

  it("Should render the secret kind as a neutral pill (ADR-012 §8 — no tone until enum lands)", () => {
    render(<VaultSecretsTable secrets={secrets} />);
    const kindPill = screen.getByTestId(`vault-secrets-kind-${secrets[0].ref}`);
    expect(kindPill).toHaveAttribute("data-tone", "neutral");
    expect(kindPill).toHaveTextContent("api_key");
    // Empty kind falls back to "--" instead of a pill so absent kinds don't render colour.
    expect(screen.getByTestId(`vault-secrets-kind-empty-${secrets[1].ref}`)).toHaveTextContent(
      "--"
    );
  });

  it("Should render updated timestamps via <Time>", () => {
    render(<VaultSecretsTable secrets={secrets} />);
    const time = screen.getByTestId(`vault-secrets-updated-${secrets[0].ref}`);
    expect(time.tagName.toLowerCase()).toBe("time");
    expect(time.getAttribute("datetime")).toBe(secrets[0].updated_at);
  });

  it("Should route loading, error, and empty through DataSurface slots", () => {
    const { rerender } = render(<VaultSecretsTable secrets={[]} isLoading />);
    expect(screen.getByTestId("vault-secrets-table-loading")).toHaveAttribute(
      "data-slot",
      "block-loading"
    );

    rerender(<VaultSecretsTable secrets={[]} error={new Error("vault down")} />);
    expect(screen.getByTestId("vault-secrets-table-error")).toHaveTextContent("vault down");

    rerender(<VaultSecretsTable secrets={[]} emptyTitle="No provider secrets" />);
    expect(screen.getByTestId("vault-secrets-table-empty")).toHaveTextContent(
      "No provider secrets"
    );
  });

  it("Should call onDelete for the selected secret", () => {
    const onDelete = vi.fn();
    render(<VaultSecretsTable secrets={secrets} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId("vault-secrets-delete-vault:providers/codex/api_key"));
    expect(onDelete).toHaveBeenCalledWith(secrets[1]);
  });
});
