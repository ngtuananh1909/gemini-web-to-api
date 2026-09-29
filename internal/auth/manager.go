package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gemini-web-to-api/internal/commons/configs"

	"github.com/gofiber/fiber/v3"
)

type Manager struct {
	mu         sync.RWMutex
	enabled    bool
	configured bool
	key        string
	keyPath    string
}

func NewManager(cfg *configs.Config) (*Manager, error) {
	manager := &Manager{enabled: cfg.Auth.Enabled, configured: cfg.Auth.APIKey != ""}
	if !manager.enabled {
		return manager, nil
	}
	dir := cfg.Auth.DataDir
	if dir == "" {
		dir = ".gateway"
	}
	if err := prepareDirectory(dir); err != nil {
		return nil, err
	}
	manager.keyPath = filepath.Join(dir, "api-key")
	existing, err := readKeyFile(manager.keyPath)
	if err != nil {
		return nil, err
	}
	if manager.configured {
		manager.key = cfg.Auth.APIKey
		if existing != manager.key {
			if err := writeKeyFile(manager.keyPath, manager.key); err != nil {
				return nil, err
			}
		}
		return manager, nil
	}
	if existing != "" {
		manager.key = existing
		return manager, nil
	}
	manager.key, err = generateKey()
	if err != nil {
		return nil, err
	}
	if err := writeKeyFile(manager.keyPath, manager.key); err != nil {
		return nil, err
	}
	return manager, nil
}

func prepareDirectory(dir string) error {
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create gateway data directory: %w", err)
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return fmt.Errorf("inspect gateway data directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("gateway data directory must be a real directory")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("secure gateway data directory: %w", err)
	}
	return nil
}

func readKeyFile(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect gateway key file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return "", errors.New("gateway key file must be a private regular file")
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read gateway key file: %w", err)
	}
	key := strings.TrimSpace(string(contents))
	if key == "" {
		return "", errors.New("gateway key file is empty")
	}
	return key, nil
}

func writeKeyFile(path, key string) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".api-key-*")
	if err != nil {
		return fmt.Errorf("create temporary gateway key file: %w", err)
	}
	tempPath := file.Name()
	defer os.Remove(tempPath)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("secure temporary gateway key file: %w", err)
	}
	if _, err := file.WriteString(key + "\n"); err != nil {
		file.Close()
		return fmt.Errorf("write gateway key file: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync gateway key file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close gateway key file: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("replace gateway key file: %w", err)
	}
	return nil
}

func generateKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate gateway key: %w", err)
	}
	return "gw_" + hex.EncodeToString(bytes), nil
}

func (manager *Manager) Key() string {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	return manager.key
}

func (manager *Manager) Enabled() bool {
	return manager.enabled
}

func (manager *Manager) RotationAllowed() bool {
	return manager.enabled && !manager.configured
}

func (manager *Manager) Rotate() (string, error) {
	if !manager.RotationAllowed() {
		return "", errors.New("gateway key rotation is unavailable while API_KEY is configured or authentication is disabled")
	}
	key, err := generateKey()
	if err != nil {
		return "", err
	}
	manager.mu.Lock()
	defer manager.mu.Unlock()
	if err := writeKeyFile(manager.keyPath, key); err != nil {
		return "", err
	}
	manager.key = key
	return key, nil
}

func (manager *Manager) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !manager.enabled {
			return c.Next()
		}
		var candidates []string
		if authorization := c.Get("Authorization"); authorization != "" {
			scheme, value, ok := strings.Cut(authorization, " ")
			if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(value) == "" {
				return unauthorized(c)
			}
			candidates = append(candidates, strings.TrimSpace(value))
		}
		for _, header := range []string{"x-api-key", "x-goog-api-key"} {
			if value := c.Get(header); value != "" {
				candidates = append(candidates, value)
			}
		}
		if len(candidates) == 0 {
			return unauthorized(c)
		}
		manager.mu.RLock()
		key := manager.key
		manager.mu.RUnlock()
		for _, candidate := range candidates {
			if subtle.ConstantTimeCompare([]byte(candidate), []byte(key)) != 1 {
				return unauthorized(c)
			}
		}
		return c.Next()
	}
}

func unauthorized(c fiber.Ctx) error {
	return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
		"error": fiber.Map{"message": "Invalid gateway API key", "type": "authentication_error"},
	})
}
