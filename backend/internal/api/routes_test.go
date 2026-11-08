package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSPAFileServer_servesFileOrFallsBackToIndexAsHTML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<!doctype html><title>x</title>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log(1)"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := spaFileServer(dir)

	cases := []struct {
		path, wantType, wantBody string
	}{
		{"/app.js", "text/javascript", "console.log(1)"},
		// Client-side route: must get index.html with an HTML content type,
		// not the text/plain left behind by the FileServer 404.
		{"/system/unifi", "text/html", "<!doctype html>"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d", rec.Code)
			}
			if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.wantType) {
				t.Fatalf("content-type %q", ct)
			}
			if !strings.HasPrefix(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body %q", rec.Body.String())
			}
		})
	}
}
