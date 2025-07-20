-- Add use_fixed_quantity column to trade_bots table
-- This determines whether to use fixed quantity or calculate dynamic quantity based on trade amount

ALTER TABLE trade_bots 
ADD COLUMN use_fixed_quantity BOOLEAN DEFAULT true;

-- Add comment for documentation
COMMENT ON COLUMN trade_bots.use_fixed_quantity IS 'When true, use the quantity field for trading. When false, calculate dynamic quantity from tradeAmount and current price';