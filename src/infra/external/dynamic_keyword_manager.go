package external

import (
	"encoding/json"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

type DynamicKeywordManager struct {
	keywords          map[string]*KeywordData
	contextualRules   []ContextualRule
	trendingKeywords  map[string]*TrendingKeyword
	marketConditions  map[string]*MarketCondition
	keywordMutex      sync.RWMutex
	learningEnabled   bool
	adaptationRate    float64
	minOccurrences    int
}

type KeywordData struct {
	Word             string                 `json:"word"`
	BaseScore        float64               `json:"base_score"`
	DynamicScore     float64               `json:"dynamic_score"`
	Context          map[string]float64    `json:"context"`
	Frequency        int                   `json:"frequency"`
	LastSeen         time.Time             `json:"last_seen"`
	MarketCorrelation float64              `json:"market_correlation"`
	Confidence       float64               `json:"confidence"`
	Category         string                `json:"category"`
}

type ContextualRule struct {
	Pattern      string                 `json:"pattern"`
	Context      string                 `json:"context"`
	Modifier     float64               `json:"modifier"`
	Prerequisites []string              `json:"prerequisites"`
	ExpiresAt    time.Time             `json:"expires_at"`
}

type TrendingKeyword struct {
	Word              string    `json:"word"`
	TrendScore        float64   `json:"trend_score"`
	VelocityScore     float64   `json:"velocity_score"`
	FirstSeen         time.Time `json:"first_seen"`
	PeakTime          time.Time `json:"peak_time"`
	OccurrenceHistory []int     `json:"occurrence_history"`
	SentimentHistory  []float64 `json:"sentiment_history"`
}

type MarketCondition struct {
	Name        string             `json:"name"`
	IsActive    bool              `json:"is_active"`
	Confidence  float64           `json:"confidence"`
	StartTime   time.Time         `json:"start_time"`
	Keywords    map[string]float64 `json:"keywords"`
	Description string            `json:"description"`
}

type KeywordLearningResult struct {
	NewKeywords     []string `json:"new_keywords"`
	UpdatedKeywords []string `json:"updated_keywords"`
	TrendingWords   []string `json:"trending_words"`
	MarketSignals   []string `json:"market_signals"`
}

func NewDynamicKeywordManager() *DynamicKeywordManager {
	manager := &DynamicKeywordManager{
		keywords:         make(map[string]*KeywordData),
		trendingKeywords: make(map[string]*TrendingKeyword),
		marketConditions: make(map[string]*MarketCondition),
		learningEnabled:  true,
		adaptationRate:   0.1,
		minOccurrences:   3,
	}
	
	// Initialize with base keywords
	manager.initializeBaseKeywords()
	manager.initializeContextualRules()
	manager.initializeMarketConditions()
	
	return manager
}

func (d *DynamicKeywordManager) initializeBaseKeywords() {
	baseKeywords := map[string]KeywordData{
		// Crypto-specific bullish
		"hodl": {Word: "hodl", BaseScore: 0.6, Category: "crypto_culture"},
		"diamond_hands": {Word: "diamond hands", BaseScore: 0.7, Category: "crypto_culture"},
		"to_the_moon": {Word: "to the moon", BaseScore: 0.8, Category: "crypto_culture"},
		"lambo": {Word: "lambo", BaseScore: 0.6, Category: "crypto_culture"},
		"ath": {Word: "ath", BaseScore: 0.7, Category: "technical"},
		"breakout": {Word: "breakout", BaseScore: 0.6, Category: "technical"},
		"golden_cross": {Word: "golden cross", BaseScore: 0.8, Category: "technical"},
		
		// Crypto-specific bearish
		"rekt": {Word: "rekt", BaseScore: -0.8, Category: "crypto_culture"},
		"paper_hands": {Word: "paper hands", BaseScore: -0.5, Category: "crypto_culture"},
		"rugpull": {Word: "rugpull", BaseScore: -0.9, Category: "scam"},
		"fud": {Word: "fud", BaseScore: -0.6, Category: "sentiment"},
		"death_cross": {Word: "death cross", BaseScore: -0.8, Category: "technical"},
		"bear_trap": {Word: "bear trap", BaseScore: -0.6, Category: "technical"},
		
		// Market structure
		"institutional_adoption": {Word: "institutional adoption", BaseScore: 0.7, Category: "institutional"},
		"retail_fomo": {Word: "retail fomo", BaseScore: 0.4, Category: "retail"},
		"whale_movement": {Word: "whale movement", BaseScore: 0.3, Category: "whale"},
		"smart_money": {Word: "smart money", BaseScore: 0.5, Category: "institutional"},
		
		// Regulatory
		"sec_approval": {Word: "sec approval", BaseScore: 0.8, Category: "regulatory"},
		"etf_approval": {Word: "etf approval", BaseScore: 0.9, Category: "regulatory"},
		"regulatory_clarity": {Word: "regulatory clarity", BaseScore: 0.6, Category: "regulatory"},
		"cbdc": {Word: "cbdc", BaseScore: -0.2, Category: "regulatory"},
		
		// DeFi specific
		"yield_farming": {Word: "yield farming", BaseScore: 0.4, Category: "defi"},
		"liquidity_mining": {Word: "liquidity mining", BaseScore: 0.3, Category: "defi"},
		"total_value_locked": {Word: "total value locked", BaseScore: 0.5, Category: "defi"},
		"impermanent_loss": {Word: "impermanent loss", BaseScore: -0.4, Category: "defi"},
		
		// Macro economic
		"inflation_hedge": {Word: "inflation hedge", BaseScore: 0.5, Category: "macro"},
		"digital_gold": {Word: "digital gold", BaseScore: 0.6, Category: "macro"},
		"store_of_value": {Word: "store of value", BaseScore: 0.5, Category: "macro"},
		"currency_debasement": {Word: "currency debasement", BaseScore: 0.4, Category: "macro"},
	}
	
	for word, data := range baseKeywords {
		data.DynamicScore = data.BaseScore
		data.Context = make(map[string]float64)
		data.LastSeen = time.Now()
		data.Confidence = 0.8
		d.keywords[word] = &data
	}
}

func (d *DynamicKeywordManager) initializeContextualRules() {
	d.contextualRules = []ContextualRule{
		// Price movement context
		{
			Pattern:   "pump",
			Context:   "price_up",
			Modifier:  1.5,
			Prerequisites: []string{"percentage", "gain", "up"},
		},
		{
			Pattern:   "dump",
			Context:   "price_down",
			Modifier:  -1.5,
			Prerequisites: []string{"percentage", "loss", "down"},
		},
		
		// News context
		{
			Pattern:   "partnership",
			Context:   "news_positive",
			Modifier:  1.3,
			Prerequisites: []string{"announce", "collaboration"},
		},
		{
			Pattern:   "hack",
			Context:   "news_negative",
			Modifier:  -1.8,
			Prerequisites: []string{"exploit", "stolen", "breach"},
		},
		
		// Volume context
		{
			Pattern:   "accumulation",
			Context:   "volume_high",
			Modifier:  1.2,
			Prerequisites: []string{"volume", "buying"},
		},
		{
			Pattern:   "distribution",
			Context:   "volume_high",
			Modifier:  -1.2,
			Prerequisites: []string{"volume", "selling"},
		},
		
		// Time-sensitive context
		{
			Pattern:   "halving",
			Context:   "event_upcoming",
			Modifier:  1.4,
			ExpiresAt: time.Now().Add(180 * 24 * time.Hour), // 6 months
		},
		{
			Pattern:   "etf_decision",
			Context:   "regulatory_pending",
			Modifier:  1.2,
			ExpiresAt: time.Now().Add(90 * 24 * time.Hour), // 3 months
		},
	}
}

func (d *DynamicKeywordManager) initializeMarketConditions() {
	d.marketConditions = map[string]*MarketCondition{
		"bull_market": {
			Name:        "bull_market",
			IsActive:    false,
			Keywords:    map[string]float64{"rally": 1.5, "moon": 1.3, "bullish": 1.2},
			Description: "Sustained upward price movement with positive sentiment",
		},
		"bear_market": {
			Name:        "bear_market",
			IsActive:    false,
			Keywords:    map[string]float64{"crash": -1.5, "capitulation": -1.4, "bearish": -1.2},
			Description: "Sustained downward price movement with negative sentiment",
		},
		"accumulation_phase": {
			Name:        "accumulation_phase",
			IsActive:    false,
			Keywords:    map[string]float64{"accumulate": 1.1, "dca": 1.0, "buy_dip": 1.2},
			Description: "Sideways movement with institutional buying",
		},
		"distribution_phase": {
			Name:        "distribution_phase",
			IsActive:    false,
			Keywords:    map[string]float64{"profit_taking": -1.1, "sell_high": -1.0, "distribution": -1.2},
			Description: "High prices with smart money selling",
		},
		"fomo_phase": {
			Name:        "fomo_phase",
			IsActive:    false,
			Keywords:    map[string]float64{"fomo": 1.3, "retail_buying": 1.2, "euphoria": 1.4},
			Description: "Extreme retail buying driven by fear of missing out",
		},
	}
}

// LearnFromText analyzes text and updates keyword scores dynamically
func (d *DynamicKeywordManager) LearnFromText(text string, actualSentiment float64, marketMovement float64) KeywordLearningResult {
	if !d.learningEnabled {
		return KeywordLearningResult{}
	}
	
	d.keywordMutex.Lock()
	defer d.keywordMutex.Unlock()
	
	result := KeywordLearningResult{}
	words := d.extractWords(text)
	
	// Update existing keywords and discover new ones
	for _, word := range words {
		if keyword, exists := d.keywords[word]; exists {
			// Update existing keyword
			d.updateKeywordScore(keyword, actualSentiment, marketMovement)
			result.UpdatedKeywords = append(result.UpdatedKeywords, word)
		} else {
			// Potentially learn new keyword
			if d.shouldLearnKeyword(word, text, actualSentiment) {
				d.learnNewKeyword(word, actualSentiment, marketMovement)
				result.NewKeywords = append(result.NewKeywords, word)
			}
		}
		
		// Update trending analysis
		d.updateTrendingAnalysis(word, actualSentiment)
	}
	
	// Update market conditions
	d.updateMarketConditions(text, actualSentiment, marketMovement)
	
	// Identify trending keywords
	result.TrendingWords = d.identifyTrendingKeywords()
	result.MarketSignals = d.detectMarketSignals()
	
	return result
}

// GetDynamicScore returns the current dynamic score for a keyword
func (d *DynamicKeywordManager) GetDynamicScore(word string, context map[string]interface{}) float64 {
	d.keywordMutex.RLock()
	defer d.keywordMutex.RUnlock()
	
	keyword, exists := d.keywords[word]
	if !exists {
		return 0.0
	}
	
	score := keyword.DynamicScore
	
	// Apply contextual modifiers
	for _, rule := range d.contextualRules {
		if strings.Contains(word, rule.Pattern) && d.checkPrerequisites(rule, context) {
			if rule.ExpiresAt.IsZero() || time.Now().Before(rule.ExpiresAt) {
				score *= rule.Modifier
			}
		}
	}
	
	// Apply market condition modifiers
	for _, condition := range d.marketConditions {
		if condition.IsActive {
			if modifier, exists := condition.Keywords[word]; exists {
				score *= modifier
			}
		}
	}
	
	// Ensure score stays within reasonable bounds
	if score > 2.0 {
		score = 2.0
	} else if score < -2.0 {
		score = -2.0
	}
	
	return score
}

// GetTrendingKeywords returns currently trending keywords
func (d *DynamicKeywordManager) GetTrendingKeywords(limit int) []TrendingKeyword {
	d.keywordMutex.RLock()
	defer d.keywordMutex.RUnlock()
	
	var trending []TrendingKeyword
	for _, keyword := range d.trendingKeywords {
		trending = append(trending, *keyword)
	}
	
	// Sort by trend score
	sort.Slice(trending, func(i, j int) bool {
		return trending[i].TrendScore > trending[j].TrendScore
	})
	
	if limit > 0 && len(trending) > limit {
		trending = trending[:limit]
	}
	
	return trending
}

// GetMarketConditions returns current market conditions
func (d *DynamicKeywordManager) GetMarketConditions() map[string]MarketCondition {
	d.keywordMutex.RLock()
	defer d.keywordMutex.RUnlock()
	
	conditions := make(map[string]MarketCondition)
	for name, condition := range d.marketConditions {
		conditions[name] = *condition
	}
	
	return conditions
}

// Private methods

func (d *DynamicKeywordManager) extractWords(text string) []string {
	// Simple word extraction (can be enhanced with NLP)
	text = strings.ToLower(text)
	words := strings.Fields(text)
	
	var extracted []string
	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?;:()[]{}\"'")
		if len(word) > 2 { // Filter very short words
			extracted = append(extracted, word)
		}
	}
	
	// Also extract common phrases
	phrases := d.extractPhrases(text)
	extracted = append(extracted, phrases...)
	
	return extracted
}

