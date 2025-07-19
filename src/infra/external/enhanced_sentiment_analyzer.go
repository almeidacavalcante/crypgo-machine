package external

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

type EnhancedSentimentAnalyzer struct {
	basicAnalyzer       *SentimentAnalyzer
	cryptoSpecificTerms map[string]float64
	intensifiers        map[string]float64
	negationWords       []string
	contextualPhrases   map[string]float64
	pricePatterns       []*regexp.Regexp
}

type EnhancedSentimentAnalysisResult struct {
	SentimentResult
	PriceMovementIndicators []PriceIndicator
	IntensityScore          float64
	ContextualScore         float64
	SentimentStrength       string
	KeyPhrases              []string
	NegationDetected        bool
	MarketCondition         string
}

type PriceIndicator struct {
	Type       string  // "percentage", "price_target", "support", "resistance"
	Value      float64
	Direction  string  // "up", "down", "neutral"
	Confidence float64
}

func NewEnhancedSentimentAnalyzer() *EnhancedSentimentAnalyzer {
	return &EnhancedSentimentAnalyzer{
		basicAnalyzer: NewSentimentAnalyzer(),
		
		// Crypto-specific terms with weighted scores
		cryptoSpecificTerms: map[string]float64{
			// Extremely bullish
			"moonshot": 0.9, "parabolic": 0.8, "rocket": 0.7, "lambo": 0.6,
			"diamond hands": 0.7, "hodl": 0.5, "btfd": 0.6, "accumulate": 0.4,
			
			// Technical bullish
			"breakout": 0.6, "bullish divergence": 0.7, "golden cross": 0.8,
			"cup and handle": 0.5, "ascending triangle": 0.4, "inverse head and shoulders": 0.6,
			
			// Bearish crypto terms
			"rekt": -0.8, "fud": -0.6, "rugpull": -0.9, "paper hands": -0.5,
			"death cross": -0.8, "bear flag": -0.6, "head and shoulders": -0.5,
			"falling knife": -0.7, "capitulation": -0.8, "blood bath": -0.9,
			
			// Market structure
			"market maker": 0.2, "whale movement": 0.3, "institutional adoption": 0.7,
			"retail fomo": 0.4, "smart money": 0.5, "dumb money": -0.3,
		},
		
		// Intensity modifiers
		intensifiers: map[string]float64{
			"extremely": 1.5, "massively": 1.4, "heavily": 1.3, "strongly": 1.2,
			"very": 1.2, "really": 1.1, "quite": 1.1, "pretty": 1.1,
			"slightly": 0.7, "somewhat": 0.8, "moderately": 0.9,
			"absolutely": 1.6, "incredibly": 1.5, "tremendously": 1.4,
		},
		
		// Negation detection
		negationWords: []string{
			"not", "no", "never", "none", "nothing", "nowhere", "neither",
			"nobody", "cannot", "can't", "won't", "shouldn't", "wouldn't",
			"isn't", "aren't", "wasn't", "weren't", "haven't", "hasn't",
			"hadn't", "don't", "doesn't", "didn't", "doubt", "unlikely",
		},
		
		// Contextual phrases that provide market insight
		contextualPhrases: map[string]float64{
			"fed pivot": 0.6, "rate cut": 0.5, "rate hike": -0.4,
			"inflation data": -0.2, "cpi data": -0.2, "recession": -0.7,
			"economic uncertainty": -0.5, "market volatility": -0.3,
			"risk on": 0.4, "risk off": -0.4, "flight to quality": -0.3,
			"etf approval": 0.8, "regulatory clarity": 0.6, "sec approval": 0.7,
			"institutional buying": 0.6, "retail selling": -0.3,
		},
		
		// Compile price patterns
		pricePatterns: []*regexp.Regexp{
			regexp.MustCompile(`(\d+(?:\.\d+)?%)`),                    // Percentages
			regexp.MustCompile(`\$(\d+(?:,\d{3})*(?:\.\d+)?)`),      // Dollar amounts
			regexp.MustCompile(`(\d+(?:,\d{3})*(?:\.\d+)?)\s*k`),    // Thousands (50k)
			regexp.MustCompile(`target.*?(\d+(?:,\d{3})*)`),         // Price targets
			regexp.MustCompile(`support.*?(\d+(?:,\d{3})*)`),        // Support levels
			regexp.MustCompile(`resistance.*?(\d+(?:,\d{3})*)`),     // Resistance levels
		},
	}
}

