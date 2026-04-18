import { http, HttpResponse, type HttpHandler } from "msw";

import {
  settingsAppliedMutationFixture,
  settingsAutomationSectionFixture,
  settingsEnvironmentsCollectionFixture,
  settingsEnvironmentFixtures,
  settingsExtensionsCollectionFixture,
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

  http.get("/api/settings/environments", () =>
    HttpResponse.json(settingsEnvironmentsCollectionFixture)
  ),
  http.get("/api/settings/environments/:name", ({ params }) => {
    const name = String(params.name);
    const environment = settingsEnvironmentFixtures.find(entry => entry.name === name);

    if (!environment) {
      return HttpResponse.json({ error: `Environment not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ environment });
  }),
  http.put("/api/settings/environments/:name", () =>
    HttpResponse.json(mutationResult("environments", true))
  ),
  http.delete("/api/settings/environments/:name", () =>
    HttpResponse.json(mutationResult("environments", true))
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

  http.post("/api/settings/actions/restart", () =>
    HttpResponse.json(settingsRestartResponseFixture, { status: 202 })
  ),
  http.get("/api/settings/actions/restart/:operation_id", () =>
    HttpResponse.json(settingsRestartStatusFixture)
  ),

  http.get("/api/extensions", () => HttpResponse.json(settingsExtensionsCollectionFixture)),
  http.post("/api/extensions/:name/enable", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ ...extension, enabled: true });
  }),
  http.post("/api/extensions/:name/disable", ({ params }) => {
    const name = String(params.name);
    const extension = settingsExtensionFixtures.find(entry => entry.name === name);

    if (!extension) {
      return HttpResponse.json({ error: `Extension not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ ...extension, enabled: false });
  }),
];
