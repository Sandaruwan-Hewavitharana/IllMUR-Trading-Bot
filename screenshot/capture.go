package screenshot

import (
	"context"
	"fmt"
	"os"
	"time"

	"goldbot/config"

	"github.com/chromedp/chromedp"
)

// Browser holds a persistent Chrome session that stays open for the app's lifetime.
type Browser struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	ctx         context.Context
	ctxCancel   context.CancelFunc
}

// NewBrowser launches Chrome once, navigates to the chart, and waits for all
// indicators to fully render. This is called ONCE at startup — subsequent
// screenshots reuse the same open tab with zero re-navigation overhead.
func NewBrowser(cfg *config.Config) (*Browser, error) {
	// Ensure the isolated user data directory exists before launching Chrome
	if err := os.MkdirAll(cfg.ChromeUserDataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chrome user data dir: %w", err)
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(cfg.ChromeUserDataDir),
		chromedp.Flag("profile-directory", cfg.ChromeProfileName),
		chromedp.Flag("headless", false),            // Required for TradingView WebGL/canvas
		chromedp.Flag("disable-extensions", false), // Keep user indicators active
		chromedp.WindowSize(1920, 1080),
		chromedp.Flag("no-sandbox", false),
		
		// Stealth anti-bot flags
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("excludeSwitches", "enable-automation"),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)

	// Suppress noisy TradingView cookie parse errors from the console
	ctx, ctxCancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithErrorf(func(string, ...interface{}) {}),
	)

	// CRITICAL FIX: Run the initial navigation directly on b.ctx — NOT on a child
	// timeout context. If we used a child context and then cancelled it via defer,
	// chromedp would interpret the cancellation as a session teardown and corrupt
	// b.ctx for all future Capture() calls, causing "context canceled" errors.
	waitSecs := time.Duration(cfg.ChartRenderWaitSecs) * time.Second
	err := chromedp.Run(ctx,
		chromedp.Navigate(cfg.TradingViewURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(waitSecs),
	)
	if err != nil {
		ctxCancel()
		allocCancel()
		return nil, fmt.Errorf("browser initialization failed: %w", err)
	}

	return &Browser{
		allocCtx:    allocCtx,
		allocCancel: allocCancel,
		ctx:         ctx,
		ctxCancel:   ctxCancel,
	}, nil
}

// Capture takes a screenshot of the already-open TradingView chart.
// TradingView auto-refreshes chart data in real-time — no navigation needed.
func (b *Browser) Capture() ([]byte, error) {
	snapCtx, cancel := context.WithTimeout(b.ctx, 15*time.Second)
	defer cancel()

	var buf []byte
	if err := chromedp.Run(snapCtx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return nil, fmt.Errorf("screenshot capture failed: %w", err)
	}
	return buf, nil
}

// Close shuts down the persistent Chrome browser gracefully.
func (b *Browser) Close() {
	b.ctxCancel()
	b.allocCancel()
}
