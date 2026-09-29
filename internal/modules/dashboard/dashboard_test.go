package dashboard

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"gemini-web-to-api/internal/auth"
	"gemini-web-to-api/internal/commons/configs"
	"gemini-web-to-api/internal/modules/providers"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func TestDashboardRequiresLocalHostAndProtectsKeyControls(t *testing.T) {
	cfg := &configs.Config{
		Auth:      configs.AuthConfig{Enabled: true, DataDir: t.TempDir()},
		Dashboard: configs.DashboardConfig{Enabled: true},
	}
	manager, err := auth.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	app := fiber.New()
	RegisterRoutes(app, cfg, manager, providers.NewClient(cfg, zap.NewNop()), providers.NewRuntimeStatus())
	request := func(method, path, host, origin, contentType, action string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, "http://"+host+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		if action != "" {
			req.Header.Set("X-Dashboard-Action", action)
		}
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return response
	}
	response := request(http.MethodGet, "/", "localhost:4981", "", "", "")
	body, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), "Gemini Web API Gateway") {
		t.Fatalf("homepage status %d, body %q", response.StatusCode, body)
	}
	response = request(http.MethodGet, "/dashboard/api/status", "localhost:4981", "", "", "")
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"status":"connecting"`) {
		t.Fatalf("status response %d, body %q", response.StatusCode, body)
	}
	response = request(http.MethodGet, "/dashboard/api/connection", "localhost:4981", "", "", "")
	body, _ = io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), manager.Key()) || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("connection status %d, cache %q", response.StatusCode, response.Header.Get("Cache-Control"))
	}
	response = request(http.MethodGet, "/dashboard/api/connection", "gateway.example", "", "", "")
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Errorf("remote host got %d, want 403", response.StatusCode)
	}
	oldKey := manager.Key()
	response = request(http.MethodPost, "/dashboard/api/api-key/rotate", "localhost:4981", "", "application/json", "rotate")
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden || manager.Key() != oldKey {
		t.Errorf("cross-origin rotation got %d", response.StatusCode)
	}
	response = request(http.MethodPost, "/dashboard/api/api-key/rotate", "localhost:4981", "http://localhost:4981", "application/json", "rotate")
	response.Body.Close()
	if response.StatusCode != http.StatusOK || manager.Key() == oldKey {
		t.Errorf("local rotation got %d", response.StatusCode)
	}
}

func TestDashboardCanBeDisabled(t *testing.T) {
	cfg := &configs.Config{Dashboard: configs.DashboardConfig{Enabled: false}}
	app := fiber.New()
	RegisterRoutes(app, cfg, nil, nil, nil)
	request, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Errorf("disabled dashboard got %d, want 404", response.StatusCode)
	}
}
