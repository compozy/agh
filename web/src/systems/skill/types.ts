import type { OperationRequestBody, OperationResponse } from "@/lib/api-contract";

export type SkillsResponse = OperationResponse<"listSkills", 200>;
export type SkillPayload = SkillsResponse["skills"][number];
export type SkillResponse = OperationResponse<"getSkill", 200>;
export type SkillContentResponse = OperationResponse<"getSkillContent", 200>;
export type SkillActionResponse = OperationResponse<"enableSkill", 200>;
export type ProvenancePayload = NonNullable<SkillPayload["provenance"]>;

export type SkillMarketplaceSearchResponse = OperationResponse<"searchSkillMarketplace", 200>;
export type SkillMarketplaceListingPayload = SkillMarketplaceSearchResponse["skills"][number];

export type SkillMarketplaceInfoResponse = OperationResponse<"getSkillMarketplaceInfo", 200>;
export type SkillMarketplaceDetailPayload = SkillMarketplaceInfoResponse["skill"];

export type SkillMarketplaceInstallResponse = OperationResponse<"installSkillMarketplace", 200>;
export type SkillMarketplaceInstallPayload = SkillMarketplaceInstallResponse["skill"];
export type SkillMarketplaceInstallRequest = OperationRequestBody<"installSkillMarketplace">;

export type SkillMarketplaceUpdateResponse = OperationResponse<"updateSkillMarketplace", 200>;
export type SkillMarketplaceUpdatePayload = SkillMarketplaceUpdateResponse["skills"][number];
export type SkillMarketplaceUpdateRequest = OperationRequestBody<"updateSkillMarketplace">;

export type SkillMarketplaceRemoveResponse = OperationResponse<"removeSkillMarketplace", 200>;
export type SkillMarketplaceRemovePayload = SkillMarketplaceRemoveResponse["skill"];