func (d *DynamicKeywordManager) extractPhrases(text string) []string {
	commonPhrases := []string{
		"all time high", "bear market", "bull run", "diamond hands", "paper hands",
		"to the moon", "smart money", "retail fomo", "whale movement", "market maker",
		"pump and dump", "fear and greed", "buy the dip", "sell the top",
	}
	
	var found []string
	for _, phrase := range commonPhrases {
		if strings.Contains(text, phrase) {
			found = append(found, phrase)
		}
	}
	
	return found
}

func (d *DynamicKeywordManager) shouldLearnKeyword(word, text string, sentiment float64) bool {
	// Don't learn very common words
	commonWords := []string{"the", "and", "or", "but", "is", "are", "was", "were", "a", "an"}
	for _, common := range commonWords {
		if word == common {
			return false
		}
	}
	
	// Learn if word appears with strong sentiment
	return math.Abs(sentiment) > 0.3 && len(word) > 3
}

func (d *DynamicKeywordManager) learnNewKeyword(word string, sentiment, marketMovement float64) {
	category := d.categorizeWord(word)
	
	keyword := &KeywordData{
		Word:         word,
		BaseScore:    sentiment * 0.5, // Conservative initial score
		DynamicScore: sentiment * 0.5,
		Context:      make(map[string]float64),
		Frequency:    1,
		LastSeen:     time.Now(),
		MarketCorrelation: marketMovement,
		Confidence:   0.3, // Low initial confidence
		Category:     category,
	}
	
	d.keywords[word] = keyword
}

