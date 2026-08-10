package bridge

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func EnsureToken(path string) ([]byte, error) {
	if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) != "" {
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, err
		}
		return []byte(strings.TrimSpace(string(data))), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	token := []byte(hex.EncodeToString(raw))
	if err := os.WriteFile(path, token, 0o600); err != nil {
		return nil, err
	}
	return token, nil
}

func validToken(got, want []byte) bool {
	return len(got) == len(want) && subtle.ConstantTimeCompare(got, want) == 1
}
func bearerToken(header string) ([]byte, error) {
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return nil, errors.New("missing bearer token")
	}
	value := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if value == "" {
		return nil, errors.New("empty bearer token")
	}
	return []byte(value), nil
}
