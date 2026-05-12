package builtin

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	toolspkg "github.com/pedronauck/agh/internal/tools"
)

func TestBuiltinNativeDescriptors(t *testing.T) {
	t.Parallel()

	t.Run("Should expose exactly the MVP native tool scope", func(t *testing.T) {
		t.Parallel()

		descriptors := NativeDescriptors()
		got := make(map[toolspkg.ToolID]toolspkg.Descriptor, len(descriptors))
		for _, descriptor := range descriptors {
			if err := descriptor.Validate(); err != nil {
				t.Fatalf("descriptor %q Validate() error = %v", descriptor.ID, err)
			}
			got[descriptor.ID] = descriptor
		}

		want := []toolspkg.ToolID{
			toolspkg.ToolIDToolList,
			toolspkg.ToolIDToolSearch,
			toolspkg.ToolIDToolInfo,
			toolspkg.ToolIDSkillList,
			toolspkg.ToolIDSkillSearch,
			toolspkg.ToolIDSkillView,
			toolspkg.ToolIDNetworkStatus,
			toolspkg.ToolIDNetworkChannels,
			toolspkg.ToolIDNetworkInbox,
			toolspkg.ToolIDNetworkPeers,
			toolspkg.ToolIDNetworkSend,
			toolspkg.ToolIDNetworkThreads,
			toolspkg.ToolIDNetworkThreadMessages,
			toolspkg.ToolIDNetworkDirects,
			toolspkg.ToolIDNetworkDirectResolve,
			toolspkg.ToolIDNetworkDirectMessages,
			toolspkg.ToolIDNetworkWork,
			toolspkg.ToolIDSessionList,
			toolspkg.ToolIDSessionStatus,
			toolspkg.ToolIDSessionHistory,
			toolspkg.ToolIDSessionEvents,
			toolspkg.ToolIDSessionDescribe,
			toolspkg.ToolIDSessionHealth,
			toolspkg.ToolIDAgentHeartbeatStatus,
			toolspkg.ToolIDAgentHeartbeatWake,
			toolspkg.ToolIDWorkspaceList,
			toolspkg.ToolIDWorkspaceInfo,
			toolspkg.ToolIDWorkspaceDescribe,
			toolspkg.ToolIDProviderModelsList,
			toolspkg.ToolIDProviderModelsRefresh,
			toolspkg.ToolIDProviderModelsStatus,
			toolspkg.ToolIDMemoryList,
			toolspkg.ToolIDMemoryShow,
			toolspkg.ToolIDMemorySearch,
			toolspkg.ToolIDMemoryPropose,
			toolspkg.ToolIDMemoryNote,
			toolspkg.ToolIDMemoryHealth,
			toolspkg.ToolIDMemoryScopeShow,
			toolspkg.ToolIDMemoryAdminHistory,
			toolspkg.ToolIDMemoryReindex,
			toolspkg.ToolIDMemoryPromote,
			toolspkg.ToolIDMemoryReset,
			toolspkg.ToolIDMemoryReload,
			toolspkg.ToolIDMemoryDecisionsList,
			toolspkg.ToolIDMemoryDecisionsShow,
			toolspkg.ToolIDMemoryDecisionsRevert,
			toolspkg.ToolIDMemoryRecallTrace,
			toolspkg.ToolIDMemoryDreamStatus,
			toolspkg.ToolIDMemoryDreamList,
			toolspkg.ToolIDMemoryDreamShow,
			toolspkg.ToolIDMemoryDreamTrigger,
			toolspkg.ToolIDMemoryDreamRetry,
			toolspkg.ToolIDMemoryDailyList,
			toolspkg.ToolIDMemoryExtractorStatus,
			toolspkg.ToolIDMemoryExtractorFailures,
			toolspkg.ToolIDMemoryExtractorRetry,
			toolspkg.ToolIDMemoryExtractorDrain,
			toolspkg.ToolIDMemoryProviderList,
			toolspkg.ToolIDMemoryProviderGet,
			toolspkg.ToolIDMemoryProviderSelect,
			toolspkg.ToolIDMemoryProviderEnable,
			toolspkg.ToolIDMemoryProviderDisable,
			toolspkg.ToolIDMemorySessionLedger,
			toolspkg.ToolIDMemorySessionReplay,
			toolspkg.ToolIDMemorySessionsPrune,
			toolspkg.ToolIDMemorySessionsRepair,
			toolspkg.ToolIDObserveEvents,
			toolspkg.ToolIDObserveMetrics,
			toolspkg.ToolIDObserveSearch,
			toolspkg.ToolIDBridgesList,
			toolspkg.ToolIDBridgesStatus,
			toolspkg.ToolIDTaskList,
			toolspkg.ToolIDTaskRead,
			toolspkg.ToolIDTaskCreate,
			toolspkg.ToolIDTaskChildCreate,
			toolspkg.ToolIDTaskUpdate,
			toolspkg.ToolIDTaskCancel,
			toolspkg.ToolIDTaskRunList,
			toolspkg.ToolIDTaskRunReviewRequest,
			toolspkg.ToolIDTaskRunReviewList,
			toolspkg.ToolIDTaskRunReviewShow,
			toolspkg.ToolIDTaskExecutionProfileGet,
			toolspkg.ToolIDTaskExecutionProfileSet,
			toolspkg.ToolIDTaskExecutionProfileDelete,
			toolspkg.ToolIDTaskNotificationSubscribe,
			toolspkg.ToolIDTaskNotificationList,
			toolspkg.ToolIDTaskNotificationShow,
			toolspkg.ToolIDTaskNotificationDelete,
			toolspkg.ToolIDTaskRunClaimNext,
			toolspkg.ToolIDTaskRunHeartbeat,
			toolspkg.ToolIDTaskRunComplete,
			toolspkg.ToolIDTaskRunFail,
			toolspkg.ToolIDTaskRunRelease,
			toolspkg.ToolIDTaskRunReviewSubmit,
			toolspkg.ToolIDConfigShow,
			toolspkg.ToolIDConfigList,
			toolspkg.ToolIDConfigGet,
			toolspkg.ToolIDConfigSet,
			toolspkg.ToolIDConfigUnset,
			toolspkg.ToolIDConfigDiff,
			toolspkg.ToolIDConfigPath,
			toolspkg.ToolIDHooksList,
			toolspkg.ToolIDHooksInfo,
			toolspkg.ToolIDHooksEvents,
			toolspkg.ToolIDHooksRuns,
			toolspkg.ToolIDHooksCreate,
			toolspkg.ToolIDHooksUpdate,
			toolspkg.ToolIDHooksDelete,
			toolspkg.ToolIDHooksEnable,
			toolspkg.ToolIDHooksDisable,
			toolspkg.ToolIDAutomationJobsList,
			toolspkg.ToolIDAutomationJobsGet,
			toolspkg.ToolIDAutomationJobsCreate,
			toolspkg.ToolIDAutomationJobsUpdate,
			toolspkg.ToolIDAutomationJobsDelete,
			toolspkg.ToolIDAutomationJobsEnable,
			toolspkg.ToolIDAutomationJobsDisable,
			toolspkg.ToolIDAutomationJobsTrigger,
			toolspkg.ToolIDAutomationJobsHistory,
			toolspkg.ToolIDAutomationTriggersList,
			toolspkg.ToolIDAutomationTriggersGet,
			toolspkg.ToolIDAutomationTriggersCreate,
			toolspkg.ToolIDAutomationTriggersUpdate,
			toolspkg.ToolIDAutomationTriggersDelete,
			toolspkg.ToolIDAutomationTriggersEnable,
			toolspkg.ToolIDAutomationTriggersDisable,
			toolspkg.ToolIDAutomationTriggersHistory,
			toolspkg.ToolIDAutomationRunsList,
			toolspkg.ToolIDAutomationRunsGet,
			toolspkg.ToolIDExtensionsSearch,
			toolspkg.ToolIDExtensionsList,
			toolspkg.ToolIDExtensionsInfo,
			toolspkg.ToolIDExtensionsInstall,
			toolspkg.ToolIDExtensionsUpdate,
			toolspkg.ToolIDExtensionsRemove,
			toolspkg.ToolIDExtensionsEnable,
			toolspkg.ToolIDExtensionsDisable,
			toolspkg.ToolIDMCPAuthStatus,
		}
		if gotLen, wantLen := len(got), len(want); gotLen != wantLen {
			t.Fatalf("len(NativeDescriptors()) = %d, want %d", gotLen, wantLen)
		}
		for _, id := range want {
			descriptor, ok := got[id]
			if !ok {
				t.Fatalf("descriptor %q missing from MVP native scope", id)
			}
			if descriptor.Backend.Kind != toolspkg.BackendNativeGo {
				t.Fatalf("%s backend kind = %q, want native_go", id, descriptor.Backend.Kind)
			}
			if descriptor.Backend.NativeName == "" {
				t.Fatalf("%s backend native name is empty", id)
			}
			if descriptor.Source != Source() {
				t.Fatalf("%s source = %#v, want builtin source", id, descriptor.Source)
			}
			if descriptor.Visibility != toolspkg.VisibilityModel {
				t.Fatalf("%s visibility = %q, want model", id, descriptor.Visibility)
			}
		}

		excluded := []toolspkg.ToolID{
			"agh__skill_install",
			"agh__skill_update",
			"agh__skill_remove",
			"agh__task_claim",
			"agh__task_release",
			"agh__task_complete",
			"agh__task_fail",
			"agh__task_run_start",
			"agh__task_run_cancel",
			"agh__mcp_auth_login",
			"agh__mcp_auth_logout",
			"agh__memory_read",
			"agh__memory_history",
			"agh__memory_write",
			"agh__memory_edit",
			"agh__memory_delete",
		}
		for _, id := range excluded {
			if _, ok := got[id]; ok {
				t.Fatalf("descriptor %q is registered but must be excluded from MVP native scope", id)
			}
		}
	})

	t.Run("Should expose provider-compatible top-level input schemas", func(t *testing.T) {
		t.Parallel()

		for _, descriptor := range NativeDescriptors() {
			var schema map[string]json.RawMessage
			if err := json.Unmarshal(descriptor.InputSchema, &schema); err != nil {
				t.Fatalf("%s input schema unmarshal error = %v", descriptor.ID, err)
			}
			for _, forbidden := range []string{"oneOf", "anyOf", "allOf"} {
				if _, ok := schema[forbidden]; ok {
					t.Fatalf(
						"%s input schema has top-level %s, want provider-compatible object schema",
						descriptor.ID,
						forbidden,
					)
				}
			}
		}
	})

	t.Run("Should classify read mutating open world and destructive risk flags", func(t *testing.T) {
		t.Parallel()

		descriptors := descriptorMap(NativeDescriptors())

		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDToolList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSkillView], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkStatus], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkChannels], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkInbox], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkPeers], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkThreads], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDNetworkThreadMessages],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkDirects], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDNetworkDirectMessages],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkWork], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionStatus], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionHistory], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionEvents], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionDescribe], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionHealth], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAgentHeartbeatStatus],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAgentHeartbeatWake],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceInfo], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceDescribe], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDProviderModelsList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDProviderModelsRefresh],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDProviderModelsStatus],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryShow], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemorySearch], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryPropose], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryNote], toolspkg.RiskMutating, false, false, false)
		for _, id := range []toolspkg.ToolID{
			toolspkg.ToolIDMemoryHealth,
			toolspkg.ToolIDMemoryScopeShow,
			toolspkg.ToolIDMemoryAdminHistory,
			toolspkg.ToolIDMemoryDecisionsList,
			toolspkg.ToolIDMemoryDecisionsShow,
			toolspkg.ToolIDMemoryRecallTrace,
			toolspkg.ToolIDMemoryDreamStatus,
			toolspkg.ToolIDMemoryDreamList,
			toolspkg.ToolIDMemoryDreamShow,
			toolspkg.ToolIDMemoryDailyList,
			toolspkg.ToolIDMemoryExtractorStatus,
			toolspkg.ToolIDMemoryExtractorFailures,
			toolspkg.ToolIDMemoryProviderList,
			toolspkg.ToolIDMemoryProviderGet,
			toolspkg.ToolIDMemorySessionLedger,
		} {
			requireDescriptorRisk(t, descriptors[id], toolspkg.RiskRead, true, false, false)
		}
		for _, id := range []toolspkg.ToolID{
			toolspkg.ToolIDMemoryReindex,
			toolspkg.ToolIDMemoryPromote,
			toolspkg.ToolIDMemoryReload,
			toolspkg.ToolIDMemoryDreamTrigger,
			toolspkg.ToolIDMemoryDreamRetry,
			toolspkg.ToolIDMemoryExtractorRetry,
			toolspkg.ToolIDMemoryExtractorDrain,
			toolspkg.ToolIDMemoryProviderSelect,
			toolspkg.ToolIDMemoryProviderEnable,
			toolspkg.ToolIDMemoryProviderDisable,
			toolspkg.ToolIDMemorySessionReplay,
			toolspkg.ToolIDMemorySessionsRepair,
		} {
			requireDescriptorRisk(t, descriptors[id], toolspkg.RiskMutating, false, false, false)
		}
		for _, id := range []toolspkg.ToolID{
			toolspkg.ToolIDMemoryReset,
			toolspkg.ToolIDMemoryDecisionsRevert,
			toolspkg.ToolIDMemorySessionsPrune,
		} {
			requireDescriptorRisk(t, descriptors[id], toolspkg.RiskDestructive, false, true, false)
		}
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveEvents], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveMetrics], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveSearch], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDBridgesList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDBridgesStatus], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRead], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRunList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunReviewRequest],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunReviewList],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunReviewShow],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkSend], toolspkg.RiskOpenWorld, false, false, true)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDNetworkDirectResolve],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskCreate], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskChildCreate],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskUpdate], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskCancel], toolspkg.RiskDestructive, false, true, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskExecutionProfileGet],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskExecutionProfileSet],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskExecutionProfileDelete],
			toolspkg.RiskDestructive,
			false,
			true,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskNotificationSubscribe],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskNotificationList],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskNotificationShow],
			toolspkg.RiskRead,
			true,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskNotificationDelete],
			toolspkg.RiskDestructive,
			false,
			true,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunClaimNext],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunHeartbeat],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunComplete],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRunFail], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRunRelease], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDTaskRunReviewSubmit],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		if got := descriptors[toolspkg.ToolIDTaskRunReviewSubmit].Backend.NativeName; got != "submit_run_review" {
			t.Fatalf("submit review native name = %q, want submit_run_review", got)
		}
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDConfigShow], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDConfigSet], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDConfigUnset], toolspkg.RiskDestructive, false, true, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDHooksList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDHooksCreate], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDHooksDelete], toolspkg.RiskDestructive, false, true, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDHooksDisable], toolspkg.RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDAutomationJobsList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAutomationJobsCreate],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAutomationJobsDelete],
			toolspkg.RiskDestructive,
			false,
			true,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAutomationJobsTrigger],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDAutomationRunsGet], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAutomationTriggersCreate],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDAutomationTriggersDelete],
			toolspkg.RiskDestructive,
			false,
			true,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDExtensionsSearch], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDExtensionsList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDExtensionsInfo], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDExtensionsInstall],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDExtensionsUpdate],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDExtensionsRemove],
			toolspkg.RiskDestructive,
			false,
			true,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDExtensionsEnable],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(
			t,
			descriptors[toolspkg.ToolIDExtensionsDisable],
			toolspkg.RiskMutating,
			false,
			false,
			false,
		)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMCPAuthStatus], toolspkg.RiskRead, true, false, false)
	})

	t.Run("Should keep network schemas closed and hard-cut vocabulary out of descriptors", func(t *testing.T) {
		t.Parallel()

		descriptors := descriptorMap(NativeDescriptors())
		networkIDs := []toolspkg.ToolID{
			toolspkg.ToolIDNetworkSend,
			toolspkg.ToolIDNetworkThreads,
			toolspkg.ToolIDNetworkThreadMessages,
			toolspkg.ToolIDNetworkDirects,
			toolspkg.ToolIDNetworkDirectResolve,
			toolspkg.ToolIDNetworkDirectMessages,
			toolspkg.ToolIDNetworkWork,
		}
		for _, id := range networkIDs {
			descriptor := descriptors[id]
			var schema map[string]json.RawMessage
			if err := json.Unmarshal(descriptor.InputSchema, &schema); err != nil {
				t.Fatalf("%s input schema is invalid JSON: %v", id, err)
			}
			var additionalProperties bool
			if err := json.Unmarshal(schema["additionalProperties"], &additionalProperties); err != nil {
				t.Fatalf("%s additionalProperties = %s: %v", id, schema["additionalProperties"], err)
			}
			if additionalProperties {
				t.Fatalf("%s additionalProperties = true, want false", id)
			}
			schemaText := string(descriptor.InputSchema)
			if strings.Contains(schemaText, "interaction_id") {
				t.Fatalf("%s schema includes deleted interaction_id field: %s", id, schemaText)
			}
			if strings.Contains(schemaText, `"kind":"direct"`) ||
				strings.Contains(descriptor.Description, `kind:"direct"`) {
				t.Fatalf("%s descriptor teaches legacy direct message kind", id)
			}
		}

		for _, id := range []toolspkg.ToolID{
			toolspkg.ToolIDNetworkSend,
			toolspkg.ToolIDNetworkDirects,
			toolspkg.ToolIDNetworkDirectResolve,
			toolspkg.ToolIDNetworkDirectMessages,
		} {
			description := strings.ToLower(descriptors[id].Description)
			if !strings.Contains(description, "runtime/audit access") ||
				!strings.Contains(description, "not cryptographic privacy") {
				t.Fatalf("%s description = %q, want explicit direct-room visibility boundary", id, description)
			}
			if strings.Contains(description, "encrypted") {
				t.Fatalf("%s description = %q, must not imply encrypted direct rooms", id, description)
			}
		}
	})

	t.Run("Should return cloned descriptors", func(t *testing.T) {
		t.Parallel()

		first := NativeDescriptors()
		first[0].ID = "agh__mutated"
		first[0].InputSchema[0] = '['

		second := NativeDescriptors()
		if second[0].ID == "agh__mutated" {
			t.Fatal("NativeDescriptors() reused descriptor slice")
		}
		if len(second[0].InputSchema) == 0 || second[0].InputSchema[0] == '[' {
			t.Fatal("NativeDescriptors() reused input schema bytes")
		}
	})
}

