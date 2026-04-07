package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const sidecarFileName = ".agh-meta.json"

// HashMismatchError reports a provenance hash mismatch for a marketplace skill.
type HashMismatchError struct {
	ExpectedHash string
	ActualHash   string
}

// Error implements the error interface.
func (e *HashMismatchError) Error() string {
	if e == nil {
		return "skills: provenance hash mismatch"
	}

	return fmt.Sprintf(
		"skills: provenance hash mismatch: expected %s, got %s",
		e.ExpectedHash,
		e.ActualHash,
	)
}

// ComputeHash returns the SHA-256 hex digest for raw SKILL.md content bytes.
func ComputeHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

// WriteSidecar writes marketplace provenance metadata alongside a skill's SKILL.md file.
func WriteSidecar(skillDir string, provenance Provenance) error {
	sidecarPath, err := sidecarPath(skillDir)
	if err != nil {
		return err
	}

	payload, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil {
		return fmt.Errorf("skills: marshal provenance sidecar %q: %w", sidecarPath, err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(sidecarPath, payload, 0o644); err != nil {
		return fmt.Errorf("skills: write provenance sidecar %q: %w", sidecarPath, err)
	}

	return nil
}

// ReadSidecar reads marketplace provenance metadata from a skill directory.
func ReadSidecar(skillDir string) (*Provenance, error) {
	sidecarPath, err := sidecarPath(skillDir)
	if err != nil {
		return nil, err
	}

	payload, err := os.ReadFile(sidecarPath)
	if err != nil {
		return nil, fmt.Errorf("skills: read provenance sidecar %q: %w", sidecarPath, err)
	}

	var provenance Provenance
	if err := json.Unmarshal(payload, &provenance); err != nil {
		return nil, fmt.Errorf("skills: parse provenance sidecar %q: %w", sidecarPath, err)
	}

	return &provenance, nil
}

// VerifyHash recomputes the SKILL.md hash for a skill directory and compares it
// with the stored provenance hash.
func VerifyHash(skillDir string, provenance *Provenance) error {
	if provenance == nil {
		return errors.New("skills: provenance is required")
	}

	skillPath, err := skillFilePath(skillDir)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(skillPath)
	if err != nil {
		return fmt.Errorf("skills: read skill file %q for hash verification: %w", skillPath, err)
	}

	actualHash := ComputeHash(content)
	if actualHash == provenance.Hash {
		return nil
	}

	return &HashMismatchError{
		ExpectedHash: provenance.Hash,
		ActualHash:   actualHash,
	}
}

// HasSidecar reports whether a skill directory contains marketplace provenance metadata.
func HasSidecar(skillDir string) (bool, error) {
	sidecarPath, err := sidecarPath(skillDir)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(sidecarPath)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("skills: stat provenance sidecar %q: %w", sidecarPath, err)
}

func sidecarPath(skillDir string) (string, error) {
	return resolveSkillPath(skillDir, sidecarFileName)
}

func skillFilePath(skillDir string) (string, error) {
	return resolveSkillPath(skillDir, skillFileName)
}

func resolveSkillPath(skillDir string, fileName string) (string, error) {
	root := strings.TrimSpace(skillDir)
	if root == "" {
		return "", errors.New("skills: skill directory is required")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("skills: resolve skill directory %q: %w", skillDir, err)
	}

	return filepath.Join(absRoot, fileName), nil
}
