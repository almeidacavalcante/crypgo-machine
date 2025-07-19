package external

import (
	"regexp"
	"strings"
)

type SentimentAnalyzer struct {
	positiveKeywords []string
	negativeKeywords []string
	neutralKeywords  []string
}

type SentimentResult struct {
	Score          float64 // Range: -1.0 to +1.0
	PositiveCount  int
	NegativeCount  int
	NeutralCount   int
	Classification string // "positive", "negative", "neutral"
	Confidence     float64 // 0.0 to 1.0
	AnalyzedText   string
}

type NewsAnalysisResult struct {
	OverallScore    float64
	TotalArticles   int
	PositiveArticles int
	NegativeArticles int
	NeutralArticles  int
	SourceBreakdown  map[string]SentimentResult
}

func NewSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{
		// Positive keywords as defined in the plan
		positiveKeywords: []string{
			"bullish", "rally", "surge", "moon", "pump", "adoption", 
			"institutional", "breakthrough", "innovation", "growth",
			"rising", "gains", "profit", "buy", "long", "optimistic",
			"bull run", "all-time high", "ath", "breakout", "momentum",
			"uptrend", "green", "recovery", "rebound", "bounce",
		},
		
		// Negative keywords as defined in the plan  
		negativeKeywords: []string{
			"bearish", "crash", "dump", "bear", "regulation", "ban",
			"decline", "fall", "drop", "sell", "short", "pessimistic",
			"bear market", "correction", "pullback", "dip", "red",
			"loss", "fear", "panic", "uncertainty", "risk", "volatile",
			"downtrend", "resistance", "rejection", "liquidation",
		},
		
		// Neutral keywords for context
		neutralKeywords: []string{
			"analysis", "prediction", "market", "trading", "price",
			"volume", "technical", "chart", "support", "bitcoin",
			"ethereum", "crypto", "blockchain", "defi", "nft",
		},
	}
}

func (s *SentimentAnalyzer) AnalyzeText(text string) SentimentResult {
	lowerText := strings.ToLower(text)
	
	positiveCount := s.countKeywords(lowerText, s.positiveKeywords)
	negativeCount := s.countKeywords(lowerText, s.negativeKeywords)
	neutralCount := s.countKeywords(lowerText, s.neutralKeywords)
	
	// Calculate raw score
	rawScore := positiveCount - negativeCount
	
	// Normalize score to -1.0 to +1.0 range
	totalKeywords := positiveCount + negativeCount
	var normalizedScore float64
	
	if totalKeywords > 0 {
		normalizedScore = float64(rawScore) / float64(totalKeywords)
	} else {
		normalizedScore = 0.0
	}
	
	// Ensure score stays within bounds
	if normalizedScore > 1.0 {
		normalizedScore = 1.0
	} else if normalizedScore < -1.0 {
		normalizedScore = -1.0
	}
	
	// Determine classification
	classification := "neutral"
	if normalizedScore > 0.1 {
		classification = "positive"
	} else if normalizedScore < -0.1 {
		classification = "negative"
	}
	
	// Calculate confidence based on keyword density
	wordCount := len(strings.Fields(text))
	confidence := 0.0
	if wordCount > 0 {
		keywordDensity := float64(totalKeywords) / float64(wordCount)
		confidence = keywordDensity * 10.0 // Scale up for better range
		if confidence > 1.0 {
			confidence = 1.0
		}
	}
	
	return SentimentResult{
		Score:          normalizedScore,
		PositiveCount:  positiveCount,
		NegativeCount:  negativeCount,
		NeutralCount:   neutralCount,
		Classification: classification,
		Confidence:     confidence,
		AnalyzedText:   text,
	}
}

func (s *SentimentAnalyzer) AnalyzeNews(newsItems []NewsItem) NewsAnalysisResult {
	result := NewsAnalysisResult{
		TotalArticles:   len(newsItems),
		SourceBreakdown: make(map[string]SentimentResult),
	}
	
	var totalScore float64
	sourceScores := make(map[string][]float64)
	
	for _, item := range newsItems {
		analysis := s.AnalyzeText(item.Content)
		totalScore += analysis.Score
		
		// Track by classification
		switch analysis.Classification {
		case "positive":
			result.PositiveArticles++
		case "negative":
			result.NegativeArticles++
		default:
			result.NeutralArticles++
		}
		
		// Aggregate by source
		if _, exists := sourceScores[item.Source]; !exists {
			sourceScores[item.Source] = []float64{}
		}
		sourceScores[item.Source] = append(sourceScores[item.Source], analysis.Score)
	}
	
	// Calculate overall score
	if result.TotalArticles > 0 {
		result.OverallScore = totalScore / float64(result.TotalArticles)
	}
	
	// Calculate source breakdown
	for source, scores := range sourceScores {
		var sourceTotal float64
		var posCount, negCount, neutCount int
		
		for _, score := range scores {
			sourceTotal += score
			if score > 0.1 {
				posCount++
			} else if score < -0.1 {
				negCount++
			} else {
				neutCount++
			}
		}
		
		avgScore := sourceTotal / float64(len(scores))
		classification := "neutral"
		if avgScore > 0.1 {
			classification = "positive"
		} else if avgScore < -0.1 {
			classification = "negative"
		}
		
		result.SourceBreakdown[source] = SentimentResult{
			Score:          avgScore,
			PositiveCount:  posCount,
			NegativeCount:  negCount,
			NeutralCount:   neutCount,
			Classification: classification,
			Confidence:     float64(len(scores)) / float64(result.TotalArticles),
		}
	}
	
	return result
}

// countKeywords counts occurrences of keywords in text using regex for whole words
func (s *SentimentAnalyzer) countKeywords(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		// Create regex pattern for whole word matching
		pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(text, -1)
		count += len(matches)
	}
	return count
}

// GetSentimentLevel returns human-readable sentiment level based on score
func (s *SentimentResult) GetSentimentLevel() string {
	switch {
	case s.Score >= 0.5:
		return "Very Positive"
	case s.Score >= 0.2:
		return "Positive"
	case s.Score >= -0.2:
		return "Neutral"
	case s.Score >= -0.5:
		return "Negative"
	default:
		return "Very Negative"
	}
}