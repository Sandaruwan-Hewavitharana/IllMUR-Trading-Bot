package analysis

import (
	"context"
	"fmt"

	"goldbot/config"
	"goldbot/signal"
)

// Analyze is the central smart-routing engine with multiple API key rotation.
// It iterates through Gemini keys (Primary). If a key fails (rate limit or otherwise),
// it moves to the next key. If all Gemini keys fail, it falls back to Mistral keys.
func Analyze(ctx context.Context, cfg *config.Config, imgBytes []byte) (*signal.TradeSignal, error) {
	// ── PRIMARY: Gemini 2.5 Flash ─────────────────────────────────────
	if len(cfg.GeminiKeys) > 0 {
		for i, key := range cfg.GeminiKeys {
			if key == "" {
				continue
			}
			sig, err := analyzeGemini(key, imgBytes)
			if err == nil {
				return sig, nil
			}
			fmt.Printf("[WARN] Gemini key %d failed (%v). Trying next key...\n", i+1, err)
		}
		fmt.Printf("[WARN] All Gemini keys exhausted/failed. Falling back to Mistral...\n")
	}

	// ── FALLBACK: Mistral Pixtral Large ────────────────────────────────
	if len(cfg.MistralKeys) > 0 {
		for i, key := range cfg.MistralKeys {
			if key == "" {
				continue
			}
			sig, err := analyzeMistral(key, imgBytes)
			if err == nil {
				return sig, nil
			}
			fmt.Printf("[WARN] Mistral key %d failed (%v). Trying next key...\n", i+1, err)
		}
	}

	return nil, fmt.Errorf("all AI engines and API keys exhausted or failed")
}
