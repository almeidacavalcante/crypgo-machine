-- Migration: Create sentiment_suggestions table
-- Purpose: Store sentiment analysis suggestions and user responses

CREATE TABLE IF NOT EXISTS sentiment_suggestions (
    id VARCHAR(36) PRIMARY KEY,
    
    -- Sentiment Analysis Data
    overall_score DECIMAL(5,3) NOT NULL CHECK (overall_score >= -1.0 AND overall_score <= 1.0),
    level VARCHAR(20) NOT NULL CHECK (level IN ('very_bearish', 'bearish', 'neutral', 'bullish', 'very_bullish')),
    
    -- Source Data
    fear_greed_index INTEGER NOT NULL CHECK (fear_greed_index >= 0 AND fear_greed_index <= 100),
    news_score DECIMAL(5,3) NOT NULL CHECK (news_score >= -1.0 AND news_score <= 1.0),
    reddit_score DECIMAL(5,3) NOT NULL CHECK (reddit_score >= -1.0 AND reddit_score <= 1.0),
    social_score DECIMAL(5,3) NOT NULL CHECK (social_score >= -1.0 AND social_score <= 1.0),
    
    -- Suggested Values
    suggested_multiplier DECIMAL(5,2) NOT NULL CHECK (suggested_multiplier > 0),
    suggested_threshold DECIMAL(5,2) NOT NULL CHECK (suggested_threshold > 0),
    suggested_interval INTEGER NOT NULL CHECK (suggested_interval > 0),
    
    -- Analysis Metadata
    reasoning TEXT NOT NULL,
    confidence DECIMAL(5,3) NOT NULL CHECK (confidence >= 0.0 AND confidence <= 1.0),
    
    -- User Response
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'ignored', 'customized')),
    user_notes TEXT,
    
    -- Applied Values (when approved/customized)
    applied_multiplier DECIMAL(5,2) CHECK (applied_multiplier > 0),
    applied_threshold DECIMAL(5,2) CHECK (applied_threshold > 0),
    applied_interval INTEGER CHECK (applied_interval > 0),
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_applied_values CHECK (
        (status = 'pending' AND applied_multiplier IS NULL AND applied_threshold IS NULL AND applied_interval IS NULL AND responded_at IS NULL) OR
        (status = 'ignored' AND applied_multiplier IS NULL AND applied_threshold IS NULL AND applied_interval IS NULL AND responded_at IS NOT NULL) OR
        (status IN ('approved', 'customized') AND applied_multiplier IS NOT NULL AND applied_threshold IS NOT NULL AND applied_interval IS NOT NULL AND responded_at IS NOT NULL)
    )
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_status ON sentiment_suggestions(status);
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_created_at ON sentiment_suggestions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_level ON sentiment_suggestions(level);
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_pending ON sentiment_suggestions(status, created_at) WHERE status = 'pending';

-- Composite index for analytics queries
CREATE INDEX IF NOT EXISTS idx_sentiment_suggestions_analytics ON sentiment_suggestions(status, created_at, responded_at);

-- Comments for documentation
COMMENT ON TABLE sentiment_suggestions IS 'Stores sentiment analysis suggestions and user approval decisions';
COMMENT ON COLUMN sentiment_suggestions.overall_score IS 'Calculated overall sentiment score (-1.0 to 1.0)';
COMMENT ON COLUMN sentiment_suggestions.level IS 'Sentiment level classification';
COMMENT ON COLUMN sentiment_suggestions.fear_greed_index IS 'Fear & Greed Index value (0-100)';
COMMENT ON COLUMN sentiment_suggestions.confidence IS 'Confidence level of the analysis (0.0 to 1.0)';
COMMENT ON COLUMN sentiment_suggestions.status IS 'User response to the suggestion';
COMMENT ON COLUMN sentiment_suggestions.reasoning IS 'Explanation of why these suggestions were made';

-- Sample data for testing (optional)
-- INSERT INTO sentiment_suggestions (
--     id, overall_score, level, fear_greed_index, news_score, reddit_score, social_score,
--     suggested_multiplier, suggested_threshold, suggested_interval, reasoning, confidence
-- ) VALUES (
--     'sample-uuid-1', 0.25, 'bullish', 68, 0.3, 0.1, 0.2,
--     1.2, 1.0, 600, 'Positive sentiment indicates opportunity for increased aggressiveness', 0.75
-- );