func TestBuiltinToolsetCatalog(t *testing.T) {
	t.Parallel()

	t.Run("Should expand built-in toolsets into canonical MVP tools", func(t *testing.T) {
		t.Parallel()

		descriptors := NativeDescriptors()
		universe := make([]toolspkg.ToolID, 0, len(descriptors))
		for _, descriptor := range descriptors {
			universe = append(universe, descriptor.ID)
		}
		catalog, err := ToolsetCatalog()
		if err != nil {
			t.Fatalf("ToolsetCatalog() error = %v", err)
		}

		bootstrap, err := catalog.Expand(toolspkg.ToolsetIDBootstrap, universe)
		if err != nil {
			t.Fatalf("Expand(bootstrap) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDToolInfo,
			toolspkg.ToolIDToolList,
			toolspkg.ToolIDToolSearch,
		}; !slices.Equal(
			bootstrap,
			want,
		) {
			t.Fatalf("bootstrap expansion = %#v, want %#v", bootstrap, want)
		}

		tasks, err := catalog.Expand(toolspkg.ToolsetIDTasks, universe)
		if err != nil {
			t.Fatalf("Expand(tasks) error = %v", err)
		}
		if !slices.Contains(tasks, toolspkg.ToolIDTaskChildCreate) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskRunReviewRequest) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskRunReviewList) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskRunReviewShow) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskExecutionProfileGet) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskExecutionProfileSet) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskExecutionProfileDelete) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskNotificationSubscribe) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskNotificationList) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskNotificationShow) ||
			!slices.Contains(tasks, toolspkg.ToolIDTaskNotificationDelete) ||
			slices.Contains(tasks, toolspkg.ToolIDTaskRunClaimNext) {
			t.Fatalf("task toolset expansion = %#v, want bounded task scope", tasks)
		}
		autonomy, err := catalog.Expand(toolspkg.ToolsetIDAutonomy, universe)
		if err != nil {
			t.Fatalf("Expand(autonomy) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDTaskRunClaimNext,
			toolspkg.ToolIDTaskRunComplete,
			toolspkg.ToolIDTaskRunFail,
			toolspkg.ToolIDTaskRunHeartbeat,
			toolspkg.ToolIDTaskRunRelease,
			toolspkg.ToolIDTaskRunReviewSubmit,
		}; !slices.Equal(autonomy, want) {
			t.Fatalf("autonomy expansion = %#v, want %#v", autonomy, want)
		}

		coordination, err := catalog.Expand(toolspkg.ToolsetIDCoordination, universe)
		if err != nil {
			t.Fatalf("Expand(coordination) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDNetworkChannels,
			toolspkg.ToolIDNetworkDirectMessages,
			toolspkg.ToolIDNetworkDirectResolve,
			toolspkg.ToolIDNetworkDirects,
			toolspkg.ToolIDNetworkInbox,
			toolspkg.ToolIDNetworkPeers,
			toolspkg.ToolIDNetworkSend,
			toolspkg.ToolIDNetworkStatus,
			toolspkg.ToolIDNetworkThreadMessages,
			toolspkg.ToolIDNetworkThreads,
			toolspkg.ToolIDNetworkWork,
		}; !slices.Equal(coordination, want) {
			t.Fatalf("coordination expansion = %#v, want %#v", coordination, want)
		}

		sessions, err := catalog.Expand(toolspkg.ToolsetIDSessions, universe)
		if err != nil {
			t.Fatalf("Expand(sessions) error = %v", err)
		}
		if !slices.Contains(sessions, toolspkg.ToolIDSessionList) ||
			!slices.Contains(sessions, toolspkg.ToolIDSessionDescribe) ||
			!slices.Contains(sessions, toolspkg.ToolIDSessionHealth) ||
			slices.Contains(sessions, toolspkg.ToolID("agh__session_stop")) {
			t.Fatalf("sessions toolset expansion = %#v, want read-only session tools", sessions)
		}

		authoredContext, err := catalog.Expand(toolspkg.ToolsetIDAuthoredContext, universe)
		if err != nil {
			t.Fatalf("Expand(authored_context) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDAgentHeartbeatStatus,
			toolspkg.ToolIDAgentHeartbeatWake,
			toolspkg.ToolIDSessionHealth,
		}; !slices.Equal(authoredContext, want) {
			t.Fatalf("authored context expansion = %#v, want %#v", authoredContext, want)
		}

		workspace, err := catalog.Expand(toolspkg.ToolsetIDWorkspace, universe)
		if err != nil {
			t.Fatalf("Expand(workspace) error = %v", err)
		}
		if !slices.Contains(workspace, toolspkg.ToolIDWorkspaceList) ||
			!slices.Contains(workspace, toolspkg.ToolIDWorkspaceDescribe) ||
			slices.Contains(workspace, toolspkg.ToolID("agh__workspace_remove")) {
			t.Fatalf("workspace toolset expansion = %#v, want read-only workspace tools", workspace)
		}

		providerModels, err := catalog.Expand(toolspkg.ToolsetIDProviderModels, universe)
		if err != nil {
			t.Fatalf("Expand(provider_models) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDProviderModelsList,
			toolspkg.ToolIDProviderModelsRefresh,
			toolspkg.ToolIDProviderModelsStatus,
		}; !slices.Equal(providerModels, want) {
			t.Fatalf("provider models expansion = %#v, want %#v", providerModels, want)
		}

		memory, err := catalog.Expand(toolspkg.ToolsetIDMemory, universe)
		if err != nil {
			t.Fatalf("Expand(memory) error = %v", err)
		}
		if !slices.Contains(memory, toolspkg.ToolIDMemoryShow) ||
			!slices.Contains(memory, toolspkg.ToolIDMemoryPropose) ||
			!slices.Contains(memory, toolspkg.ToolIDMemoryNote) ||
			slices.Contains(memory, toolspkg.ToolIDMemoryHealth) ||
			slices.Contains(memory, toolspkg.ToolIDMemoryReset) ||
			slices.Contains(memory, toolspkg.ToolID("agh__memory_read")) ||
			slices.Contains(memory, toolspkg.ToolID("agh__memory_history")) ||
			slices.Contains(memory, toolspkg.ToolID("agh__memory_write")) {
			t.Fatalf("memory toolset expansion = %#v, want Memory v2 Slice 1 tools", memory)
		}

		memoryAdmin, err := catalog.Expand(toolspkg.ToolsetIDMemoryAdmin, universe)
		if err != nil {
			t.Fatalf("Expand(memory_admin) error = %v", err)
		}
		if !slices.Contains(memoryAdmin, toolspkg.ToolIDMemoryHealth) ||
			!slices.Contains(memoryAdmin, toolspkg.ToolIDMemoryAdminHistory) ||
			!slices.Contains(memoryAdmin, toolspkg.ToolIDMemoryReset) ||
			!slices.Contains(memoryAdmin, toolspkg.ToolIDMemorySessionLedger) ||
			slices.Contains(memoryAdmin, toolspkg.ToolID("agh__memory_history")) ||
			slices.Contains(memoryAdmin, toolspkg.ToolIDMemoryPropose) {
			t.Fatalf("memory admin toolset expansion = %#v, want operational Memory v2 tools only", memoryAdmin)
		}

		observe, err := catalog.Expand(toolspkg.ToolsetIDObserve, universe)
		if err != nil {
			t.Fatalf("Expand(observe) error = %v", err)
		}
		if !slices.Contains(observe, toolspkg.ToolIDObserveEvents) ||
			!slices.Contains(observe, toolspkg.ToolIDObserveMetrics) ||
			slices.Contains(observe, toolspkg.ToolID("agh__observe_delete")) {
			t.Fatalf("observe toolset expansion = %#v, want read-only observe tools", observe)
		}

		bridges, err := catalog.Expand(toolspkg.ToolsetIDBridges, universe)
		if err != nil {
			t.Fatalf("Expand(bridges) error = %v", err)
		}
		if !slices.Contains(bridges, toolspkg.ToolIDBridgesList) ||
			!slices.Contains(bridges, toolspkg.ToolIDBridgesStatus) ||
			slices.Contains(bridges, toolspkg.ToolID("agh__bridges_update")) {
			t.Fatalf("bridges toolset expansion = %#v, want read-only bridge tools", bridges)
		}

		config, err := catalog.Expand(toolspkg.ToolsetIDConfig, universe)
		if err != nil {
			t.Fatalf("Expand(config) error = %v", err)
		}
		if !slices.Contains(config, toolspkg.ToolIDConfigSet) ||
			!slices.Contains(config, toolspkg.ToolIDConfigUnset) {
			t.Fatalf("config toolset expansion = %#v, want mutable config tools", config)
		}

		hooks, err := catalog.Expand(toolspkg.ToolsetIDHooks, universe)
		if err != nil {
			t.Fatalf("Expand(hooks) error = %v", err)
		}
		if !slices.Contains(hooks, toolspkg.ToolIDHooksCreate) ||
			!slices.Contains(hooks, toolspkg.ToolIDHooksDisable) {
			t.Fatalf("hooks toolset expansion = %#v, want mutable hook tools", hooks)
		}

		automation, err := catalog.Expand(toolspkg.ToolsetIDAutomation, universe)
		if err != nil {
			t.Fatalf("Expand(automation) error = %v", err)
		}
		if !slices.Contains(automation, toolspkg.ToolIDAutomationJobsCreate) ||
			!slices.Contains(automation, toolspkg.ToolIDAutomationRunsGet) ||
			slices.Contains(automation, toolspkg.ToolID("agh__automation_webhook_secret_set")) {
			t.Fatalf("automation toolset expansion = %#v, want bounded automation tools", automation)
		}

		extensions, err := catalog.Expand(toolspkg.ToolsetIDExtensions, universe)
		if err != nil {
			t.Fatalf("Expand(extensions) error = %v", err)
		}
		if !slices.Contains(extensions, toolspkg.ToolIDExtensionsInstall) ||
			!slices.Contains(extensions, toolspkg.ToolIDExtensionsRemove) ||
			slices.Contains(extensions, toolspkg.ToolID("agh__extensions_trust_root_set")) {
			t.Fatalf("extensions toolset expansion = %#v, want bounded extension lifecycle tools", extensions)
		}

		mcpAuth, err := catalog.Expand(toolspkg.ToolsetIDMCPAuth, universe)
		if err != nil {
			t.Fatalf("Expand(mcp_auth) error = %v", err)
		}
		if want := []toolspkg.ToolID{toolspkg.ToolIDMCPAuthStatus}; !slices.Equal(mcpAuth, want) {
			t.Fatalf("mcp auth expansion = %#v, want %#v", mcpAuth, want)
		}
	})
}

func descriptorMap(descriptors []toolspkg.Descriptor) map[toolspkg.ToolID]toolspkg.Descriptor {
	values := make(map[toolspkg.ToolID]toolspkg.Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		values[descriptor.ID] = descriptor
	}
	return values
}

func requireDescriptorRisk(
	t *testing.T,
	descriptor toolspkg.Descriptor,
	risk toolspkg.RiskClass,
	readOnly bool,
	destructive bool,
	openWorld bool,
) {
	t.Helper()

	if descriptor.Risk != risk ||
		descriptor.ReadOnly != readOnly ||
		descriptor.Destructive != destructive ||
		descriptor.OpenWorld != openWorld {
		t.Fatalf(
			"%s risk flags = (%s, read=%v, destructive=%v, open_world=%v), want (%s, read=%v, destructive=%v, open_world=%v)",
			descriptor.ID,
			descriptor.Risk,
			descriptor.ReadOnly,
			descriptor.Destructive,
			descriptor.OpenWorld,
			risk,
			readOnly,
			destructive,
			openWorld,
		)
	}
}
