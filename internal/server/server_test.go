package server

import (
	"net/http"
	"testing"

	"gemini-web-to-api/internal/auth"
	"gemini-web-to-api/internal/commons/configs"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func TestProtocolRoutesRequireGatewayKey(t *testing.T) {
	cfg := &configs.Config{
		Auth: configs.AuthConfig{Enabled: true, DataDir: t.TempDir()},
	}
	manager, err := auth.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app := NewGeminiWebToAPI(zap.NewNop(), cfg, manager)
	for _, prefix := range []string{"/openai/v1", "/claude/v1", "/gemini/v1beta"} {
		app.Get(prefix+"/probe", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })
		request, err := http.NewRequest(http.MethodGet, "http://localhost"+prefix+"/probe", nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without key: got %d, want 401", prefix, response.StatusCode)
		}

		request.Header.Set("Authorization", "Bearer "+manager.Key())
		response, err = app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusNoContent {
			t.Errorf("%s with key: got %d, want 204", prefix, response.StatusCode)
		}
	}
}

func TestPublicRoutesAndAPIPreflight(t *testing.T) {
	cfg := &configs.Config{Auth: configs.AuthConfig{Enabled: true, DataDir: t.TempDir()}}
	manager, err := auth.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app := NewGeminiWebToAPI(zap.NewNop(), cfg, manager)
	for _, path := range []string{"/health", "/docs", "/openapi.json"} {
		request, err := http.NewRequest(http.MethodGet, "http://localhost"+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Errorf("%s: got %d, want 200", path, response.StatusCode)
		}
	}
	request, err := http.NewRequest(http.MethodOptions, "http://localhost/openai/v1/models", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "https://example.org")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "authorization")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("preflight: got %d, want 204", response.StatusCode)
	}
	if response.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("preflight missing API CORS allowance")
	}
}

func TestAPIPreflightBypassesRateLimit(t *testing.T) {
	cfg := &configs.Config{
		Auth:      configs.AuthConfig{Enabled: true, DataDir: t.TempDir()},
		RateLimit: configs.RateLimitConfig{Enabled: true, MaxRequests: 1, WindowMs: 60000},
	}
	manager, err := auth.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app := NewGeminiWebToAPI(zap.NewNop(), cfg, manager)
	app.Get("/openai/v1/probe", func(c fiber.Ctx) error { return c.SendStatus(http.StatusNoContent) })
	request, err := http.NewRequest(http.MethodGet, "http://localhost/openai/v1/probe", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+manager.Key())
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("initial request: got %d, want 204", response.StatusCode)
	}
	preflight, err := http.NewRequest(http.MethodOptions, "http://localhost/openai/v1/probe", nil)
	if err != nil {
		t.Fatal(err)
	}
	preflight.Header.Set("Origin", "https://example.org")
	preflight.Header.Set("Access-Control-Request-Method", http.MethodGet)
	response, err = app.Test(preflight)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Errorf("preflight after limit reached: got %d, want 204", response.StatusCode)
	}
}
