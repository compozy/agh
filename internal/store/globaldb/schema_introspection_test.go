package globaldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/testutil"
)

type schemaQueryExecutor interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func tableExists(ctx context.Context, exec schemaQueryExecutor, table string) (bool, error) {
	var count int
	if err := exec.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`,
		strings.TrimSpace(table),
	).Scan(&count); err != nil {
		return false, fmt.Errorf("query table %q existence: %w", table, err)
	}
	return count > 0, nil
}

func tableColumns(
	ctx context.Context,
	exec schemaQueryExecutor,
	table string,
) (columns map[string]struct{}, err error) {
	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return nil, fmt.Errorf("inspect table %q: %w", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close table %q column rows: %w", table, closeErr)
			err = errors.Join(err, closeErr)
		}
	}()

	columns = make(map[string]struct{})
	for rows.Next() {
		var (
			position     int
			name         string
			dataType     string
			notNull      int
			defaultValue sql.NullString
			primaryKey   int
		)
		if scanErr := rows.Scan(&position, &name, &dataType, &notNull, &defaultValue, &primaryKey); scanErr != nil {
			return nil, fmt.Errorf("scan table %q column: %w", table, scanErr)
		}
		columns[name] = struct{}{}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate table %q columns: %w", table, rowsErr)
	}
	return columns, nil
}

func assertTableHasColumns(t *testing.T, db *sql.DB, table string, want []string) {
	t.Helper()
	columns, err := tableColumns(context.Background(), db, table)
	if err != nil {
		t.Fatalf("tableColumns(%q) error = %v", table, err)
	}
	for _, column := range want {
		if _, ok := columns[column]; !ok {
			t.Fatalf("table %q missing column %q", table, column)
		}
	}
}

func tableHasForeignKey(
	ctx context.Context,
	exec schemaQueryExecutor,
	table string,
	referencedTable string,
) (found bool, err error) {
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		return false, err
	}
	rows, err := exec.QueryContext(ctx, fmt.Sprintf("PRAGMA foreign_key_list(%s)", name))
	if err != nil {
		return false, fmt.Errorf("query foreign keys for table %q: %w", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			closeErr = fmt.Errorf("close table %q foreign-key rows: %w", table, closeErr)
			err = errors.Join(err, closeErr)
		}
	}()

	target := strings.TrimSpace(referencedTable)
	for rows.Next() {
		var (
			id       int
			sequence int
			refTable string
			from     string
			to       string
			onUpdate string
			onDelete string
			match    string
		)
		if scanErr := rows.Scan(&id, &sequence, &refTable, &from, &to, &onUpdate, &onDelete, &match); scanErr != nil {
			return false, fmt.Errorf("scan foreign keys for table %q: %w", table, scanErr)
		}
		if strings.EqualFold(strings.TrimSpace(refTable), target) {
			return true, nil
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return false, fmt.Errorf("iterate foreign keys for table %q: %w", table, rowsErr)
	}
	return false, nil
}

func assertTableHasNoForeignKeys(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	name, err := store.NormalizeSQLiteIdentifier(table)
	if err != nil {
		t.Fatalf("NormalizeSQLiteIdentifier(%q) error = %v", table, err)
	}
	rows, err := db.QueryContext(testutil.Context(t), fmt.Sprintf("PRAGMA foreign_key_list(%s)", name))
	if err != nil {
		t.Fatalf("query foreign keys for table %q error = %v", table, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			t.Fatalf("close foreign keys for table %q error = %v", table, closeErr)
		}
	}()
	if rows.Next() {
		t.Fatalf("table %q has a foreign key, want none", table)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign keys for table %q error = %v", table, err)
	}
}
