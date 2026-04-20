package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"project-neo/shared/model"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

type ollamaResponse struct {
	RideType          *string  `json:"ride_type"`
	FromLocation      *string  `json:"from_location"`
	ToLocation        *string  `json:"to_location"`
	IsImmediate       bool     `json:"is_immediate"`
	DepartureTimeText *string  `json:"departure_time_text"`
	Cost              *float64 `json:"cost"`
	Currency          *string  `json:"currency"`
	NotARideMessage   bool     `json:"not_a_ride_message"`
}

// OllamaProvider implements LLMProvider using an OpenAI-compatible Ollama endpoint.
// Works with local Ollama (http://localhost:11434/v1) and Ollama Cloud (https://ollama.com/v1).
type OllamaProvider struct {
	client openai.Client
	model  string
	logger *slog.Logger
}

// newOllamaProvider constructs the provider eagerly so env vars are read and
// logged only once at startup.
func newOllamaProvider(baseURL, llmModel, apiKey string, logger *slog.Logger) *OllamaProvider {
	client := openai.NewClient(
		option.WithBaseURL(baseURL),
		option.WithAPIKey(apiKey),
	)
	return &OllamaProvider{client: client, model: llmModel, logger: logger}
}

// buildSystemPrompt constructs the extraction prompt with group context.
func buildSystemPrompt(groupName string) string {
	return fmt.Sprintf(`You are a ride-sharing message parser. Extract ride information from the message.
Group context: %s

Return ONLY a JSON object with no markdown formatting:
{
  "ride_type": "need_ride" | "ride_available" | null,
  "from_location": "string" | null,
  "to_location": "string" | null,
  "is_immediate": true | false,
  "departure_time_text": "string" | null,
  "cost": number | null,
  "currency": "string" | null,
  "not_a_ride_message": true | false
}

If this is not a ride message at all, set not_a_ride_message: true.`, groupName)
}

// Extract calls the Ollama API and parses the response into a ParsedRide.
// Returns ErrNotARide when the message is not a ride request/offer.
func (p *OllamaProvider) Extract(ctx context.Context, content, groupName string) (*ParsedRide, error) {
	rfp := shared.NewResponseFormatJSONObjectParam()

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(buildSystemPrompt(groupName)),
			openai.UserMessage(content),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &rfp,
		},
		MaxTokens: param.NewOpt[int64](512),
	})
	if err != nil {
		return nil, fmt.Errorf("ollama api: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("ollama: empty response")
	}

	raw := resp.Choices[0].Message.Content

	var or ollamaResponse
	if err := json.Unmarshal([]byte(raw), &or); err != nil {
		return nil, fmt.Errorf("ollama: unmarshal response: %w", err)
	}

	if or.NotARideMessage {
		return nil, ErrNotARide
	}

	if or.RideType == nil || (or.FromLocation == nil && or.ToLocation == nil) {
		return nil, fmt.Errorf("ollama: missing ride type or locations")
	}

	parsed := &ParsedRide{
		IsImmediate:      or.IsImmediate,
		Cost:             or.Cost,
		Currency:         or.Currency,
		FromLocationText: or.FromLocation,
		ToLocationText:   or.ToLocation,
	}

	switch *or.RideType {
	case "need_ride":
		parsed.RideType = model.RideTypeNeedRide
	case "ride_available":
		parsed.RideType = model.RideTypeRideAvailable
	default:
		return nil, fmt.Errorf("ollama: unknown ride_type %q", *or.RideType)
	}

	p.logger.Debug("ollama parsed (regex miss)",
		"content", content,
		"ride_type", parsed.RideType,
		"from", parsed.FromLocationText,
		"to", parsed.ToLocationText,
	)

	return parsed, nil
}
