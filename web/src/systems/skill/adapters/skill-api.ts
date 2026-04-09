import {
  skillsResponseSchema,
  skillContentResponseSchema,
  skillResponseSchema,
  skillActionResponseSchema,
  type SkillPayload,
  type SkillActionResponse,
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
  const res = await fetch(`/api/skills?workspace=${encodeURIComponent(workspace)}`, { signal });
  if (!res.ok) {
    throw new SkillApiError(`Failed to fetch skills: ${res.status}`, res.status);
  }
  const json = await res.json();
  const parsed = skillsResponseSchema.parse(json);
  return parsed.skills;
}

export async function getSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillPayload> {
  const res = await fetch(
    `/api/skills/${encodeURIComponent(name)}?workspace=${encodeURIComponent(workspace)}`,
    { signal }
  );
  if (!res.ok) {
    if (res.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(`Failed to fetch skill "${name}": ${res.status}`, res.status);
  }
  const json = await res.json();
  const parsed = skillResponseSchema.parse(json);
  return parsed.skill;
}

export async function getSkillContent(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<string> {
  const res = await fetch(
    `/api/skills/${encodeURIComponent(name)}/content?workspace=${encodeURIComponent(workspace)}`,
    { signal }
  );
  if (!res.ok) {
    if (res.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(`Failed to fetch skill content "${name}": ${res.status}`, res.status);
  }
  const json = await res.json();
  const parsed = skillContentResponseSchema.parse(json);
  return parsed.content;
}

export async function enableSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillActionResponse> {
  const res = await fetch(
    `/api/skills/${encodeURIComponent(name)}/enable?workspace=${encodeURIComponent(workspace)}`,
    { method: "POST", signal }
  );
  if (!res.ok) {
    if (res.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(`Failed to enable skill "${name}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return skillActionResponseSchema.parse(json);
}

export async function disableSkill(
  name: string,
  workspace: string,
  signal?: AbortSignal
): Promise<SkillActionResponse> {
  const res = await fetch(
    `/api/skills/${encodeURIComponent(name)}/disable?workspace=${encodeURIComponent(workspace)}`,
    { method: "POST", signal }
  );
  if (!res.ok) {
    if (res.status === 404) {
      throw new SkillApiError(`Skill not found: ${name}`, 404);
    }
    throw new SkillApiError(`Failed to disable skill "${name}": ${res.status}`, res.status);
  }
  const json = await res.json();
  return skillActionResponseSchema.parse(json);
}
