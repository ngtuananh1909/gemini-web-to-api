package modules_test

import (
	"testing"

	"gemini-web-to-api/internal/commons/configs"
	"gemini-web-to-api/internal/modules/claude"
	"gemini-web-to-api/internal/modules/gemini"
	"gemini-web-to-api/internal/modules/openai"
	"gemini-web-to-api/internal/modules/providers"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

func TestProtocolRouteContract(t *testing.T) {
	logger := zap.NewNop()
	client := providers.NewClient(&configs.Config{}, logger)
	app := fiber.New()
	openai.NewOpenAIController(openai.NewOpenAIService(client, logger), logger).Register(app.Group("/openai/v1"))
	claude.NewClaudeController(claude.NewClaudeService(client, logger), logger).Register(app.Group("/claude/v1"))
	geminiController := gemini.NewGeminiController(gemini.NewGeminiService(client, logger), logger)
	defer geminiController.Close()
	geminiController.Register(app.Group("/gemini/v1beta"))

	registered := make(map[string]bool)
	for _, route := range app.GetRoutes() {
		registered[route.Method+" "+route.Path] = true
	}
	for _, route := range []string{
		"GET /openai/v1/models",
		"GET /openai/v1/models/:model",
		"POST /openai/v1/chat/completions",
		"POST /openai/v1/images/generations",
		"GET /claude/v1/models",
		"POST /claude/v1/messages",
		"POST /claude/v1/messages/count_tokens",
		"GET /gemini/v1beta/models",
		"POST /gemini/v1beta/deepresearch",
		"POST /gemini/v1beta/interactions",
		"GET /gemini/v1beta/interactions/:id",
	} {
		if !registered[route] {
			t.Errorf("missing existing route %s", route)
		}
	}
}
