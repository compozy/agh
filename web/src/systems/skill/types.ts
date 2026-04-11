import type { OperationResponse } from "@/lib/api-contract";

export type SkillsResponse = OperationResponse<"listSkills", 200>;
export type SkillPayload = SkillsResponse["skills"][number];
export type SkillResponse = OperationResponse<"getSkill", 200>;
export type SkillContentResponse = OperationResponse<"getSkillContent", 200>;
export type SkillActionResponse = OperationResponse<"enableSkill", 200>;
export type ProvenancePayload = NonNullable<SkillPayload["provenance"]>;
