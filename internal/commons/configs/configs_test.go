package configs

import (
	"strings"
	"testing"
)

func TestCookieValueExtractsAuthFromFullHeader(t *testing.T) {
	header := "NID=rollout; __Secure-1PSID=account=value; __Secure-1PSIDTS=timestamp; SIDCC=flags"
	if got := cookieValue(header, "__Secure-1PSID"); got != "account=value" {
		t.Fatalf("1PSID = %q", got)
	}
	if got := cookieValue(header, "__Secure-1PSIDTS"); got != "timestamp" {
		t.Fatalf("1PSIDTS = %q", got)
	}
}

func TestNewRejectsInvalidGeminiAuthUser(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("GEMINI_AUTH_USER", "account-two")
	if _, err := New(); err == nil || !strings.Contains(err.Error(), "GEMINI_AUTH_USER") {
		t.Fatalf("expected invalid account slot error, got %v", err)
	}
}

func TestNewRequiresGeminiCookiesHeaderWithPSID(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "")
	// Legacy variables must not create a second authentication path.
	t.Setenv("GEMINI_1PSID", "legacy-psid")
	t.Setenv("GEMINI_1PSIDTS", "legacy-psidts")
	if _, err := New(); err == nil || !strings.Contains(err.Error(), "GEMINI_COOKIES") {
		t.Fatalf("expected GEMINI_COOKIES requirement, got %v", err)
	}

	t.Setenv("GEMINI_COOKIES", "NID=rollout; SIDCC=some-flag")
	if _, err := New(); err == nil || !strings.Contains(err.Error(), "__Secure-1PSID") {
		t.Fatalf("expected missing __Secure-1PSID error, got %v", err)
	}
}

func TestNewAcceptsPSIDOnlyOrRawToken(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "NID=rollout; __Secure-1PSID=psid-only")
	cfg, err := New()
	if err != nil {
		t.Fatalf("expected success with PSID only, got %v", err)
	}
	if cfg.Gemini.Secure1PSID != "psid-only" || cfg.Gemini.Secure1PSIDTS != "" {
		t.Fatalf("unexpected config: %#v", cfg.Gemini)
	}

	// Test raw token without prefix
	t.Setenv("GEMINI_COOKIES", "raw-psid-token-value")
	cfgRaw, err := New()
	if err != nil {
		t.Fatalf("expected success with raw PSID token, got %v", err)
	}
	if cfgRaw.Gemini.Secure1PSID != "raw-psid-token-value" || cfgRaw.Gemini.Secure1PSIDTS != "" {
		t.Fatalf("unexpected raw config: %#v", cfgRaw.Gemini)
	}
}

func TestNewAcceptsCookieEditorJSON(t *testing.T) {
	jsonPayload := `[
		{"name": "NID", "value": "nid-val"},
		{"name": "__Secure-1PSID", "value": "json-psid-val"},
		{"name": "__Secure-1PSIDTS", "value": "json-psidts-val"}
	]`
	t.Setenv("GEMINI_COOKIES", jsonPayload)
	cfg, err := New()
	if err != nil {
		t.Fatalf("expected success with Cookie-Editor JSON format, got %v", err)
	}
	if cfg.Gemini.Secure1PSID != "json-psid-val" || cfg.Gemini.Secure1PSIDTS != "json-psidts-val" {
		t.Fatalf("unexpected extracted cookies: %#v", cfg.Gemini)
	}
	if !strings.Contains(cfg.Gemini.Cookies, "NID=nid-val") {
		t.Fatalf("expected converted cookie string, got %q", cfg.Gemini.Cookies)
	}
}

func TestNewDerivesAuthenticationFromGeminiCookies(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "NID=rollout; __Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("GEMINI_AUTH_USER", "2")
	cfg, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Gemini.Secure1PSID != "psid" || cfg.Gemini.Secure1PSIDTS != "psidts" || cfg.Gemini.AuthUser != "2" {
		t.Fatalf("incorrect Gemini config derived from full Cookie header: %#v", cfg.Gemini)
	}
}

func TestNewLoadsGatewayDefaultsAndOverrides(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("API_AUTH_ENABLED", "false")
	t.Setenv("API_KEY", "configured-key")
	t.Setenv("GATEWAY_DATA_DIR", "/tmp/gateway-state")
	t.Setenv("DASHBOARD_ENABLED", "false")

	cfg, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("host = %q, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Auth.Enabled || cfg.Auth.APIKey != "configured-key" || cfg.Auth.DataDir != "/tmp/gateway-state" {
		t.Errorf("unexpected auth config: %#v", cfg.Auth)
	}
	if cfg.Dashboard.Enabled {
		t.Errorf("dashboard should be disabled")
	}
}

func TestNewUsesGatewayDefaults(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	for _, name := range []string{"HOST", "API_AUTH_ENABLED", "API_KEY", "GATEWAY_DATA_DIR", "DASHBOARD_ENABLED"} {
		t.Setenv(name, "")
	}

	cfg, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("host = %q, want 127.0.0.1", cfg.Server.Host)
	}
	if !cfg.Auth.Enabled || cfg.Auth.DataDir != ".gateway" || cfg.Auth.APIKey != "" {
		t.Errorf("unexpected auth defaults: %#v", cfg.Auth)
	}
	if !cfg.Dashboard.Enabled {
		t.Errorf("dashboard should default to enabled")
	}
}

func TestNewDisablesDashboardOnPublicListenerByDefault(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("DASHBOARD_ENABLED", "")
	cfg, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dashboard.Enabled {
		t.Fatal("dashboard should default to disabled on a public listener")
	}
}

func TestNewRejectsPublicDashboardWithoutLocalPortGuarantee(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("DASHBOARD_ENABLED", "true")
	t.Setenv("DASHBOARD_LOCAL_PORT", "")
	if _, err := New(); err == nil {
		t.Fatal("public listener with dashboard enabled should require a local host port guarantee")
	}
}

func TestNewAllowsContainerListenerBehindLocalPort(t *testing.T) {
	t.Setenv("GEMINI_COOKIES", "__Secure-1PSID=psid; __Secure-1PSIDTS=psidts")
	t.Setenv("HOST", "0.0.0.0")
	t.Setenv("DASHBOARD_ENABLED", "true")
	t.Setenv("DASHBOARD_LOCAL_PORT", "true")
	cfg, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Dashboard.Enabled || !cfg.Dashboard.LocalPortGuaranteed {
		t.Fatalf("unexpected dashboard config: %#v", cfg.Dashboard)
	}
}