func (d *DynamicKeywordManager) updateKeywordScore(keyword *KeywordData, sentiment, marketMovement float64) {
	keyword.Frequency++
	keyword.LastSeen = time.Now()
	
	// Update dynamic score with exponential moving average
	keyword.DynamicScore = keyword.DynamicScore*(1-d.adaptationRate) + sentiment*d.adaptationRate
	
	// Update market correlation
	keyword.MarketCorrelation = keyword.MarketCorrelation*0.9 + marketMovement*0.1
	
	// Increase confidence as we see more occurrences
	maxConfidence := 0.95
	confidenceIncrease := 0.05
	if keyword.Confidence < maxConfidence {
		keyword.Confidence = math.Min(maxConfidence, keyword.Confidence+confidenceIncrease)
	}
}

func (d *DynamicKeywordManager) categorizeWord(word string) string {
	cryptoTerms := []string{"bitcoin", "ethereum", "crypto", "blockchain", "defi", "nft"}
	techTerms := []string{"breakout", "support", "resistance", "volume", "chart", "technical"}
	sentimentTerms := []string{"bullish", "bearish", "fud", "fomo", "euphoria", "panic"}
	
	for _, term := range cryptoTerms {
		if strings.Contains(word, term) {
			return "crypto"
		}
	}
	for _, term := range techTerms {
		if strings.Contains(word, term) {
			return "technical"
		}
	}
	for _, term := range sentimentTerms {
		if strings.Contains(word, term) {
			return "sentiment"
		}
	}
	
	return "general"
}

