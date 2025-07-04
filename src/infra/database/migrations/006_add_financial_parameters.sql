-- Migration: 006_add_financial_parameters
-- Description: Add financial parameters to trade_bots table
-- Date: 2025-07-03

ALTER TABLE trade_bots
    ADD COLUMN initial_capital DOUBLE PRECISION,
    ADD COLUMN trade_amount DOUBLE PRECISION,
    ADD COLUMN currency VARCHAR(10),
    ADD COLUMN trading_fees DOUBLE PRECISION,
    ADD COLUMN minimum_profit_threshold DOUBLE PRECISION;