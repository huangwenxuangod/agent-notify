package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hellolib/agent-notify/internal/bridge"
)

func newHandler(dataDir string) (http.Handler, error) {
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, err
	}
	token, err := bridge.EnsureToken(filepath.Join(dataDir, "bridge.token"))
	if err != nil {
		return nil, err
	}
	svc, err := bridge.NewService(bridge.Options{
		ConfigPath: filepath.Join(dataDir, "config.yaml"),
		StatePath:  filepath.Join(dataDir, "state.json"),
		LogPath:    filepath.Join(dataDir, "agent-notify.log"),
		BinaryPath: "/usr/local/bin/agent-notify",
	})
	if err != nil {
		return nil, err
	}
	return bridge.NewHTTPHandler(svc, token), nil
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "45175"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/var/lib/agent-notify"
	}
	h, err := newHandler(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("control plane listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, h))
}
