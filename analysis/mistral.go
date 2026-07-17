package analysis

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"goldbot/signal"
)

const mistralSystemPrompt = `You are an expert forex and gold trading analyst. Analyze the TradingView chart image provided and return a JSON trading signal. Be precise and data-driven.`

const mistralUserPrompt = `Analyze this XAUUSD chart and return ONLY a raw JSON object with no markdown, no explanation, no preamble. Strictly use this exact schema:

{
  "pair": "XAUUSD",
  "signal": "BUY",
  "entry_range": { "min": 3320.00, "max": 3325.00 },
  "take_profit": [3350.00],
  "stop_loss": 3310.00,
  "analysis_rationale": "Brief rationale here"
}

signal must be one of: BUY, SELL, HOLD
All price values must be floats. Return raw JSON only.`

type mistralMsg struct {
	Role    string        `json:"role"`
	Content []mistralPart `json:"content"`
}

type mistralPart struct {
	Type     string              `json:"type"`
	Text     string              `json:"text,omitempty"`
	ImageURL *mistralImageURLObj `json:"image_url,omitempty"`
}

type mistralImageURLObj struct {
	URL string `json:"url"`
}

type mistralRequest struct {
	Model          string       `json:"model"`
	Messages       []mistralMsg `json:"messages"`
	Temperature    float64      `json:"temperature"`
	MaxTokens      int          `json:"max_tokens"`
	ResponseFormat struct {
		Type string `json:"type"`
	} `json:"response_format"`
}

type mistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// analyzeMistral sends the screenshot to Mistral Pixtral Large and parses the signal.
// It includes a 2-attempt retry loop in case the model returns malformed JSON.
func analyzeMistral(apiKey string, imgBytes []byte) (*signal.TradeSignal, error) {
	b64Image := base64.StdEncoding.EncodeToString(imgBytes)
	dataURI := fmt.Sprintf("data:image/jpeg;base64,%s", b64Image)

	reqBody := mistralRequest{
		Model: "pixtral-large-2411",
		Messages: []mistralMsg{
			{
				Role:    "system",
				Content: []mistralPart{{Type: "text", Text: mistralSystemPrompt}},
			},
			{
				Role: "user",
				Content: []mistralPart{
					{Type: "text", Text: mistralUserPrompt},
					{Type: "image_url", ImageURL: &mistralImageURLObj{URL: dataURI}},
				},
			},
		},
		Temperature: 0,
		MaxTokens:   512,
	}
	reqBody.ResponseFormat.Type = "json_object"

	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		sig, err := doMistralRequest(apiKey, reqBody)
		if err == nil {
			return sig, nil
		}
		lastErr = fmt.Errorf("mistral attempt %d: %w", attempt, err)
		time.Sleep(1 * time.Second)
	}
	return nil, lastErr
}

func doMistralRequest(apiKey string, reqBody mistralRequest) (*signal.TradeSignal, error) {
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.mistral.ai/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var mistralResp mistralResponse
	if err := json.Unmarshal(bodyBytes, &mistralResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if len(mistralResp.Choices) == 0 {
		return nil, fmt.Errorf("empty choices in response")
	}

	return parseSignalJSON(mistralResp.Choices[0].Message.Content)
}

// parseSignalJSON is a shared JSON extraction utility used by both Mistral and Groq.
func parseSignalJSON(raw string) (*signal.TradeSignal, error) {
	// Strip markdown wrappers if present
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	// Extract just the JSON object
	startIdx := strings.Index(raw, "{")
	endIdx := strings.LastIndex(raw, "}")
	if startIdx == -1 || endIdx == -1 || endIdx <= startIdx {
		return nil, fmt.Errorf("no valid JSON object found in response: %q", raw)
	}
	raw = raw[startIdx : endIdx+1]

	var sig signal.TradeSignal
	if err := json.Unmarshal([]byte(raw), &sig); err != nil {
		return nil, fmt.Errorf("json unmarshal failed (%w): %q", err, raw)
	}

	// Normalise the signal field
	switch strings.ToUpper(sig.Signal) {
	case "BUY", "SELL", "HOLD":
		sig.Signal = strings.ToUpper(sig.Signal)
	default:
		sig.Signal = "HOLD"
	}

	return &sig, nil
}
