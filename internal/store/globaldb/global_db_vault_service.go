package globaldb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/compozy/agh/internal/vault"
)

func (g *GlobalDB) vaultService() (*vault.Service, error) {
	return g.vaultServiceForStore(g)
}

func (g *GlobalDB) vaultServiceForStore(vaultStore vault.Store) (*vault.Service, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	if vaultStore == nil {
		return nil, errors.New("store: vault store is required")
	}
	lookupEnv := func(key string) (string, bool) {
		value, ok := os.LookupEnv(key)
		return value, ok && strings.TrimSpace(value) != ""
	}
	return vault.NewService(
		vaultStore,
		vault.NewFileKeyProvider(filepath.Dir(g.path), lookupEnv),
		vault.WithLookupEnv(lookupEnv),
		vault.WithNow(g.now),
	)
}
