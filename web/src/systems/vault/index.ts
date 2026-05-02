// Types
export type {
  PutVaultSecretRequest,
  VaultListFilter,
  VaultNamespace,
  VaultSecret,
  VaultSecretDetail,
  VaultSecretResponse,
  VaultSecretsResponse,
} from "./types";
export { VAULT_NAMESPACES } from "./types";

// Adapters
export {
  deleteVaultSecret,
  getVaultSecret,
  listVaultSecrets,
  putVaultSecret,
  VaultApiError,
} from "./adapters/vault-api";

// Query infrastructure
export { vaultKeys } from "./lib/query-keys";
export {
  sessionVaultSecretsOptions,
  vaultSecretDetailOptions,
  vaultSecretsListOptions,
  VAULT_QUERY_INTERVALS,
} from "./lib/query-options";

// Hooks
export { useSessionVaultSecrets, useVaultSecret, useVaultSecrets } from "./hooks/use-vault";
export { useDeleteVaultSecret, usePutVaultSecret } from "./hooks/use-vault-actions";

// Components
export { SessionVaultPanel, VaultSecretsTable } from "./components";
