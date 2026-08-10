package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestNewHandlerExposesBridgeHealth(t *testing.T) {
	h, err := newHandler(filepath.Join(t.TempDir(), "data"))
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", r.Code, r.Body.String())
	}
}
