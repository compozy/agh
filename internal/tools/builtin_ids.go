package tools

const (
	// BuiltinSourceOwner is the source owner for daemon-compiled AGH tools.
	BuiltinSourceOwner = "daemon"
)

const (
	// ToolIDToolList lists tools in the caller's effective registry projection.
	ToolIDToolList ToolID = "agh__tool_list"
	// ToolIDToolSearch searches tools in the caller's effective registry projection.
	ToolIDToolSearch ToolID = "agh__tool_search"
	// ToolIDToolInfo reads one tool descriptor and diagnostics view.
	ToolIDToolInfo ToolID = "agh__tool_info"
	// ToolIDSkillList lists skills through the existing skill registry.
	ToolIDSkillList ToolID = "agh__skill_list"
	// ToolIDSkillSearch searches skills through the existing skill registry.
	ToolIDSkillSearch ToolID = "agh__skill_search"
	// ToolIDSkillView reads one skill and its verified body.
	ToolIDSkillView ToolID = "agh__skill_view"
	// ToolIDNetworkPeers lists visible network peers.
	ToolIDNetworkPeers ToolID = "agh__network_peers"
	// ToolIDNetworkSend sends one network message through the existing network manager.
	ToolIDNetworkSend ToolID = "agh__network_send"
	// ToolIDTaskList lists task summaries through the task service.
	ToolIDTaskList ToolID = "agh__task_list"
	// ToolIDTaskRead reads one task view through the task service.
	ToolIDTaskRead ToolID = "agh__task_read"
	// ToolIDTaskCreate creates one root task through the task service.
	ToolIDTaskCreate ToolID = "agh__task_create"
	// ToolIDTaskChildCreate creates one child task through the task service.
	ToolIDTaskChildCreate ToolID = "agh__task_child_create"
	// ToolIDTaskUpdate updates one task through the task service.
	ToolIDTaskUpdate ToolID = "agh__task_update"
	// ToolIDTaskCancel cancels one task through the task service.
	ToolIDTaskCancel ToolID = "agh__task_cancel"
	// ToolIDTaskRunList lists task runs through the task service.
	ToolIDTaskRunList ToolID = "agh__task_run_list"
)

const (
	// ToolsetIDBootstrap groups registry self-inspection tools.
	ToolsetIDBootstrap ToolsetID = "agh__bootstrap"
	// ToolsetIDCatalog groups registry and skill catalog tools.
	ToolsetIDCatalog ToolsetID = "agh__catalog"
	// ToolsetIDCoordination groups network coordination tools.
	ToolsetIDCoordination ToolsetID = "agh__coordination"
	// ToolsetIDTasks groups bounded task tools.
	ToolsetIDTasks ToolsetID = "agh__tasks"
)

// BuiltinSource returns the provenance shared by daemon-compiled AGH tools.
func BuiltinSource() SourceRef {
	return SourceRef{
		Kind:  SourceBuiltin,
		Owner: BuiltinSourceOwner,
		Scope: "daemon",
	}
}
