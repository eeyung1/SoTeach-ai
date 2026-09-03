package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"soteach/api"
	"soteach/session"
	"soteach/web"
)

// newWebHandler composes the API routes and the embedded static web files the
// same way cmd/server/main.go does, so the production serving setup is
// exercised by tests rather than only by running main.
func newWebHandler(store *session.MemoryStore) http.Handler {
	mux := http.NewServeMux()
	api.AddRoutes(mux, store)
	mux.Handle("/", http.FileServer(http.FS(web.Files)))
	return mux
}

// TestRootServesIndexPage proves the static web client is served at the site
// root: GET / returns the embedded index.html (200, text/html, containing the
// page title).
func TestRootServesIndexPage(t *testing.T) {
	handler := newWebHandler(session.NewMemoryStore())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for the index page, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Fatalf("expected text/html, got %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "SoTeach") {
		t.Fatal("expected the index page to contain the title")
	}
}

// TestAssetsAndAPICoexistOnSameOrigin confirms the static "/" file handler
// does not shadow the API routes registered alongside it: / serves the page,
// /app.js serves the embedded script, and /learners/Amaka/session still
// reaches the API (404 for an unsaved learner, not the file server).
func TestAssetsAndAPICoexistOnSameOrigin(t *testing.T) {
	handler := newWebHandler(session.NewMemoryStore())

	root := httptest.NewRecorder()
	handler.ServeHTTP(root, httptest.NewRequest(http.MethodGet, "/", nil))
	if root.Code != http.StatusOK {
		t.Fatalf("expected / to serve the page, got %d", root.Code)
	}

	js := httptest.NewRecorder()
	handler.ServeHTTP(js, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if js.Code != http.StatusOK {
		t.Fatalf("expected /app.js to be served, got %d", js.Code)
	}

	api := httptest.NewRecorder()
	handler.ServeHTTP(api, httptest.NewRequest(http.MethodGet, "/learners/Amaka/session", nil))
	if api.Code != http.StatusNotFound {
		t.Fatalf("expected the API route to win over the static handler (404 for unsaved learner), got %d", api.Code)
	}
}
