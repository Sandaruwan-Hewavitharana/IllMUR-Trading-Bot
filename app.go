package main

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"goldbot/analysis"
	"goldbot/config"
	"goldbot/screenshot"
	"goldbot/server"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	cfg      *config.Config
	isPaused atomic.Bool
	browser  *screenshot.Browser // persistent Chrome session
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.isPaused.Store(false)

	cfg, err := config.Load()
	if err != nil {
		a.emitLog("ERROR", fmt.Sprintf("Failed to load config: %v", err))
		return
	}
	a.cfg = cfg

	// Start MT5 bridge server inside the Wails lifecycle (not in main())
	// This prevents it from running during wails build's binding generation phase
	go server.StartHTTP()

	go a.runBotLoop()
}

// TogglePause pauses or resumes the bot loop.
// Returns true if the bot is now paused, false if running.
func (a *App) TogglePause() bool {
	current := a.isPaused.Load()
	a.isPaused.Store(!current)
	if !current {
		a.emitLog("WARN", "Bot paused by user. Next cycle will be skipped.")
		runtime.EventsEmit(a.ctx, "pause-state-event", true)
	} else {
		a.emitLog("INFO", "Bot resumed by user. Resuming analysis cycles.")
		runtime.EventsEmit(a.ctx, "pause-state-event", false)
	}
	return !current
}

// GetSettings returns the current config to the frontend UI
func (a *App) GetSettings() *config.Config {
	return a.cfg
}

// SaveSettings is called by the UI to persist updated API keys and configurations
func (a *App) SaveSettings(updatedCfg config.Config) error {
	if err := config.Save(&updatedCfg); err != nil {
		a.emitLog("ERROR", fmt.Sprintf("Failed to save settings: %v", err))
		return err
	}
	a.cfg = &updatedCfg
	a.emitLog("SUCCESS", "Settings saved successfully")
	return nil
}

func (a *App) runBotLoop() {
	// Give the UI a moment to fully render before we start
	time.Sleep(2 * time.Second)

	interval := time.Duration(a.cfg.LoopIntervalMinutes) * time.Minute
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial run
	a.ensureBrowserAndRun()

	for {
		select {
		case <-a.ctx.Done():
			a.emitLog("WARN", "App shutting down. Closing browser...")
			if a.browser != nil {
				a.browser.Close()
			}
			return
		case <-ticker.C:
			if a.isPaused.Load() {
				// If paused and browser is still open, close it to free resources/locks
				if a.browser != nil {
					a.emitLog("INFO", "Closing Chrome to free resources while paused...")
					a.browser.Close()
					a.browser = nil
				}
				a.emitLog("WARN", "Bot is paused. Skipping this analysis cycle.")
				continue
			}
			a.ensureBrowserAndRun()
		}
	}
}

func (a *App) ensureBrowserAndRun() {
	if a.browser == nil {
		a.emitLog("STEP", "[1/4] Launching Chrome browser...")
		a.emitLog("INFO", fmt.Sprintf("Loading profile from: %s", a.cfg.ChromeUserDataDir))
		a.emitLog("INFO", fmt.Sprintf("Waiting %d seconds for indicators to render...", a.cfg.ChartRenderWaitSecs))

		browser, err := screenshot.NewBrowser(a.cfg)
		if err != nil {
			a.emitLog("ERROR", fmt.Sprintf("Browser launch failed: %v", err))
			return
		}
		a.browser = browser
		a.emitLog("SUCCESS", "Browser ready. Chart is live.")
	}

	a.runIteration()
}

func (a *App) runIteration() {
	a.emitLog("STEP", fmt.Sprintf("[2/4] Capturing live chart screenshot..."))

	imgBytes, err := a.browser.Capture()
	if err != nil {
		a.emitLog("ERROR", fmt.Sprintf("Screenshot failed: %v", err))
		return
	}
	a.emitLog("SUCCESS", "Screenshot captured successfully")

	a.emitLog("STEP", "[3/4] Sending image to Gemini 2.5 Flash...")

	tradeSig, err := analysis.Analyze(a.ctx, a.cfg, imgBytes)
	if err != nil {
		a.emitLog("ERROR", fmt.Sprintf("Gemini analysis failed: %v", err))
		return
	}
	a.emitLog("SUCCESS", fmt.Sprintf("AI Analysis complete → Signal: %s", tradeSig.Signal))

	// Push signal to the UI
	runtime.EventsEmit(a.ctx, "new-signal-event", tradeSig)

	// Push signal to the MT5 bridge server
	server.UpdateSignal(tradeSig)

	a.emitLog("STEP", "[4/4] Transmitting signal to MT5 Terminal...")
	a.emitLog("INFO", fmt.Sprintf("Target: %s", a.cfg.MT5Endpoint))
	a.emitLog("SUCCESS", "Cycle complete. Next cycle in 5 minutes.")
}

// emitLog sends a structured log event to the frontend UI.
func (a *App) emitLog(level string, msg string) {
	fmt.Printf("[%s] %s\n", level, msg)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "log-event", map[string]string{
			"level":   level,
			"message": msg,
			"time":    time.Now().Format("15:04:05"),
		})
	}
}
