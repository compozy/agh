package builtin

import (
	"encoding/json"

	toolspkg "github.com/pedronauck/agh/internal/tools"
)

const emptyInputSchema = `{"type":"object","additionalProperties":false}`

// NativeDescriptors returns the MVP native_go built-in descriptors.
func NativeDescriptors() []toolspkg.Descriptor {
	groups := [][]toolspkg.Descriptor{
		catalogDescriptors(),
		skillDescriptors(),
		networkDescriptors(),
		sessionDescriptors(),
		authoredContextDescriptors(),
		workspaceDescriptors(),
		providerModelsDescriptors(),
		memoryDescriptors(),
		memoryAdminDescriptors(),
		observeDescriptors(),
		bridgeDescriptors(),
		taskDescriptors(),
		autonomyDescriptors(),
		configDescriptors(),
		hookDescriptors(),
		automationDescriptors(),
		extensionDescriptors(),
		mcpAuthDescriptors(),
	}
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	descriptors := make([]toolspkg.Descriptor, 0, total)
	for _, group := range groups {
		for _, descriptor := range group {
			descriptors = append(descriptors, cloneDescriptor(descriptor))
		}
	}
	return descriptors
}

func nativeDescriptor(
	id toolspkg.ToolID,
	nativeName string,
	title string,
	description string,
	inputSchema string,
	risk toolspkg.RiskClass,
	readOnly bool,
	destructive bool,
	openWorld bool,
	toolsets []toolspkg.ToolsetID,
	tags []string,
	searchHints []string,
) toolspkg.Descriptor {
	return toolspkg.Descriptor{
		ID:              id,
		Backend:         toolspkg.BackendRef{Kind: toolspkg.BackendNativeGo, NativeName: nativeName},
		DisplayTitle:    title,
		Description:     description,
		InputSchema:     json.RawMessage(inputSchema),
		OutputSchema:    json.RawMessage(`{"type":"object"}`),
		Source:          Source(),
		Visibility:      toolspkg.VisibilityModel,
		Risk:            risk,
		ReadOnly:        readOnly,
		Destructive:     destructive,
		OpenWorld:       openWorld,
		ConcurrencySafe: true,
		Toolsets:        cloneToolsets(toolsets),
		Tags:            cloneStrings(tags),
		SearchHints:     cloneStrings(searchHints),
	}
}

func cloneDescriptor(src toolspkg.Descriptor) toolspkg.Descriptor {
	cloned := src
	cloned.InputSchema = cloneRawMessage(src.InputSchema)
	cloned.OutputSchema = cloneRawMessage(src.OutputSchema)
	cloned.Toolsets = cloneToolsets(src.Toolsets)
	cloned.Tags = cloneStrings(src.Tags)
	cloned.SearchHints = cloneStrings(src.SearchHints)
	return cloned
}

func cloneRawMessage(src json.RawMessage) json.RawMessage {
	if len(src) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), src...)
}

func cloneToolsets(src []toolspkg.ToolsetID) []toolspkg.ToolsetID {
	if len(src) == 0 {
		return nil
	}
	return append([]toolspkg.ToolsetID(nil), src...)
}

func cloneStrings(src []string) []string {
	if len(src) == 0 {
		return nil
	}
	return append([]string(nil), src...)
}
