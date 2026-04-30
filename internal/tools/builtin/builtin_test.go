package builtin

import (
	"slices"
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
			toolspkg.ToolIDSessionList,
			toolspkg.ToolIDSessionStatus,
			toolspkg.ToolIDSessionHistory,
			toolspkg.ToolIDSessionEvents,
			toolspkg.ToolIDSessionDescribe,
			toolspkg.ToolIDWorkspaceList,
			toolspkg.ToolIDWorkspaceInfo,
			toolspkg.ToolIDWorkspaceDescribe,
			toolspkg.ToolIDMemoryList,
			toolspkg.ToolIDMemoryRead,
			toolspkg.ToolIDMemorySearch,
			toolspkg.ToolIDMemoryHistory,
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
			"agh__task_run_complete",
			"agh__task_run_cancel",
		}
		for _, id := range excluded {
			if _, ok := got[id]; ok {
				t.Fatalf("descriptor %q is registered but must be excluded from MVP native scope", id)
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
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionStatus], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionHistory], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionEvents], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDSessionDescribe], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceInfo], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDWorkspaceDescribe], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryRead], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemorySearch], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDMemoryHistory], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveEvents], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveMetrics], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDObserveSearch], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDBridgesList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDBridgesStatus], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRead], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDTaskRunList], toolspkg.RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[toolspkg.ToolIDNetworkSend], toolspkg.RiskOpenWorld, false, false, true)
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
			slices.Contains(tasks, toolspkg.ToolID("agh__task_claim")) {
			t.Fatalf("task toolset expansion = %#v, want bounded task scope", tasks)
		}

		coordination, err := catalog.Expand(toolspkg.ToolsetIDCoordination, universe)
		if err != nil {
			t.Fatalf("Expand(coordination) error = %v", err)
		}
		if want := []toolspkg.ToolID{
			toolspkg.ToolIDNetworkChannels,
			toolspkg.ToolIDNetworkInbox,
			toolspkg.ToolIDNetworkPeers,
			toolspkg.ToolIDNetworkSend,
			toolspkg.ToolIDNetworkStatus,
		}; !slices.Equal(coordination, want) {
			t.Fatalf("coordination expansion = %#v, want %#v", coordination, want)
		}

		sessions, err := catalog.Expand(toolspkg.ToolsetIDSessions, universe)
		if err != nil {
			t.Fatalf("Expand(sessions) error = %v", err)
		}
		if !slices.Contains(sessions, toolspkg.ToolIDSessionList) ||
			!slices.Contains(sessions, toolspkg.ToolIDSessionDescribe) ||
			slices.Contains(sessions, toolspkg.ToolID("agh__session_stop")) {
			t.Fatalf("sessions toolset expansion = %#v, want read-only session tools", sessions)
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

		memory, err := catalog.Expand(toolspkg.ToolsetIDMemory, universe)
		if err != nil {
			t.Fatalf("Expand(memory) error = %v", err)
		}
		if !slices.Contains(memory, toolspkg.ToolIDMemoryRead) ||
			!slices.Contains(memory, toolspkg.ToolIDMemoryHistory) ||
			slices.Contains(memory, toolspkg.ToolID("agh__memory_write")) {
			t.Fatalf("memory toolset expansion = %#v, want read-only memory tools", memory)
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
