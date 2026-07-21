package config

const (
	defaultDesktopStateMaxValueKiB         = 256
	defaultDesktopStateMaxKeysPerWorkspace = 512
	minDesktopStateMaxValueKiB             = 1
	maxDesktopStateMaxValueKiB             = 4096
	minDesktopStateMaxKeysPerWorkspace     = 16
	maxDesktopStateMaxKeysPerWorkspace     = 8192
	desktopStateMaxValueKiBPath            = "desktop_state.max_value_kib"
	desktopStateMaxKeysPerWorkspacePath    = "desktop_state.max_keys_per_workspace"
)

// DesktopStateConfig bounds the daemon-side desktop-state store.
type DesktopStateConfig struct {
	MaxValueKiB         int `toml:"max_value_kib"`
	MaxKeysPerWorkspace int `toml:"max_keys_per_workspace"`
}

// DefaultDesktopStateConfig returns the built-in desktop-state limits.
func DefaultDesktopStateConfig() DesktopStateConfig {
	return DesktopStateConfig{
		MaxValueKiB:         defaultDesktopStateMaxValueKiB,
		MaxKeysPerWorkspace: defaultDesktopStateMaxKeysPerWorkspace,
	}
}

// Validate ensures desktop-state limits stay within their supported ranges.
func (c DesktopStateConfig) Validate() error {
	if c.MaxValueKiB < minDesktopStateMaxValueKiB || c.MaxValueKiB > maxDesktopStateMaxValueKiB {
		return ValidationError{
			Path:    desktopStateMaxValueKiBPath,
			Message: "must be between 1 and 4096",
		}
	}
	if c.MaxKeysPerWorkspace < minDesktopStateMaxKeysPerWorkspace ||
		c.MaxKeysPerWorkspace > maxDesktopStateMaxKeysPerWorkspace {
		return ValidationError{
			Path:    desktopStateMaxKeysPerWorkspacePath,
			Message: "must be between 16 and 8192",
		}
	}
	return nil
}
