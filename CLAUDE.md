# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Claude Code Guidance

- Never add yourself as a co-author on commits. The repository owner is always the sole author.
- Access the VPS only with the data in the local `.env.production` (see `CLAUDE.local.md`).
- Do not print or copy values from `.env`, `.env.production`, `.env.llm` or `http-client.private.env.json`.

## What This Project Is

CrypGo Machine is a single-binary Go service that runs automated spot-trading bots against Binance. Each bot is a goroutine that wakes up on a fixed interval, pulls the last 100 klines, asks its strategy for a BUY / SELL / HOLD decision, logs that decision, and places a market order when needed. Around that core sit a backtesting engine, an advisory market-sentiment pipeline (Fear & Greed + RSS + OpenAI), email and Telegram notifications over RabbitMQ, a JWT-protected REST API, and a static web dashboard.

- Module: `crypgo-machine`, Go 1.23, standard `net/http` (no web framework).
- Entry point: `main.go` wires everything by hand (no DI container).
- Persistence: PostgreSQL via `lib/pq` and hand-written SQL. Messaging: RabbitMQ via `streadway/amqp`.
- Default market is BRL pairs (`SOLBRL`, `BTCBRL`); nothing prevents other Binance spot pairs.

## Architecture

DDD + Clean Architecture in three layers under `src/`. Dependencies should point inward, with the exceptions listed under "Known Gaps".

### Domain (`src/domain/`)
- **Entities** (`entity/`): `TradingBot` (aggregate root), `Strategy` (name + params config), `TradingDecisionLog`, `SentimentSuggestion`, `BacktestResult`, `BacktestTrade`.
- **Strategies** (`entity/`): `TradingStrategy` interface with `MovingAverageStrategy` and `RSIStrategy`. There is no Breakout strategy.
- **Value objects** (`vo/`): `Symbol`, `Kline`, `Price`, `Currency`, `EntityId`, `Name`, `MinimumSpread`, `Timeframe`, `SymbolFilter`, `RSIResult`, `SentimentScore`, `SentimentLevel`, `SentimentSources`, `ProfitLoss`, `WinRate`, `Drawdown`, `User`.
- **Services** (`service/`): `NewTradeStrategyFactory` (typed params to strategy) and `OrderValidatorService` (Binance LOT_SIZE / PRICE_FILTER / MIN_NOTIONAL).

### Application (`src/application/`)
- **Use cases** (`usecase/`): create, list, start, stop bot; list trading logs; auth; two backtest use cases; generate, list, approve sentiment suggestions.
- **Ports** (`service/`): `MarketDataSource` (live vs historical klines) and `TradingExecutionContext` (live orders vs simulated fills). The same `StartTradingBotUseCase.ExecuteAnalysisAndTrade` runs against either pair.
- **Repository interfaces** (`repository/`): `TradingBotRepository`, `TradingDecisionLogRepository`, `SentimentSuggestionRepository`.

### Infrastructure (`src/infra/`)
- `api/` and `http/controller/`: HTTP handlers. `middleware/` + `auth/`: Bearer JWT (HS256, 24h expiry).
- `repository/`: PostgreSQL implementations plus in-memory ones for bots and decision logs.
- `external/`: Binance client interface, real wrapper and `BinanceClientFake` (all three live in `binance_client_fake.go`), exchange-info cache, historical klines, Fear & Greed client, RSS reader/cache/health monitor, keyword and OpenAI sentiment analyzers, aggregator.
- `queue/`: `MessageBroker` interface and RabbitMQ adapter (topic exchange).
- `notification/`: email (SMTP) and Telegram consumers, Telegram command handler.
- `scheduler/`: sentiment scheduler (see Known Gaps: it is never started).
- `database/migrations/`: plain SQL files, applied manually or by the deploy workflow.

### Other top-level pieces
- `cmd/backtest`, `cmd/backtest-demo`: CLI backtests. `cmd/test-email`, `cmd/test-minimum-profit`: manual checks.
- `cmd/basis-monitor`: read-only Deribit futures basis alert, run by GitHub Actions (see "Futures Basis Monitor").
- `web/`: static dashboard (vanilla JS, pt-BR copy) served by the Go binary at `/`.
- `scripts/`: deploy, backup, monitoring and parameter-optimization shell scripts. The Python files and `models/` (a ~34 GB local Ministral-8B checkpoint) are an untracked local-LLM experiment; never commit `models/`.

