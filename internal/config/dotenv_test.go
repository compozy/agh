package config

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestDotEnvParserSanitizesAndRepairsStructuredEntries(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	contents := strings.Join([]string{
		"# keep comments",
		"AGH_HOME=/tmp/agh-home",
		"OPENAI_API_KEY=sk-live\u200b ANTHROPIC_API_KEY=anthropic\u2011key",
		`PLAIN_VALUE="hello world"`,
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}

	report, err := InspectDotEnvFile(path)
	if err != nil {
		t.Fatalf("InspectDotEnvFile() error = %v", err)
	}
	if report.Status != DotEnvStatusRepairable {
		t.Fatalf("InspectDotEnvFile() Status = %q, want %q", report.Status, DotEnvStatusRepairable)
	}
	if len(report.Diagnostics) != 3 {
		t.Fatalf("InspectDotEnvFile() diagnostics = %#v, want multi-key plus two sanitizers", report.Diagnostics)
	}

	repair, err := RepairDotEnvFile(path)
	if err != nil {
		t.Fatalf("RepairDotEnvFile() error = %v", err)
	}
	if repair.Status != DotEnvStatusRepaired || !repair.Repaired {
		t.Fatalf("RepairDotEnvFile() = %#v, want repaired status", repair)
	}

	repaired, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(repaired .env) error = %v", err)
	}
	repairedText := string(repaired)
	for _, want := range []string{
		"# keep comments",
		"AGH_HOME=/tmp/agh-home",
		"OPENAI_API_KEY=sk-live",
		"ANTHROPIC_API_KEY=anthropickey",
		`PLAIN_VALUE="hello world"`,
	} {
		if !strings.Contains(repairedText, want) {
			t.Fatalf("repaired .env missing %q:\n%s", want, repairedText)
		}
	}
	if strings.Contains(repairedText, "\u200b") || strings.Contains(repairedText, "\u2011") {
		t.Fatalf("repaired .env retained non-ASCII secret characters:\n%s", repairedText)
	}

	parsed := parseDotEnvDocument(repairedText)
	if parsed.unsupported || parsed.needsRepair {
		t.Fatalf("parseDotEnvDocument(repaired) = %#v, want clean parse", parsed)
	}
	wantValues := map[string]string{
		"AGH_HOME":          "/tmp/agh-home",
		"OPENAI_API_KEY":    "sk-live",
		"ANTHROPIC_API_KEY": "anthropickey",
		"PLAIN_VALUE":       "hello world",
	}
	if !reflect.DeepEqual(parsed.values, wantValues) {
		t.Fatalf("parsed values = %#v, want %#v", parsed.values, wantValues)
	}
}

func TestReplaceDotEnvFileUsesDurableWriteProtocol(t *testing.T) {
	t.Parallel()

	source, err := os.ReadFile("dotenv.go")
	if err != nil {
		t.Fatalf("os.ReadFile(dotenv.go) error = %v", err)
	}
	body := string(source)
	start := strings.Index(body, "func replaceDotEnvFile(")
	if start < 0 {
		t.Fatal("replaceDotEnvFile function not found")
	}
	end := strings.Index(body[start:], "\nfunc dotEnvUnsupportedError(")
	if end < 0 {
		t.Fatal("replaceDotEnvFile function end not found")
	}
	fn := body[start : start+end]
	if !strings.Contains(fn, ".Sync()") {
		t.Fatalf("replaceDotEnvFile missing temp file Sync before rename:\n%s", fn)
	}
	if !strings.Contains(fn, "syncPersistedDir(dir)") {
		t.Fatalf("replaceDotEnvFile missing parent directory sync after rename:\n%s", fn)
	}
}

func TestRepairDotEnvFileRejectsUnsupportedContentWithoutWriting(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), ".env")
	before := "VALID=value\nsource ./secrets.env\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}

	report, err := RepairDotEnvFile(path)
	if err == nil {
		t.Fatal("RepairDotEnvFile() error = nil, want unsupported content error")
	}
	if !errors.Is(err, ErrDotEnvUnsupported) {
		t.Fatalf("RepairDotEnvFile() error = %v, want ErrDotEnvUnsupported", err)
	}
	if report.Status != DotEnvStatusUnsupported {
		t.Fatalf("RepairDotEnvFile() report = %#v, want unsupported status", report)
	}
	if strings.Contains(err.Error(), "VALID=value") {
		t.Fatalf("RepairDotEnvFile() error leaked .env value: %v", err)
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("os.ReadFile(.env after repair) error = %v", readErr)
	}
	if string(after) != before {
		t.Fatalf(".env changed after unsupported repair\nbefore:\n%s\nafter:\n%s", before, string(after))
	}
}

func TestRepairDotEnvFileRejectsSymlinkWithoutReadingTarget(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary on Windows")
	}

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "actual.env")
	before := "AGH_TASK09_API_KEY=secret\u200b-value\n"
	if err := os.WriteFile(targetPath, []byte(before), 0o600); err != nil {
		t.Fatalf("os.WriteFile(target .env) error = %v", err)
	}
	linkPath := filepath.Join(dir, ".env")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Fatalf("os.Symlink(.env) error = %v", err)
	}

	report, err := RepairDotEnvFile(linkPath)
	if err == nil {
		t.Fatal("RepairDotEnvFile(symlink) error = nil, want unsupported symlink")
	}
	if !errors.Is(err, ErrDotEnvUnsupported) {
		t.Fatalf("RepairDotEnvFile(symlink) error = %v, want ErrDotEnvUnsupported", err)
	}
	if report.Status != DotEnvStatusUnsupported {
		t.Fatalf("RepairDotEnvFile(symlink) report = %#v, want unsupported status", report)
	}

	after, readErr := os.ReadFile(targetPath)
	if readErr != nil {
		t.Fatalf("os.ReadFile(target .env after repair) error = %v", readErr)
	}
	if string(after) != before {
		t.Fatalf("symlink repair changed target .env\nbefore:\n%s\nafter:\n%s", before, string(after))
	}
}

func TestLoadDotEnvLookupUsesSanitizedInMemoryValuesWithoutMutatingFile(t *testing.T) {
	t.Parallel()

	workspace := t.TempDir()
	path := filepath.Join(workspace, ".env")
	before := "AGH_CONFIG_TASK09_TOKEN=tok\u200ben OTHER=value\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatalf("os.WriteFile(.env) error = %v", err)
	}

	lookup, err := loadDotEnvLookup(workspace)
	if err != nil {
		t.Fatalf("loadDotEnvLookup() error = %v", err)
	}
	value, ok := lookup("AGH_CONFIG_TASK09_TOKEN")
	if !ok || value != "token" {
		t.Fatalf("lookup(token) = %q, %t; want sanitized token", value, ok)
	}
	other, ok := lookup("OTHER")
	if !ok || other != "value" {
		t.Fatalf("lookup(OTHER) = %q, %t; want value", other, ok)
	}

	after, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("os.ReadFile(.env after load) error = %v", readErr)
	}
	if string(after) != before {
		t.Fatalf("loadDotEnvLookup mutated .env\nbefore:\n%s\nafter:\n%s", before, string(after))
	}
}
