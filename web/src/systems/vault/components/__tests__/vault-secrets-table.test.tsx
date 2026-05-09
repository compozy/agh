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
  it("Should render ready rows with namespace tones and ASCII fallbacks", () => {
    render(<VaultSecretsTable secrets={secrets} />);

    expect(screen.getByTestId("vault-secrets-table")).toHaveAttribute(
      "data-slot",
      "data-surface-content"
    );
    expect(screen.getByText("sessions")).toHaveAttribute("data-tone", "info");
    expect(screen.getByText("providers")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getAllByText("--").length).toBeGreaterThanOrEqual(2);
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