## Live Trading Loop

`StartTradingBotUseCase.Execute` flips the bot to `RUNNING`, persists it, and launches `runStrategyLoop` in a goroutine:

1. Run one analysis immediately, then on every `interval_seconds` tick.
2. Re-read the bot from the database. If it is missing or not `RUNNING`, the loop exits. This is how `stop` works: `StopTradingBotUseCase` only writes `STOPPED`; the goroutine notices on its next tick.
3. `MarketDataSource.GetMarketData` fetches 100 klines at the Binance interval matching `interval_seconds` (`external.SecondsToInterval`).
4. `strategy.Decide(klines, bot)` returns a decision plus an `AnalysisData` map (always includes a `reason` code).
5. A `TradingDecisionLog` is saved for every tick, including HOLD.
6. `TradingExecutionContext.ExecuteTrade` validates and rounds the quantity, sends a MARKET order, updates position state on the bot, persists it, and publishes `trading.buy_executed` / `trading.sell_executed`.

On boot, `recoverRunningBots` in `main.go` finds bots still marked `RUNNING`, resets them to `STOPPED`, and starts them again so loops survive a restart (with `SkipNotification: true`, so a restart does not send one "started" notification per bot).

`Execute` refuses to start a bot whose `interval_seconds` is not accepted by `external.SecondsToInterval`. The loop has a deferred recover: a panic stops that bot (status `STOPPED`, `trading_bot.stopped` published with a `reason`) instead of killing the process.

The in-memory `TradingBot` held by the goroutine is the source of truth for position state while the loop runs; only `status` is re-read from the database.

## Strategies

Both strategies are position-aware (buy only when flat, sell only when positioned), check stoploss first, and refuse to sell below the bot's `minimum_profit_threshold` unless stoploss fired. `possibleProfit` is the raw percentage change from `entryPrice`; fees are not subtracted.

### MovingAverage
Simple moving averages of the close over `FastWindow` and `SlowWindow`.
- **BUY** when `fast < slow`, not positioned, and the spread between the averages is at least `MinimumSpread` percent (anti-whipsaw).
- **SELL** when `fast > slow`, positioned, and profit is at least the minimum threshold.
- This is a buy-the-dip rule, not a classic golden-cross: it buys while the fast average is below the slow one.

### RSI
Wilder-smoothed RSI over `Period` (needs `Period + 1` klines).
- **BUY** when RSI is below `OversoldThreshold` (default 30) and not positioned.
- **SELL** when RSI is above `OverboughtThreshold` (default 70), positioned, and profit is at least the minimum threshold.

### Stoploss
`StoplossThreshold` (percent, 0 = disabled) on either strategy: when positioned and loss reaches the threshold, SELL with reason `stoploss_triggered`, bypassing the minimum-profit rule.

### Where strategies get built
Strategy construction is duplicated in three places that must stay in sync when adding a strategy or parameter:
1. `service.NewTradeStrategyFactory` (bot creation, typed params).
2. `entity.BuildStrategy` (rehydration from the JSON in `trade_bots.strategy_params`).
3. `BacktestStrategyUseCase.createStrategy` (backtest request params).

Live bots always get `MinimumSpread = 0.1`; only the backtest path reads `MinimumSpread` from params. In the factory an RSI threshold of 0 means "use the default" (30 / 70).

## Fees and Quantity

- `trading_fees` is a percentage per trade (0.1 means 0.1%).
- On BUY, `actualQuantityHeld = executedQuantity * (1 - fee)` is stored, where the executed quantity is the order response's `ExecutedQuantity` (falling back to the rounded quantity that was sent). On SELL, `CalculateQuantityForSell()` uses it so the order never exceeds the balance. The fee is still modelled, not read from the fills.
- `use_fixed_quantity = true`: trade the bot's `quantity`. `false`: trade `trade_amount / currentPrice`.
- `OrderValidatorService.ValidateOrderBeforePlacement` rounds down to the symbol step size (never up: a quantity below the symbol minimum is refused, not raised) and rejects orders under the minimum notional. On refusal its `warnings` return value carries the refusal reasons. Exchange info is cached in `ExchangeInfoService`.
- `placeBuyOrder` / `placeSellOrder` return `(executedQuantity, placed)`; a failed order is logged and the tick ends without changing position state.

