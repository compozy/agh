package tools

import (
	"slices"
	"testing"
)

func TestBuiltinNativeDescriptors(t *testing.T) {
	t.Parallel()

	t.Run("Should expose exactly the MVP native tool scope", func(t *testing.T) {
		t.Parallel()

		descriptors := BuiltinNativeDescriptors()
		got := make(map[ToolID]Descriptor, len(descriptors))
		for _, descriptor := range descriptors {
			if err := descriptor.Validate(); err != nil {
				t.Fatalf("descriptor %q Validate() error = %v", descriptor.ID, err)
			}
			got[descriptor.ID] = descriptor
		}

		want := []ToolID{
			ToolIDToolList,
			ToolIDToolSearch,
			ToolIDToolInfo,
			ToolIDSkillList,
			ToolIDSkillSearch,
			ToolIDSkillView,
			ToolIDNetworkPeers,
			ToolIDNetworkSend,
			ToolIDTaskList,
			ToolIDTaskRead,
			ToolIDTaskCreate,
			ToolIDTaskChildCreate,
			ToolIDTaskUpdate,
			ToolIDTaskCancel,
			ToolIDTaskRunList,
		}
		if gotLen, wantLen := len(got), len(want); gotLen != wantLen {
			t.Fatalf("len(BuiltinNativeDescriptors()) = %d, want %d", gotLen, wantLen)
		}
		for _, id := range want {
			descriptor, ok := got[id]
			if !ok {
				t.Fatalf("descriptor %q missing from MVP native scope", id)
			}
			if descriptor.Backend.Kind != BackendNativeGo {
				t.Fatalf("%s backend kind = %q, want native_go", id, descriptor.Backend.Kind)
			}
			if descriptor.Backend.NativeName == "" {
				t.Fatalf("%s backend native name is empty", id)
			}
			if descriptor.Source != BuiltinSource() {
				t.Fatalf("%s source = %#v, want builtin source", id, descriptor.Source)
			}
			if descriptor.Visibility != VisibilityModel {
				t.Fatalf("%s visibility = %q, want model", id, descriptor.Visibility)
			}
		}

		excluded := []ToolID{
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

		descriptors := descriptorMap(BuiltinNativeDescriptors())

		requireDescriptorRisk(t, descriptors[ToolIDToolList], RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDSkillView], RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDNetworkPeers], RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDTaskRead], RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDTaskRunList], RiskRead, true, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDNetworkSend], RiskOpenWorld, false, false, true)
		requireDescriptorRisk(t, descriptors[ToolIDTaskCreate], RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDTaskChildCreate], RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDTaskUpdate], RiskMutating, false, false, false)
		requireDescriptorRisk(t, descriptors[ToolIDTaskCancel], RiskDestructive, false, true, false)
	})

	t.Run("Should return cloned descriptors", func(t *testing.T) {
		t.Parallel()

		first := BuiltinNativeDescriptors()
		first[0].ID = "agh__mutated"
		first[0].InputSchema[0] = '['

		second := BuiltinNativeDescriptors()
		if second[0].ID == "agh__mutated" {
			t.Fatal("BuiltinNativeDescriptors() reused descriptor slice")
		}
		if len(second[0].InputSchema) == 0 || second[0].InputSchema[0] == '[' {
			t.Fatal("BuiltinNativeDescriptors() reused input schema bytes")
		}
	})
}

func TestBuiltinToolsetCatalog(t *testing.T) {
	t.Parallel()

	t.Run("Should expand built-in toolsets into canonical MVP tools", func(t *testing.T) {
		t.Parallel()

		descriptors := BuiltinNativeDescriptors()
		universe := make([]ToolID, 0, len(descriptors))
		for _, descriptor := range descriptors {
			universe = append(universe, descriptor.ID)
		}
		catalog, err := BuiltinToolsetCatalog()
		if err != nil {
			t.Fatalf("BuiltinToolsetCatalog() error = %v", err)
		}

		bootstrap, err := catalog.Expand(ToolsetIDBootstrap, universe)
		if err != nil {
			t.Fatalf("Expand(bootstrap) error = %v", err)
		}
		if want := []ToolID{ToolIDToolInfo, ToolIDToolList, ToolIDToolSearch}; !slices.Equal(bootstrap, want) {
			t.Fatalf("bootstrap expansion = %#v, want %#v", bootstrap, want)
		}

		tasks, err := catalog.Expand(ToolsetIDTasks, universe)
		if err != nil {
			t.Fatalf("Expand(tasks) error = %v", err)
		}
		if !slices.Contains(tasks, ToolIDTaskChildCreate) || slices.Contains(tasks, ToolID("agh__task_claim")) {
			t.Fatalf("task toolset expansion = %#v, want bounded task scope", tasks)
		}
	})
}

func descriptorMap(descriptors []Descriptor) map[ToolID]Descriptor {
	values := make(map[ToolID]Descriptor, len(descriptors))
	for _, descriptor := range descriptors {
		values[descriptor.ID] = descriptor
	}
	return values
}

func requireDescriptorRisk(
	t *testing.T,
	descriptor Descriptor,
	risk RiskClass,
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
