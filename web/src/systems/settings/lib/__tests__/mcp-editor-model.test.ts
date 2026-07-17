import { describe, expect, it } from "vitest";

import type { SettingsMCPServerEntry } from "../../types";
import {
  emptyDraft,
  type MCPDraft,
  toDraft,
  toRequest,
  validateDraft,
  withTransport,
} from "../mcp-editor-model";

function stdioEntry(): SettingsMCPServerEntry {
  return {
    name: "github-local",
    transport: "stdio",
    command: "npx",
    args: ["-y", "@modelcontextprotocol/server-github"],
    env: { LOG_LEVEL: "info" },
    secret_env: { GITHUB_TOKEN: "vault:mcp/ws/ws-platform/github-local/env/github_token" },
    scope: "workspace",
    source_metadata: {},
  } as unknown as SettingsMCPServerEntry;
}

function remoteEntry(): SettingsMCPServerEntry {
  return {
    name: "linear",
    transport: "http",
    url: "https://mcp.linear.app/mcp",
    auth: {
      type: "oauth2_pkce",
      client_id: "agh-linear-public",
      issuer_url: "https://auth.linear.app",
      scopes: ["read", "write"],
      client_secret_ref: "vault:mcp/ws/ws-platform/linear/oauth/client-secret",
    },
    scope: "workspace",
    source_metadata: {},
  } as unknown as SettingsMCPServerEntry;
}

describe("emptyDraft", () => {
  it("defaults to stdio and carries an empty oauth block", () => {
    const draft = emptyDraft();
    expect(draft.transport).toBe("stdio");
    expect(draft.oauth.enabled).toBe(false);
    expect(draft.args).toEqual([]);
  });

  it("honors an explicit transport", () => {
    expect(emptyDraft("sse").transport).toBe("sse");
  });
});

describe("withTransport", () => {
  it("clears stdio-only fields switching stdio → remote so the form stays completable", () => {
    const stdio: MCPDraft = {
      ...emptyDraft("stdio"),
      name: "srv",
      command: "npx",
      args: ["-y", "server"],
      env: [{ key: "LOG_LEVEL", value: "info" }],
      secretEnv: [{ key: "TOKEN", binding: { mode: "typed", typedValue: "x", vaultRef: "" } }],
    };
    const remote = withTransport(stdio, "http");
    expect(remote.transport).toBe("http");
    expect(remote.command).toBe("");
    expect(remote.args).toEqual([]);
    expect(remote.env).toEqual([]);
    expect(remote.secretEnv).toEqual([]);
    expect(remote.name).toBe("srv");
    // No leaked stdio field deadlocks Save; only the url remains to be filled.
    const validated = validateDraft({ ...remote, url: "https://mcp.example/mcp" });
    expect(validated.valid).toBe(true);
    expect(validated.errors.remoteFields).toBeUndefined();
  });

  it("clears remote-only fields switching remote → stdio", () => {
    const remote: MCPDraft = {
      ...emptyDraft("http"),
      name: "srv",
      url: "https://mcp.example/mcp",
      oauth: { ...emptyDraft("http").oauth, enabled: true, clientId: "agh" },
    };
    const stdio = withTransport(remote, "stdio");
    expect(stdio.transport).toBe("stdio");
    expect(stdio.url).toBe("");
    expect(stdio.oauth.enabled).toBe(false);
    expect(stdio.oauth.clientId).toBe("");
  });

  it("preserves shared url + oauth switching within the remote family (http ↔ sse)", () => {
    const http: MCPDraft = {
      ...emptyDraft("http"),
      name: "srv",
      url: "https://mcp.example/mcp",
      oauth: {
        ...emptyDraft("http").oauth,
        enabled: true,
        clientId: "agh",
        discovery: "issuer",
        issuerUrl: "https://auth",
      },
    };
    const sse = withTransport(http, "sse");
    expect(sse.transport).toBe("sse");
    expect(sse.url).toBe("https://mcp.example/mcp");
    expect(sse.oauth.enabled).toBe(true);
    expect(sse.oauth.clientId).toBe("agh");
  });

  it("returns the same draft reference when the transport is unchanged", () => {
    const draft: MCPDraft = { ...emptyDraft("stdio"), command: "npx" };
    expect(withTransport(draft, "stdio")).toBe(draft);
  });
});