## Backtesting

There are two engines; know which one you are touching.

| | `BacktestStrategyUseCase` | `BacktestTradingBotUseCase` |
|---|---|---|
| Used by | `POST /api/v1/trading/backtest` | `cmd/backtest`, `cmd/backtest-demo` |
| How it runs | Own simulator loop, calls `strategy.Decide` with a dummy bot | Reuses the live `ExecuteAnalysisAndTrade` with `HistoricalMarketDataSource` + `BacktestTradingExecutionContext` |
| Data window | All klines from the start up to the current one | Sliding 100-kline window, same as live |
| Extra rule | Never closes a trade at a loss unless stoploss fired | Follows the strategy decision as is |

API request fields: `strategy_name`, `symbol`, `params`, `initial_capital`, `trade_amount`, `currency`, `trading_fees`, `minimum_profit_threshold`, and either inline `historical_data` or one of `use_binance_data` (+ `start_date`, `interval`), `use_yesterday`, `use_last_week`. Working examples: `example_rsi_backtest_request.json`, `example_ma_stoploss_backtest_request.json`, `simple_ma_backtest_request.json`.

```bash
# CLI backtest (needs Binance keys in .env)
go run cmd/backtest/main.go -start=2024-01-01 -end=2024-01-31 -symbol=SOLBRL -fast=5 -slow=10 -interval=1h
```

## Sentiment Analysis (advisory)

`SentimentAggregator.CollectAndAnalyze` gathers the alternative.me Fear & Greed index and RSS feeds (CoinDesk, CoinTelegraph, Bitcoin.com, Decrypt, r/cryptocurrency), scores the articles with OpenAI (`gpt-4o-mini` by default) and falls back to a keyword analyzer when the LLM is disabled, fails, or hits its daily cost limit.

- Overall score in [-1, 1] = Fear & Greed 0.40 + news 0.35 + reddit 0.15 + social 0.10.
- Levels: `very_bullish` at 0.3 or more, `bullish` at 0.1 or more, `bearish` at -0.1 or less, `very_bearish` at -0.3 or less, otherwise `neutral`.
- Each analysis produces a `SentimentSuggestion` (exposure multiplier, profit threshold, interval) saved as `pending`; the user approves, customizes or ignores it.
- Suggestions never change bots. `ApproveSentimentSuggestionUseCase` records the decision on the suggestion only; `affected_bots` is always 0 and the response says that applying to bots is not implemented. Making approval change bots is an open product decision.
- The scheduler (full analysis every 4h, quick check hourly) starts only when `SENTIMENT_SCHEDULER_ENABLED=true`; each full analysis calls OpenAI.
- Telegram commands (only from `TELEGRAM_CHAT_ID`): `/sentiment`, `/quick`, `/status`, `/help`, `/oi`.

## Futures Basis Monitor (cash-and-carry signal)

`cmd/basis-monitor` runs once per invocation: it reads the Deribit public API (no key), computes the basis of every dated BTC future over the index, prints the curve, and sends a Telegram alert when the gross annualized basis reaches the threshold. It never places orders. It is independent from the main binary: no `.env`, Postgres or RabbitMQ.

- Schedule: `.github/workflows/basis-monitor.yml`, every 15 minutes, plus manual dispatch (`threshold_pct=0` forces a test alert; manual runs are stateless).
- Code: `vo.FuturesBasis` (math), `src/application/basismonitor` (use case, ports, alert text catalog for `pt-BR` and `en`), `src/infra/deribit` (market data), `src/infra/alerting` (stdlib Telegram sender, alert state stores). These packages import only the standard library and `domain/vo`; keep it that way so the scheduled job stays fast.
- Basis = best bid of the future over the index; annualized with simple (not compounded) scaling. The net figure subtracts the futures leg only (taker fee from the API plus the 0.025% delivery fee on monthly contracts, hardcoded in `deribit_client.go`).
- Repeat alerts are held back for `BASIS_COOLDOWN_HOURS` unless the best basis rose by `BASIS_REALERT_STEP_PCT` points. The last alert is kept in a JSON file that the workflow persists with `actions/cache`.
- Telegram delivery retries transient failures (network errors, HTTP 429 and 5xx) up to 3 times; a failed delivery fails the job and the next run tries again, because the cooldown only starts after a successful send.
- The Telegram bot token is part of the request URL, so `TelegramSender` strips the URL from errors: this repository is public and so are its job logs.

