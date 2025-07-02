-- Migration: 002_add_strategy_params_column
-- Description: Add strategy_params JSON column to store strategy parameters
-- Date: 2025-06-28

ALTER TABLE trade_bots 
ADD COLUMN strategy_params TEXT;

-- Set default empty JSON for existing records (if any)
UPDATE trade_bots 
SET strategy_params = '{}' 
WHERE strategy_params IS NULL;

-- Make the column NOT NULL after setting defaults
ALTER TABLE trade_bots 
ALTER COLUMN strategy_params SET NOT NULL;