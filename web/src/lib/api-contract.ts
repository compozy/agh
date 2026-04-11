import type { operations } from "@/generated/agh-openapi";

export type OperationId = keyof operations;

type JsonContent<T> = T extends { content: { "application/json": infer Body } } ? Body : never;

export type OperationResponse<
  Id extends OperationId,
  Status extends keyof operations[Id]["responses"],
> = JsonContent<operations[Id]["responses"][Status]>;

export type OperationRequestBody<Id extends OperationId> = operations[Id] extends {
  requestBody: { content: { "application/json": infer Body } };
}
  ? Body
  : never;

export type OperationQuery<Id extends OperationId> = operations[Id] extends {
  parameters: { query?: infer Query };
}
  ? Query
  : never;

export type OperationPath<Id extends OperationId> = operations[Id] extends {
  parameters: { path?: infer Path };
}
  ? Path
  : never;