```bash
# Print the live curve without sending anything
BASIS_DRY_RUN=true go run ./cmd/basis-monitor
```

## Events (RabbitMQ)

One topic exchange, `trading_bot`.

| Routing key | Published by | Consumed by |
|---|---|---|
| `trading_bot.created` | `CreateTradingBotUseCase` | email, Telegram |
| `trading_bot.started` | `StartTradingBotUseCase` (not on boot recovery) | email, Telegram |
| `trading_bot.stopped` | `StopTradingBotUseCase`, and the loop's panic recovery | email, Telegram |
| `trading.buy_executed`, `trading.sell_executed` | `LiveTradingExecutionContext` | email, Telegram |
| `sentiment.analysis.completed` | `SentimentScheduler` | Telegram sentiment consumer |
| `sentiment.suggestion.pending`, `sentiment.extreme.detected` | nobody yet | subscribed, never received |

Queues: `email.notification.queue`, `telegram.notification.queue`, `telegram.sentiment.queue`. Consumers auto-ack and handler errors are dropped.

## HTTP API

All routes are registered in `main.go` on the default mux, port 8080. Protected routes need `Authorization: Bearer <token>` from `/api/v1/auth/login`.

| Route | Auth | Purpose |
|---|---|---|
| `/api/v1/health` | public | Liveness |
| `/api/v1/auth/login`, `/refresh`, `/validate` | public | JWT issue, refresh, check |
| `/api/v1/trading/create_trading_bot` | JWT | Create bot (`MovingAverage` or `RSI`; strategy params go in `params`) |
| `/api/v1/trading/list` | JWT | List bots |
| `/api/v1/trading/start`, `/stop` | JWT | Body `{"bot_id": "..."}` |
| `/api/v1/trading/backtest` | JWT | Run backtest |
| `/api/v1/trading/logs` | JWT | Decision logs with `decision`, `symbol`, `limit`, `offset` |
| `/api/v1/sentiment/generate`, `/suggestions`, `/approve`, `/analytics` | JWT | Suggestion workflow |
| `/api/v1/sentiment/analyze`, `/quick-check`, `/data-sources/validate` | JWT | Manual analysis |
| `/api/v1/sentiment/scheduler/status`, `/scheduler/trigger` | JWT | Scheduler state, one-off run |
| `/api/v1/sentiment/health` | public | Sentiment health |
| `/api/v1/telegram/test` | JWT | Sends a test message |
| `/api/v1/telegram/status` | public | Reports Telegram status |
| `/api/v1/telegram/test-sentiment`, `/sentiment/status` | JWT | Sentiment notification test |
| `/` | public | Static dashboard from `./web/` |

`api-tests.http` (JetBrains HTTP client) has ready-made requests; environments live in `http-client.env.json`.

## Database

Three tables: `trade_bots`, `trading_decision_logs` (FK to `trade_bots`), `sentiment_suggestions`. Strategy params, analysis data and market data are stored as JSON text.

There is no migration tool and no migrations table. Apply files in order; note that two files share the `004` prefix.

```bash
psql -d crypgo_machine -f src/infra/database/migrations/001_create_trade_bots_table.sql
psql -d crypgo_machine -f src/infra/database/migrations/002_add_strategy_params_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/003_create_trading_decision_logs_table.sql
psql -d crypgo_machine -f src/infra/database/migrations/004_add_interval_seconds_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/004_create_sentiment_suggestions_table.sql
psql -d crypgo_machine -f src/infra/database/migrations/005_add_current_possible_profit_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/006_add_financial_parameters.sql
psql -d crypgo_machine -f src/infra/database/migrations/007_add_entry_price_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/008_add_llm_analysis_columns.sql
psql -d crypgo_machine -f src/infra/database/migrations/009_add_actual_quantity_held_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/010_add_use_fixed_quantity_column.sql
```

