package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDocCommand_Hidden(t *testing.T) {
	t.Parallel()

	cmd := newDocCommand()
	if !cmd.Hidden {
		t.Error("doc command should be hidden")
	}
}

func TestNewDocCommand_NotInHelp(t *testing.T) {
	t.Parallel()

	root := newRootCommand(commandDeps{})
	help := root.UsageString()
	if strings.Contains(help, "doc") {
		t.Error("doc command should not appear in root help output")
	}
}

func TestNewDocCommand_DefaultOutputDir(t *testing.T) {
	t.Parallel()

	cmd := newDocCommand()
	flag := cmd.Flags().Lookup("output-dir")
	if flag == nil {
		t.Fatal("doc command should have --output-dir flag")
	}
	if flag.DefValue != defaultCLIDocsDir {
		t.Errorf("default output-dir = %q, want %q", flag.DefValue, defaultCLIDocsDir)
	}
}

func TestNewDocCommand_GeneratesDocs(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "cli-docs")

	root := newRootCommand(commandDeps{})
	root.SetArgs([]string{"doc", "--output-dir", outputDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("doc command failed: %v", err)
	}

	// Verify output directory was created.
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatalf("could not read output dir: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("doc command should generate files")
	}

	// Verify .mdx files exist.
	var mdxCount int
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".mdx") {
			mdxCount++
		}
	}
	if mdxCount == 0 {
		t.Error("doc command should generate .mdx files")
	}

	// Verify meta.json exists.
	if _, err := os.Stat(filepath.Join(outputDir, "meta.json")); err != nil {
		t.Error("doc command should generate meta.json")
	}

	// Verify agh.mdx exists (root command).
	aghMDX := filepath.Join(outputDir, "agh.mdx")
	data, err := os.ReadFile(aghMDX)
	if err != nil {
		t.Fatalf("agh.mdx should exist: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "---") {
		t.Error("agh.mdx should have YAML frontmatter")
	}
	if !strings.Contains(content, `title: "agh"`) {
		t.Error("agh.mdx frontmatter should have title")
	}

	// Verify no absolute paths in any generated file.
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fileData, err := os.ReadFile(filepath.Join(outputDir, e.Name()))
		if err != nil {
			continue
		}
		fileContent := string(fileData)
		if strings.Contains(fileContent, "/Users/") || strings.Contains(fileContent, "/home/") {
			t.Errorf("file %s contains absolute paths", e.Name())
		}
	}
}

func TestNewDocCommand_CreatesOutputDir(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "nested", "deep", "output")

	root := newRootCommand(commandDeps{})
	root.SetArgs([]string{"doc", "--output-dir", outputDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("doc command should create output dir: %v", err)
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Error("output directory should have been created")
	}
}

func TestNewDocCommand_GeneratesAllCommands(t *testing.T) {
	t.Parallel()

	outputDir := filepath.Join(t.TempDir(), "cli-docs")

	root := newRootCommand(commandDeps{})
	root.SetArgs([]string{"doc", "--output-dir", outputDir})

	if err := root.Execute(); err != nil {
		t.Fatalf("doc command failed: %v", err)
	}

	// Verify some expected command files exist.
	expectedFiles := []string{
		"agh.mdx",
		"agh_session.mdx",
		"agh_daemon.mdx",
		"agh_version.mdx",
	}
	for _, name := range expectedFiles {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Errorf("expected file %s to exist", name)
		}
	}
}
