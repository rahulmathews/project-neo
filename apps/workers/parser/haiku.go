package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"project-neo/shared/model"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var (
	haikuClient     anthropic.Client //nolint:gochecknoglobals // singleton lazy-init via sync.Once
	haikuClientOnce sync.Once        //nolint:gochecknoglobals // singleton lazy-init via sync.Once
)

func getHaikuClient() (*anthropic.Client, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not configured")
	}
	haikuClientOnce.Do(func() {
		haikuClient = anthropic.NewClient(option.WithAPIKey(apiKey))
	})
	return &haikuClient, nil
}

type haikuResponse struct {
	RideType          *string  `json:"ride_type"`
	FromLocation      *string  `json:"from_location"`
	ToLocation        *string  `json:"to_location"`
	IsImmediate       bool     `json:"is_immediate"`
	DepartureTimeText *string  `json:"departure_time_text"`
	Cost              *float64 `json:"cost"`
	Currency          *string  `json:"currency"`
	NotARideMessage   bool     `json:"not_a_ride_message"`
}

// extractWithHaiku calls Claude Haiku to parse a freeform message.
// Returns ErrNotARide if the message is not a ride request/offer.
// Returns an error wrapping "ANTHROPIC_API_KEY not configured" if the key is missing.
func extractWithHaiku(ctx context.Context, content string, groupName string) (*ParsedRide, error) {
	client, err := getHaikuClient()
	if err != nil {
		return nil, err
	}

	systemPrompt := fmt.Sprintf(`You are a ride-sharing message parser. Extract ride information from the message.
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

	resp, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     "claude-haiku-4-5-20251001",
		MaxTokens: 512,
		System: []anthropic.TextBlockParam{{
			Text: systemPrompt,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(content)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("haiku api: %w", err)
	}

	// Extract text from first content block
	var raw string
	for _, block := range resp.Content {
		if block.Type == "text" {
			raw = block.Text
			break
		}
	}

	var hr haikuResponse
	if err := json.Unmarshal([]byte(raw), &hr); err != nil {
		return nil, fmt.Errorf("haiku: unmarshal response: %w", err)
	}

	if hr.NotARideMessage {
		return nil, ErrNotARide
	}

	if hr.RideType == nil || (hr.FromLocation == nil && hr.ToLocation == nil) {
		return nil, fmt.Errorf("haiku: missing ride type or locations")
	}

	parsed := &ParsedRide{
		IsImmediate:      hr.IsImmediate,
		Cost:             hr.Cost,
		Currency:         hr.Currency,
		FromLocationText: hr.FromLocation,
		ToLocationText:   hr.ToLocation,
	}

	switch *hr.RideType {
	case "need_ride":
		parsed.RideType = model.RideTypeNeedRide
	case "ride_available":
		parsed.RideType = model.RideTypeRideAvailable
	default:
		return nil, fmt.Errorf("haiku: unknown ride_type %q", *hr.RideType)
	}

	return parsed, nil
}
