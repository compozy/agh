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
		HostAPIMethodMemoryRecall,
		HostAPIMethodMemoryStore,
		HostAPIMethodMemoryForget,
		HostAPIMethodObserveHealth,
		HostAPIMethodObserveEvents,
		HostAPIMethodSkillsList,
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
		HostAPIMethodTasksCreate,
		HostAPIMethodTasksUpdate,
		HostAPIMethodTasksCancel,
		HostAPIMethodTasksRuns,
		HostAPIMethodTasksRunsEnqueue,
		HostAPIMethodTasksRunsClaim,
		HostAPIMethodTasksRunsStart,
		HostAPIMethodTasksRunsAttachSession,
		HostAPIMethodTasksRunsComplete,
		HostAPIMethodTasksRunsFail,
		HostAPIMethodTasksRunsCancel,
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