describe("toDraft", () => {
  it("reads stdio command/args/env and renders existing secret refs as bindings", () => {
    const draft = toDraft(stdioEntry());
    expect(draft.transport).toBe("stdio");
    expect(draft.command).toBe("npx");
    expect(draft.args).toEqual(["-y", "@modelcontextprotocol/server-github"]);
    expect(draft.env).toEqual([{ key: "LOG_LEVEL", value: "info" }]);
    expect(draft.secretEnv).toHaveLength(1);
    const binding = draft.secretEnv[0].binding;
    expect(binding.mode).toBe("ref");
    expect(binding.existingRef).toBe("vault:mcp/ws/ws-platform/github-local/env/github_token");
    // Plaintext is never reflected.
    expect(binding.typedValue).toBe("");
  });

  it("reads remote url and infers the oauth discovery form from the auth block", () => {
    const draft = toDraft(remoteEntry());
    expect(draft.transport).toBe("http");
    expect(draft.url).toBe("https://mcp.linear.app/mcp");
    expect(draft.oauth.enabled).toBe(true);
    expect(draft.oauth.clientId).toBe("agh-linear-public");
    expect(draft.oauth.discovery).toBe("issuer");
    expect(draft.oauth.scopes).toBe("read write");
    expect(draft.oauth.clientSecret.existingRef).toBe(
      "vault:mcp/ws/ws-platform/linear/oauth/client-secret"
    );
  });
});

describe("toRequest stdio", () => {
  it("keeps existing secret refs on the server and sends typed values via secret_values", () => {
    const draft = toDraft(stdioEntry());
    // Rotate one secret to a typed write-only value.
    draft.secretEnv[0].binding = { mode: "typed", typedValue: "ghp_new", vaultRef: "" };
    const request = toRequest(draft);
    expect(request.server.transport).toBe("stdio");
    expect(request.server.command).toBe("npx");
    expect(request.server.args).toEqual(["-y", "@modelcontextprotocol/server-github"]);
    expect(request.server.env).toEqual({ LOG_LEVEL: "info" });
    // The rotated secret is not on the server object.
    expect(request.server.secret_env).toBeUndefined();
    expect(request.secret_values).toEqual({ secret_env: { GITHUB_TOKEN: "ghp_new" } });
    // Never emits url/auth for stdio.
    expect(request.server.url).toBeUndefined();
    expect(request.server.auth).toBeUndefined();
  });

  it("preserves an unchanged existing ref without echoing a value", () => {
    const request = toRequest(toDraft(stdioEntry()));
    expect(request.server.secret_env).toEqual({
      GITHUB_TOKEN: "vault:mcp/ws/ws-platform/github-local/env/github_token",
    });
    expect(request.secret_values).toBeUndefined();
  });
});

describe("toRequest remote", () => {
  it("emits url + oauth and never command/args/env/secret_env", () => {
    const request = toRequest(toDraft(remoteEntry()));
    expect(request.server.transport).toBe("http");
    expect(request.server.url).toBe("https://mcp.linear.app/mcp");
    expect(request.server.command).toBeUndefined();
    expect(request.server.args).toBeUndefined();
    expect(request.server.env).toBeUndefined();
    expect(request.server.secret_env).toBeUndefined();
    expect(request.server.auth?.type).toBe("oauth2_pkce");
    expect(request.server.auth?.issuer_url).toBe("https://auth.linear.app");
    expect(request.server.auth?.scopes).toEqual(["read", "write"]);
    // Existing client secret ref is preserved, not echoed.
    expect(request.server.auth?.client_secret_ref).toBe(
      "vault:mcp/ws/ws-platform/linear/oauth/client-secret"
    );
  });

  it("sends a rotated client secret as a write-only value", () => {
    const draft = toDraft(remoteEntry());
    draft.oauth.clientSecret = { mode: "typed", typedValue: "new-secret", vaultRef: "" };
    const request = toRequest(draft);
    expect(request.server.auth?.client_secret_ref).toBeUndefined();
    expect(request.secret_values).toEqual({ oauth_client_secret: "new-secret" });
  });

  it("omits the auth block entirely when oauth is disabled", () => {
    const draft = toDraft(remoteEntry());
    draft.oauth.enabled = false;
    const request = toRequest(draft);
    expect(request.server.auth).toBeUndefined();
    expect(request.secret_values).toBeUndefined();
  });
});

