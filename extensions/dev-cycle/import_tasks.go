package devcycle

import (
	"fmt"
	"strings"

	looppkg "github.com/compozy/agh/internal/loop"
)

type importTasksInput struct {
	Pattern string `json:"pattern"`
}

type importTasksOutput struct {
	Tasks []markdownTaskPayload `json:"tasks"`
	Count int                   `json:"count"`
}

func importTasks(input importTasksInput) (importTasksOutput, error) {
	pattern := strings.TrimSpace(input.Pattern)
	if pattern == "" {
		return importTasksOutput{}, fmt.Errorf("%w: import_tasks pattern is required", looppkg.ErrValidation)
	}
	result, err := importMarkdownTasks(pattern)
	if err != nil {
		return importTasksOutput{}, err
	}
	return importTasksOutput{
		Tasks: result.Tasks,
		Count: len(result.Tasks),
	}, nil
}
