//go:build integration && !windows

package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/compozy/agh/internal/agentidentity"
	aghcontract "github.com/compozy/agh/internal/api/contract"
	e2etest "github.com/compozy/agh/internal/testutil/e2e"
)

func TestDaemonE2EAgentDefinitionLifecycleParity(t *testing.T) {
	t.Parallel()

	t.Run("Should preserve lifecycle parity across CLI HTTP and restart", func(t *testing.T) {
		// not parallel: lifecycle steps share one ordered runtime state.
		runDaemonE2EAgentDefinitionLifecycleParity(t)
	})
}

func runDaemonE2EAgentDefinitionLifecycleParity(t *testing.T) {
	t.Helper()

	harness := e2etest.StartRuntimeHarness(t, &e2etest.RuntimeHarnessOptions{})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	const sourceName = "parity-coder"
	var cliCreated aghcontract.AgentPayload
	if err := harness.CLI.RunJSONInDir(
		ctx,
		harness.WorkspaceRoot,
		&cliCreated,
		"agent",
		"create",
		sourceName,
		"--workspace",
		harness.WorkspaceRoot,
		"--provider",
		"fake",
		"--model",
		"model-v1",
		"--tool",
		"builtin__shell",
		"--prompt",
		"Write code.",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI agent create error = %v", err)
	}

	t.Run("Should expose a CLI create through HTTP", func(t *testing.T) {
		// not parallel: lifecycle steps share one ordered runtime state.
		var response aghcontract.AgentResponse
		if err := harness.HTTPJSON(
			ctx,
			http.MethodGet,
			agentDefinitionE2EPath(sourceName, harness.WorkspaceRoot),
			nil,
			&response,
		); err != nil {
			t.Fatalf("HTTP get CLI-created agent error = %v", err)
		}
		if !reflect.DeepEqual(agentDefinitionE2EProjection(response.Agent), agentDefinitionE2EProjection(cliCreated)) {
			t.Fatalf("HTTP agent = %#v, want CLI parity %#v", response.Agent, cliCreated)
		}
	})

	updateRequest := aghcontract.UpdateAgentRequest{
		Workspace: harness.WorkspaceRoot,
		Agent: aghcontract.CreateAgentPayload{
			Name: sourceName, Provider: "fake", Model: "model-v2",
			Tools: []string{"builtin__shell"}, Prompt: "Review code.",
		},
		ExpectedDigest: cliCreated.DefinitionDigest,
	}
	var httpUpdated aghcontract.AgentResponse
	if err := harness.HTTPJSON(
		ctx,
		http.MethodPut,
		"/api/agents/"+url.PathEscape(sourceName),
		updateRequest,
		&httpUpdated,
	); err != nil {
		t.Fatalf("HTTP agent update error = %v", err)
	}

	t.Run("Should expose an HTTP update through CLI info", func(t *testing.T) {
		// not parallel: lifecycle steps share one ordered runtime state.
		var cliInfo aghcontract.AgentPayload
		if err := harness.CLI.RunJSONInDir(
			ctx,
			harness.WorkspaceRoot,
			&cliInfo,
			"agent",
			"info",
			sourceName,
			"--workspace",
			harness.WorkspaceRoot,
			"-o",
			"json",
		); err != nil {
			t.Fatalf("CLI agent info error = %v", err)
		}
		if !reflect.DeepEqual(agentDefinitionE2EProjection(cliInfo), agentDefinitionE2EProjection(httpUpdated.Agent)) {
			t.Fatalf("CLI info = %#v, want HTTP parity %#v", cliInfo, httpUpdated.Agent)
		}
	})

	const duplicateName = "parity-reviewer"
	duplicateRequest := aghcontract.DuplicateAgentRequest{
		Name:      duplicateName,
		Workspace: harness.WorkspaceRoot,
		Overrides: &aghcontract.DuplicateAgentOverrides{Model: "model-review"},
	}
	var udsDuplicated aghcontract.AgentResponse
	if err := harness.UDSJSON(
		ctx,
		http.MethodPost,
		"/api/agents/"+url.PathEscape(sourceName)+"/duplicate",
		duplicateRequest,
		&udsDuplicated,
	); err != nil {
		t.Fatalf("UDS agent duplicate error = %v", err)
	}

	t.Run("Should expose an identical UDS duplicate through every read surface", func(t *testing.T) {
		// not parallel: lifecycle steps share one ordered runtime state.
		var httpInfo aghcontract.AgentResponse
		if err := harness.HTTPJSON(
			ctx,
			http.MethodGet,
			agentDefinitionE2EPath(duplicateName, harness.WorkspaceRoot),
			nil,
			&httpInfo,
		); err != nil {
			t.Fatalf("HTTP duplicated agent info error = %v", err)
		}
		var udsInfo aghcontract.AgentResponse
		if err := harness.UDSJSON(
			ctx,
			http.MethodGet,
			agentDefinitionE2EPath(duplicateName, harness.WorkspaceRoot),
			nil,
			&udsInfo,
		); err != nil {
			t.Fatalf("UDS duplicated agent info error = %v", err)
		}
		var cliInfo aghcontract.AgentPayload
		if err := harness.CLI.RunJSONInDir(
			ctx,
			harness.WorkspaceRoot,
			&cliInfo,
			"agent",
			"info",
			duplicateName,
			"--workspace",
			harness.WorkspaceRoot,
			"-o",
			"json",
		); err != nil {
			t.Fatalf("CLI duplicated agent info error = %v", err)
		}
		want := agentDefinitionE2EProjection(udsDuplicated.Agent)
		for surface, got := range map[string]agentDefinitionE2EView{
			"HTTP": agentDefinitionE2EProjection(httpInfo.Agent),
			"UDS":  agentDefinitionE2EProjection(udsInfo.Agent),
			"CLI":  agentDefinitionE2EProjection(cliInfo),
		} {
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s duplicate = %#v, want %#v", surface, got, want)
			}
		}
	})

	conflictPath := "/api/agents/" + url.PathEscape(sourceName) + "/duplicate"
	httpConflict := agentDefinitionE2ERequestError(
		t,
		ctx,
		harness.HTTPClient,
		harness.HTTPURL(conflictPath),
		http.MethodPost,
		duplicateRequest,
	)
	udsConflict := agentDefinitionE2ERequestError(
		t,
		ctx,
		harness.UDSClient,
		harness.UDSURL(conflictPath),
		http.MethodPost,
		duplicateRequest,
	)
	if httpConflict.Status != http.StatusConflict || udsConflict.Status != http.StatusConflict ||
		httpConflict.Payload.Error != udsConflict.Payload.Error {
		t.Fatalf("duplicate conflicts = HTTP %#v / UDS %#v, want identical 409", httpConflict, udsConflict)
	}
	_, cliConflictStderr, cliConflictErr := harness.CLI.RunInDir(
		ctx,
		harness.WorkspaceRoot,
		"agent",
		"duplicate",
		sourceName,
		duplicateName,
		"--workspace",
		harness.WorkspaceRoot,
		"-o",
		"json",
	)
	assertAgentDefinitionE2ECLIError(
		t,
		cliConflictErr,
		cliConflictStderr,
		agentidentity.ExitIdentityInvalid,
		httpConflict.Payload.Error,
	)

	var deleted aghcontract.DeleteAgentResponse
	if err := harness.CLI.RunJSONInDir(
		ctx,
		harness.WorkspaceRoot,
		&deleted,
		"agent",
		"delete",
		duplicateName,
		"--workspace",
		harness.WorkspaceRoot,
		"--yes",
		"-o",
		"json",
	); err != nil {
		t.Fatalf("CLI agent delete error = %v", err)
	}
	if deleted.Name != duplicateName || deleted.Origin != aghcontract.AgentOriginWorkspace {
		t.Fatalf("CLI delete = %#v, want workspace duplicate", deleted)
	}

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := harness.Stop(stopCtx); err != nil {
		stopCancel()
		t.Fatalf("stop runtime before restart error = %v", err)
	}
	stopCancel()

	restarted := e2etest.StartRuntimeHarness(t, &e2etest.RuntimeHarnessOptions{
		BinaryPath: harness.BinaryPath,
		HomePaths:  harness.HomePaths,
		Workspace:  e2etest.WorkspaceSeedOptions{Root: harness.WorkspaceRoot},
	})

	notFoundPath := agentDefinitionE2EPath(duplicateName, restarted.WorkspaceRoot)
	httpNotFound := agentDefinitionE2ERequestError(
		t,
		ctx,
		restarted.HTTPClient,
		restarted.HTTPURL(notFoundPath),
		http.MethodGet,
		nil,
	)
	udsNotFound := agentDefinitionE2ERequestError(
		t,
		ctx,
		restarted.UDSClient,
		restarted.UDSURL(notFoundPath),
		http.MethodGet,
		nil,
	)
	if httpNotFound.Status != http.StatusNotFound || udsNotFound.Status != http.StatusNotFound ||
		httpNotFound.Payload.Error != udsNotFound.Payload.Error {
		t.Fatalf("post-restart not found = HTTP %#v / UDS %#v, want identical 404", httpNotFound, udsNotFound)
	}
	_, cliNotFoundStderr, cliNotFoundErr := restarted.CLI.RunInDir(
		ctx,
		restarted.WorkspaceRoot,
		"agent",
		"info",
		duplicateName,
		"--workspace",
		restarted.WorkspaceRoot,
		"-o",
		"json",
	)
	assertAgentDefinitionE2ECLIError(
		t,
		cliNotFoundErr,
		cliNotFoundStderr,
		agentidentity.ExitUnavailable,
		httpNotFound.Payload.Error,
	)
}

