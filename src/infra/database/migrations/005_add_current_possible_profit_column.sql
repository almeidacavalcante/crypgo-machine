-- Migration: 005_add_current_possible_profit_column
-- Description: Add current_possible_profit column to trading_decision_logs table for profit tracking
-- Date: 2025-07-02

ALTER TABLE trading_decision_logs 
ADD COLUMN current_possible_profit DOUBLE PRECISION DEFAULT 0.0;