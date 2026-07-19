package config

type desktopStateOverlay struct {
	MaxValueKiB         *int `toml:"max_value_kib"`
	MaxKeysPerWorkspace *int `toml:"max_keys_per_workspace"`
}

func (o desktopStateOverlay) configured() bool {
	return o.MaxValueKiB != nil || o.MaxKeysPerWorkspace != nil
}

func (o desktopStateOverlay) Apply(dst *DesktopStateConfig) {
	if o.MaxValueKiB != nil {
		dst.MaxValueKiB = *o.MaxValueKiB
	}
	if o.MaxKeysPerWorkspace != nil {
		dst.MaxKeysPerWorkspace = *o.MaxKeysPerWorkspace
	}
}
