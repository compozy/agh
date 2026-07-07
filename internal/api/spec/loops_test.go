package spec

import (
	"slices"
	"sort"
	"strconv"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestLoopOpenAPIContract(t *testing.T) {
	t.Parallel()

	t.Run("Should expose every Loop route with expected status bodies", func(t *testing.T) {
		t.Parallel()

		doc, err := Document()
		if err != nil {
			t.Fatalf("Document() error = %v", err)
		}

		tests := []struct {
			name       string
			path       string
			method     string
			statuses   []int
			parameters []string
		}{
			{
				name:       "list catalog",
				path:       "/api/workspaces/{workspace_id}/loops",
				method:     "GET",
				statuses:   []int{200, 400, 503, 500},
				parameters: []string{"workspace_id"},
			},
			{
				name:       "create catalog entry",
				path:       "/api/workspaces/{workspace_id}/loops",
				method:     "POST",
				statuses:   []int{201, 400, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id"},
			},
			{
				name:       "inspect loop",
				path:       "/api/workspaces/{workspace_id}/loops/{name}",
				method:     "GET",
				statuses:   []int{200, 400, 404, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "publish loop",
				path:       "/api/workspaces/{workspace_id}/loops/{name}",
				method:     "PATCH",
				statuses:   []int{200, 400, 403, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "delete loop",
				path:       "/api/workspaces/{workspace_id}/loops/{name}",
				method:     "DELETE",
				statuses:   []int{204, 400, 403, 404, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "validate loop",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/validate",
				method:     "POST",
				statuses:   []int{200, 400, 422, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "run loop",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/run",
				method:     "POST",
				statuses:   []int{200, 201, 401, 400, 403, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "name", "dry"},
			},
			{
				name:       "get config",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/config",
				method:     "GET",
				statuses:   []int{200, 400, 404, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "put config",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/config",
				method:     "PUT",
				statuses:   []int{200, 400, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "get annotations",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/annotations",
				method:     "GET",
				statuses:   []int{200, 400, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "put annotations",
				path:       "/api/workspaces/{workspace_id}/loops/{name}/annotations",
				method:     "PUT",
				statuses:   []int{200, 400, 503, 500},
				parameters: []string{"workspace_id", "name"},
			},
			{
				name:       "list runs",
				path:       "/api/workspaces/{workspace_id}/loop-runs",
				method:     "GET",
				statuses:   []int{200, 400, 503, 500},
				parameters: []string{"workspace_id", "loop", "status", "limit"},
			},
			{
				name:       "get run",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}",
				method:     "GET",
				statuses:   []int{200, 400, 404, 503, 500},
				parameters: []string{"workspace_id", "run_id"},
			},
			{
				name:       "stop run",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}/stop",
				method:     "POST",
				statuses:   []int{200, 400, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "run_id"},
			},
			{
				name:       "pause run",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}/pause",
				method:     "POST",
				statuses:   []int{200, 400, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "run_id"},
			},
			{
				name:       "resume run",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}/resume",
				method:     "POST",
				statuses:   []int{200, 400, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "run_id"},
			},
			{
				name:       "approve run",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}/approve",
				method:     "POST",
				statuses:   []int{200, 400, 404, 409, 422, 503, 500},
				parameters: []string{"workspace_id", "run_id"},
			},
			{
				name:       "stream events",
				path:       "/api/workspaces/{workspace_id}/loop-runs/{run_id}/events",
				method:     "GET",
				statuses:   []int{200, 400, 404, 503, 500},
				parameters: []string{"workspace_id", "run_id", "after_sequence", "Last-Event-ID"},
			},
		}

		for _, tc := range tests {
			t.Run("Should describe "+tc.name, func(t *testing.T) {
				t.Parallel()

				operation := operationFor(t, doc, tc.path, tc.method)
				assertTagsContain(t, operation, specLoopsKey)
				assertLoopResponseStatusesExactly(t, operation, tc.statuses)
				for _, parameter := range tc.parameters {
					switch parameter {
					case "workspace_id", "name", "run_id":
						assertParameter(t, operation, parameter, openapi3.ParameterInPath, true)
					case "Last-Event-ID":
						assertParameter(t, operation, parameter, openapi3.ParameterInHeader, false)
					default:
						assertParameter(t, operation, parameter, openapi3.ParameterInQuery, false)
					}
				}
			})
		}

		stream := operationFor(t, doc, "/api/workspaces/{workspace_id}/loop-runs/{run_id}/events", "GET")
		_ = responseSchema(t, stream, 200, "text/event-stream")

		patchLoop := operationFor(t, doc, "/api/workspaces/{workspace_id}/loops/{name}", "PATCH")
		patchLintSchema := jsonResponseSchema(t, patchLoop, 422)
		assertRequired(t, patchLintSchema, "valid")
		_ = propertySchema(t, patchLintSchema, "errors")

		runLoop := operationFor(t, doc, "/api/workspaces/{workspace_id}/loops/{name}/run", "POST")
		assertRequired(t, jsonResponseSchema(t, runLoop, 422), "error")

		pauseRun := operationFor(t, doc, "/api/workspaces/{workspace_id}/loop-runs/{run_id}/pause", "POST")
		assertRequired(t, jsonResponseSchema(t, pauseRun, 422), "error")
	})

	t.Run("Should co-ship automation Loop target additions", func(t *testing.T) {
		t.Parallel()

		doc, err := Document()
		if err != nil {
			t.Fatalf("Document() error = %v", err)
		}

		for _, op := range []*openapi3.Operation{
			operationFor(t, doc, "/api/automation/jobs", "GET"),
			operationFor(t, doc, "/api/automation/triggers", "GET"),
		} {
			assertParameter(t, op, "loop", openapi3.ParameterInQuery, false)
		}

		for _, op := range []*openapi3.Operation{
			operationFor(t, doc, "/api/automation/jobs", "POST"),
			operationFor(t, doc, "/api/automation/jobs/{id}", "PATCH"),
			operationFor(t, doc, "/api/automation/triggers", "POST"),
			operationFor(t, doc, "/api/automation/triggers/{id}", "PATCH"),
		} {
			assertResponseStatus(t, op, 422)
		}

		createJobSchema := jsonRequestSchema(t, operationFor(t, doc, "/api/automation/jobs", "POST"))
		assertNotRequired(t, createJobSchema, "target_kind", "loop_target")
		_ = propertySchema(t, createJobSchema, "target_kind")
		loopTargetSchema := propertySchema(t, createJobSchema, "loop_target")
		assertRequired(t, loopTargetSchema, "workspace_id", "loop_name")

		updateTriggerSchema := jsonRequestSchema(t, operationFor(t, doc, "/api/automation/triggers/{id}", "PATCH"))
		assertNotRequired(t, updateTriggerSchema, "target_kind", "loop_target")
	})
}

func assertLoopResponseStatusesExactly(t *testing.T, operation *openapi3.Operation, statuses []int) {
	t.Helper()

	want := make([]string, 0, len(statuses))
	for _, status := range statuses {
		want = append(want, strconv.Itoa(status))
	}
	sort.Strings(want)
	got := operation.Responses.Keys()
	sort.Strings(got)
	if !slices.Equal(got, want) {
		t.Fatalf("response statuses = %v, want %v", got, want)
	}
}
