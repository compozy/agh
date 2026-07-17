import { HttpResponse, type HttpHandler } from "msw";
import { aghApiMock } from "@/storybook/openapi-msw";

import type { SettingsNotificationPresetCollection } from "@/systems/settings";

import {
  settingsAppliedMutationFixture,
  settingsApplyRecordsFixture,
  settingsAutomationSectionFixture,
  settingsSandboxesCollectionFixture,
  settingsSandboxFixtures,
  settingsGeneralSectionFixture,
  settingsHooksCollectionFixture,
  settingsHooksExtensionsSectionFixture,
  mcpAuthBeginFixture,
  mcpAuthStatusAuthenticatedFixture,
  settingsMCPServersCollectionFixture,
  settingsMemorySectionFixture,
  settingsNetworkSectionFixture,
  settingsNotificationPresetCollectionFixture,
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
  aghApiMock.get("/api/settings/general", () => HttpResponse.json(settingsGeneralSectionFixture)),
  aghApiMock.patch("/api/settings/general", () =>
    HttpResponse.json(mutationResult("general", true))
  ),

  aghApiMock.get("/api/settings/memory", () => HttpResponse.json(settingsMemorySectionFixture)),
  aghApiMock.patch("/api/settings/memory", () => HttpResponse.json(mutationResult("memory"))),

  aghApiMock.get("/api/settings/skills", () => HttpResponse.json(settingsSkillsSectionFixture)),
  aghApiMock.patch("/api/settings/skills", () => HttpResponse.json(mutationResult("skills", true))),

  aghApiMock.get("/api/settings/automation", () =>
    HttpResponse.json(settingsAutomationSectionFixture)
  ),
  aghApiMock.patch("/api/settings/automation", () =>
    HttpResponse.json(mutationResult("automation", true))
  ),

  aghApiMock.get("/api/settings/network", () => HttpResponse.json(settingsNetworkSectionFixture)),
  aghApiMock.patch("/api/settings/network", () =>
    HttpResponse.json(mutationResult("network", true))
  ),

  aghApiMock.get("/api/settings/observability", () =>
    HttpResponse.json(settingsObservabilitySectionFixture)
  ),
  aghApiMock.patch("/api/settings/observability", () =>
    HttpResponse.json(mutationResult("observability"))
  ),

  aghApiMock.get("/api/settings/hooks-extensions", () =>
    HttpResponse.json(settingsHooksExtensionsSectionFixture)
  ),
  aghApiMock.patch("/api/settings/hooks-extensions", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),

  aghApiMock.get("/api/notifications/presets", () =>
    HttpResponse.json(settingsNotificationPresetCollectionFixture)
  ),
  aghApiMock.post("/api/notifications/presets", async ({ request }) => {
    const body = (await request.json()) as {
      name?: string;
      events?: string[];
      targets?: SettingsNotificationPresetCollection["presets"][number]["targets"];
      filter?: string;
      enabled?: boolean;
    };
    return HttpResponse.json(
      {
        preset: {
          name: body.name ?? "custom",
          events: body.events ?? [],
          targets: body.targets ?? [],
          filter: body.filter ?? "",
          enabled: body.enabled ?? false,
          built_in: false,
          default_version: "",
          default_hash: "",
          user_modified: false,
          default_update_available: false,
          created_at: "2026-04-17T11:30:00Z",
          updated_at: "2026-04-17T11:30:00Z",
        },
      },
      { status: 201 }
    );
  }),
  aghApiMock.put("/api/notifications/presets/{name}", async ({ params, request }) => {
    const name = String(params.name);
    const body = (await request.json()) as { enabled?: boolean };
    const existing = settingsNotificationPresetCollectionFixture.presets.find(
      preset => preset.name === name
    );
    return HttpResponse.json({
      preset: {
        ...(existing ?? settingsNotificationPresetCollectionFixture.presets[0]),
        name,
        enabled: body.enabled ?? existing?.enabled ?? true,
        updated_at: "2026-04-17T11:45:00Z",
      },
    });
  }),
  aghApiMock.delete(
    "/api/notifications/presets/{name}",
    () => new HttpResponse(null, { status: 204 })
  ),

  aghApiMock.get("/api/settings/providers", () =>
    HttpResponse.json(settingsProvidersCollectionFixture)
  ),
  aghApiMock.get("/api/settings/providers/{name}", ({ params }) => {
    const name = String(params.name);
    const provider = settingsProviderFixtures.find(entry => entry.name === name);

    if (!provider) {
      return HttpResponse.json({ error: `Provider not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ provider });
  }),
  aghApiMock.put("/api/settings/providers/{name}", () =>
    HttpResponse.json(mutationResult("providers", true))
  ),
  aghApiMock.delete("/api/settings/providers/{name}", () =>
    HttpResponse.json(mutationResult("providers", true))
  ),

  aghApiMock.get("/api/settings/sandboxes", () =>
    HttpResponse.json(settingsSandboxesCollectionFixture)
  ),
  aghApiMock.get("/api/settings/sandboxes/{name}", ({ params }) => {
    const name = String(params.name);
    const sandbox = settingsSandboxFixtures.find(entry => entry.name === name);

    if (!sandbox) {
      return HttpResponse.json({ error: `Sandbox not found: ${name}` }, { status: 404 });
    }

    return HttpResponse.json({ sandbox });
  }),
  aghApiMock.put("/api/settings/sandboxes/{name}", () =>
    HttpResponse.json(mutationResult("sandboxes", true))
  ),
  aghApiMock.delete("/api/settings/sandboxes/{name}", () =>
    HttpResponse.json(mutationResult("sandboxes", true))
  ),

  aghApiMock.get("/api/settings/hooks", () => HttpResponse.json(settingsHooksCollectionFixture)),
  aghApiMock.put("/api/settings/hooks/{name}", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),
  aghApiMock.delete("/api/settings/hooks/{name}", () =>
    HttpResponse.json(mutationResult("hooks-extensions", true))
  ),

  aghApiMock.get("/api/settings/mcp-servers", ({ request }) => {
    const url = new URL(request.url);
    const scope = url.searchParams.get("scope");
    const workspaceId = url.searchParams.get("workspace_id");

    if (scope === "workspace" && workspaceId) {
      return HttpResponse.json({
        ...settingsMCPServersCollectionFixture,
        mcp_servers: [],
        scope: "workspace",
        workspace_id: workspaceId,
      });
    }

    return HttpResponse.json(settingsMCPServersCollectionFixture);
  }),
  aghApiMock.put("/api/settings/mcp-servers/{name}", () =>
    HttpResponse.json(mutationResult("mcp-servers", true))
  ),
  aghApiMock.delete("/api/settings/mcp-servers/{name}", () =>
    HttpResponse.json(mutationResult("mcp-servers", true))
  ),
  aghApiMock.post("/api/settings/mcp-servers/{name}/auth/begin", () =>
    HttpResponse.json(mcpAuthBeginFixture)
  ),
  aghApiMock.post("/api/settings/mcp-servers/{name}/auth/exchange", () =>
    HttpResponse.json(mcpAuthStatusAuthenticatedFixture)
  ),
  aghApiMock.post("/api/settings/mcp-servers/{name}/auth/logout", () =>
    HttpResponse.json({
      ...mcpAuthStatusAuthenticatedFixture,
      status: "needs_login",
      token_present: false,
    })
  ),

  aghApiMock.get("/api/settings/apply", ({ request }) =>
    HttpResponse.json(applyRecordsForUrl(request))
  ),
  aghApiMock.post("/api/settings/reload", () => HttpResponse.json(settingsReloadBlockedFixture)),

  aghApiMock.post("/api/settings/actions/restart", () =>
    HttpResponse.json(settingsRestartResponseFixture, { status: 202 })
  ),
  aghApiMock.get("/api/settings/actions/restart/{operation_id}", () =>
    HttpResponse.json(settingsRestartStatusFixture)
  ),
];