func (d *DynamicKeywordManager) updateTrendingAnalysis(word string, sentiment float64) {
	trending, exists := d.trendingKeywords[word]
	if !exists {
		trending = &TrendingKeyword{
			Word:              word,
			FirstSeen:         time.Now(),
			OccurrenceHistory: make([]int, 24), // 24 hours
			SentimentHistory:  make([]float64, 24),
		}
		d.trendingKeywords[word] = trending
	}
	
	// Update occurrence count for current hour
	hour := time.Now().Hour()
	trending.OccurrenceHistory[hour]++
	trending.SentimentHistory[hour] = (trending.SentimentHistory[hour] + sentiment) / 2
	
	// Calculate trend score based on recent activity
	recentOccurrences := 0
	for i := 0; i < 6; i++ { // Last 6 hours
		idx := (hour - i + 24) % 24
		recentOccurrences += trending.OccurrenceHistory[idx]
	}
	
	// Calculate velocity (change in occurrences)
	oldOccurrences := 0
	for i := 6; i < 12; i++ { // 6-12 hours ago
		idx := (hour - i + 24) % 24
		oldOccurrences += trending.OccurrenceHistory[idx]
	}
	
	if oldOccurrences > 0 {
		trending.VelocityScore = float64(recentOccurrences-oldOccurrences) / float64(oldOccurrences)
	} else {
		trending.VelocityScore = float64(recentOccurrences)
	}
	
	trending.TrendScore = float64(recentOccurrences) * (1.0 + trending.VelocityScore)
	
	if trending.TrendScore > 10 { // Arbitrary threshold for "trending"
		trending.PeakTime = time.Now()
	}
}

