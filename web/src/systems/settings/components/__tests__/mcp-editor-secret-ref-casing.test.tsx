import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { emptyDraft } from "../../lib/mcp-editor-model";
import { MCPEditorOAuthSection } from "../mcp-editor-oauth-section";
import { MCPEditorStdioSection } from "../mcp-editor-stdio-section";

/**
 * Regression for the MCP editor Vault-ref casing defect: a persisted,
 * case-sensitive `vault:` reference must render byte-for-byte at every editor
 * call site. Both rendered sites (stdio secret_env and OAuth client secret)
 * push the bound ref through the shared MonoId primitive, which previously
 * lowercased it. Each case asserts the exact primitive value slot
 * (`data-slot="mono-id-value"`) so a same-value <option> elsewhere in the
 * section cannot mask a regression.
 */
const REF = "vault:mcp/ws/ws-platform/github-local/env/QA_TYPED_TOKEN";

function monoIdValue(container: HTMLElement): string | null | undefined {
  return container.querySelector('[data-slot="mono-id-value"]')?.textContent;
}

describe("MCP editor Vault ref casing", () => {
  it("Should render the stdio secret_env Vault ref verbatim", () => {
    const { container } = render(
      <MCPEditorStdioSection
        draft={{
          ...emptyDraft("stdio"),
          secretEnv: [
            {
              key: "QA_TYPED_TOKEN",
              binding: { mode: "ref", typedValue: "", vaultRef: REF, existingRef: REF },
            },
          ],
        }}
        errors={{}}
        vaultRefs={[]}
        isCreate={false}
        target="config"
        availableTargets={["config"]}
        onChange={() => undefined}
        onTargetChange={() => undefined}
      />
    );

    expect(monoIdValue(container)).toBe(REF);
  });

  it("Should render the OAuth client secret Vault ref verbatim", () => {
    const { container } = render(
      <MCPEditorOAuthSection
        oauth={{
          ...emptyDraft("http").oauth,
          enabled: true,
          clientSecret: { mode: "ref", typedValue: "", vaultRef: REF, existingRef: REF },
        }}
        vaultRefs={[]}
        errors={{}}
        onChange={() => undefined}
      />
    );

    expect(monoIdValue(container)).toBe(REF);
  });
});

describe("MCP editor stdio secret binding errors", () => {
  it("Should surface a per-row binding error and mark the key input invalid", () => {
    const { getByTestId } = render(
      <MCPEditorStdioSection
        draft={{
          ...emptyDraft("stdio"),
          secretEnv: [{ key: "TOKEN", binding: { mode: "typed", typedValue: "", vaultRef: "" } }],
        }}
        errors={{ secretEnv: { 0: "Enter a value or select a Vault reference" } }}
        vaultRefs={[]}
        isCreate
        target="config"
        availableTargets={["config"]}
        onChange={() => undefined}
        onTargetChange={() => undefined}
      />
    );

    expect(getByTestId("settings-mcp-servers-editor-secret-error-0").textContent).toBe(
      "Enter a value or select a Vault reference"
    );
    expect(
      getByTestId("settings-mcp-servers-editor-secret-key-0").getAttribute("aria-invalid")
    ).toBe("true");
  });
});
