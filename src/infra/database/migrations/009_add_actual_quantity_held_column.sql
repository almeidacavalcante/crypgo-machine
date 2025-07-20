-- Add actual_quantity_held column to trade_bots table
-- This tracks the actual quantity of crypto held after trading fees

ALTER TABLE trade_bots 
ADD COLUMN actual_quantity_held DECIMAL(20, 8) DEFAULT 0.0;

-- Add comment for documentation
COMMENT ON COLUMN trade_bots.actual_quantity_held IS 'Actual quantity of cryptocurrency held after trading fees are applied';