func (d *DynamicKeywordManager) checkPrerequisites(rule ContextualRule, context map[string]interface{}) bool {
	for _, prereq := range rule.Prerequisites {
		if _, exists := context[prereq]; !exists {
			return false
		}
	}
	return true
}

func (d *DynamicKeywordManager) updateMarketConditions(text string, sentiment, marketMovement float64) {
	// Simple market condition detection based on sentiment and movement
	if sentiment > 0.6 && marketMovement > 0.05 {
		d.activateCondition("bull_market")
		d.deactivateCondition("bear_market")
	} else if sentiment < -0.6 && marketMovement < -0.05 {
		d.activateCondition("bear_market")
		d.deactivateCondition("bull_market")
	}
	
	// FOMO detection
	if strings.Contains(text, "fomo") || strings.Contains(text, "fear of missing out") {
		d.activateCondition("fomo_phase")
	}
}

func (d *DynamicKeywordManager) activateCondition(name string) {
	if condition, exists := d.marketConditions[name]; exists {
		if !condition.IsActive {
			condition.IsActive = true
			condition.StartTime = time.Now()
			condition.Confidence = 0.7
		}
	}
}

func (d *DynamicKeywordManager) deactivateCondition(name string) {
	if condition, exists := d.marketConditions[name]; exists {
		condition.IsActive = false
	}
}

func (d *DynamicKeywordManager) identifyTrendingKeywords() []string {
	var trending []string
	for word, data := range d.trendingKeywords {
		if data.TrendScore > 5 { // Threshold for trending
			trending = append(trending, word)
		}
	}
	return trending
}

func (d *DynamicKeywordManager) detectMarketSignals() []string {
	var signals []string
	for name, condition := range d.marketConditions {
		if condition.IsActive && condition.Confidence > 0.6 {
			signals = append(signals, name)
		}
	}
	return signals
}

// ExportKeywords exports current keyword data for persistence
func (d *DynamicKeywordManager) ExportKeywords() ([]byte, error) {
	d.keywordMutex.RLock()
	defer d.keywordMutex.RUnlock()
	
	data := map[string]interface{}{
		"keywords":          d.keywords,
		"trending_keywords": d.trendingKeywords,
		"market_conditions": d.marketConditions,
		"contextual_rules":  d.contextualRules,
	}
	
	return json.Marshal(data)
}

// ImportKeywords imports keyword data from persistence
func (d *DynamicKeywordManager) ImportKeywords(data []byte) error {
	d.keywordMutex.Lock()
	defer d.keywordMutex.Unlock()
	
	var imported map[string]interface{}
	if err := json.Unmarshal(data, &imported); err != nil {
		return err
	}
	
	// Import keywords
	if keywordsData, exists := imported["keywords"]; exists {
		if keywordsBytes, err := json.Marshal(keywordsData); err == nil {
			var keywords map[string]*KeywordData
			if err := json.Unmarshal(keywordsBytes, &keywords); err == nil {
				d.keywords = keywords
			}
		}
	}
	
	// Import trending keywords
	if trendingData, exists := imported["trending_keywords"]; exists {
		if trendingBytes, err := json.Marshal(trendingData); err == nil {
			var trending map[string]*TrendingKeyword
			if err := json.Unmarshal(trendingBytes, &trending); err == nil {
				d.trendingKeywords = trending
			}
		}
	}
	
	return nil
}