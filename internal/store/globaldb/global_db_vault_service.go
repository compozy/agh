package globaldb

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pedronauck/agh/internal/vault"
)

func (g *GlobalDB) vaultService() (*vault.Service, error) {
	if g == nil {
		return nil, errors.New("store: global database is required")
	}
	lookupEnv := func(key string) (string, bool) {
		value, ok := os.LookupEnv(key)
		return value, ok && strings.TrimSpace(value) != ""
	}
	return vault.NewService(
		g,
		vault.NewFileKeyProvider(filepath.Dir(g.path), lookupEnv),
		vault.WithLookupEnv(lookupEnv),
		vault.WithNow(g.now),
	)
}