func (e *EnhancedSentimentAnalyzer) AnalyzeText(text string) EnhancedSentimentAnalysisResult {
	// Start with basic analysis
	basicResult := e.basicAnalyzer.AnalyzeText(text)
	
	// Enhanced analysis
	lowerText := strings.ToLower(text)
	
	// Detect price indicators
	priceIndicators := e.extractPriceIndicators(text)
	
	// Calculate crypto-specific sentiment
	cryptoScore := e.calculateCryptoSpecificScore(lowerText)
	
	// Apply intensity modifiers
	intensityScore := e.calculateIntensityScore(lowerText, basicResult.Score)
	
	// Detect negation and adjust score
	negationDetected := e.detectNegation(lowerText)
	adjustedScore := basicResult.Score
	if negationDetected {
		adjustedScore *= -0.7 // Flip and reduce intensity for negation
	}
	
	// Calculate contextual score
	contextualScore := e.calculateContextualScore(lowerText)
	
	// Combine all scores with weights
	finalScore := (adjustedScore * 0.4) + (cryptoScore * 0.3) + (contextualScore * 0.2) + (intensityScore * 0.1)
	
	// Ensure score bounds
	if finalScore > 1.0 {
		finalScore = 1.0
	} else if finalScore < -1.0 {
		finalScore = -1.0
	}
	
	// Extract key phrases
	keyPhrases := e.extractKeyPhrases(text)
	
	// Determine market condition
	marketCondition := e.determineMarketCondition(finalScore, priceIndicators)
	
	// Calculate sentiment strength
	sentimentStrength := e.calculateSentimentStrength(finalScore, intensityScore)
	
	return EnhancedSentimentAnalysisResult{
		SentimentResult: SentimentResult{
			Score:          finalScore,
			PositiveCount:  basicResult.PositiveCount,
			NegativeCount:  basicResult.NegativeCount,
			NeutralCount:   basicResult.NeutralCount,
			Classification: e.getEnhancedClassification(finalScore),
			Confidence:     e.calculateEnhancedConfidence(basicResult, cryptoScore, contextualScore),
			AnalyzedText:   text,
		},
		PriceMovementIndicators: priceIndicators,
		IntensityScore:          intensityScore,
		ContextualScore:         contextualScore,
		SentimentStrength:       sentimentStrength,
		KeyPhrases:              keyPhrases,
		NegationDetected:        negationDetected,
		MarketCondition:         marketCondition,
	}
}

func (e *EnhancedSentimentAnalyzer) calculateCryptoSpecificScore(text string) float64 {
	var score float64
	var matchCount int
	
	for term, value := range e.cryptoSpecificTerms {
		if strings.Contains(text, term) {
			score += value
			matchCount++
		}
	}
	
	if matchCount > 0 {
		return score / float64(matchCount) // Average score
	}
	return 0.0
}

func (e *EnhancedSentimentAnalyzer) calculateIntensityScore(text string, baseScore float64) float64 {
	maxIntensity := 1.0
	
	for intensifier, multiplier := range e.intensifiers {
		if strings.Contains(text, intensifier) {
			if multiplier > maxIntensity {
				maxIntensity = multiplier
			}
		}
	}
	
	return baseScore * maxIntensity
}

func (e *EnhancedSentimentAnalyzer) detectNegation(text string) bool {
	for _, negation := range e.negationWords {
		if strings.Contains(text, negation) {
			return true
		}
	}
	return false
}

func (e *EnhancedSentimentAnalyzer) calculateContextualScore(text string) float64 {
	var score float64
	var matchCount int
	
	for phrase, value := range e.contextualPhrases {
		if strings.Contains(text, phrase) {
			score += value
			matchCount++
		}
	}
	
	if matchCount > 0 {
		return score / float64(matchCount)
	}
	return 0.0
}

