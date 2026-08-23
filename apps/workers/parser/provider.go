package parser

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ErrLLMUnavailable marks LLM failures that retrying cannot fix in-process:
// the provider is disabled, unreachable, or timing out at the transport level.
// The extractor fails such messages fast instead of burning the retry budget.
var ErrLLMUnavailable = errors.New("llm unavailable")

// ErrLLMDisabled marks the operator-chosen regex-only mode (OLLAMA_ENABLED=false).
// It wraps ErrLLMUnavailable so availability handling still applies, but lets the
// extractor record a regex miss as a plain parse failure rather than an outage.
var ErrLLMDisabled = fmt.Errorf("%w: disabled via OLLAMA_ENABLED=false", ErrLLMUnavailable)

// LLMProvider extracts structured ride data from a freeform message.
// Implementations must be safe for concurrent use.
type LLMProvider interface {
	Extract(ctx context.Context, content, groupName string) (*ParsedRide, error)
}

// NewLLMProvider reads env vars and returns a configured OllamaProvider.
//
// Env vars (all optional):
//
//	OLLAMA_ENABLED   set to "false" to disable the LLM fallback entirely
//	                 (regex-only parsing; regex misses fail fast)
//	OLLAMA_BASE_URL  defaults to http://localhost:11434/v1
//	OLLAMA_MODEL     defaults to qwen2.5:3b
//	OLLAMA_API_KEY   empty for local Ollama; set to your Ollama Cloud token for cloud
func NewLLMProvider(logger *slog.Logger) LLMProvider {
	if strings.EqualFold(os.Getenv("OLLAMA_ENABLED"), "false") {
		logger.Info("llm provider disabled (OLLAMA_ENABLED=false) — regex-only parsing")
		return disabledProvider{}
	}

	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}

	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen2.5:3b"
	}

	apiKey := os.Getenv("OLLAMA_API_KEY")

	logger.Info(
		"llm provider configured",
		"base_url", baseURL,
		"model", model,
		"auth", apiKey != "",
	)

	return newOllamaProvider(baseURL, model, apiKey, logger)
}

// disabledProvider is returned when OLLAMA_ENABLED=false. It performs no
// network calls; every Extract reports the provider as unavailable.
type disabledProvider struct{}

func (disabledProvider) Extract(context.Context, string, string) (*ParsedRide, error) {
	return nil, ErrLLMDisabled
}
