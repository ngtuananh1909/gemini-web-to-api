package configs

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Gemini    GeminiConfig
	Claude    ClaudeConfig
	OpenAI    OpenAIConfig
	Server    ServerConfig
	Auth      AuthConfig
	Dashboard DashboardConfig
	RateLimit RateLimitConfig
	LogLevel  string
}

type AuthConfig struct {
	Enabled bool
	APIKey  string
	DataDir string
}

type DashboardConfig struct {
	Enabled             bool
	LocalPortGuaranteed bool
}

type RateLimitConfig struct {
	Enabled     bool
	WindowMs    int
	MaxRequests int
}

type GeminiConfig struct {
	Secure1PSID     string
	Secure1PSIDTS   string
	AuthUser        string
	RefreshInterval int
	MaxRetries      int
	Cookies         string
	Temporary       bool
}

type ClaudeConfig struct {
	APIKey  string
	Model   string
	Cookies string
}

type OpenAIConfig struct {
	APIKey  string
	Model   string
	Cookies string
}

type ServerConfig struct {
	Host string
	Port string
}

const (
	defaultServerPort            = "4981"
	defaultGeminiRefreshInterval = 5
	defaultGeminiMaxRetries      = 3
	defaultLogLevel              = "info"
)

func New() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	var cfg Config

	// Server
	cfg.Server.Host = strings.TrimSpace(getEnv("HOST", "127.0.0.1"))
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	cfg.Server.Port = getEnv("PORT", defaultServerPort)
	cfg.Auth.Enabled = getEnvBool("API_AUTH_ENABLED", true)
	cfg.Auth.APIKey = strings.TrimSpace(os.Getenv("API_KEY"))
	cfg.Auth.DataDir = strings.TrimSpace(getEnv("GATEWAY_DATA_DIR", ".gateway"))
	if cfg.Auth.DataDir == "" {
		cfg.Auth.DataDir = ".gateway"
	}
	cfg.Dashboard.Enabled = getEnvBool("DASHBOARD_ENABLED", loopbackHost(cfg.Server.Host))
	cfg.Dashboard.LocalPortGuaranteed = getEnvBool("DASHBOARD_LOCAL_PORT", false)

	// General
	cfg.LogLevel = getEnv("LOG_LEVEL", defaultLogLevel)

	// Rate Limit
	cfg.RateLimit.Enabled = getEnvBool("RATE_LIMIT_ENABLED", false)
	cfg.RateLimit.WindowMs = getEnvInt("RATE_LIMIT_WINDOW_MS", 60000)
	cfg.RateLimit.MaxRequests = getEnvInt("RATE_LIMIT_MAX_REQUESTS", 10)

	// Gemini
	cfg.Gemini.Cookies = os.Getenv("GEMINI_COOKIES")
	cfg.Gemini.AuthUser = strings.TrimSpace(os.Getenv("GEMINI_AUTH_USER"))
	cfg.Gemini.Secure1PSID = cookieValue(cfg.Gemini.Cookies, "__Secure-1PSID")
	cfg.Gemini.Secure1PSIDTS = cookieValue(cfg.Gemini.Cookies, "__Secure-1PSIDTS")
	cfg.Gemini.RefreshInterval = getEnvInt("GEMINI_REFRESH_INTERVAL", defaultGeminiRefreshInterval)
	cfg.Gemini.MaxRetries = getEnvInt("GEMINI_MAX_RETRIES", defaultGeminiMaxRetries)
	cfg.Gemini.Temporary = getEnvBool("GEMINI_TEMPORARY", false)
	if cfg.Gemini.AuthUser != "" {
		if n, err := strconv.Atoi(cfg.Gemini.AuthUser); err != nil || n < 0 {
			return nil, fmt.Errorf("invalid GEMINI_AUTH_USER value: %q (must be a non-negative account slot)", cfg.Gemini.AuthUser)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func cookieValue(header, wanted string) string {
	for _, pair := range strings.Split(header, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if ok && strings.TrimSpace(name) == wanted {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

// Validate checks if the configuration has required values
func (c *Config) Validate() error {
	var missingVars []string
	if c.Dashboard.Enabled && !loopbackHost(c.Server.Host) && !c.Dashboard.LocalPortGuaranteed {
		return fmt.Errorf("dashboard requires a loopback HOST or DASHBOARD_LOCAL_PORT=true with a verified loopback-only host port mapping")
	}

	if strings.TrimSpace(c.Gemini.Cookies) == "" {
		missingVars = append(missingVars, "GEMINI_COOKIES")
	} else if c.Gemini.Secure1PSID == "" || c.Gemini.Secure1PSIDTS == "" {
		return fmt.Errorf("GEMINI_COOKIES must contain __Secure-1PSID and __Secure-1PSIDTS")
	}

	// Check Server port is valid
	if c.Server.Port == "" {
		c.Server.Port = defaultServerPort
	}

	if _, err := strconv.Atoi(c.Server.Port); err != nil {
		return fmt.Errorf("invalid PORT value: %q (must be a number)", c.Server.Port)
	}

	if len(missingVars) > 0 {
		return fmt.Errorf("missing required environment variables: %v. Please set them before running the application", missingVars)
	}

	return nil
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func getEnvBool(key string, defaultValue bool) bool {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