describe("validateDraft", () => {
  function base(): MCPDraft {
    return emptyDraft("stdio");
  }

  it("requires a name", () => {
    const result = validateDraft({ ...base(), name: "  " });
    expect(result.valid).toBe(false);
    expect(result.errors.name).toBe("Name is required");
  });

  it("requires a command for stdio", () => {
    const result = validateDraft({ ...base(), name: "srv", command: "" });
    expect(result.errors.command).toBe("Command is required");
  });

  it("requires a url for remote transports", () => {
    const draft = emptyDraft("http");
    const result = validateDraft({ ...draft, name: "srv", url: "" });
    expect(result.errors.url).toBe("URL is required for http transport");
  });

  it("blocks command and secret_env on a remote draft with the daemon message", () => {
    const draft = emptyDraft("sse");
    const withCommand = validateDraft({ ...draft, name: "srv", url: "https://x", command: "npx" });
    expect(withCommand.errors.remoteFields).toBe("Command is only valid for stdio transport");
    const withSecret = validateDraft({
      ...draft,
      name: "srv",
      url: "https://x",
      secretEnv: [{ key: "TOKEN", binding: { mode: "typed", typedValue: "x", vaultRef: "" } }],
    });
    expect(withSecret.errors.remoteFields).toBe("Secret env is only valid for stdio transport");
  });

  it("requires client_id and a metadata form when oauth is enabled", () => {
    const draft = emptyDraft("http");
    const missingClient = validateDraft({
      ...draft,
      name: "srv",
      url: "https://x",
      oauth: { ...draft.oauth, enabled: true, discovery: "issuer", issuerUrl: "https://auth" },
    });
    expect(missingClient.errors.clientId).toBe("Client ID is required for OAuth");

    const missingMetadata = validateDraft({
      ...draft,
      name: "srv",
      url: "https://x",
      oauth: { ...draft.oauth, enabled: true, clientId: "agh", discovery: "metadata" },
    });
    expect(missingMetadata.errors.metadata).toBe("Metadata URL is required");

    const endpointsMissing = validateDraft({
      ...draft,
      name: "srv",
      url: "https://x",
      oauth: { ...draft.oauth, enabled: true, clientId: "agh", discovery: "endpoints" },
    });
    expect(endpointsMissing.errors.metadata).toBe("Authorization URL and token URL are required");
  });

  it("flags a non-empty stdio secret key whose binding resolves to nothing", () => {
    const result = validateDraft({
      ...base(),
      name: "srv",
      command: "npx",
      secretEnv: [{ key: "TOKEN", binding: { mode: "typed", typedValue: "", vaultRef: "" } }],
    });
    expect(result.valid).toBe(false);
    expect(result.errors.secretEnv?.[0]).toBe("Enter a value or select a Vault reference");
  });

  it("accepts a stdio secret row completed by a typed value, a vault ref, or a preserved ref", () => {
    const typed = validateDraft({
      ...base(),
      name: "srv",
      command: "npx",
      secretEnv: [{ key: "TOKEN", binding: { mode: "typed", typedValue: "ghp_x", vaultRef: "" } }],
    });
    expect(typed.valid).toBe(true);
    expect(typed.errors.secretEnv).toBeUndefined();

    const ref = validateDraft({
      ...base(),
      name: "srv",
      command: "npx",
      secretEnv: [
        {
          key: "TOKEN",
          binding: { mode: "ref", typedValue: "", vaultRef: "vault:mcp/ws/x/env/t" },
        },
      ],
    });
    expect(ref.valid).toBe(true);

    const preserved = validateDraft({
      ...base(),
      name: "srv",
      command: "npx",
      secretEnv: [
        {
          key: "TOKEN",
          binding: {
            mode: "typed",
            typedValue: "",
            vaultRef: "",
            existingRef: "vault:mcp/ws/x/env/t",
          },
        },
      ],
    });
    expect(preserved.valid).toBe(true);
  });

  it("ignores a pristine blank-key secret row", () => {
    const result = validateDraft({
      ...base(),
      name: "srv",
      command: "npx",
      secretEnv: [{ key: "  ", binding: { mode: "typed", typedValue: "", vaultRef: "" } }],
    });
    expect(result.valid).toBe(true);
    expect(result.errors.secretEnv).toBeUndefined();
  });

  it("accepts a valid stdio draft and a valid remote oauth draft", () => {
    expect(validateDraft({ ...base(), name: "srv", command: "npx" }).valid).toBe(true);
    const remote = emptyDraft("http");
    expect(
      validateDraft({
        ...remote,
        name: "srv",
        url: "https://mcp.linear.app/mcp",
        oauth: {
          ...remote.oauth,
          enabled: true,
          clientId: "agh",
          discovery: "issuer",
          issuerUrl: "https://auth",
        },
      }).valid
    ).toBe(true);
  });
});
