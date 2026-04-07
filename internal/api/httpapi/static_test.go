package httpapi

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
	"testing"
)

func TestStaticRoutesServeEmbeddedIndexForRootAndDeepLinks(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	rootResp := performRequest(t, engine, http.MethodGet, "/", nil)
	if rootResp.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d; body=%s", rootResp.Code, http.StatusOK, rootResp.Body.String())
	}
	if !strings.Contains(rootResp.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("GET / body = %q, want SPA shell", rootResp.Body.String())
	}

	deepLinkResp := performRequest(t, engine, http.MethodGet, "/session/sess-001", nil)
	if deepLinkResp.Code != http.StatusOK {
		t.Fatalf(
			"GET /session/sess-001 status = %d, want %d; body=%s",
			deepLinkResp.Code,
			http.StatusOK,
			deepLinkResp.Body.String(),
		)
	}
	if !strings.Contains(deepLinkResp.Body.String(), `<div id="app"></div>`) {
		t.Fatalf("GET /session/sess-001 body = %q, want SPA shell", deepLinkResp.Body.String())
	}
}

func TestStaticRoutesServeEmbeddedAssets(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	requestPath, expected := firstEmbeddedAsset(t)
	resp := performRequest(t, engine, http.MethodGet, requestPath, nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, want %d; body=%s", requestPath, resp.Code, http.StatusOK, resp.Body.String())
	}
	if got, want := resp.Body.String(), string(expected); got != want {
		t.Fatalf("GET %s body mismatch", requestPath)
	}
	if strings.Contains(resp.Body.String(), "<!doctype html>") {
		t.Fatalf("GET %s returned SPA HTML instead of asset payload", requestPath)
	}
}

func TestStaticRoutesDoNotFallbackForMissingAssetsOrAPIRoutes(t *testing.T) {
	homePaths := newTestHomePaths(t)
	engine := newTestRouter(t, newTestHandlers(t, stubSessionManager{}, stubObserver{}, homePaths))

	missingAssetResp := performRequest(t, engine, http.MethodGet, "/assets/does-not-exist.js", nil)
	if missingAssetResp.Code != http.StatusNotFound {
		t.Fatalf(
			"GET missing asset status = %d, want %d; body=%s",
			missingAssetResp.Code,
			http.StatusNotFound,
			missingAssetResp.Body.String(),
		)
	}
	if strings.Contains(missingAssetResp.Body.String(), "<!doctype html>") {
		t.Fatalf("GET missing asset body = %q, want plain 404", missingAssetResp.Body.String())
	}

	missingAPIResp := performRequest(t, engine, http.MethodGet, "/api/missing", nil)
	if missingAPIResp.Code != http.StatusNotFound {
		t.Fatalf(
			"GET /api/missing status = %d, want %d; body=%s",
			missingAPIResp.Code,
			http.StatusNotFound,
			missingAPIResp.Body.String(),
		)
	}
	if strings.Contains(missingAPIResp.Body.String(), "<!doctype html>") {
		t.Fatalf("GET /api/missing body = %q, want plain 404", missingAPIResp.Body.String())
	}
}

func firstEmbeddedAsset(t *testing.T) (string, []byte) {
	t.Helper()

	staticFS := mustStaticFS(t)
	entries, err := fs.ReadDir(staticFS, "assets")
	if err != nil {
		t.Fatalf("fs.ReadDir(assets) error = %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch ext := path.Ext(entry.Name()); ext {
		case ".js", ".css":
			assetPath := path.Join("assets", entry.Name())
			data, readErr := fs.ReadFile(staticFS, assetPath)
			if readErr != nil {
				t.Fatalf("fs.ReadFile(%s) error = %v", assetPath, readErr)
			}
			return "/" + assetPath, data
		}
	}

	t.Fatal("expected at least one embedded .js or .css asset")
	return "", nil
}
