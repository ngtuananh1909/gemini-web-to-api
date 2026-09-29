package auth

import (
	"encoding/hex"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gemini-web-to-api/internal/commons/configs"

	"github.com/gofiber/fiber/v3"
)

func TestNewManagerGeneratesPersistentKeyWithPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(manager.Key(), "gw_") {
		t.Fatalf("key %q does not have gateway prefix", manager.Key())
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(manager.Key(), "gw_"))
	if err != nil || len(decoded) != 32 {
		t.Fatalf("key must encode 32 random bytes, got %q (%v)", manager.Key(), err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("directory permissions = %o, want 700", info.Mode().Perm())
	}
	keyPath := filepath.Join(dir, "api-key")
	info, err = os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key permissions = %o, want 600", info.Mode().Perm())
	}

	reloaded, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Key() != manager.Key() {
		t.Errorf("persisted key = %q, want generated key", reloaded.Key())
	}
}

func TestNewManagerUsesAndPersistsEnvironmentKey(t *testing.T) {
	dir := t.TempDir()
	const configuredKey = "configured-gateway-key"
	manager, err := NewManager(&configs.Config{Auth: configs.AuthConfig{
		Enabled: true,
		APIKey:  configuredKey,
		DataDir: dir,
	}})
	if err != nil {
		t.Fatal(err)
	}
	if manager.Key() != configuredKey || manager.RotationAllowed() {
		t.Errorf("unexpected configured manager state")
	}
	if _, err := manager.Rotate(); err == nil {
		t.Fatal("rotation should fail while API_KEY is configured")
	}
	contents, err := os.ReadFile(filepath.Join(dir, "api-key"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(contents)) != configuredKey {
		t.Errorf("persisted key = %q, want configured key", contents)
	}
}

func TestNewManagerRejectsUnsafeKeyFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "api-key")); err != nil {
		t.Fatal(err)
	}
	if _, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: dir}}); err == nil {
		t.Fatal("expected unsafe symlink rejection")
	}
}

func TestRotateInvalidatesOldKeyAndPersistsNewKey(t *testing.T) {
	dir := t.TempDir()
	manager, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	oldKey := manager.Key()
	newKey, err := manager.Rotate()
	if err != nil {
		t.Fatal(err)
	}
	if newKey == oldKey {
		t.Fatal("rotation reused the old key")
	}
	app := fiber.New()
	app.Use(manager.Middleware())
	app.Get("/protected", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })
	for key, want := range map[string]int{oldKey: http.StatusUnauthorized, newKey: http.StatusNoContent} {
		request, err := http.NewRequest(http.MethodGet, "http://localhost/protected", nil)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", "Bearer "+key)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != want {
			t.Errorf("key status = %d, want %d", response.StatusCode, want)
		}
	}
	reloaded, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: dir}})
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Key() != newKey {
		t.Fatal("rotated key did not survive reload")
	}
}

func TestMiddlewareAcceptsOneValidHeaderAndRejectsMissingInvalidOrConflictingKeys(t *testing.T) {
	manager, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	app.Use(manager.Middleware())
	app.Get("/protected", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })

	cases := []struct {
		name    string
		headers map[string]string
		want    int
	}{
		{name: "missing", want: http.StatusUnauthorized},
		{name: "invalid", headers: map[string]string{"X-API-Key": "wrong"}, want: http.StatusUnauthorized},
		{name: "bearer", headers: map[string]string{"Authorization": "Bearer " + manager.Key()}, want: http.StatusNoContent},
		{name: "x-api-key", headers: map[string]string{"X-API-Key": manager.Key()}, want: http.StatusNoContent},
		{name: "goog-api-key", headers: map[string]string{"X-Goog-Api-Key": manager.Key()}, want: http.StatusNoContent},
		{name: "conflicting", headers: map[string]string{"X-API-Key": manager.Key(), "X-Goog-Api-Key": "wrong"}, want: http.StatusUnauthorized},
		{name: "url key ignored", headers: nil, want: http.StatusUnauthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			target := "http://example.test/protected"
			if tc.name == "url key ignored" {
				target += "?key=" + manager.Key()
			}
			req, err := http.NewRequest(http.MethodGet, target, nil)
			if err != nil {
				t.Fatal(err)
			}
			for name, value := range tc.headers {
				req.Header.Set(name, value)
			}
			response, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != tc.want {
				t.Errorf("status = %d, want %d", response.StatusCode, tc.want)
			}
		})
	}
}

func TestMiddlewareBypassesChecksWhenDisabled(t *testing.T) {
	manager, err := NewManager(&configs.Config{Auth: configs.AuthConfig{Enabled: false, DataDir: t.TempDir()}})
	if err != nil {
		t.Fatal(err)
	}
	if manager.Enabled() {
		t.Fatal("disabled auth manager reports enabled")
	}
	app := fiber.New()
	app.Use(manager.Middleware())
	app.Get("/protected", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })
	req, err := http.NewRequest(http.MethodGet, "http://example.test/protected", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want 204", response.StatusCode)
	}
}
