-- Migration: 007_add_entry_price_column
-- Description: Add entry_price column to trade_bots table for position tracking
-- Date: 2025-07-18

ALTER TABLE trade_bots
    ADD COLUMN entry_price DOUBLE PRECISION DEFAULT 0.0;