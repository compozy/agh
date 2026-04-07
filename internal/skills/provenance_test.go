package skills

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestComputeHashReturnsConsistentSHA256ForSameContent(t *testing.T) {
	t.Parallel()

	content := []byte(strings.Join([]string{
		"---",
		"name: marketplace-skill",
		"description: Example skill",
		"---",
		"Review carefully.",
	}, "\n"))

	first := ComputeHash(content)
	second := ComputeHash(content)

	if first != second {
		t.Fatalf("ComputeHash() first = %q, second = %q, want identical hashes", first, second)
	}
}

func TestComputeHashReturnsDifferentHashForDifferentContent(t *testing.T) {
	t.Parallel()

	first := ComputeHash([]byte("first content"))
	second := ComputeHash([]byte("second content"))

	if first == second {
		t.Fatalf("ComputeHash() hashes match for different content: %q", first)
	}
}

func TestWriteSidecarCreatesStableHumanReadableJSON(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	provenance := testProvenance()

	if err := WriteSidecar(skillDir, provenance); err != nil {
		t.Fatalf("WriteSidecar() error = %v", err)
	}

	sidecarBytes, err := os.ReadFile(filepath.Join(skillDir, sidecarFileName))
	if err != nil {
		t.Fatalf("ReadFile(sidecar) error = %v", err)
	}

	want := strings.Join([]string{
		"{",
		`  "hash": "abc123",`,
		`  "registry": "clawhub",`,
		`  "slug": "@author/marketplace-skill",`,
		`  "version": "1.2.3",`,
		`  "installed_at": "2026-04-07T12:00:00Z"`,
		"}",
		"",
	}, "\n")

	if string(sidecarBytes) != want {
		t.Fatalf("sidecar JSON mismatch\nwant:\n%s\ngot:\n%s", want, string(sidecarBytes))
	}
}

func TestReadSidecarParsesValidJSONIntoProvenance(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	want := testProvenance()
	if err := WriteSidecar(skillDir, want); err != nil {
		t.Fatalf("WriteSidecar() error = %v", err)
	}

	got, err := ReadSidecar(skillDir)
	if err != nil {
		t.Fatalf("ReadSidecar() error = %v", err)
	}

	if got == nil {
		t.Fatal("ReadSidecar() = nil, want provenance")
	}
	if *got != want {
		t.Fatalf("ReadSidecar() = %#v, want %#v", *got, want)
	}
}

func TestReadSidecarReturnsDescriptiveErrorForMalformedJSON(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	sidecarPath := filepath.Join(skillDir, sidecarFileName)
	if err := os.WriteFile(sidecarPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", sidecarPath, err)
	}

	_, err := ReadSidecar(skillDir)
	if err == nil {
		t.Fatal("ReadSidecar() error = nil, want malformed JSON error")
	}
	if !strings.Contains(err.Error(), "parse provenance sidecar") {
		t.Fatalf("ReadSidecar() error = %v, want parse provenance sidecar context", err)
	}
}

func TestReadSidecarReturnsNotExistForMissingSidecar(t *testing.T) {
	t.Parallel()

	_, err := ReadSidecar(t.TempDir())
	if err == nil {
		t.Fatal("ReadSidecar() error = nil, want fs.ErrNotExist")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadSidecar() error = %v, want fs.ErrNotExist", err)
	}
}

func TestVerifyHashReturnsNilWhenHashMatches(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	content := strings.Join([]string{
		"---",
		"name: verified-skill",
		"description: Verified skill",
		"---",
		"Everything is intact.",
	}, "\n")
	writeSkillFile(t, skillDir, skillFileName, content)

	err := VerifyHash(skillDir, &Provenance{
		Hash: ComputeHash([]byte(content)),
	})
	if err != nil {
		t.Fatalf("VerifyHash() error = %v, want nil", err)
	}
}

func TestVerifyHashReturnsExpectedAndActualHashWhenTampered(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	original := strings.Join([]string{
		"---",
		"name: tampered-skill",
		"description: Original content",
		"---",
		"Original body.",
	}, "\n")
	tampered := strings.Join([]string{
		"---",
		"name: tampered-skill",
		"description: Tampered content",
		"---",
		"Tampered body.",
	}, "\n")
	writeSkillFile(t, skillDir, skillFileName, tampered)

	err := VerifyHash(skillDir, &Provenance{
		Hash: ComputeHash([]byte(original)),
	})
	if err == nil {
		t.Fatal("VerifyHash() error = nil, want hash mismatch")
	}

	var mismatch *HashMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("VerifyHash() error = %v, want HashMismatchError", err)
	}
	if mismatch.ExpectedHash != ComputeHash([]byte(original)) {
		t.Fatalf("HashMismatchError.ExpectedHash = %q, want %q", mismatch.ExpectedHash, ComputeHash([]byte(original)))
	}
	if mismatch.ActualHash != ComputeHash([]byte(tampered)) {
		t.Fatalf("HashMismatchError.ActualHash = %q, want %q", mismatch.ActualHash, ComputeHash([]byte(tampered)))
	}
	if !strings.Contains(err.Error(), mismatch.ExpectedHash) || !strings.Contains(err.Error(), mismatch.ActualHash) {
		t.Fatalf("VerifyHash() error = %q, want expected and actual hashes in message", err.Error())
	}
}

func TestHasSidecarReturnsTrueWhenSidecarExists(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	if err := WriteSidecar(skillDir, testProvenance()); err != nil {
		t.Fatalf("WriteSidecar() error = %v", err)
	}

	hasSidecar, err := HasSidecar(skillDir)
	if err != nil {
		t.Fatalf("HasSidecar() error = %v", err)
	}
	if !hasSidecar {
		t.Fatal("HasSidecar() = false, want true")
	}
}

func TestHasSidecarReturnsFalseWhenSidecarMissing(t *testing.T) {
	t.Parallel()

	hasSidecar, err := HasSidecar(t.TempDir())
	if err != nil {
		t.Fatalf("HasSidecar() error = %v", err)
	}
	if hasSidecar {
		t.Fatal("HasSidecar() = true, want false")
	}
}

func TestSidecarRoundTripProducesIdenticalProvenance(t *testing.T) {
	t.Parallel()

	skillDir := t.TempDir()
	want := testProvenance()

	if err := WriteSidecar(skillDir, want); err != nil {
		t.Fatalf("WriteSidecar() error = %v", err)
	}

	got, err := ReadSidecar(skillDir)
	if err != nil {
		t.Fatalf("ReadSidecar() error = %v", err)
	}

	marshaledGot, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json.Marshal(got) error = %v", err)
	}
	marshaledWant, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("json.Marshal(want) error = %v", err)
	}

	if string(marshaledGot) != string(marshaledWant) {
		t.Fatalf("round-trip mismatch\ngot:  %s\nwant: %s", marshaledGot, marshaledWant)
	}
}

func testProvenance() Provenance {
	return Provenance{
		Hash:        "abc123",
		Registry:    "clawhub",
		Slug:        "@author/marketplace-skill",
		Version:     "1.2.3",
		InstalledAt: time.Date(2026, 4, 7, 12, 0, 0, 0, time.UTC),
	}
}
