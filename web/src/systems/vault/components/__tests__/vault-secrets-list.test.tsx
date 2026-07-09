import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { VaultSecret } from "../../types";
import { VaultSecretsList } from "../vault-secrets-list";

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

describe("VaultSecretsList", () => {
  it("Should render ready rows with namespace tones", () => {
    render(<VaultSecretsList secrets={secrets} />);

    expect(screen.getByTestId("vault-secrets-list")).toHaveAttribute(
      "data-slot",
      "data-surface-content"
    );
    expect(screen.getByText("sessions")).toHaveAttribute("data-tone", "info");
    expect(screen.getByText("providers")).toHaveAttribute("data-tone", "neutral");
    expect(screen.getAllByTestId("vault-secrets-row")).toHaveLength(2);
  });

  it("Should render the secret kind as a neutral pill (— no tone until enum lands)", () => {
    render(<VaultSecretsList secrets={secrets} />);
    const kindPill = screen.getByTestId(`vault-secrets-kind-${secrets[0].ref}`);
    expect(kindPill).toHaveAttribute("data-tone", "neutral");
    expect(kindPill).toHaveTextContent("api_key");
    // Empty kind falls back to "--" instead of a pill so absent kinds don't render colour.
    expect(screen.getByTestId(`vault-secrets-kind-empty-${secrets[1].ref}`)).toHaveTextContent(
      "--"
    );
  });

  it("Should render updated timestamps via <Time>", () => {
    render(<VaultSecretsList secrets={secrets} />);
    const time = screen.getByTestId(`vault-secrets-updated-${secrets[0].ref}`);
    expect(time.tagName.toLowerCase()).toBe("time");
    expect(time.getAttribute("datetime")).toBe(secrets[0].updated_at);
  });

  it("Should route loading, error, and empty through DataSurface slots", () => {
    const { rerender } = render(<VaultSecretsList secrets={[]} isLoading />);
    expect(screen.getByTestId("vault-secrets-list-loading")).toHaveAttribute(
      "data-slot",
      "block-loading"
    );

    rerender(<VaultSecretsList secrets={[]} error={new Error("vault down")} />);
    expect(screen.getByTestId("vault-secrets-list-error")).toHaveTextContent("vault down");

    rerender(<VaultSecretsList secrets={[]} emptyTitle="No provider secrets" />);
    expect(screen.getByTestId("vault-secrets-list-empty")).toHaveTextContent("No provider secrets");
  });

  it("Should call onDelete for the selected secret", () => {
    const onDelete = vi.fn();
    render(<VaultSecretsList secrets={secrets} onDelete={onDelete} />);

    fireEvent.click(screen.getByTestId("vault-secrets-delete-vault:providers/codex/api_key"));
    expect(onDelete).toHaveBeenCalledWith(secrets[1]);
  });
});
