package signal

import (
	"encoding/json"
)

// TradeSignal is the structured output matching the user's specific JSON requirement.
type TradeSignal struct {
	Pair              string      `json:"pair"`
	Signal            string      `json:"signal"`
	EntryRange        EntryRange  `json:"entry_range"`
	TakeProfit        []float64   `json:"take_profit"`
	StopLoss          float64     `json:"stop_loss"`
	AnalysisRationale string      `json:"analysis_rationale"`
}

type EntryRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// ToJSON returns the pretty-printed JSON string representation of the signal.
func (s *TradeSignal) ToJSON() string {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
