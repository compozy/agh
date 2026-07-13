package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/compozy/agh/internal/memory"
	memoryschema "github.com/compozy/agh/internal/memory/schema"
	"github.com/compozy/agh/internal/store"
	"github.com/compozy/agh/internal/store/globaldb"
	globalschema "github.com/compozy/agh/internal/store/globaldb/schema"
	"github.com/compozy/agh/internal/store/sessiondb"
	sessionschema "github.com/compozy/agh/internal/store/sessiondb/schema"
	"github.com/compozy/agh/internal/store/workspacedb"
	workspaceschema "github.com/compozy/agh/internal/store/workspacedb/schema"
	"github.com/compozy/agh/internal/testutil"
	_ "modernc.org/sqlite"
)

type productionMigrationStream struct {
	name              string
	stream            store.MigrationStream
	schemaFS          fs.FS
	declarativeSource string
}

func productionMigrationStreams() []productionMigrationStream {
	return []productionMigrationStream{
		{
			name:              "global",
			stream:            globaldb.MigrationStream(),
			schemaFS:          globalschema.Files,
			declarativeSource: "definitions",
		},
		{
			name:              "session",
			stream:            sessiondb.MigrationStream(),
			schemaFS:          sessionschema.Files,
			declarativeSource: "schema.sql",
		},
		{
			name:              "memory",
			stream:            memory.MigrationStream(),
			schemaFS:          memoryschema.Files,
			declarativeSource: "schema.sql",
		},
		{
			name:              "workspace",
			stream:            workspacedb.MigrationStream(),
			schemaFS:          workspaceschema.Files,
			declarativeSource: "schema.sql",
		},
	}
}

func TestProductionMigrationStreams(t *testing.T) {
	t.Run("Should embed four distinct sequential baseline streams", func(t *testing.T) {
		t.Parallel()

		seenTables := make(map[string]string)
		for _, item := range productionMigrationStreams() {
			if item.stream.Name != item.name {
				t.Fatalf("stream name = %q, want %q", item.stream.Name, item.name)
			}
			if owner, exists := seenTables[item.stream.VersionTable]; exists {
				t.Fatalf("version table %q shared by %s and %s", item.stream.VersionTable, owner, item.name)
			}
			seenTables[item.stream.VersionTable] = item.name
			entries, err := fs.ReadDir(item.stream.FS, item.stream.Dir)
			if err != nil {
				t.Fatalf("read %s migration directory: %v", item.name, err)
			}
			versions := make([]int, 0)
			foundBaseline := false
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
					continue
				}
				separator := strings.IndexByte(entry.Name(), '_')
				if separator <= 0 {
					t.Fatalf("%s migration filename %q has no version prefix", item.name, entry.Name())
				}
				version, err := strconv.Atoi(entry.Name()[:separator])
				if err != nil {
					t.Fatalf("parse %s migration version: %v", item.name, err)
				}
				versions = append(versions, version)
				if version == 1 {
					if entry.Name() != "00001_baseline.sql" {
						t.Fatalf("%s first migration = %q, want 00001_baseline.sql", item.name, entry.Name())
					}
					foundBaseline = true
				}
			}
			sort.Ints(versions)
			if !foundBaseline || len(versions) == 0 {
				t.Fatalf("%s migrations have no 00001_baseline.sql", item.name)
			}
			for index, version := range versions {
				if want := index + 1; version != want {
					t.Fatalf("%s migration versions = %v, want sequential versions from 1", item.name, versions)
				}
			}
			if _, err := fs.ReadFile(item.stream.FS, item.stream.Dir+"/atlas.sum"); err != nil {
				t.Fatalf("read %s atlas.sum: %v", item.name, err)
			}
		}
	})

	t.Run("Should keep global and memory domain table ownership disjoint", func(t *testing.T) {
		t.Parallel()

		globalTables := schemaOwnedTables(t, globalschema.Files, "definitions")
		memoryTables := schemaOwnedTables(t, memoryschema.Files, "schema.sql")
		for table := range globalTables {
			if memoryTables[table] {
				t.Fatalf("table %q is owned by both global and memory baselines", table)
			}
		}
		if globalTables["memory_events"] {
			t.Fatal("global baseline owns memory_events, want memory stream ownership")
		}
		if !memoryTables["memory_events"] {
			t.Fatal("memory baseline does not own memory_events")
		}
	})

	t.Run("Should apply global and memory baselines to one physical database", func(t *testing.T) {
		t.Parallel()

		ctx := testutil.Context(t)
		db := openStreamTestDB(t, "shared-agh.db")
		globalStream := globaldb.MigrationStream()
		memoryStream := memory.MigrationStream()
		if err := store.Apply(ctx, db, globalStream); err != nil {
			t.Fatalf("Apply(global) error = %v", err)
		}
		if err := store.Apply(ctx, db, memoryStream); err != nil {
			t.Fatalf("Apply(memory) error = %v", err)
		}
		for _, stream := range []store.MigrationStream{globalStream, memoryStream} {
			status, err := store.Status(ctx, db, stream)
			if err != nil {
				t.Fatalf("Status(%s) error = %v", stream.Name, err)
			}
			if status.Version != 1 || status.AppliedCount != 1 {
				t.Fatalf("Status(%s) = %#v, want baseline v1", stream.Name, status)
			}
		}
		if !sqliteTableExists(t, db, "memory_events") {
			t.Fatal("memory_events missing after shared-file baseline application")
		}
	})
}

