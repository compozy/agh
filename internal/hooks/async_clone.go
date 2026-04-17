package hooks

import (
	"encoding/json"
	"reflect"
)

type asyncPayloadCloner[P any] interface {
	cloneForAsync() P
}

func cloneAsyncPayload[P any](payload P) P {
	cloner, ok := any(payload).(asyncPayloadCloner[P])
	if !ok {
		return payload
	}

	return cloner.cloneForAsync()
}

func (payload InputPreSubmitPayload) cloneForAsync() InputPreSubmitPayload {
	return cloneInputPreSubmitPayload(payload)
}

func (payload PromptPayload) cloneForAsync() PromptPayload {
	return clonePromptPayload(payload)
}

func (payload EventRecordPayload) cloneForAsync() EventRecordPayload {
	return cloneEventRecordPayload(payload)
}

func (payload EnvironmentPreparePayload) cloneForAsync() EnvironmentPreparePayload {
	return cloneEnvironmentPreparePayload(payload)
}

func (payload EnvironmentReadyPayload) cloneForAsync() EnvironmentReadyPayload {
	return cloneEnvironmentReadyPayload(payload)
}

func (payload EnvironmentSyncBeforePayload) cloneForAsync() EnvironmentSyncBeforePayload {
	return cloneEnvironmentSyncBeforePayload(payload)
}

func (payload EnvironmentSyncAfterPayload) cloneForAsync() EnvironmentSyncAfterPayload {
	return cloneEnvironmentSyncAfterPayload(payload)
}

func (payload AgentPreStartPayload) cloneForAsync() AgentPreStartPayload {
	return cloneAgentPreStartPayload(payload)
}

func (payload AgentLifecyclePayload) cloneForAsync() AgentLifecyclePayload {
	return cloneAgentLifecyclePayload(payload)
}

func (payload MessagePayload) cloneForAsync() MessagePayload {
	return cloneMessagePayload(payload)
}

func (payload ToolPreCallPayload) cloneForAsync() ToolPreCallPayload {
	return cloneToolPreCallPayload(payload)
}

func (payload ToolPostCallPayload) cloneForAsync() ToolPostCallPayload {
	return cloneToolPostCallPayload(payload)
}

func (payload ToolPostErrorPayload) cloneForAsync() ToolPostErrorPayload {
	return cloneToolPostErrorPayload(payload)
}

func (payload PermissionRequestPayload) cloneForAsync() PermissionRequestPayload {
	return clonePermissionRequestPayload(payload)
}

func (payload PermissionResolutionPayload) cloneForAsync() PermissionResolutionPayload {
	return clonePermissionResolutionPayload(payload)
}

func (payload ContextCompactPayload) cloneForAsync() ContextCompactPayload {
	return cloneContextCompactPayload(payload)
}

func (payload AutomationJobPreFirePayload) cloneForAsync() AutomationJobPreFirePayload {
	return cloneAutomationJobPreFirePayload(payload)
}

func (payload AutomationTriggerPreFirePayload) cloneForAsync() AutomationTriggerPreFirePayload {
	return cloneAutomationTriggerPreFirePayload(payload)
}

func cloneEnvironmentPreparePayload(payload EnvironmentPreparePayload) EnvironmentPreparePayload {
	payload.Profile = cloneEnvironmentProfilePayload(payload.Profile)
	payload.LocalAdditionalDirs = cloneStringSlice(payload.LocalAdditionalDirs)
	payload.AgentEnv = cloneStringSlice(payload.AgentEnv)
	payload.EnvOverrides = cloneStringMap(payload.EnvOverrides)
	return payload
}

func cloneEnvironmentReadyPayload(payload EnvironmentReadyPayload) EnvironmentReadyPayload {
	payload.RuntimeAdditionalDirs = cloneStringSlice(payload.RuntimeAdditionalDirs)
	return payload
}

func cloneEnvironmentSyncBeforePayload(payload EnvironmentSyncBeforePayload) EnvironmentSyncBeforePayload {
	payload.ExcludePatterns = cloneStringSlice(payload.ExcludePatterns)
	return payload
}

func cloneEnvironmentSyncAfterPayload(payload EnvironmentSyncAfterPayload) EnvironmentSyncAfterPayload {
	payload.Errors = cloneStringSlice(payload.Errors)
	return payload
}

func cloneInputPreSubmitPayload(payload InputPreSubmitPayload) InputPreSubmitPayload {
	payload.ContextBlocks = cloneContextBlocks(payload.ContextBlocks)
	return payload
}

func clonePromptPayload(payload PromptPayload) PromptPayload {
	payload.ContextBlocks = cloneContextBlocks(payload.ContextBlocks)
	return payload
}

func cloneEventRecordPayload(payload EventRecordPayload) EventRecordPayload {
	payload.Content = cloneRawJSON(payload.Content)
	return payload
}

func cloneAutomationJobPreFirePayload(payload AutomationJobPreFirePayload) AutomationJobPreFirePayload {
	payload.Schedule = cloneAutomationSchedulePayload(payload.Schedule)
	return payload
}

func cloneAutomationTriggerPreFirePayload(payload AutomationTriggerPreFirePayload) AutomationTriggerPreFirePayload {
	payload.Payload = cloneAnyMap(payload.Payload)
	return payload
}

