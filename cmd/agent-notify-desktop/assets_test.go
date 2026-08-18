package main

import (
	"io/fs"
	"strings"
	"testing"
)

func TestAssetRootServesBuiltFrontend(t *testing.T) {
	root, err := assetRoot()
	if err != nil {
		t.Fatal(err)
	}
	index, err := fs.ReadFile(root, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	page := string(index)
	if !strings.Contains(page, `href="/main.css"`) || !strings.Contains(page, `src="/main.js"`) {
		t.Fatalf("index must use asset-root URLs, got %s", page)
	}
	for _, name := range []string{"main.css", "main.js"} {
		if _, err := fs.ReadFile(root, name); err != nil {
			t.Fatalf("asset %s is not served from the frontend root: %v", name, err)
		}
	}
}

func TestBuiltFrontendKeepsDraggableTitleBar(t *testing.T) {
	root, err := assetRoot()
	if err != nil {
		t.Fatal(err)
	}
	css, err := fs.ReadFile(root, "main.css")
	if err != nil {
		t.Fatal(err)
	}
	page := string(css)
	for _, want := range []string{"--wails-draggable: drag", "--wails-draggable: no-drag"} {
		if !strings.Contains(page, want) {
			t.Fatalf("main.css missing %q", want)
		}
	}
}

func TestBuiltFrontendKeepsTitleBarSafeArea(t *testing.T) {
	root, err := assetRoot()
	if err != nil {
		t.Fatal(err)
	}
	css, err := fs.ReadFile(root, "main.css")
	if err != nil {
		t.Fatal(err)
	}
	page := string(css)
	for _, want := range []string{"height: 88px", "padding: 28px 28px 0"} {
		if !strings.Contains(page, want) {
			t.Fatalf("main.css missing %q", want)
		}
	}
}

func TestBuiltFrontendFitsDefaultWindowHeight(t *testing.T) {
	root, err := assetRoot()
	if err != nil {
		t.Fatal(err)
	}
	css, err := fs.ReadFile(root, "main.css")
	if err != nil {
		t.Fatal(err)
	}
	page := string(css)
	for _, want := range []string{
		"height: calc(100vh - 88px)",
		"overflow: hidden",
		"grid-template-columns: repeat(8, minmax(0, 1fr))",
		"padding: 12px 0",
	} {
		if !strings.Contains(page, want) {
			t.Fatalf("main.css missing compact layout rule %q", want)
		}
	}
}
