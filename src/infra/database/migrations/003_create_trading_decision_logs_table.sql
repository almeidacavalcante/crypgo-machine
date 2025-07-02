-- Migration: 003_create_trading_decision_logs_table
-- Description: Create table to store trading decision logs with analysis data
-- Date: 2025-06-28

CREATE TABLE trading_decision_logs
(
    id              VARCHAR(36)      PRIMARY KEY,
    trading_bot_id  VARCHAR(36)      NOT NULL,
    decision        VARCHAR(10)      NOT NULL,
    strategy_name   VARCHAR(50)      NOT NULL,
    analysis_data   TEXT             NOT NULL,
    market_data     TEXT             NOT NULL,
    current_price   DOUBLE PRECISION NOT NULL,
    timestamp       TIMESTAMP        NOT NULL,
    
    FOREIGN KEY (trading_bot_id) REFERENCES trade_bots(id)
);