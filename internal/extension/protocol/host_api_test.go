package protocol

import "testing"

func TestAllHostAPIMethodsReturnsCanonicalWireOrder(t *testing.T) {
	t.Parallel()

	want := []HostAPIMethod{
		HostAPIMethodSessionsList,
		HostAPIMethodSessionsCreate,
		HostAPIMethodSessionsPrompt,
		HostAPIMethodSessionsStop,
		HostAPIMethodSessionsStatus,
		HostAPIMethodSessionsEvents,
		HostAPIMethodSessionsSoulRefresh,
		HostAPIMethodSessionsHealthGet,
		HostAPIMethodSessionsStatusGet,
		HostAPIMethodSandboxList,
		HostAPIMethodSandboxInfo,
		HostAPIMethodSandboxExec,
		HostAPIMethodMemoryRecall,
		HostAPIMethodMemoryStore,
		HostAPIMethodMemoryForget,
		HostAPIMethodObserveHealth,
		HostAPIMethodObserveEvents,
		HostAPIMethodSkillsList,
		HostAPIMethodAgentsSoulGet,
		HostAPIMethodAgentsSoulValidate,
		HostAPIMethodAgentsSoulPut,
		HostAPIMethodAgentsSoulDelete,
		HostAPIMethodAgentsSoulHistory,
		HostAPIMethodAgentsSoulRollback,
		HostAPIMethodAgentsHeartbeatGet,
		HostAPIMethodAgentsHeartbeatValidate,
		HostAPIMethodAgentsHeartbeatPut,
		HostAPIMethodAgentsHeartbeatDelete,
		HostAPIMethodAgentsHeartbeatHistory,
		HostAPIMethodAgentsHeartbeatRollback,
		HostAPIMethodAgentsHeartbeatStatus,
		HostAPIMethodAgentsHeartbeatWake,
		HostAPIMethodAutomationJobs,
		HostAPIMethodAutomationJobsGet,
		HostAPIMethodAutomationJobsCreate,
		HostAPIMethodAutomationJobsUpdate,
		HostAPIMethodAutomationJobsDelete,
		HostAPIMethodAutomationJobsTrigger,
		HostAPIMethodAutomationJobsRuns,
		HostAPIMethodAutomationTriggers,
		HostAPIMethodAutomationTriggersGet,
		HostAPIMethodAutomationTriggersCreate,
		HostAPIMethodAutomationTriggersUpdate,
		HostAPIMethodAutomationTriggersDelete,
		HostAPIMethodAutomationTriggersRuns,
		HostAPIMethodAutomationTriggersFire,
		HostAPIMethodAutomationRuns,
		HostAPIMethodTasks,
		HostAPIMethodTasksGet,
		HostAPIMethodTasksTimeline,
		HostAPIMethodTasksTree,
		HostAPIMethodTasksDashboard,
		HostAPIMethodTasksInbox,
		HostAPIMethodTasksCreate,
		HostAPIMethodTasksUpdate,
		HostAPIMethodTasksCancel,
		HostAPIMethodTasksRuns,
		HostAPIMethodTasksRunsGet,
		HostAPIMethodTasksRunsEnqueue,
		HostAPIMethodTasksRunsClaim,
		HostAPIMethodTasksRunsStart,
		HostAPIMethodTasksRunsAttachSession,
		HostAPIMethodTasksRunsComplete,
		HostAPIMethodTasksRunsFail,
		HostAPIMethodTasksRunsCancel,
		HostAPIMethodResourcesList,
		HostAPIMethodResourcesGet,
		HostAPIMethodResourcesSnapshot,
		HostAPIMethodBridgesInstancesList,
		HostAPIMethodBridgesMessagesIngest,
		HostAPIMethodBridgesInstancesGet,
		HostAPIMethodBridgesInstancesReportState,
	}

	got := AllHostAPIMethods()
	if len(got) != len(want) {
		t.Fatalf("len(AllHostAPIMethods()) = %d, want %d", len(got), len(want))
	}
	for idx := range want {
		if got[idx] != want[idx] {
			t.Fatalf("AllHostAPIMethods()[%d] = %q, want %q", idx, got[idx], want[idx])
		}
	}
}