`scripts/run-migrations.sh` applies them in order using the database settings from `.env.production` or `.env`. When adding a column to `trade_bots`, update the INSERT, UPDATE and every SELECT in `trading_bot_repository.database.go`, plus `entity.Restore`.

## Common Commands

```bash
# Local dependencies: Postgres (5432), test Postgres (5433), RabbitMQ (5672, UI 15672)
docker-compose up -d

# Run (fails fast without .env, Postgres and RabbitMQ)
go run main.go

# Build
go build -o crypgo-machine

# Tests
go test ./...
go test -v ./src/domain/entity/
go test -v -run TestCreateTradingBotUseCase_Success ./src/application/usecase
```

## Environment Variables

- **Required at boot**: `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`, `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `RABBIT_MQ_URL`.
- **Auth (required at boot, no fallbacks)**: `AUTH_EMAIL`, `AUTH_PASSWORD`, `JWT_SECRET`. The app calls `log.Fatalf` when any is empty. `.env.example` lists every variable with placeholder values.
- **Email**: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `FROM_EMAIL`, `TARGET_EMAIL`. With `TARGET_EMAIL` empty the email consumer is not started.
- **Telegram**: `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`. Telegram is disabled, not fatal, when these are missing.
- **Sentiment / LLM**: `OPENAI_API_KEY`, `OPENAI_MODEL`, `OPENAI_MAX_TOKENS`, `OPENAI_TEMPERATURE`, `USE_LLM_ANALYSIS`, `SENTIMENT_SCHEDULER_ENABLED` (default off).
- **Basis monitor** (process environment or GitHub repository variables, never `.env`): `BASIS_CURRENCY` (BTC), `BASIS_THRESHOLD_PCT` (8), `BASIS_MIN_DAYS` (30), `BASIS_COOLDOWN_HOURS` (24), `BASIS_REALERT_STEP_PCT` (1), `BASIS_ALERT_LOCALE` (pt-BR), `BASIS_STATE_FILE`, `BASIS_DRY_RUN`, plus `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` as GitHub secrets.
- **Local-LLM experiment only**: `HUGGINGFACE_TOKEN` in `.env.llm`.

`loadEnv` calls `log.Fatalf` when `.env` is absent. The Docker image copies `.env.production` to `.env` at build time.

## Testing

- `BinanceClientFake` gives canned klines and orders; `CreateWhipsawKlines()` and `CreateStrongTrendKlines()` generate market scenarios. Its exchange info only knows the symbols listed in `FakeGetExchangeInfoService`.
- In-memory repositories exist for bots and decision logs; use them for use-case tests.
- Database-backed tests load `.env` and need Postgres. `src/infra/api` tests need the test database on port 5433 (`postgres-test` in `docker-compose.yml`).
- Clean up with `cleanupTradeBot()`; delete `trading_decision_logs` before `trade_bots` because of the foreign key.
- Constructor signatures to keep in sync in tests: `NewCreateTradingBotUseCase(repo, client, broker, exchange)` and `Decide(klines, tradingBot)`.

### Current test status (2026-11-20, branch `feat/dashboard-analysis`, uncommitted bug-fix work)
`go build ./...` and `go vet ./...` pass. `go test ./...` passes in every package except `src/infra/api`, whose five older tests need the test database on port 5433 (not running when this was checked; the newer controller tests use the in-memory repository and pass).

## Deployment

- **VPS**: `/opt/crypgo-machine`, run with `docker-compose.full.yml`: `crypgo-app`, `postgres`, `rabbitmq`, `n8n`, `n8n-postgres`, `nginx`, `certbot`.
- **Scheduled job**: `.github/workflows/basis-monitor.yml` runs the futures basis monitor on GitHub-hosted runners; it does not touch the VPS.
- **CD**: `.github/workflows/deploy.yml` is the only deployment workflow (there is no CI workflow). It runs on push to `main` or manual dispatch with an optional `skip_backup` input: SSH in, back up, rebuild containers, run migrations, health check, roll back on failure, clean up, security check. Secrets: `SSH_PRIVATE_KEY`, `SSH_HOST`, `SSH_USER`, `SSH_PORT`. Setup notes in `SETUP_SSH_GITHUB_ACTIONS.md`.
- **nginx** (`nginx.conf`): a `geo $allowed_ip` whitelist guards the API, health, dashboard and n8n (403 for everyone else); webhooks under `/webhook/` are open; dotfiles, script extensions and common admin paths return 404; rate limits are 10 r/s (burst 20) for `/api/` and 5 r/s (burst 10) elsewhere. Server blocks: default on 80, n8n on 8081, plus TLS vhosts for the trading and n8n subdomains.
- **Adding an IP**: add a line to the `geo $allowed_ip` block, then `docker-compose -f docker-compose.full.yml exec nginx nginx -s reload`.

```bash
# Tail app logs on the VPS
ssh root@31.97.249.4 "cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml logs -f --tail 50 crypgo-app"

