# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Architecture Overview

This is a cryptocurrency trading bot application built in Go using Domain-Driven Design (DDD) and Clean Architecture principles. The system integrates with Binance API for trading operations and uses PostgreSQL for persistence, RabbitMQ for message queuing, and supports multiple trading strategies.

### Domain Layer (`src/domain/`)
- **Entities**: `TradingBot` (aggregate root), `Strategy`, `TradingDecisionLog` 
- **Value Objects**: `Symbol`, `Kline`, `Price`, `Currency`, `EntityId`, `MinimumSpread`, `Timeframe`
- **Strategy Pattern**: `TradingStrategy` interface with `MovingAverageStrategy` and `BreakoutStrategy` implementations
- **Domain Services**: Strategy factory with type-safe parameter validation

### Application Layer (`src/application/`)
- **Use Cases**: Create, list, start, and stop trading bots with comprehensive business logic
- **Repository Interfaces**: Clean abstraction over data persistence

### Infrastructure Layer (`src/infra/`)
- **Database**: PostgreSQL with manual SQL migrations
- **External APIs**: Binance integration with interface-based design for testability
- **Message Queue**: RabbitMQ adapter for event-driven notifications
- **HTTP Controllers**: REST API endpoints for bot management (create, list, start, stop)

## New Features

### Backtesting System
Complete backtesting functionality for validating trading strategies with historical data:
- **API Endpoint**: `POST /api/v1/trading/backtest`
- **Supported Strategies**: MovingAverage, Breakout
- **Metrics**: ROI, Win Rate, Max Drawdown, P&L tracking
- **Real Trading Simulation**: Includes fees, position tracking, anti-whipsaw protection

Example usage:
```bash
curl -X POST http://localhost:8080/api/v1/trading/backtest \
  -H "Content-Type: application/json" \
  -d @example_backtest_request.json
```

## Common Development Commands

### Build and Run
```bash
# Build the application
go build -o crypgo-machine

# Run the application (requires .env file and database)
go run main.go

# Start dependencies with Docker Compose
docker-compose up -d
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -v -run TestCreateTradingBotUseCase_Success ./src/application/usecase

# Run tests for specific package
go test -v ./src/domain/vo/
```

### Database Management
```bash
# Execute migrations manually (in order)
psql -d crypgo_machine -f src/infra/database/migrations/001_create_trade_bots_table.sql
psql -d crypgo_machine -f src/infra/database/migrations/002_add_strategy_params_column.sql
psql -d crypgo_machine -f src/infra/database/migrations/003_create_trading_decision_logs_table.sql
```

### Production Monitoring
```bash
# Monitor logs in real-time (basic)
./scripts/monitor-logs.sh

# Advanced dashboard with tmux (multiple windows)
./scripts/monitor-dashboard.sh

# Alert system for critical errors
./scripts/monitor-alerts.sh
```

#### Monitoring Commands for VPS (31.97.249.4)
```bash
# Quick log monitoring
ssh root@31.97.249.4 "cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml logs -f --tail 50 crypgo-app"

# Check container status
ssh root@31.97.249.4 "cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml ps"

# View resource usage
ssh root@31.97.249.4 "docker stats --no-stream"

# Follow error logs only
ssh root@31.97.249.4 "cd /opt/crypgo-machine && docker-compose -f docker-compose.full.yml logs -f | grep -i -E 'error|warn|fatal'"
```

## Testing Patterns and Infrastructure

### Repository Testing
- Database implementations use real PostgreSQL connections with cleanup procedures
- In-memory implementations provide fast, isolated testing
- Both implementations are tested to ensure interface compliance

### Mock and Fake Implementations
- `BinanceClientFake`: Configurable responses with predefined market scenarios (whipsaw, strong trends)
- `MockMessageBroker`: No-op implementation for isolated testing
- In-memory repositories with thread-safe concurrent access

### Test Data Management
- Environment configuration loaded from `.env` files during tests
- `cleanupTradeBot()` functions prevent test interference
- Predefined market data generators in `external.CreateWhipsawKlines()` and `external.CreateStrongTrendKlines()`

### Use Case Testing Patterns
- Constructor parameter validation: `NewCreateTradingBotUseCase(repo, client, broker, exchange)`
- Strategy parameter testing with both valid and invalid configurations
- Anti-whipsaw protection testing with `MinimumSpread` validation

## Key Domain Concepts

### Trading Strategies
- Strategies implement `TradingStrategy` interface with `Decide(klines, tradingBot)` method
- Position-aware: bots can only buy when not positioned, sell when positioned
- Anti-whipsaw protection using `MinimumSpread` to prevent false signals
- Strategy parameters stored as JSON in database and reconstructed via factory

### Value Object Validation
- `Symbol`: Validates against whitelist (BTCBRL, SOLBRL)
- `Kline`: Comprehensive OHLCV validation with proper ordering
- `EntityId`: UUID-based with validation
- `MinimumSpread`: Percentage-based for strategy configuration

### Repository Pattern
- Interface definitions in `src/application/repository/`
- Database implementations in `src/infra/repository/`
- Thread-safe in-memory implementations for testing

## Environment Configuration

Required environment variables:
- `BINANCE_API_KEY`, `BINANCE_SECRET_KEY`: Binance API credentials
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`: PostgreSQL connection
- `RABBIT_MQ_URL`: RabbitMQ connection string

## Known Issues and Patterns

### Test Compilation Issues
When adding new use cases or modifying constructors:
- Ensure `MockMessageBroker` implements complete `queue.MessageBroker` interface
- Update all `NewCreateTradingBotUseCase` calls with proper parameter count (repo, client, broker, exchange)
- Import `"crypgo-machine/src/infra/queue"` when using message broker mocks

### Strategy Testing
- Strategy `Decide()` method requires both `klines` and `tradingBot` parameters
- Test cases should verify both decision type and analysis data (spread calculations, reasons)
- Use predefined klines generators for consistent market scenarios

### Database Testing
- Always clean up test data using `cleanupTradeBot()` to handle foreign key constraints
- Delete `trading_decision_logs` before `trade_bots` due to foreign key relationships
- Use both in-memory and database repositories in tests for speed vs. integration coverage