type agentDefinitionE2EView struct {
	Name             string
	Provider         string
	Model            string
	Prompt           string
	Origin           aghcontract.AgentOrigin
	WorkspaceID      string
	DefinitionDigest string
	Tools            []string
}

func agentDefinitionE2EProjection(agent aghcontract.AgentPayload) agentDefinitionE2EView {
	return agentDefinitionE2EView{
		Name:             agent.Name,
		Provider:         agent.Provider,
		Model:            agent.Model,
		Prompt:           agent.Prompt,
		Origin:           agent.Origin,
		WorkspaceID:      agent.WorkspaceID,
		DefinitionDigest: agent.DefinitionDigest,
		Tools:            append([]string(nil), agent.Tools...),
	}
}
func agentDefinitionE2EPath(name string, workspace string) string {
	return "/api/agents/" + url.PathEscape(name) + "?workspace=" + url.QueryEscape(workspace)
}

type agentDefinitionE2EError struct {
	Status  int
	Payload aghcontract.ErrorPayload
}

func agentDefinitionE2ERequestError(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	target string,
	method string,
	body any,
) agentDefinitionE2EError {
	t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("json.Marshal(%s %s body) error = %v", method, target, err)
		}
		reader = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		t.Fatalf("http.NewRequestWithContext(%s %s) error = %v", method, target, err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("HTTP client %s %s error = %v", method, target, err)
	}
	payload, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err := errors.Join(readErr, closeErr); err != nil {
		t.Fatalf("read/close %s %s response error = %v", method, target, err)
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		t.Fatalf("%s %s status = %d, want error; body=%s", method, target, response.StatusCode, payload)
	}
	var errorPayload aghcontract.ErrorPayload
	if err := json.Unmarshal(payload, &errorPayload); err != nil {
		t.Fatalf("json.Unmarshal(%s %s error) = %v; body=%s", method, target, err, payload)
	}
	return agentDefinitionE2EError{Status: response.StatusCode, Payload: errorPayload}
}

func assertAgentDefinitionE2ECLIError(
	t *testing.T,
	err error,
	stderr string,
	wantExit int,
	wantMessage string,
) {
	t.Helper()

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("CLI error = %T %[1]v, want *exec.ExitError", err)
	}
	if exitErr.ExitCode() != wantExit {
		t.Fatalf("CLI exit code = %d, want %d; stderr=%s", exitErr.ExitCode(), wantExit, stderr)
	}
	var payload aghcontract.ErrorPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(stderr)), &payload); err != nil {
		t.Fatalf("json.Unmarshal(CLI stderr) = %v; stderr=%s", err, stderr)
	}
	if payload.Error != wantMessage {
		t.Fatalf("CLI error = %q, want %q", payload.Error, wantMessage)
	}
}
