package globaldb

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const bundleActivationResourceKind = "bundle.activation"

type bundleActivationResourceSpec struct {
	ExtensionName string `json:"extension_name"`
}

func (g *GlobalDB) CountBundleActivationsForExtension(ctx context.Context, extensionName string) (int, error) {
	if err := g.checkReady(ctx, "count bundle activations for extension"); err != nil {
		return 0, err
	}

	trimmed := strings.TrimSpace(extensionName)
	if trimmed == "" {
		return 0, errors.New("store: extension name is required")
	}
	count, err := countBundleActivationResourcesForExtension(ctx, g.db, trimmed)
	if err != nil {
		return 0, fmt.Errorf("store: count bundle activations for extension %q: %w", trimmed, err)
	}
	return count, nil
}

func countBundleActivationResourcesForExtension(
	ctx context.Context,
	db *sql.DB,
	extensionName string,
) (int, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT spec_json FROM resource_records WHERE kind = ?`,
		bundleActivationResourceKind,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return 0, nil
		}
		return 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	count := 0
	trimmed := strings.TrimSpace(extensionName)
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return 0, fmt.Errorf("scan bundle activation resource: %w", err)
		}
		var spec bundleActivationResourceSpec
		if err := json.Unmarshal([]byte(raw), &spec); err != nil {
			return 0, fmt.Errorf("decode bundle activation resource: %w", err)
		}
		if strings.TrimSpace(spec.ExtensionName) == trimmed {
			count++
		}
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate bundle activation resources: %w", err)
	}
	return count, nil
}
