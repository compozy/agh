package daemon

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	bridgepkg "github.com/pedronauck/agh/internal/bridges"
)

const bridgeSecretEnvRefPrefix = "env:"

var bridgeSecretEnvNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type bridgeSecretBindingValidator interface {
	ValidateBridgeSecretBinding(binding bridgepkg.BridgeSecretBinding) error
}

type envBridgeSecretResolver struct {
	getenv func(string) string
}

var _ BridgeSecretResolver = envBridgeSecretResolver{}
var _ bridgeSecretBindingValidator = envBridgeSecretResolver{}

func (r envBridgeSecretResolver) ValidateBridgeSecretBinding(binding bridgepkg.BridgeSecretBinding) error {
	if _, err := parseEnvBridgeSecretRef(binding.VaultRef); err != nil {
		return fmt.Errorf("%w: %w", bridgepkg.ErrInvalidBridgeSecretBinding, err)
	}
	return nil
}

func (r envBridgeSecretResolver) ResolveBridgeSecret(ctx context.Context, binding bridgepkg.BridgeSecretBinding) (string, error) {
	if ctx == nil {
		return "", errors.New("daemon: resolve bridge secret context is required")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if r.getenv == nil {
		return "", errors.New("daemon: bridge secret env source is not configured")
	}

	envName, err := parseEnvBridgeSecretRef(binding.VaultRef)
	if err != nil {
		return "", fmt.Errorf("%w: %w", bridgepkg.ErrInvalidBridgeSecretBinding, err)
	}

	value := strings.TrimSpace(r.getenv(envName))
	if value == "" {
		return "", fmt.Errorf("%w: bridge secret env %q is not set or empty", bridgepkg.ErrInvalidBridgeSecretBinding, envName)
	}

	return value, nil
}

func parseEnvBridgeSecretRef(vaultRef string) (string, error) {
	trimmed := strings.TrimSpace(vaultRef)
	if trimmed == "" {
		return "", errors.New("stock daemon bridge secret refs must use env:NAME")
	}
	if !strings.HasPrefix(trimmed, bridgeSecretEnvRefPrefix) {
		return "", fmt.Errorf("stock daemon bridge secret refs must use env:NAME, got %q", trimmed)
	}

	envName := strings.TrimSpace(strings.TrimPrefix(trimmed, bridgeSecretEnvRefPrefix))
	if envName == "" {
		return "", errors.New("stock daemon bridge secret refs must include an env var name after env")
	}
	if !bridgeSecretEnvNamePattern.MatchString(envName) {
		return "", fmt.Errorf("stock daemon bridge secret env %q is invalid", envName)
	}

	return envName, nil
}
