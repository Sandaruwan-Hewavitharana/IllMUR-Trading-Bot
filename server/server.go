package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"goldbot/signal"
)

// SignalStore holds the current actionable signal in a thread-safe way.
type SignalStore struct {
	mu            sync.RWMutex
	currentSignal *signal.TradeSignal
	timestamp     time.Time
}

var store = &SignalStore{}

// UpdateSignal is called by the Go bot to push a new signal to the MT5 bridge
func UpdateSignal(sig *signal.TradeSignal) {
	store.mu.Lock()
	defer store.mu.Unlock()
	
	// Optional: You could ensure that HOLD signals clear the store
	// or just store everything. The EA filters out HOLDs.
	store.currentSignal = sig
	store.timestamp = time.Now()
}

// StartHTTP starts the local server that the MT5 Expert Advisor polls
func StartHTTP() {
	http.HandleFunc("/signal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		store.mu.RLock()
		sig := store.currentSignal
		ts := store.timestamp
		store.mu.RUnlock()

		// If no signal or it's older than 5 minutes (TTL), return empty/NONE
		if sig == nil || time.Since(ts) > 5*time.Minute {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"signal":"NONE"}`))
			return
		}

		// Flatten the TakeProfit array so MT5 can parse it easily as a double
		tp := 0.0
		if len(sig.TakeProfit) > 0 {
			tp = sig.TakeProfit[0]
		}

		mt5Sig := struct {
			Signal     string  `json:"signal"`
			Pair       string  `json:"pair"`
			StopLoss   float64 `json:"stop_loss"`
			TakeProfit float64 `json:"take_profit"`
		}{
			Signal:     sig.Signal,
			Pair:       sig.Pair,
			StopLoss:   sig.StopLoss,
			TakeProfit: tp,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mt5Sig)
	})

	http.HandleFunc("/signal/consume", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		store.mu.Lock()
		// Clear the signal so EA doesn't execute it twice
		store.currentSignal = nil
		store.mu.Unlock()

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Kill-switch: MT5 calls this when max daily loss is breached
	http.HandleFunc("/risk/kill", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		log.Println("🛑 CRITICAL: Max daily loss limit breached on MT5! Shuting down Go Server completely via Kill-Switch...")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Kill-switch acknowledged. Server shutting down."))

		// Allow 1 second for the HTTP response to flush back to MT5 before exiting
		time.AfterFunc(1*time.Second, func() {
			os.Exit(0)
		})
	})

	fmt.Println("Starting MT5 Bridge Server on :8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
