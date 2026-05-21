import {
  apiClient,
  apiRequestFailed,
  defaultApiErrorMessage,
  requireResponseData,
} from "@/lib/api-client";

import type {
  SkillActionResponse,
  SkillMarketplaceDetailPayload,
  SkillMarketplaceInstallPayload,
  SkillMarketplaceInstallRequest,
  SkillMarketplaceListingPayload,
  SkillMarketplaceRemovePayload,
  SkillMarketplaceUpdatePayload,
  SkillMarketplaceUpdateRequest,
  SkillPayload,
  SkillShadowsResponse,
} from "../types";

export class SkillApiError extends Error {
  constructor(
    message: string,
    public readonly status: number
  ) {
    super(message);
    this.name = "SkillApiError";
  }
}

export async function listSkills(workspace: string, signal?: AbortSignal): Promise<SkillPayload[]> {
  const { data, error, response } = await apiClient.GET("/api/skills", {
    params: { query: { workspace } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new SkillApiError(
      defaultApiErrorMessage("Failed to fetch skills", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to fetch skills").skills;
}

export async function getSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillPayload> {
  const { data, error, response } = await apiClient.GET("/api/skills/{name}", {
    params: {
      path: { name },
      query: { workspace },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to fetch skill "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to fetch skill "${name}"`).skill;
}

export async function getSkillContent(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<string> {
  const { data, error, response } = await apiClient.GET("/api/skills/{name}/content", {
    params: {
      path: { name },
      query: { workspace },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to fetch skill content "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to fetch skill content "${name}"`).content;
}

export async function getSkillShadows(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillShadowsResponse> {
  const { data, error, response } = await apiClient.GET("/api/skills/{name}/shadows", {
    params: {
      path: { name },
      query: { workspace },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError("Skill not found: " + name, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage('Failed to fetch skill resolution "' + name + '"', response, error),
      response.status
    );
  }
  return requireResponseData(data, response, 'Failed to fetch skill resolution "' + name + '"');
}

export async function enableSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillActionResponse> {
  const { data, error, response } = await apiClient.POST("/api/skills/{name}/enable", {
    params: {
      path: { name },
      query: { workspace },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to enable skill "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to enable skill "${name}"`);
}

export async function disableSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillActionResponse> {
  const { data, error, response } = await apiClient.POST("/api/skills/{name}/disable", {
    params: {
      path: { name },
      query: { workspace },
    },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to disable skill "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to disable skill "${name}"`);
}

export interface SearchSkillMarketplaceParams {
  query: string;
  limit?: number;
}

export async function searchSkillMarketplace(
  params: SearchSkillMarketplaceParams,
  signal?: AbortSignal
): Promise<SkillMarketplaceListingPayload[]> {
  const queryParams: { query: string; limit?: number } = { query: params.query };
  if (params.limit !== undefined) {
    queryParams.limit = params.limit;
  }
  const { data, error, response } = await apiClient.GET("/api/skills/marketplace/search", {
    params: { query: queryParams },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    throw new SkillApiError(
      defaultApiErrorMessage("Failed to search marketplace skills", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to search marketplace skills").skills;
}

export async function getSkillMarketplaceInfo(
  slug: string,
  signal?: AbortSignal
): Promise<SkillMarketplaceDetailPayload> {
  const { data, error, response } = await apiClient.GET("/api/skills/marketplace/info", {
    params: { query: { slug } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Marketplace skill not found: ${slug}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to fetch marketplace info for "${slug}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to fetch marketplace info for "${slug}"`)
    .skill;
}

export async function installSkillMarketplace(
  body: SkillMarketplaceInstallRequest,
  signal?: AbortSignal
): Promise<SkillMarketplaceInstallPayload> {
  const { data, error, response } = await apiClient.POST("/api/skills/marketplace/install", {
    body,
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Marketplace skill not found: ${body.slug}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to install marketplace skill "${body.slug}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to install marketplace skill "${body.slug}"`)
    .skill;
}

export async function updateSkillMarketplace(
  body: SkillMarketplaceUpdateRequest,
  signal?: AbortSignal
): Promise<SkillMarketplaceUpdatePayload[]> {
  const { data, error, response } = await apiClient.POST("/api/skills/marketplace/update", {
    body,
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(
        `Installed marketplace skill not found: ${body.name ?? "<all>"}`,
        404
      );
    }
    throw new SkillApiError(
      defaultApiErrorMessage("Failed to update marketplace skills", response, error),
      response.status
    );
  }
  return requireResponseData(data, response, "Failed to update marketplace skills").skills;
}

export async function removeSkillMarketplace(
  name: string,
  signal?: AbortSignal
): Promise<SkillMarketplaceRemovePayload> {
  const { data, error, response } = await apiClient.DELETE("/api/skills/marketplace/{name}", {
    params: { path: { name } },
    signal,
  });
  if (apiRequestFailed(response, error)) {
    if (response.status === 404) {
      throw new SkillApiError(`Installed marketplace skill not found: ${name}`, 404);
    }
    throw new SkillApiError(
      defaultApiErrorMessage(`Failed to remove marketplace skill "${name}"`, response, error),
      response.status
    );
  }
  return requireResponseData(data, response, `Failed to remove marketplace skill "${name}"`).skill;
}
