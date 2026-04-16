package config_test

import (
	"path/filepath"
	"testing"

	aghconfig "github.com/pedronauck/agh/internal/config"
	"github.com/pedronauck/agh/internal/resources"
	"github.com/pedronauck/agh/internal/store"
	"github.com/pedronauck/agh/internal/store/globaldb"
	"github.com/pedronauck/agh/internal/testutil"
)

func TestMCPServerResourceCodecRejectsInvalidSpec(t *testing.T) {
	t.Parallel()

	codec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("NewMCPServerResourceCodec() error = %v", err)
	}

	_, err = codec.DecodeAndValidate(
		testutil.Context(t),
		resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		[]byte(`{"name":"git"}`),
	)
	if err == nil {
		t.Fatal("codec.DecodeAndValidate() error = nil, want missing command failure")
	}
}

func TestMCPServerResourceStoreRoundTripReturnsTypedRecords(t *testing.T) {
	t.Parallel()

	db, err := globaldb.OpenGlobalDB(testutil.Context(t), filepath.Join(t.TempDir(), store.GlobalDatabaseName))
	if err != nil {
		t.Fatalf("globaldb.OpenGlobalDB() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(testutil.Context(t)); err != nil {
			t.Fatalf("db.Close() error = %v", err)
		}
	})

	kernel, err := resources.NewKernel(db.DB())
	if err != nil {
		t.Fatalf("resources.NewKernel() error = %v", err)
	}
	codec, err := aghconfig.NewMCPServerResourceCodec()
	if err != nil {
		t.Fatalf("NewMCPServerResourceCodec() error = %v", err)
	}
	store, err := resources.NewStore(kernel, codec)
	if err != nil {
		t.Fatalf("resources.NewStore() error = %v", err)
	}

	actor := resources.MutationActor{
		Kind: resources.MutationActorKindDaemon,
		ID:   "config-tests",
		Source: resources.ResourceSource{
			Kind: resources.ResourceSourceKind("daemon"),
			ID:   "config-tests",
		},
		MaxScope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
	}

	record, err := store.Put(testutil.Context(t), actor, resources.Draft[aghconfig.MCPServer]{
		ID:    "git",
		Scope: resources.ResourceScope{Kind: resources.ResourceScopeKindGlobal},
		Spec: aghconfig.MCPServer{
			Name:    " git ",
			Command: " npx ",
			Args:    []string{" --stdio "},
			Env: map[string]string{
				" TOKEN ": " secret ",
			},
		},
	})
	if err != nil {
		t.Fatalf("store.Put() error = %v", err)
	}

	if got, want := record.Spec.Name, "git"; got != want {
		t.Fatalf("record.Spec.Name = %q, want %q", got, want)
	}
	if got, want := record.Spec.Command, "npx"; got != want {
		t.Fatalf("record.Spec.Command = %q, want %q", got, want)
	}
	if got, want := record.Spec.Args[0], "--stdio"; got != want {
		t.Fatalf("record.Spec.Args[0] = %q, want %q", got, want)
	}
	if got, want := record.Spec.Env["TOKEN"], "secret"; got != want {
		t.Fatalf("record.Spec.Env[TOKEN] = %q, want %q", got, want)
	}

	listed, err := store.List(testutil.Context(t), actor, resources.ResourceFilter{})
	if err != nil {
		t.Fatalf("store.List() error = %v", err)
	}
	if got, want := len(listed), 1; got != want {
		t.Fatalf("len(store.List()) = %d, want %d", got, want)
	}
	if listed[0].Spec.Name != "git" {
		t.Fatalf("listed[0].Spec = %#v, want typed normalized git server", listed[0].Spec)
	}
}
