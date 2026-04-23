package parser

import (
	"context"
	"log/slog"
	"os"
)

// LLMProvider extracts structured ride data from a freeform message.
// Implementations must be safe for concurrent use.
type LLMProvider interface {
	Extract(ctx context.Context, content, groupName string) (*ParsedRide, error)
}

// NewLLMProvider reads env vars and returns a configured OllamaProvider.
//
// Env vars (all optional):
//
//	OLLAMA_BASE_URL  defaults to http://localhost:11434/v1
//	OLLAMA_MODEL     defaults to qwen2.5:3b
//	OLLAMA_API_KEY   empty for local Ollama; set to your Ollama Cloud token for cloud
func NewLLMProvider(logger *slog.Logger) LLMProvider {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:3b"
	}

	apiKey := os.Getenv("OLLAMA_API_KEY")

	logger.Info("llm provider configured",
		"base_url", baseURL,
		"model", model,
		"auth", apiKey != "",
	)

	return newOllamaProvider(baseURL, model, apiKey, logger)
}
