-- Migration: 004_add_interval_seconds_column
-- Description: Add interval_seconds column to trade_bots table for configurable trading intervals
-- Date: 2025-07-02

ALTER TABLE trade_bots 
ADD COLUMN interval_seconds INTEGER NOT NULL DEFAULT 60;