func (e *EnhancedSentimentAnalyzer) extractPriceIndicators(text string) []PriceIndicator {
	var indicators []PriceIndicator
	
	// Extract percentages
	percentagePattern := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*%`)
	percentages := percentagePattern.FindAllStringSubmatch(text, -1)
	for _, match := range percentages {
		if value, err := strconv.ParseFloat(match[1], 64); err == nil {
			direction := "neutral"
			if strings.Contains(strings.ToLower(text), "up") || strings.Contains(strings.ToLower(text), "gain") {
				direction = "up"
			} else if strings.Contains(strings.ToLower(text), "down") || strings.Contains(strings.ToLower(text), "loss") {
				direction = "down"
			}
			
			indicators = append(indicators, PriceIndicator{
				Type:       "percentage",
				Value:      value,
				Direction:  direction,
				Confidence: 0.8,
			})
		}
	}
	
	// Extract price targets
	targetPattern := regexp.MustCompile(`target.*?(\d+(?:,\d{3})*(?:\.\d+)?)`)
	targets := targetPattern.FindAllStringSubmatch(text, -1)
	for _, match := range targets {
		if valueStr := strings.ReplaceAll(match[1], ",", ""); valueStr != "" {
			if value, err := strconv.ParseFloat(valueStr, 64); err == nil {
				indicators = append(indicators, PriceIndicator{
					Type:       "price_target",
					Value:      value,
					Direction:  "up",
					Confidence: 0.7,
				})
			}
		}
	}
	
	return indicators
}

func (e *EnhancedSentimentAnalyzer) extractKeyPhrases(text string) []string {
	var phrases []string
	
	// Simple extraction of important phrases (can be enhanced with NLP libraries)
	text = strings.ToLower(text)
	
	// Look for common crypto phrases
	importantPhrases := []string{
		"bull run", "bear market", "all time high", "price action", "market cap",
		"trading volume", "technical analysis", "fundamental analysis", "market sentiment",
		"price discovery", "accumulation phase", "distribution phase",
	}
	
	for _, phrase := range importantPhrases {
		if strings.Contains(text, phrase) {
			phrases = append(phrases, phrase)
		}
	}
	
	return phrases
}

func (e *EnhancedSentimentAnalyzer) determineMarketCondition(score float64, indicators []PriceIndicator) string {
	switch {
	case score >= 0.6:
		return "strong_bullish"
	case score >= 0.3:
		return "bullish"
	case score >= -0.3:
		return "neutral"
	case score >= -0.6:
		return "bearish"
	default:
		return "strong_bearish"
	}
}

func (e *EnhancedSentimentAnalyzer) calculateSentimentStrength(score float64, intensity float64) string {
	absScore := math.Abs(score)
	
	switch {
	case absScore >= 0.8 && intensity > 1.2:
		return "very_strong"
	case absScore >= 0.6:
		return "strong"
	case absScore >= 0.3:
		return "moderate"
	case absScore >= 0.1:
		return "weak"
	default:
		return "neutral"
	}
}

func (e *EnhancedSentimentAnalyzer) getEnhancedClassification(score float64) string {
	switch {
	case score >= 0.6:
		return "very_positive"
	case score >= 0.2:
		return "positive"
	case score >= -0.2:
		return "neutral"
	case score >= -0.6:
		return "negative"
	default:
		return "very_negative"
	}
}

func (e *EnhancedSentimentAnalyzer) calculateEnhancedConfidence(basic SentimentResult, cryptoScore, contextualScore float64) float64 {
	// Base confidence from basic analyzer
	baseConfidence := basic.Confidence
	
	// Boost confidence if we have crypto-specific or contextual matches
	cryptoBoost := 0.0
	if cryptoScore != 0 {
		cryptoBoost = 0.2
	}
	
	contextualBoost := 0.0
	if contextualScore != 0 {
		contextualBoost = 0.15
	}
	
	totalConfidence := baseConfidence + cryptoBoost + contextualBoost
	
	if totalConfidence > 1.0 {
		totalConfidence = 1.0
	}
	
	return totalConfidence
}

// Helper method to determine if text contains financial/price information
func (e *EnhancedSentimentAnalyzer) containsPriceInformation(text string) bool {
	for _, pattern := range e.pricePatterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

// Method to analyze market urgency based on sentiment and keywords
func (e *EnhancedSentimentAnalyzer) calculateMarketUrgency(text string, score float64) string {
	urgencyWords := []string{
		"urgent", "immediately", "now", "asap", "breaking", "alert",
		"emergency", "critical", "important", "time sensitive",
	}
	
	lowerText := strings.ToLower(text)
	hasUrgency := false
	
	for _, word := range urgencyWords {
		if strings.Contains(lowerText, word) {
			hasUrgency = true
			break
		}
	}
	
	if hasUrgency && math.Abs(score) > 0.5 {
		return "high"
	} else if hasUrgency || math.Abs(score) > 0.7 {
		return "medium"
	}
	
	return "low"
}