func TestMigrationSchemaEquivalence(t *testing.T) {
	for _, item := range productionMigrationStreams() {
		t.Run("Should match the declarative schema for the "+item.name+" stream", func(t *testing.T) {
			t.Parallel()

			ctx := testutil.Context(t)
			migrationDB := openStreamTestDB(t, item.name+"-migration.db")
			if err := store.Apply(ctx, migrationDB, item.stream); err != nil {
				t.Fatalf("Apply(%s) error = %v", item.name, err)
			}
			schemaDB := openStreamTestDB(t, item.name+"-schema.db")
			executeDeclarativeSchema(t, schemaDB, item.schemaFS, item.declarativeSource)
			got := normalizedSQLiteSchema(t, migrationDB, item.stream.VersionTable)
			want := normalizedSQLiteSchema(t, schemaDB, "")
			if got != want {
				t.Fatalf(
					"%s migrations schema differs from declarative schema\n--- migrations ---\n%s\n--- declarative ---\n%s",
					item.name,
					got,
					want,
				)
			}
		})
	}
}

func schemaOwnedTables(t *testing.T, schemaFS fs.FS, source string) map[string]bool {
	t.Helper()
	db := openStreamTestDB(t, "owned-tables.db")
	executeDeclarativeSchema(t, db, schemaFS, source)
	rows, err := db.QueryContext(
		testutil.Context(t),
		`SELECT name FROM pragma_table_list WHERE schema = 'main' AND type IN ('table', 'virtual')`,
	)
	if err != nil {
		t.Fatalf("query owned tables: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close owned table rows: %v", err)
		}
	}()
	tables := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan owned table: %v", err)
		}
		if !strings.HasPrefix(name, "sqlite_") {
			tables[name] = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate owned tables: %v", err)
	}
	return tables
}

func executeDeclarativeSchema(t *testing.T, db *sql.DB, schemaFS fs.FS, source string) {
	t.Helper()
	info, err := fs.Stat(schemaFS, source)
	if err != nil {
		t.Fatalf("stat declarative schema source %q: %v", source, err)
	}
	files := []string{source}
	if info.IsDir() {
		files = files[:0]
		entries, err := fs.ReadDir(schemaFS, source)
		if err != nil {
			t.Fatalf("read declarative schema directory %q: %v", source, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || path.Ext(entry.Name()) != ".sql" {
				continue
			}
			files = append(files, path.Join(source, entry.Name()))
		}
	}
	if len(files) == 0 {
		t.Fatalf("declarative schema source %q contains no SQL files", source)
	}
	for _, name := range files {
		contents, err := fs.ReadFile(schemaFS, name)
		if err != nil {
			t.Fatalf("read declarative schema file %q: %v", name, err)
		}
		if _, err := db.ExecContext(testutil.Context(t), string(contents)); err != nil {
			t.Fatalf("execute declarative schema file %q: %v", name, err)
		}
	}
}

func openStreamTestDB(t *testing.T, name string) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	if err := db.PingContext(testutil.Context(t)); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close failed database: %v", closeErr)
		}
		t.Fatalf("ping %s: %v", path, err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil && !errors.Is(err, sql.ErrConnDone) {
			t.Errorf("close %s: %v", path, err)
		}
	})
	return db
}

func normalizedSQLiteSchema(t *testing.T, db *sql.DB, excludedVersionTable string) string {
	t.Helper()
	rows, err := db.QueryContext(testutil.Context(t), `SELECT type, name, sql FROM sqlite_master
		WHERE sql IS NOT NULL AND name NOT LIKE 'sqlite_%' ORDER BY type, name`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close sqlite_master rows: %v", err)
		}
	}()
	objects := make([]string, 0)
	for rows.Next() {
		var kind string
		var name string
		var statement string
		if err := rows.Scan(&kind, &name, &statement); err != nil {
			t.Fatalf("scan sqlite_master: %v", err)
		}
		if name == excludedVersionTable {
			continue
		}
		objects = append(objects, fmt.Sprintf("%s|%s|%s", kind, name, normalizeSchemaSQL(statement)))
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate sqlite_master: %v", err)
	}
	return strings.Join(objects, "\n")
}

func sqliteTableExists(t *testing.T, db *sql.DB, table string) bool {
	t.Helper()
	var exists bool
	if err := db.QueryRowContext(
		testutil.Context(t),
		`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)`,
		table,
	).Scan(&exists); err != nil {
		t.Fatalf("query table %s existence: %v", table, err)
	}
	return exists
}

func normalizeSchemaSQL(statement string) string {
	return strings.Join(strings.Fields(statement), " ")
}
