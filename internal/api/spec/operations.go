package spec

import "sort"

// Operations returns the complete transport-neutral operation registry.
func Operations() []OperationSpec {
	ops := cloneOperationSpecs(operationRegistry)
	ops = append(ops, agentDefinitionMutationOperations()...)
	ops = append(ops, agentCatalogOperations()...)
	ops = append(ops, sessionTranscriptOperations()...)
	ops = append(ops, notificationPresetOperations()...)
	ops = append(ops, authoredContextOperations()...)
	ops = append(ops, append(loopsOperations(), goalOperations()...)...)
	ops = applyLoopAutomationContract(ops)
	ops = append(ops, modelCatalogOperations()...)
	ops = append(ops, providerOperations()...)
	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].Path == ops[j].Path {
			return ops[i].Method < ops[j].Method
		}
		return ops[i].Path < ops[j].Path
	})

	return ops
}