func cloneAgentPreStartPayload(payload AgentPreStartPayload) AgentPreStartPayload {
	payload.Args = cloneStringSlice(payload.Args)
	return payload
}

func cloneAgentLifecyclePayload(payload AgentLifecyclePayload) AgentLifecyclePayload {
	payload.Args = cloneStringSlice(payload.Args)
	return payload
}

func cloneMessagePayload(payload MessagePayload) MessagePayload {
	payload.Raw = cloneRawJSON(payload.Raw)
	return payload
}

func cloneToolPreCallPayload(payload ToolPreCallPayload) ToolPreCallPayload {
	payload.ToolInput = cloneRawJSON(payload.ToolInput)
	return payload
}

func cloneToolPostCallPayload(payload ToolPostCallPayload) ToolPostCallPayload {
	payload.ToolInput = cloneRawJSON(payload.ToolInput)
	payload.ToolResult = cloneRawJSON(payload.ToolResult)
	return payload
}

func cloneToolPostErrorPayload(payload ToolPostErrorPayload) ToolPostErrorPayload {
	payload.ToolInput = cloneRawJSON(payload.ToolInput)
	return payload
}

func clonePermissionRequestPayload(payload PermissionRequestPayload) PermissionRequestPayload {
	payload.ToolInput = cloneRawJSON(payload.ToolInput)
	payload.ToolCall = clonePermissionToolCall(payload.ToolCall)
	payload.Options = clonePermissionOptions(payload.Options)
	return payload
}

func clonePermissionResolutionPayload(payload PermissionResolutionPayload) PermissionResolutionPayload {
	payload.ToolInput = cloneRawJSON(payload.ToolInput)
	payload.ToolCall = clonePermissionToolCall(payload.ToolCall)
	return payload
}

func cloneContextCompactPayload(payload ContextCompactPayload) ContextCompactPayload {
	payload.ContextBlocks = cloneContextBlocks(payload.ContextBlocks)
	return payload
}

func cloneEnvironmentProfilePayload(payload EnvironmentProfilePayload) EnvironmentProfilePayload {
	payload.Env = cloneStringMap(payload.Env)
	return payload
}

func cloneAutomationSchedulePayload(payload *AutomationSchedulePayload) *AutomationSchedulePayload {
	if payload == nil {
		return nil
	}

	cloned := *payload
	return &cloned
}

func clonePermissionToolCall(call PermissionToolCall) PermissionToolCall {
	call.Locations = cloneToolLocations(call.Locations)
	return call
}

func clonePermissionOptions(options []PermissionOption) []PermissionOption {
	if options == nil {
		return nil
	}

	cloned := make([]PermissionOption, len(options))
	copy(cloned, options)
	return cloned
}

func cloneToolLocations(locations []ToolLocation) []ToolLocation {
	if locations == nil {
		return nil
	}

	cloned := make([]ToolLocation, len(locations))
	copy(cloned, locations)
	return cloned
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}

	dst := make(map[string]any, len(src))
	for key, value := range src {
		dst[key] = cloneAnyValue(value)
	}
	return dst
}

func cloneAnySlice(values []any) []any {
	if values == nil {
		return nil
	}

	cloned := make([]any, len(values))
	for i, value := range values {
		cloned[i] = cloneAnyValue(value)
	}
	return cloned
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneAnyMap(typed)
	case []any:
		return cloneAnySlice(typed)
	case map[string]string:
		return cloneStringMap(typed)
	case []string:
		return cloneStringSlice(typed)
	case []byte:
		return cloneRawMessage(typed)
	case json.RawMessage:
		return cloneRawJSON(typed)
	default:
		if cloned, ok := cloneDynamicContainer(reflect.ValueOf(value)); ok {
			return cloned.Interface()
		}
		return value
	}
}

func cloneDynamicContainer(value reflect.Value) (reflect.Value, bool) {
	if !value.IsValid() {
		return reflect.Value{}, false
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type()), true
		}
		cloned, ok := cloneDynamicContainer(value.Elem())
		if !ok {
			cloned = value.Elem()
		}
		out := reflect.New(value.Type()).Elem()
		out.Set(cloned)
		return out, true
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(value.Type()), true
		}
		cloned, ok := cloneDynamicContainer(value.Elem())
		if !ok {
			return value, false
		}
		out := reflect.New(value.Type().Elem())
		out.Elem().Set(cloned)
		return out, true
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type()), true
		}
		out := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			mapValue := iter.Value()
			clonedValue, ok := cloneDynamicContainer(mapValue)
			if !ok {
				clonedValue = mapValue
			}
			out.SetMapIndex(iter.Key(), clonedValue)
		}
		return out, true
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type()), true
		}
		out := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			clonedItem, ok := cloneDynamicContainer(item)
			if !ok {
				clonedItem = item
			}
			out.Index(i).Set(clonedItem)
		}
		return out, true
	case reflect.Array:
		out := reflect.New(value.Type()).Elem()
		for i := 0; i < value.Len(); i++ {
			item := value.Index(i)
			clonedItem, ok := cloneDynamicContainer(item)
			if !ok {
				clonedItem = item
			}
			out.Index(i).Set(clonedItem)
		}
		return out, true
	default:
		return value, false
	}
}
