# IllMUR (GoldBot) 🤖📈

IllMUR is a fully automated, AI-driven algorithmic trading bot. It bridges the gap between sophisticated web-based charting on TradingView, state-of-the-art visual AI models (Google Gemini & Mistral Pixtral), and MetaTrader 5 (MT5) trade execution. 

Built with Go and the [Wails framework](https://wails.io/), IllMUR operates locally with a sleek, cyberpunk-inspired UI, ensuring maximum privacy and minimal latency.

## 🚀 Key Features

- **Visual Chart Analysis:** Uses Chromedp/Playwright to run a persistent, stealth-mode headless Chrome browser. It takes high-resolution screenshots of live TradingView charts (including all your custom indicators) without relying on traditional raw data feeds.
- **Smart Dual-Engine AI:** 
  - **Primary Engine:** Google Gemini 2.5 Flash analyzes the visual chart structure to identify highly probable entry points, take profits (TP), and stop losses (SL).
  - **Fallback Engine:** Mistral Pixtral acts as a seamless backup.
  - **Key Rotation:** Automatically rotates through multiple API keys to infinitely scale past `429 Too Many Requests` API limits.
- **MT5 Execution Bridge:** A local HTTP server (`127.0.0.1:8080`) that translates AI JSON predictions into actionable MQL5 trade orders.
- **Strict Risk Management Engine:** 
  - Validates all AI signals locally on the MT5 bridge (e.g., enforcing that a BUY signal's TP is strictly above the current market Ask).
  - **Multi-Stage Trailing SL:** Secures profits automatically (e.g., locking in Breakeven + 2% at 40% of target, and BE + 10% at 75% of target).
  - **Daily Max Loss Kill-Switch:** Immediately blocks trading and sends a shutdown signal to the Go backend if the daily max loss limit (e.g., $20) is breached.
- **Dynamic Configuration:** 100% portable architecture. All API keys and preferences are stored locally in a `settings.json` file generated next to the executable.

## 🏗️ Architecture

1. **Frontend (Wails/JS/CSS):** A cyberpunk dashboard displaying real-time AI logic, live bot status, dynamic SVGs, and live terminal logs.
2. **Backend (Go):** Orchestrates the timing loops, controls the hidden Chrome instance, communicates with Google/Mistral APIs, and serves the HTTP endpoints.
3. **Execution (MQL5):** The `mt5_bridge.mq5` Expert Advisor polls the Go backend every 5 seconds. When a valid signal is detected, it validates market spread, normalizes tick sizes, executes the trade, and manages the trailing stops.

## 🛠️ Setup & Installation

### 1. Prerequisites
- [Go 1.23+](https://golang.org/)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation) (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- MetaTrader 5

### 2. Build the Application
Clone the repository and build the Wails application:
```bash
wails build
```
This generates the standalone executable `IllMUR.exe` in the `build/bin/` folder.

### 3. Configure the Bot
1. Launch `IllMUR.exe`.
2. Click the ⚙️ Settings icon in the sidebar.
3. Add your Gemini and Mistral API keys (one per line).
4. (Optional) Log into TradingView directly through the bot's isolated Chrome profile.

### 4. Connect MetaTrader 5
1. Copy the `mt5_bridge.mq5` file into your MT5 `MQL5\Experts\` folder.
2. Open MT5 and go to **Tools -> Options -> Expert Advisors**. 
3. Check **"Allow WebRequest for listed URL"** and add: `http://127.0.0.1:8080`
4. Compile the Expert Advisor in MetaEditor and drag it onto your `XAUUSD` chart!

## 🛡️ Disclaimer
This software is for educational purposes and experimental algorithmic trading research. Financial markets are highly volatile. Never trade with capital you cannot afford to lose. The developers assume zero liability for financial losses incurred by utilizing this automated bridge.
