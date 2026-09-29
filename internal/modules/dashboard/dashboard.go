package dashboard

import (
	"embed"
	"fmt"
	"strings"

	"gemini-web-to-api/internal/auth"
	"gemini-web-to-api/internal/commons/configs"
	"gemini-web-to-api/internal/modules/providers"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/fx"
)

//go:embed assets/*
var assets embed.FS

var Module = fx.Options(fx.Invoke(RegisterRoutes))

func RegisterRoutes(app *fiber.App, cfg *configs.Config, manager *auth.Manager, client *providers.Client, status *providers.RuntimeStatus) {
	if !cfg.Dashboard.Enabled {
		return
	}
	app.Use("/dashboard", localOnly)
	app.Get("/", localOnly, func(c fiber.Ctx) error {
		c.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self'; base-uri 'none'; object-src 'none'; form-action 'none'; frame-ancestors 'none'")
		c.Set("Referrer-Policy", "no-referrer")
		return serveAsset(c, "assets/index.html", "text/html; charset=utf-8")
	})
	app.Get("/dashboard/app.css", func(c fiber.Ctx) error {
		return serveAsset(c, "assets/app.css", "text/css; charset=utf-8")
	})
	app.Get("/dashboard/app.js", func(c fiber.Ctx) error {
		return serveAsset(c, "assets/app.js", "text/javascript; charset=utf-8")
	})
	app.Get("/dashboard/api/status", func(c fiber.Ctx) error {
		return c.JSON(status.Snapshot(client.IsHealthy(), len(client.ListModels())))
	})
	app.Get("/dashboard/api/connection", func(c fiber.Ctx) error {
		privateResponse(c)
		return c.JSON(fiber.Map{
			"apiKey":          manager.Key(),
			"authEnabled":     manager.Enabled(),
			"rotationAllowed": manager.RotationAllowed(),
		})
	})
	app.Post("/dashboard/api/api-key/rotate", func(c fiber.Ctx) error {
		if !sameOriginJSON(c) {
			return c.SendStatus(fiber.StatusForbidden)
		}
		if !manager.RotationAllowed() {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Rotation is unavailable while API_KEY is configured or authentication is disabled."})
		}
		if _, err := manager.Rotate(); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not persist the new gateway key."})
		}
		privateResponse(c)
		return c.JSON(fiber.Map{"rotated": true})
	})
}

func localOnly(c fiber.Ctx) error {
	switch strings.ToLower(c.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return c.Next()
	default:
		return c.SendStatus(fiber.StatusForbidden)
	}
}

func sameOriginJSON(c fiber.Ctx) bool {
	origin := c.Get("Origin")
	if origin == "" || origin != fmt.Sprintf("%s://%s", c.Scheme(), c.Host()) {
		return false
	}
	if !strings.HasPrefix(strings.ToLower(c.Get("Content-Type")), "application/json") {
		return false
	}
	return c.Get("X-Dashboard-Action") == "rotate"
}

func privateResponse(c fiber.Ctx) {
	c.Set("Cache-Control", "no-store")
	c.Set("Referrer-Policy", "no-referrer")
}

func serveAsset(c fiber.Ctx, path, contentType string) error {
	contents, err := assets.ReadFile(path)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	c.Set("Content-Type", contentType)
	return c.Send(contents)
}
