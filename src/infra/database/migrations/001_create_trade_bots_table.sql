-- Migration: 001_create_trade_bots_table
-- Description: Initial creation of trade_bots table
-- Date: 2025-06-28

CREATE TABLE trade_bots
(
    id            VARCHAR(36)      PRIMARY KEY,
    symbol        VARCHAR(20)      NOT NULL,
    quantity      DOUBLE PRECISION NOT NULL,
    strategy_name VARCHAR(50)      NOT NULL,
    status        VARCHAR(20)      NOT NULL,
    is_positioned BOOLEAN          NOT NULL DEFAULT false,
    created_at    TIMESTAMP        NOT NULL
);