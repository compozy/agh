package prompts

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Should load explicit v1 assets", func(t *testing.T) {
		t.Parallel()

		for _, name := range allAssetNames() {
			asset, err := Load(name, VersionV1)
			if err != nil {
				t.Fatalf("load %s: %v", name, err)
			}
			if asset.Name != name {
				t.Fatalf("asset name = %q, want %q", asset.Name, name)
			}
			if asset.Version != VersionV1 {
				t.Fatalf("asset version = %q, want %q", asset.Version, VersionV1)
			}
			if asset.Filename == "" {
				t.Fatalf("asset %s filename is empty", name)
			}
			if strings.TrimSpace(asset.Content) == "" {
				t.Fatalf("asset %s content is empty", name)
			}
		}
	})

	t.Run("Should select latest version deterministically", func(t *testing.T) {
		t.Parallel()

		registry := DefaultRegistry()
		for _, name := range allAssetNames() {
			explicit, err := registry.Load(name, VersionV1)
			if err != nil {
				t.Fatalf("load explicit %s: %v", name, err)
			}
			latest, err := registry.LoadLatest(name)
			if err != nil {
				t.Fatalf("load latest %s: %v", name, err)
			}
			if latest != explicit {
				t.Fatalf("latest asset = %#v, want %#v", latest, explicit)
			}
		}
	})

	t.Run("Should parse template assets with missing keys rejected", func(t *testing.T) {
		t.Parallel()

		for _, name := range allAssetNames() {
			parsed, err := ParseTemplate(name, VersionV1)
			if err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
			if name == NameWhatNotToSave {
				continue
			}
			var rendered bytes.Buffer
			err = parsed.Execute(&rendered, map[string]any{})
			if err == nil {
				t.Fatalf("execute %s with missing keys succeeded", name)
			}
		}
	})

	t.Run("Should fail clearly for unknown asset names and versions", func(t *testing.T) {
		t.Parallel()

		_, err := Load(Name("missing"), VersionV1)
		if !errors.Is(err, ErrAssetNotFound) {
			t.Fatalf("missing asset error = %v, want ErrAssetNotFound", err)
		}
		_, err = Load(NameDecide, "v2")
		if !errors.Is(err, ErrVersionNotFound) {
			t.Fatalf("missing version error = %v, want ErrVersionNotFound", err)
		}
	})

	t.Run("Should fail clearly for invalid templates and missing files", func(t *testing.T) {
		t.Parallel()

		invalidRegistry := NewRegistry(fstest.MapFS{
			"decide.v1.tmpl": {Data: []byte("{{")},
		})
		_, err := invalidRegistry.ParseTemplate(NameDecide, VersionV1)
		if err == nil {
			t.Fatalf("invalid template parse succeeded")
		}
		if !strings.Contains(err.Error(), "parse decide v1") {
			t.Fatalf("invalid template error = %q, want parse context", err.Error())
		}

		missingFileRegistry := NewRegistry(fstest.MapFS{})
		_, err = missingFileRegistry.Load(NameDecide, VersionV1)
		if err == nil {
			t.Fatalf("missing file load succeeded")
		}
		if !strings.Contains(err.Error(), "read decide v1") {
			t.Fatalf("missing file error = %q, want read context", err.Error())
		}
	})

	t.Run("Should expose controller extractor dreaming and policy fields", func(t *testing.T) {
		t.Parallel()

		assertAssetContains(t, NameDecide,
			"candidate.frontmatter.entity",
			"target.target_filename",
			"Rule trace",
		)
		assertAssetContains(t, NameExtract,
			"WHAT_NOT_TO_SAVE policy",
			"session_id",
			"Transcript snapshot",
		)
		assertAssetContains(t, NameDream,
			"_system/dreaming/",
			"deterministic DLQ replay",
		)
		assertAssetContains(t, NameWhatNotToSave,
			"Raw transcript dumps",
			"Secrets, credentials, tokens",
			"Anything already documented in AGENTS.md",
		)
	})
}

func allAssetNames() []Name {
	return []Name{NameDecide, NameDream, NameExtract, NameWhatNotToSave}
}

func assertAssetContains(t *testing.T, name Name, fragments ...string) {
	t.Helper()

	asset, err := Load(name, VersionV1)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(asset.Content, fragment) {
			t.Fatalf("asset %s missing fragment %q", name, fragment)
		}
	}
}