# Container status
ssh root@31.97.249.4 "cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml ps"
```

Local helpers: `scripts/monitor-logs.sh`, `scripts/monitor-dashboard.sh` (tmux), `scripts/monitor-alerts.sh`, `scripts/backup-database.sh`, `scripts/check-status.sh`.

## Known Gaps

Verified in code on 2026-11-20. Check before relying on any of these being fixed. The tracked bug list, with per-bug detail and status, lives in a Claude artifact database (collection `bugs`).

- **The VPS did not respond on 2026-11-20.** `31.97.249.4` answered neither ping nor ports 22, 80, 443, 8080 and 8081, and the `trading.` and `n8n.` subdomains no longer resolve. Confirm in the Hostinger panel that it still exists before relying on anything under "Deployment"; a push to `main` still triggers `deploy.yml` against it.
- **The `Deploy to Production` workflow is disabled** (manually, on 2026-11-20) so merges to `main` do not try to deploy to the unreachable VPS. Re-enable with `gh workflow enable deploy.yml`.
- **Production needs new env vars before the next deploy.** `AUTH_EMAIL`, `AUTH_PASSWORD` and `JWT_SECRET` are now mandatory; `.env.production` on the VPS does not define them yet, so the app will refuse to boot there until they are added.
- **Old credentials are still in git history**, and the tracked `test-build` binary contains the old password. Rotate the password and JWT secret and remove the binary.
- **Passwords are compared in plaintext** (constant-time) in `auth_usecase.go`; a bcrypt hash would need `golang.org/x/crypto`.
- **`/api/v1/sentiment/scheduler/trigger` only works while the scheduler is running** (`TriggerManualAnalysis` returns an error otherwise), so with `SENTIMENT_SCHEDULER_ENABLED` off, use `/api/v1/sentiment/analyze` or the Telegram commands.
- **Suggested intervals must be Binance kline intervals.** The table exists twice: `entity.generateSuggestions` and `MarketSentimentService.GetSentimentSuggestions`; keep them in sync (`bullish` is 900s because Binance has no 10m kline).
- **`use_fixed_quantity` defaults to `false` when omitted** from the create request (non-pointer bool), despite the column default of `true`.
- **A panicking bot is marked `STOPPED`, not `ERROR`.** `entity.StatusError` exists but `TradingBot` has no method to set it.
- **Fees and entry price are approximations.** Held quantity is `executed * (1 - fee%)` regardless of the commission asset; entry price is the last candle close, not the average fill price. A partially filled SELL still closes the position in state.
- **Port 8080 is published on the host** in `docker-compose.full.yml` (monitoring scripts use it), so nginx whitelisting only protects the app if the VPS firewall blocks that port from outside.
- **Shared execution context.** All live bots share one `LiveTradingExecutionContext`; its `Stop()` would halt every loop. Per-bot stop goes through the database status.
- **Layering leaks.** Application code imports `infra/external` and `infra/queue`; `domain/service/order_validator.service.go` imports `infra/external`; `CreateTradingBotUseCase` takes a concrete `binance.Client` it never uses.
- **Dead code.** `StartTradingBotUseCase.getMarketData` and `convertIntervalSecondsToBinanceInterval` are superseded by `LiveMarketDataSource`; `external.SymbolFilters.AdjustQuantityToStepSize` is unused and still rounds up to the minimum quantity (the live path uses `vo.SymbolFilter`); `main_bkp.txt` is a leftover.
- **Dashboard copy is hardcoded pt-BR.** `web/` has no i18n layer yet; new UI text needs one before it can follow the project's internationalization rule.
