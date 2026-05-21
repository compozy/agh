import { http, HttpResponse, type HttpHandler } from "msw";

import {
  settingsAppliedMutationFixture,
  settingsApplyRecordsFixture,
  settingsAutomationSectionFixture,
  settingsSandboxesCollectionFixture,
  settingsSandboxFixtures,
  settingsExtensionsCollectionFixture,
  settingsExtensionMarketplaceCollectionFixture,
  settingsExtensionFixtures,
  settingsGeneralSectionFixture,
  settingsHooksCollectionFixture,
  settingsHooksExtensionsSectionFixture,
  settingsMCPServersCollectionFixture,
  settingsMemorySectionFixture,
  settingsNetworkSectionFixture,
  settingsObservabilitySectionFixture,
  settingsProvidersCollectionFixture,
  settingsProviderFixtures,
  settingsReloadBlockedFixture,
  settingsRestartRequiredMutationFixture,
  settingsRestartResponseFixture,
  settingsRestartStatusFixture,
  settingsSkillsSectionFixture,
} from "./fixtures";

function mutationResult(section: string, restartRequired = false) {
  return {
    ...(restartRequired ? settingsRestartRequiredMutationFixture : settingsAppliedMutationFixture),
    section,
  };
}

function applyRecordsForUrl(request: Request) {
  const url = new URL(request.url);
  const status = url.searchParams.get("status");
  const actor = url.searchParams.get("actor");
  const limit = Number.parseInt(url.searchParams.get("limit") ?? "", 10);
  let entries = settingsApplyRecordsFixture.entries;

  if (status) {
    entries = entries.filter(record => record.status === status);
  }

  if (actor) {
    entries = entries.filter(record => record.actor === actor);
  }

  if (Number.isFinite(limit) && limit >= 0) {
    entries = entries.slice(0, limit);
  }

  return { entries };
}

export const handlers: HttpHandler[] = [
  http.get("/api/settings/general", () => HttpResponse.json(settingsGeneralSectionFixture)),
  http.patch("/api/settings/general", () => HttpResponse.json(mutationResult("general", true))),

  http.get("/api/settings/memory", () => HttpResponse.json(settingsMemorySectionFixture)),
  http.patch("/api/settings/memory", () => HttpResponse.json(mutationResult("memory"))),

  http.get("/api/settings/skills", () => HttpResponse.json(settingsSkillsSectionFixture)),
  http.patch("/api/settings/skills", () => HttpResponse.json(mutationResult("skills", true))),

  http.get("/api/settings/automation", () => HttpResponse.json(settingsAutomationSectionFixture)),
  http.patch("/api/settings/automation", () =>
    HttpResponse.json(mutationResult("automation", true))
  ),

  http.get("/api/settings/network", () => HttpResponse.json(settingsNetworkSectionFixture)),
  http.patch("/api/settings/network", () => HttpResponse.json(mutationResult("network", true))),

  http.get("/api/settings/observability", () =>
    HttpResponse.json(settingsObservabilitySectionFixture)
  ),
  http.patch("/api/settings/observability", () =>
    HttpResponse.json(mutationResult("observability"))
  ),

  http.get("/api/settings/hooks-extensions", () =>
    HttpResponse.json(settingsHooksExtensionsSectionFixture)
  ),
  http.patch("/api/settings/hooks-extensions", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),

  http.get("/api/settings/providers", () => HttpResponse.json(settingsProvidersCollectionFixture)),
  http.get("/api/settings/providers/:name", ({ params }) => {
    const name = String(params.name);
    const provider = settingsProviderFixtures.find(entry => entry.name === name);

    if (!provider) {
      return HttpResponse.json({ error: `Provider not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ provider });
  }),
  http.put("/api/settings/providers/:name", () =>
    HttpResponse.json(mutationResult("providers", true))
  ),
  http.delete("/api/settings/providers/:name", () =>
    HttpResponse.json(mutationResult("providers", true))
  ),

  http.get("/api/settings/sandboxes", () => HttpResponse.json(settingsSandboxesCollectionFixture)),
  http.get("/api/settings/sandboxes/:name", ({ params }) => {
    const name = String(params.name);
    const sandbox = settingsSandboxFixtures.find(entry => entry.name === name);

    if (!sandbox) {
      return HttpResponse.json({ error: `Sandbox not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ sandbox });
  }),
  http.put("/api/settings/sandboxes/:name", () =>
    HttpResponse.json(mutationResult("sandboxes", true))
  ),
  http.delete("/api/settings/sandboxes/:name", () =>
    HttpResponse.json(mutationResult("sandboxes", true))
  ),

  http.get("/api/settings/hooks", () => HttpResponse.json(settingsHooksCollectionFixture)),
  http.put("/api/settings/hooks/:name", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),
  http.delete("/api/settings/hooks/:name", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),

  http.get("/api/settings/mcp-servers", ({ request }) => {
    const url = new URL(request.url);
    const scope = url.searchParams.get("scope");
    const workspaceId = url.searchParams.get("workspace_id");

    if (scope === "workspace" && workspaceId) {
      return HttpResponse.json({ mcp_servers: [] });
    }

    return HttpResponse.json(settingsMCPServersCollectionFixture);
  }),
  http.put("/api/settings/mcp-servers/:name", () =>
    HttpResponse.json(mutationResult("mcp-servers", true))
  ),
  http.delete("/api/settings/mcp-servers/:name", () =>
    HttpResponse.json(mutationResult("mcp-servers", true))
  ),

  http.get("/api/settings/apply", ({ request }) => HttpResponse.json(applyRecordsForUrl(request))),
  http.post("/api/settings/reload", () => HttpResponse.json(settingsReloadBlockedFixture)),

  http.post("/api/settings/actions/restart", () =>
    HttpResponse.json(settingsRestartResponseFixture, { status: 202 })
  ),
  http.get("/api/settings/actions/restart/:operation_id", () =>
    HttpResponse.json(settingsRestartStatusFixture)
  ),

  http.get("/api/extensions", () => HttpResponse.json(settingsExtensionsCollectionFixture)),
  http.get("/api/extensions/marketplace", () =>
    HttpResponse.json(settingsExtensionMarketplaceCollectionFixture)
  ),
  http.post("/api/extensions", async ({ request }) => {
    const body = (await request.json()) as { slug?: string };
    const marketplaceEntry = settingsExtensionMarketplaceCollectionFixture.extensions.find(
      entry => entry.slug === body.slug
    );
    const installed = marketplaceEntry
      ? {
          ...settingsExtensionFixtures[0],
          name: marketplaceEntry.name,
          version: marketplaceEntry.version,
        }
      : settingsExtensionFixtures[0];

    return HttpResponse.json({ extension: installed }, { status: 201 });
  }),
  http.put("/api/extensions/:name", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({
      update: {
        name,
        slug: extension.provenance?.slug ?? name,
        registry: "github",
        path: `/tmp/agh/extensions/${name}`,
        current_version: extension.version,
        latest_version: extension.version,
        status: "current",
      },
    });
  }),
  http.delete("/api/extensions/:name", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({
      extension: { name, path: `/tmp/agh/extensions/${name}`, status: "removed" },
    });
  }),
  http.get("/api/extensions/:name/provenance", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension?.provenance) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ provenance: extension.provenance });
  }),
  http.post("/api/extensions/:name/enable", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ extension: { ...extension, enabled: true } });
  }),
  http.post("/api/extensions/:name/disable", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ extension: { ...extension, enabled: false } });
  }),
];
