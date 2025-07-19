-- Migration: Add LLM analysis columns to sentiment_suggestions
-- Purpose: Store enhanced LLM analysis data including insights and citations

-- Add columns for LLM enhanced analysis
ALTER TABLE sentiment_suggestions 
ADD COLUMN IF NOT EXISTS processing_method VARCHAR(20) DEFAULT 'keyword' CHECK (processing_method IN ('llm', 'fallback', 'keyword')),
ADD COLUMN IF NOT EXISTS analysis_quality VARCHAR(20) DEFAULT 'medium' CHECK (analysis_quality IN ('high', 'medium', 'low')),
ADD COLUMN IF NOT EXISTS key_insights JSONB,
ADD COLUMN IF NOT EXISTS market_context TEXT,
ADD COLUMN IF NOT EXISTS top_quotes JSONB,
ADD COLUMN IF NOT EXISTS llm_analysis_results JSONB,
ADD COLUMN IF NOT EXISTS total_articles INTEGER,
ADD COLUMN IF NOT EXISTS positive_articles INTEGER,
ADD COLUMN IF NOT EXISTS negative_articles INTEGER,
ADD COLUMN IF NOT EXISTS neutral_articles INTEGER;

-- Add index for processing method for analytics
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_processing_method ON sentiment_suggestions(processing_method);

-- Add index for quality analytics
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_quality ON sentiment_suggestions(analysis_quality);

-- Comments for documentation
COMMENT ON COLUMN sentiment_suggestions.processing_method IS 'Method used for analysis: llm, fallback, or keyword';
COMMENT ON COLUMN sentiment_suggestions.analysis_quality IS 'Quality level of the analysis: high, medium, or low';
COMMENT ON COLUMN sentiment_suggestions.key_insights IS 'JSON array of key insights from LLM analysis';
COMMENT ON COLUMN sentiment_suggestions.market_context IS 'Market context summary from analysis';
COMMENT ON COLUMN sentiment_suggestions.top_quotes IS 'JSON array of top quotes with sources and links';
COMMENT ON COLUMN sentiment_suggestions.llm_analysis_results IS 'Full LLM analysis results in JSON format';
COMMENT ON COLUMN sentiment_suggestions.total_articles IS 'Total number of articles analyzed';
COMMENT ON COLUMN sentiment_suggestions.positive_articles IS 'Number of positive sentiment articles';
COMMENT ON COLUMN sentiment_suggestions.negative_articles IS 'Number of negative sentiment articles';
COMMENT ON COLUMN sentiment_suggestions.neutral_articles IS 'Number of neutral sentiment articles';

-- Example of top_quotes JSONB structure:
-- [
--   {
--     "quote": "Bitcoin ETFs mark significant institutional adoption",
--     "source": "CoinDesk",
--     "link": "https://coindesk.com/article...",
--     "score": 0.8
--   }
-- ]

-- Example of key_insights JSONB structure:
-- ["Regulatory developments positive", "Institutional interest growing", "Market testing support levels"]