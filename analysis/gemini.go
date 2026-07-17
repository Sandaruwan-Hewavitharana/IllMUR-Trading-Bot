package analysis

import (
	"context"
	"fmt"
	"time"

	"goldbot/signal"

	"google.golang.org/genai"
)

const geminiPrompt = `Analyze this TradingView XAUUSD chart and return ONLY a raw JSON object with no markdown, no explanation, no preamble. 

Strict Rules for Trade Levels based on CURRENT MARKET PRICE:
1. If signal is "BUY":
   - Take Profit (take_profit) MUST be strictly HIGHER than the current market price.
   - Stop Loss (stop_loss) MUST be strictly LOWER than the current market price.
2. If signal is "SELL":
   - Take Profit (take_profit) MUST be strictly LOWER than the current market price.
   - Stop Loss (stop_loss) MUST be strictly HIGHER than the current market price.
3. Calculate the exact entry, tp, and sl from the actual visible chart prices, NOT the example below.

Strictly use this exact schema format:
{
  "pair": "XAUUSD",
  "signal": "BUY",
  "entry_range": { "min": 4105.00, "max": 4110.00 },
  "take_profit": [4135.00],
  "stop_loss": 4095.00,
  "analysis_rationale": "Brief rationale based on support/resistance or indicators"
}

signal must be one of: BUY, SELL, HOLD
All price values must be floats. Return raw JSON only.`

// analyzeGemini executes a single API request to Gemini using the provided key.
// It will be called in a rotation loop by the engine router.
func analyzeGemini(apiKey string, imgBytes []byte) (*signal.TradeSignal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	temp := float32(0.0)

	resp, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.5-flash",
		[]*genai.Content{
			{
				Role: "user",
				Parts: []*genai.Part{
					{Text: geminiPrompt},
					{InlineData: &genai.Blob{
						MIMEType: "image/png",
						Data:     imgBytes,
					}},
				},
			},
		},
		&genai.GenerateContentConfig{
			Temperature: &temp,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("gemini api request failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("gemini returned no candidates")
	}
	if resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned empty content")
	}

	rawJSON := resp.Candidates[0].Content.Parts[0].Text
	return parseSignalJSON(rawJSON)
}
