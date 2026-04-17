package acpmock

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kballard/go-shellquote"
	aghconfig "github.com/pedronauck/agh/internal/config"
)

// RegisterOptions describes one temporary fixture-backed AGENT.md registration.
type RegisterOptions struct {
	FixturePath     string
	FixtureAgent    string
	AgentName       string
	NodePath        string
	DriverPath      string
	DiagnosticsPath string
}

// Registration captures one temporary mock-agent definition written into AGH home.
type Registration struct {
	AgentName       string
	FixtureAgent    string
	FixturePath     string
	NodePath        string
	DriverPath      string
	DiagnosticsPath string
	AgentDefPath    string
	Command         string
	Provider        string
	Model           string
	Permissions     string
}

// Register writes one temporary fixture-backed AGENT.md file into the supplied AGH home.
func Register(homePaths aghconfig.HomePaths, opts RegisterOptions) (Registration, error) {
	if strings.TrimSpace(homePaths.AgentsDir) == "" {
		return Registration{}, errors.New("acpmock: home paths agents directory is required")
	}

	fixturePath, err := aghconfig.ResolvePath(opts.FixturePath)
	if err != nil {
		return Registration{}, fmt.Errorf("acpmock: resolve fixture path: %w", err)
	}
	fixture, err := LoadFixture(fixturePath)
	if err != nil {
		return Registration{}, err
	}

	fixtureAgentName := strings.TrimSpace(opts.FixtureAgent)
	if fixtureAgentName == "" {
		fixtureAgentName = strings.TrimSpace(opts.AgentName)
	}
	if fixtureAgentName == "" {
		return Registration{}, errors.New("acpmock: fixture agent name is required")
	}
	agent, err := fixture.Agent(fixtureAgentName)
	if err != nil {
		return Registration{}, err
	}

	runtimeAgentName := strings.TrimSpace(opts.AgentName)
	if runtimeAgentName == "" {
		runtimeAgentName = fixtureAgentName
	}

	nodePath, err := resolveNodePath(opts.NodePath)
	if err != nil {
		return Registration{}, err
	}
	driverPath, err := resolveDriverPath(opts.DriverPath)
	if err != nil {
		return Registration{}, err
	}

	diagnosticsPath, err := resolveDiagnosticsPath(homePaths, runtimeAgentName, opts.DiagnosticsPath)
	if err != nil {
		return Registration{}, err
	}
	command := BuildCommand(nodePath, driverPath, fixturePath, fixtureAgentName, diagnosticsPath)

	agentDefPath := filepath.Join(homePaths.AgentsDir, runtimeAgentName, "AGENT.md")
	if err := os.MkdirAll(filepath.Dir(agentDefPath), 0o755); err != nil {
		return Registration{}, fmt.Errorf("acpmock: create agent directory %q: %w", filepath.Dir(agentDefPath), err)
	}

	content := renderAgentDef(runtimeAgentName, agent, command)
	if err := os.WriteFile(agentDefPath, []byte(content), 0o600); err != nil {
		return Registration{}, fmt.Errorf("acpmock: write agent definition %q: %w", agentDefPath, err)
	}

	loaded, err := aghconfig.LoadAgentDefFile(agentDefPath)
	if err != nil {
		return Registration{}, fmt.Errorf("acpmock: validate written agent definition %q: %w", agentDefPath, err)
	}
	cfg := aghconfig.DefaultWithHome(homePaths)
	if _, err := cfg.ResolveAgent(loaded); err != nil {
		return Registration{}, fmt.Errorf("acpmock: resolve written agent definition %q: %w", agentDefPath, err)
	}

	return Registration{
		AgentName:       runtimeAgentName,
		FixtureAgent:    fixtureAgentName,
		FixturePath:     fixturePath,
		NodePath:        nodePath,
		DriverPath:      driverPath,
		DiagnosticsPath: diagnosticsPath,
		AgentDefPath:    agentDefPath,
		Command:         command,
		Provider:        agent.Provider,
		Model:           agent.Model,
		Permissions:     agent.Permissions,
	}, nil
}

// DefaultDriverPath resolves the committed Node driver entrypoint.
func DefaultDriverPath() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("acpmock: runtime.Caller(0) failed")
	}
	path := filepath.Join(filepath.Dir(file), "driver", "dist", "index.js")
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("acpmock: stat driver entrypoint %q: %w", path, err)
	}
	return path, nil
}

// ResolveNodePath resolves the node executable used by the test-only driver.
func ResolveNodePath() (string, error) {
	return resolveNodePath("")
}

// BuildCommand renders the test-only ACP driver command string stored in AGENT.md.
func BuildCommand(
	nodePath string,
	driverPath string,
	fixturePath string,
	fixtureAgent string,
	diagnosticsPath string,
) string {
	argv := []string{
		strings.TrimSpace(nodePath),
		strings.TrimSpace(driverPath),
		"--fixture",
		strings.TrimSpace(fixturePath),
		"--agent",
		strings.TrimSpace(fixtureAgent),
	}
	if strings.TrimSpace(diagnosticsPath) != "" {
		argv = append(argv, "--diagnostics", strings.TrimSpace(diagnosticsPath))
	}
	return shellquote.Join(argv...)
}

func renderAgentDef(name string, agent AgentFixture, command string) string {
	prompt := strings.TrimSpace(agent.Prompt)
	if prompt == "" {
		prompt = "You are " + name + "."
	}

	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("name: " + strings.TrimSpace(name) + "\n")
	builder.WriteString("provider: " + strings.TrimSpace(agent.Provider) + "\n")
	builder.WriteString("command: " + strings.TrimSpace(command) + "\n")
	if model := strings.TrimSpace(agent.Model); model != "" {
		builder.WriteString("model: " + model + "\n")
	}
	if permissions := strings.TrimSpace(agent.Permissions); permissions != "" {
		builder.WriteString("permissions: " + permissions + "\n")
	}
	builder.WriteString("---\n\n")
	builder.WriteString(prompt)
	builder.WriteString("\n")
	return builder.String()
}

func resolveNodePath(override string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed, nil
	}
	if trimmed := strings.TrimSpace(os.Getenv("AGH_TEST_NODE_BIN")); trimmed != "" {
		return trimmed, nil
	}

	path, err := exec.LookPath("node")
	if err != nil {
		return "", fmt.Errorf("acpmock: resolve node executable: %w", err)
	}
	return path, nil
}

func resolveDriverPath(override string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		return trimmed, nil
	}
	return DefaultDriverPath()
}

func resolveDiagnosticsPath(homePaths aghconfig.HomePaths, name string, override string) (string, error) {
	if trimmed := strings.TrimSpace(override); trimmed != "" {
		if err := os.MkdirAll(filepath.Dir(trimmed), 0o755); err != nil {
			return "", fmt.Errorf("acpmock: create diagnostics directory %q: %w", filepath.Dir(trimmed), err)
		}
		return trimmed, nil
	}

	dir := filepath.Join(homePaths.LogsDir, "acpmock")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("acpmock: create diagnostics directory %q: %w", dir, err)
	}
	return filepath.Join(dir, strings.TrimSpace(name)+".jsonl"), nil
}
