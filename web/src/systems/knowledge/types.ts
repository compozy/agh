import type { OperationQuery, OperationResponse } from "@/lib/api-contract";

export type MemoryHeader = OperationResponse<"listMemory", 200>[number];
export type MemoryReadResponse = OperationResponse<"readMemory", 200>;
export type MemoryMutationResponse = OperationResponse<"writeMemory", 200>;
export type MemoryConsolidateResponse = OperationResponse<"consolidateMemory", 200>;
export type MemoryScope = NonNullable<OperationQuery<"listMemory">>["scope"];
export type MemoryType = MemoryHeader["type"];
export type KnowledgeScope = Exclude<MemoryScope, undefined>;

export interface KnowledgeMemoryItem extends MemoryHeader {
  scope?: KnowledgeScope;
  key?: string;
}

export type KnowledgeFilter = {
  scope?: MemoryScope;
  workspace?: string;
  type?: MemoryType;
  search?: string;
};
