import { describe, expectTypeOf, it } from "vitest";

import type {
  OperationPath,
  OperationQuery,
  OperationRequestBody,
  OperationResponse,
} from "@/lib/api-contract";

type UpdateSettingsGeneralBody = OperationRequestBody<"updateSettingsGeneral">;
type GetSettingsGeneralResponse = OperationResponse<"getSettingsGeneral", 200>;
type PutSettingsMCPServerQuery = OperationQuery<"putSettingsMCPServer">;
type PutSettingsMCPServerBody = OperationRequestBody<"putSettingsMCPServer">;
type ListSettingsMCPServersResponse = OperationResponse<"listSettingsMCPServers", 200>;
type GetSettingsObservabilityResponse = OperationResponse<"getSettingsObservability", 200>;
type TriggerSettingsRestartResponse = OperationResponse<"triggerSettingsRestart", 202>;
type GetSettingsRestartStatusPath = OperationPath<"getSettingsRestartStatus">;
type GetSettingsRestartStatusResponse = OperationResponse<"getSettingsRestartStatus", 200>;

describe("settings openapi contract", () => {
  it("keeps generated settings operation types aligned with the API surface", () => {
    expectTypeOf<UpdateSettingsGeneralBody["config"]["session_timeout"]>().toEqualTypeOf<string>();
    expectTypeOf<UpdateSettingsGeneralBody["config"]["permissions"]["mode"]>().toEqualTypeOf<
      "approve-all" | "approve-reads" | "deny-all"
    >();

    expectTypeOf<GetSettingsGeneralResponse["section"]>().toEqualTypeOf<
      | "general"
      | "memory"
      | "skills"
      | "automation"
      | "network"
      | "observability"
      | "hooks-extensions"
    >();
    expectTypeOf<GetSettingsGeneralResponse["scope"]>().toEqualTypeOf<"global" | "workspace">();
    expectTypeOf<GetSettingsGeneralResponse["workspace_id"]>().toEqualTypeOf<string | undefined>();
    expectTypeOf<GetSettingsGeneralResponse["available_scopes"][number]>().toEqualTypeOf<
      "global" | "workspace"
    >();
    expectTypeOf<GetSettingsGeneralResponse["config_paths"]["log_file"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsGeneralResponse["config"]["session_timeout"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsGeneralResponse["runtime"]["available"]>().toEqualTypeOf<boolean>();
    expectTypeOf<GetSettingsGeneralResponse["runtime"]["started_at"]>().toEqualTypeOf<
      string | null | undefined
    >();
    expectTypeOf<
      GetSettingsGeneralResponse["actions"]["restart"]["name"]
    >().toEqualTypeOf<string>();
    expectTypeOf<
      GetSettingsGeneralResponse["actions"]["restart"]["available"]
    >().toEqualTypeOf<boolean>();
    expectTypeOf<GetSettingsGeneralResponse["actions"]["restart"]["behavior"]>().toEqualTypeOf<
      "action_trigger" | "applied_now" | "restart_required"
    >();

    expectTypeOf<PutSettingsMCPServerQuery["scope"]>().toEqualTypeOf<
      "global" | "workspace" | undefined
    >();
    expectTypeOf<PutSettingsMCPServerQuery["workspace_id"]>().toEqualTypeOf<string | undefined>();
    expectTypeOf<PutSettingsMCPServerQuery["target"]>().toEqualTypeOf<
      "auto" | "config" | "sidecar" | undefined
    >();
    expectTypeOf<PutSettingsMCPServerBody["server"]["name"]>().toEqualTypeOf<string>();
    expectTypeOf<PutSettingsMCPServerBody["server"]["transport"]>().toEqualTypeOf<
      string | undefined
    >();
    expectTypeOf<PutSettingsMCPServerBody["server"]["command"]>().toEqualTypeOf<
      string | undefined
    >();
    expectTypeOf<PutSettingsMCPServerBody["server"]["url"]>().toEqualTypeOf<string | undefined>();
    expectTypeOf<PutSettingsMCPServerBody["server"]["auth"]>().toEqualTypeOf<
      | {
          authorization_url?: string;
          client_id?: string;
          client_secret_env?: string;
          issuer_url?: string;
          metadata_url?: string;
          revocation_url?: string;
          scopes?: string[];
          token_url?: string;
          type?: string;
        }
      | null
      | undefined
    >();
    expectTypeOf<PutSettingsMCPServerBody["server"]["args"]>().toEqualTypeOf<
      string[] | undefined
    >();
    expectTypeOf<PutSettingsMCPServerBody["server"]["env"]>().toEqualTypeOf<
      Record<string, string> | undefined
    >();

    expectTypeOf<ListSettingsMCPServersResponse["collection"]>().toEqualTypeOf<
      "providers" | "mcp-servers" | "environments" | "hooks"
    >();
    expectTypeOf<ListSettingsMCPServersResponse["scope"]>().toEqualTypeOf<"global" | "workspace">();
    expectTypeOf<ListSettingsMCPServersResponse["workspace_id"]>().toEqualTypeOf<
      string | undefined
    >();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["name"]
    >().toEqualTypeOf<string>();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["transport"]
    >().toEqualTypeOf<string>();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["auth_status"]
    >().toEqualTypeOf<
      | {
          auth_type?: string;
          authorization_url?: string;
          client_id?: string;
          diagnostic?: string;
          expires_at?: string | null;
          issuer?: string;
          refreshable: boolean;
          remote_url?: string;
          revocation_url?: string;
          scopes?: string[];
          server_name: string;
          status: string;
          token_present: boolean;
          updated_at?: string | null;
        }
      | null
      | undefined
    >();
    expectTypeOf<ListSettingsMCPServersResponse["mcp_servers"][number]["scope"]>().toEqualTypeOf<
      "global" | "workspace"
    >();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["source_metadata"]["effective_source"]["kind"]
    >().toEqualTypeOf<
      | "builtin-provider"
      | "global-config"
      | "workspace-config"
      | "global-mcp-sidecar"
      | "workspace-mcp-sidecar"
    >();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["source_metadata"]["available_targets"][number]
    >().toEqualTypeOf<
      "global-config" | "workspace-config" | "global-mcp-sidecar" | "workspace-mcp-sidecar"
    >();
    expectTypeOf<
      ListSettingsMCPServersResponse["mcp_servers"][number]["source_metadata"]["shadowed_sources"]
    >().toEqualTypeOf<
      | {
          kind:
            | "builtin-provider"
            | "global-config"
            | "workspace-config"
            | "global-mcp-sidecar"
            | "workspace-mcp-sidecar";
          scope: "global" | "workspace";
          workspace_id?: string;
        }[]
      | undefined
    >();

    expectTypeOf<
      GetSettingsObservabilityResponse["log_tail"]["available"]
    >().toEqualTypeOf<boolean>();
    expectTypeOf<GetSettingsObservabilityResponse["log_tail"]["stream_url"]>().toEqualTypeOf<
      string | undefined
    >();
    expectTypeOf<GetSettingsObservabilityResponse["log_tail"]["transport"]>().toEqualTypeOf<
      "sse" | undefined
    >();

    expectTypeOf<TriggerSettingsRestartResponse["operation_id"]>().toEqualTypeOf<string>();
    expectTypeOf<TriggerSettingsRestartResponse["status"]>().toEqualTypeOf<
      "failed" | "pending" | "ready" | "starting" | "stopping" | "waiting_release"
    >();
    expectTypeOf<TriggerSettingsRestartResponse["status_url"]>().toEqualTypeOf<string>();
    expectTypeOf<TriggerSettingsRestartResponse["active_session_count"]>().toEqualTypeOf<number>();

    expectTypeOf<GetSettingsRestartStatusPath["operation_id"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsRestartStatusResponse["old_pid"]>().toEqualTypeOf<number>();
    expectTypeOf<GetSettingsRestartStatusResponse["old_started_at"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsRestartStatusResponse["old_socket_path"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsRestartStatusResponse["new_pid"]>().toEqualTypeOf<number | undefined>();
    expectTypeOf<GetSettingsRestartStatusResponse["failure_reason"]>().toEqualTypeOf<
      string | undefined
    >();
    expectTypeOf<GetSettingsRestartStatusResponse["completed_at"]>().toEqualTypeOf<
      string | null | undefined
    >();
    expectTypeOf<GetSettingsRestartStatusResponse["started_at"]>().toEqualTypeOf<string>();
    expectTypeOf<GetSettingsRestartStatusResponse["updated_at"]>().toEqualTypeOf<string>();
  });
});
